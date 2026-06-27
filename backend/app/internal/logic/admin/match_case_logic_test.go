package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestCreateMatchCaseRequiresDemand(t *testing.T) {
	logic := NewMatchCaseLogic(&fakeMatchCaseStore{})

	_, err := logic.CreateMatchCase(context.Background(), CreateMatchCaseReq{OperatorID: "admin-1"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("CreateMatchCase() error = %v, want validation error", err)
	}
}

func TestCreateMatchCasePassesCandidatesAndParticipants(t *testing.T) {
	store := &fakeMatchCaseStore{created: model.MatchCaseResult{ID: "match-1", Status: "open"}}
	logic := NewMatchCaseLogic(store)

	resp, err := logic.CreateMatchCase(context.Background(), CreateMatchCaseReq{
		OperatorID: " admin-1 ", PurchaseDemandID: " demand-1 ",
		ResourceIDs: []string{" resource-1 "}, ParticipantMerchantIDs: []string{" merchant-1 "},
	})
	if err != nil {
		t.Fatalf("CreateMatchCase() error = %v", err)
	}

	if store.createInput.OperatorID != "admin-1" || store.createInput.ResourceIDs[0] != "resource-1" {
		t.Fatalf("createInput = %#v, want trimmed ids", store.createInput)
	}
	if resp.ID != "match-1" || resp.Status != "open" {
		t.Fatalf("resp = %#v, want open match", resp)
	}
}

func TestUpdateMatchCaseStatusRequiresResultForSucceededOrFailed(t *testing.T) {
	logic := NewMatchCaseLogic(&fakeMatchCaseStore{})

	_, err := logic.UpdateMatchCaseStatus(context.Background(), UpdateMatchCaseStatusReq{
		MatchCaseID: "match-1", OperatorID: "admin-1", Status: "succeeded",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("UpdateMatchCaseStatus() error = %v, want validation error", err)
	}
}

func TestUpdateMatchCaseStatusPassesInputToStore(t *testing.T) {
	store := &fakeMatchCaseStore{updated: model.MatchCaseResult{ID: "match-1", Status: "contacted"}}
	logic := NewMatchCaseLogic(store)

	resp, err := logic.UpdateMatchCaseStatus(context.Background(), UpdateMatchCaseStatusReq{
		MatchCaseID: " match-1 ", OperatorID: " admin-1 ", Status: " contacted ",
	})
	if err != nil {
		t.Fatalf("UpdateMatchCaseStatus() error = %v", err)
	}

	if store.updateInput.MatchCaseID != "match-1" || store.updateInput.Status != "contacted" || resp.Status != "contacted" {
		t.Fatalf("updateInput = %#v, resp = %#v", store.updateInput, resp)
	}
}

type fakeMatchCaseStore struct {
	createInput model.CreateMatchCaseInput
	updateInput model.UpdateMatchCaseStatusInput
	created     model.MatchCaseResult
	updated     model.MatchCaseResult
}

func (s *fakeMatchCaseStore) CreateMatchCase(ctx context.Context, input model.CreateMatchCaseInput) (model.MatchCaseResult, error) {
	s.createInput = input
	return s.created, nil
}

func (s *fakeMatchCaseStore) ListMatchCases(ctx context.Context, filter model.ListMatchCasesFilter) (model.ListMatchCasesResult, error) {
	return model.ListMatchCasesResult{}, nil
}

func (s *fakeMatchCaseStore) UpdateMatchCaseStatus(ctx context.Context, input model.UpdateMatchCaseStatusInput) (model.MatchCaseResult, error) {
	s.updateInput = input
	return s.updated, nil
}

func (s *fakeMatchCaseStore) AddMatchCaseResources(ctx context.Context, input model.AddMatchCaseResourcesInput) error {
	return nil
}

func (s *fakeMatchCaseStore) AddMatchCaseParticipants(ctx context.Context, input model.AddMatchCaseParticipantsInput) error {
	return nil
}
