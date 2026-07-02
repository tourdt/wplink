package maplogic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublicStore interface {
	ListPublishedScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error)
	GetPublishedScene(ctx context.Context, sceneCode string) (model.MapScene, error)
	ListPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error)
	SearchPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error)
	GetPublishedObject(ctx context.Context, objectID string) (model.MapObject, error)
	ListObjectsBySceneAndTypes(ctx context.Context, sceneCode string, types []string) ([]model.MapObject, error)
	ListCategories(ctx context.Context, filter model.ListMapCategoriesFilter) ([]model.MapCategory, error)
}

type PublicLogic struct {
	store PublicStore
}

func NewPublicLogic(store PublicStore) *PublicLogic {
	return &PublicLogic{store: store}
}

type ListScenesReq struct {
	CityCode   string
	ParentCode string
	Type       string
}

type ListScenesResp struct {
	Items []MapSceneItem `json:"items"`
}

type MapSceneItem struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	ParentCode     string `json:"parentCode,omitempty"`
	BackgroundUrl  string `json:"backgroundUrl,omitempty"`
	Width          int64  `json:"width,omitempty"`
	Height         int64  `json:"height,omitempty"`
	DefaultScale   string `json:"defaultScale,omitempty"`
	DefaultCenterX string `json:"defaultCenterX,omitempty"`
	DefaultCenterY string `json:"defaultCenterY,omitempty"`
	FloorNo        string `json:"floorNo,omitempty"`
	Sort           int64  `json:"sort"`
	Status         string `json:"status"`
}

type SceneResp struct {
	Item MapSceneItem `json:"item"`
}

type ListObjectsReq struct {
	Types          string
	Categories     string
	ServiceTags    string
	PoiServiceTags string
	Keyword        string
}

type ListObjectsResp struct {
	SceneCode string          `json:"sceneCode"`
	Items     []MapObjectItem `json:"items"`
}

type SearchObjectsReq struct {
	SceneCode      string
	Keyword        string
	Types          string
	Categories     string
	ServiceTags    string
	PoiServiceTags string
	Limit          int64
}

type SearchObjectsResp struct {
	Items []MapObjectItem `json:"items"`
}

type MapObjectItem struct {
	Id             string                 `json:"id"`
	SceneCode      string                 `json:"sceneCode"`
	Code           string                 `json:"code"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Layer          string                 `json:"layer"`
	GeometryType   string                 `json:"geometryType"`
	Geometry       map[string]interface{} `json:"geometry"`
	CenterX        string                 `json:"centerX,omitempty"`
	CenterY        string                 `json:"centerY,omitempty"`
	MinZoom        int64                  `json:"minZoom,omitempty"`
	MaxZoom        int64                  `json:"maxZoom,omitempty"`
	CategoryCodes  []string               `json:"categoryCodes"`
	ServiceTags    []string               `json:"serviceTags"`
	PlatformTags   []string               `json:"platformTags"`
	PoiServiceTags []string               `json:"poiServiceTags"`
	Address        string                 `json:"address,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	Wechat         string                 `json:"wechat,omitempty"`
	Lat            string                 `json:"lat,omitempty"`
	Lng            string                 `json:"lng,omitempty"`
	Extra          map[string]interface{} `json:"extra"`
	Status         string                 `json:"status"`
}

type ObjectDetailResp struct {
	Item MapObjectItem `json:"item"`
}

type ListNearbyPoisReq struct {
	Types string
	Limit int64
}

type NearbyPoiItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	DistanceText string `json:"distanceText"`
	CenterX      string `json:"centerX,omitempty"`
	CenterY      string `json:"centerY,omitempty"`
}

type ListNearbyPoisResp struct {
	Items []NearbyPoiItem `json:"items"`
}

func (l *PublicLogic) ListScenes(ctx context.Context, req ListScenesReq) (ListScenesResp, error) {
	scenes, err := l.store.ListPublishedScenes(ctx, model.ListMapScenesFilter{
		CityCode:   strings.TrimSpace(req.CityCode),
		ParentCode: strings.TrimSpace(req.ParentCode),
		Type:       strings.TrimSpace(req.Type),
		Status:     model.MapSceneStatusPublished,
	})
	if err != nil {
		logx.Errorf("查询拿货地图场景失败: cityCode=%s parentCode=%s type=%s err=%+v", req.CityCode, req.ParentCode, req.Type, err)
		return ListScenesResp{}, errx.New(errx.CodeInternalError, "地图场景加载失败，请稍后重试")
	}
	return ListScenesResp{Items: mapSceneItems(scenes)}, nil
}

