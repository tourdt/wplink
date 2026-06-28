package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	uploadlogic "wplink/backend/app/internal/logic/upload"
)

func TestAPIRouterCreatesUploadToken(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{}, WithUploadTokenService(fakeUploadTokenService{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/token", strings.NewReader(`{"purpose":"resource","fileName":"a.png","contentType":"image/png","fileSize":128}`))
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["uploadToken"] != "upload-token" || data["objectKey"] != "uploads/resource/a.png" {
		t.Fatalf("data = %#v, want upload token response", data)
	}
}

type fakeUploadTokenService struct{}

func (fakeUploadTokenService) CreateUploadToken(_ context.Context, req uploadlogic.CreateUploadTokenReq) (uploadlogic.CreateUploadTokenResp, error) {
	return uploadlogic.CreateUploadTokenResp{
		UploadToken:   "upload-token",
		UploadURL:     "https://upload-z2.qiniup.com",
		PublicBaseURL: "https://cdn.example.com",
		ObjectKey:     "uploads/" + req.Purpose + "/" + req.FileName,
		ExpiresAt:     "2026-06-28T10:15:00Z",
	}, nil
}
