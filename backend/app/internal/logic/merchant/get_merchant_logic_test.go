package merchant

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestGetMerchantReturnsProfileTrustAndSummary(t *testing.T) {
	store := &fakeMerchantDetailStore{
		detail: model.MerchantDetail{
			ID:                 "merchant-1",
			Name:               "织里样板童装厂",
			MerchantType:       "factory",
			CityCode:           "zhili",
			MainCategories:     []string{"童装"},
			VerificationStatus: "verified",
			CreditTags:         []model.CreditTag{{Code: "verified_factory", Label: "已认证工厂"}},
			ContactName:        "李厂长",
			PhoneMasked:        "138****0000",
			WechatMasked:       "zhili_****",
			PublishedCount:     12,
			DealtCount:         3,
		},
	}
	logic := NewGetMerchantLogic(store)

	resp, err := logic.GetMerchant(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatalf("GetMerchant() error = %v", err)
	}

	if store.merchantID != "merchant-1" {
		t.Fatalf("merchantID = %q, want trimmed merchant-1", store.merchantID)
	}
	if resp.CreditTags[0].Label != "已认证工厂" {
		t.Fatalf("credit tag = %#v, want verified factory", resp.CreditTags)
	}
	if resp.ResourcesSummary.PublishedCount != 12 || resp.ResourcesSummary.DealtCount != 3 {
		t.Fatalf("summary = %#v, want published/dealt count", resp.ResourcesSummary)
	}
}

type fakeMerchantDetailStore struct {
	merchantID string
	detail     model.MerchantDetail
}

func (s *fakeMerchantDetailStore) GetMerchantDetail(ctx context.Context, merchantID string) (model.MerchantDetail, error) {
	s.merchantID = merchantID
	return s.detail, nil
}
