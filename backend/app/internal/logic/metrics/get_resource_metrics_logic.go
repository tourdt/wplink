package metrics

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ResourceMetricsStore interface {
	GetResourceMetrics(ctx context.Context, resourceID string, from string, to string) (model.ResourceMetricsResult, error)
}

type GetResourceMetricsReq struct {
	ResourceID string
	From       string
	To         string
}

type ResourceMetricsSummary struct {
	ExposureCount     int64
	DetailViewCount   int64
	PhoneClickCount   int64
	WechatCopyCount   int64
	DealFeedbackCount int64
}

type ResourceMetricDailyItem struct {
	Date            string
	ExposureCount   int64
	DetailViewCount int64
	PhoneClickCount int64
	WechatCopyCount int64
}

type ResourceMetricsResp struct {
	ResourceID string
	Summary    ResourceMetricsSummary
	Daily      []ResourceMetricDailyItem
}

type GetResourceMetricsLogic struct {
	store ResourceMetricsStore
}

func NewGetResourceMetricsLogic(store ResourceMetricsStore) *GetResourceMetricsLogic {
	return &GetResourceMetricsLogic{store: store}
}

func (l *GetResourceMetricsLogic) GetResourceMetrics(ctx context.Context, req GetResourceMetricsReq) (ResourceMetricsResp, error) {
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		return ResourceMetricsResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	result, err := l.store.GetResourceMetrics(ctx, resourceID, strings.TrimSpace(req.From), strings.TrimSpace(req.To))
	if err != nil {
		return ResourceMetricsResp{}, err
	}
	resp := ResourceMetricsResp{
		ResourceID: result.ResourceID,
		Summary: ResourceMetricsSummary{
			ExposureCount:     result.Summary.ExposureCount,
			DetailViewCount:   result.Summary.DetailViewCount,
			PhoneClickCount:   result.Summary.PhoneClickCount,
			WechatCopyCount:   result.Summary.WechatCopyCount,
			DealFeedbackCount: result.Summary.DealFeedbackCount,
		},
	}
	for _, item := range result.Daily {
		resp.Daily = append(resp.Daily, ResourceMetricDailyItem{
			Date: item.Date, ExposureCount: item.ExposureCount, DetailViewCount: item.DetailViewCount,
			PhoneClickCount: item.PhoneClickCount, WechatCopyCount: item.WechatCopyCount,
		})
	}
	return resp, nil
}
