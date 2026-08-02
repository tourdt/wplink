package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"
)

func TestAPIRouterLogsInAdmin(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{}, WithAdminLoginService(fakeAdminLoginService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["token"] != "admin-token" || data["operatorId"] != "operator-1" {
		t.Fatalf("login data = %#v, want token and operatorId", data)
	}
}

func TestAPIRouterAdminLoginHidesRawInternalError(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{}, WithAdminLoginService(rawErrorAdminLoginService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s, want %d", rec.Code, rec.Body.String(), http.StatusUnauthorized)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["msg"] != "登录失败，请稍后重试" {
		t.Fatalf("msg = %#v, want safe login failure message", body["msg"])
	}
}

func TestMerchantMapEventRouterIsOnlyRegisteredForSupportedStore(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("location_view")))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s, want route omitted for unsupported store", rec.Code, rec.Body.String())
	}
}

func TestMerchantMapEventRouterRecordsAnonymousEvent(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("location_view")))
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["recorded"] != true {
		t.Fatalf("recorded = %#v, want true", data["recorded"])
	}
	if store.mapEventInput.UserID != "" || store.mapEventInput.MerchantID != "101" {
		t.Fatalf("map event input = %#v, want anonymous event for merchant 101", store.mapEventInput)
	}
}

func TestMerchantMapEventRouterUsesTokenSubject(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("location_view")))
	req.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["recorded"] != true || store.mapEventInput.UserID != "user-1" {
		t.Fatalf("data/input = %#v/%#v, want recorded event attributed to token user", data, store.mapEventInput)
	}
}

func TestMerchantMapEventRouterRejectsCredentialWhenTokenServiceMissing(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("location_view")))
	req.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(rec, req)

	body := decodeEnvelope(t, rec, http.StatusUnauthorized)
	if body["errorCode"] != errx.CodeUnauthorized || body["msg"] != "登录已过期，请重新登录" {
		t.Fatalf("body = %#v, want expired login error when token service is unavailable", body)
	}
	if store.mapEventCalled {
		t.Fatal("store called when request credential could not be verified")
	}
}

func TestMerchantMapEventRouterRejectsExpiredTokenBeforeStore(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("location_view")))
	req.Header.Set("Authorization", "Bearer expired-token")
	router.ServeHTTP(rec, req)

	body := decodeEnvelope(t, rec, http.StatusUnauthorized)
	if body["errorCode"] != errx.CodeUnauthorized || body["msg"] != "登录已过期，请重新登录" {
		t.Fatalf("body = %#v, want expired login error", body)
	}
	if store.mapEventCalled {
		t.Fatal("store called after token validation failed")
	}
}

func TestMerchantMapEventRouterRejectsUnknownEvent(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("merchant_open")))
	router.ServeHTTP(rec, req)

	body := decodeEnvelope(t, rec, http.StatusBadRequest)
	if body["errorCode"] != errx.CodeValidationFailed || body["msg"] != "地图行为类型无效" {
		t.Fatalf("body = %#v, want invalid map event type", body)
	}
	if store.mapEventCalled {
		t.Fatal("store called for invalid event type")
	}
}

func TestMerchantMapEventRouterHidesDatabaseError(t *testing.T) {
	store := &fakeResourceAPIStore{mapEventErr: errors.New("pq: relation merchant_map_events does not exist")}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("location_view")))
	router.ServeHTTP(rec, req)

	body := decodeEnvelope(t, rec, http.StatusInternalServerError)
	if body["errorCode"] != errx.CodeInternalError || body["msg"] != "地图行为记录失败，请稍后重试" {
		t.Fatalf("body = %#v, want safe map event persistence error", body)
	}
	if strings.Contains(rec.Body.String(), "merchant_map_events") || strings.Contains(rec.Body.String(), "pq:") {
		t.Fatalf("response leaked database detail: %s", rec.Body.String())
	}
}

func TestMerchantMapEventRouterRejectsPayloadBeyondFourKiBBeforeStore(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(strings.Repeat("x", 4097)))
	router.ServeHTTP(rec, req)

	body := decodeEnvelope(t, rec, http.StatusBadRequest)
	if body["errorCode"] != errx.CodeValidationFailed || body["msg"] != "地图行为请求内容过大" {
		t.Fatalf("body = %#v, want safe oversized map event error", body)
	}
	if store.mapEventCalled {
		t.Fatal("store called for an oversized map event request")
	}
}

func TestMerchantMapEventRouterRateLimitsOnlyAfterSixtyRequestsForClientAndVisitor(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store)

	for attempt := 1; attempt <= 61; attempt++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(validMerchantMapEventJSON("location_view")))
		req.RemoteAddr = "198.51.100.7:4321"
		router.ServeHTTP(rec, req)

		if attempt <= 60 {
			decodeEnvelopeData(t, rec, http.StatusOK)
			continue
		}
		body := decodeEnvelope(t, rec, http.StatusTooManyRequests)
		if body["errorCode"] != errx.CodeRateLimited || body["msg"] != "操作频繁，请稍后再试" {
			t.Fatalf("body = %#v, want rate limited response", body)
		}
	}
	if store.mapEventCalls != 60 {
		t.Fatalf("store calls = %d, want exactly 60", store.mapEventCalls)
	}
}

func validMerchantMapEventJSON(eventType string) string {
	return `{"merchantId":"101","visitorKey":"visitor-1","sessionId":"session-1","eventType":"` + eventType + `","source":"merchant_location"}`
}

