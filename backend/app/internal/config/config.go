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
	SMS         SMSConfig
	Storage     StorageConfig
}

type PostgresConfig struct {
	DSN string
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

type SMSConfig struct {
	Provider        string
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	DevCode         string
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
