package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type clientIPKey struct{}

// ClientIPMiddleware 요청마다 클라이언트 IP를 한 번 정해 context 에 넣는다.
//
// trustedHops 는 앞단 프록시(로드밸런서) 수다. 프록시는 X-Forwarded-For 맨 뒤에 접속 IP를 붙이므로
// 오른쪽에서 trustedHops 번째 값이 실제 클라이언트다. 그보다 왼쪽 값과 CF-Connecting-IP, X-Real-IP 같은
// 헤더는 클라이언트가 위조할 수 있어 읽지 않는다.
//   - 0: 프록시 없이 직접 노출 (RemoteAddr 사용)
//   - 1: ALB/nginx 하나 뒤
//   - 2: CloudFront → ALB 처럼 두 단계 뒤
func ClientIPMiddleware(trustedHops int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveClientIP(r, trustedHops)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}
}

// ClientIP ClientIPMiddleware 가 정한 IP. 미들웨어를 거치지 않았으면 RemoteAddr 를 쓴다.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if ip, ok := r.Context().Value(clientIPKey{}).(string); ok && ip != "" {
		return ip
	}
	return normalizeIP(r.RemoteAddr)
}

func resolveClientIP(r *http.Request, trustedHops int) string {
	if trustedHops > 0 {
		var hops []string
		for _, v := range r.Header.Values("X-Forwarded-For") {
			hops = append(hops, strings.Split(v, ",")...)
		}
		if i := len(hops) - trustedHops; i >= 0 {
			if ip := normalizeIP(hops[i]); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	return normalizeIP(r.RemoteAddr)
}

func normalizeIP(value string) string {
	address := strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}
