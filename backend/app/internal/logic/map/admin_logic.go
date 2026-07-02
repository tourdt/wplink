package maplogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminStore interface {
	ListAdminScenes(ctx context.Context, filter model.ListMapScenesFilter) ([]model.MapScene, error)
	GetAdminScene(ctx context.Context, sceneCode string) (model.MapScene, error)
	SaveScene(ctx context.Context, input model.MapSceneInput) (model.MapScene, error)
	PublishScene(ctx context.Context, sceneCode string) (model.MapScene, error)
	ListAdminObjects(ctx context.Context, filter model.ListMapObjectsFilter) ([]model.MapObject, error)
	SaveObject(ctx context.Context, input model.MapObjectInput) (model.MapObject, error)
	UpdateObjectStatus(ctx context.Context, objectID string, status string) (model.MapObject, error)
	BatchCreateObjects(ctx context.Context, inputs []model.MapObjectInput) ([]model.MapObject, error)
	ListCategories(ctx context.Context, filter model.ListMapCategoriesFilter) ([]model.MapCategory, error)
	SaveCategory(ctx context.Context, input model.MapCategoryInput) (model.MapCategory, error)
}

type AdminLogic struct {
	store AdminStore
}

func NewAdminLogic(store AdminStore) *AdminLogic {
	return &AdminLogic{store: store}
}

type ListAdminScenesReq struct {
	CityCode string
	Status   string
	Type     string
}

type SaveSceneReq struct {
	CityCode       string
	Code           string
	Name           string
	Type           string
	ParentCode     string
	BackgroundUrl  string
	Width          int64
	Height         int64
	MinScale       string
	MaxScale       string
	DefaultScale   string
	DefaultCenterX string
	DefaultCenterY string
	FloorNo        string
	Sort           int64
	Status         string
}

type SaveSceneResp struct {
	Item MapSceneItem `json:"item"`
}

type PublishSceneResp struct {
	Item    MapSceneItem `json:"item"`
	Message string       `json:"message"`
}

type ListAdminObjectsReq struct {
	Types   string
	Status  string
	Keyword string
}

type SaveObjectReq struct {
	Id             string
	Code           string
	Name           string
	Type           string
	Layer          string
	GeometryType   string
	Geometry       map[string]interface{}
	MinZoom        int64
	MaxZoom        int64
	CategoryCodes  []string
	ServiceTags    []string
	PlatformTags   []string
	PoiServiceTags []string
	Address        string
	Phone          string
	Wechat         string
	Lat            string
	Lng            string
	Extra          map[string]interface{}
	Sort           int64
	Status         string
}

type SaveObjectResp struct {
	Item MapObjectItem `json:"item"`
}

type UpdateObjectStatusReq struct {
	Status string
}

type BatchGenerateObjectsReq struct {
	StartCode     string
	Count         int64
	Direction     string
	StartX        string
	StartY        string
	Width         string
	Height        string
	Gap           string
	Type          string
	Layer         string
	CategoryCodes []string
	ServiceTags   []string
}

type BatchGenerateObjectsResp struct {
	Items []MapObjectItem `json:"items"`
}

type ListCategoriesReq struct {
	Type   string
	Status string
}

type ListCategoriesResp struct {
	Items []MapCategoryItem `json:"items"`
}

type MapCategoryItem struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	IconUrl   string `json:"iconUrl,omitempty"`
	Sort      int64  `json:"sort"`
	IsVisible bool   `json:"isVisible"`
	Status    string `json:"status"`
}

type SaveCategoryReq struct {
	Code      string
	Name      string
	Type      string
	IconUrl   string
	Sort      int64
	IsVisible bool
	Status    string
}

type SaveCategoryResp struct {
	Item MapCategoryItem `json:"item"`
}

func (l *AdminLogic) ListScenes(ctx context.Context, req ListAdminScenesReq) (ListScenesResp, error) {
	scenes, err := l.store.ListAdminScenes(ctx, model.ListMapScenesFilter{
		CityCode: strings.TrimSpace(req.CityCode),
		Status:   strings.TrimSpace(req.Status),
		Type:     strings.TrimSpace(req.Type),
	})
	if err != nil {
		logx.Errorf("后台查询拿货地图场景失败: cityCode=%s status=%s type=%s err=%+v", req.CityCode, req.Status, req.Type, err)
		return ListScenesResp{}, errx.New(errx.CodeInternalError, "地图场景加载失败，请稍后重试")
	}
	return ListScenesResp{Items: mapSceneItems(scenes)}, nil
}

