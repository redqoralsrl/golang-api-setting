package middleware

import (
	"context"
	"errors"
	response "go-template/internal/http"
	"go-template/internal/jwt"
	"net/http"
	"time"
)

type AuthMiddleware struct {
	jwtAdapter               *jwt.JWTAdapter
	allowAuthorizationHeader bool
}

type Role string

const (
	UserRole Role = "user"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

const (
	UserExpired time.Duration = 1 * time.Hour
)

func NewAuthMiddleware(jwtAdapter *jwt.JWTAdapter, allowAuthorizationHeader bool) *AuthMiddleware {
	return &AuthMiddleware{
		jwtAdapter:               jwtAdapter,
		allowAuthorizationHeader: allowAuthorizationHeader,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := ExtractAppAuthToken(r, m.allowAuthorizationHeader)
		if err != nil {
			response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("unauthorized"))
			return
		}

		claims, err := m.jwtAdapter.ValidateToken(token)
		if err != nil {
			response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("invalid token"))
			return
		}

		if claims.Role != string(UserRole) {
			response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("invalid token type"))
			return
		}

		// Context에 사용자 정보 추가
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireRole(roles []Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := r.Context().Value(RoleKey).(string)

			hasRole := false
			for _, role := range roles {
				if Role(userRole) == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				response.WriteError(w, http.StatusForbidden, response.Forbidden, errors.New("insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
