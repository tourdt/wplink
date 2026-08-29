package model

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateMerchantLocationReportHidesLocationAtThirdReporter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	expectVisibleMerchantLocationForUpdate(mock, "101")
	mock.ExpectQuery(`SELECT mlr.id::text, mlr.status,.*FROM merchant_location_reports mlr.*mlr.merchant_id = \$1.*mlr.reporter_user_id = \$2.*mlr.status = 'pending'`).
		WithArgs("101", "203").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO merchant_location_reports`).
		WithArgs("101", "203", "wrong_booth", "A108 已搬走", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("9001", MerchantLocationReportStatusPending))
	mock.ExpectQuery(`SELECT COUNT\(DISTINCT reporter_user_id\).*FROM merchant_location_reports.*merchant_id = \$1.*status = 'pending'`).
		WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(3)))
	mock.ExpectExec(`UPDATE merchants.*SET location_status = 'risk_hidden'.*WHERE id = \$1.*location_status = 'visible'`).
		WithArgs("101").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := NewMerchantLocationReportModel(db).CreateMerchantLocationReport(context.Background(), CreateMerchantLocationReportInput{
		MerchantID: "101", ReporterUserID: "203", ReasonCode: "wrong_booth", Description: "A108 已搬走",
	})
	if err != nil {
		t.Fatalf("CreateMerchantLocationReport() error = %v", err)
	}
	if result.ID != "9001" || result.Status != MerchantLocationReportStatusPending || !result.AutoHidden || result.ActiveReporterCount != 3 {
		t.Fatalf("result = %#v, want third reporter to auto-hide location", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("transaction expectations: %v", err)
	}
}

func TestCreateMerchantLocationReportReturnsExistingPendingReportIdempotently(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	// clear 会保留 pending 报告但清空坐标；同一用户重复提交仍须返回原报告，不能在可空坐标扫描处失败。
	expectMerchantLocationWithoutCoordinatesForUpdate(mock, "101", MerchantLocationStatusHidden)
	mock.ExpectQuery(`SELECT mlr.id::text, mlr.status,.*FROM merchant_location_reports mlr.*mlr.merchant_id = \$1.*mlr.reporter_user_id = \$2.*mlr.status = 'pending'`).
		WithArgs("101", "201").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "active_reporter_count"}).AddRow("8001", MerchantLocationReportStatusPending, int64(2)))
	mock.ExpectCommit()

	result, err := NewMerchantLocationReportModel(db).CreateMerchantLocationReport(context.Background(), CreateMerchantLocationReportInput{
		MerchantID: "101", ReporterUserID: "201", ReasonCode: "other", Description: "重复点击不应新增",
	})
	if err != nil {
		t.Fatalf("CreateMerchantLocationReport() error = %v", err)
	}
	if result.ID != "8001" || result.Status != MerchantLocationReportStatusPending || result.ActiveReporterCount != 2 || !result.Idempotent || result.AutoHidden {
		t.Fatalf("result = %#v, want existing pending report without increasing reporter count", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("idempotency expectations: %v", err)
	}
}

func TestCreateMerchantLocationReportRollsBackWhenReporterCountFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	expectVisibleMerchantLocationForUpdate(mock, "101")
	mock.ExpectQuery(`SELECT mlr.id::text, mlr.status,.*FROM merchant_location_reports mlr`).
		WithArgs("101", "203").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO merchant_location_reports`).
		WithArgs("101", "203", "wrong_market", "市场不正确", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("9001", MerchantLocationReportStatusPending))
	mock.ExpectQuery(`SELECT COUNT\(DISTINCT reporter_user_id\).*FROM merchant_location_reports`).
		WithArgs("101").
		WillReturnError(errors.New("count failed"))
	mock.ExpectRollback()

	result, err := NewMerchantLocationReportModel(db).CreateMerchantLocationReport(context.Background(), CreateMerchantLocationReportInput{
		MerchantID: "101", ReporterUserID: "203", ReasonCode: "wrong_market", Description: "市场不正确",
	})
	if err == nil {
		t.Fatal("CreateMerchantLocationReport() error = nil, want rollback error")
	}
	if result.ID != "9001" || result.Status != MerchantLocationReportStatusPending {
		t.Fatalf("result = %#v, want identifiers retained for failure diagnostics", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("rollback expectations: %v", err)
	}
}

