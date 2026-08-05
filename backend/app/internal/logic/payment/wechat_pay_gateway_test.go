package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/externalcall"

	"github.com/zeromicro/go-zero/core/logx/logtest"
)

func TestHTTPWechatPayGatewayQueriesAndClosesOrder(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	var queried bool
	var closed bool
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v3/pay/transactions/out-trade-no/"):
			queried = true
			if r.URL.Query().Get("mchid") != "merchant-1" {
				t.Errorf("mchid = %q, want merchant-1", r.URL.Query().Get("mchid"))
			}
			return signedWechatHTTPResponse(t, privateKey, http.StatusOK, `{"out_trade_no":"VIP202607250001","transaction_id":"wx-1","attach":"vip:order-1","trade_state":"SUCCESS","success_time":"2026-07-25T12:00:00Z","amount":{"total":1990}}`), nil
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/close"):
			closed = true
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				return nil, readErr
			}
			var decoded map[string]string
			if err := json.Unmarshal(body, &decoded); err != nil || decoded["mchid"] != "merchant-1" {
				t.Errorf("close body = %s err=%v, want merchant id", body, err)
			}
			return signedWechatHTTPResponse(t, privateKey, http.StatusNoContent, ""), nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"code":"NOT_FOUND"}`)),
			}, nil
		}
	})}

	observer := &recordingWechatPayObserver{}
	gateway := &HTTPWechatPayGateway{
		cfg:        config.WechatPayConfig{MchID: "merchant-1", MerchantSerialNo: "serial-1"},
		httpClient: client,
		apiBaseURL: "https://wechat.test",
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		observer:   observer,
	}

	order, err := gateway.QueryOrder(context.Background(), "VIP202607250001")
	if err != nil {
		t.Fatalf("QueryOrder() error = %v", err)
	}
	if order.TradeState != "SUCCESS" || order.Attach != "vip:order-1" || order.AmountTotal != 1990 {
		t.Fatalf("order = %#v, want paid order", order)
	}
	if err := gateway.CloseOrder(context.Background(), "VIP202607250001"); err != nil {
		t.Fatalf("CloseOrder() error = %v", err)
	}
	if !queried || !closed {
		t.Fatalf("queried=%t closed=%t, want both requests", queried, closed)
	}
	assertWechatPayEvents(t, observer.events, []externalcall.Event{
		{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayQuery, Outcome: externalcall.OutcomeSuccess, StatusCode: http.StatusOK},
		{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayClose, Outcome: externalcall.OutcomeSuccess, StatusCode: http.StatusNoContent},
	})
}

func TestHTTPWechatPayGatewayObservesCreatePrepaySuccess(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	observer := &recordingWechatPayObserver{}
	gateway := &HTTPWechatPayGateway{
		cfg: config.WechatPayConfig{
			AppID:            "wx-app-1",
			MchID:            "merchant-1",
			MerchantSerialNo: "serial-1",
			NotifyURL:        "https://merchant.test/pay/notify",
		},
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return signedWechatHTTPResponse(t, privateKey, http.StatusOK, `{"prepay_id":"wx-prepay-1"}`), nil
		})},
		apiBaseURL: "https://wechat.test",
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		observer:   observer,
	}

	params, err := gateway.CreatePrepay(context.Background(), WechatPrepayInput{
		OutTradeNo:  "VIP202607250001",
		Description: "会员服务",
		OpenID:      "openid-1",
		AmountTotal: 1990,
	})
	if err != nil {
		t.Fatalf("CreatePrepay() error = %v", err)
	}
	if params.Package != "prepay_id=wx-prepay-1" {
		t.Fatalf("CreatePrepay() package = %q, want prepay_id=wx-prepay-1", params.Package)
	}
	assertWechatPayEvents(t, observer.events, []externalcall.Event{
		{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayCreate, Outcome: externalcall.OutcomeSuccess, StatusCode: http.StatusOK},
	})
}

func TestHTTPWechatPayGatewayObservesQueryTimeout(t *testing.T) {
	gateway, observer := newObservedWechatPayGateway(t, func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})

	if _, err := gateway.QueryOrder(context.Background(), "VIP202607250001"); err == nil {
		t.Fatal("QueryOrder() error = nil, want timeout")
	}
	assertWechatPayEvents(t, observer.events, []externalcall.Event{
		{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayQuery, Outcome: externalcall.OutcomeTimeout},
	})
}

func TestHTTPWechatPayGatewayObservesCloseHTTPError(t *testing.T) {
	gateway, observer := newObservedWechatPayGateway(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"code":"SYSTEM_ERROR","message":"temporary"}`)),
		}, nil
	})

	if err := gateway.CloseOrder(context.Background(), "VIP202607250001"); err == nil {
		t.Fatal("CloseOrder() error = nil, want HTTP error")
	}
	assertWechatPayEvents(t, observer.events, []externalcall.Event{
		{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayClose, Outcome: externalcall.OutcomeHTTPError, StatusCode: http.StatusInternalServerError},
	})
}

