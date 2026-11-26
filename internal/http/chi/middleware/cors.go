package middleware

import (
	"errors"
	"net/http"

	response "go-template/internal/http"

	"github.com/go-chi/cors"
)

// devOrigins CORS_ALLOWED_ORIGINS 가 비어 있을 때 dev 에서 허용하는 로컬 프론트 주소
var devOrigins = []string{
	"http://localhost:3000",
	"http://localhost:5173",
	"http://127.0.0.1:3000",
	"http://127.0.0.1:5173",
}

// TrustedOrigins 설정값이 있으면 그대로, dev 에서 비어 있으면 로컬 주소를 쓴다.
func TrustedOrigins(configured []string, isDev bool) []string {
	if len(configured) == 0 && isDev {
		return devOrigins
	}
	return configured
}

func CORSMiddleware(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Requested-With", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}

// CSRFMiddleware 신뢰 출처 밖에서 온 브라우저 쓰기 요청(POST 등)을 막는다.
// 같은 상위 도메인(*.example.com)끼리는 SameSite 쿠키만으로 막히지 않아 필요하다.
// Origin/Sec-Fetch-Site 헤더가 없는 서버 간 호출은 통과한다.
func CSRFMiddleware(origins []string) func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, o := range origins {
		if err := protection.AddTrustedOrigin(o); err != nil {
			panic(err) // 설정 오류는 시작 시점에 드러낸다
		}
	}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusForbidden, response.Forbidden, errors.New("cross-origin request denied"))
	}))
	return protection.Handler
}
