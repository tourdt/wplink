package resource

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetResourceStore interface {
	GetPublishedResourceDetail(ctx context.Context, resourceID string) (model.ResourceDetail, error)
}

type ResourceContactMasked struct {
	Name         string `json:"name"`
	PhoneMasked  string `json:"phoneMasked"`
	WechatMasked string `json:"wechatMasked,omitempty"`
}

type ResourcePresentationField struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Value  string `json:"value"`
	Layout string `json:"layout"`
}

type ResourcePresentation struct {
	Fields []ResourcePresentationField `json:"fields"`
}

type ResourceContactAccess struct {
	Mode             string `json:"mode"`
	PriceCent        int64  `json:"priceCent"`
	Currency         string `json:"currency"`
	VIPFree          bool   `json:"vipFree"`
	RepeatUnlockDays int64  `json:"repeatUnlockDays"`
	Unlocked         bool   `json:"unlocked"`
	ActionText       string `json:"actionText"`
}

type ResourceDetailResp struct {
	ID            string                `json:"id"`
	Status        string                `json:"status"`
	TypeCode      string                `json:"typeCode"`
	Direction     string                `json:"direction"`
	TypeName      string                `json:"typeName,omitempty"`
	Title         string                `json:"title"`
	Category      string                `json:"category"`
	Description   string                `json:"description"`
	Attributes    model.JSONMap         `json:"attributes"`
	Presentation  ResourcePresentation  `json:"presentation"`
	Tags          []string              `json:"tags"`
	Images        []string              `json:"images"`
	Merchant      ResourceMerchantBrief `json:"merchant"`
	Contact       ResourceContactMasked `json:"contact"`
	ContactAccess ResourceContactAccess `json:"contactAccess"`
	PublishedAt   string                `json:"publishedAt,omitempty"`
	ExpiresAt     string                `json:"expiresAt,omitempty"`
	DealtAt       string                `json:"dealtAt,omitempty"`
}

type resourcePresentationFieldConfig struct {
	Source string
	Label  string
	Role   string
	Layout string
	Order  int64
	Index  int
}

type GetResourceLogic struct {
	store GetResourceStore
}

func NewGetResourceLogic(store GetResourceStore) *GetResourceLogic {
	return &GetResourceLogic{store: store}
}

func (l *GetResourceLogic) GetResource(ctx context.Context, resourceID string) (ResourceDetailResp, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return ResourceDetailResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}

	detail, err := l.store.GetPublishedResourceDetail(ctx, resourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 详情页只展示已发布且未过期资源，查不到时统一按下架/不存在处理，避免把数据库空结果暴露成 500。
			return ResourceDetailResp{}, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
		}
		logx.Errorf("加载公开资源详情失败: resourceId=%s err=%+v", resourceID, err)
		return ResourceDetailResp{}, errx.New(errx.CodeInternalError, "资源详情加载失败，请稍后重试")
	}
	resp, err := resourceDetailRespFromModel(detail)
	if err != nil {
		// 快照配置异常属于服务端数据问题，日志保留资源和类型定位信息，接口仅返回稳定的友好提示。
		logx.Errorf("解析资源详情展示配置失败: resourceId=%s typeCode=%s err=%+v", resourceID, detail.TypeCode, err)
		return ResourceDetailResp{}, errx.New(errx.CodeInternalError, "资源详情展示配置异常，请稍后重试")
	}
	return resp, nil
}

