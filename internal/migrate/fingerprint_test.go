package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMigrations creates a throwaway migrations directory
func writeMigrations(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}

func fingerprintOf(t *testing.T, files map[string]string) string {
	t.Helper()

	got, err := Fingerprint(writeMigrations(t, files))
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	return got
}

func TestFingerprintIsStable(t *testing.T) {
	files := map[string]string{
		"001_create_database.sql": "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
		"002_add_users.sql":       "ALTER TABLE users ADD COLUMN full_name TEXT;",
	}

	first := fingerprintOf(t, files)
	second := fingerprintOf(t, files)

	if first != second {
		t.Errorf("same migrations produced different fingerprints:\n  %s\n  %s", first, second)
	}
	if len(first) != 64 {
		t.Errorf("fingerprint should be a 64-char hex sha256, got %d chars: %q", len(first), first)
	}
}

// TestFingerprintIgnoresNonMigrations documents that unrelated files in the
// migrations directory must not invalidate the schema, or every README or
// .gitkeep dropped in there would trigger a rebuild.
func TestFingerprintIgnoresNonMigrations(t *testing.T) {
	base := map[string]string{"001_create.sql": "SELECT 1;"}
	before := fingerprintOf(t, base)

	withExtras := map[string]string{
		"001_create.sql":    "SELECT 1;",
		"README.md":         "notes",
		"notes.txt":         "scratch",
		"003.sql":           "SELECT 3;", // no underscore: skipped as malformed
		".hidden_thing.sql": "SELECT 9;",
	}
	after := fingerprintOf(t, withExtras)

	if before != after {
		t.Errorf("non-migration files changed the fingerprint:\n  before %s\n  after  %s", before, after)
	}
}

func TestFingerprintDetectsChanges(t *testing.T) {
	base := map[string]string{
		"001_create_database.sql": "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
		"002_add_users.sql":       "ALTER TABLE users ADD COLUMN full_name TEXT;",
	}
	original := fingerprintOf(t, base)

	tests := []struct {
		name  string
		files map[string]string
	}{
		{
			// The case version-based tracking structurally cannot catch.
			name: "edited body",
			files: map[string]string{
				"001_create_database.sql": "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
				"002_add_users.sql":       "ALTER TABLE users ADD COLUMN full_name VARCHAR(255);",
			},
		},
		{
			// A comment-only change must still count: it changes the file, and
			// pretending otherwise would hide real edits.
			name: "comment-only edit",
			files: map[string]string{
				"001_create_database.sql": "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
				"002_add_users.sql":       "-- new note\nALTER TABLE users ADD COLUMN full_name TEXT;",
			},
		},
		{
			name: "added file",
			files: map[string]string{
				"001_create_database.sql": "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
				"002_add_users.sql":       "ALTER TABLE users ADD COLUMN full_name TEXT;",
				"003_add_photo_url.sql":   "ALTER TABLE users ADD COLUMN photo_url VARCHAR(255);",
			},
		},
		{
			name: "deleted file",
			files: map[string]string{
				"001_create_database.sql": "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
			},
		},
		{
			// What actually happened to 003 during this project.
			name: "renamed file",
			files: map[string]string{
				"001_create_database.sql":    "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
				"002_create_users_index.sql": "CREATE INDEX IF NOT EXISTS idx_users_full_name ON users(full_name);",
			},
		},
		{
			name:  "whitespace-only edit",
			files: map[string]string{"001_create.sql": "SELECT 1;\n\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fingerprintOf(t, tt.files); got == original {
				t.Errorf("%s did not change the fingerprint (%s)", tt.name, original)
			}
		})
	}
}

// TestFingerprintDetectsRename guards the specific regression where a migration
// is renamed but keeps its version: schema_migrations still says the version is
// applied, so only the fingerprint notices.
func TestFingerprintDetectsRename(t *testing.T) {
	body := "CREATE INDEX IF NOT EXISTS idx_users_full_name ON users(full_name);"

	before := fingerprintOf(t, map[string]string{"003_insert_users_data.sql": body})
	after := fingerprintOf(t, map[string]string{"003_create_users_full_name_index.sql": body})

	if before == after {
		t.Error("renaming a migration kept the same fingerprint, so the change would go unnoticed")
	}
}

func TestFingerprintEmptyDirectory(t *testing.T) {
	empty, err := Fingerprint(t.TempDir())
	if err != nil {
		t.Fatalf("Fingerprint on empty dir: %v", err)
	}

	// The hash of no input, so an empty directory is well-defined rather than an
	// error the caller has to special-case.
	if len(empty) != 64 {
		t.Errorf("expected 64-char hash for empty dir, got %q", empty)
	}
}

func TestFingerprintMissingDirectory(t *testing.T) {
	if _, err := Fingerprint(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected an error for a missing migrations directory")
	}
}
