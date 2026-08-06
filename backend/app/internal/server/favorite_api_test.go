package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestFavoriteHandlersThroughGeneratedRoutes(t *testing.T) {
	svcCtx, mock := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	mock.ExpectQuery(`(?s)FROM user_favorite_resources ufr`).
		WithArgs("user-1", int64(3), int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type_code", "title", "category", "district", "price_text", "quantity_text",
			"merchant_id", "merchant_name", "refreshed_at", "dealt_at", "total",
		}))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/me/favorite-resources?userId=attacker&page=2&pageSize=3", ""), http.StatusOK)

	mock.ExpectQuery(`(?s)FROM user_favorite_resources`).
		WithArgs("user-1", "resource-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/me/favorite-resources/resource-1?userId=attacker", ""), http.StatusOK)

	mock.ExpectQuery(`(?s)INSERT INTO user_favorite_resources`).
		WithArgs("user-1", "resource-2", "canceled").
		WillReturnRows(sqlmock.NewRows([]string{"resource_id", "favorited"}).AddRow("resource-2", false))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodPost, "/api/v1/me/favorite-resources/resource-2?userId=attacker", `{"userId":"attacker","favorited":false}`), http.StatusOK)

	mock.ExpectQuery(`(?s)FROM user_followed_merchants ufm`).
		WithArgs("user-1", int64(4), int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "merchant_type", "profile_status", "main_categories", "logo_url", "followed_at", "total",
		}))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/me/followed-merchants?userId=attacker&page=2&pageSize=4", ""), http.StatusOK)

	mock.ExpectQuery(`(?s)FROM user_followed_merchants`).
		WithArgs("user-1", "merchant-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/me/followed-merchants/merchant-1?userId=attacker", ""), http.StatusOK)

	mock.ExpectQuery(`(?s)INSERT INTO user_followed_merchants`).
		WithArgs("user-1", "merchant-2", "active").
		WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "followed"}).AddRow("merchant-2", true))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodPost, "/api/v1/me/followed-merchants/merchant-2?userId=attacker", `{"userId":"attacker","followed":true}`), http.StatusOK)

	mock.ExpectQuery(`(?s)FROM user_saved_searches uss`).
		WithArgs("user-1", int64(5), int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "city_code", "type_code", "keyword", "category", "verified_only", "created_at", "total",
		}))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodGet, "/api/v1/me/saved-searches?userId=attacker&page=2&pageSize=5", ""), http.StatusOK)

	mock.ExpectQuery(`(?s)INSERT INTO user_saved_searches`).
		WithArgs("user-1", "童装库存", "zhili", "inventory", "库存", "童装", false).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("saved-1"))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodPost, "/api/v1/me/saved-searches?userId=attacker", `{"userId":"attacker","name":"童装库存","cityCode":"zhili","typeCode":"inventory","keyword":"库存","category":"童装"}`), http.StatusOK)

	mock.ExpectExec(`(?s)UPDATE user_saved_searches`).
		WithArgs("saved-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	assertTask6GeneratedStatus(t, server, authenticatedRequest(http.MethodDelete, "/api/v1/me/saved-searches/saved-1?userId=attacker", ""), http.StatusOK)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("favorite generated SQL expectations: %v", err)
	}
}

func TestFavoriteGeneratedRoutesRequireTokenIdentity(t *testing.T) {
	svcCtx, _ := newTask6GeneratedServiceContext(t)
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, svcCtx)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/saved-searches?userId=attacker", strings.NewReader(`{"userId":"attacker","keyword":"库存"}`))
	assertTask6GeneratedStatus(t, server, req, http.StatusUnauthorized)
}

func newTask6GeneratedServiceContext(t *testing.T) (*svc.ServiceContext, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	apiStore := &svc.APIStore{
		UserModel:                model.NewUserModel(db),
		MerchantModel:            model.NewMerchantModel(db),
		ResourceModel:            model.NewResourceModel(db),
		MessageModel:             model.NewMessageModel(db),
		FavoriteModel:            model.NewFavoriteModel(db),
		ResourceMetricDailyModel: model.NewResourceMetricDailyModel(db),
		MerchantMapEventsModel:   model.NewMerchantMapEventsModel(sqlx.NewSqlConnFromDB(db)),
	}
	return &svc.ServiceContext{
		APIStore:         apiStore,
		UserTokenService: &fakeUserTokenService{},
		// 非 nil 的校验服务会安全拒绝普通用户 Token，随后 RequireMerchant 再校验用户管理关系。
		AdminTokenService: adminauth.NewValidatingAdminTokenService(nil, nil),
	}, mock
}

