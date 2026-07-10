package growth

import (
	"context"
	"fmt"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

const (
	growthTaskGroupStarter = "starter"
	growthTaskGroupDaily   = "daily"

	growthTaskStatusTodo          = "todo"
	growthTaskStatusInProgress    = "in_progress"
	growthTaskStatusPendingReview = "pending_review"
	growthTaskStatusCompleted     = "completed"
	growthTaskStatusGranted       = "granted"

	growthTaskActionPublish        = "publish"
	growthTaskActionShare          = "share"
	growthTaskActionManageResource = "manage_resources"
	growthTaskActionUseEntitlement = "use_entitlement"
	growthTaskActionNone           = "none"
)

type GrowthTaskStore interface {
	ListActiveGrowthCampaigns(ctx context.Context) ([]model.PublicGrowthCampaign, error)
	ListMerchantEntitlements(ctx context.Context, merchantID string) ([]model.MerchantEntitlement, error)
	GetGrowthTaskProgress(ctx context.Context, merchantID string) (model.GrowthTaskProgress, error)
	ListMerchantGrowthRewardGrants(ctx context.Context, merchantID string) ([]model.MerchantGrowthRewardGrant, error)
}

type GrowthTaskCampaignInfo struct {
	Code  string `json:"code,omitempty"`
	Title string `json:"title,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

type GrowthTaskSummary struct {
	PublishQuotaRemaining int64 `json:"publishQuotaRemaining"`
	RefreshQuotaRemaining int64 `json:"refreshQuotaRemaining"`
	StarterCompletedCount int64 `json:"starterCompletedCount"`
	StarterTotalCount     int64 `json:"starterTotalCount"`
}

type GrowthTaskItem struct {
	TaskCode        string `json:"taskCode"`
	Group           string `json:"group"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	ProgressCurrent int64  `json:"progressCurrent"`
	ProgressTarget  int64  `json:"progressTarget"`
	Status          string `json:"status"`
	RewardType      string `json:"rewardType"`
	RewardAmount    int64  `json:"rewardAmount"`
	RewardText      string `json:"rewardText"`
	ValidDays       int64  `json:"validDays"`
	ActionType      string `json:"actionType"`
	ActionText      string `json:"actionText"`
	Hint            string `json:"hint,omitempty"`
}

type GetGrowthTasksResp struct {
	Campaign GrowthTaskCampaignInfo `json:"campaign"`
	Summary  GrowthTaskSummary      `json:"summary"`
	Tasks    []GrowthTaskItem       `json:"tasks"`
}

type GrowthTaskLogic struct {
	store GrowthTaskStore
}

func NewGrowthTaskLogic(store GrowthTaskStore) *GrowthTaskLogic {
	return &GrowthTaskLogic{store: store}
}

func (l *GrowthTaskLogic) GetGrowthTasks(ctx context.Context, merchantID string) (GetGrowthTasksResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return GetGrowthTasksResp{}, errx.New(errx.CodeValidationFailed, "商家不存在")
	}

	entitlements, err := l.store.ListMerchantEntitlements(ctx, merchantID)
	if err != nil {
		return GetGrowthTasksResp{}, err
	}
	resp := GetGrowthTasksResp{Summary: growthTaskSummaryFromEntitlements(entitlements), Tasks: []GrowthTaskItem{}}

	campaigns, err := l.store.ListActiveGrowthCampaigns(ctx)
	if err != nil {
		return GetGrowthTasksResp{}, err
	}
	if len(campaigns) == 0 {
		return resp, nil
	}

	progress, err := l.store.GetGrowthTaskProgress(ctx, merchantID)
	if err != nil {
		return GetGrowthTasksResp{}, err
	}
	grants, err := l.store.ListMerchantGrowthRewardGrants(ctx, merchantID)
	if err != nil {
		return GetGrowthTasksResp{}, err
	}

	campaign := campaigns[0]
	resp.Campaign = GrowthTaskCampaignInfo{Code: campaign.Code, Title: campaign.Title, Hint: campaign.Hint}
	grantedRules := grantedRuleSet(grants)
	for _, rule := range campaign.Rules {
		task, ok := growthTaskFromRule(rule, progress, grantedRules)
		if !ok {
			continue
		}
		resp.Tasks = append(resp.Tasks, task)
		if task.Group == growthTaskGroupStarter {
			resp.Summary.StarterTotalCount++
			if task.Status == growthTaskStatusGranted || task.Status == growthTaskStatusCompleted {
				resp.Summary.StarterCompletedCount++
			}
		}
	}
	return resp, nil
}

func growthTaskSummaryFromEntitlements(entitlements []model.MerchantEntitlement) GrowthTaskSummary {
	var summary GrowthTaskSummary
	for _, item := range entitlements {
		switch item.Type {
		case model.EntitlementTypePublishQuota:
			summary.PublishQuotaRemaining += item.RemainingAmount
		case model.EntitlementTypeRefreshQuota:
			summary.RefreshQuotaRemaining += item.RemainingAmount
		}
	}
	return summary
}

func grantedRuleSet(grants []model.MerchantGrowthRewardGrant) map[string]bool {
	result := make(map[string]bool, len(grants))
	for _, grant := range grants {
		if grant.Status == "granted" {
			result[grant.RuleCode] = true
		}
	}
	return result
}

