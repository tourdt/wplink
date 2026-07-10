package growth

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestGetGrowthTasksBuildsStarterAndDailyTasks(t *testing.T) {
	store := &fakeGrowthTaskStore{
		campaigns: []model.PublicGrowthCampaign{{
			Code:  "starter_growth_2026_q3",
			Name:  "新手成长权益活动",
			Title: "新手发布权益",
			Hint:  "发布优质资源、有效分享可获得更多曝光权益",
			Rules: []model.PublicGrowthRule{
				{
					RuleCode:     "first_login_publish_quota",
					RuleName:     "首次登录赠送发布次数",
					TriggerEvent: model.GrowthEventUserFirstLogin,
					RewardType:   model.EntitlementTypePublishQuota,
					RewardAmount: 3,
					ValidDays:    30,
					PerUserLimit: 1,
				},
				{
					RuleCode:     "first_resource_approved_publish_quota",
					RuleName:     "首条资源审核通过奖励",
					TriggerEvent: model.GrowthEventResourceFirstApproved,
					RewardType:   model.EntitlementTypePublishQuota,
					RewardAmount: 5,
					ValidDays:    30,
					PerUserLimit: 1,
				},
				{
					RuleCode:     "three_approved_resources_publish_quota",
					RuleName:     "连续优质发布奖励发布次数",
					TriggerEvent: model.GrowthEventResourceApprovedCountReached,
					RewardType:   model.EntitlementTypePublishQuota,
					RewardAmount: 5,
					ValidDays:    30,
					PerUserLimit: 1,
					Conditions:   model.JSONMap{"approvedCount": float64(3), "windowDays": float64(30)},
				},
				{
					RuleCode:          "share_contact_refresh_quota",
					RuleName:          "分享带来有效联系奖励",
					TriggerEvent:      model.GrowthEventResourceShareEffectiveContact,
					RewardType:        model.EntitlementTypeRefreshQuota,
					RewardAmount:      1,
					ValidDays:         15,
					PerUserDailyLimit: 3,
				},
			},
		}},
		entitlements: []model.MerchantEntitlement{
			{Type: model.EntitlementTypePublishQuota, RemainingAmount: 4},
			{Type: model.EntitlementTypeRefreshQuota, RemainingAmount: 2},
		},
		progress: model.GrowthTaskProgress{
			TotalResourceCount:         1,
			PendingResourceCount:       1,
			PublishedResourceCount:     1,
			ApprovedWithinWindowCount:  1,
			TodayEffectiveContactCount: 1,
		},
		grants: []model.MerchantGrowthRewardGrant{{RuleCode: "first_login_publish_quota", Status: "granted"}},
	}

	resp, err := NewGrowthTaskLogic(store).GetGrowthTasks(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatalf("GetGrowthTasks() error = %v", err)
	}
	if resp.Summary.PublishQuotaRemaining != 4 || resp.Summary.RefreshQuotaRemaining != 2 {
		t.Fatalf("summary = %#v, want quota totals", resp.Summary)
	}
	if resp.Summary.StarterTotalCount != 3 || resp.Summary.StarterCompletedCount != 1 {
		t.Fatalf("starter summary = %#v, want one completed of three", resp.Summary)
	}
	if len(resp.Tasks) != 4 {
		t.Fatalf("tasks length = %d, want 4", len(resp.Tasks))
	}
	assertGrowthTask(t, resp.Tasks, "first_login_publish_quota", "starter", "granted", 1, 1, "去发布")
	assertGrowthTask(t, resp.Tasks, "first_resource_approved_publish_quota", "starter", "pending_review", 1, 1, "查看发布")
	assertGrowthTask(t, resp.Tasks, "three_approved_resources_publish_quota", "starter", "in_progress", 1, 3, "去发布")
	assertGrowthTask(t, resp.Tasks, "share_contact_refresh_quota", "daily", "in_progress", 1, 3, "去分享")
}

func TestGetGrowthTasksReturnsEmptyWhenNoActiveCampaign(t *testing.T) {
	store := &fakeGrowthTaskStore{
		entitlements: []model.MerchantEntitlement{{Type: model.EntitlementTypePublishQuota, RemainingAmount: 3}},
	}

	resp, err := NewGrowthTaskLogic(store).GetGrowthTasks(context.Background(), "merchant-1")
	if err != nil {
		t.Fatalf("GetGrowthTasks() error = %v", err)
	}
	if resp.Campaign.Code != "" || len(resp.Tasks) != 0 {
		t.Fatalf("resp = %#v, want empty campaign and tasks", resp)
	}
	if resp.Summary.PublishQuotaRemaining != 3 {
		t.Fatalf("summary = %#v, want entitlement balance even without activity", resp.Summary)
	}
}

