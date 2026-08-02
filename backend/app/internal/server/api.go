package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/logic/adminauth"
	authlogic "wplink/backend/app/internal/logic/auth"
	citylogic "wplink/backend/app/internal/logic/city"
	contentauditlogic "wplink/backend/app/internal/logic/contentaudit"
	locationlogic "wplink/backend/app/internal/logic/location"
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
	resourcelogic.ContactUnlockOrderStore
	resourcelogic.MyResourceStore
	adminlogic.PendingResourceStore
	adminlogic.ReviewResourceStore
	metricslogic.ContactStore
	metricslogic.MetricUpsertStore
	paymentlogic.ContactUnlockPaymentStore
}

type MerchantPermissionStore interface {
	UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error)
}

type ManagedMerchantStore interface {
	ListManagedMerchantIDs(ctx context.Context, userID string) ([]string, error)
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
	adminLoginService            AdminLoginService
	adminTokenService            AdminTokenService
	uploadTokenService           UploadTokenService
	userTokenService             authlogic.TokenService
	wechatSessionClient          authlogic.WechatSessionClient
	smsVerifier                  authlogic.SMSVerifier
	wechatPayGateway             paymentlogic.WechatPayGateway
	wechatPayDevMock             bool
	contentAuditor               resourcelogic.ContentAuditor
	contentAuditCallbackVerifier contentauditlogic.WechatCallbackVerifier
	contentAuditCallbackAppID    string
	locationGeocoder             locationlogic.ReverseGeocoder
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

func WithWechatPayDevMock(enabled bool) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.wechatPayDevMock = enabled
	}
}

func WithContentAuditor(auditor resourcelogic.ContentAuditor) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.contentAuditor = auditor
	}
}

func WithContentAuditCallbackVerifier(verifier contentauditlogic.WechatCallbackVerifier, appID string) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.contentAuditCallbackVerifier = verifier
		options.contentAuditCallbackAppID = strings.TrimSpace(appID)
	}
}

func WithLocationGeocoder(geocoder locationlogic.ReverseGeocoder) APIRouterOption {
	return func(options *apiRouterOptions) {
		options.locationGeocoder = geocoder
	}
}

func NewAPIRouter(store CityAPIStore, opts ...APIRouterOption) http.Handler {
	return newAPIRouterWithOptions(store, buildAPIRouterOptions(opts...))
}

func NewProductionAPIRouter(store CityAPIStore, opts ...APIRouterOption) (http.Handler, error) {
	options := buildAPIRouterOptions(opts...)
	if err := validateProductionAPIRouterDependencies(store, options); err != nil {
		return nil, err
	}
	return newAPIRouterWithOptions(store, options), nil
}

func buildAPIRouterOptions(opts ...APIRouterOption) apiRouterOptions {
	options := apiRouterOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

func validateProductionAPIRouterDependencies(store CityAPIStore, options apiRouterOptions) error {
	var missing []string
	if store == nil {
		missing = append(missing, "APIStore")
	}
	if options.adminLoginService == nil {
		missing = append(missing, "AdminLoginService")
	}
	if options.adminTokenService == nil {
		missing = append(missing, "AdminTokenService")
	}
	if options.userTokenService == nil {
		missing = append(missing, "UserTokenService")
	}
	if options.uploadTokenService == nil {
		missing = append(missing, "UploadTokenService")
	}
	if options.wechatSessionClient == nil {
		missing = append(missing, "WechatSessionClient")
	}
	if options.smsVerifier == nil {
		missing = append(missing, "SMSVerifier")
	}
	if options.contentAuditCallbackVerifier == nil || options.contentAuditCallbackAppID == "" {
		missing = append(missing, "ContentAuditCallbackVerifier")
	}
	if _, ok := any(store).(authlogic.UserStore); !ok {
		missing = append(missing, "UserStore")
	}
	if _, ok := any(store).(ResourceAPIStore); !ok {
		missing = append(missing, "ResourceAPIStore")
	}
	if _, ok := any(store).(AdminPermissionAPIStore); !ok {
		missing = append(missing, "AdminPermissionAPIStore")
	}
	if permissionStoreFromStore(store) == nil {
		missing = append(missing, "MerchantPermissionStore")
	}
	if len(missing) > 0 {
		return errors.New("生产 API 路由依赖缺失: " + strings.Join(missing, ", "))
	}
	return nil
}

func newAPIRouterWithOptions(store CityAPIStore, options apiRouterOptions) http.Handler {
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
		resp, err := citylogic.NewListResourceTypesLogic(store).ListResourceTypes(r.Context(), citylogic.ListResourceTypesReq{
			CityCode:  cityCode,
			Direction: r.URL.Query().Get("direction"),
		})
		response.JSON(w, resp, err)
	})
	registerLocationRoutes(mux, options.locationGeocoder)
	if mapEventStore, ok := any(store).(metricslogic.MerchantMapEventStore); ok {
		registerMerchantMapEventRoute(mux, mapEventStore, options.userTokenService)
	}
	if exposureStore, ok := any(store).(metricslogic.ResourceExposureStore); ok {
		registerResourceExposureRoute(mux, exposureStore, options.userTokenService)
	}
	if resourceStore, ok := any(store).(ResourceAPIStore); ok {
		permissionStore, _ := any(store).(MerchantPermissionStore)
		registerResourceRoutes(
			mux,
			resourceStore,
			options.userTokenService,
			options.adminTokenService,
			permissionStore,
			options.wechatPayGateway,
			options.wechatPayDevMock,
			options.contentAuditor,
			options.contentAuditCallbackVerifier,
			options.contentAuditCallbackAppID,
		)
	}
	registerOptionalDomainRoutes(mux, store, options.userTokenService, options.adminTokenService, permissionStoreFromStore(store), options.smsVerifier, options.wechatPayGateway, options.wechatPayDevMock)
	if options.adminTokenService != nil {
		return requireAdminToken(mux, options.adminTokenService)
	}
	return mux
}

