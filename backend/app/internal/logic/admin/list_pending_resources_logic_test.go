package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListPendingResourcesPassesManualReviewStatus(t *testing.T) {
	store := &fakePendingResourceStore{
		result: model.ListPendingResourcesResult{
			Items: []model.PendingResourceItem{{ID: "resource-1", Status: model.ResourceStatusManualReview, Title: "待审核库存", TypeCode: "inventory", MerchantName: "织里样板童装厂"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListPendingResourcesLogic(store)

	resp, err := logic.ListPendingResources(context.Background(), ListPendingResourcesReq{CityCode: " zhili ", TypeCode: "inventory"})
	if err != nil {
		t.Fatalf("ListPendingResources() error = %v", err)
	}

	if store.filter.CityCode != "zhili" || store.filter.Status != model.ResourceStatusManualReview {
		t.Fatalf("filter = %#v, want zhili manual_review", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "待审核库存" {
		t.Fatalf("items = %#v, want pending resource", resp.Items)
	}
}

func TestListAdminResourcesAllowsAllStatuses(t *testing.T) {
	store := &fakePendingResourceStore{
		result: model.ListPendingResourcesResult{
			Items: []model.PendingResourceItem{{ID: "resource-1", Status: model.ResourceStatusPublished, Title: "已发布库存", TypeCode: "inventory", MerchantName: "织里样板童装厂"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListPendingResourcesLogic(store)

	resp, err := logic.ListAdminResources(context.Background(), ListPendingResourcesReq{CityCode: "zhili"})
	if err != nil {
		t.Fatalf("ListAdminResources() error = %v", err)
	}

	if store.filter.Status != "" {
		t.Fatalf("filter status = %q, want all statuses", store.filter.Status)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != model.ResourceStatusPublished {
		t.Fatalf("items = %#v, want published resource", resp.Items)
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
