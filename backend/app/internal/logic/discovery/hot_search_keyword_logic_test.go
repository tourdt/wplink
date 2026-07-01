package discovery

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListHotSearchKeywordsUsesActiveCityFilter(t *testing.T) {
	store := &fakeHotSearchKeywordDiscoveryStore{
		items: []model.HotSearchKeywordConfig{
			{ID: "keyword-1", Keyword: "夏款现货", Status: "active"},
			{ID: "keyword-2", Keyword: "小单快返", Status: "active"},
		},
	}
	logic := NewHotSearchKeywordDiscoveryLogic(store)

	resp, err := logic.ListHotSearchKeywords(context.Background(), ListHotSearchKeywordsReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("ListHotSearchKeywords() error = %v", err)
	}

	if store.cityCode != "zhili" {
		t.Fatalf("cityCode = %q, want zhili", store.cityCode)
	}
	if len(resp.Items) != 2 || resp.Items[0].Keyword != "夏款现货" {
		t.Fatalf("items = %#v, want active keyword items", resp.Items)
	}
}

func TestListHotSearchKeywordsSkipsEmptyKeywords(t *testing.T) {
	store := &fakeHotSearchKeywordDiscoveryStore{
		items: []model.HotSearchKeywordConfig{
			{ID: "keyword-1", Keyword: "  "},
			{ID: "keyword-2", Keyword: "急清库存"},
		},
	}
	logic := NewHotSearchKeywordDiscoveryLogic(store)

	resp, err := logic.ListHotSearchKeywords(context.Background(), ListHotSearchKeywordsReq{CityCode: "zhili"})
	if err != nil {
		t.Fatalf("ListHotSearchKeywords() error = %v", err)
	}

	if len(resp.Items) != 1 || resp.Items[0].Keyword != "急清库存" {
		t.Fatalf("items = %#v, want non-empty keywords only", resp.Items)
	}
}

type fakeHotSearchKeywordDiscoveryStore struct {
	cityCode string
	items    []model.HotSearchKeywordConfig
}

func (s *fakeHotSearchKeywordDiscoveryStore) ListActiveHotSearchKeywords(ctx context.Context, cityCode string) ([]model.HotSearchKeywordConfig, error) {
	s.cityCode = cityCode
	return append([]model.HotSearchKeywordConfig(nil), s.items...), nil
}
