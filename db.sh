#!/bin/bash

# Database Management Script
# Comprehensive DB operations like Rails rake commands

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Load .env file if exists
if [ -f .env ]; then
    echo -e "${BLUE}Loading configuration from .env file...${NC}"
    export $(grep -v '^#' .env | xargs)
fi

# Configuration from environment or defaults
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-deployer}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-merk_buku_db}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"

MIGRATIONS_DIR="./migrations"
MIGRATE_BINARY="./bin/migrate"

# Export for psql (important!)
export PGHOST="${DB_HOST}"
export PGPORT="${DB_PORT}"
export PGUSER="${DB_USER}"
export PGPASSWORD="${DB_PASSWORD}"
export PGDATABASE="${DB_NAME}"

# Helper functions
print_header() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Execute psql as postgres superuser
psql_as_postgres() {
    # Always connect to 'postgres' database to avoid "database username does not exist" error
    psql -U "$POSTGRES_USER" -d postgres "$@" 2>/dev/null
}

# Execute psql as regular user
psql_as_user() {
    # Connect to the target database
    if [ -n "$DB_NAME" ]; then
        psql -U "$DB_USER" -d "$DB_NAME" "$@" 2>/dev/null
    else
        psql -U "$DB_USER" -d postgres "$@" 2>/dev/null
    fi
}

# Check if PostgreSQL is running
check_postgres() {
    print_info "Checking PostgreSQL connection..."
    if pg_isready -h $DB_HOST -p $DB_PORT > /dev/null 2>&1; then
        print_success "PostgreSQL is running"
        return 0
    else
        print_error "PostgreSQL is not running or not accessible"
        print_info "Please start PostgreSQL: sudo systemctl start postgresql"
        return 1
    fi
}

# Check if database exists
db_exists() {
    psql -U "$POSTGRES_USER" -d postgres -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw "$DB_NAME"
}

# Check if user exists
user_exists() {
    psql -U "$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" 2>/dev/null | grep -q 1
}

# Ensure migration binary exists
ensure_migrate_binary() {
    if [ ! -f "$MIGRATE_BINARY" ]; then
        print_info "Building migration tool..."
        mkdir -p bin
        go build -o "$MIGRATE_BINARY" migrate.go
        print_success "Migration tool built"
    fi
}

# Create database user if not exists
create_user() {
    print_info "Checking database user..."
    
    if user_exists; then
        print_success "User '$DB_USER' already exists"
    else
        print_info "Creating user '$DB_USER'..."
        # Connect to postgres database explicitly
        psql -U "$POSTGRES_USER" -d postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null || true
        psql -U "$POSTGRES_USER" -d postgres -c "ALTER USER $DB_USER CREATEDB;" 2>/dev/null || true
        print_success "User '$DB_USER' created"
    fi
}

# Create database
db_create() {
    print_header "Creating Database"
    
    check_postgres || exit 1
    create_user
    
    if db_exists; then
        print_warning "Database '$DB_NAME' already exists"
        return 0
    fi
    
    print_info "Creating database '$DB_NAME'..."
    
    # Method 1: Use createdb (this works on macOS!)
    print_info "Using createdb command..."
    if createdb -U "$POSTGRES_USER" -O "$DB_USER" "$DB_NAME" 2>/dev/null; then
        print_success "Database '$DB_NAME' created successfully"
        return 0
    fi
    
    # Method 2: Try psql with explicit postgres database
    print_info "Trying psql method..."
    if psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null; then
        psql -U "$POSTGRES_USER" -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>/dev/null || true
        print_success "Database '$DB_NAME' created successfully"
        return 0
    fi
    
    # Method 3: As deployer user (has CREATEDB)
    print_info "Trying as deployer user..."
    if createdb -U "$DB_USER" "$DB_NAME" 2>/dev/null; then
        print_success "Database '$DB_NAME' created successfully"
        return 0
    fi
    
    print_error "Failed to create database"
    echo ""
    print_info "Please try manually:"
    echo "  createdb -U $POSTGRES_USER -O $DB_USER $DB_NAME"
    exit 1
}

# Drop database
db_drop() {
    print_header "Dropping Database"
    
    check_postgres || exit 1
    
    if ! db_exists; then
        print_warning "Database '$DB_NAME' does not exist"
        return 0
    fi
    
    print_warning "This will permanently delete database '$DB_NAME'"
    read -p "Are you sure? Type 'yes' to confirm: " response
    
    if [ "$response" != "yes" ]; then
        print_info "Drop cancelled"
        return 1
    fi
    
    print_info "Dropping database '$DB_NAME'..."
    
    # Terminate connections
    psql -U "$POSTGRES_USER" -d postgres -c "
        SELECT pg_terminate_backend(pid) 
        FROM pg_stat_activity 
        WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();
    " 2>/dev/null || true
    
    # Drop database using dropdb command (works best on macOS)
    if dropdb -U "$POSTGRES_USER" "$DB_NAME" 2>/dev/null; then
        print_success "Database '$DB_NAME' dropped successfully"
        return 0
    fi
    
    # Fallback to psql
    if psql -U "$POSTGRES_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null; then
        print_success "Database '$DB_NAME' dropped successfully"
        return 0
    fi
    
    print_error "Failed to drop database"
    exit 1
}

