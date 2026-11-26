package chi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go-template/config"
	_ "go-template/docs/api"
	response "go-template/internal/http"
	"go-template/internal/http/chi/middleware"
	"go-template/internal/jwt"
	"go-template/internal/logger"
	"go-template/internal/validator"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"golang.org/x/time/rate"
)

const (
	jsonBodyLimit   = 1 << 20  // 1MB
	uploadBodyLimit = 20 << 20 // 20MB
)

// ReadyCheck /ready 에서 호출한다. 의존 서비스(DB 등)에 연결할 수 없으면 에러를 돌려준다.
type ReadyCheck func(ctx context.Context) error

// NewApiRouter godoc
// @title Go Template API
// @version 1.0
// @description Go Template HTTP API documentation.
// @BasePath /api/v1
func NewApiRouter(services *Services, conf *config.Config, v validator.ValidationService, j *jwt.JWTAdapter, l logger.Logger, ready ReadyCheck) *chi.Mux {
	r := chi.NewRouter()
	origins := middleware.TrustedOrigins(conf.CORSAllowedOrigins, conf.IsDev())

	// 순서: 요청 식별 → 로깅(최종 status 기록) → panic 복구 → 차단·보안
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.ClientIPMiddleware(conf.TrustedProxyHops))
	r.Use(middleware.LoggerMiddleware(l, services.ErrorLogService))
	r.Use(middleware.RecoveryMiddleware(l))
	r.Use(middleware.RateLimitMiddleware(rate.Limit(30), 60))
	r.Use(middleware.CORSMiddleware(origins))
	r.Use(middleware.SecurityHeadersMiddleware(conf.IsDev()))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusNotFound, response.NotFound, nil)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusMethodNotAllowed, response.Fail, nil)
	})

	// system: 헬스체크, 스웨거
	r.Get(middleware.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		response.WriteResponse(w, http.StatusOK, response.Success, nil, map[string]string{"status": "ok"})
	})
	r.Get(middleware.ReadyPath, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := ready(ctx); err != nil {
			l.Warn("readiness check failed", logger.NewError(err))
			response.WriteError(w, http.StatusServiceUnavailable, response.ServiceUnavailable, errors.New("not ready"))
			return
		}
		response.WriteResponse(w, http.StatusOK, response.Success, nil, map[string]string{"status": "ready"})
	})
	if conf.IsDev() {
		swaggerAuth := middleware.NewSwaggerAuthMiddleware(conf.SwaggerID, conf.SwaggerPassword, "Go Template Swagger")
		r.With(swaggerAuth.Authenticate).Get("/swagger/*", httpSwagger.Handler(httpSwagger.InstanceName("api")))
	}

	// route groups
	csrf := middleware.CSRFMiddleware(origins)
	apiKey := middleware.APIKeyMiddleware(conf.APIKey)
	authMiddleware := middleware.NewAuthMiddleware(j, conf.IsDev())

	public := chi.NewRouter()
	public.Use(csrf, apiKey)

	app := chi.NewRouter()
	app.Use(csrf, apiKey, authMiddleware.Authenticate)

	admin := chi.NewRouter()
	admin.Use(csrf)

	r.Mount("/api/v1/app", app)
	r.Mount("/api/v1", public)
	r.Mount("/admin/v1", admin)

	handler := Handler{
		Routes: Routes{
			Public:    public.With(middleware.BodyLimitMiddleware(jsonBodyLimit)),
			App:       app.With(middleware.BodyLimitMiddleware(jsonBodyLimit)),
			AppUpload: app.With(middleware.BodyLimitMiddleware(uploadBodyLimit)),
			Admin:     admin.With(middleware.BodyLimitMiddleware(jsonBodyLimit)),
		},
		v: v,
	}

	// 도메인 핸들러 등록
	UserHandler(services.UserService, handler, l, conf.IsDev())

	return r
}
