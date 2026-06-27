package resource

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type ListResourcesStore interface {
	ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error)
}

type ListResourcesReq struct {
	CityCode     string
	TypeCode     string
	Keyword      string
	Category     string
	VerifiedOnly bool
	Page         int64
	PageSize     int64
}

type ResourceMerchantBrief struct {
	ID                 string
	Name               string
	VerificationStatus string
}

type ResourceListItem struct {
	ID           string
	TypeCode     string
	Title        string
	Category     string
	District     string
	PriceText    string
	QuantityText string
	Merchant     ResourceMerchantBrief
	CreditTags   []string
	RefreshedAt  string
}

type ListResourcesResp struct {
	Items    []ResourceListItem
	Page     int64
	PageSize int64
	Total    int64
}

type ListResourcesLogic struct {
	store ListResourcesStore
}

func NewListResourcesLogic(store ListResourcesStore) *ListResourcesLogic {
	return &ListResourcesLogic{store: store}
}

func (l *ListResourcesLogic) ListResources(ctx context.Context, req ListResourcesReq) (ListResourcesResp, error) {
	result, err := l.store.ListResources(ctx, model.ListResourcesFilter{
		CityCode:     strings.TrimSpace(req.CityCode),
		TypeCode:     strings.TrimSpace(req.TypeCode),
		Keyword:      strings.TrimSpace(req.Keyword),
		Category:     strings.TrimSpace(req.Category),
		VerifiedOnly: req.VerifiedOnly,
		Status:       model.ResourceStatusPublished,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return ListResourcesResp{}, err
	}

	items := make([]ResourceListItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, ResourceListItem{
			ID:           item.ID,
			TypeCode:     item.TypeCode,
			Title:        item.Title,
			Category:     item.Category,
			District:     item.District,
			PriceText:    item.PriceText,
			QuantityText: item.QuantityText,
			Merchant: ResourceMerchantBrief{
				ID:                 item.Merchant.ID,
				Name:               item.Merchant.Name,
				VerificationStatus: item.Merchant.VerificationStatus,
			},
			CreditTags:  append([]string(nil), item.CreditTags...),
			RefreshedAt: item.RefreshedAt,
		})
	}
	return ListResourcesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}
