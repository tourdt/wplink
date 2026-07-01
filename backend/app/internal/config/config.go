package config

import "time"

type Config struct {
	Name        string
	RuntimeMode string
	Host        string
	Port        int
	Postgres    PostgresConfig
	AdminAuth   AdminAuthConfig
	Wechat      WechatConfig
	WechatPay   WechatPayConfig
	SMS         SMSConfig
	Tasks       TasksConfig
	Storage     StorageConfig
}

type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type AdminAuthConfig struct {
	TokenSecret string
	TokenTTL    time.Duration
}

type WechatConfig struct {
	AppID        string
	AppSecret    string
	AllowDevCode bool
}

type WechatPayConfig struct {
	Enabled                bool
	DevMockEnabled         bool
	MchID                  string
	AppID                  string
	APIv3Key               string
	MerchantSerialNo       string
	MerchantPrivateKeyPath string
	PlatformPublicKeyPath  string
	NotifyURL              string
	RequestTimeout         time.Duration
}

type SMSConfig struct {
	Provider        string
	SendURL         string
	VerifyURL       string
	SendMinInterval time.Duration
	DailySendLimit  int
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	DevCode         string
}

type TasksConfig struct {
	ResourceLifecycleInterval time.Duration
}

type StorageConfig struct {
	Provider            string
	Endpoint            string
	Bucket              string
	Region              string
	AccessKeyID         string
	AccessKeySecret     string
	PublicBaseURL       string
	UploadExpire        time.Duration
	MaxFileSizeBytes    int64
	AllowedContentTypes []string
}
