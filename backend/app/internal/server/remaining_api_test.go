package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	paymentlogic "wplink/backend/app/internal/logic/payment"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
)

func TestAPIRouterRequiresAdminTokenWhenConfigured(t *testing.T) {
	tokenService := &fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "admin-1",
		Roles:      []string{permission.RolePlatformOperator},
		Modules:    []string{permission.AdminModuleDashboard},
	}}
	router := NewAPIRouter(newFakeFullAPIStore(), WithAdminTokenService(tokenService))

	unauthorizedRec := httptest.NewRecorder()
	unauthorizedReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	router.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s, want unauthorized", unauthorizedRec.Code, unauthorizedRec.Body.String())
	}

	authorizedRec := httptest.NewRecorder()
	authorizedReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	authorizedReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(authorizedRec, authorizedReq)
	decodeEnvelopeData(t, authorizedRec, http.StatusOK)
}

func TestAPIRouterRejectsAdminModuleOutsideTokenPermissions(t *testing.T) {
	router := NewAPIRouter(newFakeFullAPIStore(), WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "admin-1",
		Roles:      []string{permission.RolePlatformOperator},
		Modules:    []string{permission.AdminModuleResourceReview},
	}}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", rec.Code, rec.Body.String())
	}
}

func TestAPIRouterUsesTokenSubjectForPrivateUserRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	messageRec := httptest.NewRecorder()
	messageReq := httptest.NewRequest(http.MethodGet, "/api/v1/messages?userId=attacker&page=1&pageSize=20", nil)
	messageReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(messageRec, messageReq)
	decodeEnvelopeData(t, messageRec, http.StatusOK)
	if store.messageFilter.UserID != "user-1" {
		t.Fatalf("message userID = %q, want token user", store.messageFilter.UserID)
	}

	readRec := httptest.NewRecorder()
	readReq := httptest.NewRequest(http.MethodPost, "/api/v1/messages/message-1/read", strings.NewReader(`{"userId":"attacker"}`))
	readReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(readRec, readReq)
	decodeEnvelopeData(t, readRec, http.StatusOK)
	if store.readMessageUserID != "user-1" {
		t.Fatalf("readMessageUserID = %q, want token user", store.readMessageUserID)
	}
}

func TestAPIRouterRequiresMerchantPermissionForMerchantMessages(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/messages?roleCode=merchant:merchant-2", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodGet, "/api/v1/messages?roleCode=merchant:merchant-1", nil)
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
	if store.messageFilter.RoleCode != "merchant:merchant-1" {
		t.Fatalf("message roleCode = %q, want merchant role", store.messageFilter.RoleCode)
	}
}

func TestAPIRouterDerivesMessageRecipientsFromToken(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true, "merchant-2": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/messages?page=1&pageSize=20", nil)
	listReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(listRec, listReq)
	decodeEnvelopeData(t, listRec, http.StatusOK)
	if store.messageFilter.UserID != "user-1" {
		t.Fatalf("message userID = %q, want token user", store.messageFilter.UserID)
	}
	wantRoleCodes := []string{"merchant:merchant-1", "merchant:merchant-2"}
	if strings.Join(store.messageFilter.RoleCodes, ",") != strings.Join(wantRoleCodes, ",") {
		t.Fatalf("message roleCodes = %#v, want %#v", store.messageFilter.RoleCodes, wantRoleCodes)
	}

	readRec := httptest.NewRecorder()
	readReq := httptest.NewRequest(http.MethodPost, "/api/v1/messages/message-1/read", strings.NewReader(`{}`))
	readReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(readRec, readReq)
	decodeEnvelopeData(t, readRec, http.StatusOK)
	if store.readMessageUserID != "user-1" || strings.Join(store.readMessageRoleCodes, ",") != strings.Join(wantRoleCodes, ",") {
		t.Fatalf("read identity userID=%q roleCodes=%#v, want token user and managed merchant roles", store.readMessageUserID, store.readMessageRoleCodes)
	}
}

func TestAPIRouterReturnsEditableMerchantContactForManagerOnly(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	authorizedRec := httptest.NewRecorder()
	authorizedReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1", nil)
	authorizedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(authorizedRec, authorizedReq)
	authorizedData := decodeEnvelopeData(t, authorizedRec, http.StatusOK)
	authorizedContact := authorizedData["contact"].(map[string]interface{})
	if authorizedContact["phone"] != "18800000002" || authorizedContact["wechat"] != "stock-demo" {
		t.Fatalf("authorized contact = %#v, want editable phone and wechat", authorizedContact)
	}

	publicRec := httptest.NewRecorder()
	publicReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1", nil)
	router.ServeHTTP(publicRec, publicReq)
	publicData := decodeEnvelopeData(t, publicRec, http.StatusOK)
	publicContact := publicData["contact"].(map[string]interface{})
	if _, ok := publicContact["phone"]; ok {
		t.Fatalf("public contact = %#v, should not expose raw phone", publicContact)
	}
	if _, ok := publicContact["wechat"]; ok {
		t.Fatalf("public contact = %#v, should not expose raw wechat", publicContact)
	}
}

func TestAPIRouterDoesNotExposePurchaseDemandRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create user demand", method: http.MethodPost, path: "/api/v1/purchase-demands", body: `{"title":"找童装库存"}`},
		{name: "list my demands", method: http.MethodGet, path: "/api/v1/me/purchase-demands?userId=attacker"},
		{name: "list admin demands", method: http.MethodGet, path: "/api/v1/admin/purchase-demands"},
		{name: "get admin demand", method: http.MethodGet, path: "/api/v1/admin/purchase-demands/demand-1"},
		{name: "update admin demand", method: http.MethodPost, path: "/api/v1/admin/purchase-demands/demand-1/status", body: `{"status":"matching"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer user-token")
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d body = %s, want not found", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAPIRouterRejectsUnmappedRetiredAdminRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "admin-1", Roles: []string{"platform_operator"}}}))

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create match case", method: http.MethodPost, path: "/api/v1/admin/match-cases", body: `{"purchaseDemandId":"demand-1"}`},
		{name: "list match cases", method: http.MethodGet, path: "/api/v1/admin/match-cases"},
		{name: "update match case", method: http.MethodPost, path: "/api/v1/admin/match-cases/match-1/status", body: `{"status":"contacted"}`},
		{name: "add match resources", method: http.MethodPost, path: "/api/v1/admin/match-cases/match-1/resources", body: `{"resourceIds":["resource-1"]}`},
		{name: "add match participants", method: http.MethodPost, path: "/api/v1/admin/match-cases/match-1/participants", body: `{"participantMerchantIds":["merchant-1"]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer admin-token")
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d body = %s, want forbidden for unmapped admin route", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAPIRouterAdminPermissionRoutesRequireSuperAdmin(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "admin-1", Roles: []string{"platform_operator"}}}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/operators", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", rec.Code, rec.Body.String())
	}
}

