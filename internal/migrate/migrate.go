// Package migrate applies the SQL files in the migrations directory to a
// PostgreSQL database, tracking what has run in a schema_migrations table.
//
// The runner is driver-agnostic: callers own the *sql.DB. Use Connect or pass
// your own connection, which is what the integration tests do so they can
// point it at a throwaway database.
package migrate

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

// Initialize creates the tracking tables if they don't exist
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

	if err := mr.initializeStateTable(); err != nil {
		return err
	}

	log.Printf("✓ Migration tracking table '%s' is ready", mr.TableName)
	return nil
}

// readMigrationFiles loads every migration file in dir, parsed and sorted by
// version. It is the single source of truth for what counts as a migration, so
// the runner and Fingerprint cannot disagree about it.
func readMigrationFiles(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		// Dotfiles are skipped as well. macOS leaves AppleDouble companions
		// (._001_create_database.sql) behind when a repo is copied onto a
		// non-native filesystem, and those end in .sql and contain an
		// underscore, so they would otherwise parse as real migrations.
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Skip files that don't follow the naming convention
		if !strings.Contains(entry.Name(), "_") {
			log.Printf("⚠ Skipping file with invalid naming: %s", entry.Name())
			continue
		}

		// Parse version from filename (e.g., "001_create_database.sql" -> "001")
		parts := strings.SplitN(entry.Name(), "_", 2)
		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".sql")

		// Read file content
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Filename: entry.Name(),
			SQL:      string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// GetAllMigrations reads all migration files from the migrations directory
func (mr *MigrationRunner) GetAllMigrations() ([]Migration, error) {
	return readMigrationFiles(mr.MigrationsPath)
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

	return applied, rows.Err()
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

// runMigration executes a single migration inside a transaction
func (mr *MigrationRunner) runMigration(migration Migration) error {
	log.Printf("→ Running: %s (%s)", migration.Version, migration.Name)

	tx, err := mr.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(migration.SQL); err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	recordQuery := fmt.Sprintf(
		"INSERT INTO %s (version, name, applied_at) VALUES ($1, $2, $3)",
		mr.TableName,
	)
	if _, err := tx.Exec(recordQuery, migration.Version, migration.Name, time.Now()); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

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

// Rollback removes the record of the last applied migration. It does not undo
// the schema change itself — migrations are append-only by design.
func (mr *MigrationRunner) Rollback() error {
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

	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE version = $1", mr.TableName)
	if _, err := mr.DB.Exec(deleteQuery, version); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	log.Printf("✓ Rolled back: %s (%s)", version, name)
	return nil
}

// Reset removes all migration records without touching the schema
func (mr *MigrationRunner) Reset() error {
	log.Println("⚠ WARNING: This will remove all migration records!")
	log.Println("Database structure will remain unchanged.")

	query := fmt.Sprintf("DELETE FROM %s", mr.TableName)
	if _, err := mr.DB.Exec(query); err != nil {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	log.Println("✓ All migration records have been reset")
	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
