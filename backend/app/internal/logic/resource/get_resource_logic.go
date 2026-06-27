package resource

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type GetResourceStore interface {
	GetPublishedResourceDetail(ctx context.Context, resourceID string) (model.ResourceDetail, error)
}

type ResourceContactMasked struct {
	Name         string
	PhoneMasked  string
	WechatMasked string
}

type ResourceDetailResp struct {
	ID           string
	Status       string
	TypeCode     string
	Title        string
	Category     string
	Description  string
	PriceText    string
	QuantityText string
	Attributes   model.JSONMap
	Merchant     ResourceMerchantBrief
	Contact      ResourceContactMasked
	PublishedAt  string
	ExpiresAt    string
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
