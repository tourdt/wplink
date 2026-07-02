package maplogic

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestPublicMapLogicListsPublishedScenes(t *testing.T) {
	store := &fakePublicMapStore{
		scenes: []model.MapScene{{Code: "zhili_lijilu_middle", Name: "利济路中段", Status: model.MapSceneStatusPublished}},
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListScenes(context.Background(), ListScenesReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("ListScenes() error = %v", err)
	}

	if store.sceneFilter.CityCode != "zhili" || store.sceneFilter.Status != model.MapSceneStatusPublished {
		t.Fatalf("filter = %#v, want zhili published", store.sceneFilter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Code != "zhili_lijilu_middle" {
		t.Fatalf("items = %#v, want scene", resp.Items)
	}
}

func TestPublicMapLogicListsNormalObjectsWithParsedFilters(t *testing.T) {
	store := &fakePublicMapStore{
		objects: []model.MapObject{{ID: "object-1", SceneCode: "scene-1", Code: "A001", Name: "A001 小鹿童装", Status: model.MapObjectStatusNormal}},
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListObjects(context.Background(), " scene-1 ", ListObjectsReq{
		Types:      "booth,packing_station",
		Categories: " girl , middle_child ",
		Keyword:    "  女童 ",
	})
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}

	if store.objectFilter.SceneCode != "scene-1" || store.objectFilter.Status != model.MapObjectStatusNormal {
		t.Fatalf("filter = %#v, want scene normal", store.objectFilter)
	}
	if len(store.objectFilter.Types) != 2 || store.objectFilter.Types[1] != "packing_station" {
		t.Fatalf("types = %#v, want parsed types", store.objectFilter.Types)
	}
	if store.objectFilter.Keyword != "女童" {
		t.Fatalf("keyword = %q, want trimmed keyword", store.objectFilter.Keyword)
	}
	if resp.SceneCode != "scene-1" || len(resp.Items) != 1 {
		t.Fatalf("resp = %#v, want scene objects", resp)
	}
}

func TestPublicMapLogicSearchUsesDefaultLimit(t *testing.T) {
	store := &fakePublicMapStore{
		objects: []model.MapObject{{ID: "object-1", SceneCode: "scene-1", Code: "A001", Name: "A001 小鹿童装"}},
	}
	logic := NewPublicLogic(store)

	_, err := logic.SearchObjects(context.Background(), SearchObjectsReq{
		Keyword:        " 女童 ",
		Categories:     " girl ",
		ServiceTags:    " spot ",
		PoiServiceTags: " packing ",
	})
	if err != nil {
		t.Fatalf("SearchObjects() error = %v", err)
	}

	if store.objectFilter.Keyword != "女童" || store.objectFilter.Limit != 10 {
		t.Fatalf("filter = %#v, want trimmed keyword and default limit", store.objectFilter)
	}
	if len(store.objectFilter.Categories) != 1 || store.objectFilter.Categories[0] != "girl" {
		t.Fatalf("categories = %#v, want parsed category filter", store.objectFilter.Categories)
	}
	if len(store.objectFilter.ServiceTags) != 1 || store.objectFilter.ServiceTags[0] != "spot" {
		t.Fatalf("serviceTags = %#v, want parsed service tag filter", store.objectFilter.ServiceTags)
	}
	if len(store.objectFilter.PoiServiceTags) != 1 || store.objectFilter.PoiServiceTags[0] != "packing" {
		t.Fatalf("poiServiceTags = %#v, want parsed poi service filter", store.objectFilter.PoiServiceTags)
	}
}

func TestPublicMapLogicListsVisibleNormalCategories(t *testing.T) {
	store := &fakePublicMapStore{
		categories: []model.MapCategory{
			{Code: "girl", Name: "女童", Type: "booth_category", IsVisible: true, Status: model.MapCategoryStatusNormal},
			{Code: "hidden", Name: "隐藏分类", Type: "booth_category", IsVisible: true, Status: model.MapCategoryStatusHidden},
			{Code: "closed", Name: "停用分类", Type: "booth_category", IsVisible: true, Status: model.MapCategoryStatusClosed},
			{Code: "invisible", Name: "不展示分类", Type: "booth_category", IsVisible: false, Status: model.MapCategoryStatusNormal},
		},
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListCategories(context.Background(), ListCategoriesReq{Type: " booth_category "})
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}

	if store.categoryType != "booth_category" {
		t.Fatalf("categoryType = %q, want booth_category", store.categoryType)
	}
	if len(resp.Items) != 1 || resp.Items[0].Code != "girl" || resp.Items[0].Name != "女童" {
		t.Fatalf("items = %#v, want only visible normal category", resp.Items)
	}
}

func TestPublicMapLogicListsNearbyPois(t *testing.T) {
	store := &fakePublicMapStore{
		object: model.MapObject{ID: "booth-1", SceneCode: "scene-1", CenterX: 100, CenterY: 100},
		nearby: []model.MapObject{
			{ID: "poi-far", Name: "远处物流", Type: "logistics_point", CenterX: 300, CenterY: 100},
			{ID: "poi-near", Name: "近处打包", Type: "packing_station", CenterX: 130, CenterY: 100},
		},
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListNearbyPois(context.Background(), " booth-1 ", ListNearbyPoisReq{Types: "packing_station,logistics_point", Limit: 1})
	if err != nil {
		t.Fatalf("ListNearbyPois() error = %v", err)
	}

	if store.objectID != "booth-1" {
		t.Fatalf("objectID = %q, want trimmed booth-1", store.objectID)
	}
	if store.nearbySceneCode != "scene-1" {
		t.Fatalf("nearby scene = %q, want scene-1", store.nearbySceneCode)
	}
	if len(resp.Items) != 1 || resp.Items[0].Id != "poi-near" || resp.Items[0].DistanceText != "30m" {
		t.Fatalf("nearby = %#v, want nearest poi", resp.Items)
	}
}

type fakePublicMapStore struct {
	sceneFilter     model.ListMapScenesFilter
	objectFilter    model.ListMapObjectsFilter
	objectID        string
	categoryType    string
	nearbySceneCode string
	nearbyTypes     []string
	scenes          []model.MapScene
	scene           model.MapScene
	objects         []model.MapObject
	object          model.MapObject
	nearby          []model.MapObject
	categories      []model.MapCategory
}

func (s *fakePublicMapStore) ListPublishedScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error) {
	s.sceneFilter = filter
	return append([]model.MapScene(nil), s.scenes...), nil
}

func (s *fakePublicMapStore) GetPublishedScene(ctx context.Context, sceneCode string) (model.MapScene, error) {
	return s.scene, nil
}

func (s *fakePublicMapStore) ListPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	s.objectFilter = filter
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakePublicMapStore) SearchPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	s.objectFilter = filter
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakePublicMapStore) GetPublishedObject(ctx context.Context, objectID string) (model.MapObject, error) {
	s.objectID = objectID
	return s.object, nil
}

func (s *fakePublicMapStore) ListObjectsBySceneAndTypes(ctx context.Context, sceneCode string, types []string) ([]model.MapObject, error) {
	s.nearbySceneCode = sceneCode
	s.nearbyTypes = append([]string(nil), types...)
	return append([]model.MapObject(nil), s.nearby...), nil
}

func (s *fakePublicMapStore) ListCategories(ctx context.Context, categoryType string) ([]model.MapCategory, error) {
	s.categoryType = categoryType
	return append([]model.MapCategory(nil), s.categories...), nil
}
