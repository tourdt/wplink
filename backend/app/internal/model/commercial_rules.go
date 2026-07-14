package model

import "strings"

const (
	ResourcePublishModeConsumeQuota = "consume_quota"
	ResourcePublishModeFree         = "free"
	ResourcePublishModeDisabled     = "disabled"

	ContactUnlockModeLoginFree = "login_free"
	ContactUnlockModePaid      = "paid"
	ContactUnlockModePaidOrVIP = "paid_or_vip"
	ContactUnlockModeVIPOnly   = "vip_only"
	ContactUnlockModeDisabled  = "disabled"

	DefaultContactUnlockCurrency   = "CNY"
	DefaultContactRepeatUnlockDays = int64(30)
)

type ResourceCommercialRules struct {
	Publish       ResourcePublishRules
	ContactUnlock ContactUnlockRules
}

type ResourcePublishRules struct {
	Mode string
}

type ContactUnlockRules struct {
	Mode             string
	PriceCent        int64
	Currency         string
	VIPFree          bool
	RepeatUnlockDays int64
}

func DefaultCommercialRules() JSONMap {
	return ResourceCommercialRules{
		Publish:       ResourcePublishRules{Mode: ResourcePublishModeConsumeQuota},
		ContactUnlock: defaultContactUnlockRules(),
	}.ToJSONMap()
}

func CommercialRulesFromJSON(values JSONMap) ResourceCommercialRules {
	rules := ResourceCommercialRules{
		Publish:       ResourcePublishRules{Mode: ResourcePublishModeConsumeQuota},
		ContactUnlock: defaultContactUnlockRules(),
	}
	if values == nil {
		return rules
	}
	if publish, ok := jsonObject(values["publish"]); ok {
		if mode := strings.TrimSpace(stringFromJSON(publish["mode"])); mode != "" {
			rules.Publish.Mode = mode
		}
	}
	if contactUnlock, ok := jsonObject(values["contactUnlock"]); ok {
		if mode := strings.TrimSpace(stringFromJSON(contactUnlock["mode"])); mode != "" {
			rules.ContactUnlock.Mode = mode
		}
		rules.ContactUnlock.PriceCent = int64FromJSON(contactUnlock["priceCent"])
		if currency := strings.TrimSpace(stringFromJSON(contactUnlock["currency"])); currency != "" {
			rules.ContactUnlock.Currency = currency
		}
		rules.ContactUnlock.VIPFree = boolFromJSON(contactUnlock["vipFree"])
		if days := int64FromJSON(contactUnlock["repeatUnlockDays"]); days > 0 {
			rules.ContactUnlock.RepeatUnlockDays = days
		}
	}
	return rules
}

func (r ResourceCommercialRules) ToJSONMap() JSONMap {
	publishMode := strings.TrimSpace(r.Publish.Mode)
	if publishMode == "" {
		publishMode = ResourcePublishModeConsumeQuota
	}
	contact := r.ContactUnlock
	if strings.TrimSpace(contact.Mode) == "" {
		contact = defaultContactUnlockRules()
	}
	if strings.TrimSpace(contact.Currency) == "" {
		contact.Currency = DefaultContactUnlockCurrency
	}
	if contact.RepeatUnlockDays <= 0 {
		contact.RepeatUnlockDays = DefaultContactRepeatUnlockDays
	}
	return JSONMap{
		"publish": JSONMap{
			"mode": publishMode,
		},
		"contactUnlock": JSONMap{
			"mode":             strings.TrimSpace(contact.Mode),
			"priceCent":        contact.PriceCent,
			"currency":         strings.TrimSpace(contact.Currency),
			"vipFree":          contact.VIPFree,
			"repeatUnlockDays": contact.RepeatUnlockDays,
		},
	}
}

func PublishModeFromCommercialRules(values JSONMap) string {
	return CommercialRulesFromJSON(values).Publish.Mode
}

func ContactUnlockRulesFromCommercialRules(values JSONMap) ContactUnlockRules {
	return CommercialRulesFromJSON(values).ContactUnlock
}

func defaultContactUnlockRules() ContactUnlockRules {
	return ContactUnlockRules{
		Mode:             ContactUnlockModeLoginFree,
		PriceCent:        0,
		Currency:         DefaultContactUnlockCurrency,
		VIPFree:          false,
		RepeatUnlockDays: DefaultContactRepeatUnlockDays,
	}
}

func jsonObject(value interface{}) (JSONMap, bool) {
	switch typed := value.(type) {
	case JSONMap:
		return typed, true
	case map[string]interface{}:
		return JSONMap(typed), true
	default:
		return nil, false
	}
}
