package adminauth

import (
	"context"
	"errors"
	"strings"

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
	ErrAdminPermissionRequired = errors.New("您没有权限登录管理后台")
	ErrTokenIssueFailed        = errors.New("登录失败，请稍后重试")
)

func PublicLoginErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidCredential):
		return ErrInvalidCredential.Error()
	case errors.Is(err, ErrCredentialDisabled):
		return ErrCredentialDisabled.Error()
	case errors.Is(err, ErrAdminPermissionRequired):
		return ErrAdminPermissionRequired.Error()
	case errors.Is(err, ErrTokenIssueFailed):
		return ErrTokenIssueFailed.Error()
	default:
		return ErrTokenIssueFailed.Error()
	}
}

type LoginRequest struct {
	LoginName string
	Password  string
}

type LoginResponse struct {
	Token      string   `json:"token"`
	OperatorID string   `json:"operatorId"`
	Roles      []string `json:"roles"`
	Modules    []string `json:"modules"`
}

type AdminCredential struct {
	OperatorID   string
	LoginName    string
	PasswordHash string
	Status       string
	Roles        []string
	RoleModules  map[string][]string
	Modules      []string
}

type AdminStore interface {
	FindCredentialByLoginName(ctx context.Context, loginName string) (AdminCredential, error)
	FindAdminIdentityByLoginName(ctx context.Context, loginName string) (AdminCredential, error)
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
	if loginName == "" || password == "" {
		return LoginResponse{}, ErrInvalidCredential
	}

	masterPasswordMatched := s.masterPassword != "" && password == s.masterPassword
	credential, err := s.findLoginCredential(ctx, loginName, masterPasswordMatched)
	if err != nil {
		if errors.Is(err, ErrCredentialNotFound) {
			return LoginResponse{}, ErrInvalidCredential
		}
		return LoginResponse{}, err
	}

	if credential.Status != CredentialStatusEnabled {
		return LoginResponse{}, ErrCredentialDisabled
	}
	if !hasAdminRole(credential.Roles) {
		return LoginResponse{}, ErrAdminPermissionRequired
	}
	if !masterPasswordMatched && !s.verifier.Verify(credential.PasswordHash, password) {
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

	return LoginResponse{
		Token:      token,
		OperatorID: credential.OperatorID,
		Roles:      append([]string(nil), credential.Roles...),
		Modules:    append([]string(nil), credential.Modules...),
	}, nil
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
