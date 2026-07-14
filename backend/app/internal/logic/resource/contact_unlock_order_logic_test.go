package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestCreateContactUnlockOrderUsesTokenUser(t *testing.T) {
	store := &fakeContactUnlockOrderStore{
		result: model.ContactUnlockOrderResult{
			ID:         "order-1",
			ResourceID: "resource-1",
			Status:     model.PaymentOrderStatusPending,
			PriceCent:  500,
			Currency:   "CNY",
		},
	}
	logic := NewCreateContactUnlockOrderLogic(store)

	resp, err := logic.CreateContactUnlockOrder(context.Background(), CreateContactUnlockOrderReq{
		ResourceID:       " resource-1 ",
		UserID:           " user-1 ",
		ViewerMerchantID: " merchant-2 ",
		Action:           "phone",
	})
	if err != nil {
		t.Fatalf("CreateContactUnlockOrder() error = %v", err)
	}
	if store.input.ResourceID != "resource-1" || store.input.UserID != "user-1" || store.input.ViewerMerchantID != "merchant-2" {
		t.Fatalf("input = %#v, want trimmed token user and viewer merchant", store.input)
	}
	if resp.OrderID != "order-1" || resp.Status != model.PaymentOrderStatusPending || resp.PriceCent != 500 {
		t.Fatalf("resp = %#v, want pending paid order", resp)
	}
}

func TestCreateContactUnlockOrderRejectsMissingLogin(t *testing.T) {
	logic := NewCreateContactUnlockOrderLogic(&fakeContactUnlockOrderStore{})

	_, err := logic.CreateContactUnlockOrder(context.Background(), CreateContactUnlockOrderReq{ResourceID: "resource-1", Action: "phone"})

	if errx.CodeOf(err) != errx.CodeUnauthorized {
		t.Fatalf("error code = %q, want unauthorized", errx.CodeOf(err))
	}
}

type fakeContactUnlockOrderStore struct {
	input  model.CreateContactUnlockOrderInput
	result model.ContactUnlockOrderResult
}

func (s *fakeContactUnlockOrderStore) CreateContactUnlockOrder(ctx context.Context, input model.CreateContactUnlockOrderInput) (model.ContactUnlockOrderResult, error) {
	s.input = input
	return s.result, nil
}
