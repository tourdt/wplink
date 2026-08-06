package maplogic

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestAdminMapLogicRejectsWriteWithoutTrustedOperator(t *testing.T) {
	logic := NewAdminLogic(&fakeAdminMapStore{})

	_, err := logic.SaveCategory(context.Background(), SaveCategoryReq{
		Code: "booth", Name: "档口", Type: "booth_category",
	}, "")
	if err == nil || errx.CodeOf(err) != errx.CodeUnauthorized {
		t.Fatalf("SaveCategory() error = %v, want unauthorized", err)
	}
}

func TestAdminMapLogicAuditsTrustedOperatorForEveryWrite(t *testing.T) {
	var logBuffer bytes.Buffer
	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(&logBuffer))
	t.Cleanup(func() {
		if currentWriter := logx.Reset(); currentWriter != nil {
			_ = currentWriter.Close()
		}
		if previousWriter != nil {
			logx.SetWriter(previousWriter)
		}
	})

	store := &fakeAdminMapStore{
		scene:  model.MapScene{Code: "scene-1", BackgroundURL: "https://img.example/map.png", Width: 1000, Height: 800},
		object: model.MapObject{ID: "object-1", Code: "A001", CategoryCodes: []string{}, ServiceTags: []string{}, PlatformTags: []string{}, PoiServiceTags: []string{}},
		objects: []model.MapObject{{
			ID: "object-1", Code: "A001", Name: "档口 A001", Type: "booth", Layer: "booth",
			GeometryType:  model.MapGeometryTypeRect,
			Geometry:      model.JSONMap{"x": float64(10), "y": float64(20), "width": float64(80), "height": float64(50)},
			CategoryCodes: []string{"girl"}, ServiceTags: []string{"spot"}, Phone: "13800000000", Status: model.MapObjectStatusNormal,
		}},
	}
	logic := NewAdminLogic(store)
	operatorID := "operator-map-audit"

	if _, err := logic.SaveScene(context.Background(), SaveSceneReq{
		Code: "scene-1", Name: "利济路", Type: "street_segment", BackgroundUrl: "https://img.example/map.png", Width: 1000, Height: 800,
	}, operatorID); err != nil {
		t.Fatalf("SaveScene() error = %v", err)
	}
	if _, err := logic.PublishScene(context.Background(), "scene-1", operatorID); err != nil {
		t.Fatalf("PublishScene() error = %v", err)
	}
	objectReq := SaveObjectReq{
		Code: "A001", Name: "档口 A001", Type: "booth", Layer: "booth", GeometryType: model.MapGeometryTypeRect,
		Geometry: map[string]interface{}{"x": float64(10), "y": float64(20), "width": float64(80), "height": float64(50)},
	}
	if _, err := logic.SaveObject(context.Background(), "scene-1", objectReq, operatorID); err != nil {
		t.Fatalf("SaveObject(create) error = %v", err)
	}
	objectReq.Id = "object-1"
	if _, err := logic.SaveObject(context.Background(), "", objectReq, operatorID); err != nil {
		t.Fatalf("SaveObject(update) error = %v", err)
	}
	if _, err := logic.UpdateObjectStatus(context.Background(), "object-1", UpdateObjectStatusReq{Status: model.MapObjectStatusHidden}, operatorID); err != nil {
		t.Fatalf("UpdateObjectStatus() error = %v", err)
	}
	if _, err := logic.BatchGenerateObjects(context.Background(), "scene-1", BatchGenerateObjectsReq{
		StartCode: "B001", Count: 1, Direction: "horizontal", StartX: "100", StartY: "100", Width: "80", Height: "50", Gap: "5", Type: "booth", Layer: "booth",
	}, operatorID); err != nil {
		t.Fatalf("BatchGenerateObjects() error = %v", err)
	}
	if _, err := logic.SaveCategory(context.Background(), SaveCategoryReq{Code: "girl", Name: "女童", Type: "booth_category"}, operatorID); err != nil {
		t.Fatalf("SaveCategory() error = %v", err)
	}
	if _, err := logic.SaveCategory(context.Background(), SaveCategoryReq{Code: "invalid"}, operatorID); err == nil {
		t.Fatal("SaveCategory(invalid) error = nil, want audit failure")
	}

	logs := logBuffer.String()
	for _, marker := range []string{
		`"operatorId":"operator-map-audit"`, `"operation":"save_scene"`, `"operation":"publish_scene"`,
		`"operation":"save_object"`, `"operation":"update_object_status"`, `"operation":"batch_generate_objects"`,
		`"operation":"save_category"`, `地图后台写操作成功`, `地图后台写操作失败`,
	} {
		if !strings.Contains(logs, marker) {
			t.Fatalf("audit logs = %q, want marker %q", logs, marker)
		}
	}
	if strings.Count(logs, `"operation":"save_object"`) < 2 {
		t.Fatalf("audit logs = %q, want separate create/update object attribution", logs)
	}
}

