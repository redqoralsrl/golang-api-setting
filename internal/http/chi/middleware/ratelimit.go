package middleware

import (
	"errors"
	"net/http"
	"sync"
	"time"

	response "go-template/internal/http"

	"golang.org/x/time/rate"
)

const visitorTTL = 3 * time.Minute

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimiter IP별 토큰 버킷. 서버 한 대 메모리 기준이라 여러 대면 대당 한도다.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     rate.Limit
	burst    int
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter.Allow()
}

func (rl *rateLimiter) cleanup() {
	for range time.Tick(time.Minute) {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > visitorTTL {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware IP별 초당 rps 요청, 순간 burst 까지 허용한다.
func RateLimitMiddleware(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	rl := &rateLimiter{visitors: make(map[string]*visitor), rate: rps, burst: burst}
	go rl.cleanup()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isProbePath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if !rl.allow(ClientIP(r)) {
				w.Header().Set("Retry-After", "1")
				response.WriteError(w, http.StatusTooManyRequests, response.TooManyReq, errors.New("rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
