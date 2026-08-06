package server

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/logic/adminauth"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	"wplink/backend/app/internal/middleware"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

func TestGrowthHandlersThroughGeneratedRoutesDoNotUseMigrationSkeleton(t *testing.T) {
	svcCtx, _ := newGrowthGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)
	tests := []struct {
		name   string
		method string
		target string
		body   string
		admin  bool
	}{
		{name: "public campaigns", method: http.MethodGet, target: "/api/v1/growth-campaigns/active"},
		{name: "merchant tasks", method: http.MethodGet, target: "/api/v1/merchants/merchant-1/growth-tasks"},
		{name: "admin campaigns", method: http.MethodGet, target: "/api/v1/admin/growth-campaigns", admin: true},
		{name: "create campaign", method: http.MethodPost, target: "/api/v1/admin/growth-campaigns", body: `{}`, admin: true},
		{name: "update campaign", method: http.MethodPost, target: "/api/v1/admin/growth-campaigns/starter", body: `{}`, admin: true},
		{name: "admin rules", method: http.MethodGet, target: "/api/v1/admin/growth-campaigns/starter/rules", admin: true},
		{name: "create rule", method: http.MethodPost, target: "/api/v1/admin/growth-campaigns/starter/rules", body: `{}`, admin: true},
		{name: "update rule", method: http.MethodPost, target: "/api/v1/admin/growth-campaigns/starter/rules/first-login", body: `{}`, admin: true},
		{name: "admin grants", method: http.MethodGet, target: "/api/v1/admin/growth-campaigns/starter/grants", admin: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tc.admin {
				req.Header.Set("Authorization", "Bearer admin-token")
			}
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			body := decodeEnvelope(t, rec, rec.Code)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("route %s %s was not registered", tc.method, tc.target)
			}
			if body["msg"] == "接口暂不可用，请稍后重试" {
				t.Fatalf("route %s %s still uses migration skeleton", tc.method, tc.target)
			}
		})
	}
}

func TestGrowthPublicCampaignsStayAnonymousAndMatchExistingDTO(t *testing.T) {
	svcCtx, mock := newGrowthGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)
	mock.ExpectQuery(`(?s)FROM growth_campaigns gc`).WillReturnRows(sqlmock.NewRows([]string{
		"code", "name", "config_snapshot", "rule_code", "rule_name", "trigger_event", "reward_type", "reward_amount", "valid_days",
		"per_user_limit", "per_user_daily_limit", "per_resource_daily_limit", "description", "conditions",
	}).AddRow(
		"starter", "新手活动", []byte(`{"frontendTitle":"新手发布权益","frontendHint":"完成任务即可领取"}`),
		"first-login", "首次登录", model.GrowthEventUserFirstLogin, model.EntitlementTypePublishQuota, int64(5), int64(30),
		nil, nil, nil, "登录得发布次数", []byte(`{}`),
	))

	body := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/growth-campaigns/active", nil), http.StatusOK)
	data := body["data"].(map[string]interface{})
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("data=%#v, want one anonymous growth campaign", data)
	}
	campaign := items[0].(map[string]interface{})
	rules := campaign["rules"].([]interface{})
	if campaign["code"] != "starter" || campaign["title"] != "新手发布权益" || len(rules) != 1 {
		t.Fatalf("campaign=%#v, want existing public campaign DTO", campaign)
	}
	rule := rules[0].(map[string]interface{})
	if rule["ruleCode"] != "first-login" || rule["rewardText"] != "5 次发布次数" {
		t.Fatalf("rule=%#v, want existing public rule DTO", rule)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("public growth SQL expectations: %v", err)
	}
}

func TestGrowthMerchantTasksUsePathMerchantAndRequireRealPermission(t *testing.T) {
	t.Run("missing dependencies", func(t *testing.T) {
		server := newGeneratedAPIServer(t, &svc.ServiceContext{AdminAuth: growthAdminAuthMiddleware()})
		body := assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/merchants/merchant-1/growth-tasks", ""), http.StatusInternalServerError)
		if body["errorCode"] != errx.CodeInternalError {
			t.Fatalf("body=%#v, want fail-closed dependency error", body)
		}
	})

	t.Run("typed nil growth store", func(t *testing.T) {
		server := newGeneratedAPIServer(t, &svc.ServiceContext{
			APIStore:  &svc.APIStore{GrowthCampaignModel: (*model.GrowthCampaignModel)(nil)},
			AdminAuth: growthAdminAuthMiddleware(),
		})
		body := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/growth-campaigns/active", nil), http.StatusInternalServerError)
		if body["errorCode"] != errx.CodeInternalError {
			t.Fatalf("body=%#v, want typed-nil store rejection", body)
		}
	})

	svcCtx, mock := newGrowthGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)

	badToken := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1/growth-tasks", nil)
	badToken.Header.Set("Authorization", "Bearer expired-token")
	assertTask6GeneratedStatus(t, server, badToken, http.StatusUnauthorized)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-2").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/merchants/merchant-2/growth-tasks", ""), http.StatusForbidden)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`(?s)FROM merchant_entitlements`).
		WithArgs("merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`(?s)FROM growth_campaigns gc`).
		WillReturnRows(sqlmock.NewRows([]string{"code"}))
	body := assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/merchants/merchant-1/growth-tasks?merchantId=merchant-2", ""), http.StatusOK)
	data := body["data"].(map[string]interface{})
	if tasks, ok := data["tasks"].([]interface{}); !ok || len(tasks) != 0 {
		t.Fatalf("data=%#v, want path merchant and tasks: []", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("merchant growth SQL expectations: %v", err)
	}
}

func TestGrowthAdminRoutesAlwaysUseAdminAuth(t *testing.T) {
	tests := []struct {
		method string
		target string
		body   string
	}{
		{method: http.MethodGet, target: "/api/v1/admin/growth-campaigns"},
		{method: http.MethodPost, target: "/api/v1/admin/growth-campaigns", body: `{}`},
		{method: http.MethodPost, target: "/api/v1/admin/growth-campaigns/starter", body: `{}`},
		{method: http.MethodGet, target: "/api/v1/admin/growth-campaigns/starter/rules"},
		{method: http.MethodPost, target: "/api/v1/admin/growth-campaigns/starter/rules", body: `{}`},
		{method: http.MethodPost, target: "/api/v1/admin/growth-campaigns/starter/rules/first-login", body: `{}`},
		{method: http.MethodGet, target: "/api/v1/admin/growth-campaigns/starter/grants"},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			svcCtx, _ := newGrowthGeneratedServiceContext(t)
			server := newGeneratedAPIServer(t, svcCtx)
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			assertTask6GeneratedStatus(t, server, req, http.StatusUnauthorized)
		})
	}
}

func TestGrowthAdminListsPreserveStatusGrantFiltersAndEmptyArrays(t *testing.T) {
	svcCtx, mock := newGrowthGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)

	mock.ExpectQuery(`(?s)FROM growth_campaigns`).
		WithArgs("paused").
		WillReturnRows(sqlmock.NewRows([]string{"code"}))
	campaigns := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet, "/api/v1/admin/growth-campaigns?status=paused&operatorId=attacker", ""), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := campaigns["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("campaigns=%#v, want filtered items: []", campaigns)
	}

	mock.ExpectQuery(`(?s)FROM growth_campaign_rules`).
		WithArgs("starter").
		WillReturnRows(sqlmock.NewRows([]string{"campaign_code"}))
	rules := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet, "/api/v1/admin/growth-campaigns/starter/rules", ""), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := rules["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("rules=%#v, want items: []", rules)
	}

	mock.ExpectQuery(`(?s)FROM growth_reward_grants`).
		WithArgs("starter", "first-login", "merchant-1", "granted", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	grants := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet, "/api/v1/admin/growth-campaigns/starter/grants?ruleCode=first-login&merchantId=merchant-1&status=granted&pageSize=7", ""), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := grants["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("grants=%#v, want filtered items: []", grants)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin growth list SQL expectations: %v", err)
	}
}

func TestGrowthAdminWritesUsePathCodesAndContextOperator(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		body         string
		queryPattern string
		queryArgs    []driver.Value
		resultCode   string
		action       string
	}{
		{
			name: "create campaign", target: "/api/v1/admin/growth-campaigns?operatorId=attacker",
			body:         `{"operatorId":"attacker","code":"created-campaign","name":"新活动","status":"draft"}`,
			queryPattern: `(?s)INSERT INTO growth_campaigns`,
			queryArgs:    []driver.Value{"created-campaign", "新活动", "draft", "", "", sqlmock.AnyArg(), "operator-9"},
			resultCode:   "created-campaign", action: "growth_campaign_save",
		},
		{
			name: "update campaign", target: "/api/v1/admin/growth-campaigns/path-campaign?operatorId=attacker",
			body:         `{"operatorId":"attacker","code":"body-campaign","name":"更新活动","status":"paused"}`,
			queryPattern: `(?s)INSERT INTO growth_campaigns`,
			queryArgs:    []driver.Value{"path-campaign", "更新活动", "paused", "", "", sqlmock.AnyArg(), "operator-9"},
			resultCode:   "path-campaign", action: "growth_campaign_save",
		},
		{
			name: "create rule", target: "/api/v1/admin/growth-campaigns/starter/rules?operatorId=attacker",
			body:         `{"operatorId":"attacker","ruleCode":"created-rule","ruleName":"首次登录","triggerEvent":"user_first_login","rewardType":"publish_quota","rewardAmount":5,"validDays":30}`,
			queryPattern: `(?s)INSERT INTO growth_campaign_rules`,
			queryArgs:    []driver.Value{"starter", "created-rule", "首次登录", "user_first_login", "active", int64(100), sqlmock.AnyArg(), "publish_quota", int64(5), int64(30), int64(0), int64(0), int64(0), ""},
			resultCode:   "created-rule", action: "growth_campaign_rule_save",
		},
		{
			name: "update rule", target: "/api/v1/admin/growth-campaigns/starter/rules/path-rule?operatorId=attacker",
			body:         `{"operatorId":"attacker","ruleCode":"body-rule","ruleName":"首次登录更新","triggerEvent":"user_first_login","status":"inactive","priority":8,"rewardType":"publish_quota","rewardAmount":3,"validDays":10}`,
			queryPattern: `(?s)INSERT INTO growth_campaign_rules`,
			queryArgs:    []driver.Value{"starter", "path-rule", "首次登录更新", "user_first_login", "inactive", int64(8), sqlmock.AnyArg(), "publish_quota", int64(3), int64(10), int64(0), int64(0), int64(0), ""},
			resultCode:   "path-rule", action: "growth_campaign_rule_save",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, mock := newGrowthGeneratedServiceContext(t)
			server := newGeneratedAPIServer(t, svcCtx)
			updatedAt := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
			mock.ExpectBegin()
			mock.ExpectQuery(tc.queryPattern).WithArgs(tc.queryArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"code", "updated_at"}).AddRow(tc.resultCode, updatedAt))
			mock.ExpectExec(`(?s)INSERT INTO operation_logs`).
				WithArgs("operator-9", "platform_operator", tc.action, "growth_campaign", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			data := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost, tc.target, tc.body), http.StatusOK)["data"].(map[string]interface{})
			if data["code"] != tc.resultCode || data["updatedAt"] != updatedAt.Format(time.RFC3339) {
				t.Fatalf("data=%#v, want saved code and updatedAt", data)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("admin growth write SQL expectations: %v", err)
			}
		})
	}
}

