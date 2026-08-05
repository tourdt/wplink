package contentaudit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	resourcelogic "wplink/backend/app/internal/logic/resource"
	"wplink/backend/common/externalcall"

	"github.com/zeromicro/go-zero/core/logx/logtest"
)

func TestWechatAuditorRejectsLegacyDevOpenIDWithoutCallingWechat(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	auditor := NewWechatAuditor(
		config.WechatConfig{AppID: "wx-app", AppSecret: "secret"},
		config.ContentAuditConfig{Enabled: true, TextScene: 3, RequestTimeout: time.Second},
		"",
		server.Client(),
	).WithURLs(server.URL+"/token", server.URL+"/msg", "")

	_, err := auditor.AuditResource(context.Background(), resourcelogic.ContentAuditInput{OpenID: "dev:local-dev-123"})
	if !errors.Is(err, resourcelogic.ErrLegacyWechatAuditIdentity) {
		t.Fatalf("AuditResource() error = %v, want legacy identity error", err)
	}
	if called {
		t.Fatal("wechat content audit endpoint was called for legacy dev openid")
	}
}

func TestWechatAuditorReturnsRiskyTextDecision(t *testing.T) {
	observer := &recordingWechatAuditObserver{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeJSON(t, w, map[string]any{"access_token": "token-1", "expires_in": 7200})
		case "/msg":
			if r.URL.Query().Get("access_token") != "token-1" {
				t.Fatalf("access_token = %q, want token-1", r.URL.Query().Get("access_token"))
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode msg body: %v", err)
			}
			if body["openid"] != "openid-1" || !strings.Contains(body["content"].(string), "女童春款卫衣库存整包清") {
				t.Fatalf("body = %#v, want openid and content", body)
			}
			writeJSON(t, w, map[string]any{
				"errcode":  0,
				"trace_id": "trace-text",
				"result":   map[string]any{"suggest": "risky", "label": 20006},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	auditor := NewWechatAuditor(
		config.WechatConfig{AppID: "wx-app", AppSecret: "secret"},
		config.ContentAuditConfig{Enabled: true, TextScene: 3, RequestTimeout: time.Second, MaxTextChars: 2500},
		"",
		server.Client(),
		observer,
	).WithURLs(server.URL+"/token", server.URL+"/msg", "")

	result, err := auditor.AuditResource(context.Background(), resourcelogic.ContentAuditInput{
		OpenID:      "openid-1",
		MerchantID:  "merchant-1",
		TypeCode:    "inventory",
		Title:       "女童春款卫衣库存整包清",
		Description: "整包优先，可现场看货。",
	})
	if err != nil {
		t.Fatalf("AuditResource() error = %v", err)
	}
	if result.Decision != resourcelogic.ContentAuditDecisionRisky {
		t.Fatalf("decision = %q, want risky", result.Decision)
	}
	if len(result.Labels) == 0 || result.Labels[0] != "20006" {
		t.Fatalf("labels = %#v, want label 20006", result.Labels)
	}
	if len(result.TraceIDs) != 1 || result.TraceIDs[0] != "trace-text" {
		t.Fatalf("traceIDs = %#v, want trace-text", result.TraceIDs)
	}
	assertWechatAuditEvents(t, observer.Events(), []expectedWechatAuditEvent{
		{operation: externalcall.OperationAccessToken, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentTextCheck, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
	})
}

func TestWechatAuditorObservesFullAuditAndSkipsCachedTokenEvent(t *testing.T) {
	observer := &recordingWechatAuditObserver{}
	mediaCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeJSON(t, w, map[string]any{"access_token": "token-1", "expires_in": 7200})
		case "/msg":
			writeJSON(t, w, map[string]any{
				"errcode":  0,
				"trace_id": "trace-text",
				"result":   map[string]any{"suggest": "pass", "label": 100},
			})
		case "/media":
			mediaCalls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode media body: %v", err)
			}
			if body["media_type"] != float64(2) || (body["media_url"] != "https://cdn.example.com/a.png" && body["media_url"] != "https://img.example.test/b.png") {
				t.Fatalf("media body = %#v, want image media payload", body)
			}
			writeJSON(t, w, map[string]any{"errcode": 0, "trace_id": fmt.Sprintf("trace-media-%d", mediaCalls)})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	auditor := NewWechatAuditor(
		config.WechatConfig{AppID: "wx-app", AppSecret: "secret"},
		config.ContentAuditConfig{Enabled: true, TextScene: 3, MediaEnabled: true, MediaScene: 3, RequestTimeout: time.Second, MaxTextChars: 2500},
		"https://cdn.example.com",
		server.Client(),
		observer,
	).WithURLs(server.URL+"/token", server.URL+"/msg", server.URL+"/media")

	input := resourcelogic.ContentAuditInput{
		OpenID:      "openid-1",
		MerchantID:  "merchant-1",
		TypeCode:    "inventory",
		Title:       "女童春款卫衣库存整包清",
		Description: "整包优先，可现场看货。",
		Images:      []string{"/a.png", "https://img.example.test/b.png"},
	}
	result, err := auditor.AuditResource(context.Background(), input)
	if err != nil {
		t.Fatalf("AuditResource() error = %v", err)
	}
	if result.Decision != resourcelogic.ContentAuditDecisionPass {
		t.Fatalf("decision = %q, want pass", result.Decision)
	}
	if len(result.TraceIDs) != 3 || result.TraceIDs[0] != "trace-text" || result.TraceIDs[1] != "trace-media-1" || result.TraceIDs[2] != "trace-media-2" {
		t.Fatalf("traceIDs = %#v, want text and media traces", result.TraceIDs)
	}
	if len(result.MediaTasks) != 2 || result.MediaTasks[0].TraceID != "trace-media-1" || result.MediaTasks[0].MediaURL != "https://cdn.example.com/a.png" || result.MediaTasks[1].TraceID != "trace-media-2" || result.MediaTasks[1].MediaURL != "https://img.example.test/b.png" {
		t.Fatalf("mediaTasks = %#v, want persisted image audit task", result.MediaTasks)
	}
	assertWechatAuditEvents(t, observer.Events(), []expectedWechatAuditEvent{
		{operation: externalcall.OperationAccessToken, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentTextCheck, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentMediaSubmit, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentMediaSubmit, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
	})

	if _, err := auditor.AuditResource(context.Background(), input); err != nil {
		t.Fatalf("second AuditResource() error = %v", err)
	}
	assertWechatAuditEvents(t, observer.Events(), []expectedWechatAuditEvent{
		{operation: externalcall.OperationAccessToken, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentTextCheck, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentMediaSubmit, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentMediaSubmit, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentTextCheck, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentMediaSubmit, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
		{operation: externalcall.OperationContentMediaSubmit, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
	})
}

func TestWechatAuditorObservesAccessTokenTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		response    *http.Response
		transport   error
		wantOutcome externalcall.Outcome
		wantStatus  int
	}{
		{name: "provider rejected", response: wechatAuditHTTPResponse(http.StatusOK, `{"errcode":40013,"errmsg":"invalid appid"}`), wantOutcome: externalcall.OutcomeProviderRejected, wantStatus: http.StatusOK},
		{name: "http error", response: wechatAuditHTTPResponse(http.StatusServiceUnavailable, `{"errcode":40013}`), wantOutcome: externalcall.OutcomeHTTPError, wantStatus: http.StatusServiceUnavailable},
		{name: "invalid json", response: wechatAuditHTTPResponse(http.StatusOK, `{`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "missing access token", response: wechatAuditHTTPResponse(http.StatusOK, `{"expires_in":7200}`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "timeout", transport: context.DeadlineExceeded, wantOutcome: externalcall.OutcomeTimeout},
		{
			name: "body read failure preserves status",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatAuditErrorReadCloser{err: fmt.Errorf("stream failed: %w", io.ErrUnexpectedEOF)},
			},
			wantOutcome: externalcall.OutcomeTransportError,
			wantStatus:  http.StatusPartialContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingWechatAuditObserver{}
			auditor := NewWechatAuditor(
				config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
				config.ContentAuditConfig{Enabled: true, TextScene: 3, RequestTimeout: time.Second},
				"",
				&http.Client{Transport: wechatAuditRoundTripFunc(func(*http.Request) (*http.Response, error) {
					return tt.response, tt.transport
				})},
				observer,
			).WithURLs("https://wechat.example.test/token", "https://wechat.example.test/msg", "")

			_, err := auditor.AuditResource(context.Background(), wechatAuditInput())
			if err == nil {
				t.Fatal("AuditResource() error = nil, want access token failure")
			}
			assertWechatAuditEvents(t, observer.Events(), []expectedWechatAuditEvent{
				{operation: externalcall.OperationAccessToken, outcome: tt.wantOutcome, statusCode: tt.wantStatus},
			})
		})
	}
}

func TestWechatAuditorObservesTextCheckTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		response    *http.Response
		transport   error
		wantOutcome externalcall.Outcome
		wantStatus  int
	}{
		{name: "provider rejected", response: wechatAuditHTTPResponse(http.StatusOK, `{"errcode":87014,"errmsg":"content risky"}`), wantOutcome: externalcall.OutcomeProviderRejected, wantStatus: http.StatusOK},
		{name: "http error", response: wechatAuditHTTPResponse(http.StatusBadGateway, `{"errcode":87014}`), wantOutcome: externalcall.OutcomeHTTPError, wantStatus: http.StatusBadGateway},
		{name: "invalid json", response: wechatAuditHTTPResponse(http.StatusOK, `{`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "missing trace id", response: wechatAuditHTTPResponse(http.StatusOK, `{"errcode":0,"result":{"suggest":"pass","label":100}}`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "timeout", transport: context.DeadlineExceeded, wantOutcome: externalcall.OutcomeTimeout},
		{
			name: "body read failure preserves status",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatAuditErrorReadCloser{err: fmt.Errorf("stream failed: %w", io.ErrUnexpectedEOF)},
			},
			wantOutcome: externalcall.OutcomeTransportError,
			wantStatus:  http.StatusPartialContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingWechatAuditObserver{}
			auditor := newWechatAuditorForTest(observer, false, func(request *http.Request) (*http.Response, error) {
				if request.URL.Path == "/token" {
					return wechatAuditHTTPResponse(http.StatusOK, `{"access_token":"access-token","expires_in":7200}`), nil
				}
				return tt.response, tt.transport
			})

			_, err := auditor.AuditResource(context.Background(), wechatAuditInput())
			if err == nil {
				t.Fatal("AuditResource() error = nil, want text check failure")
			}
			assertWechatAuditEvents(t, observer.Events(), []expectedWechatAuditEvent{
				{operation: externalcall.OperationAccessToken, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
				{operation: externalcall.OperationContentTextCheck, outcome: tt.wantOutcome, statusCode: tt.wantStatus},
			})
		})
	}
}

func TestWechatAuditorObservesMediaSubmitTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		response    *http.Response
		transport   error
		wantOutcome externalcall.Outcome
		wantStatus  int
	}{
		{name: "provider rejected", response: wechatAuditHTTPResponse(http.StatusOK, `{"errcode":87014,"errmsg":"content risky"}`), wantOutcome: externalcall.OutcomeProviderRejected, wantStatus: http.StatusOK},
		{name: "http error", response: wechatAuditHTTPResponse(http.StatusBadGateway, `{"errcode":87014}`), wantOutcome: externalcall.OutcomeHTTPError, wantStatus: http.StatusBadGateway},
		{name: "invalid json", response: wechatAuditHTTPResponse(http.StatusOK, `{`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "missing trace id", response: wechatAuditHTTPResponse(http.StatusOK, `{"errcode":0}`), wantOutcome: externalcall.OutcomeDecodeError, wantStatus: http.StatusOK},
		{name: "timeout", transport: context.DeadlineExceeded, wantOutcome: externalcall.OutcomeTimeout},
		{
			name: "body read failure preserves status",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatAuditErrorReadCloser{err: fmt.Errorf("stream failed: %w", io.ErrUnexpectedEOF)},
			},
			wantOutcome: externalcall.OutcomeTransportError,
			wantStatus:  http.StatusPartialContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingWechatAuditObserver{}
			auditor := newWechatAuditorForTest(observer, true, func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/token":
					return wechatAuditHTTPResponse(http.StatusOK, `{"access_token":"access-token","expires_in":7200}`), nil
				case "/msg":
					return wechatAuditHTTPResponse(http.StatusOK, `{"errcode":0,"trace_id":"trace-text","result":{"suggest":"pass","label":100}}`), nil
				default:
					return tt.response, tt.transport
				}
			})

			input := wechatAuditInput()
			input.Images = []string{"/image.png"}
			_, err := auditor.AuditResource(context.Background(), input)
			if err == nil {
				t.Fatal("AuditResource() error = nil, want media submit failure")
			}
			assertWechatAuditEvents(t, observer.Events(), []expectedWechatAuditEvent{
				{operation: externalcall.OperationAccessToken, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
				{operation: externalcall.OperationContentTextCheck, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
				{operation: externalcall.OperationContentMediaSubmit, outcome: tt.wantOutcome, statusCode: tt.wantStatus},
			})
		})
	}
}

