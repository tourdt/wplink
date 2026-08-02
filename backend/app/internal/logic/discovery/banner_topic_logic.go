package discovery

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/webview"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	homeResourcesLimit       int64 = 30
	homeRecentMerchantsLimit int64 = 6
)

type BannerTopicDiscoveryStore interface {
	ListActiveHomeOperationConfigs(ctx context.Context, cityCode string) ([]model.BannerTopicConfig, error)
	GetActiveTopic(ctx context.Context, topicID string, cityCode string) (model.BannerTopicConfig, error)
	ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error)
	ListHomeRecentMerchants(ctx context.Context, cityCode string, limit int64) ([]model.HomeRecentMerchant, error)
}

type GetHomeOperationConfigReq struct {
	CityCode string
}

type ListHomeResourcesReq struct {
	CityCode string
}

type ListHomeRecentMerchantsReq struct {
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

type HomeRecommendCardItem struct {
	ID         string `json:"id"`
	Tag        string `json:"tag,omitempty"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle,omitempty"`
	JumpType   string `json:"jumpType"`
	JumpTarget string `json:"jumpTarget"`
}

type GetHomeOperationConfigResp struct {
	Banners        []DiscoveryBannerItem   `json:"banners"`
	RecommendCards []HomeRecommendCardItem `json:"recommendCards"`
}

type HomeResourceMerchantBrief struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	VerificationStatus string `json:"verificationStatus"`
	VIPStatus          string `json:"vipStatus"`
}

type HomeResourceItem struct {
	ID           string                    `json:"id"`
	Direction    string                    `json:"direction"`
	TypeCode     string                    `json:"typeCode"`
	TypeName     string                    `json:"typeName,omitempty"`
	Title        string                    `json:"title"`
	Category     string                    `json:"category"`
	CoverURL     string                    `json:"coverUrl,omitempty"`
	District     string                    `json:"district,omitempty"`
	PriceText    string                    `json:"priceText,omitempty"`
	QuantityText string                    `json:"quantityText,omitempty"`
	Merchant     HomeResourceMerchantBrief `json:"merchant"`
	CreditTags   []string                  `json:"creditTags"`
	RefreshedAt  string                    `json:"refreshedAt,omitempty"`
	DealtAt      string                    `json:"dealtAt,omitempty"`
}

type ListHomeResourcesResp struct {
	Items    []HomeResourceItem `json:"items"`
	Page     int64              `json:"page"`
	PageSize int64              `json:"pageSize"`
	Total    int64              `json:"total"`
}

type HomeRecentMerchantItem struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	MerchantType   string   `json:"merchantType"`
	MainCategories []string `json:"mainCategories"`
	LogoURL        string   `json:"logoUrl,omitempty"`
	AddressText    string   `json:"addressText,omitempty"`
	OnboardedAt    string   `json:"onboardedAt"`
}

type ListHomeRecentMerchantsResp struct {
	Items []HomeRecentMerchantItem `json:"items"`
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
	CoverURL     string `json:"coverUrl,omitempty"`
	District     string `json:"district,omitempty"`
	PriceText    string `json:"priceText,omitempty"`
	QuantityText string `json:"quantityText,omitempty"`
	MerchantName string `json:"merchantName"`
	DealtAt      string `json:"dealtAt,omitempty"`
}

