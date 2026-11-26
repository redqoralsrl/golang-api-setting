---
name: add-domain
description: Add a new domain to this Go API project end to end - goose migration, sqlc queries, domain model/errors/service, repository, HTTP handler with Swagger comments, and wiring in main.go and the router. Produces code that builds and passes tests. Usage: /add-domain <domain-name> [field:type ...]
tools: Read, Write, Edit, Glob, Grep, Bash
---

# Add Domain Skill

Create a complete, compiling domain with Create and Get endpoints.

**Arguments**: `$ARGUMENTS` — first token is the domain name (singular, lowercase, e.g. `order`). Optional extra tokens are columns as `name:sqltype` (e.g. `title:text amount:bigint`). Without them, add one `name text not null` column.

Derive and use literally (never write placeholders like `<Pascal>` into files):
- **domain**: lowercase name (`order`)
- **Pascal**: PascalCase (`Order`)
- **table**: `v1.<domain>s` (`v1.orders`)
- **Row**: sqlc model name for the table, `V1<Pascal>` (`V1Order`)

Read `CLAUDE.md` first. Follow its rules (no raw SQL in Go, domain imports no HTTP/DB implementation packages, never edit `gen/`).

---

## Step 0: Validate

If `$ARGUMENTS` is blank, print `Usage: /add-domain <domain-name> [field:type ...]` with the example `/add-domain order title:text amount:bigint` and stop.
If `domain/<domain>` already exists, stop and tell the user.

## Step 1: Migration

```bash
go run ./cmd/migrate create create_<domain>s
```

Fill the new file in `migrations/`:

```sql
-- +goose Up
create table v1.<domain>s (
    id bigint generated always as identity primary key,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    -- columns from arguments, e.g.
    name text not null
);

create trigger <domain>s_set_updated_at before update on v1.<domain>s
    for each row execute function v1.set_updated_at();

comment on table v1.<domain>s is '<domain> 테이블';

-- +goose Down
drop table if exists v1.<domain>s;
```

Type rules: strings `text`, integers `bigint` (or `integer`), money/amount `numeric`, time `timestamptz`, flags `boolean not null default false`. Foreign keys: `<x>_id bigint not null references v1.<x>s (id)` plus an index.

## Step 2: Queries + sqlc

Create `domain/<domain>/postgresql/query.sql`:

```sql
-- name: Create<Pascal> :one
insert into v1.<domain>s (name)
values (sqlc.arg(name))
returning *;

-- name: Get<Pascal>ByID :one
select * from v1.<domain>s
where id = sqlc.arg(id);
```

Add `- "./domain/<domain>/postgresql/query.sql"` to `queries:` in `sqlc.yaml`, then:

```bash
go tool sqlc generate
```

Read `internal/database/postgresql/gen/models.go` and `query.sql.go` to get the exact generated names (`gen.V1<Pascal>`, `gen.Create<Pascal>Params`, `gen.Get<Pascal>ByIDParams`) and field types. Use them as generated.

## Step 3: Domain files

### `domain/<domain>/<domain>.go`

```go
package <domain>

import (
	"context"
	"time"
)

type <Pascal> struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
}

type CreateInput struct {
	Name string
}

type Reader interface {
	GetByID(ctx context.Context, id int64) (*<Pascal>, error)
}

type Writer interface {
	Create(ctx context.Context, in CreateInput) (*<Pascal>, error)
}

type Repository interface {
	Reader
	Writer
}

type UseCase interface {
	Create(ctx context.Context, in CreateInput) (*<Pascal>, error)
	Get(ctx context.Context, id int64) (*<Pascal>, error)
}
```

### `domain/<domain>/error.go`

```go
package <domain>

import (
	"errors"
	"fmt"
)

// 도메인 에러. 문구는 4xx 응답으로 사용자에게 보인다.
var (
	Err<Pascal>NotFound      = errors.New("<domain> not found")
	Err<Pascal>AlreadyExists = errors.New("<domain> already exists")
)

func NewDatabaseError(operation string, cause error) error {
	return fmt.Errorf("database operation %q failed: %w", operation, cause)
}
```

### `domain/<domain>/service.go`

```go
package <domain>

import (
	"context"

	"go-template/internal/database"
	"go-template/internal/logger"
)

type Service struct {
	repo      Repository
	txManager database.TxManager
	logger    logger.Logger
}

var _ UseCase = (*Service)(nil)

func NewService(r Repository, tm database.TxManager, l logger.Logger) *Service {
	return &Service{repo: r, txManager: tm, logger: l}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*<Pascal>, error) {
	return s.repo.Create(ctx, in)
}

func (s *Service) Get(ctx context.Context, id int64) (*<Pascal>, error) {
	return s.repo.GetByID(ctx, id)
}
```

Use `s.txManager.WithTransaction(ctx, func(ctx context.Context) error { ... })` only when one operation writes more than once.

### `domain/<domain>/postgresql/storage.go`

