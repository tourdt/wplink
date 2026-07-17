package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/logic/adminauth"
	authlogic "wplink/backend/app/internal/logic/auth"
	discoverylogic "wplink/backend/app/internal/logic/discovery"
	entitlementlogic "wplink/backend/app/internal/logic/entitlement"
	favoritelogic "wplink/backend/app/internal/logic/favorite"
	growthlogic "wplink/backend/app/internal/logic/growth"
	maplogic "wplink/backend/app/internal/logic/map"
	merchantlogic "wplink/backend/app/internal/logic/merchant"
	messagelogic "wplink/backend/app/internal/logic/message"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	verificationlogic "wplink/backend/app/internal/logic/verification"
	viplogic "wplink/backend/app/internal/logic/vip"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/task"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type MerchantAPIStore interface {
	merchantlogic.CreateMerchantStore
	merchantlogic.GetMerchantStore
	merchantlogic.UpdateMerchantStore
	adminlogic.MerchantAdminStore
}

type DiscoveryAPIStore interface {
	discoverylogic.BannerTopicDiscoveryStore
	discoverylogic.HotSearchKeywordDiscoveryStore
	adminlogic.BannerTopicAdminStore
	adminlogic.HotSearchKeywordAdminStore
}

type VerificationAPIStore interface {
	verificationlogic.VerificationStore
	adminlogic.VerificationAdminStore
}

type VerificationBillingAPIStore interface {
	adminlogic.VerificationBillingConfigStore
}

type VerificationPaymentAPIStore interface {
	paymentlogic.VerificationPaymentStore
}

const publicMerchantVerificationDisabledMessage = "商家资质服务已下线，请在资料设置中完善公开资料"

type ContactUnlockPaymentAPIStore interface {
	paymentlogic.ContactUnlockPaymentStore
}

type EntitlementAPIStore interface {
	entitlementlogic.EntitlementStore
	adminlogic.EntitlementAdminStore
}

type VIPAPIStore interface {
	viplogic.Store
	paymentlogic.VIPPaymentStore
	adminlogic.VIPConfigAdminStore
}

type GrowthCampaignAPIStore interface {
	adminlogic.GrowthCampaignAdminStore
}

type GrowthCampaignPublicAPIStore interface {
	growthlogic.CampaignPublicStore
}

type GrowthTaskAPIStore interface {
	growthlogic.GrowthTaskStore
}

type TopVoucherMerchantStore interface {
	GetTopVoucherMerchantID(ctx context.Context, voucherID string) (string, error)
}

type MessageAPIStore interface {
	messagelogic.Store
}

type MetricsQueryAPIStore interface {
	metricslogic.ResourceMetricsStore
	metricslogic.MerchantMetricsStore
}

type InteractionAPIStore interface {
	favoritelogic.InteractionStore
}

type AdminUtilityAPIStore interface {
	adminlogic.DashboardStore
	adminlogic.OperationLogStore
	adminlogic.ResourceTypeConfigStore
	adminlogic.SearchLogStore
	task.ResourceLifecycleStore
}

type AdminPermissionAPIStore interface {
	adminlogic.AdminPermissionStore
}

type MapAPIStore interface {
	maplogic.PublicStore
	maplogic.AdminStore
	maplogic.BindingStore
}

