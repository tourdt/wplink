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
	ID         string
	Title      string
	Subtitle   string
	CoverURL   string
	JumpType   string
	JumpTarget string
	Tags       []string
}

type ListHomeBannersResp struct {
	Items []DiscoveryBannerItem
}

type TopicInfo struct {
	ID       string
	Title    string
	Subtitle string
	CoverURL string
	Tags     []string
}

type DiscoveryResourceItem struct {
	ID           string
	TypeCode     string
	Title        string
	Category     string
	District     string
	PriceText    string
	QuantityText string
	MerchantName string
}

type DemandEntry struct {
	Title      string
	ButtonText string
}

type TopicResourcesResp struct {
	Topic       TopicInfo
	Items       []DiscoveryResourceItem
	Page        int64
	PageSize    int64
	Total       int64
	DemandEntry *DemandEntry
}

type ValidateWebviewURLResp struct {
	Allowed bool
	URL     string
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
		items = append(items, DiscoveryBannerItem{
			ID:         config.ID,
			Title:      config.Title,
			Subtitle:   config.Subtitle,
			CoverURL:   config.CoverURL,
			JumpType:   config.JumpType,
			JumpTarget: config.JumpTarget,
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
