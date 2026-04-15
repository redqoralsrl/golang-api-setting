package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	errorlogDomain "go-template/domain/errorlog"
	"go-template/internal/logger"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

var (
	slowRequestThreshold = 1000 * time.Millisecond

	// 완전히 로깅하지 않을 엔드포인트 목록
	skipLoggingPaths = []string{
		"/health",
		"/ping",
	}

	// 바디를 로깅하지 않을 엔드포인트 목록
	skipBodyLoggingPaths = []string{
		"/health",
	}

	// 필드 마스킹 목록
	sensitiveFields = []string{
		"password",
		"credit_card",
		"secret",
	}

	// URL path에 이런 조각이 들어가면 의심스러운 스캐너/봇 요청
	suspiciousPathFragments = []string{
		".env",
		".php",
		"phpunit",
		"vendor/phpunit",
	}

	// query string에서 이런 패턴이 보이면 역시 공격성 요청
	suspiciousQueryFragments = []string{
		"allow_url_include",
		"auto_prepend_file",
		"php://input",
	}
)

// maskSensitiveData 민감한 데이터를 마스킹
func maskSensitiveData(data map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{})
	for k, v := range data {
		if v == nil {
			masked[k] = v
			continue
		}

		if nested, ok := v.(map[string]interface{}); ok {
			masked[k] = maskSensitiveData(nested)
			continue
		}

		shouldMask := false
		for _, sensitive := range sensitiveFields {
			if strings.Contains(strings.ToLower(k), sensitive) {
				shouldMask = true
				break
			}
		}

		if shouldMask {
			masked[k] = "******"
		} else {
			masked[k] = v
		}
	}
	return masked
}

// shouldSkipLogging 완전한 로깅을 스킵할지 결정
func shouldSkipLogging(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}

	path := strings.TrimSpace(r.URL.Path)
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if path == "/" && (method == http.MethodGet || method == http.MethodHead) {
		return true
	}

	for _, skip := range skipLoggingPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

// shouldSkipBodyLogging 바디 로깅을 스킵할지 결정
func shouldSkipBodyLogging(path string) bool {
	for _, skip := range skipBodyLoggingPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

func isSuspiciousProbeRequest(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}

	path := strings.ToLower(strings.TrimSpace(r.URL.Path))
	query := strings.ToLower(r.URL.RawQuery)

	for _, fragment := range suspiciousPathFragments {
		if strings.Contains(path, fragment) {
			return true
		}
	}

	for _, fragment := range suspiciousQueryFragments {
		if strings.Contains(query, fragment) {
			return true
		}
	}

	return false
}

type errorResponseBody struct {
	Message string  `json:"message"`
	Error   *string `json:"error"`
}

