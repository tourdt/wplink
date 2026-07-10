package admin

import (
	"context"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GrowthCampaignAdminStore interface {
	ListAdminGrowthCampaigns(ctx context.Context, status string) ([]model.AdminGrowthCampaignConfig, error)
	SaveAdminGrowthCampaign(ctx context.Context, input model.SaveAdminGrowthCampaignInput) (model.AdminGrowthConfigSaveResult, error)
	ListAdminGrowthRules(ctx context.Context, campaignCode string) ([]model.AdminGrowthRuleConfig, error)
	SaveAdminGrowthRule(ctx context.Context, input model.SaveAdminGrowthRuleInput) (model.AdminGrowthConfigSaveResult, error)
	ListAdminGrowthRewardGrants(ctx context.Context, filter model.AdminGrowthRewardGrantFilter) ([]model.AdminGrowthRewardGrant, error)
}

type GrowthCampaignAdminLogic struct {
	store GrowthCampaignAdminStore
}

func NewGrowthCampaignAdminLogic(store GrowthCampaignAdminStore) *GrowthCampaignAdminLogic {
	return &GrowthCampaignAdminLogic{store: store}
}

type GrowthCampaignItem struct {
	Code      string        `json:"code"`
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	StartsAt  string        `json:"startsAt,omitempty"`
	EndsAt    string        `json:"endsAt,omitempty"`
	Config    model.JSONMap `json:"config,omitempty"`
	UpdatedAt string        `json:"updatedAt,omitempty"`
}

type ListGrowthCampaignsResp struct {
	Items []GrowthCampaignItem `json:"items"`
}

type SaveGrowthCampaignReq struct {
	Code           string        `json:"code,omitempty"`
	Name           string        `json:"name"`
	Status         string        `json:"status,omitempty"`
	StartsAt       string        `json:"startsAt,omitempty"`
	EndsAt         string        `json:"endsAt,omitempty"`
	ConfigSnapshot model.JSONMap `json:"configSnapshot,omitempty"`
	DisableReason  string        `json:"disableReason,omitempty"`
}

type SaveGrowthConfigResp struct {
	Code      string `json:"code"`
	UpdatedAt string `json:"updatedAt"`
}

type GrowthRuleItem struct {
	CampaignCode          string        `json:"campaignCode"`
	RuleCode              string        `json:"ruleCode"`
	RuleName              string        `json:"ruleName"`
	TriggerEvent          string        `json:"triggerEvent"`
	Status                string        `json:"status"`
	Priority              int64         `json:"priority"`
	Conditions            model.JSONMap `json:"conditions,omitempty"`
	RewardType            string        `json:"rewardType"`
	RewardAmount          int64         `json:"rewardAmount"`
	ValidDays             int64         `json:"validDays"`
	PerUserLimit          int64         `json:"perUserLimit,omitempty"`
	PerUserDailyLimit     int64         `json:"perUserDailyLimit,omitempty"`
	PerResourceDailyLimit int64         `json:"perResourceDailyLimit,omitempty"`
	Description           string        `json:"description,omitempty"`
	UpdatedAt             string        `json:"updatedAt,omitempty"`
}

type ListGrowthRulesResp struct {
	Items []GrowthRuleItem `json:"items"`
}

type SaveGrowthRuleReq struct {
	RuleCode              string        `json:"ruleCode,omitempty"`
	RuleName              string        `json:"ruleName"`
	TriggerEvent          string        `json:"triggerEvent"`
	Status                string        `json:"status,omitempty"`
	Priority              int64         `json:"priority,omitempty"`
	Conditions            model.JSONMap `json:"conditions,omitempty"`
	RewardType            string        `json:"rewardType"`
	RewardAmount          int64         `json:"rewardAmount"`
	ValidDays             int64         `json:"validDays"`
	PerUserLimit          int64         `json:"perUserLimit,omitempty"`
	PerUserDailyLimit     int64         `json:"perUserDailyLimit,omitempty"`
	PerResourceDailyLimit int64         `json:"perResourceDailyLimit,omitempty"`
	Description           string        `json:"description,omitempty"`
}

type GrowthGrantItem struct {
	ID           string `json:"id"`
	CampaignCode string `json:"campaignCode"`
	RuleCode     string `json:"ruleCode"`
	RuleName     string `json:"ruleName,omitempty"`
	MerchantID   string `json:"merchantId"`
	ResourceID   string `json:"resourceId,omitempty"`
	RewardType   string `json:"rewardType"`
	RewardAmount int64  `json:"rewardAmount"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type ListGrowthRewardGrantsResp struct {
	Items []GrowthGrantItem `json:"items"`
}

func (l *GrowthCampaignAdminLogic) ListGrowthCampaigns(ctx context.Context, status string) (ListGrowthCampaignsResp, error) {
	status = strings.TrimSpace(status)
	if status != "" && !isGrowthCampaignStatus(status) {
		return ListGrowthCampaignsResp{}, errx.New(errx.CodeValidationFailed, "活动状态不正确")
	}
	campaigns, err := l.store.ListAdminGrowthCampaigns(ctx, status)
	if err != nil {
		logx.Errorf("查询增长活动失败: status=%s err=%+v", status, err)
		return ListGrowthCampaignsResp{}, errx.New(errx.CodeInternalError, "查询失败，请稍后重试")
	}
	items := make([]GrowthCampaignItem, 0, len(campaigns))
	for _, campaign := range campaigns {
		items = append(items, GrowthCampaignItem{
			Code:      campaign.Code,
			Name:      campaign.Name,
			Status:    campaign.Status,
			StartsAt:  campaign.StartsAt,
			EndsAt:    campaign.EndsAt,
			Config:    campaign.ConfigSnapshot,
			UpdatedAt: campaign.UpdatedAt,
		})
	}
	return ListGrowthCampaignsResp{Items: items}, nil
}

func (l *GrowthCampaignAdminLogic) SaveGrowthCampaign(ctx context.Context, pathCode string, req SaveGrowthCampaignReq, operatorID string) (SaveGrowthConfigResp, error) {
	input := model.SaveAdminGrowthCampaignInput{
		Code:           configCode(pathCode, req.Code),
		Name:           strings.TrimSpace(req.Name),
		Status:         normalizeGrowthCampaignStatus(req.Status),
		StartsAt:       strings.TrimSpace(req.StartsAt),
		EndsAt:         strings.TrimSpace(req.EndsAt),
		ConfigSnapshot: req.ConfigSnapshot,
		OperatorID:     strings.TrimSpace(operatorID),
		DisableReason:  strings.TrimSpace(req.DisableReason),
	}
	if err := validateGrowthCampaignInput(input); err != nil {
		return SaveGrowthConfigResp{}, err
	}
	result, err := l.store.SaveAdminGrowthCampaign(ctx, input)
	if err != nil {
		logx.Errorf("保存增长活动失败: operatorId=%s campaignCode=%s status=%s err=%+v", input.OperatorID, input.Code, input.Status, err)
		return SaveGrowthConfigResp{}, errx.New(errx.CodeInternalError, "保存失败，请稍后重试")
	}
	return SaveGrowthConfigResp{Code: result.Code, UpdatedAt: result.UpdatedAt}, nil
}

func (l *GrowthCampaignAdminLogic) ListGrowthRules(ctx context.Context, campaignCode string) (ListGrowthRulesResp, error) {
	campaignCode = strings.TrimSpace(campaignCode)
	if campaignCode == "" {
		return ListGrowthRulesResp{}, errx.New(errx.CodeValidationFailed, "活动不存在")
	}
	rules, err := l.store.ListAdminGrowthRules(ctx, campaignCode)
	if err != nil {
		logx.Errorf("查询增长活动规则失败: campaignCode=%s err=%+v", campaignCode, err)
		return ListGrowthRulesResp{}, errx.New(errx.CodeInternalError, "查询失败，请稍后重试")
	}
	items := make([]GrowthRuleItem, 0, len(rules))
	for _, rule := range rules {
		items = append(items, GrowthRuleItem{
			CampaignCode:          rule.CampaignCode,
			RuleCode:              rule.RuleCode,
			RuleName:              rule.RuleName,
			TriggerEvent:          rule.TriggerEvent,
			Status:                rule.Status,
			Priority:              rule.Priority,
			Conditions:            rule.Conditions,
			RewardType:            rule.RewardType,
			RewardAmount:          rule.RewardAmount,
			ValidDays:             rule.ValidDays,
			PerUserLimit:          rule.PerUserLimit,
			PerUserDailyLimit:     rule.PerUserDailyLimit,
			PerResourceDailyLimit: rule.PerResourceDailyLimit,
			Description:           rule.Description,
			UpdatedAt:             rule.UpdatedAt,
		})
	}
	return ListGrowthRulesResp{Items: items}, nil
}

func (l *GrowthCampaignAdminLogic) SaveGrowthRule(ctx context.Context, campaignCode string, pathRuleCode string, req SaveGrowthRuleReq, operatorID string) (SaveGrowthConfigResp, error) {
	input := model.SaveAdminGrowthRuleInput{
		CampaignCode:          strings.TrimSpace(campaignCode),
		RuleCode:              configCode(pathRuleCode, req.RuleCode),
		RuleName:              strings.TrimSpace(req.RuleName),
		TriggerEvent:          strings.TrimSpace(req.TriggerEvent),
		Status:                normalizeActiveInactive(req.Status),
		Priority:              req.Priority,
		Conditions:            req.Conditions,
		RewardType:            strings.TrimSpace(req.RewardType),
		RewardAmount:          req.RewardAmount,
		ValidDays:             req.ValidDays,
		PerUserLimit:          req.PerUserLimit,
		PerUserDailyLimit:     req.PerUserDailyLimit,
		PerResourceDailyLimit: req.PerResourceDailyLimit,
		Description:           strings.TrimSpace(req.Description),
		OperatorID:            strings.TrimSpace(operatorID),
	}
	if input.Priority == 0 {
		input.Priority = 100
	}
	if err := validateGrowthRuleInput(input); err != nil {
		return SaveGrowthConfigResp{}, err
	}
	result, err := l.store.SaveAdminGrowthRule(ctx, input)
	if err != nil {
		logx.Errorf("保存增长活动规则失败: operatorId=%s campaignCode=%s ruleCode=%s err=%+v", input.OperatorID, input.CampaignCode, input.RuleCode, err)
		return SaveGrowthConfigResp{}, errx.New(errx.CodeInternalError, "保存失败，请稍后重试")
	}
	return SaveGrowthConfigResp{Code: result.Code, UpdatedAt: result.UpdatedAt}, nil
}

func (l *GrowthCampaignAdminLogic) ListGrowthRewardGrants(ctx context.Context, filter model.AdminGrowthRewardGrantFilter) (ListGrowthRewardGrantsResp, error) {
	filter.CampaignCode = strings.TrimSpace(filter.CampaignCode)
	filter.RuleCode = strings.TrimSpace(filter.RuleCode)
	filter.MerchantID = strings.TrimSpace(filter.MerchantID)
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.CampaignCode == "" {
		return ListGrowthRewardGrantsResp{}, errx.New(errx.CodeValidationFailed, "活动不存在")
	}
	grants, err := l.store.ListAdminGrowthRewardGrants(ctx, filter)
	if err != nil {
		logx.Errorf("查询成长权益发放记录失败: campaignCode=%s merchantId=%s status=%s err=%+v", filter.CampaignCode, filter.MerchantID, filter.Status, err)
		return ListGrowthRewardGrantsResp{}, errx.New(errx.CodeInternalError, "查询失败，请稍后重试")
	}
	items := make([]GrowthGrantItem, 0, len(grants))
	for _, grant := range grants {
		items = append(items, GrowthGrantItem{
			ID:           grant.ID,
			CampaignCode: grant.CampaignCode,
			RuleCode:     grant.RuleCode,
			RuleName:     grant.RuleName,
			MerchantID:   grant.MerchantID,
			ResourceID:   grant.ResourceID,
			RewardType:   grant.RewardType,
			RewardAmount: grant.RewardAmount,
			Status:       grant.Status,
			Reason:       grant.Reason,
			CreatedAt:    grant.CreatedAt,
		})
	}
	return ListGrowthRewardGrantsResp{Items: items}, nil
}

func validateGrowthCampaignInput(input model.SaveAdminGrowthCampaignInput) error {
	if input.Code == "" || input.Name == "" {
		return errx.New(errx.CodeValidationFailed, "请填写活动编码和名称")
	}
	if !isGrowthCampaignStatus(input.Status) {
		return errx.New(errx.CodeValidationFailed, "活动状态不正确")
	}
	if input.Status == "disabled" && input.DisableReason == "" {
		return errx.New(errx.CodeValidationFailed, "停用活动必须填写原因")
	}
	if err := validateRFC3339Time(input.StartsAt, "开始时间格式不正确"); err != nil {
		return err
	}
	if err := validateRFC3339Time(input.EndsAt, "结束时间格式不正确"); err != nil {
		return err
	}
	if input.StartsAt != "" && input.EndsAt != "" {
		startsAt, _ := time.Parse(time.RFC3339, input.StartsAt)
		endsAt, _ := time.Parse(time.RFC3339, input.EndsAt)
		if !endsAt.After(startsAt) {
			return errx.New(errx.CodeValidationFailed, "结束时间必须晚于开始时间")
		}
	}
	return nil
}

func validateGrowthRuleInput(input model.SaveAdminGrowthRuleInput) error {
	if input.CampaignCode == "" || input.RuleCode == "" {
		return errx.New(errx.CodeValidationFailed, "规则信息不完整，请刷新后重试")
	}
	if input.RuleName == "" {
		return errx.New(errx.CodeValidationFailed, "请填写规则名称")
	}
	if !isGrowthRuleStatus(input.Status) {
		return errx.New(errx.CodeValidationFailed, "规则状态不正确")
	}
	if !isGrowthTriggerEvent(input.TriggerEvent) {
		return errx.New(errx.CodeValidationFailed, "触发事件不正确")
	}
	if !isGrowthRewardType(input.RewardType) {
		return errx.New(errx.CodeValidationFailed, "奖励类型不正确")
	}
	if input.RewardAmount <= 0 {
		return errx.New(errx.CodeValidationFailed, "奖励数量必须大于 0")
	}
	if input.ValidDays <= 0 {
		return errx.New(errx.CodeValidationFailed, "权益有效期必须大于 0")
	}
	if strings.Contains(input.TriggerEvent, "share") && input.PerUserDailyLimit <= 0 && input.PerResourceDailyLimit <= 0 {
		return errx.New(errx.CodeValidationFailed, "分享类奖励必须设置每日或资源上限")
	}
	return nil
}

func normalizeGrowthCampaignStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "active"
	}
	return status
}

func isGrowthCampaignStatus(status string) bool {
	switch status {
	case "draft", "active", "paused", "ended", "disabled":
		return true
	default:
		return false
	}
}

func isGrowthRuleStatus(status string) bool {
	return status == "active" || status == "inactive"
}

func isGrowthTriggerEvent(event string) bool {
	switch event {
	case model.GrowthEventUserFirstLogin,
		model.GrowthEventResourceFirstApproved,
		model.GrowthEventResourceApprovedCountReached,
		model.GrowthEventResourceShareEffectiveView,
		model.GrowthEventResourceShareEffectiveContact,
		model.GrowthEventInviteeFirstResourceApproved:
		return true
	default:
		return false
	}
}

func isGrowthRewardType(rewardType string) bool {
	switch rewardType {
	case model.EntitlementTypePublishQuota, model.EntitlementTypeRefreshQuota:
		return true
	default:
		return false
	}
}

func validateRFC3339Time(value string, message string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err != nil {
		return errx.New(errx.CodeValidationFailed, message)
	}
	return nil
}
