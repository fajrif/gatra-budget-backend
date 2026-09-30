package migrate

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// DBConfig holds the connection settings for a PostgreSQL database
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// LoadConfig reads the standard DB_* environment variables, falling back to
// the same defaults the rest of the project uses in .env.
func LoadConfig() *DBConfig {
	return &DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "deployer"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "gatra_budget_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// ConnectionString renders the config as a libpq keyword/value DSN
func (c *DBConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// Connect opens and verifies a connection using the pgx driver
func Connect(cfg *DBConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to %s: %w", cfg.DBName, err)
	}
	return db, nil
}

// EnsureDatabase creates the configured database if it does not already exist.
// It connects to the maintenance "postgres" database because CREATE DATABASE
// cannot run inside a transaction or while connected to the target itself.
func EnsureDatabase(cfg *DBConfig) error {
	adminCfg := *cfg
	adminCfg.DBName = "postgres"

	admin, err := Connect(&adminCfg)
	if err != nil {
		return fmt.Errorf("cannot reach server to create %s: %w", cfg.DBName, err)
	}
	defer admin.Close()

	var exists bool
	err = admin.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for database %s: %w", cfg.DBName, err)
	}
	if exists {
		return nil
	}

	// Identifier cannot be parameterised; quote it instead.
	if _, err := admin.Exec(fmt.Sprintf("CREATE DATABASE %q", cfg.DBName)); err != nil {
		return fmt.Errorf("failed to create database %s: %w", cfg.DBName, err)
	}
	return nil
}

// DropDatabase deletes the configured database, terminating any sessions still
// attached to it first.
//
// The terminate step is not optional housekeeping: PostgreSQL refuses to drop a
// database that anyone is connected to, and on a machine used daily that is
// guaranteed to happen — a previous test run, an idle psql window, an editor's
// database client. Without this the drop fails intermittently and the failure
// looks like a permissions problem rather than a stuck session.
func DropDatabase(cfg *DBConfig) error {
	adminCfg := *cfg
	adminCfg.DBName = "postgres"

	admin, err := Connect(&adminCfg)
	if err != nil {
		return fmt.Errorf("cannot reach server to drop %s: %w", cfg.DBName, err)
	}
	defer admin.Close()

	var exists bool
	if err := admin.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check for database %s: %w", cfg.DBName, err)
	}
	if !exists {
		return nil
	}

	// Disconnect everyone else. pg_backend_pid() excludes this connection,
	// which keeps the statement from terminating its own session mid-flight.
	if _, err := admin.Exec(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()",
		cfg.DBName,
	); err != nil {
		return fmt.Errorf("failed to disconnect sessions from %s: %w", cfg.DBName, err)
	}

	// Identifier cannot be parameterised; quote it instead.
	if _, err := admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %q", cfg.DBName)); err != nil {
		return fmt.Errorf("failed to drop database %s: %w", cfg.DBName, err)
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
