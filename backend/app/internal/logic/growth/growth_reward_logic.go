package growth

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type RewardStore interface {
	TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
}

type RewardLogic struct {
	store RewardStore
}

func NewRewardLogic(store RewardStore) *RewardLogic {
	return &RewardLogic{store: store}
}

func (l *RewardLogic) ProcessEvent(ctx context.Context, input model.GrowthEventInput) error {
	if l == nil || l.store == nil {
		return nil
	}
	input.EventType = strings.TrimSpace(input.EventType)
	input.MerchantID = strings.TrimSpace(input.MerchantID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.EventID = strings.TrimSpace(input.EventID)
	if input.EventType == "" {
		return nil
	}
	_, err := l.store.TriggerGrowthEvent(ctx, input)
	return err
}
