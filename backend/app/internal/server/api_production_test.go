package server

import (
	"context"
	"strings"
	"testing"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
)

func TestNewProductionAPIRouterRequiresAuthorizationDependencies(t *testing.T) {
	_, err := NewProductionAPIRouter(&fakeCityAPIStore{})
	if err == nil {
		t.Fatal("NewProductionAPIRouter() error = nil, want missing dependency error")
	}
	message := err.Error()
	for _, want := range []string{
		"AdminLoginService",
		"AdminTokenService",
		"UserTokenService",
		"UploadTokenService",
		"WechatSessionClient",
		"SMSVerifier",
		"MerchantPermissionStore",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want missing %s", message, want)
		}
	}
}

func TestNewProductionAPIRouterAcceptsCompleteAuthorizationDependencies(t *testing.T) {
	router, err := NewProductionAPIRouter(
		&fakeProductionDependencyStore{},
		WithAdminLoginService(fakeAdminLoginService{}),
		WithAdminTokenService(&strictUploadAdminTokenService{
			subject: session.AdminTokenSubject{UserID: "admin-1", Roles: []string{"platform_operator"}},
		}),
		WithUserTokenService(&strictUploadUserTokenService{}),
		WithUploadTokenService(fakeUploadTokenService{}),
		WithWechatSessionClient(fakeWechatSessionClient{}),
		WithSMSVerifier(fakeSMSVerifier{}),
	)
	if err != nil {
		t.Fatalf("NewProductionAPIRouter() error = %v, want nil", err)
	}
	if router == nil {
		t.Fatal("NewProductionAPIRouter() router = nil, want handler")
	}
}

type fakeProductionDependencyStore struct {
	fakeCityAPIStore
}

func (s *fakeProductionDependencyStore) UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error) {
	return true, nil
}

func (s *fakeProductionDependencyStore) UpsertWechatUser(ctx context.Context, input model.UpsertWechatUserInput) (model.UserProfile, error) {
	return model.UserProfile{ID: "user-1"}, nil
}

func (s *fakeProductionDependencyStore) GetUserProfile(ctx context.Context, userID string) (model.UserProfile, error) {
	return model.UserProfile{ID: userID}, nil
}

func (s *fakeProductionDependencyStore) BindUserPhone(ctx context.Context, userID string, phone string) (model.UserProfile, error) {
	return model.UserProfile{ID: userID, Phone: phone}, nil
}

type fakeWechatSessionClient struct{}

func (fakeWechatSessionClient) Code2Session(ctx context.Context, code string) (authlogic.WechatSession, error) {
	return authlogic.WechatSession{OpenID: "openid"}, nil
}

func (fakeWechatSessionClient) GetPhoneNumber(ctx context.Context, code string) (authlogic.WechatPhoneNumber, error) {
	return authlogic.WechatPhoneNumber{PurePhoneNumber: "18800000003"}, nil
}

type fakeSMSVerifier struct{}

func (fakeSMSVerifier) VerifySMSCode(ctx context.Context, phone string, code string) error {
	return nil
}

func (fakeSMSVerifier) SendSMSCode(ctx context.Context, phone string) error {
	return nil
}
