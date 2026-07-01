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
	for _, want := range []string{"Postgres.DSN", "AdminAuth.TokenSecret", "Wechat.AppID", "Wechat.AppSecret", "SMS.Provider", "Storage.AccessKeyID", "Tasks.ResourceLifecycleInterval", "Log.Mode", "Log.Path", "Log.KeepDays"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want mention %s", message, want)
		}
	}
}

func TestValidateForProductionAcceptsRequiredConfig(t *testing.T) {
	cfg := Config{
		RuntimeMode: "production",
		Postgres:    productionPostgresConfig(),
		AdminAuth:   AdminAuthConfig{TokenSecret: "secret", TokenTTL: time.Hour},
		Wechat:      WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		SMS:         SMSConfig{Provider: "aliyun", AccessKeyID: "sms-ak", AccessKeySecret: "sms-sk", SignName: "衣货通", TemplateCode: "SMS_001"},
		Log:         defaultProductionLogConfig(),
		Storage: StorageConfig{
			Provider:            "qiniu-kodo",
			Endpoint:            "https://upload-z2.qiniup.com",
			Bucket:              "wplink-prod",
			AccessKeyID:         "qiniu-ak",
			AccessKeySecret:     "qiniu-sk",
			PublicBaseURL:       "https://cdn.example.com",
			AllowedContentTypes: []string{"image/png"},
		},
		Tasks: TasksConfig{ResourceLifecycleInterval: time.Hour},
	}

	if err := ValidateForProduction(cfg); err != nil {
		t.Fatalf("ValidateForProduction() error = %v", err)
	}
}

func TestValidateForProductionAcceptsHTTPProviderConfig(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.SMS = SMSConfig{
		Provider:        "http",
		SendURL:         "https://sms.example.test/send",
		VerifyURL:       "https://sms.example.test/verify",
		AccessKeySecret: "sms-secret",
	}

	if err := ValidateForProduction(cfg); err != nil {
		t.Fatalf("ValidateForProduction() error = %v", err)
	}
}

func TestValidateForProductionRequiresHTTPProviderURLs(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.SMS = SMSConfig{Provider: "http", AccessKeySecret: "sms-secret"}

	err := ValidateForProduction(cfg)
	if err == nil {
		t.Fatal("ValidateForProduction() error = nil, want missing http sms urls")
	}
	message := err.Error()
	for _, want := range []string{"SMS.SendURL", "SMS.VerifyURL"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want mention %s", message, want)
		}
	}
}

func TestValidateForProductionRejectsDevSMSProvider(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.SMS = SMSConfig{Provider: "dev", DevCode: "123456"}

	err := ValidateForProduction(cfg)
	if err == nil || !strings.Contains(err.Error(), "SMS.Provider") {
		t.Fatalf("ValidateForProduction() error = %v, want reject dev sms provider", err)
	}
}

func TestValidateForProductionRejectsWechatPayDevMock(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.WechatPay.DevMockEnabled = true

	err := ValidateForProduction(cfg)
	if err == nil || !strings.Contains(err.Error(), "WechatPay.DevMockEnabled") {
		t.Fatalf("ValidateForProduction() error = %v, want reject wechat pay dev mock", err)
	}
}

func TestValidateForProductionRejectsConsoleLogMode(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.Log.Mode = "console"

	err := ValidateForProduction(cfg)
	if err == nil || !strings.Contains(err.Error(), "Log.Mode") {
		t.Fatalf("ValidateForProduction() error = %v, want reject console log mode", err)
	}
}

func TestValidateForProductionRejectsLogRetentionOverSevenDays(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.Log.KeepDays = 8

	err := ValidateForProduction(cfg)
	if err == nil || !strings.Contains(err.Error(), "Log.KeepDays") {
		t.Fatalf("ValidateForProduction() error = %v, want reject log retention over 7 days", err)
	}
}

func TestValidateForProductionRejectsNonDailyLogRotation(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.Log.Rotation = "size"

	err := ValidateForProduction(cfg)
	if err == nil || !strings.Contains(err.Error(), "Log.Rotation") {
		t.Fatalf("ValidateForProduction() error = %v, want reject non-daily log rotation", err)
	}
}

func TestValidateForProductionRequiresPostgresPoolConfig(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.Postgres = PostgresConfig{DSN: cfg.Postgres.DSN}

	err := ValidateForProduction(cfg)
	if err == nil {
		t.Fatal("ValidateForProduction() error = nil, want missing postgres pool config")
	}
	message := err.Error()
	for _, want := range []string{"Postgres.MaxOpenConns", "Postgres.MaxIdleConns", "Postgres.ConnMaxLifetime", "Postgres.ConnMaxIdleTime"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want mention %s", message, want)
		}
	}
}

func requiredProductionConfig() Config {
	return Config{
		RuntimeMode: "production",
		Postgres:    productionPostgresConfig(),
		AdminAuth:   AdminAuthConfig{TokenSecret: "secret", TokenTTL: time.Hour},
		Wechat:      WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		SMS:         SMSConfig{Provider: "aliyun", AccessKeyID: "sms-ak", AccessKeySecret: "sms-sk", SignName: "衣货通", TemplateCode: "SMS_001"},
		Log:         defaultProductionLogConfig(),
		Storage: StorageConfig{
			Provider:            "qiniu-kodo",
			Endpoint:            "https://upload-z2.qiniup.com",
			Bucket:              "wplink-prod",
			AccessKeyID:         "qiniu-ak",
			AccessKeySecret:     "qiniu-sk",
			PublicBaseURL:       "https://cdn.example.com",
			AllowedContentTypes: []string{"image/png"},
		},
		Tasks: TasksConfig{ResourceLifecycleInterval: time.Hour},
	}
}

func defaultProductionLogConfig() LogConfig {
	return LogConfig{
		Mode:     "file",
		Encoding: "json",
		Path:     "logs",
		Level:    "info",
		Rotation: "daily",
		KeepDays: 7,
		Stat:     true,
	}
}

func productionPostgresConfig() PostgresConfig {
	return PostgresConfig{
		DSN:             "postgres://user:pass@127.0.0.1:5432/wplink?sslmode=disable",
		MaxOpenConns:    30,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}
