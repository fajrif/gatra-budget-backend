#!/bin/bash

# Database Migration Script
# Wrapper for Go migration tool with friendly interface

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Shared helpers (safe .env loading, staleness-aware builds)
# shellcheck source=scripts/common.sh
. "$(dirname "$0")/scripts/common.sh"

load_env_file

# Configuration
MIGRATE_BINARY="./bin/migrate"
MIGRATIONS_DIR="./migrations"
# The CLI lives in cmd/migrate since the migration logic moved into
# internal/migrate; this path used to point at the deleted root migrate.go.
GO_FILE="./cmd/migrate/main.go"

# Build the migration tool unless the existing binary is already up to date
check_migrate_binary() {
  build_if_stale "$MIGRATE_BINARY" "$GO_FILE" || exit 1
  [ "$BUILD_REBUILT" = "1" ] && echo -e "${GREEN}✓ Migration tool built${NC}"
  return 0
}

# Show usage
show_usage() {
  echo "=========================================="
  echo "  Database Migration Tool"
  echo "=========================================="
  echo ""
  echo "Usage: $0 <command> [options]"
  echo ""
  echo "Commands:"
  echo "  migrate, up          Run all pending migrations"
  echo "  status, st           Show migration status"
  echo "  rollback, down       Rollback the last migration"
  echo "  reset                Reset all migration records (⚠ dangerous!)"
  echo "  create <name>        Create a new migration file"
  echo "  rebuild              Rebuild migration tool"
  echo "  help, -h, --help     Show this help message"
  echo ""
  echo "Examples:"
  echo "  $0 migrate                    Run pending migrations"
  echo "  $0 status                     Check migration status"
  echo "  $0 create add_users_table     Create new migration"
  echo "  $0 rollback                   Rollback last migration"
  echo ""
  echo "Environment Variables:"
  echo "  DB_HOST         Database host (default: localhost)"
  echo "  DB_PORT         Database port (default: 5432)"
  echo "  DB_USER         Database user (default: deployer)"
  echo "  DB_PASSWORD     Database password"
  echo "  DB_NAME         Database name (default: gatra_budget_db)"
  echo ""
}

# Create new migration file
create_migration() {
  if [ -z "$1" ]; then
    echo -e "${RED}Error: Migration name is required${NC}"
    echo "Usage: $0 create <migration_name>"
    echo "Example: $0 create add_users_table"
    exit 1
  fi

  migration_name="$1"

  # Get next version number
  version=$(ls -1 "$MIGRATIONS_DIR"/*.sql 2>/dev/null | wc -l | xargs printf "%03d")
  version=$(expr $version + 1 | xargs printf "%03d")

  # Create filename
  filename="$MIGRATIONS_DIR/${version}_${migration_name}.sql"

  # Create migration file with template
  cat >"$filename" <<EOF
-- Migration: ${migration_name}
-- Description: [Add description here]
-- Created: $(date +"%Y-%m-%d %H:%M:%S")

-- Ensure UUID extension is enabled (if needed)
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Write your SQL here


-- Add indexes if needed
-- CREATE INDEX IF NOT EXISTS idx_table_column ON table_name(column_name);

-- Add comments
-- COMMENT ON TABLE table_name IS 'Description';
-- COMMENT ON COLUMN table_name.column_name IS 'Description';
EOF

  echo -e "${GREEN}✓ Created migration file:${NC}"
  echo -e "${BLUE}  $filename${NC}"
  echo ""
  echo "Edit the file to add your SQL statements, then run:"
  echo -e "${YELLOW}  $0 migrate${NC}"
}

# Run migrations
run_migrate() {
  check_migrate_binary
  echo -e "${BLUE}Running migrations...${NC}"
  echo ""
  "$MIGRATE_BINARY" migrate
}

# Show status
show_status() {
  check_migrate_binary
  echo -e "${BLUE}Migration Status:${NC}"
  echo ""
  "$MIGRATE_BINARY" status
}

# Rollback migration
rollback_migration() {
  check_migrate_binary
  echo -e "${YELLOW}⚠ Warning: This will rollback the last migration${NC}"
  echo -e "${YELLOW}Note: Only the migration record will be removed.${NC}"
  echo -e "${YELLOW}You may need to manually revert database changes.${NC}"
  echo ""
  read -p "Are you sure? [y/N] " -n 1 -r
  echo ""

  if [[ $REPLY =~ ^[Yy]$ ]]; then
    "$MIGRATE_BINARY" rollback
  else
    echo "Rollback cancelled"
  fi
}

# Reset migrations
reset_migrations() {
  check_migrate_binary
  echo -e "${RED}⚠ WARNING: This will remove ALL migration records!${NC}"
  echo -e "${YELLOW}Database structure will remain unchanged.${NC}"
  echo -e "${YELLOW}This is a dangerous operation!${NC}"
  echo ""
  read -p "Type 'yes' to confirm: " response

  if [ "$response" = "yes" ]; then
    "$MIGRATE_BINARY" reset
  else
    echo "Reset cancelled"
  fi
}

# Rebuild migration tool
rebuild_tool() {
  echo -e "${BLUE}Rebuilding migration tool...${NC}"
  rm -f "$MIGRATE_BINARY"
  mkdir -p bin
  go build -o "$MIGRATE_BINARY" "$GO_FILE"
  echo -e "${GREEN}✓ Migration tool rebuilt${NC}"
}

# Main command handler
case "${1:-help}" in
migrate | up)
  run_migrate
  ;;

status | st)
  show_status
  ;;

rollback | down)
  rollback_migration
  ;;

reset)
  reset_migrations
  ;;

create)
  create_migration "$2"
  ;;

rebuild)
  rebuild_tool
  ;;

help | -h | --help)
  show_usage
  ;;

*)
  echo -e "${RED}Error: Unknown command '$1'${NC}"
  echo ""
  show_usage
  exit 1
  ;;
esac
