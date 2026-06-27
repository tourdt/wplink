package metrics

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestRecordContactRejectsUnsupportedAction(t *testing.T) {
	logic := NewRecordContactLogic(&fakeContactStore{})

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", Action: "email"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("RecordContact() error = %v, want validation error", err)
	}
}

func TestRecordContactRecordsPhoneEventAndMetric(t *testing.T) {
	store := &fakeContactStore{eventResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"}}
	logic := NewRecordContactLogic(store)

	resp, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: " resource-1 ", UserID: " user-1 ", Action: "phone"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}

	if store.eventInput.ResourceID != "resource-1" || store.eventInput.Action != "phone" || store.eventInput.UserID != "user-1" {
		t.Fatalf("eventInput = %#v, want trimmed phone event", store.eventInput)
	}
	if store.metricDelta.ContactClickCount != 1 || store.metricDelta.PhoneClickCount != 1 {
		t.Fatalf("metricDelta = %#v, want phone contact counters", store.metricDelta)
	}
	if resp.Message == "" {
		t.Fatal("message is empty")
	}
}

func TestRecordContactRecordsShareMetric(t *testing.T) {
	store := &fakeContactStore{eventResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"}}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", Action: "share"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}

	if store.metricDelta.ContactClickCount != 1 || store.metricDelta.ShareCount != 1 {
		t.Fatalf("metricDelta = %#v, want share counters", store.metricDelta)
	}
}

type fakeContactStore struct {
	eventInput  model.ResourceContactEventInput
	eventResult model.ResourceContactEventResult
	metricDelta model.ResourceMetricDelta
}

func (s *fakeContactStore) RecordResourceContactEvent(ctx context.Context, input model.ResourceContactEventInput) (model.ResourceContactEventResult, error) {
	s.eventInput = input
	return s.eventResult, nil
}

func (s *fakeContactStore) UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error {
	s.metricDelta = delta
	return nil
}