func newGrowthGeneratedServiceContext(t *testing.T) (*svc.ServiceContext, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	apiStore := &svc.APIStore{
		UserModel:                model.NewUserModel(db),
		MerchantEntitlementModel: model.NewMerchantEntitlementModel(db),
		GrowthCampaignModel:      model.NewGrowthCampaignModel(db),
	}
	return &svc.ServiceContext{
		APIStore:          apiStore,
		UserTokenService:  &fakeUserTokenService{},
		AdminTokenService: adminauth.NewValidatingAdminTokenService(nil, nil),
		AdminAuth:         growthAdminAuthMiddleware(),
	}, mock
}

func growthAdminAuthMiddleware() rest.Middleware {
	adminService := &fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "operator-9",
		Roles:      []string{permission.RoleSuperAdmin},
	}}
	return middleware.NewAdminAuthMiddleware(adminService).Handle
}

func growthAdminRequest(method string, target string, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin-token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestAdminHandlersThroughGeneratedRoutesUseRealAdminAuthAndNoMigrationSkeleton(t *testing.T) {
	svcCtx := &svc.ServiceContext{AdminAuth: middleware.NewAdminAuthMiddleware(&fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "admin-task11",
		Roles:      []string{permission.RoleSuperAdmin},
	}}).Handle}
	server := newGeneratedAPIServer(t, svcCtx)
	tests := []struct {
		method string
		target string
		body   string
	}{
		{http.MethodGet, "/api/v1/admin/operators", ""},
		{http.MethodPost, "/api/v1/admin/operators", `{}`},
		{http.MethodPost, "/api/v1/admin/operators/operator-1", `{}`},
		{http.MethodPost, "/api/v1/admin/operators/operator-1/status", `{}`},
		{http.MethodGet, "/api/v1/admin/module-permissions", ""},
		{http.MethodPost, "/api/v1/admin/module-permissions/platform_operator", `{}`},
		{http.MethodGet, "/api/v1/admin/dashboard/overview", ""},
		{http.MethodGet, "/api/v1/admin/resources?status=published", ""},
		{http.MethodGet, "/api/v1/admin/resources/pending", ""},
		{http.MethodPost, "/api/v1/admin/resources/resource-1/review", `{}`},
		{http.MethodGet, "/api/v1/admin/resource-reports", ""},
		{http.MethodPost, "/api/v1/admin/resource-reports/report-1/review", `{}`},
		{http.MethodPost, "/api/v1/admin/merchants/merchant-1/entitlements", `{}`},
		{http.MethodGet, "/api/v1/admin/operation-logs", ""},
		{http.MethodGet, "/api/v1/admin/search-logs", ""},
		{http.MethodPost, "/api/v1/admin/tasks/resource-lifecycle/run", ""},
		{http.MethodGet, "/api/v1/admin/merchants", ""},
		{http.MethodGet, "/api/v1/admin/banner-topics", ""},
		{http.MethodPost, "/api/v1/admin/banner-topics", `{}`},
		{http.MethodPost, "/api/v1/admin/banner-topics/banner-1", `{}`},
		{http.MethodGet, "/api/v1/admin/hot-search-keywords", ""},
		{http.MethodPost, "/api/v1/admin/hot-search-keywords", `{}`},
		{http.MethodPost, "/api/v1/admin/hot-search-keywords/keyword-1", `{}`},
		{http.MethodGet, "/api/v1/admin/vip/plans", ""},
		{http.MethodPost, "/api/v1/admin/vip/plans", `{}`},
		{http.MethodPost, "/api/v1/admin/vip/plans/monthly", `{}`},
		{http.MethodGet, "/api/v1/admin/vip/quota-packs", ""},
		{http.MethodPost, "/api/v1/admin/vip/quota-packs", `{}`},
		{http.MethodPost, "/api/v1/admin/vip/quota-packs/publish-5", `{}`},
		{http.MethodGet, "/api/v1/admin/vip/promotions", ""},
		{http.MethodPost, "/api/v1/admin/vip/promotions", `{}`},
		{http.MethodPost, "/api/v1/admin/vip/promotions/promotion-1", `{}`},
		{http.MethodGet, "/api/v1/admin/resource-type-configs", ""},
		{http.MethodPost, "/api/v1/admin/resource-type-configs", `{}`},
		{http.MethodPost, "/api/v1/admin/resource-type-configs/config-1", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			unauthorizedReq := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				unauthorizedReq.Header.Set("Content-Type", "application/json")
			}
			assertTask6GeneratedStatus(t, server, unauthorizedReq, http.StatusUnauthorized)

			req := growthAdminRequest(tc.method, tc.target, tc.body)
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			body := decodeEnvelope(t, rec, rec.Code)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("route %s %s was not registered", tc.method, tc.target)
			}
			if body["msg"] == "接口暂不可用，请稍后重试" {
				t.Fatalf("route %s %s still uses migration skeleton", tc.method, tc.target)
			}
		})
	}
}

func TestAdminGeneratedListRoutesMapFiltersAndEncodeEmptyArrays(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)

	mock.ExpectQuery(`(?s)WITH filtered AS`).
		WithArgs("运营", permission.RolePlatformOperator, model.AdminCredentialStatusEnabled, int64(5), int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"operator_id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/operators?keyword=%E8%BF%90%E8%90%A5&role=platform_operator&status=enabled&page=2&pageSize=5")

	mock.ExpectQuery(`(?s)SELECT\s+\(SELECT COUNT\(\*\) FROM resources`).
		WithArgs("zhili").WillReturnRows(sqlmock.NewRows([]string{"pending", "contacts"}).AddRow(int64(2), int64(3)))
	mock.ExpectQuery(`(?s)FROM \(\s+SELECT '资源审核'`).
		WithArgs("zhili").WillReturnRows(sqlmock.NewRows([]string{"type"}))
	dashboard := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet, "/api/v1/admin/dashboard/overview?cityCode=zhili", ""), http.StatusOK)["data"].(map[string]interface{})
	if tasks, ok := dashboard["tasks"].([]interface{}); !ok || len(tasks) != 0 {
		t.Fatalf("dashboard=%#v, want tasks: []", dashboard)
	}

	mock.ExpectQuery(`(?s)FROM resources r\s+JOIN merchants`).
		WithArgs("published", "zhili", "inventory", int64(5), int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	adminResources := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/resources?cityCode=zhili&typeCode=inventory&status=published&page=2&pageSize=5", ""), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := adminResources["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("admin resources=%#v, want items: []", adminResources)
	}

	mock.ExpectQuery(`(?s)FROM resources r\s+JOIN merchants`).
		WithArgs(model.ResourceStatusManualReview, "hangzhou", "demand", int64(7), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	pendingResources := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/resources/pending?cityCode=hangzhou&typeCode=demand&pageSize=7", ""), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := pendingResources["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("pending resources=%#v, want items: []", pendingResources)
	}

	mock.ExpectQuery(`(?s)WITH grouped_reports AS`).
		WithArgs(model.ResourceReportStatusPending, int64(4), int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/resource-reports?status=pending&page=2&pageSize=4")

	mock.ExpectQuery(`(?s)FROM merchants m`).
		WithArgs("zhili", "factory", model.MerchantStatusActive, "童装", int64(9), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/merchants?cityCode=zhili&merchantType=factory&status=active&keyword=%E7%AB%A5%E8%A3%85&page=2&pageSize=9")

	mock.ExpectQuery(`(?s)FROM banner_topics bt`).WithArgs("zhili", "banner", "active").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/banner-topics?cityCode=zhili&kind=banner&status=active")

	mock.ExpectQuery(`(?s)FROM hot_search_keywords hsk`).WithArgs("zhili", "active").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/hot-search-keywords?cityCode=zhili&status=active")

	mock.ExpectQuery(`(?s)FROM vip_plans p`).WillReturnRows(sqlmock.NewRows([]string{"code"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/vip/plans")
	mock.ExpectQuery(`(?s)FROM vip_quota_packs`).WillReturnRows(sqlmock.NewRows([]string{"code"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/vip/quota-packs")
	mock.ExpectQuery(`(?s)FROM vip_promotions promo`).WillReturnRows(sqlmock.NewRows([]string{"code"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/vip/promotions")

	mock.ExpectQuery(`(?s)FROM resource_type_configs rtc`).WithArgs("zhili", "active").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/resource-type-configs?cityCode=zhili&status=active")

	mock.ExpectQuery(`(?s)FROM operation_logs`).WithArgs("resource", "resource-1", "admin-task11", int64(6), int64(6)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/operation-logs?objectType=resource&objectId=resource-1&operatorId=admin-task11&page=2&pageSize=6")

	mock.ExpectQuery(`(?s)FROM search_logs sl`).WithArgs("zhili", "童装", int64(8), int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	assertAdminItemsArray(t, server, "/api/v1/admin/search-logs?cityCode=zhili&keyword=%E7%AB%A5%E8%A3%85&page=2&pageSize=8")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin list SQL expectations: %v", err)
	}
}

func TestAdminGeneratedResourceListReturnsDeclaredStatusAndCompleteItem(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	createdAt := time.Date(2026, 8, 6, 13, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)FROM resources r\s+JOIN merchants`).
		WithArgs(model.ResourceStatusPublished, "zhili", "inventory", int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "title", "type_code", "merchant_name", "created_at", "total",
		}).AddRow(
			"resource-1", model.ResourceStatusPublished, "秋季童装库存", "inventory", "织里童装厂", createdAt, int64(1),
		))
	data := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/resources?cityCode=zhili&typeCode=inventory&status=published", ""), http.StatusOK)["data"].(map[string]interface{})
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("data=%#v, want one admin resource", data)
	}
	item := items[0].(map[string]interface{})
	want := map[string]interface{}{
		"id": "resource-1", "status": model.ResourceStatusPublished, "title": "秋季童装库存",
		"typeCode": "inventory", "merchantName": "织里童装厂", "createdAt": createdAt.Format(time.RFC3339),
	}
	if !reflect.DeepEqual(item, want) {
		t.Fatalf("item=%#v, want complete declared item %#v", item, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin resource SQL expectations: %v", err)
	}
}

func TestAdminGeneratedListItemsEncodeNonOptionalNestedSlicesAsArrays(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	now := time.Date(2026, 8, 6, 13, 30, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)WITH filtered AS`).
		WithArgs("", "", "", int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"operator_id", "login_name", "real_name", "status", "roles", "created_at", "last_login_at", "total",
		}).AddRow("operator-empty-roles", "operator", "运营", model.AdminCredentialStatusEnabled, "{}", now, nil, int64(1)))
	operatorData := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/operators", ""), http.StatusOK)["data"].(map[string]interface{})
	assertAdminNestedArray(t, operatorData, "roles")

	mock.ExpectQuery(`(?s)FROM banner_topics bt`).WithArgs("", "", "").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "city_code", "kind", "title", "subtitle", "cover_url", "type_scope", "jump_type",
			"jump_target", "tags", "start_at", "end_at", "sort_order", "status", "created_at", "updated_at",
		}).AddRow(
			"banner-empty-arrays", "zhili", "banner", "运营位", "", "", `[]`, "internal", "/pages/home/index",
			`[]`, time.Unix(0, 0), time.Unix(0, 0), int64(1), "active", now, now,
		))
	bannerData := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/banner-topics", ""), http.StatusOK)["data"].(map[string]interface{})
	assertAdminNestedArray(t, bannerData, "typeScope")
	assertAdminNestedArray(t, bannerData, "tags")

	mock.ExpectQuery(`(?s)FROM resource_type_configs rtc`).WithArgs("", "").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "version", "city_code", "type_code", "type_name", "direction", "field_schema",
			"required_fields", "filter_fields", "display_template", "review_rules", "sort_weights",
			"message_rules", "commercial_rules", "default_valid_days", "status",
		}).AddRow(
			"config-empty-arrays", int64(1), "zhili", "inventory", "库存", "supply", `{}`,
			`[]`, `[]`, `{}`, `{}`, `{}`, `{}`, `{}`, int64(7), "active",
		))
	configData := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/resource-type-configs", ""), http.StatusOK)["data"].(map[string]interface{})
	assertAdminNestedArray(t, configData, "requiredFields")
	assertAdminNestedArray(t, configData, "filterFields")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin nested slice SQL expectations: %v", err)
	}
}

func assertAdminNestedArray(t *testing.T, data map[string]interface{}, field string) {
	t.Helper()
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("data=%#v, want one item for %s", data, field)
	}
	value, ok := items[0].(map[string]interface{})[field].([]interface{})
	if !ok || len(value) != 0 {
		t.Fatalf("data=%#v, want %s: []", data, field)
	}
}

