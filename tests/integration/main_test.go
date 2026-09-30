// Package integration contains tests that run against a real PostgreSQL
// database. They cover the behaviour that sqlmock cannot verify: real schema
// constraints, foreign-key cascades, unique indexes, and real SQL generation.
//
// These tests are intentionally kept out of tests/handlers. Run them with
// `make test-integration`.
package integration

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gatra-budget-backend/internal/migrate"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB is the pooled connection shared by every test in this package.
// Individual tests never write to it directly; they go through withTx.
var testDB *gorm.DB

// testJWTSecret is pinned so the tests do not depend on the developer's .env.
const testJWTSecret = "integration-test-secret"

func TestMain(m *testing.M) {
	// The test binary runs from tests/integration, so the project root is two
	// levels up. A missing .env is fine: the TEST_DB_* variables may be set
	// in the environment instead.
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))

	// Pin the auth settings before any handler reads them.
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Setenv("JWT_EXPIRE_HOURS", "1")

	cfg := testConfig()

	// Hard safety gate. A mistyped DB_NAME must never let these tests drop,
	// migrate, or write to a real development database.
	if !strings.Contains(cfg.DBName, "_test") {
		fmt.Fprintf(os.Stderr,
			"integration tests refusing to run: target database %q does not look like a test database.\n"+
				"Set TEST_DB_NAME to a database whose name contains \"_test\" (e.g. %s_test).\n",
			cfg.DBName, cfg.DBName)
		os.Exit(1)
	}

	if err := prepareTestDB(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "integration test setup failed: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if sqlDB, err := testDB.DB(); err == nil {
		sqlDB.Close()
	}
	os.Exit(code)
}

// testConfig resolves the test database settings, falling back to the
// development DB_* values with a "_test" suffix so a developer who has not set
// up TEST_DB_* still gets an isolated database rather than their own.
func testConfig() *migrate.DBConfig {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "deployer")
	pass := getEnv("DB_PASSWORD", "")
	baseName := getEnv("DB_NAME", "gatra_budget_db")
	sslMode := getEnv("DB_SSLMODE", "disable")

	return &migrate.DBConfig{
		Host:     getEnv("TEST_DB_HOST", host),
		Port:     getEnv("TEST_DB_PORT", port),
		User:     getEnv("TEST_DB_USER", user),
		Password: getEnv("TEST_DB_PASSWORD", pass),
		DBName:   getEnv("TEST_DB_NAME", baseName+"_test"),
		SSLMode:  getEnv("TEST_DB_SSLMODE", sslMode),
	}
}

// prepareTestDB makes sure the test database exists and that its schema matches
// the migration files on disk, rebuilding it if the two have drifted apart.
func prepareTestDB(cfg *migrate.DBConfig) error {
	path := migrationsPath()

	sqlDB, err := syncTestSchema(cfg, path)
	if err != nil {
		return err
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		sqlDB.Close()
		return fmt.Errorf("failed to open gorm: %w", err)
	}

	testDB = gormDB
	return nil
}

// openTestDB connects to the test database, creating it if it does not exist
// yet, and returns a runner with its tracking tables in place.
func openTestDB(cfg *migrate.DBConfig, path string) (*sql.DB, *migrate.MigrationRunner, error) {
	sqlDB, err := migrate.Connect(cfg)
	if err != nil {
		// Most likely the database simply has not been created yet.
		if createErr := migrate.EnsureDatabase(cfg); createErr != nil {
			return nil, nil, fmt.Errorf("cannot reach %s and could not create it: %w", cfg.DBName, createErr)
		}
		if sqlDB, err = migrate.Connect(cfg); err != nil {
			return nil, nil, err
		}
	}

	runner := migrate.NewMigrationRunner(sqlDB, path)
	if err := runner.Initialize(); err != nil {
		sqlDB.Close()
		return nil, nil, err
	}
	return sqlDB, runner, nil
}

// syncTestSchema brings the test database's schema in line with the migration
// files, and returns a connection ready for use.
//
// Applying pending migrations is not enough on its own. schema_migrations
// records a version as applied forever, so a migration that was edited after it
// ran is skipped silently and the tests would pass against a schema that no
// longer matches the migrations. When the fingerprint of the files on disk no
// longer matches the one recorded in the database, the only reliable fix is to
// throw the schema away and rebuild it.
func syncTestSchema(cfg *migrate.DBConfig, path string) (*sql.DB, error) {
	sqlDB, runner, err := openTestDB(cfg, path)
	if err != nil {
		return nil, err
	}

	drift, err := runner.DetectDrift()
	if err != nil {
		sqlDB.Close()
		return nil, err
	}

	applied, err := runner.GetAppliedMigrations()
	if err != nil {
		sqlDB.Close()
		return nil, err
	}

	// A database with nothing applied has no stale schema to discard, so it is
	// never dropped — that would add a pointless drop/create cycle to every
	// first run.
	stale := drift == migrate.DriftChanged || (drift == migrate.DriftUnknown && len(applied) > 0)

	if stale {
		log.Printf("⚠ %s: rebuilding test database %s from scratch", drift, cfg.DBName)

		// This must happen before the drop: PostgreSQL refuses to drop a
		// database that the current session is still attached to.
		sqlDB.Close()
		if err := migrate.DropDatabase(cfg); err != nil {
			return nil, err
		}
		if sqlDB, runner, err = openTestDB(cfg, path); err != nil {
			return nil, err
		}
	} else if len(applied) > 0 {
		log.Printf("✓ Test database %s is up to date", cfg.DBName)
	} else {
		log.Printf("✓ Creating test database %s from migrations", cfg.DBName)
	}

	if err := runner.Migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	fingerprint, err := migrate.Fingerprint(path)
	if err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err := runner.SetFingerprint(fingerprint); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return sqlDB, nil
}

// migrationsPath locates the migrations directory relative to this package.
func migrationsPath() string {
	for _, candidate := range []string{
		filepath.Join("..", "..", "migrations"),
		filepath.Join("..", "migrations"),
		"migrations",
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join("..", "..", "migrations")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
