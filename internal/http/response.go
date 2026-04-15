package http

import (
	"encoding/json"
	"go-template/internal/logger"
	"net/http"

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
	ServiceUnavailable Message = "service unavailable"
	DatabaseError      Message = "Database Error"
	InternalError      Message = "Internal Server Error"
)

type Response struct {
	Code    int         `json:"code"`
	Message Message     `json:"message"`
	Error   *string     `json:"error,omitempty"`
	Data    interface{} `json:"data"`
}

func WriteResponse(w http.ResponseWriter, code int, message Message, err *string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if encodeErr := json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Error:   err,
		Data:    data,
	}); encodeErr != nil {
		// JSON encoding 실패 시 최소한의 응답
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func WriteError(w http.ResponseWriter, code int, message Message, err error) {
	str := ""
	if code >= 500 {
		str = "Internal Server Error"
	} else if err != nil {
		str = err.Error()
	}
	WriteResponse(w, code, message, &str, nil)
}

func WriteErrorWithRequest(w http.ResponseWriter, r *http.Request, l logger.Logger, code int, message Message, err error) {
	logErrorWithRequest(r, l, code, message, err)
	WriteError(w, code, message, err)
}

// func LogErrorWithRequest(r *http.Request, l logger.Logger, code int, message Message, err error) {
// 	logErrorWithRequest(r, l, code, message, err)
// }

func logErrorWithRequest(r *http.Request, l logger.Logger, code int, message Message, err error) {
	if code >= 500 && l != nil {
		fields := []logger.Field{
			logger.NewField("status", code),
			logger.NewField("response_message", message),
		}
		if err != nil {
			fields = append(fields, logger.NewError(err))
		}
		if r != nil {
			routePattern := r.URL.Path
			if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
				if pattern := routeContext.RoutePattern(); pattern != "" {
					routePattern = pattern
				}
			}

			fields = append(fields,
				logger.NewField("method", r.Method),
				logger.NewField("path", r.URL.Path),
				logger.NewField("route_pattern", routePattern),
				logger.NewField("remote_addr", r.RemoteAddr),
				logger.NewField("user_agent", r.UserAgent()),
			)
			if query := r.URL.RawQuery; query != "" {
				fields = append(fields, logger.NewField("query", query))
			}
			if requestID := chiMiddleware.GetReqID(r.Context()); requestID != "" {
				fields = append(fields, logger.NewField("request_id", requestID))
			}
		}

		l.Error("http error response", fields...)
	}
}

func IsValidMessage(msg string) bool {
	switch Message(msg) {
	case Success, Fail, NotFound, Conflict, TooManyReq, Forbidden, ServiceUnavailable, UnAuthorized, InvalidInput, DatabaseError, InternalError:
		return true
	default:
		return false
	}
}
