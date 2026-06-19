# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Development
```bash
# Hot reload (air)
go tool air -c ./cmd/api/.air.toml

# Run directly
go run ./cmd/api

# Full stack (API + PostgreSQL + Redis + Prometheus + Grafana)
make local-run

# DB + Redis only (run app locally)
docker compose --env-file .env.local up -d db cache_db
```

### Build & Test
```bash
make build        # Production Docker image (linux/amd64)
make test         # All tests with verbose output
make test-cover   # HTML coverage report
make lint         # golangci-lint (errcheck, goconst, govet, staticcheck, unused)
```

### Code Generation
```bash
make sqlc           # Generate DB queries from domain/*/postgresql/query.sql
make proto          # Generate protobuf from proto/user/v1/user.proto
make create-swagger # Generate Swagger docs from internal/http/chi/api_handler.go
make tools-upgrade  # Install/upgrade air, swag, sqlc
```

## Architecture

Dual-protocol server: **HTTP REST** (Chi v5) + **gRPC** (port 8888). Entry point is `cmd/api/main.go`.

### Layer Structure

```
cmd/api/           → DI wiring, server startup
config/            → env-based config (os.Getenv)
domain/<name>/     → service interface, entity types, domain errors
  postgresql/      → repository implementation (sqlc-generated queries)
internal/
  http/chi/        → HTTP handlers + route registration
    middleware/    → CORS, auth, logging, metrics, rate limit, recovery
  grpc/<name>/     → gRPC service implementations
  database/postgresql/ → connection, transaction manager, cursor encryption
    gen/           → sqlc-generated query structs
  jwt/             → HS256 token generation/validation
  metrics/         → Prometheus registries (HTTP, DB, per-domain)
  logger/zerolog/  → structured logging (console dev, JSON prod)
ops/
  db/init.sql      → schema
  prometheus/      → scrape config (5s interval)
  grafana/         → dashboards
proto/user/v1/     → protobuf definitions
```

### Key Patterns

**Dependency Injection**: Constructor-based, no framework. `main.go` wires: config → logger → DB → repos → services → handlers → router.

**Repository pattern**: Each domain defines `Reader`, `Writer`, and `Repository` interfaces in `domain/<name>/<name>.go`. PostgreSQL implementations live in `domain/<name>/postgresql/storage.go` and use sqlc-generated queries via `queryRower.Queries.*`.

**Transaction support**: `WithTransaction(ctx, isolationLevel, readOnly, fn)` wraps operations in a DB transaction and injects the tx into context. Repositories call `db.GetQueryRowerFromContext(ctx)` to pick up the transaction automatically.

**Route groups** (`internal/http/chi/api_handler.go`):
- `/` — health, swagger, metrics (public)
- `/api/v1` — public endpoints (e.g. user registration)
- `/api/v1/app` — authenticated endpoints (JWT middleware applied)
- `/admin` — reserved

**Error mapping**: Domain errors are defined in `domain/<name>/error.go`; PostgreSQL error codes are mapped to domain errors in the repository layer. HTTP handlers switch on error type to produce the correct status code.

**Cursor-based pagination**: Cursors are AES-encrypted (`internal/database/postgresql/cursor.go`) before being sent to clients.

### Middleware Stack (applied in order)

`RequestID` → `RealIP` → `CORSMiddleware` → `SecurityHeadersMiddleware` → `BodyLimitMiddleware` (10 MB) → `RateLimitMiddleware` (30 req/s per IP) → `RecoveryMiddleware` → `LoggerMiddleware` → `MetricsMiddleware`

`LoggerMiddleware` skips health/metrics/swagger paths, flags slow requests (>1s), detects scanning probes, and persists errors (HTTP ≥ 400, excluding 401/403/404/405/503) to the `v1.error_log` table.

### Auth

JWT (HS256) stored in an `HttpOnly` cookie (`SameSite=Lax`, `Secure` in prod). Claims carry `user_id` and `role`. `RequireRole()` middleware enforces RBAC. Token default expiry: 1 hour.

### Monitoring

Prometheus metrics exposed at `/metrics`. Per-domain metric files (`domain/<name>/metrics.go`) register domain-level counters. DB connection pool stats are scraped every 10 seconds.

### Database

PostgreSQL with exponential-backoff connection (500 ms → 60 s, 10 attempts). Pool: 15 idle / 30 max open. SSL disabled in dev, required in prod (driven by `STAGE` env var).

### gRPC

Started in a goroutine from `main.go`. Port controlled by `GRPC_PORT`. TLS and JWT metadata auth are not yet implemented (noted as TODO in `internal/api/grpc.go`).

## Environment Variables

Copy `.env.template` to `.env.local`. Key variables:

| Variable | Purpose |
|----------|---------|
| `STAGE` | `dev` / `staging` / `prod` |
| `API_HOST`, `API_PORT` | HTTP server bind |
| `GRPC_PORT` | gRPC server port |
| `DB_*` | PostgreSQL connection |
| `REDIS_HOST`, `REDIS_PORT` | Redis connection |
| `API_SECRET_KEY` | JWT signing key (≥32 chars) |
| `CURSOR_SECRET` | Cursor encryption key (exactly 32 chars) |
| `API_X_API_KEY` | Required `X-API-Key` header |
| `SWAGGER_ID`, `SWAGGER_PASSWORD` | Basic auth for `/swagger/*` |

## 새 도메인 추가 순서

1. `domain/<name>/` 폴더 생성
   - `<name>.go` - Repository/UseCase 인터페이스 + Model struct
   - `service.go` - UseCase 구현체
   - `error.go` - 도메인 에러 (PostgreSQL 에러코드 매핑)
   - `postgresql/storage.go` - Repository 구현체
   - `postgresql/query.sql` - SQL 쿼리 (make sqlc로 생성)

2. `internal/http/chi/<name>.go` 생성
   - Handler struct + 생성자 함수 (UseCase, validator, logger 주입)
   - 각 엔드포인트 함수 (Swagger 주석 포함)

3. `cmd/api/main.go` 수정
   - storage → service → handler 순서로 wiring 추가

## 금지 사항

- Go 파일 안에 raw SQL 작성 금지 → 반드시 query.sql에 작성 후 make sqlc
- DI 프레임워크 사용 금지 → cmd/api/main.go에서 수동 wiring
- 도메인 레이어에서 외부 패키지 직접 임포트 금지 (인터페이스로 추상화)

## 트랜잭션 사용법

TransactionManager를 context에 붙여서 전달한다.
service.go에서 직접 Begin/Commit 하지 않음.

// 올바른 방법
func (s *service) SomeOperation(ctx context.Context) error {
    return s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // ctx 안에 tx가 들어있음
        if err := s.repo.DoA(ctx); err != nil { return err }
        return s.repo.DoB(ctx)
    })
}