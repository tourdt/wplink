package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListMerchantsPassesFiltersToStore(t *testing.T) {
	store := &fakeAdminMerchantStore{
		result: model.ListMerchantsResult{
			Items: []model.MerchantListItem{
				{ID: "merchant-1", Name: "织里样板童装厂", MerchantType: "factory", VerificationStatus: "verified", Status: "active"},
			},
			Page:     1,
			PageSize: 20,
			Total:    1,
		},
	}
	logic := NewMerchantAdminLogic(store)

	resp, err := logic.ListMerchants(context.Background(), ListMerchantsReq{
		CityCode:     " zhili ",
		MerchantType: "factory",
		Keyword:      " 样板 ",
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("ListMerchants() error = %v", err)
	}

	if store.filter.CityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.filter.CityCode)
	}
	if store.filter.Keyword != "样板" {
		t.Fatalf("keyword = %q, want trimmed 样板", store.filter.Keyword)
	}
	if len(resp.Items) != 1 || resp.Items[0].Name != "织里样板童装厂" {
		t.Fatalf("items = %#v, want merchant list item", resp.Items)
	}
}

type fakeAdminMerchantStore struct {
	filter model.ListMerchantsFilter
	result model.ListMerchantsResult
}

func (s *fakeAdminMerchantStore) ListMerchants(ctx context.Context, filter model.ListMerchantsFilter) (model.ListMerchantsResult, error) {
	s.filter = filter
	return s.result, nil
}
