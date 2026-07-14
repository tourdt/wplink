package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResourceTypeConfigStore interface {
	ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]model.AdminResourceTypeConfig, error)
	CreateResourceTypeConfig(ctx context.Context, input model.CreateResourceTypeConfigInput) (model.CreateResourceTypeConfigResult, error)
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
	Direction        string                 `json:"direction"`
	FieldSchema      map[string]interface{} `json:"fieldSchema"`
	RequiredFields   []string               `json:"requiredFields"`
	FilterFields     []string               `json:"filterFields"`
	DisplayTemplate  map[string]interface{} `json:"displayTemplate"`
	ReviewRules      map[string]interface{} `json:"reviewRules"`
	SortWeights      map[string]interface{} `json:"sortWeights"`
	MessageRules     map[string]interface{} `json:"messageRules"`
	CommercialRules  map[string]interface{} `json:"commercialRules"`
	DefaultValidDays int64                  `json:"defaultValidDays"`
	Status           string                 `json:"status"`
}

type ListResourceTypeConfigsResp struct {
	Items []ResourceTypeConfigItem `json:"items"`
}

type CreateResourceTypeConfigReq struct {
	CityCode         string
	TypeCode         string
	TypeName         string
	Direction        string
	GroupCode        string
	GroupName        string
	GroupSort        int64
	FieldSchema      map[string]interface{}
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  map[string]interface{}
	ReviewRules      map[string]interface{}
	SortWeights      map[string]interface{}
	MessageRules     map[string]interface{}
	CommercialRules  map[string]interface{}
	DefaultValidDays int64
	Status           string
}

type CreateResourceTypeConfigResp struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
}

type UpdateResourceTypeConfigReq struct {
	FieldSchema      map[string]interface{}
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  map[string]interface{}
	ReviewRules      map[string]interface{}
	SortWeights      map[string]interface{}
	MessageRules     map[string]interface{}
	CommercialRules  map[string]interface{}
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

var resourceSummaryTargetFields = map[string]struct{}{
	"category":     {},
	"quantityText": {},
	"priceText":    {},
}

const defaultResourceTypeStatus = "active"

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
			Direction:        config.Direction,
			FieldSchema:      map[string]interface{}(config.FieldSchema),
			RequiredFields:   append([]string(nil), config.RequiredFields...),
			FilterFields:     append([]string(nil), config.FilterFields...),
			DisplayTemplate:  map[string]interface{}(config.DisplayTemplate),
			ReviewRules:      map[string]interface{}(config.ReviewRules),
			SortWeights:      map[string]interface{}(config.SortWeights),
			MessageRules:     map[string]interface{}(config.MessageRules),
			CommercialRules:  map[string]interface{}(config.CommercialRules),
			DefaultValidDays: config.DefaultValidDays,
			Status:           config.Status,
		})
	}
	return ListResourceTypeConfigsResp{Items: items}, nil
}

func (l *ResourceTypeConfigLogic) CreateResourceTypeConfig(ctx context.Context, req CreateResourceTypeConfigReq) (CreateResourceTypeConfigResp, error) {
	input, err := buildCreateResourceTypeConfigInput(req)
	if err != nil {
		return CreateResourceTypeConfigResp{}, err
	}
	result, err := l.store.CreateResourceTypeConfig(ctx, input)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			logx.Infof("创建供需二级类型被拦截: cityCode=%s typeCode=%s reason=city_not_found", input.CityCode, input.TypeCode)
			return CreateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "城市站不存在或未启用")
		}
		if isDuplicateResourceTypeConfigError(err) {
			logx.Infof("创建供需二级类型被拦截: cityCode=%s typeCode=%s reason=duplicate_type_code", input.CityCode, input.TypeCode)
			return CreateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "二级分类编码已存在，请更换编码")
		}
		logx.Errorf("创建供需二级类型失败: cityCode=%s typeCode=%s groupCode=%s err=%+v", input.CityCode, input.TypeCode, groupCodeFromDisplayTemplate(input.DisplayTemplate), err)
		return CreateResourceTypeConfigResp{}, errx.New(errx.CodeInternalError, "新增供需类型失败，请稍后重试")
	}
	logx.Infof("创建供需二级类型成功: cityCode=%s typeCode=%s groupCode=%s configId=%s", input.CityCode, input.TypeCode, groupCodeFromDisplayTemplate(input.DisplayTemplate), result.ID)
	return CreateResourceTypeConfigResp{ID: result.ID, UpdatedAt: result.UpdatedAt}, nil
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
	commercialRules, err := normalizeCommercialRules(req.CommercialRules)
	if err != nil {
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
		CommercialRules:  commercialRules,
		DefaultValidDays: req.DefaultValidDays,
		Status:           req.Status,
	})
	if err != nil {
		return UpdateResourceTypeConfigResp{}, err
	}
	return UpdateResourceTypeConfigResp{ID: configID, UpdatedAt: updatedAt}, nil
}

