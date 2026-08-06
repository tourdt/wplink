package entitlement

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/handler/handlerx"
	entitlementlogic "wplink/backend/app/internal/logic/entitlement"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestEntitlementHTTPHandlersUseMerchantPermissionAndKeepEmptyArrays(t *testing.T) {
	store := &fakeEntitlementHandlerStore{ownerMerchantID: "merchant-real"}
	permissionStore := &fakeEntitlementPermissionStore{allowedMerchantID: "merchant-real"}
	deps := entitlementTestPermissionDeps(permissionStore)

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		method     string
		target     string
		body       string
		wantItems  bool
		wantStatus int
	}{
		{name: "entitlements", handler: listMerchantEntitlementsHTTPHandler(store, deps), method: http.MethodGet, target: "/api/v1/merchants/merchant-real/entitlements", wantItems: true, wantStatus: http.StatusOK},
		{name: "usage records", handler: listEntitlementUsageRecordsHTTPHandler(store, deps), method: http.MethodGet, target: "/api/v1/merchants/merchant-real/entitlements/entitlement-1/usage-records", wantItems: true, wantStatus: http.StatusOK},
		{name: "top vouchers", handler: listTopVouchersHTTPHandler(store, deps), method: http.MethodGet, target: "/api/v1/merchants/merchant-real/top-vouchers", wantItems: true, wantStatus: http.StatusOK},
		{name: "redeem", handler: redeemTopVoucherHTTPHandler(store, store, deps), method: http.MethodPost, target: "/api/v1/top-vouchers/voucher-1/redeem", body: `{"merchantId":"merchant-attacker","resourceId":"resource-1"}`, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := entitlementHandlerRequest(tc.method, tc.target, tc.body)
			rec := httptest.NewRecorder()
			tc.handler(rec, req)
			data := entitlementHandlerData(t, rec, tc.wantStatus)
			if tc.wantItems {
				items, ok := data["items"].([]interface{})
				if !ok || len(items) != 0 {
					t.Fatalf("data=%#v, want items: []", data)
				}
			}
		})
	}

	if store.redeemVoucherID != "voucher-1" || store.redeemResourceID != "resource-1" {
		t.Fatalf("redeem=(%q,%q), want path voucher and DTO resource", store.redeemVoucherID, store.redeemResourceID)
	}
	if permissionStore.lastMerchantID != "merchant-real" {
		t.Fatalf("permission merchant=%q, want stored voucher owner", permissionStore.lastMerchantID)
	}
}

func TestEntitlementHTTPHandlersRejectMissingAndTypedNilDependencies(t *testing.T) {
	var typedNilStore *fakeEntitlementHandlerStore
	var typedNilPermissionStore *fakeEntitlementPermissionStore
	var typedNilUserTokenService *fakeEntitlementUserTokenService
	var typedNilAdminTokenService *fakeEntitlementAdminTokenService
	typedNilUserDeps := entitlementTestPermissionDeps(&fakeEntitlementPermissionStore{})
	typedNilUserDeps.UserTokenService = typedNilUserTokenService
	typedNilAdminDeps := entitlementTestPermissionDeps(&fakeEntitlementPermissionStore{})
	typedNilAdminDeps.AdminTokenService = typedNilAdminTokenService
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "nil store", handler: listMerchantEntitlementsHTTPHandler(nil, entitlementTestPermissionDeps(&fakeEntitlementPermissionStore{}))},
		{name: "typed nil store", handler: listMerchantEntitlementsHTTPHandler(typedNilStore, entitlementTestPermissionDeps(&fakeEntitlementPermissionStore{}))},
		{name: "typed nil permission store", handler: listTopVouchersHTTPHandler(&fakeEntitlementHandlerStore{}, entitlementTestPermissionDeps(typedNilPermissionStore))},
		{name: "typed nil user token service", handler: listMerchantEntitlementsHTTPHandler(&fakeEntitlementHandlerStore{}, typedNilUserDeps)},
		{name: "typed nil admin token service", handler: listMerchantEntitlementsHTTPHandler(&fakeEntitlementHandlerStore{}, typedNilAdminDeps)},
		{name: "nil voucher owner store", handler: redeemTopVoucherHTTPHandler(&fakeEntitlementHandlerStore{}, nil, entitlementTestPermissionDeps(&fakeEntitlementPermissionStore{}))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, entitlementHandlerRequest(http.MethodGet, "/api/v1/merchants/merchant-1/entitlements", ""))
			body := entitlementHandlerEnvelope(t, rec, http.StatusInternalServerError)
			if body["errorCode"] != "INTERNAL_ERROR" {
				t.Fatalf("body=%#v, want safe dependency error", body)
			}
		})
	}
}

