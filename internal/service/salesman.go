package service

import (
	"context"
	"database/sql"
	"time"

	"sales/internal/model"
	"sales/internal/pkg/app"
	"sales/pb/sales"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Salesman struct {
	Db *sql.DB
	sales.UnimplementedSalesmanServiceServer
}

func (u *Salesman) Create(ctx context.Context, in *sales.Salesman) (*sales.Salesman, error) {
	var salesmanModel model.Salesman
	var err error

	if len(in.GetName()) == 0 {
		return &salesmanModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid name")
	}

	if len(in.GetEmail()) == 0 {
		return &salesmanModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid email")
	}

	if len(in.GetAddress()) == 0 {
		return &salesmanModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid address")
	}

	if len(in.GetPhone()) == 0 {
		return &salesmanModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid phone")
	}

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesmanModel.Pb, err
	}

	// code validation
	{
		if len(in.GetCode()) == 0 {
			return &salesmanModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid code")
		}

		salesmanModel = model.Salesman{}
		salesmanModel.Pb.Code = in.GetCode()
		err = salesmanModel.GetByCode(ctx, u.Db)
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() != codes.NotFound {
				return &salesmanModel.Pb, err
			}
		}

		if len(salesmanModel.Pb.GetId()) > 0 {
			return &salesmanModel.Pb, status.Error(codes.AlreadyExists, "code must be unique")
		}
	}

	salesmanModel.Pb = sales.Salesman{
		Code:    in.GetCode(),
		Name:    in.GetName(),
		Email:   in.GetEmail(),
		Address: in.GetAddress(),
		Phone:   in.GetPhone(),
	}
	err = salesmanModel.Create(ctx, u.Db)
	if err != nil {
		return &salesmanModel.Pb, err
	}

	return &salesmanModel.Pb, nil
}

func (u *Salesman) Update(ctx context.Context, in *sales.Salesman) (*sales.Salesman, error) {
	var salesmanModel model.Salesman
	var err error

	if len(in.GetId()) == 0 {
		return &salesmanModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	salesmanModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesmanModel.Pb, err
	}

	err = salesmanModel.Get(ctx, u.Db)
	if err != nil {
		return &salesmanModel.Pb, err
	}

	if len(in.GetName()) > 0 {
		salesmanModel.Pb.Name = in.GetName()
	}

	if len(in.GetEmail()) > 0 {
		salesmanModel.Pb.Email = in.GetEmail()
	}

	if len(in.GetAddress()) > 0 {
		salesmanModel.Pb.Address = in.GetAddress()
	}

	if len(in.GetPhone()) > 0 {
		salesmanModel.Pb.Phone = in.GetPhone()
	}

	err = salesmanModel.Update(ctx, u.Db)
	if err != nil {
		return &salesmanModel.Pb, err
	}

	return &salesmanModel.Pb, nil
}

func (u *Salesman) View(ctx context.Context, in *sales.Id) (*sales.Salesman, error) {
	var salesmanModel model.Salesman
	var err error

	if len(in.GetId()) == 0 {
		return &salesmanModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	salesmanModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesmanModel.Pb, err
	}

	err = salesmanModel.Get(ctx, u.Db)
	if err != nil {
		return &salesmanModel.Pb, err
	}

	return &salesmanModel.Pb, nil
}

func (u *Salesman) Delete(ctx context.Context, in *sales.Id) (*sales.MyBoolean, error) {
	var output sales.MyBoolean
	output.Boolean = false

	var salesmanModel model.Salesman
	var err error

	if len(in.GetId()) == 0 {
		return &output, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	salesmanModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &output, err
	}

	err = salesmanModel.Get(ctx, u.Db)
	if err != nil {
		return &output, err
	}

	err = salesmanModel.Delete(ctx, u.Db)
	if err != nil {
		return &output, err
	}

	output.Boolean = true
	return &output, nil
}

func (u *Salesman) List(in *sales.Pagination, stream sales.SalesmanService_SalesmanListServer) error {
	ctx := stream.Context()
	ctx, err := app.GetMetadata(ctx)
	if err != nil {
		return err
	}

	var salesmanModel model.Salesman
	query, paramQueries, paginationResponse, err := salesmanModel.ListQuery(ctx, u.Db, in)

	rows, err := u.Db.QueryContext(ctx, query, paramQueries...)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer rows.Close()
	paginationResponse.Pagination = in

	for rows.Next() {
		err := app.ContextError(ctx)
		if err != nil {
			return err
		}

		var pbSalesman sales.Salesman
		var companyID string
		var createdAt, updatedAt time.Time
		err = rows.Scan(&pbSalesman.Id, &companyID, &pbSalesman.Code, &pbSalesman.Name, &pbSalesman.Email, &pbSalesman.Address, &pbSalesman.Phone, &createdAt, &pbSalesman.CreatedBy, &updatedAt, &pbSalesman.UpdatedBy)
		if err != nil {
			return status.Errorf(codes.Internal, "scan data: %v", err)
		}

		pbSalesman.CreatedAt = createdAt.String()
		pbSalesman.UpdatedAt = updatedAt.String()

		res := &sales.ListSalesmanResponse{
			Pagination: paginationResponse,
			Salesman:   &pbSalesman,
		}

		err = stream.Send(res)
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot send stream response: %v", err)
		}
	}
	return nil
}
