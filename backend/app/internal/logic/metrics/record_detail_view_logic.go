package metrics

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type MetricUpsertStore interface {
	UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error
}

type RecordDetailViewReq struct {
	ResourceID string
}

type RecordDetailViewLogic struct {
	store MetricUpsertStore
}

func NewRecordDetailViewLogic(store MetricUpsertStore) *RecordDetailViewLogic {
	return &RecordDetailViewLogic{store: store}
}

func (l *RecordDetailViewLogic) RecordDetailView(ctx context.Context, req RecordDetailViewReq) error {
	resourceID := strings.TrimSpace(req.ResourceID)
	if resourceID == "" {
		return errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	return l.store.UpsertResourceMetric(ctx, model.ResourceMetricDelta{ResourceID: resourceID, DetailViewCount: 1})
}
