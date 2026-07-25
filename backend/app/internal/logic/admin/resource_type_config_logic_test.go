package admin

import (
	"context"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListResourceTypeConfigsReturnsStoreItems(t *testing.T) {
	store := &fakeResourceTypeConfigStore{
		items: []model.AdminResourceTypeConfig{
			{
				ID:               "config-1",
				Version:          4,
				CityCode:         "zhili",
				TypeCode:         "buy_kids_goods",
				TypeName:         "求购尾货",
				Direction:        model.ResourceDirectionDemand,
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
	if resp.Items[0].TypeCode != "buy_kids_goods" {
		t.Fatalf("typeCode = %q, want buy_kids_goods", resp.Items[0].TypeCode)
	}
	if resp.Items[0].Direction != model.ResourceDirectionDemand {
		t.Fatalf("direction = %q, want demand", resp.Items[0].Direction)
	}
	if resp.Items[0].Version != 4 {
		t.Fatalf("version = %d, want 4", resp.Items[0].Version)
	}
}

func TestUpdateResourceTypeConfigRejectsEmptyID(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "", UpdateResourceTypeConfigReq{})
	if err == nil {
		t.Fatal("UpdateResourceTypeConfig() error = nil, want validation error")
	}
}

func TestUpdateResourceTypeConfigRequiresExpectedVersion(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		DefaultValidDays: 10,
		Status:           "active",
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed || !strings.Contains(err.Error(), "版本") {
		t.Fatalf("UpdateResourceTypeConfig() error = %v, want version validation", err)
	}
}

func TestCreateResourceTypeConfigPassesGroupAndDefaultsToStore(t *testing.T) {
	store := &fakeResourceTypeConfigStore{createdID: "config-2", updatedAt: "2026-07-14T10:00:00+08:00"}
	logic := NewResourceTypeConfigLogic(store)

	resp, err := logic.CreateResourceTypeConfig(context.Background(), CreateResourceTypeConfigReq{
		CityCode:         " zhili ",
		TypeCode:         " kids_brand_stock ",
		TypeName:         " 品牌库存 ",
		Direction:        model.ResourceDirectionSupply,
		GroupCode:        " kids_wholesale ",
		GroupName:        " 童装批发 ",
		GroupSort:        10,
		DefaultValidDays: 15,
	})
	if err != nil {
		t.Fatalf("CreateResourceTypeConfig() error = %v", err)
	}

	if store.createInput.CityCode != "zhili" {
		t.Fatalf("cityCode = %q, want zhili", store.createInput.CityCode)
	}
	if store.createInput.TypeCode != "kids_brand_stock" {
		t.Fatalf("typeCode = %q, want kids_brand_stock", store.createInput.TypeCode)
	}
	if store.createInput.TypeName != "品牌库存" {
		t.Fatalf("typeName = %q, want 品牌库存", store.createInput.TypeName)
	}
	if store.createInput.Direction != model.ResourceDirectionSupply {
		t.Fatalf("direction = %q, want supply", store.createInput.Direction)
	}
	group, _ := store.createInput.DisplayTemplate["group"].(map[string]interface{})
	if group["code"] != "kids_wholesale" || group["name"] != "童装批发" {
		t.Fatalf("group = %#v, want kids_wholesale/童装批发", group)
	}
	if store.createInput.RequiredFields[0] != "title" || store.createInput.RequiredFields[1] != "contactPhone" {
		t.Fatalf("required fields = %#v, want default title/contactPhone", store.createInput.RequiredFields)
	}
	createRules := model.CommercialRulesFromJSON(store.createInput.CommercialRules)
	if createRules.Publish.Mode != model.ResourcePublishModeConsumeQuota {
		t.Fatalf("commercial publish rules = %#v, want consume quota default", store.createInput.CommercialRules)
	}
	if createRules.ContactUnlock.Mode != model.ContactUnlockModeLoginFree || createRules.ContactUnlock.Currency != "CNY" {
		t.Fatalf("commercial contact rules = %#v, want login free default", store.createInput.CommercialRules)
	}
	if resp.ID != "config-2" || resp.UpdatedAt != "2026-07-14T10:00:00+08:00" {
		t.Fatalf("resp = %#v, want created id and updatedAt", resp)
	}
}

func TestCreateResourceTypeConfigRejectsMissingGroup(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.CreateResourceTypeConfig(context.Background(), CreateResourceTypeConfigReq{
		CityCode:         "zhili",
		TypeCode:         "kids_brand_stock",
		TypeName:         "品牌库存",
		Direction:        model.ResourceDirectionSupply,
		DefaultValidDays: 15,
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "请选择一级分类") {
		t.Fatalf("error = %v, want missing group message", err)
	}
}

func TestUpdateResourceTypeConfigPassesPatchToStore(t *testing.T) {
	store := &fakeResourceTypeConfigStore{updatedAt: "2026-06-27T10:00:00+08:00"}
	logic := NewResourceTypeConfigLogic(store)

	resp, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
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
	patchRules := model.CommercialRulesFromJSON(store.patch.CommercialRules)
	if patchRules.ContactUnlock.Mode != model.ContactUnlockModeLoginFree {
		t.Fatalf("commercial rules = %#v, want login free default", store.patch.CommercialRules)
	}
	if resp.UpdatedAt != "2026-06-27T10:00:00+08:00" {
		t.Fatalf("updatedAt = %q, want fixed time", resp.UpdatedAt)
	}
	if resp.Version != 2 || store.patch.ExpectedVersion != 1 {
		t.Fatalf("response version = %d expected version = %d, want 2/1", resp.Version, store.patch.ExpectedVersion)
	}
}

func TestUpdateResourceTypeConfigAcceptsAddressFieldType(t *testing.T) {
	store := &fakeResourceTypeConfigStore{updatedAt: "2026-07-16T10:00:00+08:00"}
	logic := NewResourceTypeConfigLogic(store)

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
		DefaultValidDays: 10,
		Status:           "active",
		FieldSchema: map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{"key": "locationText", "label": "详细位置", "type": "address", "required": true, "filterable": true, "displayIn": []interface{}{"detail"}},
			},
		},
		RequiredFields: []string{"title", "locationText"},
		FilterFields:   []string{"locationText"},
		DisplayTemplate: map[string]interface{}{
			"detail": []interface{}{"locationText"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateResourceTypeConfig() error = %v, want address field accepted", err)
	}

	fields, _ := store.patch.FieldSchema["fields"].([]interface{})
	if len(fields) != 1 {
		t.Fatalf("fields length = %d, want 1", len(fields))
	}
	field, _ := fields[0].(map[string]interface{})
	if field["type"] != "address" || field["label"] != "详细位置" {
		t.Fatalf("field = %#v, want address detailed location field", field)
	}
}

func TestUpdateResourceTypeConfigRejectsPaidContactWithoutPrice(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
		DefaultValidDays: 15,
		Status:           "active",
		CommercialRules: map[string]interface{}{
			"publish": map[string]interface{}{"mode": model.ResourcePublishModeFree},
			"contactUnlock": map[string]interface{}{
				"mode":             model.ContactUnlockModePaidOrVIP,
				"priceCent":        float64(0),
				"currency":         "CNY",
				"repeatUnlockDays": float64(30),
			},
		},
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "查看价格") {
		t.Fatalf("error = %v, want contact price validation", err)
	}
}

func TestUpdateResourceTypeConfigRejectsDuplicateDynamicFieldKeys(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
		DefaultValidDays: 10,
		Status:           "active",
		FieldSchema: map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{"key": "season", "label": "季节", "type": "select", "options": []interface{}{"春季"}},
				map[string]interface{}{"key": "season", "label": "季节说明", "type": "text"},
			},
		},
	})
	if err == nil {
		t.Fatal("UpdateResourceTypeConfig() error = nil, want duplicate field validation error")
	}
	if !strings.Contains(err.Error(), "字段编码重复") {
		t.Fatalf("error = %v, want duplicate field key message", err)
	}
}

