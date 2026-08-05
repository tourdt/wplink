package config

import (
	"strings"
	"testing"
	"time"
)

const (
	testAdminTokenSecret = "admin-token-secret-at-least-32-bytes"
	testUserTokenSecret  = "user-token-secret-at-least-32-bytes!"
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
	for _, want := range []string{"Postgres.DSN", "AdminAuth.TokenSecret", "UserAuth.TokenSecret", "Wechat.AppID", "Wechat.AppSecret", "SMS.Provider", "Storage.AccessKeyID", "Tasks.ResourceLifecycleInterval", "Tasks.MerchantMapEventCleanupInterval", "Tasks.MerchantMapEventRetentionDays", "Log.Mode", "Log.Path", "Log.KeepDays"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want mention %s", message, want)
		}
	}
}

func TestValidateForProductionAcceptsRequiredConfig(t *testing.T) {
	cfg := Config{
		RuntimeMode: "production",
		Postgres:    productionPostgresConfig(),
		AdminAuth:   AdminAuthConfig{TokenSecret: testAdminTokenSecret, TokenTTL: time.Hour},
		UserAuth:    UserAuthConfig{TokenSecret: testUserTokenSecret, TokenTTL: time.Hour},
		Wechat:      WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		ContentAudit: ContentAuditConfig{
			Enabled:         true,
			MediaEnabled:    true,
			CallbackToken:   "callback-token",
			CallbackMaxSkew: 5 * time.Minute,
		},
		SMS: SMSConfig{Provider: "aliyun", AccessKeyID: "sms-ak", AccessKeySecret: "sms-sk", SignName: "衣货通", TemplateCode: "SMS_001"},
		Log: defaultProductionLogConfig(),
		Storage: StorageConfig{
			Provider:            "qiniu-kodo",
			Endpoint:            "https://upload-z2.qiniup.com",
			Bucket:              "wplink-prod",
			AccessKeyID:         "qiniu-ak",
			AccessKeySecret:     "qiniu-sk",
			PublicBaseURL:       "https://cdn.example.com",
			AllowedContentTypes: []string{"image/png"},
		},
		Tasks: TasksConfig{
			Enabled:                         true,
			ResourceLifecycleInterval:       time.Hour,
			ResourceLifecycleTimeout:        5 * time.Minute,
			ContentAuditRetryInterval:       time.Minute,
			ContentAuditRetryTimeout:        10 * time.Minute,
			ContentAuditRetryBatchSize:      20,
			MerchantMapEventCleanupInterval: 24 * time.Hour,
			MerchantMapEventCleanupTimeout:  10 * time.Minute,
			MerchantMapEventRetentionDays:   90,
			PaymentReconcileTimeout:         5 * time.Minute,
		},
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

func TestValidateForProductionRejectsWeakOrSharedTokenSecrets(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.AdminAuth.TokenSecret = "short"
	if err := ValidateForProduction(cfg); err == nil || !strings.Contains(err.Error(), "至少需要 32 字节") {
		t.Fatalf("ValidateForProduction(weak secret) error = %v, want minimum length error", err)
	}

	cfg = requiredProductionConfig()
	cfg.UserAuth.TokenSecret = cfg.AdminAuth.TokenSecret
	if err := ValidateForProduction(cfg); err == nil || !strings.Contains(err.Error(), "必须相互独立") {
		t.Fatalf("ValidateForProduction(shared secret) error = %v, want independent secret error", err)
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

func TestValidateForProductionRequiresPaymentReconciliationWhenWechatPayEnabled(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.WechatPay = WechatPayConfig{
		Enabled:                true,
		MchID:                  "merchant-1",
		AppID:                  "wx-app",
		APIv3Key:               "12345678901234567890123456789012",
		MerchantSerialNo:       "serial-1",
		MerchantPrivateKeyPath: "/etc/wplink/apiclient_key.pem",
		PlatformPublicKeyPath:  "/etc/wplink/wechatpay_pub.pem",
		NotifyURL:              "https://api.example.com/api/v1/wechat-pay/notify",
		RequestTimeout:         10 * time.Second,
		OrderExpire:            30 * time.Minute,
	}

	err := ValidateForProduction(cfg)
	if err == nil {
		t.Fatal("ValidateForProduction() error = nil, want payment reconciliation config error")
	}
	for _, want := range []string{
		"Tasks.PaymentReconcileInterval",
		"Tasks.PaymentQueryDelay",
		"Tasks.PaymentBatchSize",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want mention %s", err, want)
		}
	}
}

func TestValidateForProductionRequiresContentAudit(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.ContentAudit.MediaEnabled = false

	err := ValidateForProduction(cfg)
	if err == nil || !strings.Contains(err.Error(), "ContentAudit.MediaEnabled") {
		t.Fatalf("ValidateForProduction() error = %v, want require media content audit", err)
	}
}

func TestValidateForProductionRequiresPositiveMerchantMapEventRetentionTask(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.Tasks.MerchantMapEventCleanupInterval = 0
	cfg.Tasks.MerchantMapEventRetentionDays = 0

	err := ValidateForProduction(cfg)
	if err == nil {
		t.Fatal("ValidateForProduction() error = nil, want map event retention task config error")
	}
	for _, want := range []string{"Tasks.MerchantMapEventCleanupInterval", "Tasks.MerchantMapEventRetentionDays"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want mention %s", err, want)
		}
	}
}

func TestValidateForProductionRequiresPositiveTaskTimeouts(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.Tasks.ResourceLifecycleTimeout = 0
	cfg.Tasks.ContentAuditRetryTimeout = 0
	cfg.Tasks.MerchantMapEventCleanupTimeout = 0
	cfg.Tasks.PaymentReconcileTimeout = 0

	err := ValidateForProduction(cfg)
	if err == nil {
		t.Fatal("ValidateForProduction() error = nil, want task timeout config error")
	}
	for _, want := range []string{
		"Tasks.ResourceLifecycleTimeout",
		"Tasks.ContentAuditRetryTimeout",
		"Tasks.MerchantMapEventCleanupTimeout",
		"Tasks.PaymentReconcileTimeout",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want mention %s", err, want)
		}
	}
}

func TestValidateForProductionRejectsAdminMasterPassword(t *testing.T) {
	cfg := requiredProductionConfig()
	cfg.AdminAuth.MasterPassword = "a123456"

	err := ValidateForProduction(cfg)
	if err == nil || !strings.Contains(err.Error(), "AdminAuth.MasterPassword") {
		t.Fatalf("ValidateForProduction() error = %v, want reject admin master password", err)
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
		AdminAuth:   AdminAuthConfig{TokenSecret: testAdminTokenSecret, TokenTTL: time.Hour},
		UserAuth:    UserAuthConfig{TokenSecret: testUserTokenSecret, TokenTTL: time.Hour},
		Wechat:      WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"},
		ContentAudit: ContentAuditConfig{
			Enabled:         true,
			MediaEnabled:    true,
			CallbackToken:   "callback-token",
			CallbackMaxSkew: 5 * time.Minute,
		},
		SMS: SMSConfig{Provider: "aliyun", AccessKeyID: "sms-ak", AccessKeySecret: "sms-sk", SignName: "衣货通", TemplateCode: "SMS_001"},
		Log: defaultProductionLogConfig(),
		Storage: StorageConfig{
			Provider:            "qiniu-kodo",
			Endpoint:            "https://upload-z2.qiniup.com",
			Bucket:              "wplink-prod",
			AccessKeyID:         "qiniu-ak",
			AccessKeySecret:     "qiniu-sk",
			PublicBaseURL:       "https://cdn.example.com",
			AllowedContentTypes: []string{"image/png"},
		},
		Tasks: TasksConfig{
			Enabled:                         true,
			ResourceLifecycleInterval:       time.Hour,
			ResourceLifecycleTimeout:        5 * time.Minute,
			ContentAuditRetryInterval:       time.Minute,
			ContentAuditRetryTimeout:        10 * time.Minute,
			ContentAuditRetryBatchSize:      20,
			MerchantMapEventCleanupInterval: 24 * time.Hour,
			MerchantMapEventCleanupTimeout:  10 * time.Minute,
			MerchantMapEventRetentionDays:   90,
			PaymentReconcileTimeout:         5 * time.Minute,
		},
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
