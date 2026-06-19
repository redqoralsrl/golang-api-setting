---
name: add-handler
description: Add a new HTTP endpoint to an existing domain in this Go API project. Adds the UseCase method signature, service stub, request/response structs, handler method with Swagger comments, and route registration. Usage: /add-handler <domain> <HandlerName> <HTTP_METHOD> <route> [public|app|admin]. Example: /add-handler user GetUser GET /user/{id} app
tools: Read, Write, Edit, Glob, Grep, Bash
---

# Add Handler Skill

Add a new HTTP endpoint to an **existing** domain in this Go API project.

**Arguments**: `$ARGUMENTS`

Parse `$ARGUMENTS` as 5 space-separated tokens:
1. **domain** — existing domain name, lowercase (e.g. `user`)
2. **HandlerName** — PascalCase name for the handler function (e.g. `GetUser`, `UpdateUser`, `DeleteUser`)
3. **HTTP_METHOD** — one of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
4. **route** — URL path, may include `{id}` params (e.g. `/user/{id}`, `/user/search`)
5. **group** — route group: `public`, `app`, or `admin` (default: `public`)

Derive from the parsed values:
- **pascal** = HandlerName as-is (already PascalCase)
- **method** = HTTP_METHOD lowercased for chi registration (e.g. `Get`, `Post`, `Put`, `Patch`, `Delete`)
- **routeGroup** = `handler.Routes.Public` / `handler.Routes.App` / `handler.Routes.Admin` based on group

---

## Step 0: Validate

If `$ARGUMENTS` has fewer than 4 tokens, print:
```
Usage: /add-handler <domain> <HandlerName> <HTTP_METHOD> <route> [public|app|admin]

Examples:
  /add-handler user GetUser GET /user/{id} app
  /add-handler user UpdateUser PUT /user/{id} app
  /add-handler user ListUsers GET /user/list public
  /add-handler user DeleteUser DELETE /user/{id} app
```
Then stop.

---

## Step 1: Verify the domain exists

Check that these files exist:
- `domain/<domain>/<domain>.go`
- `internal/http/chi/<domain>.go`

If either is missing, tell the user:
```
Domain "<domain>" not found. Run /add-domain <domain> first.
```
Then stop.

---

## Step 2: Read existing files

Read all three files before making any edits:
- `domain/<domain>/<domain>.go`
- `domain/<domain>/service.go`
- `internal/http/chi/<domain>.go`

---

## Step 3: Add UseCase method to `domain/<domain>/<domain>.go`

Find the `UseCase interface` block and add the new method signature.

For **GET** handlers (typically return a single entity or list):
```go
<HandlerName>(ctx context.Context, id int) (*<Pascal_entity>, error)
```

For **POST** handlers:
```go
<HandlerName>(ctx context.Context) (*<Pascal_entity>, error)
```

For **PUT/PATCH** handlers:
```go
<HandlerName>(ctx context.Context, id int) (*<Pascal_entity>, error)
```

For **DELETE** handlers:
```go
<HandlerName>(ctx context.Context, id int) error
```

Use the existing model type in the file (the struct defined in the same package). If the domain entity has fields like `ID int`, the return type is `*<DomainPascal>` (e.g. `*User`, `*Product`).

**Important**: infer the right signature by reading what methods already exist in the UseCase interface and what the domain struct looks like. Match the style of existing methods.

---

## Step 4: Add service stub to `domain/<domain>/service.go`

Add a new method on `*Service` that satisfies the UseCase interface method just added.

For GET/PUT/PATCH (returns entity):
```go
func (s *Service) <HandlerName>(ctx context.Context, id int) (*<DomainPascal>, error) {
	// TODO: implement
	return nil, nil
}
```

For POST (returns entity, no id param):
```go
func (s *Service) <HandlerName>(ctx context.Context) (*<DomainPascal>, error) {
	// TODO: implement
	return nil, nil
}
```

For DELETE (returns error):
```go
func (s *Service) <HandlerName>(ctx context.Context, id int) error {
	// TODO: implement
	return nil
}
```

Match the exact signature added to the UseCase interface in Step 3.

---

## Step 5: Add handler to `internal/http/chi/<domain>.go`

### 5a: Add request struct (skip for GET/DELETE)

For POST/PUT/PATCH, add a request struct before the new handler function:

```go
type <HandlerName>Request struct {
	// TODO: add request fields
}
```

### 5b: Add the handler method

Append to the file a new method on the handler struct. Use the existing handler struct name in the file (e.g. `userHandler`, `productHandler`).

**GET with `{id}` in route**:
```go
// <HandlerName> godoc
// @Summary <HandlerName>
// @Description <HandlerName> for <domain>.
// @Tags <DomainPascal>s
// @Produce json
// @Param id path int true "<DomainPascal> ID"
// @Success 200 {object} response.Response{data=<domain>.<DomainPascal>}
// @Failure 404 {object} response.Response "<domain> not found"
// @Failure 500 {object} response.Response "internal server error"
// @Router <route> [get]
func (h *<domain>Handler) <HandlerName>(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, err)
		return
	}

	data, err := h.useCase.<HandlerName>(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, <domain>.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, response.NotFound, err)
		default:
			response.WriteErrorWithRequest(w, r, nil, http.StatusInternalServerError, response.InternalError, err)
		}
		return
	}

	response.WriteResponse(w, http.StatusOK, response.Success, nil, data)
}
```