func TestAdminGeneratedPermissionModuleDenialAndSuperAdminCheck(t *testing.T) {
	platformCtx, _ := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "operator-plain", Roles: []string{permission.RolePlatformOperator}, Modules: []string{permission.AdminModuleResourceReview},
	})
	platformServer := newGeneratedAPIServer(t, platformCtx)
	assertTask6GeneratedStatus(t, platformServer, growthAdminRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", ""), http.StatusForbidden)

	permissionCtx, _ := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "operator-plain", Roles: []string{permission.RolePlatformOperator}, Modules: []string{permission.AdminModuleAdminPermissions},
	})
	permissionServer := newGeneratedAPIServer(t, permissionCtx)
	body := assertTask6GeneratedStatus(t, permissionServer, growthAdminRequest(http.MethodGet, "/api/v1/admin/operators", ""), http.StatusForbidden)
	if body["msg"] != "只有超级管理员可以管理管理员权限" {
		t.Fatalf("body=%#v, want Logic super-admin check", body)
	}
}

func TestAdminGeneratedWritesUseContextOperatorAndPathIdentifiers(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	updatedAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO merchant_entitlements`).
		WithArgs("merchant-path", "publish_quota", "manual", int64(3), "", model.EntitlementTypeTopVoucher).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("entitlement-1"))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).
		WithArgs("admin-task11", "platform_operator", "entitlement_grant", "merchant", "merchant-path", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/merchants/merchant-path/entitlements?merchantId=merchant-query&operatorId=attacker",
		`{"merchantId":"merchant-body","operatorId":"attacker","entitlementType":"publish_quota","sourceType":"manual","totalAmount":3,"reason":"运营补发"}`), http.StatusOK)

	mock.ExpectQuery(`(?s)UPDATE banner_topics`).WithArgs(
		"banner-path", "zhili", "banner", "路径主键测试", "", "", model.JSONStringSlice{}, "internal", "/pages/home/index",
		model.JSONStringSlice{}, "", "", int64(0), "active",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow("banner-path", updatedAt))
	banner := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/banner-topics/banner-path?configId=banner-query",
		`{"id":"banner-body","kind":"banner","title":"路径主键测试","jumpType":"internal","jumpTarget":"/pages/home/index","cityCode":"zhili","status":"active"}`), http.StatusOK)["data"].(map[string]interface{})
	if banner["id"] != "banner-path" {
		t.Fatalf("banner=%#v, want path id", banner)
	}

	mock.ExpectQuery(`(?s)UPDATE hot_search_keywords`).WithArgs(
		"keyword-path", "zhili", "夏款现货", int64(20), "active", "", "",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow("keyword-path", updatedAt))
	hot := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/hot-search-keywords/keyword-path?configId=keyword-query",
		`{"id":"keyword-body","cityCode":"zhili","keyword":"夏款现货","sortOrder":20,"status":"active"}`), http.StatusOK)["data"].(map[string]interface{})
	if hot["id"] != "keyword-path" {
		t.Fatalf("hot=%#v, want path id", hot)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM vip_quota_packs\s+WHERE code`).WithArgs("pack-path").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectQuery(`(?s)INSERT INTO vip_quota_packs`).WithArgs(
		"pack-path", "发布次数包", "临时补充次数", int64(2500), int64(1990), "限时", sqlmock.AnyArg(), "active", int64(10),
	).WillReturnRows(sqlmock.NewRows([]string{"code", "updated_at"}).AddRow("pack-path", updatedAt))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).
		WithArgs("admin-task11", "platform_operator", "vip_quota_pack_save", "vip_config", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	pack := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/vip/quota-packs/pack-path?operatorId=attacker",
		`{"code":"pack-body","operatorId":"attacker","name":"发布次数包","description":"临时补充次数","standardPriceCent":2500,"salePriceCent":1990,"saleLabel":"限时","status":"active","displayOrder":10,"benefits":{"publishQuota":5,"refreshQuota":0,"topVoucherCount":0,"topDurationHours":0}}`), http.StatusOK)["data"].(map[string]interface{})
	if pack["code"] != "pack-path" {
		t.Fatalf("pack=%#v, want path code", pack)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin write SQL expectations: %v", err)
	}
}

func TestAdminGeneratedResourceTypeVersionConflictUsesPathIDAndChineseMessage(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	mock.ExpectQuery(`(?s)UPDATE resource_type_configs`).WithArgs(
		"config-path", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		sqlmock.AnyArg(), sqlmock.AnyArg(), int64(7), "active", sqlmock.AnyArg(), int64(3),
	).WillReturnRows(sqlmock.NewRows([]string{"version", "updated_at"}))
	body := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/resource-type-configs/config-path?configId=config-query",
		`{"configId":"config-body","version":3,"fieldSchema":{},"requiredFields":[],"filterFields":[],"displayTemplate":{},"reviewRules":{},"sortWeights":{},"messageRules":{},"commercialRules":{},"defaultValidDays":7,"status":"active"}`), http.StatusConflict)
	if body["msg"] != "资源类型配置已被其他人修改，请刷新后重试" {
		t.Fatalf("body=%#v, want optimistic version conflict message", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("resource type SQL expectations: %v", err)
	}
}

func TestAdminGeneratedLifecycleRunsExistingTaskAndFailsClosedOnTypedNil(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	mock.ExpectQuery(`(?s)WITH newly_expired AS`).WillReturnRows(sqlmock.NewRows([]string{"id", "merchant_id", "title"}))
	mock.ExpectQuery(`(?s)SELECT r.id::text, r.merchant_id::text, r.title\s+FROM resources r`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "merchant_id", "title"}))
	data := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/tasks/resource-lifecycle/run", ""), http.StatusOK)["data"].(map[string]interface{})
	if data["expiredCount"] != float64(0) || data["expiringReminderCount"] != float64(0) {
		t.Fatalf("data=%#v, want empty lifecycle result", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("lifecycle SQL expectations: %v", err)
	}

	typedNilCases := []struct {
		name   string
		mutate func(*svc.APIStore)
	}{
		{name: "resource model", mutate: func(store *svc.APIStore) { store.ResourceModel = (*model.ResourceModel)(nil) }},
		{name: "message model", mutate: func(store *svc.APIStore) { store.MessageModel = (*model.MessageModel)(nil) }},
	}
	for _, tc := range typedNilCases {
		t.Run(tc.name, func(t *testing.T) {
			typedNilCtx, _ := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin}})
			tc.mutate(typedNilCtx.APIStore)
			typedNilServer := newGeneratedAPIServer(t, typedNilCtx)
			body := assertTask6GeneratedStatus(t, typedNilServer, growthAdminRequest(http.MethodPost,
				"/api/v1/admin/tasks/resource-lifecycle/run", ""), http.StatusInternalServerError)
			if body["errorCode"] != errx.CodeInternalError {
				t.Fatalf("body=%#v, want fail-closed dependency error", body)
			}
		})
	}
}

