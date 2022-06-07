package route

import (
	"database/sql"
	"sales/internal/service"
	"sales/pb/inventories"
	"sales/pb/sales"
	"sales/pb/users"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// GrpcRoute func
func GrpcRoute(grpcServer *grpc.Server, db *sql.DB, log *logrus.Entry, userConn *grpc.ClientConn, inventoryConn *grpc.ClientConn) {
	purchaseServer := service.Sales{
		Db:             db,
		UserClient:     users.NewUserServiceClient((userConn)),
		RegionClient:   users.NewRegionServiceClient(userConn),
		BranchClient:   users.NewBranchServiceClient(userConn),
		ProductClient:  inventories.NewProductServiceClient(inventoryConn),
		DeliveryClient: inventories.NewDeliveryServiceClient(inventoryConn),
	}
	sales.RegisterSalesServiceServer(grpcServer, &purchaseServer)

	purchaseReturnServer := service.SalesReturn{
		Db:             db,
		UserClient:     users.NewUserServiceClient((userConn)),
		RegionClient:   users.NewRegionServiceClient(userConn),
		BranchClient:   users.NewBranchServiceClient(userConn),
		DeliveryClient: inventories.NewDeliveryServiceClient(inventoryConn),
	}
	sales.RegisterSalesReturnServiceServer(grpcServer, &purchaseReturnServer)

	customerServer := service.Customer{
		Db: db,
	}
	sales.RegisterCustomerServiceServer(grpcServer, &customerServer)

	salesmanServer := service.Salesman{
		Db: db,
	}
	sales.RegisterSalesmanServiceServer(grpcServer, &salesmanServer)
}
