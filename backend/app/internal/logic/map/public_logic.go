package maplogic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	merchantLocationRadiusMeters int64 = 1000
	merchantLocationNearbyLimit  int64 = 20
)

type PublicStore interface {
	ListPublishedScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error)
	GetPublishedScene(ctx context.Context, sceneCode string) (model.MapScene, error)
	ListPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error)
	SearchPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error)
	CountPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) (int64, error)
	GetPublishedObject(ctx context.Context, objectID string) (model.MapObject, error)
	ListObjectsBySceneAndTypes(ctx context.Context, sceneCode string, types []string) ([]model.MapObject, error)
	ListCategories(ctx context.Context, filter model.ListMapCategoriesFilter) ([]model.MapCategory, error)
	ListMerchantPlaces(ctx context.Context, filter model.MerchantPlaceFilter) ([]model.MerchantPlace, error)
	CountMerchantPlaces(ctx context.Context, filter model.MerchantPlaceFilter) (int64, error)
	GetPublishedMerchantPlaceByMerchantID(ctx context.Context, merchantID string) (model.MerchantPlace, error)
	ListNearbyMerchantPlaces(ctx context.Context, origin model.MerchantPlace, radiusMeters int64, limit int64) ([]model.MerchantPlace, error)
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
	MinX           string
	MinY           string
	MaxX           string
	MaxY           string
	Zoom           int64
}

type ListObjectsResp struct {
	SceneCode string          `json:"sceneCode"`
	Items     []MapObjectItem `json:"items"`
	Total     int64           `json:"total"`
}

type SearchObjectsReq struct {
	SceneCode      string
	Keyword        string
	Types          string
	Categories     string
	ServiceTags    string
	PoiServiceTags string
	MinX           string
	MinY           string
	MaxX           string
	MaxY           string
	Zoom           int64
	Limit          int64
}

type SearchObjectsResp struct {
	Items []MapObjectItem `json:"items"`
	Total int64           `json:"total"`
}

type ListMerchantPlacesReq struct {
	CityCode      string
	Keyword       string
	Categories    string
	MerchantTypes string
	Claimed       string
	Page          int64
	PageSize      int64
	MinLat        string
	MaxLat        string
	MinLng        string
	MaxLng        string
	Lat           string
	Lng           string
}

type ListMerchantPlacesResp struct {
	Items    []MerchantPlaceItem `json:"items"`
	Total    int64               `json:"total"`
	Page     int64               `json:"page"`
	PageSize int64               `json:"pageSize"`
}

type MerchantPlaceItem struct {
	ObjectId       string   `json:"objectId"`
	MerchantId     string   `json:"merchantId,omitempty"`
	Name           string   `json:"name"`
	Code           string   `json:"code"`
	MerchantType   string   `json:"merchantType,omitempty"`
	CategoryCodes  []string `json:"categoryCodes"`
	ServiceTags    []string `json:"serviceTags"`
	PlatformTags   []string `json:"platformTags"`
	CityCode       string   `json:"cityCode,omitempty"`
	MarketName     string   `json:"marketName,omitempty"`
	BuildingName   string   `json:"buildingName,omitempty"`
	FloorNo        string   `json:"floorNo,omitempty"`
	Address        string   `json:"address,omitempty"`
	CoverUrl       string   `json:"coverUrl,omitempty"`
	Claimed        bool     `json:"claimed"`
	SourceType     string   `json:"sourceType"`
	Lat            string   `json:"lat,omitempty"`
	Lng            string   `json:"lng,omitempty"`
	DistanceText   string   `json:"distanceText,omitempty"`
	DistanceMeters int64    `json:"distanceMeters,omitempty"`
	RiskWarning    bool     `json:"riskWarning,omitempty"`
}

type MerchantLocationContextResp struct {
	Current         MerchantPlaceItem   `json:"current"`
	Nearby          []MerchantPlaceItem `json:"nearby"`
	RadiusMeters    int64               `json:"radiusMeters"`
	NearbyAvailable bool                `json:"nearbyAvailable"`
}

