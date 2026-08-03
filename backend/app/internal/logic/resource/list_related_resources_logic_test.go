package resource

import (
	"context"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListRelatedResourcesUsesDefaultAndMaximumPageSize(t *testing.T) {
	store := &fakeRelatedResourcesStore{}
	logic := NewListRelatedResourcesLogic(store)

	if _, err := logic.ListRelatedResources(context.Background(), "1", RelatedResourcesReq{}); err != nil {
		t.Fatalf("ListRelatedResources() error = %v", err)
	}
	if store.limit != 3 {
		t.Fatalf("default limit = %d, want 3", store.limit)
	}

	if _, err := logic.ListRelatedResources(context.Background(), "1", RelatedResourcesReq{PageSize: 20}); err != nil {
		t.Fatalf("ListRelatedResources() error = %v", err)
	}
	if store.limit != 6 {
		t.Fatalf("maximum limit = %d, want 6", store.limit)
	}
}

func TestListRelatedResourcesMapsRankedItemsWithoutChangingOrder(t *testing.T) {
	store := &fakeRelatedResourcesStore{items: []model.ResourceListItem{
		{ID: "same-type", TypeCode: "inventory", Title: "同类型资源", Merchant: model.ResourceMerchantBrief{ID: "merchant-1", VIPStatus: model.VIPStatusActive}},
		{ID: "same-group", TypeCode: "warehouse", Title: "同分组资源", Merchant: model.ResourceMerchantBrief{ID: "merchant-2"}},
	}}
	logic := NewListRelatedResourcesLogic(store)

	resp, err := logic.ListRelatedResources(context.Background(), "1", RelatedResourcesReq{PageSize: 3})
	if err != nil {
		t.Fatalf("ListRelatedResources() error = %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].ID != "same-type" || resp.Items[1].ID != "same-group" {
		t.Fatalf("items = %#v, want ranked store order preserved", resp.Items)
	}
	if resp.Items[0].Merchant.VIPStatus != model.VIPStatusActive || resp.Items[1].Merchant.VIPStatus != model.VIPStatusNone {
		t.Fatalf("merchant VIP statuses = %#v, want normalized values", resp.Items)
	}
}

func TestListRelatedResourcesReturnsEmptyItemsInsteadOfNil(t *testing.T) {
	logic := NewListRelatedResourcesLogic(&fakeRelatedResourcesStore{})

	resp, err := logic.ListRelatedResources(context.Background(), "1", RelatedResourcesReq{PageSize: 3})
	if err != nil {
		t.Fatalf("ListRelatedResources() error = %v", err)
	}
	if resp.Items == nil || len(resp.Items) != 0 {
		t.Fatalf("items = %#v, want non-nil empty slice", resp.Items)
	}
}

func TestListRelatedResourcesRejectsInvalidBigintResourceIDBeforeStore(t *testing.T) {
	invalidIDs := []string{" ", "abc", "0", "-1", "9223372036854775808"}
	for _, resourceID := range invalidIDs {
		t.Run(resourceID, func(t *testing.T) {
			store := &fakeRelatedResourcesStore{}
			logic := NewListRelatedResourcesLogic(store)

			_, err := logic.ListRelatedResources(context.Background(), resourceID, RelatedResourcesReq{PageSize: 3})
			if err == nil || !strings.Contains(err.Error(), "资源不存在或暂不可查看") {
				t.Fatalf("ListRelatedResources(%q) error = %v, want hidden invalid resource", resourceID, err)
			}
			if store.calls != 0 {
				t.Fatalf("store calls = %d, want 0 for invalid resource id %q", store.calls, resourceID)
			}
		})
	}
}

type fakeRelatedResourcesStore struct {
	items []model.ResourceListItem
	limit int64
	calls int
}

func (s *fakeRelatedResourcesStore) ListRelatedResources(ctx context.Context, resourceID string, limit int64) ([]model.ResourceListItem, error) {
	s.calls++
	s.limit = limit
	return s.items, nil
}
