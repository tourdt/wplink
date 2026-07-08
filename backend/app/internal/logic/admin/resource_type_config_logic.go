package admin

import (
	"context"
	"fmt"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ResourceTypeConfigStore interface {
	ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]model.AdminResourceTypeConfig, error)
	UpdateResourceTypeConfig(ctx context.Context, configID string, patch model.ResourceTypeConfigPatch) (string, error)
}

type ListResourceTypeConfigsReq struct {
	CityCode string
	Status   string
}

type ResourceTypeConfigItem struct {
	ID               string                 `json:"id"`
	CityCode         string                 `json:"cityCode,omitempty"`
	TypeCode         string                 `json:"typeCode"`
	TypeName         string                 `json:"typeName"`
	FieldSchema      map[string]interface{} `json:"fieldSchema"`
	RequiredFields   []string               `json:"requiredFields"`
	FilterFields     []string               `json:"filterFields"`
	DisplayTemplate  map[string]interface{} `json:"displayTemplate"`
	ReviewRules      map[string]interface{} `json:"reviewRules"`
	SortWeights      map[string]interface{} `json:"sortWeights"`
	MessageRules     map[string]interface{} `json:"messageRules"`
	DefaultValidDays int64                  `json:"defaultValidDays"`
	Status           string                 `json:"status"`
}

type ListResourceTypeConfigsResp struct {
	Items []ResourceTypeConfigItem `json:"items"`
}

type UpdateResourceTypeConfigReq struct {
	FieldSchema      map[string]interface{}
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  map[string]interface{}
	ReviewRules      map[string]interface{}
	SortWeights      map[string]interface{}
	MessageRules     map[string]interface{}
	DefaultValidDays int64
	Status           string
}

type UpdateResourceTypeConfigResp struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
}

type ResourceTypeConfigLogic struct {
	store ResourceTypeConfigStore
}

var supportedResourceFieldTypes = map[string]struct{}{
	"text":     {},
	"select":   {},
	"boolean":  {},
	"number":   {},
	"textarea": {},
}

var baseResourceConfigFields = map[string]struct{}{
	"merchantId":    {},
	"cityCode":      {},
	"typeCode":      {},
	"title":         {},
	"category":      {},
	"district":      {},
	"quantityText":  {},
	"priceText":     {},
	"description":   {},
	"contactName":   {},
	"contactPhone":  {},
	"contactWechat": {},
	"images":        {},
	"tags":          {},
}

func NewResourceTypeConfigLogic(store ResourceTypeConfigStore) *ResourceTypeConfigLogic {
	return &ResourceTypeConfigLogic{store: store}
}

func (l *ResourceTypeConfigLogic) ListResourceTypeConfigs(ctx context.Context, req ListResourceTypeConfigsReq) (ListResourceTypeConfigsResp, error) {
	configs, err := l.store.ListResourceTypeConfigs(ctx, strings.TrimSpace(req.CityCode), strings.TrimSpace(req.Status))
	if err != nil {
		return ListResourceTypeConfigsResp{}, err
	}

	items := make([]ResourceTypeConfigItem, 0, len(configs))
	for _, config := range configs {
		items = append(items, ResourceTypeConfigItem{
			ID:               config.ID,
			CityCode:         config.CityCode,
			TypeCode:         config.TypeCode,
			TypeName:         config.TypeName,
			FieldSchema:      map[string]interface{}(config.FieldSchema),
			RequiredFields:   append([]string(nil), config.RequiredFields...),
			FilterFields:     append([]string(nil), config.FilterFields...),
			DisplayTemplate:  map[string]interface{}(config.DisplayTemplate),
			ReviewRules:      map[string]interface{}(config.ReviewRules),
			SortWeights:      map[string]interface{}(config.SortWeights),
			MessageRules:     map[string]interface{}(config.MessageRules),
			DefaultValidDays: config.DefaultValidDays,
			Status:           config.Status,
		})
	}
	return ListResourceTypeConfigsResp{Items: items}, nil
}