type MapObjectItem struct {
	Id                 string                 `json:"id"`
	SceneCode          string                 `json:"sceneCode"`
	MerchantId         string                 `json:"merchantId,omitempty"`
	Code               string                 `json:"code"`
	Name               string                 `json:"name"`
	Type               string                 `json:"type"`
	Layer              string                 `json:"layer"`
	DisplaySource      string                 `json:"displaySource"`
	DisplayLevel       string                 `json:"displayLevel"`
	IsVerifiedMerchant bool                   `json:"isVerifiedMerchant"`
	Merchant           *MapObjectMerchantItem `json:"merchant,omitempty"`
	GeometryType       string                 `json:"geometryType"`
	Geometry           map[string]interface{} `json:"geometry"`
	CenterX            string                 `json:"centerX,omitempty"`
	CenterY            string                 `json:"centerY,omitempty"`
	MinZoom            int64                  `json:"minZoom,omitempty"`
	MaxZoom            int64                  `json:"maxZoom,omitempty"`
	CategoryCodes      []string               `json:"categoryCodes"`
	ServiceTags        []string               `json:"serviceTags"`
	PlatformTags       []string               `json:"platformTags"`
	PoiServiceTags     []string               `json:"poiServiceTags"`
	Address            string                 `json:"address,omitempty"`
	Phone              string                 `json:"phone,omitempty"`
	Wechat             string                 `json:"wechat,omitempty"`
	Lat                string                 `json:"lat,omitempty"`
	Lng                string                 `json:"lng,omitempty"`
	Extra              map[string]interface{} `json:"extra"`
	Status             string                 `json:"status"`
}

type MapObjectMerchantItem struct {
	Id                 string   `json:"id"`
	Name               string   `json:"name"`
	MerchantType       string   `json:"merchantType"`
	VerificationStatus string   `json:"verificationStatus"`
	LogoUrl            string   `json:"logoUrl,omitempty"`
	MainCategories     []string `json:"mainCategories"`
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

func (l *PublicLogic) ListMerchantPlaces(ctx context.Context, req ListMerchantPlacesReq) (ListMerchantPlacesResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		return ListMerchantPlacesResp{}, errx.New(errx.CodeValidationFailed, "每页商家数量不能超过 100 条")
	}
	claimed, err := parseMerchantPlaceClaimed(req.Claimed)
	if err != nil {
		return ListMerchantPlacesResp{}, err
	}
	bounds, err := parseGeoBoundsFilter(req.MinLat, req.MaxLat, req.MinLng, req.MaxLng)
	if err != nil {
		return ListMerchantPlacesResp{}, err
	}
	filter := model.MerchantPlaceFilter{
		CityCode:      strings.TrimSpace(req.CityCode),
		Keyword:       strings.TrimSpace(req.Keyword),
		Categories:    splitCSV(req.Categories),
		MerchantTypes: splitCSV(req.MerchantTypes),
		Claimed:       claimed,
		Bounds:        bounds,
		Page:          page,
		PageSize:      pageSize,
	}
	places, err := l.store.ListMerchantPlaces(ctx, filter)
	if err != nil {
		logx.Errorf("查询商家目录失败: cityCode=%s keyword=%s page=%d pageSize=%d err=%+v", filter.CityCode, filter.Keyword, page, pageSize, err)
		return ListMerchantPlacesResp{}, errx.New(errx.CodeInternalError, "商家列表加载失败，请稍后重试")
	}
	total, err := l.store.CountMerchantPlaces(ctx, filter)
	if err != nil {
		logx.Errorf("统计商家目录失败: cityCode=%s keyword=%s err=%+v", filter.CityCode, filter.Keyword, err)
		return ListMerchantPlacesResp{}, errx.New(errx.CodeInternalError, "商家数量加载失败，请稍后重试")
	}
	return ListMerchantPlacesResp{
		Items:    mapMerchantPlaceItems(places),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (l *PublicLogic) GetMerchantLocationContext(ctx context.Context, merchantID string) (MerchantLocationContextResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return MerchantLocationContextResp{}, errx.New(errx.CodeValidationFailed, "该商家暂时无法查看")
	}

	current, err := l.store.GetPublishedMerchantPlaceByMerchantID(ctx, merchantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MerchantLocationContextResp{}, errx.New(errx.CodeResourceNotFound, "该商家暂时无法查看")
		}
		logx.Errorf("查询商家位置上下文原点失败: merchantId=%s err=%+v", merchantID, err)
		return MerchantLocationContextResp{}, errx.New(errx.CodeInternalError, "该商家暂时无法查看")
	}
	if !validMerchantPlaceCoordinates(current) {
		return MerchantLocationContextResp{}, errx.New(errx.CodeValidationFailed, "该商家位置待完善")
	}

	currentItem := mapMerchantPlaceItems([]model.MerchantPlace{current})[0]
	nearby, err := l.store.ListNearbyMerchantPlaces(ctx, current, merchantLocationRadiusMeters, merchantLocationNearbyLimit)
	if err != nil {
		// 周边推荐是位置页的辅助能力，查询失败时保留当前商家，避免影响用户查看地址和导航。
		logx.Errorf(
			"查询商家位置上下文周边商家失败: merchantId=%s radiusMeters=%d limit=%d err=%+v",
			merchantID,
			merchantLocationRadiusMeters,
			merchantLocationNearbyLimit,
			err,
		)
		return MerchantLocationContextResp{
			Current:         currentItem,
			Nearby:          []MerchantPlaceItem{},
			RadiusMeters:    merchantLocationRadiusMeters,
			NearbyAvailable: false,
		}, nil
	}

	return MerchantLocationContextResp{
		Current:         currentItem,
		Nearby:          mapMerchantPlaceItems(nearby),
		RadiusMeters:    merchantLocationRadiusMeters,
		NearbyAvailable: true,
	}, nil
}