func TestResourceAPIRouterRunsPublishReviewSearchContactFlow(t *testing.T) {
	store := &fakeResourceAPIStore{
		merchantStatus: model.MerchantStatusActive,
		publishConfig: model.ResourcePublishConfig{
			ID:               "config-1",
			TypeCode:         "inventory",
			RequiredFields:   []string{"merchantId", "cityCode", "typeCode", "title", "category", "contactName", "contactPhone"},
			DefaultValidDays: 7,
		},
	}
	router := NewAPIRouter(store)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{
		"merchantId":"merchant-1",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"女童春款卫衣库存",
		"category":"童装卫衣",
		"quantityText":"3800件",
		"priceText":"18元/件",
		"description":"可拿样",
		"tags":["急清","支持看货"],
		"contact":{"name":"周经理","phone":"18800000002","wechat":"stock-demo"}
	}`))
	router.ServeHTTP(createRec, createReq)

	createData := decodeEnvelopeData(t, createRec, http.StatusOK)
	if createData["id"] != "resource-1" || createData["status"] != model.ResourceStatusPending {
		t.Fatalf("create data = %#v, want resource-1 pending", createData)
	}
	if store.created.Title != "女童春款卫衣库存" || store.created.ContactName != "周经理" {
		t.Fatalf("created input = %#v, want mapped publish fields", store.created)
	}
	if store.created.ConsumePublishQuota {
		t.Fatalf("consumePublishQuota = true, want publish quota consumed only after auto audit publish")
	}
	if len(store.created.Tags) != 2 || store.created.Tags[0] != "急清" || store.created.Tags[1] != "支持看货" {
		t.Fatalf("created tags = %#v, want mapped publish tags", store.created.Tags)
	}

	submitRec := httptest.NewRecorder()
	submitReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/submit", nil)
	router.ServeHTTP(submitRec, submitReq)
	submitData := decodeEnvelopeData(t, submitRec, http.StatusOK)
	if submitData["id"] != "resource-1" || submitData["status"] != model.ResourceStatusPending {
		t.Fatalf("submit data = %#v, want resource-1 pending", submitData)
	}

	pendingRec := httptest.NewRecorder()
	pendingReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/resources/pending?cityCode=zhili&typeCode=inventory", nil)
	router.ServeHTTP(pendingRec, pendingReq)
	pendingData := decodeEnvelopeData(t, pendingRec, http.StatusOK)
	if store.pendingFilter.CityCode != "zhili" || store.pendingFilter.TypeCode != "inventory" {
		t.Fatalf("pending filter = %#v, want zhili inventory", store.pendingFilter)
	}
	if len(pendingData["items"].([]interface{})) != 1 {
		t.Fatalf("pending items = %#v, want one item", pendingData["items"])
	}

	adminListRec := httptest.NewRecorder()
	adminListReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/resources?cityCode=zhili&status=published&page=2&pageSize=5", nil)
	router.ServeHTTP(adminListRec, adminListReq)
	_ = decodeEnvelopeData(t, adminListRec, http.StatusOK)
	if store.pendingFilter.CityCode != "zhili" || store.pendingFilter.Status != model.ResourceStatusPublished || store.pendingFilter.Page != 2 || store.pendingFilter.PageSize != 5 {
		t.Fatalf("admin resource filter = %#v, want published resources", store.pendingFilter)
	}

	reviewRec := httptest.NewRecorder()
	reviewReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/resources/resource-1/review", strings.NewReader(`{"action":"approve"}`))
	router.ServeHTTP(reviewRec, reviewReq)
	reviewData := decodeEnvelopeData(t, reviewRec, http.StatusOK)
	if reviewData["status"] != model.ResourceStatusPublished {
		t.Fatalf("review data = %#v, want published", reviewData)
	}
	if store.reviewAction != "approve" {
		t.Fatalf("review action = %q, want approve", store.reviewAction)
	}

	searchRec := httptest.NewRecorder()
	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/resource-search?cityCode=zhili&typeCode=inventory&keyword=卫衣&tags=急清,支持看货&page=2&pageSize=5", nil)
	router.ServeHTTP(searchRec, searchReq)
	searchData := decodeEnvelopeData(t, searchRec, http.StatusOK)
	if store.listFilter.Keyword != "卫衣" || store.listFilter.Page != 2 || store.listFilter.PageSize != 5 {
		t.Fatalf("list filter = %#v, want query values", store.listFilter)
	}
	if len(store.listFilter.Tags) != 2 || store.listFilter.Tags[0] != "急清" || store.listFilter.Tags[1] != "支持看货" {
		t.Fatalf("list filter tags = %#v, want query tag filters", store.listFilter.Tags)
	}
	if store.searchLog.Keyword != "卫衣" || store.searchLog.ResultCount != 1 {
		t.Fatalf("search log = %#v, want keyword and result count", store.searchLog)
	}
	logTags, ok := store.searchLog.Filters["tags"].([]string)
	if !ok || len(logTags) != 2 || logTags[0] != "急清" || logTags[1] != "支持看货" {
		t.Fatalf("search log tags = %#v, want query tag filters", store.searchLog.Filters["tags"])
	}
	if len(searchData["items"].([]interface{})) != 1 {
		t.Fatalf("search items = %#v, want one item", searchData["items"])
	}
	searchItem := searchData["items"].([]interface{})[0].(map[string]interface{})
	if tags := searchItem["tags"].([]interface{}); len(tags) != 2 || tags[0] != "急清" || tags[1] != "支持看货" {
		t.Fatalf("search item tags = %#v, want resource tags in list response", searchItem["tags"])
	}
	if searchItem["coverUrl"] != "https://img.example.com/list-cover.jpg" {
		t.Fatalf("search item coverUrl = %#v, want list cover", searchItem["coverUrl"])
	}

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources/resource-1", nil)
	router.ServeHTTP(detailRec, detailReq)
	detailData := decodeEnvelopeData(t, detailRec, http.StatusOK)
	if detailData["id"] != "resource-1" || detailData["title"] != "女童春款卫衣库存" {
		t.Fatalf("detail data = %#v, want resource detail", detailData)
	}

	detailViewRec := httptest.NewRecorder()
	detailViewReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/detail-view", nil)
	router.ServeHTTP(detailViewRec, detailViewReq)
	detailViewData := decodeEnvelopeData(t, detailViewRec, http.StatusOK)
	if detailViewData["message"] != "浏览行为已记录" {
		t.Fatalf("detail view data = %#v, want message", detailViewData)
	}
	if store.metricDelta.DetailViewCount != 1 {
		t.Fatalf("metric delta after detail view = %#v, want detail view count", store.metricDelta)
	}

	contactRec := httptest.NewRecorder()
	contactReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"action":"phone"}`))
	contactReq.Header.Set("Authorization", "Bearer user-token")
	contactRouter := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))
	contactRouter.ServeHTTP(contactRec, contactReq)
	contactData := decodeEnvelopeData(t, contactRec, http.StatusOK)
	if contactData["message"] != "电话已解锁" || contactData["phone"] != "18800000002" {
		t.Fatalf("contact data = %#v, want message", contactData)
	}
	if store.contactInput.Action != "phone" || store.metricDelta.PhoneClickCount != 1 {
		t.Fatalf("contact input = %#v metric = %#v, want phone metric", store.contactInput, store.metricDelta)
	}

	wechatRec := httptest.NewRecorder()
	wechatReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"action":"wechat"}`))
	wechatReq.Header.Set("Authorization", "Bearer user-token")
	contactRouter.ServeHTTP(wechatRec, wechatReq)
	wechatData := decodeEnvelopeData(t, wechatRec, http.StatusOK)
	if wechatData["message"] != "微信号已解锁" || wechatData["wechat"] != "stock-demo" {
		t.Fatalf("wechat data = %#v, want message", wechatData)
	}
	if store.contactInput.Action != "wechat" || store.metricDelta.WechatCopyCount != 1 {
		t.Fatalf("contact input = %#v metric = %#v, want wechat metric", store.contactInput, store.metricDelta)
	}

	metricsRec := httptest.NewRecorder()
	metricsReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources/resource-1/metrics", nil)
	router.ServeHTTP(metricsRec, metricsReq)
	metricsData := decodeEnvelopeData(t, metricsRec, http.StatusOK)
	summary := metricsData["summary"].(map[string]interface{})
	if summary["detailViewCount"] != float64(1) || summary["phoneClickCount"] != float64(1) || summary["wechatCopyCount"] != float64(1) {
		t.Fatalf("metrics summary = %#v, want detail, phone and wechat counts", summary)
	}

	myRec := httptest.NewRecorder()
	myReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/resources?merchantId=merchant-1&status=published", nil)
	router.ServeHTTP(myRec, myReq)
	myData := decodeEnvelopeData(t, myRec, http.StatusOK)
	if store.myFilter.MerchantID != "merchant-1" || store.myFilter.Status != "published" {
		t.Fatalf("my filter = %#v, want merchant published", store.myFilter)
	}
	myItems := myData["items"].([]interface{})
	if len(myItems) != 1 {
		t.Fatalf("my items = %#v, want one item", myData["items"])
	}
	myItem := myItems[0].(map[string]interface{})
	myMetrics := myItem["metrics"].(map[string]interface{})
	if myMetrics["detailViewCount"] != float64(1) || myMetrics["phoneClickCount"] != float64(1) || myMetrics["wechatCopyCount"] != float64(1) {
		t.Fatalf("my metrics = %#v, want updated loop metrics", myMetrics)
	}
	if myItem["coverUrl"] != "https://img.example.com/resource-cover.jpg" {
		t.Fatalf("my coverUrl = %#v, want compact list cover", myItem["coverUrl"])
	}
}

func TestResourceAPIRouterHandlesWechatContentAuditMediaCallback(t *testing.T) {
	store := &fakeResourceAPIStore{
		auditCompletion: model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
	}
	verifier := &fakeWechatCallbackVerifier{}
	router := NewAPIRouter(store, WithContentAuditCallbackVerifier(verifier, "wx-app"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/content-audit/media-callback?signature=sig&timestamp=1784971200&nonce=nonce", strings.NewReader(`{
		"appid":"wx-app",
		"trace_id":"trace-media",
		"errcode":0,
		"result":{"suggest":"pass","label":100}
	}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "success" {
		t.Fatalf("status=%d body=%q, want wechat success acknowledgement", rec.Code, rec.Body.String())
	}
	if store.completedAuditInput.TraceID != "trace-media" || store.publishedAuditResourceID != "resource-1" {
		t.Fatalf("completedAuditInput = %#v publishedAuditResourceID = %q, want completed and published", store.completedAuditInput, store.publishedAuditResourceID)
	}
	if verifier.remember != true {
		t.Fatal("callback verifier remember = false, want replay protection enabled")
	}
}

