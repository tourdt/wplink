package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListResourcesRequestsPublishedOnly(t *testing.T) {
	store := &fakeListResourcesStore{
		result: model.ListResourcesResult{
			Items: []model.ResourceListItem{{
				ID: "resource-1", TypeCode: "stock_clearance", TypeName: "尾货/库存出售", Title: "库存资源",
				Tags:     []string{"急清", "支持看货"},
				Merchant: model.ResourceMerchantBrief{ID: "merchant-1", Name: "织里云仓", VIPStatus: model.VIPStatusActive},
			}},
			Page: 1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListResourcesLogic(store)

	resp, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili", TypeCode: "inventory", MerchantID: " merchant-1 "})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if store.filter.Status != "published" {
		t.Fatalf("status = %q, want published", store.filter.Status)
	}
	if store.filter.MerchantID != "merchant-1" {
		t.Fatalf("merchantID = %q, want trimmed merchant-1", store.filter.MerchantID)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != "resource-1" {
		t.Fatalf("items = %#v, want resource item", resp.Items)
	}
	if resp.Items[0].TypeName != "尾货/库存出售" {
		t.Fatalf("typeName = %q, want dynamic category item name", resp.Items[0].TypeName)
	}
	if resp.Items[0].Merchant.VIPStatus != model.VIPStatusActive {
		t.Fatalf("merchant vipStatus = %q, want active", resp.Items[0].Merchant.VIPStatus)
	}
	if len(resp.Items[0].Tags) != 2 || resp.Items[0].Tags[0] != "急清" || resp.Items[0].Tags[1] != "支持看货" {
		t.Fatalf("tags = %#v, want resource tags copied to response", resp.Items[0].Tags)
	}
}

func TestListResourcesPassesDirectionFilter(t *testing.T) {
	store := &fakeListResourcesStore{
		result: model.ListResourcesResult{
			Items: []model.ResourceListItem{{ID: "demand-1", Direction: model.ResourceDirectionDemand, TypeCode: "buy_goods", Title: "找童装现货"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewListResourcesLogic(store)

	resp, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili", Direction: " demand ", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if store.filter.Direction != model.ResourceDirectionDemand {
		t.Fatalf("direction = %q, want demand", store.filter.Direction)
	}
	if len(resp.Items) != 1 || resp.Items[0].Direction != model.ResourceDirectionDemand {
		t.Fatalf("items = %#v, want demand item with direction", resp.Items)
	}
}

func TestListResourcesDoesNotDefaultToSupplyWhenDirectionOmitted(t *testing.T) {
	store := &fakeListResourcesStore{}
	logic := NewListResourcesLogic(store)

	_, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if store.filter.Direction != "" {
		t.Fatalf("direction = %q, want empty direction for category-driven mixed results", store.filter.Direction)
	}
}

func TestListResourcesPassesGroupCodeFilter(t *testing.T) {
	store := &fakeListResourcesStore{}
	logic := NewListResourcesLogic(store)

	_, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili", GroupCode: " factory_warehouse ", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if store.filter.GroupCode != "factory_warehouse" {
		t.Fatalf("groupCode = %q, want trimmed factory_warehouse", store.filter.GroupCode)
	}
}

func TestListResourcesPassesNormalizedTagFilters(t *testing.T) {
	store := &fakeListResourcesStore{}
	logic := NewListResourcesLogic(store)

	_, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili", Tags: []string{" 交通便利 ", "带车位", "交通便利"}, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if len(store.filter.Tags) != 2 || store.filter.Tags[0] != "交通便利" || store.filter.Tags[1] != "带车位" {
		t.Fatalf("tags = %#v, want trimmed and de-duplicated tag filters", store.filter.Tags)
	}
}

func TestListResourcesRejectsInvalidTagFilter(t *testing.T) {
	store := &fakeListResourcesStore{}
	logic := NewListResourcesLogic(store)

	_, err := logic.ListResources(context.Background(), ListResourcesReq{CityCode: "zhili", Tags: []string{"交通\n便利"}, Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("ListResources() error = nil, want tag validation error")
	}
	if store.filter.Status != "" {
		t.Fatalf("store filter = %#v, want query blocked before store call", store.filter)
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
