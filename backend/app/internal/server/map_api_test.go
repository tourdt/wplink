package server

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestMapGeneratedPublicRuntimeBehavior(t *testing.T) {
	t.Run("merchant directory maps complete filters and non-null arrays", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		filterArgs := []driver.Value{
			"zhili", "童装", pq.Array([]string{"girl"}), pq.Array([]string{"factory"}),
			float64(30), float64(31), float64(120), float64(121),
		}
		listArgs := append(append([]driver.Value{}, filterArgs...), int64(5), int64(5))
		mock.ExpectQuery(`(?s)FROM map_object o.*ORDER BY \(m.id IS NOT NULL\) DESC`).
			WithArgs(listArgs...).
			WillReturnRows(mapMerchantPlaceRows(mapObjectRowFixture{
				ID: "object-merchant", SceneCode: "scene-1", MerchantID: "merchant-1", MerchantName: "小鹿童装",
				ProfileStatus: model.MerchantProfileStatusCompleted, Code: "A001", Name: "预录档口", Type: "booth", Layer: "booth",
				Lat: "30.5", Lng: "120.5",
			}, "zhili", "一楼", "童装城", "1F"))
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM map_object o`).
			WithArgs(filterArgs...).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

		req := httptest.NewRequest(http.MethodGet,
			"/api/v1/map/merchant-places?cityCode=zhili&keyword=%E7%AB%A5%E8%A3%85&categories=girl&merchantTypes=factory&claimed=claimed&page=2&pageSize=5&minLat=30&maxLat=31&minLng=120&maxLng=121", nil)
		body, raw := assertMapGeneratedRawStatus(t, server, req, http.StatusOK)
		data := body["data"].(map[string]interface{})
		if data["total"] != float64(1) || data["page"] != float64(2) || data["pageSize"] != float64(5) {
			t.Fatalf("merchant directory data=%#v", data)
		}
		for _, fragment := range []string{`"categoryCodes":[]`, `"serviceTags":[]`, `"platformTags":[]`} {
			if !strings.Contains(raw, fragment) {
				t.Fatalf("merchant directory raw=%s, want %s", raw, fragment)
			}
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("search maps viewport and exposes verified merchant DTO fields", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)^SELECT .*FROM map_object o`).
			WithArgs("scene-1", "童装", model.MapObjectStatusNormal, float64(500), float64(0), float64(400), float64(0), int64(4), int64(2)).
			WillReturnRows(mapObjectRows(mapObjectRowFixture{
				ID: "object-1", SceneCode: "scene-1", MerchantID: "merchant-1", MerchantName: "小鹿童装",
				ProfileStatus: model.MerchantProfileStatusCompleted, Code: "A001", Name: "预录档口", Type: "booth", Layer: "booth",
			}))
		mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM map_object o`).
			WithArgs("scene-1", "童装", model.MapObjectStatusNormal).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

		body, raw := assertMapGeneratedRawStatus(t, server, httptest.NewRequest(http.MethodGet,
			"/api/v1/map/objects/search?sceneCode=scene-1&keyword=%E7%AB%A5%E8%A3%85&minX=0&minY=0&maxX=500&maxY=400&zoom=4&limit=2", nil), http.StatusOK)
		item := body["data"].(map[string]interface{})["items"].([]interface{})[0].(map[string]interface{})
		merchant := item["merchant"].(map[string]interface{})
		if item["isVerifiedMerchant"] != true || merchant["verificationStatus"] != model.MerchantProfileStatusCompleted {
			t.Fatalf("search item=%#v, want verified merchant runtime fields", item)
		}
		for _, fragment := range []string{
			`"categoryCodes":[]`, `"serviceTags":[]`, `"platformTags":[]`, `"poiServiceTags":[]`, `"mainCategories":[]`,
		} {
			if !strings.Contains(raw, fragment) {
				t.Fatalf("search raw=%s, want %s", raw, fragment)
			}
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("nearby pois use path object and real coordinates", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)WHERE o.id::text = \$1`).
			WithArgs("object-path", model.MapObjectStatusNormal).
			WillReturnRows(mapObjectRows(mapObjectRowFixture{ID: "object-path", SceneCode: "scene-1", Code: "A001", Name: "档口", Type: "booth", Layer: "booth", CenterX: 10, CenterY: 20}))
		mock.ExpectQuery(`(?s)FROM map_object o.*ORDER BY o.sort ASC`).
			WithArgs("scene-1", pq.Array([]string{"parking"}), model.MapObjectStatusNormal).
			WillReturnRows(mapObjectRows(mapObjectRowFixture{ID: "poi-1", SceneCode: "scene-1", Code: "P001", Name: "停车场", Type: "parking", Layer: "poi", CenterX: 13, CenterY: 24}))

		data := assertMapGeneratedStatus(t, server, httptest.NewRequest(http.MethodGet,
			"/api/v1/map/objects/object-path/nearby-pois?objectId=query-attacker&types=parking&limit=1", nil), http.StatusOK)["data"].(map[string]interface{})
		items := data["items"].([]interface{})
		if len(items) != 1 || items[0].(map[string]interface{})["id"] != "poi-1" {
			t.Fatalf("nearby data=%#v", data)
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("merchant location context keeps current and nearby arrays", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)WHERE o.merchant_id = \$1::bigint`).
			WithArgs("merchant-path").
			WillReturnRows(mapMerchantPlaceRows(mapObjectRowFixture{
				ID: "object-1", SceneCode: "scene-1", MerchantID: "merchant-path", MerchantName: "小鹿童装",
				ProfileStatus: model.MerchantProfileStatusCompleted, Code: "A001", Name: "档口", Type: "booth", Layer: "booth",
				Lat: "30.87", Lng: "120.12",
			}, "zhili", "一楼", "童装城", "1F"))
		mock.ExpectQuery(`(?s)o.merchant_id <> \$1::bigint`).
			WithArgs("merchant-path", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(mapMerchantPlaceRows())

		body, raw := assertMapGeneratedRawStatus(t, server, httptest.NewRequest(http.MethodGet,
			"/api/v1/map/merchants/merchant-path/location-context?merchantId=query-attacker", nil), http.StatusOK)
		data := body["data"].(map[string]interface{})
		if data["nearbyAvailable"] != true || data["radiusMeters"] != float64(1000) {
			t.Fatalf("location context=%#v", data)
		}
		for _, fragment := range []string{`"nearby":[]`, `"categoryCodes":[]`, `"serviceTags":[]`, `"platformTags":[]`} {
			if !strings.Contains(raw, fragment) {
				t.Fatalf("location raw=%s, want %s", raw, fragment)
			}
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("invalid viewport is rejected before SQL", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		body := assertMapGeneratedStatus(t, server, httptest.NewRequest(http.MethodGet,
			"/api/v1/map/objects/search?minX=0", nil), http.StatusBadRequest)
		if body["errorCode"] != errx.CodeValidationFailed {
			t.Fatalf("invalid viewport body=%#v", body)
		}
		assertMapSQLExpectations(t, mock)
	})
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

func TestMapGeneratedBindingAuthorizationAndSubmissionBehavior(t *testing.T) {
	t.Run("owner lists candidates with exact filters", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
			WithArgs("user-1", "merchant-1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`(?s)FROM map_object o.*o.layer = 'booth'`).
			WithArgs("object-1", "scene-1", "A001", int64(3)).
			WillReturnRows(mapBindCandidateRows("object-1", ""))

		data := assertMapGeneratedStatus(t, server, authenticatedMapRequest(http.MethodGet,
			"/api/v1/map/bind-candidates?merchantId=merchant-1&objectId=object-1&sceneCode=scene-1&keyword=A001&limit=3", ""), http.StatusOK)["data"].(map[string]interface{})
		items := data["items"].([]interface{})
		if len(items) != 1 || items[0].(map[string]interface{})["objectId"] != "object-1" {
			t.Fatalf("candidate data=%#v", data)
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("ordinary user cannot access another merchant", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
			WithArgs("user-1", "merchant-other").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		body := assertMapGeneratedStatus(t, server, authenticatedMapRequest(http.MethodGet,
			"/api/v1/map/bind-candidates?merchantId=merchant-other", ""), http.StatusForbidden)
		if body["errorCode"] != errx.CodeForbidden {
			t.Fatalf("forbidden body=%#v", body)
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("platform admin can list candidates without merchant membership", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		svcCtx.AdminTokenService = adminauth.NewValidatingAdminTokenService(
			&strictUploadAdminTokenService{subject: session.AdminTokenSubject{OperatorID: "operator-map-1", AuthVersion: 1}},
			strictUploadAdminSessionStore{credential: adminauth.AdminCredential{
				OperatorID: "operator-map-1", AuthVersion: 1, Status: adminauth.CredentialStatusEnabled, Roles: []string{permission.RoleSuperAdmin},
			}},
		)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)FROM map_object o.*o.layer = 'booth'`).
			WithArgs(int64(20)).
			WillReturnRows(mapBindCandidateRows("object-1", ""))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/map/bind-candidates?merchantId=merchant-other", nil)
		req.Header.Set("Authorization", "Bearer admin-token")
		data := assertMapGeneratedStatus(t, server, req, http.StatusOK)["data"].(map[string]interface{})
		if len(data["items"].([]interface{})) != 1 {
			t.Fatalf("admin candidates=%#v", data)
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("owner submission uses token applicant and path merchant", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)FROM merchant_admin_bindings mab`).
			WithArgs("user-1", "merchant-path").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectBegin()
		mock.ExpectQuery(`(?s)SELECT o.scene_code.*FOR UPDATE`).
			WithArgs("object-path").
			WillReturnRows(sqlmock.NewRows([]string{"scene_code", "merchant_id"}).AddRow("scene-1", ""))
		mock.ExpectQuery(`(?s)SELECT id::text.*WHERE merchant_id::text = \$1.*FOR UPDATE`).
			WithArgs("merchant-path").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec(`(?s)UPDATE map_object.*SET merchant_id`).
			WithArgs("merchant-path", "object-path").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`(?s)INSERT INTO map_object_bind_request`).
			WithArgs("merchant-path", "object-path", "scene-1", "user-1", sqlmock.AnyArg(), "").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("request-1"))
		mock.ExpectQuery(`(?s)FROM map_object_bind_request r.*WHERE r.id::text = \$1`).
			WithArgs("request-1").
			WillReturnRows(mapBindRequestRows("request-1", "merchant-path", "object-path", "user-1"))
		mock.ExpectCommit()

		req := authenticatedMapRequest(http.MethodPost,
			"/api/v1/merchants/merchant-path/map-binding-requests?merchantId=query-attacker",
			`{"merchantId":"body-attacker","objectId":"object-path","applicantUserId":"body-attacker","evidenceImages":[]}`)
		body, raw := assertMapGeneratedRawStatus(t, server, req, http.StatusOK)
		item := body["data"].(map[string]interface{})["item"].(map[string]interface{})
		if item["merchantId"] != "merchant-path" || item["applicantUserId"] != "user-1" {
			t.Fatalf("submission item=%#v", item)
		}
		if !strings.Contains(raw, `"evidenceImages":[]`) {
			t.Fatalf("submission raw=%s, want evidenceImages: []", raw)
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("typed nil permission dependencies fail closed", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*svc.ServiceContext)
		}{
			{name: "user model", mutate: func(ctx *svc.ServiceContext) { ctx.APIStore.UserModel = (*model.UserModel)(nil) }},
			{name: "user token", mutate: func(ctx *svc.ServiceContext) { ctx.UserTokenService = (*fakeUserTokenService)(nil) }},
			{name: "admin token", mutate: func(ctx *svc.ServiceContext) { ctx.AdminTokenService = (*adminauth.ValidatingAdminTokenService)(nil) }},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svcCtx, mock := newMapGeneratedServiceContext(t)
				tc.mutate(svcCtx)
				server := newGeneratedAPIServer(t, svcCtx)
				body := assertMapGeneratedStatus(t, server, authenticatedMapRequest(http.MethodGet,
					"/api/v1/map/bind-candidates?merchantId=merchant-1", ""), http.StatusInternalServerError)
				if body["errorCode"] != errx.CodeInternalError {
					t.Fatalf("typed nil body=%#v", body)
				}
				assertMapSQLExpectations(t, mock)
			})
		}
	})
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

func TestMapGeneratedAdminModuleAndWriteBehavior(t *testing.T) {
	t.Run("module permission rejects before map store", func(t *testing.T) {
		adminAuth := middleware.NewAdminAuthMiddleware(&fakeAdminTokenService{subject: session.AdminTokenSubject{
			OperatorID: "operator-no-map", Roles: []string{permission.RolePlatformOperator}, Modules: []string{permission.AdminModuleDashboard},
		}}).Handle
		server := newGeneratedAPIServer(t, &svc.ServiceContext{AdminAuth: adminAuth})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/map/categories", strings.NewReader(`{"code":"girl","name":"女童","type":"booth_category"}`))
		req.Header.Set("Authorization", "Bearer admin-token")
		req.Header.Set("Content-Type", "application/json")
		body := assertMapGeneratedStatus(t, server, req, http.StatusForbidden)
		if body["errorCode"] != errx.CodeForbidden {
			t.Fatalf("module rejection body=%#v", body)
		}
	})

	t.Run("scene save and publish execute logic", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)INSERT INTO map_scene`).
			WithArgs("", "scene-created", "利济路", "street_segment", "", "https://img.example/map.png", int64(1000), int64(800), "", "", "", "", "", "", int64(0), model.MapSceneStatusDraft).
			WillReturnRows(mapSceneRows("scene-created", model.MapSceneStatusDraft))
		saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/map/scenes?operatorId=query-attacker",
			strings.NewReader(`{"operatorId":"body-attacker","code":"scene-created","name":"利济路","type":"street_segment","backgroundUrl":"https://img.example/map.png","width":1000,"height":800}`))
		saveReq.Header.Set("Authorization", "Bearer admin-token")
		saveReq.Header.Set("Content-Type", "application/json")
		saved := assertMapGeneratedStatus(t, server, saveReq, http.StatusOK)["data"].(map[string]interface{})
		if saved["item"].(map[string]interface{})["code"] != "scene-created" {
			t.Fatalf("saved scene=%#v", saved)
		}

		mock.ExpectQuery(`(?s)FROM map_scene s.*WHERE s.code = \$1`).
			WithArgs("scene-created", "").
			WillReturnRows(mapSceneRows("scene-created", model.MapSceneStatusDraft))
		mock.ExpectQuery(`(?s)FROM map_object o.*ORDER BY o.sort ASC`).
			WithArgs("scene-created", model.MapObjectStatusNormal).
			WillReturnRows(mapObjectRows(mapObjectRowFixture{
				ID: "object-1", SceneCode: "scene-created", Code: "A001", Name: "档口 A001", Type: "booth", Layer: "booth", PublishReady: true,
			}))
		mock.ExpectQuery(`(?s)UPDATE map_scene.*SET status = 'published'`).
			WithArgs("scene-created").
			WillReturnRows(mapSceneRows("scene-created", model.MapSceneStatusPublished))
		publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/map/scenes/scene-created/publish?operatorId=query-attacker", nil)
		publishReq.Header.Set("Authorization", "Bearer admin-token")
		published := assertMapGeneratedStatus(t, server, publishReq, http.StatusOK)["data"].(map[string]interface{})
		if published["message"] != "地图场景已发布" {
			t.Fatalf("published scene=%#v", published)
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("object update status and batch execute path identifiers", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)UPDATE map_object.*WHERE id::text = \$1`).
			WithArgs(mapAnyArgs("object-path", 29)...).
			WillReturnRows(mapObjectRows(mapObjectRowFixture{ID: "object-path", SceneCode: "scene-1", Code: "A001", Name: "档口 A001"}))
		updateReq := httptest.NewRequest(http.MethodPost,
			"/api/v1/admin/map/objects/object-path?objectId=query-attacker&operatorId=query-attacker",
			strings.NewReader(`{"id":"body-attacker","operatorId":"body-attacker","code":"A001","name":"档口 A001","type":"booth","layer":"booth","geometryType":"rect","geometry":{"x":10,"y":20,"width":80,"height":50}}`))
		updateReq.Header.Set("Authorization", "Bearer admin-token")
		updateReq.Header.Set("Content-Type", "application/json")
		updated := assertMapGeneratedStatus(t, server, updateReq, http.StatusOK)["data"].(map[string]interface{})
		if updated["item"].(map[string]interface{})["id"] != "object-path" {
			t.Fatalf("updated object=%#v", updated)
		}

		mock.ExpectQuery(`(?s)UPDATE map_object.*SET status = \$2`).
			WithArgs("object-path", model.MapObjectStatusHidden).
			WillReturnRows(mapObjectRows(mapObjectRowFixture{ID: "object-path", SceneCode: "scene-1", Code: "A001", Name: "档口 A001"}))
		statusReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/map/objects/object-path/status?operatorId=query-attacker",
			strings.NewReader(`{"operatorId":"body-attacker","status":"hidden"}`))
		statusReq.Header.Set("Authorization", "Bearer admin-token")
		statusReq.Header.Set("Content-Type", "application/json")
		assertMapGeneratedStatus(t, server, statusReq, http.StatusOK)

		mock.ExpectQuery(`(?s)FROM map_scene s.*WHERE s.code = \$1`).
			WithArgs("scene-path", "").
			WillReturnRows(mapSceneRows("scene-path", model.MapSceneStatusDraft))
		mock.ExpectQuery(`(?s)INSERT INTO map_object`).
			WithArgs(mapAnyArgs("scene-path", 29)...).
			WillReturnRows(mapObjectRows(mapObjectRowFixture{ID: "batch-object-1", SceneCode: "scene-path", Code: "B001", Name: "B001"}))
		batchReq := httptest.NewRequest(http.MethodPost,
			"/api/v1/admin/map/scenes/scene-path/objects/batch-generate?sceneCode=query-attacker&operatorId=query-attacker",
			strings.NewReader(`{"operatorId":"body-attacker","startCode":"B001","count":1,"direction":"horizontal","startX":"100","startY":"100","width":"80","height":"50","gap":"5","type":"booth","layer":"booth"}`))
		batchReq.Header.Set("Authorization", "Bearer admin-token")
		batchReq.Header.Set("Content-Type", "application/json")
		batch, raw := assertMapGeneratedRawStatus(t, server, batchReq, http.StatusOK)
		if len(batch["data"].(map[string]interface{})["items"].([]interface{})) != 1 || !strings.Contains(raw, `"categoryCodes":[]`) {
			t.Fatalf("batch response=%s", raw)
		}
		assertMapSQLExpectations(t, mock)
	})

	t.Run("category list filters and save execute logic", func(t *testing.T) {
		svcCtx, mock := newMapGeneratedServiceContext(t)
		server := newGeneratedAPIServer(t, svcCtx)
		mock.ExpectQuery(`(?s)FROM map_category.*WHERE type = \$1 AND status = \$2`).
			WithArgs("booth_category", model.MapCategoryStatusHidden).
			WillReturnRows(mapCategoryRows("girl", model.MapCategoryStatusHidden))
		listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/map/categories?type=booth_category&status=hidden", nil)
		listReq.Header.Set("Authorization", "Bearer admin-token")
		listed := assertMapGeneratedStatus(t, server, listReq, http.StatusOK)["data"].(map[string]interface{})
		if len(listed["items"].([]interface{})) != 1 {
			t.Fatalf("listed categories=%#v", listed)
		}

		mock.ExpectQuery(`(?s)INSERT INTO map_category`).
			WithArgs("girl", "女童", "booth_category", "", int64(1), true, model.MapCategoryStatusNormal).
			WillReturnRows(mapCategoryRows("girl", model.MapCategoryStatusNormal))
		saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/map/categories?operatorId=query-attacker",
			strings.NewReader(`{"operatorId":"body-attacker","code":"girl","name":"女童","type":"booth_category","sort":1,"isVisible":true}`))
		saveReq.Header.Set("Authorization", "Bearer admin-token")
		saveReq.Header.Set("Content-Type", "application/json")
		saved := assertMapGeneratedStatus(t, server, saveReq, http.StatusOK)["data"].(map[string]interface{})
		if saved["item"].(map[string]interface{})["code"] != "girl" {
			t.Fatalf("saved category=%#v", saved)
		}
		assertMapSQLExpectations(t, mock)
	})
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

type mapObjectRowFixture struct {
	ID            string
	SceneCode     string
	MerchantID    string
	MerchantName  string
	ProfileStatus string
	Code          string
	Name          string
	Type          string
	Layer         string
	CenterX       float64
	CenterY       float64
	Lat           string
	Lng           string
	PublishReady  bool
}

func mapObjectColumns() []string {
	return []string{
		"id", "scene_code", "merchant_id", "merchant_name", "merchant_type", "profile_status", "merchant_logo_url", "merchant_main_categories",
		"code", "name", "type", "layer", "geometry_type", "geometry",
		"center_x", "center_y", "min_x", "min_y", "max_x", "max_y", "min_zoom", "max_zoom",
		"category_codes", "service_tags", "platform_tags", "poi_service_tags",
		"address", "phone", "wechat", "lat", "lng", "search_text", "extra", "sort", "status", "created_at", "updated_at",
	}
}

func mapObjectRowValues(item mapObjectRowFixture) []driver.Value {
	now := time.Date(2026, time.August, 6, 8, 0, 0, 0, time.UTC)
	if item.Code == "" {
		item.Code = item.ID
	}
	if item.Name == "" {
		item.Name = item.Code
	}
	if item.Type == "" {
		item.Type = "booth"
	}
	if item.Layer == "" {
		item.Layer = "booth"
	}
	categoryCodes := []byte(`[]`)
	serviceTags := []byte(`[]`)
	phone := ""
	if item.PublishReady {
		categoryCodes = []byte(`["girl"]`)
		serviceTags = []byte(`["spot"]`)
		phone = "13800000000"
	}
	return []driver.Value{
		item.ID, item.SceneCode, item.MerchantID, item.MerchantName, "factory", item.ProfileStatus, "", []byte(`[]`),
		item.Code, item.Name, item.Type, item.Layer, model.MapGeometryTypeRect, []byte(`{"x":10,"y":20,"width":80,"height":50}`),
		item.CenterX, item.CenterY, item.CenterX, item.CenterY, item.CenterX + 80, item.CenterY + 50, int64(1), int64(5),
		categoryCodes, serviceTags, []byte(`[]`), []byte(`[]`),
		"", phone, "", item.Lat, item.Lng, item.Name, []byte(`{}`), int64(0), model.MapObjectStatusNormal, now, now,
	}
}

func mapObjectRows(items ...mapObjectRowFixture) *sqlmock.Rows {
	rows := sqlmock.NewRows(mapObjectColumns())
	for _, item := range items {
		rows.AddRow(mapObjectRowValues(item)...)
	}
	return rows
}

func mapMerchantPlaceRows(items ...interface{}) *sqlmock.Rows {
	columns := append(append([]string{}, mapObjectColumns()...), "city_code", "scene_name", "market_name", "floor_no")
	rows := sqlmock.NewRows(columns)
	for index := 0; index+4 < len(items); index += 5 {
		fixture := items[index].(mapObjectRowFixture)
		values := mapObjectRowValues(fixture)
		values = append(values, items[index+1], items[index+2], items[index+3], items[index+4])
		rows.AddRow(values...)
	}
	return rows
}

func assertMapGeneratedRawStatus(t *testing.T, server interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, req *http.Request, wantStatus int) (map[string]interface{}, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	raw := rec.Body.String()
	return decodeEnvelope(t, rec, wantStatus), raw
}

func assertMapSQLExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("map SQL expectations: %v", err)
	}
}

func mapAnyArgs(first driver.Value, total int) []driver.Value {
	args := make([]driver.Value, total)
	args[0] = first
	for index := 1; index < total; index++ {
		args[index] = sqlmock.AnyArg()
	}
	return args
}

func mapSceneRows(code string, status string) *sqlmock.Rows {
	now := time.Date(2026, time.August, 6, 8, 0, 0, 0, time.UTC)
	return sqlmock.NewRows([]string{
		"id", "city_code", "code", "name", "type", "parent_code", "background_url", "width", "height",
		"min_scale", "max_scale", "default_scale", "default_center_x", "default_center_y", "floor_no", "sort", "revision", "status", "created_at", "updated_at",
	}).AddRow("scene-id", "zhili", code, "利济路", "street_segment", "", "https://img.example/map.png", int64(1000), int64(800),
		"0.5", "5", "1", "", "", "", int64(0), int64(1), status, now, now)
}

func mapCategoryRows(code string, status string) *sqlmock.Rows {
	now := time.Date(2026, time.August, 6, 8, 0, 0, 0, time.UTC)
	return sqlmock.NewRows([]string{"id", "code", "name", "type", "icon_url", "sort", "is_visible", "status", "created_at", "updated_at"}).
		AddRow("category-id", code, "女童", "booth_category", "", int64(1), true, status, now, now)
}

func mapBindCandidateRows(objectID string, merchantID string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"object_id", "scene_code", "scene_name", "code", "name", "address", "merchant_id", "merchant_name", "is_bound"}).
		AddRow(objectID, "scene-1", "利济路", "A001", "档口 A001", "一楼", merchantID, "小鹿童装", merchantID != "")
}

func mapBindRequestRows(requestID string, merchantID string, objectID string, applicantUserID string) *sqlmock.Rows {
	now := time.Date(2026, time.August, 6, 8, 0, 0, 0, time.UTC)
	return sqlmock.NewRows([]string{
		"id", "merchant_id", "merchant_name", "object_id", "scene_code", "scene_name", "object_code", "object_name", "applicant_user_id",
		"evidence_images", "note", "status", "review_note", "reviewed_by", "reviewed_at", "created_at", "updated_at",
	}).AddRow(requestID, merchantID, "小鹿童装", objectID, "scene-1", "利济路", "A001", "档口 A001", applicantUserID,
		[]byte(`[]`), "", model.MapBindRequestStatusApproved, "系统自动绑定", "", nil, now, now)
}
