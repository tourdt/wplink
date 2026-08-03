package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func IsProductionMode(mode string) bool {
	mode = strings.TrimSpace(strings.ToLower(mode))
	return mode == "prod" || mode == "production"
}

func ValidateForProduction(cfg Config) error {
	if !IsProductionMode(cfg.RuntimeMode) {
		return nil
	}

	var missing []string
	require := func(name string, value string) {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	requirePositiveInt := func(name string, value int) {
		if value <= 0 {
			missing = append(missing, name)
		}
	}
	requirePositiveDuration := func(name string, value time.Duration) {
		if value <= 0 {
			missing = append(missing, name)
		}
	}

	require("Postgres.DSN", cfg.Postgres.DSN)
	requirePositiveInt("Postgres.MaxOpenConns", cfg.Postgres.MaxOpenConns)
	requirePositiveInt("Postgres.MaxIdleConns", cfg.Postgres.MaxIdleConns)
	requirePositiveDuration("Postgres.ConnMaxLifetime", cfg.Postgres.ConnMaxLifetime)
	requirePositiveDuration("Postgres.ConnMaxIdleTime", cfg.Postgres.ConnMaxIdleTime)
	require("AdminAuth.TokenSecret", cfg.AdminAuth.TokenSecret)
	requirePositiveDuration("AdminAuth.TokenTTL", cfg.AdminAuth.TokenTTL)
	require("UserAuth.TokenSecret", cfg.UserAuth.TokenSecret)
	requirePositiveDuration("UserAuth.TokenTTL", cfg.UserAuth.TokenTTL)
	require("Wechat.AppID", cfg.Wechat.AppID)
	require("Wechat.AppSecret", cfg.Wechat.AppSecret)
	validateProductionWechatPay(cfg.WechatPay, require, requirePositiveDuration)
	if cfg.WechatPay.Enabled {
		requirePositiveDuration("Tasks.PaymentReconcileInterval", cfg.Tasks.PaymentReconcileInterval)
		requirePositiveDuration("Tasks.PaymentQueryDelay", cfg.Tasks.PaymentQueryDelay)
		if cfg.Tasks.PaymentBatchSize <= 0 {
			missing = append(missing, "Tasks.PaymentBatchSize")
		}
	}
	validateProductionSMS(cfg.SMS, require, &missing)
	validateProductionLog(cfg.Log, require, requirePositiveInt)
	requirePositiveDuration("Tasks.ResourceLifecycleInterval", cfg.Tasks.ResourceLifecycleInterval)
	requirePositiveDuration("Tasks.ContentAuditRetryInterval", cfg.Tasks.ContentAuditRetryInterval)
	requirePositiveDuration("Tasks.MerchantMapEventCleanupInterval", cfg.Tasks.MerchantMapEventCleanupInterval)
	requirePositiveInt("Tasks.MerchantMapEventRetentionDays", cfg.Tasks.MerchantMapEventRetentionDays)
	if cfg.Tasks.ContentAuditRetryBatchSize <= 0 {
		missing = append(missing, "Tasks.ContentAuditRetryBatchSize")
	}
	require("Storage.Provider", cfg.Storage.Provider)
	require("Storage.Endpoint", cfg.Storage.Endpoint)
	require("Storage.Bucket", cfg.Storage.Bucket)
	require("Storage.AccessKeyID", cfg.Storage.AccessKeyID)
	require("Storage.AccessKeySecret", cfg.Storage.AccessKeySecret)
	require("Storage.PublicBaseURL", cfg.Storage.PublicBaseURL)

	if len(missing) > 0 {
		return fmt.Errorf("生产配置缺失: %s", strings.Join(missing, ", "))
	}
	if len([]byte(cfg.AdminAuth.TokenSecret)) < 32 {
		return fmt.Errorf("生产配置 AdminAuth.TokenSecret 至少需要 32 字节")
	}
	if len([]byte(cfg.UserAuth.TokenSecret)) < 32 {
		return fmt.Errorf("生产配置 UserAuth.TokenSecret 至少需要 32 字节")
	}
	if cfg.AdminAuth.TokenSecret == cfg.UserAuth.TokenSecret {
		return fmt.Errorf("生产配置 AdminAuth.TokenSecret 与 UserAuth.TokenSecret 必须相互独立")
	}
	if strings.TrimSpace(cfg.AdminAuth.MasterPassword) != "" {
		return fmt.Errorf("生产配置不允许启用 AdminAuth.MasterPassword")
	}
	if !cfg.ContentAudit.Enabled {
		return fmt.Errorf("生产配置必须启用 ContentAudit.Enabled")
	}
	if !cfg.ContentAudit.MediaEnabled {
		return fmt.Errorf("生产配置必须启用 ContentAudit.MediaEnabled")
	}
	if strings.TrimSpace(cfg.ContentAudit.CallbackToken) == "" {
		return fmt.Errorf("生产配置缺失: ContentAudit.CallbackToken")
	}
	if cfg.ContentAudit.CallbackMaxSkew <= 0 {
		return fmt.Errorf("生产配置缺失: ContentAudit.CallbackMaxSkew")
	}
	if cfg.WechatPay.DevMockEnabled {
		return fmt.Errorf("生产配置不允许启用 WechatPay.DevMockEnabled")
	}
	if cfg.WechatPay.Enabled {
		if len(cfg.WechatPay.APIv3Key) != 32 {
			return fmt.Errorf("生产配置 WechatPay.APIv3Key 必须为 32 字节")
		}
		notifyURL, err := url.Parse(strings.TrimSpace(cfg.WechatPay.NotifyURL))
		if err != nil || notifyURL.Scheme != "https" || notifyURL.Host == "" || notifyURL.Path != "/api/v1/wechat-pay/notify" {
			return fmt.Errorf("生产配置 WechatPay.NotifyURL 必须是 HTTPS 统一回调地址，路径固定为 /api/v1/wechat-pay/notify")
		}
	}
	if err := validateProductionLogPolicy(cfg.Log); err != nil {
		return err
	}
	return nil
}

func validateProductionWechatPay(cfg WechatPayConfig, require func(string, string), requirePositiveDuration func(string, time.Duration)) {
	if !cfg.Enabled {
		return
	}
	require("WechatPay.MchID", cfg.MchID)
	require("WechatPay.AppID", cfg.AppID)
	require("WechatPay.APIv3Key", cfg.APIv3Key)
	require("WechatPay.MerchantSerialNo", cfg.MerchantSerialNo)
	require("WechatPay.MerchantPrivateKeyPath", cfg.MerchantPrivateKeyPath)
	require("WechatPay.PlatformPublicKeyPath", cfg.PlatformPublicKeyPath)
	require("WechatPay.NotifyURL", cfg.NotifyURL)
	requirePositiveDuration("WechatPay.RequestTimeout", cfg.RequestTimeout)
	requirePositiveDuration("WechatPay.OrderExpire", cfg.OrderExpire)
}

func validateProductionSMS(cfg SMSConfig, require func(string, string), missing *[]string) {
	provider := strings.TrimSpace(strings.ToLower(cfg.Provider))
	require("SMS.Provider", cfg.Provider)
	switch provider {
	case "":
		return
	case "dev":
		*missing = append(*missing, "SMS.Provider(不能使用 dev)")
	case "http":
		require("SMS.SendURL", cfg.SendURL)
		require("SMS.VerifyURL", cfg.VerifyURL)
		require("SMS.AccessKeySecret", cfg.AccessKeySecret)
	default:
		require("SMS.AccessKeyID", cfg.AccessKeyID)
		require("SMS.AccessKeySecret", cfg.AccessKeySecret)
		require("SMS.SignName", cfg.SignName)
		require("SMS.TemplateCode", cfg.TemplateCode)
	}
}

func validateProductionLog(cfg LogConfig, require func(string, string), requirePositiveInt func(string, int)) {
	require("Log.Mode", cfg.Mode)
	require("Log.Path", cfg.Path)
	require("Log.Rotation", cfg.Rotation)
	requirePositiveInt("Log.KeepDays", cfg.KeepDays)
}

func validateProductionLogPolicy(cfg LogConfig) error {
	mode := strings.TrimSpace(strings.ToLower(cfg.Mode))
	if mode != "file" && mode != "volume" {
		return fmt.Errorf("生产配置 Log.Mode 必须为 file 或 volume")
	}
	if strings.TrimSpace(strings.ToLower(cfg.Rotation)) != "daily" {
		return fmt.Errorf("生产配置 Log.Rotation 必须为 daily")
	}
	if cfg.KeepDays > 7 {
		return fmt.Errorf("生产配置 Log.KeepDays 不能超过 7 天")
	}
	return nil
}
