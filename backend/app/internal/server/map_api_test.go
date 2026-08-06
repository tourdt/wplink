package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/middleware"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestMapGeneratedHandlersDoNotUseMigrationSkeleton(t *testing.T) {
	adminAuth := middleware.NewAdminAuthMiddleware(&fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "operator-map-1",
		Roles:      []string{permission.RoleSuperAdmin},
	}}).Handle
	server := newGeneratedAPIServer(t, &svc.ServiceContext{
		AdminTokenService: adminauth.NewValidatingAdminTokenService(nil, nil),
		AdminAuth:         adminAuth,
	})

	tests := []struct {
		name   string
		method string
		target string
		body   string
		admin  bool
	}{
		{name: "merchant places", method: http.MethodGet, target: "/api/v1/map/merchant-places"},
		{name: "merchant location context", method: http.MethodGet, target: "/api/v1/map/merchants/merchant-1/location-context"},
		{name: "public scenes", method: http.MethodGet, target: "/api/v1/map/scenes"},
		{name: "public scene", method: http.MethodGet, target: "/api/v1/map/scenes/scene-1"},
		{name: "public objects", method: http.MethodGet, target: "/api/v1/map/scenes/scene-1/objects"},
		{name: "search objects", method: http.MethodGet, target: "/api/v1/map/objects/search"},
		{name: "public object", method: http.MethodGet, target: "/api/v1/map/objects/object-1"},
		{name: "nearby pois", method: http.MethodGet, target: "/api/v1/map/objects/object-1/nearby-pois"},
		{name: "location correction", method: http.MethodPost, target: "/api/v1/map/objects/object-1/location-corrections", body: `{}`},
		{name: "risk report", method: http.MethodPost, target: "/api/v1/map/objects/object-1/risk-reports", body: `{}`},
		{name: "public categories", method: http.MethodGet, target: "/api/v1/map/categories"},
		{name: "bind candidates", method: http.MethodGet, target: "/api/v1/map/bind-candidates?merchantId=merchant-1"},
		{name: "binding status", method: http.MethodGet, target: "/api/v1/merchants/merchant-1/map-binding"},
		{name: "binding request", method: http.MethodPost, target: "/api/v1/merchants/merchant-1/map-binding-requests", body: `{}`},
		{name: "admin scenes", method: http.MethodGet, target: "/api/v1/admin/map/scenes", admin: true},
		{name: "admin save scene", method: http.MethodPost, target: "/api/v1/admin/map/scenes", body: `{}`, admin: true},
		{name: "admin scene", method: http.MethodGet, target: "/api/v1/admin/map/scenes/scene-1", admin: true},
		{name: "admin update scene", method: http.MethodPost, target: "/api/v1/admin/map/scenes/scene-1", body: `{}`, admin: true},
		{name: "admin publish scene", method: http.MethodPost, target: "/api/v1/admin/map/scenes/scene-1/publish", admin: true},
		{name: "admin objects", method: http.MethodGet, target: "/api/v1/admin/map/scenes/scene-1/objects", admin: true},
		{name: "admin save object", method: http.MethodPost, target: "/api/v1/admin/map/scenes/scene-1/objects", body: `{}`, admin: true},
		{name: "admin update object", method: http.MethodPost, target: "/api/v1/admin/map/objects/object-1", body: `{}`, admin: true},
		{name: "admin object status", method: http.MethodPost, target: "/api/v1/admin/map/objects/object-1/status", body: `{}`, admin: true},
		{name: "admin batch objects", method: http.MethodPost, target: "/api/v1/admin/map/scenes/scene-1/objects/batch-generate", body: `{}`, admin: true},
		{name: "admin categories", method: http.MethodGet, target: "/api/v1/admin/map/categories", admin: true},
		{name: "admin save category", method: http.MethodPost, target: "/api/v1/admin/map/categories", body: `{}`, admin: true},
		{name: "admin bind requests", method: http.MethodGet, target: "/api/v1/admin/map/bind-requests", admin: true},
		{name: "admin review bind request", method: http.MethodPost, target: "/api/v1/admin/map/bind-requests/request-1/review", body: `{}`, admin: true},
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
			if !tc.admin && rec.Code == http.StatusUnauthorized {
				t.Fatalf("public route %s %s was unexpectedly protected by AdminAuth", tc.method, tc.target)
			}
			if body["msg"] == "接口暂不可用，请稍后重试" {
				t.Fatalf("route %s %s still uses migration skeleton", tc.method, tc.target)
			}
		})
	}
}

