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
	router := NewAPIRouter(store, WithUserTokenService(tokenService))

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/wechat-login", strings.NewReader(`{"code":"wx-code","defaultCityCode":"zhili"}`))
	router.ServeHTTP(loginRec, loginReq)
	loginData := decodeEnvelopeData(t, loginRec, http.StatusOK)
	if loginData["token"] != "user-token" {
		t.Fatalf("login data = %#v, want user-token", loginData)
	}
	if store.upsertInput.WechatOpenID != "dev:wx-code" || store.upsertInput.DefaultCityCode != "zhili" {
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