func TestUpdateResourceTypeConfigRejectsUnsupportedDynamicFieldType(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
		DefaultValidDays: 10,
		Status:           "active",
		FieldSchema: map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{"key": "season", "label": "季节", "type": "dropdown"},
			},
		},
	})
	if err == nil {
		t.Fatal("UpdateResourceTypeConfig() error = nil, want unsupported field type validation error")
	}
	if !strings.Contains(err.Error(), "字段类型不支持") {
		t.Fatalf("error = %v, want unsupported field type message", err)
	}
}

func TestUpdateResourceTypeConfigRejectsSelectWithoutOptionsWhenCustomDisabled(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
		DefaultValidDays: 10,
		Status:           "active",
		FieldSchema: map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{"key": "season", "label": "季节", "type": "select"},
			},
		},
	})
	if err == nil {
		t.Fatal("UpdateResourceTypeConfig() error = nil, want select options validation error")
	}
	if !strings.Contains(err.Error(), "请为下拉字段配置选项") {
		t.Fatalf("error = %v, want select options message", err)
	}
}

func TestUpdateResourceTypeConfigRejectsInvalidSummaryTarget(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
		DefaultValidDays: 10,
		Status:           "active",
		FieldSchema: map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{"key": "serviceType", "label": "服务类型", "type": "text"},
			},
		},
		RequiredFields: []string{"title", "serviceType", "contactPhone"},
		DisplayTemplate: map[string]interface{}{
			"summary": map[string]interface{}{
				"unknown": "serviceType",
			},
		},
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "摘要字段") {
		t.Fatalf("error = %v, want summary validation message", err)
	}
}