func TestMapGeneratedPublicQueriesUseGeneratedDTOAndPathVariables(t *testing.T) {
	svcCtx, mock := newMapGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)

	mock.ExpectQuery(`(?s)FROM map_scene s`).
		WithArgs("zhili", "published").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	scenes := assertMapGeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/map/scenes?cityCode=zhili", nil), http.StatusOK)["data"].(map[string]interface{})
	if items, ok := scenes["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("scenes=%#v, want items: []", scenes)
	}

	mock.ExpectQuery(`(?s)FROM map_object o`).
		WithArgs(
			"scene-path", pq.Array([]string{"booth"}), pq.Array([]string{"girl"}),
			pq.Array([]string{"spot"}), pq.Array([]string{"packing"}), "童装", model.MapObjectStatusNormal,
			float64(510), float64(10), float64(420), float64(20), int64(4),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM map_object o`).
		WithArgs(
			"scene-path", pq.Array([]string{"booth"}), pq.Array([]string{"girl"}),
			pq.Array([]string{"spot"}), pq.Array([]string{"packing"}), "童装", model.MapObjectStatusNormal,
		).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(600)))
	objects := assertMapGeneratedStatus(t, server, httptest.NewRequest(http.MethodGet,
		"/api/v1/map/scenes/scene-path/objects?sceneCode=scene-query&types=booth&categories=girl&serviceTags=spot&poiServiceTags=packing&keyword=%E7%AB%A5%E8%A3%85&minX=10&minY=20&maxX=510&maxY=420&zoom=4", nil), http.StatusOK)["data"].(map[string]interface{})
	if objects["sceneCode"] != "scene-path" || objects["total"] != float64(600) {
		t.Fatalf("objects=%#v, want path scene and total", objects)
	}
	if items, ok := objects["items"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("objects=%#v, want items: []", objects)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("map public query expectations: %v", err)
	}
}

func TestMapGeneratedReportsUseTokenUserAndRejectBadToken(t *testing.T) {
	svcCtx, mock := newMapGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)

	badToken := httptest.NewRequest(http.MethodPost, "/api/v1/map/objects/object-1/risk-reports", strings.NewReader(`{"reasonCode":"false_information"}`))
	badToken.Header.Set("Authorization", "Bearer expired-token")
	badBody := assertMapGeneratedStatus(t, server, badToken, http.StatusUnauthorized)
	if badBody["errorCode"] != errx.CodeUnauthorized {
		t.Fatalf("bad token body=%#v, want unauthorized", badBody)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)jsonb_build_object`).
		WithArgs("object-1").
		WillReturnRows(sqlmock.NewRows([]string{"scene_code", "snapshot"}).AddRow("scene-1", []byte(`{}`)))
	mock.ExpectQuery(`(?s)INSERT INTO map_object_reports`).
		WithArgs("object-1", "scene-1", "user-1", model.MapObjectReportKindLocationCorrection, "navigation_inaccurate", "导航偏移", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("report-1", model.MapObjectReportStatusPending))
	mock.ExpectQuery(`(?s)SELECT COUNT\(DISTINCT reporter_user_id\)`).
		WithArgs("object-1", model.MapObjectReportKindLocationCorrection, "navigation_inaccurate").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectCommit()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/map/objects/object-1/location-corrections?reporterUserId=query-attacker",
		strings.NewReader(`{"reporterUserId":"body-attacker","reasonCode":"navigation_inaccurate","description":"导航偏移"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("Content-Type", "application/json")
	data := assertMapGeneratedStatus(t, server, req, http.StatusOK)["data"].(map[string]interface{})
	if data["message"] != "反馈已记录，感谢帮助完善拿货地图" {
		t.Fatalf("data=%#v, want stored location correction", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("map report expectations: %v", err)
	}
}

func TestMapGeneratedBindingUsesRealMerchantAndFailsClosed(t *testing.T) {
	t.Run("missing and typed nil dependencies", func(t *testing.T) {
		server := newGeneratedAPIServer(t, &svc.ServiceContext{AdminAuth: mapAdminAuthMiddleware()})
		body := assertMapGeneratedStatus(t, server, authenticatedMapRequest(http.MethodGet, "/api/v1/merchants/merchant-1/map-binding", ""), http.StatusInternalServerError)
		if body["errorCode"] != errx.CodeInternalError {
			t.Fatalf("body=%#v, want fail-closed map dependency", body)
		}

		server = newGeneratedAPIServer(t, &svc.ServiceContext{
			APIStore: &svc.APIStore{MapModel: (*model.MapModel)(nil)}, AdminAuth: mapAdminAuthMiddleware(),
		})
		body = assertMapGeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/map/scenes", nil), http.StatusInternalServerError)
		if body["errorCode"] != errx.CodeInternalError {
			t.Fatalf("body=%#v, want typed-nil map store rejection", body)
		}
	})

	svcCtx, mock := newMapGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)
	mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
		WithArgs("user-1", "merchant-path").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`(?s)FROM map_object o.*o.merchant_id::text = \$1`).
		WithArgs("merchant-path").
		WillReturnRows(sqlmock.NewRows([]string{"object_id"}))
	mock.ExpectQuery(`(?s)FROM map_object_bind_request r.*r.merchant_id::text = \$1`).
		WithArgs("merchant-path").
		WillReturnRows(sqlmock.NewRows([]string{"request_id"}))
	data := assertMapGeneratedStatus(t, server, authenticatedMapRequest(http.MethodGet,
		"/api/v1/merchants/merchant-path/map-binding?merchantId=merchant-query-attacker", ""), http.StatusOK)["data"].(map[string]interface{})
	if data["boundObject"] != nil || data["latestRequest"] != nil {
		t.Fatalf("data=%#v, want empty binding status", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("map binding expectations: %v", err)
	}
}

func TestMapGeneratedAdminRoutesAlwaysUseAdminAuth(t *testing.T) {
	tests := []struct {
		method string
		target string
		body   string
	}{
		{http.MethodGet, "/api/v1/admin/map/scenes", ""},
		{http.MethodPost, "/api/v1/admin/map/scenes", `{}`},
		{http.MethodGet, "/api/v1/admin/map/scenes/scene-1", ""},
		{http.MethodPost, "/api/v1/admin/map/scenes/scene-1", `{}`},
		{http.MethodPost, "/api/v1/admin/map/scenes/scene-1/publish", ""},
		{http.MethodGet, "/api/v1/admin/map/scenes/scene-1/objects", ""},
		{http.MethodPost, "/api/v1/admin/map/scenes/scene-1/objects", `{}`},
		{http.MethodPost, "/api/v1/admin/map/objects/object-1", `{}`},
		{http.MethodPost, "/api/v1/admin/map/objects/object-1/status", `{}`},
		{http.MethodPost, "/api/v1/admin/map/scenes/scene-1/objects/batch-generate", `{}`},
		{http.MethodGet, "/api/v1/admin/map/categories", ""},
		{http.MethodPost, "/api/v1/admin/map/categories", `{}`},
		{http.MethodGet, "/api/v1/admin/map/bind-requests", ""},
		{http.MethodPost, "/api/v1/admin/map/bind-requests/request-1/review", `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			svcCtx, _ := newMapGeneratedServiceContext(t)
			server := newGeneratedAPIServer(t, svcCtx)
			assertMapGeneratedStatus(t, server, httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body)), http.StatusUnauthorized)
		})
	}
}

