package main

import (
	"log"
	"net"
	"os"

	"github.com/jacky-htg/erp-pkg/db/postgres"
	"github.com/jacky-htg/sales-service/internal/config"
	"github.com/jacky-htg/sales-service/internal/route"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
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
	log := log.New(os.Stdout, "ERROR : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// create postgres database connection
	db, err := postgres.Open()
	if err != nil {
		log.Fatalf("connecting to db: %v", err)
		return
	}
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
}
