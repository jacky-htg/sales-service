package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/jacky-htg/erp-pkg/app"
	"github.com/jacky-htg/erp-proto/go/pb/inventories"
	"github.com/jacky-htg/erp-proto/go/pb/sales"
	"github.com/jacky-htg/erp-proto/go/pb/users"
	"github.com/jacky-htg/sales-service/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Sales struct {
	Db             *sql.DB
	UserClient     users.UserServiceClient
	RegionClient   users.RegionServiceClient
	BranchClient   users.BranchServiceClient
	ProductClient  inventories.ProductServiceClient
	DeliveryClient inventories.DeliveryServiceClient
	sales.UnimplementedSalesServiceServer
}

func (u *Sales) Create(ctx context.Context, in *sales.Sales) (*sales.Sales, error) {
	var salesModel model.Sales
	var err error

	// TODO : if this month any closing account, create transaction for thus month will be blocked

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesModel.Pb, err
	}

	products, err := u.createValidation(ctx, in)
	if err != nil {
		return &salesModel.Pb, err
	}

	var sumPrice float64
	for _, detail := range in.GetDetails() {
		for _, p := range products {
			if detail.GetProductId() == p.Product.GetId() {
				detail.ProductCode = p.Product.GetCode()
				detail.ProductName = p.Product.GetName()
			}
		}

		if detail.DiscPercentage > 0 {
			detail.DiscAmount = detail.GetPrice() * float64(detail.DiscPercentage) / 100
		}
		detail.TotalPrice = (detail.GetPrice() + detail.DiscAmount) * float64(detail.Quantity)
		sumPrice += detail.TotalPrice
	}

	mBranch := model.Branch{
		UserClient:   u.UserClient,
		RegionClient: u.RegionClient,
		BranchClient: u.BranchClient,
		Id:           in.GetBranchId(),
	}
	err = mBranch.IsYourBranch(ctx)
	if err != nil {
		return &salesModel.Pb, err
	}

	err = mBranch.Get(ctx)
	if err != nil {
		return &salesModel.Pb, err
	}

	if in.GetAdditionalDiscPercentage() > 0 {
		in.AdditionalDiscAmount = sumPrice * float64(in.GetAdditionalDiscPercentage()) / 100
	}
	salesModel.Pb = sales.Sales{
		BranchId:                 in.GetBranchId(),
		BranchName:               mBranch.Pb.GetName(),
		Code:                     in.GetCode(),
		SalesDate:                in.GetSalesDate(),
		Customer:                 in.GetCustomer(),
		Salesman:                 in.GetSalesman(),
		Remark:                   in.GetRemark(),
		Price:                    sumPrice,
		AdditionalDiscAmount:     in.GetAdditionalDiscAmount(),
		AdditionalDiscPercentage: in.GetAdditionalDiscPercentage(),
		TotalPrice:               (sumPrice - in.GetAdditionalDiscAmount()),
		Details:                  in.GetDetails(),
	}

	tx, err := u.Db.BeginTx(ctx, nil)
	if err != nil {
		return &salesModel.Pb, status.Errorf(codes.Internal, "begin transaction: %v", err)
	}

	err = salesModel.Create(ctx, tx)
	if err != nil {
		tx.Rollback()
		return &salesModel.Pb, err
	}

	err = tx.Commit()
	if err != nil {
		return &salesModel.Pb, status.Errorf(codes.Internal, "failed commit transaction: %v", err)
	}

	return &salesModel.Pb, nil
}

