package resource

import (
	"context"
	"errors"
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

func TestSearchResourcesPassesGroupCodeToListAndSearchLog(t *testing.T) {
	store := &fakeSearchResourceStore{
		result: model.ListResourcesResult{Items: []model.ResourceListItem{{ID: "resource-1"}}, Total: 1, Page: 1, PageSize: 20},
	}
	logic := NewSearchResourcesLogic(store)

	_, err := logic.SearchResources(context.Background(), SearchResourcesReq{
		UserID: "user-1", CityCode: "zhili", GroupCode: "factory_warehouse", Keyword: "仓库", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("SearchResources() error = %v", err)
	}

	if store.filter.GroupCode != "factory_warehouse" {
		t.Fatalf("groupCode = %q, want factory_warehouse", store.filter.GroupCode)
	}
	if got := store.searchLog.Filters["groupCode"]; got != "factory_warehouse" {
		t.Fatalf("search log groupCode = %#v, want factory_warehouse", got)
	}
}

func TestSearchResourcesPassesTagsToListAndSearchLog(t *testing.T) {
	store := &fakeSearchResourceStore{
		result: model.ListResourcesResult{Items: []model.ResourceListItem{{ID: "resource-1"}}, Total: 1, Page: 1, PageSize: 20},
	}
	logic := NewSearchResourcesLogic(store)

	_, err := logic.SearchResources(context.Background(), SearchResourcesReq{
		UserID: "user-1", CityCode: "zhili", Tags: []string{"交通便利", "带车位"}, Keyword: "出租", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("SearchResources() error = %v", err)
	}

	if len(store.filter.Tags) != 2 || store.filter.Tags[0] != "交通便利" || store.filter.Tags[1] != "带车位" {
		t.Fatalf("filter tags = %#v, want selected tags", store.filter.Tags)
	}
	tags, ok := store.searchLog.Filters["tags"].([]string)
	if !ok || len(tags) != 2 || tags[0] != "交通便利" || tags[1] != "带车位" {
		t.Fatalf("search log tags = %#v, want selected tags", store.searchLog.Filters["tags"])
	}
}

func TestSearchResourcesReturnsResultsWhenSearchLogFails(t *testing.T) {
	store := &fakeSearchResourceStore{
		result:    model.ListResourcesResult{Items: []model.ResourceListItem{{ID: "resource-1"}}, Total: 1, Page: 1, PageSize: 20},
		recordErr: errors.New("insert search log failed"),
	}
	logic := NewSearchResourcesLogic(store)

	resp, err := logic.SearchResources(context.Background(), SearchResourcesReq{
		UserID: "user-1", CityCode: "zhili", Keyword: "卫衣", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("SearchResources() error = %v, want nil when search log fails", err)
	}
	if resp.Total != 1 || store.searchLog.Keyword != "卫衣" {
		t.Fatalf("resp = %#v searchLog = %#v, want search result and attempted log", resp, store.searchLog)
	}
}

type fakeSearchResourceStore struct {
	filter    model.ListResourcesFilter
	searchLog model.SearchLogInput
	result    model.ListResourcesResult
	recordErr error
}

func (s *fakeSearchResourceStore) ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error) {
	s.filter = filter
	return s.result, nil
}

func (s *fakeSearchResourceStore) RecordSearchLog(ctx context.Context, input model.SearchLogInput) error {
	s.searchLog = input
	return s.recordErr
}
