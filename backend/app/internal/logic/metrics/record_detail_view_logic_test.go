package metrics

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestRecordDetailViewRejectsEmptyResourceID(t *testing.T) {
	logic := NewRecordDetailViewLogic(&fakeMetricStore{})

	err := logic.RecordDetailView(context.Background(), RecordDetailViewReq{ResourceID: " "})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("RecordDetailView() error = %v, want validation error", err)
	}
}

func TestRecordDetailViewIncrementsDetailCounter(t *testing.T) {
	store := &fakeMetricStore{}
	logic := NewRecordDetailViewLogic(store)

	err := logic.RecordDetailView(context.Background(), RecordDetailViewReq{ResourceID: " resource-1 "})
	if err != nil {
		t.Fatalf("RecordDetailView() error = %v", err)
	}

	if store.metricDelta.ResourceID != "resource-1" || store.metricDelta.DetailViewCount != 1 {
		t.Fatalf("metricDelta = %#v, want detail view increment", store.metricDelta)
	}
}

type fakeMetricStore struct {
	metricDelta model.ResourceMetricDelta
}

func (s *fakeMetricStore) UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error {
	s.metricDelta = delta
	return nil
}
