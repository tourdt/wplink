package resource

import (
	"context"
	"errors"
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

func TestCreateResourceGeneratesTitleWhenTitleIsMissing(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			Version:        3,
			TypeCode:       "inventory",
			TypeName:       "库存出售",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:   "merchant-1",
		CityCode:     "zhili",
		TypeCode:     "inventory",
		Category:     "童装",
		QuantityText: "3200 件",
		Description:  "整包优先，可现场看货，支持同城自提。",
		Contact:      ResourceContactReq{Name: "张老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	if store.input.Title != "童装 3200 件" {
		t.Fatalf("title = %q, want generated summary title", store.input.Title)
	}
	if store.input.ResourceTypeConfigVersion != 3 {
		t.Fatalf("config version = %d, want 3", store.input.ResourceTypeConfigVersion)
	}
	if store.input.ResourceTypeSnapshot["typeName"] != "库存出售" || store.input.ResourceTypeSnapshot["version"] != int64(3) {
		t.Fatalf("config snapshot = %#v, want immutable version 3 snapshot", store.input.ResourceTypeSnapshot)
	}
}

func TestCreateResourceRejectsMissingDescription(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "buy_kids_goods",
			Direction:      model.ResourceDirectionDemand,
			RequiredFields: []string{"title", "category", "contactPhone"},
		},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "buy_kids_goods",
		Title:       "急找童装库存",
		Category:    "童装",
		Description: " ",
		Contact:     ResourceContactReq{Name: "王采购", Phone: "13800000000"},
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "请填写需求描述" {
		t.Fatalf("message = %q, want demand description message", errx.PublicMessage(err))
	}
	if store.input.MerchantID != "" {
		t.Fatalf("CreateResource was called despite missing description: %#v", store.input)
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

func TestCreateResourceAcceptsManualAddressAttributeWithoutGPS(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "shop_office_rental",
			RequiredFields: []string{"title", "locationText", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "locationText", "label": "位置", "type": "address"},
				},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: "pending"},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "shop_office_rental",
		Title:       "童装城附近档口出租",
		Description: "一楼档口，可随时看房。",
		Attributes:  model.JSONMap{"locationText": "织里童装城一区附近"},
		Contact:     ResourceContactReq{Name: "王老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	address, ok := store.input.Attributes["locationText"].(model.JSONMap)
	if !ok {
		t.Fatalf("address attribute = %#v, want normalized JSON map", store.input.Attributes["locationText"])
	}
	if address["address"] != "织里童装城一区附近" {
		t.Fatalf("address attribute = %#v, want manual address text", address)
	}
	if _, ok := address["latitude"]; ok {
		t.Fatalf("address attribute = %#v, manual address should not keep GPS", address)
	}
}

func TestCreateResourceAcceptsMapAddressAttributeWithGPS(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "job_hiring",
			RequiredFields: []string{"title", "workLocation", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "workLocation", "label": "工作地点", "type": "address"},
				},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: "pending"},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "job_hiring",
		Title:       "招聘平车工",
		Description: "计件，多劳多得。",
		Attributes: model.JSONMap{"workLocation": map[string]interface{}{
			"address":   "浙江省湖州市吴兴区织里镇童装城",
			"name":      "织里童装城",
			"latitude":  "30.8732",
			"longitude": 120.2255,
		}},
		Contact: ResourceContactReq{Name: "陈老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	address, ok := store.input.Attributes["workLocation"].(model.JSONMap)
	if !ok {
		t.Fatalf("address attribute = %#v, want normalized JSON map", store.input.Attributes["workLocation"])
	}
	if address["address"] != "浙江省湖州市吴兴区织里镇童装城" || address["name"] != "织里童装城" {
		t.Fatalf("address attribute = %#v, want address and name preserved", address)
	}
	if address["latitude"] != 30.8732 || address["longitude"] != 120.2255 {
		t.Fatalf("address coordinates = %#v, want normalized GPS", address)
	}
}

func TestCreateResourceRejectsIncompleteAddressGPS(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "job_hiring",
			RequiredFields: []string{"title", "workLocation", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "workLocation", "label": "工作地点", "type": "address"},
				},
			},
		},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "job_hiring",
		Title:       "招聘后道",
		Description: "长期稳定。",
		Attributes: model.JSONMap{"workLocation": map[string]interface{}{
			"address":  "织里镇某园区",
			"latitude": 30.8732,
		}},
		Contact: ResourceContactReq{Name: "陈老板", Phone: "13800000000"},
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "请重新选择工作地点地图位置" {
		t.Fatalf("message = %q, want address gps error", errx.PublicMessage(err))
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
	if store.input.ConsumePublishQuota {
		t.Fatalf("consumePublishQuota = true, want false before auto audit publish")
	}
	if store.input.CoverURL != "https://example.com/a.jpg" {
		t.Fatalf("coverURL = %q, want first resource image", store.input.CoverURL)
	}
	if resp.ID != "resource-1" || resp.Status != "pending" {
		t.Fatalf("resp = %#v, want pending resource", resp)
	}
}

