.PHONY: help install run build test clean dev
.PHONY: db-create db-drop db-drop-test db-migrate db-setup db-reset db-seed db-rollback db-status db-version db-console db-info
.PHONY: migrate migrate-status migrate-rollback migrate-create build-migrate
.PHONY: seed seed-list seed-rebuild build-seed
.PHONY: swagger swagger-init swagger-update

# Load .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Default target
help:
	@echo "=========================================="
	@echo "  Gatra Budgets Backend - Makefile Commands"
	@echo "=========================================="
	@echo ""
	@echo "📦 Application Commands:"
	@echo "  make install          - Install Go dependencies"
	@echo "  make run              - Run the application"
	@echo "  make build            - Build the application binary"
	@echo "  make dev              - Run with hot reload (requires air)"
	@echo "  make test             - Run all tests (unit + integration)"
	@echo "  make test-unit        - Run unit tests only (no database needed)"
	@echo "  make test-integration - Run integration tests only (needs PostgreSQL)"
	@echo "  make clean            - Clean build artifacts"
	@echo ""
	@echo "🗄️  Database Commands (Rails-style with dash):"
	@echo "  make db-create        - Create database"
	@echo "  make db-drop          - Drop database, also offers to drop the test DB (⚠ dangerous!)"
	@echo "  make db-drop-test     - Drop only the test database (⚠ dangerous!)"
	@echo "  make db-migrate       - Run pending migrations"
	@echo "  make db-setup         - Create + migrate (first time)"
	@echo "  make db-reset         - Drop + create + migrate (⚠ dangerous!)"
	@echo "  make db-seed          - Run all seeders (or NAME=<seeder> for one)"
	@echo "  make db-rollback      - Rollback last migration"
	@echo "  make db-status        - Show migration status"
	@echo "  make db-version       - Show current migration version"
	@echo "  make db-console       - Open PostgreSQL console"
	@echo "  make db-info          - Show database information"
	@echo ""
	@echo "🔄 Alternative Commands:"
	@echo "  make migrate          - Same as db-migrate"
	@echo "  make migrate-status   - Same as db-status"
	@echo "  make migrate-rollback - Same as db-rollback"
	@echo "  make migrate-create NAME=<name> - Create new migration file"
	@echo ""
	@echo "🌱 Database Seeding Commands:"
	@echo "  make db-seed          - Run all seeders"
	@echo "  make db-seed NAME=<seeder> - Run specific seeder (e.g., NAME=address_reference)"
	@echo "  make seed             - Alias for db-seed"
	@echo "  make seed-list        - List available seeders"
	@echo "  make seed-rebuild     - Rebuild seed tool"
	@echo ""
	@echo "📚 Swagger Documentation Commands:"
	@echo "  make swagger          - Generate Swagger documentation"
	@echo "  make swagger-init     - Initialize Swagger (first time)"
	@echo "  make swagger-update   - Update Swagger docs (alias)"
	@echo ""
	@echo "💡 Examples:"
	@echo "  make db-setup                               # First time setup"
	@echo "  make db-migrate                             # Run migrations"
	@echo "  make migrate-create NAME=add_products       # Create new migration"
	@echo "  make db-seed                                # Run all seeders"
	@echo "  make db-seed NAME=address_reference         # Run specific seeder"
	@echo "  make db-drop db-create db-migrate db-seed   # Full reset with seed"
	@echo "  make db-reset                               # Full reset (one command)"
	@echo ""
	@echo "Note: Commands use dash (-) instead of colon (:) for Make compatibility"
	@echo "      Shell scripts (./db.sh) can still use original names"
	@echo ""

# ============================================================================
# Application Commands
# ============================================================================

install:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies installed"

run:
	@echo "🚀 Starting application..."
	go run main.go

build:
	@echo "🔨 Building application..."
	mkdir -p bin
	go build -o bin/app main.go
	@echo "✓ Binary built: bin/app"

