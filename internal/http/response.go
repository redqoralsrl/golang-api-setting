package http

import (
	"encoding/json"
	"net/http"

	"go-template/internal/logger"
	"go-template/internal/validator"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type Message string

const (
	Success            Message = "success"
	Fail               Message = "fail"
	TooManyReq         Message = "too many request"
	NotFound           Message = "not found"
	Conflict           Message = "conflict"
	UnAuthorized       Message = "unauthorized"
	InvalidInput       Message = "invalid input"
	Forbidden          Message = "forbidden"
	PayloadTooLarge    Message = "payload too large"
	ServiceUnavailable Message = "service unavailable"
	DatabaseError      Message = "database error"
	InternalError      Message = "internal server error"
)

// Response 모든 JSON 응답의 공통 형태
type Response struct {
	Code    int                       `json:"code"`
	Message Message                   `json:"message"`
	Error   *string                   `json:"error,omitempty"`
	Errors  []validator.ErrorResponse `json:"errors,omitempty"` // 입력 검증 실패 시 필드별 사유
	Data    any                       `json:"data"`
}

func write(w http.ResponseWriter, res Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(res.Code)
	_ = json.NewEncoder(w).Encode(res)
}

func WriteResponse(w http.ResponseWriter, code int, message Message, err *string, data any) {
	write(w, Response{Code: code, Message: message, Error: err, Data: data})
}

// WriteError 5xx 는 내부 에러 내용을 숨기고 고정 문구만 보낸다.
func WriteError(w http.ResponseWriter, code int, message Message, err error) {
	str := ""
	if code >= http.StatusInternalServerError {
		str = "Internal Server Error"
	} else if err != nil {
		str = err.Error()
	}
	WriteResponse(w, code, message, &str, nil)
}

func WriteValidationError(w http.ResponseWriter, errs []validator.ErrorResponse) {
	msg := "validation failed"
	write(w, Response{Code: http.StatusBadRequest, Message: InvalidInput, Error: &msg, Errors: errs})
}

// WriteErrorWithRequest 5xx 면 요청 정보와 원인 에러를 로그로 남긴 뒤 응답한다.
func WriteErrorWithRequest(w http.ResponseWriter, r *http.Request, l logger.Logger, code int, message Message, err error) {
	if code >= http.StatusInternalServerError && l != nil && r != nil {
		pattern := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			pattern = rc.RoutePattern()
		}
		l.Error("http error response",
			logger.NewField("status", code),
			logger.NewField("response_message", message),
			logger.NewError(err),
			logger.NewField("method", r.Method),
			logger.NewField("path", r.URL.Path),
			logger.NewField("route_pattern", pattern),
			logger.NewField("request_id", chiMiddleware.GetReqID(r.Context())),
		)
	}
	WriteError(w, code, message, err)
}
