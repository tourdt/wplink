package auth

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"wplink/backend/app/internal/config"
)

func TestWechatSessionClientExchangesCode(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("appid") != "wx-app" || r.URL.Query().Get("secret") != "wx-secret" || r.URL.Query().Get("js_code") != "wx-code" {
			t.Fatalf("query = %s, want app secret and code", r.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioNopCloser{Buffer: bytes.NewBufferString(`{"openid":"openid-1","unionid":"union-1"}`)},
			Header:     make(http.Header),
		}, nil
	})}

	client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient)
	session, err := client.Code2Session(context.Background(), "wx-code")
	if err != nil {
		t.Fatalf("Code2Session() error = %v", err)
	}
	if session.OpenID != "openid-1" || session.UnionID != "union-1" {
		t.Fatalf("session = %#v, want wechat ids", session)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type ioNopCloser struct {
	*bytes.Buffer
}

func (c ioNopCloser) Close() error {
	return nil
}

func TestWechatSessionClientAllowsDevCodeOnlyWhenConfigured(t *testing.T) {
	client := NewWechatSessionClient(config.WechatConfig{AllowDevCode: true}, "", nil)

	session, err := client.Code2Session(context.Background(), "local-dev-123")
	if err != nil {
		t.Fatalf("Code2Session() error = %v", err)
	}
	if session.OpenID != "dev:local-dev-123" {
		t.Fatalf("openid = %q, want dev openid", session.OpenID)
	}
}
