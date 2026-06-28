package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateForProductionRejectsMissingCriticalConfig(t *testing.T) {
	cfg := Config{
		RuntimeMode: "production",
		AdminAuth:   AdminAuthConfig{TokenTTL: time.Hour},
		Storage:     StorageConfig{Provider: "qiniu-kodo"},
	}

	err := ValidateForProduction(cfg)
	if err == nil {
		t.Fatal("ValidateForProduction() error = nil, want missing config error")
	}
	message := err.Error()
	for _, want := range []string{"Postgres.DSN", "AdminAuth.TokenSecret", "Wechat.AppID", "Wechat.AppSecret", "SMS.Provider", "Storage.AccessKeyID"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want mention %s", message, want)
		}
	}
}

func TestValidateForProductionAcceptsRequiredConfig(t *testing.T) {
	cfg := Config{
		RuntimeMode: "production",
		Postgres:    PostgresConfig{DSN: "postgres://user:pass@127.0.0.1:5432/wplink?sslmode=disable"},
		AdminAuth:   AdminAuthConfig{TokenSecret: "secret", TokenTTL: time.Hour},
		Wechat:      WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		SMS:         SMSConfig{Provider: "aliyun", AccessKeyID: "sms-ak", AccessKeySecret: "sms-sk", SignName: "服链通", TemplateCode: "SMS_001"},
		Storage: StorageConfig{
			Provider:            "qiniu-kodo",
			Endpoint:            "https://upload-z2.qiniup.com",
			Bucket:              "wplink-prod",
			AccessKeyID:         "qiniu-ak",
			AccessKeySecret:     "qiniu-sk",
			PublicBaseURL:       "https://cdn.example.com",
			AllowedContentTypes: []string{"image/png"},
		},
	}

	if err := ValidateForProduction(cfg); err != nil {
		t.Fatalf("ValidateForProduction() error = %v", err)
	}
}
