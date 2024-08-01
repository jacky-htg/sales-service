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

type SalesReturnDetail struct {
	Pb            sales.SalesReturnDetail
	PbSalesReturn sales.SalesReturn
}

func (u *SalesReturnDetail) Get(ctx context.Context, tx *sql.Tx) error {
	query := `
		SELECT sales_return_details.id, sales_returns.company_id, sales_return_details.sales_return_id, sales_return_details.product_id, sales_return_details.quantity,
			sales_return_details.price, sales_return_details.disc_amount, sales_return_details.disc_percentage, sales_return_details.total_price
		FROM sales_return_details 
		JOIN sales_returns ON sales_return_details.sales_return_id = sales_returns.id
		WHERE sales_return_details.id = $1 AND sales_return_details.sales_return_id = $2
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare statement Get sales return detail: %v", err)
	}
	defer stmt.Close()

	var companyID string
	err = stmt.QueryRowContext(ctx, u.Pb.GetId(), u.Pb.GetSalesReturnId()).Scan(
		&u.Pb.Id, &companyID, &u.Pb.SalesReturnId, &u.Pb.ProductId, &u.Pb.Quantity,
		&u.Pb.Price, &u.Pb.DiscAmount, &u.Pb.DiscPercentage, &u.Pb.TotalPrice,
	)

	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "Query Raw get by code sales return detail: %v", err)
	}

	if err != nil {
		return status.Errorf(codes.Internal, "Query Raw get by code sales return detail: %v", err)
	}

	if companyID != ctx.Value(app.Ctx("companyID")).(string) {
		return status.Error(codes.Unauthenticated, "its not your company")
	}

	return nil
}

func (u *SalesReturnDetail) Create(ctx context.Context, tx *sql.Tx) error {
	u.Pb.Id = uuid.New().String()
	query := `
		INSERT INTO sales_return_details (id, sales_return_id, product_id, quantity, price, disc_amount, disc_percentage, total_price) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare insert sales return detail: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetId(),
		u.Pb.GetSalesReturnId(),
		u.Pb.GetProductId(),
		u.Pb.Quantity,
		u.Pb.Price,
		u.Pb.DiscAmount,
		u.Pb.DiscPercentage,
		u.Pb.TotalPrice,
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec insert sales return detail: %v", err)
	}

	return nil
}

func (u *SalesReturnDetail) Update(ctx context.Context, tx *sql.Tx) error {
	query := `
		UPDATE sales_return_details SET
		quantity = $1,
		total_price = $2
		WHERE id = $3
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare update sales return detail: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetQuantity(),
		u.Pb.TotalPrice,
		u.Pb.GetId(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec update sales return detail: %v", err)
	}

	return nil
}

func (u *SalesReturnDetail) Delete(ctx context.Context, tx *sql.Tx) error {
	stmt, err := tx.PrepareContext(ctx, `DELETE FROM sales_return_details WHERE id = $1 AND sales_return_id = $2`)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare delete sales return detail: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, u.Pb.GetId(), u.Pb.GetSalesReturnId())
	if err != nil {
		return status.Errorf(codes.Internal, "Exec delete sales return detail: %v", err)
	}

	return nil
}