func TestAPIRouterAdminPermissionRoutesUseTokenActor(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "super-1", Roles: []string{permission.RoleSuperAdmin}}}))

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/operators?keyword=%E8%BF%90%E8%90%A5&role=platform_operator&status=enabled&page=2&pageSize=5", nil)
	listReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(listRec, listReq)
	decodeEnvelopeData(t, listRec, http.StatusOK)
	if store.adminOperatorFilter.Keyword != "运营" || store.adminOperatorFilter.Role != "platform_operator" || store.adminOperatorFilter.Page != 2 {
		t.Fatalf("admin operator filter = %#v, want query filter from request", store.adminOperatorFilter)
	}

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/operators", strings.NewReader(`{
		"loginName":"18800000001",
		"realName":"运营一号",
		"password":"secret123",
		"roles":["platform_operator"],
		"status":"enabled"
	}`))
	createReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(createRec, createReq)
	decodeEnvelopeData(t, createRec, http.StatusOK)
	if store.createAdminOperatorInput.ActorID != "super-1" {
		t.Fatalf("create admin operator actor id = %q, want token operator", store.createAdminOperatorInput.ActorID)
	}

	moduleListRec := httptest.NewRecorder()
	moduleListReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/module-permissions", nil)
	moduleListReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(moduleListRec, moduleListReq)
	moduleData := decodeEnvelopeData(t, moduleListRec, http.StatusOK)
	if _, ok := moduleData["modules"].([]interface{}); !ok {
		t.Fatalf("module permissions data = %#v, want modules array", moduleData)
	}

	moduleSaveRec := httptest.NewRecorder()
	moduleSaveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/module-permissions/platform_operator", strings.NewReader(`{"modules":["merchants","resource_review"]}`))
	moduleSaveReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(moduleSaveRec, moduleSaveReq)
	decodeEnvelopeData(t, moduleSaveRec, http.StatusOK)
	if store.adminRoleModuleInput.OperatorID != "super-1" {
		t.Fatalf("module permission operatorID = %q, want token user", store.adminRoleModuleInput.OperatorID)
	}
}

func TestAPIRouterUsesTokenSubjectAndMerchantPermissionForVerification(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-2/verifications", strings.NewReader(`{
		"applicantUserId":"attacker",
		"verificationType":"stockist",
		"businessName":"织里云仓"
	}`))
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1/verifications", strings.NewReader(`{
		"applicantUserId":"attacker",
		"verificationType":"stockist",
		"businessName":"织里云仓"
	}`))
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	if allowedRec.Code != http.StatusForbidden || !strings.Contains(allowedRec.Body.String(), publicMerchantVerificationDisabledMessage) {
		t.Fatalf("status = %d body = %s, want disabled certification response", allowedRec.Code, allowedRec.Body.String())
	}
	if store.submitVerificationInput.MerchantID != "" {
		t.Fatalf("submitVerificationInput = %#v, want no public certification submit", store.submitVerificationInput)
	}

	latestForbiddenRec := httptest.NewRecorder()
	latestForbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-2/verifications/latest", nil)
	latestForbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(latestForbiddenRec, latestForbiddenReq)
	if latestForbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("latest status = %d body = %s, want forbidden", latestForbiddenRec.Code, latestForbiddenRec.Body.String())
	}

	latestAllowedRec := httptest.NewRecorder()
	latestAllowedReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1/verifications/latest", nil)
	latestAllowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(latestAllowedRec, latestAllowedReq)
	decodeEnvelopeData(t, latestAllowedRec, http.StatusOK)
	if store.latestVerificationMerchantID != "merchant-1" {
		t.Fatalf("latest merchantID = %q, want merchant-1", store.latestVerificationMerchantID)
	}

	billingRec := httptest.NewRecorder()
	billingReq := httptest.NewRequest(http.MethodGet, "/api/v1/verification-billing", nil)
	router.ServeHTTP(billingRec, billingReq)
	if billingRec.Code != http.StatusForbidden || !strings.Contains(billingRec.Body.String(), publicMerchantVerificationDisabledMessage) {
		t.Fatalf("billing status = %d body = %s, want disabled certification billing", billingRec.Code, billingRec.Body.String())
	}
}

func TestAPIRouterMarksMerchantRoleMessageRead(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/message-1/read", strings.NewReader(`{"userId":"attacker","roleCode":"merchant:merchant-1"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(rec, req)
	decodeEnvelopeData(t, rec, http.StatusOK)
	if store.readMessageUserID != "user-1" || store.readMessageRoleCode != "merchant:merchant-1" {
		t.Fatalf("read identity userID=%q roleCode=%q, want token user and merchant role", store.readMessageUserID, store.readMessageRoleCode)
	}
}

func TestAPIRouterRequiresMerchantPermissionForEntitlements(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "list entitlements", method: http.MethodGet, path: "/api/v1/merchants/merchant-2/entitlements"},
		{name: "list top vouchers", method: http.MethodGet, path: "/api/v1/merchants/merchant-2/top-vouchers"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer user-token")
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d body = %s, want forbidden", rec.Code, rec.Body.String())
			}
		})
	}

	redeemRec := httptest.NewRecorder()
	redeemReq := httptest.NewRequest(http.MethodPost, "/api/v1/top-vouchers/voucher-1/redeem", strings.NewReader(`{"merchantId":"merchant-1","resourceId":"resource-1"}`))
	redeemReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(redeemRec, redeemReq)
	decodeEnvelopeData(t, redeemRec, http.StatusOK)
	if store.redeemVoucherID != "voucher-1" || store.redeemResourceID != "resource-1" {
		t.Fatalf("redeem voucherID=%q resourceID=%q, want voucher/resource IDs", store.redeemVoucherID, store.redeemResourceID)
	}
}

func TestAPIRouterRequiresMerchantPermissionForGrowthTasks(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-2/growth-tasks", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1/growth-tasks", nil)
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	data := decodeEnvelopeData(t, allowedRec, http.StatusOK)
	if _, ok := data["tasks"].([]interface{}); !ok {
		t.Fatalf("data = %#v, want tasks array", data)
	}
}

func TestAPIRouterRegistersVIPMembershipRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}), WithWechatPayDevMock(true))

	plansRec := httptest.NewRecorder()
	plansReq := httptest.NewRequest(http.MethodGet, "/api/v1/vip/plans", nil)
	router.ServeHTTP(plansRec, plansReq)
	plansData := decodeEnvelopeData(t, plansRec, http.StatusOK)
	plans, ok := plansData["items"].([]interface{})
	if !ok || len(plans) != 1 {
		t.Fatalf("plans data = %#v, want one vip plan", plansData)
	}

	packsRec := httptest.NewRecorder()
	packsReq := httptest.NewRequest(http.MethodGet, "/api/v1/vip/quota-packs", nil)
	router.ServeHTTP(packsRec, packsReq)
	packsData := decodeEnvelopeData(t, packsRec, http.StatusOK)
	packs, ok := packsData["items"].([]interface{})
	if !ok || len(packs) != 1 {
		t.Fatalf("quota packs data = %#v, want one quota pack", packsData)
	}

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-2/vip", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	statusRec := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1/vip", nil)
	statusReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(statusRec, statusReq)
	statusData := decodeEnvelopeData(t, statusRec, http.StatusOK)
	if statusData["status"] != model.VIPStatusActive {
		t.Fatalf("vip status data = %#v, want active", statusData)
	}

	orderRec := httptest.NewRecorder()
	orderReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1/vip/orders", strings.NewReader(`{"planCode":"monthly"}`))
	orderReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(orderRec, orderReq)
	decodeEnvelopeData(t, orderRec, http.StatusOK)
	if store.createVIPOrderInput.UserID != "user-1" || store.createVIPOrderInput.PlanCode != "monthly" {
		t.Fatalf("createVIPOrderInput = %#v, want token user and plan", store.createVIPOrderInput)
	}

	quotaOrderRec := httptest.NewRecorder()
	quotaOrderReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1/vip/orders", strings.NewReader(`{"productType":"quota_pack","productCode":"publish_5","resourceId":"resource-1"}`))
	quotaOrderReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(quotaOrderRec, quotaOrderReq)
	decodeEnvelopeData(t, quotaOrderRec, http.StatusOK)
	if store.createQuotaPackOrderInput.UserID != "user-1" || store.createQuotaPackOrderInput.PackCode != "publish_5" || store.createQuotaPackOrderInput.ResourceID != "resource-1" {
		t.Fatalf("createQuotaPackOrderInput = %#v, want token user and pack code", store.createQuotaPackOrderInput)
	}

	paymentRec := httptest.NewRecorder()
	paymentReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1/vip/orders/order-1/payment", strings.NewReader(`{"userId":"attacker"}`))
	paymentReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(paymentRec, paymentReq)
	decodeEnvelopeData(t, paymentRec, http.StatusOK)
	if store.markVIPOrderPaidInput.OutTradeNo != "VIP202607090001" {
		t.Fatalf("markVIPOrderPaidInput = %#v, want dev mock payment", store.markVIPOrderPaidInput)
	}
}

