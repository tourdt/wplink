package auth

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
)

func TestWechatLoginCreatesUserAndIssuesToken(t *testing.T) {
	store := &fakeAuthStore{}
	tokenService := &fakeTokenService{}
	logic := NewWechatLoginLogic(store, tokenService)

	resp, err := logic.WechatLogin(context.Background(), WechatLoginReq{Code: " wx-code ", DefaultCityCode: " zhili "})
	if err != nil {
		t.Fatalf("wechat login: %v", err)
	}
	if store.upsertInput.WechatOpenID != "dev:wx-code" || store.upsertInput.DefaultCityCode != "zhili" {
		t.Fatalf("upsert input = %#v, want trimmed dev openid and city", store.upsertInput)
	}
	if tokenService.subject.UserID != "user-1" || tokenService.subject.Roles[0] != RoleNormalUser {
		t.Fatalf("token subject = %#v, want user-1 normal_user", tokenService.subject)
	}
	if resp.Token != "user-token" || resp.User.ID != "user-1" {
		t.Fatalf("resp = %#v, want token and user", resp)
	}
}

func TestMeLogicReturnsProfileAndManagedMerchants(t *testing.T) {
	logic := NewMeLogic(&fakeAuthStore{})

	resp, err := logic.GetMe(context.Background(), " user-1 ")
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if resp.ID != "user-1" || resp.Phone != "18800000000" || len(resp.ManagedMerchants) != 1 {
		t.Fatalf("resp = %#v, want profile and merchant", resp)
	}
}

func TestBindPhoneRequiresPhoneAndSmsCode(t *testing.T) {
	logic := NewMeLogic(&fakeAuthStore{})

	_, err := logic.BindPhone(context.Background(), "user-1", BindPhoneReq{Phone: "18800000000"})
	if err == nil {
		t.Fatal("err = nil, want validation error")
	}
}

func TestBindPhoneUpdatesUserPhone(t *testing.T) {
	store := &fakeAuthStore{}
	logic := NewMeLogic(store)

	resp, err := logic.BindPhone(context.Background(), " user-1 ", BindPhoneReq{Phone: " 18800000001 ", SmsCode: "123456"})
	if err != nil {
		t.Fatalf("bind phone: %v", err)
	}
	if store.boundUserID != "user-1" || store.boundPhone != "18800000001" || resp.Phone != "18800000001" {
		t.Fatalf("bound user = %q phone = %q resp = %#v, want trimmed phone", store.boundUserID, store.boundPhone, resp)
	}
}

type fakeAuthStore struct {
	upsertInput model.UpsertWechatUserInput
	boundUserID string
	boundPhone  string
}

func (s *fakeAuthStore) UpsertWechatUser(ctx context.Context, input model.UpsertWechatUserInput) (model.UserProfile, error) {
	s.upsertInput = input
	return s.profile(""), nil
}

func (s *fakeAuthStore) GetUserProfile(ctx context.Context, userID string) (model.UserProfile, error) {
	return s.profile(""), nil
}

func (s *fakeAuthStore) BindUserPhone(ctx context.Context, userID string, phone string) (model.UserProfile, error) {
	s.boundUserID = userID
	s.boundPhone = phone
	return s.profile(phone), nil
}

func (s *fakeAuthStore) profile(phone string) model.UserProfile {
	if phone == "" {
		phone = "18800000000"
	}
	return model.UserProfile{
		ID:              "user-1",
		Phone:           phone,
		DefaultCityCode: "zhili",
		Roles:           []string{},
		ManagedMerchants: []model.ManagedMerchantInfo{{
			ID: "merchant-1", Name: "织里云仓", Role: "owner",
		}},
	}
}

type fakeTokenService struct {
	subject session.UserTokenSubject
}

func (s *fakeTokenService) IssueUserToken(ctx context.Context, subject session.UserTokenSubject) (string, error) {
	s.subject = subject
	return "user-token", nil
}

func (s *fakeTokenService) ParseUserToken(ctx context.Context, token string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{UserID: "user-1", Roles: []string{RoleNormalUser}}, nil
}
