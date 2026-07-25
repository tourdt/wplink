package adminauth

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
)

func TestValidatingAdminTokenServiceUsesCurrentDatabasePermissions(t *testing.T) {
	parser := &fakeAdminTokenParser{subject: session.AdminTokenSubject{
		OperatorID:  "operator-1",
		AuthVersion: 4,
		Roles:       []string{permission.RoleSuperAdmin},
		Modules:     permission.AllAdminModules(),
	}}
	store := &fakeAdminSessionStore{credential: AdminCredential{
		OperatorID:  "operator-1",
		AuthVersion: 4,
		Status:      CredentialStatusEnabled,
		Roles:       []string{permission.RolePlatformOperator},
		RoleModules: map[string][]string{
			permission.RolePlatformOperator: {permission.AdminModuleDashboard},
		},
	}}

	subject, err := NewValidatingAdminTokenService(parser, store).ParseAdminToken(context.Background(), "token")
	if err != nil {
		t.Fatalf("ParseAdminToken() error = %v", err)
	}
	if len(subject.Roles) != 1 || subject.Roles[0] != permission.RolePlatformOperator {
		t.Fatalf("roles = %#v, want current database role", subject.Roles)
	}
	if len(subject.Modules) != 1 || subject.Modules[0] != permission.AdminModuleDashboard {
		t.Fatalf("modules = %#v, want current database modules", subject.Modules)
	}
}

func TestValidatingAdminTokenServiceRejectsChangedOrDisabledOperator(t *testing.T) {
	parser := &fakeAdminTokenParser{subject: session.AdminTokenSubject{OperatorID: "operator-1", AuthVersion: 3}}
	store := &fakeAdminSessionStore{credential: AdminCredential{
		OperatorID:  "operator-1",
		AuthVersion: 4,
		Status:      CredentialStatusEnabled,
		Roles:       []string{permission.RolePlatformOperator},
	}}
	service := NewValidatingAdminTokenService(parser, store)
	if _, err := service.ParseAdminToken(context.Background(), "token"); !errors.Is(err, ErrAdminSessionInvalid) {
		t.Fatalf("ParseAdminToken(changed version) error = %v, want ErrAdminSessionInvalid", err)
	}

	store.credential.AuthVersion = 3
	store.credential.Status = CredentialStatusDisabled
	if _, err := service.ParseAdminToken(context.Background(), "token"); !errors.Is(err, ErrAdminSessionInvalid) {
		t.Fatalf("ParseAdminToken(disabled) error = %v, want ErrAdminSessionInvalid", err)
	}
}

type fakeAdminTokenParser struct {
	subject session.AdminTokenSubject
	err     error
}

func (p *fakeAdminTokenParser) ParseAdminToken(_ context.Context, _ string) (session.AdminTokenSubject, error) {
	return p.subject, p.err
}

type fakeAdminSessionStore struct {
	credential AdminCredential
	err        error
}

func (s *fakeAdminSessionStore) FindCurrentAdminSession(_ context.Context, _ string) (AdminCredential, error) {
	return s.credential, s.err
}
