// Command migrate applies pending SQL migrations to the configured database.
//
// The migration logic lives in internal/migrate so that test packages can
// reuse it. This file only wires up configuration, the connection, and the CLI.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gatra-budget-backend/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	migrationsPath := "./migrations"
	if command == "-path" {
		if len(os.Args) < 4 {
			printUsage()
			os.Exit(1)
		}
		migrationsPath = os.Args[2]
		command = os.Args[3]
	}

	cfg := migrate.LoadConfig()

	db, err := migrate.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Printf("✓ Connected to database '%s'", cfg.DBName)

	runner := migrate.NewMigrationRunner(db, migrationsPath)

	if err := runner.Initialize(); err != nil {
		log.Fatal(err)
	}

	warnOnDrift(runner)

	switch command {
	case "migrate", "up":
		if err := runMigrate(runner, migrationsPath); err != nil {
			log.Fatal("Migration failed: ", err)
		}

	case "status":
		if err := runner.Status(); err != nil {
			log.Fatal("Failed to get status: ", err)
		}

	case "rollback", "down":
		if err := runner.Rollback(); err != nil {
			log.Fatal("Rollback failed: ", err)
		}

	case "reset":
		if err := runner.Reset(); err != nil {
			log.Fatal("Reset failed: ", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// runMigrate applies pending migrations and, when this run built the whole
// schema from the current files, records a fingerprint of those files so later
// runs can tell whether the migrations have been edited since.
func runMigrate(runner *migrate.MigrationRunner, migrationsPath string) error {
	// A database that already has versions applied is not recorded, because
	// this tool cannot tell whether its existing schema matches the files.
	// Blessing it would hide real drift.
	preExisting := false
	if applied, err := runner.GetAppliedMigrations(); err == nil {
		preExisting = len(applied) > 0
	}

	if err := runner.Migrate(); err != nil {
		return err
	}

	if preExisting {
		return nil
	}

	fingerprint, err := migrate.Fingerprint(migrationsPath)
	if err != nil {
		return nil
	}
	if err := runner.SetFingerprint(fingerprint); err != nil {
		log.Printf("⚠ Could not record migration fingerprint: %v", err)
	}
	return nil
}

// warnOnDrift reports when the migration files no longer match what was applied
// to this database.
//
// Unlike the test suite, this database keeps its schema between runs, and an
// already-applied version is never re-run. So if a migration was edited,
// renamed, or deleted after it ran, `migrate` will correctly report nothing to
// do while the schema quietly keeps the old version. Saying so here is the
// difference between noticing now and debugging it a week later.
func warnOnDrift(runner *migrate.MigrationRunner) {
	drift, err := runner.DetectDrift()
	if err != nil {
		log.Printf("⚠ Could not check the migration fingerprint: %v", err)
		return
	}

	if drift != migrate.DriftChanged {
		if drift == migrate.DriftUnknown {
			// A database built before the fingerprint existed cannot be checked,
			// so say so instead of appearing healthy. The fix is a reset, which
			// rebuilds the schema from the current files and records them.
			log.Println("ℹ Drift checking is not active for this database: it was")
			log.Println("  populated before migration fingerprints were recorded.")
			log.Println("  Run './db.sh db:reset' to rebuild it from the current files.")
		}
		return
	}

	log.Println(strings.Repeat("!", 72))
	log.Println("! WARNING: the migration files have changed since they were applied")
	log.Println("!")
	log.Println("! This database already has versions in schema_migrations, so they")
	log.Println("! will never run again. If you edited or removed one of those files,")
	log.Println("! `migrate` will report nothing to do while the schema keeps the old")
	log.Println("! version — your files and your database now disagree.")
	log.Println("!")
	log.Println("! Add a NEW migration file instead of editing an applied one.")
	log.Println("! For a disposable database, './db.sh db:reset' rebuilds it from the")
	log.Println("! current files.")
	log.Println(strings.Repeat("!", 72))
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
	fmt.Println("Environment: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE")
}
