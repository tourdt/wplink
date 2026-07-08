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

func TestCreateResourceRejectsMissingRequiredAttributeWithFieldLabel(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "season", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "season", "label": "季节", "type": "select"},
				},
			},
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
		Contact:     ResourceContactReq{Name: "张老板", Phone: "13800000000"},
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "请补充季节" {
		t.Fatalf("message = %q, want friendly dynamic field label", errx.PublicMessage(err))
	}
	if store.input.MerchantID != "" {
		t.Fatalf("CreateResource was called despite missing required attribute: %#v", store.input)
	}
}

func TestCreateResourceAcceptsRequiredAttributeBooleanFalse(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "allowLiveSale", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "allowLiveSale", "label": "支持直播", "type": "boolean"},
				},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: "pending"},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "inventory",
		Title:       "女童春款卫衣库存整包清",
		Category:    "童装",
		Attributes:  model.JSONMap{"allowLiveSale": false},
		Description: "整包优先，可现场看货。",
		Contact:     ResourceContactReq{Name: "张老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if store.input.Attributes["allowLiveSale"] != false {
		t.Fatalf("attributes = %#v, want boolean false preserved", store.input.Attributes)
	}
}

func TestCreateResourceRejectsSelectAttributeOutsideConfiguredOptions(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "season", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{
						"key":         "season",
						"label":       "季节",
						"type":        "select",
						"options":     []interface{}{"春季", "夏季", "秋季", "冬季"},
						"allowCustom": false,
					},
				},
			},
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
		Attributes:  model.JSONMap{"season": "春夏"},
		Contact:     ResourceContactReq{Name: "张老板", Phone: "13800000000"},
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "请选择正确的季节" {
		t.Fatalf("message = %q, want strict select option message", errx.PublicMessage(err))
	}
	if store.input.MerchantID != "" {
		t.Fatalf("CreateResource was called despite invalid select attribute: %#v", store.input)
	}
}

func TestCreateResourceAllowsCustomSelectAttributeWhenConfigured(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "goods",
			RequiredFields: []string{"title", "category", "style", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{
						"key":         "style",
						"label":       "风格",
						"type":        "select",
						"options":     []interface{}{"韩版", "学院风", "运动风"},
						"allowCustom": true,
					},
				},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: "pending"},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "goods",
		Title:       "童装套装一件代发货源",
		Category:    "童装套装",
		Description: "工厂直供套装货源，可一件代发。",
		Attributes:  model.JSONMap{"style": "原创设计款"},
		Contact:     ResourceContactReq{Name: "陈厂长", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if store.input.Attributes["style"] != "原创设计款" {
		t.Fatalf("attributes = %#v, want custom select value preserved", store.input.Attributes)
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
	if store.input.CoverURL != "https://example.com/a.jpg" {
		t.Fatalf("coverURL = %q, want first resource image", store.input.CoverURL)
	}
	if resp.ID != "resource-1" || resp.Status != "pending" {
		t.Fatalf("resp = %#v, want pending resource", resp)
	}
}

func TestCreateResourceCreatesDemandDirectionResource(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-demand-1",
			TypeCode:       "buy_goods",
			Direction:      model.ResourceDirectionDemand,
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "demand-resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store)

	resp, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:   "buyer-merchant-1",
		CityCode:     "zhili",
		TypeCode:     "buy_goods",
		Title:        "找童装防晒衣现货",
		Category:     "童装",
		QuantityText: "5000 件",
		PriceText:    "预算 30 元以内",
		Description:  "需要一周内可发货，接受外地发货。",
		Attributes:   model.JSONMap{"deliveryDeadline": "7天", "acceptRemoteShipping": true},
		Contact:      ResourceContactReq{Name: "王采购", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if store.input.Direction != model.ResourceDirectionDemand {
		t.Fatalf("direction = %q, want demand", store.input.Direction)
	}
	if resp.ID != "demand-resource-1" || resp.Status != model.ResourceStatusPending {
		t.Fatalf("resp = %#v, want demand pending resource", resp)
	}
}

func TestCreateResourceUsesMerchantContactPhoneWhenRequestPhoneIsMasked(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result:               model.CreateResourceResult{ID: "resource-1", Status: "pending"},
		merchantContactPhone: "18800000002",
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:   "merchant-1",
		CityCode:     "zhili",
		TypeCode:     "inventory",
		Title:        "女童春款卫衣库存整包清",
		Category:     "童装",
		QuantityText: "3200 件",
		Contact:      ResourceContactReq{Name: "张老板", Phone: "188****0002"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if store.contactMerchantID != "merchant-1" {
		t.Fatalf("contact merchantID = %q, want merchant-1", store.contactMerchantID)
	}
	if store.input.ContactPhone != "18800000002" {
		t.Fatalf("contact phone = %q, want merchant profile phone", store.input.ContactPhone)
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

func TestUpdateResourceDraftTurnsRejectedResourceIntoDraft(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		updateResult: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusDraft},
	}
	logic := NewCreateResourceLogic(store)

	resp, err := logic.UpdateResourceDraft(context.Background(), " resource-1 ", CreateResourceReq{
		MerchantID:   " merchant-1 ",
		CityCode:     "zhili",
		TypeCode:     "inventory",
		Title:        "修改后的童装库存",
		Category:     "童装",
		QuantityText: "3200 件",
		Description:  "补充清晰图片后重新保存。",
		Images:       []string{"https://example.com/draft-cover.jpg"},
		Contact:      ResourceContactReq{Name: "张老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("UpdateResourceDraft() error = %v", err)
	}

	if store.updatedResourceID != "resource-1" || store.updateInput.Status != model.ResourceStatusDraft {
		t.Fatalf("updatedResourceID = %q input = %#v", store.updatedResourceID, store.updateInput)
	}
	if store.updateInput.CoverURL != "https://example.com/draft-cover.jpg" {
		t.Fatalf("coverURL = %q, want first draft image", store.updateInput.CoverURL)
	}
	if store.updateInput.MerchantID != "merchant-1" || resp.Status != model.ResourceStatusDraft {
		t.Fatalf("resp = %#v input = %#v, want draft update", resp, store.updateInput)
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
	merchantStatus       string
	config               model.ResourcePublishConfig
	input                model.CreateResourceInput
	result               model.CreateResourceResult
	updatedResourceID    string
	updateInput          model.CreateResourceInput
	updateResult         model.CreateResourceResult
	operationLog         model.OperationLogInput
	merchantContactPhone string
	contactMerchantID    string
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

func (s *fakeCreateResourceStore) GetMerchantContactPhone(ctx context.Context, merchantID string) (string, error) {
	s.contactMerchantID = merchantID
	return s.merchantContactPhone, nil
}

func (s *fakeCreateResourceStore) CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error) {
	s.input = input
	return s.result, nil
}

func (s *fakeCreateResourceStore) UpdateResourceDraft(ctx context.Context, resourceID string, input model.CreateResourceInput) (model.CreateResourceResult, error) {
	s.updatedResourceID = resourceID
	s.updateInput = input
	return s.updateResult, nil
}

func (s *fakeCreateResourceStore) RecordOperationLog(ctx context.Context, input model.OperationLogInput) error {
	s.operationLog = input
	return nil
}

func (s *fakeCreateResourceStore) SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error) {
	return model.SubmitResourceResult{ID: resourceID, Status: "pending"}, nil
}
