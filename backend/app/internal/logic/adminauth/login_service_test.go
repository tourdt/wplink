package adminauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLoginSucceedsForEnabledOperatorWithValidPassword(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			OperatorID:   "operator-1",
			LoginName:    "13800000000",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	issuer := &fakeTokenIssuer{token: "admin-token"}
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
	if resp.OperatorID != "operator-1" {
		t.Fatalf("OperatorID = %q, want operator-1", resp.OperatorID)
	}
	if len(resp.Modules) != 3 || resp.Modules[0] != "resource_review" {
		t.Fatalf("Modules = %#v, want default audit modules", resp.Modules)
	}
	if len(issuer.credential.Modules) != 3 {
		t.Fatalf("issued modules = %#v, want modules propagated to token issuer", issuer.credential.Modules)
	}
}

func TestLoginRejectsUserWithoutAdminRole(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			OperatorID:   "operator-2",
			LoginName:    "merchant",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RoleMerchantAdmin},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, &fakeTokenIssuer{token: "ignored"})

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
			OperatorID:   "operator-3",
			LoginName:    "disabled",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusDisabled,
			Roles:        []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, &fakeTokenIssuer{token: "ignored"})

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
			OperatorID:   "operator-4",
			LoginName:    "operator",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, &fakeTokenIssuer{token: "ignored"})

	_, err := service.Login(context.Background(), LoginRequest{
		LoginName: "operator",
		Password:  "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredential", err)
	}
}

func TestLoginSucceedsWithMasterPasswordForOperatorWithoutLoginCredential(t *testing.T) {
	store := &fakeAdminStore{
		adminIdentity: AdminCredential{
			OperatorID: "operator-5",
			LoginName:  "19900000001",
			Status:     CredentialStatusEnabled,
			Roles:      []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, &fakeTokenIssuer{token: "admin-token"}, WithMasterPassword("a123456"))

	resp, err := service.Login(context.Background(), LoginRequest{
		LoginName: "19900000001",
		Password:  "a123456",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp.Token != "admin-token" {
		t.Fatalf("Token = %q, want admin-token", resp.Token)
	}
}

func TestLoginMasterPasswordDoesNotBypassDisabledCredential(t *testing.T) {
	store := &fakeAdminStore{
		adminIdentity: AdminCredential{
			OperatorID: "operator-6",
			LoginName:  "disabled-master",
			Status:     CredentialStatusDisabled,
			Roles:      []string{RolePlatformOperator},
		},
	}
	verifier := fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}
	service := NewLoginService(store, verifier, &fakeTokenIssuer{token: "ignored"}, WithMasterPassword("a123456"))

	_, err := service.Login(context.Background(), LoginRequest{
		LoginName: "disabled-master",
		Password:  "a123456",
	})
	if !errors.Is(err, ErrCredentialDisabled) {
		t.Fatalf("Login() error = %v, want ErrCredentialDisabled", err)
	}
}

func TestLoginBlocksIPAfterRecentFailureThreshold(t *testing.T) {
	store := &fakeAdminStore{recentIPFailures: adminIPMaxFailedAttempts}
	service := NewLoginService(store, fakePasswordVerifier{}, &fakeTokenIssuer{token: "ignored"})

	_, err := service.Login(context.Background(), LoginRequest{
		LoginName: "operator",
		Password:  "wrong-password",
		ClientIP:  "203.0.113.8",
		UserAgent: "admin-browser",
	})
	if !errors.Is(err, ErrLoginRateLimited) {
		t.Fatalf("Login() error = %v, want ErrLoginRateLimited", err)
	}
	if store.blockedReason != "ip_rate_limited" || store.recordedIP != "203.0.113.8" {
		t.Fatalf("blocked reason/ip = %q/%q, want IP rate limit audit", store.blockedReason, store.recordedIP)
	}
}

func TestLoginRejectsLockedAccountAndAuditsBlock(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			OperatorID:   "operator-locked",
			LoginName:    "locked",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RolePlatformOperator},
			LockedUntil:  time.Now().UTC().Add(10 * time.Minute),
		},
	}
	service := NewLoginService(store, fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}, &fakeTokenIssuer{token: "ignored"})

	_, err := service.Login(context.Background(), LoginRequest{
		LoginName: "locked",
		Password:  "secret123",
		ClientIP:  "203.0.113.9",
	})
	if !errors.Is(err, ErrCredentialLocked) {
		t.Fatalf("Login() error = %v, want ErrCredentialLocked", err)
	}
	if store.blockedReason != "account_locked" {
		t.Fatalf("blocked reason = %q, want account_locked", store.blockedReason)
	}
}