func TestAdminGeneratedReviewsUseContextReviewerAndPathIDs(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)UPDATE resources\s+SET status`).WithArgs(
		"resource-path", model.ResourceStatusRejected, "reject", sqlmock.AnyArg(), "资料不完整",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "merchant_id", "title", "status"}).
		AddRow("resource-path", "merchant-1", "童装库存", model.ResourceStatusRejected))
	mock.ExpectExec(`(?s)INSERT INTO resource_review_records`).
		WithArgs("resource-path", "admin-task11", "reject", "资料不完整").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO messages`).
		WithArgs("merchant:merchant-1", "resource_reject", "resource-path", "资源审核驳回", "童装库存 审核未通过，请修改后重新提交", model.MerchantMyResourcesTargetURL("merchant-1")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).
		WithArgs("admin-task11", "platform_operator", "resource_reject", "resource", "resource-path", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	resource := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/resources/resource-path/review?resourceId=resource-query&reviewerId=attacker",
		`{"resourceId":"resource-body","reviewerId":"attacker","action":"reject","reason":"资料不完整"}`), http.StatusOK)["data"].(map[string]interface{})
	if resource["id"] != "resource-path" || resource["status"] != model.ResourceStatusRejected {
		t.Fatalf("resource=%#v, want path resource reviewed", resource)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM resource_reports rr`).WithArgs("report-path").
		WillReturnRows(sqlmock.NewRows([]string{"report_id", "resource_id", "merchant_id", "title", "status", "deleted"}).
			AddRow("report-path", "resource-1", "merchant-1", "童装库存", model.ResourceStatusPublished, false))
	mock.ExpectQuery(`(?s)UPDATE resource_reports`).WithArgs(
		"resource-1", model.ResourceReportStatusInvalid, "admin-task11", model.ResourceReportActionInvalid,
		"举报不成立，资源维持原状态", false, "report-path",
	).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(model.ResourceReportStatusInvalid))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\)\s+FROM resource_reports`).WithArgs("resource-1", "report-path").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		"admin-task11", "platform_operator", "resource_report_review", "resource_report", "report-path", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	report := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/resource-reports/report-path/review?reportId=report-query&reviewerId=attacker",
		`{"reportId":"report-body","reviewerId":"attacker","action":"invalid"}`), http.StatusOK)["data"].(map[string]interface{})
	if report["id"] != "report-path" || report["resolvedReportCount"] != float64(2) {
		t.Fatalf("report=%#v, want path report reviewed", report)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin review SQL expectations: %v", err)
	}
}

func TestAdminGeneratedPermissionWriteUsesContextActorAndPathRole(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM admin_roles\s+WHERE code`).WithArgs(permission.RolePlatformOperator).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "modules"}).
			AddRow("role-1", permission.RolePlatformOperator, []byte(`["resource_review"]`)))
	mock.ExpectExec(`(?s)UPDATE admin_roles`).WithArgs(permission.RolePlatformOperator, model.JSONStringSlice{
		permission.AdminModuleMerchants, permission.AdminModuleResourceReview,
	}).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE admin_operators ao`).WithArgs("role-1").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		"admin-task11", permission.RoleSuperAdmin, "admin_role_modules_update", "admin_role", "role-1", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`(?s)FROM admin_roles\s+WHERE code`).WithArgs(permission.RolePlatformOperator).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "modules"}).
			AddRow("role-1", permission.RolePlatformOperator, []byte(`["merchants","resource_review"]`)))
	mock.ExpectCommit()
	data := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/module-permissions/platform_operator?roleCode=super_admin&operatorId=attacker",
		`{"roleCode":"super_admin","operatorId":"attacker","modules":["merchants","resource_review"]}`), http.StatusOK)["data"].(map[string]interface{})
	if data["roleCode"] != permission.RolePlatformOperator {
		t.Fatalf("data=%#v, want path role code", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin permission SQL expectations: %v", err)
	}
}

func TestAdminGeneratedPermissionRoutesSucceedWithCompleteMapping(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "9001", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	now := time.Date(2026, 8, 6, 14, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT EXISTS`).WithArgs("new.operator", "").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`(?s)INSERT INTO admin_operators`).
		WithArgs("new.operator", "新管理员", model.AdminCredentialStatusEnabled, "9001").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("9101"))
	mock.ExpectExec(`(?s)INSERT INTO admin_login_credentials`).WithArgs("9101", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)DELETE FROM admin_operator_role_assignments`).WithArgs("9101").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO admin_operator_role_assignments`).WithArgs("9101", permission.RolePlatformOperator).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		"9001", permission.RoleSuperAdmin, "admin_operator_create", "admin_operator", "9101", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`(?s)FROM admin_operators ao`).WithArgs("9101").
		WillReturnRows(adminOperatorRows("9101", "new.operator", "新管理员", model.AdminCredentialStatusEnabled, "{platform_operator}", now))
	mock.ExpectCommit()
	created := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/operators?operatorId=attacker",
		`{"operatorId":"attacker","loginName":" new.operator ","realName":" 新管理员 ","password":"secret123","roles":["platform_operator"],"status":"enabled"}`), http.StatusOK)["data"]
	if !reflect.DeepEqual(created, map[string]interface{}{"operatorId": "9101", "message": "管理员账号已创建"}) {
		t.Fatalf("created=%#v, want complete create response", created)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM admin_operators ao`).WithArgs("9201").
		WillReturnRows(adminOperatorRows("9201", "before.operator", "原管理员", model.AdminCredentialStatusEnabled, "{platform_operator}", now))
	mock.ExpectQuery(`(?s)SELECT EXISTS`).WithArgs("updated.operator", "9201").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`(?s)UPDATE admin_operators`).WithArgs("9201", "updated.operator", "更新管理员", model.AdminCredentialStatusEnabled).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)UPDATE admin_login_credentials`).WithArgs("9201", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)DELETE FROM admin_operator_role_assignments`).WithArgs("9201").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO admin_operator_role_assignments`).WithArgs("9201", permission.RoleSuperAdmin).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		"9001", permission.RoleSuperAdmin, "admin_operator_update", "admin_operator", "9201", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`(?s)FROM admin_operators ao`).WithArgs("9201").
		WillReturnRows(adminOperatorRows("9201", "updated.operator", "更新管理员", model.AdminCredentialStatusEnabled, "{super_admin}", now))
	mock.ExpectCommit()
	updated := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/operators/9201?operatorId=attacker-path",
		`{"operatorId":"attacker-body","loginName":" updated.operator ","realName":" 更新管理员 ","roles":["super_admin"],"status":"enabled"}`), http.StatusOK)["data"]
	if !reflect.DeepEqual(updated, map[string]interface{}{"operatorId": "9201", "message": "管理员账号已更新"}) {
		t.Fatalf("updated=%#v, want path operator and complete update response", updated)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM admin_operators ao`).WithArgs("9301").
		WillReturnRows(adminOperatorRows("9301", "status.operator", "状态管理员", model.AdminCredentialStatusEnabled, "{platform_operator}", now))
	mock.ExpectExec(`(?s)UPDATE admin_operators\s+SET status`).WithArgs("9301", model.AdminCredentialStatusDisabled).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		"9001", permission.RoleSuperAdmin, "admin_operator_status_update", "admin_operator", "9301", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`(?s)FROM admin_operators ao`).WithArgs("9301").
		WillReturnRows(adminOperatorRows("9301", "status.operator", "状态管理员", model.AdminCredentialStatusDisabled, "{platform_operator}", now))
	mock.ExpectCommit()
	statusUpdated := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/operators/9301/status?operatorId=attacker-query",
		`{"operatorId":"attacker-body","status":"disabled"}`), http.StatusOK)["data"]
	if !reflect.DeepEqual(statusUpdated, map[string]interface{}{"operatorId": "9301", "message": "管理员账号状态已更新"}) {
		t.Fatalf("statusUpdated=%#v, want path operator and complete status response", statusUpdated)
	}

	mock.ExpectQuery(`(?s)FROM admin_roles`).WithArgs(permission.RolePlatformOperator).
		WillReturnRows(sqlmock.NewRows([]string{"code", "admin_modules"}).AddRow(permission.RolePlatformOperator, `[]`))
	moduleData := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/module-permissions", ""), http.StatusOK)["data"].(map[string]interface{})
	assertCompleteAdminModulePermissionDTO(t, moduleData)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin permission success SQL expectations: %v", err)
	}
}