func (l *AdminLogic) GetScene(ctx context.Context, sceneCode string) (SceneResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	if sceneCode == "" {
		return SceneResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	scene, err := l.store.GetAdminScene(ctx, sceneCode)
	if err != nil {
		logx.Errorf("后台查询拿货地图场景详情失败: sceneCode=%s err=%+v", sceneCode, err)
		return SceneResp{}, errx.New(errx.CodeInternalError, "地图场景加载失败，请稍后重试")
	}
	return SceneResp{Item: mapSceneItem(scene)}, nil
}

func (l *AdminLogic) SaveScene(ctx context.Context, req SaveSceneReq) (SaveSceneResp, error) {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = model.MapSceneStatusDraft
	}
	if err := validateSceneInput(req, status); err != nil {
		return SaveSceneResp{}, err
	}
	scene, err := l.store.SaveScene(ctx, model.MapSceneInput{
		CityCode:       strings.TrimSpace(req.CityCode),
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		Type:           strings.TrimSpace(req.Type),
		ParentCode:     strings.TrimSpace(req.ParentCode),
		BackgroundURL:  strings.TrimSpace(req.BackgroundUrl),
		Width:          req.Width,
		Height:         req.Height,
		MinScale:       strings.TrimSpace(req.MinScale),
		MaxScale:       strings.TrimSpace(req.MaxScale),
		DefaultScale:   strings.TrimSpace(req.DefaultScale),
		DefaultCenterX: strings.TrimSpace(req.DefaultCenterX),
		DefaultCenterY: strings.TrimSpace(req.DefaultCenterY),
		FloorNo:        strings.TrimSpace(req.FloorNo),
		Sort:           req.Sort,
		Status:         status,
	})
	if err != nil {
		logx.Errorf("后台保存拿货地图场景失败: code=%s name=%s err=%+v", req.Code, req.Name, err)
		return SaveSceneResp{}, errx.New(errx.CodeInternalError, "地图场景保存失败，请稍后重试")
	}
	return SaveSceneResp{Item: mapSceneItem(scene)}, nil
}

func (l *AdminLogic) PublishScene(ctx context.Context, sceneCode string) (PublishSceneResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	if sceneCode == "" {
		return PublishSceneResp{}, errx.New(errx.CodeValidationFailed, "请选择要发布的地图场景")
	}
	// 发布前必须至少有一个可展示对象，避免小程序拿到空地图造成运营误发布。
	objects, err := l.store.ListAdminObjects(ctx, model.ListMapObjectsFilter{SceneCode: sceneCode, Status: model.MapObjectStatusNormal})
	if err != nil {
		logx.Errorf("发布前检查地图对象失败: sceneCode=%s err=%+v", sceneCode, err)
		return PublishSceneResp{}, errx.New(errx.CodeInternalError, "地图场景发布失败，请稍后重试")
	}
	if len(objects) == 0 {
		return PublishSceneResp{}, errx.New(errx.CodeValidationFailed, "请先标注至少一个地图点位后再发布")
	}
	scene, err := l.store.PublishScene(ctx, sceneCode)
	if err != nil {
		logx.Errorf("发布拿货地图场景失败: sceneCode=%s err=%+v", sceneCode, err)
		return PublishSceneResp{}, errx.New(errx.CodeInternalError, "地图场景发布失败，请稍后重试")
	}
	return PublishSceneResp{Item: mapSceneItem(scene), Message: "地图场景已发布"}, nil
}

