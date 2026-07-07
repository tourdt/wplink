package svc

import (
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
)

func TestNewServiceContextBuildsServerDependencies(t *testing.T) {
	cfg := config.Config{
		Name: "wplink-api",
		AdminAuth: config.AdminAuthConfig{
			TokenSecret: "secret",
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
	if ctx.UserTokenService == nil {
		t.Fatal("UserTokenService = nil, want initialized user token service")
	}
}

func TestNewServiceContextReturnsWechatPayInitError(t *testing.T) {
	cfg := config.Config{
		Name: "wplink-api",
		AdminAuth: config.AdminAuthConfig{
			TokenSecret: "secret",
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
