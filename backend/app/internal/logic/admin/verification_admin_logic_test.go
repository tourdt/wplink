package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListPendingVerificationsReturnsItems(t *testing.T) {
	store := &fakeVerificationAdminStore{
		pending: model.ListPendingVerificationsResult{
			Items: []model.PendingVerificationItem{{ID: "verification-1", MerchantName: "织里样板童装厂", VerificationType: "factory", Status: "pending"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewVerificationAdminLogic(store)

	resp, err := logic.ListPendingVerifications(context.Background(), ListPendingVerificationsReq{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListPendingVerifications() error = %v", err)
	}

	if len(resp.Items) != 1 || resp.Items[0].MerchantName != "织里样板童装厂" {
		t.Fatalf("items = %#v, want pending verification", resp.Items)
	}
}

func TestReviewVerificationRejectsMissingReasonForReject(t *testing.T) {
	logic := NewVerificationAdminLogic(&fakeVerificationAdminStore{})

	_, err := logic.ReviewVerification(context.Background(), ReviewVerificationReq{
		VerificationID: "verification-1", ReviewerID: "admin-1", Action: "reject",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("ReviewVerification() error = %v, want validation error", err)
	}
}

func TestReviewVerificationApprovesAndPassesReviewer(t *testing.T) {
	store := &fakeVerificationAdminStore{reviewResult: model.ReviewVerificationResult{ID: "verification-1", Status: "verified"}}
	logic := NewVerificationAdminLogic(store)

	resp, err := logic.ReviewVerification(context.Background(), ReviewVerificationReq{
		VerificationID: " verification-1 ", ReviewerID: " admin-1 ", Action: " approve ",
	})
	if err != nil {
		t.Fatalf("ReviewVerification() error = %v", err)
	}

	if store.reviewInput.VerificationID != "verification-1" || store.reviewInput.ReviewerID != "admin-1" {
		t.Fatalf("reviewInput = %#v, want trimmed input", store.reviewInput)
	}
	if resp.Status != "verified" {
		t.Fatalf("status = %q, want verified", resp.Status)
	}
}

type fakeVerificationAdminStore struct {
	pending      model.ListPendingVerificationsResult
	reviewInput  model.ReviewVerificationInput
	reviewResult model.ReviewVerificationResult
}

func (s *fakeVerificationAdminStore) ListPendingVerifications(ctx context.Context, filter model.PendingVerificationsFilter) (model.ListPendingVerificationsResult, error) {
	return s.pending, nil
}

func (s *fakeVerificationAdminStore) ReviewVerification(ctx context.Context, input model.ReviewVerificationInput) (model.ReviewVerificationResult, error) {
	s.reviewInput = input
	return s.reviewResult, nil
}
