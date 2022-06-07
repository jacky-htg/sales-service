package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"sales/internal/pkg/app"
	"sales/pb/sales"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Salesman struct {
	Pb sales.Salesman
}

func (u *Salesman) Get(ctx context.Context, db *sql.DB) error {
	query := `
		SELECT id, company_id, code, name, email, address, phone, created_at, created_by, updated_at, updated_by 
		FROM salesman WHERE id = $1 AND company_id = $2
	`

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare statement Get salesman: %v", err)
	}
	defer stmt.Close()

	var companyID string
	var createdAt, updatedAt time.Time
	err = stmt.QueryRowContext(ctx, u.Pb.GetId(), ctx.Value(app.Ctx("companyID")).(string)).Scan(
		&u.Pb.Id, &companyID, &u.Pb.Code, &u.Pb.Name, &u.Pb.Email, &u.Pb.Address, &u.Pb.Phone, &createdAt, &u.Pb.CreatedBy, &updatedAt, &u.Pb.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "Query Raw get salesman: %v", err)
	}

	if err != nil {
		return status.Errorf(codes.Internal, "Query Raw get salesman: %v", err)
	}

	if companyID != ctx.Value(app.Ctx("companyID")).(string) {
		return status.Error(codes.Unauthenticated, "its not your company data")
	}

	u.Pb.CreatedAt = createdAt.String()
	u.Pb.UpdatedAt = updatedAt.String()

	return nil
}

func (u *Salesman) GetByCode(ctx context.Context, db *sql.DB) error {
	query := `
		SELECT id, company_id, code, name, email, address, phone, created_at, created_by, updated_at, updated_by 
		FROM salesman WHERE company_id = $1 AND code = $2
	`

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare statement Get salesman by code: %v", err)
	}
	defer stmt.Close()

	var companyID string
	var createdAt, updatedAt time.Time
	err = stmt.QueryRowContext(ctx, ctx.Value(app.Ctx("companyID")).(string), u.Pb.GetCode()).Scan(
		&u.Pb.Id, &companyID, &u.Pb.Code, &u.Pb.Name, &u.Pb.Email, &u.Pb.Address, &u.Pb.Phone, &createdAt, &u.Pb.CreatedBy, &updatedAt, &u.Pb.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "Query Raw get salesman by code: %v", err)
	}

	if err != nil {
		return status.Errorf(codes.Internal, "Query Raw get salesman by code: %v", err)
	}

	u.Pb.CreatedAt = createdAt.String()
	u.Pb.UpdatedAt = updatedAt.String()

	return nil
}

func (u *Salesman) Create(ctx context.Context, db *sql.DB) error {
	u.Pb.Id = uuid.New().String()
	now := time.Now().UTC()
	u.Pb.CreatedBy = ctx.Value(app.Ctx("userID")).(string)
	u.Pb.UpdatedBy = ctx.Value(app.Ctx("userID")).(string)

	query := `
		INSERT INTO salesman (id, company_id, code, name, email, address, phone, created_at, created_by, updated_at, updated_by) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare insert salesman: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetId(),
		ctx.Value(app.Ctx("companyID")).(string),
		u.Pb.GetCode(),
		u.Pb.GetName(),
		u.Pb.GetEmail(),
		u.Pb.GetAddress(),
		u.Pb.GetPhone(),
		now,
		u.Pb.GetCreatedBy(),
		now,
		u.Pb.GetUpdatedBy(),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec insert salesman: %v", err)
	}

	u.Pb.CreatedAt = now.String()
	u.Pb.UpdatedAt = u.Pb.CreatedAt

	return nil
}

func (u *Salesman) Update(ctx context.Context, db *sql.DB) error {
	now := time.Now().UTC()
	u.Pb.UpdatedBy = ctx.Value(app.Ctx("userID")).(string)

	query := `
		UPDATE salesman SET
		name = $1,
		email = $2,
		address = $3,
		phone = $4, 
		updated_at = $5, 
		updated_by= $6
		WHERE id = $7 AND company_id = $8
	`
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare update salesman: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		u.Pb.GetName(),
		u.Pb.GetEmail(),
		u.Pb.GetAddress(),
		u.Pb.GetPhone(),
		now,
		u.Pb.GetUpdatedBy(),
		u.Pb.GetId(),
		ctx.Value(app.Ctx("companyID")).(string),
	)
	if err != nil {
		return status.Errorf(codes.Internal, "Exec update salesman: %v", err)
	}

	u.Pb.UpdatedAt = now.String()

	return nil
}

func (u *Salesman) Delete(ctx context.Context, db *sql.DB) error {
	stmt, err := db.PrepareContext(ctx, `DELETE FROM salesman WHERE company_id = $1 AND id = $2`)
	if err != nil {
		return status.Errorf(codes.Internal, "Prepare delete salesman: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, ctx.Value(app.Ctx("companyID")).(string), u.Pb.GetId())
	if err != nil {
		return status.Errorf(codes.Internal, "Exec delete salesman: %v", err)
	}

	return nil
}

func (u *Salesman) ListQuery(ctx context.Context, db *sql.DB, in *sales.Pagination) (string, []interface{}, *sales.SalesmanPaginationResponse, error) {
	var paginationResponse sales.SalesmanPaginationResponse
	query := `SELECT id, company_id, code, name, email, address, phone, created_at, created_by, updated_at, updated_by FROM salesman`
	where := []string{"company_id = $1"}
	paramQueries := []interface{}{ctx.Value(app.Ctx("companyID")).(string)}

	if len(in.GetSearch()) > 0 {
		paramQueries = append(paramQueries, in.GetSearch())
		where = append(where, fmt.Sprintf(`(name ILIKE $%d OR code ILIKE $%d OR address ILIKE $%d OR phone ILIKE $%d)`, len(paramQueries), len(paramQueries), len(paramQueries), len(paramQueries)))
	}

	{
		qCount := `SELECT COUNT(*) FROM salesman`
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

	if len(in.GetOrderBy()) == 0 || !(in.GetOrderBy() == "name" || in.GetOrderBy() == "code" || in.GetOrderBy() == "email") {
		if in == nil {
			in = &sales.Pagination{OrderBy: "created_at"}
		} else {
			in.OrderBy = "created_at"
		}
	}

	query += ` ORDER BY ` + in.GetOrderBy() + ` ` + in.GetSort().String()

	if in.GetLimit() > 0 {
		query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, (len(paramQueries) + 1), (len(paramQueries) + 2))
		paramQueries = append(paramQueries, in.GetLimit(), in.GetOffset())
	}

	return query, paramQueries, &paginationResponse, nil
}
