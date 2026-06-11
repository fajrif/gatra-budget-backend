//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Migration represents a single migration file
type Migration struct {
	Version   string
	Name      string
	Filename  string
	SQL       string
	AppliedAt *time.Time
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	DB             *sql.DB
	MigrationsPath string
	TableName      string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *sql.DB, migrationsPath string) *MigrationRunner {
	return &MigrationRunner{
		DB:             db,
		MigrationsPath: migrationsPath,
		TableName:      "schema_migrations",
	}
}

// Initialize creates the schema_migrations table if it doesn't exist
func (mr *MigrationRunner) Initialize() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, mr.TableName)

	_, err := mr.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	log.Printf("✓ Migration tracking table '%s' is ready", mr.TableName)
	return nil
}

// GetAllMigrations reads all migration files from the migrations directory
func (mr *MigrationRunner) GetAllMigrations() ([]Migration, error) {
	files, err := ioutil.ReadDir(mr.MigrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Skip files that don't follow the naming convention
		if !strings.Contains(file.Name(), "_") {
			log.Printf("⚠ Skipping file with invalid naming: %s", file.Name())
			continue
		}

		// Parse version from filename (e.g., "001_create_database.sql" -> "001")
		parts := strings.SplitN(file.Name(), "_", 2)
		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".sql")

		// Read file content
		filePath := filepath.Join(mr.MigrationsPath, file.Name())
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Filename: file.Name(),
			SQL:      string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// GetAppliedMigrations retrieves all migrations that have been applied
func (mr *MigrationRunner) GetAppliedMigrations() (map[string]time.Time, error) {
	query := fmt.Sprintf("SELECT version, applied_at FROM %s ORDER BY version", mr.TableName)
	rows, err := mr.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]time.Time)
	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}

	return applied, nil
}

// GetPendingMigrations returns migrations that haven't been applied yet
func (mr *MigrationRunner) GetPendingMigrations() ([]Migration, error) {
	allMigrations, err := mr.GetAllMigrations()
	if err != nil {
		return nil, err
	}

	appliedMigrations, err := mr.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	var pending []Migration
	for _, migration := range allMigrations {
		if _, applied := appliedMigrations[migration.Version]; !applied {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// Migrate runs all pending migrations
func (mr *MigrationRunner) Migrate() error {
	pending, err := mr.GetPendingMigrations()
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		log.Println("✓ No pending migrations")
		return nil
	}

	log.Printf("Found %d pending migration(s)", len(pending))
	log.Println(strings.Repeat("=", 60))

	for _, migration := range pending {
		if err := mr.runMigration(migration); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.Version, err)
		}
	}

	log.Println(strings.Repeat("=", 60))
	log.Printf("✓ Successfully applied %d migration(s)", len(pending))
	return nil
}

