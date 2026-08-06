package vip

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/handler/handlerx"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	viplogic "wplink/backend/app/internal/logic/vip"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestVIPPublicHTTPHandlersStayAnonymousAndKeepEmptyArrays(t *testing.T) {
	store := &fakeVIPHandlerStore{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
		target  string
	}{
		{name: "plans", handler: listVIPPlansHTTPHandler(store), target: "/api/v1/vip/plans"},
		{name: "quota packs", handler: listQuotaPacksHTTPHandler(store), target: "/api/v1/vip/quota-packs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, httptest.NewRequest(http.MethodGet, tc.target, nil))
			data := vipHandlerData(t, rec, http.StatusOK)
			items, ok := data["items"].([]interface{})
			if !ok || len(items) != 0 {
				t.Fatalf("data=%#v, want anonymous items: []", data)
			}
		})
	}
}

func TestVIPPrivateHTTPHandlersUseMerchantPermissionAndTokenUser(t *testing.T) {
	store := &fakeVIPHandlerStore{}
	permissionStore := &fakeVIPPermissionStore{allowedMerchantID: "merchant-1"}
	deps := vipTestPermissionDeps(permissionStore)
	gateway := &fakeVIPPaymentGateway{params: paymentlogic.WechatPayParams{TimeStamp: "1", NonceStr: "n", Package: "prepay_id=p", SignType: "RSA", PaySign: "s"}}

	t.Run("merchant vip", func(t *testing.T) {
		rec := httptest.NewRecorder()
		getMerchantVIPHTTPHandler(store, deps)(rec, vipHandlerRequest(http.MethodGet, "/api/v1/merchants/merchant-1/vip", ""))
		data := vipHandlerData(t, rec, http.StatusOK)
		if data["merchantId"] != "merchant-1" {
			t.Fatalf("data=%#v, want path merchant", data)
		}
	})

	t.Run("order ignores client user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		createVIPOrderHTTPHandler(store, deps)(rec, vipHandlerRequest(http.MethodPost, "/api/v1/merchants/merchant-1/vip/orders", `{"userId":"attacker","planCode":"monthly"}`))
		vipHandlerData(t, rec, http.StatusOK)
		if store.createOrderInput.UserID != "user-real" || store.createOrderInput.MerchantID != "merchant-1" || store.createOrderInput.PlanCode != "monthly" {
			t.Fatalf("create order input=%#v, want token user and path merchant", store.createOrderInput)
		}
	})

	t.Run("payment ignores client user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		createVIPPaymentHTTPHandler(store, gateway, deps, config.Config{})(rec, vipHandlerRequest(http.MethodPost, "/api/v1/merchants/merchant-1/vip/orders/order-1/payment", `{"userId":"attacker"}`))
		vipHandlerData(t, rec, http.StatusOK)
		if store.paymentContextInput.UserID != "user-real" || store.paymentContextInput.MerchantID != "merchant-1" || store.paymentContextInput.OrderID != "order-1" {
			t.Fatalf("payment context input=%#v, want token user and path IDs", store.paymentContextInput)
		}
	})
}

