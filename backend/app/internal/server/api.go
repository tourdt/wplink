package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/logic/adminauth"
	authlogic "wplink/backend/app/internal/logic/auth"
	citylogic "wplink/backend/app/internal/logic/city"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	resourcelogic "wplink/backend/app/internal/logic/resource"
	uploadlogic "wplink/backend/app/internal/logic/upload"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"
)

type CityAPIStore interface {
	citylogic.CityStationStore
	citylogic.ResourceTypeStore
}

type ResourceAPIStore interface {
	resourcelogic.CreateResourceStore
	resourcelogic.SubmitResourceStore
	resourcelogic.ListResourcesStore
	resourcelogic.SearchResourceStore
	resourcelogic.GetResourceStore
	resourcelogic.MyResourceStore
	adminlogic.PendingResourceStore
	adminlogic.ReviewResourceStore
	metricslogic.ContactStore
	metricslogic.MetricUpsertStore
}

type MerchantPermissionStore interface {
	UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error)
}

type ResourceMerchantStore interface {
	GetResourceMerchantID(ctx context.Context, resourceID string) (string, error)
}

type AdminLoginService interface {
	Login(ctx context.Context, req adminauth.LoginRequest) (adminauth.LoginResponse, error)
}

type AdminTokenService interface {
	ParseAdminToken(ctx context.Context, token string) (session.AdminTokenSubject, error)
}

type UploadTokenService interface {
	CreateUploadToken(ctx context.Context, req uploadlogic.CreateUploadTokenReq) (uploadlogic.CreateUploadTokenResp, error)
}

type apiRouterOptions struct {
	adminLoginService   AdminLoginService
	adminTokenService   AdminTokenService
	uploadTokenService  UploadTokenService
	userTokenService    authlogic.TokenService
	wechatSessionClient authlogic.WechatSessionClient
	smsVerifier         authlogic.SMSVerifier
	wechatPayGateway    paymentlogic.WechatPayGateway
}

type APIRouterOption func(*apiRouterOptions)

func WithAdminLoginService(service AdminLoginService) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.adminLoginService = service
	}
}

func WithAdminTokenService(service AdminTokenService) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.adminTokenService = service
	}
}

func WithUserTokenService(service authlogic.TokenService) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.userTokenService = service
	}
}

func WithUploadTokenService(service UploadTokenService) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.uploadTokenService = service
	}
}

func WithWechatSessionClient(client authlogic.WechatSessionClient) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.wechatSessionClient = client
	}
}

func WithSMSVerifier(verifier authlogic.SMSVerifier) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.smsVerifier = verifier
	}
}

func WithWechatPayGateway(gateway paymentlogic.WechatPayGateway) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.wechatPayGateway = gateway
	}
}

func NewAPIRouter(store CityAPIStore, opts ...APIRouterOption) http.Handler {
	options := apiRouterOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	mux := http.NewServeMux()
	if options.adminLoginService != nil {
		registerAdminAuthRoutes(mux, options.adminLoginService)
	}
	if options.userTokenService != nil {
		if authStore, ok := any(store).(authlogic.UserStore); ok {
			registerAuthRoutes(mux, authStore, options.userTokenService, options.wechatSessionClient, options.smsVerifier)
		}
	}
	if options.uploadTokenService != nil {
		registerUploadRoutes(mux, options.uploadTokenService, options.userTokenService, options.adminTokenService)
	}
	mux.HandleFunc("GET /api/v1/city-stations", func(w http.ResponseWriter, r *http.Request) {
		resp, err := citylogic.NewListCityStationsLogic(store).ListCityStations(r.Context())
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/city-stations/", func(w http.ResponseWriter, r *http.Request) {
		cityCode, ok := cityCodeFromResourceTypePath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		resp, err := citylogic.NewListResourceTypesLogic(store).ListResourceTypes(r.Context(), cityCode)
		response.JSON(w, resp, err)
	})
	if resourceStore, ok := any(store).(ResourceAPIStore); ok {
		permissionStore, _ := any(store).(MerchantPermissionStore)
		registerResourceRoutes(mux, resourceStore, options.userTokenService, options.adminTokenService, permissionStore)
	}
	registerOptionalDomainRoutes(mux, store, options.userTokenService, options.adminTokenService, permissionStoreFromStore(store), options.smsVerifier, options.wechatPayGateway)
	if options.adminTokenService != nil {
		return requireAdminToken(mux, options.adminTokenService)
	}
	return mux
}

