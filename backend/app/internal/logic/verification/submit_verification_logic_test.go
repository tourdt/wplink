package verification

import (
	"context"
	"database/sql"
	"reflect"
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

func TestSubmitVerificationRejectsPaymentPendingRecord(t *testing.T) {
	store := &fakeVerificationStore{
		latest: model.VerificationBrief{
			ID:     "verification-1",
			Status: model.VerificationStatusPaymentPending,
		},
		submitResult: model.VerificationResult{ID: "verification-2", Status: "pending"},
	}
	logic := NewSubmitVerificationLogic(store)

	_, err := logic.SubmitVerification(context.Background(), SubmitVerificationReq{
		MerchantID: "merchant-1", ApplicantUserID: "user-1", VerificationType: "factory",
	})

	if err == nil || errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("SubmitVerification() error = %v, want state conflict", err)
	}
	if store.submitCalled {
		t.Fatal("SubmitVerification() should not insert a new record while payment is pending")
	}
}

func TestSubmitVerificationRejectsPendingRecord(t *testing.T) {
	store := &fakeVerificationStore{
		latest: model.VerificationBrief{
			ID:     "verification-1",
			Status: model.VerificationStatusPending,
		},
		submitResult: model.VerificationResult{ID: "verification-2", Status: "pending"},
	}
	logic := NewSubmitVerificationLogic(store)

	_, err := logic.SubmitVerification(context.Background(), SubmitVerificationReq{
		MerchantID: "merchant-1", ApplicantUserID: "user-1", VerificationType: "factory",
	})

	if err == nil || errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("SubmitVerification() error = %v, want state conflict", err)
	}
	if store.submitCalled {
		t.Fatal("SubmitVerification() should not insert a new record while review is pending")
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
		ExpiresAt:  "2027-07-01T12:00:00+08:00",
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
	if resp.ExpiresAt != "2027-07-01T12:00:00+08:00" {
		t.Fatalf("ExpiresAt = %q, want mapped annual expiration", resp.ExpiresAt)
	}
	if resp.Materials["socialCreditCode"] != "91330000MA00000000" || resp.Materials["contactPhone"] != "13800138000" {
		t.Fatalf("materials = %#v, want submitted verification materials", resp.Materials)
	}
}

func TestLatestVerificationRespExposesAnnualExpiration(t *testing.T) {
	field, ok := reflect.TypeOf(LatestVerificationResp{}).FieldByName("ExpiresAt")
	if !ok {
		t.Fatal("LatestVerificationResp should expose ExpiresAt for annual recertification")
	}
	if field.Tag.Get("json") != "expiresAt,omitempty" {
		t.Fatalf("ExpiresAt json tag = %q, want expiresAt,omitempty", field.Tag.Get("json"))
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
	submitCalled     bool
	submitResult     model.VerificationResult
	latestMerchantID string
	latest           model.VerificationBrief
	latestErr        error
}

func (s *fakeVerificationStore) SubmitVerification(ctx context.Context, input model.SubmitVerificationInput) (model.VerificationResult, error) {
	s.submitInput = input
	s.submitCalled = true
	return s.submitResult, nil
}

func (s *fakeVerificationStore) GetLatestVerification(ctx context.Context, merchantID string) (model.VerificationBrief, error) {
	s.latestMerchantID = merchantID
	return s.latest, s.latestErr
}
