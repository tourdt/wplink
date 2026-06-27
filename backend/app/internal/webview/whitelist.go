package webview

import (
	"net/url"
	"strings"
)

var allowedHosts = map[string]struct{}{
	"www.wplink.cn":    {},
	"m.wplink.cn":      {},
	"mp.weixin.qq.com": {},
}

func IsAllowedURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	_, ok := allowedHosts[strings.ToLower(parsed.Hostname())]
	return ok
}
