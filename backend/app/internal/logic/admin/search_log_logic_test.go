package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListSearchLogsPassesFiltersToStore(t *testing.T) {
	store := &fakeSearchLogStore{
		result: model.ListSearchLogsResult{
			Items: []model.SearchLogItem{{
				ID:          "search-1",
				CityCode:    "zhili",
				Keyword:     "童装库存",
				ResultCount: 0,
				CreatedAt:   "2026-06-28T10:00:00Z",
			}},
			Page: 1, PageSize: 20, Total: 1,
		},
	}
	logic := NewSearchLogLogic(store)

	resp, err := logic.ListSearchLogs(context.Background(), SearchLogsReq{CityCode: " zhili ", Keyword: " 童装 ", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListSearchLogs() error = %v", err)
	}

	if store.filter.CityCode != "zhili" || store.filter.Keyword != "童装" {
		t.Fatalf("filter = %#v, want trimmed filters", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Keyword != "童装库存" || resp.Items[0].ResultCount != 0 {
		t.Fatalf("resp = %#v, want search log item", resp)
	}
}

type fakeSearchLogStore struct {
	filter model.SearchLogFilter
	result model.ListSearchLogsResult
}

func (s *fakeSearchLogStore) ListSearchLogs(ctx context.Context, filter model.SearchLogFilter) (model.ListSearchLogsResult, error) {
	s.filter = filter
	return s.result, nil
}