func validMerchantPlaceCoordinates(place model.MerchantPlace) bool {
	lat, latErr := strconv.ParseFloat(strings.TrimSpace(place.Object.Lat), 64)
	lng, lngErr := strconv.ParseFloat(strings.TrimSpace(place.Object.Lng), 64)
	return latErr == nil && lngErr == nil &&
		!math.IsNaN(lat) && !math.IsInf(lat, 0) && lat >= -90 && lat <= 90 &&
		!math.IsNaN(lng) && !math.IsInf(lng, 0) && lng >= -180 && lng <= 180
}

func parseMerchantPlaceClaimed(value string) (*bool, error) {
	switch strings.TrimSpace(value) {
	case "", "all":
		return nil, nil
	case "claimed":
		value := true
		return &value, nil
	case "prelisted":
		value := false
		return &value, nil
	default:
		return nil, errx.New(errx.CodeValidationFailed, "商家状态筛选不正确，请重新选择")
	}
}

func parseGeoBoundsFilter(minLatText, maxLatText, minLngText, maxLngText string) (*model.GeoBoundsFilter, error) {
	values := []string{minLatText, maxLatText, minLngText, maxLngText}
	hasValue := false
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			hasValue = true
			break
		}
	}
	if !hasValue {
		return nil, nil
	}
	minLat, err := parseGeoNumber(minLatText, "最小纬度", -90, 90)
	if err != nil {
		return nil, err
	}
	maxLat, err := parseGeoNumber(maxLatText, "最大纬度", -90, 90)
	if err != nil {
		return nil, err
	}
	minLng, err := parseGeoNumber(minLngText, "最小经度", -180, 180)
	if err != nil {
		return nil, err
	}
	maxLng, err := parseGeoNumber(maxLngText, "最大经度", -180, 180)
	if err != nil {
		return nil, err
	}
	if minLat > maxLat || minLng > maxLng {
		return nil, errx.New(errx.CodeValidationFailed, "地图区域范围不正确，请重新搜索")
	}
	return &model.GeoBoundsFilter{MinLat: minLat, MaxLat: maxLat, MinLng: minLng, MaxLng: maxLng}, nil
}

