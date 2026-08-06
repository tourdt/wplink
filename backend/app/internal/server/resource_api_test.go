package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"wplink/backend/app/internal/config"
	callbackhandler "wplink/backend/app/internal/handler/callback"
	"wplink/backend/app/internal/logic/adminauth"
	contentauditlogic "wplink/backend/app/internal/logic/contentaudit"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestResourceInvalidRelatedIDThroughGeneratedRoute(t *testing.T) {
	svcCtx, _ := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	body := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/resources/not-a-bigint/related", nil), http.StatusNotFound)
	if body["errorCode"] != errx.CodeResourceNotFound || body["msg"] != "资源不存在或暂不可查看" {
		t.Fatalf("body=%#v, want hidden invalid related resource", body)
	}
}

func TestResourceHandlersThroughGeneratedRoutesDoNotUseMigrationSkeleton(t *testing.T) {
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{})
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "create", method: http.MethodPost, target: "/api/v1/resources", body: `{}`},
		{name: "create draft", method: http.MethodPost, target: "/api/v1/resources/drafts", body: `{}`},
		{name: "update draft", method: http.MethodPut, target: "/api/v1/resources/1/draft", body: `{}`},
		{name: "submit", method: http.MethodPost, target: "/api/v1/resources/1/submit", body: `{}`},
		{name: "list", method: http.MethodGet, target: "/api/v1/resources"},
		{name: "related", method: http.MethodGet, target: "/api/v1/resources/1/related"},
		{name: "search", method: http.MethodGet, target: "/api/v1/resource-search"},
		{name: "my list", method: http.MethodGet, target: "/api/v1/me/resources?merchantId=1"},
		{name: "own detail", method: http.MethodGet, target: "/api/v1/me/resources/1/detail"},
		{name: "editable", method: http.MethodGet, target: "/api/v1/me/resources/1/edit"},
		{name: "public detail", method: http.MethodGet, target: "/api/v1/resources/1"},
		{name: "report", method: http.MethodPost, target: "/api/v1/resources/1/reports", body: `{}`},
		{name: "detail view", method: http.MethodPost, target: "/api/v1/resources/1/detail-view"},
		{name: "refresh", method: http.MethodPost, target: "/api/v1/resources/1/refresh"},
		{name: "deal", method: http.MethodPost, target: "/api/v1/resources/1/deal-feedback", body: `{}`},
		{name: "take down", method: http.MethodPost, target: "/api/v1/resources/1/take-down", body: `{}`},
		{name: "delete", method: http.MethodDelete, target: "/api/v1/resources/1", body: `{}`},
		{name: "repost", method: http.MethodPost, target: "/api/v1/resources/1/repost-similar"},
		{name: "contact", method: http.MethodPost, target: "/api/v1/resources/1/contact-events", body: `{}`},
		{name: "unlock order", method: http.MethodPost, target: "/api/v1/resources/1/contact-unlock-orders", body: `{}`},
		{name: "unlock payment", method: http.MethodPost, target: "/api/v1/resources/1/contact-unlock-orders/1/payment", body: `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			body := assertTask6GeneratedStatus(t, server, req, http.StatusInternalServerError)
			if body["msg"] == "接口暂不可用，请稍后重试" {
				t.Fatalf("route %s %s still uses migration skeleton", tc.method, tc.target)
			}
		})
	}
}

func TestResourceOwnerActionsThroughGeneratedRoutesUseStoredOwner(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "update draft", method: http.MethodPut, target: "/api/v1/resources/resource-2/draft", body: `{}`},
		{name: "submit", method: http.MethodPost, target: "/api/v1/resources/resource-2/submit", body: `{}`},
		{name: "own detail", method: http.MethodGet, target: "/api/v1/me/resources/resource-2/detail"},
		{name: "editable", method: http.MethodGet, target: "/api/v1/me/resources/resource-2/edit"},
		{name: "refresh", method: http.MethodPost, target: "/api/v1/resources/resource-2/refresh"},
		{name: "deal", method: http.MethodPost, target: "/api/v1/resources/resource-2/deal-feedback", body: `{"isDealt":true}`},
		{name: "take down", method: http.MethodPost, target: "/api/v1/resources/resource-2/take-down", body: `{"reason":"已售罄"}`},
		{name: "delete", method: http.MethodDelete, target: "/api/v1/resources/resource-2"},
		{name: "repost", method: http.MethodPost, target: "/api/v1/resources/resource-2/repost-similar"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, mock := newTask6GeneratedServiceContext(t)
			server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
			mock.ExpectQuery(`(?s)SELECT merchant_id::text.*FROM resources`).
				WithArgs("resource-2").
				WillReturnRows(sqlmock.NewRows([]string{"merchant_id"}).AddRow("merchant-2"))
			mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
				WithArgs("user-1", "merchant-2").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			body := assertTask6GeneratedStatus(t, server, authenticatedRequest(tc.method, tc.target, tc.body), http.StatusForbidden)
			if body["errorCode"] != errx.CodeForbidden {
				t.Fatalf("body=%#v, want stored owner permission denial", body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("owner permission SQL expectations: %v", err)
			}
		})
	}
}

func TestResourcePublicListThroughGeneratedRouteKeepsEmptyArrayAndPagination(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
	mock.ExpectQuery(`(?s)FROM resources r`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	body := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/resources?page=2&pageSize=3&tags=%E6%80%A5%E6%B8%85,%E6%94%AF%E6%8C%81%E7%9C%8B%E8%B4%A7", nil), http.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body=%#v, want data object", body)
	}
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 0 || data["page"] != float64(2) || data["pageSize"] != float64(3) || data["total"] != float64(0) {
		t.Fatalf("data=%#v, want empty items with requested pagination", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("public list SQL expectations: %v", err)
	}
}

func TestContentAuditCallbackVerificationThroughGeneratedRoute(t *testing.T) {
	verifier := &fakeWechatCallbackVerifier{}
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{ContentAuditCallbackVerifier: verifier})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wechat/content-audit/media-callback?signature=sig&timestamp=1784971200&nonce=nonce&echostr=challenge", nil)

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "challenge" || rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("status=%d contentType=%q body=%q, want raw callback challenge", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	if _, remembered := verifier.snapshot(); remembered {
		t.Fatal("URL verification must not consume replay key")
	}
}

func TestWechatPayNotifyGeneratedRouteUsesProviderResponse(t *testing.T) {
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat-pay/notify", strings.NewReader(`{"id":"notify-1"}`))

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" || rec.Body.String() != `{"code":"FAIL","message":"处理失败"}` {
		t.Fatalf("status=%d contentType=%q body=%q, want provider failure response", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
}

func TestContentAuditCallbackGeneratedRouteRejectsBeforeBusinessProcessing(t *testing.T) {
	tests := []struct {
		name       string
		verifier   *fakeWechatCallbackVerifier
		appid      string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "bad signature", verifier: &fakeWechatCallbackVerifier{checkErr: errx.New(errx.CodeUnauthorized, "微信回调校验失败")}, appid: "wx-app", body: `{"appid":"wx-app"}`, wantStatus: http.StatusUnauthorized, wantCode: errx.CodeUnauthorized},
		{name: "oversized body", verifier: &fakeWechatCallbackVerifier{}, appid: "wx-app", body: strings.Repeat("x", (256<<10)+1), wantStatus: http.StatusBadRequest, wantCode: errx.CodeValidationFailed},
		{name: "appid mismatch", verifier: &fakeWechatCallbackVerifier{}, appid: "wx-app", body: `{"appid":"attacker-app","trace_id":"trace-1"}`, wantStatus: http.StatusUnauthorized, wantCode: errx.CodeUnauthorized},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{
				Config:                       config.Config{Wechat: config.WechatConfig{AppID: tc.appid}},
				ContentAuditCallbackVerifier: tc.verifier,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/content-audit/media-callback?signature=sensitive-signature&timestamp=1784971200&nonce=sensitive-nonce", strings.NewReader(tc.body))
			body := assertTask6GeneratedStatus(t, server, req, tc.wantStatus)
			if body["errorCode"] != tc.wantCode {
				t.Fatalf("body=%#v, want errorCode=%s", body, tc.wantCode)
			}
			serialized, _ := json.Marshal(body)
			for _, secret := range []string{"sensitive-signature", "sensitive-nonce", "attacker-app"} {
				if strings.Contains(string(serialized), secret) {
					t.Fatalf("callback response leaked sensitive value %q: %s", secret, serialized)
				}
			}
			calls, _ := tc.verifier.snapshot()
			if len(calls) != 1 || calls[0] {
				t.Fatalf("verifier calls=%v, failed callback must not record replay fingerprint", calls)
			}
		})
	}
}

func TestContentAuditCallbackReplayThroughGeneratedRouteReturnsSuccess(t *testing.T) {
	verifier := &fakeWechatCallbackVerifier{checkErr: contentauditlogic.ErrWechatCallbackReplay}
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{ContentAuditCallbackVerifier: verifier})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wechat/content-audit/media-callback?signature=sig&timestamp=1784971200&nonce=nonce", strings.NewReader(`{"appid":"wx-app"}`))

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "success" {
		t.Fatalf("status=%d body=%q, replay must use provider success acknowledgement", rec.Code, rec.Body.String())
	}
	calls, _ := verifier.snapshot()
	if len(calls) != 1 || calls[0] {
		t.Fatalf("verifier calls=%v, replay must stop after check-only verification", calls)
	}
}

func TestContentAuditCallbackGeneratedRouteRecordsReplayOnlyAfterLogicSuccess(t *testing.T) {
	t.Run("logic failure remains retryable", func(t *testing.T) {
		svcCtx, mock, verifier := newContentAuditCallbackGeneratedContext(t)
		server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
		mock.ExpectBegin()
		mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).
			WillReturnError(errors.New("raw database secret callback failure"))
		mock.ExpectRollback()

		body := assertTask6GeneratedStatus(t, server, contentAuditCallbackRequest(`{"appid":"wx-app","trace_id":"trace-fail","errcode":0}`), http.StatusInternalServerError)
		if body["msg"] == "raw database secret callback failure" || strings.Contains(fmt.Sprint(body), "trace-fail") {
			t.Fatalf("callback error leaked internal or body detail: %#v", body)
		}
		calls, _ := verifier.snapshot()
		if len(calls) != 1 || calls[0] {
			t.Fatalf("verifier calls=%v, failed logic must remain retryable", calls)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("logic failure SQL expectations: %v", err)
		}
	})

	t.Run("state conflict is acknowledged and remembered", func(t *testing.T) {
		svcCtx, mock, verifier := newContentAuditCallbackGeneratedContext(t)
		server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
		mock.ExpectBegin()
		mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, contentAuditCallbackRequest(`{"appid":"wx-app","trace_id":"trace-conflict","errcode":0}`))
		if rec.Code != http.StatusOK || rec.Body.String() != "success" {
			t.Fatalf("status=%d body=%q, state conflict must acknowledge success", rec.Code, rec.Body.String())
		}
		calls, _ := verifier.snapshot()
		if len(calls) != 2 || calls[0] || !calls[1] {
			t.Fatalf("verifier calls=%v, want check then remember", calls)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("state conflict SQL expectations: %v", err)
		}
	})

	t.Run("successful pending result is remembered last", func(t *testing.T) {
		svcCtx, mock, verifier := newContentAuditCallbackGeneratedContext(t)
		server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
		mock.ExpectBegin()
		mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).
			WillReturnRows(sqlmock.NewRows([]string{"resource_id"}).AddRow("resource-1"))
		mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER`).
			WithArgs("resource-1").
			WillReturnRows(sqlmock.NewRows([]string{"pending", "rejected", "failed"}).AddRow(int64(1), int64(0), int64(0)))
		mock.ExpectCommit()
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, contentAuditCallbackRequest(`{"appid":"wx-app","trace_id":"trace-success","errcode":0}`))
		if rec.Code != http.StatusOK || rec.Body.String() != "success" {
			t.Fatalf("status=%d body=%q, successful callback must acknowledge success", rec.Code, rec.Body.String())
		}
		calls, _ := verifier.snapshot()
		if len(calls) != 2 || calls[0] || !calls[1] {
			t.Fatalf("verifier calls=%v, want check then remember", calls)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("success SQL expectations: %v", err)
		}
	})
}

