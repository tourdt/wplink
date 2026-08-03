package auth

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
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

func TestWechatSessionClientGetsPhoneNumber(t *testing.T) {
	requestedToken := false
	requestedPhone := false
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/token":
			requestedToken = true
			if r.URL.Query().Get("appid") != "wx-app" || r.URL.Query().Get("secret") != "wx-secret" || r.URL.Query().Get("grant_type") != "client_credential" {
				t.Fatalf("token query = %s, want app secret grant", r.URL.RawQuery)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioNopCloser{Buffer: bytes.NewBufferString(`{"access_token":"access-token","expires_in":7200}`)},
				Header:     make(http.Header),
			}, nil
		case "/phone":
			requestedPhone = true
			if r.URL.Query().Get("access_token") != "access-token" {
				t.Fatalf("phone query = %s, want access token", r.URL.RawQuery)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read phone body: %v", err)
			}
			if string(body) != `{"code":"phone-code"}`+"\n" {
				t.Fatalf("phone body = %q, want phone code json", string(body))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioNopCloser{Buffer: bytes.NewBufferString(`{"errcode":0,"phone_info":{"phoneNumber":"8618800000003","purePhoneNumber":"18800000003","countryCode":"86"}}`)},
				Header:     make(http.Header),
			}, nil
		default:
			t.Fatalf("unexpected path = %s", r.URL.Path)
			return nil, nil
		}
	})}

	client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient).
		WithPhoneNumberURLs("https://wechat.example.test/token", "https://wechat.example.test/phone")
	phone, err := client.GetPhoneNumber(context.Background(), " phone-code ")
	if err != nil {
		t.Fatalf("GetPhoneNumber() error = %v", err)
	}
	if !requestedToken || !requestedPhone {
		t.Fatalf("requested token=%t phone=%t, want both endpoints called", requestedToken, requestedPhone)
	}
	if phone.PurePhoneNumber != "18800000003" || phone.CountryCode != "86" {
		t.Fatalf("phone = %#v, want wechat phone info", phone)
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

func TestWechatSessionClientRejectsLegacyDevLoginCodeWithoutCallingWechat(t *testing.T) {
	called := false
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioNopCloser{Buffer: bytes.NewBufferString(`{"openid":"openid-1"}`)},
			Header:     make(http.Header),
		}, nil
	})}

	client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient)
	_, err := client.Code2Session(context.Background(), "local-dev-123")
	if errx.CodeOf(err) != errx.CodeUnauthorized || errx.PublicMessage(err) != "请在微信内重新登录" {
		t.Fatalf("Code2Session() error = %v, want local unauthorized error", err)
	}
	if called {
		t.Fatal("wechat session endpoint was called for legacy dev code")
	}
}

func TestWechatSessionClientRejectsLegacyDevPhoneCodeWithoutCallingWechat(t *testing.T) {
	called := false
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioNopCloser{Buffer: bytes.NewBufferString(`{"access_token":"access-token"}`)},
			Header:     make(http.Header),
		}, nil
	})}

	client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient)
	_, err := client.GetPhoneNumber(context.Background(), "local-dev-phone-18800000000")
	if errx.CodeOf(err) != errx.CodeUnauthorized || errx.PublicMessage(err) != "请在微信内重新授权手机号" {
		t.Fatalf("GetPhoneNumber() error = %v, want local unauthorized error", err)
	}
	if called {
		t.Fatal("wechat phone endpoint was called for legacy dev code")
	}
}
