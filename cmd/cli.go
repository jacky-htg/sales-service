package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jacky-htg/erp-pkg/db/postgres"
	"github.com/jacky-htg/sales-service/internal/config"
	"github.com/jacky-htg/sales-service/internal/schema"
	_ "github.com/lib/pq"
)

func main() {
	if _, ok := os.LookupEnv("APP_ENV"); !ok {
		_, err := os.Stat(".env.prod")
		if os.IsNotExist(err) {
			config.Setup(".env")
		} else {
			config.Setup(".env.prod")
		}
	}

	log := log.New(os.Stdout, "grpc skeleton : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	if err := run(log); err != nil {
		log.Printf("error: shutting down: %s", err)
		os.Exit(1)
	}
}

func run(log *log.Logger) error {
	log.Printf("main : Started")
	defer log.Println("main : Completed")

	db, err := postgres.Open()
	if err != nil {
		return fmt.Errorf("connecting to db: %v", err)
	}
	defer db.Close()

	flag.Parse()

	switch flag.Arg(0) {
	case "migrate":
		if err := schema.Migrate(db); err != nil {
			return fmt.Errorf("applying migrations: %v", err)
		}
		log.Println("Migrations complete")
		return nil

	case "seed":
		if err := schema.Seed(db); err != nil {
			return fmt.Errorf("seeding database: %v", err)
		}
		log.Println("Seed data complete")
		return nil
	}

	return nil
}