func parseGeoNumber(value, label string, minValue, maxValue float64) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errx.New(errx.CodeValidationFailed, "地图区域参数不完整，请重新搜索")
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < minValue || number > maxValue {
		return 0, errx.New(errx.CodeValidationFailed, label+"不正确，请重新搜索")
	}
	return number, nil
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
	viewport, err := parseMapObjectViewportFilter(req.MinX, req.MinY, req.MaxX, req.MaxY)
	if err != nil {
		return ListObjectsResp{}, err
	}
	filter := model.ListMapObjectsFilter{
		SceneCode:      sceneCode,
		Types:          splitCSV(req.Types),
		Categories:     splitCSV(req.Categories),
		ServiceTags:    splitCSV(req.ServiceTags),
		PoiServiceTags: splitCSV(req.PoiServiceTags),
		Keyword:        strings.TrimSpace(req.Keyword),
		Status:         model.MapObjectStatusNormal,
		Viewport:       viewport,
		Zoom:           req.Zoom,
	}
	objects, err := l.store.ListPublishedObjects(ctx, filter)
	if err != nil {
		logx.Errorf("查询拿货地图对象失败: sceneCode=%s keyword=%s viewport=%+v zoom=%d err=%+v", sceneCode, req.Keyword, viewport, req.Zoom, err)
		return ListObjectsResp{}, errx.New(errx.CodeInternalError, "地图点位加载失败，请稍后重试")
	}
	total, err := l.countPublishedObjects(ctx, filter)
	if err != nil {
		logx.Errorf("统计拿货地图对象总数失败: sceneCode=%s keyword=%s err=%+v", sceneCode, req.Keyword, err)
		return ListObjectsResp{}, errx.New(errx.CodeInternalError, "地图点位数量加载失败，请稍后重试")
	}
	return ListObjectsResp{SceneCode: sceneCode, Items: mapPublicObjectItems(objects), Total: total}, nil
}

func (l *PublicLogic) SearchObjects(ctx context.Context, req SearchObjectsReq) (SearchObjectsResp, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	viewport, err := parseMapObjectViewportFilter(req.MinX, req.MinY, req.MaxX, req.MaxY)
	if err != nil {
		return SearchObjectsResp{}, err
	}
	filter := model.ListMapObjectsFilter{
		SceneCode:      strings.TrimSpace(req.SceneCode),
		Types:          splitCSV(req.Types),
		Categories:     splitCSV(req.Categories),
		ServiceTags:    splitCSV(req.ServiceTags),
		PoiServiceTags: splitCSV(req.PoiServiceTags),
		Keyword:        strings.TrimSpace(req.Keyword),
		Status:         model.MapObjectStatusNormal,
		Viewport:       viewport,
		Zoom:           req.Zoom,
		Limit:          limit,
	}
	objects, err := l.store.SearchPublishedObjects(ctx, filter)
	if err != nil {
		logx.Errorf("搜索拿货地图对象失败: sceneCode=%s keyword=%s viewport=%+v zoom=%d err=%+v", req.SceneCode, req.Keyword, viewport, req.Zoom, err)
		return SearchObjectsResp{}, errx.New(errx.CodeInternalError, "地图搜索失败，请稍后重试")
	}
	total, err := l.countPublishedObjects(ctx, filter)
	if err != nil {
		logx.Errorf("统计拿货地图搜索结果总数失败: sceneCode=%s keyword=%s err=%+v", req.SceneCode, req.Keyword, err)
		return SearchObjectsResp{}, errx.New(errx.CodeInternalError, "地图搜索数量加载失败，请稍后重试")
	}
	return SearchObjectsResp{Items: mapPublicObjectItems(objects), Total: total}, nil
}

func (l *PublicLogic) countPublishedObjects(ctx context.Context, filter model.ListMapObjectsFilter) (int64, error) {
	filter.Viewport = nil
	filter.Zoom = 0
	filter.Limit = 0
	return l.store.CountPublishedObjects(ctx, filter)
}

