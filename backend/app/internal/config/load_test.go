package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReadsAppYAMLAndExpandsEnv(t *testing.T) {
	t.Setenv("POSTGRES_PASSWORD", "secret-pass")
	t.Setenv("JWT_SECRET", "secret-token")

	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte(`
Name: wplink-api
Host: 127.0.0.1
Port: 4000

Postgres:
  DSN: "postgres://wplink_app:${POSTGRES_PASSWORD}@127.0.0.1:5432/wplink?sslmode=disable"
  MaxOpenConns: 30
  MaxIdleConns: 10
  ConnMaxLifetime: 30m
  ConnMaxIdleTime: 5m

AdminAuth:
  TokenSecret: "${JWT_SECRET}"
  TokenTTL: 24h

SMS:
  Provider: "http"
  SendURL: "https://sms.example.test/send"
  VerifyURL: "https://sms.example.test/verify"
  SendMinInterval: 45s
  DailySendLimit: 8
  AccessKeySecret: "sms-secret"

WechatPay:
  Enabled: true
  MchID: "1900000001"
  AppID: "wx-pay-app"
  APIv3Key: "${WECHAT_PAY_API_V3_KEY}"
  MerchantSerialNo: "serial-no"
  MerchantPrivateKeyPath: "/secure/apiclient_key.pem"
  PlatformPublicKeyPath: "/secure/wechatpay_pub.pem"
  NotifyURL: "https://api.example.com/api/v1/wechat-pay/verification/notify"
  RequestTimeout: 10s

Tasks:
  ResourceLifecycleInterval: 1h

Storage:
  Provider: "qiniu-kodo"
  Endpoint: "https://upload-z2.qiniup.com"
  Bucket: "wplink-prod"
  Region: "z2"
  AccessKeyID: "${QINIU_ACCESS_KEY}"
  AccessKeySecret: "${QINIU_SECRET_KEY}"
  PublicBaseURL: "https://cdn.example.com"
  UploadExpire: 15m
  MaxFileSizeBytes: 10485760
  AllowedContentTypes:
    - "image/jpeg"
    - "image/png"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Name != "wplink-api" || cfg.Host != "127.0.0.1" || cfg.Port != 4000 {
		t.Fatalf("http config = %#v, want name host port", cfg)
	}
	if cfg.Postgres.DSN != "postgres://wplink_app:secret-pass@127.0.0.1:5432/wplink?sslmode=disable" {
		t.Fatalf("dsn = %q, want env expanded", cfg.Postgres.DSN)
	}
	if cfg.Postgres.MaxOpenConns != 30 || cfg.Postgres.MaxIdleConns != 10 || cfg.Postgres.ConnMaxLifetime != 30*time.Minute || cfg.Postgres.ConnMaxIdleTime != 5*time.Minute {
		t.Fatalf("postgres pool = %#v, want configured pool", cfg.Postgres)
	}
	if cfg.AdminAuth.TokenSecret != "secret-token" || cfg.AdminAuth.TokenTTL != 24*time.Hour {
		t.Fatalf("admin auth = %#v, want env token and ttl", cfg.AdminAuth)
	}
	if cfg.SMS.Provider != "http" || cfg.SMS.SendMinInterval != 45*time.Second || cfg.SMS.DailySendLimit != 8 {
		t.Fatalf("sms = %#v, want http rate limit config", cfg.SMS)
	}
	if !cfg.WechatPay.Enabled || cfg.WechatPay.MchID != "1900000001" || cfg.WechatPay.RequestTimeout != 10*time.Second {
		t.Fatalf("wechat pay = %#v, want enabled merchant config", cfg.WechatPay)
	}
	if cfg.Tasks.ResourceLifecycleInterval != time.Hour {
		t.Fatalf("tasks = %#v, want resource lifecycle interval", cfg.Tasks)
	}
	if cfg.Storage.Provider != "qiniu-kodo" || cfg.Storage.UploadExpire != 15*time.Minute || cfg.Storage.MaxFileSizeBytes != 10485760 {
		t.Fatalf("storage = %#v, want qiniu config", cfg.Storage)
	}
	if len(cfg.Storage.AllowedContentTypes) != 2 || cfg.Storage.AllowedContentTypes[0] != "image/jpeg" {
		t.Fatalf("allowed content types = %#v", cfg.Storage.AllowedContentTypes)
	}
}
