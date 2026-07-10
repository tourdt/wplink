package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestReviewResourceRejectsInvalidAction(t *testing.T) {
	logic := NewReviewResourceLogic(&fakeReviewResourceStore{})

	_, err := logic.ReviewResource(context.Background(), "resource-1", ReviewResourceReq{Action: "bad"})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestReviewResourceApprovesResource(t *testing.T) {
	store := &fakeReviewResourceStore{result: model.ReviewResourceResult{ID: "resource-1", Status: "published"}}
	logic := NewReviewResourceLogic(store)

	resp, err := logic.ReviewResource(context.Background(), "resource-1", ReviewResourceReq{Action: "approve"})
	if err != nil {
		t.Fatalf("ReviewResource() error = %v", err)
	}

	if store.input.Action != "approve" {
		t.Fatalf("action = %q, want approve", store.input.Action)
	}
	if resp.Status != "published" || resp.Message != "资源已审核通过" {
		t.Fatalf("resp = %#v, want published approve message", resp)
	}
	if len(store.growthInputs) != 2 {
		t.Fatalf("growthInputs = %#v, want first-approved and approved-count events", store.growthInputs)
	}
	if store.growthInputs[0].EventType != model.GrowthEventResourceFirstApproved || store.growthInputs[0].ResourceID != "resource-1" {
		t.Fatalf("growthInputs[0] = %#v, want first approved event", store.growthInputs[0])
	}
	if store.growthInputs[1].EventType != model.GrowthEventResourceApprovedCountReached || store.growthInputs[1].ResourceID != "resource-1" {
		t.Fatalf("growthInputs[1] = %#v, want approved count event", store.growthInputs[1])
	}
}

func TestReviewResourceRejectRequiresReason(t *testing.T) {
	logic := NewReviewResourceLogic(&fakeReviewResourceStore{})

	_, err := logic.ReviewResource(context.Background(), "resource-1", ReviewResourceReq{Action: "reject"})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

type fakeReviewResourceStore struct {
	input        model.ReviewResourceInput
	result       model.ReviewResourceResult
	growthInputs []model.GrowthEventInput
}

func (s *fakeReviewResourceStore) ReviewResource(ctx context.Context, resourceID string, input model.ReviewResourceInput) (model.ReviewResourceResult, error) {
	s.input = input
	return s.result, nil
}

func (s *fakeReviewResourceStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.growthInputs = append(s.growthInputs, input)
	return nil, nil
}
