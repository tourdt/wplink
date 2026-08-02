package maplogic

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
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

func TestPublicMapLogicListsClaimedAndPrelistedMerchantPlacesWithoutContact(t *testing.T) {
	store := &fakePublicMapStore{
		merchantPlaces: []model.MerchantPlace{
			{
				Object: model.MapObject{
					ID: "object-1", MerchantID: "merchant-1", MerchantName: "小鹿童装",
					MerchantType: "factory", MerchantLogoURL: "https://img.example.com/logo.png",
					Code: "A001", Name: "后台档口", Address: "利济路 A001",
					Phone: "18800000001", Wechat: "xiaolu", Lat: "30.9000000", Lng: "120.2000000",
				},
				CityCode: "zhili", SceneName: "利济路市场", FloorNo: "1F",
			},
			{
				Object:   model.MapObject{ID: "object-2", Code: "B008", Name: "B008 童装档口", Address: "利济路 B008"},
				CityCode: "zhili", SceneName: "利济路市场", FloorNo: "2F",
			},
		},
		merchantPlaceTotal: 2,
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListMerchantPlaces(context.Background(), ListMerchantPlacesReq{
		CityCode: " zhili ", Keyword: " 童装 ", Claimed: "all", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("ListMerchantPlaces() error = %v", err)
	}

	if store.merchantPlaceFilter.CityCode != "zhili" || store.merchantPlaceFilter.Keyword != "童装" {
		t.Fatalf("filter = %#v, want trimmed directory filters", store.merchantPlaceFilter)
	}
	if len(resp.Items) != 2 || resp.Total != 2 || resp.Page != 1 || resp.PageSize != 20 {
		t.Fatalf("resp = %#v, want two paged places", resp)
	}
	claimed := resp.Items[0]
	if claimed.SourceType != "merchant_claimed" || !claimed.Claimed || claimed.Name != "小鹿童装" || claimed.MerchantId != "merchant-1" {
		t.Fatalf("claimed item = %#v, want merchant identity", claimed)
	}
	claimedJSON, err := json.Marshal(claimed)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(claimedJSON), "phone") || strings.Contains(string(claimedJSON), "wechat") {
		t.Fatalf("claimed json = %s, want contact fields absent", claimedJSON)
	}
	prelisted := resp.Items[1]
	if prelisted.SourceType != "platform_prelisted" || prelisted.Claimed || prelisted.Name != "B008 童装档口" {
		t.Fatalf("prelisted item = %#v, want platform identity", prelisted)
	}
}

func TestPublicMapLogicGetsMerchantLocationContext(t *testing.T) {
	store := &fakePublicMapStore{
		merchantPlace: model.MerchantPlace{Object: model.MapObject{
			ID: "object-1", MerchantID: "merchant-1", MerchantName: "小熊星球童装",
			Lat: "30.8700000", Lng: "120.1200000",
		}},
		nearbyMerchantPlaces: []model.MerchantPlace{{
			Object: model.MapObject{
				ID: "object-2", MerchantID: "merchant-2", MerchantName: "布谷童装",
				Lat: "30.8705000", Lng: "120.1200000",
			},
			DistanceMeters: 56,
		}},
	}

	resp, err := NewPublicLogic(store).GetMerchantLocationContext(context.Background(), " merchant-1 ")
	if err != nil {
		t.Fatal(err)
	}

	if store.merchantID != "merchant-1" {
		t.Fatalf("merchantID = %q, want trimmed merchant-1", store.merchantID)
	}
	if store.nearbyRadiusMeters != 1000 || store.nearbyLimit != 20 {
		t.Fatalf("nearby query radius=%d limit=%d, want 1000 and 20", store.nearbyRadiusMeters, store.nearbyLimit)
	}
	if resp.Current.MerchantId != "merchant-1" || len(resp.Nearby) != 1 || resp.RadiusMeters != 1000 || !resp.NearbyAvailable {
		t.Fatalf("resp = %#v", resp)
	}
	if resp.Nearby[0].MerchantId != "merchant-2" || resp.Nearby[0].DistanceMeters != 56 || resp.Nearby[0].DistanceText != "56m" {
		t.Fatalf("nearby = %#v", resp.Nearby[0])
	}
}

func TestPublicMapLogicRejectsUnavailableMerchantLocationContext(t *testing.T) {
	tests := []struct {
		name       string
		merchantID string
		store      *fakePublicMapStore
		wantCode   string
		wantError  string
	}{
		{
			name:       "empty merchant id",
			merchantID: "  ",
			store:      &fakePublicMapStore{},
			wantCode:   errx.CodeValidationFailed,
			wantError:  "该商家暂时无法查看",
		},
		{
			name:       "merchant has no published place",
			merchantID: "merchant-missing",
			store:      &fakePublicMapStore{merchantPlaceErr: sql.ErrNoRows},
			wantCode:   errx.CodeResourceNotFound,
			wantError:  "该商家暂时无法查看",
		},
		{
			name:       "merchant place coordinates invalid",
			merchantID: "merchant-1",
			store: &fakePublicMapStore{merchantPlace: model.MerchantPlace{Object: model.MapObject{
				ID: "object-1", MerchantID: "merchant-1", MerchantName: "小熊星球童装", Lat: "NaN", Lng: "120.1200000",
			}}},
			wantCode:  errx.CodeValidationFailed,
			wantError: "该商家位置待完善",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPublicLogic(tt.store).GetMerchantLocationContext(context.Background(), tt.merchantID)
			if err == nil || errx.CodeOf(err) != tt.wantCode || err.Error() != tt.wantError {
				t.Fatalf("error = %v code=%s, want %s (%s)", err, errx.CodeOf(err), tt.wantError, tt.wantCode)
			}
		})
	}
}

