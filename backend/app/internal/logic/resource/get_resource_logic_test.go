package resource

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestGetResourceRejectsEmptyID(t *testing.T) {
	logic := NewGetResourceLogic(&fakeGetResourceStore{})

	_, err := logic.GetResource(context.Background(), " ")

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestGetResourceReturnsPublishedDetail(t *testing.T) {
	store := &fakeGetResourceStore{
		detail: model.ResourceDetail{
			ID: "resource-1", Status: "published", TypeCode: "stock_clearance", Title: "尾货资源",
			Direction:  model.ResourceDirectionSupply,
			TypeName:   "库存出售",
			MerchantID: "merchant-1", MerchantName: "织里样板童装厂", MerchantVIPStatus: model.VIPStatusActive,
			ContactName: "张老板", PhoneMasked: "138****0000", WechatMasked: "zhili_****",
		},
	}
	logic := NewGetResourceLogic(store)

	resp, err := logic.GetResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}

	if resp.Merchant.Name != "织里样板童装厂" {
		t.Fatalf("merchant = %#v, want merchant detail", resp.Merchant)
	}
	if resp.Merchant.VIPStatus != model.VIPStatusActive {
		t.Fatalf("merchant vipStatus = %q, want active", resp.Merchant.VIPStatus)
	}
	if resp.Contact.PhoneMasked != "138****0000" {
		t.Fatalf("phone = %q, want masked phone", resp.Contact.PhoneMasked)
	}
	if resp.TypeName != "库存出售" {
		t.Fatalf("typeName = %q, want resource type display name", resp.TypeName)
	}
	if resp.Direction != model.ResourceDirectionSupply {
		t.Fatalf("direction = %q, want supply", resp.Direction)
	}
}

func TestGetResourceBuildsSortedPresentationFieldsWithConfiguredLayouts(t *testing.T) {
	store := &fakeGetResourceStore{
		detail: model.ResourceDetail{
			ID: "resource-1", Status: "published", TypeCode: "factory_direct", Direction: model.ResourceDirectionSupply,
			CityCode: "zhili", District: "织里镇",
			QuantityText: " 100 件 ",
			Attributes: model.JSONMap{
				"season":        " 春季 ",
				"allowLiveSale": true,
			},
			DisplayTemplate: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"source": "district", "label": "所在区域", "role": "detail", "layout": "half", "order": float64(15)},
					map[string]interface{}{"source": "cityCode", "label": "城市编码", "role": "detail", "layout": "half", "order": float64(5)},
					map[string]interface{}{"source": "allowLiveSale", "label": "支持直播", "role": "detail", "layout": "full", "order": float64(30)},
					map[string]interface{}{"source": "season", "label": "季节", "role": "detail", "layout": "half", "order": float64(20)},
					map[string]interface{}{"source": "quantityText", "label": "起批量", "role": "core", "layout": "half", "order": float64(10)},
				},
			},
		},
	}
	logic := NewGetResourceLogic(store)

	resp, err := logic.GetResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}

	want := []ResourcePresentationField{
		{Key: "cityCode", Label: "城市编码", Value: "zhili", Layout: "half"},
		{Key: "quantityText", Label: "起批量", Value: "100 件", Layout: "half"},
		{Key: "district", Label: "所在区域", Value: "织里镇", Layout: "half"},
		{Key: "season", Label: "季节", Value: "春季", Layout: "half"},
		{Key: "allowLiveSale", Label: "支持直播", Value: "是", Layout: "full"},
	}
	if !presentationFieldsEqual(resp.Presentation.Fields, want) {
		t.Fatalf("presentation.fields = %#v, want %#v", resp.Presentation.Fields, want)
	}
}

func TestGetResourcePresentationPrefersTrustedTopLevelValuesOverCollidingAttributes(t *testing.T) {
	store := &fakeGetResourceStore{detail: model.ResourceDetail{
		ID: "resource-1", MerchantID: "merchant-trusted", CityCode: "zhili", District: "织里镇", Title: "可信标题",
		Attributes: model.JSONMap{
			"merchantId": "merchant-forged",
			"cityCode":   "forged-city",
			"district":   "伪造区域",
			"title":      "伪造标题",
			"season":     "春季",
		},
		DisplayTemplate: model.JSONMap{"fields": []interface{}{
			map[string]interface{}{"source": "merchantId", "label": "商家", "role": "detail", "layout": "half", "order": float64(10)},
			map[string]interface{}{"source": "cityCode", "label": "城市", "role": "detail", "layout": "half", "order": float64(20)},
			map[string]interface{}{"source": "district", "label": "区域", "role": "detail", "layout": "half", "order": float64(30)},
			map[string]interface{}{"source": "title", "label": "标题", "role": "detail", "layout": "full", "order": float64(40)},
			map[string]interface{}{"source": "season", "label": "季节", "role": "detail", "layout": "half", "order": float64(50)},
		}},
	}}
	logic := NewGetResourceLogic(store)

	resp, err := logic.GetResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}

	want := []ResourcePresentationField{
		{Key: "merchantId", Label: "商家", Value: "merchant-trusted", Layout: "half"},
		{Key: "cityCode", Label: "城市", Value: "zhili", Layout: "half"},
		{Key: "district", Label: "区域", Value: "织里镇", Layout: "half"},
		{Key: "title", Label: "标题", Value: "可信标题", Layout: "full"},
		{Key: "season", Label: "季节", Value: "春季", Layout: "half"},
	}
	if !presentationFieldsEqual(resp.Presentation.Fields, want) {
		t.Fatalf("presentation.fields = %#v, want trusted top-level values and dynamic attribute %#v", resp.Presentation.Fields, want)
	}
}

