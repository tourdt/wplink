package resource

import (
	"context"
	"database/sql"
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
			ID: "resource-1", Status: "published", TypeCode: "inventory", Title: "库存资源",
			TypeName:   "库存清仓",
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
	if resp.TypeName != "库存清仓" {
		t.Fatalf("typeName = %q, want resource type display name", resp.TypeName)
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

func TestGetResourceMapsMissingPublishedResourceToNotFound(t *testing.T) {
	logic := NewGetResourceLogic(&fakeGetResourceStore{err: sql.ErrNoRows})

	_, err := logic.GetResource(context.Background(), "resource-missing")

	if errx.CodeOf(err) != errx.CodeResourceNotFound {
		t.Fatalf("error code = %q, want resource not found", errx.CodeOf(err))
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
