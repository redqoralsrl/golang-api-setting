package middleware

import (
	"net"
	"net/http"
	"strings"
)

func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	for _, candidate := range []string{
		r.Header.Get("CF-Connecting-IP"),
		r.Header.Get("X-Real-IP"),
		firstForwardedIP(r.Header.Get("X-Forwarded-For")),
		r.RemoteAddr,
	} {
		if ip := normalizeIP(candidate); ip != "" {
			return ip
		}
	}

	return ""
}

func firstForwardedIP(value string) string {
	if value == "" {
		return ""
	}

	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return ""
	}

	return strings.TrimSpace(parts[0])
}

func normalizeIP(value string) string {
	address := strings.TrimSpace(value)
	if address == "" {
		return ""
	}

	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return strings.TrimSpace(host)
	}

	return address
}
