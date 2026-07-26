package auth

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"
)

func TestWechatLoginCreatesUserAndIssuesToken(t *testing.T) {
	store := &fakeAuthStore{}
	tokenService := &fakeTokenService{}
	sessionClient := &fakeWechatSessionClient{session: WechatSession{OpenID: "openid-1", UnionID: "union-1"}}
	logic := NewWechatLoginLogic(store, tokenService, sessionClient)

	resp, err := logic.WechatLogin(context.Background(), validWechatLoginReq(" wx-code "))
	if err != nil {
		t.Fatalf("wechat login: %v", err)
	}
	if sessionClient.code != "wx-code" {
		t.Fatalf("code = %q, want trimmed code", sessionClient.code)
	}
	if store.upsertInput.WechatOpenID != "openid-1" || store.upsertInput.DefaultCityCode != "zhili" {
		t.Fatalf("upsert input = %#v, want wechat openid and city", store.upsertInput)
	}
	if tokenService.subject.UserID != "user-1" || tokenService.subject.Roles[0] != RoleNormalUser {
		t.Fatalf("token subject = %#v, want user-1 normal_user", tokenService.subject)
	}
	if resp.Token != "user-token" || resp.User.ID != "user-1" {
		t.Fatalf("resp = %#v, want token and user", resp)
	}
	if len(resp.ManagedMerchants) != 1 || resp.ManagedMerchants[0].ID != "merchant-1" {
		t.Fatalf("managed merchants = %#v, want merchant-1", resp.ManagedMerchants)
	}
	if store.growthInput.EventType != model.GrowthEventUserFirstLogin || store.growthInput.MerchantID != "merchant-1" || store.growthInput.UserID != "user-1" {
		t.Fatalf("growthInput = %#v, want first login event for default merchant", store.growthInput)
	}
}

func TestWechatLoginHidesStoreFailure(t *testing.T) {
	store := &fakeAuthStore{upsertErr: errors.New(`pq: relation "users" does not exist`)}
	tokenService := &fakeTokenService{}
	sessionClient := &fakeWechatSessionClient{session: WechatSession{OpenID: "openid-1"}}
	logic := NewWechatLoginLogic(store, tokenService, sessionClient)

	_, err := logic.WechatLogin(context.Background(), validWechatLoginReq("wx-code"))
	if err == nil {
		t.Fatal("err = nil, want friendly internal error")
	}
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "登录失败，请稍后重试" {
		t.Fatalf("err = %#v code=%s msg=%q, want safe login failure", err, errx.CodeOf(err), errx.PublicMessage(err))
	}
}

func TestWechatLoginHidesTokenIssueFailure(t *testing.T) {
	store := &fakeAuthStore{}
	tokenService := &fakeTokenService{issueErr: errors.New("用户 token 密钥未配置")}
	sessionClient := &fakeWechatSessionClient{session: WechatSession{OpenID: "openid-1"}}
	logic := NewWechatLoginLogic(store, tokenService, sessionClient)

	_, err := logic.WechatLogin(context.Background(), validWechatLoginReq("wx-code"))
	if err == nil {
		t.Fatal("err = nil, want friendly internal error")
	}
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "登录状态生成失败，请稍后重试" {
		t.Fatalf("err = %#v code=%s msg=%q, want safe token failure", err, errx.CodeOf(err), errx.PublicMessage(err))
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
	verifier := &fakeSMSVerifier{}
	logic := NewMeLogic(store, verifier)

	resp, err := logic.BindPhone(context.Background(), " user-1 ", BindPhoneReq{Phone: " 18800000001 ", SmsCode: "123456"})
	if err != nil {
		t.Fatalf("bind phone: %v", err)
	}
	if store.boundUserID != "user-1" || store.boundPhone != "18800000001" || resp.Phone != "18800000001" {
		t.Fatalf("bound user = %q phone = %q resp = %#v, want trimmed phone", store.boundUserID, store.boundPhone, resp)
	}
	if verifier.phone != "18800000001" || verifier.code != "123456" {
		t.Fatalf("sms verifier phone/code = %q/%q, want trimmed values", verifier.phone, verifier.code)
	}
}

func TestBindWechatPhoneUpdatesUserPhone(t *testing.T) {
	store := &fakeAuthStore{}
	wechatClient := &fakeWechatSessionClient{phoneNumber: WechatPhoneNumber{PurePhoneNumber: "18800000003"}}
	logic := NewMeLogic(store)

	resp, err := logic.BindWechatPhone(context.Background(), " user-1 ", BindWechatPhoneReq{Code: " phone-code "}, wechatClient)
	if err != nil {
		t.Fatalf("bind wechat phone: %v", err)
	}
	if wechatClient.phoneCode != "phone-code" {
		t.Fatalf("wechat phone code = %q, want trimmed code", wechatClient.phoneCode)
	}
	if store.boundUserID != "user-1" || store.boundPhone != "18800000003" || resp.Phone != "18800000003" {
		t.Fatalf("bound user = %q phone = %q resp = %#v, want wechat phone", store.boundUserID, store.boundPhone, resp)
	}
}

func TestBindWechatPhoneRequiresCode(t *testing.T) {
	logic := NewMeLogic(&fakeAuthStore{})

	_, err := logic.BindWechatPhone(context.Background(), "user-1", BindWechatPhoneReq{}, &fakeWechatSessionClient{})
	if err == nil {
		t.Fatal("err = nil, want validation error")
	}
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("err code = %s, want validation failed", errx.CodeOf(err))
	}
}