```go
package postgresql

import (
	"context"
	"errors"

	"go-template/domain/<domain>"
	"go-template/internal/database"
	"go-template/internal/database/postgresql"
	"go-template/internal/database/postgresql/gen"
)

type <Pascal>Storage struct {
	db *postgresql.Database
}

var _ <domain>.Repository = (*<Pascal>Storage)(nil)

func New<Pascal>(db *postgresql.Database) *<Pascal>Storage {
	return &<Pascal>Storage{db: db}
}

func (s *<Pascal>Storage) Create(ctx context.Context, in <domain>.CreateInput) (*<domain>.<Pascal>, error) {
	row, err := s.db.Queries(ctx).Create<Pascal>(ctx, gen.Create<Pascal>Params{Name: in.Name})
	if err != nil {
		return nil, mapError(err)
	}
	return to<Pascal>(row), nil
}

func (s *<Pascal>Storage) GetByID(ctx context.Context, id int64) (*<domain>.<Pascal>, error) {
	row, err := s.db.Queries(ctx).Get<Pascal>ByID(ctx, gen.Get<Pascal>ByIDParams{ID: id})
	if err != nil {
		return nil, mapError(err)
	}
	return to<Pascal>(row), nil
}

func mapError(err error) error {
	err = postgresql.MapError(err)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, database.ErrNotFound):
		return <domain>.Err<Pascal>NotFound
	case errors.Is(err, database.ErrUniqueViolation):
		return <domain>.Err<Pascal>AlreadyExists
	default:
		return <domain>.NewDatabaseError("<domain>", err)
	}
}

func to<Pascal>(row gen.V1<Pascal>) *<domain>.<Pascal> {
	return &<domain>.<Pascal>{
		ID:        row.ID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Name:      row.Name,
	}
}
```

Nullable columns come out of sqlc as pointers (`*string`, `*time.Time`); map them to pointer fields in the domain model.

## Step 4: HTTP handler — `internal/http/chi/<domain>.go`

```go
package chi

import (
	"errors"
	"net/http"
	"strconv"

	"go-template/domain/<domain>"
	response "go-template/internal/http"
	"go-template/internal/logger"
	"go-template/internal/validator"

	"github.com/go-chi/chi/v5"
)

type <domain>Handler struct {
	useCase   <domain>.UseCase
	validator validator.ValidationService
	logger    logger.Logger
}

func <Pascal>Handler(s <domain>.UseCase, handler Handler, l logger.Logger) {
	h := &<domain>Handler{useCase: s, validator: handler.v, logger: l}

	handler.Routes.App.Post("/<domain>", h.Create)
	handler.Routes.App.Get("/<domain>/{id}", h.Get)
}

type Create<Pascal>Request struct {
	Name string `json:"name" validate:"required,max=255"`
}

// Create godoc
// @Summary Create <domain>
// @Tags <Pascal>s
// @Accept json
// @Produce json
// @Param request body Create<Pascal>Request true "Create <domain> request"
// @Success 201 {object} response.Response{data=<domain>.<Pascal>}
// @Failure 400 {object} response.Response "invalid input"
// @Failure 401 {object} response.Response "unauthorized"
// @Failure 409 {object} response.Response "<domain> already exists"
// @Failure 500 {object} response.Response "internal server error"
// @Router /app/<domain> [post]
func (h *<domain>Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req Create<Pascal>Request
	if !bindJSON(w, r, h.validator, &req) {
		return
	}

	result, err := h.useCase.Create(r.Context(), <domain>.CreateInput{Name: req.Name})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.WriteResponse(w, http.StatusCreated, response.Success, nil, result)
}

// Get godoc
// @Summary Get <domain>
// @Tags <Pascal>s
// @Produce json
// @Param id path int true "<domain> id"
// @Success 200 {object} response.Response{data=<domain>.<Pascal>}
// @Failure 400 {object} response.Response "invalid id"
// @Failure 404 {object} response.Response "<domain> not found"
// @Router /app/<domain>/{id} [get]
func (h *<domain>Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, errors.New("invalid id"))
		return
	}

	result, err := h.useCase.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.WriteResponse(w, http.StatusOK, response.Success, nil, result)
}

// writeError 도메인 에러 → status. 모르는 에러는 5xx 로 로그를 남긴다.
func (h *<domain>Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, <domain>.Err<Pascal>NotFound):
		response.WriteError(w, http.StatusNotFound, response.NotFound, <domain>.Err<Pascal>NotFound)
	case errors.Is(err, <domain>.Err<Pascal>AlreadyExists):
		response.WriteError(w, http.StatusConflict, response.Conflict, <domain>.Err<Pascal>AlreadyExists)
	default:
		response.WriteErrorWithRequest(w, r, h.logger, http.StatusInternalServerError, response.InternalError, err)
	}
}
```

Pick the route group by need: `Routes.Public` (no auth), `Routes.App` (user JWT), `Routes.AppUpload` (20MB body). Never register on `Routes.Admin` until admin auth exists. Swagger `@Router` paths are relative to `/api/v1`.

## Step 5: Wiring

1. `internal/http/chi/handler.go` — import `go-template/domain/<domain>`, add `<Pascal>Service <domain>.UseCase` to `Services`.
2. `internal/http/chi/api_handler.go` — under `// 도메인 핸들러 등록` add `<Pascal>Handler(services.<Pascal>Service, handler, l)`.
3. `cmd/api/main.go` — import `"go-template/domain/<domain>"` and `<domain>Postgresql "go-template/domain/<domain>/postgresql"`, then add in the matching sections:
   ```go
   <domain>Storage := <domain>Postgresql.New<Pascal>(db)            // setup Storage
   <domain>Service := <domain>.NewService(<domain>Storage, transactionManager, l) // setup Service
   <Pascal>Service: <domain>Service,                                 // stdHandler.Services{...}
   ```
4. `internal/http/chi/router_test.go` — `Services` literal still compiles with the new zero field; nothing to add unless you write tests for the new routes.

## Step 6: Verify

```bash
gofmt -w domain/<domain> internal/http/chi
go build ./... && go vet ./... && go test ./...
go tool sqlc diff
go tool swag init -g ./internal/http/chi/api_handler.go -o ./docs/api --parseDependency --parseInternal --instanceName api
```

Fix every error before reporting. If Docker DB is running (`make db-up`), also run `make migrate-up`.

## Step 7: Report

List created/modified files, the new endpoints, and what the user should fill in next (real columns, validation rules, business logic in service.go).
