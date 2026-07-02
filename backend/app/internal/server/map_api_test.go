package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestMapAPIRouterListsPublishedScenes(t *testing.T) {
	store := &fakeMapAPIStore{
		fakeCityAPIStore: fakeCityAPIStore{},
		scenes: []model.MapScene{
			{Code: "zhili_lijilu_middle", Name: "利济路中段", Type: "street_segment", Status: model.MapSceneStatusPublished},
		},
	}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/scenes?cityCode=zhili", nil)
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	items := data["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items = %#v, want one scene", items)
	}
	if store.sceneFilter.CityCode != "zhili" || store.sceneFilter.Status != model.MapSceneStatusPublished {
		t.Fatalf("scene filter = %#v, want zhili published", store.sceneFilter)
	}
}

func TestMapAPIRouterSavesAdminScene(t *testing.T) {
	store := &fakeMapAPIStore{
		fakeCityAPIStore: fakeCityAPIStore{},
		savedScene:       model.MapScene{Code: "zhili_lijilu_middle", Name: "利济路中段", Type: "street_segment", Status: model.MapSceneStatusDraft},
	}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/map/scenes", strings.NewReader(`{
		"code":"zhili_lijilu_middle",
		"name":"利济路中段",
		"type":"street_segment",
		"backgroundUrl":"https://img.example.com/maps/lijilu.png",
		"width":3000,
		"height":1800
	}`))
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	item := data["item"].(map[string]interface{})
	if item["code"] != "zhili_lijilu_middle" {
		t.Fatalf("item = %#v, want saved scene", item)
	}
	if store.savedSceneInput.Code != "zhili_lijilu_middle" || store.savedSceneInput.Status != model.MapSceneStatusDraft {
		t.Fatalf("saved input = %#v, want draft scene", store.savedSceneInput)
	}
}

func TestMapAPIRouterGetsAdminDraftScene(t *testing.T) {
	store := &fakeMapAPIStore{
		fakeCityAPIStore: fakeCityAPIStore{},
		savedScene:       model.MapScene{Code: "zhili_lijilu_middle", Name: "利济路中段", Type: "street_segment", Status: model.MapSceneStatusDraft},
	}
	router := NewAPIRouter(store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/map/scenes/zhili_lijilu_middle", nil)
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	item := data["item"].(map[string]interface{})
	if item["status"] != model.MapSceneStatusDraft {
		t.Fatalf("item = %#v, want draft scene", item)
	}
	if store.adminSceneCode != "zhili_lijilu_middle" {
		t.Fatalf("admin scene code = %q, want zhili_lijilu_middle", store.adminSceneCode)
	}
}

type fakeMapAPIStore struct {
	fakeCityAPIStore
	sceneFilter      model.ListMapScenesFilter
	objectFilter     model.ListMapObjectsFilter
	adminSceneCode   string
	savedSceneInput  model.MapSceneInput
	savedObjectInput model.MapObjectInput
	scenes           []model.MapScene
	savedScene       model.MapScene
	objects          []model.MapObject
	object           model.MapObject
	categories       []model.MapCategory
}

func (s *fakeMapAPIStore) ListPublishedScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error) {
	s.sceneFilter = filter
	return append([]model.MapScene(nil), s.scenes...), nil
}

func (s *fakeMapAPIStore) GetPublishedScene(ctx context.Context, sceneCode string) (model.MapScene, error) {
	return s.savedScene, nil
}

func (s *fakeMapAPIStore) ListPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	s.objectFilter = filter
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeMapAPIStore) SearchPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	s.objectFilter = filter
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeMapAPIStore) GetPublishedObject(ctx context.Context, objectID string) (model.MapObject, error) {
	return s.object, nil
}

func (s *fakeMapAPIStore) ListObjectsBySceneAndTypes(ctx context.Context, sceneCode string, types []string) ([]model.MapObject, error) {
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeMapAPIStore) ListAdminScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error) {
	s.sceneFilter = filter
	return append([]model.MapScene(nil), s.scenes...), nil
}

func (s *fakeMapAPIStore) GetAdminScene(ctx context.Context, sceneCode string) (model.MapScene, error) {
	s.adminSceneCode = sceneCode
	return s.savedScene, nil
}

func (s *fakeMapAPIStore) SaveScene(ctx context.Context, input model.MapSceneInput) (model.MapScene, error) {
	s.savedSceneInput = input
	return s.savedScene, nil
}

func (s *fakeMapAPIStore) PublishScene(ctx context.Context, sceneCode string) (model.MapScene, error) {
	return model.MapScene{Code: sceneCode, Status: model.MapSceneStatusPublished}, nil
}

func (s *fakeMapAPIStore) ListAdminObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	s.objectFilter = filter
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeMapAPIStore) SaveObject(ctx context.Context, input model.MapObjectInput) (model.MapObject, error) {
	s.savedObjectInput = input
	return s.object, nil
}

func (s *fakeMapAPIStore) UpdateObjectStatus(ctx context.Context, objectID string, status string) (model.MapObject, error) {
	return s.object, nil
}

func (s *fakeMapAPIStore) BatchCreateObjects(ctx context.Context, inputs []model.MapObjectInput) ([]model.MapObject, error) {
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeMapAPIStore) ListCategories(ctx context.Context, categoryType string) ([]model.MapCategory, error) {
	return append([]model.MapCategory(nil), s.categories...), nil
}

func (s *fakeMapAPIStore) SaveCategory(ctx context.Context, input model.MapCategoryInput) (model.MapCategory, error) {
	return model.MapCategory{Code: input.Code, Name: input.Name, Type: input.Type, Status: input.Status}, nil
}
