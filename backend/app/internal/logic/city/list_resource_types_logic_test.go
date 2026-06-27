package city

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListResourceTypesReturnsActiveConfigForCity(t *testing.T) {
	store := &fakeResourceTypeStore{
		configs: []model.ResourceTypeConfig{
			{
				ID:               "type-1",
				TypeCode:         "inventory",
				TypeName:         "库存",
				DefaultValidDays: 7,
				RequiredFields:   []string{"title", "category"},
				FilterFields:     []string{"season"},
				DisplayTemplate:  model.JSONMap{"list": []interface{}{"priceText"}},
			},
		},
	}
	logic := NewListResourceTypesLogic(store)

	resp, err := logic.ListResourceTypes(context.Background(), " zhili ")
	if err != nil {
		t.Fatalf("ListResourceTypes() error = %v", err)
	}

	if store.cityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.cityCode)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].TypeCode != "inventory" {
		t.Fatalf("typeCode = %q, want inventory", resp.Items[0].TypeCode)
	}
	if resp.Items[0].RequiredFields[0] != "title" {
		t.Fatalf("required fields = %#v, want title first", resp.Items[0].RequiredFields)
	}
}

func TestListResourceTypesRejectsEmptyCityCode(t *testing.T) {
	logic := NewListResourceTypesLogic(&fakeResourceTypeStore{})

	_, err := logic.ListResourceTypes(context.Background(), " ")
	if err == nil {
		t.Fatal("ListResourceTypes() error = nil, want validation error")
	}
}

type fakeResourceTypeStore struct {
	cityCode string
	configs  []model.ResourceTypeConfig
}

func (s *fakeResourceTypeStore) ListActiveResourceTypesByCityCode(_ context.Context, cityCode string) ([]model.ResourceTypeConfig, error) {
	s.cityCode = cityCode
	return append([]model.ResourceTypeConfig(nil), s.configs...), nil
}