func (u *Sales) Update(ctx context.Context, in *sales.Sales) (*sales.Sales, error) {
	var salesModel model.Sales
	var err error

	// TODO : if this month any closing account, create transaction for thus month will be blocked

	if len(in.GetId()) == 0 {
		return &salesModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	salesModel.Pb.Id = in.GetId()

	// if any return, do update will be blocked
	{
		purchaseReturnModel := model.SalesReturn{
			Pb: sales.SalesReturn{
				Sales: &sales.Sales{Id: in.GetId()},
			},
		}
		if hasReturn, err := purchaseReturnModel.HasReturn(ctx, u.Db); err != nil {
			return &salesModel.Pb, err
		} else if hasReturn {
			return &salesModel.Pb, status.Error(codes.PermissionDenied, "Can not updated because the sales has return transaction")
		}
	}

	// if any delivery transaction, do update will be blocked
	mDelivery := model.Delivery{Client: u.DeliveryClient}
	if hasDelivery, err := mDelivery.HasTransactionBySales(ctx, in.GetId()); err != nil {
		return &salesModel.Pb, err
	} else if hasDelivery {
		return &salesModel.Pb, status.Error(codes.PermissionDenied, "Can not updated because the sales has delivery transaction")
	}

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesModel.Pb, err
	}

	err = salesModel.Get(ctx, u.Db)
	if err != nil {
		return &salesModel.Pb, err
	}

	// update field of sales header
	{
		if len(in.GetCustomer().Id) > 0 {
			salesModel.Pb.GetCustomer().Id = in.GetCustomer().GetId()
		}

		if len(in.GetSalesman().Id) > 0 {
			salesModel.Pb.GetSalesman().Id = in.GetSalesman().GetId()
		}

		if _, err := time.Parse("2006-01-02T15:04:05.000Z", in.GetSalesDate()); err == nil {
			salesModel.Pb.SalesDate = in.GetSalesDate()
		}

		if len(in.GetRemark()) > 0 {
			salesModel.Pb.Remark = in.GetRemark()
		}

		if in.GetAdditionalDiscPercentage() > 0 {
			salesModel.Pb.AdditionalDiscPercentage = in.AdditionalDiscPercentage
		}
	}

	tx, err := u.Db.BeginTx(ctx, nil)
	if err != nil {
		return &salesModel.Pb, status.Errorf(codes.Internal, "begin transaction: %v", err)
	}

	var newDetails []*sales.SalesDetail
	var productIds []string
	for _, detail := range in.GetDetails() {
		if len(detail.GetProductId()) == 0 {
			tx.Rollback()
			return &salesModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid product")
		}

		productIds = append(productIds, detail.GetProductId())
	}

	mProduct := model.Product{
		Client: u.ProductClient,
		Pb:     &inventories.Product{},
	}

	inProductList := inventories.ListProductRequest{
		Ids: productIds,
	}
	products, err := mProduct.List(ctx, &inProductList)

	if len(products) != len(productIds) {
		return &salesModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid product")
	}

	var sumPrice float64
	for _, detail := range in.GetDetails() {
		for _, p := range products {
			if detail.GetProductId() == p.Product.GetId() {
				detail.ProductCode = p.Product.GetCode()
				detail.ProductName = p.Product.GetName()
			}
		}

		if detail.DiscPercentage > 0 {
			detail.DiscAmount = detail.GetPrice() * float64(detail.DiscPercentage) / 100
		}
		detail.TotalPrice = (detail.GetPrice() - detail.DiscAmount) * float64(detail.Quantity)
		sumPrice += detail.GetTotalPrice()

		if len(detail.GetId()) > 0 {
			for index, data := range salesModel.Pb.GetDetails() {
				if data.GetId() == detail.GetId() {
					salesModel.Pb.Details = append(salesModel.Pb.Details[:index], salesModel.Pb.Details[index+1:]...)
					// update detail
					if detail.Price > 0 {
						data.Price = detail.Price
					}

					if detail.DiscAmount > 0 {
						data.DiscAmount = detail.DiscAmount
					}

					if detail.Quantity > 0 {
						data.Quantity = detail.Quantity
					}

					if detail.DiscPercentage > 0 {
						data.DiscPercentage = detail.DiscPercentage
						data.DiscAmount = detail.DiscAmount
						data.TotalPrice = detail.TotalPrice
					}

					var purchaseDetailModel model.SalesDetail
					purchaseDetailModel.SetPbFromPointer(data)
					if err := purchaseDetailModel.Update(ctx, tx); err != nil {
						tx.Rollback()
						return &salesModel.Pb, err
					}
					break
				}
			}
		} else {
			// operasi insert
			purchaseDetailModel := model.SalesDetail{
				Pb: sales.SalesDetail{
					SalesId:        salesModel.Pb.GetId(),
					ProductId:      detail.ProductId,
					ProductCode:    mProduct.Pb.GetCode(),
					ProductName:    mProduct.Pb.GetName(),
					Price:          detail.GetPrice(),
					DiscAmount:     detail.GetDiscAmount(),
					DiscPercentage: detail.GetDiscPercentage(),
					TotalPrice:     detail.GetTotalPrice(),
				},
			}

			err = purchaseDetailModel.Create(ctx, tx)
			if err != nil {
				tx.Rollback()
				return &salesModel.Pb, err
			}

			newDetails = append(newDetails, &purchaseDetailModel.Pb)
		}
	}

	// delete existing detail
	for _, data := range salesModel.Pb.GetDetails() {
		purchaseDetailModel := model.SalesDetail{Pb: sales.SalesDetail{
			SalesId: salesModel.Pb.GetId(),
			Id:      data.GetId(),
		}}
		err = purchaseDetailModel.Delete(ctx, tx)
		if err != nil {
			tx.Rollback()
			return &salesModel.Pb, err
		}
	}

	salesModel.Pb.Price = sumPrice
	if salesModel.Pb.AdditionalDiscPercentage > 0 {
		salesModel.Pb.AdditionalDiscAmount = sumPrice * float64(salesModel.Pb.AdditionalDiscPercentage) / 100
	}
	salesModel.Pb.TotalPrice = sumPrice - salesModel.Pb.AdditionalDiscAmount

	err = salesModel.Update(ctx, tx)
	if err != nil {
		tx.Rollback()
		return &salesModel.Pb, err
	}

	err = tx.Commit()
	if err != nil {
		return &salesModel.Pb, status.Errorf(codes.Internal, "failed commit transaction: %v", err)
	}

	return &salesModel.Pb, nil
}

