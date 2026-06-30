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
			MerchantID: "merchant-1", MerchantName: "织里样板童装厂", MerchantVerificationStatus: "verified",
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
	if resp.Contact.PhoneMasked != "138****0000" {
		t.Fatalf("phone = %q, want masked phone", resp.Contact.PhoneMasked)
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