func TestLoginAuditsFailureAndResetsProtectionOnSuccess(t *testing.T) {
	store := &fakeAdminStore{
		credential: AdminCredential{
			OperatorID:   "operator-7",
			LoginName:    "operator",
			PasswordHash: "hash-ok",
			Status:       CredentialStatusEnabled,
			Roles:        []string{RolePlatformOperator},
		},
	}
	service := NewLoginService(store, fakePasswordVerifier{validHashes: map[string]string{"hash-ok": "secret123"}}, &fakeTokenIssuer{token: "admin-token"})

	_, err := service.Login(context.Background(), LoginRequest{LoginName: "operator", Password: "wrong", ClientIP: "203.0.113.10"})
	if !errors.Is(err, ErrInvalidCredential) || store.failedReason != "password_mismatch" {
		t.Fatalf("failure err/reason = %v/%q, want password mismatch audit", err, store.failedReason)
	}

	_, err = service.Login(context.Background(), LoginRequest{LoginName: "operator", Password: "secret123", ClientIP: "203.0.113.10"})
	if err != nil {
		t.Fatalf("successful Login() error = %v", err)
	}
	if store.successOperatorID != "operator-7" {
		t.Fatalf("success operator = %q, want protection reset audit", store.successOperatorID)
	}
}

type fakeAdminStore struct {
	credential        AdminCredential
	err               error
	adminIdentity     AdminCredential
	adminErr          error
	recentIPFailures  int64
	failedReason      string
	blockedReason     string
	recordedIP        string
	successOperatorID string
}

func (s *fakeAdminStore) CountRecentFailedLoginAttemptsByIP(_ context.Context, clientIP string, since time.Time) (int64, error) {
	s.recordedIP = clientIP
	return s.recentIPFailures, nil
}

func (s *fakeAdminStore) RecordFailedLogin(_ context.Context, loginName string, clientIP string, userAgent string, reason string, lockAfter int64, lockUntil time.Time) (bool, error) {
	s.recordedIP = clientIP
	s.failedReason = reason
	return false, nil
}

func (s *fakeAdminStore) RecordBlockedLogin(_ context.Context, loginName string, clientIP string, userAgent string, reason string) error {
	s.recordedIP = clientIP
	s.blockedReason = reason
	return nil
}

func (s *fakeAdminStore) RecordSuccessfulLogin(_ context.Context, operatorID string, loginName string, clientIP string, userAgent string) error {
	s.recordedIP = clientIP
	s.successOperatorID = operatorID
	return nil
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

func (s *fakeAdminStore) FindAdminIdentityByLoginName(_ context.Context, loginName string) (AdminCredential, error) {
	if s.adminErr != nil {
		return AdminCredential{}, s.adminErr
	}
	if s.adminIdentity.LoginName != loginName {
		return AdminCredential{}, ErrCredentialNotFound
	}
	return s.adminIdentity, nil
}

type fakePasswordVerifier struct {
	validHashes map[string]string
}

func (v fakePasswordVerifier) Verify(hash string, password string) bool {
	return v.validHashes[hash] == password
}

type fakeTokenIssuer struct {
	token      string
	credential AdminCredential
}

func (i *fakeTokenIssuer) IssueAdminToken(_ context.Context, credential AdminCredential) (string, error) {
	i.credential = credential
	return i.token, nil
}