# -count=1 on the suites that read the migrations directory.
#
# Go's test cache keys on files the test binary has already opened. Editing an
# existing migration therefore invalidates it, but *adding* a new one does not:
# the binary never opened that path, so the run is served from cache and the new
# migration is never applied. That produces a green tick against a schema that
# is missing the change, which is the worst possible failure mode. The unit
# tests keep their cache because source changes reliably invalidate it.
test:
	@echo "🧪 Running all tests (unit + integration)..."
	go test -count=1 ./internal/... ./tests/...

# Unit tests only: mocked database, no PostgreSQL required.
test-unit:
	@echo "🧪 Running unit tests (no database required)..."
	go test ./internal/... ./tests/handlers/... ./tests/middleware/... ./tests/helpers/... ./tests/services/...

# Integration tests: real PostgreSQL. The suite refuses to run unless the
# target database name contains "_test", so it can never touch dev data.
# The test database is rebuilt automatically whenever the migration files
# change, so a new or edited migration is always tested from a clean schema.
# Pass ARGS for extra flags, e.g. make test-integration ARGS="-v"
test-integration:
	@echo "🧪 Running integration tests (real PostgreSQL)..."
	go test -count=1 ./tests/integration/... $(ARGS)

clean:
	@echo "🧹 Cleaning build artifacts..."
	go clean
	rm -rf bin/
	@echo "✓ Clean completed"

dev:
	@echo "🔥 Starting development server with hot reload..."
	@if ! command -v air > /dev/null; then \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	air

# ============================================================================
# Database Commands (using dash instead of colon for Make compatibility)
# ============================================================================

db-create:
	@chmod +x db.sh
	@./db.sh create

db-drop:
	@chmod +x db.sh
	@./db.sh drop

# Drop only the test database (TEST_DB_NAME, default DB_NAME_test).
# It is recreated automatically on the next `make test-integration`.
db-drop-test:
	@chmod +x db.sh
	@./db.sh drop:test

db-migrate:
	@chmod +x db.sh
	@./db.sh migrate

db-setup:
	@chmod +x db.sh
	@./db.sh setup

db-reset:
	@chmod +x db.sh
	@./db.sh reset

db-seed:
	@chmod +x seed.sh
	@if [ -z "$(NAME)" ]; then \
		./seed.sh all; \
	else \
		./seed.sh $(NAME); \
	fi

db-rollback:
	@chmod +x db.sh
	@./db.sh rollback

db-status:
	@chmod +x db.sh
	@./db.sh status

db-version:
	@chmod +x db.sh
	@./db.sh version

db-console:
	@chmod +x db.sh
	@./db.sh console

db-info:
	@chmod +x db.sh
	@./db.sh info

# ============================================================================
# Alternative Migration Commands (without db- prefix)
# ============================================================================

migrate: db-migrate

migrate-status: db-status

migrate-rollback: db-rollback

migrate-create:
	@chmod +x migrate.sh
	@if [ -z "$(NAME)" ]; then \
		echo "Error: Migration name is required"; \
		echo "Usage: make migrate-create NAME=migration_name"; \
		echo "Example: make migrate-create NAME=insert_jenis_buku_data"; \
		exit 1; \
	fi
	@./migrate.sh create $(NAME)

# ============================================================================
# Database Seeding Commands
# ============================================================================

seed: db-seed

seed-list:
	@chmod +x seed.sh
	@./seed.sh list

seed-rebuild:
	@chmod +x seed.sh
	@./seed.sh rebuild

# ============================================================================
# Build Tools
# ============================================================================

build-migrate:
	@echo "🔨 Building migration tool..."
	mkdir -p bin
	go build -o bin/migrate ./cmd/migrate
	@echo "✓ Migration tool built: bin/migrate"

build-seed:
	@echo "🔨 Building seed tool..."
	mkdir -p bin
	go build -o bin/seed ./cmd/seed/main.go
	@echo "✓ Seed tool built: bin/seed"

# ============================================================================
# Swagger Documentation
# ============================================================================

swagger:
	@echo "📚 Generating Swagger documentation..."
	@if ! command -v swag > /dev/null; then \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	swag init -g main.go --output ./docs
	@echo "✓ Swagger docs generated in ./docs"
	@echo "📖 Access docs at: http://localhost:8080/swagger/index.html"

swagger-init: swagger

swagger-update: swagger
