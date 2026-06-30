package favorite

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestToggleResourceFavoriteRequiresUserAndResource(t *testing.T) {
	logic := NewInteractionLogic(&fakeInteractionStore{})

	if _, err := logic.SetResourceFavorite(context.Background(), "", SetResourceFavoriteReq{ResourceID: "resource-1", Favorited: true}); err == nil {
		t.Fatalf("SetResourceFavorite() expected missing user error")
	}
	if _, err := logic.SetResourceFavorite(context.Background(), "user-1", SetResourceFavoriteReq{Favorited: true}); err == nil {
		t.Fatalf("SetResourceFavorite() expected missing resource error")
	}
}

func TestToggleResourceFavoritePersistsState(t *testing.T) {
	store := &fakeInteractionStore{}
	logic := NewInteractionLogic(store)

	resp, err := logic.SetResourceFavorite(context.Background(), " user-1 ", SetResourceFavoriteReq{ResourceID: " resource-1 ", Favorited: true})
	if err != nil {
		t.Fatalf("SetResourceFavorite() error = %v", err)
	}
	if !resp.Favorited || store.resourceInput.UserID != "user-1" || store.resourceInput.ResourceID != "resource-1" {
		t.Fatalf("favorite state not persisted correctly: resp=%+v input=%+v", resp, store.resourceInput)
	}
}

func TestToggleResourceFavoriteRejectsOwnResource(t *testing.T) {
	store := &fakeInteractionStore{resourceOwnedByUser: true}
	logic := NewInteractionLogic(store)

	_, err := logic.SetResourceFavorite(context.Background(), "user-1", SetResourceFavoriteReq{ResourceID: "resource-1", Favorited: true})
	if err == nil {
		t.Fatalf("SetResourceFavorite() expected own resource error")
	}
	if errx.CodeOf(err) != errx.CodeForbidden {
		t.Fatalf("SetResourceFavorite() code = %s, want %s", errx.CodeOf(err), errx.CodeForbidden)
	}
	if store.setResourceFavoriteCalled {
		t.Fatalf("SetResourceFavorite() should not persist own resource favorite")
	}
}

func TestToggleResourceFavoriteAllowsOwnResourceUnfavorite(t *testing.T) {
	store := &fakeInteractionStore{resourceOwnedByUser: true}
	logic := NewInteractionLogic(store)

	resp, err := logic.SetResourceFavorite(context.Background(), "user-1", SetResourceFavoriteReq{ResourceID: "resource-1", Favorited: false})
	if err != nil {
		t.Fatalf("SetResourceFavorite() error = %v", err)
	}
	if resp.Favorited || !store.setResourceFavoriteCalled {
		t.Fatalf("unfavorite state not persisted correctly: resp=%+v called=%v", resp, store.setResourceFavoriteCalled)
	}
}

func TestToggleMerchantFollowPersistsState(t *testing.T) {
	store := &fakeInteractionStore{}
	logic := NewInteractionLogic(store)

	resp, err := logic.SetMerchantFollow(context.Background(), " user-1 ", SetMerchantFollowReq{MerchantID: " merchant-1 ", Followed: true})
	if err != nil {
		t.Fatalf("SetMerchantFollow() error = %v", err)
	}
	if !resp.Followed || store.merchantInput.UserID != "user-1" || store.merchantInput.MerchantID != "merchant-1" {
		t.Fatalf("follow state not persisted correctly: resp=%+v input=%+v", resp, store.merchantInput)
	}
}

func TestCreateSavedSearchRequiresSearchCondition(t *testing.T) {
	logic := NewInteractionLogic(&fakeInteractionStore{})

	_, err := logic.CreateSavedSearch(context.Background(), "user-1", CreateSavedSearchReq{CityCode: "zhili"})
	if err == nil {
		t.Fatalf("CreateSavedSearch() expected condition validation error")
	}
}

func TestCreateSavedSearchPersistsNormalizedQuery(t *testing.T) {
	store := &fakeInteractionStore{savedSearchResult: model.SavedSearchResult{ID: "saved-1"}}
	logic := NewInteractionLogic(store)

	resp, err := logic.CreateSavedSearch(context.Background(), " user-1 ", CreateSavedSearchReq{
		Name: "  急清库存  ", CityCode: " zhili ", TypeCode: " inventory ", Keyword: "  夏款  ", VerifiedOnly: true,
	})
	if err != nil {
		t.Fatalf("CreateSavedSearch() error = %v", err)
	}
	if resp.ID != "saved-1" || store.savedSearchInput.UserID != "user-1" || store.savedSearchInput.Keyword != "夏款" || store.savedSearchInput.Name != "急清库存" {
		t.Fatalf("saved search not normalized: resp=%+v input=%+v", resp, store.savedSearchInput)
	}
}

type fakeInteractionStore struct {
	resourceInput             model.ResourceFavoriteInput
	merchantInput             model.MerchantFollowInput
	savedSearchInput          model.SavedSearchInput
	savedSearchResult         model.SavedSearchResult
	resourceOwnedByUser       bool
	setResourceFavoriteCalled bool
}

func (s *fakeInteractionStore) SetResourceFavorite(ctx context.Context, input model.ResourceFavoriteInput) (model.ResourceFavoriteState, error) {
	s.resourceInput = input
	s.setResourceFavoriteCalled = true
	return model.ResourceFavoriteState{ResourceID: input.ResourceID, Favorited: input.Favorited}, nil
}

func (s *fakeInteractionStore) ResourceBelongsToUser(ctx context.Context, userID string, resourceID string) (bool, error) {
	return s.resourceOwnedByUser, nil
}

func (s *fakeInteractionStore) GetResourceFavoriteState(ctx context.Context, userID string, resourceID string) (model.ResourceFavoriteState, error) {
	return model.ResourceFavoriteState{ResourceID: resourceID, Favorited: false}, nil
}

func (s *fakeInteractionStore) ListFavoriteResources(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListResourcesResult, error) {
	return model.ListResourcesResult{Page: 1, PageSize: 20}, nil
}

func (s *fakeInteractionStore) SetMerchantFollow(ctx context.Context, input model.MerchantFollowInput) (model.MerchantFollowState, error) {
	s.merchantInput = input
	return model.MerchantFollowState{MerchantID: input.MerchantID, Followed: input.Followed}, nil
}

func (s *fakeInteractionStore) GetMerchantFollowState(ctx context.Context, userID string, merchantID string) (model.MerchantFollowState, error) {
	return model.MerchantFollowState{MerchantID: merchantID, Followed: false}, nil
}

func (s *fakeInteractionStore) ListFollowedMerchants(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListFollowedMerchantsResult, error) {
	return model.ListFollowedMerchantsResult{Page: 1, PageSize: 20}, nil
}

func (s *fakeInteractionStore) CreateSavedSearch(ctx context.Context, input model.SavedSearchInput) (model.SavedSearchResult, error) {
	s.savedSearchInput = input
	return s.savedSearchResult, nil
}

func (s *fakeInteractionStore) ListSavedSearches(ctx context.Context, userID string, filter model.ListInteractionFilter) (model.ListSavedSearchesResult, error) {
	return model.ListSavedSearchesResult{Page: 1, PageSize: 20}, nil
}

func (s *fakeInteractionStore) DeleteSavedSearch(ctx context.Context, userID string, savedSearchID string) error {
	return nil
}
