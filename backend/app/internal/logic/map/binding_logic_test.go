package maplogic

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestBindingLogicCreatesPendingRequest(t *testing.T) {
	store := &fakeBindingStore{
		createdRequest: model.MapBindRequest{
			ID:         "request-1",
			MerchantID: "merchant-1",
			ObjectID:   "object-1",
			SceneCode:  "scene-1",
			Status:     model.MapBindRequestStatusPending,
		},
	}
	logic := NewBindingLogic(store)

	resp, err := logic.SubmitRequest(context.Background(), " merchant-1 ", SubmitMapBindRequestReq{
		ObjectID:        " object-1 ",
		ApplicantUserID: " user-1 ",
		EvidenceImages:  []string{" https://img.example.com/booth.jpg "},
		Note:            " 我是 A001 档口 ",
	})
	if err != nil {
		t.Fatalf("SubmitRequest() error = %v", err)
	}

	if store.createInput.MerchantID != "merchant-1" || store.createInput.ObjectID != "object-1" || store.createInput.ApplicantUserID != "user-1" {
		t.Fatalf("input = %#v, want trimmed merchant/object/user", store.createInput)
	}
	if len(store.createInput.EvidenceImages) != 1 || store.createInput.EvidenceImages[0] != "https://img.example.com/booth.jpg" {
		t.Fatalf("evidence images = %#v, want trimmed image", store.createInput.EvidenceImages)
	}
	if resp.Item.Status != model.MapBindRequestStatusPending {
		t.Fatalf("resp = %#v, want pending request", resp)
	}
}

func TestBindingLogicRejectsDuplicatePendingRequest(t *testing.T) {
	store := &fakeBindingStore{
		createErr: model.ErrMapBindRequestPending,
	}
	logic := NewBindingLogic(store)

	_, err := logic.SubmitRequest(context.Background(), "merchant-1", SubmitMapBindRequestReq{ObjectID: "object-1"})
	if err == nil || errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("SubmitRequest() error = %v, want conflict", err)
	}
}

func TestBindingLogicRejectsReviewWithoutRejectReason(t *testing.T) {
	logic := NewBindingLogic(&fakeBindingStore{})

	_, err := logic.ReviewRequest(context.Background(), "request-1", ReviewMapBindRequestReq{Action: "reject"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("ReviewRequest() error = %v, want validation failed", err)
	}
}

func TestBindingLogicMapsAlreadyBoundConflict(t *testing.T) {
	store := &fakeBindingStore{
		reviewErr: model.ErrMapObjectAlreadyBound,
	}
	logic := NewBindingLogic(store)

	_, err := logic.ReviewRequest(context.Background(), " request-1 ", ReviewMapBindRequestReq{
		Action:     "approve",
		ReviewerID: " admin-1 ",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("ReviewRequest() error = %v, want conflict", err)
	}
	if store.reviewInput.ID != "request-1" || store.reviewInput.ReviewerID != "admin-1" || store.reviewInput.Status != model.MapBindRequestStatusApproved {
		t.Fatalf("review input = %#v, want approve input", store.reviewInput)
	}
}

type fakeBindingStore struct {
	status          model.MapBindingStatus
	candidates      []model.MapBindCandidate
	requests        []model.MapBindRequest
	createdRequest  model.MapBindRequest
	reviewedRequest model.MapBindRequest
	createInput     model.MapBindRequestInput
	reviewInput     model.ReviewMapBindRequestInput
	candidateFilter model.MapBindCandidateFilter
	requestFilter   model.ListMapBindRequestsFilter
	createErr       error
	reviewErr       error
}

func (s *fakeBindingStore) GetMapBindingStatus(ctx context.Context, merchantID string) (model.MapBindingStatus, error) {
	return s.status, nil
}

func (s *fakeBindingStore) ListMapBindCandidates(ctx context.Context, filter model.MapBindCandidateFilter) ([]model.MapBindCandidate, error) {
	s.candidateFilter = filter
	return append([]model.MapBindCandidate(nil), s.candidates...), nil
}

func (s *fakeBindingStore) CreateMapBindRequest(ctx context.Context, input model.MapBindRequestInput) (model.MapBindRequest, error) {
	s.createInput = input
	if s.createErr != nil {
		return model.MapBindRequest{}, s.createErr
	}
	return s.createdRequest, nil
}

func (s *fakeBindingStore) ListMapBindRequests(ctx context.Context, filter model.ListMapBindRequestsFilter) ([]model.MapBindRequest, error) {
	s.requestFilter = filter
	return append([]model.MapBindRequest(nil), s.requests...), nil
}

func (s *fakeBindingStore) ReviewMapBindRequest(ctx context.Context, input model.ReviewMapBindRequestInput) (model.MapBindRequest, error) {
	s.reviewInput = input
	if s.reviewErr != nil {
		return model.MapBindRequest{}, s.reviewErr
	}
	return s.reviewedRequest, nil
}
