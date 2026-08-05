package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	adminauthlogic "wplink/backend/app/internal/logic/adminauth"
	authlogic "wplink/backend/app/internal/logic/auth"
	uploadlogic "wplink/backend/app/internal/logic/upload"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"
)

func TestUploadGeneratedRequiresValidUserOrAdmin(t *testing.T) {
	adminSubject := session.AdminTokenSubject{
		OperatorID:  "admin-1",
		AuthVersion: 1,
		Roles:       []string{permission.RolePlatformOperator},
	}
	adminTokenService := adminauthlogic.NewValidatingAdminTokenService(
		&strictUploadAdminTokenService{subject: adminSubject},
		strictUploadAdminSessionStore{credential: adminauthlogic.AdminCredential{
			OperatorID:  adminSubject.OperatorID,
			AuthVersion: adminSubject.AuthVersion,
			LoginName:   "operator",
			Status:      adminauthlogic.CredentialStatusEnabled,
			Roles:       append([]string(nil), adminSubject.Roles...),
		}},
	)
	server := newGeneratedAPIServer(t, &svc.ServiceContext{
		UploadTokenService: uploadlogic.NewUploadTokenLogic(testUploadStorageConfig()),
		UserTokenService:   &strictUploadUserTokenService{},
		AdminTokenService:  adminTokenService,
	})

	unauthorizedRec := httptest.NewRecorder()
	server.ServeHTTP(unauthorizedRec, uploadRequest(""))
	assertErrorEnvelope(t, unauthorizedRec, http.StatusUnauthorized, errx.CodeUnauthorized, "请先登录")

	badTokenRec := httptest.NewRecorder()
	server.ServeHTTP(badTokenRec, uploadRequest("bad-token"))
	assertErrorEnvelope(t, badTokenRec, http.StatusUnauthorized, errx.CodeUnauthorized, "请先登录")
	if strings.Contains(badTokenRec.Body.String(), "invalid user token") {
		t.Fatalf("bad token response leaked raw verifier error: %s", badTokenRec.Body.String())
	}

	expiredTokenRec := httptest.NewRecorder()
	server.ServeHTTP(expiredTokenRec, uploadRequest("expired-token"))
	assertErrorEnvelope(t, expiredTokenRec, http.StatusUnauthorized, errx.CodeUnauthorized, "请先登录")
	if strings.Contains(expiredTokenRec.Body.String(), "jwt expired") || strings.Contains(expiredTokenRec.Body.String(), "Authorization") {
		t.Fatalf("expired token response leaked raw verifier error: %s", expiredTokenRec.Body.String())
	}

	userRec := httptest.NewRecorder()
	server.ServeHTTP(userRec, uploadRequest("user-token"))
	userData := decodeEnvelopeData(t, userRec, http.StatusOK)
	assertUploadData(t, userData)

	adminRec := httptest.NewRecorder()
	server.ServeHTTP(adminRec, uploadRequest("admin-token"))
	adminData := decodeEnvelopeData(t, adminRec, http.StatusOK)
	assertUploadData(t, adminData)
}

func TestUploadGeneratedPreservesUserAuthenticationDependencyFailures(t *testing.T) {
	var typedNilService *typedNilUploadUserTokenService
	tests := []struct {
		name    string
		service authlogic.TokenService
	}{
		{name: "nil token service", service: nil},
		{name: "typed nil token service", service: typedNilService},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := newGeneratedAPIServer(t, &svc.ServiceContext{
				UploadTokenService: uploadlogic.NewUploadTokenLogic(testUploadStorageConfig()),
				UserTokenService:   tc.service,
				AdminTokenService: adminauthlogic.NewValidatingAdminTokenService(
					&strictUploadAdminTokenService{},
					strictUploadAdminSessionStore{},
				),
			})

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, uploadRequest("user-token"))

			assertErrorEnvelope(t, rec, http.StatusInternalServerError, errx.CodeInternalError, "登录服务暂不可用，请稍后重试")
			if strings.Contains(rec.Body.String(), "typed nil user token service was called") || strings.Contains(rec.Body.String(), "Authorization") {
				t.Fatalf("dependency failure response leaked internal authentication detail: %s", rec.Body.String())
			}
		})
	}
}

func uploadRequest(token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/token", strings.NewReader(`{"purpose":"resource","fileName":"a.png","contentType":"image/png","fileSize":128}`))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func assertUploadData(t *testing.T, data map[string]interface{}) {
	t.Helper()
	if data["uploadUrl"] != "https://upload-z2.qiniup.com" || data["publicBaseUrl"] != "https://cdn.example.com" {
		t.Fatalf("upload endpoints = %#v", data)
	}
	if token, _ := data["uploadToken"].(string); !strings.HasPrefix(token, "ak-test:") {
		t.Fatalf("upload token = %#v, want signed test token", data["uploadToken"])
	}
	if objectKey, _ := data["objectKey"].(string); !strings.HasPrefix(objectKey, "wplink/uploads/resource/") || !strings.HasSuffix(objectKey, ".png") {
		t.Fatalf("object key = %#v, want resource png key", data["objectKey"])
	}
}

func testUploadStorageConfig() config.StorageConfig {
	return config.StorageConfig{
		Provider:            "qiniu-kodo",
		Endpoint:            "https://upload-z2.qiniup.com",
		Bucket:              "wplink-test",
		Region:              "z2",
		AccessKeyID:         "ak-test",
		AccessKeySecret:     "sk-test",
		PublicBaseURL:       "https://cdn.example.com",
		UploadExpire:        time.Minute,
		MaxFileSizeBytes:    1024,
		AllowedContentTypes: []string{"image/png"},
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

type strictUploadUserTokenService struct{}

func (s *strictUploadUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "user-token", nil
}

func (s *strictUploadUserTokenService) ParseUserToken(_ context.Context, token string) (session.UserTokenSubject, error) {
	if token == "expired-token" {
		return session.UserTokenSubject{}, errors.New("jwt expired with Authorization detail")
	}
	if token != "user-token" {
		return session.UserTokenSubject{}, errors.New("invalid user token")
	}
	return session.UserTokenSubject{UserID: "user-1", Roles: []string{authlogic.RoleNormalUser}}, nil
}

type typedNilUploadUserTokenService struct{}

func (*typedNilUploadUserTokenService) IssueUserToken(context.Context, session.UserTokenSubject) (string, error) {
	return "", errors.New("typed nil user token service was called")
}

func (*typedNilUploadUserTokenService) ParseUserToken(context.Context, string) (session.UserTokenSubject, error) {
	return session.UserTokenSubject{}, errors.New("typed nil user token service was called with Authorization")
}

type strictUploadAdminTokenService struct {
	subject session.AdminTokenSubject
}

func (s *strictUploadAdminTokenService) ParseAdminToken(_ context.Context, token string) (session.AdminTokenSubject, error) {
	if token != "admin-token" {
		return session.AdminTokenSubject{}, errors.New("invalid admin token")
	}
	return s.subject, nil
}

type strictUploadAdminSessionStore struct {
	credential adminauthlogic.AdminCredential
}

func (s strictUploadAdminSessionStore) FindCurrentAdminSession(context.Context, string) (adminauthlogic.AdminCredential, error) {
	return s.credential, nil
}