func registerMerchantMapEventRoute(mux *http.ServeMux, store metricslogic.MerchantMapEventStore, tokenService authlogic.TokenService) {
	limiter := newMerchantMapEventRateLimiter(
		merchantMapEventRateLimit,
		merchantMapEventRateWindowSize,
		merchantMapEventRateMaxKeys,
		time.Now,
	)
	mux.HandleFunc("POST /api/v1/metrics/merchant-map-events", func(w http.ResponseWriter, r *http.Request) {
		rawBody, err := readLimitedBody(r, merchantMapEventRequestBodyLimit)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "地图行为请求内容过大"))
			return
		}
		var body metricslogic.RecordMerchantMapEventReq
		decoder := json.NewDecoder(bytes.NewReader(rawBody))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请求参数格式不正确"))
			return
		}
		if !limiter.Allow(requestClientIP(r), body.VisitorKey) {
			response.JSON(w, nil, errx.New(errx.CodeRateLimited, "操作频繁，请稍后再试"))
			return
		}
		// 请求主动携带凭证时禁止静默降级为匿名；缺少解析服务意味着该凭证无法验证，应按登录过期处理。
		if tokenService == nil && strings.TrimSpace(r.Header.Get("Authorization")) != "" {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "登录已过期，请重新登录"))
			return
		}
		userID, err := optionalUserIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		// 匿名请求保留空 userId；携带 token 时只信任服务端解析结果，禁止前端伪造事件归因。
		body.UserID = userID
		resp, err := metricslogic.NewRecordMerchantMapEventLogic(store).RecordMerchantMapEvent(r.Context(), body)
		response.JSON(w, resp, err)
	})
}

func registerResourceExposureRoute(mux *http.ServeMux, store metricslogic.ResourceExposureStore, tokenService authlogic.TokenService) {
	mux.HandleFunc("POST /api/v1/metrics/exposures/batch", func(w http.ResponseWriter, r *http.Request) {
		var body metricslogic.RecordResourceExposuresReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		userID, err := optionalUserIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.UserID = userID
		resp, err := metricslogic.NewRecordResourceExposuresLogic(store).RecordResourceExposures(r.Context(), body)
		response.JSON(w, resp, err)
	})
}

