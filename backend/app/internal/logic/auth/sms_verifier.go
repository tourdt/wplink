package auth

import (
	"context"
	"strings"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
)

type SMSVerifier interface {
	VerifySMSCode(ctx context.Context, phone string, code string) error
}

type ConfiguredSMSVerifier struct {
	cfg config.SMSConfig
}

func NewConfiguredSMSVerifier(cfg config.SMSConfig) *ConfiguredSMSVerifier {
	return &ConfiguredSMSVerifier{cfg: cfg}
}

func (v *ConfiguredSMSVerifier) VerifySMSCode(_ context.Context, phone string, code string) error {
	provider := strings.TrimSpace(strings.ToLower(v.cfg.Provider))
	if provider == "" {
		return errx.New(errx.CodeInternalError, "短信服务未配置，请稍后重试")
	}
	if strings.TrimSpace(phone) == "" || strings.TrimSpace(code) == "" {
		return errx.New(errx.CodeValidationFailed, "请填写手机号和短信验证码")
	}
	if provider == "dev" {
		devCode := strings.TrimSpace(v.cfg.DevCode)
		if devCode == "" {
			devCode = "123456"
		}
		if strings.TrimSpace(code) != devCode {
			return errx.New(errx.CodeValidationFailed, "短信验证码不正确")
		}
		return nil
	}
	return errx.New(errx.CodeInternalError, "短信服务供应商尚未接入，请稍后重试")
}
