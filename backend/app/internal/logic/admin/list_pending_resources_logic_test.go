package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListPendingResourcesPassesPendingStatus(t *testing.T) {
	store := &fakePendingResourceStore{
		result: model.ListPendingResourcesResult{
			Items: []model.PendingResourceItem{{ID: "resource-1", Title: "待审核库存", TypeCode: "inventory", MerchantName: "织里样板童装厂"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListPendingResourcesLogic(store)

	resp, err := logic.ListPendingResources(context.Background(), ListPendingResourcesReq{CityCode: " zhili ", TypeCode: "inventory"})
	if err != nil {
		t.Fatalf("ListPendingResources() error = %v", err)
	}

	if store.filter.CityCode != "zhili" || store.filter.Status != "pending" {
		t.Fatalf("filter = %#v, want zhili pending", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "待审核库存" {
		t.Fatalf("items = %#v, want pending resource", resp.Items)
	}
}

type fakePendingResourceStore struct {
	filter model.ListPendingResourcesFilter
	result model.ListPendingResourcesResult
}

func (s *fakePendingResourceStore) ListPendingResources(ctx context.Context, filter model.ListPendingResourcesFilter) (model.ListPendingResourcesResult, error) {
	s.filter = filter
	return s.result, nil
}