func TestUpdateResourceTypeConfigRejectsInvalidSummarySource(t *testing.T) {
	logic := NewResourceTypeConfigLogic(&fakeResourceTypeConfigStore{})

	_, err := logic.UpdateResourceTypeConfig(context.Background(), "config-1", UpdateResourceTypeConfigReq{
		Version:          1,
		DefaultValidDays: 10,
		Status:           "active",
		FieldSchema: map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{"key": "serviceType", "label": "服务类型", "type": "text"},
			},
		},
		RequiredFields: []string{"title", "serviceType", "contactPhone"},
		DisplayTemplate: map[string]interface{}{
			"summary": map[string]interface{}{
				"category": "missingField",
			},
		},
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "摘要字段") {
		t.Fatalf("error = %v, want summary validation message", err)
	}
}

type fakeResourceTypeConfigStore struct {
	cityCode    string
	status      string
	configID    string
	patch       model.ResourceTypeConfigPatch
	updatedAt   string
	createdID   string
	createInput model.CreateResourceTypeConfigInput
	items       []model.AdminResourceTypeConfig
}

func (s *fakeResourceTypeConfigStore) ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]model.AdminResourceTypeConfig, error) {
	s.cityCode = cityCode
	s.status = status
	return append([]model.AdminResourceTypeConfig(nil), s.items...), nil
}

func (s *fakeResourceTypeConfigStore) UpdateResourceTypeConfig(ctx context.Context, configID string, patch model.ResourceTypeConfigPatch) (model.UpdateResourceTypeConfigResult, error) {
	s.configID = configID
	s.patch = patch
	return model.UpdateResourceTypeConfigResult{Version: 2, UpdatedAt: s.updatedAt}, nil
}

func (s *fakeResourceTypeConfigStore) CreateResourceTypeConfig(ctx context.Context, input model.CreateResourceTypeConfigInput) (model.CreateResourceTypeConfigResult, error) {
	s.createInput = input
	return model.CreateResourceTypeConfigResult{ID: s.createdID, Version: 1, UpdatedAt: s.updatedAt}, nil
}