func TestCreateResourceNormalizesConfiguredTags(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-rental",
			TypeCode:       "apartment_rental",
			RequiredFields: []string{"title", "category", "contactPhone"},
			FieldSchema: model.JSONMap{
				"tagOptions": []interface{}{"交通便利", "带车位", "靠近商圈", "电梯房"},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "apartment_rental",
		Title:       "套房出租",
		Category:    "套房",
		Description: "交通方便，可随时看房。",
		Tags:        []string{" 交通便利 ", "", "带车位", "交通便利"},
		Contact:     ResourceContactReq{Name: "李经理", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if len(store.input.Tags) != 2 || store.input.Tags[0] != "交通便利" || store.input.Tags[1] != "带车位" {
		t.Fatalf("tags = %#v, want trimmed and de-duplicated configured tags", store.input.Tags)
	}
}

func TestCreateResourceRejectsTagOutsideConfiguredOptions(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-rental",
			TypeCode:       "apartment_rental",
			RequiredFields: []string{"title", "category", "contactPhone"},
			FieldSchema: model.JSONMap{
				"tagOptions": []interface{}{"交通便利", "带车位"},
			},
		},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "apartment_rental",
		Title:       "套房出租",
		Category:    "套房",
		Description: "交通方便，可随时看房。",
		Tags:        []string{"靠近商圈"},
		Contact:     ResourceContactReq{Name: "李经理", Phone: "13800000000"},
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "请选择正确的供应标签" {
		t.Fatalf("message = %q, want configured tag message", errx.PublicMessage(err))
	}
	if store.input.MerchantID != "" {
		t.Fatalf("CreateResource was called despite invalid tag: %#v", store.input)
	}
}

func TestCreateResourceFreePublishDoesNotConsumePublishQuota(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-job-seeking",
			TypeCode:       "job_seeking",
			Direction:      model.ResourceDirectionDemand,
			RequiredFields: []string{"title", "category", "contactPhone"},
			CommercialRules: model.JSONMap{
				"publish":       model.JSONMap{"mode": model.ResourcePublishModeFree},
				"contactUnlock": model.JSONMap{"mode": model.ContactUnlockModePaidOrVIP, "priceCent": int64(500), "currency": "CNY", "repeatUnlockDays": int64(30)},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "job_seeking",
		Title:       "本人找平车岗位",
		Category:    "平车",
		Description: "三年经验，可长期稳定做。",
		Contact:     ResourceContactReq{Name: "李师傅", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	if store.input.ConsumePublishQuota {
		t.Fatalf("consumePublishQuota = true, want false for free publish category")
	}
}

func TestCreateResourceBlocksRiskyContentAudit(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result:     model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
		userOpenID: "openid-1",
	}
	auditor := &fakeContentAuditor{result: ContentAuditResult{Decision: ContentAuditDecisionRisky, Labels: []string{"20006"}}}
	logic := NewCreateResourceLogic(store, auditor)

	resp, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:    "merchant-1",
		CityCode:      "zhili",
		TypeCode:      "inventory",
		Title:         "女童春款卫衣库存整包清",
		Category:      "童装",
		QuantityText:  "3200 件",
		Description:   "整包优先，可现场看货。",
		Contact:       ResourceContactReq{Name: "张老板", Phone: "13800000000"},
		CreatedByUser: "user-1",
		CreatedByRole: "merchant_admin",
	})

	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	if resp.Status != model.ResourceStatusRejected || store.rejectedResourceID != "resource-1" {
		t.Fatalf("resp = %#v rejectedResourceID = %q, want rejected resource", resp, store.rejectedResourceID)
	}
	if auditor.input.OpenID != "openid-1" || auditor.input.Title != "女童春款卫衣库存整包清" {
		t.Fatalf("audit input = %#v, want openid and title", auditor.input)
	}
	if store.rejectReason != "内容疑似包含违法违规信息，请修改后重新提交" {
		t.Fatalf("rejectReason = %q, want mapped WeChat label reason", store.rejectReason)
	}
}