func TestResolvePendingLocationReportsCommitsStatusChange(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE merchant_location_reports.*SET status = 'resolved'.*WHERE merchant_id = \$1.*status = 'pending'`).
		WithArgs("101").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	resolved, err := NewMerchantLocationReportModel(db).ResolvePendingLocationReports(context.Background(), "101")
	if err != nil {
		t.Fatalf("ResolvePendingLocationReports() error = %v", err)
	}
	if resolved != 3 {
		t.Fatalf("resolved = %d, want 3", resolved)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("resolve expectations: %v", err)
	}
}

func TestReviewMerchantLocationReportWritesAuditLogInSameTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT merchant_id::text, status, reason_code, description FROM merchant_location_reports WHERE id = \$1 FOR UPDATE`).
		WithArgs("9001").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "status", "reason_code", "description"}).AddRow("101", MerchantLocationReportStatusPending, "wrong_booth", "A108 已搬走"))
	mock.ExpectExec(`UPDATE merchant_location_reports.*SET status = \$2,.*reviewed_by = \$3.*review_note = NULLIF\(\$4, ''\).*reviewed_at = now\(\).*WHERE id = \$1.*status = 'pending'`).
		WithArgs("9001", MerchantLocationReportStatusValid, "701", "情况属实").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO operation_logs`).
		WithArgs("701", "platform_operator", "merchant_location_report_review", "merchant_location_report", "9001", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := NewMerchantLocationReportModel(db).ReviewMerchantLocationReport(context.Background(), ReviewMerchantLocationReportInput{
		ReportID: "9001", Status: MerchantLocationReportStatusValid, ReviewNote: "情况属实", AdminID: "701", AdminRole: "platform_operator",
	})
	if err != nil {
		t.Fatalf("ReviewMerchantLocationReport() error = %v", err)
	}
	if result.ReportID != "9001" || result.MerchantID != "101" || result.Status != MerchantLocationReportStatusValid {
		t.Fatalf("result = %#v, want reviewed report", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("review transaction expectations: %v", err)
	}
}

func TestListAdminMerchantLocationReportsReturnsPagedReports(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	createdAt := time.Date(2026, time.August, 29, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT mlr.id::text, mlr.status, mlr.merchant_id::text, m.name, mlr.reporter_user_id::text, mlr.reason_code, mlr.description, mlr.location_snapshot, COALESCE\(mlr.reviewed_by::text, ''\), COALESCE\(mlr.review_note, ''\), mlr.created_at, mlr.reviewed_at, COUNT\(\*\) OVER\(\) AS total FROM merchant_location_reports mlr JOIN merchants m ON m.id = mlr.merchant_id WHERE \(\$1 = '' OR mlr.status = \$1\) ORDER BY mlr.created_at DESC, mlr.id DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(MerchantLocationReportStatusPending, int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "merchant_id", "merchant_name", "reporter_user_id", "reason_code", "description", "location_snapshot", "reviewed_by", "review_note", "created_at", "reviewed_at", "total",
		}).AddRow("9001", MerchantLocationReportStatusPending, "101", "晨星童装", "201", "wrong_booth", "A108 已搬走", []byte(`{"boothNo":"A108"}`), "", "", createdAt, nil, int64(1)))

	result, err := NewMerchantLocationReportModel(db).ListAdminMerchantLocationReports(context.Background(), AdminMerchantLocationReportFilter{
		Status: MerchantLocationReportStatusPending, Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("ListAdminMerchantLocationReports() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != "9001" || result.Items[0].LocationSnapshot["boothNo"] != "A108" || result.Items[0].CreatedAt != createdAt.Format(time.RFC3339) {
		t.Fatalf("result = %#v, want one paged report", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("list expectations: %v", err)
	}
}

func TestListAdminMerchantLocationReportsRejectsOverflowingOffsetBeforeQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	_, err = NewMerchantLocationReportModel(db).ListAdminMerchantLocationReports(context.Background(), AdminMerchantLocationReportFilter{
		Page: math.MaxInt64, PageSize: 100,
	})
	if err == nil || err.Error() != "位置反馈分页参数超出范围" {
		t.Fatalf("error = %v, want pagination overflow rejection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database must not be queried: %v", err)
	}
}

func TestReviewMerchantLocationReportRollsBackWhenAuditLogFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT merchant_id::text, status, reason_code, description FROM merchant_location_reports WHERE id = \$1 FOR UPDATE`).
		WithArgs("9001").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "status", "reason_code", "description"}).AddRow("101", MerchantLocationReportStatusPending, "wrong_booth", ""))
	mock.ExpectExec(`UPDATE merchant_location_reports`).
		WithArgs("9001", MerchantLocationReportStatusInvalid, "701", "无法复现").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO operation_logs`).
		WithArgs("701", "platform_operator", "merchant_location_report_review", "merchant_location_report", "9001", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("audit unavailable"))
	mock.ExpectRollback()

	result, err := NewMerchantLocationReportModel(db).ReviewMerchantLocationReport(context.Background(), ReviewMerchantLocationReportInput{
		ReportID: "9001", Status: MerchantLocationReportStatusInvalid, ReviewNote: "无法复现", AdminID: "701", AdminRole: "platform_operator",
	})
	if err == nil {
		t.Fatal("ReviewMerchantLocationReport() error = nil, want audit rollback")
	}
	if result.ReportID != "9001" || result.MerchantID != "101" || result.Status != MerchantLocationReportStatusInvalid {
		t.Fatalf("result = %#v, want identifiers retained for failure diagnostics", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("review rollback expectations: %v", err)
	}
}

func TestUpdateMerchantLocationStatusRestoresOnlyValidCoordinatesAndAudits(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT location_status, latitude, longitude FROM merchants WHERE id = \$1 AND deleted_at IS NULL FOR UPDATE`).
		WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"location_status", "latitude", "longitude"}).AddRow(MerchantLocationStatusRiskHidden, 30.8732, 120.2255))
	mock.ExpectExec(`UPDATE merchants SET location_status = \$2, updated_at = now\(\) WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs("101", MerchantLocationStatusVisible).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO operation_logs`).
		WithArgs("701", "super_admin", "merchant_location_restore", "merchant", "101", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := NewMerchantLocationReportModel(db).UpdateMerchantLocationStatus(context.Background(), UpdateMerchantLocationStatusInput{
		MerchantID: "101", Status: MerchantLocationStatusVisible, Reason: "商家已核对", AdminID: "701", AdminRole: "super_admin",
	})
	if err != nil {
		t.Fatalf("UpdateMerchantLocationStatus() error = %v", err)
	}
	if result.OldStatus != MerchantLocationStatusRiskHidden || result.NewStatus != MerchantLocationStatusVisible {
		t.Fatalf("result = %#v, want risk_hidden to visible", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("restore transaction expectations: %v", err)
	}
}

func TestUpdateMerchantLocationStatusRejectsRestoreWithoutCoordinates(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT location_status, latitude, longitude FROM merchants WHERE id = \$1 AND deleted_at IS NULL FOR UPDATE`).
		WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"location_status", "latitude", "longitude"}).AddRow(MerchantLocationStatusHidden, nil, nil))
	mock.ExpectRollback()

	_, err = NewMerchantLocationReportModel(db).UpdateMerchantLocationStatus(context.Background(), UpdateMerchantLocationStatusInput{
		MerchantID: "101", Status: MerchantLocationStatusVisible, Reason: "人工恢复", AdminID: "701", AdminRole: "platform_operator",
	})
	if !errors.Is(err, ErrMerchantLocationCoordinatesMissing) {
		t.Fatalf("UpdateMerchantLocationStatus() error = %v, want missing coordinates", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("restore rejection expectations: %v", err)
	}
}

func TestUpdateMerchantLocationStatusRollsBackHideWhenAuditLogFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT location_status, latitude, longitude FROM merchants WHERE id = \$1 AND deleted_at IS NULL FOR UPDATE`).
		WithArgs("101").
		WillReturnRows(sqlmock.NewRows([]string{"location_status", "latitude", "longitude"}).AddRow(MerchantLocationStatusVisible, 30.8732, 120.2255))
	mock.ExpectExec(`UPDATE merchants SET location_status = \$2, updated_at = now\(\) WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs("101", MerchantLocationStatusHidden).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO operation_logs`).
		WithArgs("701", "platform_operator", "merchant_location_hide", "merchant", "101", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("audit unavailable"))
	mock.ExpectRollback()

	result, err := NewMerchantLocationReportModel(db).UpdateMerchantLocationStatus(context.Background(), UpdateMerchantLocationStatusInput{
		MerchantID: "101", Status: MerchantLocationStatusHidden, Reason: "等待商家核对", AdminID: "701", AdminRole: "platform_operator",
	})
	if err == nil {
		t.Fatal("UpdateMerchantLocationStatus() error = nil, want audit rollback")
	}
	if result.MerchantID != "101" || result.OldStatus != MerchantLocationStatusVisible || result.NewStatus != MerchantLocationStatusHidden {
		t.Fatalf("result = %#v, want hide identifiers retained for failure diagnostics", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("hide rollback expectations: %v", err)
	}
}

func expectVisibleMerchantLocationForUpdate(mock sqlmock.Sqlmock, merchantID string) {
	expectMerchantLocationForUpdate(mock, merchantID, MerchantLocationStatusVisible)
}

func expectMerchantLocationForUpdate(mock sqlmock.Sqlmock, merchantID string, status string) {
	expectMerchantLocationRowForUpdate(mock, merchantID, status, 30.8732, 120.2255)
}

func expectMerchantLocationWithoutCoordinatesForUpdate(mock sqlmock.Sqlmock, merchantID string, status string) {
	expectMerchantLocationRowForUpdate(mock, merchantID, status, nil, nil)
}

func expectMerchantLocationRowForUpdate(mock sqlmock.Sqlmock, merchantID string, status string, latitude any, longitude any) {
	mock.ExpectQuery(`SELECT.*market_name.*building_name.*floor_no.*booth_no.*address_text.*latitude.*longitude.*location_name.*location_status.*FROM merchants.*WHERE id = \$1.*FOR UPDATE`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"market_name", "building_name", "floor_no", "booth_no", "address_text", "latitude", "longitude", "location_name", "location_status",
		}).AddRow("织里童装城", "一期", "1F", "A108", "织里镇利济路", latitude, longitude, "织里童装城", status))
}
