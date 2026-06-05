package middleware

import (
	"net/http"
	"strconv"
	"time"

	"go-template/internal/metrics"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		method := r.Method

		ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		metrics.RecordHTTPRequestStart(method)
		defer metrics.RecordHTTPRequestEnd(method)

		next.ServeHTTP(ww, r)

		routePattern := "unknown"
		if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
			if pattern := routeContext.RoutePattern(); pattern != "" {
				routePattern = pattern
			}
		}
		statusCode := ww.Status()
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		status := strconv.Itoa(statusCode)
		metrics.RecordHTTPRequest(method, routePattern, status, time.Since(start))
	})
}
