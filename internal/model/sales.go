package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jacky-htg/erp-pkg/app"
	"github.com/jacky-htg/erp-proto/go/pb/sales"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Sales struct {
	Pb sales.Sales
}

func (u *Sales) Get(ctx context.Context, db *sql.DB) error {
	query := `
		SELECT sales.id, sales.company_id, sales.branch_id, sales.branch_name, sales.customer_id, sales.salesman_id, sales.code, 
		sales.sales_date, sales.remark, sales.price, sales.additional_disc_amount, sales.additional_disc_percentage, sales.total_price,
		sales.created_at, sales.created_by, sales.updated_at, sales.updated_by,
		json_agg(DISTINCT jsonb_build_object(
			'id', sales_details.id,
			'sales_id', sales_details.sales_id,
			'product_id', sales_details.product_id,
			'price', sales_details.price,
			'disc_amount', sales_details.disc_amount,
			'disc_percentage', sales_details.disc_percentage,
			'quantity', sales_details.quantity,
			'total_price', sales_details.total_price
		)) as details
		FROM sales 
		JOIN sales_details ON sales.id = sales_details.sales_id
		WHERE sales.id = $1
	`

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare statement Get sales: %v", err)
	}
	defer stmt.Close()

	var dateSales, createdAt, updatedAt time.Time
	var companyID, details string
	err = stmt.QueryRowContext(ctx, u.Pb.GetId()).Scan(
		&u.Pb.Id, &companyID, &u.Pb.BranchId, &u.Pb.BranchName, &u.Pb.GetCustomer().Id, &u.Pb.GetSalesman().Id,
		&u.Pb.Code, &dateSales, &u.Pb.Remark,
		&u.Pb.Price, &u.Pb.AdditionalDiscAmount, &u.Pb.AdditionalDiscPercentage, &u.Pb.TotalPrice,
		&createdAt, &u.Pb.CreatedBy, &updatedAt, &u.Pb.UpdatedBy, &details,
	)

	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "Query Raw get by code sales: %v", err)
	}

	if err != nil {
		return status.Errorf(codes.Internal, "Query Raw get by code sales: %v", err)
	}

	if companyID != ctx.Value(app.Ctx("companyID")).(string) {
		return status.Error(codes.Unauthenticated, "its not your company")
	}

	u.Pb.SalesDate = dateSales.String()
	u.Pb.CreatedAt = createdAt.String()
	u.Pb.UpdatedAt = updatedAt.String()

	detailSales := []struct {
		ID             string
		SalesID        string
		ProductID      string
		Price          float64
		DiscAmount     float64
		DiscPercentage float32
		Quantity       int
		TotalPrice     float64
	}{}
	err = json.Unmarshal([]byte(details), &detailSales)
	if err != nil {
		return status.Errorf(codes.Internal, "unmarshal access: %v", err)
	}

	for _, detail := range detailSales {
		u.Pb.Details = append(u.Pb.Details, &sales.SalesDetail{
			Id:             detail.ID,
			ProductId:      detail.ProductID,
			SalesId:        detail.SalesID,
			Price:          detail.Price,
			DiscAmount:     detail.DiscAmount,
			DiscPercentage: detail.DiscPercentage,
			TotalPrice:     detail.TotalPrice,
		})
	}

	return nil
}

func (u *Sales) GetByCode(ctx context.Context, db *sql.DB) error {
	query := `
		SELECT id, branch_id, branch_name, customer_id, salesman_id, code, sales_date, remark, 
			price, additional_disc_amount, additional_disc_percentage, total_price, created_at, created_by, updated_at, updated_by 
		FROM sales WHERE sales.code = $1 AND sales.company_id = $2
	`

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare statement Get by code sales: %v", err)
	}
	defer stmt.Close()

	var dateSales, createdAt, updatedAt time.Time
	err = stmt.QueryRowContext(ctx, u.Pb.GetCode(), ctx.Value(app.Ctx("companyID")).(string)).Scan(
		&u.Pb.Id, &u.Pb.BranchId, &u.Pb.BranchName, &u.Pb.GetCustomer().Id, &u.Pb.GetSalesman().Id,
		&u.Pb.Code, &dateSales, &u.Pb.Remark,
		&u.Pb.Price, &u.Pb.AdditionalDiscAmount, &u.Pb.AdditionalDiscPercentage, &u.Pb.TotalPrice,
		&createdAt, &u.Pb.CreatedBy, &updatedAt, &u.Pb.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "Query Raw get by code sales: %v", err)
	}

	if err != nil {
		return status.Errorf(codes.Internal, "Query Raw get by code sales: %v", err)
	}

	u.Pb.SalesDate = dateSales.String()
	u.Pb.CreatedAt = createdAt.String()
	u.Pb.UpdatedAt = updatedAt.String()

	return nil
}