func parseMapObjectViewportFilter(minXText, minYText, maxXText, maxYText string) (*model.MapViewportFilter, error) {
	values := []string{minXText, minYText, maxXText, maxYText}
	hasValue := false
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			hasValue = true
			break
		}
	}
	if !hasValue {
		return nil, nil
	}
	minX, err := parseViewportNumber(minXText, "视口最小 X")
	if err != nil {
		return nil, err
	}
	minY, err := parseViewportNumber(minYText, "视口最小 Y")
	if err != nil {
		return nil, err
	}
	maxX, err := parseViewportNumber(maxXText, "视口最大 X")
	if err != nil {
		return nil, err
	}
	maxY, err := parseViewportNumber(maxYText, "视口最大 Y")
	if err != nil {
		return nil, err
	}
	if minX > maxX || minY > maxY {
		return nil, errx.New(errx.CodeValidationFailed, "地图视口范围不正确，请刷新后重试")
	}
	return &model.MapViewportFilter{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}, nil
}

func parseViewportNumber(value string, label string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errx.New(errx.CodeValidationFailed, "地图视口参数不完整，请刷新后重试")
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, errx.New(errx.CodeValidationFailed, label+"格式不正确，请刷新后重试")
	}
	if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, errx.New(errx.CodeValidationFailed, label+"必须是有效数字，请刷新后重试")
	}
	return parsed, nil
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
	return ObjectDetailResp{Item: mapPublicObjectItem(object)}, nil
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

func mapPublicObjectItems(objects []model.MapObject) []MapObjectItem {
	items := make([]MapObjectItem, 0, len(objects))
	for _, object := range objects {
		items = append(items, mapPublicObjectItem(object))
	}
	return items
}

func mapMerchantPlaceItems(places []model.MerchantPlace) []MerchantPlaceItem {
	items := make([]MerchantPlaceItem, 0, len(places))
	for _, place := range places {
		object := place.Object
		claimed := strings.TrimSpace(object.MerchantID) != "" && strings.TrimSpace(object.MerchantName) != ""
		item := MerchantPlaceItem{
			ObjectId:       object.ID,
			Name:           object.Name,
			Code:           object.Code,
			CategoryCodes:  nonNilStringSlice(object.CategoryCodes),
			ServiceTags:    nonNilStringSlice(object.ServiceTags),
			PlatformTags:   nonNilStringSlice(object.PlatformTags),
			CityCode:       place.CityCode,
			MarketName:     place.MarketName,
			BuildingName:   place.SceneName,
			FloorNo:        place.FloorNo,
			Address:        object.Address,
			Claimed:        claimed,
			SourceType:     model.MerchantPlaceSourcePrelisted,
			Lat:            object.Lat,
			Lng:            object.Lng,
			DistanceText:   formatMerchantPlaceDistance(place.DistanceMeters),
			DistanceMeters: place.DistanceMeters,
		}
		if claimed {
			item.MerchantId = strings.TrimSpace(object.MerchantID)
			item.Name = strings.TrimSpace(object.MerchantName)
			item.MerchantType = strings.TrimSpace(object.MerchantType)
			item.CoverUrl = strings.TrimSpace(object.MerchantLogoURL)
			if len(object.MerchantMainCategories) > 0 {
				item.CategoryCodes = nonNilStringSlice(object.MerchantMainCategories)
			}
			item.SourceType = model.MerchantPlaceSourceClaimed
		}
		items = append(items, item)
	}
	return items
}

func formatMerchantPlaceDistance(distanceMeters int64) string {
	if distanceMeters <= 0 {
		return ""
	}
	if distanceMeters < 1000 {
		return fmt.Sprintf("%dm", distanceMeters)
	}
	return fmt.Sprintf("%.1fkm", float64(distanceMeters)/1000)
}

