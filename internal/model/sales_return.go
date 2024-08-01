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
	"github.com/jacky-htg/erp-pkg/util"
	"github.com/jacky-htg/erp-proto/go/pb/sales"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SalesReturn struct {
	Pb sales.SalesReturn
}

func (u *SalesReturn) Get(ctx context.Context, db *sql.DB) error {
	query := `
		SELECT purchase_returns.id, purchase_returns.company_id, purchase_returns.branch_id, 
			purchase_returns.branch_name, purchase_returns.purchase_id, purchase_returns.code, 
			purchase_returns.return_date, purchase_returns.remark, 
			purchase_returns.price, purchase_returns.additional_disc_amount, purchase_returns.additional_disc_percentage, purchase_returns.total_price,
			purchase_returns.created_at, purchase_returns.created_by, purchase_returns.updated_at, purchase_returns.updated_by,
		json_agg(DISTINCT jsonb_build_object(
			'id', purchase_return_details.id,
			'purchase_return_id', purchase_return_details.purchase_return_id,
			'product_id', purchase_return_details.product_id,
			'quantity', purchase_return_details.quantity,
			'price', purchase_return_details.price,
			'disc_amount', purchase_return_details.disc_amount,
			'disc_percentage', purchase_return_details.disc_percentage,
			'total_price', purchase_return_details.total_price
		)) as details
		FROM purchase_returns 
		JOIN purchase_return_details ON purchase_returns.id = purchase_return_details.purchase_return_id
		WHERE purchase_returns.id = $1
	`

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare statement Get sales return: %v", err)
	}
	defer stmt.Close()

	var dateReturn, createdAt, updatedAt time.Time
	var companyID, details string
	err = stmt.QueryRowContext(ctx, u.Pb.GetId()).Scan(
		&u.Pb.Id, &companyID, &u.Pb.BranchId, &u.Pb.BranchName,
		&u.Pb.Sales.Id, &u.Pb.Code, &dateReturn, &u.Pb.Remark,
		&u.Pb.Price, &u.Pb.AdditionalDiscAmount, &u.Pb.AdditionalDiscPercentage, &u.Pb.TotalPrice,
		&createdAt, &u.Pb.CreatedBy, &updatedAt, &u.Pb.UpdatedBy, &details,
	)

	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "Query Raw get by code sales return: %v", err)
	}

	if err != nil {
		return status.Errorf(codes.Internal, "Query Raw get by code sales return: %v", err)
	}

	if companyID != ctx.Value(app.Ctx("companyID")).(string) {
		return status.Error(codes.Unauthenticated, "its not your company")
	}

	u.Pb.ReturnDate = dateReturn.String()
	u.Pb.CreatedAt = createdAt.String()
	u.Pb.UpdatedAt = updatedAt.String()

	detailSalesReturns := []struct {
		ID             string
		SalesReturnID  string
		ProductID      string
		Quantity       int32
		Price          float64
		DiscAmount     float64
		DiscPercentage float32
		TotalPrice     float64
	}{}
	err = json.Unmarshal([]byte(details), &detailSalesReturns)
	if err != nil {
		return status.Errorf(codes.Internal, "unmarshal detailSalesReturns: %v", err)
	}

	for _, detail := range detailSalesReturns {
		u.Pb.Details = append(u.Pb.Details, &sales.SalesReturnDetail{
			Id:             detail.ID,
			ProductId:      detail.ProductID,
			Quantity:       detail.Quantity,
			Price:          detail.Price,
			DiscAmount:     detail.DiscAmount,
			DiscPercentage: detail.DiscPercentage,
			TotalPrice:     detail.TotalPrice,
			SalesReturnId:  detail.SalesReturnID,
		})
	}

	return nil
}

func (u *SalesReturn) HasReturn(ctx context.Context, db *sql.DB) (bool, error) {
	query := `
		SELECT purchase_returns.id
		FROM purchase_returns 
		WHERE purchase_returns.purchase_id = $1 
		LIMIT 0,1
	`

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return false, status.Errorf(codes.Internal, "Prepare statement 'Has Sales Return': %v", err)
	}
	defer stmt.Close()

	var myId string
	err = stmt.QueryRowContext(ctx, u.Pb.Sales.GetId()).Scan(&myId)

	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, status.Errorf(codes.Internal, "Query Raw get by code sales return: %v", err)
	}

	return true, nil
}

