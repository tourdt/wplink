package discovery

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/webview"
	"wplink/backend/common/errx"
)

type BannerTopicDiscoveryStore interface {
	ListActiveBannerTopics(ctx context.Context, filter model.BannerTopicFilter) ([]model.BannerTopicConfig, error)
	GetActiveTopic(ctx context.Context, topicID string, cityCode string) (model.BannerTopicConfig, error)
	ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error)
}

type ListHomeBannersReq struct {
	CityCode string
}

type TopicResourcesReq struct {
	TopicID  string
	CityCode string
	Page     int64
	PageSize int64
}

type ValidateWebviewURLReq struct {
	URL string
}

type DiscoveryBannerItem struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Subtitle   string   `json:"subtitle,omitempty"`
	CoverURL   string   `json:"coverUrl,omitempty"`
	JumpType   string   `json:"jumpType"`
	JumpTarget string   `json:"jumpTarget"`
	Tags       []string `json:"tags"`
}

type ListHomeBannersResp struct {
	Items []DiscoveryBannerItem `json:"items"`
}

type TopicInfo struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle,omitempty"`
	CoverURL string   `json:"coverUrl,omitempty"`
	Tags     []string `json:"tags"`
}

type DiscoveryResourceItem struct {
	ID           string `json:"id"`
	TypeCode     string `json:"typeCode"`
	Title        string `json:"title"`
	Category     string `json:"category"`
	District     string `json:"district,omitempty"`
	PriceText    string `json:"priceText,omitempty"`
	QuantityText string `json:"quantityText,omitempty"`
	MerchantName string `json:"merchantName"`
}

type DemandEntry struct {
	Title      string `json:"title"`
	ButtonText string `json:"buttonText"`
}

type TopicResourcesResp struct {
	Topic       TopicInfo               `json:"topic"`
	Items       []DiscoveryResourceItem `json:"items"`
	Page        int64                   `json:"page"`
	PageSize    int64                   `json:"pageSize"`
	Total       int64                   `json:"total"`
	DemandEntry *DemandEntry            `json:"demandEntry,omitempty"`
}

type ValidateWebviewURLResp struct {
	Allowed bool   `json:"allowed"`
	URL     string `json:"url"`
}

type BannerTopicDiscoveryLogic struct {
	store BannerTopicDiscoveryStore
}

func NewBannerTopicDiscoveryLogic(store BannerTopicDiscoveryStore) *BannerTopicDiscoveryLogic {
	return &BannerTopicDiscoveryLogic{store: store}
}

func (l *BannerTopicDiscoveryLogic) ListHomeBanners(ctx context.Context, req ListHomeBannersReq) (ListHomeBannersResp, error) {
	configs, err := l.store.ListActiveBannerTopics(ctx, model.BannerTopicFilter{
		CityCode: strings.TrimSpace(req.CityCode),
		Kind:     "banner",
		Status:   "active",
	})
	if err != nil {
		return ListHomeBannersResp{}, err
	}
	items := make([]DiscoveryBannerItem, 0, len(configs))
	for _, config := range configs {
		jumpTarget := config.JumpTarget
		if config.JumpType == "topic" && strings.TrimSpace(jumpTarget) == "" {
			jumpTarget = config.ID
		}
		items = append(items, DiscoveryBannerItem{
			ID:         config.ID,
			Title:      config.Title,
			Subtitle:   config.Subtitle,
			CoverURL:   config.CoverURL,
			JumpType:   config.JumpType,
			JumpTarget: jumpTarget,
			Tags:       append([]string(nil), config.Tags...),
		})
	}
	return ListHomeBannersResp{Items: items}, nil
}

func (l *BannerTopicDiscoveryLogic) GetTopicResources(ctx context.Context, req TopicResourcesReq) (TopicResourcesResp, error) {
	topicID := strings.TrimSpace(req.TopicID)
	if topicID == "" {
		return TopicResourcesResp{}, errx.New(errx.CodeValidationFailed, "专题不存在")
	}
	topic, err := l.store.GetActiveTopic(ctx, topicID, strings.TrimSpace(req.CityCode))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TopicResourcesResp{}, errx.New(errx.CodeResourceNotFound, "专题不存在或已下线")
		}
		return TopicResourcesResp{}, err
	}

	typeCode := ""
	if len(topic.TypeScope) > 0 {
		typeCode = topic.TypeScope[0]
	}
	result, err := l.store.ListResources(ctx, model.ListResourcesFilter{
		CityCode: strings.TrimSpace(req.CityCode),
		TypeCode: typeCode,
		Status:   model.ResourceStatusPublished,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return TopicResourcesResp{}, err
	}

	items := make([]DiscoveryResourceItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, DiscoveryResourceItem{
			ID:           item.ID,
			TypeCode:     item.TypeCode,
			Title:        item.Title,
			Category:     item.Category,
			District:     item.District,
			PriceText:    item.PriceText,
			QuantityText: item.QuantityText,
			MerchantName: item.Merchant.Name,
		})
	}
	resp := TopicResourcesResp{
		Topic: TopicInfo{
			ID:       topic.ID,
			Title:    topic.Title,
			Subtitle: topic.Subtitle,
			CoverURL: topic.CoverURL,
			Tags:     append([]string(nil), topic.Tags...),
		},
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	}
	if result.Total == 0 {
		resp.DemandEntry = &DemandEntry{Title: "没有找到合适资源", ButtonText: "提交采购需求"}
	}
	return resp, nil
}

func (l *BannerTopicDiscoveryLogic) ValidateWebviewURL(ctx context.Context, req ValidateWebviewURLReq) (ValidateWebviewURLResp, error) {
	rawURL := strings.TrimSpace(req.URL)
	if !webview.IsAllowedURL(rawURL) {
		return ValidateWebviewURLResp{}, errx.New(errx.CodeValidationFailed, "活动链接不在允许访问范围内")
	}
	return ValidateWebviewURLResp{Allowed: true, URL: rawURL}, nil
}