func (l *ResourceTypeConfigLogic) UpdateResourceTypeConfig(ctx context.Context, configID string, req UpdateResourceTypeConfigReq) (UpdateResourceTypeConfigResp, error) {
	configID = strings.TrimSpace(configID)
	if configID == "" {
		return UpdateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "资源类型配置不存在")
	}
	if req.DefaultValidDays <= 0 {
		return UpdateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "默认有效期必须大于 0")
	}
	if req.Status != "active" && req.Status != "disabled" {
		return UpdateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "资源类型状态不正确")
	}
	if err := validateResourceTypeConfigPatch(req); err != nil {
		return UpdateResourceTypeConfigResp{}, err
	}

	updatedAt, err := l.store.UpdateResourceTypeConfig(ctx, configID, model.ResourceTypeConfigPatch{
		FieldSchema:      model.JSONMap(req.FieldSchema),
		RequiredFields:   append([]string(nil), req.RequiredFields...),
		FilterFields:     append([]string(nil), req.FilterFields...),
		DisplayTemplate:  model.JSONMap(req.DisplayTemplate),
		ReviewRules:      model.JSONMap(req.ReviewRules),
		SortWeights:      model.JSONMap(req.SortWeights),
		MessageRules:     model.JSONMap(req.MessageRules),
		DefaultValidDays: req.DefaultValidDays,
		Status:           req.Status,
	})
	if err != nil {
		return UpdateResourceTypeConfigResp{}, err
	}
	return UpdateResourceTypeConfigResp{ID: configID, UpdatedAt: updatedAt}, nil
}

func validateResourceTypeConfigPatch(req UpdateResourceTypeConfigReq) error {
	dynamicFields, err := validateResourceFieldSchema(req.FieldSchema)
	if err != nil {
		return err
	}
	allowedFields := make(map[string]struct{}, len(baseResourceConfigFields)+len(dynamicFields))
	for field := range baseResourceConfigFields {
		allowedFields[field] = struct{}{}
	}
	for field := range dynamicFields {
		allowedFields[field] = struct{}{}
	}

	for _, field := range req.RequiredFields {
		if err := validateReferencedConfigField("必填字段", field, allowedFields); err != nil {
			return err
		}
	}
	for _, field := range req.FilterFields {
		if err := validateReferencedConfigField("筛选字段", field, allowedFields); err != nil {
			return err
		}
	}
	for _, field := range stringsFromConfigValue(req.DisplayTemplate["list"]) {
		if err := validateReferencedConfigField("列表展示字段", field, allowedFields); err != nil {
			return err
		}
	}
	for _, field := range stringsFromConfigValue(req.DisplayTemplate["detail"]) {
		if err := validateReferencedConfigField("详情展示字段", field, allowedFields); err != nil {
			return err
		}
	}
	return nil
}

func validateResourceFieldSchema(fieldSchema map[string]interface{}) (map[string]struct{}, error) {
	dynamicFields := make(map[string]struct{})
	if len(fieldSchema) == 0 || fieldSchema["fields"] == nil {
		return dynamicFields, nil
	}
	fields, ok := configList(fieldSchema["fields"])
	if !ok {
		return nil, errx.New(errx.CodeValidationFailed, "字段配置格式不正确")
	}

	for index, entry := range fields {
		field, ok := configMap(entry)
		if !ok {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("第 %d 个字段配置格式不正确", index+1))
		}
		key, err := requiredConfigString(field, "key", fmt.Sprintf("第 %d 个字段", index+1), "字段编码")
		if err != nil {
			return nil, err
		}
		if !validResourceConfigFieldKey(key) {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段编码 %q 只能使用英文字母、数字和下划线，并且必须以字母开头", key))
		}
		if _, exists := dynamicFields[key]; exists {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段编码重复：%s", key))
		}
		if _, exists := baseResourceConfigFields[key]; exists {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段编码 %q 与基础字段冲突，请更换编码", key))
		}
		label, err := requiredConfigString(field, "label", fmt.Sprintf("字段 %s", key), "字段名称")
		if err != nil {
			return nil, err
		}
		fieldType, err := optionalConfigString(field, "type", "text")
		if err != nil {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段 %s 的字段类型不支持", label))
		}
		if _, ok := supportedResourceFieldTypes[fieldType]; !ok {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段 %s 的字段类型不支持", label))
		}
		if err := validateOptionalBoolField(field, "required", label); err != nil {
			return nil, err
		}
		if err := validateOptionalBoolField(field, "filterable", label); err != nil {
			return nil, err
		}
		allowCustom, err := optionalConfigBool(field, "allowCustom")
		if err != nil {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段 %s 的允许手动输入配置不正确", label))
		}
		if err := validateDisplayInConfig(field, label); err != nil {
			return nil, err
		}
		options := stringsFromConfigValue(field["options"])
		if field["options"] != nil && options == nil {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段 %s 的下拉选项格式不正确", label))
		}
		if fieldType == "select" && !allowCustom && len(options) == 0 {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("请为下拉字段配置选项：%s，或开启允许手动输入", label))
		}
		dynamicFields[key] = struct{}{}
	}
	return dynamicFields, nil
}

