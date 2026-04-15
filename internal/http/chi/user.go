package chi

import (
	"encoding/json"
	"errors"
	"go-template/domain/user"
	response "go-template/internal/http"
	"go-template/internal/http/chi/middleware"
	"go-template/internal/logger"
	"go-template/internal/validator"
	"net/http"
	"time"
)

type userHandler struct {
	useCase   user.UseCase
	validator validator.ValidationService
	logger    logger.Logger
	isDev     bool
}

func UserHandler(s user.UseCase, handler Handler, l logger.Logger, isDev bool) {
	h := &userHandler{useCase: s, validator: handler.v, logger: l, isDev: isDev}

	handler.Routes.Public.Post("/user/create", h.CreateUser)
	handler.Routes.App.Post("/user/logout", h.Logout)
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// CreateUser godoc
// @Summary Create user
// @Description Create a new user account and issue an app access token cookie.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "Create user request"
// @Success 201 {object} response.Response{data=user.User}
// @Header 201 {string} Set-Cookie "go_template_access_token=...; Path=/api/v1/app; HttpOnly; SameSite=Lax"
// @Failure 400 {object} response.Response "invalid input"
// @Failure 409 {object} response.Response "user already exists"
// @Failure 500 {object} response.Response "internal server error"
// @Router /user/create [post]
func (h *userHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, err)
		return
	}

	if validationErrs := h.validator.Validate(req); validationErrs != nil {
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, errors.New("validation failed"))
		return
	}

	data, err := h.useCase.Create(ctx, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUserUniqueViolation):
			response.WriteError(w, http.StatusConflict, response.Conflict, err)
		default:
			response.WriteErrorWithRequest(w, r, nil, http.StatusInternalServerError, response.InternalError, err)
		}
		return
	}

	middleware.SetAppAuthCookie(w, data.AccessToken, time.Now().Add(middleware.UserExpired), h.isDev)

	if !h.isDev {
		data.AccessToken = ""
	}

	response.WriteResponse(w, http.StatusCreated, response.Success, nil, data)
}

// Logout godoc
// @Summary Logout user
// @Description Clear the app access token cookie.
// @Tags Users
// @Produce json
// @Success 200 {object} response.Response
// @Header 200 {string} Set-Cookie "go_template_access_token=; Path=/api/v1/app; Max-Age=0; HttpOnly; SameSite=Lax"
// @Router /app/user/logout [post]
func (h *userHandler) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearAppAuthCookie(w, h.isDev)
	response.WriteResponse(w, http.StatusOK, response.Success, nil, nil)
}
