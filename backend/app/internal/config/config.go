package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type Config struct {
	Name         string
	RuntimeMode  string
	Host         string
	Port         int
	Log          LogConfig
	Postgres     PostgresConfig
	AdminAuth    AdminAuthConfig
	UserAuth     UserAuthConfig
	Wechat       WechatConfig
	TencentMap   TencentMapConfig
	ContentAudit ContentAuditConfig
	WechatPay    WechatPayConfig
	SMS          SMSConfig
	Tasks        TasksConfig
	Storage      StorageConfig
}

type LogConfig = logx.LogConf

type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type AdminAuthConfig struct {
	TokenSecret    string
	TokenTTL       time.Duration
	MasterPassword string
}

type UserAuthConfig struct {
	TokenSecret string
	TokenTTL    time.Duration
}

type WechatConfig struct {
	AppID        string `yaml:"AppID"`
	AppSecret    string `yaml:"AppSecret"`
	AllowDevCode bool   `yaml:"AllowDevCode"`
}

type TencentMapConfig struct {
	Key            string
	RequestTimeout time.Duration
}

type ContentAuditConfig struct {
	Enabled         bool
	TextScene       int
	MediaEnabled    bool
	MediaScene      int
	RequestTimeout  time.Duration
	MaxTextChars    int
	CallbackToken   string
	CallbackMaxSkew time.Duration
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
	OrderExpire            time.Duration
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
	ResourceLifecycleInterval       time.Duration
	ContentAuditRetryInterval       time.Duration
	ContentAuditRetryBatchSize      int64
	MerchantMapEventCleanupInterval time.Duration
	MerchantMapEventRetentionDays   int
	PaymentReconcileInterval        time.Duration
	PaymentQueryDelay               time.Duration
	PaymentBatchSize                int64
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