func TestResourceAPIRouterRejectsUnauthenticatedWechatContentAuditCallback(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store, WithContentAuditCallbackVerifier(
		&fakeWechatCallbackVerifier{err: errors.New("invalid signature")},
		"wx-app",
	))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/content-audit/media-callback", strings.NewReader(`{"appid":"wx-app","trace_id":"trace-media"}`))
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status=%d body=%s, want rejected callback", rec.Code, rec.Body.String())
	}
	if store.completedAuditInput.TraceID != "" {
		t.Fatalf("completedAuditInput = %#v, callback must not reach store", store.completedAuditInput)
	}
}

func TestResourceAPIRouterHandlesWechatCallbackURLVerification(t *testing.T) {
	verifier := &fakeWechatCallbackVerifier{}
	router := NewAPIRouter(&fakeResourceAPIStore{}, WithContentAuditCallbackVerifier(verifier, "wx-app"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wechat/content-audit/media-callback?signature=sig&timestamp=1784971200&nonce=nonce&echostr=challenge", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "challenge" {
		t.Fatalf("status=%d body=%q, want callback challenge", rec.Code, rec.Body.String())
	}
	if verifier.remember {
		t.Fatal("URL verification must not consume replay key")
	}
}

type fakeWechatCallbackVerifier struct {
	err      error
	remember bool
}

func (v *fakeWechatCallbackVerifier) Verify(signature string, timestamp string, nonce string, remember bool) error {
	v.remember = remember
	return v.err
}

func TestResourceAPIRouterRequiresManagedMerchantWhenTokenConfigured(t *testing.T) {
	store := &fakeResourceAPIStore{
		publishConfig: model.ResourcePublishConfig{
			ID:               "config-1",
			TypeCode:         "inventory",
			RequiredFields:   []string{"merchantId", "cityCode", "typeCode", "title", "category", "contactName", "contactPhone"},
			DefaultValidDays: 7,
		},
		merchantStatus:   model.MerchantStatusActive,
		managedMerchants: map[string]bool{"merchant-1": true},
	}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{
		"merchantId":"merchant-2",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"女童春款卫衣库存",
		"category":"童装卫衣",
		"description":"库存充足，可现场看货。",
		"contact":{"name":"周经理","phone":"18800000002"}
	}`))
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{
		"merchantId":"merchant-1",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"女童春款卫衣库存",
		"category":"童装卫衣",
		"description":"库存充足，可现场看货。",
		"contact":{"name":"周经理","phone":"18800000002"}
	}`))
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)

	submitForbiddenRec := httptest.NewRecorder()
	store.resourceMerchantIDs = map[string]string{"resource-1": "merchant-2"}
	submitForbiddenReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/submit", strings.NewReader(`{"merchantId":"merchant-2"}`))
	submitForbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(submitForbiddenRec, submitForbiddenReq)
	if submitForbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("submit status = %d body = %s, want forbidden", submitForbiddenRec.Code, submitForbiddenRec.Body.String())
	}
}

