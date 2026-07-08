package resource

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateResourceStore interface {
	GetMerchantPublishStatus(ctx context.Context, merchantID string) (string, error)
	GetMerchantContactPhone(ctx context.Context, merchantID string) (string, error)
	GetResourcePublishConfig(ctx context.Context, cityCode string, typeCode string) (model.ResourcePublishConfig, error)
	CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error)
	UpdateResourceDraft(ctx context.Context, resourceID string, input model.CreateResourceInput) (model.CreateResourceResult, error)
	RecordOperationLog(ctx context.Context, input model.OperationLogInput) error
}

type ResourceContactReq struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Wechat string `json:"wechat,omitempty"`
}

type CreateResourceReq struct {
	MerchantID    string
	CityCode      string
	TypeCode      string
	Direction     string
	Title         string
	Category      string
	District      string
	PriceText     string
	QuantityText  string
	Description   string
	Attributes    model.JSONMap
	Tags          []string
	Images        []string
	Contact       ResourceContactReq
	CreatedByUser string
	CreatedByRole string
}

type CreateResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type CreateResourceLogic struct {
	store CreateResourceStore
}

func NewCreateResourceLogic(store CreateResourceStore) *CreateResourceLogic {
	return &CreateResourceLogic{store: store}
}

func (l *CreateResourceLogic) CreateResource(ctx context.Context, req CreateResourceReq) (CreateResourceResp, error) {
	return l.create(ctx, req, model.ResourceStatusPending, "已提交审核，审核通过后将展示给买家")
}

func (l *CreateResourceLogic) CreateResourceDraft(ctx context.Context, req CreateResourceReq) (CreateResourceResp, error) {
	return l.create(ctx, req, model.ResourceStatusDraft, "草稿已保存")
}

func (l *CreateResourceLogic) create(ctx context.Context, req CreateResourceReq, status string, message string) (CreateResourceResp, error) {
	input, typeCode, err := l.buildResourceInput(ctx, req, status)
	if err != nil {
		return CreateResourceResp{}, err
	}
	result, err := l.store.CreateResource(ctx, input)
	if err != nil {
		logx.Errorf("创建资源失败: merchantId=%s typeCode=%s targetStatus=%s err=%+v", strings.TrimSpace(req.MerchantID), typeCode, status, err)
		return CreateResourceResp{}, err
	}
	if isOperatorProxy(req.CreatedByRole) && strings.TrimSpace(req.CreatedByUser) != "" {
		if err := l.store.RecordOperationLog(ctx, model.OperationLogInput{
			OperatorID:     strings.TrimSpace(req.CreatedByUser),
			OperatorRole:   strings.TrimSpace(req.CreatedByRole),
			Action:         "proxy_create_resource",
			ObjectType:     "resource",
			ObjectID:       result.ID,
			BeforeSnapshot: model.JSONMap{},
			AfterSnapshot:  model.JSONMap{"status": result.Status, "typeCode": typeCode},
		}); err != nil {
			logx.Errorf("记录代发布资源操作日志失败: operatorId=%s resourceId=%s status=%s err=%+v", strings.TrimSpace(req.CreatedByUser), result.ID, result.Status, err)
			return CreateResourceResp{}, err
		}
	}
	// 新建资源可能直接进入审核队列，也可能只是草稿；日志记录目标状态，避免排查时只看到写入成功但不知道用户路径。
	logx.Infof("创建资源成功: merchantId=%s resourceId=%s typeCode=%s status=%s createdByRole=%s", strings.TrimSpace(req.MerchantID), result.ID, typeCode, result.Status, strings.TrimSpace(req.CreatedByRole))
	return CreateResourceResp{ID: result.ID, Status: result.Status, Message: message}, nil
}

func (l *CreateResourceLogic) UpdateResourceDraft(ctx context.Context, resourceID string, req CreateResourceReq) (CreateResourceResp, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return CreateResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	input, _, err := l.buildResourceInput(ctx, req, model.ResourceStatusDraft)
	if err != nil {
		return CreateResourceResp{}, err
	}
	result, err := l.store.UpdateResourceDraft(ctx, resourceID, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logx.Infof("更新资源草稿被拦截: merchantId=%s resourceId=%s reason=not_editable", strings.TrimSpace(req.MerchantID), resourceID)
			return CreateResourceResp{}, errx.New(errx.CodeStateConflict, "资源不存在或当前状态不可编辑")
		}
		logx.Errorf("更新资源草稿失败: merchantId=%s resourceId=%s err=%+v", strings.TrimSpace(req.MerchantID), resourceID, err)
		return CreateResourceResp{}, err
	}
	logx.Infof("更新资源草稿成功: merchantId=%s resourceId=%s status=%s", strings.TrimSpace(req.MerchantID), result.ID, result.Status)
	return CreateResourceResp{ID: result.ID, Status: result.Status, Message: "草稿已保存，请重新提交审核"}, nil
}

