package resource

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type GetResourceStore interface {
	GetPublishedResourceDetail(ctx context.Context, resourceID string) (model.ResourceDetail, error)
}

type ResourceContactMasked struct {
	Name         string `json:"name"`
	PhoneMasked  string `json:"phoneMasked"`
	WechatMasked string `json:"wechatMasked,omitempty"`
}

type ResourceAttributeItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type ResourceDetailResp struct {
	ID             string                  `json:"id"`
	Status         string                  `json:"status"`
	TypeCode       string                  `json:"typeCode"`
	TypeName       string                  `json:"typeName,omitempty"`
	Title          string                  `json:"title"`
	Category       string                  `json:"category"`
	Description    string                  `json:"description"`
	PriceText      string                  `json:"priceText,omitempty"`
	QuantityText   string                  `json:"quantityText,omitempty"`
	Attributes     model.JSONMap           `json:"attributes"`
	AttributeItems []ResourceAttributeItem `json:"attributeItems"`
	Tags           []string                `json:"tags"`
	Images         []string                `json:"images"`
	Merchant       ResourceMerchantBrief   `json:"merchant"`
	Contact        ResourceContactMasked   `json:"contact"`
	PublishedAt    string                  `json:"publishedAt,omitempty"`
	ExpiresAt      string                  `json:"expiresAt,omitempty"`
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
		return ResourceDetailResp{}, err
	}
	return resourceDetailRespFromModel(detail), nil
}

func resourceDetailRespFromModel(detail model.ResourceDetail) ResourceDetailResp {
	return ResourceDetailResp{
		ID:             detail.ID,
		Status:         detail.Status,
		TypeCode:       detail.TypeCode,
		TypeName:       detail.TypeName,
		Title:          detail.Title,
		Category:       detail.Category,
		Description:    detail.Description,
		PriceText:      detail.PriceText,
		QuantityText:   detail.QuantityText,
		Attributes:     detail.Attributes,
		AttributeItems: buildResourceAttributeItems(detail),
		Tags:           append([]string(nil), detail.Tags...),
		Images:         append([]string(nil), detail.Images...),
		Merchant: ResourceMerchantBrief{
			ID:                 detail.MerchantID,
			Name:               detail.MerchantName,
			VerificationStatus: detail.MerchantVerificationStatus,
		},
		Contact: ResourceContactMasked{
			Name:         detail.ContactName,
			PhoneMasked:  detail.PhoneMasked,
			WechatMasked: detail.WechatMasked,
		},
		PublishedAt: detail.PublishedAt,
		ExpiresAt:   detail.ExpiresAt,
	}
}

func buildResourceAttributeItems(detail model.ResourceDetail) []ResourceAttributeItem {
	labels := resourceFieldLabels(detail.FieldSchema)
	keys := resourceDisplayKeys(detail.DisplayTemplate, detail.FieldSchema)
	items := make([]ResourceAttributeItem, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value := detail.Attributes[key]
		if resourceAttributeMissing(value) {
			continue
		}
		label := labels[key]
		if label == "" {
			label = key
		}
		items = append(items, ResourceAttributeItem{
			Key:   key,
			Label: label,
			Value: resourceAttributeDisplayValue(value),
		})
	}
	return items
}

func resourceDisplayKeys(displayTemplate model.JSONMap, fieldSchema model.JSONMap) []string {
	if keys := stringSliceFromInterface(displayTemplate["detail"]); len(keys) > 0 {
		return keys
	}
	fields, ok := fieldSchema["fields"].([]interface{})
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for _, entry := range fields {
		field, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := field["key"].(string)
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func stringSliceFromInterface(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if ok && strings.TrimSpace(text) != "" {
				items = append(items, text)
			}
		}
		return items
	default:
		return nil
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
	default:
		return fmt.Sprint(typed)
	}
}