func TestGetResourceOmitsDuplicateEmptyAndAddressPresentationFields(t *testing.T) {
	store := &fakeGetResourceStore{
		detail: model.ResourceDetail{
			ID: "resource-1", Status: "published", TypeCode: "shop_office_rental", Title: "档口出租", TypeName: "商铺/办公出租",
			Attributes: model.JSONMap{
				"rentText":  "3000 元/月",
				"emptyText": "  ",
				"locationText": model.JSONMap{
					"address":   "浙江省湖州市吴兴区织里镇童装城",
					"name":      "织里童装城",
					"latitude":  30.8732,
					"longitude": 120.2255,
				},
			},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "locationText", "label": "位置", "type": "address"},
				},
			},
			DisplayTemplate: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"source": "rentText", "label": "租金", "role": "core_price", "layout": "half", "order": float64(10)},
					map[string]interface{}{"source": "rentText", "label": "重复租金", "role": "detail", "layout": "full", "order": float64(20)},
					map[string]interface{}{"source": "emptyText", "label": "空字段", "role": "detail", "layout": "half", "order": float64(30)},
					map[string]interface{}{"source": "locationText", "label": "位置", "role": "detail", "layout": "full", "order": float64(40)},
				},
			},
			MerchantID: "merchant-1", MerchantName: "织里档口",
			ContactName: "王老板", PhoneMasked: "138****0000",
		},
	}
	logic := NewGetResourceLogic(store)

	resp, err := logic.GetResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}

	want := []ResourcePresentationField{{Key: "rentText", Label: "租金", Value: "3000 元/月", Layout: "half"}}
	if !presentationFieldsEqual(resp.Presentation.Fields, want) {
		t.Fatalf("presentation.fields = %#v, want only one non-address field %#v", resp.Presentation.Fields, want)
	}
}

func presentationFieldsEqual(got []ResourcePresentationField, want []ResourcePresentationField) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func TestGetResourceReturnsContactAccessWithoutRawContact(t *testing.T) {
	store := &fakeGetResourceStore{
		detail: model.ResourceDetail{
			ID: "resource-1", Status: model.ResourceStatusPublished, TypeCode: "job_seek", Title: "求职需求",
			TypeName: "我要求职", Category: "求职",
			Attributes: model.JSONMap{}, MerchantID: "merchant-1", MerchantName: "求职用户",
			ContactName: "李先生", PhoneMasked: "188****0002", WechatMasked: "",
			CommercialRules: model.JSONMap{
				"contactUnlock": model.JSONMap{
					"mode":             model.ContactUnlockModePaidOrVIP,
					"priceCent":        int64(500),
					"currency":         "CNY",
					"vipFree":          true,
					"repeatUnlockDays": int64(30),
				},
			},
		},
	}
	logic := NewGetResourceLogic(store)

	resp, err := logic.GetResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}

	if resp.ContactAccess.Mode != model.ContactUnlockModePaidOrVIP || resp.ContactAccess.PriceCent != 500 || resp.ContactAccess.Unlocked {
		t.Fatalf("contactAccess = %#v, want paid_or_vip locked", resp.ContactAccess)
	}
	if resp.Contact.PhoneMasked == "18800000002" || resp.Contact.WechatMasked == "stock-demo" {
		t.Fatalf("contact = %#v, public detail should only expose masked contact", resp.Contact)
	}
}

func TestGetResourceMapsMissingPublishedResourceToNotFound(t *testing.T) {
	logic := NewGetResourceLogic(&fakeGetResourceStore{err: sql.ErrNoRows})

	_, err := logic.GetResource(context.Background(), "resource-missing")

	if errx.CodeOf(err) != errx.CodeResourceNotFound {
		t.Fatalf("error code = %q, want resource not found", errx.CodeOf(err))
	}
}

func TestGetResourceReturnsFriendlyStoreError(t *testing.T) {
	logic := NewGetResourceLogic(&fakeGetResourceStore{
		err: errors.New("pq: column rtc.commercial_rules does not exist"),
	})

	_, err := logic.GetResource(context.Background(), "8030000000000000003")

	if err == nil {
		t.Fatal("GetResource() error = nil, want friendly internal error")
	}
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "资源详情加载失败，请稍后重试" {
		t.Fatalf("error code=%q message=%q, want friendly resource detail load failure", errx.CodeOf(err), errx.PublicMessage(err))
	}
}

func TestGetResourceReturnsFriendlyErrorForOverflowPresentationOrder(t *testing.T) {
	logic := NewGetResourceLogic(&fakeGetResourceStore{detail: model.ResourceDetail{
		ID: "resource-1", TypeCode: "factory_direct",
		Attributes: model.JSONMap{"season": "春季"},
		DisplayTemplate: model.JSONMap{"fields": []interface{}{
			map[string]interface{}{"source": "season", "label": "季节", "role": "detail", "layout": "half", "order": math.Exp2(63)},
		}},
	}})

	_, err := logic.GetResource(context.Background(), "resource-1")

	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "资源详情展示配置异常，请稍后重试" {
		t.Fatalf("GetResource() error = %v, want friendly presentation config error", err)
	}
}

type fakeGetResourceStore struct {
	resourceID string
	detail     model.ResourceDetail
	err        error
}

func (s *fakeGetResourceStore) GetPublishedResourceDetail(ctx context.Context, resourceID string) (model.ResourceDetail, error) {
	s.resourceID = resourceID
	if s.err != nil {
		return model.ResourceDetail{}, s.err
	}
	return s.detail, nil
}
