package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

const BusinessProfileSchemaVersion = 1

type BusinessProfile struct {
	SchemaVersion     int      `json:"schemaVersion"`
	EntityKind        string   `json:"entityKind"`
	PrimaryRole       string   `json:"primaryRole"`
	Roles             []string `json:"roles"`
	ServiceArea       string   `json:"serviceArea"`
	ProcessingMethods []string `json:"processingMethods"`
	PropertyTypes     []string `json:"propertyTypes"`
	ServiceItems      []string `json:"serviceItems"`
}

// EmptyBusinessProfile 是新账号的明确初始态；身份空白表示待填写，不从旧类型推断。
func EmptyBusinessProfile() BusinessProfile {
	return BusinessProfile{SchemaVersion: BusinessProfileSchemaVersion, Roles: []string{}, ProcessingMethods: []string{}, PropertyTypes: []string{}, ServiceItems: []string{}}
}

// MerchantValidationError 只保存可向用户展示的中文校验原因，持久层可在事务中返回该错误。
type MerchantValidationError struct{ Message string }

func (e *MerchantValidationError) Error() string { return e.Message }

func businessProfileError(message string) error { return &MerchantValidationError{Message: message} }

func validBusinessRole(role string) bool {
	switch role {
	case "stall", "factory", "landlord", "service_provider", "supplier":
		return true
	default:
		return false
	}
}

func DecodeBusinessProfile(raw []byte) (BusinessProfile, error) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "{}" {
		return BusinessProfile{}, errors.New("业务档案缺少版本与初始化结构")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return BusinessProfile{}, err
	}
	for _, key := range []string{"schemaVersion", "entityKind", "primaryRole", "roles", "serviceArea", "processingMethods", "propertyTypes", "serviceItems"} {
		value, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return BusinessProfile{}, businessProfileError("业务档案结构不完整，请重新填写")
		}
	}
	var profile BusinessProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return BusinessProfile{}, err
	}
	if profile.SchemaVersion != BusinessProfileSchemaVersion {
		return BusinessProfile{}, businessProfileError("业务档案版本暂不支持，请更新后重试")
	}
	if profile.PrimaryRole == "" {
		if profile.EntityKind != "" || len(profile.Roles) != 0 || profile.ServiceArea != "" || len(profile.ProcessingMethods) != 0 || len(profile.PropertyTypes) != 0 || len(profile.ServiceItems) != 0 {
			return BusinessProfile{}, errors.New("未完善业务档案状态不正确")
		}
		return EmptyBusinessProfile(), nil
	}
	var err error
	profile, err = NormalizeBusinessProfile(profile)
	if err != nil {
		return BusinessProfile{}, err
	}
	return profile, nil
}

func NormalizeBusinessProfile(input BusinessProfile) (BusinessProfile, error) {
	if input.SchemaVersion != BusinessProfileSchemaVersion {
		return BusinessProfile{}, businessProfileError("业务档案版本暂不支持，请更新后重试")
	}
	input.EntityKind = strings.TrimSpace(input.EntityKind)
	if input.EntityKind != "individual" && input.EntityKind != "organization" {
		return BusinessProfile{}, businessProfileError("请选择个人或企业机构")
	}
	input.PrimaryRole = strings.TrimSpace(input.PrimaryRole)
	if !validBusinessRole(input.PrimaryRole) {
		return BusinessProfile{}, businessProfileError("请选择正确的主要业务")
	}
	var err error
	input.Roles, err = normalizeBusinessItems(input.Roles, 5, 32, true)
	if err != nil {
		return BusinessProfile{}, businessProfileError("兼营业务填写不正确")
	}
	if len(input.Roles) == 0 {
		return BusinessProfile{}, businessProfileError("请至少选择一项业务")
	}
	primaryFound := false
	for _, role := range input.Roles {
		if !validBusinessRole(role) {
			return BusinessProfile{}, businessProfileError("业务类型不正确，请重新选择")
		}
		if role == input.PrimaryRole {
			primaryFound = true
		}
	}
	if !primaryFound {
		return BusinessProfile{}, businessProfileError("兼营业务需要包含主要业务")
	}
	input.ServiceArea = strings.TrimSpace(input.ServiceArea)
	if utf8.RuneCountInString(input.ServiceArea) > 200 {
		return BusinessProfile{}, businessProfileError("服务区域不能超过200字")
	}
	input.ProcessingMethods, err = normalizeBusinessItems(input.ProcessingMethods, 20, 50, false)
	if err != nil {
		return BusinessProfile{}, businessProfileError("加工工艺最多20项，每项不超过50字")
	}
	input.PropertyTypes, err = normalizeBusinessItems(input.PropertyTypes, 20, 50, false)
	if err != nil {
		return BusinessProfile{}, businessProfileError("出租类型最多20项，每项不超过50字")
	}
	input.ServiceItems, err = normalizeBusinessItems(input.ServiceItems, 20, 50, false)
	if err != nil {
		return BusinessProfile{}, businessProfileError("服务项目最多20项，每项不超过50字")
	}
	return input, nil
}

func normalizeBusinessItems(values []string, maxCount, maxLength int, rejectDuplicates bool) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if utf8.RuneCountInString(value) > maxLength {
			return nil, businessProfileError("填写内容过长")
		}
		if _, exists := seen[value]; exists {
			if rejectDuplicates {
				return nil, businessProfileError("业务类型不能重复")
			}
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
		if len(result) > maxCount {
			return nil, businessProfileError("填写项目过多")
		}
	}
	return result, nil
}

func ValidateBusinessProfileComplete(profile BusinessProfile, name string, categories []string) error {
	if strings.TrimSpace(name) == "" {
		return businessProfileError("请填写主体名称")
	}
	categoryRequired := false
	for _, role := range profile.Roles {
		switch role {
		case "stall", "factory", "supplier":
			categoryRequired = true
		case "landlord":
			if len(profile.PropertyTypes) == 0 {
				return businessProfileError("请填写出租类型")
			}
		case "service_provider":
			if len(profile.ServiceItems) == 0 {
				return businessProfileError("请填写服务项目")
			}
		}
	}
	if categoryRequired {
		items, err := normalizeBusinessItems(categories, 20, 50, false)
		if err != nil {
			return businessProfileError("主营品类最多20项，每项不超过50字")
		}
		if len(items) == 0 {
			return businessProfileError("请至少选择一项经营品类")
		}
	}
	for _, role := range profile.Roles {
		if role == "factory" && len(profile.ProcessingMethods) == 0 {
			return businessProfileError("请填写加工工艺")
		}
	}
	return nil
}