func assertTask6GeneratedStatus(t *testing.T, server http.Handler, req *http.Request, wantStatus int) map[string]interface{} {
	t.Helper()
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return decodeEnvelope(t, rec, wantStatus)
}

func TestInteractionAPIRouterBindsUserToken(t *testing.T) {
	store := &fakeInteractionAPIStore{}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/favorite-resources/resource-1", strings.NewReader(`{"favorited":true}`))
	req.Header.Set("Authorization", "Bearer user-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if store.resourceInput.UserID != "user-1" || store.resourceInput.ResourceID != "resource-1" || !store.resourceInput.Favorited {
		t.Fatalf("route did not bind token user/resource: %+v", store.resourceInput)
	}
}

func TestInteractionAPIRouterRequiresLogin(t *testing.T) {
	store := &fakeInteractionAPIStore{}
	router := NewAPIRouter(store, WithUserTokenService(&fakeUserTokenService{}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/saved-searches", strings.NewReader(`{"keyword":"库存"}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}

type fakeInteractionAPIStore struct {
	*fakeCityAPIStore
	resourceInput model.ResourceFavoriteInput
}

func (s *fakeInteractionAPIStore) ListCityStations(ctx context.Context) ([]model.CityStation, error) {
	return nil, nil
}

func (s *fakeInteractionAPIStore) ListResourceTypes(ctx context.Context, cityCode string) ([]model.ResourceTypeConfig, error) {
	return nil, nil
}

func (s *fakeInteractionAPIStore) SetResourceFavorite(ctx context.Context, input model.ResourceFavoriteInput) (model.ResourceFavoriteState, error) {
	s.resourceInput = input
	return model.ResourceFavoriteState{ResourceID: input.ResourceID, Favorited: input.Favorited}, nil
}

func (s *fakeInteractionAPIStore) ResourceBelongsToUser(ctx context.Context, userID string, resourceID string) (bool, error) {
	return false, nil
}

func (s *fakeInteractionAPIStore) GetResourceFavoriteState(ctx context.Context, userID string, resourceID string) (model.ResourceFavoriteState, error) {
	return model.ResourceFavoriteState{ResourceID: resourceID, Favorited: true}, nil
}

func (s *fakeInteractionAPIStore) ListFavoriteResources(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListResourcesResult, error) {
	return model.ListResourcesResult{Page: 1, PageSize: 20}, nil
}

func (s *fakeInteractionAPIStore) SetMerchantFollow(ctx context.Context, input model.MerchantFollowInput) (model.MerchantFollowState, error) {
	return model.MerchantFollowState{MerchantID: input.MerchantID, Followed: input.Followed}, nil
}

func (s *fakeInteractionAPIStore) GetMerchantFollowState(ctx context.Context, userID string, merchantID string) (model.MerchantFollowState, error) {
	return model.MerchantFollowState{MerchantID: merchantID, Followed: true}, nil
}

func (s *fakeInteractionAPIStore) ListFollowedMerchants(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListFollowedMerchantsResult, error) {
	return model.ListFollowedMerchantsResult{Page: 1, PageSize: 20}, nil
}

func (s *fakeInteractionAPIStore) CreateSavedSearch(ctx context.Context, input model.SavedSearchInput) (model.SavedSearchResult, error) {
	return model.SavedSearchResult{ID: "saved-1"}, nil
}

func (s *fakeInteractionAPIStore) ListSavedSearches(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListSavedSearchesResult, error) {
	return model.ListSavedSearchesResult{Page: 1, PageSize: 20}, nil
}

func (s *fakeInteractionAPIStore) DeleteSavedSearch(ctx context.Context, userID string, savedSearchID string) error {
	return nil
}
