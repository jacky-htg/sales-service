package model

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jacky-htg/erp-pkg/app"
	"github.com/jacky-htg/erp-proto/go/pb/sales"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SalesDetail struct {
	Pb      sales.SalesDetail
	PbSales sales.Sales
}

func (u *SalesDetail) Get(ctx context.Context, tx *sql.Tx) error {
	query := `
		SELECT purchase_details.id, sales.company_id, purchase_details.purchase_id, purchase_details.product_id, 
			purchase_details.price, purchase_details.disc_amount, purchase_details.disc_percentage, purchase_details.quantity, purchase_details.total_price
		FROM purchase_details 
		JOIN sales ON purchase_details.purchase_id = sales.id
		WHERE purchase_details.id = $1 AND purchase_details.purchase_id = $2
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare statement Get sales detail: %v", err)
	}
	defer stmt.Close()

	var companyID string
	err = stmt.QueryRowContext(ctx, u.Pb.GetId(), u.Pb.GetSalesId()).Scan(
		&u.Pb.Id, &companyID, &u.Pb.SalesId, &u.Pb.ProductId, &u.Pb.Price, &u.Pb.DiscAmount, &u.Pb.DiscPercentage, &u.Pb.Quantity, &u.Pb.TotalPrice,
	)

	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "Query Raw get by code sales detail: %v", err)
	}

	if err != nil {
		return status.Errorf(codes.Internal, "Query Raw get by code sales detail: %v", err)
	}

	if companyID != ctx.Value(app.Ctx("companyID")).(string) {
		return status.Error(codes.Unauthenticated, "its not your company")
	}

	return nil
}

func (u *SalesDetail) Create(ctx context.Context, tx *sql.Tx) error {
	u.Pb.Id = uuid.New().String()
	query := `
		INSERT INTO purchase_details (id, purchase_id, product_id, price, disc_amount, disc_percentage, quantity, total_price) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare insert sales detail: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetId(),
		u.Pb.GetSalesId(),
		u.Pb.GetProductId(),
		u.Pb.GetPrice(),
		u.Pb.GetDiscAmount(),
		u.Pb.GetDiscPercentage(),
		u.Pb.GetQuantity(),
		u.Pb.GetTotalPrice(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec insert sales detail: %v", err)
	}

	return nil
}

func (u *SalesDetail) Update(ctx context.Context, tx *sql.Tx) error {
	query := `
		UPDATE purchase_details
		SET price = $1,
			disc_amount = $2,
			disc_percentage = $3,
			quantity = $4,
			total_price = $5
		WHERE id = $6
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare update sales detail: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetPrice(),
		u.Pb.GetDiscAmount(),
		u.Pb.GetDiscPercentage(),
		u.Pb.GetQuantity(),
		u.Pb.GetTotalPrice(),
		u.Pb.GetId(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec insert sales detail: %v", err)
	}

	return nil
}

func (u *SalesDetail) Delete(ctx context.Context, tx *sql.Tx) error {
	stmt, err := tx.PrepareContext(ctx, `DELETE FROM purchase_details WHERE id = $1 AND purchase_id = $2`)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare delete sales detail: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, u.Pb.GetId(), u.Pb.GetSalesId())
	if err != nil {
		return status.Errorf(codes.Internal, "Exec delete sales detail: %v", err)
	}

	return nil
}

func (u *SalesDetail) SetPbFromPointer(data *sales.SalesDetail) {
	u.Pb = sales.SalesDetail{
		Id:             data.GetId(),
		SalesId:        data.GetSalesId(),
		ProductId:      data.GetProductId(),
		ProductCode:    data.GetProductCode(),
		ProductName:    data.GetProductName(),
		Price:          data.GetPrice(),
		DiscAmount:     data.GetDiscAmount(),
		DiscPercentage: data.GetDiscPercentage(),
		Quantity:       data.GetQuantity(),
		TotalPrice:     data.GetTotalPrice(),
	}
}