func buildCreateResourceTypeConfigInput(req CreateResourceTypeConfigReq) (model.CreateResourceTypeConfigInput, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	typeCode := strings.TrimSpace(req.TypeCode)
	typeName := strings.TrimSpace(req.TypeName)
	groupCode := strings.TrimSpace(req.GroupCode)
	groupName := strings.TrimSpace(req.GroupName)
	if cityCode == "" {
		return model.CreateResourceTypeConfigInput{}, errx.New(errx.CodeValidationFailed, "请选择城市站")
	}
	if typeCode == "" || !validResourceConfigFieldKey(typeCode) {
		return model.CreateResourceTypeConfigInput{}, errx.New(errx.CodeValidationFailed, "请填写正确的二级分类编码，只能使用英文字母、数字和下划线，并且必须以字母开头")
	}
	if typeName == "" {
		return model.CreateResourceTypeConfigInput{}, errx.New(errx.CodeValidationFailed, "请填写二级分类名称")
	}
	if groupCode == "" || groupName == "" {
		return model.CreateResourceTypeConfigInput{}, errx.New(errx.CodeValidationFailed, "请选择一级分类")
	}
	if !validResourceConfigFieldKey(groupCode) {
		return model.CreateResourceTypeConfigInput{}, errx.New(errx.CodeValidationFailed, "请填写正确的一级分类编码，只能使用英文字母、数字和下划线，并且必须以字母开头")
	}
	direction, err := normalizeRequiredResourceDirection(req.Direction)
	if err != nil {
		return model.CreateResourceTypeConfigInput{}, err
	}
	defaultValidDays := req.DefaultValidDays
	if defaultValidDays <= 0 {
		defaultValidDays = 15
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = defaultResourceTypeStatus
	}
	if status != "active" && status != "disabled" {
		return model.CreateResourceTypeConfigInput{}, errx.New(errx.CodeValidationFailed, "资源类型状态不正确")
	}

	fieldSchema := cloneConfigMap(req.FieldSchema)
	requiredFields := normalizeRequiredFieldsForCreate(req.RequiredFields)
	filterFields := append([]string(nil), req.FilterFields...)
	displayTemplate := cloneConfigMap(req.DisplayTemplate)
	commercialRules, err := normalizeCommercialRules(req.CommercialRules)
	if err != nil {
		return model.CreateResourceTypeConfigInput{}, err
	}
	// 一级分类不是独立表，必须写入 display_template.group，供小程序和后台统一按该字段分组。
	displayTemplate["group"] = map[string]interface{}{"code": groupCode, "name": groupName, "sort": req.GroupSort}
	if displayTemplate["summary"] == nil {
		displayTemplate["summary"] = map[string]interface{}{"category": "category", "quantityText": "quantityText", "priceText": "priceText"}
	}
	if displayTemplate["list"] == nil {
		displayTemplate["list"] = []interface{}{"priceText", "quantityText", "district"}
	}
	if displayTemplate["detail"] == nil {
		displayTemplate["detail"] = []interface{}{}
	}

	patch := UpdateResourceTypeConfigReq{
		FieldSchema:      fieldSchema,
		RequiredFields:   requiredFields,
		FilterFields:     filterFields,
		DisplayTemplate:  displayTemplate,
		ReviewRules:      cloneConfigMap(req.ReviewRules),
		SortWeights:      cloneConfigMap(req.SortWeights),
		MessageRules:     cloneConfigMap(req.MessageRules),
		CommercialRules:  map[string]interface{}(commercialRules),
		DefaultValidDays: defaultValidDays,
		Status:           status,
	}
	if err := validateResourceTypeConfigPatch(patch); err != nil {
		return model.CreateResourceTypeConfigInput{}, err
	}

	return model.CreateResourceTypeConfigInput{
		CityCode:         cityCode,
		TypeCode:         typeCode,
		TypeName:         typeName,
		Direction:        direction,
		FieldSchema:      model.JSONMap(fieldSchema),
		RequiredFields:   requiredFields,
		FilterFields:     filterFields,
		DisplayTemplate:  model.JSONMap(displayTemplate),
		ReviewRules:      model.JSONMap(patch.ReviewRules),
		SortWeights:      model.JSONMap(patch.SortWeights),
		MessageRules:     model.JSONMap(patch.MessageRules),
		CommercialRules:  commercialRules,
		DefaultValidDays: defaultValidDays,
		Status:           status,
	}, nil
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
	if err := validateDisplaySummaryTemplate(req.DisplayTemplate["summary"], allowedFields); err != nil {
		return err
	}
	return nil
}

