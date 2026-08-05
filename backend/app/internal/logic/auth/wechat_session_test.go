package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
	"wplink/backend/common/externalcall"

	"github.com/zeromicro/go-zero/core/logx/logtest"
)

func TestWechatSessionClientExchangesCode(t *testing.T) {
	observer := &recordingExternalCallObserver{}
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

	client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient, observer)
	session, err := client.Code2Session(context.Background(), "wx-code")
	if err != nil {
		t.Fatalf("Code2Session() error = %v", err)
	}
	if session.OpenID != "openid-1" || session.UnionID != "union-1" {
		t.Fatalf("session = %#v, want wechat ids", session)
	}
	if len(observer.events) != 1 {
		t.Fatalf("events = %+v, want exactly one code_to_session event", observer.events)
	}
	assertWechatEvent(t, observer.events, 0, externalcall.OperationCodeToSession, externalcall.OutcomeSuccess, http.StatusOK)
}

func TestWechatSessionClientGetsPhoneNumber(t *testing.T) {
	observer := &recordingExternalCallObserver{}
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

	client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient, observer).
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
	if len(observer.events) != 2 {
		t.Fatalf("events = %+v, want exactly token and phone events", observer.events)
	}
	assertWechatEvent(t, observer.events, 0, externalcall.OperationAccessToken, externalcall.OutcomeSuccess, http.StatusOK)
	assertWechatEvent(t, observer.events, 1, externalcall.OperationPhoneNumber, externalcall.OutcomeSuccess, http.StatusOK)
}

func assertWechatEvent(t *testing.T, events []externalcall.Event, index int, operation string, outcome externalcall.Outcome, statusCode int) {
	t.Helper()
	if len(events) <= index {
		t.Fatalf("events = %+v, want event at index %d", events, index)
	}
	event := events[index]
	if event.Provider != externalcall.ProviderWechat || event.Operation != operation || event.Outcome != outcome || event.StatusCode != statusCode {
		t.Fatalf("event[%d] = %+v, want wechat/%s/%s/%d", index, event, operation, outcome, statusCode)
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

func TestWechatSessionClientObservesCode2SessionFailures(t *testing.T) {
	tests := []struct {
		name        string
		response    *http.Response
		transport   error
		wantOutcome externalcall.Outcome
		wantStatus  int
		wantCode    string
	}{
		{
			name:        "timeout",
			transport:   context.DeadlineExceeded,
			wantOutcome: externalcall.OutcomeTimeout,
			wantCode:    errx.CodeInternalError,
		},
		{
			name:        "http error takes priority over provider body",
			response:    wechatHTTPResponse(http.StatusBadGateway, `{"errcode":40029,"errmsg":"invalid code"}`),
			wantOutcome: externalcall.OutcomeHTTPError,
			wantStatus:  http.StatusBadGateway,
			wantCode:    errx.CodeUnauthorized,
		},
		{
			name:        "provider rejected",
			response:    wechatHTTPResponse(http.StatusOK, `{"errcode":40029,"errmsg":"invalid code"}`),
			wantOutcome: externalcall.OutcomeProviderRejected,
			wantStatus:  http.StatusOK,
			wantCode:    errx.CodeUnauthorized,
		},
		{
			name:        "invalid json",
			response:    wechatHTTPResponse(http.StatusOK, `{`),
			wantOutcome: externalcall.OutcomeDecodeError,
			wantStatus:  http.StatusOK,
			wantCode:    errx.CodeInternalError,
		},
		{
			name:        "missing openid",
			response:    wechatHTTPResponse(http.StatusOK, `{"unionid":"union-1"}`),
			wantOutcome: externalcall.OutcomeDecodeError,
			wantStatus:  http.StatusOK,
			wantCode:    errx.CodeUnauthorized,
		},
		{
			name: "body read failure",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatErrorReadCloser{err: errors.New("response stream interrupted")},
			},
			wantOutcome: externalcall.OutcomeTransportError,
			wantStatus:  http.StatusPartialContent,
			wantCode:    errx.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingExternalCallObserver{}
			client := NewWechatSessionClient(
				config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
				"https://wechat.example.test/session",
				&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return tt.response, tt.transport
				})},
				observer,
			)

			_, err := client.Code2Session(context.Background(), "wx-code")
			if err == nil || errx.CodeOf(err) != tt.wantCode {
				t.Fatalf("Code2Session() error = %v, code = %q, want %q", err, errx.CodeOf(err), tt.wantCode)
			}
			if tt.wantOutcome == externalcall.OutcomeProviderRejected && errx.PublicMessage(err) != "微信登录凭证无效，请重新登录" {
				t.Fatalf("public message = %q, want unchanged unauthorized guidance", errx.PublicMessage(err))
			}
			if len(observer.events) != 1 {
				t.Fatalf("events = %+v, want exactly one terminal event", observer.events)
			}
			assertWechatEvent(t, observer.events, 0, externalcall.OperationCodeToSession, tt.wantOutcome, tt.wantStatus)
		})
	}
}

