package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	response "go-template/internal/http"
	"go-template/internal/logger"
)

func RecoveryMiddleware(l logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rvr := recover()
				if rvr == nil {
					return
				}
				// net/http 가 연결을 끊으려고 일부러 던지는 값은 그대로 다시 던진다
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				l.Error("panic recovered",
					logger.NewField("error", fmt.Sprintf("%v", rvr)),
					logger.NewField("stack", string(debug.Stack())),
					logger.NewField("path", r.URL.Path),
					logger.NewField("method", r.Method),
				)
				response.WriteError(w, http.StatusInternalServerError, response.InternalError, errors.New("panic"))
			}()

			next.ServeHTTP(w, r)
		})
	}
}
