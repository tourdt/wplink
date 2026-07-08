package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestSearchResourcesRecordsSearchLog(t *testing.T) {
	store := &fakeSearchResourceStore{
		result: model.ListResourcesResult{
			Items: []model.ResourceListItem{{ID: "resource-1", Title: "女童卫衣库存"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewSearchResourcesLogic(store)

	_, err := logic.SearchResources(context.Background(), SearchResourcesReq{
		UserID: "user-1", CityCode: "zhili", Keyword: "卫衣", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("SearchResources() error = %v", err)
	}

	if store.searchLog.Keyword != "卫衣" || store.searchLog.ResultCount != 1 {
		t.Fatalf("searchLog = %#v, want keyword and result count", store.searchLog)
	}
}

func TestSearchResourcesPassesDirectionToListAndSearchLog(t *testing.T) {
	store := &fakeSearchResourceStore{
		result: model.ListResourcesResult{Items: []model.ResourceListItem{{ID: "demand-1"}}, Total: 1, Page: 1, PageSize: 20},
	}
	logic := NewSearchResourcesLogic(store)

	_, err := logic.SearchResources(context.Background(), SearchResourcesReq{
		UserID: "user-1", CityCode: "zhili", Direction: model.ResourceDirectionDemand, Keyword: "防晒衣", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("SearchResources() error = %v", err)
	}

	if store.filter.Direction != model.ResourceDirectionDemand {
		t.Fatalf("direction = %q, want demand", store.filter.Direction)
	}
	if got := store.searchLog.Filters["direction"]; got != model.ResourceDirectionDemand {
		t.Fatalf("search log direction = %#v, want demand", got)
	}
}

type fakeSearchResourceStore struct {
	filter    model.ListResourcesFilter
	searchLog model.SearchLogInput
	result    model.ListResourcesResult
}

func (s *fakeSearchResourceStore) ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error) {
	s.filter = filter
	return s.result, nil
}

func (s *fakeSearchResourceStore) RecordSearchLog(ctx context.Context, input model.SearchLogInput) error {
	s.searchLog = input
	return nil
}
