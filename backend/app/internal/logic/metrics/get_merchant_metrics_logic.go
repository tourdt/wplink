package metrics

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type MerchantMetricsStore interface {
	GetMerchantMetricsSummary(ctx context.Context, merchantID string) (model.MerchantMetricsSummary, error)
}

type MerchantLast7DaysMetrics struct {
	ExposureCount     int64
	DetailViewCount   int64
	ContactClickCount int64
}

type MerchantMetricsSummaryResp struct {
	MerchantID             string
	PublishedResourceCount int64
	ExpiringResourceCount  int64
	DealtResourceCount     int64
	Last7Days              MerchantLast7DaysMetrics
}

type GetMerchantMetricsLogic struct {
	store MerchantMetricsStore
}

func NewGetMerchantMetricsLogic(store MerchantMetricsStore) *GetMerchantMetricsLogic {
	return &GetMerchantMetricsLogic{store: store}
}

func (l *GetMerchantMetricsLogic) GetMerchantMetrics(ctx context.Context, merchantID string) (MerchantMetricsSummaryResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return MerchantMetricsSummaryResp{}, errx.New(errx.CodeValidationFailed, "商家不存在")
	}
	result, err := l.store.GetMerchantMetricsSummary(ctx, merchantID)
	if err != nil {
		return MerchantMetricsSummaryResp{}, err
	}
	return MerchantMetricsSummaryResp{
		MerchantID:             result.MerchantID,
		PublishedResourceCount: result.PublishedResourceCount,
		ExpiringResourceCount:  result.ExpiringResourceCount,
		DealtResourceCount:     result.DealtResourceCount,
		Last7Days: MerchantLast7DaysMetrics{
			ExposureCount:     result.Last7Days.ExposureCount,
			DetailViewCount:   result.Last7Days.DetailViewCount,
			ContactClickCount: result.Last7Days.ContactClickCount,
		},
	}, nil
}
