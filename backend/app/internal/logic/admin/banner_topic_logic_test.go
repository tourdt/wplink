package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListBannerTopicsPassesTrimmedFilters(t *testing.T) {
	store := &fakeBannerTopicAdminStore{
		items: []model.BannerTopicConfig{{ID: "topic-1", Kind: "topic", Title: "夏季童装"}},
	}
	logic := NewBannerTopicAdminLogic(store)

	resp, err := logic.ListBannerTopics(context.Background(), ListBannerTopicsReq{CityCode: " zhili ", Kind: " topic ", Status: " active "})
	if err != nil {
		t.Fatalf("ListBannerTopics() error = %v", err)
	}

	if store.filter.CityCode != "zhili" || store.filter.Kind != "topic" || store.filter.Status != "active" {
		t.Fatalf("filter = %#v, want trimmed filters", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "夏季童装" {
		t.Fatalf("items = %#v, want topic item", resp.Items)
	}
}

func TestCreateBannerTopicRejectsInvalidJumpURL(t *testing.T) {
	logic := NewBannerTopicAdminLogic(&fakeBannerTopicAdminStore{})

	_, err := logic.CreateBannerTopic(context.Background(), SaveBannerTopicReq{
		CityCode:   "zhili",
		Kind:       "banner",
		Title:      "活动",
		JumpType:   "webview",
		JumpTarget: "https://evil.example.com/a",
		Status:     "active",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("CreateBannerTopic() error = %v, want validation error", err)
	}
}

func TestCreateBannerTopicPassesInputToStore(t *testing.T) {
	store := &fakeBannerTopicAdminStore{saved: model.SaveBannerTopicResult{ID: "banner-1", UpdatedAt: "2026-06-27T10:00:00Z"}}
	logic := NewBannerTopicAdminLogic(store)

	resp, err := logic.CreateBannerTopic(context.Background(), SaveBannerTopicReq{
		CityCode:   " zhili ",
		Kind:       "banner",
		Title:      " 产业带精选 ",
		JumpType:   "topic",
		JumpTarget: "topic-1",
		Status:     "active",
		SortOrder:  10,
	})
	if err != nil {
		t.Fatalf("CreateBannerTopic() error = %v", err)
	}

	if store.input.CityCode != "zhili" || store.input.Title != "产业带精选" || store.input.SortOrder != 10 {
		t.Fatalf("input = %#v, want trimmed input", store.input)
	}
	if resp.ID != "banner-1" {
		t.Fatalf("id = %q, want banner-1", resp.ID)
	}
}

func TestCreateBannerTopicAllowsTopicJumpTargetToBeEmpty(t *testing.T) {
	store := &fakeBannerTopicAdminStore{saved: model.SaveBannerTopicResult{ID: "banner-topic-1", UpdatedAt: "2026-06-27T10:00:00Z"}}
	logic := NewBannerTopicAdminLogic(store)

	_, err := logic.CreateBannerTopic(context.Background(), SaveBannerTopicReq{
		CityCode: "zhili",
		Kind:     "banner",
		Title:    "急清库存专题",
		JumpType: "topic",
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("CreateBannerTopic() error = %v, want nil", err)
	}
	if store.input.JumpTarget != "" {
		t.Fatalf("jump target = %q, want empty self topic target", store.input.JumpTarget)
	}
}

func TestCreateBannerTopicAllowsHomeRecommendCardWithoutCover(t *testing.T) {
	store := &fakeBannerTopicAdminStore{saved: model.SaveBannerTopicResult{ID: "recommend-card-1", UpdatedAt: "2026-06-27T10:00:00Z"}}
	logic := NewBannerTopicAdminLogic(store)

	resp, err := logic.CreateBannerTopic(context.Background(), SaveBannerTopicReq{
		CityCode:   "zhili",
		Kind:       "home_recommend_card",
		Title:      "本周空档工厂：4 条针织生产线",
		Subtitle:   "认证工厂 · 适合小单快返 · 运营已核实",
		CoverURL:   "   ",
		JumpType:   "internal",
		JumpTarget: "/pages/search/index",
		Tags:       []string{" 平台推荐 "},
		Status:     "active",
	})
	if err != nil {
		t.Fatalf("CreateBannerTopic() error = %v, want nil", err)
	}

	if store.input.Kind != "home_recommend_card" || store.input.CoverURL != "" || store.input.Tags[0] != "平台推荐" {
		t.Fatalf("input = %#v, want home recommend card without cover", store.input)
	}
	if resp.ID != "recommend-card-1" {
		t.Fatalf("id = %q, want recommend-card-1", resp.ID)
	}
}

func TestCreateHomeRecommendCardRequiresExplicitTopicTarget(t *testing.T) {
	logic := NewBannerTopicAdminLogic(&fakeBannerTopicAdminStore{})

	_, err := logic.CreateBannerTopic(context.Background(), SaveBannerTopicReq{
		CityCode: "zhili",
		Kind:     "home_recommend_card",
		Title:    "专题推荐",
		JumpType: "topic",
		Status:   "active",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("CreateBannerTopic() error = %v, want validation error", err)
	}
}

type fakeBannerTopicAdminStore struct {
	filter model.BannerTopicFilter
	input  model.SaveBannerTopicInput
	items  []model.BannerTopicConfig
	saved  model.SaveBannerTopicResult
}

func (s *fakeBannerTopicAdminStore) ListBannerTopics(ctx context.Context, filter model.BannerTopicFilter) ([]model.BannerTopicConfig, error) {
	s.filter = filter
	return append([]model.BannerTopicConfig(nil), s.items...), nil
}

func (s *fakeBannerTopicAdminStore) CreateBannerTopic(ctx context.Context, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error) {
	s.input = input
	return s.saved, nil
}

func (s *fakeBannerTopicAdminStore) UpdateBannerTopic(ctx context.Context, configID string, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error) {
	s.input = input
	s.input.ID = configID
	return s.saved, nil
}