func mapAdminObjectItems(objects []model.MapObject) []MapObjectItem {
	items := make([]MapObjectItem, 0, len(objects))
	for _, object := range objects {
		items = append(items, mapAdminObjectItem(object))
	}
	return items
}

func mapPublicObjectItem(object model.MapObject) MapObjectItem {
	return mapObjectItem(object, false, false)
}

func mapAdminObjectItem(object model.MapObject) MapObjectItem {
	return mapObjectItem(object, true, true)
}

func mapObjectItem(object model.MapObject, includeUnverifiedMerchant bool, includeContact bool) MapObjectItem {
	verifiedMerchant := isVerifiedMapMerchant(object)
	displaySource := model.MapObjectDisplaySourceAdminObject
	displayLevel := model.MapObjectDisplayLevelWeak
	name := object.Name
	merchantID := ""
	var merchant *MapObjectMerchantItem
	if verifiedMerchant {
		displaySource = model.MapObjectDisplaySourceVerifiedMerchant
		displayLevel = model.MapObjectDisplayLevelHighlight
		name = strings.TrimSpace(object.MerchantName)
		merchantID = object.MerchantID
		merchant = mapObjectMerchantItem(object)
	} else if includeUnverifiedMerchant && strings.TrimSpace(object.MerchantID) != "" {
		merchantID = strings.TrimSpace(object.MerchantID)
		merchant = mapObjectMerchantItem(object)
	}

	item := MapObjectItem{
		Id:                 object.ID,
		SceneCode:          object.SceneCode,
		MerchantId:         merchantID,
		Code:               object.Code,
		Name:               name,
		Type:               object.Type,
		Layer:              object.Layer,
		DisplaySource:      displaySource,
		DisplayLevel:       displayLevel,
		IsVerifiedMerchant: verifiedMerchant,
		Merchant:           merchant,
		GeometryType:       object.GeometryType,
		Geometry:           map[string]interface{}(object.Geometry),
		CenterX:            formatFloat(object.CenterX),
		CenterY:            formatFloat(object.CenterY),
		MinZoom:            object.MinZoom,
		MaxZoom:            object.MaxZoom,
		CategoryCodes:      nonNilStringSlice(object.CategoryCodes),
		ServiceTags:        nonNilStringSlice(object.ServiceTags),
		PlatformTags:       nonNilStringSlice(object.PlatformTags),
		PoiServiceTags:     nonNilStringSlice(object.PoiServiceTags),
		Address:            object.Address,
		Lat:                object.Lat,
		Lng:                object.Lng,
		Extra:              map[string]interface{}(object.Extra),
		Status:             object.Status,
	}
	if includeContact {
		item.Phone = object.Phone
		item.Wechat = object.Wechat
	}
	return item
}

func isVerifiedMapMerchant(object model.MapObject) bool {
	verificationStatus := strings.TrimSpace(object.MerchantVerificationStatus)
	return strings.TrimSpace(object.MerchantID) != "" &&
		strings.TrimSpace(object.MerchantName) != "" &&
		(verificationStatus == "verified" || verificationStatus == model.MerchantProfileStatusCompleted)
}

func mapObjectMerchantItem(object model.MapObject) *MapObjectMerchantItem {
	if strings.TrimSpace(object.MerchantID) == "" {
		return nil
	}
	return &MapObjectMerchantItem{
		Id:                 strings.TrimSpace(object.MerchantID),
		Name:               strings.TrimSpace(object.MerchantName),
		MerchantType:       strings.TrimSpace(object.MerchantType),
		VerificationStatus: strings.TrimSpace(object.MerchantVerificationStatus),
		LogoUrl:            strings.TrimSpace(object.MerchantLogoURL),
		MainCategories:     nonNilStringSlice(object.MerchantMainCategories),
	}
}

// nonNilStringSlice 保持 API 中非 optional 数组的运行时契约：空集合必须编码为 []，
// 不能把数据库扫描得到的 nil slice 直接暴露成 JSON null。
func nonNilStringSlice(values []string) []string {
	items := make([]string, len(values))
	copy(items, values)
	return items
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
