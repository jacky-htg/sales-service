package model

import (
	"context"
	"io"
	"log"
	"sales/internal/pkg/app"
	"sales/pb/inventories"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Delivery struct {
	Client inventories.DeliveryServiceClient
}

func (u *Delivery) HasTransactionBySales(ctx context.Context, salesId string) (bool, error) {
	streamClient, err := u.Client.List(app.SetMetadata(ctx), &inventories.ListDeliveryRequest{SalesOrderId: salesId})
	if s, ok := status.FromError(err); ok {
		if s.Code() == codes.Unknown {
			err = status.Errorf(codes.Internal, "Error when calling Sales.HasTreansaction service: %s", err)
		}

		return false, err
	}

	var response []*inventories.ListDeliveryResponse
	for {
		resp, err := streamClient.Recv()
		if err == io.EOF {
			log.Print("end stream")
			break
		}
		if err != nil {
			return false, status.Errorf(codes.Internal, "cannot delivery %v", err)
		}

		response = append(response, resp)
	}

	if len(response) > 0 {
		return true, nil
	}

	return false, nil
}
