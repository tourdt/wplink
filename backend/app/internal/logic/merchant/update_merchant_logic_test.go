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
	logic := NewUpdateMerchantLogic(store)

	resp, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		MainCategories: []string{"童装", "卫衣"},
		Description:    "支持小单快反",
		Images:         []string{"https://example.com/factory.jpg"},
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
	if resp.UpdatedAt != "2026-06-27T10:00:00+08:00" {
		t.Fatalf("updatedAt = %q, want fixed time", resp.UpdatedAt)
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