func TestGetGrowthTasksDoesNotTreatDailyHistoricalGrantAsCompleted(t *testing.T) {
	store := &fakeGrowthTaskStore{
		campaigns: []model.PublicGrowthCampaign{{
			Code:  "starter_growth_2026_q3",
			Title: "新手发布权益",
			Rules: []model.PublicGrowthRule{{
				RuleCode:          "share_contact_refresh_quota",
				RuleName:          "分享带来有效联系奖励",
				TriggerEvent:      model.GrowthEventResourceShareEffectiveContact,
				RewardType:        model.EntitlementTypeRefreshQuota,
				RewardAmount:      1,
				ValidDays:         15,
				PerUserDailyLimit: 3,
			}},
		}},
		progress: model.GrowthTaskProgress{TodayEffectiveContactCount: 1},
		grants:   []model.MerchantGrowthRewardGrant{{RuleCode: "share_contact_refresh_quota", Status: "granted"}},
	}

	resp, err := NewGrowthTaskLogic(store).GetGrowthTasks(context.Background(), "merchant-1")
	if err != nil {
		t.Fatalf("GetGrowthTasks() error = %v", err)
	}
	assertGrowthTask(t, resp.Tasks, "share_contact_refresh_quota", "daily", "in_progress", 1, 3, "去分享")
}

func TestGetGrowthTasksKeepsStarterTasksWhenLegacyRuleLimitMissing(t *testing.T) {
	store := &fakeGrowthTaskStore{
		campaigns: []model.PublicGrowthCampaign{{
			Code:  "starter_growth_2026_q3",
			Title: "新手发布权益",
			Rules: []model.PublicGrowthRule{{
				RuleCode:     "first_resource_approved_publish_quota",
				RuleName:     "首条资源审核通过奖励",
				TriggerEvent: model.GrowthEventResourceFirstApproved,
				RewardType:   model.EntitlementTypePublishQuota,
				RewardAmount: 5,
				ValidDays:    30,
			}},
		}},
	}

	resp, err := NewGrowthTaskLogic(store).GetGrowthTasks(context.Background(), "merchant-1")
	if err != nil {
		t.Fatalf("GetGrowthTasks() error = %v", err)
	}
	assertGrowthTask(t, resp.Tasks, "first_resource_approved_publish_quota", "starter", "todo", 0, 1, "查看发布")
}

func TestGetGrowthTasksRejectsEmptyMerchantID(t *testing.T) {
	_, err := NewGrowthTaskLogic(&fakeGrowthTaskStore{}).GetGrowthTasks(context.Background(), " ")
	if err == nil || errx.PublicMessage(err) != "商家不存在" {
		t.Fatalf("err = %v, want merchant validation error", err)
	}
}

func assertGrowthTask(t *testing.T, tasks []GrowthTaskItem, code string, group string, status string, current int64, target int64, actionText string) {
	t.Helper()
	for _, task := range tasks {
		if task.TaskCode != code {
			continue
		}
		if task.Group != group || task.Status != status || task.ProgressCurrent != current || task.ProgressTarget != target || task.ActionText != actionText {
			t.Fatalf("task %s = %#v, want group=%s status=%s progress=%d/%d action=%s", code, task, group, status, current, target, actionText)
		}
		return
	}
	t.Fatalf("task %s not found in %#v", code, tasks)
}

type fakeGrowthTaskStore struct {
	campaigns    []model.PublicGrowthCampaign
	entitlements []model.MerchantEntitlement
	progress     model.GrowthTaskProgress
	grants       []model.MerchantGrowthRewardGrant
}

func (s *fakeGrowthTaskStore) ListActiveGrowthCampaigns(ctx context.Context) ([]model.PublicGrowthCampaign, error) {
	return s.campaigns, nil
}

func (s *fakeGrowthTaskStore) ListMerchantEntitlements(ctx context.Context, merchantID string) ([]model.MerchantEntitlement, error) {
	return s.entitlements, nil
}

func (s *fakeGrowthTaskStore) GetGrowthTaskProgress(ctx context.Context, merchantID string) (model.GrowthTaskProgress, error) {
	return s.progress, nil
}

func (s *fakeGrowthTaskStore) ListMerchantGrowthRewardGrants(ctx context.Context, merchantID string) ([]model.MerchantGrowthRewardGrant, error) {
	return s.grants, nil
}
