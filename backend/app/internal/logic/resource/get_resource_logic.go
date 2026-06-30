package resource

import (
	"context"
	"database/sql"
	"errors"
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

type ResourceDetailResp struct {
	ID           string                `json:"id"`
	Status       string                `json:"status"`
	TypeCode     string                `json:"typeCode"`
	Title        string                `json:"title"`
	Category     string                `json:"category"`
	Description  string                `json:"description"`
	PriceText    string                `json:"priceText,omitempty"`
	QuantityText string                `json:"quantityText,omitempty"`
	Attributes   model.JSONMap         `json:"attributes"`
	Merchant     ResourceMerchantBrief `json:"merchant"`
	Contact      ResourceContactMasked `json:"contact"`
	PublishedAt  string                `json:"publishedAt,omitempty"`
	ExpiresAt    string                `json:"expiresAt,omitempty"`
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
	return ResourceDetailResp{
		ID:           detail.ID,
		Status:       detail.Status,
		TypeCode:     detail.TypeCode,
		Title:        detail.Title,
		Category:     detail.Category,
		Description:  detail.Description,
		PriceText:    detail.PriceText,
		QuantityText: detail.QuantityText,
		Attributes:   detail.Attributes,
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
	}, nil
}
