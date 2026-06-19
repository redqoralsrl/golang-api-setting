---
name: add-domain
description: Add a new domain to this Go API project. Creates all boilerplate files (domain interfaces, service, errors, metrics, repository, SQL, HTTP handler) and wires everything into main.go and api_handler.go following the existing patterns. Usage: /add-domain <domain-name>
tools: Read, Write, Edit, Glob, Grep, Bash
---

# Add Domain Skill

Create a complete new domain in this Go API project.

**Argument**: `$ARGUMENTS` (the domain name, e.g. `product`, `order`, `payment`)

Before writing any file, derive these two values from `$ARGUMENTS` and use them literally — never write template placeholders like `$DOMAIN_PASCAL` into files:

- **domain** = `$ARGUMENTS` as-is in lowercase (e.g. `product`)
- **Pascal** = PascalCase of domain (e.g. `Product`)

---

## Step 0: Validate

If `$ARGUMENTS` is blank, print:
```
Usage: /add-domain <domain-name>
Example: /add-domain product
```
Then stop.

---

## Step 1: Check the domain doesn't already exist

```bash
ls domain/$ARGUMENTS 2>/dev/null && echo "EXISTS" || echo "OK"
```

If it prints `EXISTS`, stop and tell the user.

---

## Step 2: Create domain files

All paths below use the **real** domain name and PascalCase — write actual Go identifiers, not placeholders.

### `domain/<domain>/<domain>.go`

```go
package <domain>

import (
	"context"
	"time"
)

type <Pascal> struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Reader interface {
}

type Writer interface {
	Create(ctx context.Context) (*<Pascal>, error)
}

type Repository interface {
	Reader
	Writer
}

type UseCase interface {
	Create(ctx context.Context) (*<Pascal>, error)
}
```

### `domain/<domain>/error.go`

```go
package <domain>

import (
	"errors"
	"fmt"
)

const (
	ErrDBSyntaxError               = "syntax_error"
	ErrDBUniqueViolation           = "unique_violation"
	ErrDBNotNullViolation          = "not_null_violation"
	ErrDBStringDataRightTruncation = "string_data_right_truncation"
	ErrDBInvalidTextRepresentation = "invalid_text_representation"
)

var (
	ErrSyntaxError               = errors.New("input type error")
	ErrUniqueViolation           = errors.New("<domain> already exists")
	ErrNotNullViolation          = errors.New("null value error")
	ErrStringDataRightTruncation = errors.New("data truncation error")
	ErrInvalidTextRepresentation = errors.New("data value error")
	ErrNotFound                  = errors.New("<domain> not found")
)

func NewDatabaseError(operation string, cause error) error {
	return fmt.Errorf("database operation %q failed: %w", operation, cause)
}
```

### `domain/<domain>/metrics.go`

```go
package <domain>

import "go-template/internal/metrics"

var (
	<Pascal>CreatedTotal = metrics.NewSimpleCounter(
		"api_<domain>_created_total",
		"Total number of successfully created <domain>.",
	)
)
```

### `domain/<domain>/service.go`

```go
package <domain>

import (
	"context"
	"database/sql"
	"go-template/internal/database/postgresql"
	"go-template/internal/logger"
)

type Service struct {
	repo               Repository
	transactionManager postgresql.DBTransactionManager
	logger             logger.Logger
}

var _ UseCase = (*Service)(nil)

func NewService(r Repository, tm postgresql.DBTransactionManager, logger logger.Logger) *Service {
	return &Service{
		repo:               r,
		transactionManager: tm,
		logger:             logger,
	}
}

func (s *Service) Create(ctx context.Context) (*<Pascal>, error) {
	var result *<Pascal>
	var err error

	if err = s.transactionManager.WithTransaction(ctx, sql.LevelReadCommitted, false, func(txCtx context.Context) error {
		result, err = s.repo.Create(txCtx)
		if err != nil {
			s.logger.Error("failed to create <domain>", logger.Field{Key: "error", Value: err.Error()})
			return NewDatabaseError("create <domain>", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	<Pascal>CreatedTotal.Inc()

	return result, nil
}
```

### `domain/<domain>/postgresql/storage.go`

```go
package postgresql

import (
	"context"
	"errors"
	"go-template/domain/<domain>"
	"go-template/internal/database/postgresql"

	"github.com/lib/pq"
)

type <Pascal>Storage struct {
	db *postgresql.Database
}

var _ <domain>.Repository = (*<Pascal>Storage)(nil)

func New<Pascal>(db *postgresql.Database) *<Pascal>Storage {
	return &<Pascal>Storage{
		db: db,
	}
}

func (s *<Pascal>Storage) Create(ctx context.Context) (*<domain>.<Pascal>, error) {
	queryRower := s.db.GetQueryRowerFromContext(ctx)
	_ = queryRower

	var pqErr *pq.Error
	_ = errors.As(nil, &pqErr)

	// TODO: implement using sqlc-generated query
	// row, err := queryRower.Queries.Create<Pascal>(ctx, gen.Create<Pascal>Params{...})
	// if err != nil {
	//   if errors.As(err, &pqErr) {
	//     switch pqErr.Code.Name() {
	//     case <domain>.ErrDBUniqueViolation:
	//       return nil, <domain>.ErrUniqueViolation
	//     }
	//   }
	//   return nil, err
	// }
	// return &<domain>.<Pascal>{ID: int(row.ID), ...}, nil

	return &<domain>.<Pascal>{}, nil
}
```