func TestAdminGeneratedOperationalConfigCreateRoutesSucceedWithCompleteMapping(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	updatedAt := time.Date(2026, 8, 6, 15, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)INSERT INTO banner_topics`).WithArgs(
		"zhili", "banner", "秋季上新", "今日精选", "https://img.example/banner.jpg",
		jsonArgument(`["inventory"]`), "internal", "/pages/home/index", jsonArgument(`["秋装"]`),
		"2026-08-07T00:00:00Z", "2026-08-31T23:59:59Z", int64(8), "active",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow("banner-created", updatedAt))
	banner := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/banner-topics?operatorId=attacker",
		`{"operatorId":"attacker","cityCode":" zhili ","kind":"banner","title":" 秋季上新 ","subtitle":"今日精选","coverUrl":"https://img.example/banner.jpg","typeScope":[" inventory "],"jumpType":"internal","jumpTarget":"/pages/home/index","tags":[" 秋装 "],"startAt":"2026-08-07T00:00:00Z","endAt":"2026-08-31T23:59:59Z","sortOrder":8,"status":"active"}`), http.StatusOK)["data"]
	if !reflect.DeepEqual(banner, map[string]interface{}{"id": "banner-created", "updatedAt": updatedAt.Format(time.RFC3339)}) {
		t.Fatalf("banner=%#v, want complete create response", banner)
	}

	mock.ExpectQuery(`(?s)INSERT INTO hot_search_keywords`).WithArgs(
		"zhili", "秋季童装", int64(9), "active", "2026-08-07T00:00:00Z", "2026-08-31T23:59:59Z",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow("keyword-created", updatedAt))
	hot := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/hot-search-keywords?operatorId=attacker",
		`{"operatorId":"attacker","cityCode":" zhili ","keyword":" 秋季童装 ","sortOrder":9,"status":"active","startAt":"2026-08-07T00:00:00Z","endAt":"2026-08-31T23:59:59Z"}`), http.StatusOK)["data"]
	if !reflect.DeepEqual(hot, map[string]interface{}{"id": "keyword-created", "updatedAt": updatedAt.Format(time.RFC3339)}) {
		t.Fatalf("hot=%#v, want complete create response", hot)
	}

	mock.ExpectQuery(`(?s)INSERT INTO resource_type_configs`).WithArgs(
		"zhili", "fabric_stock", "面料库存", model.ResourceDirectionSupply,
		jsonArgument(`{"fields":[{"key":"color","label":"颜色","type":"text"}]}`), jsonArgument(`["title","color"]`), jsonArgument(`["color"]`),
		jsonContainsArgument{required: map[string]interface{}{"group": map[string]interface{}{"code": "fabric", "name": "面辅料", "sort": float64(3)}}},
		jsonArgument(`{"manual":true}`), jsonArgument(`{"freshness":2}`), jsonArgument(`{"published":"已发布"}`),
		jsonArgument(`{"contactUnlock":{"currency":"CNY","mode":"login_free","priceCent":0,"repeatUnlockDays":30,"vipFree":false},"publish":{"mode":"consume_quota"}}`), int64(20), "active", sqlmock.AnyArg(),
	).WillReturnRows(sqlmock.NewRows([]string{"id", "version", "updated_at"}).AddRow("config-created", int64(1), updatedAt))
	resourceType := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/resource-type-configs?operatorId=attacker",
		`{"operatorId":"attacker","cityCode":" zhili ","typeCode":" fabric_stock ","typeName":" 面料库存 ","direction":"supply","groupCode":" fabric ","groupName":" 面辅料 ","groupSort":3,"fieldSchema":{"fields":[{"key":"color","label":"颜色","type":"text"}]},"requiredFields":["title","color"],"filterFields":["color"],"displayTemplate":{},"reviewRules":{"manual":true},"sortWeights":{"freshness":2},"messageRules":{"published":"已发布"},"commercialRules":{"contactUnlock":{"enabled":false}},"defaultValidDays":20,"status":"active"}`), http.StatusOK)["data"]
	if !reflect.DeepEqual(resourceType, map[string]interface{}{
		"id": "config-created", "version": float64(1), "updatedAt": updatedAt.Format(time.RFC3339),
	}) {
		t.Fatalf("resourceType=%#v, want complete create response", resourceType)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin operational config create SQL expectations: %v", err)
	}
}

func TestAdminGeneratedVIPCreateAndUpdateRoutesUseContextOperatorAndPathCodes(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "9001", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	updatedAt := time.Date(2026, 8, 6, 16, 0, 0, 0, time.UTC)

	expectAdminVIPPlanSave(t, mock, "plan-create", "创建套餐", "9001", updatedAt)
	createdPlan := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/vip/plans?operatorId=attacker",
		adminVIPPlanBody("plan-create", "创建套餐", "attacker")), http.StatusOK)["data"]
	assertAdminVIPSaveResponse(t, createdPlan, "plan-create", updatedAt)

	expectAdminVIPPlanSave(t, mock, "plan-path", "更新套餐", "9001", updatedAt)
	updatedPlan := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/vip/plans/plan-path?operatorId=attacker&planCode=plan-query",
		adminVIPPlanBody("plan-body", "更新套餐", "attacker")), http.StatusOK)["data"]
	assertAdminVIPSaveResponse(t, updatedPlan, "plan-path", updatedAt)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM vip_quota_packs\s+WHERE code`).WithArgs("pack-create").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectQuery(`(?s)INSERT INTO vip_quota_packs`).WithArgs(
		"pack-create", "创建次数包", "补充次数", int64(3000), int64(2500), "限时",
		jsonContainsArgument{required: map[string]interface{}{"publishQuota": float64(6)}}, "active", int64(5),
	).WillReturnRows(sqlmock.NewRows([]string{"code", "updated_at"}).AddRow("pack-create", updatedAt))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		"9001", "platform_operator", "vip_quota_pack_save", "vip_config", "", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	createdPack := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/vip/quota-packs?operatorId=attacker",
		`{"operatorId":"attacker","code":"pack-create","name":"创建次数包","description":"补充次数","standardPriceCent":3000,"salePriceCent":2500,"saleLabel":"限时","status":"active","displayOrder":5,"benefits":{"publishPolicy":"quota","publishQuota":6,"refreshQuota":2,"topVoucherCount":1,"topDurationHours":24,"homepageImageLimit":4}}`), http.StatusOK)["data"]
	assertAdminVIPSaveResponse(t, createdPack, "pack-create", updatedAt)

	expectAdminVIPPromotionSave(t, mock, "promotion-create", "9001", updatedAt)
	createdPromotion := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/vip/promotions?operatorId=attacker",
		adminVIPPromotionBody("promotion-create", "attacker")), http.StatusOK)["data"]
	assertAdminVIPSaveResponse(t, createdPromotion, "promotion-create", updatedAt)

	expectAdminVIPPromotionSave(t, mock, "promotion-path", "9001", updatedAt)
	updatedPromotion := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/vip/promotions/promotion-path?operatorId=attacker&promotionCode=promotion-query",
		adminVIPPromotionBody("promotion-body", "attacker")), http.StatusOK)["data"]
	assertAdminVIPSaveResponse(t, updatedPromotion, "promotion-path", updatedAt)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin VIP success SQL expectations: %v", err)
	}
}

func adminOperatorRows(operatorID string, loginName string, realName string, status string, roles string, now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"operator_id", "login_name", "real_name", "status", "roles", "created_at", "last_login_at", "total",
	}).AddRow(operatorID, loginName, realName, status, roles, now, nil, int64(1))
}

func assertCompleteAdminModulePermissionDTO(t *testing.T, data map[string]interface{}) {
	t.Helper()
	if len(data) != 2 {
		t.Fatalf("data=%#v, want only modules and roles", data)
	}
	modules, ok := data["modules"].([]interface{})
	if !ok || len(modules) != 13 {
		t.Fatalf("modules=%#v, want all 13 configurable modules", data["modules"])
	}
	for _, module := range modules {
		item, ok := module.(map[string]interface{})
		if !ok || len(item) != 3 || item["code"] == "" || item["label"] == "" || item["group"] == "" {
			t.Fatalf("module=%#v, want complete code/label/group DTO", module)
		}
	}
	roles, ok := data["roles"].([]interface{})
	if !ok || len(roles) != 1 {
		t.Fatalf("roles=%#v, want platform operator permission", data["roles"])
	}
	role := roles[0].(map[string]interface{})
	wantModules := []interface{}{permission.AdminModuleResourceReview, permission.AdminModuleResourceReports}
	if role["roleCode"] != permission.RolePlatformOperator || role["roleName"] != "平台运营" || !reflect.DeepEqual(role["modules"], wantModules) {
		t.Fatalf("role=%#v, want complete default platform permission DTO", role)
	}
}

type jsonArgument string

func (expected jsonArgument) Match(value driver.Value) bool {
	var actualValue interface{}
	var expectedValue interface{}
	actual, ok := value.([]byte)
	if !ok {
		if text, textOK := value.(string); textOK {
			actual = []byte(text)
		} else {
			return false
		}
	}
	return json.Unmarshal(actual, &actualValue) == nil &&
		json.Unmarshal([]byte(expected), &expectedValue) == nil &&
		reflect.DeepEqual(actualValue, expectedValue)
}

type jsonContainsArgument struct {
	required map[string]interface{}
}

func (expected jsonContainsArgument) Match(value driver.Value) bool {
	actual := map[string]interface{}{}
	bytes, ok := value.([]byte)
	if !ok {
		if text, textOK := value.(string); textOK {
			bytes = []byte(text)
		} else {
			return false
		}
	}
	if json.Unmarshal(bytes, &actual) != nil {
		return false
	}
	for key, wanted := range expected.required {
		if !reflect.DeepEqual(actual[key], wanted) {
			return false
		}
	}
	return true
}

func expectAdminVIPPlanSave(t *testing.T, mock sqlmock.Sqlmock, code string, name string, operatorID string, updatedAt time.Time) {
	t.Helper()
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM vip_plans p.*WHERE p.code`).WithArgs(code).
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectQuery(`(?s)INSERT INTO vip_plans`).WithArgs(code, name, int64(3), int64(9900), "active", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "updated_at"}).AddRow("plan-id", code, updatedAt))
	mock.ExpectExec(`(?s)INSERT INTO vip_plan_versions`).WithArgs("plan-id", jsonContainsArgument{required: map[string]interface{}{
		"publishQuota": float64(30), "refreshQuota": float64(10), "topVoucherCount": float64(1), "topDurationHours": float64(24),
	}}).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		operatorID, "platform_operator", "vip_plan_save", "vip_config", "", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
}

func adminVIPPlanBody(code string, name string, operatorID string) string {
	return fmt.Sprintf(`{"operatorId":%q,"code":%q,"name":%q,"durationMonths":3,"standardPriceCent":9900,"status":"active","displayOrder":7,"benefits":{"publishPolicy":"quota","publishQuota":30,"refreshQuota":10,"topVoucherCount":1,"topDurationHours":24,"homepageImageLimit":5}}`, operatorID, code, name)
}

func expectAdminVIPPromotionSave(t *testing.T, mock sqlmock.Sqlmock, code string, operatorID string, updatedAt time.Time) {
	t.Helper()
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM vip_promotions promo.*WHERE promo.code`).WithArgs(code).
		WillReturnRows(sqlmock.NewRows([]string{"plan_code"}))
	mock.ExpectQuery(`(?s)WITH selected_plan AS`).WithArgs(
		code, "monthly", "launch", int64(3900), "2026-08-07T00:00:00Z", "2026-08-31T23:59:59Z", int64(100), "active",
	).WillReturnRows(sqlmock.NewRows([]string{"code", "updated_at"}).AddRow(code, updatedAt))
	mock.ExpectExec(`(?s)INSERT INTO operation_logs`).WithArgs(
		operatorID, "platform_operator", "vip_promotion_save", "vip_config", "", sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
}

func adminVIPPromotionBody(code string, operatorID string) string {
	return fmt.Sprintf(`{"operatorId":%q,"code":%q,"planCode":"monthly","promotionType":"launch","salePriceCent":3900,"startsAt":"2026-08-07T00:00:00Z","endsAt":"2026-08-31T23:59:59Z","quotaLimit":100,"status":"active"}`, operatorID, code)
}

func assertAdminVIPSaveResponse(t *testing.T, value interface{}, code string, updatedAt time.Time) {
	t.Helper()
	want := map[string]interface{}{"code": code, "updatedAt": updatedAt.Format(time.RFC3339)}
	if !reflect.DeepEqual(value, want) {
		t.Fatalf("value=%#v, want complete VIP save response %#v", value, want)
	}
}