func TestAdminMapLogicRejectsSceneWithoutBackground(t *testing.T) {
	logic := NewAdminLogic(&fakeAdminMapStore{})

	_, err := logic.SaveScene(context.Background(), SaveSceneReq{
		Code:   "zhili_lijilu_middle",
		Name:   "利济路中段",
		Type:   "street_segment",
		Width:  3000,
		Height: 1800,
	}, "operator-test")

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
	}, "operator-test")

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
		scene: model.MapScene{Code: "zhili_lijilu_middle", BackgroundURL: "https://img.example.com/maps/lijilu.png", Width: 3000, Height: 1800, Status: model.MapSceneStatusDraft},
	}
	logic := NewAdminLogic(store)

	_, err := logic.PublishScene(context.Background(), "zhili_lijilu_middle", "operator-test")
	if err == nil {
		t.Fatal("PublishScene() error = nil, want validation error")
	}
}

func TestAdminMapLogicRejectsPublishWithIncompleteVisibleObjects(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "zhili_lijilu_middle", BackgroundURL: "https://img.example.com/maps/lijilu.png", Width: 3000, Height: 1800, Status: model.MapSceneStatusDraft},
		objects: []model.MapObject{{
			ID:           "object-1",
			Code:         "A001",
			Name:         "A001 小鹿童装",
			Layer:        "booth",
			Status:       model.MapObjectStatusNormal,
			GeometryType: model.MapGeometryTypeRect,
			Geometry:     model.JSONMap{"x": float64(100), "y": float64(120), "width": float64(80), "height": float64(50)},
		}},
	}
	logic := NewAdminLogic(store)

	_, err := logic.PublishScene(context.Background(), "zhili_lijilu_middle", "operator-test")
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("PublishScene() error = %v, want validation error", err)
	}
}

func TestAdminMapLogicPublishesCompleteVisibleObjects(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "zhili_lijilu_middle", BackgroundURL: "https://img.example.com/maps/lijilu.png", Width: 3000, Height: 1800, Status: model.MapSceneStatusDraft},
		objects: []model.MapObject{{
			ID:            "object-1",
			Code:          "A001",
			Name:          "A001 小鹿童装",
			Layer:         "booth",
			Status:        model.MapObjectStatusNormal,
			GeometryType:  model.MapGeometryTypeRect,
			Geometry:      model.JSONMap{"x": float64(100), "y": float64(120), "width": float64(80), "height": float64(50)},
			CategoryCodes: []string{"girl"},
			ServiceTags:   []string{"spot"},
			Phone:         "18800000001",
		}},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.PublishScene(context.Background(), "zhili_lijilu_middle", "operator-test")
	if err != nil {
		t.Fatalf("PublishScene() error = %v", err)
	}
	if resp.Item.Status != model.MapSceneStatusPublished {
		t.Fatalf("resp = %#v, want published scene", resp)
	}
}

