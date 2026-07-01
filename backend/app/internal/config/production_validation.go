package config

import (
	"fmt"
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
	require("Wechat.AppID", cfg.Wechat.AppID)
	require("Wechat.AppSecret", cfg.Wechat.AppSecret)
	validateProductionWechatPay(cfg.WechatPay, require, requirePositiveDuration)
	validateProductionSMS(cfg.SMS, require, &missing)
	requirePositiveDuration("Tasks.ResourceLifecycleInterval", cfg.Tasks.ResourceLifecycleInterval)
	require("Storage.Provider", cfg.Storage.Provider)
	require("Storage.Endpoint", cfg.Storage.Endpoint)
	require("Storage.Bucket", cfg.Storage.Bucket)
	require("Storage.AccessKeyID", cfg.Storage.AccessKeyID)
	require("Storage.AccessKeySecret", cfg.Storage.AccessKeySecret)
	require("Storage.PublicBaseURL", cfg.Storage.PublicBaseURL)

	if len(missing) > 0 {
		return fmt.Errorf("生产配置缺失: %s", strings.Join(missing, ", "))
	}
	if cfg.Wechat.AllowDevCode {
		return fmt.Errorf("生产配置不允许启用 Wechat.AllowDevCode")
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