**GET without `{id}`** (list):
```go
// <HandlerName> godoc
// @Summary <HandlerName>
// @Description <HandlerName> for <domain>.
// @Tags <DomainPascal>s
// @Produce json
// @Success 200 {object} response.Response{data=[]<domain>.<DomainPascal>}
// @Failure 500 {object} response.Response "internal server error"
// @Router <route> [get]
func (h *<domain>Handler) <HandlerName>(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := h.useCase.<HandlerName>(ctx)
	if err != nil {
		response.WriteErrorWithRequest(w, r, nil, http.StatusInternalServerError, response.InternalError, err)
		return
	}

	response.WriteResponse(w, http.StatusOK, response.Success, nil, data)
}
```

**POST**:
```go
// <HandlerName> godoc
// @Summary <HandlerName>
// @Description <HandlerName> for <domain>.
// @Tags <DomainPascal>s
// @Accept json
// @Produce json
// @Param request body <HandlerName>Request true "<HandlerName> request"
// @Success 201 {object} response.Response{data=<domain>.<DomainPascal>}
// @Failure 400 {object} response.Response "invalid input"
// @Failure 500 {object} response.Response "internal server error"
// @Router <route> [post]
func (h *<domain>Handler) <HandlerName>(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req <HandlerName>Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, err)
		return
	}

	if validationErrs := h.validator.Validate(req); validationErrs != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, errors.New("validation failed"))
		return
	}

	data, err := h.useCase.<HandlerName>(ctx)
	if err != nil {
		response.WriteErrorWithRequest(w, r, nil, http.StatusInternalServerError, response.InternalError, err)
		return
	}

	response.WriteResponse(w, http.StatusCreated, response.Success, nil, data)
}
```

**PUT/PATCH**:
```go
// <HandlerName> godoc
// @Summary <HandlerName>
// @Description <HandlerName> for <domain>.
// @Tags <DomainPascal>s
// @Accept json
// @Produce json
// @Param id path int true "<DomainPascal> ID"
// @Param request body <HandlerName>Request true "<HandlerName> request"
// @Success 200 {object} response.Response{data=<domain>.<DomainPascal>}
// @Failure 400 {object} response.Response "invalid input"
// @Failure 404 {object} response.Response "<domain> not found"
// @Failure 500 {object} response.Response "internal server error"
// @Router <route> [put]
func (h *<domain>Handler) <HandlerName>(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, err)
		return
	}

	var req <HandlerName>Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, err)
		return
	}

	if validationErrs := h.validator.Validate(req); validationErrs != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, errors.New("validation failed"))
		return
	}

	data, err := h.useCase.<HandlerName>(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, <domain>.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, response.NotFound, err)
		default:
			response.WriteErrorWithRequest(w, r, nil, http.StatusInternalServerError, response.InternalError, err)
		}
		return
	}

	response.WriteResponse(w, http.StatusOK, response.Success, nil, data)
}
```

**DELETE**:
```go
// <HandlerName> godoc
// @Summary <HandlerName>
// @Description <HandlerName> for <domain>.
// @Tags <DomainPascal>s
// @Produce json
// @Param id path int true "<DomainPascal> ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response "<domain> not found"
// @Failure 500 {object} response.Response "internal server error"
// @Router <route> [delete]
func (h *<domain>Handler) <HandlerName>(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, err)
		return
	}

	if err := h.useCase.<HandlerName>(ctx, id); err != nil {
		switch {
		case errors.Is(err, <domain>.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, response.NotFound, err)
		default:
			response.WriteErrorWithRequest(w, r, nil, http.StatusInternalServerError, response.InternalError, err)
		}
		return
	}

	response.WriteResponse(w, http.StatusOK, response.Success, nil, nil)
}
```

### 5c: Add missing imports if needed

If the handler file doesn't already import them, add:
- `"strconv"` — when route contains `{id}` (GET/PUT/PATCH/DELETE with id)
- `"github.com/go-chi/chi/v5"` — when using `chi.URLParam`

Check existing imports before adding to avoid duplicates.

---

## Step 6: Register the route

Find the `<DomainPascal>Handler` function in `internal/http/chi/<domain>.go`. Add the new route inside it, after the last existing route registration:

```go
handler.Routes.<Group>.<Method>("<route>", h.<HandlerName>)
```

Where:
- `<Group>` = `Public`, `App`, or `Admin` (capitalized)
- `<Method>` = `Get`, `Post`, `Put`, `Patch`, or `Delete` (capitalized)
- `<route>` = the route path from the argument

---

## Step 7: Verify

```bash
cd /Users/minki/Desktop/golang-api-setting && go build ./... 2>&1
```

If the build fails, read the errors and fix them before reporting.

---

## Step 8: Report

```
✓ Modified domain/<domain>/<domain>.go  (added <HandlerName> to UseCase interface)
✓ Modified domain/<domain>/service.go   (added <HandlerName> stub)
✓ Modified internal/http/chi/<domain>.go (added handler + route registration)

Endpoint: <HTTP_METHOD> /api/v1[/app|/admin]<route>

Next steps:
1. domain/<domain>/service.go — implement <HandlerName> logic
2. domain/<domain>/<domain>.go — add Repository method if DB access needed
3. domain/<domain>/postgresql/storage.go — implement the repository method
4. domain/<domain>/postgresql/query.sql — write SQL, then: make sqlc
5. make create-swagger  (after Swagger comments are final)
```
