package middleware

import (
	"errors"
	"net/http"
	"slices"

	"go-template/internal/auth"
	response "go-template/internal/http"
	"go-template/internal/jwt"
)

type AuthMiddleware struct {
	jwtAdapter               *jwt.JWTAdapter
	allowAuthorizationHeader bool
}

// NewAuthMiddleware allowAuthorizationHeader 가 true(dev)면 쿠키가 없을 때 Authorization: Bearer 도 받는다.
func NewAuthMiddleware(jwtAdapter *jwt.JWTAdapter, allowAuthorizationHeader bool) *AuthMiddleware {
	return &AuthMiddleware{jwtAdapter: jwtAdapter, allowAuthorizationHeader: allowAuthorizationHeader}
}

// Authenticate 앱 사용자 토큰을 확인하고 auth.Identity 를 context 에 넣는다. auth.FromContext 로 꺼낸다.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := ExtractAppAuthToken(r, m.allowAuthorizationHeader)
		if err != nil {
			response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("unauthorized"))
			return
		}

		claims, err := m.jwtAdapter.ValidateToken(token)
		if err != nil || claims.Role != string(auth.RoleUser) {
			response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("invalid token"))
			return
		}

		ctx := auth.WithIdentity(r.Context(), auth.Identity{UserID: claims.UserID, Role: auth.Role(claims.Role)})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole Authenticate 뒤에 붙인다. 역할이 roles 중 하나가 아니면 403.
func RequireRole(roles ...auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := auth.FromContext(r.Context())
			if !ok {
				response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("unauthorized"))
				return
			}
			if !slices.Contains(roles, id.Role) {
				response.WriteError(w, http.StatusForbidden, response.Forbidden, errors.New("insufficient permissions"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
