package growth

import (
	"context"
	"fmt"

	"wplink/backend/app/internal/model"
)

type CampaignPublicStore interface {
	ListActiveGrowthCampaigns(ctx context.Context) ([]model.PublicGrowthCampaign, error)
}

type CampaignPublicLogic struct {
	store CampaignPublicStore
}

type PublicGrowthCampaignItem struct {
	Code  string                 `json:"code"`
	Name  string                 `json:"name"`
	Title string                 `json:"title"`
	Hint  string                 `json:"hint,omitempty"`
	Rules []PublicGrowthRuleItem `json:"rules"`
}

type PublicGrowthRuleItem struct {
	RuleCode              string        `json:"ruleCode"`
	RuleName              string        `json:"ruleName"`
	TriggerEvent          string        `json:"triggerEvent"`
	RewardType            string        `json:"rewardType"`
	RewardAmount          int64         `json:"rewardAmount"`
	RewardText            string        `json:"rewardText"`
	ValidDays             int64         `json:"validDays"`
	PerUserLimit          int64         `json:"perUserLimit,omitempty"`
	PerUserDailyLimit     int64         `json:"perUserDailyLimit,omitempty"`
	PerResourceDailyLimit int64         `json:"perResourceDailyLimit,omitempty"`
	Description           string        `json:"description,omitempty"`
	Conditions            model.JSONMap `json:"conditions,omitempty"`
}

type ListActiveGrowthCampaignsResp struct {
	Items []PublicGrowthCampaignItem `json:"items"`
}

func NewCampaignPublicLogic(store CampaignPublicStore) *CampaignPublicLogic {
	return &CampaignPublicLogic{store: store}
}

func (l *CampaignPublicLogic) ListActiveCampaigns(ctx context.Context) (ListActiveGrowthCampaignsResp, error) {
	campaigns, err := l.store.ListActiveGrowthCampaigns(ctx)
	if err != nil {
		return ListActiveGrowthCampaignsResp{}, err
	}
	items := make([]PublicGrowthCampaignItem, 0, len(campaigns))
	for _, campaign := range campaigns {
		item := PublicGrowthCampaignItem{
			Code:  campaign.Code,
			Name:  campaign.Name,
			Title: campaign.Title,
			Hint:  campaign.Hint,
			Rules: make([]PublicGrowthRuleItem, 0, len(campaign.Rules)),
		}
		for _, rule := range campaign.Rules {
			item.Rules = append(item.Rules, PublicGrowthRuleItem{
				RuleCode:              rule.RuleCode,
				RuleName:              rule.RuleName,
				TriggerEvent:          rule.TriggerEvent,
				RewardType:            rule.RewardType,
				RewardAmount:          rule.RewardAmount,
				RewardText:            growthRewardText(rule.RewardType, rule.RewardAmount),
				ValidDays:             rule.ValidDays,
				PerUserLimit:          rule.PerUserLimit,
				PerUserDailyLimit:     rule.PerUserDailyLimit,
				PerResourceDailyLimit: rule.PerResourceDailyLimit,
				Description:           rule.Description,
				Conditions:            rule.Conditions,
			})
		}
		items = append(items, item)
	}
	return ListActiveGrowthCampaignsResp{Items: items}, nil
}

func growthRewardText(rewardType string, amount int64) string {
	switch rewardType {
	case model.EntitlementTypePublishQuota:
		return fmt.Sprintf("%d 次发布次数", amount)
	case model.EntitlementTypeRefreshQuota:
		return fmt.Sprintf("%d 次刷新次数", amount)
	default:
		return fmt.Sprintf("%d 次权益", amount)
	}
}