func TestWechatAuditorTreatsAllTextDecisionsAsSuccessfulCalls(t *testing.T) {
	for _, decision := range []string{
		resourcelogic.ContentAuditDecisionPass,
		resourcelogic.ContentAuditDecisionReview,
		resourcelogic.ContentAuditDecisionRisky,
	} {
		t.Run(decision, func(t *testing.T) {
			observer := &recordingWechatAuditObserver{}
			auditor := newWechatAuditorForTest(observer, false, func(request *http.Request) (*http.Response, error) {
				if request.URL.Path == "/token" {
					return wechatAuditHTTPResponse(http.StatusOK, `{"access_token":"access-token","expires_in":7200}`), nil
				}
				return wechatAuditHTTPResponse(http.StatusOK, fmt.Sprintf(`{"errcode":0,"trace_id":"trace-text","result":{"suggest":%q,"label":20006}}`, decision)), nil
			})

			result, err := auditor.AuditResource(context.Background(), wechatAuditInput())
			if err != nil {
				t.Fatalf("AuditResource() error = %v", err)
			}
			if result.Decision != decision || len(result.Labels) != 1 || result.Labels[0] != "20006" || len(result.TraceIDs) != 1 || result.TraceIDs[0] != "trace-text" {
				t.Fatalf("result = %+v, want decision/label/trace preserved", result)
			}
			assertWechatAuditEvents(t, observer.Events(), []expectedWechatAuditEvent{
				{operation: externalcall.OperationAccessToken, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
				{operation: externalcall.OperationContentTextCheck, outcome: externalcall.OutcomeSuccess, statusCode: http.StatusOK},
			})
		})
	}
}

func TestWechatAuditorUsesOnlyFirstOptionalObserver(t *testing.T) {
	first := &recordingWechatAuditObserver{}
	second := &recordingWechatAuditObserver{}
	auditor := NewWechatAuditor(
		config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		config.ContentAuditConfig{Enabled: true, TextScene: 3, RequestTimeout: time.Second},
		"",
		&http.Client{Transport: wechatAuditRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Path == "/token" {
				return wechatAuditHTTPResponse(http.StatusOK, `{"access_token":"access-token","expires_in":7200}`), nil
			}
			return wechatAuditHTTPResponse(http.StatusOK, `{"errcode":0,"trace_id":"trace-text","result":{"suggest":"pass"}}`), nil
		})},
		first,
		second,
	).WithURLs("https://wechat.example.test/token", "https://wechat.example.test/msg", "")

	if _, err := auditor.AuditResource(context.Background(), wechatAuditInput()); err != nil {
		t.Fatalf("AuditResource() error = %v", err)
	}
	if len(first.Events()) != 2 || len(second.Events()) != 0 {
		t.Fatalf("first events = %+v, second events = %+v, want only first observer used", first.Events(), second.Events())
	}
}