func TestWechatSessionClientObservesAccessTokenFailures(t *testing.T) {
	tests := []struct {
		name        string
		response    *http.Response
		transport   error
		wantOutcome externalcall.Outcome
		wantStatus  int
	}{
		{name: "timeout", transport: context.DeadlineExceeded, wantOutcome: externalcall.OutcomeTimeout},
		{name: "http error takes priority", response: wechatHTTPResponse(http.StatusServiceUnavailable, `{"errcode":40013}`), wantOutcome: externalcall.OutcomeHTTPError, wantStatus: http.StatusServiceUnavailable},
		{name: "provider rejected", response: wechatHTTPResponse(http.StatusOK, `{"errcode":40013,"errmsg":"invalid appid"}`), wantOutcome: externalcall.OutcomeProviderRejected, wantStatus: http.StatusOK},
		{name: "invalid json", response: wechatHTTPResponse(http.StatusOK, `{`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "missing access token", response: wechatHTTPResponse(http.StatusOK, `{"expires_in":7200}`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{
			name: "body read failure",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatErrorReadCloser{err: errors.New("response stream interrupted")},
			},
			wantOutcome: externalcall.OutcomeTransportError,
			wantStatus:  http.StatusPartialContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingExternalCallObserver{}
			client := NewWechatSessionClient(
				config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
				"https://wechat.example.test/session",
				&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return tt.response, tt.transport
				})},
				observer,
			).WithPhoneNumberURLs("https://wechat.example.test/token", "https://wechat.example.test/phone")

			_, err := client.GetPhoneNumber(context.Background(), "phone-code")
			if err == nil || errx.CodeOf(err) != errx.CodeInternalError {
				t.Fatalf("GetPhoneNumber() error = %v, want internal error", err)
			}
			if len(observer.events) != 1 {
				t.Fatalf("events = %+v, want exactly one token event", observer.events)
			}
			assertWechatEvent(t, observer.events, 0, externalcall.OperationAccessToken, tt.wantOutcome, tt.wantStatus)
		})
	}
}

func TestWechatSessionClientObservesPhoneNumberFailures(t *testing.T) {
	tests := []struct {
		name        string
		response    *http.Response
		transport   error
		wantOutcome externalcall.Outcome
		wantStatus  int
	}{
		{name: "timeout", transport: context.DeadlineExceeded, wantOutcome: externalcall.OutcomeTimeout},
		{name: "http error takes priority", response: wechatHTTPResponse(http.StatusBadGateway, `{"errcode":40029}`), wantOutcome: externalcall.OutcomeHTTPError, wantStatus: http.StatusBadGateway},
		{name: "provider rejected", response: wechatHTTPResponse(http.StatusOK, `{"errcode":40029,"errmsg":"invalid code"}`), wantOutcome: externalcall.OutcomeProviderRejected, wantStatus: http.StatusOK},
		{name: "invalid json", response: wechatHTTPResponse(http.StatusOK, `{`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "missing phone number", response: wechatHTTPResponse(http.StatusOK, `{"errcode":0,"phone_info":{}}`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{
			name: "body read failure",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatErrorReadCloser{err: errors.New("response stream interrupted")},
			},
			wantOutcome: externalcall.OutcomeTransportError,
			wantStatus:  http.StatusPartialContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingExternalCallObserver{}
			client := NewWechatSessionClient(
				config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
				"https://wechat.example.test/session",
				&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if request.URL.Path == "/token" {
						return wechatHTTPResponse(http.StatusOK, `{"access_token":"access-token","expires_in":7200}`), nil
					}
					return tt.response, tt.transport
				})},
				observer,
			).WithPhoneNumberURLs("https://wechat.example.test/token", "https://wechat.example.test/phone")

			_, err := client.GetPhoneNumber(context.Background(), "phone-code")
			if err == nil || errx.CodeOf(err) != errx.CodeInternalError {
				t.Fatalf("GetPhoneNumber() error = %v, want internal error", err)
			}
			if len(observer.events) != 2 {
				t.Fatalf("events = %+v, want token success then phone failure", observer.events)
			}
			if first := observer.events[0]; first.Operation != externalcall.OperationAccessToken || first.Outcome != externalcall.OutcomeSuccess {
				t.Fatalf("first event = %+v, want access_token/success", first)
			}
			assertWechatEvent(t, observer.events, 1, externalcall.OperationPhoneNumber, tt.wantOutcome, tt.wantStatus)
		})
	}
}

