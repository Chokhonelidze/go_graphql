package core

import (
	"fmt"
	// "net/url"
	"os"

	_ "github.com/microsoft/go-mssqldb/azuread"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func PostgresConnection() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:postgres@postgres:5432/postgres"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "public.", // add your table prefix here
			SingularTable: true,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to connect to database: %v", err))
	}
	fmt.Println("Successfully connected to the database")
	return db, nil
}

func MSSQLConnection() (*gorm.DB, error) {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "mssql_db"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "1433"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "graphql_db"
	}
	dsn := fmt.Sprintf("sqlserver://%s.database.windows.net?database=%s&fedauth=ActiveDirectoryDefault", dbHost, dbName)
	fmt.Println("Connecting to MSSQL with DSN:", dsn)
	dialector := sqlserver.Dialector{
		Config: &sqlserver.Config{
			DSN:        dsn,
			DriverName: "azuresql", // This tells GORM to use the Entra ID-aware driver
		},
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "dbo.", // add your table prefix here
			SingularTable: false,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to connect to database: %v", err))
	}
	fmt.Println("Successfully connected to the database")
	return db, nil
}