func TestContentAuditCallbackGeneratedRouteAcknowledgesReplayWhileRemembering(t *testing.T) {
	tests := []struct {
		name         string
		prepareLogic func(sqlmock.Sqlmock)
		traceID      string
	}{
		{
			name: "logic success",
			prepareLogic: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).
					WillReturnRows(sqlmock.NewRows([]string{"resource_id"}).AddRow("resource-1"))
				mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER`).
					WithArgs("resource-1").
					WillReturnRows(sqlmock.NewRows([]string{"pending", "rejected", "failed"}).AddRow(int64(1), int64(0), int64(0)))
				mock.ExpectCommit()
			},
			traceID: "trace-success-remember-replay",
		},
		{
			name: "state conflict",
			prepareLogic: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			traceID: "trace-conflict-remember-replay",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, mock, verifier := newContentAuditCallbackGeneratedContext(t)
			verifier.rememberErr = contentauditlogic.ErrWechatCallbackReplay
			tc.prepareLogic(mock)
			server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, contentAuditCallbackRequest(`{"appid":"wx-app","trace_id":"`+tc.traceID+`","errcode":0}`))

			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" || rec.Body.String() != "success" {
				t.Fatalf("status=%d contentType=%q body=%q, remember replay must acknowledge success", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
			}
			calls, _ := verifier.snapshot()
			if len(calls) != 2 || calls[0] || !calls[1] {
				t.Fatalf("verifier calls=%v, want check then remember", calls)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("callback SQL expectations: %v", err)
			}
		})
	}
}

func TestContentAuditCallbackHandlerConcurrentDuplicateRequestsBothSucceed(t *testing.T) {
	svcCtx, mock, verifier := newContentAuditCallbackGeneratedContext(t)
	verifier.checkBarrier = newCallbackCheckBarrier(2)
	verifier.replayConcurrentRemember = true
	mock.MatchExpectationsInOrder(false)

	// 两个请求都通过第一阶段验签后，一个事务完成有效流转，另一个事务观察到幂等状态冲突。
	mock.ExpectBegin()
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).
		WillReturnRows(sqlmock.NewRows([]string{"resource_id"}).AddRow("resource-1"))
	mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER`).
		WithArgs("resource-1").
		WillReturnRows(sqlmock.NewRows([]string{"pending", "rejected", "failed"}).AddRow(int64(1), int64(0), int64(0)))
	mock.ExpectCommit()
	mock.ExpectRollback()

	handler := callbackhandler.HandleContentAuditCallbackHandler(svcCtx)
	recorders := []*httptest.ResponseRecorder{httptest.NewRecorder(), httptest.NewRecorder()}
	var requests sync.WaitGroup
	requests.Add(len(recorders))
	for _, rec := range recorders {
		rec := rec
		go func() {
			defer requests.Done()
			handler.ServeHTTP(rec, contentAuditCallbackRequest(`{"appid":"wx-app","trace_id":"trace-concurrent","errcode":0}`))
		}()
	}
	requests.Wait()

	for index, rec := range recorders {
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" || rec.Body.String() != "success" {
			t.Fatalf("response[%d]: status=%d contentType=%q body=%q, concurrent duplicate must acknowledge success", index, rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
		}
	}
	calls, _ := verifier.snapshot()
	if len(calls) != 4 || calls[0] || calls[1] || !calls[2] || !calls[3] {
		t.Fatalf("verifier calls=%v, want both checks before both remember calls", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("concurrent callback SQL expectations: %v", err)
	}
}

func TestContentAuditCallbackGeneratedRouteDoesNotSwallowNonReplayRememberFailure(t *testing.T) {
	svcCtx, mock, verifier := newContentAuditCallbackGeneratedContext(t)
	verifier.rememberErr = errx.New(errx.CodeInternalError, "remember dependency unavailable")
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)UPDATE resource_content_audit_tasks`).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	body := assertTask6GeneratedStatus(t, server, contentAuditCallbackRequest(`{"appid":"wx-app","trace_id":"trace-remember-failure","errcode":0}`), http.StatusInternalServerError)

	if body["errorCode"] != errx.CodeInternalError {
		t.Fatalf("body=%#v, non-replay remember failure must stay retryable", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("remember failure SQL expectations: %v", err)
	}
}

func TestContentAuditCallbackGeneratedRouteRejectsNilLikeDependencies(t *testing.T) {
	var typedNilVerifier *fakeWechatCallbackVerifier
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{ContentAuditCallbackVerifier: typedNilVerifier})
	body := assertTask6GeneratedStatus(t, server, contentAuditCallbackRequest(`{"appid":"wx-app"}`), http.StatusInternalServerError)
	if body["errorCode"] != errx.CodeInternalError {
		t.Fatalf("body=%#v, want typed-nil verifier rejected safely", body)
	}
}

func newContentAuditCallbackGeneratedContext(t *testing.T) (*svc.ServiceContext, sqlmock.Sqlmock, *fakeWechatCallbackVerifier) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error=%v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	verifier := &fakeWechatCallbackVerifier{}
	return &svc.ServiceContext{
		Config:                       config.Config{Wechat: config.WechatConfig{AppID: "wx-app"}},
		APIStore:                     &svc.APIStore{ResourceModel: model.NewResourceModel(db)},
		ContentAuditCallbackVerifier: verifier,
	}, mock, verifier
}

func contentAuditCallbackRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/api/v1/wechat/content-audit/media-callback?signature=sig&timestamp=1784971200&nonce=nonce", strings.NewReader(body))
}

func TestMetricsHandlersThroughGeneratedRoutes(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)INSERT INTO resource_exposure_events`).
		WithArgs("resource-1", "", "visitor-1", "session-1", "search", int64(1200)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	exposureReq := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/exposures/batch", strings.NewReader(`{"userId":"attacker","visitorKey":"visitor-1","sessionId":"session-1","source":"search","items":[{"resourceId":"resource-1","visibleDurationMs":1200}]}`))
	exposureReq.Header.Set("Content-Type", "application/json")
	assertTask6GeneratedStatus(t, server, exposureReq, http.StatusOK)

	mock.ExpectExec(`(?s)INSERT INTO .*merchant_map_events`).
		WithArgs("", "101", "", "visitor-map", "session-map", "location_view", "merchant_location").
		WillReturnResult(sqlmock.NewResult(0, 1))
	assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(`{"merchantId":"101","visitorKey":"visitor-map","sessionId":"session-map","eventType":"location_view","source":"merchant_location"}`)), http.StatusOK)

	mock.ExpectQuery(`(?s)SELECT merchant_id::text.*FROM resources`).
		WithArgs("resource-1").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id"}).AddRow("merchant-1"))
	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`(?s)FROM resource_metrics_daily.*ORDER BY stat_date ASC`).
		WithArgs("resource-1", "2026-08-01", "2026-08-05").
		WillReturnRows(sqlmock.NewRows([]string{"stat_date", "exposure_count", "detail_view_count", "phone_click_count", "wechat_copy_count"}).
			AddRow("2026-08-05", int64(5), int64(2), int64(1), int64(1)))
	mock.ExpectQuery(`(?s)SUM\(deal_feedback_count\)`).
		WithArgs("resource-1", "2026-08-01", "2026-08-05").
		WillReturnRows(sqlmock.NewRows([]string{"deal_feedback_count"}).AddRow(int64(3)))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/resources/resource-1/metrics?from=2026-08-01&to=2026-08-05", ""), http.StatusOK)

	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER.*FROM resources`).
		WithArgs("merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"published", "expiring", "dealt"}).AddRow(int64(4), int64(1), int64(2)))
	mock.ExpectQuery(`(?s)SUM\(exposure_count\).*FROM resource_metrics_daily`).
		WithArgs("merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exposure", "detail", "contact"}).AddRow(int64(9), int64(5), int64(2)))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/merchants/merchant-1/metrics/summary", ""), http.StatusOK)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("metrics generated SQL expectations: %v", err)
	}
}

func TestMetricsPublicGeneratedRoutesRejectInvalidTokens(t *testing.T) {
	svcCtx, _ := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	requests := []struct {
		target string
		body   string
	}{
		{target: "/api/v1/metrics/exposures/batch", body: `{"visitorKey":"visitor-1","sessionId":"session-1","source":"search","items":[{"resourceId":"resource-1","visibleDurationMs":1200}]}`},
		{target: "/api/v1/metrics/merchant-map-events", body: `{"merchantId":"101","visitorKey":"visitor-map","sessionId":"session-map","eventType":"location_view","source":"merchant_location"}`},
	}
	for _, tc := range requests {
		req := httptest.NewRequest(http.MethodPost, tc.target, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer expired-token")
		assertTask6GeneratedStatus(t, server, req, http.StatusUnauthorized)
	}
}

func TestMetricsMerchantMapStrictJSONThroughGeneratedRoutes(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"merchantId":"101","visitorKey":"visitor-1","sessionId":"session-1","eventType":"location_view","source":"merchant_location","unexpected":true}`},
		{name: "second JSON value", body: validMerchantMapEventJSON("location_view") + ` {"merchantId":"102"}`},
		{name: "trailing garbage", body: validMerchantMapEventJSON("location_view") + ` trailing`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, mock := newTask6GeneratedServiceContext(t)
			server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(tc.body))
			body := assertTask6GeneratedStatus(t, server, req, http.StatusBadRequest)
			if body["errorCode"] != errx.CodeValidationFailed || body["msg"] != "请求参数格式不正确" {
				t.Fatalf("body=%#v, want strict JSON validation error", body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("invalid JSON reached map event SQL: %v", err)
			}
		})
	}
}

