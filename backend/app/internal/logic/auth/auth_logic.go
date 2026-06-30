package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"
)

const RoleNormalUser = "normal_user"

type UserStore interface {
	UpsertWechatUser(ctx context.Context, input model.UpsertWechatUserInput) (model.UserProfile, error)
	GetUserProfile(ctx context.Context, userID string) (model.UserProfile, error)
	BindUserPhone(ctx context.Context, userID string, phone string) (model.UserProfile, error)
}

type TokenService interface {
	IssueUserToken(ctx context.Context, subject session.UserTokenSubject) (string, error)
	ParseUserToken(ctx context.Context, token string) (session.UserTokenSubject, error)
}

type WechatLoginReq struct {
	Code            string `json:"code"`
	DefaultCityCode string `json:"defaultCityCode,omitempty"`
}

type AuthUserInfo struct {
	ID              string   `json:"id"`
	Nickname        string   `json:"nickname,omitempty"`
	AvatarURL       string   `json:"avatarUrl,omitempty"`
	DefaultCityCode string   `json:"defaultCityCode,omitempty"`
	Roles           []string `json:"roles"`
}

type WechatLoginResp struct {
	Token            string                `json:"token"`
	User             AuthUserInfo          `json:"user"`
	ManagedMerchants []ManagedMerchantInfo `json:"managedMerchants"`
}

type ManagedMerchantInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type MeResp struct {
	ID               string                `json:"id"`
	Phone            string                `json:"phone,omitempty"`
	Nickname         string                `json:"nickname,omitempty"`
	DefaultCityCode  string                `json:"defaultCityCode,omitempty"`
	Roles            []string              `json:"roles"`
	ManagedMerchants []ManagedMerchantInfo `json:"managedMerchants"`
}

type BindPhoneReq struct {
	Phone   string `json:"phone"`
	SmsCode string `json:"smsCode"`
}

type BindPhoneResp struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
}

type SendSMSCodeReq struct {
	Phone string `json:"phone"`
}

type SendSMSCodeResp struct {
	Message string `json:"message"`
}

type WechatLoginLogic struct {
	store         UserStore
	tokenService  TokenService
	sessionClient WechatSessionClient
}

func NewWechatLoginLogic(store UserStore, tokenService TokenService, sessionClient ...WechatSessionClient) *WechatLoginLogic {
	var client WechatSessionClient
	if len(sessionClient) > 0 {
		client = sessionClient[0]
	}
	return &WechatLoginLogic{store: store, tokenService: tokenService, sessionClient: client}
}

func (l *WechatLoginLogic) WechatLogin(ctx context.Context, req WechatLoginReq) (WechatLoginResp, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return WechatLoginResp{}, errx.New(errx.CodeValidationFailed, "请提供微信登录凭证")
	}
	if l.tokenService == nil {
		return WechatLoginResp{}, errx.New(errx.CodeInternalError, "登录服务未配置，请稍后重试")
	}
	if l.sessionClient == nil {
		return WechatLoginResp{}, errx.New(errx.CodeInternalError, "微信登录服务未配置，请稍后重试")
	}

	wechatSession, err := l.sessionClient.Code2Session(ctx, code)
	if err != nil {
		return WechatLoginResp{}, err
	}
	profile, err := l.store.UpsertWechatUser(ctx, model.UpsertWechatUserInput{
		WechatOpenID:    wechatSession.OpenID,
		DefaultCityCode: strings.TrimSpace(req.DefaultCityCode),
	})
	if err != nil {
		return WechatLoginResp{}, err
	}
	roles := normalizedRoles(profile.Roles)
	token, err := l.tokenService.IssueUserToken(ctx, session.UserTokenSubject{UserID: profile.ID, Roles: roles})
	if err != nil {
		return WechatLoginResp{}, err
	}
	profile.Roles = roles
	return WechatLoginResp{
		Token:            token,
		User:             authUserInfoFromProfile(profile),
		ManagedMerchants: managedMerchantInfosFromProfile(profile),
	}, nil
}

type MeLogic struct {
	store       UserStore
	smsVerifier SMSVerifier
}

func NewMeLogic(store UserStore, verifier ...SMSVerifier) *MeLogic {
	var smsVerifier SMSVerifier
	if len(verifier) > 0 {
		smsVerifier = verifier[0]
	}
	return &MeLogic{store: store, smsVerifier: smsVerifier}
}

