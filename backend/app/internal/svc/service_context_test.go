package svc

import (
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

	ctx := NewServiceContext(cfg, nil)
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