func TestWechatAuditorFailureLogsDoNotLeakSecrets(t *testing.T) {
	const (
		appID           = "sentinel-audit-appid"
		appSecret       = "sentinel-audit-secret"
		openID          = "sentinel-audit-openid"
		accessToken     = "sentinel-audit-access-token"
		mediaURL        = "https://media.example.test/sentinel-image.png"
		requestContent  = "sentinel-audit-request-body"
		providerMessage = "sentinel-audit-provider-message"
		transportRaw    = "sentinel-audit-transport-error"
		bodyReadRaw     = "sentinel-audit-body-read-error"
	)
	tests := []struct {
		name      string
		failureAt string
		response  *http.Response
		transport error
		wantCause string
	}{
		{
			name:      "transport url error",
			failureAt: "/media",
			transport: &url.Error{Op: http.MethodPost, URL: "https://wechat.example.test/media?access_token=" + accessToken, Err: fmt.Errorf("%s: %w", transportRaw, context.DeadlineExceeded)},
			wantCause: "cause=timeout",
		},
		{
			name:      "body read error",
			failureAt: "/token",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       wechatAuditErrorReadCloser{err: fmt.Errorf("%s: %w", bodyReadRaw, io.ErrUnexpectedEOF)},
			},
			wantCause: "cause=unexpected_eof",
		},
		{
			name:      "provider message",
			failureAt: "/msg",
			response:  wechatAuditHTTPResponse(http.StatusOK, `{"errcode":87014,"errmsg":"`+providerMessage+`"}`),
			wantCause: "cause=provider_rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := logtest.NewCollector(t)
			observer := &recordingWechatAuditObserver{}
			auditor := NewWechatAuditor(
				config.WechatConfig{AppID: appID, AppSecret: appSecret},
				config.ContentAuditConfig{Enabled: true, TextScene: 3, MediaEnabled: true, MediaScene: 3, RequestTimeout: time.Second},
				"",
				&http.Client{Transport: wechatAuditRoundTripFunc(func(request *http.Request) (*http.Response, error) {
					if request.URL.Path == tt.failureAt {
						return tt.response, tt.transport
					}
					switch request.URL.Path {
					case "/token":
						return wechatAuditHTTPResponse(http.StatusOK, `{"access_token":"`+accessToken+`","expires_in":7200}`), nil
					case "/msg":
						return wechatAuditHTTPResponse(http.StatusOK, `{"errcode":0,"trace_id":"trace-text","result":{"suggest":"pass"}}`), nil
					default:
						return wechatAuditHTTPResponse(http.StatusOK, `{"errcode":0,"trace_id":"trace-media"}`), nil
					}
				})},
				observer,
			).WithURLs("https://wechat.example.test/token", "https://wechat.example.test/msg", "https://wechat.example.test/media")

			_, _ = auditor.AuditResource(context.Background(), resourcelogic.ContentAuditInput{
				OpenID:      openID,
				MerchantID:  "merchant-safe-id",
				ResourceID:  "resource-safe-id",
				TypeCode:    "inventory",
				Title:       requestContent,
				Description: requestContent,
				Images:      []string{mediaURL},
			})

			logText := collector.String()
			if !strings.Contains(logText, tt.wantCause) {
				t.Fatalf("log = %q, want %q", logText, tt.wantCause)
			}
			if tt.name == "body read error" && !strings.Contains(logText, "status=206") {
				t.Fatalf("log = %q, want preserved response status", logText)
			}
			for _, forbidden := range []string{
				appID, appSecret, openID, accessToken, mediaURL, requestContent,
				providerMessage, transportRaw, bodyReadRaw,
				"appid=", "secret=", "access_token=", "https://wechat.example.test/",
			} {
				if strings.Contains(logText, forbidden) {
					t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
				}
			}
		})
	}
}