func registerLocationRoutes(mux *http.ServeMux, geocoder locationlogic.ReverseGeocoder) {
	logic := locationlogic.NewLogic(geocoder)
	mux.HandleFunc("GET /api/v1/locations/reverse-geocode", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := logic.ReverseGeocode(r.Context(), locationlogic.ReverseGeocodeReq{
			Latitude:  query.Get("latitude"),
			Longitude: query.Get("longitude"),
		})
		response.JSON(w, resp, err)
	})
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
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "登录已过期，请重新登录"))
			return
		}
		if !permission.CanAccessAdmin(subject.Roles) {
			response.JSON(w, nil, errx.New(errx.CodeForbidden, "您没有权限访问管理后台"))
			return
		}
		module := adminModuleFromPath(r.URL.Path)
		modules := append([]string(nil), subject.Modules...)
		if len(modules) == 0 {
			modules = permission.ResolveAdminModules(subject.Roles, nil)
		}
		if !permission.CanAccessAdminModule(subject.Roles, modules, module) {
			response.JSON(w, nil, errx.New(errx.CodeForbidden, "您没有权限访问该后台功能"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func adminModuleFromPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/admin/auth/"):
		return ""
	case strings.HasPrefix(path, "/api/v1/admin/dashboard/"):
		return permission.AdminModuleDashboard
	case path == "/api/v1/admin/resources", strings.HasPrefix(path, "/api/v1/admin/resources/pending"), strings.HasPrefix(path, "/api/v1/admin/resources/"):
		return permission.AdminModuleResourceReview
	case strings.HasPrefix(path, "/api/v1/admin/resource-reports"):
		return permission.AdminModuleResourceReports
	case strings.HasPrefix(path, "/api/v1/admin/merchants/") && strings.Contains(path, "/entitlements"):
		return permission.AdminModuleEntitlements
	case strings.HasPrefix(path, "/api/v1/admin/merchants"):
		return permission.AdminModuleMerchants
	case strings.HasPrefix(path, "/api/v1/admin/banner-topics"):
		return permission.AdminModuleBannerTopics
	case strings.HasPrefix(path, "/api/v1/admin/hot-search-keywords"):
		return permission.AdminModuleHotSearchKeywords
	case strings.HasPrefix(path, "/api/v1/admin/vip/"):
		return permission.AdminModuleVIPConfigs
	case strings.HasPrefix(path, "/api/v1/admin/growth-campaigns"):
		return permission.AdminModuleGrowthCampaigns
	case strings.HasPrefix(path, "/api/v1/admin/map/"):
		return permission.AdminModuleSourcingMap
	case strings.HasPrefix(path, "/api/v1/admin/resource-type-configs"):
		return permission.AdminModuleResourceTypeConfigs
	case strings.HasPrefix(path, "/api/v1/admin/operators"), strings.HasPrefix(path, "/api/v1/admin/module-permissions"):
		return permission.AdminModuleAdminPermissions
	case strings.HasPrefix(path, "/api/v1/admin/operation-logs"), strings.HasPrefix(path, "/api/v1/admin/tasks/resource-lifecycle"):
		return permission.AdminModuleOperationLogs
	case strings.HasPrefix(path, "/api/v1/admin/search-logs"):
		return permission.AdminModuleSearchLogs
	default:
		return ""
	}
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
			ClientIP:  requestClientIP(r),
			UserAgent: r.UserAgent(),
		})
		if err != nil {
			code := errx.CodeUnauthorized
			if adminauth.IsLoginRateLimited(err) {
				code = errx.CodeRateLimited
			}
			response.JSON(w, nil, errx.New(code, adminauth.PublicLoginErrorMessage(err)))
			return
		}
		response.JSON(w, resp, nil)
	})
}

func writeWechatCallbackSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}

