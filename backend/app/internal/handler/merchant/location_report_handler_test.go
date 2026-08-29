package merchant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestCreateMerchantLocationReportHandlerUsesAuthenticatedUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	expectVisibleMerchantForLocationReportHandler(mock, "101")
	mock.ExpectQuery(`SELECT mlr.id::text, mlr.status,.*FROM merchant_location_reports mlr`).
		WithArgs("101", "201").
		WillReturnError(sqlmock.ErrCancelled)
	// sql.ErrNoRows 才代表首次提交；重新建立期望以确保 Handler 传入 Token 用户而不是请求体用户。
	mock.ExpectRollback()

	svcCtx := &svc.ServiceContext{
		APIStore:         &svc.APIStore{MerchantLocationReportModel: model.NewMerchantLocationReportModel(db)},
		UserTokenService: fixedLocationReportUserTokenService{userID: "201"},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/101/location-reports", strings.NewReader(`{"reasonCode":"wrong_booth","description":"A108 已搬走"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("Content-Type", "application/json")
	req = pathvar.WithVars(req, map[string]string{"merchantId": "101"})
	recorder := httptest.NewRecorder()

	CreateMerchantLocationReportHandler(svcCtx).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want store failure after authenticated user reached transaction", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("identity SQL expectations: %v", err)
	}
}

func TestCreateMerchantLocationReportHandlerRequiresLogin(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/101/location-reports", strings.NewReader(`{"reasonCode":"wrong_booth"}`))
	req.Header.Set("Content-Type", "application/json")
	req = pathvar.WithVars(req, map[string]string{"merchantId": "101"})

	CreateMerchantLocationReportHandler(&svc.ServiceContext{UserTokenService: fixedLocationReportUserTokenService{userID: "201"}}).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want login required", recorder.Code, recorder.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func expectVisibleMerchantForLocationReportHandler(mock sqlmock.Sqlmock, merchantID string) {
	mock.ExpectQuery(`SELECT.*market_name.*FROM merchants.*WHERE id = \$1.*FOR UPDATE`).
		WithArgs(merchantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"market_name", "building_name", "floor_no", "booth_no", "address_text", "latitude", "longitude", "location_name", "location_status",
		}).AddRow("织里童装城", "一期", "1F", "A108", "织里镇利济路", 30.8732, 120.2255, "织里童装城", model.MerchantLocationStatusVisible))
}

type fixedLocationReportUserTokenService struct {
	userID string
}

func (s fixedLocationReportUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (s fixedLocationReportUserTokenService) ParseUserToken(context.Context, string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{UserID: s.userID}, nil
}

var _ authlogic.TokenService = fixedLocationReportUserTokenService{}
