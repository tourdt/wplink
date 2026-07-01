package verification

import (
	"context"
	"database/sql"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestSubmitVerificationRejectsUnsupportedType(t *testing.T) {
	logic := NewSubmitVerificationLogic(&fakeVerificationStore{})

	_, err := logic.SubmitVerification(context.Background(), SubmitVerificationReq{
		MerchantID: "merchant-1", ApplicantUserID: "user-1", VerificationType: "buyer",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SubmitVerification() error = %v, want validation error", err)
	}
}

func TestSubmitVerificationPassesTrimmedInput(t *testing.T) {
	store := &fakeVerificationStore{submitResult: model.VerificationResult{ID: "verification-1", Status: "pending"}}
	logic := NewSubmitVerificationLogic(store)

	resp, err := logic.SubmitVerification(context.Background(), SubmitVerificationReq{
		MerchantID: " merchant-1 ", ApplicantUserID: " user-1 ", VerificationType: " factory ",
		BusinessName: " 织里样板童装厂 ", Materials: model.JSONMap{"license": "ok"},
	})
	if err != nil {
		t.Fatalf("SubmitVerification() error = %v", err)
	}

	if store.submitInput.MerchantID != "merchant-1" || store.submitInput.BusinessName != "织里样板童装厂" {
		t.Fatalf("submitInput = %#v, want trimmed input", store.submitInput)
	}
	if resp.ID != "verification-1" || resp.Status != "pending" {
		t.Fatalf("resp = %#v, want pending verification", resp)
	}
}

func TestGetLatestVerificationRequiresMerchantID(t *testing.T) {
	logic := NewGetLatestVerificationLogic(&fakeVerificationStore{})

	_, err := logic.GetLatestVerification(context.Background(), " ")
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("GetLatestVerification() error = %v, want validation error", err)
	}
}

func TestGetLatestVerificationMapsStoreResult(t *testing.T) {
	store := &fakeVerificationStore{latest: model.VerificationBrief{
		ID:               "verification-1",
		VerificationType: "factory",
		Status:           "rejected",
		BusinessName:     "织里样板童装厂",
		LicenseURL:       "https://cdn.example.com/license.jpg",
		StorefrontURL:    "https://cdn.example.com/storefront.jpg",
		Materials: model.JSONMap{
			"socialCreditCode": "91330000MA00000000",
			"contactPhone":     "13800138000",
		},
		ReviewNote: "营业执照照片不清晰，请重新上传",
	}}
	logic := NewGetLatestVerificationLogic(store)

	resp, err := logic.GetLatestVerification(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatalf("GetLatestVerification() error = %v", err)
	}

	if store.latestMerchantID != "merchant-1" || resp.Status != "rejected" || resp.ReviewNote != "营业执照照片不清晰，请重新上传" {
		t.Fatalf("merchantID = %q, resp = %#v", store.latestMerchantID, resp)
	}
	if resp.BusinessName != "织里样板童装厂" || resp.LicenseURL == "" || resp.StorefrontURL == "" {
		t.Fatalf("resp = %#v, want submitted business and image fields", resp)
	}
	if resp.Materials["socialCreditCode"] != "91330000MA00000000" || resp.Materials["contactPhone"] != "13800138000" {
		t.Fatalf("materials = %#v, want submitted verification materials", resp.Materials)
	}
}

func TestGetLatestVerificationReturnsNoneWhenMissing(t *testing.T) {
	store := &fakeVerificationStore{latestErr: sql.ErrNoRows}
	logic := NewGetLatestVerificationLogic(store)

	resp, err := logic.GetLatestVerification(context.Background(), "merchant-1")
	if err != nil {
		t.Fatalf("GetLatestVerification() error = %v, want nil", err)
	}
	if resp.Status != "none" {
		t.Fatalf("status = %q, want none", resp.Status)
	}
}

type fakeVerificationStore struct {
	submitInput      model.SubmitVerificationInput
	submitResult     model.VerificationResult
	latestMerchantID string
	latest           model.VerificationBrief
	latestErr        error
}

func (s *fakeVerificationStore) SubmitVerification(ctx context.Context, input model.SubmitVerificationInput) (model.VerificationResult, error) {
	s.submitInput = input
	return s.submitResult, nil
}

func (s *fakeVerificationStore) GetLatestVerification(ctx context.Context, merchantID string) (model.VerificationBrief, error) {
	s.latestMerchantID = merchantID
	return s.latest, s.latestErr
}