type TopicResourcesResp struct {
	Topic    TopicInfo               `json:"topic"`
	Items    []DiscoveryResourceItem `json:"items"`
	Page     int64                   `json:"page"`
	PageSize int64                   `json:"pageSize"`
	Total    int64                   `json:"total"`
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

func (l *BannerTopicDiscoveryLogic) GetHomeOperationConfig(ctx context.Context, req GetHomeOperationConfigReq) (GetHomeOperationConfigResp, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	// Banner 和首页推荐卡同属首页运营位配置，一次读取后按 kind 拆分，减少首屏请求和数据库查询次数。
	configs, err := l.store.ListActiveHomeOperationConfigs(ctx, cityCode)
	if err != nil {
		logx.Errorf("加载首页运营配置失败: cityCode=%s err=%+v", cityCode, err)
		return GetHomeOperationConfigResp{}, errx.New(errx.CodeInternalError, "首页运营配置加载失败，请稍后重试")
	}

	resp := GetHomeOperationConfigResp{
		Banners:        make([]DiscoveryBannerItem, 0),
		RecommendCards: make([]HomeRecommendCardItem, 0),
	}
	for _, config := range configs {
		switch config.Kind {
		case "banner":
			resp.Banners = append(resp.Banners, discoveryBannerItem(config))
		case "home_recommend_card":
			resp.RecommendCards = append(resp.RecommendCards, homeRecommendCardItem(config))
		}
	}
	return resp, nil
}

func discoveryBannerItem(config model.BannerTopicConfig) DiscoveryBannerItem {
	jumpTarget := config.JumpTarget
	if config.JumpType == "topic" && strings.TrimSpace(jumpTarget) == "" {
		jumpTarget = config.ID
	}
	return DiscoveryBannerItem{
		ID:         config.ID,
		Title:      config.Title,
		Subtitle:   config.Subtitle,
		CoverURL:   config.CoverURL,
		JumpType:   config.JumpType,
		JumpTarget: jumpTarget,
		Tags:       append([]string(nil), config.Tags...),
	}
}

func homeRecommendCardItem(config model.BannerTopicConfig) HomeRecommendCardItem {
	tag := ""
	if len(config.Tags) > 0 {
		tag = config.Tags[0]
	}
	return HomeRecommendCardItem{
		ID:         config.ID,
		Tag:        tag,
		Title:      config.Title,
		Subtitle:   config.Subtitle,
		JumpType:   config.JumpType,
		JumpTarget: config.JumpTarget,
	}
}

func (l *BannerTopicDiscoveryLogic) ListHomeResources(ctx context.Context, req ListHomeResourcesReq) (ListHomeResourcesResp, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	// 首页资源位只暴露一个稳定入口：置顶资源优先，其余按刷新/发布时间倒序，数量上限固定为 30 条。
	result, err := l.store.ListResources(ctx, model.ListResourcesFilter{
		CityCode: cityCode,
		Status:   model.ResourceStatusPublished,
		Page:     1,
		PageSize: homeResourcesLimit,
	})
	if err != nil {
		logx.Errorf("加载首页资源失败: cityCode=%s pageSize=%d err=%+v", cityCode, homeResourcesLimit, err)
		return ListHomeResourcesResp{}, errx.New(errx.CodeInternalError, "首页资源加载失败，请稍后重试")
	}

	items := make([]HomeResourceItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, HomeResourceItem{
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
			Merchant: HomeResourceMerchantBrief{
				ID:        item.Merchant.ID,
				Name:      item.Merchant.Name,
				VIPStatus: normalizeHomeVIPStatus(item.Merchant.VIPStatus),
			},
			CreditTags:  append([]string(nil), item.CreditTags...),
			RefreshedAt: item.RefreshedAt,
			DealtAt:     item.DealtAt,
		})
	}
	return ListHomeResourcesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *BannerTopicDiscoveryLogic) ListHomeRecentMerchants(ctx context.Context, req ListHomeRecentMerchantsReq) (ListHomeRecentMerchantsResp, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	// 首页曝光位固定最多 6 家，避免客户端扩大数量后稀释单个新商家的曝光价值。
	merchants, err := l.store.ListHomeRecentMerchants(ctx, cityCode, homeRecentMerchantsLimit)
	if err != nil {
		logx.Errorf("加载首页新入驻商家失败: cityCode=%s limit=%d err=%+v", cityCode, homeRecentMerchantsLimit, err)
		return ListHomeRecentMerchantsResp{}, errx.New(errx.CodeInternalError, "新入驻商家加载失败，请稍后重试")
	}

	items := make([]HomeRecentMerchantItem, 0, len(merchants))
	for _, merchant := range merchants {
		items = append(items, HomeRecentMerchantItem{
			ID:             merchant.ID,
			Name:           merchant.Name,
			MerchantType:   merchant.MerchantType,
			MainCategories: append([]string(nil), merchant.MainCategories...),
			LogoURL:        merchant.LogoURL,
			AddressText:    merchant.AddressText,
			OnboardedAt:    merchant.OnboardedAt,
		})
	}
	return ListHomeRecentMerchantsResp{Items: items}, nil
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
			CoverURL:     item.CoverURL,
			District:     item.District,
			PriceText:    item.PriceText,
			QuantityText: item.QuantityText,
			MerchantName: item.Merchant.Name,
			DealtAt:      item.DealtAt,
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
	return resp, nil
}

func normalizeHomeVIPStatus(status string) string {
	if strings.TrimSpace(status) == model.VIPStatusActive {
		return model.VIPStatusActive
	}
	return model.VIPStatusNone
}

func (l *BannerTopicDiscoveryLogic) ValidateWebviewURL(ctx context.Context, req ValidateWebviewURLReq) (ValidateWebviewURLResp, error) {
	rawURL := strings.TrimSpace(req.URL)
	if !webview.IsAllowedURL(rawURL) {
		return ValidateWebviewURLResp{}, errx.New(errx.CodeValidationFailed, "活动链接不在允许访问范围内")
	}
	return ValidateWebviewURLResp{Allowed: true, URL: rawURL}, nil
}
