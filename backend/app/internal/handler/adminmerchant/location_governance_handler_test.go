package adminmerchant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/handler/handlerx"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestAdminReviewMerchantLocationReportHandlerUsesAdminContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT merchant_id::text, status, reason_code, description FROM merchant_location_reports WHERE id = \$1 FOR UPDATE`).
		WithArgs("9001").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "status", "reason_code", "description"}).AddRow("101", model.MerchantLocationReportStatusPending, "wrong_booth", ""))
	mock.ExpectExec(`UPDATE merchant_location_reports`).
		WithArgs("9001", model.MerchantLocationReportStatusValid, "701", "情况属实").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO operation_logs`).
		WithArgs("701", permission.RoleSuperAdmin, "merchant_location_report_review", "merchant_location_report", "9001", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svcCtx := &svc.ServiceContext{APIStore: &svc.APIStore{
		MerchantModel:               model.NewMerchantModel(db),
		MerchantLocationReportModel: model.NewMerchantLocationReportModel(db),
	}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/merchants/location-reports/9001/review", strings.NewReader(`{"status":"valid","reviewNote":"情况属实"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(handlerx.ContextWithAdmin(context.Background(), session.AdminTokenSubject{
		OperatorID: "701", Roles: []string{permission.RoleSuperAdmin}, Modules: []string{permission.AdminModuleMerchants},
	}))
	req = pathvar.WithVars(req, map[string]string{"reportId": "9001"})
	recorder := httptest.NewRecorder()

	AdminReviewMerchantLocationReportHandler(svcCtx).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want reviewed report", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("admin identity transaction expectations: %v", err)
	}
}