func (l *CreateResourceLogic) buildResourceInput(ctx context.Context, req CreateResourceReq, status string) (model.CreateResourceInput, string, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	typeCode := strings.TrimSpace(req.TypeCode)
	config, err := l.store.GetResourcePublishConfig(ctx, cityCode, typeCode)
	if err != nil {
		return model.CreateResourceInput{}, "", err
	}

	values := map[string]string{
		"merchantId":    strings.TrimSpace(req.MerchantID),
		"cityCode":      cityCode,
		"typeCode":      typeCode,
		"title":         strings.TrimSpace(req.Title),
		"category":      strings.TrimSpace(req.Category),
		"quantityText":  strings.TrimSpace(req.QuantityText),
		"priceText":     strings.TrimSpace(req.PriceText),
		"district":      strings.TrimSpace(req.District),
		"contactName":   strings.TrimSpace(req.Contact.Name),
		"contactPhone":  strings.TrimSpace(req.Contact.Phone),
		"contactWechat": strings.TrimSpace(req.Contact.Wechat),
		"description":   strings.TrimSpace(req.Description),
	}
	if shouldUseMerchantContactPhone(values["contactPhone"]) {
		// 发布页只能从公开商家资料拿到脱敏手机号，保存资源前用商家资料真实电话兜底。
		merchantPhone, err := l.store.GetMerchantContactPhone(ctx, values["merchantId"])
		if err != nil {
			return model.CreateResourceInput{}, "", err
		}
		values["contactPhone"] = strings.TrimSpace(merchantPhone)
	}
	if err := validateResourceRequiredFields(config, values, req.Attributes, req.Tags, req.Images); err != nil {
		return model.CreateResourceInput{}, "", err
	}
	if err := validateResourceDynamicFieldValues(config.FieldSchema, req.Attributes); err != nil {
		return model.CreateResourceInput{}, "", err
	}
	merchantStatus, err := l.store.GetMerchantPublishStatus(ctx, values["merchantId"])
	if err != nil {
		return model.CreateResourceInput{}, "", err
	}
	if merchantStatus != model.MerchantStatusActive {
		return model.CreateResourceInput{}, "", errx.New(errx.CodeValidationFailed, "商家已停用，不能发布资源")
	}

	return model.CreateResourceInput{
		MerchantID:           values["merchantId"],
		CityCode:             cityCode,
		ResourceTypeConfigID: config.ID,
		TypeCode:             typeCode,
		Direction:            normalizeConfigDirection(config.Direction),
		Status:               status,
		Title:                values["title"],
		Category:             values["category"],
		District:             strings.TrimSpace(req.District),
		PriceText:            values["priceText"],
		QuantityText:         values["quantityText"],
		CoverURL:             firstResourceImage(req.Images),
		Description:          values["description"],
		Attributes:           req.Attributes,
		Tags:                 append([]string(nil), req.Tags...),
		Images:               append([]string(nil), req.Images...),
		ContactName:          values["contactName"],
		ContactPhone:         values["contactPhone"],
		ContactWechat:        strings.TrimSpace(req.Contact.Wechat),
		CreatedByUser:        strings.TrimSpace(req.CreatedByUser),
	}, typeCode, nil
}

func normalizeConfigDirection(direction string) string {
	direction = strings.TrimSpace(direction)
	if direction == model.ResourceDirectionDemand {
		return model.ResourceDirectionDemand
	}
	return model.ResourceDirectionSupply
}

func isOperatorProxy(role string) bool {
	role = strings.TrimSpace(role)
	return role == "platform_operator" || role == "super_admin"
}