func (l *AdminLogic) ListObjects(ctx context.Context, sceneCode string, req ListAdminObjectsReq) (ListObjectsResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	if sceneCode == "" {
		return ListObjectsResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	objects, err := l.store.ListAdminObjects(ctx, model.ListMapObjectsFilter{
		SceneCode: sceneCode,
		Types:     splitCSV(req.Types),
		Status:    strings.TrimSpace(req.Status),
		Keyword:   strings.TrimSpace(req.Keyword),
	})
	if err != nil {
		logx.Errorf("后台查询地图点位失败: sceneCode=%s err=%+v", sceneCode, err)
		return ListObjectsResp{}, errx.New(errx.CodeInternalError, "地图点位加载失败，请稍后重试")
	}
	return ListObjectsResp{SceneCode: sceneCode, Items: mapObjectItems(objects)}, nil
}

func (l *AdminLogic) SaveObject(ctx context.Context, sceneCode string, req SaveObjectReq) (SaveObjectResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	objectID := strings.TrimSpace(req.Id)
	if sceneCode == "" && objectID == "" {
		return SaveObjectResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = model.MapObjectStatusNormal
	}
	if err := validateObjectInput(req, status); err != nil {
		return SaveObjectResp{}, err
	}
	object, err := l.store.SaveObject(ctx, model.MapObjectInput{
		ID:             objectID,
		SceneCode:      sceneCode,
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		Type:           strings.TrimSpace(req.Type),
		Layer:          strings.TrimSpace(req.Layer),
		GeometryType:   strings.TrimSpace(req.GeometryType),
		Geometry:       model.JSONMap(req.Geometry),
		MinZoom:        req.MinZoom,
		MaxZoom:        req.MaxZoom,
		CategoryCodes:  cleanStringSlice(req.CategoryCodes),
		ServiceTags:    cleanStringSlice(req.ServiceTags),
		PlatformTags:   cleanStringSlice(req.PlatformTags),
		PoiServiceTags: cleanStringSlice(req.PoiServiceTags),
		Address:        strings.TrimSpace(req.Address),
		Phone:          strings.TrimSpace(req.Phone),
		Wechat:         strings.TrimSpace(req.Wechat),
		Lat:            strings.TrimSpace(req.Lat),
		Lng:            strings.TrimSpace(req.Lng),
		Extra:          model.JSONMap(req.Extra),
		Sort:           req.Sort,
		Status:         status,
	})
	if err != nil {
		logx.Errorf("后台保存地图点位失败: sceneCode=%s code=%s err=%+v", sceneCode, req.Code, err)
		return SaveObjectResp{}, errx.New(errx.CodeInternalError, "地图点位保存失败，请稍后重试")
	}
	return SaveObjectResp{Item: mapObjectItem(object)}, nil
}

func (l *AdminLogic) UpdateObjectStatus(ctx context.Context, objectID string, req UpdateObjectStatusReq) (SaveObjectResp, error) {
	objectID = strings.TrimSpace(objectID)
	status := strings.TrimSpace(req.Status)
	if objectID == "" {
		return SaveObjectResp{}, errx.New(errx.CodeValidationFailed, "请选择地图点位")
	}
	if !validObjectStatus(status) {
		return SaveObjectResp{}, errx.New(errx.CodeValidationFailed, "地图点位状态不正确")
	}
	object, err := l.store.UpdateObjectStatus(ctx, objectID, status)
	if err != nil {
		logx.Errorf("后台更新地图点位状态失败: objectID=%s status=%s err=%+v", objectID, status, err)
		return SaveObjectResp{}, errx.New(errx.CodeInternalError, "地图点位状态保存失败，请稍后重试")
	}
	return SaveObjectResp{Item: mapObjectItem(object)}, nil
}

func (l *AdminLogic) BatchGenerateObjects(ctx context.Context, sceneCode string, req BatchGenerateObjectsReq) (BatchGenerateObjectsResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	if sceneCode == "" {
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	inputs, err := buildBatchObjectInputs(sceneCode, req)
	if err != nil {
		return BatchGenerateObjectsResp{}, err
	}
	objects, err := l.store.BatchCreateObjects(ctx, inputs)
	if err != nil {
		logx.Errorf("后台批量生成地图点位失败: sceneCode=%s startCode=%s count=%d err=%+v", sceneCode, req.StartCode, req.Count, err)
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeInternalError, "批量生成地图点位失败，请稍后重试")
	}
	return BatchGenerateObjectsResp{Items: mapObjectItems(objects)}, nil
}

func (l *AdminLogic) ListCategories(ctx context.Context, req ListCategoriesReq) (ListCategoriesResp, error) {
	filter := model.ListMapCategoriesFilter{
		Type:   strings.TrimSpace(req.Type),
		Status: strings.TrimSpace(req.Status),
	}
	categories, err := l.store.ListCategories(ctx, filter)
	if err != nil {
		logx.Errorf("后台查询地图分类失败: type=%s status=%s err=%+v", req.Type, req.Status, err)
		return ListCategoriesResp{}, errx.New(errx.CodeInternalError, "地图分类加载失败，请稍后重试")
	}
	return ListCategoriesResp{Items: mapCategoryItems(categories)}, nil
}

func (l *AdminLogic) SaveCategory(ctx context.Context, req SaveCategoryReq) (SaveCategoryResp, error) {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = model.MapCategoryStatusNormal
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
		return SaveCategoryResp{}, errx.New(errx.CodeValidationFailed, "请填写分类编码、名称和类型")
	}
	category, err := l.store.SaveCategory(ctx, model.MapCategoryInput{
		Code:      strings.TrimSpace(req.Code),
		Name:      strings.TrimSpace(req.Name),
		Type:      strings.TrimSpace(req.Type),
		IconURL:   strings.TrimSpace(req.IconUrl),
		Sort:      req.Sort,
		IsVisible: req.IsVisible,
		Status:    status,
	})
	if err != nil {
		logx.Errorf("后台保存地图分类失败: code=%s type=%s err=%+v", req.Code, req.Type, err)
		return SaveCategoryResp{}, errx.New(errx.CodeInternalError, "地图分类保存失败，请稍后重试")
	}
	return SaveCategoryResp{Item: mapCategoryItem(category)}, nil
}

