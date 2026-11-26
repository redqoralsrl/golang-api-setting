package chi

import (
	"errors"
	"net/http"
	"time"

	"go-template/domain/user"
	"go-template/internal/auth"
	response "go-template/internal/http"
	"go-template/internal/http/chi/middleware"
	"go-template/internal/logger"
	"go-template/internal/validator"
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
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"` // bcrypt 는 72바이트까지만 쓴다
}

// CreateUser godoc
// @Summary Create user
// @Description Create a new user account and issue an app access token cookie.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "Create user request"
// @Success 201 {object} response.Response{data=user.AuthResult}
// @Header 201 {string} Set-Cookie "go_template_access_token=...; Path=/api/v1/app; HttpOnly; SameSite=Lax"
// @Failure 400 {object} response.Response "invalid input"
// @Failure 409 {object} response.Response "user already exists"
// @Failure 500 {object} response.Response "internal server error"
// @Router /user/create [post]
func (h *userHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if !bindJSON(w, r, h.validator, &req) {
		return
	}

	result, err := h.useCase.Create(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUserAlreadyExists):
			response.WriteError(w, http.StatusConflict, response.Conflict, user.ErrUserAlreadyExists)
		default:
			response.WriteErrorWithRequest(w, r, h.logger, http.StatusInternalServerError, response.InternalError, err)
		}
		return
	}

	h.issueSession(w, result)
	response.WriteResponse(w, http.StatusCreated, response.Success, nil, result)
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

// issueSession 토큰을 쿠키로 내려준다. dev 가 아니면 응답 바디에서는 토큰을 지운다.
func (h *userHandler) issueSession(w http.ResponseWriter, result *user.AuthResult) {
	middleware.SetAppAuthCookie(w, result.AccessToken, time.Now().Add(auth.UserTokenTTL), h.isDev)
	if !h.isDev {
		result.AccessToken = ""
	}
}
