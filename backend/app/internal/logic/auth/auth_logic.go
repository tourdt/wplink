package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	RoleNormalUser              = "normal_user"
	CurrentPrivacyPolicyVersion = "2026-07-25"
	CurrentUserAgreementVersion = "2026-07-25"
)

type UserStore interface {
	UpsertWechatUser(ctx context.Context, input model.UpsertWechatUserInput) (model.UserProfile, error)
	GetUserProfile(ctx context.Context, userID string) (model.UserProfile, error)
	BindUserPhone(ctx context.Context, userID string, phone string) (model.UserProfile, error)
}

type DefaultMerchantStore interface {
	EnsureDefaultMerchantForUser(ctx context.Context, userID string, cityCode string) (model.ManagedMerchantInfo, error)
}

type TokenService interface {
	IssueUserToken(ctx context.Context, subject session.UserTokenSubject) (string, error)
	ParseUserToken(ctx context.Context, token string) (session.UserTokenSubject, error)
}

type GrowthEventStore interface {
	TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
}

type UserConsentStore interface {
	RecordUserConsents(ctx context.Context, userID string, privacyVersion string, agreementVersion string) error
}

type UserAccountStore interface {
	DeleteUserAccount(ctx context.Context, userID string, reason string) error
}

type WechatLoginReq struct {
	Code                 string `json:"code"`
	DefaultCityCode      string `json:"defaultCityCode,omitempty"`
	AgreedToPolicies     bool   `json:"agreedToPolicies"`
	PrivacyPolicyVersion string `json:"privacyPolicyVersion"`
	UserAgreementVersion string `json:"userAgreementVersion"`
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
	ID            string `json:"id"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	ProfileStatus string `json:"profileStatus"`
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

type BindWechatPhoneReq struct {
	Code string `json:"code"`
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

type DeleteAccountReq struct {
	Confirmation string `json:"confirmation"`
	Reason       string `json:"reason,omitempty"`
}

type DeleteAccountResp struct {
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
	privacyVersion := strings.TrimSpace(req.PrivacyPolicyVersion)
	agreementVersion := strings.TrimSpace(req.UserAgreementVersion)
	if !req.AgreedToPolicies || privacyVersion == "" || agreementVersion == "" {
		return WechatLoginResp{}, errx.New(errx.CodeValidationFailed, "请阅读并同意用户协议和隐私政策")
	}
	if privacyVersion != CurrentPrivacyPolicyVersion || agreementVersion != CurrentUserAgreementVersion {
		return WechatLoginResp{}, errx.New(errx.CodeValidationFailed, "协议版本已更新，请重新阅读并同意")
	}
	if l.tokenService == nil {
		return WechatLoginResp{}, errx.New(errx.CodeInternalError, "登录服务未配置，请稍后重试")
	}
	if l.sessionClient == nil {
		return WechatLoginResp{}, errx.New(errx.CodeInternalError, "微信登录服务未配置，请稍后重试")
	}

	wechatSession, err := l.sessionClient.Code2Session(ctx, code)
	if err != nil {
		logx.Errorf("微信登录换取 session 失败: defaultCityCode=%s err=%+v", strings.TrimSpace(req.DefaultCityCode), err)
		return WechatLoginResp{}, loginDependencyError(err, "登录失败，请稍后重试")
	}
	profile, err := l.store.UpsertWechatUser(ctx, model.UpsertWechatUserInput{
		WechatOpenID:    wechatSession.OpenID,
		DefaultCityCode: strings.TrimSpace(req.DefaultCityCode),
	})
	if err != nil {
		if errors.Is(err, model.ErrUserDisabled) {
			logx.Infof("微信登录被拦截，账号已停用: defaultCityCode=%s", strings.TrimSpace(req.DefaultCityCode))
			return WechatLoginResp{}, errx.New(errx.CodeForbidden, "账号已停用或已注销，如有疑问请联系客服")
		}
		logx.Errorf("微信登录写入用户失败: defaultCityCode=%s openidPresent=%t err=%+v", strings.TrimSpace(req.DefaultCityCode), strings.TrimSpace(wechatSession.OpenID) != "", err)
		return WechatLoginResp{}, loginDependencyError(err, "登录失败，请稍后重试")
	}
	consentStore, ok := l.store.(UserConsentStore)
	if !ok {
		logx.Errorf("微信登录缺少协议同意记录能力: userId=%s", profile.ID)
		return WechatLoginResp{}, errx.New(errx.CodeInternalError, "登录服务暂不可用，请稍后重试")
	}
	if err := consentStore.RecordUserConsents(ctx, profile.ID, privacyVersion, agreementVersion); err != nil {
		logx.Errorf("记录用户协议同意失败: userId=%s privacyVersion=%s agreementVersion=%s err=%+v", profile.ID, privacyVersion, agreementVersion, err)
		return WechatLoginResp{}, errx.New(errx.CodeInternalError, "登录服务暂不可用，请稍后重试")
	}
	profile, err = l.ensureDefaultMerchantProfile(ctx, profile, req.DefaultCityCode)
	if err != nil {
		return WechatLoginResp{}, err
	}
	l.triggerFirstLoginGrowthReward(ctx, profile)
	roles := normalizedRoles(profile.Roles)
	token, err := l.tokenService.IssueUserToken(ctx, session.UserTokenSubject{UserID: profile.ID, Roles: roles})
	if err != nil {
		logx.Errorf("微信登录签发用户 token 失败: userId=%s roles=%v err=%+v", profile.ID, roles, err)
		return WechatLoginResp{}, loginDependencyError(err, "登录状态生成失败，请稍后重试")
	}
	profile.Roles = roles
	return WechatLoginResp{
		Token:            token,
		User:             authUserInfoFromProfile(profile),
		ManagedMerchants: managedMerchantInfosFromProfile(profile),
	}, nil
}

func (l *MeLogic) DeleteAccount(ctx context.Context, userID string, req DeleteAccountReq) (DeleteAccountResp, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DeleteAccountResp{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	if strings.TrimSpace(req.Confirmation) != "确认注销" {
		return DeleteAccountResp{}, errx.New(errx.CodeValidationFailed, "请输入“确认注销”后再提交")
	}
	accountStore, ok := l.store.(UserAccountStore)
	if !ok {
		logx.Errorf("注销账号缺少数据处理能力: userId=%s", userID)
		return DeleteAccountResp{}, errx.New(errx.CodeInternalError, "注销服务暂不可用，请稍后重试")
	}
	if err := accountStore.DeleteUserAccount(ctx, userID, strings.TrimSpace(req.Reason)); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrUserDisabled) {
			return DeleteAccountResp{}, errx.New(errx.CodeStateConflict, "账号已注销或当前状态不可操作")
		}
		logx.Errorf("注销账号失败: userId=%s err=%+v", userID, err)
		return DeleteAccountResp{}, errx.New(errx.CodeInternalError, "注销失败，请稍后重试")
	}
	logx.Infof("用户账号已注销并完成个人资料匿名化: userId=%s", userID)
	return DeleteAccountResp{Message: "账号已注销，个人资料已清理"}, nil
}

func (l *WechatLoginLogic) triggerFirstLoginGrowthReward(ctx context.Context, profile model.UserProfile) {
	growthStore, ok := l.store.(GrowthEventStore)
	if !ok {
		return
	}
	for _, merchant := range profile.ManagedMerchants {
		merchantID := strings.TrimSpace(merchant.ID)
		if merchantID == "" {
			continue
		}
		// 新手发布权益以默认商家为归属发放；失败只影响权益到账，不能阻断登录主流程。
		if _, err := growthStore.TriggerGrowthEvent(ctx, model.GrowthEventInput{
			EventType:  model.GrowthEventUserFirstLogin,
			MerchantID: merchantID,
			UserID:     profile.ID,
		}); err != nil {
			logx.Errorf("登录后触发新手成长权益失败: userId=%s merchantId=%s err=%+v", profile.ID, merchantID, err)
		}
		return
	}
}

func (l *WechatLoginLogic) ensureDefaultMerchantProfile(ctx context.Context, profile model.UserProfile, cityCode string) (model.UserProfile, error) {
	if len(profile.ManagedMerchants) > 0 {
		return profile, nil
	}
	defaultStore, ok := l.store.(DefaultMerchantStore)
	if !ok {
		return profile, nil
	}
	merchant, err := defaultStore.EnsureDefaultMerchantForUser(ctx, profile.ID, strings.TrimSpace(cityCode))
	if err != nil {
		logx.Errorf("登录初始化默认商家失败: userId=%s cityCode=%s err=%+v", profile.ID, strings.TrimSpace(cityCode), err)
		return model.UserProfile{}, loginDependencyError(err, "登录初始化失败，请稍后重试")
	}
	if strings.TrimSpace(merchant.ProfileStatus) == "" {
		merchant.ProfileStatus = model.MerchantProfileStatusIncomplete
	}
	profile.ManagedMerchants = []model.ManagedMerchantInfo{merchant}
	logx.Infof("登录初始化默认商家成功: userId=%s merchantId=%s profileStatus=%s", profile.ID, merchant.ID, merchant.ProfileStatus)
	return profile, nil
}

func loginDependencyError(err error, fallbackMessage string) error {
	var appErr *errx.Error
	if errors.As(err, &appErr) {
		return err
	}
	return errx.New(errx.CodeInternalError, fallbackMessage)
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

func (l *MeLogic) BindWechatPhone(ctx context.Context, userID string, req BindWechatPhoneReq, wechatClient WechatSessionClient) (BindPhoneResp, error) {
	userID = strings.TrimSpace(userID)
	code := strings.TrimSpace(req.Code)
	if userID == "" {
		return BindPhoneResp{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	if code == "" {
		return BindPhoneResp{}, errx.New(errx.CodeValidationFailed, "请授权微信手机号")
	}
	if wechatClient == nil {
		return BindPhoneResp{}, errx.New(errx.CodeInternalError, "微信手机号服务未配置，请手动填写")
	}

	phoneInfo, err := wechatClient.GetPhoneNumber(ctx, code)
	if err != nil {
		logx.Errorf("微信手机号授权换取失败: userId=%s err=%+v", userID, err)
		return BindPhoneResp{}, loginDependencyError(err, "手机号获取失败，请手动填写")
	}
	phone := strings.TrimSpace(phoneInfo.PurePhoneNumber)
	if phone == "" {
		phone = strings.TrimSpace(phoneInfo.PhoneNumber)
	}
	if len(phone) < 6 || len(phone) > 20 {
		logx.Errorf("微信手机号授权返回号码异常: userId=%s hasPhone=%t countryCode=%s", userID, strings.TrimSpace(phoneInfo.PhoneNumber) != "" || strings.TrimSpace(phoneInfo.PurePhoneNumber) != "", strings.TrimSpace(phoneInfo.CountryCode))
		return BindPhoneResp{}, errx.New(errx.CodeInternalError, "手机号获取失败，请手动填写")
	}

	profile, err := l.store.BindUserPhone(ctx, userID, phone)
	if err != nil {
		logx.Errorf("微信手机号绑定写入失败: userId=%s err=%+v", userID, err)
		return BindPhoneResp{}, loginDependencyError(err, "手机号绑定失败，请稍后重试")
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
			ID: merchant.ID, Name: merchant.Name, Role: merchant.Role, ProfileStatus: normalizeManagedMerchantProfileStatus(merchant.ProfileStatus),
		})
	}
	return managedMerchants
}

func normalizeManagedMerchantProfileStatus(status string) string {
	switch strings.TrimSpace(status) {
	case model.MerchantProfileStatusIncomplete:
		return model.MerchantProfileStatusIncomplete
	default:
		return model.MerchantProfileStatusCompleted
	}
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