func TestWechatSessionClientDoesNotObserveLocalFailures(t *testing.T) {
	tests := []struct {
		name   string
		client *HTTPWechatSessionClient
		ctx    context.Context
		code   string
		phone  bool
	}{
		{name: "empty login code", code: ""},
		{name: "legacy login code", code: "local-dev-login"},
		{name: "missing login config", code: "wx-code", client: NewWechatSessionClient(config.WechatConfig{}, "https://wechat.example.test/session", nil)},
		{name: "invalid login url", code: "wx-code", client: NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "%", nil)},
		{name: "login request creation failure", ctx: nil, code: "wx-code"},
		{name: "empty phone code", code: "", phone: true},
		{name: "legacy phone code", code: "local-dev-phone", phone: true},
		{name: "missing phone config", code: "phone-code", phone: true, client: NewWechatSessionClient(config.WechatConfig{}, "", nil)},
		{name: "invalid token url", code: "phone-code", phone: true},
		{name: "token request creation failure", ctx: nil, code: "phone-code", phone: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingExternalCallObserver{}
			requests := 0
			if tt.ctx == nil && !strings.Contains(tt.name, "request creation") {
				tt.ctx = context.Background()
			}
			client := tt.client
			if client == nil {
				client = NewWechatSessionClient(
					config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
					"https://wechat.example.test/session",
					&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						requests++
						return wechatHTTPResponse(http.StatusOK, `{}`), nil
					})},
					observer,
				)
			} else {
				client.observer = observer
				client.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					requests++
					return wechatHTTPResponse(http.StatusOK, `{}`), nil
				})}
			}
			if tt.name == "invalid token url" {
				client.WithPhoneNumberURLs("%", "https://wechat.example.test/phone")
			}

			if tt.phone {
				_, _ = client.GetPhoneNumber(tt.ctx, tt.code)
			} else {
				_, _ = client.Code2Session(tt.ctx, tt.code)
			}
			if requests != 0 {
				t.Fatalf("requests = %d, want 0", requests)
			}
			if len(observer.events) != 0 {
				t.Fatalf("events = %+v, want none for local failure", observer.events)
			}
		})
	}
}

func TestWechatSessionClientPhoneRequestCreationFailureDoesNotObservePhone(t *testing.T) {
	observer := &recordingExternalCallObserver{}
	client := NewWechatSessionClient(
		config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		"https://wechat.example.test/session",
		&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return wechatHTTPResponse(http.StatusOK, `{"access_token":"access-token","expires_in":7200}`), nil
		})},
		observer,
	).WithPhoneNumberURLs("https://wechat.example.test/token", "%")

	_, _ = client.GetPhoneNumber(context.Background(), "phone-code")

	if len(observer.events) != 1 {
		t.Fatalf("events = %+v, want only the completed token request", observer.events)
	}
	assertWechatEvent(t, observer.events, 0, externalcall.OperationAccessToken, externalcall.OutcomeSuccess, http.StatusOK)
}

func TestWechatSessionClientUsesOnlyFirstOptionalObserver(t *testing.T) {
	first := &recordingExternalCallObserver{}
	second := &recordingExternalCallObserver{}
	client := NewWechatSessionClient(
		config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		"https://wechat.example.test/session",
		&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return wechatHTTPResponse(http.StatusOK, `{"openid":"openid-1"}`), nil
		})},
		first,
		second,
	)

	_, err := client.Code2Session(context.Background(), "wx-code")
	if err != nil {
		t.Fatalf("Code2Session() error = %v", err)
	}
	if len(first.events) != 1 || len(second.events) != 0 {
		t.Fatalf("first events = %+v, second events = %+v, want only first observer used", first.events, second.events)
	}
}