func registerOptionalDomainRoutes(mux *http.ServeMux, store any, userTokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore, smsVerifier authlogic.SMSVerifier, wechatPayGateway paymentlogic.WechatPayGateway, wechatPayDevMock bool) {
	if merchantStore, ok := store.(MerchantAPIStore); ok {
		registerMerchantRoutes(mux, merchantStore, userTokenService, adminTokenService, permissionStore, smsVerifier)
	}
	// 旧独立需求入口已从小程序和后台下线，这里不再注册旧 API，避免新客户端继续依赖已废弃流程。
	if discoveryStore, ok := store.(DiscoveryAPIStore); ok {
		registerDiscoveryRoutes(mux, discoveryStore)
	}
	if verificationStore, ok := store.(VerificationAPIStore); ok {
		paymentStore, _ := store.(VerificationPaymentAPIStore)
		registerVerificationRoutes(mux, verificationStore, paymentStore, userTokenService, adminTokenService, permissionStore, wechatPayGateway, wechatPayDevMock)
	}
	if contactUnlockPaymentStore, ok := store.(ContactUnlockPaymentAPIStore); ok {
		registerContactUnlockPaymentRoutes(mux, contactUnlockPaymentStore, wechatPayGateway)
	}
	if billingStore, ok := store.(VerificationBillingAPIStore); ok {
		registerVerificationBillingRoutes(mux, billingStore)
	}
	if entitlementStore, ok := store.(EntitlementAPIStore); ok {
		registerEntitlementRoutes(mux, entitlementStore, userTokenService, adminTokenService, permissionStore)
	}
	if vipStore, ok := store.(VIPAPIStore); ok {
		registerVIPRoutes(mux, vipStore, userTokenService, adminTokenService, permissionStore, wechatPayGateway, wechatPayDevMock)
	}
	if growthPublicStore, ok := store.(GrowthCampaignPublicAPIStore); ok {
		registerPublicGrowthCampaignRoutes(mux, growthPublicStore)
	}
	if growthTaskStore, ok := store.(GrowthTaskAPIStore); ok {
		registerGrowthTaskRoutes(mux, growthTaskStore, userTokenService, adminTokenService, permissionStore)
	}
	if growthStore, ok := store.(GrowthCampaignAPIStore); ok {
		registerGrowthCampaignRoutes(mux, growthStore, adminTokenService)
	}
	if messageStore, ok := store.(MessageAPIStore); ok {
		registerMessageRoutes(mux, messageStore, userTokenService, adminTokenService, permissionStore)
	}
	if metricsStore, ok := store.(MetricsQueryAPIStore); ok {
		registerMetricsRoutes(mux, metricsStore, userTokenService, adminTokenService, permissionStore)
	}
	if interactionStore, ok := store.(InteractionAPIStore); ok && userTokenService != nil {
		registerInteractionRoutes(mux, interactionStore, userTokenService)
	}
	if adminStore, ok := store.(AdminUtilityAPIStore); ok {
		registerAdminUtilityRoutes(mux, adminStore, adminTokenService)
	}
	if adminPermissionStore, ok := store.(AdminPermissionAPIStore); ok {
		registerAdminPermissionRoutes(mux, adminPermissionStore, adminTokenService)
	}
	if mapStore, ok := store.(MapAPIStore); ok {
		registerMapRoutes(mux, mapStore, userTokenService, adminTokenService, permissionStore)
	}
}