func permissionStoreFromStore(store any) MerchantPermissionStore {
	permissionStore, _ := store.(MerchantPermissionStore)
	return permissionStore
}

func registerUploadRoutes(mux *http.ServeMux, service UploadTokenService, userTokenService authlogic.TokenService, adminTokenService AdminTokenService) {
	mux.HandleFunc("POST /api/v1/uploads/token", func(w http.ResponseWriter, r *http.Request) {
		if err := requireUploadTokenAuth(r, userTokenService, adminTokenService); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body uploadlogic.CreateUploadTokenReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := service.CreateUploadToken(r.Context(), body)
		response.JSON(w, resp, err)
	})
}

func requireUploadTokenAuth(r *http.Request, userTokenService authlogic.TokenService, adminTokenService AdminTokenService) error {
	if userTokenService == nil && adminTokenService == nil {
		return nil
	}
	// 上传凭证可直接写入对象存储，生产接入任一 token 服务后必须绑定真实用户或后台操作员身份。
	if subject, ok := adminSubjectFromBearerToken(r, adminTokenService); ok && permission.CanAccessAdmin(subject.Roles) {
		return nil
	}
	if userTokenService != nil {
		if _, err := userSubjectFromBearerToken(r, userTokenService); err == nil {
			return nil
		}
	}
	return errx.New(errx.CodeUnauthorized, "请先登录后上传文件")
}

