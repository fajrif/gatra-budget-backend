package migrate

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// stateTableName holds a single row recording the fingerprint of the migration
// files that produced the current schema.
//
// schema_migrations answers "which versions have run?", which is all it needs
// to apply new migrations. It cannot answer "are the files I have on disk still
// the files that built this schema?" — once a version is recorded it is never
// re-run, so editing an applied migration is silently ignored. The fingerprint
// closes that gap.
const stateTableName = "schema_migration_state"

// Fingerprint returns a stable hash of the migration files in dir.
//
// Both the filename and the contents are hashed, in version order, so adding,
// editing, renaming, or deleting any migration changes the result. Identical
// migrations always produce an identical fingerprint, regardless of file
// modification times or the order the filesystem happens to return entries in.
func Fingerprint(dir string) (string, error) {
	migrations, err := readMigrationFiles(dir)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	for _, m := range migrations {
		// The length prefix stops a rename that merely shifts bytes across the
		// filename/body boundary from colliding with the original file.
		fmt.Fprintf(hash, "%s\x00%d\x00", m.Filename, len(m.SQL))
		hash.Write([]byte(m.SQL))
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Drift describes how the migration files on disk relate to what has already
// been applied to the database.
type Drift int

const (
	// DriftNone means the files on disk match the fingerprint recorded in the
	// database, so the schema is in sync with the migrations.
	DriftNone Drift = iota

	// DriftUnknown means no fingerprint has been recorded yet. That is the case
	// for a brand-new database, and for a database populated before this
	// feature existed.
	DriftUnknown

	// DriftChanged means the files no longer match what was applied: they were
	// edited, added, renamed, or removed after the last run. The schema and
	// the migrations now disagree.
	DriftChanged
)

func (d Drift) String() string {
	switch d {
	case DriftNone:
		return "migration files match the database"
	case DriftUnknown:
		return "no migration fingerprint recorded yet"
	default:
		return "migration files changed since they were last applied"
	}
}

// initializeStateTable creates the single-row table that records which
// migration files produced the current schema. The CHECK constraint is what
// keeps it to one row.
func (mr *MigrationRunner) initializeStateTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			fingerprint VARCHAR(64) NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`, stateTableName)

	if _, err := mr.DB.Exec(query); err != nil {
		return fmt.Errorf("failed to create migration state table: %w", err)
	}
	return nil
}

// GetFingerprint returns the fingerprint recorded in the database. A missing
// table or missing row is reported as "" rather than an error, so callers can
// treat "unknown" as the reason to rebuild rather than as a failure.
func (mr *MigrationRunner) GetFingerprint() (string, error) {
	var tableExists bool
	err := mr.DB.QueryRow("SELECT to_regclass($1) IS NOT NULL", stateTableName).Scan(&tableExists)
	if err != nil {
		return "", fmt.Errorf("failed to check for %s: %w", stateTableName, err)
	}
	if !tableExists {
		return "", nil
	}

	var fingerprint string
	query := fmt.Sprintf("SELECT fingerprint FROM %s WHERE id = 1", stateTableName)
	err = mr.DB.QueryRow(query).Scan(&fingerprint)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read recorded fingerprint: %w", err)
	}
	return fingerprint, nil
}

// SetFingerprint records the fingerprint of the migration files now on disk.
func (mr *MigrationRunner) SetFingerprint(fingerprint string) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (id, fingerprint) VALUES (1, $1)
		ON CONFLICT (id) DO UPDATE
			SET fingerprint = EXCLUDED.fingerprint, updated_at = CURRENT_TIMESTAMP
	`, stateTableName)

	if _, err := mr.DB.Exec(query, fingerprint); err != nil {
		return fmt.Errorf("failed to record migration fingerprint: %w", err)
	}
	return nil
}

// DetectDrift compares the migration files on disk against the fingerprint
// recorded in the database.
func (mr *MigrationRunner) DetectDrift() (Drift, error) {
	stored, err := mr.GetFingerprint()
	if err != nil {
		return DriftNone, err
	}
	if stored == "" {
		return DriftUnknown, nil
	}

	current, err := Fingerprint(mr.MigrationsPath)
	if err != nil {
		return DriftNone, err
	}
	if stored != current {
		return DriftChanged, nil
	}
	return DriftNone, nil
}
