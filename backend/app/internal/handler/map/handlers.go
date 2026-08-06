package maphandler

import (
	"context"
	"fmt"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	authlogic "wplink/backend/app/internal/logic/auth"
	maplogic "wplink/backend/app/internal/logic/map"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// mapStore 汇总现有地图 Logic 所需的数据能力。Handler 只负责 HTTP、身份与 DTO 映射，
// 地图发布、点位生成、绑定冲突和举报规则仍由对应 Logic 维护。
type mapStore interface {
	maplogic.PublicStore
	maplogic.AdminStore
	maplogic.BindingStore
	maplogic.ReportStore
}

func listMerchantPlacesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMerchantPlacesReq
		if err := parseMapRequest(r, &req, "商家地图筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).ListMerchantPlaces(r.Context(), maplogic.ListMerchantPlacesReq{
			CityCode: req.CityCode, Keyword: req.Keyword, Categories: req.Categories,
			MerchantTypes: req.MerchantTypes, Claimed: req.Claimed, Page: req.Page, PageSize: req.PageSize,
			MinLat: req.MinLat, MaxLat: req.MaxLat, MinLng: req.MinLng, MaxLng: req.MaxLng, Lat: req.Lat, Lng: req.Lng,
		})
		response.JSON(w, resp, err)
	}
}

func getMerchantLocationContextHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).GetMerchantLocationContext(r.Context(), pathvar.Vars(r)["merchantId"])
		response.JSON(w, resp, err)
	}
}

func listMapScenesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMapScenesReq
		if err := parseMapRequest(r, &req, "地图场景筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).ListScenes(r.Context(), maplogic.ListScenesReq{
			CityCode: req.CityCode, ParentCode: req.ParentCode, Type: req.Type,
		})
		response.JSON(w, resp, err)
	}
}

func getMapSceneHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).GetScene(r.Context(), pathvar.Vars(r)["sceneCode"])
		response.JSON(w, resp, err)
	}
}

func listMapObjectsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMapObjectsReq
		if err := parseMapRequest(r, &req, "地图视野筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).ListObjects(r.Context(), pathvar.Vars(r)["sceneCode"], maplogic.ListObjectsReq{
			Types: req.Types, Categories: req.Categories, ServiceTags: req.ServiceTags,
			PoiServiceTags: req.PoiServiceTags, Keyword: req.Keyword,
			MinX: req.MinX, MinY: req.MinY, MaxX: req.MaxX, MaxY: req.MaxY, Zoom: req.Zoom,
		})
		response.JSON(w, resp, err)
	}
}

func searchMapObjectsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SearchMapObjectsReq
		if err := parseMapRequest(r, &req, "地图搜索参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).SearchObjects(r.Context(), maplogic.SearchObjectsReq{
			SceneCode: req.SceneCode, Keyword: req.Keyword, Types: req.Types, Categories: req.Categories,
			ServiceTags: req.ServiceTags, PoiServiceTags: req.PoiServiceTags,
			MinX: req.MinX, MinY: req.MinY, MaxX: req.MaxX, MaxY: req.MaxY, Zoom: req.Zoom, Limit: req.Limit,
		})
		response.JSON(w, resp, err)
	}
}

func getMapObjectHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).GetObject(r.Context(), pathvar.Vars(r)["objectId"])
		response.JSON(w, resp, err)
	}
}

func listNearbyPoisHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListNearbyPoisReq
		if err := parseMapRequest(r, &req, "附近配套筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).ListNearbyPois(r.Context(), pathvar.Vars(r)["objectId"], maplogic.ListNearbyPoisReq{
			Types: req.Types, Limit: req.Limit,
		})
		response.JSON(w, resp, err)
	}
}