func growthTaskFromRule(rule model.PublicGrowthRule, progress model.GrowthTaskProgress, grantedRules map[string]bool) (GrowthTaskItem, bool) {
	group, ok := growthTaskGroup(rule)
	if !ok {
		return GrowthTaskItem{}, false
	}
	task := GrowthTaskItem{
		TaskCode:     rule.RuleCode,
		Group:        group,
		Title:        growthTaskTitle(rule),
		Description:  growthTaskDescription(rule),
		Status:       growthTaskStatusTodo,
		RewardType:   rule.RewardType,
		RewardAmount: rule.RewardAmount,
		RewardText:   growthRewardText(rule.RewardType, rule.RewardAmount),
		ValidDays:    rule.ValidDays,
		ActionType:   growthTaskActionNone,
		ActionText:   "",
		Hint:         growthTaskHint(rule),
	}

	switch rule.TriggerEvent {
	case model.GrowthEventUserFirstLogin:
		task.ProgressCurrent, task.ProgressTarget = boolProgress(grantedRules[rule.RuleCode])
		task.ActionType, task.ActionText = growthTaskActionPublish, "去发布"
	case model.GrowthEventResourceFirstApproved:
		task.ProgressCurrent = minInt64(progress.PublishedResourceCount, 1)
		task.ProgressTarget = 1
		task.ActionType, task.ActionText = growthTaskActionManageResource, "查看发布"
	case model.GrowthEventResourceApprovedCountReached:
		task.ProgressTarget = growthTaskConditionInt(rule.Conditions, "approvedCount", 3)
		task.ProgressCurrent = minInt64(progress.ApprovedWithinWindowCount, task.ProgressTarget)
		task.ActionType, task.ActionText = growthTaskActionPublish, "去发布"
	case model.GrowthEventResourceShareEffectiveContact:
		task.ProgressTarget = firstPositiveInt64(rule.PerUserDailyLimit, rule.PerResourceDailyLimit, 1)
		task.ProgressCurrent = minInt64(progress.TodayEffectiveContactCount, task.ProgressTarget)
		task.ActionType, task.ActionText = growthTaskActionShare, "去分享"
	case model.GrowthEventResourceShareEffectiveView:
		task.ProgressTarget = growthTaskConditionInt(rule.Conditions, "viewThreshold", 5)
		task.ProgressCurrent = minInt64(progress.ShareViewWindowCount, task.ProgressTarget)
		task.ActionType, task.ActionText = growthTaskActionShare, "去分享"
	default:
		return GrowthTaskItem{}, false
	}
	task.Status = growthTaskStatus(rule, task, progress, grantedRules)
	normalizeGrowthTaskAction(&task)
	return task, true
}

func growthTaskGroup(rule model.PublicGrowthRule) (string, bool) {
	switch rule.TriggerEvent {
	case model.GrowthEventUserFirstLogin, model.GrowthEventResourceFirstApproved, model.GrowthEventResourceApprovedCountReached:
		// 老库里的活动规则可能是在 per_user_limit 回填前插入的；主线任务按事件语义归类，避免前台空白。
		return growthTaskGroupStarter, true
	case model.GrowthEventResourceShareEffectiveContact, model.GrowthEventResourceShareEffectiveView:
		return growthTaskGroupDaily, true
	}
	return "", false
}

func growthTaskStatus(rule model.PublicGrowthRule, task GrowthTaskItem, progress model.GrowthTaskProgress, grantedRules map[string]bool) string {
	if task.Group == growthTaskGroupStarter && grantedRules[rule.RuleCode] {
		return growthTaskStatusGranted
	}
	if task.ProgressTarget > 0 && task.ProgressCurrent >= task.ProgressTarget {
		return growthTaskStatusCompleted
	}
	if rule.TriggerEvent == model.GrowthEventResourceFirstApproved && progress.PendingResourceCount > 0 {
		return growthTaskStatusPendingReview
	}
	if task.ProgressCurrent > 0 {
		return growthTaskStatusInProgress
	}
	return growthTaskStatusTodo
}

func normalizeGrowthTaskAction(task *GrowthTaskItem) {
	if task.Status != growthTaskStatusCompleted {
		return
	}
	// 已达标但还没有发放记录时，前台只提示等待到账，避免用户误点到不可用的权益。
	task.ActionType = growthTaskActionNone
	task.ActionText = ""
	task.Hint = "权益正在到账中"
}

func growthTaskTitle(rule model.PublicGrowthRule) string {
	if strings.TrimSpace(rule.RuleName) != "" {
		return strings.TrimSpace(rule.RuleName)
	}
	return strings.TrimSpace(rule.Description)
}

func growthTaskDescription(rule model.PublicGrowthRule) string {
	switch rule.TriggerEvent {
	case model.GrowthEventUserFirstLogin:
		return "登录得发布次数"
	case model.GrowthEventResourceFirstApproved:
		return "发资源并过审"
	case model.GrowthEventResourceApprovedCountReached:
		return "发布 3 条优质资源"
	case model.GrowthEventResourceShareEffectiveContact:
		return "分享带来联系"
	case model.GrowthEventResourceShareEffectiveView:
		return "分享带来浏览"
	default:
		return "完成任务得权益"
	}
}

func growthTaskHint(rule model.PublicGrowthRule) string {
	switch rule.TriggerEvent {
	case model.GrowthEventResourceFirstApproved:
		return "审核通过后自动到账"
	case model.GrowthEventResourceShareEffectiveContact, model.GrowthEventResourceShareEffectiveView:
		return "今日达标后自动到账"
	default:
		if rule.ValidDays > 0 {
			return fmt.Sprintf("权益有效期 %d 天", rule.ValidDays)
		}
		return ""
	}
}

func boolProgress(done bool) (int64, int64) {
	if done {
		return 1, 1
	}
	return 0, 1
}

func growthTaskConditionInt(conditions model.JSONMap, key string, fallback int64) int64 {
	value, ok := conditions[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case jsonNumber:
		parsed, err := typed.Int64()
		if err == nil {
			return parsed
		}
	}
	return fallback
}

type jsonNumber interface {
	Int64() (int64, error)
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 1
}

func minInt64(left int64, right int64) int64 {
	if left < right {
		return left
	}
	return right
}