func TestVIPHTTPHandlersRejectMissingAndTypedNilDependencies(t *testing.T) {
	var typedNilStore *fakeVIPHandlerStore
	var typedNilGateway *fakeVIPPaymentGateway
	var typedNilPermissionStore *fakeVIPPermissionStore
	var typedNilUserTokenService *fakeVIPUserTokenService
	var typedNilAdminTokenService *fakeVIPAdminTokenService
	typedNilUserDeps := vipTestPermissionDeps(&fakeVIPPermissionStore{})
	typedNilUserDeps.UserTokenService = typedNilUserTokenService
	typedNilAdminDeps := vipTestPermissionDeps(&fakeVIPPermissionStore{})
	typedNilAdminDeps.AdminTokenService = typedNilAdminTokenService
	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		target  string
		body    string
	}{
		{name: "public typed nil store", handler: listVIPPlansHTTPHandler(typedNilStore), method: http.MethodGet, target: "/api/v1/vip/plans"},
		{name: "private typed nil store", handler: createVIPOrderHTTPHandler(typedNilStore, vipTestPermissionDeps(&fakeVIPPermissionStore{})), method: http.MethodPost, target: "/api/v1/merchants/merchant-1/vip/orders", body: `{}`},
		{name: "typed nil permission", handler: getMerchantVIPHTTPHandler(&fakeVIPHandlerStore{}, vipTestPermissionDeps(typedNilPermissionStore)), method: http.MethodGet, target: "/api/v1/merchants/merchant-1/vip"},
		{name: "typed nil user token service", handler: getMerchantVIPHTTPHandler(&fakeVIPHandlerStore{}, typedNilUserDeps), method: http.MethodGet, target: "/api/v1/merchants/merchant-1/vip"},
		{name: "typed nil admin token service", handler: getMerchantVIPHTTPHandler(&fakeVIPHandlerStore{}, typedNilAdminDeps), method: http.MethodGet, target: "/api/v1/merchants/merchant-1/vip"},
		{name: "typed nil gateway", handler: createVIPPaymentHTTPHandler(&fakeVIPHandlerStore{}, typedNilGateway, vipTestPermissionDeps(&fakeVIPPermissionStore{allowedMerchantID: "merchant-1"}), config.Config{}), method: http.MethodPost, target: "/api/v1/merchants/merchant-1/vip/orders/order-1/payment", body: `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, vipHandlerRequest(tc.method, tc.target, tc.body))
			body := vipHandlerEnvelope(t, rec, http.StatusInternalServerError)
			if body["errorCode"] != "INTERNAL_ERROR" {
				t.Fatalf("body=%#v, want safe dependency error", body)
			}
		})
	}
}

func TestCreateVIPPaymentHTTPHandlerConcurrentGatewayNormalization(t *testing.T) {
	const goroutines = 256
	validGateway := &concurrentVIPPaymentGateway{}

	tests := []struct {
		name              string
		gateway           paymentlogic.WechatPayGateway
		config            config.Config
		wantStatus        int
		wantErrorCode     string
		wantContextCalls  int64
		wantCreateCalls   int64
		wantMarkPaidCalls int64
		wantPrepayCalls   int64
		concurrentGateway *concurrentVIPPaymentGateway
	}{
		{
			name:              "nil gateway keeps development mock semantics",
			config:            config.Config{RuntimeMode: "development", WechatPay: config.WechatPayConfig{DevMockEnabled: true}},
			wantStatus:        http.StatusOK,
			wantContextCalls:  goroutines,
			wantCreateCalls:   goroutines,
			wantMarkPaidCalls: goroutines,
		},
		{
			name:          "typed nil gateway remains rejected when mock disabled",
			gateway:       (*concurrentVIPPaymentGateway)(nil),
			wantStatus:    http.StatusInternalServerError,
			wantErrorCode: "INTERNAL_ERROR",
		},
		{
			name:              "valid gateway remains unchanged",
			gateway:           validGateway,
			wantStatus:        http.StatusOK,
			wantContextCalls:  goroutines,
			wantCreateCalls:   goroutines,
			wantPrepayCalls:   goroutines,
			concurrentGateway: validGateway,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &concurrentVIPPaymentStore{}
			permissionStore := &concurrentVIPPermissionStore{}
			handler := createVIPPaymentHTTPHandler(store, tc.gateway, vipTestPermissionDeps(permissionStore), tc.config)
			start := make(chan struct{})
			results := make(chan concurrentVIPPaymentResult, goroutines)
			var group sync.WaitGroup
			group.Add(goroutines)
			for index := 0; index < goroutines; index++ {
				go func() {
					defer group.Done()
					<-start
					rec := httptest.NewRecorder()
					handler(rec, vipHandlerRequest(http.MethodPost, "/api/v1/merchants/merchant-1/vip/orders/order-1/payment", `{}`))
					var body struct {
						ErrorCode string `json:"errorCode"`
					}
					decodeErr := json.Unmarshal(rec.Body.Bytes(), &body)
					results <- concurrentVIPPaymentResult{status: rec.Code, errorCode: body.ErrorCode, decodeErr: decodeErr}
				}()
			}
			close(start)
			group.Wait()
			close(results)

			for result := range results {
				if result.decodeErr != nil || result.status != tc.wantStatus || result.errorCode != tc.wantErrorCode {
					t.Fatalf("result=%+v, want status=%d errorCode=%q", result, tc.wantStatus, tc.wantErrorCode)
				}
			}
			if got := permissionStore.calls.Load(); got != goroutines {
				t.Fatalf("permission calls=%d, want %d", got, goroutines)
			}
			if got := store.contextCalls.Load(); got != tc.wantContextCalls {
				t.Fatalf("payment context calls=%d, want %d", got, tc.wantContextCalls)
			}
			if got := store.createCalls.Load(); got != tc.wantCreateCalls {
				t.Fatalf("payment create calls=%d, want %d", got, tc.wantCreateCalls)
			}
			if got := store.markPaidCalls.Load(); got != tc.wantMarkPaidCalls {
				t.Fatalf("payment mark paid calls=%d, want %d", got, tc.wantMarkPaidCalls)
			}
			if tc.concurrentGateway != nil {
				if got := tc.concurrentGateway.prepayCalls.Load(); got != tc.wantPrepayCalls {
					t.Fatalf("gateway prepay calls=%d, want %d", got, tc.wantPrepayCalls)
				}
			}
		})
	}
}

func vipTestPermissionDeps(store handlerx.MerchantPermissionStore) handlerx.MerchantPermissionDeps {
	return handlerx.MerchantPermissionDeps{UserTokenService: &fakeVIPUserTokenService{}, AdminTokenService: &fakeVIPAdminTokenService{}, Store: store}
}

func vipHandlerRequest(method string, target string, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer user-token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	vars := map[string]string{}
	parts := strings.Split(strings.Trim(target, "/"), "/")
	if len(parts) >= 4 && parts[2] == "merchants" {
		vars["merchantId"] = parts[3]
	}
	if len(parts) >= 8 && parts[4] == "vip" && parts[5] == "orders" {
		vars["orderId"] = parts[6]
	}
	return pathvar.WithVars(req, vars)
}

func vipHandlerData(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) map[string]interface{} {
	t.Helper()
	body := vipHandlerEnvelope(t, rec, wantStatus)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body=%#v, want data object", body)
	}
	return data
}

func vipHandlerEnvelope(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) map[string]interface{} {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status=%d body=%s, want %d", rec.Code, rec.Body.String(), wantStatus)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

type fakeVIPHandlerStore struct {
	createOrderInput    model.CreateVIPOrderInput
	paymentContextInput model.GetVIPPaymentContextInput
}

func (*fakeVIPHandlerStore) ListVIPPlans(context.Context) ([]model.VIPPlan, error) {
	return []model.VIPPlan{}, nil
}

func (*fakeVIPHandlerStore) ListQuotaPacks(context.Context) ([]model.QuotaPack, error) {
	return []model.QuotaPack{}, nil
}

func (*fakeVIPHandlerStore) GetMerchantVIPSummary(_ context.Context, merchantID string) (model.MerchantVIPSummary, error) {
	return model.MerchantVIPSummary{MerchantID: merchantID, Status: model.VIPStatusNone}, nil
}

func (s *fakeVIPHandlerStore) CreateVIPOrder(_ context.Context, input model.CreateVIPOrderInput) (model.VIPOrder, error) {
	s.createOrderInput = input
	return model.VIPOrder{ID: "order-1", MerchantID: input.MerchantID, UserID: input.UserID, PlanCode: input.PlanCode, Status: model.PaymentOrderStatusPending}, nil
}

func (*fakeVIPHandlerStore) CreateQuotaPackOrder(_ context.Context, input model.CreateQuotaPackOrderInput) (model.VIPOrder, error) {
	return model.VIPOrder{ID: "order-2", MerchantID: input.MerchantID, UserID: input.UserID, ProductType: model.VIPProductTypeQuotaPack, ProductCode: input.PackCode, Status: model.PaymentOrderStatusPending}, nil
}

func (s *fakeVIPHandlerStore) GetVIPPaymentContext(_ context.Context, input model.GetVIPPaymentContextInput) (model.VIPPaymentContext, error) {
	s.paymentContextInput = input
	return model.VIPPaymentContext{OrderID: input.OrderID, MerchantID: input.MerchantID, UserID: input.UserID, OpenID: "openid", Status: model.PaymentOrderStatusPending, AmountTotal: 1990, Currency: "CNY", PlanName: "VIP 月卡"}, nil
}

func (*fakeVIPHandlerStore) CreateVIPPaymentOrder(_ context.Context, input model.CreateVIPPaymentOrderInput) (model.VIPPaymentOrder, error) {
	return model.VIPPaymentOrder{ID: input.OrderID, OutTradeNo: "VIP202608060001", AmountTotal: 1990, Currency: "CNY", Status: model.PaymentOrderStatusPending, PlanName: "VIP 月卡"}, nil
}

func (*fakeVIPHandlerStore) MarkVIPOrderPaid(context.Context, model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error) {
	return model.VIPPaymentResult{}, nil
}

var _ viplogic.Store = (*fakeVIPHandlerStore)(nil)
var _ paymentlogic.VIPPaymentStore = (*fakeVIPHandlerStore)(nil)

type fakeVIPPaymentGateway struct {
	params paymentlogic.WechatPayParams
}

func (g *fakeVIPPaymentGateway) CreatePrepay(context.Context, paymentlogic.WechatPrepayInput) (paymentlogic.WechatPayParams, error) {
	return g.params, nil
}

func (*fakeVIPPaymentGateway) DecodeNotify(context.Context, paymentlogic.WechatPayNotifyReq) (paymentlogic.WechatPayNotification, error) {
	return paymentlogic.WechatPayNotification{}, errors.New("unused")
}

type fakeVIPPermissionStore struct {
	allowedMerchantID string
}

func (s *fakeVIPPermissionStore) UserCanManageMerchant(_ context.Context, _ string, merchantID string) (bool, error) {
	return merchantID == s.allowedMerchantID, nil
}

type fakeVIPUserTokenService struct{}

func (*fakeVIPUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (*fakeVIPUserTokenService) ParseUserToken(context.Context, string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{UserID: "user-real"}, nil
}

type fakeVIPAdminTokenService struct{}

func (*fakeVIPAdminTokenService) ParseAdminToken(context.Context, string) (session.AdminTokenSubject, error) {
	return session.AdminTokenSubject{}, errors.New("not admin")
}

type concurrentVIPPaymentResult struct {
	status    int
	errorCode string
	decodeErr error
}

type concurrentVIPPermissionStore struct {
	calls atomic.Int64
}

func (s *concurrentVIPPermissionStore) UserCanManageMerchant(context.Context, string, string) (bool, error) {
	s.calls.Add(1)
	return true, nil
}

type concurrentVIPPaymentStore struct {
	contextCalls  atomic.Int64
	createCalls   atomic.Int64
	markPaidCalls atomic.Int64
}

func (s *concurrentVIPPaymentStore) GetVIPPaymentContext(_ context.Context, input model.GetVIPPaymentContextInput) (model.VIPPaymentContext, error) {
	s.contextCalls.Add(1)
	return model.VIPPaymentContext{
		OrderID: input.OrderID, MerchantID: input.MerchantID, UserID: input.UserID,
		OpenID: "openid", Status: model.PaymentOrderStatusPending, AmountTotal: 1990, Currency: "CNY", PlanName: "VIP 月卡",
	}, nil
}

func (s *concurrentVIPPaymentStore) CreateVIPPaymentOrder(_ context.Context, input model.CreateVIPPaymentOrderInput) (model.VIPPaymentOrder, error) {
	s.createCalls.Add(1)
	return model.VIPPaymentOrder{
		ID: input.OrderID, OutTradeNo: "VIP202608060001", AmountTotal: 1990,
		Currency: "CNY", Status: model.PaymentOrderStatusPending, PlanName: "VIP 月卡",
	}, nil
}

func (s *concurrentVIPPaymentStore) MarkVIPOrderPaid(context.Context, model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error) {
	s.markPaidCalls.Add(1)
	return model.VIPPaymentResult{OrderID: "order-1", MerchantID: "merchant-1", Status: model.PaymentOrderStatusPaid}, nil
}

type concurrentVIPPaymentGateway struct {
	prepayCalls atomic.Int64
}

func (g *concurrentVIPPaymentGateway) CreatePrepay(context.Context, paymentlogic.WechatPrepayInput) (paymentlogic.WechatPayParams, error) {
	g.prepayCalls.Add(1)
	return paymentlogic.WechatPayParams{TimeStamp: "1", NonceStr: "n", Package: "prepay_id=p", SignType: "RSA", PaySign: "s"}, nil
}

func (*concurrentVIPPaymentGateway) DecodeNotify(context.Context, paymentlogic.WechatPayNotifyReq) (paymentlogic.WechatPayNotification, error) {
	return paymentlogic.WechatPayNotification{}, errors.New("unused")
}