func resourceDetailRespFromModel(detail model.ResourceDetail) (ResourceDetailResp, error) {
	presentation, err := buildResourcePresentation(detail)
	if err != nil {
		return ResourceDetailResp{}, err
	}
	return ResourceDetailResp{
		ID:           detail.ID,
		Status:       detail.Status,
		TypeCode:     detail.TypeCode,
		Direction:    detail.Direction,
		TypeName:     detail.TypeName,
		Title:        detail.Title,
		Category:     detail.Category,
		Description:  detail.Description,
		Attributes:   detail.Attributes,
		Presentation: presentation,
		Tags:         append([]string(nil), detail.Tags...),
		Images:       append([]string(nil), detail.Images...),
		Merchant: ResourceMerchantBrief{
			ID:        detail.MerchantID,
			Name:      detail.MerchantName,
			VIPStatus: normalizeVIPStatus(detail.MerchantVIPStatus),
		},
		Contact: ResourceContactMasked{
			Name:         detail.ContactName,
			PhoneMasked:  detail.PhoneMasked,
			WechatMasked: detail.WechatMasked,
		},
		ContactAccess: contactAccessFromCommercialRules(detail.CommercialRules, false),
		PublishedAt:   detail.PublishedAt,
		ExpiresAt:     detail.ExpiresAt,
		DealtAt:       detail.DealtAt,
	}, nil
}

func contactAccessFromCommercialRules(values model.JSONMap, unlocked bool) ResourceContactAccess {
	rules := model.ContactUnlockRulesFromCommercialRules(values)
	currency := strings.TrimSpace(rules.Currency)
	if currency == "" {
		currency = model.DefaultContactUnlockCurrency
	}
	days := rules.RepeatUnlockDays
	if days <= 0 {
		days = model.DefaultContactRepeatUnlockDays
	}
	mode := strings.TrimSpace(rules.Mode)
	if mode == "" {
		mode = model.ContactUnlockModeLoginFree
	}
	return ResourceContactAccess{
		Mode:             mode,
		PriceCent:        rules.PriceCent,
		Currency:         currency,
		VIPFree:          rules.VIPFree,
		RepeatUnlockDays: days,
		Unlocked:         unlocked,
		ActionText:       contactAccessActionText(mode, rules.PriceCent, rules.VIPFree),
	}
}

func contactAccessActionText(mode string, priceCent int64, vipFree bool) string {
	switch mode {
	case model.ContactUnlockModeDisabled:
		return "暂不开放查看"
	case model.ContactUnlockModePaid:
		if vipFree {
			return "付费或 VIP 免费查看"
		}
		return "付费查看联系方式"
	case model.ContactUnlockModePaidOrVIP:
		return "付费或 VIP 免费查看"
	case model.ContactUnlockModeVIPOnly:
		return "VIP 免费查看"
	default:
		if priceCent > 0 {
			return "查看联系方式"
		}
		return "登录后免费查看"
	}
}

func buildResourcePresentation(detail model.ResourceDetail) (ResourcePresentation, error) {
	configs, err := parseResourcePresentationFieldConfigs(detail.DisplayTemplate["fields"])
	if err != nil {
		return ResourcePresentation{}, err
	}
	sort.SliceStable(configs, func(left int, right int) bool {
		if configs[left].Order == configs[right].Order {
			return configs[left].Index < configs[right].Index
		}
		return configs[left].Order < configs[right].Order
	})

	addressSources := resourceAddressFieldSources(detail.FieldSchema)
	fields := make([]ResourcePresentationField, 0, len(configs))
	seenSources := make(map[string]struct{}, len(configs))
	for _, config := range configs {
		if _, seen := seenSources[config.Source]; seen {
			continue
		}
		seenSources[config.Source] = struct{}{}
		value, exists := resourcePresentationSourceValue(detail, config.Source)
		if !exists || resourceAttributeMissing(value) {
			continue
		}
		// 地址在详情页使用独立地图/导航区块；即使误配进 fields，也不能重复进入普通参数列表。
		if _, isAddress := addressSources[config.Source]; isAddress || isResourceAddressLikeMap(value) {
			continue
		}
		fields = append(fields, ResourcePresentationField{
			Key:    config.Source,
			Label:  config.Label,
			Value:  resourceAttributeDisplayValue(value),
			Layout: config.Layout,
		})
	}
	return ResourcePresentation{Fields: fields}, nil
}

