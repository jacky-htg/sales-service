package schema

import (
	"database/sql"

	"github.com/GuiaBolso/darwin"
)

var migrations = []darwin.Migration{
	{
		Version:     1,
		Description: "Add Customers",
		Script: `
		CREATE TABLE customers (
			id uuid NOT NULL PRIMARY KEY,
			company_id uuid NOT NULL,
			code CHAR(10) NOT NULL,
			name VARCHAR(45) NOT NULL UNIQUE,
			address VARCHAR(255) NOT NULL,
			phone VARCHAR(20) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			created_by uuid NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_by uuid NOT NULL,
			UNIQUE(company_id, code)
		);`,
	},
	{
		Version:     2,
		Description: "Add Salesman",
		Script: `
		CREATE TABLE salesman (
			id uuid NOT NULL PRIMARY KEY,
			company_id uuid NOT NULL,
			code CHAR(10) NOT NULL,
			email VARCHAR(50) NOT NULL UNIQUE,
			name VARCHAR(45) NOT NULL,
			address VARCHAR(255) NOT NULL,
			phone VARCHAR(20) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			created_by uuid NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_by uuid NOT NULL,
			UNIQUE(company_id, code)
		);`,
	},
	{
		Version:     3,
		Description: "Add Sales",
		Script: `
		CREATE TABLE sales (
			id uuid NOT NULL PRIMARY KEY,
			company_id	uuid NOT NULL,
			branch_id uuid NOT NULL,
			branch_name varchar(100) NOT NULL,
			customer_id uuid NOT NULL,
			salesman_id uuid NOT NULL,
			code	CHAR(13) NOT NULL,
			sales_date	DATE NOT NULL,
			remark VARCHAR(255) NOT NULL,
			price DOUBLE PRECISION NOT NULL,
			additional_disc_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			additional_disc_percentage REAL NOT NULL DEFAULT 0,
			total_price DOUBLE PRECISION NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			created_by uuid NOT NULL,
			updated_by uuid NOT NULL,
			UNIQUE(company_id, code),
			CONSTRAINT fk_sales_to_customers FOREIGN KEY (customer_id) REFERENCES customers(id),
			CONSTRAINT fk_sales_to_salesman FOREIGN KEY (salesman_id) REFERENCES salesman(id)
		);`,
	},
	{
		Version:     4,
		Description: "Add Sales Details",
		Script: `
		CREATE TABLE sales_details (
			id uuid NOT NULL PRIMARY KEY,
			sales_id	uuid NOT NULL,
			product_id uuid NOT NULL,
			price DOUBLE PRECISION NOT NULL,
			quantity INT NOT NULL CHECK (quantity > 0),
			disc_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			disc_percentage REAL NOT NULL DEFAULT 0,
			total_price DOUBLE PRECISION NOT NULL,
			UNIQUE(sales_id, product_id),
			CONSTRAINT fk_sales_details_to_sales FOREIGN KEY (sales_id) REFERENCES sales(id) ON DELETE CASCADE ON UPDATE CASCADE
		);`,
	},
	{
		Version:     5,
		Description: "Add Sales Return",
		Script: `
		CREATE TABLE sales_returns (
			id uuid NOT NULL PRIMARY KEY,
			company_id	uuid NOT NULL,
			branch_id uuid NOT NULL,
			branch_name varchar(100) NOT NULL,
			sales_id uuid NOT NULL,
			code	CHAR(13) NOT NULL,
			return_date	DATE NOT NULL,
			remark VARCHAR(255) NOT NULL,
			price DOUBLE PRECISION NOT NULL,
			additional_disc_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			additional_disc_percentage REAL NOT NULL DEFAULT 0,
			total_price DOUBLE PRECISION NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			created_by uuid NOT NULL,
			updated_by uuid NOT NULL,
			UNIQUE(company_id, code),
			CONSTRAINT fk_sales_returns_to_sales FOREIGN KEY (sales_id) REFERENCES sales(id)
		);`,
	},
	{
		Version:     6,
		Description: "Add Sales Return Details",
		Script: `
		CREATE TABLE sales_return_details (
			id uuid NOT NULL PRIMARY KEY,
			sales_return_id	uuid NOT NULL,
			product_id uuid NOT NULL,
			price DOUBLE PRECISION NOT NULL,
			quantity INT NOT NULL CHECK (quantity > 0),
			disc_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			disc_percentage REAL NOT NULL DEFAULT 0,
			total_price DOUBLE PRECISION NOT NULL,
			UNIQUE(sales_return_id, product_id),
			CONSTRAINT fk_sales_return_details_to_sales_returns FOREIGN KEY (sales_return_id) REFERENCES sales_returns(id) ON DELETE CASCADE ON UPDATE CASCADE
		);`,
	},
}

func Migrate(db *sql.DB) error {
	driver := darwin.NewGenericDriver(db, darwin.PostgresDialect{})

	d := darwin.New(driver, migrations, nil)

	return d.Migrate()
}