func (u *Sales) View(ctx context.Context, in *sales.Id) (*sales.Sales, error) {
	var salesModel model.Sales
	var err error

	if len(in.GetId()) == 0 {
		return &salesModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	salesModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesModel.Pb, err
	}

	err = salesModel.Get(ctx, u.Db)
	if err != nil {
		return &salesModel.Pb, err
	}

	return &salesModel.Pb, nil
}

func (u *Sales) List(in *sales.ListSalesRequest, stream sales.SalesService_SalesListServer) error {
	ctx := stream.Context()
	ctx, err := app.GetMetadata(ctx)
	if err != nil {
		return err
	}

	var salesModel model.Sales
	query, paramQueries, paginationResponse, err := salesModel.ListQuery(ctx, u.Db, in)

	rows, err := u.Db.QueryContext(ctx, query, paramQueries...)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer rows.Close()
	paginationResponse.Pagination = in.GetPagination()

	for rows.Next() {
		err := app.ContextError(ctx)
		if err != nil {
			return err
		}

		var pbSales sales.Sales
		var companyID string
		var createdAt, updatedAt time.Time
		err = rows.Scan(&pbSales.Id, &companyID, &pbSales.BranchId, &pbSales.BranchName,
			&pbSales.Customer.Id, &pbSales.Salesman.Id,
			&pbSales.Code, &pbSales.SalesDate, &pbSales.Remark,
			&pbSales.Price, &pbSales.AdditionalDiscAmount, &pbSales.AdditionalDiscPercentage, &pbSales.TotalPrice,
			&createdAt, &pbSales.CreatedBy, &updatedAt, &pbSales.UpdatedBy)
		if err != nil {
			return status.Errorf(codes.Internal, "scan data: %v", err)
		}

		pbSales.CreatedAt = createdAt.String()
		pbSales.UpdatedAt = updatedAt.String()

		res := &sales.ListSalesResponse{
			Pagination: paginationResponse,
			Sales:      &pbSales,
		}

		err = stream.Send(res)
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot send stream response: %v", err)
		}
	}
	return nil
}

func (u *Sales) createValidation(ctx context.Context, in *sales.Sales) ([]*inventories.ListProductResponse, error) {
	if len(in.GetBranchId()) == 0 {
		return []*inventories.ListProductResponse{}, status.Error(codes.InvalidArgument, "Please supply valid branch")
	}

	if len(in.GetCustomer().Id) == 0 {
		return []*inventories.ListProductResponse{}, status.Error(codes.InvalidArgument, "Please supply valid customer")
	}

	if len(in.GetSalesman().Id) == 0 {
		return []*inventories.ListProductResponse{}, status.Error(codes.InvalidArgument, "Please supply valid salesman")
	}

	if _, err := time.Parse("2006-01-02T15:04:05.000Z", in.GetSalesDate()); err != nil {
		return []*inventories.ListProductResponse{}, status.Error(codes.InvalidArgument, "Please supply valid date")
	}

	// validate bulk product by call product grpc
	var productIds []string
	for _, detail := range in.GetDetails() {
		if len(detail.GetProductId()) == 0 {
			return []*inventories.ListProductResponse{}, status.Error(codes.InvalidArgument, "Please supply valid product")
		}

		productIds = append(productIds, detail.GetProductId())
	}

	mProduct := model.Product{
		Client: u.ProductClient,
		Pb:     &inventories.Product{},
	}

	inProductList := inventories.ListProductRequest{
		Ids: productIds,
	}
	products, err := mProduct.List(ctx, &inProductList)
	if err != nil {
		return []*inventories.ListProductResponse{}, err
	}

	if len(products) != len(productIds) {
		return []*inventories.ListProductResponse{}, status.Error(codes.InvalidArgument, "Please supply valid product")
	}

	return products, nil
}