func TestCreateResourceQueuesRetryWhenAuditDependencyFails(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result:     model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
		userOpenID: "openid-1",
	}
	auditor := &fakeContentAuditor{err: errors.New("wechat unavailable")}
	logic := NewCreateResourceLogic(store, auditor)

	resp, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:    "merchant-1",
		CityCode:      "zhili",
		TypeCode:      "inventory",
		Title:         "女童春款卫衣库存整包清",
		Category:      "童装",
		QuantityText:  "3200 件",
		Description:   "整包优先，可现场看货。",
		Contact:       ResourceContactReq{Name: "张老板", Phone: "13800000000"},
		CreatedByUser: "user-1",
		CreatedByRole: "merchant_admin",
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	if resp.Status != model.ResourceStatusAuditRetry || store.auditRetryResourceID != "resource-1" {
		t.Fatalf("resp = %#v retryResourceID = %q, want audit retry after dependency failure", resp, store.auditRetryResourceID)
	}
	if store.auditRetryReason != "wechat unavailable" {
		t.Fatalf("auditRetryReason = %q, want dependency error recorded for retry", store.auditRetryReason)
	}
}

func TestCreateResourceRejectsLegacyWechatAuditIdentity(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result:     model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
		userOpenID: "dev:local-dev-123",
	}
	logic := NewCreateResourceLogic(store, &fakeContentAuditor{err: ErrLegacyWechatAuditIdentity})

	resp, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:    "merchant-1",
		CityCode:      "zhili",
		TypeCode:      "inventory",
		Title:         "女童春款卫衣库存整包清",
		Category:      "童装",
		QuantityText:  "3200 件",
		Description:   "整包优先，可现场看货。",
		Contact:       ResourceContactReq{Name: "张老板", Phone: "13800000000"},
		CreatedByUser: "user-1",
		CreatedByRole: "merchant_admin",
	})

	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	if resp.Status != model.ResourceStatusRejected || store.rejectedResourceID != "resource-1" {
		t.Fatalf("resp = %#v rejectedResourceID = %q, want rejected resource", resp, store.rejectedResourceID)
	}
	if resp.Message != "微信登录身份已失效，请重新登录后重新编辑并提交" || store.rejectReason != resp.Message {
		t.Fatalf("resp = %#v rejectReason = %q, want re-login guidance", resp, store.rejectReason)
	}
}

func TestCreateResourceQueuesRetryWhenCreatedByUserIsMissing(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store, &fakeContentAuditor{})

	resp, err := logic.CreateResource(context.Background(), CreateResourceReq{
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
		t.Fatalf("CreateResource() error = %v", err)
	}
	if resp.Status != model.ResourceStatusAuditRetry || store.auditRetryResourceID != "resource-1" {
		t.Fatalf("resp = %#v retryResourceID = %q, want automatic audit retry", resp, store.auditRetryResourceID)
	}
}

func TestCreateResourceQueuesRetryWhenOpenIDIsMissing(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store, &fakeContentAuditor{})

	resp, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:    "merchant-1",
		CityCode:      "zhili",
		TypeCode:      "inventory",
		Title:         "女童春款卫衣库存整包清",
		Category:      "童装",
		QuantityText:  "3200 件",
		Description:   "整包优先，可现场看货。",
		Contact:       ResourceContactReq{Name: "张老板", Phone: "13800000000"},
		CreatedByUser: "user-1",
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	if resp.Status != model.ResourceStatusAuditRetry || store.auditRetryResourceID != "resource-1" {
		t.Fatalf("resp = %#v retryResourceID = %q, want automatic audit retry", resp, store.auditRetryResourceID)
	}
}