func submitMapObjectReportHTTPHandler(svcCtx *svc.ServiceContext, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		user, err := handlerx.RequiredUser(r, userTokenServiceFromMapContext(svcCtx))
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SubmitMapObjectReportReq
		if err := parseMapRequest(r, &req, "地图反馈内容格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		// 举报归属只能来自服务端验证后的 Token；query/body 中的同名字段不会参与映射。
		resp, err := maplogic.NewReportLogic(store).Submit(r.Context(), pathvar.Vars(r)["objectId"], kind, maplogic.SubmitMapObjectReportReq{
			ReporterUserID: user.UserID, ReasonCode: req.ReasonCode, Description: req.Description,
		})
		response.JSON(w, resp, err)
	}
}

func listMapCategoriesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMapCategoriesReq
		if err := parseMapRequest(r, &req, "地图分类筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewPublicLogic(store).ListCategories(r.Context(), maplogic.ListCategoriesReq{Type: req.Type})
		response.JSON(w, resp, err)
	}
}

func listMapBindCandidatesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMapBindCandidatesReq
		if err := parseMapRequest(r, &req, "地图绑定候选筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if err := handlerx.RequireMerchant(r, mapPermissionDeps(svcCtx), req.MerchantId); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewBindingLogic(store).ListCandidates(r.Context(), maplogic.ListMapBindCandidatesReq{
			MerchantID: req.MerchantId, ObjectID: req.ObjectId, SceneCode: req.SceneCode, Keyword: req.Keyword, Limit: req.Limit,
		})
		response.JSON(w, resp, err)
	}
}

func getMerchantMapBindingHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, mapPermissionDeps(svcCtx), merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewBindingLogic(store).GetStatus(r.Context(), merchantID)
		response.JSON(w, resp, err)
	}
}

func submitMapBindRequestHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := requireMapStore(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, mapPermissionDeps(svcCtx), merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SubmitMapBindRequestReq
		if err := parseMapRequest(r, &req, "地图绑定申请格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		applicantUserID := ""
		if _, isAdmin := handlerx.OptionalAdmin(r, adminTokenServiceFromMapContext(svcCtx)); !isAdmin {
			user, err := handlerx.RequiredUser(r, userTokenServiceFromMapContext(svcCtx))
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			applicantUserID = user.UserID
		}
		resp, err := maplogic.NewBindingLogic(store).SubmitRequest(r.Context(), merchantID, maplogic.SubmitMapBindRequestReq{
			ObjectID: req.ObjectId, ApplicantUserID: applicantUserID,
			EvidenceImages: req.EvidenceImages, Note: req.Note,
		})
		response.JSON(w, resp, err)
	}
}

func adminListMapScenesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListMapScenesReq
		if err := parseMapRequest(r, &req, "后台地图场景筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).ListScenes(r.Context(), maplogic.ListAdminScenesReq{
			CityCode: req.CityCode, Status: req.Status, Type: req.Type,
		})
		response.JSON(w, resp, err)
	}
}

func adminSaveMapSceneHTTPHandler(svcCtx *svc.ServiceContext, usePathCode bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveMapSceneReq
		if err := parseMapRequest(r, &req, "后台地图场景参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if usePathCode {
			req.Code = pathvar.Vars(r)["sceneCode"]
		}
		resp, err := maplogic.NewAdminLogic(store).SaveScene(r.Context(), maplogic.SaveSceneReq{
			CityCode: req.CityCode, Code: req.Code, Name: req.Name, Type: req.Type, ParentCode: req.ParentCode,
			BackgroundUrl: req.BackgroundUrl, Width: req.Width, Height: req.Height,
			MinScale: req.MinScale, MaxScale: req.MaxScale, DefaultScale: req.DefaultScale,
			DefaultCenterX: req.DefaultCenterX, DefaultCenterY: req.DefaultCenterY,
			FloorNo: req.FloorNo, Sort: req.Sort, Status: req.Status,
		}, admin.OperatorID)
		logAdminMapFailure(r, admin, "保存地图场景", err)
		response.JSON(w, resp, err)
	}
}

func adminGetMapSceneHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).GetScene(r.Context(), pathvar.Vars(r)["sceneCode"])
		response.JSON(w, resp, err)
	}
}

func adminPublishMapSceneHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).PublishScene(r.Context(), pathvar.Vars(r)["sceneCode"], admin.OperatorID)
		logAdminMapFailure(r, admin, "发布地图场景", err)
		response.JSON(w, resp, err)
	}
}

func adminListMapObjectsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListMapObjectsReq
		if err := parseMapRequest(r, &req, "后台地图点位筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).ListObjects(r.Context(), pathvar.Vars(r)["sceneCode"], maplogic.ListAdminObjectsReq{
			Types: req.Types, Status: req.Status, Keyword: req.Keyword,
			MinX: req.MinX, MinY: req.MinY, MaxX: req.MaxX, MaxY: req.MaxY, Zoom: req.Zoom,
		})
		response.JSON(w, resp, err)
	}
}

func adminSaveMapObjectHTTPHandler(svcCtx *svc.ServiceContext, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveMapObjectReq
		if err := parseMapRequest(r, &req, "后台地图点位参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		sceneCode := pathvar.Vars(r)["sceneCode"]
		if update {
			req.Id = pathvar.Vars(r)["objectId"]
			sceneCode = ""
		}
		resp, err := maplogic.NewAdminLogic(store).SaveObject(r.Context(), sceneCode, maplogic.SaveObjectReq{
			Id: req.Id, MerchantID: req.MerchantId, Code: req.Code, Name: req.Name, Type: req.Type, Layer: req.Layer,
			GeometryType: req.GeometryType, Geometry: req.Geometry, MinZoom: req.MinZoom, MaxZoom: req.MaxZoom,
			CategoryCodes: req.CategoryCodes, ServiceTags: req.ServiceTags, PlatformTags: req.PlatformTags,
			PoiServiceTags: req.PoiServiceTags, Address: req.Address, Phone: req.Phone, Wechat: req.Wechat,
			Lat: req.Lat, Lng: req.Lng, Extra: req.Extra, Sort: req.Sort, Status: req.Status,
		}, admin.OperatorID)
		logAdminMapFailure(r, admin, "保存地图点位", err)
		response.JSON(w, resp, err)
	}
}

func adminUpdateMapObjectStatusHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminUpdateMapObjectStatusReq
		if err := parseMapRequest(r, &req, "地图点位状态参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).UpdateObjectStatus(r.Context(), pathvar.Vars(r)["objectId"], maplogic.UpdateObjectStatusReq{Status: req.Status}, admin.OperatorID)
		logAdminMapFailure(r, admin, "更新地图点位状态", err)
		response.JSON(w, resp, err)
	}
}

func adminBatchGenerateMapObjectsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminBatchGenerateMapObjectsReq
		if err := parseMapRequest(r, &req, "批量生成地图点位参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).BatchGenerateObjects(r.Context(), pathvar.Vars(r)["sceneCode"], maplogic.BatchGenerateObjectsReq{
			StartCode: req.StartCode, Count: req.Count, Direction: req.Direction,
			StartX: req.StartX, StartY: req.StartY, Width: req.Width, Height: req.Height, Gap: req.Gap,
			Type: req.Type, Layer: req.Layer, CategoryCodes: req.CategoryCodes, ServiceTags: req.ServiceTags,
		}, admin.OperatorID)
		// 批量坐标和原始请求体不写日志，只保留管理员与稳定场景标识。
		if err != nil {
			logx.WithContext(r.Context()).Errorw("批量生成地图点位失败",
				logx.Field("operatorId", admin.OperatorID),
				logx.Field("sceneCode", pathvar.Vars(r)["sceneCode"]),
				logx.Field("errorType", fmt.Sprintf("%T", err)))
		}
		response.JSON(w, resp, err)
	}
}

func adminListMapCategoriesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListMapCategoriesReq
		if err := parseMapRequest(r, &req, "后台地图分类筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).ListCategories(r.Context(), maplogic.ListCategoriesReq{Type: req.Type, Status: req.Status})
		response.JSON(w, resp, err)
	}
}

func adminSaveMapCategoryHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveMapCategoryReq
		if err := parseMapRequest(r, &req, "后台地图分类参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewAdminLogic(store).SaveCategory(r.Context(), maplogic.SaveCategoryReq{
			Code: req.Code, Name: req.Name, Type: req.Type, IconUrl: req.IconUrl,
			Sort: req.Sort, IsVisible: req.IsVisible, Status: req.Status,
		}, admin.OperatorID)
		logAdminMapFailure(r, admin, "保存地图分类", err)
		response.JSON(w, resp, err)
	}
}

func adminListMapBindRequestsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListMapBindRequestsReq
		if err := parseMapRequest(r, &req, "地图绑定申请筛选参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := maplogic.NewBindingLogic(store).ListAdminRequests(r.Context(), maplogic.ListAdminMapBindRequestsReq{
			Status: req.Status, Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
		})
		response.JSON(w, resp, err)
	}
}

func adminReviewMapBindRequestHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMapContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminReviewMapBindRequestReq
		if err := parseMapRequest(r, &req, "地图绑定审核参数格式不正确"); err != nil {
			response.JSON(w, nil, err)
			return
		}
		// 审核人只取 AdminAuth 写入的管理员身份，Body/query 伪造 operatorId 不会进入 Logic。
		resp, err := maplogic.NewBindingLogic(store).ReviewRequest(r.Context(), pathvar.Vars(r)["requestId"], maplogic.ReviewMapBindRequestReq{
			Action: req.Action, ReviewNote: req.ReviewNote, ReviewerID: admin.OperatorID,
		})
		response.JSON(w, resp, err)
	}
}

func parseMapRequest(r *http.Request, target any, publicMessage string) error {
	if err := httpx.Parse(r, target); err != nil {
		return errx.New(errx.CodeValidationFailed, publicMessage)
	}
	return nil
}

func requireMapStore(r *http.Request, svcCtx *svc.ServiceContext) (mapStore, error) {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MapModel == nil {
		logx.WithContext(mapRequestContext(r)).Error("地图 Handler 存储依赖未配置")
		return nil, errx.New(errx.CodeInternalError, "地图服务暂不可用，请稍后重试")
	}
	return svcCtx.APIStore, nil
}

func requireAdminMapContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, mapStore, error) {
	admin, err := handlerx.AdminFromContext(mapRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	store, err := requireMapStore(r, svcCtx)
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	return admin, store, nil
}

func mapPermissionDeps(svcCtx *svc.ServiceContext) handlerx.MerchantPermissionDeps {
	if svcCtx == nil {
		return handlerx.MerchantPermissionDeps{}
	}
	var store handlerx.MerchantPermissionStore
	if svcCtx.APIStore != nil && svcCtx.APIStore.UserModel != nil {
		store = svcCtx.APIStore
	}
	return handlerx.MerchantPermissionDeps{
		UserTokenService: svcCtx.UserTokenService, AdminTokenService: svcCtx.AdminTokenService, Store: store,
	}
}

func userTokenServiceFromMapContext(svcCtx *svc.ServiceContext) authlogic.TokenService {
	if svcCtx == nil {
		return nil
	}
	return svcCtx.UserTokenService
}

func adminTokenServiceFromMapContext(svcCtx *svc.ServiceContext) handlerx.AdminTokenService {
	if svcCtx == nil {
		return nil
	}
	return svcCtx.AdminTokenService
}

func logAdminMapFailure(r *http.Request, admin session.AdminTokenSubject, operation string, err error) {
	if err == nil {
		return
	}
	logx.WithContext(mapRequestContext(r)).Errorw(operation+"失败",
		logx.Field("operatorId", admin.OperatorID),
		logx.Field("errorType", fmt.Sprintf("%T", err)))
}

func mapRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