func TestAdminGeneratedHandlersRejectTypedNilDependencies(t *testing.T) {
	cases := []struct {
		name   string
		method string
		target string
		mutate func(*svc.APIStore)
	}{
		{name: "permissions", method: http.MethodGet, target: "/api/v1/admin/operators", mutate: func(s *svc.APIStore) { s.AdminPermissionModel = (*model.AdminPermissionModel)(nil) }},
		{name: "dashboard", method: http.MethodGet, target: "/api/v1/admin/dashboard/overview", mutate: func(s *svc.APIStore) { s.AdminDashboardModel = (*model.AdminDashboardModel)(nil) }},
		{name: "resources", method: http.MethodGet, target: "/api/v1/admin/resources", mutate: func(s *svc.APIStore) { s.ResourceModel = (*model.ResourceModel)(nil) }},
		{name: "entitlements", method: http.MethodPost, target: "/api/v1/admin/merchants/merchant-1/entitlements", mutate: func(s *svc.APIStore) { s.MerchantEntitlementModel = (*model.MerchantEntitlementModel)(nil) }},
		{name: "operation logs", method: http.MethodGet, target: "/api/v1/admin/operation-logs", mutate: func(s *svc.APIStore) { s.OperationLogModel = (*model.OperationLogModel)(nil) }},
		{name: "search logs", method: http.MethodGet, target: "/api/v1/admin/search-logs", mutate: func(s *svc.APIStore) { s.SearchLogModel = (*model.SearchLogModel)(nil) }},
		{name: "merchants", method: http.MethodGet, target: "/api/v1/admin/merchants", mutate: func(s *svc.APIStore) { s.MerchantModel = (*model.MerchantModel)(nil) }},
		{name: "banners", method: http.MethodGet, target: "/api/v1/admin/banner-topics", mutate: func(s *svc.APIStore) { s.BannerTopicModel = (*model.BannerTopicModel)(nil) }},
		{name: "hot keywords", method: http.MethodGet, target: "/api/v1/admin/hot-search-keywords", mutate: func(s *svc.APIStore) { s.HotSearchKeywordModel = (*model.HotSearchKeywordModel)(nil) }},
		{name: "vip", method: http.MethodGet, target: "/api/v1/admin/vip/plans", mutate: func(s *svc.APIStore) { s.VIPModel = (*model.VIPModel)(nil) }},
		{name: "resource configs", method: http.MethodGet, target: "/api/v1/admin/resource-type-configs", mutate: func(s *svc.APIStore) { s.ResourceTypeConfigModel = (*model.ResourceTypeConfigModel)(nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, _ := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{OperatorID: "admin-task11", Roles: []string{permission.RoleSuperAdmin}})
			tc.mutate(svcCtx.APIStore)
			server := newGeneratedAPIServer(t, svcCtx)
			body := assertTask6GeneratedStatus(t, server, growthAdminRequest(tc.method, tc.target, ""), http.StatusInternalServerError)
			if body["errorCode"] != errx.CodeInternalError {
				t.Fatalf("body=%#v, want typed-nil dependency rejection", body)
			}
		})
	}
}

func TestAdminGeneratedDependencyFailuresLogSafeCategoryAndTrustedContext(t *testing.T) {
	svcCtx, mock := newAdminGeneratedServiceContext(t, session.AdminTokenSubject{
		OperatorID: "admin-log-1", Roles: []string{permission.RoleSuperAdmin},
	})
	server := newGeneratedAPIServer(t, svcCtx)
	var logBuffer bytes.Buffer
	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(&logBuffer))
	t.Cleanup(func() {
		if currentWriter := logx.Reset(); currentWriter != nil {
			_ = currentWriter.Close()
		}
		if previousWriter != nil {
			logx.SetWriter(previousWriter)
		}
	})
	sensitiveErr := fmt.Errorf("password=db-secret Authorization=Bearer-secret requestBody={complete}: %w", context.DeadlineExceeded)

	mock.ExpectQuery(`(?s)SELECT\s+\(SELECT COUNT`).WithArgs("city-secret").WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/dashboard/overview?cityCode=city-secret", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM resources r\s+JOIN merchants`).
		WithArgs("status-secret", "city-secret", "type-secret", int64(20), int64(0)).WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/resources?cityCode=city-secret&typeCode=type-secret&status=status-secret", ""), http.StatusInternalServerError)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO merchant_entitlements`).
		WithArgs("merchant-log-1", "publish_quota", "manual", int64(2), "", model.EntitlementTypeTopVoucher).
		WillReturnError(sensitiveErr)
	mock.ExpectRollback()
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/merchants/merchant-log-1/entitlements",
		`{"operatorId":"attacker","entitlementType":"publish_quota","sourceType":"manual","totalAmount":2,"reason":"运营补发"}`), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM operation_logs`).
		WithArgs("resource", "object-log-1", "filter-operator", int64(20), int64(0)).WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/operation-logs?objectType=resource&objectId=object-log-1&operatorId=filter-operator", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM search_logs sl`).
		WithArgs("city-secret", "keyword-secret", int64(20), int64(0)).WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/search-logs?cityCode=city-secret&keyword=keyword-secret", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM merchants m`).
		WithArgs("city-secret", "factory", "active", "keyword-secret", int64(20), int64(0)).WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/merchants?cityCode=city-secret&merchantType=factory&status=active&keyword=keyword-secret", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM banner_topics bt`).WithArgs("city-secret", "banner", "active").WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/banner-topics?cityCode=city-secret&kind=banner&status=active", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM hot_search_keywords hsk`).WithArgs("city-secret", "active").WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/hot-search-keywords?cityCode=city-secret&status=active", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM vip_plans p`).WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/vip/plans", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)FROM resource_type_configs rtc`).WithArgs("city-secret", "active").WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet,
		"/api/v1/admin/resource-type-configs?cityCode=city-secret&status=active", ""), http.StatusInternalServerError)

	mock.ExpectQuery(`(?s)WITH newly_expired AS`).WillReturnError(sensitiveErr)
	assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodPost,
		"/api/v1/admin/tasks/resource-lifecycle/run", ""), http.StatusInternalServerError)

	// logx writer 异步刷盘；读取前先关闭当前 writer，保证前面十个请求的日志都已进入缓冲区。
	if currentWriter := logx.Reset(); currentWriter != nil {
		_ = currentWriter.Close()
	}
	if previousWriter != nil {
		logx.SetWriter(previousWriter)
		previousWriter = nil
	}
	logText := logBuffer.String()
	for _, operation := range []string{
		"get_admin_dashboard", "list_admin_resources", "grant_merchant_entitlement", "list_operation_logs",
		"list_search_logs", "list_admin_merchants", "list_banner_topics", "list_hot_search_keywords",
		"list_vip_plans", "list_resource_type_configs",
	} {
		entry := logEntryForOperation(logText, operation)
		if entry == "" {
			t.Fatalf("log=%q, want operation %q", logText, operation)
		}
		for _, field := range []string{`"operatorId":"admin-log-1"`, `"errorCategory":"timeout"`, `"errorType":"*fmt.wrapError"`} {
			if !strings.Contains(entry, field) {
				t.Fatalf("entry=%q, want %q for operation %q", entry, field, operation)
			}
		}
	}
	for _, secret := range []string{"db-secret", "Bearer-secret", "requestBody={complete}"} {
		for _, line := range strings.Split(logText, "\n") {
			if strings.Contains(line, `"operation":`) && strings.Contains(line, secret) {
				t.Fatalf("application operation log contains sensitive detail %q: %q", secret, line)
			}
		}
	}
	lifecycleEntry := logEntryForOperation(logText, "run_resource_lifecycle")
	for _, field := range []string{
		`"operatorId":"admin-log-1"`, `"stage":"mark_expired_resources"`, `"errorCategory":"timeout"`,
	} {
		if !strings.Contains(lifecycleEntry, field) {
			t.Fatalf("lifecycle entry=%q, want %q", lifecycleEntry, field)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin failure logging SQL expectations: %v", err)
	}
}

func logEntryForOperation(logText string, operation string) string {
	var fallback string
	for _, line := range strings.Split(logText, "\n") {
		if strings.Contains(line, `"operation":"`+operation+`"`) {
			fallback = line
			if strings.Contains(line, `"operatorId":`) {
				return line
			}
		}
	}
	return fallback
}

func assertAdminItemsArray(t *testing.T, server *rest.Server, target string) {
	t.Helper()
	data := assertTask6GeneratedStatus(t, server, growthAdminRequest(http.MethodGet, target, ""), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := data["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("target=%s data=%#v, want items: []", target, data)
	}
}

func newAdminGeneratedServiceContext(t *testing.T, subject session.AdminTokenSubject) (*svc.ServiceContext, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &svc.ServiceContext{
		APIStore: &svc.APIStore{
			ResourceTypeConfigModel:  model.NewResourceTypeConfigModel(db),
			AdminDashboardModel:      model.NewAdminDashboardModel(db),
			MerchantModel:            model.NewMerchantModel(db),
			ResourceModel:            model.NewResourceModel(db),
			BannerTopicModel:         model.NewBannerTopicModel(db),
			HotSearchKeywordModel:    model.NewHotSearchKeywordModel(db),
			MerchantEntitlementModel: model.NewMerchantEntitlementModel(db),
			VIPModel:                 model.NewVIPModel(db),
			MessageModel:             model.NewMessageModel(db),
			SearchLogModel:           model.NewSearchLogModel(db),
			OperationLogModel:        model.NewOperationLogModel(db),
			AdminPermissionModel:     model.NewAdminPermissionModel(db),
		},
		AdminAuth: middleware.NewAdminAuthMiddleware(&fakeAdminTokenService{subject: subject}).Handle,
	}, mock
}

func TestEntitlementAndVIPHandlersThroughGeneratedRoutes(t *testing.T) {
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{})
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "list entitlements", method: http.MethodGet, target: "/api/v1/merchants/merchant-1/entitlements"},
		{name: "list entitlement usage", method: http.MethodGet, target: "/api/v1/merchants/merchant-1/entitlements/entitlement-1/usage-records"},
		{name: "list top vouchers", method: http.MethodGet, target: "/api/v1/merchants/merchant-1/top-vouchers"},
		{name: "redeem top voucher", method: http.MethodPost, target: "/api/v1/top-vouchers/voucher-1/redeem", body: `{"resourceId":"resource-1"}`},
		{name: "list vip plans", method: http.MethodGet, target: "/api/v1/vip/plans"},
		{name: "list quota packs", method: http.MethodGet, target: "/api/v1/vip/quota-packs"},
		{name: "get merchant vip", method: http.MethodGet, target: "/api/v1/merchants/merchant-1/vip"},
		{name: "create vip order", method: http.MethodPost, target: "/api/v1/merchants/merchant-1/vip/orders", body: `{"planCode":"monthly"}`},
		{name: "create vip payment", method: http.MethodPost, target: "/api/v1/merchants/merchant-1/vip/orders/order-1/payment", body: `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			body := assertTask6GeneratedStatus(t, server, req, http.StatusInternalServerError)
			if body["msg"] == "接口暂不可用，请稍后重试" {
				t.Fatalf("route %s %s still uses migration skeleton", tc.method, tc.target)
			}
		})
	}
}

func TestVIPPublicGeneratedRoutesStayAnonymousAndKeepEmptyArrays(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{APIStore: &svc.APIStore{VIPModel: model.NewVIPModel(db)}})

	mock.ExpectQuery(`(?s)FROM vip_plans p`).WillReturnRows(sqlmock.NewRows([]string{"code"}))
	plans := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/vip/plans", nil), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := plans["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("plans=%#v, want anonymous items: []", plans)
	}

	mock.ExpectQuery(`(?s)FROM vip_quota_packs`).WillReturnRows(sqlmock.NewRows([]string{"code"}))
	packs := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/vip/quota-packs", nil), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := packs["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("packs=%#v, want anonymous items: []", packs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("public VIP SQL expectations: %v", err)
	}
}

func TestEntitlementAndVIPPrivateGeneratedRoutesRequireMerchantPermission(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{
		APIStore: &svc.APIStore{
			UserModel:                model.NewUserModel(db),
			MerchantEntitlementModel: model.NewMerchantEntitlementModel(db),
			VIPModel:                 model.NewVIPModel(db),
		},
		UserTokenService:  &fakeUserTokenService{},
		AdminTokenService: adminauth.NewValidatingAdminTokenService(nil, nil),
	})

	tests := []struct {
		name         string
		method       string
		target       string
		body         string
		voucherOwner bool
	}{
		{name: "list entitlements", method: http.MethodGet, target: "/api/v1/merchants/merchant-2/entitlements"},
		{name: "usage records", method: http.MethodGet, target: "/api/v1/merchants/merchant-2/entitlements/entitlement-1/usage-records"},
		{name: "top vouchers", method: http.MethodGet, target: "/api/v1/merchants/merchant-2/top-vouchers"},
		{name: "redeem voucher", method: http.MethodPost, target: "/api/v1/top-vouchers/voucher-1/redeem", body: `{"merchantId":"merchant-1","resourceId":"resource-1"}`, voucherOwner: true},
		{name: "merchant vip", method: http.MethodGet, target: "/api/v1/merchants/merchant-2/vip"},
		{name: "create vip order", method: http.MethodPost, target: "/api/v1/merchants/merchant-2/vip/orders", body: `{"userId":"attacker","planCode":"monthly"}`},
		{name: "create vip payment", method: http.MethodPost, target: "/api/v1/merchants/merchant-2/vip/orders/order-1/payment", body: `{"userId":"attacker"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.voucherOwner {
				mock.ExpectQuery(`(?s)SELECT merchant_id::text.*FROM merchant_entitlements`).
					WithArgs("voucher-1", model.EntitlementTypeTopVoucher).
					WillReturnRows(sqlmock.NewRows([]string{"merchant_id"}).AddRow("merchant-2"))
			}
			mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
				WithArgs("user-1", "merchant-2").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			body := assertTask6GeneratedStatus(t, server, authenticatedRequest(tc.method, tc.target, tc.body), http.StatusForbidden)
			if body["errorCode"] != errx.CodeForbidden {
				t.Fatalf("body=%#v, want merchant permission denial", body)
			}
		})
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("private entitlement/VIP SQL expectations: %v", err)
	}
}

