package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var envPlaceholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

const defaultDevelopmentTokenSecret = "wplink-local-development-token-secret"

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	raw := fileConfig{Log: defaultFileLogConfig()}
	// 配置文件按严格 YAML 解析，未知字段直接报错，避免拼错配置在启动时被静默忽略。
	if err := yaml.UnmarshalStrict([]byte(expandEnvPlaceholders(string(content))), &raw); err != nil {
		return Config{}, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return raw.toConfig(), nil
}

type fileConfig struct {
	Name        string              `yaml:"Name"`
	RuntimeMode string              `yaml:"RuntimeMode"`
	Host        string              `yaml:"Host"`
	Port        int                 `yaml:"Port"`
	Log         fileLogConfig       `yaml:"Log"`
	Postgres    filePostgresConfig  `yaml:"Postgres"`
	AdminAuth   fileAdminAuthConfig `yaml:"AdminAuth"`
	Wechat      WechatConfig        `yaml:"Wechat"`
	WechatPay   fileWechatPayConfig `yaml:"WechatPay"`
	SMS         fileSMSConfig       `yaml:"SMS"`
	Tasks       fileTasksConfig     `yaml:"Tasks"`
	Storage     fileStorageConfig   `yaml:"Storage"`
}

type filePostgresConfig struct {
	DSN             string         `yaml:"DSN"`
	MaxOpenConns    int            `yaml:"MaxOpenConns"`
	MaxIdleConns    int            `yaml:"MaxIdleConns"`
	ConnMaxLifetime configDuration `yaml:"ConnMaxLifetime"`
	ConnMaxIdleTime configDuration `yaml:"ConnMaxIdleTime"`
}

type fileLogConfig struct {
	ServiceName         string `yaml:"ServiceName"`
	Mode                string `yaml:"Mode"`
	Encoding            string `yaml:"Encoding"`
	TimeFormat          string `yaml:"TimeFormat"`
	Path                string `yaml:"Path"`
	Level               string `yaml:"Level"`
	MaxContentLength    uint32 `yaml:"MaxContentLength"`
	Compress            bool   `yaml:"Compress"`
	Stat                bool   `yaml:"Stat"`
	KeepDays            int    `yaml:"KeepDays"`
	StackCooldownMillis int    `yaml:"StackCooldownMillis"`
	MaxBackups          int    `yaml:"MaxBackups"`
	MaxSize             int    `yaml:"MaxSize"`
	Rotation            string `yaml:"Rotation"`
	FileTimeFormat      string `yaml:"FileTimeFormat"`
}

type fileAdminAuthConfig struct {
	TokenSecret string         `yaml:"TokenSecret"`
	TokenTTL    configDuration `yaml:"TokenTTL"`
}

type fileWechatPayConfig struct {
	Enabled                bool           `yaml:"Enabled"`
	DevMockEnabled         bool           `yaml:"DevMockEnabled"`
	MchID                  string         `yaml:"MchID"`
	AppID                  string         `yaml:"AppID"`
	APIv3Key               string         `yaml:"APIv3Key"`
	MerchantSerialNo       string         `yaml:"MerchantSerialNo"`
	MerchantPrivateKeyPath string         `yaml:"MerchantPrivateKeyPath"`
	PlatformPublicKeyPath  string         `yaml:"PlatformPublicKeyPath"`
	NotifyURL              string         `yaml:"NotifyURL"`
	RequestTimeout         configDuration `yaml:"RequestTimeout"`
}

type fileSMSConfig struct {
	Provider        string         `yaml:"Provider"`
	SendURL         string         `yaml:"SendURL"`
	VerifyURL       string         `yaml:"VerifyURL"`
	SendMinInterval configDuration `yaml:"SendMinInterval"`
	DailySendLimit  int            `yaml:"DailySendLimit"`
	AccessKeyID     string         `yaml:"AccessKeyID"`
	AccessKeySecret string         `yaml:"AccessKeySecret"`
	SignName        string         `yaml:"SignName"`
	TemplateCode    string         `yaml:"TemplateCode"`
	DevCode         string         `yaml:"DevCode"`
}

type fileTasksConfig struct {
	ResourceLifecycleInterval configDuration `yaml:"ResourceLifecycleInterval"`
}

type fileStorageConfig struct {
	Provider            string         `yaml:"Provider"`
	Endpoint            string         `yaml:"Endpoint"`
	Bucket              string         `yaml:"Bucket"`
	Region              string         `yaml:"Region"`
	AccessKeyID         string         `yaml:"AccessKeyID"`
	AccessKeySecret     string         `yaml:"AccessKeySecret"`
	PublicBaseURL       string         `yaml:"PublicBaseURL"`
	UploadExpire        configDuration `yaml:"UploadExpire"`
	MaxFileSizeBytes    int64          `yaml:"MaxFileSizeBytes"`
	AllowedContentTypes []string       `yaml:"AllowedContentTypes"`
}

type configDuration time.Duration

func (d *configDuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	if value == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("必须使用 Go duration 格式，例如 30s、15m、24h: %w", err)
	}
	*d = configDuration(parsed)
	return nil
}

func (d configDuration) Duration() time.Duration {
	return time.Duration(d)
}