func registerAdminPermissionRoutes(mux *http.ServeMux, store AdminPermissionAPIStore, adminTokenService AdminTokenService) {
	newLogic := func() *adminlogic.AdminPermissionLogic {
		return adminlogic.NewAdminPermissionLogic(store, adminauth.BcryptPasswordHasher{})
	}
	mux.HandleFunc("GET /api/v1/admin/operators", func(w http.ResponseWriter, r *http.Request) {
		actor, err := adminPermissionActorFromRequest(r, adminTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		query := r.URL.Query()
		resp, err := newLogic().ListOperators(r.Context(), adminlogic.ListAdminOperatorsReq{
			Keyword:  query.Get("keyword"),
			Role:     query.Get("role"),
			Status:   query.Get("status"),
			Page:     int64FromQuery(r, "page"),
			PageSize: int64FromQuery(r, "pageSize"),
		}, actor)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/operators", func(w http.ResponseWriter, r *http.Request) {
		actor, err := adminPermissionActorFromRequest(r, adminTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body adminlogic.SaveAdminOperatorReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := newLogic().CreateOperator(r.Context(), body, actor)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/operators/{operatorId}", func(w http.ResponseWriter, r *http.Request) {
		actor, err := adminPermissionActorFromRequest(r, adminTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body adminlogic.SaveAdminOperatorReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := newLogic().UpdateOperator(r.Context(), r.PathValue("operatorId"), body, actor)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/operators/{operatorId}/status", func(w http.ResponseWriter, r *http.Request) {
		actor, err := adminPermissionActorFromRequest(r, adminTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body adminlogic.UpdateAdminOperatorStatusReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := newLogic().UpdateOperatorStatus(r.Context(), r.PathValue("operatorId"), body, actor)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/module-permissions", func(w http.ResponseWriter, r *http.Request) {
		actor, err := adminPermissionActorFromRequest(r, adminTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := newLogic().ListModulePermissions(r.Context(), actor)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/module-permissions/{roleCode}", func(w http.ResponseWriter, r *http.Request) {
		actor, err := adminPermissionActorFromRequest(r, adminTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body adminlogic.UpdateAdminRoleModulePermissionsReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := newLogic().UpdateRoleModulePermissions(r.Context(), r.PathValue("roleCode"), body, actor)
		response.JSON(w, resp, err)
	})
}

func adminPermissionActorFromRequest(r *http.Request, adminTokenService AdminTokenService) (adminlogic.AdminPermissionActor, error) {
	subject, ok := adminSubjectFromBearerToken(r, adminTokenService)
	if !ok {
		return adminlogic.AdminPermissionActor{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	return adminlogic.AdminPermissionActor{
		OperatorID: strings.TrimSpace(subject.OperatorID),
		Roles:      append([]string(nil), subject.Roles...),
	}, nil
}

func registerMerchantRoutes(mux *http.ServeMux, store MerchantAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore, smsVerifier authlogic.SMSVerifier) {
	mux.HandleFunc("POST /api/v1/merchants", func(w http.ResponseWriter, r *http.Request) {
		var body merchantlogic.CreateMerchantReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			var err error
			body.CreatorUserID, err = userIDFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := merchantlogic.NewCreateMerchantLogic(store).CreateMerchant(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}", func(w http.ResponseWriter, r *http.Request) {
		userID, err := optionalUserIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := merchantlogic.NewGetMerchantLogic(store).GetMerchant(r.Context(), r.PathValue("merchantId"), userID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/merchants/{merchantId}", func(w http.ResponseWriter, r *http.Request) {
		var body merchantlogic.UpdateMerchantReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := merchantlogic.NewUpdateMerchantLogic(store, smsVerifier).UpdateMerchant(r.Context(), merchantID, body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/merchants", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewMerchantAdminLogic(store).ListMerchants(r.Context(), adminlogic.ListMerchantsReq{
			CityCode: query.Get("cityCode"), MerchantType: query.Get("merchantType"), Status: query.Get("status"),
			Keyword: query.Get("keyword"),
			Page:    int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
}

func registerDiscoveryRoutes(mux *http.ServeMux, store DiscoveryAPIStore) {
	mux.HandleFunc("GET /api/v1/home/operation-config", func(w http.ResponseWriter, r *http.Request) {
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(store).GetHomeOperationConfig(r.Context(), discoverylogic.GetHomeOperationConfigReq{CityCode: r.URL.Query().Get("cityCode")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/home/resources", func(w http.ResponseWriter, r *http.Request) {
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(store).ListHomeResources(r.Context(), discoverylogic.ListHomeResourcesReq{CityCode: r.URL.Query().Get("cityCode")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/search/hot-keywords", func(w http.ResponseWriter, r *http.Request) {
		resp, err := discoverylogic.NewHotSearchKeywordDiscoveryLogic(store).ListHotSearchKeywords(r.Context(), discoverylogic.ListHotSearchKeywordsReq{CityCode: r.URL.Query().Get("cityCode")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/topics/{topicId}/resources", func(w http.ResponseWriter, r *http.Request) {
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(store).GetTopicResources(r.Context(), discoverylogic.TopicResourcesReq{
			TopicID: r.PathValue("topicId"), CityCode: r.URL.Query().Get("cityCode"),
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/webview/validate", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			URL string `json:"url"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(store).ValidateWebviewURL(r.Context(), discoverylogic.ValidateWebviewURLReq{URL: body.URL})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/banner-topics", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewBannerTopicAdminLogic(store).ListBannerTopics(r.Context(), adminlogic.ListBannerTopicsReq{CityCode: query.Get("cityCode"), Kind: query.Get("kind"), Status: query.Get("status")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/hot-search-keywords", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewHotSearchKeywordAdminLogic(store).ListHotSearchKeywords(r.Context(), adminlogic.ListHotSearchKeywordsReq{CityCode: query.Get("cityCode"), Status: query.Get("status")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/hot-search-keywords", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveHotSearchKeywordReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewHotSearchKeywordAdminLogic(store).CreateHotSearchKeyword(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/hot-search-keywords/{configId}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveHotSearchKeywordReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewHotSearchKeywordAdminLogic(store).UpdateHotSearchKeyword(r.Context(), r.PathValue("configId"), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/banner-topics", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveBannerTopicReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewBannerTopicAdminLogic(store).CreateBannerTopic(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/banner-topics/{configId}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveBannerTopicReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewBannerTopicAdminLogic(store).UpdateBannerTopic(r.Context(), r.PathValue("configId"), body)
		response.JSON(w, resp, err)
	})
}

func registerVerificationRoutes(mux *http.ServeMux, store VerificationAPIStore, paymentStore VerificationPaymentAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore, wechatPayGateway paymentlogic.WechatPayGateway, wechatPayDevMock bool) {
	mux.HandleFunc("POST /api/v1/merchants/{merchantId}/verifications", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if tokenService != nil {
			if _, err := userIDFromBearerToken(r, tokenService); err != nil {
				response.JSON(w, nil, err)
				return
			}
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		// 用户侧认证入口已下线，后台仍保留历史记录和内部审核能力，避免继续对外形成平台背书。
		logx.Infof("用户侧商家认证提交已下线: merchantId=%s", merchantID)
		response.JSON(w, nil, errx.New(errx.CodeForbidden, publicMerchantVerificationDisabledMessage))
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/verifications/latest", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := verificationlogic.NewGetLatestVerificationLogic(store).GetLatestVerification(r.Context(), merchantID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/verifications/pending", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewVerificationAdminLogic(store).ListPendingVerifications(r.Context(), adminlogic.ListPendingVerificationsReq{Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/verifications/{verificationId}/review", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.ReviewVerificationReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.VerificationID = r.PathValue("verificationId")
		if reviewerID, err := adminOperatorIDFromRequest(r, adminTokenService, body.ReviewerID); err != nil {
			response.JSON(w, nil, err)
			return
		} else {
			body.ReviewerID = reviewerID
		}
		resp, err := adminlogic.NewVerificationAdminLogic(store).ReviewVerification(r.Context(), body)
		response.JSON(w, resp, err)
	})
	if paymentStore == nil {
		return
	}
	mux.HandleFunc("POST /api/v1/merchants/{merchantId}/verifications/{verificationId}/payment", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if tokenService != nil {
			if _, err := userIDFromBearerToken(r, tokenService); err != nil {
				response.JSON(w, nil, err)
				return
			}
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		// 用户侧认证支付随认证入口一并下线，避免旧客户端继续购买具有背书含义的服务。
		logx.Infof("用户侧商家认证支付已下线: merchantId=%s verificationId=%s", merchantID, r.PathValue("verificationId"))
		response.JSON(w, nil, errx.New(errx.CodeForbidden, publicMerchantVerificationDisabledMessage))
	})
	mux.HandleFunc("POST /api/v1/wechat-pay/verification/notify", func(w http.ResponseWriter, r *http.Request) {
		body, err := readLimitedBody(r, 1<<20)
		if err != nil {
			writeWechatPayNotifyError(w)
			return
		}
		headers := map[string]string{}
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		resp, err := paymentlogic.NewWechatPayNotifyLogic(paymentStore, wechatPayGateway).HandleNotify(r.Context(), paymentlogic.WechatPayNotifyReq{Headers: headers, Body: body})
		if err != nil {
			writeWechatPayNotifyError(w)
			return
		}
		writeRawJSON(w, http.StatusOK, resp)
	})
}

func registerVerificationBillingRoutes(mux *http.ServeMux, store VerificationBillingAPIStore) {
	mux.HandleFunc("GET /api/v1/verification-billing", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, nil, errx.New(errx.CodeForbidden, publicMerchantVerificationDisabledMessage))
	})
	mux.HandleFunc("GET /api/v1/admin/verification-billing", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewVerificationBillingConfigLogic(store).GetVerificationBillingConfig(r.Context(), adminlogic.GetVerificationBillingConfigReq{CityCode: r.URL.Query().Get("cityCode")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/verification-billing", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.UpdateVerificationBillingConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if body.CityCode == "" {
			body.CityCode = r.URL.Query().Get("cityCode")
		}
		resp, err := adminlogic.NewVerificationBillingConfigLogic(store).UpdateVerificationBillingConfig(r.Context(), body)
		response.JSON(w, resp, err)
	})
}

func registerContactUnlockPaymentRoutes(mux *http.ServeMux, store ContactUnlockPaymentAPIStore, wechatPayGateway paymentlogic.WechatPayGateway) {
	mux.HandleFunc("POST /api/v1/wechat-pay/contact-unlock/notify", func(w http.ResponseWriter, r *http.Request) {
		body, err := readLimitedBody(r, 1<<20)
		if err != nil {
			writeWechatPayNotifyError(w)
			return
		}
		headers := map[string]string{}
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		resp, err := paymentlogic.NewContactUnlockWechatPayNotifyLogic(store, wechatPayGateway).HandleNotify(r.Context(), paymentlogic.WechatPayNotifyReq{Headers: headers, Body: body})
		if err != nil {
			writeWechatPayNotifyError(w)
			return
		}
		writeRawJSON(w, http.StatusOK, resp)
	})
}

func registerVIPRoutes(mux *http.ServeMux, store VIPAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore, wechatPayGateway paymentlogic.WechatPayGateway, wechatPayDevMock bool) {
	mux.HandleFunc("GET /api/v1/vip/plans", func(w http.ResponseWriter, r *http.Request) {
		resp, err := viplogic.NewListVIPPlansLogic(store).ListVIPPlans(r.Context())
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/vip/quota-packs", func(w http.ResponseWriter, r *http.Request) {
		resp, err := viplogic.NewListQuotaPacksLogic(store).ListQuotaPacks(r.Context())
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/vip/plans", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).ListVIPPlans(r.Context())
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/vip/plans", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveVIPPlanConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveVIPPlan(r.Context(), "", body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/vip/plans/{planCode}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveVIPPlanConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveVIPPlan(r.Context(), r.PathValue("planCode"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/vip/quota-packs", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).ListQuotaPacks(r.Context())
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/vip/quota-packs", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveQuotaPackConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveQuotaPack(r.Context(), "", body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/vip/quota-packs/{packCode}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveQuotaPackConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveQuotaPack(r.Context(), r.PathValue("packCode"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/vip/promotions", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).ListVIPPromotions(r.Context())
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/vip/promotions", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveVIPPromotionConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveVIPPromotion(r.Context(), "", body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/vip/promotions/{promotionCode}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveVIPPromotionConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveVIPPromotion(r.Context(), r.PathValue("promotionCode"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/vip", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := viplogic.NewGetMerchantVIPLogic(store).GetMerchantVIP(r.Context(), merchantID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/merchants/{merchantId}/vip/orders", func(w http.ResponseWriter, r *http.Request) {
		var body viplogic.CreateVIPOrderReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.MerchantID = r.PathValue("merchantId")
		if tokenService != nil {
			var err error
			body.UserID, err = userIDFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, body.MerchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := viplogic.NewCreateVIPOrderLogic(store).CreateVIPOrder(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/merchants/{merchantId}/vip/orders/{orderId}/payment", func(w http.ResponseWriter, r *http.Request) {
		var body paymentlogic.CreateVIPPaymentReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.MerchantID = r.PathValue("merchantId")
		body.OrderID = r.PathValue("orderId")
		if tokenService != nil {
			var err error
			body.UserID, err = userIDFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, body.MerchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := paymentlogic.NewCreateVIPPaymentLogic(store, wechatPayGateway, wechatPayDevMock).CreateVIPPayment(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/wechat-pay/vip/notify", func(w http.ResponseWriter, r *http.Request) {
		body, err := readLimitedBody(r, 1<<20)
		if err != nil {
			writeWechatPayNotifyError(w)
			return
		}
		headers := map[string]string{}
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		resp, err := paymentlogic.NewVIPWechatPayNotifyLogic(store, wechatPayGateway).HandleNotify(r.Context(), paymentlogic.WechatPayNotifyReq{Headers: headers, Body: body})
		if err != nil {
			writeWechatPayNotifyError(w)
			return
		}
		writeRawJSON(w, http.StatusOK, resp)
	})
}

func registerEntitlementRoutes(mux *http.ServeMux, store EntitlementAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/entitlements", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewListEntitlementsLogic(store).ListEntitlements(r.Context(), merchantID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/entitlements/{entitlementId}/usage-records", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewListEntitlementUsageRecordsLogic(store).ListUsageRecords(r.Context(), merchantID, r.PathValue("entitlementId"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/top-vouchers", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewListTopVouchersLogic(store).ListTopVouchers(r.Context(), merchantID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/top-vouchers/{voucherId}/redeem", func(w http.ResponseWriter, r *http.Request) {
		var body entitlementlogic.RedeemTopVoucherReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.VoucherID = r.PathValue("voucherId")
		merchantID := body.MerchantID
		if tokenService != nil {
			var err error
			merchantID, err = topVoucherMerchantIDFromStore(r.Context(), store, body.VoucherID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			body.MerchantID = merchantID
		}
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := entitlementlogic.NewRedeemTopVoucherLogic(store).RedeemTopVoucher(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/merchants/{merchantId}/entitlements", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.GrantEntitlementReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.MerchantID = r.PathValue("merchantId")
		if operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, body.OperatorID); err != nil {
			response.JSON(w, nil, err)
			return
		} else {
			body.OperatorID = operatorID
		}
		resp, err := adminlogic.NewEntitlementAdminLogic(store).GrantMerchantEntitlement(r.Context(), body)
		response.JSON(w, resp, err)
	})
}

func registerPublicGrowthCampaignRoutes(mux *http.ServeMux, store GrowthCampaignPublicAPIStore) {
	mux.HandleFunc("GET /api/v1/growth-campaigns/active", func(w http.ResponseWriter, r *http.Request) {
		resp, err := growthlogic.NewCampaignPublicLogic(store).ListActiveCampaigns(r.Context())
		response.JSON(w, resp, err)
	})
}

func registerGrowthTaskRoutes(mux *http.ServeMux, store GrowthTaskAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/growth-tasks", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := growthlogic.NewGrowthTaskLogic(store).GetGrowthTasks(r.Context(), merchantID)
		response.JSON(w, resp, err)
	})
}

func registerGrowthCampaignRoutes(mux *http.ServeMux, store GrowthCampaignAPIStore, adminTokenService AdminTokenService) {
	mux.HandleFunc("GET /api/v1/admin/growth-campaigns", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).ListGrowthCampaigns(r.Context(), r.URL.Query().Get("status"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/growth-campaigns", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveGrowthCampaignReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).SaveGrowthCampaign(r.Context(), "", body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/growth-campaigns/{campaignCode}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveGrowthCampaignReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).SaveGrowthCampaign(r.Context(), r.PathValue("campaignCode"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/growth-campaigns/{campaignCode}/rules", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).ListGrowthRules(r.Context(), r.PathValue("campaignCode"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/growth-campaigns/{campaignCode}/rules", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveGrowthRuleReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).SaveGrowthRule(r.Context(), r.PathValue("campaignCode"), "", body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/growth-campaigns/{campaignCode}/rules/{ruleCode}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.SaveGrowthRuleReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, "")
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).SaveGrowthRule(r.Context(), r.PathValue("campaignCode"), r.PathValue("ruleCode"), body, operatorID)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/growth-campaigns/{campaignCode}/grants", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).ListGrowthRewardGrants(r.Context(), model.AdminGrowthRewardGrantFilter{
			CampaignCode: r.PathValue("campaignCode"),
			RuleCode:     query.Get("ruleCode"),
			MerchantID:   query.Get("merchantId"),
			Status:       query.Get("status"),
			PageSize:     int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
}

func readLimitedBody(r *http.Request, limit int64) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, limit))
}

func writeRawJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeWechatPayNotifyError(w http.ResponseWriter) {
	writeRawJSON(w, http.StatusInternalServerError, map[string]string{
		"code":    "FAIL",
		"message": "处理失败",
	})
}

func registerInteractionRoutes(mux *http.ServeMux, store InteractionAPIStore, tokenService authlogic.TokenService) {
	mux.HandleFunc("GET /api/v1/me/favorite-resources", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(store).ListFavoriteResources(r.Context(), userID, favoritelogic.ListInteractionReq{
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/favorite-resources/{resourceId}", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(store).GetResourceFavoriteState(r.Context(), userID, r.PathValue("resourceId"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/me/favorite-resources/{resourceId}", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body favoritelogic.SetResourceFavoriteReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.ResourceID = r.PathValue("resourceId")
		resp, err := favoritelogic.NewInteractionLogic(store).SetResourceFavorite(r.Context(), userID, body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/followed-merchants", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(store).ListFollowedMerchants(r.Context(), userID, favoritelogic.ListInteractionReq{
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/followed-merchants/{merchantId}", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(store).GetMerchantFollowState(r.Context(), userID, r.PathValue("merchantId"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/me/followed-merchants/{merchantId}", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body favoritelogic.SetMerchantFollowReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.MerchantID = r.PathValue("merchantId")
		resp, err := favoritelogic.NewInteractionLogic(store).SetMerchantFollow(r.Context(), userID, body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/saved-searches", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(store).ListSavedSearches(r.Context(), userID, favoritelogic.ListInteractionReq{
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/me/saved-searches", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var body favoritelogic.CreateSavedSearchReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(store).CreateSavedSearch(r.Context(), userID, body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("DELETE /api/v1/me/saved-searches/{savedSearchId}", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromBearerToken(r, tokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(store).DeleteSavedSearch(r.Context(), userID, r.PathValue("savedSearchId"))
		response.JSON(w, resp, err)
	})
}

func topVoucherMerchantIDFromStore(ctx context.Context, store EntitlementAPIStore, voucherID string) (string, error) {
	ownerStore, ok := any(store).(TopVoucherMerchantStore)
	if !ok {
		return "", errx.New(errx.CodeForbidden, "您没有权限操作该商家")
	}
	merchantID, err := ownerStore.GetTopVoucherMerchantID(ctx, strings.TrimSpace(voucherID))
	if errors.Is(err, sql.ErrNoRows) {
		return "", errx.New(errx.CodeValidationFailed, "置顶券不存在")
	}
	if err != nil {
		return "", err
	}
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return "", errx.New(errx.CodeForbidden, "您没有权限操作该商家")
	}
	return merchantID, nil
}

func registerMessageRoutes(mux *http.ServeMux, store MessageAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	mux.HandleFunc("GET /api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		userID := query.Get("userId")
		roleCode := query.Get("roleCode")
		var roleCodes []string
		if tokenService != nil {
			subject, err := userSubjectFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			userID = subject.UserID
			roleCodes, err = messageRoleCodesForTokenUser(r, store, tokenService, adminTokenService, permissionStore, userID, roleCode)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := messagelogic.NewListMessagesLogic(store).ListMessages(r.Context(), messagelogic.ListMessagesReq{
			UserID: userID, RoleCode: roleCode, RoleCodes: roleCodes, Type: query.Get("type"), Status: query.Get("status"),
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/messages/{messageId}/read", func(w http.ResponseWriter, r *http.Request) {
		var body messagelogic.ReadMessageReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if tokenService != nil {
			var err error
			body.UserID, err = userIDFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			body.RoleCodes, err = messageRoleCodesForTokenUser(r, store, tokenService, adminTokenService, permissionStore, body.UserID, body.RoleCode)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		body.MessageID = r.PathValue("messageId")
		resp, err := messagelogic.NewReadMessageLogic(store).ReadMessage(r.Context(), body)
		response.JSON(w, resp, err)
	})
}

func messageRoleCodesForTokenUser(r *http.Request, store MessageAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore, userID string, explicitRoleCode string) ([]string, error) {
	explicitRoleCode = strings.TrimSpace(explicitRoleCode)
	if explicitRoleCode != "" {
		if merchantID, ok := merchantIDFromRoleCode(explicitRoleCode); ok {
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
				return nil, err
			}
		}
		return []string{explicitRoleCode}, nil
	}

	managedStore, ok := any(store).(ManagedMerchantStore)
	if !ok {
		return nil, nil
	}
	merchantIDs, err := managedStore.ListManagedMerchantIDs(r.Context(), userID)
	if err != nil {
		logx.Errorf("推导消息收件商家角色失败: userId=%s err=%+v", userID, err)
		return nil, err
	}
	roleCodes := make([]string, 0, len(merchantIDs))
	for _, merchantID := range merchantIDs {
		merchantID = strings.TrimSpace(merchantID)
		if merchantID == "" {
			continue
		}
		roleCodes = append(roleCodes, "merchant:"+merchantID)
	}
	return roleCodes, nil
}

func merchantIDFromRoleCode(roleCode string) (string, bool) {
	const prefix = "merchant:"
	roleCode = strings.TrimSpace(roleCode)
	if !strings.HasPrefix(roleCode, prefix) {
		return "", false
	}
	merchantID := strings.TrimSpace(strings.TrimPrefix(roleCode, prefix))
	return merchantID, merchantID != ""
}

func registerMetricsRoutes(mux *http.ServeMux, store MetricsQueryAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	mux.HandleFunc("GET /api/v1/resources/{resourceId}/metrics", func(w http.ResponseWriter, r *http.Request) {
		resourceID := r.PathValue("resourceId")
		if tokenService != nil {
			merchantID, err := merchantIDForResourceMetrics(r.Context(), store, resourceID)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := metricslogic.NewGetResourceMetricsLogic(store).GetResourceMetrics(r.Context(), metricslogic.GetResourceMetricsReq{
			ResourceID: resourceID, From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/metrics/summary", func(w http.ResponseWriter, r *http.Request) {
		merchantID := r.PathValue("merchantId")
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := metricslogic.NewGetMerchantMetricsLogic(store).GetMerchantMetrics(r.Context(), merchantID)
		response.JSON(w, resp, err)
	})
}

type resourceMetricsOwnerStore interface {
	GetPublishedResourceDetail(ctx context.Context, resourceID string) (model.ResourceDetail, error)
}

func merchantIDForResourceMetrics(ctx context.Context, store MetricsQueryAPIStore, resourceID string) (string, error) {
	ownerStore, ok := any(store).(resourceMetricsOwnerStore)
	if !ok {
		return "", errx.New(errx.CodeForbidden, "您没有权限查看该资源指标")
	}
	detail, err := ownerStore.GetPublishedResourceDetail(ctx, resourceID)
	if err != nil {
		return "", err
	}
	merchantID := strings.TrimSpace(detail.MerchantID)
	if merchantID == "" {
		return "", errx.New(errx.CodeForbidden, "您没有权限查看该资源指标")
	}
	return merchantID, nil
}

func registerAdminUtilityRoutes(mux *http.ServeMux, store AdminUtilityAPIStore, adminTokenService AdminTokenService) {
	mux.HandleFunc("GET /api/v1/admin/dashboard/overview", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewDashboardLogic(store).GetOverview(r.Context(), adminlogic.DashboardOverviewReq{CityCode: r.URL.Query().Get("cityCode")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/resource-type-configs", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).ListResourceTypeConfigs(r.Context(), adminlogic.ListResourceTypeConfigsReq{CityCode: r.URL.Query().Get("cityCode"), Status: r.URL.Query().Get("status")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/resource-type-configs", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.CreateResourceTypeConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).CreateResourceTypeConfig(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/resource-type-configs/{configId}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.UpdateResourceTypeConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).UpdateResourceTypeConfig(r.Context(), r.PathValue("configId"), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/operation-logs", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewOperationLogLogic(store).ListOperationLogs(r.Context(), adminlogic.OperationLogsReq{
			ObjectType: query.Get("objectType"), ObjectID: query.Get("objectId"), OperatorID: query.Get("operatorId"),
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/search-logs", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewSearchLogLogic(store).ListSearchLogs(r.Context(), adminlogic.SearchLogsReq{
			CityCode: query.Get("cityCode"), Keyword: query.Get("keyword"),
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/tasks/resource-lifecycle/run", func(w http.ResponseWriter, r *http.Request) {
		result, err := task.NewResourceLifecycleTask(store).Run(r.Context())
		response.JSON(w, map[string]int64{
			"expiredCount":          result.ExpiredCount,
			"expiringReminderCount": result.ExpiringReminderCount,
		}, err)
	})
}
