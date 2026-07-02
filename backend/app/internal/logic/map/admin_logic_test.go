package maplogic

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestAdminMapLogicRejectsSceneWithoutBackground(t *testing.T) {
	logic := NewAdminLogic(&fakeAdminMapStore{})

	_, err := logic.SaveScene(context.Background(), SaveSceneReq{
		Code:   "zhili_lijilu_middle",
		Name:   "利济路中段",
		Type:   "street_segment",
		Width:  3000,
		Height: 1800,
	})
	if err == nil {
		t.Fatal("SaveScene() error = nil, want validation error")
	}
}

func TestAdminMapLogicSavesSceneAsDraftByDefault(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "zhili_lijilu_middle", Name: "利济路中段", Status: model.MapSceneStatusDraft},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.SaveScene(context.Background(), SaveSceneReq{
		Code:          " zhili_lijilu_middle ",
		Name:          " 利济路中段 ",
		Type:          " street_segment ",
		BackgroundUrl: "https://img.example.com/maps/lijilu.png",
		Width:         3000,
		Height:        1800,
	})
	if err != nil {
		t.Fatalf("SaveScene() error = %v", err)
	}

	if store.sceneInput.Code != "zhili_lijilu_middle" || store.sceneInput.Status != model.MapSceneStatusDraft {
		t.Fatalf("scene input = %#v, want trimmed draft scene", store.sceneInput)
	}
	if resp.Item.Code != "zhili_lijilu_middle" {
		t.Fatalf("resp = %#v, want saved scene", resp)
	}
}

func TestAdminMapLogicRejectsPublishWithoutObjects(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "zhili_lijilu_middle", Status: model.MapSceneStatusDraft},
	}
	logic := NewAdminLogic(store)

	_, err := logic.PublishScene(context.Background(), "zhili_lijilu_middle")
	if err == nil {
		t.Fatal("PublishScene() error = nil, want validation error")
	}
}

func TestAdminMapLogicRejectsInvalidObjectZoomRange(t *testing.T) {
	cases := []struct {
		name    string
		minZoom int64
		maxZoom int64
	}{
		{name: "min greater than max", minZoom: 5, maxZoom: 4},
		{name: "min out of range", minZoom: 6, maxZoom: 6},
		{name: "max out of range", minZoom: 1, maxZoom: 6},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeAdminMapStore{}
			logic := NewAdminLogic(store)

			_, err := logic.SaveObject(context.Background(), "scene-1", SaveObjectReq{
				Code:         "A001",
				Name:         "A001 小鹿童装",
				Type:         "booth",
				Layer:        "booth",
				GeometryType: model.MapGeometryTypeRect,
				Geometry:     map[string]interface{}{"x": float64(100), "y": float64(200), "width": float64(80), "height": float64(50)},
				MinZoom:      tc.minZoom,
				MaxZoom:      tc.maxZoom,
			})
			if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
				t.Fatalf("SaveObject() error = %v, want validation error", err)
			}
			if store.objectInput.Code != "" {
				t.Fatalf("SaveObject saved invalid zoom range: %#v", store.objectInput)
			}
		})
	}
}