func TestWechatSessionClientTransportLogsDoNotLeakSecrets(t *testing.T) {
	const (
		appID        = "sentinel-appid"
		appSecret    = "sentinel-secret"
		code         = "sentinel-code"
		accessToken  = "sentinel-access-token"
		transportRaw = "sentinel-transport-error"
	)
	tests := []struct {
		name      string
		operation string
		run       func(*HTTPWechatSessionClient)
	}{
		{name: "code to session", operation: externalcall.OperationCodeToSession, run: func(client *HTTPWechatSessionClient) { _, _ = client.Code2Session(context.Background(), code) }},
		{name: "access token", operation: externalcall.OperationAccessToken, run: func(client *HTTPWechatSessionClient) { _, _ = client.GetPhoneNumber(context.Background(), code) }},
		{name: "phone number", operation: externalcall.OperationPhoneNumber, run: func(client *HTTPWechatSessionClient) { _, _ = client.GetPhoneNumber(context.Background(), code) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := logtest.NewCollector(t)
			observer := &recordingExternalCallObserver{}
			client := NewWechatSessionClient(
				config.WechatConfig{AppID: appID, AppSecret: appSecret},
				"https://wechat.example.test/session",
				&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if tt.operation == externalcall.OperationPhoneNumber && request.URL.Path == "/token" {
						return wechatHTTPResponse(http.StatusOK, fmt.Sprintf(`{"access_token":%q,"expires_in":7200}`, accessToken)), nil
					}
					return nil, &url.Error{
						Op:  request.Method,
						URL: request.URL.String(),
						Err: fmt.Errorf("%s: %w", transportRaw, context.DeadlineExceeded),
					}
				})},
				observer,
			).WithPhoneNumberURLs("https://wechat.example.test/token", "https://wechat.example.test/phone")

			tt.run(client)

			logText := collector.String()
			if !strings.Contains(logText, "cause=timeout") {
				t.Fatalf("log = %q, want safe timeout cause", logText)
			}
			for _, forbidden := range []string{appID, appSecret, code, accessToken, transportRaw, "appid=", "secret=", "js_code=", "access_token=", "openid=", "session_key=", "https://wechat.example.test/"} {
				if strings.Contains(logText, forbidden) {
					t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
				}
			}
		})
	}
}

func TestWechatSessionClientBodyReadLogsDoNotLeakSecrets(t *testing.T) {
	const (
		appID     = "sentinel-body-appid"
		appSecret = "sentinel-body-secret"
		code      = "sentinel-body-code"
		readRaw   = "sentinel-body-read-error"
	)
	collector := logtest.NewCollector(t)
	observer := &recordingExternalCallObserver{}
	client := NewWechatSessionClient(
		config.WechatConfig{AppID: appID, AppSecret: appSecret},
		"https://wechat.example.test/session",
		&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatErrorReadCloser{err: fmt.Errorf("%s: %w", readRaw, io.ErrUnexpectedEOF)},
			}, nil
		})},
		observer,
	)

	_, _ = client.Code2Session(context.Background(), code)

	logText := collector.String()
	if !strings.Contains(logText, "cause=unexpected_eof") {
		t.Fatalf("log = %q, want safe unexpected_eof cause", logText)
	}
	for _, forbidden := range []string{appID, appSecret, code, readRaw, "appid=", "secret=", "js_code=", "https://wechat.example.test/"} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
		}
	}
}

func TestWechatSessionClientProviderLogsDoNotLeakRawMessage(t *testing.T) {
	const providerMessage = "sentinel-provider-raw-message"
	collector := logtest.NewCollector(t)
	observer := &recordingExternalCallObserver{}
	client := NewWechatSessionClient(
		config.WechatConfig{AppID: "sentinel-appid", AppSecret: "sentinel-secret"},
		"https://wechat.example.test/session",
		&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return wechatHTTPResponse(http.StatusOK, `{"errcode":40029,"errmsg":"`+providerMessage+`"}`), nil
		})},
		observer,
	)

	_, _ = client.Code2Session(context.Background(), "sentinel-code")

	logText := collector.String()
	for _, forbidden := range []string{providerMessage, "sentinel-appid", "sentinel-secret", "sentinel-code"} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
		}
	}
}

func wechatHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type wechatErrorReadCloser struct {
	err error
}

func (r wechatErrorReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (wechatErrorReadCloser) Close() error {
	return nil
}
