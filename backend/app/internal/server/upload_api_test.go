package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authlogic "wplink/backend/app/internal/logic/auth"
	uploadlogic "wplink/backend/app/internal/logic/upload"
	"wplink/backend/app/internal/session"
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

func TestAPIRouterRequiresUploadTokenAuthenticationWhenTokenServicesConfigured(t *testing.T) {
	router := NewAPIRouter(
		&fakeCityAPIStore{},
		WithUploadTokenService(fakeUploadTokenService{}),
		WithUserTokenService(&strictUploadUserTokenService{}),
		WithAdminTokenService(&strictUploadAdminTokenService{
			subject: session.AdminTokenSubject{UserID: "admin-1", Roles: []string{"platform_operator"}},
		}),
	)

	unauthorizedRec := httptest.NewRecorder()
	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/token", strings.NewReader(`{"purpose":"resource","fileName":"a.png","contentType":"image/png","fileSize":128}`))
	router.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s, want unauthorized", unauthorizedRec.Code, unauthorizedRec.Body.String())
	}

	userRec := httptest.NewRecorder()
	userReq := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/token", strings.NewReader(`{"purpose":"resource","fileName":"a.png","contentType":"image/png","fileSize":128}`))
	userReq.Header.Set("Authorization", "Bearer user-token")
	router.ServeHTTP(userRec, userReq)
	decodeEnvelopeData(t, userRec, http.StatusOK)

	adminRec := httptest.NewRecorder()
	adminReq := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/token", strings.NewReader(`{"purpose":"resource","fileName":"a.png","contentType":"image/png","fileSize":128}`))
	adminReq.Header.Set("Authorization", "Bearer admin-token")
	router.ServeHTTP(adminRec, adminReq)
	decodeEnvelopeData(t, adminRec, http.StatusOK)
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

type strictUploadUserTokenService struct{}

func (s *strictUploadUserTokenService) IssueUserToken(ctx context.Context, subject session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (s *strictUploadUserTokenService) ParseUserToken(ctx context.Context, token string) (session.UserTokenSubject, error) {
	if token != "user-token" {
		return session.UserTokenSubject{}, errors.New("invalid user token")
	}
	return session.UserTokenSubject{UserID: "user-1", Roles: []string{authlogic.RoleNormalUser}}, nil
}

type strictUploadAdminTokenService struct {
	subject session.AdminTokenSubject
}

func (s *strictUploadAdminTokenService) ParseAdminToken(ctx context.Context, token string) (session.AdminTokenSubject, error) {
	if token != "admin-token" {
		return session.AdminTokenSubject{}, errors.New("invalid admin token")
	}
	return s.subject, nil
}
