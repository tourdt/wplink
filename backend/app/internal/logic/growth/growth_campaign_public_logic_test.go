package growth

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListActiveGrowthCampaignsReturnsPublicRules(t *testing.T) {
	store := &fakeGrowthCampaignPublicStore{
		campaigns: []model.PublicGrowthCampaign{{
			Code:  "starter_growth_2026_q3",
			Name:  "新手成长权益活动",
			Title: "新手发布权益",
			Hint:  "发布优质资源、有效分享可获得更多曝光权益",
			Rules: []model.PublicGrowthRule{{
				RuleName:     "首条资源审核通过奖励",
				TriggerEvent: model.GrowthEventResourceFirstApproved,
				RewardType:   model.EntitlementTypePublishQuota,
				RewardAmount: 5,
				ValidDays:    30,
			}},
		}},
	}
	logic := NewCampaignPublicLogic(store)

	resp, err := logic.ListActiveCampaigns(context.Background())
	if err != nil {
		t.Fatalf("ListActiveCampaigns() error = %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "新手发布权益" {
		t.Fatalf("resp = %#v, want starter campaign with public title", resp)
	}
	if len(resp.Items[0].Rules) != 1 || resp.Items[0].Rules[0].RewardText != "5 次发布次数" {
		t.Fatalf("rules = %#v, want Chinese reward text", resp.Items[0].Rules)
	}
}

type fakeGrowthCampaignPublicStore struct {
	campaigns []model.PublicGrowthCampaign
}

func (s *fakeGrowthCampaignPublicStore) ListActiveGrowthCampaigns(ctx context.Context) ([]model.PublicGrowthCampaign, error) {
	return s.campaigns, nil
}