func entitlementTestPermissionDeps(store handlerx.MerchantPermissionStore) handlerx.MerchantPermissionDeps {
	return handlerx.MerchantPermissionDeps{
		UserTokenService:  &fakeEntitlementUserTokenService{},
		AdminTokenService: &fakeEntitlementAdminTokenService{},
		Store:             store,
	}
}

func entitlementHandlerRequest(method string, target string, body string) *http.Request {
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
	if len(parts) >= 6 && parts[4] == "entitlements" {
		vars["entitlementId"] = parts[5]
	}
	if len(parts) >= 4 && parts[2] == "top-vouchers" {
		vars["voucherId"] = parts[3]
	}
	return pathvar.WithVars(req, vars)
}

func entitlementHandlerData(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) map[string]interface{} {
	t.Helper()
	body := entitlementHandlerEnvelope(t, rec, wantStatus)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body=%#v, want data object", body)
	}
	return data
}

func entitlementHandlerEnvelope(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) map[string]interface{} {
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

type fakeEntitlementHandlerStore struct {
	ownerMerchantID  string
	redeemVoucherID  string
	redeemResourceID string
}

func (s *fakeEntitlementHandlerStore) ListMerchantEntitlements(context.Context, string) ([]model.MerchantEntitlement, error) {
	return []model.MerchantEntitlement{}, nil
}

func (s *fakeEntitlementHandlerStore) ListMerchantEntitlementUsageRecords(context.Context, string, string) ([]model.EntitlementUsageRecord, error) {
	return []model.EntitlementUsageRecord{}, nil
}

func (s *fakeEntitlementHandlerStore) ListTopVouchers(context.Context, string) ([]model.TopVoucher, error) {
	return []model.TopVoucher{}, nil
}

func (s *fakeEntitlementHandlerStore) RedeemTopVoucher(_ context.Context, voucherID string, resourceID string) (model.RedeemTopVoucherResult, error) {
	s.redeemVoucherID = voucherID
	s.redeemResourceID = resourceID
	return model.RedeemTopVoucherResult{VoucherID: voucherID, ResourceID: resourceID, Status: "used"}, nil
}

func (s *fakeEntitlementHandlerStore) GetTopVoucherMerchantID(context.Context, string) (string, error) {
	if strings.TrimSpace(s.ownerMerchantID) == "" {
		return "", errors.New("voucher owner unavailable")
	}
	return s.ownerMerchantID, nil
}

var _ entitlementlogic.EntitlementStore = (*fakeEntitlementHandlerStore)(nil)

type fakeEntitlementPermissionStore struct {
	allowedMerchantID string
	lastMerchantID    string
}

func (s *fakeEntitlementPermissionStore) UserCanManageMerchant(_ context.Context, _ string, merchantID string) (bool, error) {
	s.lastMerchantID = merchantID
	return merchantID == s.allowedMerchantID, nil
}

type fakeEntitlementUserTokenService struct{}

func (*fakeEntitlementUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (*fakeEntitlementUserTokenService) ParseUserToken(context.Context, string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{UserID: "user-real"}, nil
}

type fakeEntitlementAdminTokenService struct{}

func (*fakeEntitlementAdminTokenService) ParseAdminToken(context.Context, string) (session.AdminTokenSubject, error) {
	return session.AdminTokenSubject{}, errors.New("not admin")
}
