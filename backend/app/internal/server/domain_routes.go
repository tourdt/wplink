package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	adminlogic "wplink/backend/app/internal/logic/admin"
	authlogic "wplink/backend/app/internal/logic/auth"
	demandlogic "wplink/backend/app/internal/logic/demand"
	discoverylogic "wplink/backend/app/internal/logic/discovery"
	entitlementlogic "wplink/backend/app/internal/logic/entitlement"
	favoritelogic "wplink/backend/app/internal/logic/favorite"
	merchantlogic "wplink/backend/app/internal/logic/merchant"
	messagelogic "wplink/backend/app/internal/logic/message"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	verificationlogic "wplink/backend/app/internal/logic/verification"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/task"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"
)

type MerchantAPIStore interface {
	merchantlogic.CreateMerchantStore
	merchantlogic.GetMerchantStore
	merchantlogic.UpdateMerchantStore
	adminlogic.MerchantAdminStore
}

type DemandAPIStore interface {
	demandlogic.CreateDemandStore
	demandlogic.MyDemandStore
	adminlogic.DemandAdminStore
}

type DiscoveryAPIStore interface {
	discoverylogic.BannerTopicDiscoveryStore
	adminlogic.BannerTopicAdminStore
}

type VerificationAPIStore interface {
	verificationlogic.VerificationStore
	adminlogic.VerificationAdminStore
}

type EntitlementAPIStore interface {
	entitlementlogic.EntitlementStore
	adminlogic.EntitlementAdminStore
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
	adminlogic.MatchCaseStore
	adminlogic.SearchLogStore
	task.ResourceLifecycleStore
}