# Run migrations
db_migrate() {
    print_header "Running Migrations"
    
    check_postgres || exit 1
    
    if ! db_exists; then
        print_error "Database '$DB_NAME' does not exist"
        print_info "Run: $0 create"
        exit 1
    fi
    
    ensure_migrate_binary
    
    print_info "Running pending migrations..."
    "$MIGRATE_BINARY" migrate
    
    print_success "Migrations completed"
}

# Show migration status
db_status() {
    print_header "Migration Status"
    
    check_postgres || exit 1
    
    if ! db_exists; then
        print_error "Database '$DB_NAME' does not exist"
        exit 1
    fi
    
    ensure_migrate_binary
    "$MIGRATE_BINARY" status
}

# Rollback last migration
db_rollback() {
    print_header "Rolling Back Migration"
    
    check_postgres || exit 1
    
    if ! db_exists; then
        print_error "Database '$DB_NAME' does not exist"
        exit 1
    fi
    
    ensure_migrate_binary
    
    print_warning "This will rollback the last migration"
    read -p "Are you sure? [y/N] " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        "$MIGRATE_BINARY" rollback
        print_success "Rollback completed"
    else
        print_info "Rollback cancelled"
    fi
}

# Reset database (drop + create + migrate)
db_reset() {
    print_header "Resetting Database"
    
    print_warning "This will:"
    echo "  1. Drop database '$DB_NAME'"
    echo "  2. Create database '$DB_NAME'"
    echo "  3. Run all migrations"
    echo ""
    print_warning "ALL DATA WILL BE LOST!"
    echo ""
    
    read -p "Type 'yes' to confirm: " response
    
    if [ "$response" != "yes" ]; then
        print_info "Reset cancelled"
        exit 0
    fi
    
    # Drop
    if db_exists; then
        print_info "Step 1/3: Dropping database..."
        # Terminate connections
        psql -U "$POSTGRES_USER" -d postgres -c "
            SELECT pg_terminate_backend(pid) 
            FROM pg_stat_activity 
            WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();
        " 2>/dev/null || true
        
        # Drop using dropdb command
        if ! dropdb -U "$POSTGRES_USER" "$DB_NAME" 2>/dev/null; then
            # Fallback to psql
            psql -U "$POSTGRES_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null || {
                print_error "Failed to drop database"
                exit 1
            }
        fi
        print_success "Database dropped"
    fi
    
    # Create
    print_info "Step 2/3: Creating database..."
    create_user
    
    # Use createdb command (works on macOS)
    if ! createdb -U "$POSTGRES_USER" -O "$DB_USER" "$DB_NAME" 2>/dev/null; then
        # Fallback to psql
        psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null || {
            print_error "Failed to create database"
            exit 1
        }
        psql -U "$POSTGRES_USER" -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>/dev/null || true
    fi
    print_success "Database created"
    
    # Migrate
    print_info "Step 3/3: Running migrations..."
    ensure_migrate_binary
    "$MIGRATE_BINARY" migrate
    print_success "Migrations completed"
    
    echo ""
    print_success "Database reset completed successfully! 🎉"
}

# Setup database (create + migrate)
db_setup() {
    print_header "Setting Up Database"
    
    check_postgres || exit 1
    
    # Create if not exists
    if ! db_exists; then
        print_info "Step 1/2: Creating database..."
        create_user
        
        # Use createdb (works on macOS)
        if ! createdb -U "$POSTGRES_USER" -O "$DB_USER" "$DB_NAME" 2>/dev/null; then
            # Fallback to psql
            psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null || {
                print_error "Failed to create database"
                exit 1
            }
            psql -U "$POSTGRES_USER" -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>/dev/null || true
        fi
        print_success "Database created"
    else
        print_success "Database already exists"
    fi
    
    # Migrate
    print_info "Step 2/2: Running migrations..."
    ensure_migrate_binary
    "$MIGRATE_BINARY" migrate
    
    echo ""
    print_success "Database setup completed! 🎉"
}

