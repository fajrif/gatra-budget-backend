# Gatra Budget Backend

Go REST API starter kit built with Fiber + GORM + PostgreSQL.

## Quick Start

```bash
cp .env.example .env   # configure DB credentials
make db-setup          # create database and run migrations
make db-seed           # load the default users + address reference data
make dev               # hot-reload with air (port 8080)
```

Default credentials (created by `make db-seed NAME=users`): `admin@gatrahitawasana.com / Password123!`

## Commands

```bash
make dev          # hot-reload
make build        # build binary
make test         # run all tests (unit + integration)
make test-unit    # unit tests only, no database required
make test-integration # integration tests, real PostgreSQL
make swagger      # regenerate API docs
make db-migrate   # run pending migrations
make db-seed      # run all seeders (or NAME=<seeder> for one)
make db-drop-test # drop only the test database
make db-reset     # drop + recreate + migrate (destructive)
```

## Testing

Tests are layered:

- **Unit tests** (`tests/handlers`, `tests/middleware`, `tests/helpers`) use
  `sqlmock`. Every mock asserts its expectations are met, so an unexercised
  query fails the test.
- **Integration tests** (`tests/integration`) run against a real PostgreSQL
  database and cover what a mock cannot: unique constraints, foreign-key
  cascades, real SQL, and the full auth path.

Integration tests use a **separate** database — by default `DB_NAME` with a
`_test` suffix, so `gatra_budget_db` → `gatra_budget_db_test` (override with
`TEST_DB_NAME`) — and each test runs inside a transaction that is always rolled
back. The suite refuses to start unless the target database name contains
`_test`, so it cannot modify development data.

### Migrations are applied on every run

`make test` brings the test database in line with the files in `migrations/`.
If those files have **changed** since the database was last built — added, edited,
renamed, or deleted — the test database is dropped and rebuilt from scratch, so
tests always run against the migrations exactly as written:

```bash
$ vim migrations/009_add_user_photo_url.sql   # or add 010_...
$ make test
⚠ migration files changed since they were last applied: rebuilding test database
  gatra_budget_db_test from scratch
```

This matters because applied migrations are never re-run. Without the rebuild, an
edited migration would be silently ignored and the suite would pass against a
schema that no longer matches your files.

Your **development** database is not rebuilt this way — it keeps its data. Editing
an already-applied migration there is not picked up; `make db-migrate` prints a
warning saying so. Add a new migration file rather than editing an applied one.

`./db.sh drop` and `./db.sh reset` also offer to drop the test database,
answering the prompt with anything other than `y` keeps it. `make db-drop-test`
drops only the test database. It is recreated automatically the next time the
integration suite runs.

## Stack

- [Go](https://go.dev/) + [Fiber v2](https://gofiber.io/)
- [GORM](https://gorm.io/) ORM with PostgreSQL
- JWT authentication (`golang-jwt/jwt`)
- Swagger docs (`swaggo/swag`)
