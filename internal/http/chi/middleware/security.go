package middleware

import (
	"net/http"
)

// SecurityHeadersMiddleware 보안 헤더 설정
func SecurityHeadersMiddleware(stage string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			w.Header().Set("X-Content-Type-Options", "nosniff")

			w.Header().Set("X-Frame-Options", "DENY")

			if stage == "prod" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Referrer 정책 설정
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			//w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimitMiddleware 요청 body 크기 제한
func BodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}
