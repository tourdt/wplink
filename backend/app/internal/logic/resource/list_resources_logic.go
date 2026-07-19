package resource

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ListResourcesStore interface {
	ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error)
}

type ListResourcesReq struct {
	CityCode     string
	MerchantID   string
	GroupCode    string
	TypeCode     string
	Direction    string
	Keyword      string
	Category     string
	Tags         []string
	VerifiedOnly bool
	Page         int64
	PageSize     int64
}

type ResourceMerchantBrief struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	VerificationStatus string `json:"verificationStatus"`
	VIPStatus          string `json:"vipStatus"`
}

type ResourceListItem struct {
	ID           string                `json:"id"`
	Direction    string                `json:"direction"`
	TypeCode     string                `json:"typeCode"`
	TypeName     string                `json:"typeName,omitempty"`
	Title        string                `json:"title"`
	Category     string                `json:"category"`
	CoverURL     string                `json:"coverUrl,omitempty"`
	District     string                `json:"district,omitempty"`
	PriceText    string                `json:"priceText,omitempty"`
	QuantityText string                `json:"quantityText,omitempty"`
	Tags         []string              `json:"tags"`
	Merchant     ResourceMerchantBrief `json:"merchant"`
	CreditTags   []string              `json:"creditTags"`
	RefreshedAt  string                `json:"refreshedAt,omitempty"`
}

type ListResourcesResp struct {
	Items    []ResourceListItem `json:"items"`
	Page     int64              `json:"page"`
	PageSize int64              `json:"pageSize"`
	Total    int64              `json:"total"`
}

type ListResourcesLogic struct {
	store ListResourcesStore
}

func NewListResourcesLogic(store ListResourcesStore) *ListResourcesLogic {
	return &ListResourcesLogic{store: store}
}

func (l *ListResourcesLogic) ListResources(ctx context.Context, req ListResourcesReq) (ListResourcesResp, error) {
	direction, err := normalizeListDirection(req.Direction)
	if err != nil {
		return ListResourcesResp{}, err
	}
	tags, err := normalizeListResourceTags(req.Tags)
	if err != nil {
		return ListResourcesResp{}, err
	}
	result, err := l.store.ListResources(ctx, model.ListResourcesFilter{
		CityCode:     strings.TrimSpace(req.CityCode),
		MerchantID:   strings.TrimSpace(req.MerchantID),
		GroupCode:    strings.TrimSpace(req.GroupCode),
		TypeCode:     strings.TrimSpace(req.TypeCode),
		Direction:    direction,
		Keyword:      strings.TrimSpace(req.Keyword),
		Category:     strings.TrimSpace(req.Category),
		Tags:         tags,
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
			Direction:    item.Direction,
			TypeCode:     item.TypeCode,
			TypeName:     item.TypeName,
			Title:        item.Title,
			Category:     item.Category,
			CoverURL:     item.CoverURL,
			District:     item.District,
			PriceText:    item.PriceText,
			QuantityText: item.QuantityText,
			Tags:         append([]string(nil), item.Tags...),
			Merchant: ResourceMerchantBrief{
				ID:                 item.Merchant.ID,
				Name:               item.Merchant.Name,
				VerificationStatus: item.Merchant.VerificationStatus,
				VIPStatus:          normalizeVIPStatus(item.Merchant.VIPStatus),
			},
			CreditTags:  append([]string(nil), item.CreditTags...),
			RefreshedAt: item.RefreshedAt,
		})
	}
	return ListResourcesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func normalizeVIPStatus(status string) string {
	if strings.TrimSpace(status) == model.VIPStatusActive {
		return model.VIPStatusActive
	}
	return model.VIPStatusNone
}

func normalizeListDirection(direction string) (string, error) {
	direction = strings.TrimSpace(direction)
	if direction == "" {
		return "", nil
	}
	if direction == model.ResourceDirectionSupply || direction == model.ResourceDirectionDemand {
		return direction, nil
	}
	return "", errx.New(errx.CodeValidationFailed, "请选择正确的供需方向")
}

func normalizeListResourceTags(tags []string) ([]string, error) {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !validResourceTagText(tag) {
			return nil, errx.New(errx.CodeValidationFailed, "请选择正确的筛选标签")
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		if len(normalized) >= maxResourceTagCount {
			return nil, errx.New(errx.CodeValidationFailed, "筛选标签最多选择 8 个")
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized, nil
}
