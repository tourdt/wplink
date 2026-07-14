package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListHomeBannersUsesActiveBannerFilter(t *testing.T) {
	store := &fakeDiscoveryStore{
		banners: []model.BannerTopicConfig{{ID: "banner-1", Kind: "banner", Title: "产业带精选", JumpType: "topic", JumpTarget: "topic-1"}},
	}
	logic := NewBannerTopicDiscoveryLogic(store)

	resp, err := logic.ListHomeBanners(context.Background(), ListHomeBannersReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("ListHomeBanners() error = %v", err)
	}

	if store.bannerFilter.CityCode != "zhili" || store.bannerFilter.Kind != "banner" || store.bannerFilter.Status != "active" {
		t.Fatalf("filter = %#v, want active banner filter", store.bannerFilter)
	}
	if len(resp.Items) != 1 || resp.Items[0].JumpTarget != "topic-1" {
		t.Fatalf("items = %#v, want banner item", resp.Items)
	}
}

func TestListHomeBannersUsesBannerIDForTopicWithoutTarget(t *testing.T) {
	store := &fakeDiscoveryStore{
		banners: []model.BannerTopicConfig{{ID: "banner-topic-1", Kind: "banner", Title: "急清库存专题", JumpType: "topic"}},
	}
	logic := NewBannerTopicDiscoveryLogic(store)

	resp, err := logic.ListHomeBanners(context.Background(), ListHomeBannersReq{CityCode: "zhili"})
	if err != nil {
		t.Fatalf("ListHomeBanners() error = %v", err)
	}

	if len(resp.Items) != 1 || resp.Items[0].JumpTarget != "banner-topic-1" {
		t.Fatalf("items = %#v, want topic banner to target itself", resp.Items)
	}
}

