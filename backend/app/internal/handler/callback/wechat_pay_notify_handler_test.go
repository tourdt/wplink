package callback

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	paymentlogic "wplink/backend/app/internal/logic/payment"
	"wplink/backend/app/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestWechatPayNotifyHTTPHandlerPreservesBoundaryBodyAndFirstHeaders(t *testing.T) {
	body := bytes.Repeat([]byte("x"), int(wechatPayNotifyBodyLimit))
	store := &fakeWechatPayNotifyStore{}
	gateway := &fakeWechatPayNotifyGateway{notification: successfulWechatPayNotification()}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat-pay/notify", bytes.NewReader(body))
	req.Header["Wechatpay-Signature"] = []string{"first-signature", "second-signature"}
	req.Header["Wechatpay-Serial"] = []string{"first-serial", "second-serial"}
	req.Header["X-Custom"] = []string{"custom-first", "custom-second"}
	rec := httptest.NewRecorder()

	wechatPayNotifyHTTPHandler(store, gateway)(rec, req)

	assertWechatPayNotifyResponse(t, rec, http.StatusOK, `{"code":"SUCCESS","message":"成功"}`)
	if !bytes.Equal(gateway.req.Body, body) {
		t.Fatalf("gateway body changed: got=%d bytes want=%d", len(gateway.req.Body), len(body))
	}
	wantHeaders := map[string]string{
		"Wechatpay-Signature": "first-signature",
		"Wechatpay-Serial":    "first-serial",
		"X-Custom":            "custom-first",
	}
	for key, want := range wantHeaders {
		if gateway.req.Headers[key] != want {
			t.Fatalf("headers[%q]=%q, want first value %q; all=%#v", key, gateway.req.Headers[key], want, gateway.req.Headers)
		}
	}
	if len(gateway.req.Headers) != len(wantHeaders) {
		t.Fatalf("headers=%#v, want all and only request headers", gateway.req.Headers)
	}
}

func TestWechatPayNotifyHTTPHandlerRejectsBodyOverOneMiBBeforeGateway(t *testing.T) {
	gateway := &fakeWechatPayNotifyGateway{notification: successfulWechatPayNotification()}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat-pay/notify", bytes.NewReader(bytes.Repeat([]byte("x"), int(wechatPayNotifyBodyLimit)+1)))

	wechatPayNotifyHTTPHandler(&fakeWechatPayNotifyStore{}, gateway)(rec, req)

	assertWechatPayNotifyResponse(t, rec, http.StatusInternalServerError, `{"code":"FAIL","message":"处理失败"}`)
	if gateway.calls != 0 {
		t.Fatalf("gateway calls=%d, oversized body must stop before business processing", gateway.calls)
	}
}

func TestWechatPayNotifyHTTPHandlerUsesFixedFailureAndDoesNotLeakDependencyErrors(t *testing.T) {
	tests := []struct {
		name    string
		store   *fakeWechatPayNotifyStore
		gateway *fakeWechatPayNotifyGateway
	}{
		{
			name:    "bad signature gateway error",
			store:   &fakeWechatPayNotifyStore{},
			gateway: &fakeWechatPayNotifyGateway{err: errors.New("raw bad signature serial=secret-serial api-v3-key=secret-key")},
		},
		{
			name:    "store error",
			store:   &fakeWechatPayNotifyStore{err: errors.New("raw database password=secret-db-password")},
			gateway: &fakeWechatPayNotifyGateway{notification: successfulWechatPayNotification()},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			previousWriter := logx.Reset()
			logx.SetWriter(logx.NewWriter(&logs))
			t.Cleanup(func() {
				if currentWriter := logx.Reset(); currentWriter != nil {
					_ = currentWriter.Close()
				}
				if previousWriter != nil {
					logx.SetWriter(previousWriter)
				}
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat-pay/notify", strings.NewReader(`{"raw":"secret-body"}`))
			req.Header.Set("Wechatpay-Signature", "secret-signature")
			req.Header.Set("Wechatpay-Serial", "secret-serial")
			wechatPayNotifyHTTPHandler(tc.store, tc.gateway)(rec, req)

			assertWechatPayNotifyResponse(t, rec, http.StatusInternalServerError, `{"code":"FAIL","message":"处理失败"}`)
			combined := rec.Body.String() + logs.String()
			for _, forbidden := range []string{"secret-body", "secret-signature", "secret-serial", "secret-key", "secret-db-password", "raw bad signature", "raw database"} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("response/logs leaked %q: %s", forbidden, combined)
				}
			}
		})
	}
}

