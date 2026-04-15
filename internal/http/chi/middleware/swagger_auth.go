package middleware

import (
	"crypto/subtle"
	"errors"
	response "go-template/internal/http"
	"net/http"
)

// 스웨거용 기본 auth
type SwaggerAuthMiddleware struct {
	username string
	password string
	realm    string
}

func NewSwaggerAuthMiddleware(username, password, realm string) *SwaggerAuthMiddleware {
	return &SwaggerAuthMiddleware{
		username: username,
		password: password,
		realm:    realm,
	}
}

func (m *SwaggerAuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || !secureCompare(username, m.username) || !secureCompare(password, m.password) {
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