func (l *PublicLogic) GetScene(ctx context.Context, sceneCode string) (SceneResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	if sceneCode == "" {
		return SceneResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	scene, err := l.store.GetPublishedScene(ctx, sceneCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SceneResp{}, errx.New(errx.CodeResourceNotFound, "地图场景不存在或未发布")
		}
		logx.Errorf("查询拿货地图场景详情失败: sceneCode=%s err=%+v", sceneCode, err)
		return SceneResp{}, errx.New(errx.CodeInternalError, "地图场景加载失败，请稍后重试")
	}
	return SceneResp{Item: mapSceneItem(scene)}, nil
}

func (l *PublicLogic) ListObjects(ctx context.Context, sceneCode string, req ListObjectsReq) (ListObjectsResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	if sceneCode == "" {
		return ListObjectsResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	filter := model.ListMapObjectsFilter{
		SceneCode:      sceneCode,
		Types:          splitCSV(req.Types),
		Categories:     splitCSV(req.Categories),
		ServiceTags:    splitCSV(req.ServiceTags),
		PoiServiceTags: splitCSV(req.PoiServiceTags),
		Keyword:        strings.TrimSpace(req.Keyword),
		Status:         model.MapObjectStatusNormal,
	}
	objects, err := l.store.ListPublishedObjects(ctx, filter)
	if err != nil {
		logx.Errorf("查询拿货地图对象失败: sceneCode=%s keyword=%s err=%+v", sceneCode, req.Keyword, err)
		return ListObjectsResp{}, errx.New(errx.CodeInternalError, "地图点位加载失败，请稍后重试")
	}
	return ListObjectsResp{SceneCode: sceneCode, Items: mapObjectItems(objects)}, nil
}

func (l *PublicLogic) SearchObjects(ctx context.Context, req SearchObjectsReq) (SearchObjectsResp, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	filter := model.ListMapObjectsFilter{
		SceneCode:      strings.TrimSpace(req.SceneCode),
		Types:          splitCSV(req.Types),
		Categories:     splitCSV(req.Categories),
		ServiceTags:    splitCSV(req.ServiceTags),
		PoiServiceTags: splitCSV(req.PoiServiceTags),
		Keyword:        strings.TrimSpace(req.Keyword),
		Status:         model.MapObjectStatusNormal,
		Limit:          limit,
	}
	objects, err := l.store.SearchPublishedObjects(ctx, filter)
	if err != nil {
		logx.Errorf("搜索拿货地图对象失败: sceneCode=%s keyword=%s err=%+v", req.SceneCode, req.Keyword, err)
		return SearchObjectsResp{}, errx.New(errx.CodeInternalError, "地图搜索失败，请稍后重试")
	}
	return SearchObjectsResp{Items: mapObjectItems(objects)}, nil
}

func (l *PublicLogic) ListCategories(ctx context.Context, req ListCategoriesReq) (ListCategoriesResp, error) {
	filter := model.ListMapCategoriesFilter{
		Type:   strings.TrimSpace(req.Type),
		Status: model.MapCategoryStatusNormal,
	}
	categories, err := l.store.ListCategories(ctx, filter)
	if err != nil {
		logx.Errorf("查询公开地图分类失败: type=%s err=%+v", req.Type, err)
		return ListCategoriesResp{}, errx.New(errx.CodeInternalError, "地图筛选项加载失败，请稍后重试")
	}
	return ListCategoriesResp{Items: publicMapCategoryItems(categories)}, nil
}

func (l *PublicLogic) GetObject(ctx context.Context, objectID string) (ObjectDetailResp, error) {
	objectID = strings.TrimSpace(objectID)
	if objectID == "" {
		return ObjectDetailResp{}, errx.New(errx.CodeValidationFailed, "请选择地图点位")
	}
	object, err := l.store.GetPublishedObject(ctx, objectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ObjectDetailResp{}, errx.New(errx.CodeResourceNotFound, "地图点位不存在或未发布")
		}
		logx.Errorf("查询拿货地图对象详情失败: objectID=%s err=%+v", objectID, err)
		return ObjectDetailResp{}, errx.New(errx.CodeInternalError, "地图点位加载失败，请稍后重试")
	}
	return ObjectDetailResp{Item: mapObjectItem(object)}, nil
}