func (l *MeLogic) GetMe(ctx context.Context, userID string) (MeResp, error) {
	profile, err := l.userProfile(ctx, userID)
	if err != nil {
		return MeResp{}, err
	}
	return meRespFromProfile(profile), nil
}

func (l *MeLogic) BindPhone(ctx context.Context, userID string, req BindPhoneReq) (BindPhoneResp, error) {
	userID = strings.TrimSpace(userID)
	phone := strings.TrimSpace(req.Phone)
	smsCode := strings.TrimSpace(req.SmsCode)
	if userID == "" {
		return BindPhoneResp{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	if phone == "" || smsCode == "" {
		return BindPhoneResp{}, errx.New(errx.CodeValidationFailed, "请填写手机号和短信验证码")
	}
	if len(phone) < 6 || len(phone) > 20 {
		return BindPhoneResp{}, errx.New(errx.CodeValidationFailed, "手机号格式不正确")
	}

	if l.smsVerifier == nil {
		return BindPhoneResp{}, errx.New(errx.CodeInternalError, "短信服务未配置，请稍后重试")
	}
	if err := l.smsVerifier.VerifySMSCode(ctx, phone, smsCode); err != nil {
		return BindPhoneResp{}, err
	}
	profile, err := l.store.BindUserPhone(ctx, userID, phone)
	if err != nil {
		return BindPhoneResp{}, err
	}
	return BindPhoneResp{ID: profile.ID, Phone: profile.Phone}, nil
}

type SendSMSCodeLogic struct {
	sender SMSCodeSender
}

func NewSendSMSCodeLogic(sender SMSCodeSender) *SendSMSCodeLogic {
	return &SendSMSCodeLogic{sender: sender}
}

func (l *SendSMSCodeLogic) SendSMSCode(ctx context.Context, req SendSMSCodeReq) (SendSMSCodeResp, error) {
	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		return SendSMSCodeResp{}, errx.New(errx.CodeValidationFailed, "请填写手机号")
	}
	if len(phone) < 6 || len(phone) > 20 {
		return SendSMSCodeResp{}, errx.New(errx.CodeValidationFailed, "手机号格式不正确")
	}
	if l.sender == nil {
		return SendSMSCodeResp{}, errx.New(errx.CodeInternalError, "短信服务未配置，请稍后重试")
	}
	if err := l.sender.SendSMSCode(ctx, phone); err != nil {
		return SendSMSCodeResp{}, err
	}
	return SendSMSCodeResp{Message: "验证码已发送"}, nil
}

func (l *MeLogic) userProfile(ctx context.Context, userID string) (model.UserProfile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return model.UserProfile{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	profile, err := l.store.GetUserProfile(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.UserProfile{}, errx.New(errx.CodeUnauthorized, "登录状态无效，请重新登录")
	}
	if err != nil {
		return model.UserProfile{}, err
	}
	profile.Roles = normalizedRoles(profile.Roles)
	return profile, nil
}

func authUserInfoFromProfile(profile model.UserProfile) AuthUserInfo {
	return AuthUserInfo{
		ID:              profile.ID,
		Nickname:        profile.Nickname,
		AvatarURL:       profile.AvatarURL,
		DefaultCityCode: profile.DefaultCityCode,
		Roles:           normalizedRoles(profile.Roles),
	}
}

func managedMerchantInfosFromProfile(profile model.UserProfile) []ManagedMerchantInfo {
	managedMerchants := make([]ManagedMerchantInfo, 0, len(profile.ManagedMerchants))
	for _, merchant := range profile.ManagedMerchants {
		managedMerchants = append(managedMerchants, ManagedMerchantInfo{
			ID: merchant.ID, Name: merchant.Name, Role: merchant.Role,
		})
	}
	return managedMerchants
}

func meRespFromProfile(profile model.UserProfile) MeResp {
	return MeResp{
		ID:               profile.ID,
		Phone:            profile.Phone,
		Nickname:         profile.Nickname,
		DefaultCityCode:  profile.DefaultCityCode,
		Roles:            normalizedRoles(profile.Roles),
		ManagedMerchants: managedMerchantInfosFromProfile(profile),
	}
}

func normalizedRoles(roles []string) []string {
	result := make([]string, 0, len(roles))
	seen := map[string]struct{}{}
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		result = append(result, role)
	}
	if len(result) == 0 {
		return []string{RoleNormalUser}
	}
	return result
}