func requiredConfigString(field map[string]interface{}, key string, fieldName string, label string) (string, error) {
	value, err := optionalConfigString(field, key, "")
	if err != nil || value == "" {
		return "", errx.New(errx.CodeValidationFailed, fmt.Sprintf("%s缺少%s", fieldName, label))
	}
	return value, nil
}

func optionalConfigString(field map[string]interface{}, key string, fallback string) (string, error) {
	value, ok := field[key]
	if !ok || value == nil {
		return fallback, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid string")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback, nil
	}
	return text, nil
}

func validateOptionalBoolField(field map[string]interface{}, key string, label string) error {
	if _, err := optionalConfigBool(field, key); err != nil {
		return errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段 %s 的 %s 配置不正确", label, key))
	}
	return nil
}

func optionalConfigBool(field map[string]interface{}, key string) (bool, error) {
	value, ok := field[key]
	if !ok || value == nil {
		return false, nil
	}
	enabled, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("invalid bool")
	}
	return enabled, nil
}

func validateDisplayInConfig(field map[string]interface{}, label string) error {
	value, exists := field["displayIn"]
	if !exists || value == nil {
		return nil
	}
	displayIn := stringsFromConfigValue(value)
	if displayIn == nil {
		return errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段 %s 的展示位置格式不正确", label))
	}
	for _, item := range displayIn {
		if item != "list" && item != "detail" {
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("字段 %s 的展示位置不正确", label))
		}
	}
	return nil
}

func validateReferencedConfigField(label string, field string, allowedFields map[string]struct{}) error {
	field = strings.TrimSpace(field)
	if field == "" {
		return errx.New(errx.CodeValidationFailed, fmt.Sprintf("%s不能为空", label))
	}
	if _, ok := allowedFields[field]; !ok {
		return errx.New(errx.CodeValidationFailed, fmt.Sprintf("%s %q 未在基础字段或字段配置中定义", label, field))
	}
	return nil
}

func stringsFromConfigValue(value interface{}) []string {
	switch typed := value.(type) {
	case nil:
		return []string{}
	case []string:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				items = append(items, text)
			}
		}
		return items
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil
			}
			if text = strings.TrimSpace(text); text != "" {
				items = append(items, text)
			}
		}
		return items
	default:
		return nil
	}
}

func configList(value interface{}) ([]interface{}, bool) {
	switch typed := value.(type) {
	case []interface{}:
		return typed, true
	case []map[string]interface{}:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items, true
	default:
		return nil, false
	}
}

func configMap(value interface{}) (map[string]interface{}, bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed, true
	case model.JSONMap:
		return map[string]interface{}(typed), true
	default:
		return nil, false
	}
}

func validResourceConfigFieldKey(key string) bool {
	if key == "" || !asciiLetter(key[0]) {
		return false
	}
	for i := 1; i < len(key); i++ {
		if !asciiLetter(key[i]) && !asciiDigit(key[i]) && key[i] != '_' {
			return false
		}
	}
	return true
}

func asciiLetter(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func asciiDigit(char byte) bool {
	return char >= '0' && char <= '9'
}
