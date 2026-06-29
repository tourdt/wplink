package merchant

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestCreateMerchantRejectsMissingRequiredFields(t *testing.T) {
	logic := NewCreateMerchantLogic(&fakeMerchantStore{})

	_, err := logic.CreateMerchant(context.Background(), CreateMerchantReq{
		CityCode: "zhili",
		Name:     "织里样板童装厂",
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestCreateMerchantPassesInputToStore(t *testing.T) {
	store := &fakeMerchantStore{
		createResult: model.CreateMerchantResult{
			ID:                 "merchant-1",
			Name:               "织里样板童装厂",
			VerificationStatus: "unverified",
			Status:             "active",
		},
	}
	logic := NewCreateMerchantLogic(store)

	resp, err := logic.CreateMerchant(context.Background(), CreateMerchantReq{
		CityCode:       " zhili ",
		Name:           "织里样板童装厂",
		MerchantType:   "factory",
		MainCategories: []string{"童装", "卫衣"},
		AddressText:    "湖州织里镇",
		Description:    "支持小单快反",
	})
	if err != nil {
		t.Fatalf("CreateMerchant() error = %v", err)
	}

	if store.createInput.CityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.createInput.CityCode)
	}
	if resp.ID != "merchant-1" || resp.Status != "active" {
		t.Fatalf("response = %#v, want created merchant", resp)
	}
	if store.createInput.ContactName != "" || store.createInput.ContactPhone != "" || store.createInput.ContactWechat != "" {
		t.Fatalf("contact input = %#v, want optional contact fields empty", store.createInput)
	}
}

type fakeMerchantStore struct {
	createInput  model.CreateMerchantInput
	createResult model.CreateMerchantResult
}

func (s *fakeMerchantStore) CreateMerchant(ctx context.Context, input model.CreateMerchantInput) (model.CreateMerchantResult, error) {
	s.createInput = input
	return s.createResult, nil
}