func TestResourceAPIRouterBindsTokenSubjectWhenCreatingResource(t *testing.T) {
	store := &fakeResourceAPIStore{
		publishConfig: model.ResourcePublishConfig{
			ID:               "config-1",
			TypeCode:         "inventory",
			RequiredFields:   []string{"merchantId", "cityCode", "typeCode", "title", "category", "contactName", "contactPhone"},
			DefaultValidDays: 7,
		},
		merchantStatus:   model.MerchantStatusActive,
		managedMerchants: map[string]bool{"merchant-1": true},
	}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{
		"merchantId":"merchant-1",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"女童春款卫衣库存",
		"category":"童装卫衣",
		"description":"库存充足，可现场看货。",
		"contact":{"name":"周经理","phone":"18800000002"}
	}`))
	createReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(createRec, createReq)
	decodeEnvelopeData(t, createRec, http.StatusOK)
	if store.created.CreatedByUser != "user-1" {
		t.Fatalf("createdByUser = %q, want token user", store.created.CreatedByUser)
	}

	draftRec := httptest.NewRecorder()
	draftReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/drafts", strings.NewReader(`{
		"merchantId":"merchant-1",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"女童春款卫衣草稿",
		"category":"童装卫衣",
		"description":"草稿先保存，稍后补图。",
		"contact":{"name":"周经理","phone":"18800000002"}
	}`))
	draftReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(draftRec, draftReq)
	decodeEnvelopeData(t, draftRec, http.StatusOK)
	if store.created.CreatedByUser != "user-1" || store.created.Status != model.ResourceStatusDraft {
		t.Fatalf("draft createdByUser/status = %q/%q, want token user draft", store.created.CreatedByUser, store.created.Status)
	}
}

func TestResourceAPIRouterRequiresOwnedMerchantWhenSubmittingResource(t *testing.T) {
	store := &fakeResourceAPIStore{
		managedMerchants:    map[string]bool{"merchant-1": true},
		resourceMerchantIDs: map[string]string{"resource-1": "merchant-1", "resource-2": "merchant-2"},
	}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-2/submit", strings.NewReader(`{"merchantId":"merchant-1"}`))
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/submit", strings.NewReader(`{"merchantId":"merchant-2"}`))
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
}

func TestResourceAPIRouterRequiresOwnedMerchantForResourceOwnerActions(t *testing.T) {
	store := &fakeResourceAPIStore{
		managedMerchants:    map[string]bool{"merchant-1": true},
		resourceMerchantIDs: map[string]string{"resource-1": "merchant-1", "resource-2": "merchant-2"},
	}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	cases := []struct {
		name   string
		path   string
		body   string
		method string
	}{
		{name: "refresh", method: http.MethodPost, path: "/api/v1/resources/resource-2/refresh", body: `{"merchantId":"merchant-1"}`},
		{name: "deal feedback", method: http.MethodPost, path: "/api/v1/resources/resource-2/deal-feedback", body: `{"merchantId":"merchant-1","isDealt":true}`},
		{name: "take down", method: http.MethodPost, path: "/api/v1/resources/resource-2/take-down", body: `{"merchantId":"merchant-1","reason":"已售罄"}`},
		{name: "delete taken down", method: http.MethodDelete, path: "/api/v1/resources/resource-2", body: `{"merchantId":"merchant-1"}`},
		{name: "repost similar", method: http.MethodPost, path: "/api/v1/resources/resource-2/repost-similar", body: `{"merchantId":"merchant-1"}`},
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

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/refresh", strings.NewReader(`{"merchantId":"merchant-2"}`))
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	decodeEnvelopeData(t, allowedRec, http.StatusOK)
}

func TestResourceAPIRouterDeletesTakenDownResource(t *testing.T) {
	store := &fakeResourceAPIStore{
		resourceStatuses: map[string]string{"resource-1": model.ResourceStatusTakenDown},
	}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/resources/resource-1", strings.NewReader(`{"merchantId":"merchant-1"}`))
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["id"] != "resource-1" || data["message"] != "资源已删除" {
		t.Fatalf("delete data = %#v, want deleted resource response", data)
	}
	if store.deletedResourceID != "resource-1" {
		t.Fatalf("deletedResourceID = %q, want resource-1", store.deletedResourceID)
	}
}

func TestResourceAPIRouterGetsEditableRejectedResourceAndSavesDraft(t *testing.T) {
	store := &fakeResourceAPIStore{
		merchantStatus: model.MerchantStatusActive,
		publishConfig: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"merchantId", "cityCode", "typeCode", "title", "category", "contactName", "contactPhone"},
		},
		editableDetail: model.EditableResourceDetail{
			ID: "resource-1", MerchantID: "merchant-1", CityCode: "zhili", TypeCode: "inventory", Status: model.ResourceStatusRejected,
			Title: "女童春款卫衣库存", Category: "童装卫衣", Description: "可拿样", ContactName: "周经理", ContactPhone: "18800000002",
			RejectReason: "图片不清晰",
		},
	}
	router := NewAPIRouter(store)

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/resources/resource-1/edit?merchantId=merchant-1", nil)
	router.ServeHTTP(getRec, getReq)
	getData := decodeEnvelopeData(t, getRec, http.StatusOK)
	if getData["rejectReason"] != "图片不清晰" || getData["status"] != model.ResourceStatusRejected {
		t.Fatalf("editable data = %#v, want reject reason and rejected status", getData)
	}

	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/resources/resource-1/draft", strings.NewReader(`{
		"merchantId":"merchant-1",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"修改后的女童春款卫衣库存",
		"category":"童装卫衣",
		"description":"已补充清晰图片",
		"contact":{"name":"周经理","phone":"18800000002"}
	}`))
	router.ServeHTTP(updateRec, updateReq)
	updateData := decodeEnvelopeData(t, updateRec, http.StatusOK)
	if updateData["status"] != model.ResourceStatusDraft || store.updatedResourceID != "resource-1" {
		t.Fatalf("update data = %#v updatedResourceID = %q, want draft update", updateData, store.updatedResourceID)
	}
}

func TestResourceAPIRouterGetsOwnUnpublishedResourceDetail(t *testing.T) {
	store := &fakeResourceAPIStore{
		managedMerchants:    map[string]bool{"merchant-1": true},
		resourceMerchantIDs: map[string]string{"resource-1": "merchant-1", "resource-2": "merchant-2"},
		ownDetail: model.ResourceDetail{
			ID: "resource-1", Status: model.ResourceStatusPending, TypeCode: "inventory", Title: "待审核童装库存",
			Category: "童装卫衣", Description: "可拿样", PriceText: "18元/件", QuantityText: "3800件",
			Attributes: model.JSONMap{"season": "春季"}, MerchantID: "merchant-1", MerchantName: "织里云仓",
			MerchantVerificationStatus: "verified", ContactName: "周经理", PhoneMasked: "18800000002", WechatMasked: "stock-demo",
		},
	}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	forbiddenRec := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/resources/resource-2/detail?merchantId=merchant-1", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s, want forbidden", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	allowedRec := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/resources/resource-1/detail?merchantId=merchant-2", nil)
	allowedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(allowedRec, allowedReq)
	allowedData := decodeEnvelopeData(t, allowedRec, http.StatusOK)
	if allowedData["status"] != model.ResourceStatusPending || allowedData["title"] != "待审核童装库存" {
		t.Fatalf("own detail = %#v, want pending own resource", allowedData)
	}
	if store.ownDetailMerchantID != "merchant-1" || store.ownDetailResourceID != "resource-1" {
		t.Fatalf("store args = %q/%q, want real merchant/resource", store.ownDetailMerchantID, store.ownDetailResourceID)
	}
}

func TestResourceAPIRouterAllowsAdminTokenForMerchantActions(t *testing.T) {
	store := &fakeResourceAPIStore{
		publishConfig: model.ResourcePublishConfig{
			ID:               "config-1",
			TypeCode:         "inventory",
			RequiredFields:   []string{"merchantId", "cityCode", "typeCode", "title", "category", "contactName", "contactPhone"},
			DefaultValidDays: 7,
		},
		merchantStatus:   model.MerchantStatusActive,
		managedMerchants: map[string]bool{},
	}
	router := NewAPIRouter(
		store,
		WithUserTokenService(&fakeUserTokenService{}),
		WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "admin-1", Roles: []string{"platform_operator"}}}),
	)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{
		"merchantId":"merchant-2",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"运营代发童装库存",
		"category":"童装",
		"description":"运营代发库存信息，可联系确认。",
		"contact":{"name":"周经理","phone":"18800000002"}
	}`))
	createReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(createRec, createReq)
	decodeEnvelopeData(t, createRec, http.StatusOK)
	if store.created.CreatedByOperator != "admin-1" || store.created.CreatedByUser != "" {
		t.Fatalf("admin creator = user:%q operator:%q, want admin operator only", store.created.CreatedByUser, store.created.CreatedByOperator)
	}

	submitRec := httptest.NewRecorder()
	submitReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/submit", strings.NewReader(`{"merchantId":"merchant-2"}`))
	submitReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(submitRec, submitReq)
	decodeEnvelopeData(t, submitRec, http.StatusOK)
}

