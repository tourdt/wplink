package model

import (
	"fmt"
	"time"
)

const (
	VerificationStatusNone           = "none"
	VerificationStatusPending        = "pending"
	VerificationStatusVerified       = "verified"
	VerificationStatusRejected       = "rejected"
	VerificationStatusRevoked        = "revoked"
	VerificationStatusPaymentPending = "payment_pending"

	PaymentOrderStatusPending = "pending"
	PaymentOrderStatusPaid    = "paid"
	PaymentOrderStatusClosed  = "closed"
)

type VerificationBillingConfig struct {
	CityCode      string `json:"cityCode,omitempty"`
	ChargeEnabled bool   `json:"chargeEnabled"`
	FeeAmount     int64  `json:"feeAmount"`
	Currency      string `json:"currency"`
	FreeEnabled   bool   `json:"freeEnabled"`
	FreeStartAt   string `json:"freeStartAt,omitempty"`
	FreeEndAt     string `json:"freeEndAt,omitempty"`
	Notice        string `json:"notice,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

func DefaultVerificationBillingConfig(cityCode string) VerificationBillingConfig {
	return VerificationBillingConfig{
		CityCode:  cityCode,
		Currency:  "CNY",
		FeeAmount: 0,
	}
}

func VerificationBillingConfigFromJSON(cityCode string, values JSONMap) VerificationBillingConfig {
	config := DefaultVerificationBillingConfig(cityCode)
	config.ChargeEnabled = boolFromJSON(values["chargeEnabled"])
	config.FeeAmount = int64FromJSON(values["feeAmount"])
	config.Currency = stringFromJSON(values["currency"])
	if config.Currency == "" {
		config.Currency = "CNY"
	}
	config.FreeEnabled = boolFromJSON(values["freeEnabled"])
	config.FreeStartAt = stringFromJSON(values["freeStartAt"])
	config.FreeEndAt = stringFromJSON(values["freeEndAt"])
	config.Notice = stringFromJSON(values["notice"])
	config.UpdatedAt = stringFromJSON(values["updatedAt"])
	return config
}

func (c VerificationBillingConfig) ToJSONMap() JSONMap {
	currency := c.Currency
	if currency == "" {
		currency = "CNY"
	}
	return JSONMap{
		"chargeEnabled": c.ChargeEnabled,
		"feeAmount":     c.FeeAmount,
		"currency":      currency,
		"freeEnabled":   c.FreeEnabled,
		"freeStartAt":   c.FreeStartAt,
		"freeEndAt":     c.FreeEndAt,
		"notice":        c.Notice,
		"updatedAt":     c.UpdatedAt,
	}
}

func (c VerificationBillingConfig) RequiresOnlinePayment(now time.Time) bool {
	if !c.ChargeEnabled || c.FeeAmount <= 0 {
		return false
	}
	if c.IsFreeNow(now) {
		return false
	}
	return true
}

func (c VerificationBillingConfig) IsFreeNow(now time.Time) bool {
	if !c.FreeEnabled {
		return false
	}
	start, hasStart := parseBillingTime(c.FreeStartAt)
	end, hasEnd := parseBillingTime(c.FreeEndAt)
	if hasStart && now.Before(start) {
		return false
	}
	if hasEnd && now.After(end) {
		return false
	}
	return true
}

func parseBillingTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func boolFromJSON(value interface{}) bool {
	result, _ := value.(bool)
	return result
}

func stringFromJSON(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func int64FromJSON(value interface{}) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		var parsed int64
		_, _ = fmt.Sscanf(typed, "%d", &parsed)
		return parsed
	default:
		return 0
	}
}

type VerificationPaymentContext struct {
	VerificationID string
	MerchantID     string
	UserID         string
	OpenID         string
	Status         string
	Billing        VerificationBillingConfig
}

type GetVerificationPaymentContextInput struct {
	MerchantID     string
	VerificationID string
	UserID         string
}

type CreateVerificationPaymentOrderInput struct {
	VerificationID string
	MerchantID     string
	UserID         string
	OpenID         string
	AmountTotal    int64
	Currency       string
}

type VerificationPaymentOrder struct {
	ID          string
	OutTradeNo  string
	AmountTotal int64
	Currency    string
	Status      string
}

type MarkVerificationPaymentPaidInput struct {
	OutTradeNo    string
	TransactionID string
	AmountTotal   int64
	SuccessTime   string
	NotifyPayload JSONMap
}

type VerificationPaymentResult struct {
	OrderID        string
	VerificationID string
	MerchantID     string
	Status         string
}
