package middleware

import (
	"errors"
	"net/http"

	response "go-template/internal/http"
)

const APIKeyHeader = "X-API-Key"

// APIKeyMiddleware X-API-Key 헤더가 key 와 같아야 통과한다. key 가 비어 있으면 검사하지 않는다.
// 브라우저 프론트에 넣는 키는 비밀이 아니다. 봇·무단 클라이언트를 거르는 용도로만 쓴다.
func APIKeyMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS preflight 에는 커스텀 헤더가 실리지 않는다
			if r.Method != http.MethodOptions && !secureCompare(r.Header.Get(APIKeyHeader), key) {
				response.WriteError(w, http.StatusUnauthorized, response.UnAuthorized, errors.New("invalid api key"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
