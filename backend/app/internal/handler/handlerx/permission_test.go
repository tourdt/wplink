package handlerx

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type fakeMerchantPermissionStore struct {
	allowed bool
	err     error
	userID  string
	shopID  string
}

func (f *fakeMerchantPermissionStore) UserCanManageMerchant(_ context.Context, userID string, merchantID string) (bool, error) {
	f.userID = userID
	f.shopID = merchantID
	return f.allowed, f.err
}

type typedNilMerchantPermissionStore struct{}

func (*typedNilMerchantPermissionStore) UserCanManageMerchant(context.Context, string, string) (bool, error) {
	return true, nil
}

func completeMerchantPermissionDeps(store MerchantPermissionStore) MerchantPermissionDeps {
	return MerchantPermissionDeps{
		UserTokenService: fakeUserTokenService{
			subject: session.UserTokenSubject{UserID: "user-1", Roles: []string{"normal_user"}},
			token:   "user-token",
		},
		AdminTokenService: fakeAdminTokenService{err: errors.New("not an admin token")},
		Store:             store,
	}
}

func TestRequireMerchantAllowsPlatformAdmin(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", nil)
	r.Header.Set("Authorization", "Bearer admin-token")
	store := &fakeMerchantPermissionStore{}
	deps := completeMerchantPermissionDeps(store)
	deps.AdminTokenService = fakeAdminTokenService{
		token: "admin-token",
		subject: session.AdminTokenSubject{
			OperatorID: "operator-1",
			Roles:      []string{permission.RoleSuperAdmin},
		},
	}

	if err := RequireMerchant(r, deps, "merchant-1"); err != nil {
		t.Fatalf("RequireMerchant() error = %v", err)
	}
	if store.userID != "" {
		t.Fatalf("permission store called for admin with userID %q", store.userID)
	}
}

func TestRequireMerchantAllowsUserManagingOwnMerchant(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", nil)
	r.Header.Set("Authorization", "Bearer user-token")
	store := &fakeMerchantPermissionStore{allowed: true}

	err := RequireMerchant(r, completeMerchantPermissionDeps(store), "merchant-1")
	if err != nil {
		t.Fatalf("RequireMerchant() error = %v", err)
	}
	if store.userID != "user-1" || store.shopID != "merchant-1" {
		t.Fatalf("store inputs = (%q, %q), want (user-1, merchant-1)", store.userID, store.shopID)
	}
}

func TestRequireMerchantRejectsUserManagingAnotherMerchant(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-2", nil)
	r.Header.Set("Authorization", "Bearer user-token")

	err := RequireMerchant(r, completeMerchantPermissionDeps(&fakeMerchantPermissionStore{}), "merchant-2")
	if errx.CodeOf(err) != errx.CodeForbidden || errx.PublicMessage(err) != "您没有权限操作该商家" {
		t.Fatalf("error = (%q, %q), want safe forbidden error", errx.CodeOf(err), errx.PublicMessage(err))
	}
}

func TestRequireMerchantFailsSafelyForMissingDependencies(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", nil)
	r.Header.Set("Authorization", "Bearer user-token")
	complete := completeMerchantPermissionDeps(&fakeMerchantPermissionStore{allowed: true})
	tests := []struct {
		name string
		deps MerchantPermissionDeps
	}{
		{name: "缺少用户 Token 服务", deps: MerchantPermissionDeps{AdminTokenService: complete.AdminTokenService, Store: complete.Store}},
		{name: "缺少管理员 Token 服务", deps: MerchantPermissionDeps{UserTokenService: complete.UserTokenService, Store: complete.Store}},
		{name: "缺少权限存储", deps: MerchantPermissionDeps{UserTokenService: complete.UserTokenService, AdminTokenService: complete.AdminTokenService}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireMerchant(r, tc.deps, "merchant-1")
			if errx.CodeOf(err) != errx.CodeInternalError {
				t.Fatalf("error code = %q, want %q", errx.CodeOf(err), errx.CodeInternalError)
			}
		})
	}
}

func TestRequireMerchantFailsSafelyWhenStoreIsTypedNil(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", nil)
	r.Header.Set("Authorization", "Bearer user-token")
	var store *typedNilMerchantPermissionStore

	err := RequireMerchant(r, completeMerchantPermissionDeps(store), "merchant-1")
	if err == nil || errx.CodeOf(err) != errx.CodeInternalError {
		t.Fatalf("error = %v code = %q, want internal error", err, errx.CodeOf(err))
	}
}

func TestRequireMerchantHidesPermissionStoreFailure(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", nil)
	r.Header.Set("Authorization", "Bearer user-token")
	store := &fakeMerchantPermissionStore{err: errors.New("database password leaked")}

	err := RequireMerchant(r, completeMerchantPermissionDeps(store), "merchant-1")
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "商家权限校验失败，请稍后重试" {
		t.Fatalf("error = (%q, %q), want safe internal error", errx.CodeOf(err), errx.PublicMessage(err))
	}
}

func TestRequireMerchantLogsSafePermissionStoreFailureCategories(t *testing.T) {
	const sensitiveDetail = "password=db-secret token=admin-secret Authorization=Bearer-secret"
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: fmt.Errorf("%s: %w", sensitiveDetail, context.DeadlineExceeded), want: "timeout"},
		{name: "canceled", err: fmt.Errorf("%s: %w", sensitiveDetail, context.Canceled), want: "canceled"},
		{name: "unavailable", err: fmt.Errorf("%s: %w", sensitiveDetail, driver.ErrBadConn), want: "unavailable"},
		{name: "unknown", err: errors.New(sensitiveDetail), want: "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			previousWriter := logx.Reset()
			logx.SetWriter(logx.NewWriter(&logBuffer))
			t.Cleanup(func() {
				if currentWriter := logx.Reset(); currentWriter != nil {
					_ = currentWriter.Close()
				}
				if previousWriter != nil {
					logx.SetWriter(previousWriter)
				}
			})

			r := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", nil)
			r.Header.Set("Authorization", "Bearer user-token")
			err := RequireMerchant(r, completeMerchantPermissionDeps(&fakeMerchantPermissionStore{err: tc.err}), "merchant-1")
			if err == nil || errx.CodeOf(err) != errx.CodeInternalError || strings.Contains(errx.PublicMessage(err), sensitiveDetail) {
				t.Fatalf("error = %v code = %q, want safe internal error", err, errx.CodeOf(err))
			}
			logText := logBuffer.String()
			if !strings.Contains(logText, `"errorCategory":"`+tc.want+`"`) {
				t.Fatalf("log = %q, want safe category %q", logText, tc.want)
			}
			for _, forbidden := range []string{sensitiveDetail, "db-secret", "admin-secret", "Bearer-secret"} {
				if strings.Contains(logText, forbidden) {
					t.Fatalf("log contains sensitive detail %q: %q", forbidden, logText)
				}
			}
		})
	}
}
