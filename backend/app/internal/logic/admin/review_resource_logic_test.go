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
}

func TestReviewResourceRejectRequiresReason(t *testing.T) {
	logic := NewReviewResourceLogic(&fakeReviewResourceStore{})

	_, err := logic.ReviewResource(context.Background(), "resource-1", ReviewResourceReq{Action: "reject"})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

type fakeReviewResourceStore struct {
	input  model.ReviewResourceInput
	result model.ReviewResourceResult
}

func (s *fakeReviewResourceStore) ReviewResource(ctx context.Context, resourceID string, input model.ReviewResourceInput) (model.ReviewResourceResult, error) {
	s.input = input
	return s.result, nil
}
