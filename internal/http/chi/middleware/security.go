package middleware

import (
	"errors"
	"net/http"
	"strings"

	response "go-template/internal/http"
)

// SecurityHeadersMiddleware JSON API 용 보안 헤더. Swagger UI 는 스크립트가 필요해 CSP 에서 뺀다.
func SecurityHeadersMiddleware(isDev bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if !strings.HasPrefix(r.URL.Path, "/swagger") {
				h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			}
			if !isDev {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimitMiddleware 요청 body 크기 제한. 핸들러에서 한도를 넘게 읽으면 *http.MaxBytesError 가 난다.
func BodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				response.WriteError(w, http.StatusRequestEntityTooLarge, response.PayloadTooLarge, errors.New("request body too large"))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}
