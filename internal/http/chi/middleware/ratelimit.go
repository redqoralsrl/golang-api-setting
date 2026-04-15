package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter IP별 rate limiter 관리
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter 새로운 rate limiter 생성
func NewRateLimiter(rps rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rps,
		burst:    burst,
	}
}

// GetLimiter IP별 limiter 조회/생성
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// RateLimitMiddleware Rate limiting 미들웨어
func RateLimitMiddleware(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	rl := NewRateLimiter(rps, burst)

	// 주기적으로 사용하지 않는 limiter 정리
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rl.mu.Lock()
			for ip, limiter := range rl.limiters {
				// 최근 1분간 토큰이 가득 찬 limiter는 삭제
				if limiter.Tokens() == float64(burst) {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 메트릭, 헬스체크는 rate limit 제외
			if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			ip := ClientIP(r)
			if ip == "" {
				ip = r.RemoteAddr
			}

			limiter := rl.GetLimiter(ip)
			if !limiter.Allow() {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
