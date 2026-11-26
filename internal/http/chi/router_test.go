package chi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-template/config"
	"go-template/domain/user"
	"go-template/internal/http/chi/middleware"
	"go-template/internal/jwt"
	"go-template/internal/logger/zerolog"
	"go-template/internal/validator"
)

type fakeUsers struct{ tokens *jwt.JWTAdapter }

func (f fakeUsers) Create(_ context.Context, email, _ string) (*user.AuthResult, error) {
	if email == "taken@example.com" {
		return nil, user.ErrUserAlreadyExists
	}
	return f.result()
}

func (f fakeUsers) Login(_ context.Context, _, password string) (*user.AuthResult, error) {
	if password != "correct-password" {
		return nil, user.ErrInvalidCredentials
	}
	return f.result()
}

func (f fakeUsers) Get(_ context.Context, id int64) (*user.User, error) {
	return &user.User{ID: id, Email: "me@example.com", PasswordHash: "hash"}, nil
}

func (f fakeUsers) result() (*user.AuthResult, error) {
	token, err := f.tokens.GenerateToken("7", "user", time.Minute)
	return &user.AuthResult{User: &user.User{ID: 7, Email: "me@example.com"}, AccessToken: token}, err
}

func newTestRouter(t *testing.T, ready ReadyCheck) (http.Handler, string) {
	t.Helper()
	j, err := jwt.NewJWTAdapter(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	v, _ := validator.NewValidationService()
	conf := &config.Config{Stage: config.StageDev, CORSAllowedOrigins: []string{"https://app.example.com"}}
	token, _ := j.GenerateToken("7", "user", time.Minute)
	return NewApiRouter(&Services{UserService: fakeUsers{tokens: j}}, conf, v, j, zerolog.NewLogger("test"), ready), token
}

func do(h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestUserRoutes(t *testing.T) {
	h, token := newTestRouter(t, func(context.Context) error { return nil })
	bearer := map[string]string{"Authorization": "Bearer " + token}

	tests := []struct {
		name, method, path, body string
		headers                  map[string]string
		want                     int
	}{
		{"create", http.MethodPost, "/api/v1/user/create", `{"email":"new@example.com","password":"12345678"}`, nil, http.StatusCreated},
		{"create duplicate", http.MethodPost, "/api/v1/user/create", `{"email":"taken@example.com","password":"12345678"}`, nil, http.StatusConflict},
		{"create invalid", http.MethodPost, "/api/v1/user/create", `{"email":"x","password":"1"}`, nil, http.StatusBadRequest},
		{"create bad json", http.MethodPost, "/api/v1/user/create", `{`, nil, http.StatusBadRequest},
		{"login ok", http.MethodPost, "/api/v1/user/login", `{"email":"me@example.com","password":"correct-password"}`, nil, http.StatusOK},
		{"login wrong password", http.MethodPost, "/api/v1/user/login", `{"email":"me@example.com","password":"nope"}`, nil, http.StatusUnauthorized},
		{"me without token", http.MethodGet, "/api/v1/app/user/me", "", nil, http.StatusUnauthorized},
		{"me with token", http.MethodGet, "/api/v1/app/user/me", "", bearer, http.StatusOK},
		{"cross-origin write blocked", http.MethodPost, "/api/v1/user/login", `{}`, map[string]string{"Origin": "https://evil.example.com", "Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"trusted origin write allowed", http.MethodPost, "/api/v1/user/login", `{"email":"me@example.com","password":"correct-password"}`, map[string]string{"Origin": "https://app.example.com", "Sec-Fetch-Site": "cross-site"}, http.StatusOK},
		{"body too large", http.MethodPost, "/api/v1/user/create", `{"email":"` + strings.Repeat("a", 2<<20) + `"}`, nil, http.StatusRequestEntityTooLarge},
		{"health", http.MethodGet, middleware.HealthPath, "", nil, http.StatusOK},
		{"not found", http.MethodGet, "/nope", "", nil, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := do(h, tt.method, tt.path, tt.body, tt.headers); w.Code != tt.want {
				t.Errorf("got %d, want %d: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestResponsesNeverLeakPasswordHash(t *testing.T) {
	h, token := newTestRouter(t, func(context.Context) error { return nil })
	w := do(h, http.MethodGet, "/api/v1/app/user/me", "", map[string]string{"Authorization": "Bearer " + token})
	if strings.Contains(w.Body.String(), "hash") {
		t.Fatalf("password hash leaked: %s", w.Body.String())
	}
}

func TestValidationErrorsListFields(t *testing.T) {
	h, _ := newTestRouter(t, func(context.Context) error { return nil })
	w := do(h, http.MethodPost, "/api/v1/user/create", `{"email":"x","password":"1"}`, nil)

	var body struct {
		Errors []validator.ErrorResponse `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	fields := map[string]bool{}
	for _, e := range body.Errors {
		fields[e.Field] = true
	}
	if !fields["email"] || !fields["password"] {
		t.Errorf("expected email and password errors, got %+v", body.Errors)
	}
}

func TestReady(t *testing.T) {
	h, _ := newTestRouter(t, func(context.Context) error { return errors.New("db down") })
	if w := do(h, http.MethodGet, middleware.ReadyPath, "", nil); w.Code != http.StatusServiceUnavailable {
		t.Errorf("got %d, want 503", w.Code)
	}
}