func firstResourceImage(images []string) string {
	for _, image := range images {
		if trimmed := strings.TrimSpace(image); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func shouldUseMerchantContactPhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	return phone == "" || strings.Contains(phone, "*")
}

var resourceBaseFieldLabels = map[string]string{
	"merchantId":    "商家",
	"cityCode":      "城市站",
	"typeCode":      "资源类型",
	"title":         "标题",
	"category":      "品类",
	"district":      "区域",
	"quantityText":  "数量/产能",
	"priceText":     "价格描述",
	"description":   "资源描述",
	"contactName":   "联系人",
	"contactPhone":  "联系电话",
	"contactWechat": "联系微信",
	"tags":          "资源标签",
	"images":        "资源图片",
}

const maxCustomSelectAttributeLength = 32

type resourceFieldSpec struct {
	Key         string
	Label       string
	Type        string
	Options     []string
	AllowCustom bool
}

func validateResourceRequiredFields(config model.ResourcePublishConfig, values map[string]string, attributes model.JSONMap, tags []string, images []string) error {
	fieldLabels := resourceFieldLabels(config.FieldSchema)
	for key, label := range resourceBaseFieldLabels {
		if _, ok := fieldLabels[key]; !ok {
			fieldLabels[key] = label
		}
	}

	for _, field := range config.RequiredFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		missing := false
		if value, ok := values[field]; ok {
			missing = strings.TrimSpace(value) == ""
		} else if field == "tags" {
			missing = len(tags) == 0
		} else if field == "images" {
			missing = len(images) == 0
		} else {
			missing = resourceAttributeMissing(attributes[field])
		}
		if missing {
			label := fieldLabels[field]
			if label == "" {
				label = "配置字段"
			}
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请补充%s", label))
		}
	}
	return nil
}

func validateResourceDynamicFieldValues(fieldSchema model.JSONMap, attributes model.JSONMap) error {
	for _, field := range resourceFieldSpecs(fieldSchema) {
		if field.Type != "select" || len(field.Options) == 0 {
			continue
		}
		value, ok := attributes[field.Key]
		if !ok || resourceAttributeMissing(value) {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请选择正确的%s", fieldLabelOrKey(field)))
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		attributes[field.Key] = text
		if stringSliceContains(field.Options, text) {
			continue
		}
		if !field.AllowCustom {
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请选择正确的%s", fieldLabelOrKey(field)))
		}
		if !validCustomSelectAttribute(text) {
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请正确填写%s", fieldLabelOrKey(field)))
		}
	}
	return nil
}

func resourceFieldLabels(fieldSchema model.JSONMap) map[string]string {
	labels := make(map[string]string)
	for _, field := range resourceFieldSpecs(fieldSchema) {
		if field.Key != "" && field.Label != "" {
			labels[field.Key] = field.Label
		}
	}
	return labels
}

func resourceFieldSpecs(fieldSchema model.JSONMap) []resourceFieldSpec {
	fields, ok := fieldSchema["fields"].([]interface{})
	if !ok {
		return nil
	}
	specs := make([]resourceFieldSpec, 0, len(fields))
	for _, entry := range fields {
		field, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := field["key"].(string)
		label, _ := field["label"].(string)
		key = strings.TrimSpace(key)
		label = strings.TrimSpace(label)
		if key == "" {
			continue
		}
		fieldType, _ := field["type"].(string)
		allowCustom, _ := field["allowCustom"].(bool)
		specs = append(specs, resourceFieldSpec{
			Key:         key,
			Label:       label,
			Type:        strings.TrimSpace(fieldType),
			Options:     stringOptionsFromInterface(field["options"]),
			AllowCustom: allowCustom,
		})
	}
	return specs
}

func stringOptionsFromInterface(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		options := make([]string, 0, len(typed))
		for _, item := range typed {
			if option := strings.TrimSpace(item); option != "" {
				options = append(options, option)
			}
		}
		return options
	case []interface{}:
		options := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if ok {
				if option := strings.TrimSpace(text); option != "" {
					options = append(options, option)
				}
			}
		}
		return options
	default:
		return nil
	}
}

func stringSliceContains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func validCustomSelectAttribute(value string) bool {
	if len([]rune(value)) > maxCustomSelectAttributeLength {
		return false
	}
	// 自定义选项会进入公开展示和后续运营归并，先拦截控制字符，避免出现不可见内容影响审核和筛选。
	return !strings.ContainsFunc(value, func(r rune) bool {
		return r < 32 || r == 127
	})
}

func fieldLabelOrKey(field resourceFieldSpec) string {
	if field.Label != "" {
		return field.Label
	}
	return field.Key
}

func resourceAttributeMissing(value interface{}) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case bool:
		return false
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return rv.Len() == 0
	}
	return false
}
