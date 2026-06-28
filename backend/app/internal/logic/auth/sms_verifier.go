package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
)

type SMSVerifier interface {
	VerifySMSCode(ctx context.Context, phone string, code string) error
}

type SMSCodeSender interface {
	SendSMSCode(ctx context.Context, phone string) error
}

type ConfiguredSMSVerifier struct {
	cfg        config.SMSConfig
	client     *http.Client
	now        func() time.Time
	sendMu     sync.Mutex
	sendLimits map[string]smsSendLimit
}

type smsSendLimit struct {
	LastSentAt time.Time
	Day        string
	Count      int
}

func NewConfiguredSMSVerifier(cfg config.SMSConfig) *ConfiguredSMSVerifier {
	return NewConfiguredSMSVerifierWithHTTP(cfg, nil)
}

func NewConfiguredSMSVerifierWithHTTP(cfg config.SMSConfig, client *http.Client) *ConfiguredSMSVerifier {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &ConfiguredSMSVerifier{cfg: cfg, client: client, now: time.Now, sendLimits: map[string]smsSendLimit{}}
}

func (v *ConfiguredSMSVerifier) SendSMSCode(ctx context.Context, phone string) error {
	provider := strings.TrimSpace(strings.ToLower(v.cfg.Provider))
	if provider == "" {
		return errx.New(errx.CodeInternalError, "短信服务未配置，请稍后重试")
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return errx.New(errx.CodeValidationFailed, "请填写手机号")
	}
	if provider == "dev" {
		if err := v.reserveSMSSend(phone); err != nil {
			return err
		}
		return nil
	}
	if provider == "http" {
		if err := v.reserveSMSSend(phone); err != nil {
			return err
		}
		if err := v.postSMS(ctx, strings.TrimSpace(v.cfg.SendURL), map[string]string{"phone": phone}, "短信验证码发送失败，请稍后重试", false); err != nil {
			v.rollbackSMSSend(phone)
			return err
		}
		return nil
	}
	return errx.New(errx.CodeInternalError, "短信服务供应商尚未接入，请稍后重试")
}

func (v *ConfiguredSMSVerifier) reserveSMSSend(phone string) error {
	now := v.now()
	minInterval := v.cfg.SendMinInterval
	if minInterval <= 0 {
		minInterval = time.Minute
	}
	dailyLimit := v.cfg.DailySendLimit
	if dailyLimit <= 0 {
		dailyLimit = 10
	}

	v.sendMu.Lock()
	defer v.sendMu.Unlock()

	limit := v.sendLimits[phone]
	day := now.Format("2006-01-02")
	if limit.Day != day {
		limit = smsSendLimit{Day: day}
	}
	if !limit.LastSentAt.IsZero() && now.Sub(limit.LastSentAt) < minInterval {
		return errx.New(errx.CodeRateLimited, "验证码发送太频繁，请稍后再试")
	}
	if limit.Count >= dailyLimit {
		return errx.New(errx.CodeRateLimited, "今日验证码发送次数已达上限，请明天再试")
	}
	limit.LastSentAt = now
	limit.Count++
	v.sendLimits[phone] = limit
	return nil
}

func (v *ConfiguredSMSVerifier) rollbackSMSSend(phone string) {
	v.sendMu.Lock()
	defer v.sendMu.Unlock()

	limit, ok := v.sendLimits[phone]
	if !ok {
		return
	}
	if limit.Count > 0 {
		limit.Count--
	}
	limit.LastSentAt = time.Time{}
	if limit.Count == 0 {
		delete(v.sendLimits, phone)
		return
	}
	v.sendLimits[phone] = limit
}

func (v *ConfiguredSMSVerifier) VerifySMSCode(ctx context.Context, phone string, code string) error {
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
	if provider == "http" {
		return v.postSMS(ctx, strings.TrimSpace(v.cfg.VerifyURL), map[string]string{"phone": strings.TrimSpace(phone), "code": strings.TrimSpace(code)}, "短信验证码校验失败，请稍后重试", true)
	}
	return errx.New(errx.CodeInternalError, "短信服务供应商尚未接入，请稍后重试")
}

func (v *ConfiguredSMSVerifier) postSMS(ctx context.Context, endpoint string, payload map[string]string, publicError string, verifyCode bool) error {
	if endpoint == "" {
		return errx.New(errx.CodeInternalError, "短信服务未配置，请稍后重试")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return errx.New(errx.CodeInternalError, publicError)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return errx.New(errx.CodeInternalError, publicError)
	}
	req.Header.Set("Content-Type", "application/json")
	if secret := strings.TrimSpace(v.cfg.AccessKeySecret); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return errx.New(errx.CodeInternalError, publicError)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errx.New(errx.CodeInternalError, publicError)
	}
	var data struct {
		OK    *bool  `json:"ok"`
		Valid *bool  `json:"valid"`
		Error string `json:"error"`
	}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &data); err != nil {
			return errx.New(errx.CodeInternalError, publicError)
		}
		if data.Error != "" {
			if verifyCode {
				return errx.New(errx.CodeValidationFailed, "短信验证码不正确")
			}
			return errx.New(errx.CodeInternalError, publicError)
		}
		if data.Valid != nil {
			if *data.Valid {
				return nil
			}
			return errx.New(errx.CodeValidationFailed, "短信验证码不正确")
		}
		if data.OK != nil {
			if *data.OK {
				return nil
			}
			return errx.New(errx.CodeInternalError, publicError)
		}
	}
	return nil
}
