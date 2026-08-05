package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
	"wplink/backend/common/externalcall"

	"github.com/google/uuid"
)

type SMSVerifier interface {
	VerifySMSCode(ctx context.Context, phone string, code string) error
}

type SMSCodeSender interface {
	SendSMSCode(ctx context.Context, phone string) error
}

var (
	ErrSMSSendTooFrequent = errors.New("sms send too frequent")
	ErrSMSDailyLimit      = errors.New("sms daily send limit reached")
)

type SMSSendLimiter interface {
	Reserve(ctx context.Context, phone string, now time.Time, minInterval time.Duration, dailyLimit int) (string, error)
	Rollback(ctx context.Context, phone string, now time.Time, reservationToken string) error
}

type ConfiguredSMSVerifier struct {
	cfg      config.SMSConfig
	client   *http.Client
	now      func() time.Time
	limiter  SMSSendLimiter
	observer externalcall.Observer
}

func NewConfiguredSMSVerifier(cfg config.SMSConfig) *ConfiguredSMSVerifier {
	return NewConfiguredSMSVerifierWithHTTP(cfg, nil)
}

func NewConfiguredSMSVerifierWithHTTP(cfg config.SMSConfig, client *http.Client) *ConfiguredSMSVerifier {
	return NewConfiguredSMSVerifierWithLimiter(cfg, client, NewMemorySMSSendLimiter())
}

func NewConfiguredSMSVerifierWithLimiter(cfg config.SMSConfig, client *http.Client, limiter SMSSendLimiter, observers ...externalcall.Observer) *ConfiguredSMSVerifier {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if limiter == nil {
		limiter = NewMemorySMSSendLimiter()
	}
	var observer externalcall.Observer
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &ConfiguredSMSVerifier{cfg: cfg, client: client, now: time.Now, limiter: limiter, observer: observer}
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
		if _, err := v.reserveSMSSend(ctx, phone); err != nil {
			return err
		}
		return nil
	}
	if provider == "http" {
		reservationToken, err := v.reserveSMSSend(ctx, phone)
		if err != nil {
			return err
		}
		if err := v.postSMS(ctx, strings.TrimSpace(v.cfg.SendURL), map[string]string{"phone": phone}, "短信验证码发送失败，请稍后重试", externalcall.OperationSMSCodeSend); err != nil {
			_ = v.limiter.Rollback(ctx, phone, v.now(), reservationToken)
			return err
		}
		return nil
	}
	return errx.New(errx.CodeInternalError, "短信服务供应商尚未接入，请稍后重试")
}

func (v *ConfiguredSMSVerifier) reserveSMSSend(ctx context.Context, phone string) (string, error) {
	now := v.now()
	minInterval := v.cfg.SendMinInterval
	if minInterval <= 0 {
		minInterval = time.Minute
	}
	dailyLimit := v.cfg.DailySendLimit
	if dailyLimit <= 0 {
		dailyLimit = 10
	}

	reservationToken, err := v.limiter.Reserve(ctx, phone, now, minInterval, dailyLimit)
	switch {
	case errors.Is(err, ErrSMSSendTooFrequent):
		return "", errx.New(errx.CodeRateLimited, "验证码发送太频繁，请稍后再试")
	case errors.Is(err, ErrSMSDailyLimit):
		return "", errx.New(errx.CodeRateLimited, "今日验证码发送次数已达上限，请明天再试")
	case err != nil:
		return "", errx.New(errx.CodeInternalError, "验证码发送服务暂不可用，请稍后重试")
	default:
		return reservationToken, nil
	}
}

func newSMSSendReservationToken() string {
	return uuid.NewString()
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
		return v.postSMS(ctx, strings.TrimSpace(v.cfg.VerifyURL), map[string]string{"phone": strings.TrimSpace(phone), "code": strings.TrimSpace(code)}, "短信验证码校验失败，请稍后重试", externalcall.OperationSMSCodeVerify)
	}
	return errx.New(errx.CodeInternalError, "短信服务供应商尚未接入，请稍后重试")
}

func (v *ConfiguredSMSVerifier) postSMS(ctx context.Context, endpoint string, payload map[string]string, publicError string, operation string) error {
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
	// 请求创建成功才进入外呼观测，避免把本地配置或参数错误误计为供应商故障。
	call := externalcall.Start(v.observer, externalcall.ProviderSMS, operation)
	req.Header.Set("Content-Type", "application/json")
	if secret := strings.TrimSpace(v.cfg.AccessKeySecret); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		call.Finish(ctx, externalcall.ClassifyTransport(ctx, err), 0)
		return errx.New(errx.CodeInternalError, publicError)
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		call.Finish(ctx, externalcall.OutcomeTransportError, statusCode)
		return errx.New(errx.CodeInternalError, publicError)
	}
	if statusCode < 200 || statusCode >= 300 {
		call.Finish(ctx, externalcall.OutcomeHTTPError, statusCode)
		return errx.New(errx.CodeInternalError, publicError)
	}
	var data struct {
		OK    *bool  `json:"ok"`
		Valid *bool  `json:"valid"`
		Error string `json:"error"`
	}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &data); err != nil {
			call.Finish(ctx, externalcall.OutcomeDecodeError, statusCode)
			return errx.New(errx.CodeInternalError, publicError)
		}
		if data.Error != "" {
			call.Finish(ctx, externalcall.OutcomeProviderRejected, statusCode)
			if operation == externalcall.OperationSMSCodeVerify {
				return errx.New(errx.CodeValidationFailed, "短信验证码不正确")
			}
			return errx.New(errx.CodeInternalError, publicError)
		}
		if data.Valid != nil {
			if *data.Valid {
				call.Finish(ctx, externalcall.OutcomeSuccess, statusCode)
				return nil
			}
			call.Finish(ctx, externalcall.OutcomeProviderRejected, statusCode)
			return errx.New(errx.CodeValidationFailed, "短信验证码不正确")
		}
		if data.OK != nil {
			if *data.OK {
				call.Finish(ctx, externalcall.OutcomeSuccess, statusCode)
				return nil
			}
			call.Finish(ctx, externalcall.OutcomeProviderRejected, statusCode)
			return errx.New(errx.CodeInternalError, publicError)
		}
	}
	call.Finish(ctx, externalcall.OutcomeSuccess, statusCode)
	return nil
}