func validateSceneInput(req SaveSceneReq, status string) error {
	if strings.TrimSpace(req.Code) == "" {
		return errx.New(errx.CodeValidationFailed, "请填写地图场景编码")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errx.New(errx.CodeValidationFailed, "请填写地图场景名称")
	}
	if strings.TrimSpace(req.Type) == "" {
		return errx.New(errx.CodeValidationFailed, "请选择地图场景类型")
	}
	if strings.TrimSpace(req.BackgroundUrl) == "" {
		return errx.New(errx.CodeValidationFailed, "请上传地图底图")
	}
	if req.Width <= 0 || req.Height <= 0 {
		return errx.New(errx.CodeValidationFailed, "地图底图宽高必须大于 0")
	}
	if !validSceneStatus(status) {
		return errx.New(errx.CodeValidationFailed, "地图场景状态不正确")
	}
	return nil
}

func validateObjectInput(req SaveObjectReq, status string) error {
	if strings.TrimSpace(req.Code) == "" {
		return errx.New(errx.CodeValidationFailed, "请填写地图点位编码")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errx.New(errx.CodeValidationFailed, "请填写地图点位名称")
	}
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Layer) == "" {
		return errx.New(errx.CodeValidationFailed, "请选择地图点位类型和图层")
	}
	if !validGeometryType(req.GeometryType) {
		return errx.New(errx.CodeValidationFailed, "地图点位形状不支持")
	}
	if len(req.Geometry) == 0 {
		return errx.New(errx.CodeValidationFailed, "请标注地图点位位置")
	}
	if !validObjectStatus(status) {
		return errx.New(errx.CodeValidationFailed, "地图点位状态不正确")
	}
	return nil
}

