package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveClientIP(t *testing.T) {
	tests := []struct {
		name string
		hops int
		xff  []string
		want string
	}{
		{"no proxy ignores headers", 0, []string{"1.1.1.1"}, "10.0.0.1"},
		{"one proxy takes last entry", 1, []string{"6.6.6.6, 1.1.1.1"}, "1.1.1.1"},
		{"spoofed first entry is ignored", 1, []string{"6.6.6.6", "1.1.1.1"}, "1.1.1.1"},
		{"two proxies take second from right", 2, []string{"6.6.6.6, 1.1.1.1, 2.2.2.2"}, "1.1.1.1"},
		{"too few entries falls back", 2, []string{"1.1.1.1"}, "10.0.0.1"},
		{"invalid entry falls back", 1, []string{"not-an-ip"}, "10.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = "10.0.0.1:1234"
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := resolveClientIP(r, tt.hops); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMaskSensitiveData(t *testing.T) {
	got := maskSensitiveData(map[string]any{
		"id":       1,
		"password": "secret",
		"profile":  map[string]any{"email": "a@b.c", "status": "active"},
		"ids":      []any{1, 2},
	})

	if got["id"] != 1 {
		t.Errorf("id should be kept, got %v", got["id"])
	}
	if got["password"] != "******" {
		t.Errorf("password should be masked, got %v", got["password"])
	}
	profile := got["profile"].(map[string]any)
	if profile["email"] != "******" || profile["status"] != "active" {
		t.Errorf("nested masking wrong: %v", profile)
	}
	if ids := got["ids"].([]any); ids[0] != 1 {
		t.Errorf("allowed array should be kept, got %v", ids)
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	tests := []struct {
		name, key, header, method string
		want                      int
	}{
		{"disabled when key empty", "", "", http.MethodGet, http.StatusOK},
		{"missing header", "k", "", http.MethodGet, http.StatusUnauthorized},
		{"wrong header", "k", "x", http.MethodGet, http.StatusUnauthorized},
		{"correct header", "k", "k", http.MethodGet, http.StatusOK},
		{"preflight passes", "k", "", http.MethodOptions, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/", nil)
			if tt.header != "" {
				r.Header.Set(APIKeyHeader, tt.header)
			}
			w := httptest.NewRecorder()
			APIKeyMiddleware(tt.key)(ok).ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Errorf("got %d, want %d", w.Code, tt.want)
			}
		})
	}
}