func TestMetricsGeneratedRoutesRejectMissingDependencies(t *testing.T) {
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{UserTokenService: &fakeUserTokenService{}})
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/resources/resource-1/metrics", ""), http.StatusInternalServerError)
}

func TestMetricsPermissionFailuresThroughGeneratedRoutes(t *testing.T) {
	t.Run("resource owner mismatch", func(t *testing.T) {
		svcCtx, mock := newTask6GeneratedServiceContext(t)
		server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
		mock.ExpectQuery(`(?s)SELECT merchant_id::text.*FROM resources`).
			WithArgs("resource-2").
			WillReturnRows(sqlmock.NewRows([]string{"merchant_id"}).AddRow("merchant-2"))
		mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
			WithArgs("user-1", "merchant-2").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		body := assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/resources/resource-2/metrics", ""), http.StatusForbidden)
		if body["errorCode"] != errx.CodeForbidden {
			t.Fatalf("body=%#v, want owner mismatch forbidden", body)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("resource owner mismatch SQL expectations: %v", err)
		}
	})

	t.Run("resource not found", func(t *testing.T) {
		svcCtx, mock := newTask6GeneratedServiceContext(t)
		server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
		mock.ExpectQuery(`(?s)SELECT merchant_id::text.*FROM resources`).
			WithArgs("missing-resource").
			WillReturnError(sql.ErrNoRows)

		body := assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/resources/missing-resource/metrics", ""), http.StatusNotFound)
		if body["errorCode"] != errx.CodeResourceNotFound || body["msg"] != "资源不存在或已下架" {
			t.Fatalf("body=%#v, want safe resource not found", body)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("resource not found SQL expectations: %v", err)
		}
	})

	t.Run("merchant metrics missing dependency", func(t *testing.T) {
		svcCtx, _ := newTask6GeneratedServiceContext(t)
		svcCtx.APIStore.ResourceMetricDailyModel = nil
		server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
		body := assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/merchants/merchant-1/metrics/summary", ""), http.StatusInternalServerError)
		if body["errorCode"] != errx.CodeInternalError || body["msg"] != "指标服务暂不可用，请稍后重试" {
			t.Fatalf("body=%#v, want safe metrics dependency error", body)
		}
	})
}

