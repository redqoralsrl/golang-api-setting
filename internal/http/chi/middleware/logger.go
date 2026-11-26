package middleware

import (
	"bytes"
	"context"
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

const (
	HealthPath = "/health"
	ReadyPath  = "/ready"
)

var (
	slowRequestThreshold = 1000 * time.Millisecond

	// 요청 바디는 이 크기 이하일 때만 읽어 로깅한다.
	maxLoggedBodyBytes int64 = 64 << 10
	// 에러 로그에 남길 응답 바디 최대 크기. 에러 응답(4xx/5xx)만 담는다.
	maxCapturedResponseBytes = 4 << 10

	// 완전히 로깅하지 않을 경로 (prefix)
	skipLoggingPaths = []string{HealthPath, ReadyPath, "/swagger/"}

	// 요청 바디에서 값을 그대로 남겨도 되는 필드. 그 외 필드는 이름만 남기고 값은 가린다.
	// 개인정보·비밀값이 새 필드로 추가돼도 기본으로 가려지도록 허용 목록으로 둔다.
	// 개인정보나 비밀값이 아닌 필드만 추가한다.
	loggableBodyFields = map[string]bool{
		"id": true, "ids": true, "type": true, "status": true, "role": true,
		"limit": true, "cursor": true, "sort": true, "order": true,
	}

	// URL path에 이런 조각이 들어가면 스캐너/봇 요청으로 보고 정상 로그를 남기지 않는다.
	// 이 API의 라우트에는 없는 조각만 둔다. 라우트를 추가할 때 여기에 걸리지 않는지 확인한다.
	suspiciousPathFragments = []string{
		// 설정·비밀 파일
		".env", ".git", ".svn", ".aws", ".ssh", ".htaccess", ".htpasswd", ".ds_store",
		"wp-config", "config.json", "credentials", "id_rsa", "docker-compose", "web.config",
		// 다른 언어·프레임워크 엔드포인트
		".php", ".asp", ".jsp", ".cgi", "cgi-bin", "phpunit", "phpmyadmin", "vendor/",
		"wp-", "wordpress", "xmlrpc", "actuator", "_profiler", "telescope",
		"server-status", "manager/html", "solr", "boaform", "hnap1", "owa/",
		// 백업·덤프 파일
		".sql", ".bak", ".old", ".tar", ".gz", ".zip", ".log",
		// 경로 조작
		"..", "%2e%2e", "etc/passwd",
	}

	// query string에서 이런 패턴이 보이면 공격성 요청
	suspiciousQueryFragments = []string{
		"allow_url_include", "auto_prepend_file", "php://", "${jndi:", "../",
		"etc/passwd", "<script", "union select", "union+select",
	}
)

// isProbePath 헬스체크 경로. 로깅과 rate limit 에서 뺀다.
func isProbePath(path string) bool {
	return path == HealthPath || path == ReadyPath
}

// LoggerMiddleware HTTP 요청/응답 로깅. 4xx/5xx 응답은 errorLogService 로 DB에 비동기 저장한다.
func LoggerMiddleware(l logger.Logger, errorLogService errorlogDomain.UseCase) func(next http.Handler) http.Handler {
	httpLogger := l.With(logger.NewField("component", "http"))
	persister := newErrorLogPersister(errorLogService, httpLogger)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipLogging(r) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			reqLogger := httpLogger.With(logger.NewField("request_id", chiMiddleware.GetReqID(r.Context())))
			isProbe := isSuspiciousProbeRequest(r)

			if !isProbe {
				reqLogger.Info("incoming request", requestFields(r)...)
			}

			ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			capture := &errorBodyCapture{ww: ww}
			ww.Tee(capture)
			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}

			if duration >= slowRequestThreshold {
				reqLogger.Warn("slow request detected",
					logger.NewField("method", r.Method),
					logger.NewField("route_pattern", routePattern(r)),
					logger.NewField("duration_ms", duration.Milliseconds()),
					logger.NewField("status", status),
				)
			}

			if isProbe && status < http.StatusInternalServerError {
				return
			}

			fields := []logger.Field{
				logger.NewField("status", status),
				logger.NewField("bytes_written", ww.BytesWritten()),
				logger.NewField("duration_ms", duration.Milliseconds()),
			}
			switch {
			case status >= http.StatusInternalServerError:
				reqLogger.Error("request failed", fields...)
			case status >= http.StatusBadRequest:
				reqLogger.Warn("request completed with client error", fields...)
			default:
				reqLogger.Info("request completed", fields...)
				return
			}

			if shouldPersistErrorLog(status) {
				persister.enqueue(&errorlogDomain.ErrorLog{
					OccurredAt:   time.Now().UTC(),
					IPAddress:    ClientIP(r),
					UserAgent:    strings.TrimSpace(r.UserAgent()),
					Path:         r.URL.Path,
					HTTPMethod:   r.Method,
					RequestedURL: r.URL.RequestURI(),
					ErrorCode:    status,
					ErrorMessage: extractErrorMessage(status, capture.Bytes()),
				})
			}
		})
	}
}

