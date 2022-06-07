package service

import (
	"context"
	"database/sql"
	"sales/internal/model"
	"sales/internal/pkg/app"
	"sales/pb/inventories"
	"sales/pb/sales"
	"sales/pb/users"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SalesReturn struct {
	Db             *sql.DB
	UserClient     users.UserServiceClient
	RegionClient   users.RegionServiceClient
	BranchClient   users.BranchServiceClient
	DeliveryClient inventories.DeliveryServiceClient
	sales.UnimplementedSalesReturnServiceServer
}

func (u *SalesReturn) Create(ctx context.Context, in *sales.SalesReturn) (*sales.SalesReturn, error) {
	var salesReturnModel model.SalesReturn
	var err error

	// TODO : if this month any closing account, create transaction for thus month will be blocked

	if len(in.GetBranchId()) == 0 {
		return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid branch")
	}

	if len(in.GetSales().GetId()) == 0 {
		return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid sales")
	}

	if _, err := time.Parse("2006-01-02T15:04:05.000Z", in.GetReturnDate()); err != nil {
		return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid date")
	}

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	// validate not any delivery order yet
	mDelivery := model.Delivery{Client: u.DeliveryClient}
	hasDelivery, err := mDelivery.HasTransactionBySales(ctx, in.Sales.Id)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	if hasDelivery {
		return &salesReturnModel.Pb, status.Error(codes.FailedPrecondition, "Sales has delivery transaction ")
	}

	// validate outstanding sales
	mSales := model.Sales{Pb: sales.Sales{Id: in.Sales.Id}}
	outstandingSalesDetails, err := mSales.OutstandingDetail(ctx, u.Db, nil)
	if len(outstandingSalesDetails) == 0 {
		return &salesReturnModel.Pb, status.Error(codes.FailedPrecondition, "Sales has been returned ")
	}

	err = mSales.Get(ctx, u.Db)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	var sumPrice float64
	var salesQty, returnQty int32
	for _, detail := range in.GetDetails() {
		if len(detail.GetProductId()) == 0 {
			return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid product")
		}

		if !u.validateOutstandingDetail(ctx, detail, outstandingSalesDetails) {
			return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid outstanding product")
		}

		for _, p := range mSales.Pb.GetDetails() {
			salesQty += p.Quantity
			if p.GetProductId() == detail.ProductId {
				detail.Price = p.Price
				if p.DiscPercentage > 0 {
					detail.DiscPercentage = p.DiscPercentage
					detail.DiscAmount = p.GetPrice() * float64(p.DiscPercentage) / 100
				} else if p.DiscAmount > 0 {
					detail.DiscAmount = p.DiscAmount
				}
				detail.TotalPrice = (detail.Price - detail.DiscAmount) * float64(detail.Quantity)
				break
			}
		}

		returnQty += detail.Quantity
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
		return &salesReturnModel.Pb, err
	}

	err = mBranch.Get(ctx)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	in.Price = sumPrice
	if mSales.Pb.AdditionalDiscPercentage > 0 {
		in.AdditionalDiscAmount = in.Price * float64(in.AdditionalDiscPercentage) / 100
	} else if mSales.Pb.AdditionalDiscAmount > 0 {
		additionalDiscPerQty := mSales.Pb.AdditionalDiscAmount / float64(salesQty)
		in.AdditionalDiscAmount = additionalDiscPerQty * float64(returnQty)
		returnAdditionalDisc, err := mSales.GetReturnAdditionalDisc(ctx, u.Db)
		if err != nil {
			return &salesReturnModel.Pb, status.Error(codes.Internal, "Error get return additional disc")
		}
		remainingAdditionalDisc := mSales.Pb.AdditionalDiscAmount - returnAdditionalDisc
		if in.AdditionalDiscAmount > remainingAdditionalDisc {
			in.AdditionalDiscAmount = remainingAdditionalDisc
		}
	}
	in.TotalPrice = in.Price - in.AdditionalDiscAmount

	salesReturnModel.Pb = sales.SalesReturn{
		BranchId:                 in.GetBranchId(),
		BranchName:               mBranch.Pb.GetName(),
		Code:                     in.GetCode(),
		ReturnDate:               in.GetReturnDate(),
		Sales:                    in.GetSales(),
		Remark:                   in.GetRemark(),
		Price:                    in.GetPrice(),
		AdditionalDiscAmount:     in.GetAdditionalDiscAmount(),
		AdditionalDiscPercentage: in.GetAdditionalDiscPercentage(),
		TotalPrice:               in.GetTotalPrice(),
		Details:                  in.GetDetails(),
	}

	tx, err := u.Db.BeginTx(ctx, nil)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	err = salesReturnModel.Create(ctx, tx)
	if err != nil {
		tx.Rollback()
		return &salesReturnModel.Pb, err
	}

	err = tx.Commit()
	if err != nil {
		return &salesReturnModel.Pb, status.Error(codes.Internal, "Error when commit transaction")
	}

	return &salesReturnModel.Pb, nil
}

func (u *SalesReturn) View(ctx context.Context, in *sales.Id) (*sales.SalesReturn, error) {
	var salesReturnModel model.SalesReturn
	var err error

	if len(in.GetId()) == 0 {
		return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	salesReturnModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	err = salesReturnModel.Get(ctx, u.Db)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	return &salesReturnModel.Pb, nil
}

func (u *SalesReturn) Update(ctx context.Context, in *sales.SalesReturn) (*sales.SalesReturn, error) {
	var salesReturnModel model.SalesReturn
	var err error

	// TODO : if this month any closing stock, create transaction for thus month will be blocked

	if len(in.GetId()) == 0 {
		return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid id")
	}
	salesReturnModel.Pb.Id = in.GetId()

	ctx, err = app.GetMetadata(ctx)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	// validate not any delivery order yet
	mDelivery := model.Delivery{Client: u.DeliveryClient}
	hasDelivery, err := mDelivery.HasTransactionBySales(ctx, in.Sales.Id)
	if hasDelivery {
		return &salesReturnModel.Pb, status.Error(codes.FailedPrecondition, "Sales has receive transaction ")
	}

	// validate outstanding sales
	mSales := model.Sales{Pb: sales.Sales{Id: in.Sales.Id}}
	salesReturnId := in.GetId()
	outstandingSalesDetails, err := mSales.OutstandingDetail(ctx, u.Db, &salesReturnId)
	if len(outstandingSalesDetails) == 0 {
		return &salesReturnModel.Pb, status.Error(codes.FailedPrecondition, "Sales has been returned ")
	}

	err = mSales.Get(ctx, u.Db)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	err = salesReturnModel.Get(ctx, u.Db)
	if err != nil {
		return &salesReturnModel.Pb, err
	}

	if _, err := time.Parse("2006-01-02T15:04:05.000Z", in.GetReturnDate()); err == nil {
		salesReturnModel.Pb.ReturnDate = in.GetReturnDate()
	}

	tx, err := u.Db.BeginTx(ctx, nil)
	if err != nil {
		return &salesReturnModel.Pb, status.Errorf(codes.Internal, "begin transaction: %v", err)
	}

	var sumPrice float64
	var salesQty, returnQty int32
	var newDetails []*sales.SalesReturnDetail
	for _, detail := range in.GetDetails() {
		if len(detail.GetProductId()) == 0 {
			tx.Rollback()
			return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid product")
		}

		if !u.validateOutstandingDetail(ctx, detail, outstandingSalesDetails) {
			return &salesReturnModel.Pb, status.Error(codes.InvalidArgument, "Please supply valid outstanding product")
		}

		if len(detail.GetId()) > 0 {
			for _, p := range mSales.Pb.GetDetails() {
				salesQty += p.Quantity
				if p.GetProductId() == detail.ProductId {
					break
				}
			}

			returnQty += detail.Quantity
			detail.TotalPrice = (detail.Price - detail.DiscAmount) * float64(detail.Quantity)
			sumPrice += detail.TotalPrice

			// operasi update
			salesReturnDetailModel := model.SalesReturnDetail{
				Pb: sales.SalesReturnDetail{
					Id:            detail.Id,
					Quantity:      detail.Quantity,
					TotalPrice:    detail.TotalPrice,
					SalesReturnId: salesReturnModel.Pb.Id,
				},
			}

			err = salesReturnDetailModel.Update(ctx, tx)
			if err != nil {
				tx.Rollback()
				return &salesReturnModel.Pb, err
			}

			newDetails = append(newDetails, &salesReturnDetailModel.Pb)
			for index, data := range salesReturnModel.Pb.GetDetails() {
				if data.GetId() == detail.GetId() {
					salesReturnModel.Pb.Details = append(salesReturnModel.Pb.Details[:index], salesReturnModel.Pb.Details[index+1:]...)
					break
				}
			}

		} else {
			for _, p := range mSales.Pb.GetDetails() {
				salesQty += p.Quantity
				if p.GetProductId() == detail.ProductId {
					detail.Price = p.Price
					if p.DiscPercentage > 0 {
						detail.DiscPercentage = p.DiscPercentage
						detail.DiscAmount = p.GetPrice() * float64(p.DiscPercentage) / 100
					} else if p.DiscAmount > 0 {
						detail.DiscAmount = p.DiscAmount
					}
					detail.TotalPrice = (detail.Price - detail.DiscAmount) * float64(detail.Quantity)
					break
				}
			}

			returnQty += detail.Quantity
			sumPrice += detail.TotalPrice

			// operasi insert
			salesReturnDetailModel := model.SalesReturnDetail{Pb: sales.SalesReturnDetail{
				SalesReturnId:  salesReturnModel.Pb.GetId(),
				ProductId:      detail.GetProductId(),
				Quantity:       detail.GetQuantity(),
				Price:          detail.GetPrice(),
				DiscAmount:     detail.GetDiscAmount(),
				DiscPercentage: detail.GetDiscPercentage(),
				TotalPrice:     detail.GetTotalPrice(),
			}}
			salesReturnDetailModel.PbSalesReturn = sales.SalesReturn{
				Id:         salesReturnModel.Pb.Id,
				BranchId:   salesReturnModel.Pb.BranchId,
				BranchName: salesReturnModel.Pb.BranchName,
				Sales:      salesReturnModel.Pb.GetSales(),
				Code:       salesReturnModel.Pb.Code,
				ReturnDate: salesReturnModel.Pb.ReturnDate,
				Remark:     salesReturnModel.Pb.Remark,
				CreatedAt:  salesReturnModel.Pb.CreatedAt,
				CreatedBy:  salesReturnModel.Pb.CreatedBy,
				UpdatedAt:  salesReturnModel.Pb.UpdatedAt,
				UpdatedBy:  salesReturnModel.Pb.UpdatedBy,
				Details:    salesReturnModel.Pb.Details,
			}
			err = salesReturnDetailModel.Create(ctx, tx)
			if err != nil {
				tx.Rollback()
				return &salesReturnModel.Pb, err
			}

			newDetails = append(newDetails, &salesReturnDetailModel.Pb)
		}
	}

	// delete existing detail
	for _, data := range salesReturnModel.Pb.GetDetails() {
		salesReturnDetailModel := model.SalesReturnDetail{Pb: sales.SalesReturnDetail{
			SalesReturnId: salesReturnModel.Pb.GetId(),
			Id:            data.GetId(),
		}}
		err = salesReturnDetailModel.Delete(ctx, tx)
		if err != nil {
			tx.Rollback()
			return &salesReturnModel.Pb, err
		}
	}

	err = salesReturnModel.Update(ctx, tx)
	if err != nil {
		tx.Rollback()
		return &salesReturnModel.Pb, err
	}

	err = tx.Commit()
	if err != nil {
		return &salesReturnModel.Pb, status.Error(codes.Internal, "failed commit transaction")
	}

	return &salesReturnModel.Pb, nil
}

func (u *SalesReturn) List(in *sales.ListSalesReturnRequest, stream sales.SalesReturnService_SalesReturnListServer) error {
	ctx := stream.Context()
	ctx, err := app.GetMetadata(ctx)
	if err != nil {
		return err
	}

	var salesReturnModel model.SalesReturn
	query, paramQueries, paginationResponse, err := salesReturnModel.ListQuery(ctx, u.Db, in)

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

		var pbSalesReturn sales.SalesReturn
		var companyID string
		var createdAt, updatedAt time.Time
		err = rows.Scan(&pbSalesReturn.Id, &companyID, &pbSalesReturn.BranchId, &pbSalesReturn.BranchName, &pbSalesReturn.GetSales().Id,
			&pbSalesReturn.Code, &pbSalesReturn.ReturnDate, &pbSalesReturn.Remark,
			&pbSalesReturn.Price, &pbSalesReturn.AdditionalDiscAmount, &pbSalesReturn.AdditionalDiscPercentage, &pbSalesReturn.TotalPrice,
			&createdAt, &pbSalesReturn.CreatedBy, &updatedAt, &pbSalesReturn.UpdatedBy)
		if err != nil {
			return status.Errorf(codes.Internal, "scan data: %v", err)
		}

		pbSalesReturn.CreatedAt = createdAt.String()
		pbSalesReturn.UpdatedAt = updatedAt.String()

		res := &sales.ListSalesReturnResponse{
			Pagination:  paginationResponse,
			SalesReturn: &pbSalesReturn,
		}

		err = stream.Send(res)
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot send stream response: %v", err)
		}
	}
	return nil
}

func (u *SalesReturn) validateOutstandingDetail(ctx context.Context, in *sales.SalesReturnDetail, outstanding []*sales.SalesDetail) bool {
	isValid := false
	for _, out := range outstanding {
		if in.ProductId == out.ProductId && in.Quantity <= out.Quantity {
			isValid = true
			break
		}
	}

	return isValid
}
