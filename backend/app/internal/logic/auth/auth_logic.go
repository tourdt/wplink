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
	Token string       `json:"token"`
	User  AuthUserInfo `json:"user"`
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

type WechatLoginLogic struct {
	store        UserStore
	tokenService TokenService
}

func NewWechatLoginLogic(store UserStore, tokenService TokenService) *WechatLoginLogic {
	return &WechatLoginLogic{store: store, tokenService: tokenService}
}

func (l *WechatLoginLogic) WechatLogin(ctx context.Context, req WechatLoginReq) (WechatLoginResp, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return WechatLoginResp{}, errx.New(errx.CodeValidationFailed, "请提供微信登录凭证")
	}
	if l.tokenService == nil {
		return WechatLoginResp{}, errx.New(errx.CodeInternalError, "登录服务未配置，请稍后重试")
	}

	// MVP 环境没有微信 code2session 凭证，先使用 code 生成稳定开发 openid；真实接入时替换这里的 openid 获取边界。
	profile, err := l.store.UpsertWechatUser(ctx, model.UpsertWechatUserInput{
		WechatOpenID:    "dev:" + code,
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
	return WechatLoginResp{Token: token, User: authUserInfoFromProfile(profile)}, nil
}

type MeLogic struct {
	store UserStore
}

func NewMeLogic(store UserStore) *MeLogic {
	return &MeLogic{store: store}
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

	// 当前 MVP 尚未接入短信供应商，这里只做验证码非空校验，后续可替换为短信服务校验。
	profile, err := l.store.BindUserPhone(ctx, userID, phone)
	if err != nil {
		return BindPhoneResp{}, err
	}
	return BindPhoneResp{ID: profile.ID, Phone: profile.Phone}, nil
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

func meRespFromProfile(profile model.UserProfile) MeResp {
	managedMerchants := make([]ManagedMerchantInfo, 0, len(profile.ManagedMerchants))
	for _, merchant := range profile.ManagedMerchants {
		managedMerchants = append(managedMerchants, ManagedMerchantInfo{
			ID: merchant.ID, Name: merchant.Name, Role: merchant.Role,
		})
	}
	return MeResp{
		ID:               profile.ID,
		Phone:            profile.Phone,
		Nickname:         profile.Nickname,
		DefaultCityCode:  profile.DefaultCityCode,
		Roles:            normalizedRoles(profile.Roles),
		ManagedMerchants: managedMerchants,
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
