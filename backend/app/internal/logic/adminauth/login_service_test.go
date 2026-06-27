package adminauth

import (
	"context"
	"errors"
	"testing"
)

func TestLoginSucceedsForEnabledOperatorWithValidPassword(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			UserID:       "user-1",
			LoginName:    "13800000000",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	issuer := fakeTokenIssuer{token: "admin-token"}
	service := NewLoginService(store, verifier, issuer)

	resp, err := service.Login(context.Background(), LoginRequest{
		LoginName: "13800000000",
		Password:  "secret123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp.Token != "admin-token" {
		t.Fatalf("Token = %q, want admin-token", resp.Token)
	}
	if resp.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", resp.UserID)
	}
}

func TestLoginRejectsUserWithoutAdminRole(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			UserID:       "user-2",
			LoginName:    "merchant",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RoleMerchantAdmin},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, fakeTokenIssuer{token: "ignored"})

	_, err := service.Login(context.Background(), LoginRequest{
		LoginName: "merchant",
		Password:  "secret123",
	})
	if !errors.Is(err, ErrAdminPermissionRequired) {
		t.Fatalf("Login() error = %v, want ErrAdminPermissionRequired", err)
	}
}

func TestLoginRejectsDisabledCredential(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			UserID:       "user-3",
			LoginName:    "disabled",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusDisabled,
			Roles:        []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, fakeTokenIssuer{token: "ignored"})

	_, err := service.Login(context.Background(), LoginRequest{
		LoginName: "disabled",
		Password:  "secret123",
	})
	if !errors.Is(err, ErrCredentialDisabled) {
		t.Fatalf("Login() error = %v, want ErrCredentialDisabled", err)
	}
}

func TestLoginRejectsInvalidPassword(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			UserID:       "user-4",
			LoginName:    "operator",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, fakeTokenIssuer{token: "ignored"})

	_, err := service.Login(context.Background(), LoginRequest{
		LoginName: "operator",
		Password:  "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredential", err)
	}
}

type fakeAdminStore struct {
	credential AdminCredential
	err        error
}

func (s *fakeAdminStore) FindCredentialByLoginName(_ context.Context, loginName string) (AdminCredential, error) {
	if s.err != nil {
		return AdminCredential{}, s.err
	}
	if s.credential.LoginName != loginName {
		return AdminCredential{}, ErrCredentialNotFound
	}
	return s.credential, nil
}

type fakePasswordVerifier struct {
	validHashes map[string]string
}

func (v fakePasswordVerifier) Verify(hash string, password string) bool {
	return v.validHashes[hash] == password
}

type fakeTokenIssuer struct {
	token string
}

func (i fakeTokenIssuer) IssueAdminToken(_ context.Context, credential AdminCredential) (string, error) {
	return i.token, nil
}
