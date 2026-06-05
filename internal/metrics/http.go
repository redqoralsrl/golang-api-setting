package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HTTP 요청 총 개수
	HTTPRequestsTotal = NewCounter(
		"api_http_requests_total",
		"Total number of HTTP requests.",
		"method",
		"path",
		"status",
	)

	// HTTP 요청 지속 시간
	HTTPRequestDuration = NewHistogram(
		"api_http_request_duration_seconds",
		"HTTP request duration in seconds.",
		prometheus.DefBuckets,
		"method",
		"path",
		"status",
	)

	// 현재 진행 중인 HTTP 요청
	HTTPRequestsInProgress = NewGauge(
		"api_http_requests_in_progress",
		"Current number of HTTP requests being processed.",
		"method",
	)
)

func RecordHTTPRequestStart(method string) {
	HTTPRequestsInProgress.WithLabelValues(method).Inc()
}

func RecordHTTPRequestEnd(method string) {
	HTTPRequestsInProgress.WithLabelValues(method).Dec()
}

func RecordHTTPRequest(method, path, status string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
}
