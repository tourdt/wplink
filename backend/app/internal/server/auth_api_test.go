package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
)

func TestAuthAPIRouterRunsLoginMeAndBindPhoneFlow(t *testing.T) {
	store := &fakeAuthAPIStore{}
	tokenService := &fakeUserTokenService{}
	wechatClient := &fakeAuthWechatSessionClient{session: authlogic.WechatSession{OpenID: "openid-1"}}
	smsVerifier := &fakeAuthSMSVerifier{}
	router := NewAPIRouter(store, WithUserTokenService(tokenService), WithWechatSessionClient(wechatClient), WithSMSVerifier(smsVerifier))

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/wechat-login", strings.NewReader(`{"code":"wx-code","defaultCityCode":"zhili"}`))
	router.ServeHTTP(loginRec, loginReq)
	loginData := decodeEnvelopeData(t, loginRec, http.StatusOK)
	if loginData["token"] != "user-token" {
		t.Fatalf("login data = %#v, want user-token", loginData)
	}
	if wechatClient.code != "wx-code" {
		t.Fatalf("wechat code = %q, want trimmed code", wechatClient.code)
	}
	if store.upsertInput.WechatOpenID != "openid-1" || store.upsertInput.DefaultCityCode != "zhili" {
		t.Fatalf("upsert input = %#v, want login mapped to user", store.upsertInput)
	}

	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(meRec, meReq)
	meData := decodeEnvelopeData(t, meRec, http.StatusOK)
	if meData["id"] != "user-1" || len(meData["managedMerchants"].([]interface{})) != 1 {
		t.Fatalf("me data = %#v, want profile", meData)
	}

	phoneRec := httptest.NewRecorder()
	phoneReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/phone", strings.NewReader(`{"phone":"18800000001","smsCode":"123456"}`))
	phoneReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(phoneRec, phoneReq)
	phoneData := decodeEnvelopeData(t, phoneRec, http.StatusOK)
	if phoneData["phone"] != "18800000001" || store.boundPhone != "18800000001" {
		t.Fatalf("phone data = %#v bound = %q, want bound phone", phoneData, store.boundPhone)
	}
	if smsVerifier.phone != "18800000001" || smsVerifier.code != "123456" {
		t.Fatalf("sms verifier = %q/%q, want trimmed phone and code", smsVerifier.phone, smsVerifier.code)
	}

	smsRec := httptest.NewRecorder()
	smsReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sms-code", strings.NewReader(`{"phone":"18800000002"}`))
	router.ServeHTTP(smsRec, smsReq)
	smsData := decodeEnvelopeData(t, smsRec, http.StatusOK)
	if smsData["message"] != "验证码已发送" || smsVerifier.sentPhone != "18800000002" {
		t.Fatalf("sms data = %#v sent phone = %q, want sms code sent", smsData, smsVerifier.sentPhone)
	}
}

type fakeAuthAPIStore struct {
	fakeCityAPIStore

	upsertInput model.UpsertWechatUserInput
	boundPhone  string
}

func (s *fakeAuthAPIStore) UpsertWechatUser(ctx context.Context, input model.UpsertWechatUserInput) (model.UserProfile, error) {
	s.upsertInput = input
	return s.profile("18800000000"), nil
}

func (s *fakeAuthAPIStore) GetUserProfile(ctx context.Context, userID string) (model.UserProfile, error) {
	return s.profile("18800000000"), nil
}

func (s *fakeAuthAPIStore) BindUserPhone(ctx context.Context, userID string, phone string) (model.UserProfile, error) {
	s.boundPhone = phone
	return s.profile(phone), nil
}

func (s *fakeAuthAPIStore) profile(phone string) model.UserProfile {
	return model.UserProfile{
		ID:              "user-1",
		Phone:           phone,
		DefaultCityCode: "zhili",
		Roles:           []string{authlogic.RoleNormalUser},
		ManagedMerchants: []model.ManagedMerchantInfo{{
			ID: "merchant-1", Name: "织里云仓", Role: "owner",
		}},
	}
}

type fakeUserTokenService struct{}

func (s *fakeUserTokenService) IssueUserToken(ctx context.Context, subject session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (s *fakeUserTokenService) ParseUserToken(ctx context.Context, token string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{UserID: "user-1", Roles: []string{authlogic.RoleNormalUser}}, nil
}

type fakeAuthWechatSessionClient struct {
	code    string
	session authlogic.WechatSession
}

func (s *fakeAuthWechatSessionClient) Code2Session(ctx context.Context, code string) (authlogic.WechatSession, error) {
	s.code = code
	return s.session, nil
}

type fakeAuthSMSVerifier struct {
	phone     string
	code      string
	sentPhone string
}

func (s *fakeAuthSMSVerifier) VerifySMSCode(ctx context.Context, phone string, code string) error {
	s.phone = phone
	s.code = code
	return nil
}

func (s *fakeAuthSMSVerifier) SendSMSCode(ctx context.Context, phone string) error {
	s.sentPhone = phone
	return nil
}
