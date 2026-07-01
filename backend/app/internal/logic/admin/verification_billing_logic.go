package admin

import (
	"context"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

const defaultBillingCityCode = "zhili"

type VerificationBillingConfigStore interface {
	GetVerificationBillingConfig(ctx context.Context, cityCode string) (model.VerificationBillingConfig, error)
	UpdateVerificationBillingConfig(ctx context.Context, input model.VerificationBillingConfig) (model.VerificationBillingConfig, error)
}

type GetVerificationBillingConfigReq struct {
	CityCode string
}

type UpdateVerificationBillingConfigReq struct {
	CityCode      string
	ChargeEnabled bool
	FeeAmount     int64
	Currency      string
	FreeEnabled   bool
	FreeStartAt   string
	FreeEndAt     string
	Notice        string
}

type VerificationBillingConfigResp struct {
	CityCode      string `json:"cityCode"`
	ChargeEnabled bool   `json:"chargeEnabled"`
	FeeAmount     int64  `json:"feeAmount"`
	Currency      string `json:"currency"`
	FreeEnabled   bool   `json:"freeEnabled"`
	FreeStartAt   string `json:"freeStartAt,omitempty"`
	FreeEndAt     string `json:"freeEndAt,omitempty"`
	Notice        string `json:"notice,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

type VerificationBillingConfigLogic struct {
	store VerificationBillingConfigStore
}

func NewVerificationBillingConfigLogic(store VerificationBillingConfigStore) *VerificationBillingConfigLogic {
	return &VerificationBillingConfigLogic{store: store}
}

func (l *VerificationBillingConfigLogic) GetVerificationBillingConfig(ctx context.Context, req GetVerificationBillingConfigReq) (VerificationBillingConfigResp, error) {
	config, err := l.store.GetVerificationBillingConfig(ctx, normalizedBillingCityCode(req.CityCode))
	if err != nil {
		return VerificationBillingConfigResp{}, err
	}
	return verificationBillingConfigResp(config), nil
}

func (l *VerificationBillingConfigLogic) UpdateVerificationBillingConfig(ctx context.Context, req UpdateVerificationBillingConfigReq) (VerificationBillingConfigResp, error) {
	input := model.VerificationBillingConfig{
		CityCode:      normalizedBillingCityCode(req.CityCode),
		ChargeEnabled: req.ChargeEnabled,
		FeeAmount:     req.FeeAmount,
		Currency:      strings.TrimSpace(req.Currency),
		FreeEnabled:   req.FreeEnabled,
		FreeStartAt:   strings.TrimSpace(req.FreeStartAt),
		FreeEndAt:     strings.TrimSpace(req.FreeEndAt),
		Notice:        strings.TrimSpace(req.Notice),
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	if input.FeeAmount < 0 {
		return VerificationBillingConfigResp{}, errx.New(errx.CodeValidationFailed, "认证费用不能小于 0")
	}
	if input.ChargeEnabled && input.FeeAmount == 0 {
		return VerificationBillingConfigResp{}, errx.New(errx.CodeValidationFailed, "开启收费时请填写认证费用")
	}
	if input.Currency != "CNY" {
		return VerificationBillingConfigResp{}, errx.New(errx.CodeValidationFailed, "当前仅支持人民币收费")
	}
	if err := validateFreeWindow(input.FreeStartAt, input.FreeEndAt); err != nil {
		return VerificationBillingConfigResp{}, err
	}
	config, err := l.store.UpdateVerificationBillingConfig(ctx, input)
	if err != nil {
		return VerificationBillingConfigResp{}, err
	}
	return verificationBillingConfigResp(config), nil
}

func normalizedBillingCityCode(cityCode string) string {
	cityCode = strings.TrimSpace(cityCode)
	if cityCode == "" {
		return defaultBillingCityCode
	}
	return cityCode
}

func validateFreeWindow(startValue string, endValue string) error {
	if startValue == "" || endValue == "" {
		return nil
	}
	start, err := time.Parse(time.RFC3339, startValue)
	if err != nil {
		return errx.New(errx.CodeValidationFailed, "限免开始时间格式不正确")
	}
	end, err := time.Parse(time.RFC3339, endValue)
	if err != nil {
		return errx.New(errx.CodeValidationFailed, "限免结束时间格式不正确")
	}
	if end.Before(start) {
		return errx.New(errx.CodeValidationFailed, "限免结束时间不能早于开始时间")
	}
	return nil
}

func verificationBillingConfigResp(config model.VerificationBillingConfig) VerificationBillingConfigResp {
	currency := config.Currency
	if currency == "" {
		currency = "CNY"
	}
	return VerificationBillingConfigResp{
		CityCode:      config.CityCode,
		ChargeEnabled: config.ChargeEnabled,
		FeeAmount:     config.FeeAmount,
		Currency:      currency,
		FreeEnabled:   config.FreeEnabled,
		FreeStartAt:   config.FreeStartAt,
		FreeEndAt:     config.FreeEndAt,
		Notice:        config.Notice,
		UpdatedAt:     config.UpdatedAt,
	}
}