func TestMetricsMerchantMapBodyLimitThroughGeneratedRoute(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
	validBody := validMerchantMapEventJSON("location_view")
	body4096 := validBody + strings.Repeat(" ", 4096-len(validBody))

	mock.ExpectExec(`(?s)INSERT INTO .*merchant_map_events`).
		WithArgs("", "101", "", "visitor-1", "session-1", "location_view", "merchant_location").
		WillReturnResult(sqlmock.NewResult(0, 1))
	assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(body4096)), http.StatusOK)

	body4097 := body4096 + " "
	body := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(body4097)), http.StatusBadRequest)
	if body["errorCode"] != errx.CodeValidationFailed || body["msg"] != "地图行为请求内容过大" {
		t.Fatalf("body=%#v, want 4097-byte request rejected", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("map body limit SQL expectations: %v", err)
	}
}

func TestMetricsMerchantMapRateLimitThroughGeneratedRoute(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)
	body := validMerchantMapEventJSON("location_view")

	for attempt := 1; attempt <= 60; attempt++ {
		mock.ExpectExec(`(?s)INSERT INTO .*merchant_map_events`).
			WithArgs("", "101", "", "visitor-1", "session-1", "location_view", "merchant_location").
			WillReturnResult(sqlmock.NewResult(0, 1))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(body))
		req.RemoteAddr = "198.51.100.7:4321"
		assertTask6GeneratedStatus(t, server, req, http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/merchant-map-events", strings.NewReader(body))
	req.RemoteAddr = "198.51.100.7:4321"
	envelope := assertTask6GeneratedStatus(t, server, req, http.StatusTooManyRequests)
	if envelope["errorCode"] != errx.CodeRateLimited || envelope["msg"] != "操作频繁，请稍后再试" {
		t.Fatalf("body=%#v, want 61st request rate limited", envelope)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("map rate limit SQL expectations: %v", err)
	}
}