func (u *Sales) getCode(ctx context.Context, tx *sql.Tx) (string, error) {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sales 
			WHERE company_id = $1 AND to_char(created_at, 'YYYY-mm') = to_char(now(), 'YYYY-mm')`,
		ctx.Value(app.Ctx("companyID")).(string)).Scan(&count)

	if err != nil {
		return "", status.Error(codes.Internal, err.Error())
	}

	return fmt.Sprintf("DO%d%d%d",
		time.Now().UTC().Year(),
		int(time.Now().UTC().Month()),
		(count + 1)), nil
}

func (u *Sales) Create(ctx context.Context, tx *sql.Tx) error {
	u.Pb.Id = uuid.New().String()
	now := time.Now().UTC()
	u.Pb.CreatedBy = ctx.Value(app.Ctx("userID")).(string)
	u.Pb.UpdatedBy = ctx.Value(app.Ctx("userID")).(string)
	dateSales, err := time.Parse("2006-01-02T15:04:05.000Z", u.Pb.GetSalesDate())
	if err != nil {
		return status.Errorf(codes.Internal, "convert Date: %v", err)
	}

	u.Pb.Code, err = u.getCode(ctx, tx)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO sales (id, company_id, branch_id, branch_name, customer_id, salesman_id, code, sales_date, remark, price, additional_disc_amount, additional_disc_percentage, total_price, created_at, created_by, updated_at, updated_by) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare insert sales: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetId(),
		ctx.Value(app.Ctx("companyID")).(string),
		u.Pb.GetBranchId(),
		u.Pb.GetBranchName(),
		u.Pb.GetCustomer().GetId(),
		u.Pb.GetSalesman().GetId(),
		u.Pb.GetCode(),
		dateSales,
		u.Pb.GetRemark(),
		u.Pb.GetPrice(),
		u.Pb.GetAdditionalDiscAmount(),
		u.Pb.GetAdditionalDiscPercentage(),
		u.Pb.GetTotalPrice(),
		now,
		u.Pb.GetCreatedBy(),
		now,
		u.Pb.GetUpdatedBy(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec insert sales: %v", err)
	}

	u.Pb.CreatedAt = now.String()
	u.Pb.UpdatedAt = u.Pb.CreatedAt

	for _, detail := range u.Pb.GetDetails() {
		salesDetailModel := SalesDetail{}
		salesDetailModel.Pb = sales.SalesDetail{
			SalesId:        u.Pb.GetId(),
			ProductId:      detail.GetProductId(),
			Price:          detail.GetPrice(),
			DiscAmount:     detail.GetDiscAmount(),
			DiscPercentage: detail.GetDiscPercentage(),
			Quantity:       detail.GetQuantity(),
			TotalPrice:     detail.GetTotalPrice(),
		}
		salesDetailModel.PbSales = sales.Sales{
			Id:                       u.Pb.Id,
			BranchId:                 u.Pb.BranchId,
			BranchName:               u.Pb.BranchName,
			Customer:                 u.Pb.GetCustomer(),
			Salesman:                 u.Pb.GetSalesman(),
			Code:                     u.Pb.Code,
			SalesDate:                u.Pb.SalesDate,
			Remark:                   u.Pb.Remark,
			Price:                    u.Pb.Price,
			AdditionalDiscAmount:     u.Pb.AdditionalDiscAmount,
			AdditionalDiscPercentage: u.Pb.AdditionalDiscPercentage,
			TotalPrice:               u.Pb.TotalPrice,
			CreatedAt:                u.Pb.CreatedAt,
			CreatedBy:                u.Pb.CreatedBy,
			UpdatedAt:                u.Pb.UpdatedAt,
			UpdatedBy:                u.Pb.UpdatedBy,
		}
		err = salesDetailModel.Create(ctx, tx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *Sales) Update(ctx context.Context, tx *sql.Tx) error {
	now := time.Now().UTC()
	u.Pb.UpdatedBy = ctx.Value(app.Ctx("userID")).(string)
	dateSales, err := time.Parse("2006-01-02T15:04:05.000Z", u.Pb.GetSalesDate())
	if err != nil {
		return status.Errorf(codes.Internal, "convert sales date: %v", err)
	}

	query := `
		UPDATE sales SET
		customer_id = $1,
		salesman_id = $2,
		sales_date = $3,
		remark = $4, 
		price = $5,
		additional_disc_amount = $6,
		additional_disc_percentage = $7,
		total_price = $8,
		updated_at = $9, 
		updated_by= $10
		WHERE id = $11
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare update sales: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetCustomer().GetId(),
		u.Pb.GetSalesman().GetId(),
		dateSales,
		u.Pb.GetRemark(),
		u.Pb.GetPrice(),
		u.Pb.GetAdditionalDiscAmount(),
		u.Pb.GetAdditionalDiscPercentage(),
		u.Pb.GetTotalPrice(),
		now,
		u.Pb.GetUpdatedBy(),
		u.Pb.GetId(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec update sales: %v", err)
	}

	u.Pb.UpdatedAt = now.String()

	return nil
}