# Show database version
db_version() {
    print_header "Database Version"
    
    check_postgres || exit 1
    
    if ! db_exists; then
        print_error "Database '$DB_NAME' does not exist"
        exit 1
    fi
    
    # Show current migration version
    ensure_migrate_binary
    
    version=$(psql_as_user -d "$DB_NAME" -tAc "
        SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1
    " 2>/dev/null || echo "none")
    
    if [ "$version" = "none" ] || [ -z "$version" ]; then
        print_info "No migrations applied yet"
    else
        print_success "Current migration version: $version"
    fi
}

# Database console
db_console() {
    print_header "Database Console"
    
    check_postgres || exit 1
    
    if ! db_exists; then
        print_error "Database '$DB_NAME' does not exist"
        exit 1
    fi
    
    print_info "Opening PostgreSQL console..."
    print_info "Type \\q to exit"
    echo ""
    
    psql_as_user -d "$DB_NAME"
}

# Show database info
db_info() {
    print_header "Database Information"
    
    echo -e "${BLUE}Configuration:${NC}"
    echo "  Host:     $DB_HOST"
    echo "  Port:     $DB_PORT"
    echo "  User:     $DB_USER"
    echo "  Database: $DB_NAME"
    echo ""
    
    if check_postgres; then
        echo -e "${BLUE}Status:${NC}"
        if db_exists; then
            echo -e "  ${GREEN}✓ Database exists${NC}"
            
            # Get database size
            size=$(psql_as_user -d "$DB_NAME" -tAc "
                SELECT pg_size_pretty(pg_database_size('$DB_NAME'));
            " 2>/dev/null || echo "unknown")
            echo "  Size: $size"
            
            # Get table count
            table_count=$(psql_as_user -d "$DB_NAME" -tAc "
                SELECT COUNT(*) FROM information_schema.tables 
                WHERE table_schema = 'public' AND table_type = 'BASE TABLE';
            " 2>/dev/null || echo "0")
            echo "  Tables: $table_count"
            
            # Get migration version
            version=$(psql_as_user -d "$DB_NAME" -tAc "
                SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1
            " 2>/dev/null || echo "none")
            echo "  Migration: $version"
        else
            echo -e "  ${YELLOW}⚠ Database does not exist${NC}"
        fi
    fi
}

# Show usage
show_usage() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}  Database Management Tool${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
    echo "Usage: $0 <command>"
    echo ""
    echo -e "${GREEN}Main Commands:${NC}"
    echo "  create          Create database"
    echo "  drop            Drop database (⚠ dangerous!)"
    echo "  migrate         Run pending migrations"
    echo "  setup           Create database + run migrations"
    echo "  reset           Drop + create + migrate (⚠ dangerous!)"
    echo ""
    echo -e "${GREEN}Migration Commands:${NC}"
    echo "  status          Show migration status"
    echo "  rollback        Rollback last migration"
    echo "  version         Show current migration version"
    echo ""
    echo -e "${GREEN}Utility Commands:${NC}"
    echo "  console, psql   Open database console"
    echo "  info            Show database information"
    echo ""
    echo -e "${GREEN}Rails-style Shortcuts:${NC}"
    echo "  db:create       Same as 'create'"
    echo "  db:drop         Same as 'drop'"
    echo "  db:migrate      Same as 'migrate'"
    echo "  db:reset        Same as 'reset'"
    echo "  db:setup        Same as 'setup'"
    echo "  db:rollback     Same as 'rollback'"
    echo ""
    echo -e "${BLUE}Examples:${NC}"
    echo "  $0 setup                    # Initial setup"
    echo "  $0 migrate                  # Run migrations"
    echo "  $0 reset                    # Reset database"
    echo "  $0 status                   # Check status"
    echo "  $0 console                  # Open psql"
    echo ""
    echo -e "${BLUE}Rails-style (also works):${NC}"
    echo "  $0 db:drop && $0 db:create && $0 db:migrate"
    echo ""
    echo -e "${YELLOW}Environment Variables:${NC}"
    echo "  DB_HOST         Database host (default: localhost)"
    echo "  DB_PORT         Database port (default: 5432)"
    echo "  DB_USER         Database user (default: deployer)"
    echo "  DB_PASSWORD     Database password"
    echo "  DB_NAME         Database name (default: merk_buku_db)"
    echo "  POSTGRES_USER   Postgres superuser (default: postgres)"
    echo ""
}

# Main command handler
case "${1:-help}" in
    create|db:create)
        db_create
        ;;
    
    drop|db:drop)
        db_drop
        ;;
    
    migrate|db:migrate)
        db_migrate
        ;;
    
    setup|db:setup)
        db_setup
        ;;
    
    reset|db:reset)
        db_reset
        ;;
    
    status|db:status)
        db_status
        ;;
    
    rollback|db:rollback)
        db_rollback
        ;;
    
    version|db:version)
        db_version
        ;;
    
    console|psql|db:console)
        db_console
        ;;
    
    info|db:info)
        db_info
        ;;
    
    help|-h|--help)
        show_usage
        ;;
    
    *)
        echo -e "${RED}Error: Unknown command '$1'${NC}"
        show_usage
        exit 1
        ;;
esac
