package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestSaveVIPPlanRejectsInvalidTopVoucherDuration(t *testing.T) {
	logic := NewVIPConfigAdminLogic(&fakeVIPConfigAdminStore{})

	_, err := logic.SaveVIPPlan(context.Background(), "", SaveVIPPlanConfigReq{
		Code:              "monthly",
		Name:              "VIP 月卡",
		DurationMonths:    1,
		StandardPriceCent: 4900,
		Status:            "active",
		Benefits: VIPBenefitConfig{
			PublishQuota:    80,
			RefreshQuota:    30,
			TopVoucherCount: 3,
		},
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveVIPPlan() error = %v, want validation error", err)
	}
}

func TestSaveVIPPlanPassesNormalizedInputToStore(t *testing.T) {
	store := &fakeVIPConfigAdminStore{saved: model.AdminVIPConfigSaveResult{Code: "monthly", UpdatedAt: "2026-07-10T10:00:00Z"}}
	logic := NewVIPConfigAdminLogic(store)

	resp, err := logic.SaveVIPPlan(context.Background(), " monthly ", SaveVIPPlanConfigReq{
		Name:              " VIP 月卡 ",
		DurationMonths:    1,
		StandardPriceCent: 4900,
		Status:            "",
		DisplayOrder:      10,
		Benefits: VIPBenefitConfig{
			PublishQuota:       80,
			RefreshQuota:       30,
			TopVoucherCount:    3,
			TopDurationHours:   24,
			HomepageImageLimit: 18,
		},
	}, " admin-1 ")
	if err != nil {
		t.Fatalf("SaveVIPPlan() error = %v", err)
	}
	if resp.Code != "monthly" || store.planInput.Code != "monthly" || store.planInput.Name != "VIP 月卡" || store.planInput.OperatorID != "admin-1" {
		t.Fatalf("resp = %#v input = %#v, want normalized values", resp, store.planInput)
	}
	if store.planInput.Status != "active" || store.planInput.Benefits.PublishPolicy != model.VIPPublishPolicyQuota {
		t.Fatalf("input = %#v, want default active quota policy", store.planInput)
	}
}

func TestSaveQuotaPackRejectsSalePriceAboveStandardPrice(t *testing.T) {
	logic := NewVIPConfigAdminLogic(&fakeVIPConfigAdminStore{})

	_, err := logic.SaveQuotaPack(context.Background(), "", SaveQuotaPackConfigReq{
		Code:              "publish_5",
		Name:              "发布次数包",
		StandardPriceCent: 2500,
		SalePriceCent:     3000,
		Status:            "active",
		Benefits:          VIPBenefitConfig{PublishQuota: 5},
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveQuotaPack() error = %v, want validation error", err)
	}
}

func TestSaveVIPPromotionRejectsInvalidPeriod(t *testing.T) {
	logic := NewVIPConfigAdminLogic(&fakeVIPConfigAdminStore{})

	_, err := logic.SaveVIPPromotion(context.Background(), "", SaveVIPPromotionConfigReq{
		Code:          "launch_monthly",
		PlanCode:      "monthly",
		PromotionType: "launch",
		SalePriceCent: 1990,
		StartsAt:      "2026-07-11T00:00:00Z",
		EndsAt:        "2026-07-10T00:00:00Z",
		Status:        "active",
	}, "admin-1")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveVIPPromotion() error = %v, want validation error", err)
	}
}

type fakeVIPConfigAdminStore struct {
	planInput      model.SaveAdminVIPPlanInput
	quotaPackInput model.SaveAdminQuotaPackInput
	promotionInput model.SaveAdminVIPPromotionInput
	saved          model.AdminVIPConfigSaveResult
}

func (s *fakeVIPConfigAdminStore) ListAdminVIPPlans(ctx context.Context) ([]model.AdminVIPPlanConfig, error) {
	return []model.AdminVIPPlanConfig{{Code: "monthly", Name: "VIP 月卡", DurationMonths: 1, Status: "active"}}, nil
}

func (s *fakeVIPConfigAdminStore) SaveAdminVIPPlan(ctx context.Context, input model.SaveAdminVIPPlanInput) (model.AdminVIPConfigSaveResult, error) {
	s.planInput = input
	return s.saved, nil
}

func (s *fakeVIPConfigAdminStore) ListAdminQuotaPacks(ctx context.Context) ([]model.AdminQuotaPackConfig, error) {
	return []model.AdminQuotaPackConfig{{Code: "publish_5", Name: "发布次数包", Status: "active"}}, nil
}

func (s *fakeVIPConfigAdminStore) SaveAdminQuotaPack(ctx context.Context, input model.SaveAdminQuotaPackInput) (model.AdminVIPConfigSaveResult, error) {
	s.quotaPackInput = input
	return s.saved, nil
}

func (s *fakeVIPConfigAdminStore) ListAdminVIPPromotions(ctx context.Context) ([]model.AdminVIPPromotionConfig, error) {
	return []model.AdminVIPPromotionConfig{{Code: "launch_monthly", PlanCode: "monthly", PromotionType: "launch", Status: "active"}}, nil
}

func (s *fakeVIPConfigAdminStore) SaveAdminVIPPromotion(ctx context.Context, input model.SaveAdminVIPPromotionInput) (model.AdminVIPConfigSaveResult, error) {
	s.promotionInput = input
	return s.saved, nil
}
