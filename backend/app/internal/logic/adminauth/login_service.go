package adminauth

import (
	"context"
	"errors"
	"strings"
	"time"

	"wplink/backend/app/internal/permission"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	RolePlatformOperator = "platform_operator"
	RoleSuperAdmin       = "super_admin"
	RoleMerchantAdmin    = "merchant_admin"
)

const (
	CredentialStatusEnabled  = "enabled"
	CredentialStatusDisabled = "disabled"
)

var (
	ErrInvalidCredential       = errors.New("账号或密码错误")
	ErrCredentialNotFound      = errors.New("后台账号不存在")
	ErrCredentialDisabled      = errors.New("后台账号已停用，请联系管理员")
	ErrCredentialLocked        = errors.New("登录尝试过多，请稍后重试")
	ErrLoginRateLimited        = errors.New("当前网络登录尝试过多，请稍后重试")
	ErrAdminPermissionRequired = errors.New("您没有权限登录管理后台")
	ErrTokenIssueFailed        = errors.New("登录失败，请稍后重试")
)

func PublicLoginErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidCredential):
		return ErrInvalidCredential.Error()
	case errors.Is(err, ErrCredentialDisabled):
		return ErrCredentialDisabled.Error()
	case errors.Is(err, ErrCredentialLocked), errors.Is(err, ErrLoginRateLimited):
		return err.Error()
	case errors.Is(err, ErrAdminPermissionRequired):
		return ErrAdminPermissionRequired.Error()
	case errors.Is(err, ErrTokenIssueFailed):
		return ErrTokenIssueFailed.Error()
	default:
		return ErrTokenIssueFailed.Error()
	}
}

func IsLoginRateLimited(err error) bool {
	return errors.Is(err, ErrCredentialLocked) || errors.Is(err, ErrLoginRateLimited)
}

type LoginRequest struct {
	LoginName string
	Password  string
	ClientIP  string
	UserAgent string
}

type LoginResponse struct {
	Token      string   `json:"token"`
	OperatorID string   `json:"operatorId"`
	Roles      []string `json:"roles"`
	Modules    []string `json:"modules"`
}

type AdminCredential struct {
	OperatorID     string
	AuthVersion    int64
	LoginName      string
	PasswordHash   string
	Status         string
	Roles          []string
	RoleModules    map[string][]string
	Modules        []string
	FailedAttempts int64
	LockedUntil    time.Time
}

type AdminStore interface {
	FindCredentialByLoginName(ctx context.Context, loginName string) (AdminCredential, error)
	FindAdminIdentityByLoginName(ctx context.Context, loginName string) (AdminCredential, error)
}

type LoginProtectionStore interface {
	CountRecentFailedLoginAttemptsByIP(ctx context.Context, clientIP string, since time.Time) (int64, error)
	RecordFailedLogin(ctx context.Context, loginName string, clientIP string, userAgent string, reason string, lockAfter int64, lockUntil time.Time) (bool, error)
	RecordBlockedLogin(ctx context.Context, loginName string, clientIP string, userAgent string, reason string) error
	RecordSuccessfulLogin(ctx context.Context, operatorID string, loginName string, clientIP string, userAgent string) error
}

type PasswordVerifier interface {
	Verify(hash string, password string) bool
}

type TokenIssuer interface {
	IssueAdminToken(ctx context.Context, credential AdminCredential) (string, error)
}

type LoginService struct {
	store          AdminStore
	verifier       PasswordVerifier
	issuer         TokenIssuer
	masterPassword string
}

const (
	adminAccountMaxFailedAttempts = int64(5)
	adminAccountLockDuration      = 15 * time.Minute
	adminIPMaxFailedAttempts      = int64(20)
	adminIPRateLimitWindow        = 15 * time.Minute
)

type LoginServiceOption func(*LoginService)

func WithMasterPassword(password string) LoginServiceOption {
	return func(s *LoginService) {
		s.masterPassword = strings.TrimSpace(password)
	}
}