// runMigration executes a single migration
func (mr *MigrationRunner) runMigration(migration Migration) error {
	log.Printf("→ Running: %s (%s)", migration.Version, migration.Name)

	// Begin transaction
	tx, err := mr.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	_, err = tx.Exec(migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	// Record migration
	recordQuery := fmt.Sprintf(
		"INSERT INTO %s (version, name, applied_at) VALUES ($1, $2, $3)",
		mr.TableName,
	)
	_, err = tx.Exec(recordQuery, migration.Version, migration.Name, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✓ Applied: %s (%s)", migration.Version, migration.Name)
	return nil
}

// Status shows the current migration status
func (mr *MigrationRunner) Status() error {
	allMigrations, err := mr.GetAllMigrations()
	if err != nil {
		return err
	}

	appliedMigrations, err := mr.GetAppliedMigrations()
	if err != nil {
		return err
	}

	log.Println(strings.Repeat("=", 80))
	log.Println("Migration Status")
	log.Println(strings.Repeat("=", 80))
	log.Printf("%-10s %-40s %-10s %s\n", "Version", "Name", "Status", "Applied At")
	log.Println(strings.Repeat("-", 80))

	for _, migration := range allMigrations {
		status := "pending"
		appliedAt := ""
		
		if at, applied := appliedMigrations[migration.Version]; applied {
			status = "applied"
			appliedAt = at.Format("2006-01-02 15:04:05")
		}

		log.Printf("%-10s %-40s %-10s %s\n", 
			migration.Version, 
			truncateString(migration.Name, 40), 
			status, 
			appliedAt,
		)
	}

	log.Println(strings.Repeat("=", 80))
	pending, _ := mr.GetPendingMigrations()
	log.Printf("Total: %d migrations (%d applied, %d pending)\n", 
		len(allMigrations), 
		len(appliedMigrations), 
		len(pending),
	)
	log.Println(strings.Repeat("=", 80))

	return nil
}

// Rollback rolls back the last migration (optional feature)
func (mr *MigrationRunner) Rollback() error {
	// Get the last applied migration
	query := fmt.Sprintf(
		"SELECT version, name FROM %s ORDER BY version DESC LIMIT 1",
		mr.TableName,
	)
	
	var version, name string
	err := mr.DB.QueryRow(query).Scan(&version, &name)
	if err == sql.ErrNoRows {
		log.Println("No migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	log.Printf("⚠ Rolling back: %s (%s)", version, name)
	log.Println("Note: Rollback will only remove the migration record.")
	log.Println("You may need to manually revert database changes.")

	// Remove migration record
	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE version = $1", mr.TableName)
	_, err = mr.DB.Exec(deleteQuery, version)
	if err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	log.Printf("✓ Rolled back: %s (%s)", version, name)
	return nil
}

// Reset removes all migration records (dangerous!)
func (mr *MigrationRunner) Reset() error {
	log.Println("⚠ WARNING: This will remove all migration records!")
	log.Println("Database structure will remain unchanged.")
	
	query := fmt.Sprintf("DELETE FROM %s", mr.TableName)
	_, err := mr.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	log.Println("✓ All migration records have been reset")
	return nil
}

// Helper function to truncate strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Database configuration
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func (c *DBConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

func loadConfig() *DBConfig {
	return &DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "deployer"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "merk_buku_db"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	// Load configuration
	config := loadConfig()

	// Parse command line arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Connect to database
	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("✓ Database connection established")

	// Create migration runner
	migrationsPath := "./migrations"
	if len(os.Args) > 2 && os.Args[1] == "-path" {
		migrationsPath = os.Args[2]
		command = os.Args[3]
	}

	runner := NewMigrationRunner(db, migrationsPath)

	// Initialize migration tracking table
	if err := runner.Initialize(); err != nil {
		log.Fatal(err)
	}

	// Execute command
	switch command {
	case "migrate", "up":
		if err := runner.Migrate(); err != nil {
			log.Fatal("Migration failed:", err)
		}

	case "status":
		if err := runner.Status(); err != nil {
			log.Fatal("Failed to get status:", err)
		}

	case "rollback", "down":
		if err := runner.Rollback(); err != nil {
			log.Fatal("Rollback failed:", err)
		}

	case "reset":
		if err := runner.Reset(); err != nil {
			log.Fatal("Reset failed:", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Database Migration Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  migrate <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  migrate, up     Run all pending migrations")
	fmt.Println("  status          Show migration status")
	fmt.Println("  rollback, down  Rollback the last migration")
	fmt.Println("  reset           Reset all migration records (dangerous!)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -path <dir>     Specify migrations directory (default: ./migrations)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  migrate migrate              Run pending migrations")
	fmt.Println("  migrate status               Show migration status")
	fmt.Println("  migrate -path ./db/migrations migrate")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  DB_HOST         Database host (default: localhost)")
	fmt.Println("  DB_PORT         Database port (default: 5432)")
	fmt.Println("  DB_USER         Database user (default: deployer)")
	fmt.Println("  DB_PASSWORD     Database password")
	fmt.Println("  DB_NAME         Database name (default: merk_buku_db)")
}