func registerResourceRoutes(
	mux *http.ServeMux,
	store ResourceAPIStore,
	tokenService authlogic.TokenService,
	adminTokenService AdminTokenService,
	permissionStore MerchantPermissionStore,
	wechatPayGateway paymentlogic.WechatPayGateway,
	wechatPayDevMock bool,
	contentAuditor resourcelogic.ContentAuditor,
	callbackVerifier contentauditlogic.WechatCallbackVerifier,
	callbackAppID string,
) {
	mux.HandleFunc("GET /api/v1/wechat/content-audit/media-callback", func(w http.ResponseWriter, r *http.Request) {
		if callbackVerifier == nil {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "微信内容审核回调验签未配置"))
			return
		}
		query := r.URL.Query()
		if err := callbackVerifier.Verify(query.Get("signature"), query.Get("timestamp"), query.Get("nonce"), false); err != nil {
			response.JSON(w, nil, err)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(query.Get("echostr")))
	})
	mux.HandleFunc("POST /api/v1/wechat/content-audit/media-callback", func(w http.ResponseWriter, r *http.Request) {
		if callbackVerifier == nil {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "微信内容审核回调验签未配置"))
			return
		}
		query := r.URL.Query()
		signature := query.Get("signature")
		timestamp := query.Get("timestamp")
		nonce := query.Get("nonce")
		// 先完成验签和历史重放检查，业务处理成功后再记录指纹。
		// 这样数据库或下游服务瞬时失败时，微信的合法重试不会被误判为已处理。
		if err := callbackVerifier.Verify(signature, timestamp, nonce, false); err != nil {
			if errors.Is(err, contentauditlogic.ErrWechatCallbackReplay) {
				writeWechatCallbackSuccess(w)
				return
			}
			response.JSON(w, nil, err)
			return
		}
		callbackStore, ok := any(store).(contentauditlogic.MediaCheckCallbackStore)
		if !ok {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "图片审核回调服务暂不可用"))
			return
		}
		body, err := readLimitedBody(r, 256<<10)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "图片审核回调内容过大或读取失败"))
			return
		}
		var payload model.JSONMap
		if err := json.Unmarshal(body, &payload); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请求参数格式不正确"))
			return
		}
		if payload == nil {
			payload = model.JSONMap{}
		}
		if appID, _ := payload["appid"].(string); strings.TrimSpace(appID) == "" || strings.TrimSpace(appID) != callbackAppID {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "微信内容审核回调 AppID 不匹配"))
			return
		}
		_, err = contentauditlogic.NewMediaCheckCallbackLogic(callbackStore).Handle(r.Context(), payload)
		if err != nil {
			if errx.CodeOf(err) == errx.CodeStateConflict {
				_ = callbackVerifier.Verify(signature, timestamp, nonce, true)
				writeWechatCallbackSuccess(w)
				return
			}
			response.JSON(w, nil, err)
			return
		}
		_ = callbackVerifier.Verify(signature, timestamp, nonce, true)
		writeWechatCallbackSuccess(w)
	})
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
		resp, err := resourcelogic.NewCreateResourceLogic(store, contentAuditor).CreateResource(r.Context(), req)
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
		resp, err := resourcelogic.NewCreateResourceLogic(store, contentAuditor).CreateResourceDraft(r.Context(), req)
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
		resp, err := resourcelogic.NewCreateResourceLogic(store, contentAuditor).UpdateResourceDraft(r.Context(), resourceID, req)
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
		resp, err := resourcelogic.NewSubmitResourceLogic(store, contentAuditor).SubmitResource(r.Context(), resourceID)
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
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/reports", func(w http.ResponseWriter, r *http.Request) {
		reportStore, ok := any(store).(resourcelogic.ResourceReportStore)
		if !ok {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "举报服务暂不可用"))
			return
		}
		var body struct {
			UserID     string        `json:"userId"`
			ReasonCode string        `json:"reasonCode"`
			ReasonText string        `json:"reasonText"`
			Evidence   model.JSONMap `json:"evidence"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		reporterUserID := strings.TrimSpace(body.UserID)
		if tokenService != nil {
			subject, err := userSubjectFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后举报"))
				return
			}
			reporterUserID = subject.UserID
		}
		resp, err := resourcelogic.NewReportResourceLogic(reportStore).ReportResource(r.Context(), resourcelogic.ReportResourceReq{
			ResourceID:     r.PathValue("resourceId"),
			ReporterUserID: reporterUserID,
			ReasonCode:     body.ReasonCode,
			ReasonText:     body.ReasonText,
			Evidence:       body.Evidence,
		})
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
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/contact-unlock-orders", func(w http.ResponseWriter, r *http.Request) {
		subject, err := userSubjectFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后查看联系方式"))
			return
		}
		var body struct {
			Action           string `json:"action"`
			ViewerMerchantID string `json:"viewerMerchantId"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if strings.TrimSpace(body.ViewerMerchantID) != "" && permissionStore != nil {
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, body.ViewerMerchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := resourcelogic.NewCreateContactUnlockOrderLogic(store).CreateContactUnlockOrder(r.Context(), resourcelogic.CreateContactUnlockOrderReq{
			ResourceID:       r.PathValue("resourceId"),
			UserID:           subject.UserID,
			ViewerMerchantID: body.ViewerMerchantID,
			Action:           body.Action,
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/resources/{resourceId}/contact-unlock-orders/{orderId}/payment", func(w http.ResponseWriter, r *http.Request) {
		subject, err := userSubjectFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录后支付"))
			return
		}
		var body paymentlogic.CreateContactUnlockPaymentReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := paymentlogic.NewCreateContactUnlockPaymentLogic(store, wechatPayGateway, wechatPayDevMock).CreateContactUnlockPayment(r.Context(), paymentlogic.CreateContactUnlockPaymentReq{
			ResourceID: r.PathValue("resourceId"),
			OrderID:    r.PathValue("orderId"),
			UserID:     subject.UserID,
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
			Direction:  query.Get("direction"),
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
	mux.HandleFunc("GET /api/v1/admin/resources", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewListPendingResourcesLogic(store).ListAdminResources(r.Context(), adminlogic.ListPendingResourcesReq{
			CityCode: query.Get("cityCode"),
			TypeCode: query.Get("typeCode"),
			Status:   query.Get("status"),
			Page:     int64FromQuery(r, "page"),
			PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/resource-reports", func(w http.ResponseWriter, r *http.Request) {
		reportStore, ok := any(store).(adminlogic.ResourceReportAdminStore)
		if !ok {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "举报审核服务暂不可用"))
			return
		}
		query := r.URL.Query()
		resp, err := adminlogic.NewResourceReportLogic(reportStore).ListResourceReports(r.Context(), adminlogic.ListResourceReportsReq{
			Status:   query.Get("status"),
			Page:     int64FromQuery(r, "page"),
			PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/resource-reports/{reportId}/review", func(w http.ResponseWriter, r *http.Request) {
		reportStore, ok := any(store).(adminlogic.ResourceReportAdminStore)
		if !ok {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "举报审核服务暂不可用"))
			return
		}
		var body struct {
			Action             string `json:"action"`
			ResourceAction     string `json:"resourceAction"`
			Reason             string `json:"reason"`
			ReviewerID         string `json:"reviewerId"`
			RefundPublishQuota bool   `json:"refundPublishQuota"`
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
		resp, err := adminlogic.NewResourceReportLogic(reportStore).ReviewResourceReport(r.Context(), r.PathValue("reportId"), adminlogic.ReviewResourceReportReq{
			Action:             body.Action,
			ResourceAction:     body.ResourceAction,
			Reason:             body.Reason,
			ReviewerID:         body.ReviewerID,
			RefundPublishQuota: body.RefundPublishQuota,
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
		Direction    string                           `json:"direction"`
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
		Direction: body.Direction, Title: body.Title, Category: body.Category, District: body.District, PriceText: body.PriceText,
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
		CityCode:   query.Get("cityCode"),
		MerchantID: query.Get("merchantId"),
		GroupCode:  query.Get("groupCode"),
		TypeCode:   query.Get("typeCode"),
		Direction:  query.Get("direction"),
		Keyword:    query.Get("keyword"),
		Category:   query.Get("category"),
		Tags:       resourceTagsFromQuery(query),
		Page:       int64FromQuery(r, "page"),
		PageSize:   int64FromQuery(r, "pageSize"),
	}
}

func searchResourcesReqFromQuery(r *http.Request) resourcelogic.SearchResourcesReq {
	req := listResourcesReqFromQuery(r)
	return resourcelogic.SearchResourcesReq{
		UserID:    r.URL.Query().Get("userId"),
		CityCode:  req.CityCode,
		GroupCode: req.GroupCode,
		TypeCode:  req.TypeCode,
		Direction: req.Direction,
		Keyword:   req.Keyword,
		Category:  req.Category,
		Tags:      append([]string(nil), req.Tags...),
		Page:      req.Page,
		PageSize:  req.PageSize,
	}
}

func resourceTagsFromQuery(query url.Values) []string {
	rawValues := query["tags"]
	if len(rawValues) == 0 {
		return nil
	}
	tags := make([]string, 0, len(rawValues))
	for _, rawValue := range rawValues {
		for _, item := range strings.FieldsFunc(rawValue, func(r rune) bool {
			return r == ',' || r == '，'
		}) {
			if tag := strings.TrimSpace(item); tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
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
	if !ok || strings.TrimSpace(subject.OperatorID) == "" {
		return "", errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	return strings.TrimSpace(subject.OperatorID), nil
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
		req.CreatedByOperator = strings.TrimSpace(subject.OperatorID)
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
