package server

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestNewProductionAPIRouterRequiresResourceAPIStore(t *testing.T) {
	_, err := NewProductionAPIRouter(
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
	if err == nil {
		t.Fatal("NewProductionAPIRouter() error = nil, want missing ResourceAPIStore error")
	}
	if !strings.Contains(err.Error(), "ResourceAPIStore") {
		t.Fatalf("error = %q, want missing ResourceAPIStore", err.Error())
	}
}

func TestNewProductionAPIRouterAcceptsCompleteAuthorizationDependencies(t *testing.T) {
	store := newFakeFullAPIStore()
	router, err := NewProductionAPIRouter(
		store,
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

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources?cityCode=zhili&page=1&pageSize=2", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s, want 200", rec.Code, rec.Body.String())
	}
	if store.listFilter.CityCode != "zhili" || store.listFilter.Page != 1 || store.listFilter.PageSize != 2 {
		t.Fatalf("list filter = %#v, want zhili page 1 size 2", store.listFilter)
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

func (s *fakeFullAPIStore) UpsertWechatUser(ctx context.Context, input model.UpsertWechatUserInput) (model.UserProfile, error) {
	return model.UserProfile{ID: "user-1"}, nil
}

func (s *fakeFullAPIStore) GetUserProfile(ctx context.Context, userID string) (model.UserProfile, error) {
	return model.UserProfile{ID: userID}, nil
}

func (s *fakeFullAPIStore) BindUserPhone(ctx context.Context, userID string, phone string) (model.UserProfile, error) {
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
