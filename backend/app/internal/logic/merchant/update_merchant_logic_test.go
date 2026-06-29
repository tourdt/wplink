package merchant

import (
	"context"
	"database/sql"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestUpdateMerchantRejectsEmptyID(t *testing.T) {
	logic := NewUpdateMerchantLogic(&fakeMerchantUpdateStore{})

	_, err := logic.UpdateMerchant(context.Background(), " ", UpdateMerchantReq{})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestUpdateMerchantPassesPatchToStore(t *testing.T) {
	store := &fakeMerchantUpdateStore{updatedAt: "2026-06-27T10:00:00+08:00"}
	verifier := &fakeMerchantSMSVerifier{}
	logic := NewUpdateMerchantLogic(store, verifier)

	resp, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		MainCategories: []string{"童装", "卫衣"},
		MerchantType:   " stockist ",
		Description:    "支持小单快反",
		LogoURL:        "https://example.com/logo.png",
		Images:         []string{"https://example.com/factory.jpg"},
		ContactName:    "李厂长",
		ContactPhone:   "18800000001",
		ContactWechat:  "factory-demo",
		AddressText:    "织里镇利济路88号",
		Location:       model.JSONMap{"latitude": 30.1, "longitude": 120.2, "name": "织里童装城", "address": "织里镇利济路88号"},
		SmsCode:        "123456",
	})
	if err != nil {
		t.Fatalf("UpdateMerchant() error = %v", err)
	}

	if store.merchantID != "merchant-1" {
		t.Fatalf("merchantID = %q, want merchant-1", store.merchantID)
	}
	if store.patch.Description != "支持小单快反" {
		t.Fatalf("description = %q, want updated description", store.patch.Description)
	}
	if store.patch.MerchantType != "stockist" {
		t.Fatalf("merchantType = %q, want trimmed stockist", store.patch.MerchantType)
	}
	if store.patch.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("logoURL = %q, want updated logo URL", store.patch.LogoURL)
	}
	if store.patch.ContactName != "李厂长" || store.patch.ContactPhone != "18800000001" || store.patch.ContactWechat != "factory-demo" {
		t.Fatalf("contact patch = %#v, want contact name/phone/wechat", store.patch)
	}
	if store.patch.AddressText != "织里镇利济路88号" {
		t.Fatalf("addressText = %q, want updated merchant address", store.patch.AddressText)
	}
	if store.patch.Location["name"] != "织里童装城" || store.patch.Location["address"] != "织里镇利济路88号" {
		t.Fatalf("location = %#v, want updated map location", store.patch.Location)
	}
	if verifier.phone != "18800000001" || verifier.code != "123456" {
		t.Fatalf("sms verifier = %q/%q, want contact phone and sms code", verifier.phone, verifier.code)
	}
	if resp.UpdatedAt != "2026-06-27T10:00:00+08:00" {
		t.Fatalf("updatedAt = %q, want fixed time", resp.UpdatedAt)
	}
}

func TestUpdateMerchantRequiresSMSCodeForContactPhone(t *testing.T) {
	store := &fakeMerchantUpdateStore{updatedAt: "2026-06-27T10:00:00+08:00"}
	logic := NewUpdateMerchantLogic(store, &fakeMerchantSMSVerifier{})

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		MainCategories: []string{"童装"},
		ContactPhone:   "18800000001",
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if store.merchantID != "" {
		t.Fatalf("store was called for invalid contact phone update")
	}
}

func TestUpdateMerchantMapsMissingMerchant(t *testing.T) {
	store := &fakeMerchantUpdateStore{err: sql.ErrNoRows}
	logic := NewUpdateMerchantLogic(store)

	_, err := logic.UpdateMerchant(context.Background(), "merchant-missing", UpdateMerchantReq{
		MainCategories: []string{"童装"},
	})

	if errx.CodeOf(err) != errx.CodeResourceNotFound {
		t.Fatalf("error code = %q, want resource not found", errx.CodeOf(err))
	}
}

type fakeMerchantUpdateStore struct {
	merchantID string
	patch      model.UpdateMerchantPatch
	updatedAt  string
	err        error
}

func (s *fakeMerchantUpdateStore) UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error) {
	s.merchantID = merchantID
	s.patch = patch
	if s.err != nil {
		return "", s.err
	}
	return s.updatedAt, nil
}

type fakeMerchantSMSVerifier struct {
	phone string
	code  string
	err   error
}

func (s *fakeMerchantSMSVerifier) VerifySMSCode(ctx context.Context, phone string, code string) error {
	s.phone = phone
	s.code = code
	return s.err
}
