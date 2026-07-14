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
				TypeCode:         "stock_clearance",
				TypeName:         "库存出售",
				Direction:        model.ResourceDirectionSupply,
				DefaultValidDays: 7,
				FieldSchema: model.JSONMap{
					"fields": []interface{}{
						map[string]interface{}{"key": "season", "label": "季节", "type": "select"},
					},
				},
				RequiredFields:  []string{"title", "category"},
				FilterFields:    []string{"season"},
				DisplayTemplate: model.JSONMap{"list": []interface{}{"priceText"}},
			},
		},
	}
	logic := NewListResourceTypesLogic(store)

	resp, err := logic.ListResourceTypes(context.Background(), ListResourceTypesReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("ListResourceTypes() error = %v", err)
	}

	if store.cityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.cityCode)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].TypeCode != "stock_clearance" {
		t.Fatalf("typeCode = %q, want stock_clearance", resp.Items[0].TypeCode)
	}
	if resp.Items[0].Direction != model.ResourceDirectionSupply {
		t.Fatalf("direction = %q, want supply", resp.Items[0].Direction)
	}
	if resp.Items[0].RequiredFields[0] != "title" {
		t.Fatalf("required fields = %#v, want title first", resp.Items[0].RequiredFields)
	}
	fields, ok := resp.Items[0].FieldSchema["fields"].([]interface{})
	if !ok || len(fields) != 1 {
		t.Fatalf("fieldSchema = %#v, want fields returned for dynamic publish form", resp.Items[0].FieldSchema)
	}
}

func TestListResourceTypesPassesDirectionFilter(t *testing.T) {
	store := &fakeResourceTypeStore{
		configs: []model.ResourceTypeConfig{
			{
				ID:               "type-demand-1",
				TypeCode:         "buy_kids_goods",
				TypeName:         "求购尾货",
				Direction:        model.ResourceDirectionDemand,
				DefaultValidDays: 7,
			},
		},
	}
	logic := NewListResourceTypesLogic(store)

	resp, err := logic.ListResourceTypes(context.Background(), ListResourceTypesReq{CityCode: " zhili ", Direction: " demand "})
	if err != nil {
		t.Fatalf("ListResourceTypes() error = %v", err)
	}

	if store.cityCode != "zhili" || store.direction != model.ResourceDirectionDemand {
		t.Fatalf("cityCode = %q direction = %q, want zhili/demand", store.cityCode, store.direction)
	}
	if len(resp.Items) != 1 || resp.Items[0].Direction != model.ResourceDirectionDemand {
		t.Fatalf("items = %#v, want demand type", resp.Items)
	}
}

func TestListResourceTypesRejectsEmptyCityCode(t *testing.T) {
	logic := NewListResourceTypesLogic(&fakeResourceTypeStore{})

	_, err := logic.ListResourceTypes(context.Background(), ListResourceTypesReq{CityCode: " "})
	if err == nil {
		t.Fatal("ListResourceTypes() error = nil, want validation error")
	}
}

type fakeResourceTypeStore struct {
	cityCode  string
	direction string
	configs   []model.ResourceTypeConfig
}

func (s *fakeResourceTypeStore) ListActiveResourceTypesByCityCode(_ context.Context, cityCode string, direction string) ([]model.ResourceTypeConfig, error) {
	s.cityCode = cityCode
	s.direction = direction
	return append([]model.ResourceTypeConfig(nil), s.configs...), nil
}
