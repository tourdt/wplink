package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
)

func TestAPIRouterRequiresAdminTokenWhenConfigured(t *testing.T) {
	tokenService := &fakeAdminTokenService{subject: session.AdminTokenSubject{UserID: "admin-1", Roles: []string{"platform_operator"}}}
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

func TestAPIRouterUsesTokenSubjectForPrivateUserRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	demandRec := httptest.NewRecorder()
	demandReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/purchase-demands?userId=attacker&page=1&pageSize=20", nil)
	demandReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(demandRec, demandReq)
	decodeEnvelopeData(t, demandRec, http.StatusOK)
	if store.myDemandUserID != "user-1" {
		t.Fatalf("myDemandUserID = %q, want token user", store.myDemandUserID)
	}

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

func TestAPIRouterUsesTokenSubjectWhenCreatingDemand(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-demands", strings.NewReader(`{
		"userId":"attacker",
		"cityCode":"zhili",
		"demandType":"inventory",
		"title":"找童装库存",
		"category":"童装",
		"contact":{"name":"王采购","phone":"18800000005"}
	}`))
	req.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(rec, req)
	decodeEnvelopeData(t, rec, http.StatusOK)
	if store.createDemandInput.UserID != "user-1" {
		t.Fatalf("create demand userID = %q, want token user", store.createDemandInput.UserID)
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
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
	if store.submitVerificationInput.ApplicantUserID != "user-1" {
		t.Fatalf("applicantUserID = %q, want token user", store.submitVerificationInput.ApplicantUserID)
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

func TestAPIRouterRunsRemainingDomainRoutes(t *testing.T) {
	store := newFakeFullAPIStore()
	router := NewAPIRouter(store)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create merchant", method: http.MethodPost, path: "/api/v1/merchants", body: `{"cityCode":"zhili","name":"织里云仓","merchantType":"stockist","mainCategories":["童装"],"contactName":"周经理","contactPhone":"18800000002"}`},
		{name: "get merchant", method: http.MethodGet, path: "/api/v1/merchants/merchant-1"},
		{name: "update merchant", method: http.MethodPatch, path: "/api/v1/merchants/merchant-1", body: `{"mainCategories":["童装"],"description":"更新简介","images":["https://example.com/a.jpg"]}`},
		{name: "create demand", method: http.MethodPost, path: "/api/v1/purchase-demands", body: `{"userId":"user-1","cityCode":"zhili","demandType":"inventory","title":"找童装库存","category":"童装","contact":{"name":"王采购","phone":"18800000005"}}`},
		{name: "list my demands", method: http.MethodGet, path: "/api/v1/me/purchase-demands?userId=user-1"},
		{name: "home banners", method: http.MethodGet, path: "/api/v1/home/banners?cityCode=zhili"},
		{name: "topic resources", method: http.MethodGet, path: "/api/v1/topics/topic-1/resources?cityCode=zhili"},
		{name: "validate webview", method: http.MethodPost, path: "/api/v1/webview/validate", body: `{"url":"https://www.wplink.cn/activity"}`},
		{name: "submit verification", method: http.MethodPost, path: "/api/v1/merchants/merchant-1/verifications", body: `{"applicantUserId":"user-1","verificationType":"stockist","businessName":"织里云仓"}`},
		{name: "latest verification", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/verifications/latest"},
		{name: "list entitlements", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/entitlements"},
		{name: "list top vouchers", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/top-vouchers"},
		{name: "redeem top voucher", method: http.MethodPost, path: "/api/v1/top-vouchers/voucher-1/redeem", body: `{"resourceId":"resource-1"}`},
		{name: "resource metrics", method: http.MethodGet, path: "/api/v1/resources/resource-1/metrics"},
		{name: "merchant metrics", method: http.MethodGet, path: "/api/v1/merchants/merchant-1/metrics/summary"},
		{name: "list messages", method: http.MethodGet, path: "/api/v1/messages?userId=user-1"},
		{name: "read message", method: http.MethodPost, path: "/api/v1/messages/message-1/read", body: `{"userId":"user-1"}`},
		{name: "dashboard", method: http.MethodGet, path: "/api/v1/admin/dashboard/overview?cityCode=zhili"},
		{name: "list admin merchants", method: http.MethodGet, path: "/api/v1/admin/merchants?cityCode=zhili"},
		{name: "list admin demands", method: http.MethodGet, path: "/api/v1/admin/purchase-demands"},
		{name: "get admin demand", method: http.MethodGet, path: "/api/v1/admin/purchase-demands/demand-1"},
		{name: "update admin demand", method: http.MethodPatch, path: "/api/v1/admin/purchase-demands/demand-1/status", body: `{"status":"matching"}`},
		{name: "list admin banners", method: http.MethodGet, path: "/api/v1/admin/banner-topics?cityCode=zhili"},
		{name: "create admin banner", method: http.MethodPost, path: "/api/v1/admin/banner-topics", body: `{"cityCode":"zhili","kind":"banner","title":"现货活动","jumpType":"demand","jumpTarget":"/pages/demand/index","status":"active"}`},
		{name: "update admin banner", method: http.MethodPatch, path: "/api/v1/admin/banner-topics/banner-1", body: `{"cityCode":"zhili","kind":"banner","title":"现货活动","jumpType":"demand","jumpTarget":"/pages/demand/index","status":"active"}`},
		{name: "list resource configs", method: http.MethodGet, path: "/api/v1/admin/resource-type-configs?cityCode=zhili"},
		{name: "update resource config", method: http.MethodPatch, path: "/api/v1/admin/resource-type-configs/config-1", body: `{"fieldSchema":{},"requiredFields":["title"],"filterFields":["category"],"displayTemplate":{},"reviewRules":{},"sortWeights":{},"messageRules":{},"defaultValidDays":7,"status":"active"}`},
		{name: "list pending verifications", method: http.MethodGet, path: "/api/v1/admin/verifications/pending"},
		{name: "review verification", method: http.MethodPost, path: "/api/v1/admin/verifications/verification-1/review", body: `{"reviewerId":"user-1","action":"approve"}`},
		{name: "grant entitlement", method: http.MethodPost, path: "/api/v1/admin/merchants/merchant-1/entitlements", body: `{"operatorId":"user-1","entitlementType":"publish_quota","sourceType":"manual","totalAmount":3,"reason":"测试发放"}`},
		{name: "create match case", method: http.MethodPost, path: "/api/v1/admin/match-cases", body: `{"operatorId":"user-1","purchaseDemandId":"demand-1","resourceIds":["resource-1"],"participantMerchantIds":["merchant-1"]}`},
		{name: "list match cases", method: http.MethodGet, path: "/api/v1/admin/match-cases"},
		{name: "update match case", method: http.MethodPatch, path: "/api/v1/admin/match-cases/match-1/status", body: `{"operatorId":"user-1","status":"contacted"}`},
		{name: "add match resources", method: http.MethodPost, path: "/api/v1/admin/match-cases/match-1/resources", body: `{"operatorId":"user-1","resourceIds":["resource-1"]}`},
		{name: "add match participants", method: http.MethodPost, path: "/api/v1/admin/match-cases/match-1/participants", body: `{"operatorId":"user-1","participantMerchantIds":["merchant-1"]}`},
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
}

func newFakeFullAPIStore() *fakeFullAPIStore {
	return &fakeFullAPIStore{
		fakeResourceAPIStore: fakeResourceAPIStore{
			publishConfig:  model.ResourcePublishConfig{ID: "config-1", TypeCode: "inventory", RequiredFields: []string{"merchantId", "cityCode", "typeCode", "title", "category", "contactName", "contactPhone"}, DefaultValidDays: 7},
			merchantStatus: model.MerchantStatusActive,
		},
	}
}

type fakeFullAPIStore struct {
	fakeResourceAPIStore
	createDemandInput       model.CreateDemandInput
	myDemandUserID          string
	submitVerificationInput model.SubmitVerificationInput
	messageFilter           model.ListMessagesFilter
	readMessageUserID       string
	readMessageRoleCode     string
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

func (s *fakeFullAPIStore) CreateMerchant(ctx context.Context, input model.CreateMerchantInput) (model.CreateMerchantResult, error) {
	return model.CreateMerchantResult{ID: "merchant-1", Name: input.Name, VerificationStatus: "unverified", Status: model.MerchantStatusActive}, nil
}

func (s *fakeFullAPIStore) GetMerchantDetail(ctx context.Context, merchantID string) (model.MerchantDetail, error) {
	return model.MerchantDetail{ID: merchantID, Name: "织里云仓", MerchantType: "stockist", CityCode: "zhili", MainCategories: []string{"童装"}, VerificationStatus: "verified", ContactName: "周经理", PhoneMasked: "188****0002", PublishedCount: 1}, nil
}

func (s *fakeFullAPIStore) UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error) {
	return "2026-06-28T10:00:00Z", nil
}

func (s *fakeFullAPIStore) ListMerchants(ctx context.Context, filter model.ListMerchantsFilter) (model.ListMerchantsResult, error) {
	return model.ListMerchantsResult{Items: []model.MerchantListItem{{ID: "merchant-1", Name: "织里云仓", MerchantType: "stockist", VerificationStatus: "verified", Status: model.MerchantStatusActive}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) CreateDemand(ctx context.Context, input model.CreateDemandInput) (model.CreateDemandResult, error) {
	s.createDemandInput = input
	return model.CreateDemandResult{ID: "demand-1", Status: "pending"}, nil
}

func (s *fakeFullAPIStore) ListMyDemands(ctx context.Context, userID string, filter model.ListDemandsFilter) (model.ListDemandsResult, error) {
	s.myDemandUserID = userID
	return s.ListDemands(ctx, filter)
}

func (s *fakeFullAPIStore) ListDemands(ctx context.Context, filter model.ListDemandsFilter) (model.ListDemandsResult, error) {
	return model.ListDemandsResult{Items: []model.DemandListItem{{ID: "demand-1", Title: "找童装库存", DemandType: "inventory", Category: "童装", ContactName: "王采购", Status: "pending", CreatedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) GetDemand(ctx context.Context, demandID string) (model.DemandDetail, error) {
	return model.DemandDetail{ID: demandID, Title: "找童装库存", DemandType: "inventory", Category: "童装", ContactName: "王采购", ContactPhone: "18800000005", Status: "pending", CreatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) UpdateDemandStatus(ctx context.Context, demandID string, status string) (model.UpdateDemandStatusResult, error) {
	return model.UpdateDemandStatusResult{ID: demandID, Status: status}, nil
}

func (s *fakeFullAPIStore) ListBannerTopics(ctx context.Context, filter model.BannerTopicFilter) ([]model.BannerTopicConfig, error) {
	return []model.BannerTopicConfig{{ID: "banner-1", CityCode: "zhili", Kind: "banner", Title: "现货活动", JumpType: "demand", JumpTarget: "/pages/demand/index", Status: "active", UpdatedAt: "2026-06-28T10:00:00Z"}}, nil
}

func (s *fakeFullAPIStore) ListActiveBannerTopics(ctx context.Context, filter model.BannerTopicFilter) ([]model.BannerTopicConfig, error) {
	return s.ListBannerTopics(ctx, filter)
}

func (s *fakeFullAPIStore) GetActiveTopic(ctx context.Context, topicID string, cityCode string) (model.BannerTopicConfig, error) {
	return model.BannerTopicConfig{ID: topicID, CityCode: cityCode, Kind: "topic", Title: "专题", TypeScope: []string{"inventory"}, JumpType: "demand", JumpTarget: "/pages/demand/index", Tags: []string{"现货"}, Status: "active"}, nil
}

func (s *fakeFullAPIStore) CreateBannerTopic(ctx context.Context, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error) {
	return model.SaveBannerTopicResult{ID: "banner-1", UpdatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) UpdateBannerTopic(ctx context.Context, configID string, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error) {
	return model.SaveBannerTopicResult{ID: configID, UpdatedAt: "2026-06-28T10:00:00Z"}, nil
}

func (s *fakeFullAPIStore) SubmitVerification(ctx context.Context, input model.SubmitVerificationInput) (model.VerificationResult, error) {
	s.submitVerificationInput = input
	return model.VerificationResult{ID: "verification-1", Status: "pending"}, nil
}

func (s *fakeFullAPIStore) GetLatestVerification(ctx context.Context, merchantID string) (model.VerificationBrief, error) {
	return model.VerificationBrief{ID: "verification-1", VerificationType: "stockist", Status: "pending"}, nil
}

func (s *fakeFullAPIStore) ListPendingVerifications(ctx context.Context, filter model.PendingVerificationsFilter) (model.ListPendingVerificationsResult, error) {
	return model.ListPendingVerificationsResult{Items: []model.PendingVerificationItem{{ID: "verification-1", MerchantID: "merchant-1", MerchantName: "织里云仓", VerificationType: "stockist", Status: "pending", SubmittedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) ReviewVerification(ctx context.Context, input model.ReviewVerificationInput) (model.ReviewVerificationResult, error) {
	return model.ReviewVerificationResult{ID: input.VerificationID, Status: "verified"}, nil
}

func (s *fakeFullAPIStore) ListMerchantEntitlements(ctx context.Context, merchantID string) ([]model.MerchantEntitlement, error) {
	return []model.MerchantEntitlement{{Type: "publish_quota", SourceType: "manual", TotalAmount: 3, RemainingAmount: 2}}, nil
}

func (s *fakeFullAPIStore) ListTopVouchers(ctx context.Context, merchantID string) ([]model.TopVoucher, error) {
	return []model.TopVoucher{{ID: "voucher-1", Status: "unused", TopDurationHours: 24}}, nil
}

func (s *fakeFullAPIStore) RedeemTopVoucher(ctx context.Context, voucherID string, resourceID string) (model.RedeemTopVoucherResult, error) {
	return model.RedeemTopVoucherResult{VoucherID: voucherID, ResourceID: resourceID, Status: "used"}, nil
}

func (s *fakeFullAPIStore) GrantMerchantEntitlement(ctx context.Context, input model.GrantEntitlementInput) (model.GrantEntitlementResult, error) {
	return model.GrantEntitlementResult{ID: "entitlement-1"}, nil
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

func (s *fakeFullAPIStore) ReadMessage(ctx context.Context, userID string, roleCode string, messageID string) (model.ReadMessageResult, error) {
	s.readMessageUserID = userID
	s.readMessageRoleCode = roleCode
	return model.ReadMessageResult{ID: messageID, Status: "read"}, nil
}

func (s *fakeFullAPIStore) GetAdminDashboardOverview(ctx context.Context, cityCode string) (model.AdminDashboardOverview, error) {
	return model.AdminDashboardOverview{PendingResourceCount: 1, Tasks: []model.AdminDashboardTask{{Type: "资源审核", Title: "待审核资源", CityName: "织里", CreatedAt: "2026-06-28T10:00:00Z"}}}, nil
}

func (s *fakeFullAPIStore) ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]model.AdminResourceTypeConfig, error) {
	return []model.AdminResourceTypeConfig{{ID: "config-1", CityCode: "zhili", TypeCode: "inventory", TypeName: "库存", RequiredFields: []string{"title"}, DefaultValidDays: 7, Status: "active"}}, nil
}

func (s *fakeFullAPIStore) UpdateResourceTypeConfig(ctx context.Context, configID string, patch model.ResourceTypeConfigPatch) (string, error) {
	return "2026-06-28T10:00:00Z", nil
}

func (s *fakeFullAPIStore) CreateMatchCase(ctx context.Context, input model.CreateMatchCaseInput) (model.MatchCaseResult, error) {
	return model.MatchCaseResult{ID: "match-1", Status: model.MatchCaseStatusOpen}, nil
}

func (s *fakeFullAPIStore) ListMatchCases(ctx context.Context, filter model.ListMatchCasesFilter) (model.ListMatchCasesResult, error) {
	return model.ListMatchCasesResult{Items: []model.MatchCaseListItem{{ID: "match-1", PurchaseDemandID: "demand-1", DemandTitle: "找童装库存", Status: model.MatchCaseStatusOpen, CreatedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
}

func (s *fakeFullAPIStore) UpdateMatchCaseStatus(ctx context.Context, input model.UpdateMatchCaseStatusInput) (model.MatchCaseResult, error) {
	return model.MatchCaseResult{ID: input.MatchCaseID, Status: input.Status}, nil
}

func (s *fakeFullAPIStore) AddMatchCaseResources(ctx context.Context, input model.AddMatchCaseResourcesInput) error {
	return nil
}

func (s *fakeFullAPIStore) AddMatchCaseParticipants(ctx context.Context, input model.AddMatchCaseParticipantsInput) error {
	return nil
}

func (s *fakeFullAPIStore) ListOperationLogs(ctx context.Context, filter model.OperationLogFilter) (model.ListOperationLogsResult, error) {
	return model.ListOperationLogsResult{Items: []model.OperationLogItem{{ID: "log-1", OperatorID: "user-1", OperatorRole: "platform_operator", ObjectType: "resource", ObjectID: "resource-1", Action: "resource_approve", CreatedAt: "2026-06-28T10:00:00Z"}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
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

func (s *fakeFullAPIStore) CreateMessage(ctx context.Context, input model.CreateMessageInput) (model.CreateMessageResult, error) {
	return model.CreateMessageResult{ID: "message-task"}, nil
}
