package metrics

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestGetResourceMetricsReturnsSummaryWithoutVisitorIdentity(t *testing.T) {
	store := &fakeMetricsQueryStore{
		resourceMetrics: model.ResourceMetricsResult{
			ResourceID: "resource-1",
			Summary: model.ResourceMetricsSummary{
				ExposureCount: 10, DetailViewCount: 4, PhoneClickCount: 2, WechatCopyCount: 1,
			},
			Daily: []model.ResourceMetricDailyItem{{Date: "2026-06-27", DetailViewCount: 4}},
		},
	}
	logic := NewGetResourceMetricsLogic(store)

	resp, err := logic.GetResourceMetrics(context.Background(), GetResourceMetricsReq{ResourceID: " resource-1 ", From: "2026-06-01", To: "2026-06-27"})
	if err != nil {
		t.Fatalf("GetResourceMetrics() error = %v", err)
	}

	if store.resourceID != "resource-1" || resp.Summary.DetailViewCount != 4 || len(resp.Daily) != 1 {
		t.Fatalf("resp = %#v, resourceID = %q", resp, store.resourceID)
	}
}

func TestGetMerchantMetricsSummaryReturnsLast7Days(t *testing.T) {
	store := &fakeMetricsQueryStore{
		merchantSummary: model.MerchantMetricsSummary{
			MerchantID: "merchant-1", PublishedResourceCount: 3, ExpiringResourceCount: 1,
			Last7Days: model.MerchantLast7DaysMetrics{ExposureCount: 20, ContactClickCount: 5},
		},
	}
	logic := NewGetMerchantMetricsLogic(store)

	resp, err := logic.GetMerchantMetrics(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatalf("GetMerchantMetrics() error = %v", err)
	}

	if store.merchantID != "merchant-1" || resp.Last7Days.ContactClickCount != 5 {
		t.Fatalf("resp = %#v, merchantID = %q", resp, store.merchantID)
	}
}

type fakeMetricsQueryStore struct {
	resourceID      string
	merchantID      string
	from            string
	to              string
	resourceMetrics model.ResourceMetricsResult
	merchantSummary model.MerchantMetricsSummary
}

func (s *fakeMetricsQueryStore) GetResourceMetrics(ctx context.Context, resourceID string, from string, to string) (model.ResourceMetricsResult, error) {
	s.resourceID = resourceID
	s.from = from
	s.to = to
	return s.resourceMetrics, nil
}

func (s *fakeMetricsQueryStore) GetMerchantMetricsSummary(ctx context.Context, merchantID string) (model.MerchantMetricsSummary, error) {
	s.merchantID = merchantID
	return s.merchantSummary, nil
}