func requestFields(r *http.Request) []logger.Field {
	fields := []logger.Field{
		logger.NewField("method", r.Method),
		logger.NewField("path", r.URL.Path),
		logger.NewField("client_ip", ClientIP(r)),
		logger.NewField("user_agent", r.UserAgent()),
	}
	if query := r.URL.RawQuery; query != "" {
		fields = append(fields, logger.NewField("query", query))
	}
	if body := readLoggableBody(r); body != nil {
		fields = append(fields, logger.NewField("body", body))
	}
	return fields
}

// readLoggableBody 작은 JSON 바디만 읽어 마스킹한다. 읽은 바디는 핸들러가 다시 읽을 수 있게 되돌려 둔다.
func readLoggableBody(r *http.Request) map[string]interface{} {
	if r.Body == nil || r.ContentLength <= 0 || r.ContentLength > maxLoggedBodyBytes ||
		!strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxLoggedBodyBytes))
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes), r.Body))
	if err != nil {
		return nil
	}

	var body map[string]interface{}
	if json.Unmarshal(bodyBytes, &body) != nil {
		return nil
	}
	return maskSensitiveData(body)
}

// maskSensitiveData loggableBodyFields에 없는 필드의 값을 가린다. 중첩 객체와 배열도 같은 규칙을 적용한다.
func maskSensitiveData(data map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{}, len(data))
	for k, v := range data {
		masked[k] = maskValue(v, loggableBodyFields[strings.ToLower(k)])
	}
	return masked
}

func maskValue(v interface{}, allowed bool) interface{} {
	switch t := v.(type) {
	case nil:
		return nil
	case map[string]interface{}:
		return maskSensitiveData(t)
	case []interface{}:
		items := make([]interface{}, len(t))
		for i, item := range t {
			items[i] = maskValue(item, allowed)
		}
		return items
	}
	if allowed {
		return v
	}
	return "******"
}

func shouldSkipLogging(r *http.Request) bool {
	path := r.URL.Path
	if path == "/" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return true
	}
	for _, skip := range skipLoggingPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

func isSuspiciousProbeRequest(r *http.Request) bool {
	path := strings.ToLower(r.URL.Path)
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

func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := rc.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return r.URL.Path
}

// errorBodyCapture 에러 응답의 앞부분만 담는다. 정상 응답은 버퍼에 복사하지 않는다.
type errorBodyCapture struct {
	ww  chiMiddleware.WrapResponseWriter
	buf bytes.Buffer
}

func (c *errorBodyCapture) Write(p []byte) (int, error) {
	if c.ww.Status() >= http.StatusBadRequest {
		if room := maxCapturedResponseBytes - c.buf.Len(); room > 0 {
			c.buf.Write(p[:min(len(p), room)])
		}
	}
	return len(p), nil
}

func (c *errorBodyCapture) Bytes() []byte { return c.buf.Bytes() }

// shouldPersistErrorLog 인증 실패·없는 경로·rate limit 처럼 외부에서 쉽게 대량 유발할 수 있는 응답은 저장하지 않는다.
func shouldPersistErrorLog(status int) bool {
	switch status {
	case http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusRequestEntityTooLarge,
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable:
		return false
	}
	return status >= http.StatusBadRequest
}

func extractErrorMessage(status int, responseBody []byte) string {
	var payload struct {
		Message string  `json:"message"`
		Error   *string `json:"error"`
	}
	if json.Unmarshal(responseBody, &payload) == nil {
		if payload.Error != nil && strings.TrimSpace(*payload.Error) != "" {
			return strings.TrimSpace(*payload.Error)
		}
		if strings.TrimSpace(payload.Message) != "" {
			return strings.TrimSpace(payload.Message)
		}
	}
	if body := strings.TrimSpace(string(responseBody)); body != "" {
		return body
	}
	if text := http.StatusText(status); text != "" {
		return text
	}
	return "request failed"
}

// errorLogPersister 에러 로그를 요청 처리와 분리해 백그라운드에서 저장한다.
// 큐가 차면 버린다. 에러가 몰릴 때 DB 쓰기가 요청을 막거나 DB를 압박하지 않게 한다.
type errorLogPersister struct {
	service errorlogDomain.UseCase
	logger  logger.Logger
	queue   chan *errorlogDomain.ErrorLog
}

func newErrorLogPersister(service errorlogDomain.UseCase, l logger.Logger) *errorLogPersister {
	p := &errorLogPersister{service: service, logger: l, queue: make(chan *errorlogDomain.ErrorLog, 256)}
	if service != nil {
		go p.run()
	}
	return p
}

func (p *errorLogPersister) enqueue(e *errorlogDomain.ErrorLog) {
	if p.service == nil {
		return
	}
	select {
	case p.queue <- e:
	default:
		p.logger.Warn("error log queue full, dropped", logger.NewField("status", e.ErrorCode))
	}
}

func (p *errorLogPersister) run() {
	for e := range p.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := p.service.Create(ctx, e); err != nil {
			p.logger.Warn("failed to persist error log", logger.NewField("status", e.ErrorCode), logger.NewError(err))
		}
		cancel()
	}
}