func TestHTTPWechatPayGatewayRejectsUnsignedQueryResponse(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"out_trade_no":"VIP202607250001","trade_state":"NOTPAY","amount":{"total":1990}}`)),
		}, nil
	})}

	observer := &recordingWechatPayObserver{}
	gateway := &HTTPWechatPayGateway{
		cfg:        config.WechatPayConfig{MchID: "merchant-1", MerchantSerialNo: "serial-1"},
		httpClient: client,
		apiBaseURL: "https://wechat.test",
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		observer:   observer,
	}
	if _, err := gateway.QueryOrder(context.Background(), "VIP202607250001"); err == nil {
		t.Fatal("QueryOrder() error = nil, want response signature validation failure")
	}
	assertWechatPayEvents(t, observer.events, []externalcall.Event{
		{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayQuery, Outcome: externalcall.OutcomeDecodeError, StatusCode: http.StatusOK},
	})
}

func TestHTTPWechatPayGatewayObservesBodyAndPayloadFailures(t *testing.T) {
	tests := []struct {
		name       string
		response   func(*testing.T, *rsa.PrivateKey) *http.Response
		want       externalcall.Outcome
		statusCode int
	}{
		{
			name: "body read failure",
			response: func(_ *testing.T, _ *rsa.PrivateKey) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: wechatPayErrorReadCloser{err: io.ErrUnexpectedEOF}}
			},
			want:       externalcall.OutcomeTransportError,
			statusCode: http.StatusOK,
		},
		{
			name: "invalid json",
			response: func(t *testing.T, privateKey *rsa.PrivateKey) *http.Response {
				return signedWechatHTTPResponse(t, privateKey, http.StatusOK, `{`)
			},
			want:       externalcall.OutcomeDecodeError,
			statusCode: http.StatusOK,
		},
		{
			name: "missing required fields",
			response: func(t *testing.T, privateKey *rsa.PrivateKey) *http.Response {
				return signedWechatHTTPResponse(t, privateKey, http.StatusOK, `{"amount":{"total":1990}}`)
			},
			want:       externalcall.OutcomeDecodeError,
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response *http.Response
			gateway, observer := newObservedWechatPayGateway(t, func(*http.Request) (*http.Response, error) {
				return response, nil
			})
			response = tt.response(t, gateway.privateKey)

			if _, err := gateway.QueryOrder(context.Background(), "VIP202607250001"); err == nil {
				t.Fatal("QueryOrder() error = nil, want response failure")
			}
			assertWechatPayEvents(t, observer.events, []externalcall.Event{
				{Provider: externalcall.ProviderWechat, Operation: externalcall.OperationPayQuery, Outcome: tt.want, StatusCode: tt.statusCode},
			})
		})
	}
}

func TestHTTPWechatPayGatewayDoesNotObserveNotifyDecode(t *testing.T) {
	gateway, observer := newObservedWechatPayGateway(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected outbound HTTP request")
		return nil, nil
	})

	if _, err := gateway.DecodeNotify(context.Background(), WechatPayNotifyReq{}); err == nil {
		t.Fatal("DecodeNotify() error = nil, want invalid notification")
	}
	if len(observer.events) != 0 {
		t.Fatalf("events = %+v, want none for inbound notification decode", observer.events)
	}
}

func TestHTTPWechatPayGatewayTransportErrorsAreSafe(t *testing.T) {
	const (
		merchantID    = "sentinel-merchant-id"
		appID         = "sentinel-app-id"
		openID        = "sentinel-open-id"
		serialNo      = "sentinel-certificate-serial"
		authorization = "sentinel-authorization"
		signature     = "sentinel-signature"
		rawError      = "sentinel-raw-transport-error"
		fullURL       = "https://wechat.test/v3/pay/transactions/out-trade-no/sensitive-order?mchid=" + merchantID
	)

	tests := []struct {
		name          string
		operation     string
		call          func(*HTTPWechatPayGateway) error
		preservesTime bool
	}{
		{
			name:      "create",
			operation: externalcall.OperationPayCreate,
			call: func(gateway *HTTPWechatPayGateway) error {
				_, err := gateway.CreatePrepay(context.Background(), WechatPrepayInput{OutTradeNo: "sensitive-order", Description: "sensitive-body", OpenID: openID, AmountTotal: 1})
				return err
			},
		},
		{
			name:          "query",
			operation:     externalcall.OperationPayQuery,
			preservesTime: true,
			call: func(gateway *HTTPWechatPayGateway) error {
				_, err := gateway.QueryOrder(context.Background(), "sensitive-order")
				return err
			},
		},
		{
			name:          "close",
			operation:     externalcall.OperationPayClose,
			preservesTime: true,
			call: func(gateway *HTTPWechatPayGateway) error {
				return gateway.CloseOrder(context.Background(), "sensitive-order")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := logtest.NewCollector(t)
			gateway, observer := newObservedWechatPayGateway(t, func(request *http.Request) (*http.Response, error) {
				return nil, &url.Error{
					Op:  request.Method,
					URL: fullURL,
					Err: fmt.Errorf("%s %s %s: %w", rawError, authorization, signature, context.DeadlineExceeded),
				}
			})
			gateway.cfg.MchID = merchantID
			gateway.cfg.AppID = appID
			gateway.cfg.MerchantSerialNo = serialNo

			returnedErr := tt.call(gateway)
			if returnedErr == nil {
				t.Fatal("error = nil, want transport failure")
			}
			if tt.preservesTime && !errors.Is(returnedErr, context.DeadlineExceeded) {
				t.Fatalf("errors.Is(context.DeadlineExceeded) = false, error = %v", returnedErr)
			}
			var urlErr *url.Error
			if errors.As(returnedErr, &urlErr) {
				t.Fatalf("returned error exposes *url.Error with URL %q", urlErr.URL)
			}
			assertWechatPayEvents(t, observer.events, []externalcall.Event{
				{Provider: externalcall.ProviderWechat, Operation: tt.operation, Outcome: externalcall.OutcomeTimeout},
			})

			logText := collector.String()
			for _, forbidden := range []string{
				merchantID, appID, openID, serialNo, authorization, signature, rawError,
				fullURL, "mchid=", "appid=", "openid", "Authorization",
			} {
				if strings.Contains(logText, forbidden) {
					t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
				}
				if strings.Contains(returnedErr.Error(), forbidden) {
					t.Fatalf("returned error contains forbidden value %q: %s", forbidden, returnedErr)
				}
			}
		})
	}
}

type recordingWechatPayObserver struct {
	events []externalcall.Event
}

func (o *recordingWechatPayObserver) Observe(_ context.Context, event externalcall.Event) {
	o.events = append(o.events, event)
}

func assertWechatPayEvents(t *testing.T, got []externalcall.Event, want []externalcall.Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("events = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i].Duration < 0 {
			t.Fatalf("events[%d].Duration = %s, want non-negative", i, got[i].Duration)
		}
		got[i].Duration = 0
		if got[i] != want[i] {
			t.Fatalf("events[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func newObservedWechatPayGateway(t *testing.T, transport roundTripFunc) (*HTTPWechatPayGateway, *recordingWechatPayObserver) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	observer := &recordingWechatPayObserver{}
	return &HTTPWechatPayGateway{
		cfg: config.WechatPayConfig{
			AppID:            "wx-app-1",
			MchID:            "merchant-1",
			MerchantSerialNo: "serial-1",
			NotifyURL:        "https://merchant.test/pay/notify",
		},
		httpClient: &http.Client{Transport: transport},
		apiBaseURL: "https://wechat.test",
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		observer:   observer,
	}, observer
}

type wechatPayErrorReadCloser struct {
	err error
}

func (r wechatPayErrorReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (wechatPayErrorReadCloser) Close() error               { return nil }

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func signedWechatHTTPResponse(t *testing.T, privateKey *rsa.PrivateKey, status int, body string) *http.Response {
	t.Helper()
	timestamp := "1784971200"
	nonce := "response-nonce"
	message := timestamp + "\n" + nonce + "\n" + body + "\n"
	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15() error = %v", err)
	}
	headers := make(http.Header)
	headers.Set("Wechatpay-Timestamp", timestamp)
	headers.Set("Wechatpay-Nonce", nonce)
	headers.Set("Wechatpay-Signature", base64.StdEncoding.EncodeToString(signature))
	headers.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