func TestAdminMapLogicPublishesBoothWithWechatFallback(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "zhili_lijilu_middle", BackgroundURL: "https://img.example.com/maps/lijilu.png", Width: 3000, Height: 1800, Status: model.MapSceneStatusDraft},
		objects: []model.MapObject{{
			ID:            "object-1",
			Code:          "A001",
			Name:          "A001 小鹿童装",
			Layer:         "booth",
			Status:        model.MapObjectStatusNormal,
			GeometryType:  model.MapGeometryTypeRect,
			Geometry:      model.JSONMap{"x": float64(100), "y": float64(120), "width": float64(80), "height": float64(50)},
			CategoryCodes: []string{"girl"},
			ServiceTags:   []string{"spot"},
			Wechat:        "xiaolu001",
		}},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.PublishScene(context.Background(), "zhili_lijilu_middle", "operator-test")
	if err != nil {
		t.Fatalf("PublishScene() error = %v", err)
	}
	if resp.Item.Status != model.MapSceneStatusPublished {
		t.Fatalf("resp = %#v, want published scene", resp)
	}
}

func TestAdminMapLogicPublishesPoiWithoutPhone(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "zhili_lijilu_middle", BackgroundURL: "https://img.example.com/maps/lijilu.png", Width: 3000, Height: 1800, Status: model.MapSceneStatusDraft},
		objects: []model.MapObject{{
			ID:             "object-1",
			Code:           "P001",
			Name:           "P001 打包站",
			Type:           "packing_station",
			Layer:          "poi",
			Status:         model.MapObjectStatusNormal,
			GeometryType:   model.MapGeometryTypePoint,
			Geometry:       model.JSONMap{"x": float64(100), "y": float64(120)},
			PoiServiceTags: []string{"packing"},
		}},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.PublishScene(context.Background(), "zhili_lijilu_middle", "operator-test")
	if err != nil {
		t.Fatalf("PublishScene() error = %v", err)
	}
	if resp.Item.Status != model.MapSceneStatusPublished {
		t.Fatalf("resp = %#v, want published scene", resp)
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
			}, "operator-test")

			if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
				t.Fatalf("SaveObject() error = %v, want validation error", err)
			}
			if store.objectInput.Code != "" {
				t.Fatalf("SaveObject saved invalid zoom range: %#v", store.objectInput)
			}
		})
	}
}

func TestAdminMapLogicSavesMerchantBinding(t *testing.T) {
	store := &fakeAdminMapStore{
		object: model.MapObject{ID: "object-1", Code: "A001", Name: "A001 小鹿童装", MerchantID: "1001"},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.SaveObject(context.Background(), " scene-1 ", SaveObjectReq{
		Code:         " A001 ",
		Name:         " A001 小鹿童装 ",
		Type:         "booth",
		Layer:        "booth",
		GeometryType: model.MapGeometryTypeRect,
		Geometry:     map[string]interface{}{"x": float64(100), "y": float64(200), "width": float64(80), "height": float64(50)},
		MerchantID:   " 1001 ",
	}, "operator-test")

	if err != nil {
		t.Fatalf("SaveObject() error = %v", err)
	}

	if store.objectInput.SceneCode != "scene-1" || store.objectInput.MerchantID != "1001" {
		t.Fatalf("object input = %#v, want trimmed merchant binding", store.objectInput)
	}
	if resp.Item.MerchantId != "1001" {
		t.Fatalf("resp = %#v, want merchant id in admin object response", resp)
	}
}

func TestAdminMapLogicListsObjectsWithViewport(t *testing.T) {
	store := &fakeAdminMapStore{}
	logic := NewAdminLogic(store)

	_, err := logic.ListObjects(context.Background(), " scene-1 ", ListAdminObjectsReq{
		Types: "booth",
		MinX:  "10",
		MinY:  "20",
		MaxX:  "510",
		MaxY:  "420",
	})
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}

	if store.objectFilter.SceneCode != "scene-1" || len(store.objectFilter.Types) != 1 {
		t.Fatalf("filter = %#v, want scene and type", store.objectFilter)
	}
	if store.objectFilter.Viewport == nil {
		t.Fatalf("viewport = nil, want parsed viewport")
	}
	if store.objectFilter.Viewport.MinX != 10 || store.objectFilter.Viewport.MaxY != 420 {
		t.Fatalf("viewport = %#v, want parsed bounds", store.objectFilter.Viewport)
	}
}