func (u *SalesReturn) Create(ctx context.Context, tx *sql.Tx) error {
	u.Pb.Id = uuid.New().String()
	now := time.Now().UTC()
	u.Pb.CreatedBy = ctx.Value(app.Ctx("userID")).(string)
	u.Pb.UpdatedBy = ctx.Value(app.Ctx("userID")).(string)
	dateReturn, err := time.Parse("2006-01-02T15:04:05.000Z", u.Pb.GetReturnDate())
	if err != nil {
		return status.Errorf(codes.Internal, "convert Date: %v", err)
	}

	u.Pb.Code, err = util.GetCode(ctx, tx, "purchase_returns", "DR")
	if err != nil {
		return err
	}

	query := `
		INSERT INTO purchase_returns (
			id, company_id, branch_id, branch_name, purchase_id, code, return_date, remark, 
			price, additional_disc_amount, additional_disc_percentage, total_price, 
			created_at, created_by, updated_at, updated_by
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare insert sales return: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetId(),
		ctx.Value(app.Ctx("companyID")).(string),
		u.Pb.GetBranchId(),
		u.Pb.GetBranchName(),
		u.Pb.GetSales().GetId(),
		u.Pb.GetCode(),
		dateReturn,
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
		return status.Errorf(codes.Internal, "Exec insert sales return: %v", err)
	}

	u.Pb.CreatedAt = now.String()
	u.Pb.UpdatedAt = u.Pb.CreatedAt

	for _, detail := range u.Pb.GetDetails() {
		purchaseReturnDetailModel := SalesReturnDetail{}
		purchaseReturnDetailModel.Pb = sales.SalesReturnDetail{
			SalesReturnId:  u.Pb.GetId(),
			ProductId:      detail.ProductId,
			Quantity:       detail.Quantity,
			Price:          detail.Price,
			DiscAmount:     detail.DiscAmount,
			DiscPercentage: detail.DiscPercentage,
			TotalPrice:     detail.TotalPrice,
		}
		purchaseReturnDetailModel.PbSalesReturn = sales.SalesReturn{
			Id:                       u.Pb.Id,
			BranchId:                 u.Pb.BranchId,
			BranchName:               u.Pb.BranchName,
			Sales:                    u.Pb.Sales,
			Code:                     u.Pb.Code,
			ReturnDate:               u.Pb.ReturnDate,
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
		err = purchaseReturnDetailModel.Create(ctx, tx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *SalesReturn) Update(ctx context.Context, tx *sql.Tx) error {
	now := time.Now().UTC()
	u.Pb.UpdatedBy = ctx.Value(app.Ctx("userID")).(string)
	dateReturn, err := time.Parse("2006-01-02T15:04:05.000Z", u.Pb.GetReturnDate())
	if err != nil {
		return status.Errorf(codes.Internal, "convert sales return date: %v", err)
	}

	query := `
		UPDATE purchase_returns SET
		return_date = $1,
		remark = $2,
		price = $3,
		additional_disc_amount = $4,
		additional_disc_percentage = $5,
		total_price = $6,
		updated_at = $7, 
		updated_by= $8
		WHERE id = $9 AND purchase_id = $10
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare update sales return: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		dateReturn,
		u.Pb.GetRemark(),
		u.Pb.Price,
		u.Pb.AdditionalDiscAmount,
		u.Pb.AdditionalDiscPercentage,
		u.Pb.TotalPrice,
		now,
		u.Pb.GetUpdatedBy(),
		u.Pb.GetId(),
		u.Pb.GetSales().GetId(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec update sales return: %v", err)
	}

	u.Pb.UpdatedAt = now.String()

	return nil
}

// ListQuery builder
func (u *SalesReturn) ListQuery(ctx context.Context, db *sql.DB, in *sales.ListSalesReturnRequest) (string, []interface{}, *sales.SalesReturnPaginationResponse, error) {
	var paginationResponse sales.SalesReturnPaginationResponse
	query := `SELECT id, company_id, branch_id, branch_name, sales_id, code, return_date, remark, price, additional_disc_amount, additional_disc_percentage, total_price,  created_at, created_by, updated_at, updated_by FROM purchase_returns`

	where := []string{"company_id = $1"}
	paramQueries := []interface{}{ctx.Value(app.Ctx("companyID")).(string)}

	if len(in.GetBranchId()) > 0 {
		paramQueries = append(paramQueries, in.GetBranchId())
		where = append(where, fmt.Sprintf(`branch_id = $%d`, len(paramQueries)))
	}

	if len(in.GetSalesId()) > 0 {
		paramQueries = append(paramQueries, in.GetSalesId())
		where = append(where, fmt.Sprintf(`sales_id = $%d`, len(paramQueries)))
	}

	if len(in.GetPagination().GetSearch()) > 0 {
		paramQueries = append(paramQueries, in.GetPagination().GetSearch())
		where = append(where, fmt.Sprintf(`(code ILIKE $%d OR remark ILIKE $%d)`, len(paramQueries), len(paramQueries)))
	}

	{
		qCount := `SELECT COUNT(*) FROM sales_returns`
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
