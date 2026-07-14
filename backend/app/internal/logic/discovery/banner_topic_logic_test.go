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

func TestGetHomeOperationConfigSplitsActiveOperationItems(t *testing.T) {
	store := &fakeDiscoveryStore{
		operationConfigs: []model.BannerTopicConfig{
			{ID: "banner-1", Kind: "banner", Title: "产业带精选", JumpType: "topic", JumpTarget: "topic-1"},
			{
				ID:         "recommend-card-1",
				Kind:       "home_recommend_card",
				Title:      "本周空档工厂",
				Subtitle:   "认证工厂 · 适合小单快返",
				JumpType:   "search",
				JumpTarget: "小单快返",
				Tags:       []string{"平台推荐"},
			},
		},
	}
	logic := NewBannerTopicDiscoveryLogic(store)

	resp, err := logic.GetHomeOperationConfig(context.Background(), GetHomeOperationConfigReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("GetHomeOperationConfig() error = %v", err)
	}

	if store.operationCityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.operationCityCode)
	}
	if len(resp.Banners) != 1 || resp.Banners[0].JumpTarget != "topic-1" {
		t.Fatalf("banners = %#v, want banner item", resp.Banners)
	}
	if len(resp.RecommendCards) != 1 || resp.RecommendCards[0].Tag != "平台推荐" || resp.RecommendCards[0].Title != "本周空档工厂" {
		t.Fatalf("recommendCards = %#v, want recommend card item", resp.RecommendCards)
	}
}

func TestGetHomeOperationConfigUsesBannerIDForTopicWithoutTarget(t *testing.T) {
	store := &fakeDiscoveryStore{
		operationConfigs: []model.BannerTopicConfig{{ID: "banner-topic-1", Kind: "banner", Title: "急清库存专题", JumpType: "topic"}},
	}
	logic := NewBannerTopicDiscoveryLogic(store)

	resp, err := logic.GetHomeOperationConfig(context.Background(), GetHomeOperationConfigReq{CityCode: "zhili"})
	if err != nil {
		t.Fatalf("GetHomeOperationConfig() error = %v", err)
	}

	if len(resp.Banners) != 1 || resp.Banners[0].JumpTarget != "banner-topic-1" {
		t.Fatalf("banners = %#v, want topic banner to target itself", resp.Banners)
	}
}

func TestGetHomeOperationConfigReturnsFriendlyError(t *testing.T) {
	store := &fakeDiscoveryStore{operationErr: errors.New("db timeout")}
	logic := NewBannerTopicDiscoveryLogic(store)

	_, err := logic.GetHomeOperationConfig(context.Background(), GetHomeOperationConfigReq{CityCode: "zhili"})
	if err == nil || errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "首页运营配置加载失败，请稍后重试" {
		t.Fatalf("GetHomeOperationConfig() error = %v, want friendly internal error", err)
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
	operationCityCode string
	resourceFilter    model.ListResourcesFilter
	operationConfigs  []model.BannerTopicConfig
	operationErr      error
	topic             model.BannerTopicConfig
	resources         model.ListResourcesResult
	resourceErr       error
}

func (s *fakeDiscoveryStore) ListActiveHomeOperationConfigs(ctx context.Context, cityCode string) ([]model.BannerTopicConfig, error) {
	s.operationCityCode = cityCode
	if s.operationErr != nil {
		return nil, s.operationErr
	}
	return append([]model.BannerTopicConfig(nil), s.operationConfigs...), nil
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