func validMerchantMapEventJSON(eventType string) string {
	return `{"merchantId":"101","visitorKey":"visitor-1","sessionId":"session-1","eventType":"` + eventType + `","source":"merchant_location"}`
}

type fakeWechatCallbackVerifier struct {
	mu          sync.Mutex
	err         error
	checkErr    error
	rememberErr error
	remember    bool
	calls       []bool

	checkBarrier             *callbackCheckBarrier
	replayConcurrentRemember bool
	remembered               bool
}

func (v *fakeWechatCallbackVerifier) Verify(signature string, timestamp string, nonce string, remember bool) error {
	v.mu.Lock()
	v.calls = append(v.calls, remember)
	v.remember = remember
	barrier := v.checkBarrier
	if remember && v.rememberErr != nil {
		err := v.rememberErr
		v.mu.Unlock()
		return err
	}
	if !remember && v.checkErr != nil {
		err := v.checkErr
		v.mu.Unlock()
		return err
	}
	if remember && v.replayConcurrentRemember {
		if v.remembered {
			v.mu.Unlock()
			return contentauditlogic.ErrWechatCallbackReplay
		}
		v.remembered = true
	}
	err := v.err
	v.mu.Unlock()
	if !remember && barrier != nil {
		barrier.arriveAndWait()
	}
	return err
}