// LoggerMiddleware HTTP 요청/응답 로깅을 위한 미들웨어
func LoggerMiddleware(l logger.Logger, errorLogService errorlogDomain.UseCase) func(next http.Handler) http.Handler {
	httpLogger := l.With(logger.NewField("component", "http"))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipLogging(r) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			requestID := chiMiddleware.GetReqID(r.Context())
			reqLogger := httpLogger.With(logger.NewField("request_id", requestID))
			isSuspiciousProbe := isSuspiciousProbeRequest(r)

			fields := []logger.Field{
				logger.NewField("method", r.Method),
				logger.NewField("path", r.URL.Path),
				logger.NewField("remote_addr", r.RemoteAddr),
				logger.NewField("user_agent", r.UserAgent()),
			}

			if query := r.URL.Query().Encode(); query != "" {
				fields = append(fields, logger.NewField("query", query))
			}

			if !shouldSkipBodyLogging(r.URL.Path) &&
				strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				var requestBody map[string]interface{}

				if r.Body != nil {
					bodyBytes, err := io.ReadAll(r.Body)
					if err == nil {
						// 원본 바디 복원
						r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

						if err := json.Unmarshal(bodyBytes, &requestBody); err == nil {
							maskedBody := maskSensitiveData(requestBody)
							fields = append(fields, logger.NewField("body", maskedBody))
						}
					}
				}
			}

			if !isSuspiciousProbe {
				reqLogger.Info("incoming request", fields...)
			}

			var responseBody bytes.Buffer

			ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			if errorLogService != nil {
				ww.Tee(&responseBody)
			}
			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			responseFields := []logger.Field{
				logger.NewField("status", ww.Status()),
				logger.NewField("bytes_written", ww.BytesWritten()),
				logger.NewField("duration_ms", duration.Milliseconds()),
			}

			if duration >= slowRequestThreshold {
				routePattern := r.URL.Path
				if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
					if pattern := routeContext.RoutePattern(); pattern != "" {
						routePattern = pattern
					}
				}

				slowRequestFields := []logger.Field{
					logger.NewField("method", r.Method),
					logger.NewField("path", r.URL.Path),
					logger.NewField("route_pattern", routePattern),
					logger.NewField("duration_ms", duration.Milliseconds()),
					logger.NewField("status", ww.Status()),
				}
				reqLogger.Warn("slow request detected", slowRequestFields...)
			}

			if isSuspiciousProbe && ww.Status() < http.StatusInternalServerError && duration < slowRequestThreshold {
				return
			}

			switch {
			case ww.Status() >= http.StatusInternalServerError:
				persistErrorLog(r, ww.Status(), responseBody.Bytes(), errorLogService, reqLogger)
				reqLogger.Error("request failed", responseFields...)
			case ww.Status() >= http.StatusBadRequest:
				persistErrorLog(r, ww.Status(), responseBody.Bytes(), errorLogService, reqLogger)
				reqLogger.Warn("request completed with client error", responseFields...)
			default:
				reqLogger.Info("request completed", responseFields...)
			}
		})
	}
}

func persistErrorLog(r *http.Request, status int, responseBody []byte, errorLogService errorlogDomain.UseCase, reqLogger logger.Logger) {
	if errorLogService == nil || r == nil || !shouldPersistErrorLog(status) {
		return
	}

	_, err := errorLogService.Create(r.Context(), &errorlogDomain.ErrorLog{
		Timestamp:    time.Now().UTC(),
		IPAddress:    ClientIP(r),
		UserAgent:    strings.TrimSpace(r.UserAgent()),
		Path:         strings.TrimSpace(r.URL.Path),
		HTTPMethod:   strings.TrimSpace(r.Method),
		RequestedURL: requestedURL(r),
		ErrorCode:    status,
		ErrorMessage: extractErrorMessage(status, responseBody),
	})
	if err != nil {
		reqLogger.Warn("failed to persist error log",
			logger.NewField("status", status),
			logger.NewError(err),
		)
	}
}

func shouldPersistErrorLog(status int) bool {
	if status < http.StatusBadRequest {
		return false
	}

	switch status {
	case http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusServiceUnavailable:
		return false
	default:
		return true
	}
}

func extractErrorMessage(status int, responseBody []byte) string {
	body := strings.TrimSpace(string(responseBody))
	if body == "" {
		return fallbackErrorMessage(status)
	}

	var payload errorResponseBody
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return body
	}

	if payload.Error != nil && strings.TrimSpace(*payload.Error) != "" {
		return strings.TrimSpace(*payload.Error)
	}
	if strings.TrimSpace(payload.Message) != "" {
		return strings.TrimSpace(payload.Message)
	}

	return body
}

func fallbackErrorMessage(status int) string {
	message := strings.TrimSpace(http.StatusText(status))
	if message == "" {
		return "request failed"
	}

	return message
}

func requestedURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}

	requestURI := strings.TrimSpace(r.URL.RequestURI())
	if requestURI != "" {
		return requestURI
	}

	return strings.TrimSpace(r.URL.Path)
}
