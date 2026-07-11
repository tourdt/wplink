package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadReadsAppYAMLAndExpandsEnv(t *testing.T) {
	t.Setenv("POSTGRES_PASSWORD", "secret-pass")
	t.Setenv("ADMIN_TOKEN_SECRET", "secret-token")
	t.Setenv("USER_TOKEN_SECRET", "user-secret-token")
	t.Setenv("WECHAT_APP_ID", "wx-local")
	t.Setenv("WECHAT_APP_SECRET", "wechat-secret")

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
  TokenSecret: "${ADMIN_TOKEN_SECRET}"
  TokenTTL: 24h
  MasterPassword: "a123456"

UserAuth:
  TokenSecret: "${USER_TOKEN_SECRET}"
  TokenTTL: 12h

Wechat:
  AppID: "${WECHAT_APP_ID}"
  AppSecret: "${WECHAT_APP_SECRET}"
  AllowDevCode: true

SMS:
  Provider: "http"
  SendURL: "https://sms.example.test/send"
  VerifyURL: "https://sms.example.test/verify"
  SendMinInterval: 45s
  DailySendLimit: 8
  AccessKeySecret: "sms-secret"

Log:
  Mode: "file"
  Encoding: "json"
  Path: "var/log/wplink"
  Level: "debug"
  Rotation: "daily"
  KeepDays: 7
  Compress: true
  Stat: false

WechatPay:
  Enabled: true
  DevMockEnabled: true
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
	if cfg.AdminAuth.TokenSecret != "secret-token" || cfg.AdminAuth.TokenTTL != 24*time.Hour || cfg.AdminAuth.MasterPassword != "a123456" {
		t.Fatalf("admin auth = %#v, want env token, ttl and master password", cfg.AdminAuth)
	}
	if cfg.UserAuth.TokenSecret != "user-secret-token" || cfg.UserAuth.TokenTTL != 12*time.Hour {
		t.Fatalf("user auth = %#v, want independent env token and ttl", cfg.UserAuth)
	}
	if cfg.Wechat.AppID != "wx-local" || cfg.Wechat.AppSecret != "wechat-secret" || !cfg.Wechat.AllowDevCode {
		t.Fatalf("wechat = %#v, want env app config", cfg.Wechat)
	}
	if cfg.SMS.Provider != "http" || cfg.SMS.SendMinInterval != 45*time.Second || cfg.SMS.DailySendLimit != 8 {
		t.Fatalf("sms = %#v, want http rate limit config", cfg.SMS)
	}
	if cfg.Log.Mode != "file" || cfg.Log.Encoding != "json" || cfg.Log.Path != "var/log/wplink" || cfg.Log.Level != "debug" || cfg.Log.Rotation != "daily" || cfg.Log.KeepDays != 7 || !cfg.Log.Compress || cfg.Log.Stat {
		t.Fatalf("log = %#v, want file daily log config", cfg.Log)
	}
	if !cfg.WechatPay.Enabled || !cfg.WechatPay.DevMockEnabled || cfg.WechatPay.MchID != "1900000001" || cfg.WechatPay.RequestTimeout != 10*time.Second {
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

func TestLoadDefaultsToDailyFileLogsKeptSevenDays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte(`
Name: wplink-api
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Log.Mode != "file" || cfg.Log.Path != "logs" || cfg.Log.Rotation != "daily" || cfg.Log.KeepDays != 7 {
		t.Fatalf("log = %#v, want default daily file logs kept 7 days", cfg.Log)
	}
}

func TestLoadDefaultsDevelopmentTokenSecretWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte(`
Name: wplink-api
RuntimeMode: development
AdminAuth:
  TokenSecret: "${ADMIN_TOKEN_SECRET}"
  TokenTTL: 24h
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AdminAuth.TokenSecret == "" {
		t.Fatal("AdminAuth.TokenSecret = empty, want local development fallback")
	}
	if cfg.UserAuth.TokenSecret == "" {
		t.Fatal("UserAuth.TokenSecret = empty, want local development fallback")
	}
	if cfg.UserAuth.TokenSecret == cfg.AdminAuth.TokenSecret {
		t.Fatal("UserAuth.TokenSecret equals AdminAuth.TokenSecret, want isolated local development secrets")
	}
	if cfg.AdminAuth.TokenTTL != 24*time.Hour {
		t.Fatalf("AdminAuth.TokenTTL = %s, want configured TTL", cfg.AdminAuth.TokenTTL)
	}
	if cfg.UserAuth.TokenTTL != 24*time.Hour {
		t.Fatalf("UserAuth.TokenTTL = %s, want admin TTL fallback when user ttl missing", cfg.UserAuth.TokenTTL)
	}
}

func TestLoadDoesNotDefaultProductionTokenSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte(`
Name: wplink-api
RuntimeMode: production
AdminAuth:
  TokenSecret: "${ADMIN_TOKEN_SECRET}"
  TokenTTL: 24h
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AdminAuth.TokenSecret != "" {
		t.Fatalf("AdminAuth.TokenSecret = %q, want empty production config before validation", cfg.AdminAuth.TokenSecret)
	}
	if cfg.UserAuth.TokenSecret != "" {
		t.Fatalf("UserAuth.TokenSecret = %q, want empty production config before validation", cfg.UserAuth.TokenSecret)
	}
}

func TestLoadDoesNotDefaultStagingTokenSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte(`
Name: wplink-api
RuntimeMode: staging
AdminAuth:
  TokenSecret: "${ADMIN_TOKEN_SECRET}"
  TokenTTL: 24h
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AdminAuth.TokenSecret != "" {
		t.Fatalf("AdminAuth.TokenSecret = %q, want empty staging config", cfg.AdminAuth.TokenSecret)
	}
	if cfg.UserAuth.TokenSecret != "" {
		t.Fatalf("UserAuth.TokenSecret = %q, want empty staging config", cfg.UserAuth.TokenSecret)
	}
}

func TestDevelopmentAppConfigPlaceholdersExistInDeployEnvExample(t *testing.T) {
	configBytes, err := os.ReadFile(filepath.Join("..", "..", "..", "etc", "app.yaml"))
	if err != nil {
		t.Fatalf("read app.yaml: %v", err)
	}
	envBytes, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "deploy", "wplink.env.example"))
	if err != nil {
		t.Fatalf("read deploy env example: %v", err)
	}

	envKeys := map[string]struct{}{}
	for _, line := range strings.Split(string(envBytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, _, ok := strings.Cut(line, "=")
		if ok {
			envKeys[strings.TrimSpace(key)] = struct{}{}
		}
	}

	matches := envPlaceholderPattern.FindAllStringSubmatch(string(configBytes), -1)
	for _, match := range matches {
		name := match[1]
		if _, ok := envKeys[name]; !ok {
			t.Fatalf("backend/etc/app.yaml references %s but deploy/wplink.env.example does not define it", name)
		}
	}
}

func TestDevelopmentAppConfigAllowsLocalWechatDevCode(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "..", "etc", "app.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RuntimeMode != "development" {
		t.Fatalf("RuntimeMode = %q, want development", cfg.RuntimeMode)
	}
	if !cfg.Wechat.AllowDevCode {
		t.Fatal("Wechat.AllowDevCode = false, want true for local wxapp login")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte(`
Name: wplink-api
Postgres:
  DSN: "postgres://user:pass@127.0.0.1:5432/wplink?sslmode=disable"
  DSNLOCAL: "postgres://user:pass@127.0.0.1:5432/wplink_local?sslmode=disable"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := func() error {
		_, err := Load(path)
		return err
	}()
	if err == nil {
		t.Fatal("Load() error = nil, want unknown field error")
	}
	if !strings.Contains(err.Error(), "DSNLOCAL") {
		t.Fatalf("Load() error = %v, want mention unknown field DSNLOCAL", err)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte(`
Name: wplink-api
AdminAuth:
  TokenTTL: "one day"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want duration parse error")
	}
	if !strings.Contains(err.Error(), "duration") {
		t.Fatalf("Load() error = %v, want duration parse message", err)
	}
}
