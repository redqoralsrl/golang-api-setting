package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	AppAuthCookieName = "go_template_access_token"
	appAuthCookiePath = "/api/v1/app"
)

// 일반 앱 토큰 쿠키용 auth
func SetAppAuthCookie(w http.ResponseWriter, token string, expiresAt time.Time, isDev bool) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AppAuthCookieName,
		Value:    strings.TrimSpace(token),
		Path:     appAuthCookiePath,
		Expires:  expiresAt.UTC(),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !isDev,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearAppAuthCookie(w http.ResponseWriter, isDev bool) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	http.SetCookie(w, &http.Cookie{
		Name:     AppAuthCookieName,
		Value:    "",
		Path:     appAuthCookiePath,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !isDev,
		SameSite: http.SameSiteLaxMode,
	})
}

func ExtractAppAuthToken(r *http.Request, allowAuthorizationHeader bool) (string, error) {
	if r == nil {
		return "", errors.New("request is required")
	}

	if cookie, err := r.Cookie(AppAuthCookieName); err == nil {
		token := strings.TrimSpace(cookie.Value)
		if token != "" {
			return token, nil
		}
	}

	// local 개발용 쿠키 불가 시 Header auth 인증
	if !allowAuthorizationHeader {
		return "", errors.New("app auth token required")
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", errors.New("app auth token required")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("invalid authorization format")
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return "", errors.New("app auth token required")
	}

	return token, nil
}