func buildBatchObjectInputs(sceneCode string, req BatchGenerateObjectsReq) ([]model.MapObjectInput, error) {
	if req.Count <= 0 || req.Count > 200 {
		return nil, errx.New(errx.CodeValidationFailed, "批量生成数量必须在 1 到 200 之间")
	}
	direction := strings.TrimSpace(req.Direction)
	if direction == "" {
		direction = "horizontal"
	}
	if direction != "horizontal" && direction != "vertical" {
		return nil, errx.New(errx.CodeValidationFailed, "批量生成方向不正确")
	}
	startNumber, widthDigits, prefix, err := parseCodeSeed(req.StartCode)
	if err != nil {
		return nil, err
	}
	startX, err := parseRequiredFloat(req.StartX, "起始 X 坐标")
	if err != nil {
		return nil, err
	}
	startY, err := parseRequiredFloat(req.StartY, "起始 Y 坐标")
	if err != nil {
		return nil, err
	}
	width, err := parseRequiredFloat(req.Width, "档口宽度")
	if err != nil {
		return nil, err
	}
	height, err := parseRequiredFloat(req.Height, "档口高度")
	if err != nil {
		return nil, err
	}
	gap, err := parseRequiredFloat(req.Gap, "档口间距")
	if err != nil {
		return nil, err
	}
	if width <= 0 || height <= 0 || gap < 0 {
		return nil, errx.New(errx.CodeValidationFailed, "档口宽高必须大于 0，间距不能为负数")
	}
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Layer) == "" {
		return nil, errx.New(errx.CodeValidationFailed, "请选择批量生成的点位类型和图层")
	}

	inputs := make([]model.MapObjectInput, 0, req.Count)
	for i := int64(0); i < req.Count; i++ {
		x, y := startX, startY
		if direction == "horizontal" {
			x = startX + float64(i)*(width+gap)
		} else {
			y = startY + float64(i)*(height+gap)
		}
		code := fmt.Sprintf("%s%0*d", prefix, widthDigits, startNumber+i)
		inputs = append(inputs, model.MapObjectInput{
			SceneCode:      sceneCode,
			Code:           code,
			Name:           code,
			Type:           strings.TrimSpace(req.Type),
			Layer:          strings.TrimSpace(req.Layer),
			GeometryType:   model.MapGeometryTypeRect,
			Geometry:       model.JSONMap{"x": x, "y": y, "width": width, "height": height},
			MinZoom:        3,
			MaxZoom:        5,
			CategoryCodes:  cleanStringSlice(req.CategoryCodes),
			ServiceTags:    cleanStringSlice(req.ServiceTags),
			PlatformTags:   nil,
			PoiServiceTags: nil,
			Status:         model.MapObjectStatusNormal,
		})
	}
	return inputs, nil
}

func parseCodeSeed(code string) (int64, int, string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, 0, "", errx.New(errx.CodeValidationFailed, "请填写起始编号")
	}
	runes := []rune(code)
	split := len(runes)
	for split > 0 && unicode.IsDigit(runes[split-1]) {
		split--
	}
	if split == len(runes) {
		return 0, 0, "", errx.New(errx.CodeValidationFailed, "起始编号必须以数字结尾")
	}
	prefix := string(runes[:split])
	numberText := string(runes[split:])
	number, err := strconv.ParseInt(numberText, 10, 64)
	if err != nil {
		return 0, 0, "", errx.New(errx.CodeValidationFailed, "起始编号格式不正确")
	}
	return number, len(numberText), prefix, nil
}

func parseRequiredFloat(value string, fieldName string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, errx.New(errx.CodeValidationFailed, fieldName+"格式不正确")
	}
	return parsed, nil
}

func cleanStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func validSceneStatus(status string) bool {
	return status == model.MapSceneStatusDraft || status == model.MapSceneStatusPublished || status == model.MapSceneStatusArchived
}

func validObjectStatus(status string) bool {
	return status == model.MapObjectStatusNormal || status == model.MapObjectStatusHidden || status == model.MapObjectStatusClosed
}

func validGeometryType(geometryType string) bool {
	return geometryType == model.MapGeometryTypeRect || geometryType == model.MapGeometryTypePoint
}

func mapCategoryItems(categories []model.MapCategory) []MapCategoryItem {
	items := make([]MapCategoryItem, 0, len(categories))
	for _, category := range categories {
		items = append(items, mapCategoryItem(category))
	}
	return items
}

func mapCategoryItem(category model.MapCategory) MapCategoryItem {
	return MapCategoryItem{
		Code:      category.Code,
		Name:      category.Name,
		Type:      category.Type,
		IconUrl:   category.IconURL,
		Sort:      category.Sort,
		IsVisible: category.IsVisible,
		Status:    category.Status,
	}
}