func TestCreateResourceRejectsDisabledPublishCategory(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-disabled",
			TypeCode:       "job_seeking",
			RequiredFields: []string{"title", "category", "contactPhone"},
			CommercialRules: model.JSONMap{
				"publish": model.JSONMap{"mode": model.ResourcePublishModeDisabled},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "job_seeking",
		Title:       "本人找平车岗位",
		Category:    "平车",
		Description: "三年经验，可长期稳定做。",
		Contact:     ResourceContactReq{Name: "李师傅", Phone: "13800000000"},
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "该分类暂不开放发布" {
		t.Fatalf("message = %q, want disabled publish message", errx.PublicMessage(err))
	}
	if store.input.MerchantID != "" {
		t.Fatalf("CreateResource was called for disabled publish category: %#v", store.input)
	}
}

func TestCreateResourceDraftDoesNotConsumePublishQuota(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusDraft},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResourceDraft(context.Background(), CreateResourceReq{
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
	if store.input.ConsumePublishQuota {
		t.Fatalf("consumePublishQuota = true, want false for draft")
	}
}

func TestCreateResourceMapsPublishQuotaInsufficient(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-1",
			TypeCode:       "inventory",
			RequiredFields: []string{"title", "category", "quantityText", "contactPhone"},
		},
		createErr: model.ErrPublishQuotaInsufficient,
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
	if errx.CodeOf(err) != errx.CodeQuotaNotEnough {
		t.Fatalf("error code = %q, want quota not enough", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "本月免费发布次数已用完，可购买发布次数后继续发布" {
		t.Fatalf("message = %q, want publish quota upsell message", errx.PublicMessage(err))
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

func TestCreateResourceDerivesSummaryFieldsFromDisplayTemplate(t *testing.T) {
	store := &fakeCreateResourceStore{
		config: model.ResourcePublishConfig{
			ID:             "config-find-rental",
			TypeCode:       "find_rental",
			Direction:      model.ResourceDirectionDemand,
			RequiredFields: []string{"title", "rentalNeedType", "contactPhone"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "rentalNeedType", "label": "标的类型", "type": "select", "options": []interface{}{"档口", "仓库"}, "allowCustom": false},
					map[string]interface{}{"key": "expectedAreaText", "label": "期望面积", "type": "text"},
					map[string]interface{}{"key": "budgetRentText", "label": "预算租金", "type": "text"},
				},
			},
			DisplayTemplate: model.JSONMap{
				"summary": map[string]interface{}{
					"category":     "rentalNeedType",
					"quantityText": "expectedAreaText",
					"priceText":    "budgetRentText",
				},
			},
		},
		result: model.CreateResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
	}
	logic := NewCreateResourceLogic(store)

	_, err := logic.CreateResource(context.Background(), CreateResourceReq{
		MerchantID:  "merchant-1",
		CityCode:    "zhili",
		TypeCode:    "find_rental",
		Title:       "急找童装城附近档口",
		Attributes:  model.JSONMap{"rentalNeedType": "档口", "expectedAreaText": "80-120 平", "budgetRentText": "8000 元/月以内"},
		Description: "希望可以两周内入驻。",
		Contact:     ResourceContactReq{Name: "李老板", Phone: "13800000000"},
	})
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	if store.input.Category != "档口" {
		t.Fatalf("category = %q, want derived rental type", store.input.Category)
	}
	if store.input.QuantityText != "80-120 平" {
		t.Fatalf("quantityText = %q, want derived area", store.input.QuantityText)
	}
	if store.input.PriceText != "8000 元/月以内" {
		t.Fatalf("priceText = %q, want derived rent budget", store.input.PriceText)
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
		Description:  "整包优先，可现场看货。",
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
		MerchantID:        "merchant-1",
		CityCode:          "zhili",
		TypeCode:          "inventory",
		Title:             "女童春款卫衣库存整包清",
		Category:          "童装",
		QuantityText:      "3200 件",
		Description:       "整包优先，可现场看货。",
		Contact:           ResourceContactReq{Name: "张老板", Phone: "13800000000"},
		CreatedByOperator: "100000000000000001",
		CreatedByRole:     "platform_operator",
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
	userOpenID           string
	openIDUserID         string
	createErr            error
	auditTasks           []model.ResourceContentAuditTaskInput
	publishedResourceID  string
	rejectedResourceID   string
	rejectReason         string
	publishErr           error
	rejectErr            error
	auditRetryResourceID string
	auditRetryReason     string
}

func (s *fakeCreateResourceStore) MarkResourceAuditRetry(ctx context.Context, resourceID string, reason string) (int64, error) {
	s.auditRetryResourceID = resourceID
	s.auditRetryReason = reason
	return 0, nil
}

func (s *fakeCreateResourceStore) MarkResourceManualReview(ctx context.Context, resourceID string, reason string) error {
	return nil
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

func (s *fakeCreateResourceStore) GetUserWechatOpenID(ctx context.Context, userID string) (string, error) {
	s.openIDUserID = userID
	return s.userOpenID, nil
}

func (s *fakeCreateResourceStore) CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error) {
	s.input = input
	if s.createErr != nil {
		return model.CreateResourceResult{}, s.createErr
	}
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

func (s *fakeCreateResourceStore) CreateResourceContentAuditTasks(ctx context.Context, resourceID string, tasks []model.ResourceContentAuditTaskInput) error {
	s.auditTasks = append([]model.ResourceContentAuditTaskInput(nil), tasks...)
	return nil
}

func (s *fakeCreateResourceStore) PublishResourceAfterAudit(ctx context.Context, resourceID string) (model.ReviewResourceResult, error) {
	s.publishedResourceID = resourceID
	if s.publishErr != nil {
		return model.ReviewResourceResult{}, s.publishErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeCreateResourceStore) RejectResourceAfterAudit(ctx context.Context, resourceID string, reason string) (model.ReviewResourceResult, error) {
	s.rejectedResourceID = resourceID
	s.rejectReason = reason
	if s.rejectErr != nil {
		return model.ReviewResourceResult{}, s.rejectErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusRejected}, nil
}

func (s *fakeCreateResourceStore) SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error) {
	return model.SubmitResourceResult{ID: resourceID, Status: "pending"}, nil
}

type fakeContentAuditor struct {
	input  ContentAuditInput
	result ContentAuditResult
	err    error
}

func (a *fakeContentAuditor) AuditResource(ctx context.Context, input ContentAuditInput) (ContentAuditResult, error) {
	a.input = input
	return a.result, a.err
}