func (u *Sales) ListQuery(ctx context.Context, db *sql.DB, in *sales.ListSalesRequest) (string, []interface{}, *sales.SalesPaginationResponse, error) {
	var paginationResponse sales.SalesPaginationResponse
	query := `
		SELECT id, company_id, branch_id, branch_name, customer_id, salesman_id, code, sales_date, remark, 
			price, additional_disc_amount, additional_disc_percentage, total_price, 
			created_at, created_by, updated_at, updated_by 
		FROM sales
	`

	where := []string{"company_id = $1"}
	paramQueries := []interface{}{ctx.Value(app.Ctx("companyID")).(string)}

	if len(in.GetBranchId()) > 0 {
		paramQueries = append(paramQueries, in.GetBranchId())
		where = append(where, fmt.Sprintf(`branch_id = $%d`, len(paramQueries)))
	}

	if len(in.GetCustomerId()) > 0 {
		paramQueries = append(paramQueries, in.GetCustomerId())
		where = append(where, fmt.Sprintf(`customer_id = $%d`, len(paramQueries)))
	}

	if len(in.GetSalesmanId()) > 0 {
		paramQueries = append(paramQueries, in.GetSalesmanId())
		where = append(where, fmt.Sprintf(`salesman_id = $%d`, len(paramQueries)))
	}

	if len(in.GetPagination().GetSearch()) > 0 {
		paramQueries = append(paramQueries, in.GetPagination().GetSearch())
		where = append(where, fmt.Sprintf(`(code ILIKE $%d OR remark ILIKE $%d)`, len(paramQueries), len(paramQueries)))
	}

	{
		qCount := `SELECT COUNT(*) FROM sales`
		if len(where) > 0 {
			qCount += " WHERE " + strings.Join(where, " AND ")
		}
		var count int
		err := db.QueryRowContext(ctx, qCount, paramQueries...).Scan(&count)
		if err != nil && err != sql.ErrNoRows {
			return query, paramQueries, &paginationResponse, status.Error(codes.Internal, err.Error())
		}

		paginationResponse.Count = uint32(count)
	}

	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}

	if len(in.GetPagination().GetOrderBy()) == 0 || !(in.GetPagination().GetOrderBy() == "code") {
		if in.GetPagination() == nil {
			in.Pagination = &sales.Pagination{OrderBy: "created_at"}
		} else {
			in.GetPagination().OrderBy = "created_at"
		}
	}

	query += ` ORDER BY ` + in.GetPagination().GetOrderBy() + ` ` + in.GetPagination().GetSort().String()

	if in.GetPagination().GetLimit() > 0 {
		query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, (len(paramQueries) + 1), (len(paramQueries) + 2))
		paramQueries = append(paramQueries, in.GetPagination().GetLimit(), in.GetPagination().GetOffset())
	}

	return query, paramQueries, &paginationResponse, nil
}

func (u *Sales) OutstandingDetail(ctx context.Context, db *sql.DB, salesReturnId *string) ([]*sales.SalesDetail, error) {
	var list []*sales.SalesDetail

	queryReturn := `
		SELECT sales_return_details.product_id, SUM(sales_return_details.quantity) return_quantity  
		FROM sales_returns
		JOIN sales_return_details ON sales_returns.id = sales_return_details.sales_return_id
		WHERE sales_returns.sales_id = $1 
	`
	if salesReturnId != nil {
		queryReturn += ` AND sales_returns.id != $3`
	}

	queryReturn += ` GROUP BY sales_return_details.product_id`

	query := `
		SELECT sales_details.product_id, (sales_details.quantity - sales_returns.return_quantity) quantity 
		FROM sales_details 
		JOIN sales ON sales_details.sales_id = sales.id
		JOIN (
			` + queryReturn + `
		) AS sales_returns ON sales_details.product_id = sales_returns.product_id
		WHERE sales_details.sales_id = $1 
			AND (sales_details.quantity - sales_returns.return_quantity) > 0		
			AND sales.company_id = $2
	`

	params := []interface{}{
		u.Pb.Id,
		ctx.Value(app.Ctx("companyID")).(string),
	}

	if salesReturnId != nil {
		params = append(params, *salesReturnId)
	}

	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		return list, status.Error(codes.Internal, err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var pbSalesDetail sales.SalesDetail
		err = rows.Scan(&pbSalesDetail.ProductId, &pbSalesDetail.Quantity)
		if err != nil {
			return list, status.Errorf(codes.Internal, "scan data: %v", err)
		}

		list = append(list, &pbSalesDetail)
	}

	if rows.Err() == nil {
		return list, status.Errorf(codes.Internal, "rows error: %v", err)
	}

	return list, nil
}

func (u *Sales) GetReturnAdditionalDisc(ctx context.Context, db *sql.DB) (float64, error) {
	var returnAdditionalDisc float64
	query := `
		SELECT SUM(sales_returns.additional_disc_amount) return_additional_disc
		FROM sales
		JOIN sales_returns ON sales.id = sales_returns.sales_id
		WHERE sales.id = $1
		GROUP BY sales.id
	`
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return returnAdditionalDisc, status.Errorf(codes.Internal, "Prepare statement Get sales: %v", err)
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(ctx, u.Pb.GetId()).Scan(&returnAdditionalDisc)

	if err == sql.ErrNoRows {
		return returnAdditionalDisc, nil
	}

	if err != nil {
		return returnAdditionalDisc, status.Errorf(codes.Internal, "Query Raw get returnAdditionalDisc: %v", err)
	}

	return returnAdditionalDisc, nil
}
