package testutil

import (
	"testing"

	"gatra-budget-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StringPtr(s string) *string {
	return &s
}

// SetupMockDB installs a sqlmock-backed GORM connection as the global
// config.DB and returns the mock for setting expectations.
//
// The mock is closed and any unmet expectations are reported automatically
// when the test (or subtest) that called this function finishes, so there is
// no need to defer anything. Passing the *testing.T also scopes the cleanup to
// that test, which keeps nested subtests isolated from each other.
func SetupMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		db.Close()
		t.Fatalf("failed to open gorm with sqlmock: %v", err)
	}

	previousDB := config.DB
	config.DB = gormDB

	t.Cleanup(func() {
		config.DB = previousDB

		// Report before closing: closing can discard pending expectations.
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet database expectations: %v", err)
		}

		// sqlmock treats Close as just another expected call.
		mock.ExpectClose()
		if err := db.Close(); err != nil {
			t.Errorf("failed to close sqlmock: %v", err)
		}
	})

	return mock
}
