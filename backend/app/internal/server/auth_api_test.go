package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAuthGeneratedWechatLoginAndSMSCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	expectWechatLoginProfile(mock)
	tokenService := &fakeUserTokenService{}
	wechatClient := &fakeAuthWechatSessionClient{session: authlogic.WechatSession{OpenID: "openid-1"}}
	smsVerifier := &fakeAuthSMSVerifier{}
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{
		APIStore: &svc.APIStore{
			UserModel:           model.NewUserModel(db),
			GrowthCampaignModel: model.NewGrowthCampaignModel(db),
		},
		UserTokenService:    tokenService,
		WechatSessionClient: wechatClient,
		SMSVerifier:         smsVerifier,
	})

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/wechat-login", strings.NewReader(`{"code":" wx-code ","defaultCityCode":"zhili","agreedToPolicies":true,"privacyPolicyVersion":"2026-07-25","userAgreementVersion":"2026-07-25"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(loginRec, loginReq)
	loginData := decodeEnvelopeData(t, loginRec, http.StatusOK)
	if loginData["token"] != "user-token" {
		t.Fatalf("login data = %#v, want user-token", loginData)
	}
	if wechatClient.code != "wx-code" {
		t.Fatalf("wechat code = %q, want trimmed wx-code", wechatClient.code)
	}
	managedMerchants, ok := loginData["managedMerchants"].([]interface{})
	if !ok || len(managedMerchants) != 1 {
		t.Fatalf("managed merchants = %#v, want one item", loginData["managedMerchants"])
	}

	smsRec := httptest.NewRecorder()
	smsReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sms-code", strings.NewReader(`{"phone":" 18800000002 "}`))
	smsReq.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(smsRec, smsReq)
	smsData := decodeEnvelopeData(t, smsRec, http.StatusOK)
	if smsData["message"] != "验证码已发送" || smsVerifier.sentPhone != "18800000002" {
		t.Fatalf("sms data = %#v sent phone = %q", smsData, smsVerifier.sentPhone)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("auth SQL expectations: %v", err)
	}
}

func TestAuthGeneratedMeRejectsMissingAndExpiredSessionWithoutLeakingRawError(t *testing.T) {
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{
		UserTokenService: &rawErrorUserTokenService{},
	})

	missingRec := httptest.NewRecorder()
	server.ServeHTTP(missingRec, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	assertErrorEnvelope(t, missingRec, http.StatusUnauthorized, errx.CodeUnauthorized, "请先登录")

	expiredRec := httptest.NewRecorder()
	expiredReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	expiredReq.Header.Set("Authorization", "Bearer expired-token")
	server.ServeHTTP(expiredRec, expiredReq)
	assertErrorEnvelope(t, expiredRec, http.StatusUnauthorized, errx.CodeUnauthorized, "登录已过期，请重新登录")
	if strings.Contains(expiredRec.Body.String(), "jwt signature invalid") {
		t.Fatalf("expired response leaked raw authentication error: %s", expiredRec.Body.String())
	}
}

func TestAuthGeneratedPrivateRoutesUseOnlyTokenUserIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	apiStore := &svc.APIStore{UserModel: model.NewUserModel(db)}
	wechatClient := &fakeAuthWechatSessionClient{}
	smsVerifier := &fakeAuthSMSVerifier{}
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{
		APIStore:            apiStore,
		UserTokenService:    &fakeUserTokenService{},
		WechatSessionClient: wechatClient,
		SMSVerifier:         smsVerifier,
	})

	expectUserProfile(mock, "user-1", "18800000000")
	meRec := httptest.NewRecorder()
	meReq := authenticatedRequest(http.MethodGet, "/api/v1/me?userId=attacker", "")
	server.ServeHTTP(meRec, meReq)
	meData := decodeEnvelopeData(t, meRec, http.StatusOK)
	if meData["id"] != "user-1" {
		t.Fatalf("me data = %#v, want token user", meData)
	}

	expectBindPhone(mock, "user-1", "18800000001")
	phoneRec := httptest.NewRecorder()
	phoneReq := authenticatedRequest(http.MethodPost, "/api/v1/me/phone?userId=attacker", `{"userId":"attacker","phone":"18800000001","smsCode":"123456"}`)
	server.ServeHTTP(phoneRec, phoneReq)
	phoneData := decodeEnvelopeData(t, phoneRec, http.StatusOK)
	if phoneData["id"] != "user-1" || phoneData["phone"] != "18800000001" {
		t.Fatalf("phone data = %#v, want token user phone", phoneData)
	}
	if smsVerifier.phone != "18800000001" || smsVerifier.code != "123456" {
		t.Fatalf("sms verification = %q/%q", smsVerifier.phone, smsVerifier.code)
	}

	expectBindPhone(mock, "user-1", "18800000003")
	wechatPhoneRec := httptest.NewRecorder()
	wechatPhoneReq := authenticatedRequest(http.MethodPost, "/api/v1/me/wechat-phone?userId=attacker", `{"userId":"attacker","code":" phone-code "}`)
	server.ServeHTTP(wechatPhoneRec, wechatPhoneReq)
	wechatPhoneData := decodeEnvelopeData(t, wechatPhoneRec, http.StatusOK)
	if wechatPhoneData["id"] != "user-1" || wechatPhoneData["phone"] != "18800000003" {
		t.Fatalf("wechat phone data = %#v, want token user phone", wechatPhoneData)
	}
	if wechatClient.phoneCode != "phone-code" {
		t.Fatalf("wechat phone code = %q, want trimmed phone-code", wechatClient.phoneCode)
	}

	expectDeleteAccount(mock, "user-1", "不再使用")
	deleteRec := httptest.NewRecorder()
	deleteReq := authenticatedRequest(http.MethodPost, "/api/v1/me/account-deletion?userId=attacker", `{"userId":"attacker","confirmation":"确认注销","reason":" 不再使用 "}`)
	server.ServeHTTP(deleteRec, deleteReq)
	deleteData := decodeEnvelopeData(t, deleteRec, http.StatusOK)
	if deleteData["message"] != "账号已注销，个人资料已清理" {
		t.Fatalf("delete data = %#v", deleteData)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("private auth SQL expectations: %v", err)
	}
}