func TestMapGeneratedAdminReviewUsesContextOperatorAndPathRequest(t *testing.T) {
	svcCtx, mock := newMapGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)
	sensitiveError := errors.New("Authorization=Bearer admin-secret reviewBody=private")

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)FROM map_object_bind_request.*status = 'pending'`).
		WithArgs("request-path").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "object_id"}).AddRow("merchant-1", "object-1"))
	mock.ExpectExec(`(?s)UPDATE map_object_bind_request.*reviewed_by`).
		WithArgs("request-path", model.MapBindRequestStatusRejected, "资料不匹配", "operator-map-1").
		WillReturnError(sensitiveError)
	mock.ExpectRollback()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/map/bind-requests/request-path/review?requestId=request-query-attacker&operatorId=query-attacker",
		strings.NewReader(`{"requestId":"request-body-attacker","operatorId":"body-attacker","reviewerId":"body-attacker","action":"reject","reviewNote":"资料不匹配"}`))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	body := assertMapGeneratedStatus(t, server, req, http.StatusInternalServerError)
	if body["errorCode"] != errx.CodeInternalError || strings.Contains(recursiveMapString(body), sensitiveError.Error()) {
		t.Fatalf("body=%#v, want safe review error", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin map review expectations: %v", err)
	}
}

func TestMapGeneratedAdminSceneUpdateUsesPathCode(t *testing.T) {
	svcCtx, mock := newMapGeneratedServiceContext(t)
	server := newGeneratedAPIServer(t, svcCtx)
	mock.ExpectQuery(`(?s)INSERT INTO map_scene`).
		WithArgs(
			"", "scene-path", "利济路中段", "street_segment", "", "https://img.example/map.png",
			int64(3000), int64(1800), "", "", "", "", "", "", int64(0), model.MapSceneStatusDraft,
		).
		WillReturnError(errors.New("save failed"))
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/map/scenes/scene-path?sceneCode=scene-query-attacker&operatorId=query-attacker",
		strings.NewReader(`{"operatorId":"body-attacker","code":"scene-body-attacker","name":"利济路中段","type":"street_segment","backgroundUrl":"https://img.example/map.png","width":3000,"height":1800}`))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	assertMapGeneratedStatus(t, server, req, http.StatusInternalServerError)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin map scene update expectations: %v", err)
	}
}

func newMapGeneratedServiceContext(t *testing.T) (*svc.ServiceContext, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &svc.ServiceContext{
		APIStore: &svc.APIStore{
			MapModel:  model.NewMapModel(db),
			UserModel: model.NewUserModel(db),
		},
		UserTokenService:  &fakeUserTokenService{},
		AdminTokenService: adminauth.NewValidatingAdminTokenService(nil, nil),
		AdminAuth:         mapAdminAuthMiddleware(),
	}, mock
}

func mapAdminAuthMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return middleware.NewAdminAuthMiddleware(&fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "operator-map-1",
		Roles:      []string{permission.RoleSuperAdmin},
	}}).Handle
}

func authenticatedMapRequest(method string, target string, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer user-token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func assertMapGeneratedStatus(t *testing.T, server interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, req *http.Request, wantStatus int) map[string]interface{} {
	t.Helper()
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return decodeEnvelope(t, rec, wantStatus)
}

func recursiveMapString(value interface{}) string {
	return strings.TrimSpace(fmt.Sprintf("%#v", value))
}
