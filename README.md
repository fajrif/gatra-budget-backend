# Boilerplates Backend

Go REST API starter kit built with Fiber + GORM + PostgreSQL.

## Quick Start

```bash
cp .env.example .env   # configure DB credentials
make db-setup          # create database and run migrations
make dev               # hot-reload with air (port 8080)
```

Default credentials: `admin@boilerplates.dev / Password123!` and `user@boilerplates.dev / Password123!`

## Commands

```bash
make dev          # hot-reload
make build        # build binary
make test         # run tests
make swagger      # regenerate API docs
make db-migrate   # run pending migrations
make db-reset     # drop + recreate + migrate (destructive)
```

## Stack

- [Go](https://go.dev/) + [Fiber v2](https://gofiber.io/)
- [GORM](https://gorm.io/) ORM with PostgreSQL
- JWT authentication (`golang-jwt/jwt`)
- Swagger docs (`swaggo/swag`)
