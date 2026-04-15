package chi

import (
	"go-template/domain/errorlog"
	"go-template/domain/user"
	"go-template/internal/validator"

	"net/http"

	"github.com/go-chi/chi/v5"
)

type Services struct {
	UserService     user.UseCase
	ErrorLogService errorlog.UseCase
}

type Routes struct {
	Public chi.Router
	App    chi.Router
	Admin  chi.Router
}

type Guards struct {
	Access  func(http.Handler) http.Handler
	Refresh func(http.Handler) http.Handler
	Admin   func(http.Handler) http.Handler
}

// Routes Guards Services 3개를 합친 스트럭트
type Handler struct {
	Routes Routes
	Guards Guards
	v      validator.ValidationService
}
