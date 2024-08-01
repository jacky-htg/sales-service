package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/jacky-htg/erp-pkg/app"
	"github.com/jacky-htg/erp-proto/go/pb/sales"
	"github.com/jacky-htg/sales-service/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Customer struct {
	Db *sql.DB
	sales.UnimplementedCustomerServiceServer
}

func (u *Customer) CustomerCreate(ctx context.Context, in *sales.Customer) (*sales.Customer, error) {
	var customerModel model.Customer
	var err error

	if len(in.GetName()) == 0 {
		return &customerModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid name")
	}

	if len(in.GetAddress()) == 0 {
		return &customerModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid address")
	}

	if len(in.GetPhone()) == 0 {
		return &customerModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid phone")
	}

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &customerModel.Pb, err
	}

	// code validation
	{
		if len(in.GetCode()) == 0 {
			return &customerModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid code")
		}

		customerModel = model.Customer{}
		customerModel.Pb.Code = in.GetCode()
		err = customerModel.GetByCode(ctx, u.Db)
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() != codes.NotFound {
				return &customerModel.Pb, err
			}
		}

		if len(customerModel.Pb.GetId()) > 0 {
			return &customerModel.Pb, status.Error(codes.AlreadyExists, "code must be unique")
		}
	}

	customerModel.Pb = sales.Customer{
		Code:    in.GetCode(),
		Name:    in.GetName(),
		Address: in.GetAddress(),
		Phone:   in.GetPhone(),
	}
	err = customerModel.Create(ctx, u.Db)
	if err != nil {
		return &customerModel.Pb, err
	}

	return &customerModel.Pb, nil
}

func (u *Customer) CustomerUpdate(ctx context.Context, in *sales.Customer) (*sales.Customer, error) {
	var customerModel model.Customer
	var err error

	if len(in.GetId()) == 0 {
		return &customerModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	customerModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &customerModel.Pb, err
	}

	err = customerModel.Get(ctx, u.Db)
	if err != nil {
		return &customerModel.Pb, err
	}

	if len(in.GetName()) > 0 {
		customerModel.Pb.Name = in.GetName()
	}

	if len(in.GetAddress()) > 0 {
		customerModel.Pb.Address = in.GetAddress()
	}

	if len(in.GetPhone()) > 0 {
		customerModel.Pb.Phone = in.GetPhone()
	}

	err = customerModel.Update(ctx, u.Db)
	if err != nil {
		return &customerModel.Pb, err
	}

	return &customerModel.Pb, nil
}

func (u *Customer) CustomerView(ctx context.Context, in *sales.Id) (*sales.Customer, error) {
	var customerModel model.Customer
	var err error

	if len(in.GetId()) == 0 {
		return &customerModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	customerModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &customerModel.Pb, err
	}

	err = customerModel.Get(ctx, u.Db)
	if err != nil {
		return &customerModel.Pb, err
	}

	return &customerModel.Pb, nil
}

func (u *Customer) CustomerDelete(ctx context.Context, in *sales.Id) (*sales.MyBoolean, error) {
	var output sales.MyBoolean
	output.Boolean = false

	var customerModel model.Customer
	var err error

	if len(in.GetId()) == 0 {
		return &output, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	customerModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &output, err
	}

	err = customerModel.Get(ctx, u.Db)
	if err != nil {
		return &output, err
	}

	err = customerModel.Delete(ctx, u.Db)
	if err != nil {
		return &output, err
	}

	output.Boolean = true
	return &output, nil
}

func (u *Customer) CustomerList(in *sales.ListCustomerRequest, stream sales.CustomerService_CustomerListServer) error {
	ctx := stream.Context()
	ctx, err := app.GetMetadata(ctx)
	if err != nil {
		return err
	}

	var customerModel model.Customer
	query, paramQueries, paginationResponse, err := customerModel.ListQuery(ctx, u.Db, in.Pagination)
	if err != nil {
		return err
	}

	rows, err := u.Db.QueryContext(ctx, query, paramQueries...)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer rows.Close()
	paginationResponse.Pagination = in.Pagination

	for rows.Next() {
		err := app.ContextError(ctx)
		if err != nil {
			return err
		}

		var pbCustomer sales.Customer
		var companyID string
		var createdAt, updatedAt time.Time
		err = rows.Scan(&pbCustomer.Id, &companyID, &pbCustomer.Code, &pbCustomer.Name, &pbCustomer.Address, &pbCustomer.Phone, &createdAt, &pbCustomer.CreatedBy, &updatedAt, &pbCustomer.UpdatedBy)
		if err != nil {
			return status.Errorf(codes.Internal, "scan data: %v", err)
		}

		pbCustomer.CreatedAt = createdAt.String()
		pbCustomer.UpdatedAt = updatedAt.String()

		res := &sales.ListCustomerResponse{
			Pagination: paginationResponse,
			Customer:   &pbCustomer,
		}

		err = stream.Send(res)
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot send stream response: %v", err)
		}
	}
	return nil
}