func TestPublicMapLogicKeepsCurrentMerchantWhenNearbyLookupFails(t *testing.T) {
	store := &fakePublicMapStore{
		merchantPlace: model.MerchantPlace{Object: model.MapObject{
			ID: "object-1", MerchantID: "merchant-1", MerchantName: "小熊星球童装",
			Lat: "30.8700000", Lng: "120.1200000",
		}},
		nearbyMerchantPlacesErr: errors.New("nearby database timeout"),
	}
	var logBuffer bytes.Buffer
	logx.SetWriter(logx.NewWriter(&logBuffer))
	t.Cleanup(func() {
		_ = logx.Reset().Close()
	})

	resp, err := NewPublicLogic(store).GetMerchantLocationContext(context.Background(), "merchant-1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Current.MerchantId != "merchant-1" || len(resp.Nearby) != 0 || resp.NearbyAvailable || resp.RadiusMeters != 1000 {
		t.Fatalf("resp = %#v, want current merchant with unavailable nearby results", resp)
	}
	logText := logBuffer.String()
	for _, field := range []string{"merchantId=merchant-1", "radiusMeters=1000", "limit=20"} {
		if !strings.Contains(logText, field) {
			t.Fatalf("log = %q, want field %q", logText, field)
		}
	}
}

func TestPublicMapLogicRejectsInvalidMerchantPlaceClaimedFilter(t *testing.T) {
	logic := NewPublicLogic(&fakePublicMapStore{})

	_, err := logic.ListMerchantPlaces(context.Background(), ListMerchantPlacesReq{Claimed: "unknown"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed || err.Error() != "商家状态筛选不正确，请重新选择" {
		t.Fatalf("error = %v code=%s, want claimed validation", err, errx.CodeOf(err))
	}
}

func TestPublicMapLogicListsNormalObjectsWithParsedFilters(t *testing.T) {
	store := &fakePublicMapStore{
		objects: []model.MapObject{{ID: "object-1", SceneCode: "scene-1", Code: "A001", Name: "A001 小鹿童装", MinZoom: 2, MaxZoom: 4, Status: model.MapObjectStatusNormal}},
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
	if resp.Items[0].MinZoom != 2 || resp.Items[0].MaxZoom != 4 {
		t.Fatalf("zoom range = %d-%d, want 2-4", resp.Items[0].MinZoom, resp.Items[0].MaxZoom)
	}
}

func TestPublicMapLogicHighlightsVerifiedMerchantObject(t *testing.T) {
	store := &fakePublicMapStore{
		objects: []model.MapObject{{
			ID:                         "object-1",
			SceneCode:                  "scene-1",
			Code:                       "A001",
			Name:                       "后台档口 A001",
			MerchantID:                 "merchant-1",
			MerchantName:               "织里认证童装厂",
			MerchantType:               "factory",
			MerchantVerificationStatus: "verified",
			MerchantLogoURL:            "https://img.example.com/logo.png",
			MerchantMainCategories:     []string{"女童", "现货"},
			Status:                     model.MapObjectStatusNormal,
		}},
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListObjects(context.Background(), "scene-1", ListObjectsReq{})
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}

	item := resp.Items[0]
	if item.DisplaySource != "verified_merchant" || item.DisplayLevel != "highlight" || !item.IsVerifiedMerchant {
		t.Fatalf("item = %#v, want verified merchant highlight", item)
	}
	if item.Name != "织里认证童装厂" || item.MerchantId != "merchant-1" {
		t.Fatalf("item = %#v, want merchant identity to override admin object name", item)
	}
	if item.Merchant == nil || item.Merchant.Name != "织里认证童装厂" || item.Merchant.LogoUrl == "" {
		t.Fatalf("merchant = %#v, want verified merchant summary", item.Merchant)
	}
}

func TestMapObjectItemsHideContactFromPublicButKeepAdmin(t *testing.T) {
	object := model.MapObject{
		ID:     "object-1",
		Phone:  "18800000001",
		Wechat: "xiaolu001",
	}

	publicItem := mapPublicObjectItem(object)
	if publicItem.Phone != "" || publicItem.Wechat != "" {
		t.Fatalf("public item = %#v, want contact hidden outside supply and demand", publicItem)
	}
	adminItem := mapAdminObjectItem(object)
	if adminItem.Phone != "18800000001" || adminItem.Wechat != "xiaolu001" {
		t.Fatalf("admin item = %#v, want contact retained for maintenance", adminItem)
	}
}

func TestPublicMapLogicKeepsUnverifiedMerchantObjectWeak(t *testing.T) {
	store := &fakePublicMapStore{
		objects: []model.MapObject{{
			ID:                         "object-1",
			SceneCode:                  "scene-1",
			Code:                       "A001",
			Name:                       "后台档口 A001",
			MerchantID:                 "merchant-1",
			MerchantName:               "未认证童装厂",
			MerchantVerificationStatus: "unverified",
			Status:                     model.MapObjectStatusNormal,
		}},
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListObjects(context.Background(), "scene-1", ListObjectsReq{})
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}

	item := resp.Items[0]
	if item.DisplaySource != "admin_object" || item.DisplayLevel != "weak" || item.IsVerifiedMerchant {
		t.Fatalf("item = %#v, want weak admin object", item)
	}
	if item.Name != "后台档口 A001" || item.MerchantId != "" || item.Merchant != nil {
		t.Fatalf("item = %#v, want public response to hide unverified merchant display info", item)
	}
}

func TestPublicMapLogicListsObjectsWithViewportAndZoom(t *testing.T) {
	store := &fakePublicMapStore{}
	logic := NewPublicLogic(store)

	_, err := logic.ListObjects(context.Background(), " scene-1 ", ListObjectsReq{
		MinX: " 100 ",
		MinY: "200",
		MaxX: " 900 ",
		MaxY: "700",
		Zoom: 4,
	})
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}

	if store.objectFilter.Viewport == nil {
		t.Fatalf("viewport = nil, want parsed viewport")
	}
	if store.objectFilter.Viewport.MinX != 100 || store.objectFilter.Viewport.MinY != 200 || store.objectFilter.Viewport.MaxX != 900 || store.objectFilter.Viewport.MaxY != 700 {
		t.Fatalf("viewport = %#v, want parsed bounds", store.objectFilter.Viewport)
	}
	if store.objectFilter.Zoom != 4 {
		t.Fatalf("zoom = %d, want 4", store.objectFilter.Zoom)
	}
}

func TestPublicMapLogicRejectsNonFiniteViewportNumber(t *testing.T) {
	store := &fakePublicMapStore{}
	logic := NewPublicLogic(store)

	_, err := logic.ListObjects(context.Background(), "scene-1", ListObjectsReq{
		MinX: "NaN",
		MinY: "200",
		MaxX: "900",
		MaxY: "700",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed || err.Error() != "视口最小 X必须是有效数字，请刷新后重试" {
		t.Fatalf("ListObjects() error = %v, code=%s, want non-finite viewport validation", err, errx.CodeOf(err))
	}
	if store.objectFilter.SceneCode != "" || store.countFilter.SceneCode != "" {
		t.Fatalf("filters = %#v %#v, want no query after viewport validation error", store.objectFilter, store.countFilter)
	}
}

func TestPublicMapLogicListsObjectsReturnsSceneTotalIgnoringViewport(t *testing.T) {
	store := &fakePublicMapStore{
		objectTotal: 600,
	}
	logic := NewPublicLogic(store)

	resp, err := logic.ListObjects(context.Background(), " scene-1 ", ListObjectsReq{
		Types:      "booth",
		Categories: "girl",
		MinX:       "100",
		MinY:       "200",
		MaxX:       "900",
		MaxY:       "700",
		Zoom:       4,
	})
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}

	if resp.Total != 600 {
		t.Fatalf("total = %d, want 600", resp.Total)
	}
	if store.countFilter.SceneCode != "scene-1" || store.countFilter.Status != model.MapObjectStatusNormal {
		t.Fatalf("count filter = %#v, want scene normal", store.countFilter)
	}
	if store.countFilter.Viewport != nil || store.countFilter.Zoom != 0 || store.countFilter.Limit != 0 {
		t.Fatalf("count filter = %#v, want no viewport zoom or limit", store.countFilter)
	}
	if len(store.countFilter.Types) != 1 || store.countFilter.Types[0] != "booth" || len(store.countFilter.Categories) != 1 || store.countFilter.Categories[0] != "girl" {
		t.Fatalf("count filter = %#v, want parsed filters", store.countFilter)
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

func TestPublicMapLogicSearchReturnsTotalIgnoringLimitAndViewport(t *testing.T) {
	store := &fakePublicMapStore{
		objectTotal: 23,
	}
	logic := NewPublicLogic(store)

	resp, err := logic.SearchObjects(context.Background(), SearchObjectsReq{
		SceneCode: " scene-1 ",
		Keyword:   " 女童 ",
		MinX:      "100",
		MinY:      "200",
		MaxX:      "900",
		MaxY:      "700",
		Zoom:      4,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("SearchObjects() error = %v", err)
	}

	if resp.Total != 23 {
		t.Fatalf("total = %d, want 23", resp.Total)
	}
	if store.countFilter.SceneCode != "scene-1" || store.countFilter.Keyword != "女童" || store.countFilter.Status != model.MapObjectStatusNormal {
		t.Fatalf("count filter = %#v, want scene keyword normal", store.countFilter)
	}
	if store.countFilter.Viewport != nil || store.countFilter.Zoom != 0 || store.countFilter.Limit != 0 {
		t.Fatalf("count filter = %#v, want no viewport zoom or limit", store.countFilter)
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

	if store.categoryFilter.Type != "booth_category" || store.categoryFilter.Status != model.MapCategoryStatusNormal {
		t.Fatalf("category filter = %#v, want booth_category normal", store.categoryFilter)
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
	sceneFilter             model.ListMapScenesFilter
	objectFilter            model.ListMapObjectsFilter
	countFilter             model.ListMapObjectsFilter
	objectID                string
	categoryFilter          model.ListMapCategoriesFilter
	nearbySceneCode         string
	nearbyTypes             []string
	scenes                  []model.MapScene
	scene                   model.MapScene
	objects                 []model.MapObject
	objectTotal             int64
	object                  model.MapObject
	nearby                  []model.MapObject
	categories              []model.MapCategory
	merchantPlaceFilter     model.MerchantPlaceFilter
	merchantPlaces          []model.MerchantPlace
	merchantPlaceTotal      int64
	merchantID              string
	merchantPlace           model.MerchantPlace
	merchantPlaceErr        error
	nearbyMerchantPlaces    []model.MerchantPlace
	nearbyMerchantPlacesErr error
	nearbyOrigin            model.MerchantPlace
	nearbyRadiusMeters      int64
	nearbyLimit             int64
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

func (s *fakePublicMapStore) CountPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) (int64, error) {
	s.countFilter = filter
	return s.objectTotal, nil
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

func (s *fakePublicMapStore) ListCategories(ctx context.Context, filter model.ListMapCategoriesFilter) ([]model.MapCategory, error) {
	s.categoryFilter = filter
	return append([]model.MapCategory(nil), s.categories...), nil
}

func (s *fakePublicMapStore) ListMerchantPlaces(ctx context.Context, filter model.MerchantPlaceFilter) ([]model.MerchantPlace, error) {
	s.merchantPlaceFilter = filter
	return append([]model.MerchantPlace(nil), s.merchantPlaces...), nil
}

func (s *fakePublicMapStore) CountMerchantPlaces(ctx context.Context, filter model.MerchantPlaceFilter) (int64, error) {
	s.merchantPlaceFilter = filter
	return s.merchantPlaceTotal, nil
}

func (s *fakePublicMapStore) GetPublishedMerchantPlaceByMerchantID(ctx context.Context, merchantID string) (model.MerchantPlace, error) {
	s.merchantID = merchantID
	return s.merchantPlace, s.merchantPlaceErr
}

func (s *fakePublicMapStore) ListNearbyMerchantPlaces(ctx context.Context, origin model.MerchantPlace, radiusMeters int64, limit int64) ([]model.MerchantPlace, error) {
	s.nearbyOrigin = origin
	s.nearbyRadiusMeters = radiusMeters
	s.nearbyLimit = limit
	return append([]model.MerchantPlace(nil), s.nearbyMerchantPlaces...), s.nearbyMerchantPlacesErr
}