func normalizeCommercialRules(value map[string]interface{}) (model.JSONMap, error) {
	if len(value) == 0 {
		return model.DefaultCommercialRules(), nil
	}
	rules := model.CommercialRulesFromJSON(model.JSONMap(value))

	switch strings.TrimSpace(rules.Publish.Mode) {
	case model.ResourcePublishModeConsumeQuota, model.ResourcePublishModeFree, model.ResourcePublishModeDisabled:
	default:
		return nil, errx.New(errx.CodeValidationFailed, "发布策略不正确")
	}

	switch strings.TrimSpace(rules.ContactUnlock.Mode) {
	case model.ContactUnlockModeLoginFree, model.ContactUnlockModePaid, model.ContactUnlockModePaidOrVIP, model.ContactUnlockModeVIPOnly, model.ContactUnlockModeDisabled:
	default:
		return nil, errx.New(errx.CodeValidationFailed, "联系方式查看策略不正确")
	}
	if rules.ContactUnlock.Currency != model.DefaultContactUnlockCurrency {
		return nil, errx.New(errx.CodeValidationFailed, "联系方式查看暂只支持人民币")
	}
	if rules.ContactUnlock.RepeatUnlockDays < 1 || rules.ContactUnlock.RepeatUnlockDays > 365 {
		return nil, errx.New(errx.CodeValidationFailed, "重复查看有效期必须在 1 到 365 天之间")
	}
	if contactUnlockRequiresPrice(rules.ContactUnlock.Mode) && rules.ContactUnlock.PriceCent <= 0 {
		return nil, errx.New(errx.CodeValidationFailed, "请设置正确的联系方式查看价格")
	}
	if !contactUnlockRequiresPrice(rules.ContactUnlock.Mode) && rules.ContactUnlock.PriceCent < 0 {
		return nil, errx.New(errx.CodeValidationFailed, "联系方式查看价格不能小于 0")
	}
	if rules.ContactUnlock.Mode == model.ContactUnlockModePaidOrVIP || rules.ContactUnlock.Mode == model.ContactUnlockModeVIPOnly {
		rules.ContactUnlock.VIPFree = true
	}
	return rules.ToJSONMap(), nil
}

func contactUnlockRequiresPrice(mode string) bool {
	return mode == model.ContactUnlockModePaid || mode == model.ContactUnlockModePaidOrVIP
}

func normalizeRequiredResourceDirection(direction string) (string, error) {
	direction = strings.TrimSpace(direction)
	if direction == model.ResourceDirectionSupply || direction == model.ResourceDirectionDemand {
		return direction, nil
	}
	return "", errx.New(errx.CodeValidationFailed, "请选择正确的供需方向")
}

func normalizeRequiredFieldsForCreate(fields []string) []string {
	normalized := stringsFromConfigValue(fields)
	if len(normalized) == 0 {
		return []string{"title", "contactPhone"}
	}
	return normalized
}

func cloneConfigMap(value map[string]interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	cloned := make(map[string]interface{}, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func groupCodeFromDisplayTemplate(displayTemplate model.JSONMap) string {
	group, ok := configMap(displayTemplate["group"])
	if !ok {
		return ""
	}
	code, _ := group["code"].(string)
	return strings.TrimSpace(code)
}

func isDuplicateResourceTypeConfigError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505")
}

func validateDisplaySummaryTemplate(value interface{}, allowedFields map[string]struct{}) error {
	if value == nil {
		return nil
	}
	summary, ok := configMap(value)
	if !ok {
		return errx.New(errx.CodeValidationFailed, "摘要字段配置格式不正确")
	}
	for target, rawSource := range summary {
		target = strings.TrimSpace(target)
		if _, ok := resourceSummaryTargetFields[target]; !ok {
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("摘要字段 %q 不支持，请使用 category、quantityText 或 priceText", target))
		}
		source, ok := rawSource.(string)
		if !ok {
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("摘要字段 %s 的来源字段格式不正确", target))
		}
		if err := validateReferencedConfigField(fmt.Sprintf("摘要字段 %s", target), source, allowedFields); err != nil {
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
