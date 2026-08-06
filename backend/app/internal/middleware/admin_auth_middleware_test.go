package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"wplink/backend/app/internal/handler/handlerx"
	adminauthlogic "wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"
)

type fakeAdminTokenService struct {
	subject session.AdminTokenSubject
	err     error
}

func (f fakeAdminTokenService) ParseAdminToken(context.Context, string) (session.AdminTokenSubject, error) {
	return f.subject, f.err
}

func TestAdminAuthMiddlewareRejectsMissingBearerToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	rec := httptest.NewRecorder()
	called := false

	NewAdminAuthMiddleware(fakeAdminTokenService{}).Handle(func(http.ResponseWriter, *http.Request) {
		called = true
	})(rec, r)

	assertMiddlewareError(t, rec, http.StatusUnauthorized, errx.CodeUnauthorized)
	if called {
		t.Fatal("next handler called without bearer token")
	}
}

func TestAdminAuthMiddlewareWritesValidatedSubjectToContext(t *testing.T) {
	want := session.AdminTokenSubject{
		OperatorID:  "operator-1",
		AuthVersion: 3,
		Roles:       []string{permission.RolePlatformOperator},
		Modules:     []string{permission.AdminModuleDashboard},
	}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	r.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	NewAdminAuthMiddleware(fakeAdminTokenService{subject: want}).Handle(func(w http.ResponseWriter, r *http.Request) {
		got, err := handlerx.AdminFromContext(r.Context())
		if err != nil || got.OperatorID != want.OperatorID || got.AuthVersion != want.AuthVersion {
			t.Fatalf("AdminFromContext() = (%#v, %v), want %#v", got, err, want)
		}
		w.WriteHeader(http.StatusNoContent)
	})(rec, r)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s, want %d", rec.Code, rec.Body.String(), http.StatusNoContent)
	}
}

func TestAdminAuthMiddlewareRejectsExpiredToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	r.Header.Set("Authorization", "Bearer expired-token")
	rec := httptest.NewRecorder()

	NewAdminAuthMiddleware(fakeAdminTokenService{err: errors.New("raw validation detail")}).Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called for expired token")
	})(rec, r)

	assertMiddlewareError(t, rec, http.StatusUnauthorized, errx.CodeUnauthorized)
}

func TestAdminAuthMiddlewareRejectsRoleWithoutAdminAccess(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	r.Header.Set("Authorization", "Bearer merchant-admin-token")
	rec := httptest.NewRecorder()

	NewAdminAuthMiddleware(fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "operator-1",
		Roles:      []string{permission.RoleMerchantAdmin},
	}}).Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called for role without platform access")
	})(rec, r)

	assertMiddlewareError(t, rec, http.StatusForbidden, errx.CodeForbidden)
}

func TestAdminAuthMiddlewareRejectsUnconfiguredAndUnknownModules(t *testing.T) {
	service := fakeAdminTokenService{subject: session.AdminTokenSubject{
		OperatorID: "operator-1",
		Roles:      []string{permission.RolePlatformOperator},
		Modules:    []string{permission.AdminModuleResourceReview},
	}}
	for _, path := range []string{
		"/api/v1/admin/merchants",
		"/api/v1/admin/new-module",
	} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, path, nil)
			r.Header.Set("Authorization", "Bearer admin-token")
			rec := httptest.NewRecorder()
			NewAdminAuthMiddleware(service).Handle(func(http.ResponseWriter, *http.Request) {
				t.Fatal("next handler called for denied module")
			})(rec, r)
			assertMiddlewareError(t, rec, http.StatusForbidden, errx.CodeForbidden)
		})
	}
}

func TestAdminAuthMiddlewareFailsSafelyWhenDependencyIsMissing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	r.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	NewAdminAuthMiddleware(nil).Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called without token service")
	})(rec, r)

	assertMiddlewareError(t, rec, http.StatusInternalServerError, errx.CodeInternalError)
}

func TestAdminAuthMiddlewareFailsSafelyWhenDependencyIsTypedNil(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/overview", nil)
	r.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	var service *adminauthlogic.ValidatingAdminTokenService

	NewAdminAuthMiddleware(service).Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called with typed-nil token service")
	})(rec, r)

	assertMiddlewareError(t, rec, http.StatusInternalServerError, errx.CodeInternalError)
}

func assertMiddlewareError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d body = %s, want %d", rec.Code, rec.Body.String(), status)
	}
	var body struct {
		ErrorCode string `json:"errorCode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ErrorCode != code {
		t.Fatalf("error code = %q, want %q", body.ErrorCode, code)
	}
}