### `domain/<domain>/postgresql/query.sql`

```sql
-- name: Create<Pascal> :one
-- TODO: replace with actual table and columns
-- insert into v1.<domain>s (
--     ...
-- ) values (
--     ...
-- )
-- returning id, created_at, updated_at;
```

---

## Step 3: Create HTTP handler

### `internal/http/chi/<domain>.go`

```go
package chi

import (
	"encoding/json"
	"errors"
	"go-template/domain/<domain>"
	response "go-template/internal/http"
	"go-template/internal/logger"
	"go-template/internal/validator"
	"net/http"
)

type <domain>Handler struct {
	useCase   <domain>.UseCase
	validator validator.ValidationService
	logger    logger.Logger
	isDev     bool
}

func <Pascal>Handler(s <domain>.UseCase, handler Handler, l logger.Logger, isDev bool) {
	h := &<domain>Handler{useCase: s, validator: handler.v, logger: l, isDev: isDev}

	handler.Routes.Public.Post("/<domain>/create", h.Create<Pascal>)
}

type Create<Pascal>Request struct {
	// TODO: add request fields
}

// Create<Pascal> godoc
// @Summary Create <domain>
// @Description Create a new <domain>.
// @Tags <Pascal>s
// @Accept json
// @Produce json
// @Param request body Create<Pascal>Request true "Create <domain> request"
// @Success 201 {object} response.Response{data=<domain>.<Pascal>}
// @Failure 400 {object} response.Response "invalid input"
// @Failure 409 {object} response.Response "<domain> already exists"
// @Failure 500 {object} response.Response "internal server error"
// @Router /<domain>/create [post]
func (h *<domain>Handler) Create<Pascal>(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req Create<Pascal>Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, err)
		return
	}

	if validationErrs := h.validator.Validate(req); validationErrs != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, errors.New("validation failed"))
		return
	}

	data, err := h.useCase.Create(ctx)
	if err != nil {
		switch {
		case errors.Is(err, <domain>.ErrUniqueViolation):
			response.WriteError(w, http.StatusConflict, response.Conflict, err)
		default:
			response.WriteErrorWithRequest(w, r, nil, http.StatusInternalServerError, response.InternalError, err)
		}
		return
	}

	response.WriteResponse(w, http.StatusCreated, response.Success, nil, data)
}
```

---

## Step 4: Wire `internal/http/chi/handler.go`

Read the file. Add `go-template/domain/<domain>` to imports and add `<Pascal>Service <domain>.UseCase` as a new field in the `Services` struct, after the last existing field.

---

## Step 5: Wire `internal/http/chi/api_handler.go`

Read the file. Find the `// method connect` block and append after the last `*Handler(...)` call:

```go
<Pascal>Handler(services.<Pascal>Service, handler, logger, conf.IsDev())
```

---

## Step 6: Wire `cmd/api/main.go`

Read the file and make these four edits:

1. **Imports** — add after existing domain imports:
   ```go
   "go-template/domain/<domain>"
   <domain>Postgresql "go-template/domain/<domain>/postgresql"
   ```

2. **Storage** — add after the last `*Storage` line:
   ```go
   <domain>Storage := <domain>Postgresql.New<Pascal>(db)
   ```

3. **Service** — add after the last `*Service` line:
   ```go
   <domain>Service := <domain>.NewService(<domain>Storage, transactionManager, l)
   ```

4. **Services struct** — add inside `stdHandler.Services{...}`:
   ```go
   <Pascal>Service: <domain>Service,
   ```

---

## Step 7: Verify

```bash
cd /Users/minki/Desktop/golang-api-setting && go build ./... 2>&1
```

If the build fails, read the errors and fix them before proceeding.

---

## Step 8: Report

Print:

```
✓ Created domain/<domain>/<domain>.go
✓ Created domain/<domain>/error.go
✓ Created domain/<domain>/metrics.go
✓ Created domain/<domain>/service.go
✓ Created domain/<domain>/postgresql/storage.go
✓ Created domain/<domain>/postgresql/query.sql
✓ Created internal/http/chi/<domain>.go
✓ Modified internal/http/chi/handler.go
✓ Modified internal/http/chi/api_handler.go
✓ Modified cmd/api/main.go

Next steps:
1. domain/<domain>/<domain>.go — define model fields and method signatures
2. domain/<domain>/postgresql/query.sql — write SQL, then: make sqlc
3. domain/<domain>/postgresql/storage.go — implement using sqlc-generated queries
4. internal/http/chi/<domain>.go — add request fields and handler logic
5. make sqlc  (after query.sql is ready)
6. make create-swagger  (after Swagger comments are complete)
```