func TestResourceAPIRouterUsesTokenSubjectForContactEvents(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	authorizedRec := httptest.NewRecorder()
	authorizedReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"userId":"attacker","action":"phone"}`))
	authorizedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(authorizedRec, authorizedReq)
	authorizedData := decodeEnvelopeData(t, authorizedRec, http.StatusOK)
	if store.contactInput.UserID != "user-1" {
		t.Fatalf("contact userID = %q, want token user", store.contactInput.UserID)
	}
	if authorizedData["phone"] != "18800000002" {
		t.Fatalf("authorized contact data = %#v, want full phone", authorizedData)
	}

	anonymousRec := httptest.NewRecorder()
	anonymousReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"userId":"attacker","action":"wechat"}`))
	router.ServeHTTP(anonymousRec, anonymousReq)
	if anonymousRec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous contact status = %d body = %s, want 401", anonymousRec.Code, anonymousRec.Body.String())
	}

	missingTokenServiceRec := httptest.NewRecorder()
	missingTokenServiceReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"action":"phone"}`))
	missingTokenServiceReq.Header.Set("Authorization", "Bearer user-token")
	NewAPIRouter(store).ServeHTTP(missingTokenServiceRec, missingTokenServiceReq)
	if missingTokenServiceRec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token service status = %d body = %s, want 401", missingTokenServiceRec.Code, missingTokenServiceRec.Body.String())
	}
}

func TestResourceAPIRouterCreatesContactUnlockOrderAndPaymentWithTokenSubject(t *testing.T) {
	store := &fakeResourceAPIStore{managedMerchants: map[string]bool{"merchant-2": true}}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}), WithWechatPayDevMock(true))

	orderRec := httptest.NewRecorder()
	orderReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-unlock-orders", strings.NewReader(`{"action":"phone","viewerMerchantId":"merchant-2"}`))
	orderReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(orderRec, orderReq)
	orderData := decodeEnvelopeData(t, orderRec, http.StatusOK)
	if orderData["orderId"] != "order-1" || orderData["status"] != model.PaymentOrderStatusPending {
		t.Fatalf("order data = %#v, want pending contact unlock order", orderData)
	}
	if store.contactUnlockOrderInput.UserID != "user-1" || store.contactUnlockOrderInput.ResourceID != "resource-1" || store.contactUnlockOrderInput.ViewerMerchantID != "merchant-2" {
		t.Fatalf("contact unlock order input = %#v, want token user and viewer merchant", store.contactUnlockOrderInput)
	}

	paymentRec := httptest.NewRecorder()
	paymentReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-unlock-orders/order-1/payment", strings.NewReader(`{"userId":"attacker"}`))
	paymentReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(paymentRec, paymentReq)
	paymentData := decodeEnvelopeData(t, paymentRec, http.StatusOK)
	if paymentData["orderId"] != "order-1" || paymentData["status"] != model.PaymentOrderStatusPaid {
		t.Fatalf("payment data = %#v, want paid contact unlock order", paymentData)
	}
	if store.contactUnlockPaymentContextInput.UserID != "user-1" || store.contactUnlockPaymentContextInput.OrderID != "order-1" {
		t.Fatalf("payment context input = %#v, want token user and order", store.contactUnlockPaymentContextInput)
	}
	if store.contactUnlockMarkInput.OutTradeNo != "contact_unlock_1" {
		t.Fatalf("mark input = %#v, want contact unlock out trade no", store.contactUnlockMarkInput)
	}
}

func TestResourceAPIRouterUnlocksOwnResourceContactWithoutRecordingEvent(t *testing.T) {
	store := &fakeResourceAPIStore{managedMerchants: map[string]bool{"merchant-1": true}}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"action":"phone"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["phone"] != "18800000002" || data["message"] != "电话已解锁" {
		t.Fatalf("contact data = %#v, want own phone unlocked", data)
	}
	if store.contactInput.ResourceID != "" || store.metricDelta.ContactClickCount != 0 {
		t.Fatalf("contact input = %#v metric = %#v, want no event or metric for own resource", store.contactInput, store.metricDelta)
	}
}

func TestResourceAPIRouterUsesTokenSubjectForSearchLogs(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	authorizedRec := httptest.NewRecorder()
	authorizedReq := httptest.NewRequest(http.MethodGet, "/api/v1/resource-search?cityCode=zhili&keyword=%E5%8D%AB%E8%A1%A3&userId=attacker", nil)
	authorizedReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(authorizedRec, authorizedReq)
	decodeEnvelopeData(t, authorizedRec, http.StatusOK)
	if store.searchLog.UserID != "user-1" {
		t.Fatalf("search log userID = %q, want token user", store.searchLog.UserID)
	}

	anonymousRec := httptest.NewRecorder()
	anonymousReq := httptest.NewRequest(http.MethodGet, "/api/v1/resource-search?cityCode=zhili&keyword=%E5%8D%AB%E8%A1%A3&userId=attacker", nil)
	router.ServeHTTP(anonymousRec, anonymousReq)
	decodeEnvelopeData(t, anonymousRec, http.StatusOK)
	if store.searchLog.UserID != "" {
		t.Fatalf("anonymous search log userID = %q, want empty user", store.searchLog.UserID)
	}
}

func TestResourceAPIRouterUsesAdminTokenReviewerForReview(t *testing.T) {
	store := &fakeResourceAPIStore{}
	router := NewAPIRouter(
		store,
		WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "admin-1", Roles: []string{"platform_operator"}}}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/resources/resource-1/review", strings.NewReader(`{"action":"approve","reviewerId":"attacker"}`))
	req.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(rec, req)
	decodeEnvelopeData(t, rec, http.StatusOK)

	if store.reviewInput.ReviewerID != "admin-1" {
		t.Fatalf("reviewerID = %q, want admin token user", store.reviewInput.ReviewerID)
	}
}

type fakeAdminLoginService struct{}

func (fakeAdminLoginService) Login(ctx context.Context, req adminauth.LoginRequest) (adminauth.LoginResponse, error) {
	return adminauth.LoginResponse{Token: "admin-token", OperatorID: "operator-1", Roles: []string{adminauth.RolePlatformOperator}}, nil
}

type rawErrorAdminLoginService struct{}

func (rawErrorAdminLoginService) Login(ctx context.Context, req adminauth.LoginRequest) (adminauth.LoginResponse, error) {
	return adminauth.LoginResponse{}, errors.New("sql: connection refused")
}

func decodeEnvelopeData(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) map[string]interface{} {
	t.Helper()
	body := decodeEnvelope(t, rec, wantStatus)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body = %#v, want data object", body)
	}
	return data
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) map[string]interface{} {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d body = %s, want %d", rec.Code, rec.Body.String(), wantStatus)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

type fakeResourceAPIStore struct {
	fakeCityAPIStore

	publishConfig                    model.ResourcePublishConfig
	merchantStatus                   string
	created                          model.CreateResourceInput
	pendingFilter                    model.ListPendingResourcesFilter
	reviewAction                     string
	reviewInput                      model.ReviewResourceInput
	listFilter                       model.ListResourcesFilter
	searchLog                        model.SearchLogInput
	contactInput                     model.ResourceContactEventInput
	contactUnlockState               model.ContactUnlockState
	contactUnlockInput               model.ContactUnlockInput
	vipManagedMerchant               model.VIPManagedMerchant
	contactUnlockOrderInput          model.CreateContactUnlockOrderInput
	contactUnlockPaymentContextInput model.GetContactUnlockPaymentContextInput
	contactUnlockPaymentOrderInput   model.CreateContactUnlockPaymentOrderInput
	contactUnlockMarkInput           model.MarkContactUnlockOrderPaidInput
	metricDelta                      model.ResourceMetricDelta
	myFilter                         model.ListMyResourcesFilter
	editableDetail                   model.EditableResourceDetail
	ownDetail                        model.ResourceDetail
	ownDetailMerchantID              string
	ownDetailResourceID              string
	managedMerchants                 map[string]bool
	resourceMerchantIDs              map[string]string
	resourceStatuses                 map[string]string
	updatedResourceID                string
	deletedResourceID                string
	completedAuditInput              model.ResourceContentAuditTaskResultInput
	auditCompletion                  model.ResourceContentAuditTaskCompletion
	publishedAuditResourceID         string
	rejectedAuditResourceID          string
	auditRejectReason                string
	mapEventCalled                   bool
	mapEventCalls                    int
	mapEventInput                    model.MerchantMapEventInput
	mapEventErr                      error
}

var _ ResourceAPIStore = (*fakeResourceAPIStore)(nil)

func (s *fakeResourceAPIStore) GetResourcePublishConfig(ctx context.Context, cityCode string, typeCode string) (model.ResourcePublishConfig, error) {
	return s.publishConfig, nil
}

func (s *fakeResourceAPIStore) GetMerchantPublishStatus(ctx context.Context, merchantID string) (string, error) {
	return s.merchantStatus, nil
}

func (s *fakeResourceAPIStore) GetMerchantContactPhone(ctx context.Context, merchantID string) (string, error) {
	return "18800000002", nil
}

func (s *fakeResourceAPIStore) UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error) {
	return s.managedMerchants[merchantID], nil
}

func (s *fakeResourceAPIStore) CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error) {
	s.created = input
	return model.CreateResourceResult{ID: "resource-1", Status: input.Status}, nil
}

func (s *fakeResourceAPIStore) UpdateResourceDraft(ctx context.Context, resourceID string, input model.CreateResourceInput) (model.CreateResourceResult, error) {
	s.updatedResourceID = resourceID
	return model.CreateResourceResult{ID: resourceID, Status: model.ResourceStatusDraft}, nil
}

func (s *fakeResourceAPIStore) SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error) {
	return model.SubmitResourceResult{ID: resourceID, Status: model.ResourceStatusPending}, nil
}

func (s *fakeResourceAPIStore) CompleteResourceContentAuditTask(ctx context.Context, input model.ResourceContentAuditTaskResultInput) (model.ResourceContentAuditTaskCompletion, error) {
	s.completedAuditInput = input
	if s.auditCompletion.ResourceID == "" {
		return model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"}, nil
	}
	return s.auditCompletion, nil
}

func (s *fakeResourceAPIStore) PublishResourceAfterAudit(ctx context.Context, resourceID string) (model.ReviewResourceResult, error) {
	s.publishedAuditResourceID = resourceID
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeResourceAPIStore) RejectResourceAfterAudit(ctx context.Context, resourceID string, reason string) (model.ReviewResourceResult, error) {
	s.rejectedAuditResourceID = resourceID
	s.auditRejectReason = reason
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusRejected}, nil
}

func (s *fakeResourceAPIStore) ListPendingResources(ctx context.Context, filter model.ListPendingResourcesFilter) (model.ListPendingResourcesResult, error) {
	s.pendingFilter = filter
	return model.ListPendingResourcesResult{
		Items: []model.PendingResourceItem{{
			ID: "resource-1", Title: "女童春款卫衣库存", TypeCode: "inventory", MerchantName: "织里云仓", CreatedAt: "2026-06-27T10:00:00Z",
		}},
		Page: filter.Page, PageSize: filter.PageSize, Total: 1,
	}, nil
}

func (s *fakeResourceAPIStore) ReviewResource(ctx context.Context, resourceID string, input model.ReviewResourceInput) (model.ReviewResourceResult, error) {
	s.reviewAction = input.Action
	s.reviewInput = input
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeResourceAPIStore) ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error) {
	s.listFilter = filter
	return model.ListResourcesResult{
		Items: []model.ResourceListItem{{
			ID: "resource-1", TypeCode: "inventory", Title: "女童春款卫衣库存", Category: "童装卫衣",
			CoverURL:  "https://img.example.com/list-cover.jpg",
			PriceText: "18元/件", QuantityText: "3800件", Merchant: model.ResourceMerchantBrief{ID: "merchant-1", Name: "织里云仓", VerificationStatus: "verified"},
			Tags: []string{"急清", "支持看货"}, RefreshedAt: "2026-06-27T10:00:00Z",
		}},
		Page: filter.Page, PageSize: filter.PageSize, Total: 1,
	}, nil
}

func (s *fakeResourceAPIStore) RecordSearchLog(ctx context.Context, input model.SearchLogInput) error {
	s.searchLog = input
	return nil
}

func (s *fakeResourceAPIStore) GetResourceContactUnlockInfo(ctx context.Context, resourceID string) (model.ResourceContactUnlockInfo, error) {
	merchantID := "merchant-1"
	if s.resourceMerchantIDs != nil && s.resourceMerchantIDs[resourceID] != "" {
		merchantID = s.resourceMerchantIDs[resourceID]
	}
	status := model.ResourceStatusPublished
	if s.resourceStatuses != nil && s.resourceStatuses[resourceID] != "" {
		status = s.resourceStatuses[resourceID]
	}
	return model.ResourceContactUnlockInfo{
		ResourceID:      resourceID,
		MerchantID:      merchantID,
		Status:          status,
		Phone:           "18800000002",
		Wechat:          "stock-demo",
		CommercialRules: model.DefaultCommercialRules(),
	}, nil
}

func (s *fakeResourceAPIStore) HasActiveContactUnlock(ctx context.Context, resourceID string, userID string) (model.ContactUnlockState, error) {
	return s.contactUnlockState, nil
}

func (s *fakeResourceAPIStore) FindActiveVIPManagedMerchant(ctx context.Context, userID string) (model.VIPManagedMerchant, error) {
	return s.vipManagedMerchant, nil
}

func (s *fakeResourceAPIStore) UpsertContactUnlock(ctx context.Context, input model.ContactUnlockInput) (model.ContactUnlockResult, error) {
	s.contactUnlockInput = input
	return model.ContactUnlockResult{ID: "unlock-1", ResourceID: input.ResourceID, UserID: input.UserID, SourceType: input.SourceType, ExpiresAt: input.ExpiresAt}, nil
}

func (s *fakeResourceAPIStore) CreateContactUnlockOrder(ctx context.Context, input model.CreateContactUnlockOrderInput) (model.ContactUnlockOrderResult, error) {
	s.contactUnlockOrderInput = input
	return model.ContactUnlockOrderResult{
		ID:         "order-1",
		ResourceID: input.ResourceID,
		Status:     model.PaymentOrderStatusPending,
		PriceCent:  500,
		Currency:   "CNY",
	}, nil
}

func (s *fakeResourceAPIStore) GetContactUnlockPaymentContext(ctx context.Context, input model.GetContactUnlockPaymentContextInput) (model.ContactUnlockPaymentContext, error) {
	s.contactUnlockPaymentContextInput = input
	return model.ContactUnlockPaymentContext{
		OrderID:       input.OrderID,
		ResourceID:    input.ResourceID,
		UserID:        input.UserID,
		OpenID:        "openid-1",
		Status:        model.PaymentOrderStatusPending,
		OutTradeNo:    "contact_unlock_1",
		AmountTotal:   500,
		Currency:      "CNY",
		ResourceTitle: "女童春款卫衣库存",
	}, nil
}

func (s *fakeResourceAPIStore) CreateContactUnlockPaymentOrder(ctx context.Context, input model.CreateContactUnlockPaymentOrderInput) (model.ContactUnlockPaymentOrder, error) {
	s.contactUnlockPaymentOrderInput = input
	return model.ContactUnlockPaymentOrder{
		ID:            input.OrderID,
		ResourceID:    input.ResourceID,
		OutTradeNo:    "contact_unlock_1",
		AmountTotal:   500,
		Currency:      "CNY",
		Status:        model.PaymentOrderStatusPending,
		ResourceTitle: "女童春款卫衣库存",
	}, nil
}

func (s *fakeResourceAPIStore) MarkContactUnlockOrderPaid(ctx context.Context, input model.MarkContactUnlockOrderPaidInput) (model.ContactUnlockPaymentResult, error) {
	s.contactUnlockMarkInput = input
	return model.ContactUnlockPaymentResult{OrderID: "order-1", ResourceID: "resource-1", Status: model.PaymentOrderStatusPaid}, nil
}

func (s *fakeResourceAPIStore) GetPublishedResourceDetail(ctx context.Context, resourceID string) (model.ResourceDetail, error) {
	merchantID := "merchant-1"
	if s.resourceMerchantIDs != nil && s.resourceMerchantIDs[resourceID] != "" {
		merchantID = s.resourceMerchantIDs[resourceID]
	}
	return model.ResourceDetail{
		ID: resourceID, Status: model.ResourceStatusPublished, TypeCode: "inventory", Title: "女童春款卫衣库存",
		Category: "童装卫衣", Description: "可拿样", PriceText: "18元/件", QuantityText: "3800件",
		Attributes: model.JSONMap{"season": "春季"}, MerchantID: merchantID, MerchantName: "织里云仓",
		MerchantVerificationStatus: "verified", ContactName: "周经理", PhoneMasked: "188****0002", WechatMasked: "stock-demo",
	}, nil
}

func (s *fakeResourceAPIStore) GetResourceMerchantID(ctx context.Context, resourceID string) (string, error) {
	if s.resourceMerchantIDs != nil && s.resourceMerchantIDs[resourceID] != "" {
		return s.resourceMerchantIDs[resourceID], nil
	}
	return "merchant-1", nil
}

func (s *fakeResourceAPIStore) RecordResourceContactEvent(ctx context.Context, input model.ResourceContactEventInput) (model.ResourceContactEventResult, error) {
	s.contactInput = input
	return model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"}, nil
}

func (s *fakeResourceAPIStore) RecordMerchantMapEvent(ctx context.Context, input model.MerchantMapEventInput) error {
	s.mapEventCalled = true
	s.mapEventCalls++
	s.mapEventInput = input
	return s.mapEventErr
}

func (s *fakeResourceAPIStore) UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error {
	s.metricDelta = delta
	return nil
}

func (s *fakeResourceAPIStore) GetResourceMetrics(ctx context.Context, resourceID string, from string, to string) (model.ResourceMetricsResult, error) {
	return model.ResourceMetricsResult{
		ResourceID: resourceID,
		Summary: model.ResourceMetricsSummary{
			DetailViewCount: 1,
			PhoneClickCount: 1,
			WechatCopyCount: 1,
		},
		Daily: []model.ResourceMetricDailyItem{{
			Date: "2026-06-28", DetailViewCount: 1, PhoneClickCount: 1, WechatCopyCount: 1,
		}},
	}, nil
}

func (s *fakeResourceAPIStore) GetMerchantMetricsSummary(ctx context.Context, merchantID string) (model.MerchantMetricsSummary, error) {
	return model.MerchantMetricsSummary{MerchantID: merchantID}, nil
}

func (s *fakeResourceAPIStore) ListMyResources(ctx context.Context, filter model.ListMyResourcesFilter) (model.ListMyResourcesResult, error) {
	s.myFilter = filter
	return model.ListMyResourcesResult{
		Items: []model.MyResourceItem{{
			ID: "resource-1", TypeCode: "inventory", Title: "女童春款卫衣库存", Category: "童装卫衣", Status: model.ResourceStatusPublished,
			CoverURL: "https://img.example.com/resource-cover.jpg",
			Metrics:  model.MyResourceMetrics{DetailViewCount: 1, PhoneClickCount: 1, WechatCopyCount: 1},
		}},
		Page: filter.Page, PageSize: filter.PageSize, Total: 1,
	}, nil
}

func (s *fakeResourceAPIStore) GetResourceOwnershipStatus(ctx context.Context, merchantID string, resourceID string) (model.ResourceOwnershipStatus, error) {
	status := model.ResourceStatusPublished
	if s.resourceStatuses != nil && s.resourceStatuses[resourceID] != "" {
		status = s.resourceStatuses[resourceID]
	}
	return model.ResourceOwnershipStatus{ID: resourceID, MerchantID: merchantID, Status: status}, nil
}

func (s *fakeResourceAPIStore) GetEditableResourceDetail(ctx context.Context, merchantID string, resourceID string) (model.EditableResourceDetail, error) {
	return s.editableDetail, nil
}

func (s *fakeResourceAPIStore) GetOwnResourceDetail(ctx context.Context, merchantID string, resourceID string) (model.ResourceDetail, error) {
	s.ownDetailMerchantID = merchantID
	s.ownDetailResourceID = resourceID
	return s.ownDetail, nil
}

func (s *fakeResourceAPIStore) RefreshResource(ctx context.Context, merchantID string, resourceID string) (model.RefreshResourceResult, error) {
	return model.RefreshResourceResult{ID: resourceID, RefreshedAt: "2026-06-27T10:00:00Z", RemainingRefreshQuota: 1}, nil
}

func (s *fakeResourceAPIStore) MarkDealt(ctx context.Context, input model.MarkDealtInput) (model.DealFeedbackResult, error) {
	return model.DealFeedbackResult{ID: input.ResourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeResourceAPIStore) TakeDownOwnResource(ctx context.Context, input model.TakeDownOwnResourceInput) (model.TakeDownOwnResourceResult, error) {
	return model.TakeDownOwnResourceResult{ID: input.ResourceID, Status: model.ResourceStatusTakenDown}, nil
}

func (s *fakeResourceAPIStore) DeleteTakenDownResource(ctx context.Context, merchantID string, resourceID string) (model.DeleteTakenDownResourceResult, error) {
	s.deletedResourceID = resourceID
	return model.DeleteTakenDownResourceResult{ID: resourceID, Status: model.ResourceStatusTakenDown}, nil
}

func (s *fakeResourceAPIStore) RepostSimilar(ctx context.Context, merchantID string, resourceID string) (model.RepostSimilarResult, error) {
	return model.RepostSimilarResult{ID: "resource-draft-1", Status: model.ResourceStatusDraft}, nil
}

func (s *fakeResourceAPIStore) RecordOperationLog(ctx context.Context, input model.OperationLogInput) error {
	return nil
}