func (c fileConfig) toConfig() Config {
	adminAuth := AdminAuthConfig{
		TokenSecret: strings.TrimSpace(c.AdminAuth.TokenSecret),
		TokenTTL:    c.AdminAuth.TokenTTL.Duration(),
	}
	if adminAuth.TokenSecret == "" && isDevelopmentMode(c.RuntimeMode) {
		// 本地开发配置经常依赖未导出的环境变量；只在开发模式补固定开发密钥，避免登录链路因空密钥中断。
		adminAuth.TokenSecret = defaultDevelopmentTokenSecret
	}

	return Config{
		Name:        c.Name,
		RuntimeMode: c.RuntimeMode,
		Host:        c.Host,
		Port:        c.Port,
		Log:         c.Log.toLogConfig(),
		Postgres: PostgresConfig{
			DSN:             c.Postgres.DSN,
			MaxOpenConns:    c.Postgres.MaxOpenConns,
			MaxIdleConns:    c.Postgres.MaxIdleConns,
			ConnMaxLifetime: c.Postgres.ConnMaxLifetime.Duration(),
			ConnMaxIdleTime: c.Postgres.ConnMaxIdleTime.Duration(),
		},
		AdminAuth: adminAuth,
		Wechat:    c.Wechat,
		WechatPay: WechatPayConfig{
			Enabled:                c.WechatPay.Enabled,
			DevMockEnabled:         c.WechatPay.DevMockEnabled,
			MchID:                  c.WechatPay.MchID,
			AppID:                  c.WechatPay.AppID,
			APIv3Key:               c.WechatPay.APIv3Key,
			MerchantSerialNo:       c.WechatPay.MerchantSerialNo,
			MerchantPrivateKeyPath: c.WechatPay.MerchantPrivateKeyPath,
			PlatformPublicKeyPath:  c.WechatPay.PlatformPublicKeyPath,
			NotifyURL:              c.WechatPay.NotifyURL,
			RequestTimeout:         c.WechatPay.RequestTimeout.Duration(),
		},
		SMS: SMSConfig{
			Provider:        c.SMS.Provider,
			SendURL:         c.SMS.SendURL,
			VerifyURL:       c.SMS.VerifyURL,
			SendMinInterval: c.SMS.SendMinInterval.Duration(),
			DailySendLimit:  c.SMS.DailySendLimit,
			AccessKeyID:     c.SMS.AccessKeyID,
			AccessKeySecret: c.SMS.AccessKeySecret,
			SignName:        c.SMS.SignName,
			TemplateCode:    c.SMS.TemplateCode,
			DevCode:         c.SMS.DevCode,
		},
		Tasks: TasksConfig{
			ResourceLifecycleInterval: c.Tasks.ResourceLifecycleInterval.Duration(),
		},
		Storage: StorageConfig{
			Provider:            c.Storage.Provider,
			Endpoint:            c.Storage.Endpoint,
			Bucket:              c.Storage.Bucket,
			Region:              c.Storage.Region,
			AccessKeyID:         c.Storage.AccessKeyID,
			AccessKeySecret:     c.Storage.AccessKeySecret,
			PublicBaseURL:       c.Storage.PublicBaseURL,
			UploadExpire:        c.Storage.UploadExpire.Duration(),
			MaxFileSizeBytes:    c.Storage.MaxFileSizeBytes,
			AllowedContentTypes: c.Storage.AllowedContentTypes,
		},
	}
}

func defaultLogConfig() LogConfig {
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

func defaultFileLogConfig() fileLogConfig {
	cfg := defaultLogConfig()
	return fileLogConfig{
		ServiceName:         cfg.ServiceName,
		Mode:                cfg.Mode,
		Encoding:            cfg.Encoding,
		TimeFormat:          cfg.TimeFormat,
		Path:                cfg.Path,
		Level:               cfg.Level,
		MaxContentLength:    cfg.MaxContentLength,
		Compress:            cfg.Compress,
		Stat:                cfg.Stat,
		KeepDays:            cfg.KeepDays,
		StackCooldownMillis: cfg.StackCooldownMillis,
		MaxBackups:          cfg.MaxBackups,
		MaxSize:             cfg.MaxSize,
		Rotation:            cfg.Rotation,
		FileTimeFormat:      cfg.FileTimeFormat,
	}
}

func (c fileLogConfig) toLogConfig() LogConfig {
	return LogConfig{
		ServiceName:         c.ServiceName,
		Mode:                c.Mode,
		Encoding:            c.Encoding,
		TimeFormat:          c.TimeFormat,
		Path:                c.Path,
		Level:               c.Level,
		MaxContentLength:    c.MaxContentLength,
		Compress:            c.Compress,
		Stat:                c.Stat,
		KeepDays:            c.KeepDays,
		StackCooldownMillis: c.StackCooldownMillis,
		MaxBackups:          c.MaxBackups,
		MaxSize:             c.MaxSize,
		Rotation:            c.Rotation,
		FileTimeFormat:      c.FileTimeFormat,
	}
}

func expandEnvPlaceholders(content string) string {
	return envPlaceholderPattern.ReplaceAllStringFunc(content, func(match string) string {
		name := match[2 : len(match)-1]
		return os.Getenv(name)
	})
}

func isDevelopmentMode(mode string) bool {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", "dev", "development", "local":
		return true
	default:
		return false
	}
}
