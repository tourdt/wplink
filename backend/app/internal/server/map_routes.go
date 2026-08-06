package server

import (
	"net/http"

	authlogic "wplink/backend/app/internal/logic/auth"
	maplogic "wplink/backend/app/internal/logic/map"
	"wplink/backend/app/internal/model"
	"wplink/backend/common/response"
)

func registerMapRoutes(mux *http.ServeMux, store MapAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	publicLogic := maplogic.NewPublicLogic(store)
	adminLogic := maplogic.NewAdminLogic(store)
	bindingLogic := maplogic.NewBindingLogic(store)
	reportLogic := maplogic.NewReportLogic(store)

	mux.HandleFunc("GET /api/v1/map/scenes", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := publicLogic.ListScenes(r.Context(), maplogic.ListScenesReq{
			CityCode:   query.Get("cityCode"),
			ParentCode: query.Get("parentCode"),
			Type:       query.Get("type"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/merchant-places", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := publicLogic.ListMerchantPlaces(r.Context(), maplogic.ListMerchantPlacesReq{
			CityCode:      query.Get("cityCode"),
			Keyword:       query.Get("keyword"),
			Categories:    query.Get("categories"),
			MerchantTypes: query.Get("merchantTypes"),
			Claimed:       query.Get("claimed"),
			Page:          int64FromQuery(r, "page"),
			PageSize:      int64FromQuery(r, "pageSize"),
			MinLat:        query.Get("minLat"),
			MaxLat:        query.Get("maxLat"),
			MinLng:        query.Get("minLng"),
			MaxLng:        query.Get("maxLng"),
			Lat:           query.Get("lat"),
			Lng:           query.Get("lng"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/merchants/{merchantId}/location-context", func(w http.ResponseWriter, r *http.Request) {
		resp, err := publicLogic.GetMerchantLocationContext(r.Context(), r.PathValue("merchantId"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/scenes/{sceneCode}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := publicLogic.GetScene(r.Context(), r.PathValue("sceneCode"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/scenes/{sceneCode}/objects", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := publicLogic.ListObjects(r.Context(), r.PathValue("sceneCode"), maplogic.ListObjectsReq{
			Types:          query.Get("types"),
			Categories:     query.Get("categories"),
			ServiceTags:    query.Get("serviceTags"),
			PoiServiceTags: query.Get("poiServiceTags"),
			Keyword:        query.Get("keyword"),
			MinX:           query.Get("minX"),
			MinY:           query.Get("minY"),
			MaxX:           query.Get("maxX"),
			MaxY:           query.Get("maxY"),
			Zoom:           int64FromQuery(r, "zoom"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/objects/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := publicLogic.SearchObjects(r.Context(), maplogic.SearchObjectsReq{
			SceneCode:      query.Get("sceneCode"),
			Keyword:        query.Get("keyword"),
			Types:          query.Get("types"),
			Categories:     query.Get("categories"),
			ServiceTags:    query.Get("serviceTags"),
			PoiServiceTags: query.Get("poiServiceTags"),
			MinX:           query.Get("minX"),
			MinY:           query.Get("minY"),
			MaxX:           query.Get("maxX"),
			MaxY:           query.Get("maxY"),
			Zoom:           int64FromQuery(r, "zoom"),
			Limit:          int64FromQuery(r, "limit"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/objects/{objectId}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := publicLogic.GetObject(r.Context(), r.PathValue("objectId"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/objects/{objectId}/nearby-pois", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := publicLogic.ListNearbyPois(r.Context(), r.PathValue("objectId"), maplogic.ListNearbyPoisReq{
			Types: query.Get("types"),
			Limit: int64FromQuery(r, "limit"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/map/objects/{objectId}/location-corrections", func(w http.ResponseWriter, r *http.Request) {
		submitMapObjectReport(w, r, tokenService, reportLogic, model.MapObjectReportKindLocationCorrection)
	})
	mux.HandleFunc("POST /api/v1/map/objects/{objectId}/risk-reports", func(w http.ResponseWriter, r *http.Request) {
		submitMapObjectReport(w, r, tokenService, reportLogic, model.MapObjectReportKindRiskReport)
	})
	mux.HandleFunc("GET /api/v1/map/categories", func(w http.ResponseWriter, r *http.Request) {
		resp, err := publicLogic.ListCategories(r.Context(), maplogic.ListCategoriesReq{Type: r.URL.Query().Get("type")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/map/bind-candidates", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		merchantID := query.Get("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := bindingLogic.ListCandidates(r.Context(), maplogic.ListMapBindCandidatesReq{
			MerchantID: merchantID,
			ObjectID:   query.Get("objectId"),
			SceneCode:  query.Get("sceneCode"),
			Keyword:    query.Get("keyword"),
			Limit:      int64FromQuery(r, "limit"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/map-binding", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := bindingLogic.GetStatus(r.Context(), merchantID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/merchants/{merchantId}/map-binding-requests", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.SubmitMapBindRequestReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			if _, ok := adminSubjectFromBearerToken(r, adminTokenService); !ok {
				userID, err := userIDFromBearerToken(r, tokenService)
				if err != nil {
					response.JSON(w, nil, err)
					return
				}
				body.ApplicantUserID = userID
			}
		}
		resp, err := bindingLogic.SubmitRequest(r.Context(), merchantID, body)
		response.JSON(w, resp, err)
	})

	mux.HandleFunc("GET /api/v1/admin/map/scenes", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminLogic.ListScenes(r.Context(), maplogic.ListAdminScenesReq{
			CityCode: query.Get("cityCode"),
			Status:   query.Get("status"),
			Type:     query.Get("type"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/scenes", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.SaveSceneReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.SaveScene(r.Context(), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/map/scenes/{sceneCode}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminLogic.GetScene(r.Context(), r.PathValue("sceneCode"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/scenes/{sceneCode}", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.SaveSceneReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.Code = r.PathValue("sceneCode")
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.SaveScene(r.Context(), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/scenes/{sceneCode}/publish", func(w http.ResponseWriter, r *http.Request) {
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.PublishScene(r.Context(), r.PathValue("sceneCode"), operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/map/scenes/{sceneCode}/objects", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminLogic.ListObjects(r.Context(), r.PathValue("sceneCode"), maplogic.ListAdminObjectsReq{
			Types:   query.Get("types"),
			Status:  query.Get("status"),
			Keyword: query.Get("keyword"),
			MinX:    query.Get("minX"),
			MinY:    query.Get("minY"),
			MaxX:    query.Get("maxX"),
			MaxY:    query.Get("maxY"),
			Zoom:    int64FromQuery(r, "zoom"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/scenes/{sceneCode}/objects", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.SaveObjectReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.SaveObject(r.Context(), r.PathValue("sceneCode"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/objects/{objectId}", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.SaveObjectReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.Id = r.PathValue("objectId")
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.SaveObject(r.Context(), "", body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/objects/{objectId}/status", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.UpdateObjectStatusReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.UpdateObjectStatus(r.Context(), r.PathValue("objectId"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/scenes/{sceneCode}/objects/batch-generate", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.BatchGenerateObjectsReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.BatchGenerateObjects(r.Context(), r.PathValue("sceneCode"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/map/categories", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminLogic.ListCategories(r.Context(), maplogic.ListCategoriesReq{Type: query.Get("type"), Status: query.Get("status")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/categories", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.SaveCategoryReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminLogic.SaveCategory(r.Context(), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/map/bind-requests", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := bindingLogic.ListAdminRequests(r.Context(), maplogic.ListAdminMapBindRequestsReq{
			Status:   query.Get("status"),
			Keyword:  query.Get("keyword"),
			Page:     int64FromQuery(r, "page"),
			PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/map/bind-requests/{requestId}/review", func(w http.ResponseWriter, r *http.Request) {
		var body maplogic.ReviewMapBindRequestReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if subject, ok := adminSubjectFromBearerToken(r, adminTokenService); ok {
			body.ReviewerID = subject.OperatorID
		}
		resp, err := bindingLogic.ReviewRequest(r.Context(), r.PathValue("requestId"), body)
		response.JSON(w, resp, err)
	})
}

func submitMapObjectReport(w http.ResponseWriter, r *http.Request, tokenService authlogic.TokenService, reportLogic *maplogic.ReportLogic, kind string) {
	userID, err := userIDFromBearerToken(r, tokenService)
	if err != nil {
		response.JSON(w, nil, err)
		return
	}
	var body maplogic.SubmitMapObjectReportReq
	if err := decodeJSONBody(r, &body); err != nil {
		response.JSON(w, nil, err)
		return
	}
	// 提交人只采用服务端令牌身份，客户端即使传入同名字段也无法伪造举报归属。
	body.ReporterUserID = userID
	resp, err := reportLogic.Submit(r.Context(), r.PathValue("objectId"), kind, body)
	response.JSON(w, resp, err)
}