func requireAdminToken(next http.Handler, tokenService AdminTokenService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/admin/") || r.URL.Path == "/api/v1/admin/auth/login" {
			next.ServeHTTP(w, r)
			return
		}
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if header == "" || token == "" || token == header {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录管理后台"))
			return
		}
		subject, err := tokenService.ParseAdminToken(r.Context(), token)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, err.Error()))
			return
		}
		if !permission.CanAccessAdmin(subject.Roles) {
			response.JSON(w, nil, errx.New(errx.CodeForbidden, "您没有权限访问管理后台"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireMerchantPermission(r *http.Request, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore, merchantID string) error {
	if tokenService == nil || permissionStore == nil {
		return nil
	}
	if subject, ok := adminSubjectFromBearerToken(r, adminTokenService); ok && permission.CanAccessAdmin(subject.Roles) {
		return nil
	}
	subject, err := userSubjectFromBearerToken(r, tokenService)
	if err != nil {
		return err
	}
	if permission.CanAccessAdmin(subject.Roles) {
		return nil
	}
	canManage, err := permissionStore.UserCanManageMerchant(r.Context(), subject.UserID, merchantID)
	if err != nil {
		return err
	}
	if !canManage {
		return errx.New(errx.CodeForbidden, "您没有权限操作该商家")
	}
	return nil
}

func adminSubjectFromBearerToken(r *http.Request, tokenService AdminTokenService) (session.AdminTokenSubject, bool) {
	if tokenService == nil {
		return session.AdminTokenSubject{}, false
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return session.AdminTokenSubject{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || token == header {
		return session.AdminTokenSubject{}, false
	}
	subject, err := tokenService.ParseAdminToken(r.Context(), token)
	if err != nil {
		return session.AdminTokenSubject{}, false
	}
	return subject, true
}

func registerAdminAuthRoutes(mux *http.ServeMux, service AdminLoginService) {
	mux.HandleFunc("POST /api/v1/admin/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			LoginName string `json:"loginName"`
			Password  string `json:"password"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := service.Login(r.Context(), adminauth.LoginRequest{
			LoginName: body.LoginName,
			Password:  body.Password,
		})
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, err.Error()))
			return
		}
		response.JSON(w, resp, nil)
	})
}

func registerResourceRoutes(mux *http.ServeMux, store ResourceAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	mux.HandleFunc("POST /api/v1/resources", func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeCreateResourceRequest(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, req.MerchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if err := bindResourceCreatorFromRequest(r, &req, tokenService, adminTokenService); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewCreateResourceLogic(store).CreateResource(r.Context(), req)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/drafts", func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeCreateResourceRequest(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, req.MerchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if err := bindResourceCreatorFromRequest(r, &req, tokenService, adminTokenService); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewCreateResourceLogic(store).CreateResourceDraft(r.Context(), req)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("PUT /api/v1/resources/{resourceId}/draft", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		req, err := decodeCreateResourceRequest(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			req.MerchantID, err = resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, req.MerchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewCreateResourceLogic(store).UpdateResourceDraft(r.Context(), resourceID, req)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/submit", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		merchantID, err := optionalMerchantIDFromActionRequest(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			merchantID, err = resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		if merchantID != "" || tokenService != nil {
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := resourcelogic.NewSubmitResourceLogic(store).SubmitResource(r.Context(), resourceID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/resources", func(w http.ResponseWriter, r *http.Request) {
		resp, err := resourcelogic.NewListResourcesLogic(store).ListResources(r.Context(), listResourcesReqFromQuery(r))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/resource-search", func(w http.ResponseWriter, r *http.Request) {
		req := searchResourcesReqFromQuery(r)
		userID, err := optionalUserIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		req.UserID = userID
		resp, err := resourcelogic.NewSearchResourcesLogic(store).SearchResources(r.Context(), req)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/resources/{resourceId}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := resourcelogic.NewGetResourceLogic(store).GetResource(r.Context(), r.PathValue("resourceId"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/detail-view", func(w http.ResponseWriter, r *http.Request) {
		err := metricslogic.NewRecordDetailViewLogic(store).RecordDetailView(r.Context(), metricslogic.RecordDetailViewReq{ResourceID: r.PathValue("resourceId")})
		response.JSON(w, map[string]string{"message": "浏览行为已记录"}, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/contact-events", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			UserID string `json:"userId"`
			Action string `json:"action"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var userID string
		var err error
		if isContactUnlockAction(body.Action) {
			subject, authErr := userSubjectFromBearerToken(r, tokenService)
			if authErr != nil {
				response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后联系商家"))
				return
			}
			userID = subject.UserID
		} else {
			userID, err = optionalUserIDFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := metricslogic.NewRecordContactLogic(store).RecordContact(r.Context(), metricslogic.RecordContactReq{
			ResourceID: r.PathValue("resourceId"),
			UserID:     userID,
			Action:     body.Action,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/resources", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, query.Get("merchantId")); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewListMyResourcesLogic(store).ListMyResources(r.Context(), resourcelogic.ListMyResourcesReq{
			MerchantID: query.Get("merchantId"),
			Status:     query.Get("status"),
			Page:       int64FromQuery(r, "page"),
			PageSize:   int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/resources/{resourceId}/detail", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		merchantID := strings.TrimSpace(r.URL.Query().Get("merchantId"))
		var err error
		if tokenService != nil {
			// 自有资源详情允许查看待审核/下架等非公开状态，必须以资源真实归属做权限判断。
			merchantID, err = resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewGetOwnResourceLogic(store).Get(r.Context(), resourcelogic.GetOwnResourceReq{
			MerchantID: merchantID,
			ResourceID: resourceID,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/resources/{resourceId}/edit", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		merchantID := strings.TrimSpace(r.URL.Query().Get("merchantId"))
		var err error
		if tokenService != nil {
			merchantID, err = resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewGetEditableResourceLogic(store).Get(r.Context(), resourcelogic.GetEditableResourceReq{
			MerchantID: merchantID,
			ResourceID: resourceID,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/refresh", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		merchantID, err := merchantIDFromActionRequest(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			merchantID, err = resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewRefreshResourceLogic(store).RefreshResource(r.Context(), resourcelogic.RefreshResourceReq{
			MerchantID: merchantID,
			ResourceID: resourceID,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/deal-feedback", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		var body struct {
			MerchantID              string `json:"merchantId"`
			IsDealt                 bool   `json:"isDealt"`
			IsReal                  bool   `json:"isReal"`
			ResponseTimely          bool   `json:"responseTimely"`
			WillingToCooperateAgain bool   `json:"willingToCooperateAgain"`
			Note                    string `json:"note"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if strings.TrimSpace(body.MerchantID) == "" {
			body.MerchantID = r.URL.Query().Get("merchantId")
		}
		if tokenService != nil {
			merchantID, err := resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			body.MerchantID = merchantID
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, body.MerchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewMarkDealtLogic(store).MarkDealt(r.Context(), resourcelogic.MarkDealtReq{
			MerchantID: body.MerchantID, ResourceID: resourceID, IsDealt: body.IsDealt,
			IsReal: body.IsReal, ResponseTimely: body.ResponseTimely, WillingToCooperateAgain: body.WillingToCooperateAgain, Note: body.Note,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/take-down", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		var body struct {
			MerchantID string `json:"merchantId"`
			Reason     string `json:"reason"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if strings.TrimSpace(body.MerchantID) == "" {
			body.MerchantID = r.URL.Query().Get("merchantId")
		}
		if tokenService != nil {
			merchantID, err := resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			body.MerchantID = merchantID
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, body.MerchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewTakeDownOwnResourceLogic(store).TakeDown(r.Context(), resourcelogic.TakeDownOwnResourceReq{
			MerchantID: body.MerchantID,
			ResourceID: resourceID,
			Reason:     body.Reason,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("DELETE /api/v1/resources/{resourceId}", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		merchantID, err := merchantIDFromActionRequest(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			merchantID, err = resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewDeleteTakenDownResourceLogic(store).Delete(r.Context(), resourcelogic.DeleteTakenDownResourceReq{
			MerchantID: merchantID,
			ResourceID: resourceID,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/repost-similar", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		merchantID, err := merchantIDFromActionRequest(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			merchantID, err = resourceMerchantIDFromStore(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := resourcelogic.NewRepostSimilarLogic(store).RepostSimilar(r.Context(), resourcelogic.RepostSimilarReq{
			MerchantID: merchantID,
			ResourceID: resourceID,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/resources/pending", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewListPendingResourcesLogic(store).ListPendingResources(r.Context(), adminlogic.ListPendingResourcesReq{
			CityCode: query.Get("cityCode"),
			TypeCode: query.Get("typeCode"),
			Page:     int64FromQuery(r, "page"),
			PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/resources/{resourceId}/review", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Action     string `json:"action"`
			Reason     string `json:"reason"`
			ReviewerID string `json:"reviewerId"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if reviewerID, err := adminOperatorIDFromRequest(r, adminTokenService, body.ReviewerID); err != nil {
			response.JSON(w, nil, err)
			return
		} else {
			body.ReviewerID = reviewerID
		}
		resp, err := adminlogic.NewReviewResourceLogic(store).ReviewResource(r.Context(), r.PathValue("resourceId"), adminlogic.ReviewResourceReq{
			Action:     body.Action,
			Reason:     body.Reason,
			ReviewerID: body.ReviewerID,
		})
		response.JSON(w, resp, err)
	})
}

func cityCodeFromResourceTypePath(requestPath string) (string, bool) {
	const prefix = "/api/v1/city-stations/"
	const suffix = "/resource-types"
	if !strings.HasPrefix(requestPath, prefix) || !strings.HasSuffix(requestPath, suffix) {
		return "", false
	}
	cityCode := strings.TrimSuffix(strings.TrimPrefix(requestPath, prefix), suffix)
	cityCode = strings.Trim(cityCode, "/")
	return cityCode, cityCode != ""
}

func decodeCreateResourceRequest(r *http.Request) (resourcelogic.CreateResourceReq, error) {
	var body struct {
		MerchantID   string                           `json:"merchantId"`
		CityCode     string                           `json:"cityCode"`
		TypeCode     string                           `json:"typeCode"`
		Title        string                           `json:"title"`
		Category     string                           `json:"category"`
		District     string                           `json:"district"`
		PriceText    string                           `json:"priceText"`
		QuantityText string                           `json:"quantityText"`
		Description  string                           `json:"description"`
		Attributes   model.JSONMap                    `json:"attributes"`
		Tags         []string                         `json:"tags"`
		Images       []string                         `json:"images"`
		Contact      resourcelogic.ResourceContactReq `json:"contact"`
	}
	if err := decodeJSONBody(r, &body); err != nil {
		return resourcelogic.CreateResourceReq{}, err
	}
	return resourcelogic.CreateResourceReq{
		MerchantID: body.MerchantID, CityCode: body.CityCode, TypeCode: body.TypeCode,
		Title: body.Title, Category: body.Category, District: body.District, PriceText: body.PriceText,
		QuantityText: body.QuantityText, Description: body.Description, Attributes: body.Attributes,
		Tags: body.Tags, Images: body.Images, Contact: body.Contact,
	}, nil
}

func isContactUnlockAction(action string) bool {
	action = strings.TrimSpace(action)
	return action == "phone" || action == "wechat"
}

func listResourcesReqFromQuery(r *http.Request) resourcelogic.ListResourcesReq {
	query := r.URL.Query()
	return resourcelogic.ListResourcesReq{
		CityCode:     query.Get("cityCode"),
		MerchantID:   query.Get("merchantId"),
		TypeCode:     query.Get("typeCode"),
		Keyword:      query.Get("keyword"),
		Category:     query.Get("category"),
		VerifiedOnly: boolFromQuery(r, "verifiedOnly"),
		Page:         int64FromQuery(r, "page"),
		PageSize:     int64FromQuery(r, "pageSize"),
	}
}

func searchResourcesReqFromQuery(r *http.Request) resourcelogic.SearchResourcesReq {
	req := listResourcesReqFromQuery(r)
	return resourcelogic.SearchResourcesReq{
		UserID:       r.URL.Query().Get("userId"),
		CityCode:     req.CityCode,
		TypeCode:     req.TypeCode,
		Keyword:      req.Keyword,
		Category:     req.Category,
		VerifiedOnly: req.VerifiedOnly,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}
}

func merchantIDFromActionRequest(r *http.Request) (string, error) {
	merchantID, err := optionalMerchantIDFromActionRequest(r)
	if err != nil {
		return "", err
	}
	if merchantID == "" {
		return "", errx.New(errx.CodeValidationFailed, "商家不存在")
	}
	return merchantID, nil
}

func optionalMerchantIDFromActionRequest(r *http.Request) (string, error) {
	var body struct {
		MerchantID string `json:"merchantId"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSONBody(r, &body); err != nil {
			return "", err
		}
	}
	merchantID := strings.TrimSpace(body.MerchantID)
	if merchantID == "" {
		merchantID = strings.TrimSpace(r.URL.Query().Get("merchantId"))
	}
	return merchantID, nil
}

func adminOperatorIDFromRequest(r *http.Request, tokenService AdminTokenService, fallback string) (string, error) {
	if tokenService == nil {
		return strings.TrimSpace(fallback), nil
	}
	subject, ok := adminSubjectFromBearerToken(r, tokenService)
	if !ok || strings.TrimSpace(subject.UserID) == "" {
		return "", errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	return strings.TrimSpace(subject.UserID), nil
}

func resourceMerchantIDFromStore(ctx context.Context, store ResourceAPIStore, resourceID string) (string, error) {
	ownerStore, ok := any(store).(ResourceMerchantStore)
	if !ok {
		return "", errx.New(errx.CodeForbidden, "您没有权限操作该资源")
	}
	merchantID, err := ownerStore.GetResourceMerchantID(ctx, strings.TrimSpace(resourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return "", errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	if err != nil {
		return "", err
	}
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return "", errx.New(errx.CodeForbidden, "您没有权限操作该资源")
	}
	return merchantID, nil
}

func bindResourceCreatorFromRequest(r *http.Request, req *resourcelogic.CreateResourceReq, tokenService authlogic.TokenService, adminTokenService AdminTokenService) error {
	if subject, ok := adminSubjectFromBearerToken(r, adminTokenService); ok && permission.CanAccessAdmin(subject.Roles) {
		req.CreatedByUser = strings.TrimSpace(subject.UserID)
		req.CreatedByRole = primaryRole(subject.Roles, "platform_operator")
		return nil
	}
	if tokenService == nil {
		return nil
	}
	subject, err := userSubjectFromBearerToken(r, tokenService)
	if err != nil {
		return err
	}
	req.CreatedByUser = strings.TrimSpace(subject.UserID)
	req.CreatedByRole = primaryRole(subject.Roles, "merchant_admin")
	return nil
}

func primaryRole(roles []string, fallback string) string {
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role != "" {
			return role
		}
	}
	return fallback
}

func decodeJSONBody(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errx.New(errx.CodeValidationFailed, "请求参数格式不正确")
	}
	return nil
}

func int64FromQuery(r *http.Request, key string) int64 {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func boolFromQuery(r *http.Request, key string) bool {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}