func TestMerchantHandlersThroughGeneratedRoutes(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	expectTask6GeneratedMerchantDetail(mock, "merchant-1")
	publicEnvelope := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1", nil), http.StatusOK)
	publicData, ok := publicEnvelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("merchant response = %#v, want data object", publicEnvelope)
	}
	if _, exposed := publicData["contact"]; exposed {
		t.Fatalf("anonymous merchant detail exposed contact: %#v", publicData["contact"])
	}

	badTokenReq := httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1", nil)
	badTokenReq.Header.Set("Authorization", "Bearer expired-token")
	assertTask6GeneratedStatus(t, server, badTokenReq, http.StatusUnauthorized)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-2").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodPost, "/api/v1/merchants/merchant-2", `{"name":"普通商家"}`), http.StatusForbidden)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodPost, "/api/v1/merchants/merchant-1", `{"name":"官方童装店"}`), http.StatusBadRequest)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("merchant generated SQL expectations: %v", err)
	}
}

func TestMerchantGeneratedRoutesExposeEditableContactOnlyToManager(t *testing.T) {
	cases := []struct {
		name          string
		canManage     bool
		contactWanted bool
	}{
		{name: "manager", canManage: true, contactWanted: true},
		{name: "non manager", canManage: false, contactWanted: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, mock := newTask6GeneratedServiceContext(t)
			server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
			expectTask6GeneratedMerchantDetail(mock, "merchant-1")
			mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
				WithArgs("user-1", "merchant-1").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(tc.canManage))

			envelope := assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/merchants/merchant-1", ""), http.StatusOK)
			data := envelope["data"].(map[string]interface{})
			contact, present := data["contact"]
			if present != tc.contactWanted {
				t.Fatalf("contact present=%v value=%#v, want present=%v", present, contact, tc.contactWanted)
			}
			if tc.contactWanted {
				contactData := contact.(map[string]interface{})
				if contactData["phone"] != "18800000002" || contactData["wechat"] != "stock-demo" {
					t.Fatalf("contact=%#v, want editable phone/wechat", contactData)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("merchant contact SQL expectations: %v", err)
			}
		})
	}
}

