package adminauth

import (
	"context"
	"errors"

	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
)

var ErrAdminSessionInvalid = errors.New("后台登录状态已失效")

type AdminTokenParser interface {
	ParseAdminToken(ctx context.Context, token string) (session.AdminTokenSubject, error)
}

type AdminSessionStore interface {
	FindCurrentAdminSession(ctx context.Context, operatorID string) (AdminCredential, error)
}

// ValidatingAdminTokenService 每次请求都以数据库中的账号状态、角色和模块权限为准。
// 管理员被停用、密码或角色被修改、角色模块权限被调整后，旧 token 会立即失效。
type ValidatingAdminTokenService struct {
	parser AdminTokenParser
	store  AdminSessionStore
}

func NewValidatingAdminTokenService(parser AdminTokenParser, store AdminSessionStore) *ValidatingAdminTokenService {
	return &ValidatingAdminTokenService{parser: parser, store: store}
}

func (s *ValidatingAdminTokenService) ParseAdminToken(ctx context.Context, token string) (session.AdminTokenSubject, error) {
	if s == nil || s.parser == nil || s.store == nil {
		return session.AdminTokenSubject{}, ErrAdminSessionInvalid
	}
	subject, err := s.parser.ParseAdminToken(ctx, token)
	if err != nil {
		return session.AdminTokenSubject{}, err
	}
	credential, err := s.store.FindCurrentAdminSession(ctx, subject.OperatorID)
	if err != nil {
		return session.AdminTokenSubject{}, ErrAdminSessionInvalid
	}
	if credential.Status != CredentialStatusEnabled ||
		credential.AuthVersion <= 0 ||
		credential.AuthVersion != subject.AuthVersion ||
		!permission.CanAccessAdmin(credential.Roles) {
		return session.AdminTokenSubject{}, ErrAdminSessionInvalid
	}
	return session.AdminTokenSubject{
		OperatorID:  credential.OperatorID,
		AuthVersion: credential.AuthVersion,
		Roles:       append([]string(nil), credential.Roles...),
		Modules:     permission.ResolveAdminModules(credential.Roles, credential.RoleModules),
	}, nil
}
