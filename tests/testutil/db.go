package testutil

import (
	"database/sql"
	"boilerplates-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StringPtr(s string) *string {
	return &s
}

// SetupMockDB creates a mock database connection for testing
func SetupMockDB() (*sql.DB, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	dialector := postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	// Set the global DB instance for testing
	config.DB = gormDB

	return db, mock, nil
}

// CloseMockDB closes the mock database connection
func CloseMockDB(db *sql.DB) {
	db.Close()
}