func TestWechatPayNotifyHTTPHandlerRejectsNilAndTypedNilDependencies(t *testing.T) {
	var typedNilStore *fakeWechatPayNotifyStore
	var typedNilGateway *fakeWechatPayNotifyGateway
	tests := []struct {
		name    string
		store   paymentlogic.UnifiedWechatPayNotifyStore
		gateway paymentlogic.WechatPayGateway
	}{
		{name: "nil store", store: nil, gateway: &fakeWechatPayNotifyGateway{}},
		{name: "typed nil store", store: typedNilStore, gateway: &fakeWechatPayNotifyGateway{}},
		{name: "nil gateway", store: &fakeWechatPayNotifyStore{}, gateway: nil},
		{name: "typed nil gateway", store: &fakeWechatPayNotifyStore{}, gateway: typedNilGateway},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat-pay/notify", strings.NewReader(`{}`))
			wechatPayNotifyHTTPHandler(tc.store, tc.gateway)(rec, req)
			assertWechatPayNotifyResponse(t, rec, http.StatusInternalServerError, `{"code":"FAIL","message":"处理失败"}`)
		})
	}
}

func assertWechatPayNotifyResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantBody string) {
	t.Helper()
	if rec.Code != wantStatus || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" || rec.Body.String() != wantBody {
		t.Fatalf("status=%d contentType=%q body=%q, want status=%d provider JSON=%q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String(), wantStatus, wantBody)
	}
}

func successfulWechatPayNotification() paymentlogic.WechatPayNotification {
	return paymentlogic.WechatPayNotification{
		OutTradeNo:    "VIP202608060001",
		TransactionID: "wx-transaction-1",
		Attach:        "vip:order-1",
		AmountTotal:   1990,
		SuccessTime:   "2026-08-06T09:00:00+08:00",
		RawPayload:    map[string]interface{}{"trade_state": "SUCCESS"},
	}
}

type fakeWechatPayNotifyStore struct {
	err   error
	input model.MarkVIPOrderPaidInput
}

func (*fakeWechatPayNotifyStore) MarkContactUnlockOrderPaid(context.Context, model.MarkContactUnlockOrderPaidInput) (model.ContactUnlockPaymentResult, error) {
	return model.ContactUnlockPaymentResult{}, errors.New("unexpected contact unlock notification")
}

func (s *fakeWechatPayNotifyStore) MarkVIPOrderPaid(_ context.Context, input model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error) {
	s.input = input
	if s.err != nil {
		return model.VIPPaymentResult{}, s.err
	}
	return model.VIPPaymentResult{OrderID: "order-1", MerchantID: "merchant-1", Status: model.PaymentOrderStatusPaid}, nil
}

type fakeWechatPayNotifyGateway struct {
	req          paymentlogic.WechatPayNotifyReq
	notification paymentlogic.WechatPayNotification
	err          error
	calls        int
}

func (*fakeWechatPayNotifyGateway) CreatePrepay(context.Context, paymentlogic.WechatPrepayInput) (paymentlogic.WechatPayParams, error) {
	return paymentlogic.WechatPayParams{}, errors.New("unused")
}

func (g *fakeWechatPayNotifyGateway) DecodeNotify(_ context.Context, req paymentlogic.WechatPayNotifyReq) (paymentlogic.WechatPayNotification, error) {
	g.calls++
	g.req = req
	return g.notification, g.err
}
