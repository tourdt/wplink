package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/middleware"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAdminListMerchantsGeneratedRouteReturnsLocationGovernanceContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	lastActiveAt := time.Date(2026, time.August, 29, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT m.id::text, m.name, m.merchant_type, m.status, m.location_status, COALESCE\(m.market_name, ''\), COALESCE\(m.building_name, ''\), COALESCE\(m.floor_no, ''\), COALESCE\(m.booth_no, ''\), m.last_active_at, \( SELECT COUNT\(\*\) FROM merchant_location_reports mlr WHERE mlr.merchant_id = m.id AND mlr.status = 'pending' \) AS pending_location_report_count, COUNT\(\*\) OVER\(\) AS total FROM merchants m JOIN city_stations cs ON cs.id = m.city_station_id WHERE m.deleted_at IS NULL AND \(\$1 = '' OR cs.code = \$1\) AND \(\$2 = '' OR m.merchant_type = \$2\) AND \(\$3 = '' OR m.status = \$3\) AND \(\$4 = '' OR m.location_status = \$4\) AND \(\$5 = '' OR m.name ILIKE '%' \|\| \$5 \|\| '%' OR m.contact_name ILIKE '%' \|\| \$5 \|\| '%'\) ORDER BY m.created_at DESC LIMIT \$6 OFFSET \$7`).
		WithArgs("zhili", "factory", model.MerchantStatusActive, model.MerchantLocationStatusRiskHidden, "晨星", int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "merchant_type", "status", "location_status", "market_name", "building_name", "floor_no", "booth_no", "last_active_at", "pending_location_report_count", "total",
		}).AddRow("101", "织里晨星童装", "factory", model.MerchantStatusActive, model.MerchantLocationStatusRiskHidden, "织里童装城", "一期", "1F", "A108", lastActiveAt, int64(3), int64(1)))

	adminAuth := middleware.NewAdminAuthMiddleware(&fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "701", Roles: []string{permission.RolePlatformOperator}, Modules: []string{permission.AdminModuleMerchants},
	}}).Handle
	server := newGeneratedAPIServer(t, &svc.ServiceContext{
		AdminAuth: adminAuth,
		APIStore:  &svc.APIStore{MerchantModel: model.NewMerchantModel(db)},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/merchants?cityCode=zhili&merchantType=factory&status=active&locationStatus=risk_hidden&keyword=%E6%99%A8%E6%98%9F&page=1&pageSize=20", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("data = %#v, want one merchant", data)
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("item = %#v, want object", items[0])
	}
	if item["status"] != model.MerchantStatusActive || item["locationStatus"] != model.MerchantLocationStatusRiskHidden || item["marketName"] != "织里童装城" || item["buildingName"] != "一期" || item["floorNo"] != "1F" || item["boothNo"] != "A108" || item["pendingLocationReportCount"] != float64(3) {
		t.Fatalf("item = %#v, want independent status and complete location governance response", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin merchant route SQL expectations: %v", err)
	}
}

func TestCreateMerchantLocationReportGeneratedRouteSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	expectVisibleMerchantForGeneratedLocationReport(mock, "101")
	mock.ExpectQuery(`SELECT mlr.id::text, mlr.status,.*FROM merchant_location_reports mlr`).
		WithArgs("101", "201").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO merchant_location_reports`).
		WithArgs("101", "201", "wrong_booth", "A108 已搬走", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("9001", model.MerchantLocationReportStatusPending))
	mock.ExpectQuery(`SELECT COUNT\(DISTINCT reporter_user_id\).*FROM merchant_location_reports`).
		WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectCommit()

	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{
		APIStore:         &svc.APIStore{MerchantLocationReportModel: model.NewMerchantLocationReportModel(db)},
		UserTokenService: numericLocationReportUserTokenService{},
	})
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, authenticatedRequest(http.MethodPost, "/api/v1/merchants/101/location-reports", `{"reasonCode":"wrong_booth","description":"A108 已搬走"}`))

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["reportId"] != "9001" || data["status"] != model.MerchantLocationReportStatusPending || data["activeReporterCount"] != float64(1) || data["autoHidden"] != false || data["idempotent"] != false {
		t.Fatalf("data = %#v, want successful report response envelope", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("generated report route SQL expectations: %v", err)
	}
}

type numericLocationReportUserTokenService struct{}

func (numericLocationReportUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (numericLocationReportUserTokenService) ParseUserToken(_ context.Context, token string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{UserID: "201"}, nil
}

func TestAdminMerchantLocationGovernanceGeneratedWriteRoutesSucceed(t *testing.T) {
	t.Run("review report", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT merchant_id::text, status, reason_code, description FROM merchant_location_reports WHERE id = \$1 FOR UPDATE`).
			WithArgs("9001").
			WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "status", "reason_code", "description"}).AddRow("101", model.MerchantLocationReportStatusPending, "wrong_booth", "A108 已搬走"))
		mock.ExpectExec(`UPDATE merchant_location_reports`).
			WithArgs("9001", model.MerchantLocationReportStatusValid, "701", "情况属实").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO operation_logs`).
			WithArgs("701", permission.RolePlatformOperator, "merchant_location_report_review", "merchant_location_report", "9001", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		server := newGeneratedLocationGovernanceAdminServer(t, db)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/merchants/location-reports/9001/review", strings.NewReader(`{"status":"valid","reviewNote":"情况属实"}`))
		req.Header.Set("Authorization", "Bearer admin-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)

		data := decodeEnvelopeData(t, rec, http.StatusOK)
		if data["reportId"] != "9001" || data["merchantId"] != "101" || data["status"] != model.MerchantLocationReportStatusValid {
			t.Fatalf("data = %#v, want reviewed report envelope", data)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("generated review route SQL expectations: %v", err)
		}
	})

	t.Run("hide location", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT location_status, latitude, longitude FROM merchants WHERE id = \$1 AND deleted_at IS NULL FOR UPDATE`).
			WithArgs("101").
			WillReturnRows(sqlmock.NewRows([]string{"location_status", "latitude", "longitude"}).AddRow(model.MerchantLocationStatusVisible, 30.8732, 120.2255))
		mock.ExpectExec(`UPDATE merchants SET location_status = \$2, updated_at = now\(\) WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs("101", model.MerchantLocationStatusHidden).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO operation_logs`).
			WithArgs("701", permission.RolePlatformOperator, "merchant_location_hide", "merchant", "101", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		server := newGeneratedLocationGovernanceAdminServer(t, db)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/merchants/101/location-status", strings.NewReader(`{"action":"hide","reason":"等待商家核对"}`))
		req.Header.Set("Authorization", "Bearer admin-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)

		data := decodeEnvelopeData(t, rec, http.StatusOK)
		if data["merchantId"] != "101" || data["locationStatus"] != model.MerchantLocationStatusHidden {
			t.Fatalf("data = %#v, want hidden merchant location envelope", data)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("generated location status route SQL expectations: %v", err)
		}
	})
}

func newGeneratedLocationGovernanceAdminServer(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	adminAuth := middleware.NewAdminAuthMiddleware(&fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "701", Roles: []string{permission.RolePlatformOperator}, Modules: []string{permission.AdminModuleMerchants},
	}}).Handle
	return newGeneratedAPIServer(t, &svc.ServiceContext{
		AdminAuth: adminAuth,
		APIStore:  &svc.APIStore{MerchantLocationReportModel: model.NewMerchantLocationReportModel(db)},
	})
}

func expectVisibleMerchantForGeneratedLocationReport(mock sqlmock.Sqlmock, merchantID string) {
	mock.ExpectQuery(`SELECT.*market_name.*FROM merchants.*WHERE id = \$1.*FOR UPDATE`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"market_name", "building_name", "floor_no", "booth_no", "address_text", "latitude", "longitude", "location_name", "location_status",
		}).AddRow("织里童装城", "一期", "1F", "A108", "织里镇利济路", 30.8732, 120.2255, "织里童装城", model.MerchantLocationStatusVisible))
}
