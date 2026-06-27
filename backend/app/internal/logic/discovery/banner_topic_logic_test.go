package discovery

import (
	"context"
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

func TestGetTopicResourcesReturnsDemandEntryWhenEmpty(t *testing.T) {
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
	if resp.DemandEntry == nil || resp.DemandEntry.Title == "" {
		t.Fatalf("demandEntry = %#v, want fallback entry", resp.DemandEntry)
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
	return s.resources, nil
}