func (v *fakeWechatCallbackVerifier) snapshot() ([]bool, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]bool(nil), v.calls...), v.remember
}

type callbackCheckBarrier struct {
	mu        sync.Mutex
	target    int
	arrived   int
	releaseCh chan struct{}
}

func newCallbackCheckBarrier(target int) *callbackCheckBarrier {
	return &callbackCheckBarrier{target: target, releaseCh: make(chan struct{})}
}

func (b *callbackCheckBarrier) arriveAndWait() {
	b.mu.Lock()
	b.arrived++
	if b.arrived == b.target {
		close(b.releaseCh)
	}
	releaseCh := b.releaseCh
	b.mu.Unlock()
	<-releaseCh
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
	managedMerchantErr               error
	resourceMerchantIDs              map[string]string
	resourceStatuses                 map[string]string
	updatedResourceID                string
	deletedResourceID                string
	completedAuditInput              model.ResourceContentAuditTaskResultInput
	auditCompletion                  model.ResourceContentAuditTaskCompletion
	publishedAuditResourceID         string
	publishedAuditTraceID            string
	rejectedAuditResourceID          string
	rejectedAuditTraceID             string
	retriedAuditTraceID              string
	auditRejectReason                string
	mapEventCalled                   bool
	mapEventCalls                    int
	mapEventInput                    model.MerchantMapEventInput
	mapEventErr                      error
	relatedSource                    model.RelatedResourceSource
	relatedSourceErr                 error
	relatedSourceCalls               int
	relatedItems                     []model.ResourceListItem
	relatedResourceID                string
	relatedLimit                     int64
	relatedCalls                     int
}

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
	if s.managedMerchantErr != nil {
		return false, s.managedMerchantErr
	}
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