func registerOptionalDomainRoutes(mux *http.ServeMux, store any, userTokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore, smsVerifier authlogic.SMSVerifier) {
	if merchantStore, ok := store.(MerchantAPIStore); ok {
		registerMerchantRoutes(mux, merchantStore, userTokenService, adminTokenService, permissionStore, smsVerifier)
	}
	if demandStore, ok := store.(DemandAPIStore); ok {
		registerDemandRoutes(mux, demandStore, userTokenService)
	}
	if discoveryStore, ok := store.(DiscoveryAPIStore); ok {
		registerDiscoveryRoutes(mux, discoveryStore)
	}
	if verificationStore, ok := store.(VerificationAPIStore); ok {
		registerVerificationRoutes(mux, verificationStore, userTokenService, adminTokenService, permissionStore)
	}
	if entitlementStore, ok := store.(EntitlementAPIStore); ok {
		registerEntitlementRoutes(mux, entitlementStore, userTokenService, adminTokenService, permissionStore)
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
		resp, err := merchantlogic.NewGetMerchantLogic(store).GetMerchant(r.Context(), r.PathValue("merchantId"))
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

func registerDemandRoutes(mux *http.ServeMux, store DemandAPIStore, tokenService authlogic.TokenService) {
	mux.HandleFunc("POST /api/v1/purchase-demands", func(w http.ResponseWriter, r *http.Request) {
		var body demandlogic.CreateDemandReq
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
		}
		resp, err := demandlogic.NewCreateDemandLogic(store).CreateDemand(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/me/purchase-demands", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		userID := query.Get("userId")
		if tokenService != nil {
			var err error
			userID, err = userIDFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := demandlogic.NewListMyDemandsLogic(store).ListMyDemands(r.Context(), userID, demandlogic.ListMyDemandsReq{
			Status: query.Get("status"), Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/purchase-demands", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewDemandAdminLogic(store).ListDemands(r.Context(), adminlogic.ListDemandsReq{
			CityCode: query.Get("cityCode"), DemandType: query.Get("demandType"), Status: query.Get("status"),
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/purchase-demands/{demandId}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewDemandAdminLogic(store).GetDemand(r.Context(), adminlogic.GetDemandReq{DemandID: r.PathValue("demandId")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/purchase-demands/{demandId}/status", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.UpdateDemandStatusReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.DemandID = r.PathValue("demandId")
		resp, err := adminlogic.NewDemandAdminLogic(store).UpdateDemandStatus(r.Context(), body)
		response.JSON(w, resp, err)
	})
}

func registerDiscoveryRoutes(mux *http.ServeMux, store DiscoveryAPIStore) {
	mux.HandleFunc("GET /api/v1/home/banners", func(w http.ResponseWriter, r *http.Request) {
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(store).ListHomeBanners(r.Context(), discoverylogic.ListHomeBannersReq{CityCode: r.URL.Query().Get("cityCode")})
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

func registerVerificationRoutes(mux *http.ServeMux, store VerificationAPIStore, tokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	mux.HandleFunc("POST /api/v1/merchants/{merchantId}/verifications", func(w http.ResponseWriter, r *http.Request) {
		var body verificationlogic.SubmitVerificationReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.MerchantID = r.PathValue("merchantId")
		if tokenService != nil {
			var err error
			body.ApplicantUserID, err = userIDFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, body.MerchantID); err != nil {
				response.JSON(w, nil, err)
				return
			}
		}
		resp, err := verificationlogic.NewSubmitVerificationLogic(store).SubmitVerification(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/verifications/latest", func(w http.ResponseWriter, r *http.Request) {
		resp, err := verificationlogic.NewGetLatestVerificationLogic(store).GetLatestVerification(r.Context(), r.PathValue("merchantId"))
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
		if tokenService != nil {
			subject, err := userSubjectFromBearerToken(r, tokenService)
			if err != nil {
				response.JSON(w, nil, err)
				return
			}
			userID = subject.UserID
			if merchantID, ok := merchantIDFromRoleCode(roleCode); ok {
				if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
					response.JSON(w, nil, err)
					return
				}
			}
		}
		resp, err := messagelogic.NewListMessagesLogic(store).ListMessages(r.Context(), messagelogic.ListMessagesReq{
			UserID: userID, RoleCode: roleCode, Type: query.Get("type"), Status: query.Get("status"),
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
			if merchantID, ok := merchantIDFromRoleCode(body.RoleCode); ok {
				if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, merchantID); err != nil {
					response.JSON(w, nil, err)
					return
				}
			}
		}
		body.MessageID = r.PathValue("messageId")
		resp, err := messagelogic.NewReadMessageLogic(store).ReadMessage(r.Context(), body)
		response.JSON(w, resp, err)
	})
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
	mux.HandleFunc("POST /api/v1/admin/resource-type-configs/{configId}", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.UpdateResourceTypeConfigReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).UpdateResourceTypeConfig(r.Context(), r.PathValue("configId"), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/match-cases", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.CreateMatchCaseReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, body.OperatorID); err != nil {
			response.JSON(w, nil, err)
			return
		} else {
			body.OperatorID = operatorID
		}
		resp, err := adminlogic.NewMatchCaseLogic(store).CreateMatchCase(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/match-cases", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewMatchCaseLogic(store).ListMatchCases(r.Context(), adminlogic.ListMatchCasesReq{Status: r.URL.Query().Get("status"), Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/match-cases/{matchCaseId}/status", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.UpdateMatchCaseStatusReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.MatchCaseID = r.PathValue("matchCaseId")
		if operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, body.OperatorID); err != nil {
			response.JSON(w, nil, err)
			return
		} else {
			body.OperatorID = operatorID
		}
		resp, err := adminlogic.NewMatchCaseLogic(store).UpdateMatchCaseStatus(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/match-cases/{matchCaseId}/resources", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OperatorID  string   `json:"operatorId"`
			ResourceIDs []string `json:"resourceIds"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, body.OperatorID); err != nil {
			response.JSON(w, nil, err)
			return
		} else {
			body.OperatorID = operatorID
		}
		resp, err := adminlogic.NewMatchCaseLogic(store).AddMatchCaseResources(r.Context(), adminlogic.AddMatchCaseResourcesReq{MatchCaseID: r.PathValue("matchCaseId"), OperatorID: body.OperatorID, ResourceIDs: body.ResourceIDs})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("POST /api/v1/admin/match-cases/{matchCaseId}/participants", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OperatorID             string   `json:"operatorId"`
			ParticipantMerchantIDs []string `json:"participantMerchantIds"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		if operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, body.OperatorID); err != nil {
			response.JSON(w, nil, err)
			return
		} else {
			body.OperatorID = operatorID
		}
		resp, err := adminlogic.NewMatchCaseLogic(store).AddMatchCaseParticipants(r.Context(), adminlogic.AddMatchCaseParticipantsReq{MatchCaseID: r.PathValue("matchCaseId"), OperatorID: body.OperatorID, MerchantIDs: body.ParticipantMerchantIDs})
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