func TestAdminMapLogicBatchGenerateHorizontalBooths(t *testing.T) {
	store := &fakeAdminMapStore{
		objects: []model.MapObject{
			{ID: "A001", Code: "A001"},
			{ID: "A002", Code: "A002"},
			{ID: "A003", Code: "A003"},
		},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.BatchGenerateObjects(context.Background(), " scene-1 ", BatchGenerateObjectsReq{
		StartCode:     "A001",
		Count:         3,
		Direction:     "horizontal",
		StartX:        "100",
		StartY:        "200",
		Width:         "80",
		Height:        "50",
		Gap:           "5",
		Type:          "booth",
		Layer:         "booth",
		CategoryCodes: []string{"girl"},
		ServiceTags:   []string{"spot"},
	})
	if err != nil {
		t.Fatalf("BatchGenerateObjects() error = %v", err)
	}

	if len(store.batchInputs) != 3 {
		t.Fatalf("batch inputs = %#v, want 3", store.batchInputs)
	}
	if store.batchInputs[0].Code != "A001" || store.batchInputs[1].Code != "A002" || store.batchInputs[2].Code != "A003" {
		t.Fatalf("codes = %#v, want A001-A003", store.batchInputs)
	}
	if store.batchInputs[0].Geometry["x"] != float64(100) || store.batchInputs[1].Geometry["x"] != float64(185) || store.batchInputs[2].Geometry["x"] != float64(270) {
		t.Fatalf("geometry = %#v, want horizontal positions", store.batchInputs)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("items = %#v, want 3 generated objects", resp.Items)
	}
}

func TestAdminMapLogicListsCategoriesWithTypeAndStatus(t *testing.T) {
	store := &fakeAdminMapStore{
		categories: []model.MapCategory{{Code: "hidden", Name: "隐藏分类", Type: "booth_category", IsVisible: true, Status: model.MapCategoryStatusHidden}},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.ListCategories(context.Background(), ListCategoriesReq{Type: " booth_category ", Status: " hidden "})
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}

	if store.categoryFilter.Type != "booth_category" || store.categoryFilter.Status != model.MapCategoryStatusHidden {
		t.Fatalf("category filter = %#v, want type and hidden status", store.categoryFilter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != model.MapCategoryStatusHidden {
		t.Fatalf("items = %#v, want hidden category", resp.Items)
	}
}

type fakeAdminMapStore struct {
	sceneInput     model.MapSceneInput
	objectInput    model.MapObjectInput
	objectID       string
	objectStatus   string
	batchInputs    []model.MapObjectInput
	categoryFilter model.ListMapCategoriesFilter
	categoryInput  model.MapCategoryInput
	scene          model.MapScene
	scenes         []model.MapScene
	object         model.MapObject
	objects        []model.MapObject
	categories     []model.MapCategory
}

func (s *fakeAdminMapStore) ListAdminScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error) {
	return append([]model.MapScene(nil), s.scenes...), nil
}

func (s *fakeAdminMapStore) GetAdminScene(ctx context.Context, sceneCode string) (model.MapScene, error) {
	return s.scene, nil
}

func (s *fakeAdminMapStore) SaveScene(ctx context.Context, input model.MapSceneInput) (model.MapScene, error) {
	s.sceneInput = input
	return s.scene, nil
}

func (s *fakeAdminMapStore) PublishScene(ctx context.Context, sceneCode string) (model.MapScene, error) {
	return model.MapScene{Code: sceneCode, Status: model.MapSceneStatusPublished}, nil
}

func (s *fakeAdminMapStore) ListPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeAdminMapStore) ListAdminObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeAdminMapStore) ListPublishedScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error) {
	return nil, nil
}

func (s *fakeAdminMapStore) GetPublishedScene(ctx context.Context, sceneCode string) (model.MapScene, error) {
	return model.MapScene{}, nil
}

func (s *fakeAdminMapStore) SearchPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error) {
	return nil, nil
}

func (s *fakeAdminMapStore) GetPublishedObject(ctx context.Context, objectID string) (model.MapObject, error) {
	return model.MapObject{}, nil
}

func (s *fakeAdminMapStore) ListObjectsBySceneAndTypes(ctx context.Context, sceneCode string, types []string) ([]model.MapObject, error) {
	return nil, nil
}

func (s *fakeAdminMapStore) SaveObject(ctx context.Context, input model.MapObjectInput) (model.MapObject, error) {
	s.objectInput = input
	return s.object, nil
}

func (s *fakeAdminMapStore) UpdateObjectStatus(ctx context.Context, objectID string, status string) (model.MapObject, error) {
	s.objectID = objectID
	s.objectStatus = status
	return s.object, nil
}

func (s *fakeAdminMapStore) BatchCreateObjects(ctx context.Context, inputs []model.MapObjectInput) ([]model.MapObject, error) {
	s.batchInputs = append([]model.MapObjectInput(nil), inputs...)
	return append([]model.MapObject(nil), s.objects...), nil
}

func (s *fakeAdminMapStore) ListCategories(ctx context.Context, filter model.ListMapCategoriesFilter) ([]model.MapCategory, error) {
	s.categoryFilter = filter
	return append([]model.MapCategory(nil), s.categories...), nil
}

func (s *fakeAdminMapStore) SaveCategory(ctx context.Context, input model.MapCategoryInput) (model.MapCategory, error) {
	s.categoryInput = input
	return model.MapCategory{Code: input.Code, Name: input.Name, Type: input.Type, Status: input.Status}, nil
}