func TestMerchantUpdateSucceedsThroughGeneratedRoute(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT.*merchant_type.*FOR UPDATE`).
		WithArgs("merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_type"}).AddRow("factory"))
	mock.ExpectExec(`(?s)UPDATE merchants`).
		WithArgs(
			"merchant-1", "晨星新名", []byte(`["童装","配饰"]`), "factory", "新简介", "https://img.example.com/new-logo.jpg",
			[]byte(`["https://img.example.com/new-a.jpg"]`), sqlmock.AnyArg(), "李经理", "18800000003", "wechat-new", "织里新地址", true,
			[]byte(`{"latitude":30.2,"longitude":120.3}`),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := authenticatedRequest(http.MethodPost, "/api/v1/merchants/merchant-1?merchantId=merchant-2", `{
		"name":" 晨星新名 ",
		"mainCategories":["童装","配饰"],
		"merchantType":"factory",
		"description":" 新简介 ",
		"logoUrl":"https://img.example.com/new-logo.jpg",
		"images":["https://img.example.com/new-a.jpg"],
		"contactName":"李经理",
		"contactPhone":"18800000003",
		"contactWechat":"wechat-new",
		"addressText":"织里新地址",
		"location":{"latitude":30.2,"longitude":120.3}
	}`)
	envelope := assertTask6GeneratedStatus(t, server, req, http.StatusOK)
	data := envelope["data"].(map[string]interface{})
	if data["id"] != "merchant-1" || strings.TrimSpace(data["updatedAt"].(string)) == "" {
		t.Fatalf("data=%#v, want path merchant id and updatedAt", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("merchant update SQL expectations: %v", err)
	}
}

func expectTask6GeneratedMerchantDetail(mock sqlmock.Sqlmock, merchantID string) {
	mock.ExpectQuery(`(?s)FROM merchants m.*GROUP BY m.id, cs.code`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "merchant_no", "name", "merchant_type", "city_code", "main_categories", "profile_status", "vip_status",
			"contact_name", "contact_phone", "contact_wechat", "address_text", "location", "description", "logo_url", "images",
			"last_active_at", "published_count", "dealt_count", "follower_count",
		}).AddRow(
			merchantID, "M001", "晨星童装", "factory", "zhili", []byte(`["童装"]`), "completed", "none",
			"周经理", "18800000002", "stock-demo", "织里镇", []byte(`{"latitude":30.1}`), "童装工厂", "https://img.example.com/logo.jpg", []byte(`["https://img.example.com/a.jpg"]`),
			nil, int64(2), int64(1), int64(3),
		))
	mock.ExpectQuery(`(?s)FROM credit_records`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows([]string{"tag_code", "tag_label"}))
}

func TestMessageHandlersEnforceRoleScopeThroughGeneratedRoutes(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`(?s)FROM messages`).
		WithArgs("user-1", pq.Array([]string{"merchant:merchant-1"}), "review", "unread", int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "message_type", "trigger_id", "title", "content", "target_url", "status", "created_at", "total",
		}))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/messages?userId=attacker&roleCode=merchant:merchant-1&type=review&status=unread&page=1&pageSize=20", ""), http.StatusOK)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-2").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodPost, "/api/v1/messages/message-2/read", `{"userId":"attacker","roleCode":"merchant:merchant-2"}`), http.StatusForbidden)

	mock.ExpectQuery(`(?s)SELECT m.id::text.*FROM merchant_admin_bindings mab`).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("merchant-1").AddRow("merchant-3"))
	mock.ExpectQuery(`(?s)UPDATE messages`).
		WithArgs("message-1", "user-1", pq.Array([]string{"merchant:merchant-1", "merchant:merchant-3"})).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("message-1", "read"))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodPost, "/api/v1/messages/message-1/read", `{"userId":"attacker"}`), http.StatusOK)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("message generated SQL expectations: %v", err)
	}
}

func TestMessageGeneratedRoutesRejectClientControlledInvalidRoleCodes(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		target   string
		body     string
		roleCode string
	}{
		{name: "list non merchant role", method: http.MethodGet, target: "/api/v1/messages?roleCode=platform_operator", roleCode: "platform_operator"},
		{name: "list empty merchant role", method: http.MethodGet, target: "/api/v1/messages?roleCode=merchant:", roleCode: "merchant:"},
		{name: "list malformed merchant role", method: http.MethodGet, target: "/api/v1/messages?roleCode=merchant:merchant-1:extra", roleCode: "merchant:merchant-1:extra"},
		{name: "list merchant role with inner whitespace", method: http.MethodGet, target: "/api/v1/messages?roleCode=merchant:%20merchant-1", roleCode: "merchant: merchant-1"},
		{name: "read non merchant role", method: http.MethodPost, target: "/api/v1/messages/message-1/read", body: `{"roleCode":"platform_operator"}`, roleCode: "platform_operator"},
		{name: "read empty merchant role", method: http.MethodPost, target: "/api/v1/messages/message-1/read", body: `{"roleCode":"merchant:"}`, roleCode: "merchant:"},
		{name: "read malformed merchant role", method: http.MethodPost, target: "/api/v1/messages/message-1/read", body: `{"roleCode":"merchant:merchant-1:extra"}`, roleCode: "merchant:merchant-1:extra"},
		{name: "read merchant role with inner whitespace", method: http.MethodPost, target: "/api/v1/messages/message-1/read", body: `{"roleCode":"merchant: merchant-1"}`, roleCode: "merchant: merchant-1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, mock := newTask6GeneratedServiceContext(t)
			server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
			req := authenticatedRequest(tc.method, tc.target, tc.body)
			body := assertTask6GeneratedStatus(t, server, req, http.StatusForbidden)
			if body["errorCode"] != errx.CodeForbidden {
				t.Fatalf("roleCode=%q body=%#v, want forbidden", tc.roleCode, body)
			}
			// 非法客户端角色必须在 Handler 边界拒绝，不能进入消息 ANY 查询或更新。
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("invalid roleCode reached SQL: %v", err)
			}
		})
	}
}

func TestMessageGeneratedRoutesRejectMissingDependencies(t *testing.T) {
	var typedNilUserTokenService *fakeUserTokenService
	cases := []struct {
		name   string
		mutate func(*svc.ServiceContext)
	}{
		{name: "nil store", mutate: func(svcCtx *svc.ServiceContext) { svcCtx.APIStore = nil }},
		{name: "nil user token service", mutate: func(svcCtx *svc.ServiceContext) { svcCtx.UserTokenService = nil }},
		{name: "typed nil user token service", mutate: func(svcCtx *svc.ServiceContext) { svcCtx.UserTokenService = typedNilUserTokenService }},
	}
	requests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "list", method: http.MethodGet, target: "/api/v1/messages"},
		{name: "read", method: http.MethodPost, target: "/api/v1/messages/message-1/read", body: `{}`},
	}

	for _, dependencyCase := range cases {
		for _, requestCase := range requests {
			t.Run(dependencyCase.name+"/"+requestCase.name, func(t *testing.T) {
				svcCtx, _ := newTask6GeneratedServiceContext(t)
				dependencyCase.mutate(svcCtx)
				server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
				body := assertTask6GeneratedStatus(t, server, authenticatedRequest(requestCase.method, requestCase.target, requestCase.body), http.StatusInternalServerError)
				if body["errorCode"] != errx.CodeInternalError {
					t.Fatalf("body=%#v, want internal dependency error", body)
				}
			})
		}
	}
}

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
	if publicContact, ok := publicData["contact"]; ok {
		t.Fatalf("public contact = %#v, public merchant detail should omit contact entirely", publicContact)
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

/*
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
*/

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

func TestDiscoveryPublicEndpointsThroughGeneratedRoutes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 discovery sqlmock 失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	apiStore := &svc.APIStore{
		BannerTopicModel:      model.NewBannerTopicModel(db),
		ResourceModel:         model.NewResourceModel(db),
		MerchantModel:         model.NewMerchantModel(db),
		HotSearchKeywordModel: model.NewHotSearchKeywordModel(db),
	}
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{APIStore: apiStore})
	now := time.Date(2026, time.August, 5, 10, 0, 0, 0, time.UTC)

	t.Run("首页运营配置", func(t *testing.T) {
		mock.ExpectQuery(`(?s)FROM banner_topics bt.*bt.kind IN`).
			WithArgs("zhili").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "city_code", "kind", "title", "subtitle", "cover_url", "type_scope", "jump_type",
				"jump_target", "tags", "start_at", "end_at", "sort_order", "status", "created_at", "updated_at",
			}).AddRow(
				"banner-1", "zhili", "banner", "现货活动", "今日上新", "https://img.example/banner.jpg", `[]`, "topic",
				"topic-1", `["现货"]`, time.Unix(0, 0), time.Unix(0, 0), int64(10), "active", now, now,
			))

		data := serveGeneratedRequest(t, server, http.MethodGet, "/api/v1/home/operation-config?cityCode=zhili", "")
		banners := data["banners"].([]interface{})
		if len(banners) != 1 || banners[0].(map[string]interface{})["id"] != "banner-1" {
			t.Fatalf("banners = %#v, want generated banner response", banners)
		}
		if cards, ok := data["recommendCards"].([]interface{}); !ok || len(cards) != 0 {
			t.Fatalf("recommendCards = %#v, want []", data["recommendCards"])
		}
	})

	t.Run("首页资源", func(t *testing.T) {
		mock.ExpectQuery(`(?s)FROM resources r`).
			WithArgs("published", "zhili", "", "", "", "", "", "", int64(30), int64(0), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "direction", "type_code", "type_name", "title", "category", "cover_url", "district",
				"price_text", "quantity_text", "tags", "merchant_id", "merchant_name", "vip_status", "refreshed_at", "dealt_at", "total",
			}).AddRow(
				"resource-1", "supply", "inventory", "库存现货", "秋款现货", "童装", "", "织里", "面议", "1000 件", `[]`,
				"merchant-1", "小鹿童装", model.VIPStatusNone, now, nil, int64(1),
			))

		data := serveGeneratedRequest(t, server, http.MethodGet, "/api/v1/home/resources?cityCode=zhili", "")
		items := data["items"].([]interface{})
		if len(items) != 1 || items[0].(map[string]interface{})["id"] != "resource-1" {
			t.Fatalf("items = %#v, want generated home resource", items)
		}
		if tags, ok := items[0].(map[string]interface{})["creditTags"].([]interface{}); !ok || len(tags) != 0 {
			t.Fatalf("creditTags = %#v, want []", items[0].(map[string]interface{})["creditTags"])
		}
	})

	t.Run("最近商家", func(t *testing.T) {
		mock.ExpectQuery(`(?s)FROM merchants m`).
			WithArgs("zhili", int64(6)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "merchant_type", "main_categories", "logo_url", "address_text", "onboarded_at",
			}).AddRow("merchant-1", "小鹿童装", "stall", `["女童"]`, "", "织里童装城", now))

		data := serveGeneratedRequest(t, server, http.MethodGet, "/api/v1/home/recent-merchants?cityCode=zhili", "")
		items := data["items"].([]interface{})
		if len(items) != 1 || items[0].(map[string]interface{})["name"] != "小鹿童装" {
			t.Fatalf("items = %#v, want generated recent merchant", items)
		}
	})

	t.Run("热词", func(t *testing.T) {
		mock.ExpectQuery(`(?s)FROM hot_search_keywords hsk`).
			WithArgs("zhili").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "city_code", "keyword", "sort_order", "status", "start_at", "end_at", "created_at", "updated_at",
			}).AddRow("keyword-1", "zhili", "夏款现货", int64(10), "active", time.Unix(0, 0), time.Unix(0, 0), now, now))

		data := serveGeneratedRequest(t, server, http.MethodGet, "/api/v1/search/hot-keywords?cityCode=zhili", "")
		items := data["items"].([]interface{})
		if len(items) != 1 || items[0].(map[string]interface{})["keyword"] != "夏款现货" {
			t.Fatalf("items = %#v, want generated hot keyword", items)
		}
	})

	t.Run("专题资源", func(t *testing.T) {
		mock.ExpectQuery(`(?s)FROM banner_topics bt.*bt.id =`).
			WithArgs("topic-1", "zhili").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "city_code", "kind", "title", "subtitle", "cover_url", "type_scope", "jump_type",
				"jump_target", "tags", "start_at", "end_at", "sort_order", "status", "created_at", "updated_at",
			}).AddRow(
				"topic-1", "zhili", "topic", "专题", "本周精选", "", `["inventory"]`, "internal",
				"/pages/search/index", `["现货"]`, time.Unix(0, 0), time.Unix(0, 0), int64(10), "active", now, now,
			))
		mock.ExpectQuery(`(?s)FROM resources r`).
			WithArgs("published", "zhili", "", "", "inventory", "", "", "", int64(5), int64(5), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "direction", "type_code", "type_name", "title", "category", "cover_url", "district",
				"price_text", "quantity_text", "tags", "merchant_id", "merchant_name", "vip_status", "refreshed_at", "dealt_at", "total",
			}))

		data := serveGeneratedRequest(t, server, http.MethodGet, "/api/v1/topics/topic-1/resources?cityCode=zhili&page=2&pageSize=5", "")
		topic := data["topic"].(map[string]interface{})
		if topic["id"] != "topic-1" || data["page"] != float64(2) || data["pageSize"] != float64(5) {
			t.Fatalf("data = %#v, want topic path and generated pagination", data)
		}
		if items, ok := data["items"].([]interface{}); !ok || len(items) != 0 {
			t.Fatalf("items = %#v, want []", data["items"])
		}
	})

	t.Run("webview 校验", func(t *testing.T) {
		data := serveGeneratedRequest(t, server, http.MethodPost, "/api/v1/webview/validate", `{"url":"https://www.wplink.cn/activity"}`)
		if data["allowed"] != true || data["url"] != "https://www.wplink.cn/activity" {
			t.Fatalf("data = %#v, want allowed generated webview response", data)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("discovery SQL expectations not met: %v", err)
	}
}

func serveGeneratedRequest(t *testing.T, server http.Handler, method string, path string, body string) map[string]interface{} {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	server.ServeHTTP(rec, req)
	return decodeEnvelopeData(t, rec, http.StatusOK)
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
	}
}

type fakeFullAPIStore struct {
	fakeResourceAPIStore
	updateMerchantPatch       model.UpdateMerchantPatch
	messageFilter             model.ListMessagesFilter
	readMessageUserID         string
	readMessageRoleCode       string
	readMessageRoleCodes      []string
	redeemVoucherID           string
	redeemResourceID          string
	topVoucherMerchantIDs     map[string]string
	grantEntitlementInput     model.GrantEntitlementInput
	createVIPOrderInput       model.CreateVIPOrderInput
	createQuotaPackOrderInput model.CreateQuotaPackOrderInput
	createVIPPaymentInput     model.CreateVIPPaymentOrderInput
	markVIPOrderPaidInput     model.MarkVIPOrderPaidInput
	saveVIPPlanInput          model.SaveAdminVIPPlanInput
	saveQuotaPackInput        model.SaveAdminQuotaPackInput
	saveVIPPromotionInput     model.SaveAdminVIPPromotionInput
	adminOperatorFilter       model.AdminOperatorFilter
	createAdminOperatorInput  model.AdminOperatorInput
	updateAdminOperatorInput  model.AdminOperatorInput
	statusAdminOperatorInput  model.AdminOperatorStatusInput
	adminRoleModules          []string
	adminRoleModuleInput      model.AdminRoleModulePermissionInput
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
	return model.MerchantDetail{ID: merchantID, Name: "织里云仓", MerchantType: "stockist", CityCode: "zhili", MainCategories: []string{"童装"}, ContactName: "周经理", ContactPhone: "18800000002", ContactWechat: "stock-demo", PhoneMasked: "188****0002", WechatMasked: "stock-demo", AddressText: "织里镇利济路88号", Location: model.JSONMap{"latitude": 30.1, "longitude": 120.2, "name": "织里童装城", "address": "织里镇利济路88号"}, PublishedCount: 1}, nil
}

func (s *fakeFullAPIStore) UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error) {
	s.updateMerchantPatch = patch
	return "2026-06-28T10:00:00Z", nil
}

func (s *fakeFullAPIStore) ListMerchants(ctx context.Context, filter model.ListMerchantsFilter) (model.ListMerchantsResult, error) {
	return model.ListMerchantsResult{Items: []model.MerchantListItem{{ID: "merchant-1", Name: "织里云仓", MerchantType: "stockist", Status: model.MerchantStatusActive}}, Page: filter.Page, PageSize: filter.PageSize, Total: 1}, nil
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

func (s *fakeFullAPIStore) ListHomeRecentMerchants(ctx context.Context, cityCode string, limit int64) ([]model.HomeRecentMerchant, error) {
	return []model.HomeRecentMerchant{{
		ID:             "merchant-1",
		Name:           "小鹿童装",
		MerchantType:   "stall",
		MainCategories: []string{"女童"},
		AddressText:    "织里童装城 A 区 101",
		OnboardedAt:    "2026-07-31T10:00:00Z",
	}}, nil
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

/*
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
*/

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
