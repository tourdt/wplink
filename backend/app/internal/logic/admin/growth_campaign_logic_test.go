package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestSaveGrowthCampaignRequiresDisableReason(t *testing.T) {
	logic := NewGrowthCampaignAdminLogic(&fakeGrowthCampaignAdminStore{})

	_, err := logic.SaveGrowthCampaign(context.Background(), "", SaveGrowthCampaignReq{
		Code:   "starter_growth_2026_q3",
		Name:   "新手成长权益活动",
		Status: "disabled",
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveGrowthCampaign() error = %v, want validation error", err)
	}
}

func TestSaveGrowthCampaignPassesNormalizedInputToStore(t *testing.T) {
	store := &fakeGrowthCampaignAdminStore{saveResult: model.AdminGrowthConfigSaveResult{Code: "starter_growth_2026_q3", UpdatedAt: "2026-07-10T10:00:00Z"}}
	logic := NewGrowthCampaignAdminLogic(store)

	resp, err := logic.SaveGrowthCampaign(context.Background(), " starter_growth_2026_q3 ", SaveGrowthCampaignReq{
		Name:   " 新手成长权益活动 ",
		Status: "",
	}, " admin-1 ")
	if err != nil {
		t.Fatalf("SaveGrowthCampaign() error = %v", err)
	}

	if resp.Code != "starter_growth_2026_q3" || store.campaignInput.Code != "starter_growth_2026_q3" || store.campaignInput.Name != "新手成长权益活动" {
		t.Fatalf("resp = %#v input = %#v, want normalized campaign", resp, store.campaignInput)
	}
	if store.campaignInput.Status != "active" || store.campaignInput.OperatorID != "admin-1" {
		t.Fatalf("input = %#v, want active status and trimmed operator", store.campaignInput)
	}
}

func TestSaveGrowthRuleRejectsShareRewardWithoutLimit(t *testing.T) {
	logic := NewGrowthCampaignAdminLogic(&fakeGrowthCampaignAdminStore{})

	_, err := logic.SaveGrowthRule(context.Background(), "starter_growth_2026_q3", "", SaveGrowthRuleReq{
		RuleCode:     "share_contact_refresh_quota",
		TriggerEvent: model.GrowthEventResourceShareEffectiveContact,
		Status:       "active",
		RewardType:   model.EntitlementTypeRefreshQuota,
		RewardAmount: 1,
		ValidDays:    15,
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveGrowthRule() error = %v, want validation error", err)
	}
}

func TestSaveGrowthRuleRejectsTopVoucherBeforeTopParametersExist(t *testing.T) {
	logic := NewGrowthCampaignAdminLogic(&fakeGrowthCampaignAdminStore{})

	_, err := logic.SaveGrowthRule(context.Background(), "starter_growth_2026_q3", "", SaveGrowthRuleReq{
		RuleCode:     "first_login_top_voucher",
		TriggerEvent: model.GrowthEventUserFirstLogin,
		Status:       "active",
		RewardType:   model.EntitlementTypeTopVoucher,
		RewardAmount: 1,
		ValidDays:    30,
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveGrowthRule() error = %v, want validation error", err)
	}
}

func TestSaveGrowthRulePassesNormalizedInputToStore(t *testing.T) {
	store := &fakeGrowthCampaignAdminStore{saveResult: model.AdminGrowthConfigSaveResult{Code: "first_login_publish_quota", UpdatedAt: "2026-07-10T10:00:00Z"}}
	logic := NewGrowthCampaignAdminLogic(store)

	resp, err := logic.SaveGrowthRule(context.Background(), " starter_growth_2026_q3 ", " first_login_publish_quota ", SaveGrowthRuleReq{
		TriggerEvent: model.GrowthEventUserFirstLogin,
		RewardType:   model.EntitlementTypePublishQuota,
		RewardAmount: 3,
		ValidDays:    30,
	}, " admin-1 ")
	if err != nil {
		t.Fatalf("SaveGrowthRule() error = %v", err)
	}

	if resp.Code != "first_login_publish_quota" || store.ruleInput.CampaignCode != "starter_growth_2026_q3" || store.ruleInput.RuleCode != "first_login_publish_quota" {
		t.Fatalf("resp = %#v input = %#v, want normalized rule", resp, store.ruleInput)
	}
	if store.ruleInput.Status != "active" || store.ruleInput.OperatorID != "admin-1" {
		t.Fatalf("input = %#v, want active status and trimmed operator", store.ruleInput)
	}
}

type fakeGrowthCampaignAdminStore struct {
	campaignInput model.SaveAdminGrowthCampaignInput
	ruleInput     model.SaveAdminGrowthRuleInput
	saveResult    model.AdminGrowthConfigSaveResult
}

func (s *fakeGrowthCampaignAdminStore) ListAdminGrowthCampaigns(ctx context.Context, status string) ([]model.AdminGrowthCampaignConfig, error) {
	return []model.AdminGrowthCampaignConfig{{Code: "starter_growth_2026_q3", Name: "新手成长权益活动", Status: "active"}}, nil
}

func (s *fakeGrowthCampaignAdminStore) SaveAdminGrowthCampaign(ctx context.Context, input model.SaveAdminGrowthCampaignInput) (model.AdminGrowthConfigSaveResult, error) {
	s.campaignInput = input
	return s.saveResult, nil
}

func (s *fakeGrowthCampaignAdminStore) ListAdminGrowthRules(ctx context.Context, campaignCode string) ([]model.AdminGrowthRuleConfig, error) {
	return []model.AdminGrowthRuleConfig{{CampaignCode: campaignCode, RuleCode: "first_login_publish_quota", Status: "active"}}, nil
}

func (s *fakeGrowthCampaignAdminStore) SaveAdminGrowthRule(ctx context.Context, input model.SaveAdminGrowthRuleInput) (model.AdminGrowthConfigSaveResult, error) {
	s.ruleInput = input
	return s.saveResult, nil
}

func (s *fakeGrowthCampaignAdminStore) ListAdminGrowthRewardGrants(ctx context.Context, filter model.AdminGrowthRewardGrantFilter) ([]model.AdminGrowthRewardGrant, error) {
	return []model.AdminGrowthRewardGrant{{ID: "grant-1", CampaignCode: filter.CampaignCode, RuleCode: "first_login_publish_quota", Status: "granted"}}, nil
}
