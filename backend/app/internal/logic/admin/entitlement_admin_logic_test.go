package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestGrantMerchantEntitlementRejectsInvalidAmount(t *testing.T) {
	logic := NewEntitlementAdminLogic(&fakeEntitlementAdminStore{})

	_, err := logic.GrantMerchantEntitlement(context.Background(), GrantEntitlementReq{
		MerchantID: "merchant-1", EntitlementType: "publish_quota", SourceType: "manual", TotalAmount: 0, Reason: "补发",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("GrantMerchantEntitlement() error = %v, want validation error", err)
	}
}

func TestGrantMerchantEntitlementPassesInputToStore(t *testing.T) {
	store := &fakeEntitlementAdminStore{result: model.GrantEntitlementResult{ID: "entitlement-1"}}
	logic := NewEntitlementAdminLogic(store)

	resp, err := logic.GrantMerchantEntitlement(context.Background(), GrantEntitlementReq{
		MerchantID: " merchant-1 ", EntitlementType: " refresh_quota ", SourceType: " manual ",
		TotalAmount: 10, Reason: " 活动补发 ",
	})
	if err != nil {
		t.Fatalf("GrantMerchantEntitlement() error = %v", err)
	}

	if store.input.MerchantID != "merchant-1" || store.input.Reason != "活动补发" || resp.ID != "entitlement-1" {
		t.Fatalf("input = %#v, resp = %#v", store.input, resp)
	}
}

type fakeEntitlementAdminStore struct {
	input  model.GrantEntitlementInput
	result model.GrantEntitlementResult
}

func (s *fakeEntitlementAdminStore) GrantMerchantEntitlement(ctx context.Context, input model.GrantEntitlementInput) (model.GrantEntitlementResult, error) {
	s.input = input
	return s.result, nil
}
