package svc

import (
	"context"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/session"
)

func TestNewServiceContextBuildsServerDependencies(t *testing.T) {
	cfg := config.Config{
		Name: "wplink-api",
		AdminAuth: config.AdminAuthConfig{
			TokenSecret: "secret",
			TokenTTL:    time.Hour,
		},
		UserAuth: config.UserAuthConfig{
			TokenSecret: "user-secret",
			TokenTTL:    time.Hour,
		},
	}

	ctx, err := NewServiceContext(cfg, nil)
	if err != nil {
		t.Fatalf("NewServiceContext() error = %v", err)
	}
	if ctx.Config.Name != "wplink-api" || ctx.DB != nil {
		t.Fatalf("context config/db = %#v/%v, want configured name and nil db", ctx.Config, ctx.DB)
	}
	if ctx.APIStore == nil {
		t.Fatal("APIStore = nil, want initialized model store")
	}
	if ctx.CityStore == nil {
		t.Fatal("CityStore = nil, want initialized city store")
	}
	if ctx.AdminLoginService == nil {
		t.Fatal("AdminLoginService = nil, want initialized admin login service")
	}
	if ctx.AdminAuth == nil {
		t.Fatal("AdminAuth = nil, want initialized fail-closed admin middleware")
	}
	if ctx.UserTokenService == nil {
		t.Fatal("UserTokenService = nil, want initialized user token service")
	}
	if ctx.ExternalCallObserver == nil {
		t.Fatal("ExternalCallObserver = nil, want shared production observer")
	}
}

func TestNewServiceContextReturnsWechatPayInitError(t *testing.T) {
	cfg := config.Config{
		Name: "wplink-api",
		AdminAuth: config.AdminAuthConfig{
			TokenSecret: "secret",
			TokenTTL:    time.Hour,
		},
		UserAuth: config.UserAuthConfig{
			TokenSecret: "user-secret",
			TokenTTL:    time.Hour,
		},
		WechatPay: config.WechatPayConfig{
			Enabled:                true,
			MerchantPrivateKeyPath: "/path/not/exist/apiclient_key.pem",
			PlatformPublicKeyPath:  "/path/not/exist/wechatpay_pub.pem",
			RequestTimeout:         10 * time.Second,
		},
	}

	ctx, err := NewServiceContext(cfg, nil)
	if err == nil {
		t.Fatal("NewServiceContext() error = nil, want wechat pay init error")
	}
	if ctx != nil {
		t.Fatalf("NewServiceContext() context = %#v, want nil on dependency init failure", ctx)
	}
	if !strings.Contains(err.Error(), "初始化微信支付网关失败") {
		t.Fatalf("NewServiceContext() error = %v, want wrapped wechat pay message", err)
	}
}

func TestNewServiceContextUsesUserAuthForUserTokens(t *testing.T) {
	cfg := config.Config{
		AdminAuth: config.AdminAuthConfig{TokenSecret: "admin-secret", TokenTTL: time.Hour},
		UserAuth:  config.UserAuthConfig{TokenSecret: "user-secret", TokenTTL: time.Hour},
	}
	ctx, err := NewServiceContext(cfg, nil)
	if err != nil {
		t.Fatalf("NewServiceContext() error = %v", err)
	}

	token, err := ctx.UserTokenService.IssueUserToken(context.Background(), session.UserTokenSubject{UserID: "user-1"})
	if err != nil {
		t.Fatalf("IssueUserToken() error = %v", err)
	}
	if _, err := session.NewHMACUserTokenService("user-secret", time.Hour).ParseUserToken(context.Background(), token); err != nil {
		t.Fatalf("ParseUserToken() with user secret error = %v", err)
	}
	if _, err := session.NewHMACUserTokenService("admin-secret", time.Hour).ParseUserToken(context.Background(), token); err == nil {
		t.Fatal("ParseUserToken() with admin secret error = nil, want signature mismatch")
	}
}

func TestEnabledAdminMasterPasswordOnlyAllowsDevelopmentModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{name: "development", mode: "development", want: "a123456"},
		{name: "local", mode: "local", want: "a123456"},
		{name: "empty defaults to development", mode: "", want: "a123456"},
		{name: "staging", mode: "staging", want: ""},
		{name: "production", mode: "production", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				RuntimeMode: tt.mode,
				AdminAuth:   config.AdminAuthConfig{MasterPassword: " a123456 "},
			}

			if got := enabledAdminMasterPassword(cfg); got != tt.want {
				t.Fatalf("enabledAdminMasterPassword() = %q, want %q", got, tt.want)
			}
		})
	}
}
