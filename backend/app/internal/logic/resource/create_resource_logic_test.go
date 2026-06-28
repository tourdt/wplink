package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestCreateResourceRejectsMissingConfiguredRequiredField(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "inventory",
		Title:       "女童春款卫衣库存整包清",
		Category:    "童装",
		Description: "整包优先，可现场看货。",
		Contact: ResourceContactReq{
			Name:  "张老板",
			Phone: "13800000000",
		},
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestCreateResourceCreatesPendingResource(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{
			ID:     "resource-1",
			Status: "pending",
		},
	}
	logic := NewCreateResourceLogic(store)

	resp, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:    "merchant-1",
		CityCode:      " zhili ",
		TypeCode:      "inventory",
		Title:         "女童春款卫衣库存整包清",
		Category:      "童装",
		QuantityText:  "3200 件",
		Description:   "整包优先，可现场看货。",
		Attributes:    model.JSONMap{"season": "春款"},
		Tags:          []string{"急清"},
		Images:        []string{"https://example.com/a.jpg"},
		Contact:       ResourceContactReq{Name: "张老板", Phone: "13800000000", Wechat: "zhili_stock"},
		CreatedByUser: "user-1",
		CreatedByRole: "merchant_admin",
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if store.input.CityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.input.CityCode)
	}
	if store.input.Status != "pending" {
		t.Fatalf("status = %q, want pending", store.input.Status)
	}
	if resp.ID != "resource-1" || resp.Status != "pending" {
		t.Fatalf("resp = %#v, want pending resource", resp)
	}
}

func TestCreateResourceRejectsBlockedMerchant(t *testing.T) {
	store := &fakeCreateResourceStore{
		merchantStatus: "blocked",
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: "pending"},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:   "merchant-1",
		CityCode:     "zhili",
		TypeCode:     "inventory",
		Title:        "女童春款卫衣库存整包清",
		Category:     "童装",
		QuantityText: "3200 件",
		Description:  "整包优先，可现场看货。",
		Contact:      ResourceContactReq{Name: "张老板", Phone: "13800000000"},
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if store.input.MerchantID != "" {
		t.Fatalf("CreateResource was called for blocked merchant: %#v", store.input)
	}
}

func TestCreateResourceDraftCreatesDraftResource(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: "draft"},
	}
	logic := NewCreateResourceLogic(store)

	resp, err := logic.CreateResourceDraft(context.Background(), CreateResourceReq{
		MerchantID:   "merchant-1",
		CityCode:     "zhili",
		TypeCode:     "inventory",
		Title:        "女童春款卫衣库存整包清",
		Category:     "童装",
		QuantityText: "3200 件",
		Description:  "整包优先，可现场看货。",
		Contact:      ResourceContactReq{Name: "张老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResourceDraft() error = %v", err)
	}

	if store.input.Status != "draft" {
		t.Fatalf("status = %q, want draft", store.input.Status)
	}
	if resp.Status != "draft" || resp.Message != "草稿已保存" {
		t.Fatalf("resp = %#v, want draft saved message", resp)
	}
}

func TestCreateResourceRecordsOperationLogForOperatorProxy(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: "pending"},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:    "merchant-1",
		CityCode:      "zhili",
		TypeCode:      "inventory",
		Title:         "女童春款卫衣库存整包清",
		Category:      "童装",
		QuantityText:  "3200 件",
		Description:   "整包优先，可现场看货。",
		Contact:       ResourceContactReq{Name: "张老板", Phone: "13800000000"},
		CreatedByUser: "100000000000000001",
		CreatedByRole: "platform_operator",
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if store.operationLog.Action != "proxy_create_resource" {
		t.Fatalf("operation action = %q, want proxy_create_resource", store.operationLog.Action)
	}
	if store.operationLog.ObjectID != "resource-1" {
		t.Fatalf("operation object = %q, want resource-1", store.operationLog.ObjectID)
	}
}

type fakeCreateResourceStore struct {
	merchantStatus string
	config         model.ResourcePublishConfig
	input          model.CreateResourceInput
	result         model.CreateResourceResult
	operationLog   model.OperationLogInput
}

func (s *fakeCreateResourceStore) GetMerchantPublishStatus(ctx context.Context, merchantID string) (string, error) {
	if s.merchantStatus == "" {
		return "active", nil
	}
	return s.merchantStatus, nil
}

func (s *fakeCreateResourceStore) GetResourcePublishConfig(ctx context.Context, cityCode string, typeCode string) (model.ResourcePublishConfig, error) {
	return s.config, nil
}

func (s *fakeCreateResourceStore) CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error) {
	s.input = input
	return s.result, nil
}

func (s *fakeCreateResourceStore) RecordOperationLog(ctx context.Context, input model.OperationLogInput) error {
	s.operationLog = input
	return nil
}

func (s *fakeCreateResourceStore) SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error) {
	return model.SubmitResourceResult{ID: resourceID, Status: "pending"}, nil
}
