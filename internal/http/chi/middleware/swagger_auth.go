package middleware

import (
	"crypto/subtle"
	"errors"
	"net/http"

	response "go-template/internal/http"
)

// SwaggerAuthMiddleware 스웨거용 basic auth. 계정이 설정되지 않았으면 모두 거부한다.
type SwaggerAuthMiddleware struct {
	username string
	password string
	realm    string
}

func NewSwaggerAuthMiddleware(username, password, realm string) *SwaggerAuthMiddleware {
	return &SwaggerAuthMiddleware{username: username, password: password, realm: realm}
}

func (m *SwaggerAuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if m.username == "" || m.password == "" || !ok ||
			!secureCompare(username, m.username) || !secureCompare(password, m.password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+m.realm+`"`)
			response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("unauthorized"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
