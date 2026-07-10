package growth

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestProcessGrowthEventTrimsAndPassesInput(t *testing.T) {
	store := &fakeGrowthStore{}
	logic := NewRewardLogic(store)

	err := logic.ProcessEvent(context.Background(), model.GrowthEventInput{
		EventType:  " user_first_login ",
		MerchantID: " merchant-1 ",
		UserID:     " user-1 ",
	})
	if err != nil {
		t.Fatalf("ProcessEvent() error = %v", err)
	}
	if store.input.EventType != "user_first_login" || store.input.MerchantID != "merchant-1" || store.input.UserID != "user-1" {
		t.Fatalf("input = %#v, want trimmed growth event", store.input)
	}
}

func TestProcessGrowthEventSkipsMissingStore(t *testing.T) {
	logic := NewRewardLogic(nil)
	if err := logic.ProcessEvent(context.Background(), model.GrowthEventInput{EventType: "user_first_login"}); err != nil {
		t.Fatalf("ProcessEvent() error = %v, want nil when store missing", err)
	}
}

func TestProcessGrowthEventReturnsStoreError(t *testing.T) {
	logic := NewRewardLogic(&fakeGrowthStore{err: errors.New("db down")})
	err := logic.ProcessEvent(context.Background(), model.GrowthEventInput{EventType: "user_first_login", MerchantID: "merchant-1"})
	if err == nil {
		t.Fatal("ProcessEvent() error = nil, want store error")
	}
}

type fakeGrowthStore struct {
	input model.GrowthEventInput
	err   error
}

func (s *fakeGrowthStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.input = input
	return nil, s.err
}