func (s *fakeResourceAPIStore) PublishResourceAfterMediaAudit(ctx context.Context, resourceID string, traceID string) (model.ReviewResourceResult, error) {
	s.publishedAuditTraceID = traceID
	return s.PublishResourceAfterAudit(ctx, resourceID)
}

func (s *fakeResourceAPIStore) RejectResourceAfterMediaAudit(ctx context.Context, resourceID string, traceID string, reason string) (model.ReviewResourceResult, error) {
	s.rejectedAuditTraceID = traceID
	return s.RejectResourceAfterAudit(ctx, resourceID, reason)
}

func (s *fakeResourceAPIStore) MarkResourceAuditRetryAfterMediaAudit(_ context.Context, _ string, traceID string, _ string) (int64, error) {
	s.retriedAuditTraceID = traceID
	return 1, nil
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
			PriceText: "18元/件", QuantityText: "3800件", Merchant: model.ResourceMerchantBrief{ID: "merchant-1", Name: "织里云仓"},
			Tags: []string{"急清", "支持看货"}, RefreshedAt: "2026-06-27T10:00:00Z",
		}},
		Page: filter.Page, PageSize: filter.PageSize, Total: 1,
	}, nil
}

func (s *fakeResourceAPIStore) GetRelatedResourceSource(ctx context.Context, resourceID string) (model.RelatedResourceSource, error) {
	s.relatedSourceCalls++
	if s.relatedSourceErr != nil {
		return model.RelatedResourceSource{}, s.relatedSourceErr
	}
	return s.relatedSource, nil
}

func (s *fakeResourceAPIStore) ListRelatedResources(ctx context.Context, resourceID string, limit int64) ([]model.ResourceListItem, error) {
	s.relatedCalls++
	s.relatedResourceID = resourceID
	s.relatedLimit = limit
	return s.relatedItems, nil
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
		ContactName: "周经理", PhoneMasked: "188****0002", WechatMasked: "stock-demo",
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