type recordingWechatAuditObserver struct {
	mu     sync.Mutex
	events []externalcall.Event
}

func (o *recordingWechatAuditObserver) Observe(_ context.Context, event externalcall.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}

func (o *recordingWechatAuditObserver) Events() []externalcall.Event {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]externalcall.Event(nil), o.events...)
}

type expectedWechatAuditEvent struct {
	operation  string
	outcome    externalcall.Outcome
	statusCode int
}

func assertWechatAuditEvents(t *testing.T, got []externalcall.Event, want []expectedWechatAuditEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("events = %+v, want exactly %d events", got, len(want))
	}
	for index, expected := range want {
		event := got[index]
		if event.Provider != externalcall.ProviderWechat || event.Operation != expected.operation || event.Outcome != expected.outcome || event.StatusCode != expected.statusCode {
			t.Fatalf("event[%d] = %+v, want wechat/%s/%s/%d", index, event, expected.operation, expected.outcome, expected.statusCode)
		}
	}
}

func newWechatAuditorForTest(observer externalcall.Observer, mediaEnabled bool, transport wechatAuditRoundTripFunc) *WechatAuditor {
	return NewWechatAuditor(
		config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		config.ContentAuditConfig{Enabled: true, TextScene: 3, MediaEnabled: mediaEnabled, MediaScene: 3, RequestTimeout: time.Second},
		"https://media.example.test",
		&http.Client{Transport: transport},
		observer,
	).WithURLs("https://wechat.example.test/token", "https://wechat.example.test/msg", "https://wechat.example.test/media")
}

func wechatAuditInput() resourcelogic.ContentAuditInput {
	return resourcelogic.ContentAuditInput{
		OpenID:      "openid-1",
		MerchantID:  "merchant-1",
		ResourceID:  "resource-1",
		TypeCode:    "inventory",
		Title:       "女童春款卫衣库存整包清",
		Description: "整包优先，可现场看货。",
	}
}

type wechatAuditRoundTripFunc func(*http.Request) (*http.Response, error)

func (f wechatAuditRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func wechatAuditHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type wechatAuditErrorReadCloser struct {
	err error
}

func (r wechatAuditErrorReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (wechatAuditErrorReadCloser) Close() error {
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
