package config

import (
	"fmt"
	"strings"
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

	require("Postgres.DSN", cfg.Postgres.DSN)
	require("AdminAuth.TokenSecret", cfg.AdminAuth.TokenSecret)
	require("Wechat.AppID", cfg.Wechat.AppID)
	require("Wechat.AppSecret", cfg.Wechat.AppSecret)
	require("SMS.Provider", cfg.SMS.Provider)
	require("SMS.AccessKeyID", cfg.SMS.AccessKeyID)
	require("SMS.AccessKeySecret", cfg.SMS.AccessKeySecret)
	require("SMS.SignName", cfg.SMS.SignName)
	require("SMS.TemplateCode", cfg.SMS.TemplateCode)
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
