package maplogic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
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
	MinX    string
	MinY    string
	MaxX    string
	MaxY    string
	Zoom    int64
}

type SaveObjectReq struct {
	Id             string
	MerchantID     string
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
		LogMapDependencyFailure(ctx, "后台查询拿货地图场景失败", "list_admin_scenes", err)
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
		LogMapDependencyFailure(ctx, "后台查询拿货地图场景详情失败", "get_admin_scene", err,
			logx.Field("sceneCode", sceneCode))
		return SceneResp{}, errx.New(errx.CodeInternalError, "地图场景加载失败，请稍后重试")
	}
	return SceneResp{Item: mapSceneItem(scene)}, nil
}

func (l *AdminLogic) SaveScene(ctx context.Context, req SaveSceneReq, operatorID string) (resp SaveSceneResp, err error) {
	operatorID = strings.TrimSpace(operatorID)
	entityID := strings.TrimSpace(req.Code)
	defer func() { logAdminMapWrite(ctx, "save_scene", operatorID, entityID, err) }()
	if operatorID == "" {
		return SaveSceneResp{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
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
		LogMapDependencyFailure(ctx, "后台保存拿货地图场景失败", "save_admin_scene", err,
			logx.Field("sceneCode", strings.TrimSpace(req.Code)), logx.Field("operatorId", operatorID))
		return SaveSceneResp{}, errx.New(errx.CodeInternalError, "地图场景保存失败，请稍后重试")
	}
	return SaveSceneResp{Item: mapSceneItem(scene)}, nil
}

func (l *AdminLogic) PublishScene(ctx context.Context, sceneCode string, operatorID string) (resp PublishSceneResp, err error) {
	sceneCode = strings.TrimSpace(sceneCode)
	operatorID = strings.TrimSpace(operatorID)
	defer func() { logAdminMapWrite(ctx, "publish_scene", operatorID, sceneCode, err) }()
	if operatorID == "" {
		return PublishSceneResp{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	if sceneCode == "" {
		return PublishSceneResp{}, errx.New(errx.CodeValidationFailed, "请选择要发布的地图场景")
	}
	scene, err := l.store.GetAdminScene(ctx, sceneCode)
	if err != nil {
		LogMapDependencyFailure(ctx, "发布前查询地图场景失败", "get_scene_for_publish", err,
			logx.Field("sceneCode", sceneCode), logx.Field("operatorId", operatorID))
		return PublishSceneResp{}, errx.New(errx.CodeInternalError, "地图场景发布失败，请稍后重试")
	}
	if err := validateScenePublishReadiness(scene); err != nil {
		logx.WithContext(ctx).Infow("地图场景发布前检查未通过",
			logx.Field("operation", "validate_scene_for_publish"), logx.Field("sceneCode", sceneCode),
			logx.Field("operatorId", operatorID), logx.Field("errorCode", errx.CodeOf(err)))
		return PublishSceneResp{}, err
	}
	// 发布前必须至少有一个可展示对象，避免小程序拿到空地图造成运营误发布。
	objects, err := l.store.ListAdminObjects(ctx, model.ListMapObjectsFilter{SceneCode: sceneCode, Status: model.MapObjectStatusNormal})
	if err != nil {
		LogMapDependencyFailure(ctx, "发布前检查地图对象失败", "list_objects_for_publish", err,
			logx.Field("sceneCode", sceneCode), logx.Field("operatorId", operatorID))
		return PublishSceneResp{}, errx.New(errx.CodeInternalError, "地图场景发布失败，请稍后重试")
	}
	if issues := mapPublishChecklistIssues(objects); len(issues) > 0 {
		logx.Infof("地图场景发布前检查未通过: sceneCode=%s issues=%s", sceneCode, strings.Join(issues, "；"))
		return PublishSceneResp{}, errx.New(errx.CodeValidationFailed, "发布前检查未通过："+strings.Join(issues, "；"))
	}
	scene, err = l.store.PublishScene(ctx, sceneCode)
	if err != nil {
		LogMapDependencyFailure(ctx, "发布拿货地图场景失败", "publish_admin_scene", err,
			logx.Field("sceneCode", sceneCode), logx.Field("operatorId", operatorID))
		return PublishSceneResp{}, errx.New(errx.CodeInternalError, "地图场景发布失败，请稍后重试")
	}
	return PublishSceneResp{Item: mapSceneItem(scene), Message: "地图场景已发布"}, nil
}

func (l *AdminLogic) ListObjects(ctx context.Context, sceneCode string, req ListAdminObjectsReq) (ListObjectsResp, error) {
	sceneCode = strings.TrimSpace(sceneCode)
	if sceneCode == "" {
		return ListObjectsResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	viewport, err := parseMapObjectViewportFilter(req.MinX, req.MinY, req.MaxX, req.MaxY)
	if err != nil {
		return ListObjectsResp{}, err
	}
	objects, err := l.store.ListAdminObjects(ctx, model.ListMapObjectsFilter{
		SceneCode: sceneCode,
		Types:     splitCSV(req.Types),
		Status:    strings.TrimSpace(req.Status),
		Keyword:   strings.TrimSpace(req.Keyword),
		Viewport:  viewport,
		Zoom:      req.Zoom,
	})
	if err != nil {
		LogMapDependencyFailure(ctx, "后台查询地图点位失败", "list_admin_objects", err,
			logx.Field("sceneCode", strings.TrimSpace(sceneCode)), logx.Field("zoom", req.Zoom))
		return ListObjectsResp{}, errx.New(errx.CodeInternalError, "地图点位加载失败，请稍后重试")
	}
	return ListObjectsResp{SceneCode: sceneCode, Items: mapAdminObjectItems(objects), Total: int64(len(objects))}, nil
}

func (l *AdminLogic) SaveObject(ctx context.Context, sceneCode string, req SaveObjectReq, operatorID string) (resp SaveObjectResp, err error) {
	sceneCode = strings.TrimSpace(sceneCode)
	objectID := strings.TrimSpace(req.Id)
	operatorID = strings.TrimSpace(operatorID)
	entityID := objectID
	if entityID == "" {
		entityID = strings.TrimSpace(req.Code)
	}
	defer func() { logAdminMapWrite(ctx, "save_object", operatorID, entityID, err) }()
	if operatorID == "" {
		return SaveObjectResp{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
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
		MerchantID:     strings.TrimSpace(req.MerchantID),
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
		LogMapDependencyFailure(ctx, "后台保存地图点位失败", "save_admin_object", err,
			logx.Field("sceneCode", strings.TrimSpace(sceneCode)), logx.Field("objectId", strings.TrimSpace(req.Id)),
			logx.Field("operatorId", operatorID))
		return SaveObjectResp{}, errx.New(errx.CodeInternalError, "地图点位保存失败，请稍后重试")
	}
	return SaveObjectResp{Item: mapAdminObjectItem(object)}, nil
}

func (l *AdminLogic) UpdateObjectStatus(ctx context.Context, objectID string, req UpdateObjectStatusReq, operatorID string) (resp SaveObjectResp, err error) {
	objectID = strings.TrimSpace(objectID)
	status := strings.TrimSpace(req.Status)
	operatorID = strings.TrimSpace(operatorID)
	defer func() { logAdminMapWrite(ctx, "update_object_status", operatorID, objectID, err) }()
	if operatorID == "" {
		return SaveObjectResp{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	if objectID == "" {
		return SaveObjectResp{}, errx.New(errx.CodeValidationFailed, "请选择地图点位")
	}
	if !validObjectStatus(status) {
		return SaveObjectResp{}, errx.New(errx.CodeValidationFailed, "地图点位状态不正确")
	}
	object, err := l.store.UpdateObjectStatus(ctx, objectID, status)
	if err != nil {
		LogMapDependencyFailure(ctx, "后台更新地图点位状态失败", "update_admin_object_status", err,
			logx.Field("objectId", objectID), logx.Field("status", status), logx.Field("operatorId", operatorID))
		return SaveObjectResp{}, errx.New(errx.CodeInternalError, "地图点位状态保存失败，请稍后重试")
	}
	return SaveObjectResp{Item: mapAdminObjectItem(object)}, nil
}

func (l *AdminLogic) BatchGenerateObjects(ctx context.Context, sceneCode string, req BatchGenerateObjectsReq, operatorID string) (resp BatchGenerateObjectsResp, err error) {
	sceneCode = strings.TrimSpace(sceneCode)
	operatorID = strings.TrimSpace(operatorID)
	defer func() { logAdminMapWrite(ctx, "batch_generate_objects", operatorID, sceneCode, err) }()
	if operatorID == "" {
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	if sceneCode == "" {
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeValidationFailed, "请选择地图场景")
	}
	scene, err := l.store.GetAdminScene(ctx, sceneCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BatchGenerateObjectsResp{}, errx.New(errx.CodeResourceNotFound, "地图场景不存在或已删除")
		}
		LogMapDependencyFailure(ctx, "批量生成前查询地图场景失败", "get_scene_for_batch_generate", err,
			logx.Field("sceneCode", sceneCode), logx.Field("operatorId", operatorID))
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeInternalError, "批量生成地图点位失败，请稍后重试")
	}
	if scene.Width <= 0 || scene.Height <= 0 {
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeValidationFailed, "地图底图宽高必须大于 0，请先完善场景底图尺寸")
	}
	inputs, err := buildBatchObjectInputs(sceneCode, req)
	if err != nil {
		return BatchGenerateObjectsResp{}, err
	}
	inputs, skippedCount, err := filterBatchObjectInputsBySceneBounds(inputs, scene)
	if err != nil {
		return BatchGenerateObjectsResp{}, err
	}
	if len(inputs) == 0 {
		logx.Infof("批量生成地图点位全部越界: sceneCode=%s count=%d sceneWidth=%d sceneHeight=%d", sceneCode, req.Count, scene.Width, scene.Height)
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeValidationFailed, "批量生成的点位均超出地图范围，请调整起点、数量、尺寸或间距")
	}
	if skippedCount > 0 {
		logx.Infof("批量生成地图点位跳过越界点位: sceneCode=%s requested=%d generated=%d skipped=%d sceneWidth=%d sceneHeight=%d", sceneCode, req.Count, len(inputs), skippedCount, scene.Width, scene.Height)
	}
	objects, err := l.store.BatchCreateObjects(ctx, inputs)
	if err != nil {
		LogMapDependencyFailure(ctx, "后台批量生成地图点位失败", "batch_generate_admin_objects", err,
			logx.Field("sceneCode", sceneCode), logx.Field("count", req.Count), logx.Field("operatorId", operatorID))
		return BatchGenerateObjectsResp{}, errx.New(errx.CodeInternalError, "批量生成地图点位失败，请稍后重试")
	}
	return BatchGenerateObjectsResp{Items: mapAdminObjectItems(objects)}, nil
}

func (l *AdminLogic) ListCategories(ctx context.Context, req ListCategoriesReq) (ListCategoriesResp, error) {
	filter := model.ListMapCategoriesFilter{
		Type:   strings.TrimSpace(req.Type),
		Status: strings.TrimSpace(req.Status),
	}
	categories, err := l.store.ListCategories(ctx, filter)
	if err != nil {
		LogMapDependencyFailure(ctx, "后台查询地图分类失败", "list_admin_categories", err,
			logx.Field("status", strings.TrimSpace(req.Status)))
		return ListCategoriesResp{}, errx.New(errx.CodeInternalError, "地图分类加载失败，请稍后重试")
	}
	return ListCategoriesResp{Items: mapCategoryItems(categories)}, nil
}

func (l *AdminLogic) SaveCategory(ctx context.Context, req SaveCategoryReq, operatorID string) (resp SaveCategoryResp, err error) {
	operatorID = strings.TrimSpace(operatorID)
	entityID := strings.TrimSpace(req.Code)
	defer func() { logAdminMapWrite(ctx, "save_category", operatorID, entityID, err) }()
	if operatorID == "" {
		return SaveCategoryResp{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
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
		LogMapDependencyFailure(ctx, "后台保存地图分类失败", "save_admin_category", err,
			logx.Field("categoryCode", strings.TrimSpace(req.Code)), logx.Field("operatorId", operatorID))
		return SaveCategoryResp{}, errx.New(errx.CodeInternalError, "地图分类保存失败，请稍后重试")
	}
	return SaveCategoryResp{Item: mapCategoryItem(category)}, nil
}

// logAdminMapWrite 统一记录地图后台写操作归因。只写可信管理员、动作、实体和错误类型，
// 不记录请求正文、坐标、联系方式或审核内容，避免审计日志自身泄露业务敏感信息。
func logAdminMapWrite(ctx context.Context, operation string, operatorID string, entityID string, err error) {
	fields := []logx.LogField{
		logx.Field("operatorId", strings.TrimSpace(operatorID)),
		logx.Field("operation", strings.TrimSpace(operation)),
		logx.Field("entityId", strings.TrimSpace(entityID)),
	}
	if err != nil {
		fields = append(fields,
			logx.Field("errorCode", errx.CodeOf(err)),
			logx.Field("errorType", fmt.Sprintf("%T", err)))
		logx.WithContext(ctx).Errorw("地图后台写操作失败", fields...)
		return
	}
	logx.WithContext(ctx).Infow("地图后台写操作成功", fields...)
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
	if err := validateObjectZoomRange(req.MinZoom, req.MaxZoom); err != nil {
		return err
	}
	if merchantID := strings.TrimSpace(req.MerchantID); merchantID != "" {
		if _, err := strconv.ParseInt(merchantID, 10, 64); err != nil {
			return errx.New(errx.CodeValidationFailed, "绑定商家不正确，请重新选择")
		}
	}
	if !validObjectStatus(status) {
		return errx.New(errx.CodeValidationFailed, "地图点位状态不正确")
	}
	return nil
}

func validateObjectZoomRange(minZoom int64, maxZoom int64) error {
	if minZoom <= 0 {
		minZoom = 1
	}
	if maxZoom <= 0 {
		maxZoom = 5
	}
	if minZoom < 1 || minZoom > 5 || maxZoom < 1 || maxZoom > 5 {
		return errx.New(errx.CodeValidationFailed, "显示级别必须在 1 到 5 之间")
	}
	if minZoom > maxZoom {
		return errx.New(errx.CodeValidationFailed, "最小显示级别不能大于最大显示级别")
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

func filterBatchObjectInputsBySceneBounds(inputs []model.MapObjectInput, scene model.MapScene) ([]model.MapObjectInput, int, error) {
	validInputs := make([]model.MapObjectInput, 0, len(inputs))
	skippedCount := 0
	for _, input := range inputs {
		fields, err := model.BuildMapObjectDerivedFields(input)
		if err != nil {
			return nil, 0, errx.New(errx.CodeValidationFailed, "批量生成的点位位置不正确，请检查起点、尺寸和间距")
		}
		// 地图对象坐标必须完整落在底图内，避免发布后出现看不见或无法操作的越界点位。
		if fields.MinX < 0 || fields.MinY < 0 || fields.MaxX > float64(scene.Width) || fields.MaxY > float64(scene.Height) {
			skippedCount++
			continue
		}
		validInputs = append(validInputs, input)
	}
	return validInputs, skippedCount, nil
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
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errx.New(errx.CodeValidationFailed, "请填写"+fieldName)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, errx.New(errx.CodeValidationFailed, fieldName+"格式不正确")
	}
	if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, errx.New(errx.CodeValidationFailed, fieldName+"必须是有效数字")
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

func validateScenePublishReadiness(scene model.MapScene) error {
	if strings.TrimSpace(scene.BackgroundURL) == "" {
		return errx.New(errx.CodeValidationFailed, "发布前检查未通过：请先配置地图底图")
	}
	if scene.Width <= 0 || scene.Height <= 0 {
		return errx.New(errx.CodeValidationFailed, "发布前检查未通过：请先配置地图底图宽高")
	}
	return nil
}

func mapPublishChecklistIssues(objects []model.MapObject) []string {
	if len(objects) == 0 {
		return []string{"请先标注至少一个正常状态的点位"}
	}

	var invalidGeometryCount int
	var missingContactCount int
	var missingTagCount int
	for _, object := range objects {
		if object.Status != "" && object.Status != model.MapObjectStatusNormal {
			continue
		}
		if !hasPublishReadyGeometry(object) {
			invalidGeometryCount++
		}
		if requiresPublishReadyContact(object) && !hasPublishReadyContact(object) {
			missingContactCount++
		}
		if !hasPublishReadyTags(object) {
			missingTagCount++
		}
	}

	issues := make([]string, 0, 3)
	if invalidGeometryCount > 0 {
		issues = append(issues, fmt.Sprintf("请补齐 %d 个点位的地图坐标", invalidGeometryCount))
	}
	if missingContactCount > 0 {
		issues = append(issues, fmt.Sprintf("请补齐 %d 个档口的电话或微信", missingContactCount))
	}
	if missingTagCount > 0 {
		issues = append(issues, fmt.Sprintf("请补齐 %d 个点位的分类或服务标签", missingTagCount))
	}
	return issues
}

func hasPublishReadyGeometry(object model.MapObject) bool {
	fields, err := model.BuildMapObjectDerivedFields(model.MapObjectInput{
		Code:         object.Code,
		Name:         object.Name,
		Type:         object.Type,
		GeometryType: object.GeometryType,
		Geometry:     object.Geometry,
	})
	if err != nil {
		return false
	}
	return fields.MinX >= 0 && fields.MinY >= 0 && fields.MaxX >= fields.MinX && fields.MaxY >= fields.MinY
}

func hasPublishReadyTags(object model.MapObject) bool {
	if strings.TrimSpace(object.Layer) == "poi" {
		return len(cleanStringSlice(object.PoiServiceTags)) > 0
	}
	return len(cleanStringSlice(object.CategoryCodes)) > 0 &&
		(len(cleanStringSlice(object.ServiceTags)) > 0 || len(cleanStringSlice(object.PlatformTags)) > 0)
}

func requiresPublishReadyContact(object model.MapObject) bool {
	// 配套 POI 主要用于地图引导，第一期不强制补联系电话；档口仍需电话或微信，保证用户能发起联系。
	return strings.TrimSpace(object.Layer) != "poi"
}

func hasPublishReadyContact(object model.MapObject) bool {
	return strings.TrimSpace(object.Phone) != "" || strings.TrimSpace(object.Wechat) != ""
}

func validSceneStatus(status string) bool {
	return status == model.MapSceneStatusDraft || status == model.MapSceneStatusPublished || status == model.MapSceneStatusArchived
}

func validObjectStatus(status string) bool {
	return status == model.MapObjectStatusNormal || status == model.MapObjectStatusHidden || status == model.MapObjectStatusClosed
}

func validGeometryType(geometryType string) bool {
	return geometryType == model.MapGeometryTypeRect || geometryType == model.MapGeometryTypePoint || geometryType == model.MapGeometryTypePolygon
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
