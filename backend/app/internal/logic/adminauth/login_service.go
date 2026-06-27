package adminauth

import (
	"context"
	"errors"
	"strings"
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

type LoginRequest struct {
	LoginName string
	Password  string
}

type LoginResponse struct {
	Token  string
	UserID string
	Roles  []string
}

type AdminCredential struct {
	UserID       string
	LoginName    string
	PasswordHash string
	Status       string
	Roles        []string
}

type AdminStore interface {
	FindCredentialByLoginName(ctx context.Context, loginName string) (AdminCredential, error)
}

type PasswordVerifier interface {
	Verify(hash string, password string) bool
}

type TokenIssuer interface {
	IssueAdminToken(ctx context.Context, credential AdminCredential) (string, error)
}

type LoginService struct {
	store    AdminStore
	verifier PasswordVerifier
	issuer   TokenIssuer
}

func NewLoginService(store AdminStore, verifier PasswordVerifier, issuer TokenIssuer) *LoginService {
	return &LoginService{
		store:    store,
		verifier: verifier,
		issuer:   issuer,
	}
}

func (s *LoginService) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	loginName := strings.TrimSpace(req.LoginName)
	password := strings.TrimSpace(req.Password)
	if loginName == "" || password == "" {
		return LoginResponse{}, ErrInvalidCredential
	}

	credential, err := s.store.FindCredentialByLoginName(ctx, loginName)
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
	if !s.verifier.Verify(credential.PasswordHash, password) {
		return LoginResponse{}, ErrInvalidCredential
	}

	// 后台 token 独立签发，避免小程序登录态被误用于管理后台。
	token, err := s.issuer.IssueAdminToken(ctx, credential)
	if err != nil {
		return LoginResponse{}, ErrTokenIssueFailed
	}

	return LoginResponse{
		Token:  token,
		UserID: credential.UserID,
		Roles:  append([]string(nil), credential.Roles...),
	}, nil
}

func hasAdminRole(roles []string) bool {
	for _, role := range roles {
		if role == RolePlatformOperator || role == RoleSuperAdmin {
			return true
		}
	}
	return false
}
