package main

import (
	"net"
	"os"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"sales/internal/config"
	"sales/internal/pkg/db/postgres"
	"sales/internal/pkg/log/logruslog"
	"sales/internal/route"
)

const defaultPort = "8002"

func main() {
	// lookup and setup env
	if _, ok := os.LookupEnv("PORT"); !ok {
		config.Setup(".env")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// init log
	log := logruslog.Init()

	// create postgres database connection
	db, err := postgres.Open()
	if err != nil {
		log.Errorf("connecting to db: %v", err)
		return
	}
	log.Info("connecting to postgresql database")

	defer db.Close()

	// listen tcp port
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	userConn, err := grpc.Dial(os.Getenv("USER_SERVICE"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("create user service connection: %v", err)
	}
	defer userConn.Close()

	inventoryConn, err := grpc.Dial(os.Getenv("INVENTORY_SERVICE"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("create inventory service connection: %v", err)
	}
	defer userConn.Close()

	// routing grpc services
	route.GrpcRoute(grpcServer, db, log, userConn, inventoryConn)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
		return
	}
	log.Info("serve grpc on port: " + port)
}