func parseResourcePresentationFieldConfigs(value interface{}) ([]resourcePresentationFieldConfig, error) {
	if value == nil {
		return []resourcePresentationFieldConfig{}, nil
	}
	entries, ok := resourcePresentationConfigList(value)
	if !ok {
		return nil, fmt.Errorf("displayTemplate.fields 不是数组")
	}
	configs := make([]resourcePresentationFieldConfig, 0, len(entries))
	for index, entry := range entries {
		field, ok := resourcePresentationConfigMap(entry)
		if !ok {
			return nil, fmt.Errorf("第 %d 个展示字段不是对象", index+1)
		}
		source, ok := resourcePresentationConfigString(field["source"])
		if !ok || source == "" {
			return nil, fmt.Errorf("第 %d 个展示字段缺少 source", index+1)
		}
		label, ok := resourcePresentationConfigString(field["label"])
		if !ok || label == "" {
			return nil, fmt.Errorf("第 %d 个展示字段缺少 label", index+1)
		}
		role, ok := resourcePresentationConfigString(field["role"])
		if !ok || (role != "core" && role != "core_price" && role != "detail") {
			return nil, fmt.Errorf("第 %d 个展示字段 role 不支持", index+1)
		}
		layout, ok := resourcePresentationConfigString(field["layout"])
		if !ok || (layout != "half" && layout != "full") {
			return nil, fmt.Errorf("第 %d 个展示字段 layout 不支持", index+1)
		}
		order, ok := resourcePresentationConfigOrder(field["order"])
		if !ok {
			return nil, fmt.Errorf("第 %d 个展示字段 order 必须是整数", index+1)
		}
		configs = append(configs, resourcePresentationFieldConfig{Source: source, Label: label, Role: role, Layout: layout, Order: order, Index: index})
	}
	return configs, nil
}

func resourcePresentationSourceValue(detail model.ResourceDetail, source string) (interface{}, bool) {
	if value, ok := detail.Attributes[source]; ok {
		return value, true
	}
	switch source {
	case "merchantId":
		return detail.MerchantID, true
	case "cityCode":
		return detail.CityCode, true
	case "typeCode":
		return detail.TypeCode, true
	case "typeName":
		return detail.TypeName, true
	case "direction":
		return detail.Direction, true
	case "title":
		return detail.Title, true
	case "category":
		return detail.Category, true
	case "district":
		return detail.District, true
	case "quantityText":
		return detail.QuantityText, true
	case "priceText":
		return detail.PriceText, true
	case "description":
		return detail.Description, true
	case "contactName":
		return detail.ContactName, true
	case "tags":
		return detail.Tags, true
	case "images":
		return detail.Images, true
	default:
		return nil, false
	}
}

func resourceAddressFieldSources(fieldSchema model.JSONMap) map[string]struct{} {
	sources := make(map[string]struct{})
	entries, ok := resourcePresentationConfigList(fieldSchema["fields"])
	if !ok {
		return sources
	}
	for _, entry := range entries {
		field, ok := resourcePresentationConfigMap(entry)
		if !ok {
			continue
		}
		fieldType, _ := resourcePresentationConfigString(field["type"])
		if fieldType != "address" {
			continue
		}
		key, _ := resourcePresentationConfigString(field["key"])
		if key != "" {
			sources[key] = struct{}{}
		}
	}
	return sources
}

func resourcePresentationConfigList(value interface{}) ([]interface{}, bool) {
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

func resourcePresentationConfigMap(value interface{}) (map[string]interface{}, bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed, true
	case model.JSONMap:
		return map[string]interface{}(typed), true
	default:
		return nil, false
	}
}

func resourcePresentationConfigString(value interface{}) (string, bool) {
	text, ok := value.(string)
	return strings.TrimSpace(text), ok
}

func resourcePresentationConfigOrder(value interface{}) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || math.Trunc(typed) != typed || typed < math.MinInt64 || typed >= 1<<63 {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

func resourceAttributeDisplayValue(value interface{}) string {
	switch typed := value.(type) {
	case bool:
		if typed {
			return "是"
		}
		return "否"
	case string:
		return strings.TrimSpace(typed)
	case model.JSONMap, map[string]interface{}:
		if text := resourceAddressAttributeText(typed); text != "" {
			return text
		}
		return strings.TrimSpace(fmt.Sprint(typed))
	default:
		return fmt.Sprint(typed)
	}
}
