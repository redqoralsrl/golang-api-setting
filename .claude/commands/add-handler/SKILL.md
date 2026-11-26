---
name: add-handler
description: Add a new HTTP endpoint to an existing domain in this Go API project - SQL query (sqlc), repository method, UseCase method and service, handler with Swagger comments, and route registration. Usage: /add-handler <domain> <HandlerName> <HTTP_METHOD> <route> [public|app|upload]. Example: /add-handler order ListOrders GET /order app
tools: Read, Write, Edit, Glob, Grep, Bash
---

# Add Handler Skill

Add one endpoint to an **existing** domain, through every layer, so it builds and works.

**Arguments**: `$ARGUMENTS` = `<domain> <HandlerName> <HTTP_METHOD> <route> [group]`
- **domain**: existing domain, lowercase (`order`)
- **HandlerName**: PascalCase (`ListOrders`, `UpdateOrder`, `DeleteOrder`)
- **HTTP_METHOD**: `GET` `POST` `PUT` `PATCH` `DELETE` → chi method `Get` `Post` ...
- **route**: path relative to the group, chi params in braces (`/order/{id}`)
- **group**: `public` → `handler.Routes.Public` (`/api/v1`), `app` (default) → `handler.Routes.App` (`/api/v1/app`, user JWT), `upload` → `handler.Routes.AppUpload` (20MB body). Never `Admin` until admin auth exists.

If fewer than 4 tokens, print the usage line with examples and stop:
```
/add-handler order GetOrder GET /order/{id} app
/add-handler order ListOrders GET /order app
/add-handler order UpdateOrder PUT /order/{id} app
/add-handler order DeleteOrder DELETE /order/{id} app
```

Read `CLAUDE.md` first and follow its rules.

## Step 1: Read the domain

Stop with `Domain "<domain>" not found. Run /add-domain <domain> first.` if `domain/<domain>/<domain>.go` or `internal/http/chi/<domain>.go` is missing. Otherwise read, before editing:
- `domain/<domain>/<domain>.go`, `error.go`, `service.go`
- `domain/<domain>/postgresql/query.sql`, `storage.go`
- `internal/http/chi/<domain>.go`
- the migration that creates the table (grep `migrations/` for `create table v1.<domain>s`, plus later `alter table` files) to know the real columns

## Step 2: Query (skip if the handler needs no new SQL)

Append to `domain/<domain>/postgresql/query.sql`, then `go tool sqlc generate` and read the generated signature in `internal/database/postgresql/gen/query.sql.go`.

Patterns:

```sql
-- name: List<Pascal>s :many
-- 커서 페이지네이션: id 내림차순, cursor 보다 작은 id 부터 limit 개
select * from v1.<domain>s
where (sqlc.narg(cursor)::bigint is null or id < sqlc.narg(cursor)::bigint)
order by id desc
limit sqlc.arg(page_size);

-- name: Update<Pascal> :one
update v1.<domain>s
set name = sqlc.arg(name)
where id = sqlc.arg(id)
returning *;

-- name: Delete<Pascal> :execrows
delete from v1.<domain>s where id = sqlc.arg(id);
```

`:execrows` returns the affected row count; treat 0 as not found. Rows owned by a user should filter by `user_id = sqlc.arg(user_id)` in the query, not only in Go.

## Step 3: Repository + domain interface

1. Add the method to `Reader` (reads) or `Writer` (writes) in `domain/<domain>/<domain>.go`. Put multi-field inputs in a struct (`UpdateInput`), and paginated results in a struct:
   ```go
   type ListResult struct {
       Items      []*<Pascal> `json:"items"`
       NextCursor string      `json:"next_cursor,omitempty"`
   }
   ```
2. Implement it in `storage.go`: call `s.db.Queries(ctx).<Query>(ctx, gen.<Query>Params{...})`, wrap errors with the existing `mapError`, convert rows with the existing `to<Pascal>`.
   For a cursor list: decrypt the incoming cursor with `s.db.DecryptCursor`, fetch `limit+1` rows, and if there is an extra row encrypt the last returned id with `s.db.EncryptCursor` as `NextCursor`. An invalid cursor is a client error: add `ErrInvalidCursor` to `error.go` and return it.

## Step 4: UseCase + service

Add the method to `UseCase` and implement it in `service.go`. Keep HTTP types out of the domain. Use `s.txManager.WithTransaction(ctx, func(ctx context.Context) error {...})` only when the operation writes more than once. New failure cases get a new `Err...` in `error.go` (its text is shown to clients in 4xx responses).

## Step 5: Handler

In `internal/http/chi/<domain>.go`:

1. Register the route inside `<Pascal>Handler(...)`: `handler.Routes.<Group>.<Method>("<route>", h.<HandlerName>)`.
2. Request struct with `json` + `validate` tags (skip for GET/DELETE). Query params for lists: read `r.URL.Query()`, clamp `limit` to 1..100 (default 20).
3. Handler method:
   - JSON body: `if !bindJSON(w, r, h.validator, &req) { return }`
   - Path id: `strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)`, reject `<= 0` with 400
   - Current user (app group): `id, ok := auth.FromContext(r.Context())`, parse `id.UserID`
   - Errors: reuse/extend the domain's `writeError` switch. Unknown errors go to `response.WriteErrorWithRequest(..., 500, ...)`
   - Success: `response.WriteResponse(w, <status>, response.Success, nil, data)`. 201 for create, 200 otherwise (DELETE may return 200 with `nil` data)
4. Swagger comment block above the method: `@Summary`, `@Tags <Pascal>s`, `@Accept json` (if body), `@Produce json`, `@Param` for body/path/query, `@Success <code> {object} response.Response{data=<type>}`, each `@Failure`, `@Router <path relative to /api/v1> [<method>]` — app group paths start with `/app/`.

## Step 6: Verify

```bash
gofmt -w domain/<domain> internal/http/chi
go build ./... && go vet ./... && go test ./...
go tool sqlc diff
go tool swag init -g ./internal/http/chi/api_handler.go -o ./docs/api --parseDependency --parseInternal --instanceName api
```

Fix all errors. When the change has branching logic (pagination, ownership checks), add a table test next to the existing ones (`internal/http/chi/router_test.go` shows the fake-UseCase pattern).

## Step 7: Report

Print the endpoint (`<METHOD> /api/v1/...`), the files changed, and anything left for the user (business rules, extra validation).
