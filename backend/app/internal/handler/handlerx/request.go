package handlerx

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
)

// ReadLimitedBody 最多读取 limit 字节，并在返回前关闭请求体。
func ReadLimitedBody(r *http.Request, limit int64) ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, errors.New("request body is unavailable")
	}
	if limit < 0 {
		return nil, errors.New("request body limit is invalid")
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("request body exceeds limit")
	}
	return body, nil
}

func ClientIP(r *http.Request) string {
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
