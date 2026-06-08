# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Commands

```bash
# Development
make local-run       # Start API + PostgreSQL + Redis via Docker Compose with .env.local
make lint            # Run golangci-lint run ./...
make test            # go test -v ./...
make test-cover      # Generate coverage.html

# Code generation (run after changing SQL queries or adding new domains)
make sqlc            # Regenerate type-safe SQL code from .sql files → internal/database/postgresql/gen/
make create-swagger  # Regenerate Swagger docs from handler annotations
make mocks           # Regenerate mocks via mockery
make domain          # Scaffold a new domain skeleton from DB schema
make proto           # Generate gRPC code from .proto files

# Build
make build           # Build production Docker image (linux/amd64)
```

Hot reload in local dev is provided by `air` (configured in `cmd/api/.air.toml`).

## Architecture

This project follows **Clean Architecture** with a DDD-inspired domain layer. Dependencies flow inward: HTTP → Service → Repository.

### Layers

**`cmd/api/main.go`** — Entry point. Wires all dependencies manually (no DI framework): logger → DB → transaction manager → repositories → services → HTTP router → server.

**`domain/`** — Core business logic. Each domain (e.g., `user`, `errorlog`) contains:
- `<domain>.go` — interfaces (Repository, UseCase) and models
- `error.go` — domain-specific errors mapped from PostgreSQL error codes
- `service.go` — UseCase implementation (business logic, bcrypt, JWT calls)
- `postgresql/storage.go` — Repository implementation using sqlc-generated queries
- `postgresql/query.sql` — Named SQL queries consumed by `make sqlc`

**`internal/`** — Shared infrastructure adapters:
- `database/postgresql/` — DB connection with retry logic, context-based transaction manager (`transaction.go`), pagination cursor encryption
- `grpc/pb/` — Generated gRPC code from protobuf definitions
- `http/chi/` — Chi router setup, route groups (system/public/app/admin), handler files per domain
- `http/chi/middleware/` — CORS, security headers, rate limiting (30 req/s), panic recovery, request logging, JWT auth, cookie auth
- `http/response.go` — Unified JSON response envelope: `{ code, message, error, data }`
- `jwt/` — HS256 JWT with custom claims (UserID, Role)
- `logger/` — Interface with zerolog implementation (console+color in dev, JSON in prod)
- `validator/` — `go-playground/validator` wrapper with translation support

**`config/config.go`** — All configuration loaded from environment variables (no config files at runtime). See `.env.template` for required vars.

**`ops/db/init.sql`** — PostgreSQL schema. Two schemas: `v1` (API) and `admin`. Run automatically when using `make local-run`.

### Key Patterns

**Transactions**: Use `TransactionManager` from `internal/database/postgresql/transaction.go`. Pass `context.Context` through the call chain; the transaction manager attaches the `*sql.Tx` to the context so nested calls reuse it automatically.

**SQL queries**: Never write raw SQL in Go files. Add named queries to `domain/<name>/postgresql/query.sql`, run `make sqlc`, then call the generated querier from `storage.go`.

**Error handling**: Map PostgreSQL error codes to domain errors in `domain/<name>/error.go`. Handlers convert domain errors to HTTP responses using `internal/http/response.go` helpers.

**New domain**: Run `make domain` to scaffold the skeleton, then implement the interfaces defined in `<domain>.go`.

**Swagger**: Annotate handlers with `swaggo` comments, then run `make create-swagger`. Swagger UI is only mounted in dev stage, behind basic auth (`SWAGGER_ID`/`SWAGGER_PASSWORD`).

**gRPC**: Define service definitions in `proto/<domain>/v1/<domain>.proto`, run `make proto` to generate Go code in `internal/grpc/pb/`, then implement the generated service interfaces.

### Environment

- `STAGE=dev` enables debug logging, relaxed CORS, Swagger UI, and disables background jobs
- Required env vars are documented in `.env.template`
- Local dev uses `.env.local`; Docker dev container uses `.env.development`
