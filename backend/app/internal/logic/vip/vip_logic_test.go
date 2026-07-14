package vip

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListVIPPlansReturnsPromotionalBenefits(t *testing.T) {
	store := &fakeVIPStore{
		plans: []model.VIPPlan{
			{
				Code:              "yearly",
				Name:              "VIP 年卡",
				DurationMonths:    12,
				StandardPriceCent: 49900,
				SalePriceCent:     29900,
				SaleLabel:         "限时优惠",
				Benefits: model.VIPBenefitSnapshot{
					PublishPolicy:    model.VIPPublishPolicyQuota,
					PublishQuota:     80,
					RefreshQuota:     30,
					TopVoucherCount:  3,
					TopDurationHours: 24,
				},
			},
		},
	}

	resp, err := NewListVIPPlansLogic(store).ListVIPPlans(context.Background())
	if err != nil {
		t.Fatalf("ListVIPPlans() error = %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Code != "yearly" || resp.Items[0].SalePriceCent != 29900 {
		t.Fatalf("resp = %#v, want yearly sale plan", resp)
	}
	if resp.Items[0].Benefits.PublishQuota != 80 || resp.Items[0].Benefits.TopVoucherCount != 3 {
		t.Fatalf("benefits = %#v, want quota benefits", resp.Items[0].Benefits)
	}
}

func TestListQuotaPacksReturnsOnlinePurchaseProducts(t *testing.T) {
	store := &fakeVIPStore{
		quotaPacks: []model.QuotaPack{
			{
				Code:              "publish_5",
				Name:              "发布次数包",
				Description:       "临时多发供需",
				StandardPriceCent: 2500,
				SalePriceCent:     2500,
				SaleLabel:         "限时特价",
				Benefits:          model.VIPBenefitSnapshot{PublishQuota: 5},
			},
			{
				Code:              "top_3",
				Name:              "置顶券包",
				Description:       "单张可置顶 24 小时",
				StandardPriceCent: 2900,
				Benefits:          model.VIPBenefitSnapshot{TopVoucherCount: 3, TopDurationHours: 24},
			},
		},
	}

	resp, err := NewListQuotaPacksLogic(store).ListQuotaPacks(context.Background())
	if err != nil {
		t.Fatalf("ListQuotaPacks() error = %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].Code != "publish_5" || resp.Items[0].Benefits.PublishQuota != 5 {
		t.Fatalf("resp = %#v, want publish quota pack", resp)
	}
	if resp.Items[1].Benefits.TopVoucherCount != 3 || resp.Items[1].Benefits.TopDurationHours != 24 {
		t.Fatalf("top pack benefits = %#v, want top voucher benefits", resp.Items[1].Benefits)
	}
}

func TestCreateVIPOrderValidatesPlanAndReturnsOrderSnapshot(t *testing.T) {
	store := &fakeVIPStore{
		order: model.VIPOrder{
			ID:                "order-1",
			MerchantID:        "merchant-1",
			UserID:            "user-1",
			PlanCode:          "monthly",
			PlanName:          "VIP 月卡",
			Status:            model.PaymentOrderStatusPending,
			StandardPriceCent: 4900,
			ActualPriceCent:   1990,
			PromotionCode:     "launch_monthly_first",
			Benefits: model.VIPBenefitSnapshot{
				PublishPolicy:    model.VIPPublishPolicyQuota,
				PublishQuota:     80,
				RefreshQuota:     30,
				TopVoucherCount:  3,
				TopDurationHours: 24,
			},
		},
	}
	resp, err := NewCreateVIPOrderLogic(store).CreateVIPOrder(context.Background(), CreateVIPOrderReq{
		MerchantID: " merchant-1 ",
		UserID:     " user-1 ",
		PlanCode:   " monthly ",
	})
	if err != nil {
		t.Fatalf("CreateVIPOrder() error = %v", err)
	}
	if store.createInput.MerchantID != "merchant-1" || store.createInput.UserID != "user-1" || store.createInput.PlanCode != "monthly" {
		t.Fatalf("createInput = %#v, want trimmed input", store.createInput)
	}
	if resp.OrderID != "order-1" || resp.ActualPriceCent != 1990 || resp.PromotionCode != "launch_monthly_first" {
		t.Fatalf("resp = %#v, want discounted order snapshot", resp)
	}

	_, err = NewCreateVIPOrderLogic(store).CreateVIPOrder(context.Background(), CreateVIPOrderReq{MerchantID: "merchant-1", UserID: "user-1"})
	if err == nil || errx.PublicMessage(err) != "请选择有效的 VIP 套餐" {
		t.Fatalf("error = %v, want friendly invalid plan message", err)
	}
}

func TestCreateVIPOrderCreatesQuotaPackOrderSnapshot(t *testing.T) {
	store := &fakeVIPStore{
		quotaOrder: model.VIPOrder{
			ID:                "order-2",
			MerchantID:        "merchant-1",
			UserID:            "user-1",
			ProductType:       model.VIPProductTypeQuotaPack,
			ProductCode:       "publish_5",
			ProductName:       "发布次数包",
			Status:            model.PaymentOrderStatusPending,
			StandardPriceCent: 2500,
			ActualPriceCent:   2500,
			Benefits:          model.VIPBenefitSnapshot{PublishQuota: 5},
		},
	}
	resp, err := NewCreateVIPOrderLogic(store).CreateVIPOrder(context.Background(), CreateVIPOrderReq{
		MerchantID:  " merchant-1 ",
		UserID:      " user-1 ",
		ProductType: " quota_pack ",
		ProductCode: " publish_5 ",
		ResourceID:  " resource-1 ",
	})
	if err != nil {
		t.Fatalf("CreateVIPOrder() error = %v", err)
	}
	if store.createQuotaInput.MerchantID != "merchant-1" || store.createQuotaInput.UserID != "user-1" || store.createQuotaInput.PackCode != "publish_5" || store.createQuotaInput.ResourceID != "resource-1" {
		t.Fatalf("createQuotaInput = %#v, want trimmed quota pack input", store.createQuotaInput)
	}
	if resp.ProductType != model.VIPProductTypeQuotaPack || resp.ProductCode != "publish_5" || resp.Benefits.PublishQuota != 5 {
		t.Fatalf("resp = %#v, want quota pack order snapshot", resp)
	}

	_, err = NewCreateVIPOrderLogic(store).CreateVIPOrder(context.Background(), CreateVIPOrderReq{
		MerchantID: "merchant-1", UserID: "user-1", ProductType: model.VIPProductTypeQuotaPack,
	})
	if err == nil || errx.PublicMessage(err) != "请选择有效的次数包" {
		t.Fatalf("error = %v, want friendly invalid quota pack message", err)
	}
}

func TestCreateVIPOrderRejectsTopServiceWithoutResource(t *testing.T) {
	store := &fakeVIPStore{quotaErr: model.ErrTopServiceResourceRequired}
	_, err := NewCreateVIPOrderLogic(store).CreateVIPOrder(context.Background(), CreateVIPOrderReq{
		MerchantID:  "merchant-1",
		UserID:      "user-1",
		ProductType: model.VIPProductTypeQuotaPack,
		ProductCode: "top_1d",
	})
	if err == nil || errx.PublicMessage(err) != "请选择要置顶的供需信息" {
		t.Fatalf("error = %v, want missing top service resource message", err)
	}
}

func TestCreateVIPOrderRejectsUnknownProductType(t *testing.T) {
	_, err := NewCreateVIPOrderLogic(&fakeVIPStore{}).CreateVIPOrder(context.Background(), CreateVIPOrderReq{
		MerchantID:  "merchant-1",
		UserID:      "user-1",
		ProductType: "unknown",
		PlanCode:    "monthly",
	})
	if err == nil || errx.PublicMessage(err) != "请选择有效的权益商品" {
		t.Fatalf("error = %v, want friendly invalid product message", err)
	}
}

func TestGetMerchantVIPReturnsActiveQuota(t *testing.T) {
	store := &fakeVIPStore{
		summary: model.MerchantVIPSummary{
			MerchantID:            "merchant-1",
			Status:                model.VIPStatusActive,
			PlanCode:              "yearly",
			PlanName:              "VIP 年卡",
			ExpiresAt:             "2027-07-09T00:00:00+08:00",
			PublishQuotaRemaining: 80,
			RefreshQuotaRemaining: 30,
			TopVoucherCount:       3,
		},
	}
	resp, err := NewGetMerchantVIPLogic(store).GetMerchantVIP(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatalf("GetMerchantVIP() error = %v", err)
	}
	if resp.Status != model.VIPStatusActive || resp.PublishQuotaRemaining != 80 || resp.TopVoucherCount != 3 {
		t.Fatalf("resp = %#v, want active vip quota summary", resp)
	}
}

type fakeVIPStore struct {
	plans            []model.VIPPlan
	quotaPacks       []model.QuotaPack
	order            model.VIPOrder
	quotaOrder       model.VIPOrder
	quotaErr         error
	summary          model.MerchantVIPSummary
	createInput      model.CreateVIPOrderInput
	createQuotaInput model.CreateQuotaPackOrderInput
}

func (s *fakeVIPStore) ListVIPPlans(ctx context.Context) ([]model.VIPPlan, error) {
	return s.plans, nil
}

func (s *fakeVIPStore) ListQuotaPacks(ctx context.Context) ([]model.QuotaPack, error) {
	return s.quotaPacks, nil
}

func (s *fakeVIPStore) CreateVIPOrder(ctx context.Context, input model.CreateVIPOrderInput) (model.VIPOrder, error) {
	s.createInput = input
	return s.order, nil
}

func (s *fakeVIPStore) CreateQuotaPackOrder(ctx context.Context, input model.CreateQuotaPackOrderInput) (model.VIPOrder, error) {
	s.createQuotaInput = input
	if s.quotaErr != nil {
		return model.VIPOrder{}, s.quotaErr
	}
	return s.quotaOrder, nil
}

func (s *fakeVIPStore) GetMerchantVIPSummary(ctx context.Context, merchantID string) (model.MerchantVIPSummary, error) {
	return s.summary, nil
}
