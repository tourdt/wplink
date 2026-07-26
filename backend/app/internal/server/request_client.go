package server

import (
	"net"
	"net/http"
	"strings"
)

func requestClientIP(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	// 生产 API 仅监听本机并由 Nginx 转发，因此优先读取代理写入的首个来源 IP。
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if remoteAddr := strings.TrimSpace(r.RemoteAddr); remoteAddr != "" {
		return remoteAddr
	}
	return "unknown"
}
