package handlerx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"
)

type fakeUserTokenService struct {
	subject session.UserTokenSubject
	err     error
	token   string
}

func (f fakeUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeUserTokenService) ParseUserToken(_ context.Context, token string) (session.UserTokenSubject, error) {
	if f.token != "" && token != f.token {
		return session.UserTokenSubject{}, errors.New("unexpected token")
	}
	return f.subject, f.err
}

var _ authlogic.TokenService = fakeUserTokenService{}

type fakeAdminTokenService struct {
	subject session.AdminTokenSubject
	err     error
	token   string
}

func (f fakeAdminTokenService) ParseAdminToken(_ context.Context, token string) (session.AdminTokenSubject, error) {
	if f.token != "" && token != f.token {
		return session.AdminTokenSubject{}, errors.New("unexpected token")
	}
	return f.subject, f.err
}

func TestRequiredUserRejectsMissingAndNonBearerAuthorization(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "空 Header"},
		{name: "非 Bearer", header: "Basic dXNlcjpwYXNz"},
		{name: "空 Bearer", header: "Bearer   "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			r.Header.Set("Authorization", tc.header)
			_, err := RequiredUser(r, fakeUserTokenService{})
			if errx.CodeOf(err) != errx.CodeUnauthorized {
				t.Fatalf("error code = %q, want %q", errx.CodeOf(err), errx.CodeUnauthorized)
			}
		})
	}
}

func TestRequiredUserReturnsValidatedSubject(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	r.Header.Set("Authorization", "Bearer user-token")
	want := session.UserTokenSubject{UserID: "user-1", Roles: []string{"normal_user"}}

	got, err := RequiredUser(r, fakeUserTokenService{subject: want, token: "user-token"})
	if err != nil {
		t.Fatalf("RequiredUser() error = %v", err)
	}
	if got.UserID != want.UserID || len(got.Roles) != 1 || got.Roles[0] != want.Roles[0] {
		t.Fatalf("subject = %#v, want %#v", got, want)
	}
}

func TestRequiredUserMapsExpiredTokenToSafeUnauthorizedError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	r.Header.Set("Authorization", "Bearer expired-token")

	_, err := RequiredUser(r, fakeUserTokenService{err: errors.New("raw token detail")})
	if errx.CodeOf(err) != errx.CodeUnauthorized || errx.PublicMessage(err) != "登录已过期，请重新登录" {
		t.Fatalf("error = (%q, %q), want safe expired-session error", errx.CodeOf(err), errx.PublicMessage(err))
	}
}

func TestRequiredUserFailsSafelyWhenTokenServiceIsMissing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	r.Header.Set("Authorization", "Bearer user-token")

	_, err := RequiredUser(r, nil)
	if errx.CodeOf(err) != errx.CodeInternalError {
		t.Fatalf("error code = %q, want %q", errx.CodeOf(err), errx.CodeInternalError)
	}
}

func TestOptionalUserAllowsAnonymousRequestWithoutHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)

	subject, ok, err := OptionalUser(r, fakeUserTokenService{})
	if err != nil || ok || subject.UserID != "" {
		t.Fatalf("OptionalUser() = (%#v, %t, %v), want anonymous request", subject, ok, err)
	}
}

func TestOptionalUserRejectsBadTokenInsteadOfTreatingItAsAnonymous(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	r.Header.Set("Authorization", "Bearer bad-token")

	_, ok, err := OptionalUser(r, fakeUserTokenService{err: errors.New("signature mismatch")})
	if ok || errx.CodeOf(err) != errx.CodeUnauthorized {
		t.Fatalf("OptionalUser() = (ok %t, code %q), want rejected token", ok, errx.CodeOf(err))
	}
}

func TestOptionalUserFailsSafelyWhenTokenServiceIsMissing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)

	_, ok, err := OptionalUser(r, nil)
	if ok || errx.CodeOf(err) != errx.CodeInternalError {
		t.Fatalf("OptionalUser() = (ok %t, code %q), want internal error", ok, errx.CodeOf(err))
	}
}

func TestOptionalAdminReturnsValidatedAdminSubject(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	r.Header.Set("Authorization", "Bearer admin-token")
	want := session.AdminTokenSubject{OperatorID: "operator-1", AuthVersion: 2, Roles: []string{"super_admin"}}

	got, ok := OptionalAdmin(r, fakeAdminTokenService{subject: want, token: "admin-token"})
	if !ok || got.OperatorID != want.OperatorID || got.AuthVersion != want.AuthVersion {
		t.Fatalf("OptionalAdmin() = (%#v, %t), want %#v", got, ok, want)
	}
}

func TestOptionalAdminRejectsMissingDependencyAndBadToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	r.Header.Set("Authorization", "Bearer bad-token")

	if _, ok := OptionalAdmin(r, nil); ok {
		t.Fatal("OptionalAdmin() ok = true with missing token service")
	}
	if _, ok := OptionalAdmin(r, fakeAdminTokenService{err: errors.New("invalid")}); ok {
		t.Fatal("OptionalAdmin() ok = true for invalid token")
	}
}

func TestAdminFromContextReturnsTypedSubject(t *testing.T) {
	want := session.AdminTokenSubject{OperatorID: "operator-1", Roles: []string{"super_admin"}}
	got, err := AdminFromContext(ContextWithAdmin(context.Background(), want))
	if err != nil || got.OperatorID != want.OperatorID {
		t.Fatalf("AdminFromContext() = (%#v, %v), want %#v", got, err, want)
	}
}

func TestAdminFromContextRejectsMissingSubject(t *testing.T) {
	_, err := AdminFromContext(context.Background())
	if errx.CodeOf(err) != errx.CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", errx.CodeOf(err), errx.CodeUnauthorized)
	}
}