func TestAdminMapLogicBatchGenerateHorizontalBooths(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "scene-1", Width: 3000, Height: 1800},
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
	}, "operator-test")

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

func TestAdminMapLogicBatchGenerateSkipsObjectsOutsideSceneBounds(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "scene-1", Width: 300, Height: 300},
		objects: []model.MapObject{
			{ID: "A001", Code: "A001"},
			{ID: "A002", Code: "A002"},
		},
	}
	logic := NewAdminLogic(store)

	resp, err := logic.BatchGenerateObjects(context.Background(), "scene-1", BatchGenerateObjectsReq{
		StartCode: "A001",
		Count:     3,
		Direction: "horizontal",
		StartX:    "100",
		StartY:    "200",
		Width:     "80",
		Height:    "50",
		Gap:       "5",
		Type:      "booth",
		Layer:     "booth",
	}, "operator-test")

	if err != nil {
		t.Fatalf("BatchGenerateObjects() error = %v", err)
	}

	if len(store.batchInputs) != 2 {
		t.Fatalf("batch inputs = %#v, want only 2 in-range objects", store.batchInputs)
	}
	if store.batchInputs[0].Code != "A001" || store.batchInputs[1].Code != "A002" {
		t.Fatalf("codes = %#v, want A001-A002", store.batchInputs)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items = %#v, want 2 generated objects", resp.Items)
	}
}

func TestAdminMapLogicBatchGenerateRejectsWhenAllObjectsOutsideSceneBounds(t *testing.T) {
	store := &fakeAdminMapStore{
		scene: model.MapScene{Code: "scene-1", Width: 300, Height: 300},
	}
	logic := NewAdminLogic(store)

	_, err := logic.BatchGenerateObjects(context.Background(), "scene-1", BatchGenerateObjectsReq{
		StartCode: "A001",
		Count:     2,
		Direction: "horizontal",
		StartX:    "320",
		StartY:    "200",
		Width:     "80",
		Height:    "50",
		Gap:       "5",
		Type:      "booth",
		Layer:     "booth",
	}, "operator-test")

	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("BatchGenerateObjects() error = %v, want validation error", err)
	}
	if len(store.batchInputs) != 0 {
		t.Fatalf("batch inputs = %#v, want no saved objects", store.batchInputs)
	}
}

func TestAdminMapLogicBatchGenerateReturnsActionableNumberValidationMessage(t *testing.T) {
	cases := []struct {
		name    string
		startX  string
		wantMsg string
	}{
		{name: "missing start x", startX: " ", wantMsg: "请填写起始 X 坐标"},
		{name: "non finite start x", startX: "NaN", wantMsg: "起始 X 坐标必须是有效数字"},
		{name: "invalid start x", startX: "abc", wantMsg: "起始 X 坐标格式不正确"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeAdminMapStore{
				scene: model.MapScene{Code: "scene-1", Width: 300, Height: 300},
			}
			logic := NewAdminLogic(store)

			_, err := logic.BatchGenerateObjects(context.Background(), "scene-1", BatchGenerateObjectsReq{
				StartCode: "A001",
				Count:     1,
				Direction: "horizontal",
				StartX:    tc.startX,
				StartY:    "200",
				Width:     "80",
				Height:    "50",
				Gap:       "5",
				Type:      "booth",
				Layer:     "booth",
			}, "operator-test")

			if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed || err.Error() != tc.wantMsg {
				t.Fatalf("BatchGenerateObjects() error = %v, code=%s, want validation %q", err, errx.CodeOf(err), tc.wantMsg)
			}
			if len(store.batchInputs) != 0 {
				t.Fatalf("batch inputs = %#v, want no saved objects after validation error", store.batchInputs)
			}
		})
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
	objectFilter   model.ListMapObjectsFilter
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
	s.objectFilter = filter
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
