package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestSubmitResourceRejectsEmptyID(t *testing.T) {
	logic := NewSubmitResourceLogic(&fakeSubmitResourceStore{})

	_, err := logic.SubmitResource(context.Background(), " ")

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestSubmitResourceSetsPending(t *testing.T) {
	store := &fakeSubmitResourceStore{result: model.SubmitResourceResult{ID: "resource-1", Status: "pending"}}
	logic := NewSubmitResourceLogic(store)

	resp, err := logic.SubmitResource(context.Background(), " resource-1 ")
	if err != nil {
		t.Fatalf("SubmitResource() error = %v", err)
	}

	if store.resourceID != "resource-1" {
		t.Fatalf("resourceID = %q, want trimmed resource-1", store.resourceID)
	}
	if resp.Status != "pending" || resp.Message != "已提交审核，审核通过后将展示给买家" {
		t.Fatalf("resp = %#v, want pending submit message", resp)
	}
}

type fakeSubmitResourceStore struct {
	resourceID string
	result     model.SubmitResourceResult
}

func (s *fakeSubmitResourceStore) SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error) {
	s.resourceID = resourceID
	return s.result, nil
}
