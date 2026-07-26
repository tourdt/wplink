package metrics

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestRecordResourceExposuresNormalizesAndDeduplicatesBatch(t *testing.T) {
	store := &fakeResourceExposureStore{recorded: 1}
	logic := NewRecordResourceExposuresLogic(store)

	resp, err := logic.RecordResourceExposures(context.Background(), RecordResourceExposuresReq{
		VisitorKey: " visitor-1 ",
		SessionID:  " session-1 ",
		Source:     "search",
		UserID:     " user-1 ",
		Items: []ResourceExposureItem{
			{ResourceID: " resource-1 ", VisibleDurationMS: 900},
			{ResourceID: "resource-1", VisibleDurationMS: 1500},
			{ResourceID: "resource-2", VisibleDurationMS: 300},
		},
	})
	if err != nil {
		t.Fatalf("RecordResourceExposures() error = %v", err)
	}
	if resp.RecordedCount != 1 {
		t.Fatalf("RecordedCount = %d, want 1", resp.RecordedCount)
	}
	if store.input.VisitorKey != "visitor-1" || store.input.SessionID != "session-1" || store.input.UserID != "user-1" {
		t.Fatalf("input identity = %#v, want normalized values", store.input)
	}
	if len(store.input.Items) != 1 || store.input.Items[0].ResourceID != "resource-1" {
		t.Fatalf("items = %#v, want one valid deduplicated resource", store.input.Items)
	}
}

func TestRecordResourceExposuresRejectsUnknownSource(t *testing.T) {
	logic := NewRecordResourceExposuresLogic(&fakeResourceExposureStore{})
	_, err := logic.RecordResourceExposures(context.Background(), RecordResourceExposuresReq{
		VisitorKey: "visitor-1",
		SessionID:  "session-1",
		Source:     "detail",
		Items:      []ResourceExposureItem{{ResourceID: "resource-1", VisibleDurationMS: 900}},
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error = %v, want validation failed", err)
	}
}

type fakeResourceExposureStore struct {
	input    model.RecordResourceExposuresInput
	recorded int64
}

func (s *fakeResourceExposureStore) RecordResourceExposures(ctx context.Context, input model.RecordResourceExposuresInput) (int64, error) {
	s.input = input
	return s.recorded, nil
}
