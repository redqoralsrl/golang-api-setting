package middleware

import (
	"fmt"
	response "go-template/internal/http"
	"go-template/internal/logger"
	"net/http"
	"runtime/debug"
)

func RecoveryMiddleware(l logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					stack := debug.Stack()

					l.Error("panic recovered",
						logger.NewField("error", fmt.Sprintf("%v", rvr)),
						logger.NewField("stack", string(stack)),
						logger.NewField("path", r.URL.Path),
						logger.NewField("method", r.Method),
					)

					errMsg := "An unexpected error occurred"

					response.WriteResponse(w, http.StatusInternalServerError, response.InternalError, &errMsg, nil)
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