func NewLoginService(store AdminStore, verifier PasswordVerifier, issuer TokenIssuer, opts ...LoginServiceOption) *LoginService {
	service := &LoginService{
		store:    store,
		verifier: verifier,
		issuer:   issuer,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *LoginService) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	loginName := strings.TrimSpace(req.LoginName)
	password := strings.TrimSpace(req.Password)
	clientIP := strings.TrimSpace(req.ClientIP)
	if clientIP == "" {
		clientIP = "unknown"
	}
	userAgent := strings.TrimSpace(req.UserAgent)
	protectionStore, _ := s.store.(LoginProtectionStore)
	if protectionStore != nil {
		failedCount, err := protectionStore.CountRecentFailedLoginAttemptsByIP(ctx, clientIP, time.Now().UTC().Add(-adminIPRateLimitWindow))
		if err != nil {
			logx.Errorf("查询后台登录 IP 失败次数失败: clientIp=%s loginName=%s err=%+v", clientIP, loginName, err)
			return LoginResponse{}, ErrTokenIssueFailed
		}
		if failedCount >= adminIPMaxFailedAttempts {
			_ = protectionStore.RecordBlockedLogin(ctx, loginName, clientIP, userAgent, "ip_rate_limited")
			logx.Errorf("后台登录触发 IP 限流告警: clientIp=%s loginName=%s failedCount=%d window=%s", clientIP, loginName, failedCount, adminIPRateLimitWindow)
			return LoginResponse{}, ErrLoginRateLimited
		}
	}
	if loginName == "" || password == "" {
		s.recordFailedLogin(ctx, protectionStore, loginName, clientIP, userAgent, "missing_credential")
		return LoginResponse{}, ErrInvalidCredential
	}

	masterPasswordMatched := s.masterPassword != "" && password == s.masterPassword
	credential, err := s.findLoginCredential(ctx, loginName, masterPasswordMatched)
	if err != nil {
		if errors.Is(err, ErrCredentialNotFound) {
			s.recordFailedLogin(ctx, protectionStore, loginName, clientIP, userAgent, "credential_not_found")
			return LoginResponse{}, ErrInvalidCredential
		}
		return LoginResponse{}, err
	}

	if !credential.LockedUntil.IsZero() && credential.LockedUntil.After(time.Now().UTC()) {
		if protectionStore != nil {
			_ = protectionStore.RecordBlockedLogin(ctx, loginName, clientIP, userAgent, "account_locked")
		}
		logx.Errorf("后台账号仍在锁定期: operatorId=%s loginName=%s clientIp=%s lockedUntil=%s", credential.OperatorID, loginName, clientIP, credential.LockedUntil.Format(time.RFC3339))
		return LoginResponse{}, ErrCredentialLocked
	}
	if credential.Status != CredentialStatusEnabled {
		if protectionStore != nil {
			_ = protectionStore.RecordBlockedLogin(ctx, loginName, clientIP, userAgent, "credential_disabled")
		}
		return LoginResponse{}, ErrCredentialDisabled
	}
	if !hasAdminRole(credential.Roles) {
		s.recordFailedLogin(ctx, protectionStore, loginName, clientIP, userAgent, "admin_permission_required")
		return LoginResponse{}, ErrAdminPermissionRequired
	}
	if !masterPasswordMatched && !s.verifier.Verify(credential.PasswordHash, password) {
		s.recordFailedLogin(ctx, protectionStore, loginName, clientIP, userAgent, "password_mismatch")
		return LoginResponse{}, ErrInvalidCredential
	}
	credential.Modules = permission.ResolveAdminModules(credential.Roles, credential.RoleModules)
	if masterPasswordMatched {
		// 万能密码只作为本地开发兜底入口，日志记录账号维度，避免泄露密码内容。
		logx.Infof("后台万能密码登录成功: loginName=%s operatorId=%s roles=%v modules=%v", loginName, credential.OperatorID, credential.Roles, credential.Modules)
	}

	// 后台 token 独立签发，避免小程序登录态被误用于管理后台。
	token, err := s.issuer.IssueAdminToken(ctx, credential)
	if err != nil {
		return LoginResponse{}, ErrTokenIssueFailed
	}
	if protectionStore != nil {
		if err := protectionStore.RecordSuccessfulLogin(ctx, credential.OperatorID, loginName, clientIP, userAgent); err != nil {
			logx.Errorf("记录后台登录成功审计失败: operatorId=%s loginName=%s clientIp=%s err=%+v", credential.OperatorID, loginName, clientIP, err)
			return LoginResponse{}, ErrTokenIssueFailed
		}
	}

	return LoginResponse{
		Token:      token,
		OperatorID: credential.OperatorID,
		Roles:      append([]string(nil), credential.Roles...),
		Modules:    append([]string(nil), credential.Modules...),
	}, nil
}

func (s *LoginService) recordFailedLogin(ctx context.Context, store LoginProtectionStore, loginName string, clientIP string, userAgent string, reason string) {
	if store == nil {
		return
	}
	locked, err := store.RecordFailedLogin(
		ctx,
		loginName,
		clientIP,
		userAgent,
		reason,
		adminAccountMaxFailedAttempts,
		time.Now().UTC().Add(adminAccountLockDuration),
	)
	if err != nil {
		logx.Errorf("记录后台登录失败审计失败: loginName=%s clientIp=%s reason=%s err=%+v", loginName, clientIP, reason, err)
		return
	}
	if locked {
		logx.Errorf("后台账号连续登录失败已锁定告警: loginName=%s clientIp=%s lockDuration=%s reason=%s", loginName, clientIP, adminAccountLockDuration, reason)
	}
}

func (s *LoginService) findLoginCredential(ctx context.Context, loginName string, masterPasswordMatched bool) (AdminCredential, error) {
	if masterPasswordMatched {
		// 万能密码用于本地演示账号调试，演示数据只写 users 和角色，不一定写后台登录凭据。
		return s.store.FindAdminIdentityByLoginName(ctx, loginName)
	}
	return s.store.FindCredentialByLoginName(ctx, loginName)
}

func hasAdminRole(roles []string) bool {
	for _, role := range roles {
		if role == RolePlatformOperator || role == RoleSuperAdmin {
			return true
		}
	}
	return false
}
