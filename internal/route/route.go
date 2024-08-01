package route

import (
	"database/sql"
	"log"

	"github.com/jacky-htg/erp-proto/go/pb/inventories"
	"github.com/jacky-htg/erp-proto/go/pb/sales"
	"github.com/jacky-htg/erp-proto/go/pb/users"
	"github.com/jacky-htg/sales-service/internal/service"
	"google.golang.org/grpc"
)

// GrpcRoute func
func GrpcRoute(grpcServer *grpc.Server, db *sql.DB, log *log.Logger, userConn *grpc.ClientConn, inventoryConn *grpc.ClientConn) {
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
