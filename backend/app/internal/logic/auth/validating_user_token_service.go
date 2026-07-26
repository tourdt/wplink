package auth

import (
	"context"
	"errors"

	"wplink/backend/app/internal/session"
)

var ErrUserSessionDisabled = errors.New("user session disabled")

type UserStatusStore interface {
	IsUserActive(ctx context.Context, userID string) (bool, error)
}

// ValidatingUserTokenService 在校验签名和有效期后再读取账号状态，
// 保证后台停用或用户注销后旧 token 立即失效，而不是继续存活到 JWT 过期。
type ValidatingUserTokenService struct {
	base  TokenService
	store UserStatusStore
}

func NewValidatingUserTokenService(base TokenService, store UserStatusStore) *ValidatingUserTokenService {
	return &ValidatingUserTokenService{base: base, store: store}
}

func (s *ValidatingUserTokenService) IssueUserToken(ctx context.Context, subject session.UserTokenSubject) (string, error) {
	return s.base.IssueUserToken(ctx, subject)
}

func (s *ValidatingUserTokenService) ParseUserToken(ctx context.Context, token string) (session.UserTokenSubject, error) {
	subject, err := s.base.ParseUserToken(ctx, token)
	if err != nil {
		return session.UserTokenSubject{}, err
	}
	if s.store == nil {
		return session.UserTokenSubject{}, ErrUserSessionDisabled
	}
	active, err := s.store.IsUserActive(ctx, subject.UserID)
	if err != nil {
		return session.UserTokenSubject{}, err
	}
	if !active {
		return session.UserTokenSubject{}, ErrUserSessionDisabled
	}
	return subject, nil
}