func TestAPIRouterRegistersUnifiedWechatPayNotifyRoute(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithWechatPayGateway(&fakeServerWechatPayGateway{
		notify: paymentlogic.WechatPayNotification{
			OutTradeNo:    "CU202607140001",
			TransactionID: "wx-transaction-1",
			Attach:        "contact_unlock:order-1",
			AmountTotal:   500,
			SuccessTime:   "2026-07-14T12:00:00Z",
			RawPayload:    map[string]interface{}{"trade_state": "SUCCESS"},
		},
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat-pay/notify", strings.NewReader(`{"id":"notify-1"}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s, want 200", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"SUCCESS"`) {
		t.Fatalf("body = %s, want SUCCESS notify response", rec.Body.String())
	}
	if store.contactUnlockMarkInput.OutTradeNo != "CU202607140001" {
		t.Fatalf("contactUnlockMarkInput = %#v, want contact unlock notify", store.contactUnlockMarkInput)
	}
}

func TestAPIRouterRegistersPublicGrowthCampaignRoute(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/growth-campaigns/active", nil)
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("growth campaign data = %#v, want one active campaign", data)
	}
	campaign := items[0].(map[string]interface{})
	if campaign["title"] != "新手发布权益" {
		t.Fatalf("campaign = %#v, want public starter title", campaign)
	}
	rules := campaign["rules"].([]interface{})
	if len(rules) != 1 || rules[0].(map[string]interface{})["rewardText"] != "5 次发布次数" {
		t.Fatalf("rules = %#v, want Chinese public rule", rules)
	}
}

func TestAPIRouterServesAdminVIPConfigRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "admin-1", Roles: []string{permission.RoleSuperAdmin}}}))

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/vip/plans", nil)
	listReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(listRec, listReq)
	data := decodeEnvelopeData(t, listRec, http.StatusOK)
	items, ok := data["items"].([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("items = %#v, want vip plan items", data["items"])
	}

	saveRec := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/vip/plans/monthly", strings.NewReader(`{
		"name":"VIP 月卡",
		"durationMonths":1,
		"standardPriceCent":4900,
		"status":"active",
		"displayOrder":10,
		"benefits":{"publishQuota":80,"refreshQuota":30,"topVoucherCount":3,"topDurationHours":24}
	}`))
	saveReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(saveRec, saveReq)
	decodeEnvelopeData(t, saveRec, http.StatusOK)
	if store.saveVIPPlanInput.OperatorID != "admin-1" {
		t.Fatalf("operatorID = %q, want token admin user", store.saveVIPPlanInput.OperatorID)
	}
}

func TestAPIRouterRequiresOwnedMerchantForTopVoucherRedeem(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	store.topVoucherMerchantIDs = map[string]string{"voucher-1": "merchant-2"}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodPost, "/api/v1/top-vouchers/voucher-1/redeem", strings.NewReader(`{"merchantId":"merchant-1","resourceId":"resource-1"}`))
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}
	if store.redeemVoucherID != "" {
		t.Fatalf("redeemVoucherID = %q, want no redeem call", store.redeemVoucherID)
	}

	store.topVoucherMerchantIDs = map[string]string{"voucher-1": "merchant-1"}
	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodPost, "/api/v1/top-vouchers/voucher-1/redeem", strings.NewReader(`{"merchantId":"merchant-2","resourceId":"resource-1"}`))
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
	if store.redeemVoucherID != "voucher-1" || store.redeemResourceID != "resource-1" {
		t.Fatalf("redeem voucherID=%q resourceID=%q, want voucher/resource IDs", store.redeemVoucherID, store.redeemResourceID)
	}
}

func TestAPIRouterRequiresMerchantPermissionForMerchantMetrics(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-2/metrics/summary", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1/metrics/summary", nil)
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
}

func TestAPIRouterRequiresMerchantPermissionForResourceMetrics(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	store.resourceMerchantIDs = map[string]string{"resource-1": "merchant-1", "resource-2": "merchant-2"}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources/resource-2/metrics", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources/resource-1/metrics", nil)
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
}

func TestAPIRouterDisablesSecondMerchantCreationAndProtectsMerchantUpdate(t *testing.T) {
	store := newFakeFullAPIStore()
	store.managedMerchants = map[string]bool{"merchant-1": true}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants", strings.NewReader(`{}`))
	createReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s, want create merchant route removed", createRec.Code, createRec.Body.String())
	}

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-2", strings.NewReader(`{"mainCategories":["童装"],"description":"更新简介"}`))
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", strings.NewReader(`{"mainCategories":["童装"],"description":"更新简介"}`))
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
}

func TestAPIRouterUsesAdminTokenOperatorForAdminActions(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(
		store,
		WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "admin-1", Roles: []string{permission.RoleSuperAdmin}}}),
	)

	entitlementRec := httptest.NewRecorder()
	entitlementReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/merchants/merchant-1/entitlements", strings.NewReader(`{
		"operatorId":"attacker",
		"entitlementType":"publish_quota",
		"sourceType":"manual",
		"totalAmount":3,
		"reason":"测试发放"
	}`))
	entitlementReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(entitlementRec, entitlementReq)
	decodeEnvelopeData(t, entitlementRec, http.StatusOK)
	if store.grantEntitlementInput.OperatorID != "admin-1" {
		t.Fatalf("grant operatorID = %q, want admin token user", store.grantEntitlementInput.OperatorID)
	}

}

func TestAPIRouterRunsRemainingDomainRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "get merchant", method: http.MethodGet, path: "/api/v1/merchants/merchant-1"},
		{name: "update merchant", method: http.MethodPost, path: "/api/v1/merchants/merchant-1", body: `{"name":"  织里晨星童装  ","mainCategories":["童装"],"merchantType":"service_provider","description":"更新简介","logoUrl":"https://example.com/logo.png","images":["https://example.com/a.jpg"],"addressText":"织里镇利济路88号","location":{"latitude":30.1,"longitude":120.2,"name":"织里童装城","address":"织里镇利济路88号"}}`},
		{name: "home operation config", method: http.MethodGet, path: "/api/v1/home/operation-config?cityCode=zhili"},
		{name: "home resources", method: http.MethodGet, path: "/api/v1/home/resources?cityCode=zhili"},
		{name: "hot search keywords", method: http.MethodGet, path: "/api/v1/search/hot-keywords?cityCode=zhili"},
		{name: "topic resources", method: http.MethodGet, path: "/api/v1/topics/topic-1/resources?cityCode=zhili"},
		{name: "validate webview", method: http.MethodPost, path: "/api/v1/webview/validate", body: `{"url":"https://www.wplink.cn/activity"}`},
		{name: "latest verification", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/verifications/latest"},
		{name: "list entitlements", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/entitlements"},
		{name: "list entitlement usage records", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/entitlements/entitlement-1/usage-records"},
		{name: "list top vouchers", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/top-vouchers"},
		{name: "redeem top voucher", method: http.MethodPost, path: "/api/v1/top-vouchers/voucher-1/redeem", body: `{"resourceId":"resource-1"}`},
		{name: "resource metrics", method: http.MethodGet, path: "/api/v1/resources/resource-1/metrics"},
		{name: "merchant metrics", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/metrics/summary"},
		{name: "list messages", method: http.MethodGet, path: "/api/v1/messages?userId=user-1"},
		{name: "read message", method: http.MethodPost, path: "/api/v1/messages/message-1/read", body: `{"userId":"user-1"}`},
		{name: "dashboard", method: http.MethodGet, path: "/api/v1/admin/dashboard/overview?cityCode=zhili"},
		{name: "list admin merchants", method: http.MethodGet, path: "/api/v1/admin/merchants?cityCode=zhili"},
		{name: "list admin banners", method: http.MethodGet, path: "/api/v1/admin/banner-topics?cityCode=zhili"},
		{name: "create admin banner", method: http.MethodPost, path: "/api/v1/admin/banner-topics", body: `{"cityCode":"zhili","kind":"banner","title":"现货活动","jumpType":"internal","jumpTarget":"/pages/search/index","status":"active"}`},
		{name: "update admin banner", method: http.MethodPost, path: "/api/v1/admin/banner-topics/banner-1", body: `{"cityCode":"zhili","kind":"banner","title":"现货活动","jumpType":"internal","jumpTarget":"/pages/search/index","status":"active"}`},
		{name: "list admin hot keywords", method: http.MethodGet, path: "/api/v1/admin/hot-search-keywords?cityCode=zhili"},
		{name: "create admin hot keyword", method: http.MethodPost, path: "/api/v1/admin/hot-search-keywords", body: `{"cityCode":"zhili","keyword":"夏款现货","status":"active","sortOrder":20}`},
		{name: "update admin hot keyword", method: http.MethodPost, path: "/api/v1/admin/hot-search-keywords/keyword-1", body: `{"cityCode":"zhili","keyword":"夏款现货","status":"active","sortOrder":20}`},
		{name: "list resource configs", method: http.MethodGet, path: "/api/v1/admin/resource-type-configs?cityCode=zhili"},
		{name: "create resource config", method: http.MethodPost, path: "/api/v1/admin/resource-type-configs", body: `{"cityCode":"zhili","typeCode":"kids_brand_stock","typeName":"品牌库存","direction":"supply","groupCode":"kids_wholesale","groupName":"童装批发","defaultValidDays":15}`},
		{name: "update resource config", method: http.MethodPost, path: "/api/v1/admin/resource-type-configs/config-1", body: `{"version":1,"fieldSchema":{},"requiredFields":["title"],"filterFields":["category"],"displayTemplate":{},"reviewRules":{},"sortWeights":{},"messageRules":{},"defaultValidDays":7,"status":"active"}`},
		{name: "list pending verifications", method: http.MethodGet, path: "/api/v1/admin/verifications/pending"},
		{name: "review verification", method: http.MethodPost, path: "/api/v1/admin/verifications/verification-1/review", body: `{"reviewerId":"user-1","action":"approve"}`},
		{name: "grant entitlement", method: http.MethodPost, path: "/api/v1/admin/merchants/merchant-1/entitlements", body: `{"operatorId":"user-1","entitlementType":"publish_quota","sourceType":"manual","totalAmount":3,"reason":"测试发放"}`},
		{name: "operation logs", method: http.MethodGet, path: "/api/v1/admin/operation-logs?objectType=resource"},
		{name: "search logs", method: http.MethodGet, path: "/api/v1/admin/search-logs?cityCode=zhili&keyword=童装"},
		{name: "run lifecycle task", method: http.MethodPost, path: "/api/v1/admin/tasks/resource-lifecycle/run"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			router.ServeHTTP(rec, req)
			decodeEnvelopeData(t, rec, http.StatusOK)
		})
	}
	if store.updateMerchantPatch.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("update merchant logoURL = %q, want decoded logo URL", store.updateMerchantPatch.LogoURL)
	}
	if store.updateMerchantPatch.Name != "织里晨星童装" {
		t.Fatalf("update merchant name = %q, want decoded display name", store.updateMerchantPatch.Name)
	}
	if store.updateMerchantPatch.MerchantType != "service_provider" {
		t.Fatalf("update merchant merchantType = %q, want decoded service_provider", store.updateMerchantPatch.MerchantType)
	}
	if store.updateMerchantPatch.AddressText != "织里镇利济路88号" {
		t.Fatalf("update merchant addressText = %q, want decoded address", store.updateMerchantPatch.AddressText)
	}
	if store.updateMerchantPatch.Location["name"] != "织里童装城" || store.updateMerchantPatch.Location["address"] != "织里镇利济路88号" {
		t.Fatalf("update merchant location = %#v, want decoded map location", store.updateMerchantPatch.Location)
	}
}

func newFakeFullAPIStore() *fakeFullAPIStore {
	return &fakeFullAPIStore{
		fakeResourceAPIStore: fakeResourceAPIStore{
			publishConfig:  model.ResourcePublishConfig{ID: "config-1", TypeCode: "inventory", RequiredFields: []string{"merchantId", "cityCode", "typeCode", "title", "category", "contactName", "contactPhone"}, DefaultValidDays: 7},
			merchantStatus: model.MerchantStatusActive,
		},
		latestVerificationErr: sql.ErrNoRows,
	}
}

type fakeFullAPIStore struct {
	fakeResourceAPIStore
	updateMerchantPatch          model.UpdateMerchantPatch
	submitVerificationInput      model.SubmitVerificationInput
	latestVerificationMerchantID string
	latestVerificationErr        error
	messageFilter                model.ListMessagesFilter
	readMessageUserID            string
	readMessageRoleCode          string
	readMessageRoleCodes         []string
	redeemVoucherID              string
	redeemResourceID             string
	topVoucherMerchantIDs        map[string]string
	grantEntitlementInput        model.GrantEntitlementInput
	createVIPOrderInput          model.CreateVIPOrderInput
	createQuotaPackOrderInput    model.CreateQuotaPackOrderInput
	createVIPPaymentInput        model.CreateVIPPaymentOrderInput
	markVIPOrderPaidInput        model.MarkVIPOrderPaidInput
	markVerificationPaidInput    model.MarkVerificationPaymentPaidInput
	saveVIPPlanInput             model.SaveAdminVIPPlanInput
	saveQuotaPackInput           model.SaveAdminQuotaPackInput
	saveVIPPromotionInput        model.SaveAdminVIPPromotionInput
	adminOperatorFilter          model.AdminOperatorFilter
	createAdminOperatorInput     model.AdminOperatorInput
	updateAdminOperatorInput     model.AdminOperatorInput
	statusAdminOperatorInput     model.AdminOperatorStatusInput
	adminRoleModules             []string
	adminRoleModuleInput         model.AdminRoleModulePermissionInput
}

func (s *fakeFullAPIStore) MarkVerificationPaymentPaid(ctx context.Context, input model.MarkVerificationPaymentPaidInput) (model.VerificationPaymentResult, error) {
	s.markVerificationPaidInput = input
	return model.VerificationPaymentResult{
		OrderID:        "verification-payment-1",
		VerificationID: "verification-1",
		MerchantID:     "merchant-1",
		Status:         model.PaymentOrderStatusPaid,
	}, nil
}

type fakeServerWechatPayGateway struct {
	prepayInput paymentlogic.WechatPrepayInput
	prepay      paymentlogic.WechatPayParams
	notify      paymentlogic.WechatPayNotification
}

func (g *fakeServerWechatPayGateway) CreatePrepay(ctx context.Context, input paymentlogic.WechatPrepayInput) (paymentlogic.WechatPayParams, error) {
	g.prepayInput = input
	return g.prepay, nil
}

func (g *fakeServerWechatPayGateway) DecodeNotify(ctx context.Context, req paymentlogic.WechatPayNotifyReq) (paymentlogic.WechatPayNotification, error) {
	return g.notify, nil
}

type fakeAdminTokenService struct {
	subject session.AdminTokenSubject
	err     error
	token   string
}

func (s *fakeAdminTokenService) ParseAdminToken(ctx context.Context, token string) (session.AdminTokenSubject, error) {
	s.token = token
	return s.subject, s.err
}

func (s *fakeFullAPIStore) GetMerchantDetail(ctx context.Context, merchantID string) (model.MerchantDetail, error) {
	return model.MerchantDetail{ID: merchantID, Name: "织里云仓", MerchantType: "stockist", CityCode: "zhili", MainCategories: []string{"童装"}, VerificationStatus: "verified", ContactName: "周经理", ContactPhone: "18800000002", ContactWechat: "stock-demo", PhoneMasked: "188****0002", WechatMasked: "stock-demo", AddressText: "织里镇利济路88号", Location: model.JSONMap{"latitude": 30.1, "longitude": 120.2, "name": "织里童装城", "address": "织里镇利济路88号"}, PublishedCount: 1}, nil
}

func (s *fakeFullAPIStore) UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error) {
	s.updateMerchantPatch = patch
	return "2026-06-28T10:00:00Z", nil
}

func (s *fakeFullAPIStore) ListMerchants(ctx context.Context, filter model.ListMerchantsFilter) (model.ListMerchantsResult, error) {
	return model.ListMerchantsResult{Items: []model.MerchantListItem{{ID: "merchant-1", Name: "织里云仓", MerchantType: "stockist", VerificationStatus: "verified", Status: model.MerchantStatusActive}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) ListBannerTopics(ctx context.Context, filter model.BannerTopicFilter) ([]model.BannerTopicConfig, error) {
	return []model.BannerTopicConfig{{ID: "banner-1", CityCode: "zhili", Kind: "banner", Title: "现货活动", JumpType: "internal", JumpTarget: "/pages/search/index", Status: "active", UpdatedAt: "2026-06-28T10:00:00Z"}}, nil
}

func (s *fakeFullAPIStore) ListActiveHomeOperationConfigs(ctx context.Context, cityCode string) ([]model.BannerTopicConfig, error) {
	return []model.BannerTopicConfig{
		{ID: "banner-1", CityCode: cityCode, Kind: "banner", Title: "现货活动", JumpType: "internal", JumpTarget: "/pages/search/index", Status: "active", UpdatedAt: "2026-06-28T10:00:00Z"},
		{ID: "recommend-card-1", CityCode: cityCode, Kind: "home_recommend_card", Title: "本周空档工厂", JumpType: "search", JumpTarget: "小单快返", Tags: []string{"平台推荐"}, Status: "active", UpdatedAt: "2026-06-28T10:00:00Z"},
	}, nil
}

func (s *fakeFullAPIStore) GetActiveTopic(ctx context.Context, topicID string, cityCode string) (model.BannerTopicConfig, error) {
	return model.BannerTopicConfig{ID: topicID, CityCode: cityCode, Kind: "topic", Title: "专题", TypeScope: []string{"inventory"}, JumpType: "internal", JumpTarget: "/pages/search/index", Tags: []string{"现货"}, Status: "active"}, nil
}

func (s *fakeFullAPIStore) CreateBannerTopic(ctx context.Context, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error) {
	return model.SaveBannerTopicResult{ID: "banner-1", UpdatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) UpdateBannerTopic(ctx context.Context, configID string, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error) {
	return model.SaveBannerTopicResult{ID: configID, UpdatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) ListHotSearchKeywords(ctx context.Context, filter model.HotSearchKeywordFilter) ([]model.HotSearchKeywordConfig, error) {
	return []model.HotSearchKeywordConfig{{ID: "keyword-1", CityCode: "zhili", Keyword: "夏款现货", Status: "active", UpdatedAt: "2026-06-28T10:00:00Z"}}, nil
}

func (s *fakeFullAPIStore) ListActiveHotSearchKeywords(ctx context.Context, cityCode string) ([]model.HotSearchKeywordConfig, error) {
	return s.ListHotSearchKeywords(ctx, model.HotSearchKeywordFilter{CityCode: cityCode, Status: "active"})
}

func (s *fakeFullAPIStore) CreateHotSearchKeyword(ctx context.Context, input model.SaveHotSearchKeywordInput) (model.SaveHotSearchKeywordResult, error) {
	return model.SaveHotSearchKeywordResult{ID: "keyword-1", UpdatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) UpdateHotSearchKeyword(ctx context.Context, configID string, input model.SaveHotSearchKeywordInput) (model.SaveHotSearchKeywordResult, error) {
	return model.SaveHotSearchKeywordResult{ID: configID, UpdatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) SubmitVerification(ctx context.Context, input model.SubmitVerificationInput) (model.VerificationResult, error) {
	s.submitVerificationInput = input
	return model.VerificationResult{ID: "verification-1", Status: "pending"}, nil
}

func (s *fakeFullAPIStore) GetLatestVerification(ctx context.Context, merchantID string) (model.VerificationBrief, error) {
	s.latestVerificationMerchantID = merchantID
	if s.latestVerificationErr != nil {
		return model.VerificationBrief{}, s.latestVerificationErr
	}
	return model.VerificationBrief{ID: "verification-1", VerificationType: "stockist", Status: "pending"}, nil
}

func (s *fakeFullAPIStore) ListPendingVerifications(ctx context.Context, filter model.PendingVerificationsFilter) (model.ListPendingVerificationsResult, error) {
	return model.ListPendingVerificationsResult{Items: []model.PendingVerificationItem{{ID: "verification-1", MerchantID: "merchant-1", MerchantName: "织里云仓", VerificationType: "stockist", Status: "pending", SubmittedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) GetVerificationBillingConfigForVerification(ctx context.Context, verificationID string) (model.VerificationBillingConfig, error) {
	return model.VerificationBillingConfig{}, nil
}

func (s *fakeFullAPIStore) GetVerificationBillingConfig(ctx context.Context, cityCode string) (model.VerificationBillingConfig, error) {
	return model.VerificationBillingConfig{CityCode: cityCode, Currency: "CNY"}, nil
}

func (s *fakeFullAPIStore) UpdateVerificationBillingConfig(ctx context.Context, input model.VerificationBillingConfig) (model.VerificationBillingConfig, error) {
	return input, nil
}

func (s *fakeFullAPIStore) ReviewVerification(ctx context.Context, input model.ReviewVerificationInput) (model.ReviewVerificationResult, error) {
	return model.ReviewVerificationResult{ID: input.VerificationID, Status: "verified"}, nil
}

func (s *fakeFullAPIStore) ListMerchantEntitlements(ctx context.Context, merchantID string) ([]model.MerchantEntitlement, error) {
	return []model.MerchantEntitlement{{ID: "entitlement-1", Type: "publish_quota", SourceType: "manual", Status: "active", TotalAmount: 3, RemainingAmount: 2}}, nil
}

func (s *fakeFullAPIStore) ListMerchantEntitlementUsageRecords(ctx context.Context, merchantID string, entitlementID string) ([]model.EntitlementUsageRecord, error) {
	return []model.EntitlementUsageRecord{{
		ID: "usage-1", EntitlementID: entitlementID, EntitlementType: "publish_quota", ActionType: "publish_resource",
		Amount: 1, ResourceID: "resource-1", BeforeRemainingAmount: 3, AfterRemainingAmount: 2, UsedAt: "2026-07-09T10:00:00Z",
	}}, nil
}

func (s *fakeFullAPIStore) ListTopVouchers(ctx context.Context, merchantID string) ([]model.TopVoucher, error) {
	return []model.TopVoucher{{ID: "voucher-1", Status: "unused", RemainingAmount: 1, TopDurationHours: 24}}, nil
}

func (s *fakeFullAPIStore) GetTopVoucherMerchantID(ctx context.Context, voucherID string) (string, error) {
	if s.topVoucherMerchantIDs == nil {
		return "merchant-1", nil
	}
	merchantID, ok := s.topVoucherMerchantIDs[voucherID]
	if !ok {
		return "", sql.ErrNoRows
	}
	return merchantID, nil
}

func (s *fakeFullAPIStore) RedeemTopVoucher(ctx context.Context, voucherID string, resourceID string) (model.RedeemTopVoucherResult, error) {
	s.redeemVoucherID = voucherID
	s.redeemResourceID = resourceID
	return model.RedeemTopVoucherResult{VoucherID: voucherID, ResourceID: resourceID, Status: "used"}, nil
}

func (s *fakeFullAPIStore) GrantMerchantEntitlement(ctx context.Context, input model.GrantEntitlementInput) (model.GrantEntitlementResult, error) {
	s.grantEntitlementInput = input
	return model.GrantEntitlementResult{ID: "entitlement-1"}, nil
}

func (s *fakeFullAPIStore) ListActiveGrowthCampaigns(ctx context.Context) ([]model.PublicGrowthCampaign, error) {
	return []model.PublicGrowthCampaign{{
		Code:  "starter_growth_2026_q3",
		Name:  "新手成长权益活动",
		Title: "新手发布权益",
		Hint:  "发布优质资源、有效分享可获得更多曝光权益",
		Rules: []model.PublicGrowthRule{{
			RuleName:     "首条资源审核通过奖励",
			TriggerEvent: model.GrowthEventResourceFirstApproved,
			RewardType:   model.EntitlementTypePublishQuota,
			RewardAmount: 5,
			ValidDays:    30,
		}},
	}}, nil
}

func (s *fakeFullAPIStore) GetGrowthTaskProgress(ctx context.Context, merchantID string) (model.GrowthTaskProgress, error) {
	return model.GrowthTaskProgress{TotalResourceCount: 1, PendingResourceCount: 1}, nil
}

func (s *fakeFullAPIStore) ListMerchantGrowthRewardGrants(ctx context.Context, merchantID string) ([]model.MerchantGrowthRewardGrant, error) {
	return []model.MerchantGrowthRewardGrant{}, nil
}

func (s *fakeFullAPIStore) ListVIPPlans(ctx context.Context) ([]model.VIPPlan, error) {
	return []model.VIPPlan{{
		Code:              "monthly",
		Name:              "VIP 月卡",
		DurationMonths:    1,
		StandardPriceCent: 4900,
		SalePriceCent:     1990,
		SaleLabel:         "首月优惠",
		Benefits:          model.VIPBenefitSnapshot{PublishPolicy: model.VIPPublishPolicyQuota, PublishQuota: 80, RefreshQuota: 30, TopVoucherCount: 3, TopDurationHours: 24},
	}}, nil
}

func (s *fakeFullAPIStore) ListQuotaPacks(ctx context.Context) ([]model.QuotaPack, error) {
	return []model.QuotaPack{{
		Code:              "publish_5",
		Name:              "发布次数包",
		Description:       "临时多发供需",
		StandardPriceCent: 2500,
		SalePriceCent:     2500,
		SaleLabel:         "限时特价",
		Benefits:          model.VIPBenefitSnapshot{PublishQuota: 5},
	}}, nil
}

func (s *fakeFullAPIStore) ListAdminVIPPlans(ctx context.Context) ([]model.AdminVIPPlanConfig, error) {
	return []model.AdminVIPPlanConfig{{
		Code:              "monthly",
		Name:              "VIP 月卡",
		DurationMonths:    1,
		StandardPriceCent: 4900,
		Status:            "active",
		DisplayOrder:      10,
		Benefits:          model.VIPBenefitSnapshot{PublishPolicy: model.VIPPublishPolicyQuota, PublishQuota: 80, RefreshQuota: 30, TopVoucherCount: 3, TopDurationHours: 24},
		UpdatedAt:         "2026-07-10T10:00:00Z",
	}}, nil
}

func (s *fakeFullAPIStore) SaveAdminVIPPlan(ctx context.Context, input model.SaveAdminVIPPlanInput) (model.AdminVIPConfigSaveResult, error) {
	s.saveVIPPlanInput = input
	return model.AdminVIPConfigSaveResult{Code: input.Code, UpdatedAt: "2026-07-10T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) ListAdminQuotaPacks(ctx context.Context) ([]model.AdminQuotaPackConfig, error) {
	return []model.AdminQuotaPackConfig{{
		Code:              "publish_5",
		Name:              "发布次数包",
		Description:       "临时多发供需",
		StandardPriceCent: 2500,
		Status:            "active",
		DisplayOrder:      10,
		Benefits:          model.VIPBenefitSnapshot{PublishQuota: 5},
		UpdatedAt:         "2026-07-10T10:00:00Z",
	}}, nil
}

func (s *fakeFullAPIStore) SaveAdminQuotaPack(ctx context.Context, input model.SaveAdminQuotaPackInput) (model.AdminVIPConfigSaveResult, error) {
	s.saveQuotaPackInput = input
	return model.AdminVIPConfigSaveResult{Code: input.Code, UpdatedAt: "2026-07-10T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) ListAdminVIPPromotions(ctx context.Context) ([]model.AdminVIPPromotionConfig, error) {
	return []model.AdminVIPPromotionConfig{{
		Code:          "launch_monthly",
		PlanCode:      "monthly",
		PlanName:      "VIP 月卡",
		PromotionType: "launch",
		SalePriceCent: 1990,
		StartsAt:      "2026-07-10T00:00:00Z",
		Status:        "active",
		UpdatedAt:     "2026-07-10T10:00:00Z",
	}}, nil
}

func (s *fakeFullAPIStore) SaveAdminVIPPromotion(ctx context.Context, input model.SaveAdminVIPPromotionInput) (model.AdminVIPConfigSaveResult, error) {
	s.saveVIPPromotionInput = input
	return model.AdminVIPConfigSaveResult{Code: input.Code, UpdatedAt: "2026-07-10T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) GetMerchantVIPSummary(ctx context.Context, merchantID string) (model.MerchantVIPSummary, error) {
	return model.MerchantVIPSummary{
		MerchantID:            merchantID,
		Status:                model.VIPStatusActive,
		PlanCode:              "monthly",
		PlanName:              "VIP 月卡",
		PublishQuotaRemaining: 80,
		RefreshQuotaRemaining: 30,
		TopVoucherCount:       3,
	}, nil
}

func (s *fakeFullAPIStore) CreateVIPOrder(ctx context.Context, input model.CreateVIPOrderInput) (model.VIPOrder, error) {
	s.createVIPOrderInput = input
	return model.VIPOrder{
		ID:                "order-1",
		MerchantID:        input.MerchantID,
		UserID:            input.UserID,
		PlanCode:          input.PlanCode,
		PlanName:          "VIP 月卡",
		OutTradeNo:        "VIP202607090001",
		Status:            model.PaymentOrderStatusPending,
		Currency:          "CNY",
		StandardPriceCent: 4900,
		ActualPriceCent:   1990,
		PromotionCode:     "launch_monthly_first",
		Benefits:          model.VIPBenefitSnapshot{PublishPolicy: model.VIPPublishPolicyQuota, PublishQuota: 80, RefreshQuota: 30, TopVoucherCount: 3, TopDurationHours: 24},
	}, nil
}

func (s *fakeFullAPIStore) CreateQuotaPackOrder(ctx context.Context, input model.CreateQuotaPackOrderInput) (model.VIPOrder, error) {
	s.createQuotaPackOrderInput = input
	return model.VIPOrder{
		ID:                "order-2",
		MerchantID:        input.MerchantID,
		UserID:            input.UserID,
		ProductType:       model.VIPProductTypeQuotaPack,
		ProductCode:       input.PackCode,
		ProductName:       "发布次数包",
		OutTradeNo:        "VIP202607090002",
		Status:            model.PaymentOrderStatusPending,
		Currency:          "CNY",
		StandardPriceCent: 2500,
		ActualPriceCent:   2500,
		Benefits:          model.VIPBenefitSnapshot{PublishQuota: 5},
	}, nil
}

func (s *fakeFullAPIStore) GetVIPPaymentContext(ctx context.Context, input model.GetVIPPaymentContextInput) (model.VIPPaymentContext, error) {
	return model.VIPPaymentContext{
		OrderID:     input.OrderID,
		MerchantID:  input.MerchantID,
		UserID:      input.UserID,
		OpenID:      "dev:openid",
		Status:      model.PaymentOrderStatusPending,
		OutTradeNo:  "VIP202607090001",
		AmountTotal: 1990,
		Currency:    "CNY",
		PlanName:    "VIP 月卡",
	}, nil
}

func (s *fakeFullAPIStore) CreateVIPPaymentOrder(ctx context.Context, input model.CreateVIPPaymentOrderInput) (model.VIPPaymentOrder, error) {
	s.createVIPPaymentInput = input
	return model.VIPPaymentOrder{
		ID:          input.OrderID,
		OutTradeNo:  "VIP202607090001",
		AmountTotal: 1990,
		Currency:    "CNY",
		Status:      model.PaymentOrderStatusPending,
		PlanName:    "VIP 月卡",
	}, nil
}

func (s *fakeFullAPIStore) MarkVIPOrderPaid(ctx context.Context, input model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error) {
	s.markVIPOrderPaidInput = input
	return model.VIPPaymentResult{OrderID: "order-1", MerchantID: "merchant-1", Status: model.PaymentOrderStatusPaid}, nil
}

func (s *fakeFullAPIStore) GetResourceMetrics(ctx context.Context, resourceID string, from string, to string) (model.ResourceMetricsResult, error) {
	return model.ResourceMetricsResult{ResourceID: resourceID, Summary: model.ResourceMetricsSummary{DetailViewCount: 1}, Daily: []model.ResourceMetricDailyItem{{Date: "2026-06-28", DetailViewCount: 1}}}, nil
}

func (s *fakeFullAPIStore) GetMerchantMetricsSummary(ctx context.Context, merchantID string) (model.MerchantMetricsSummary, error) {
	return model.MerchantMetricsSummary{MerchantID: merchantID, PublishedResourceCount: 1, Last7Days: model.MerchantLast7DaysMetrics{DetailViewCount: 1}}, nil
}

func (s *fakeFullAPIStore) ListMessages(ctx context.Context, filter model.ListMessagesFilter) (model.ListMessagesResult, error) {
	s.messageFilter = filter
	return model.ListMessagesResult{Items: []model.MessageItem{{ID: "message-1", MessageType: "resource_review", Title: "审核通过", Content: "资源已发布", Status: "unread", CreatedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) ReadMessage(ctx context.Context, userID string, roleCodes []string, messageID string) (model.ReadMessageResult, error) {
	s.readMessageUserID = userID
	s.readMessageRoleCodes = roleCodes
	if len(roleCodes) == 1 {
		s.readMessageRoleCode = roleCodes[0]
	} else {
		s.readMessageRoleCode = ""
	}
	return model.ReadMessageResult{ID: messageID, Status: "read"}, nil
}

func (s *fakeFullAPIStore) ListManagedMerchantIDs(ctx context.Context, userID string) ([]string, error) {
	merchantIDs := make([]string, 0, len(s.managedMerchants))
	for merchantID, managed := range s.managedMerchants {
		if managed {
			merchantIDs = append(merchantIDs, merchantID)
		}
	}
	sort.Strings(merchantIDs)
	return merchantIDs, nil
}

func (s *fakeFullAPIStore) GetAdminDashboardOverview(ctx context.Context, cityCode string) (model.AdminDashboardOverview, error) {
	return model.AdminDashboardOverview{PendingResourceCount: 1, Tasks: []model.AdminDashboardTask{{Type: "资源审核", Title: "待审核资源", CityName: "织里", CreatedAt: "2026-06-28T10:00:00Z"}}}, nil
}

func (s *fakeFullAPIStore) ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]model.AdminResourceTypeConfig, error) {
	return []model.AdminResourceTypeConfig{{ID: "config-1", CityCode: "zhili", TypeCode: "stock_clearance", TypeName: "库存出售", RequiredFields: []string{"title"}, DefaultValidDays: 7, Status: "active"}}, nil
}

func (s *fakeFullAPIStore) CreateResourceTypeConfig(ctx context.Context, input model.CreateResourceTypeConfigInput) (model.CreateResourceTypeConfigResult, error) {
	return model.CreateResourceTypeConfigResult{ID: "config-2", Version: 1, UpdatedAt: "2026-07-14T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) UpdateResourceTypeConfig(ctx context.Context, configID string, patch model.ResourceTypeConfigPatch) (model.UpdateResourceTypeConfigResult, error) {
	return model.UpdateResourceTypeConfigResult{Version: 2, UpdatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) ListOperationLogs(ctx context.Context, filter model.OperationLogFilter) (model.ListOperationLogsResult, error) {
	return model.ListOperationLogsResult{Items: []model.OperationLogItem{{ID: "log-1", OperatorID: "user-1", OperatorRole: "platform_operator", ObjectType: "resource", ObjectID: "resource-1", Action: "resource_approve", CreatedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) ListAdminOperators(ctx context.Context, filter model.AdminOperatorFilter) (model.ListAdminOperatorsResult, error) {
	s.adminOperatorFilter = filter
	return model.ListAdminOperatorsResult{Items: []model.AdminOperatorItem{{OperatorID: "admin-1", LoginName: "operator", RealName: "运营", Status: "enabled", Roles: []string{"platform_operator"}, CreatedAt: "2026-07-16T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) CreateAdminOperator(ctx context.Context, input model.AdminOperatorInput) (model.AdminOperatorItem, error) {
	s.createAdminOperatorInput = input
	return model.AdminOperatorItem{OperatorID: "admin-created", LoginName: input.LoginName, RealName: input.RealName, Status: input.Status, Roles: input.Roles, CreatedAt: "2026-07-16T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) UpdateAdminOperator(ctx context.Context, input model.AdminOperatorInput) (model.AdminOperatorItem, error) {
	s.updateAdminOperatorInput = input
	return model.AdminOperatorItem{OperatorID: input.OperatorID, LoginName: input.LoginName, RealName: input.RealName, Status: input.Status, Roles: input.Roles, CreatedAt: "2026-07-16T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) UpdateAdminOperatorStatus(ctx context.Context, input model.AdminOperatorStatusInput) (model.AdminOperatorItem, error) {
	s.statusAdminOperatorInput = input
	return model.AdminOperatorItem{OperatorID: input.OperatorID, Status: input.Status, CreatedAt: "2026-07-16T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) GetAdminRoleModulePermissions(ctx context.Context, roleCode string) (model.AdminRoleModulePermission, error) {
	return model.AdminRoleModulePermission{RoleCode: roleCode, Modules: append([]string(nil), s.adminRoleModules...)}, nil
}

func (s *fakeFullAPIStore) UpdateAdminRoleModulePermissions(ctx context.Context, input model.AdminRoleModulePermissionInput) (model.AdminRoleModulePermission, error) {
	s.adminRoleModuleInput = input
	s.adminRoleModules = append([]string(nil), input.Modules...)
	return model.AdminRoleModulePermission{RoleCode: input.RoleCode, Modules: append([]string(nil), input.Modules...)}, nil
}

func (s *fakeFullAPIStore) ListSearchLogs(ctx context.Context, filter model.SearchLogFilter) (model.ListSearchLogsResult, error) {
	return model.ListSearchLogsResult{Items: []model.SearchLogItem{{ID: "search-1", CityCode: filter.CityCode, CityName: "织里", Keyword: "童装库存", ResultCount: 0, CreatedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) MarkExpiredResources(ctx context.Context) ([]model.LifecycleResource, error) {
	return []model.LifecycleResource{{ID: "resource-expired", MerchantID: "merchant-1", Title: "过期资源"}}, nil
}

func (s *fakeFullAPIStore) ListResourcesExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error) {
	return []model.LifecycleResource{{ID: "resource-expiring", MerchantID: "merchant-1", Title: "即将过期资源"}}, nil
}

func (s *fakeFullAPIStore) MarkExpiredVerifications(ctx context.Context) ([]model.LifecycleResource, error) {
	return []model.LifecycleResource{{ID: "verification-expired", MerchantID: "merchant-1", Title: "源头工厂认证"}}, nil
}

func (s *fakeFullAPIStore) ListVerificationsExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error) {
	return []model.LifecycleResource{{ID: "verification-expiring", MerchantID: "merchant-1", Title: "源头工厂认证"}}, nil
}

func (s *fakeFullAPIStore) CreateMessage(ctx context.Context, input model.CreateMessageInput) (model.CreateMessageResult, error) {
	return model.CreateMessageResult{ID: "message-task", Created: true}, nil
}
