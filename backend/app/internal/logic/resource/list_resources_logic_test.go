package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListResourcesRequestsPublishedOnly(t *testing.T) {
	store := &fakeListResourcesStore{
		result: model.ListResourcesResult{
			Items: []model.ResourceListItem{{ID: "resource-1", TypeCode: "inventory", Title: "库存资源"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListResourcesLogic(store)

	resp, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili", TypeCode: "inventory"})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if store.filter.Status != "published" {
		t.Fatalf("status = %q, want published", store.filter.Status)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != "resource-1" {
		t.Fatalf("items = %#v, want resource item", resp.Items)
	}
}

type fakeListResourcesStore struct {
	filter model.ListResourcesFilter
	result model.ListResourcesResult
}

func (s *fakeListResourcesStore) ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error) {
	s.filter = filter
	return s.result, nil
}
