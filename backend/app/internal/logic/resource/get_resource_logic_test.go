package resource

import (
	"context"
	"database/sql"
	"errors"
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
			Attributes: model.JSONMap{"season": "春款", "allowLiveSale": true, "internalNote": "不展示"},
			FieldSchema: model.JSONMap{
				"fields": []interface{}{
					map[string]interface{}{"key": "season", "label": "季节"},
					map[string]interface{}{"key": "allowLiveSale", "label": "支持直播"},
				},
			},
			DisplayTemplate: model.JSONMap{"detail": []interface{}{"season", "allowLiveSale"}},
			MerchantID:      "merchant-1", MerchantName: "织里样板童装厂", MerchantVerificationStatus: "verified", MerchantVIPStatus: model.VIPStatusActive,
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
	if len(resp.AttributeItems) != 2 {
		t.Fatalf("attributeItems = %#v, want two display attributes", resp.AttributeItems)
	}
	if resp.AttributeItems[0].Label != "季节" || resp.AttributeItems[0].Value != "春款" {
		t.Fatalf("first attribute item = %#v, want season label/value", resp.AttributeItems[0])
	}
	if resp.AttributeItems[1].Value != "是" {
		t.Fatalf("boolean attribute value = %q, want 是", resp.AttributeItems[1].Value)
	}
}

func TestGetResourceDisplaysAddressAttributeText(t *testing.T) {
	store := &fakeGetResourceStore{
		detail: model.ResourceDetail{
			ID: "resource-1", Status: "published", TypeCode: "shop_office_rental", Title: "档口出租",
			TypeName: "商铺/办公出租",
			Attributes: model.JSONMap{
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
			DisplayTemplate: model.JSONMap{"detail": []interface{}{"locationText"}},
			MerchantID:      "merchant-1", MerchantName: "织里档口",
			ContactName: "王老板", PhoneMasked: "138****0000",
		},
	}
	logic := NewGetResourceLogic(store)

	resp, err := logic.GetResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}

	if len(resp.AttributeItems) != 1 {
		t.Fatalf("attributeItems = %#v, want one address attribute", resp.AttributeItems)
	}
	if resp.AttributeItems[0].Label != "位置" || resp.AttributeItems[0].Value != "浙江省湖州市吴兴区织里镇童装城" {
		t.Fatalf("address attribute item = %#v, want display address text", resp.AttributeItems[0])
	}
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
