package chi

import (
	"go-template/config"
	_ "go-template/docs/api"
	response "go-template/internal/http"
	"go-template/internal/http/chi/middleware"
	"go-template/internal/jwt"
	"go-template/internal/logger"
	"go-template/internal/validator"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"golang.org/x/time/rate"
)

// NewApiRouter godoc
// @title Go Template API
// @version 1.0
// @description Go Template HTTP API documentation.
// @BasePath /api/v1
func NewApiRouter(services *Services, conf *config.Config, v validator.ValidationService, j *jwt.JWTAdapter, logger logger.Logger) *chi.Mux {
	r := chi.NewRouter()

	// 기본 미들웨어 설정
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)

	// 보안 미들웨어 설정
	r.Use(middleware.CORSMiddleware(conf.IsDev()))
	r.Use(middleware.SecurityHeadersMiddleware(conf.Stage))
	r.Use(middleware.BodyLimitMiddleware(10 << 20)) // 10MB 제한
	r.Use(middleware.RateLimitMiddleware(rate.Limit(30), 60))

	// 모니터링 미들웨어
	r.Use(middleware.RecoveryMiddleware(logger))
	r.Use(middleware.LoggerMiddleware(logger, services.ErrorLogService))

	// auth 설정
	authMiddleware := middleware.NewAuthMiddleware(j, conf.IsDev())
	swaggerAuthMiddleware := middleware.NewSwaggerAuthMiddleware(
		conf.SwaggerID,
		conf.SwaggerPassword,
		"Go Template Swagger",
	)

	// router group
	system := chi.NewRouter()
	public := chi.NewRouter()
	app := chi.NewRouter()
	admin := chi.NewRouter()

	r.Mount("/", system)
	r.Mount("/api/v1", public)
	r.Mount("/api/v1/app", app)
	r.Mount("/admin", admin)

	// api auth 연결
	app.Use(authMiddleware.Authenticate)

	// routes
	routes := Routes{
		Public: public,
		App:    app,
		Admin:  admin,
	}

	// guards
	guards := Guards{}

	// handler
	handler := Handler{
		Routes: routes,
		Guards: guards,
		v:      v,
	}

	// Swagger
	if conf.IsDev() {
		system.With(swaggerAuthMiddleware.Authenticate).Get("/swagger/*", httpSwagger.Handler(httpSwagger.InstanceName("api")))
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.WriteResponse(w, http.StatusOK, response.Success, nil, map[string]string{
			"status": "ok",
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusNotFound, response.NotFound, nil)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusMethodNotAllowed, response.Fail, nil)
	})

	// method connect
	UserHandler(services.UserService, handler, logger, conf.IsDev())

	return r
}