func TestSendSMSCodeTrimsPhoneAndCallsSender(t *testing.T) {
	sender := &fakeSMSVerifier{}
	logic := NewSendSMSCodeLogic(sender)

	resp, err := logic.SendSMSCode(context.Background(), SendSMSCodeReq{Phone: " 18800000002 "})
	if err != nil {
		t.Fatalf("send sms code: %v", err)
	}
	if resp.Message != "验证码已发送" || sender.sentPhone != "18800000002" {
		t.Fatalf("resp = %#v sent phone = %q, want trimmed phone", resp, sender.sentPhone)
	}
}

type fakeAuthStore struct {
	upsertInput model.UpsertWechatUserInput
	growthInput model.GrowthEventInput
	boundUserID string
	boundPhone  string
	upsertErr   error
}

func validWechatLoginReq(code string) WechatLoginReq {
	return WechatLoginReq{
		Code:                 code,
		DefaultCityCode:      " zhili ",
		AgreedToPolicies:     true,
		PrivacyPolicyVersion: "2026-07-25",
		UserAgreementVersion: "2026-07-25",
	}
}

func (s *fakeAuthStore) RecordUserConsents(ctx context.Context, userID string, privacyVersion string, agreementVersion string) error {
	return nil
}

func (s *fakeAuthStore) DeleteUserAccount(ctx context.Context, userID string, reason string) error {
	return nil
}

func (s *fakeAuthStore) UpsertWechatUser(ctx context.Context, input model.UpsertWechatUserInput) (model.UserProfile, error) {
	s.upsertInput = input
	if s.upsertErr != nil {
		return model.UserProfile{}, s.upsertErr
	}
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

func (s *fakeAuthStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.growthInput = input
	return nil, nil
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
	subject  session.UserTokenSubject
	issueErr error
}

func (s *fakeTokenService) IssueUserToken(ctx context.Context, subject session.UserTokenSubject) (string, error) {
	s.subject = subject
	if s.issueErr != nil {
		return "", s.issueErr
	}
	return "user-token", nil
}

func (s *fakeTokenService) ParseUserToken(ctx context.Context, token string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{UserID: "user-1", Roles: []string{RoleNormalUser}}, nil
}

type fakeWechatSessionClient struct {
	code        string
	phoneCode   string
	session     WechatSession
	phoneNumber WechatPhoneNumber
}

func (s *fakeWechatSessionClient) Code2Session(ctx context.Context, code string) (WechatSession, error) {
	s.code = code
	return s.session, nil
}

func (s *fakeWechatSessionClient) GetPhoneNumber(ctx context.Context, code string) (WechatPhoneNumber, error) {
	s.phoneCode = code
	return s.phoneNumber, nil
}

type fakeSMSVerifier struct {
	phone     string
	code      string
	sentPhone string
}

func (s *fakeSMSVerifier) VerifySMSCode(ctx context.Context, phone string, code string) error {
	s.phone = phone
	s.code = code
	return nil
}

func (s *fakeSMSVerifier) SendSMSCode(ctx context.Context, phone string) error {
	s.sentPhone = phone
	return nil
}
