package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
)

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