func authenticatedRequest(method string, target string, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer user-token")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func assertErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder, status int, code string, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d body = %s, want %d", rec.Code, rec.Body.String(), status)
	}
	var body struct {
		ErrorCode string `json:"errorCode"`
		Message   string `json:"msg"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.ErrorCode != code || body.Message != message {
		t.Fatalf("error = (%q, %q), want (%q, %q)", body.ErrorCode, body.Message, code, message)
	}
}

func expectWechatLoginProfile(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`WITH city AS`).
		WithArgs("openid-1", "zhili").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("user-1", model.UserStatusActive))
	mock.ExpectExec(`INSERT INTO user_role_assignments`).
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectUserProfile(mock, "user-1", "18800000000")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO user_consents`).
		WithArgs("user-1", "privacy_policy", authlogic.CurrentPrivacyPolicyVersion).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO user_consents`).
		WithArgs("user-1", "user_agreement", authlogic.CurrentUserAgreementVersion).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT gcr.campaign_code`).
		WithArgs(model.GrowthEventUserFirstLogin).
		WillReturnRows(sqlmock.NewRows([]string{
			"campaign_code", "rule_code", "trigger_event", "conditions", "reward_type",
			"reward_amount", "valid_days", "per_user_limit", "per_user_daily_limit", "per_resource_daily_limit",
		}))
}

func expectUserProfile(mock sqlmock.Sqlmock, userID string, phone string) {
	mock.ExpectQuery(`SELECT u.id::text, u.status`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "phone", "wechat_openid", "nickname", "avatar_url", "city_code"}).
			AddRow(userID, model.UserStatusActive, phone, "openid-1", "微信用户", "", "zhili"))
	mock.ExpectQuery(`SELECT DISTINCT r.code`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow(authlogic.RoleNormalUser))
	mock.ExpectQuery(`SELECT m.id::text, m.name, mab.role`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role", "profile_status"}).
			AddRow("merchant-1", "织里云仓", "owner", model.MerchantProfileStatusCompleted))
}

func expectBindPhone(mock sqlmock.Sqlmock, userID string, phone string) {
	mock.ExpectQuery(`UPDATE users SET phone = \$2`).
		WithArgs(userID, phone).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))
	expectUserProfile(mock, userID, phone)
}

func expectDeleteAccount(mock sqlmock.Sqlmock, userID string, reason string) {
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM users`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(model.UserStatusActive))
	mock.ExpectExec(`INSERT INTO user_account_deletion_requests`).
		WithArgs(userID, reason).
		WillReturnResult(sqlmock.NewResult(0, 1))
	for _, query := range []string{
		`UPDATE merchant_admin_bindings`,
		`UPDATE user_consents`,
		`UPDATE user_favorite_resources`,
		`UPDATE user_followed_merchants`,
		`UPDATE user_saved_searches`,
		`UPDATE search_logs`,
		`UPDATE resource_contact_events`,
		`UPDATE merchant_map_events`,
		`UPDATE users SET phone = NULL`,
	} {
		mock.ExpectExec(query).WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()
}

type fakeUserTokenService struct{}

func (s *fakeUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (s *fakeUserTokenService) ParseUserToken(_ context.Context, token string) (session.UserTokenSubject, error) {
	if token != "user-token" {
		return session.UserTokenSubject{}, errors.New("登录状态无效，请重新登录")
	}
	return session.UserTokenSubject{UserID: "user-1", Roles: []string{authlogic.RoleNormalUser}}, nil
}

type rawErrorUserTokenService struct{}

func (s *rawErrorUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (s *rawErrorUserTokenService) ParseUserToken(context.Context, string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{}, errors.New("jwt signature invalid: raw verifier detail")
}

type fakeAuthWechatSessionClient struct {
	code        string
	phoneCode   string
	session     authlogic.WechatSession
	phoneNumber authlogic.WechatPhoneNumber
}

func (s *fakeAuthWechatSessionClient) Code2Session(_ context.Context, code string) (authlogic.WechatSession, error) {
	s.code = code
	return s.session, nil
}

func (s *fakeAuthWechatSessionClient) GetPhoneNumber(_ context.Context, code string) (authlogic.WechatPhoneNumber, error) {
	s.phoneCode = code
	if strings.TrimSpace(s.phoneNumber.PurePhoneNumber) == "" {
		return authlogic.WechatPhoneNumber{PurePhoneNumber: "18800000003"}, nil
	}
	return s.phoneNumber, nil
}

type fakeAuthSMSVerifier struct {
	phone     string
	code      string
	sentPhone string
}

func (s *fakeAuthSMSVerifier) VerifySMSCode(_ context.Context, phone string, code string) error {
	s.phone = phone
	s.code = code
	return nil
}

func (s *fakeAuthSMSVerifier) SendSMSCode(_ context.Context, phone string) error {
	s.sentPhone = phone
	return nil
}
