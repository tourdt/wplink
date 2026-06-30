package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
)

func TestAPIRouterLogsInAdmin(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{}, WithAdminLoginService(fakeAdminLoginService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["token"] != "admin-token" || data["userId"] != "user-1" {
		t.Fatalf("login data = %#v, want token and userId", data)
	}
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
	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/resource-search?cityCode=zhili&typeCode=inventory&keyword=卫衣&page=2&pageSize=5", nil)
	router.ServeHTTP(searchRec, searchReq)
	searchData := decodeEnvelopeData(t, searchRec, http.StatusOK)
	if store.listFilter.Keyword != "卫衣" || store.listFilter.Page != 2 || store.listFilter.PageSize != 5 {
		t.Fatalf("list filter = %#v, want query values", store.listFilter)
	}
	if store.searchLog.Keyword != "卫衣" || store.searchLog.ResultCount != 1 {
		t.Fatalf("search log = %#v, want keyword and result count", store.searchLog)
	}
	if len(searchData["items"].([]interface{})) != 1 {
		t.Fatalf("search items = %#v, want one item", searchData["items"])
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
	router.ServeHTTP(contactRec, contactReq)
	contactData := decodeEnvelopeData(t, contactRec, http.StatusOK)
	if contactData["message"] != "联系行为已记录" {
		t.Fatalf("contact data = %#v, want message", contactData)
	}
	if store.contactInput.Action != "phone" || store.metricDelta.PhoneClickCount != 1 {
		t.Fatalf("contact input = %#v metric = %#v, want phone metric", store.contactInput, store.metricDelta)
	}

	wechatRec := httptest.NewRecorder()
	wechatReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"action":"wechat"}`))
	router.ServeHTTP(wechatRec, wechatReq)
	wechatData := decodeEnvelopeData(t, wechatRec, http.StatusOK)
	if wechatData["message"] != "联系行为已记录" {
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
		WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{UserID: "admin-1", Roles: []string{"platform_operator"}}}),
	)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{
		"merchantId":"merchant-2",
		"cityCode":"zhili",
		"typeCode":"inventory",
		"title":"运营代发童装库存",
		"category":"童装",
		"contact":{"name":"周经理","phone":"18800000002"}
	}`))
	createReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(createRec, createReq)
	decodeEnvelopeData(t, createRec, http.StatusOK)
	if store.created.CreatedByUser != "admin-1" {
		t.Fatalf("admin createdByUser = %q, want admin token user", store.created.CreatedByUser)
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
	decodeEnvelopeData(t, authorizedRec, http.StatusOK)
	if store.contactInput.UserID != "user-1" {
		t.Fatalf("contact userID = %q, want token user", store.contactInput.UserID)
	}

	anonymousRec := httptest.NewRecorder()
	anonymousReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"userId":"attacker","action":"wechat"}`))
	router.ServeHTTP(anonymousRec, anonymousReq)
	decodeEnvelopeData(t, anonymousRec, http.StatusOK)
	if store.contactInput.UserID != "" {
		t.Fatalf("anonymous contact userID = %q, want empty user", store.contactInput.UserID)
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
		WithAdminTokenService(&fakeAdminTokenService{subject: session.AdminTokenSubject{UserID: "admin-1", Roles: []string{"platform_operator"}}}),
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
	return adminauth.LoginResponse{Token: "admin-token", UserID: "user-1", Roles: []string{adminauth.RolePlatformOperator}}, nil
}

func decodeEnvelopeData(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) map[string]interface{} {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d body = %s, want %d", rec.Code, rec.Body.String(), wantStatus)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body = %#v, want data object", body)
	}
	return data
}

type fakeResourceAPIStore struct {
	fakeCityAPIStore

	publishConfig       model.ResourcePublishConfig
	merchantStatus      string
	created             model.CreateResourceInput
	pendingFilter       model.ListPendingResourcesFilter
	reviewAction        string
	reviewInput         model.ReviewResourceInput
	listFilter          model.ListResourcesFilter
	searchLog           model.SearchLogInput
	contactInput        model.ResourceContactEventInput
	metricDelta         model.ResourceMetricDelta
	myFilter            model.ListMyResourcesFilter
	editableDetail      model.EditableResourceDetail
	ownDetail           model.ResourceDetail
	ownDetailMerchantID string
	ownDetailResourceID string
	managedMerchants    map[string]bool
	resourceMerchantIDs map[string]string
	resourceStatuses    map[string]string
	updatedResourceID   string
	deletedResourceID   string
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
			PriceText: "18元/件", QuantityText: "3800件", Merchant: model.ResourceMerchantBrief{ID: "merchant-1", Name: "织里云仓", VerificationStatus: "verified"},
			RefreshedAt: "2026-06-27T10:00:00Z",
		}},
		Page: filter.Page, PageSize: filter.PageSize, Total: 1,
	}, nil
}

func (s *fakeResourceAPIStore) RecordSearchLog(ctx context.Context, input model.SearchLogInput) error {
	s.searchLog = input
	return nil
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
			Metrics: model.MyResourceMetrics{DetailViewCount: 1, PhoneClickCount: 1, WechatCopyCount: 1},
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
