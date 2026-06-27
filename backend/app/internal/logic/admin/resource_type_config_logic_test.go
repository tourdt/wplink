package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListResourceTypeConfigsReturnsStoreItems(t *testing.T) {
	store := &fakeResourceTypeConfigStore{
		items: []model.AdminResourceTypeConfig{
			{
				ID:               "config-1",
				CityCode:         "zhili",
				TypeCode:         "inventory",
				TypeName:         "库存",
				DefaultValidDays: 7,
				Status:           "active",
			},
		},
	}
	logic := NewResourceTypeConfigLogic(store)

	resp, err := logic.ListResourceTypeConfigs(context.Background(), ListResourceTypeConfigsReq{
		CityCode: "zhili",
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("ListResourceTypeConfigs() error = %v", err)
	}

	if store.cityCode != "zhili" || store.status != "active" {
		t.Fatalf("filters = (%q, %q), want (zhili, active)", store.cityCode, store.status)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].TypeCode != "inventory" {
		t.Fatalf("typeCode = %q, want inventory", resp.Items[0].TypeCode)
	}
}

func TestUpdateResourceTypeConfigRejectsEmptyID(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "", UpdateResourceTypeConfigReq{})
	if err == nil {
		t.Fatal("UpdateResourceTypeConfig() error = nil, want validation error")
	}
}

func TestUpdateResourceTypeConfigPassesPatchToStore(t *testing.T) {
	store := &fakeResourceTypeConfigStore{updatedAt: "2026-06-27T10:00:00+08:00"}
	logic := NewResourceTypeConfigLogic(store)

	resp, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		DefaultValidDays: 10,
		Status:           "active",
		RequiredFields:   []string{"title"},
	})
	if err != nil {
		t.Fatalf("UpdateResourceTypeConfig() error = %v", err)
	}

	if store.configID != "config-1" {
		t.Fatalf("configID = %q, want config-1", store.configID)
	}
	if store.patch.DefaultValidDays != 10 {
		t.Fatalf("default valid days = %d, want 10", store.patch.DefaultValidDays)
	}
	if resp.UpdatedAt != "2026-06-27T10:00:00+08:00" {
		t.Fatalf("updatedAt = %q, want fixed time", resp.UpdatedAt)
	}
}

type fakeResourceTypeConfigStore struct {
	cityCode  string
	status    string
	configID  string
	patch     model.ResourceTypeConfigPatch
	updatedAt string
	items     []model.AdminResourceTypeConfig
}

func (s *fakeResourceTypeConfigStore) ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]model.AdminResourceTypeConfig, error) {
	s.cityCode = cityCode
	s.status = status
	return append([]model.AdminResourceTypeConfig(nil), s.items...), nil
}

func (s *fakeResourceTypeConfigStore) UpdateResourceTypeConfig(ctx context.Context, configID string, patch model.ResourceTypeConfigPatch) (string, error) {
	s.configID = configID
	s.patch = patch
	return s.updatedAt, nil
}