func (l *PublicLogic) ListNearbyPois(ctx context.Context, objectID string, req ListNearbyPoisReq) (ListNearbyPoisResp, error) {
	objectID = strings.TrimSpace(objectID)
	if objectID == "" {
		return ListNearbyPoisResp{}, errx.New(errx.CodeValidationFailed, "请选择地图点位")
	}
	origin, err := l.store.GetPublishedObject(ctx, objectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ListNearbyPoisResp{}, errx.New(errx.CodeResourceNotFound, "地图点位不存在或未发布")
		}
		logx.Errorf("查询附近配套原点失败: objectID=%s err=%+v", objectID, err)
		return ListNearbyPoisResp{}, errx.New(errx.CodeInternalError, "附近配套加载失败，请稍后重试")
	}
	types := splitCSV(req.Types)
	if len(types) == 0 {
		types = []string{"packing_station", "logistics_point", "express_point", "parking"}
	}
	candidates, err := l.store.ListObjectsBySceneAndTypes(ctx, origin.SceneCode, types)
	if err != nil {
		logx.Errorf("查询附近配套候选失败: objectID=%s sceneCode=%s types=%v err=%+v", objectID, origin.SceneCode, types, err)
		return ListNearbyPoisResp{}, errx.New(errx.CodeInternalError, "附近配套加载失败，请稍后重试")
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 5
	}
	nearby := model.SortNearbyMapObjects(origin, candidates, limit)
	items := make([]NearbyPoiItem, 0, len(nearby))
	for _, item := range nearby {
		items = append(items, NearbyPoiItem{
			Id:           item.ID,
			Name:         item.Name,
			Type:         item.Type,
			DistanceText: item.DistanceText,
			CenterX:      formatFloat(item.CenterX),
			CenterY:      formatFloat(item.CenterY),
		})
	}
	return ListNearbyPoisResp{Items: items}, nil
}

func publicMapCategoryItems(categories []model.MapCategory) []MapCategoryItem {
	items := make([]MapCategoryItem, 0, len(categories))
	for _, category := range categories {
		if !category.IsVisible || category.Status != model.MapCategoryStatusNormal {
			continue
		}
		items = append(items, mapCategoryItem(category))
	}
	return items
}

func mapSceneItems(scenes []model.MapScene) []MapSceneItem {
	items := make([]MapSceneItem, 0, len(scenes))
	for _, scene := range scenes {
		items = append(items, mapSceneItem(scene))
	}
	return items
}

func mapSceneItem(scene model.MapScene) MapSceneItem {
	return MapSceneItem{
		Code:           scene.Code,
		Name:           scene.Name,
		Type:           scene.Type,
		ParentCode:     scene.ParentCode,
		BackgroundUrl:  scene.BackgroundURL,
		Width:          scene.Width,
		Height:         scene.Height,
		DefaultScale:   scene.DefaultScale,
		DefaultCenterX: scene.DefaultCenterX,
		DefaultCenterY: scene.DefaultCenterY,
		FloorNo:        scene.FloorNo,
		Sort:           scene.Sort,
		Status:         scene.Status,
	}
}

func mapObjectItems(objects []model.MapObject) []MapObjectItem {
	items := make([]MapObjectItem, 0, len(objects))
	for _, object := range objects {
		items = append(items, mapObjectItem(object))
	}
	return items
}

func mapObjectItem(object model.MapObject) MapObjectItem {
	return MapObjectItem{
		Id:             object.ID,
		SceneCode:      object.SceneCode,
		Code:           object.Code,
		Name:           object.Name,
		Type:           object.Type,
		Layer:          object.Layer,
		GeometryType:   object.GeometryType,
		Geometry:       map[string]interface{}(object.Geometry),
		CenterX:        formatFloat(object.CenterX),
		CenterY:        formatFloat(object.CenterY),
		MinZoom:        object.MinZoom,
		MaxZoom:        object.MaxZoom,
		CategoryCodes:  append([]string(nil), object.CategoryCodes...),
		ServiceTags:    append([]string(nil), object.ServiceTags...),
		PlatformTags:   append([]string(nil), object.PlatformTags...),
		PoiServiceTags: append([]string(nil), object.PoiServiceTags...),
		Address:        object.Address,
		Phone:          object.Phone,
		Wechat:         object.Wechat,
		Lat:            object.Lat,
		Lng:            object.Lng,
		Extra:          map[string]interface{}(object.Extra),
		Status:         object.Status,
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func formatFloat(value float64) string {
	text := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
	if text == "-0" {
		return "0"
	}
	return text
}