func TestListHomeRecommendCardsUsesActiveRecommendCardFilter(t *testing.T) {
	store := &fakeDiscoveryStore{
		banners: []model.BannerTopicConfig{{
			ID:         "recommend-card-1",
			Kind:       "home_recommend_card",
			Title:      "本周空档工厂",
			Subtitle:   "认证工厂 · 适合小单快返",
			JumpType:   "search",
			JumpTarget: "小单快返",
			Tags:       []string{"平台推荐"},
		}},
	}
	logic := NewBannerTopicDiscoveryLogic(store)

	resp, err := logic.ListHomeRecommendCards(context.Background(), ListHomeRecommendCardsReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("ListHomeRecommendCards() error = %v", err)
	}

	if store.bannerFilter.CityCode != "zhili" || store.bannerFilter.Kind != "home_recommend_card" || store.bannerFilter.Status != "active" {
		t.Fatalf("filter = %#v, want active home recommend card filter", store.bannerFilter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Tag != "平台推荐" || resp.Items[0].Title != "本周空档工厂" {
		t.Fatalf("items = %#v, want recommend card item", resp.Items)
	}
}

func TestListHomeResourcesUsesHomepageRule(t *testing.T) {
	store := &fakeDiscoveryStore{
		resources: model.ListResourcesResult{
			Items: []model.ResourceListItem{{
				ID: "resource-1", Direction: model.ResourceDirectionSupply, TypeCode: "stock_clearance", TypeName: "库存清仓", Title: "女童卫衣库存",
				Category: "童装卫衣", PriceText: "18元/件", QuantityText: "3000件",
				Merchant:    model.ResourceMerchantBrief{ID: "merchant-1", Name: "织里云仓", VerificationStatus: "verified", VIPStatus: model.VIPStatusActive},
				CreditTags:  []string{"认证商家"},
				RefreshedAt: "2026-07-14T10:00:00Z",
			}},
			Page: 1, PageSize: homeResourcesLimit, Total: 1,
		},
	}
	logic := NewBannerTopicDiscoveryLogic(store)

	resp, err := logic.ListHomeResources(context.Background(), ListHomeResourcesReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("ListHomeResources() error = %v", err)
	}

	if store.resourceFilter.CityCode != "zhili" || store.resourceFilter.Status != model.ResourceStatusPublished {
		t.Fatalf("resource filter = %#v, want zhili published resources", store.resourceFilter)
	}
	if store.resourceFilter.Page != 1 || store.resourceFilter.PageSize != homeResourcesLimit {
		t.Fatalf("page filter = page %d pageSize %d, want 1/%d", store.resourceFilter.Page, store.resourceFilter.PageSize, homeResourcesLimit)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != "resource-1" || resp.Items[0].Merchant.VIPStatus != model.VIPStatusActive {
		t.Fatalf("items = %#v, want home resource item with merchant state", resp.Items)
	}
	if resp.PageSize != homeResourcesLimit || resp.Total != 1 {
		t.Fatalf("pagination = %#v, want homepage limit and total", resp)
	}
}

func TestListHomeResourcesReturnsFriendlyError(t *testing.T) {
	store := &fakeDiscoveryStore{resourceErr: errors.New("db timeout")}
	logic := NewBannerTopicDiscoveryLogic(store)

	_, err := logic.ListHomeResources(context.Background(), ListHomeResourcesReq{CityCode: "zhili"})
	if err == nil || errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "首页资源加载失败，请稍后重试" {
		t.Fatalf("ListHomeResources() error = %v, want friendly internal error", err)
	}
}

func TestGetTopicResourcesDoesNotReturnDemandEntryWhenEmpty(t *testing.T) {
	store := &fakeDiscoveryStore{
		topic: model.BannerTopicConfig{ID: "topic-1", Kind: "topic", Title: "夏季童装", TypeScope: []string{"inventory"}},
		resources: model.ListResourcesResult{
			Items: []model.ResourceListItem{},
			Page:  1, PageSize: 20, Total: 0,
		},
	}
	logic := NewBannerTopicDiscoveryLogic(store)

	resp, err := logic.GetTopicResources(context.Background(), TopicResourcesReq{TopicID: "topic-1", CityCode: "zhili"})
	if err != nil {
		t.Fatalf("GetTopicResources() error = %v", err)
	}

	if store.resourceFilter.TypeCode != "inventory" {
		t.Fatalf("resource filter = %#v, want inventory scope", store.resourceFilter)
	}
	if resp.Total != 0 || len(resp.Items) != 0 {
		t.Fatalf("resources = total %d items %#v, want empty result", resp.Total, resp.Items)
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if strings.Contains(string(data), "demandEntry") {
		t.Fatalf("response json contains retired demandEntry: %s", string(data))
	}
}

func TestValidateWebviewURLRejectsUnknownDomain(t *testing.T) {
	logic := NewBannerTopicDiscoveryLogic(&fakeDiscoveryStore{})

	_, err := logic.ValidateWebviewURL(context.Background(), ValidateWebviewURLReq{URL: "https://evil.example.com/promo"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("ValidateWebviewURL() error = %v, want validation error", err)
	}
}

func TestValidateWebviewURLAcceptsAllowedDomain(t *testing.T) {
	logic := NewBannerTopicDiscoveryLogic(&fakeDiscoveryStore{})

	resp, err := logic.ValidateWebviewURL(context.Background(), ValidateWebviewURLReq{URL: "https://www.wplink.cn/promo"})
	if err != nil {
		t.Fatalf("ValidateWebviewURL() error = %v", err)
	}

	if !resp.Allowed {
		t.Fatalf("allowed = false, want true")
	}
}

type fakeDiscoveryStore struct {
	bannerFilter   model.BannerTopicFilter
	resourceFilter model.ListResourcesFilter
	banners        []model.BannerTopicConfig
	topic          model.BannerTopicConfig
	resources      model.ListResourcesResult
	resourceErr    error
}

func (s *fakeDiscoveryStore) ListActiveBannerTopics(ctx context.Context, filter model.BannerTopicFilter) ([]model.BannerTopicConfig, error) {
	s.bannerFilter = filter
	return append([]model.BannerTopicConfig(nil), s.banners...), nil
}

func (s *fakeDiscoveryStore) GetActiveTopic(ctx context.Context, topicID string, cityCode string) (model.BannerTopicConfig, error) {
	return s.topic, nil
}

func (s *fakeDiscoveryStore) ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error) {
	s.resourceFilter = filter
	if s.resourceErr != nil {
		return model.ListResourcesResult{}, s.resourceErr
	}
	return s.resources, nil
}
