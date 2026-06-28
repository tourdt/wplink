package server

import (
	"net/http"
	"strings"

	adminlogic "wplink/backend/app/internal/logic/admin"
	authlogic "wplink/backend/app/internal/logic/auth"
	demandlogic "wplink/backend/app/internal/logic/demand"
	discoverylogic "wplink/backend/app/internal/logic/discovery"
	entitlementlogic "wplink/backend/app/internal/logic/entitlement"
	merchantlogic "wplink/backend/app/internal/logic/merchant"
	messagelogic "wplink/backend/app/internal/logic/message"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	verificationlogic "wplink/backend/app/internal/logic/verification"
	"wplink/backend/app/internal/task"
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

type MessageAPIStore interface {
	messagelogic.Store
}

type MetricsQueryAPIStore interface {
	metricslogic.ResourceMetricsStore
	metricslogic.MerchantMetricsStore
}

type AdminUtilityAPIStore interface {
	adminlogic.DashboardStore
	adminlogic.OperationLogStore
	adminlogic.ResourceTypeConfigStore
	adminlogic.MatchCaseStore
	adminlogic.SearchLogStore
	task.ResourceLifecycleStore
}

func registerOptionalDomainRoutes(mux *http.ServeMux, store any, userTokenService authlogic.TokenService, adminTokenService AdminTokenService, permissionStore MerchantPermissionStore) {
	if merchantStore, ok := store.(MerchantAPIStore); ok {
		registerMerchantRoutes(mux, merchantStore)
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
		registerMetricsRoutes(mux, metricsStore)
	}
	if adminStore, ok := store.(AdminUtilityAPIStore); ok {
		registerAdminUtilityRoutes(mux, adminStore)
	}
}

func registerMerchantRoutes(mux *http.ServeMux, store MerchantAPIStore) {
	mux.HandleFunc("POST /api/v1/merchants", func(w http.ResponseWriter, r *http.Request) {
		var body merchantlogic.CreateMerchantReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := merchantlogic.NewCreateMerchantLogic(store).CreateMerchant(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := merchantlogic.NewGetMerchantLogic(store).GetMerchant(r.Context(), r.PathValue("merchantId"))
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("PATCH /api/v1/merchants/{merchantId}", func(w http.ResponseWriter, r *http.Request) {
		var body merchantlogic.UpdateMerchantReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := merchantlogic.NewUpdateMerchantLogic(store).UpdateMerchant(r.Context(), r.PathValue("merchantId"), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/merchants", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		resp, err := adminlogic.NewMerchantAdminLogic(store).ListMerchants(r.Context(), adminlogic.ListMerchantsReq{
			CityCode: query.Get("cityCode"), MerchantType: query.Get("merchantType"), Status: query.Get("status"),
			Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize"),
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
	mux.HandleFunc("PATCH /api/v1/admin/purchase-demands/{demandId}/status", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("PATCH /api/v1/admin/banner-topics/{configId}", func(w http.ResponseWriter, r *http.Request) {
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
		if err := requireMerchantPermission(r, tokenService, adminTokenService, permissionStore, body.MerchantID); err != nil {
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
		resp, err := adminlogic.NewEntitlementAdminLogic(store).GrantMerchantEntitlement(r.Context(), body)
		response.JSON(w, resp, err)
	})
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

func registerMetricsRoutes(mux *http.ServeMux, store MetricsQueryAPIStore) {
	mux.HandleFunc("GET /api/v1/resources/{resourceId}/metrics", func(w http.ResponseWriter, r *http.Request) {
		resp, err := metricslogic.NewGetResourceMetricsLogic(store).GetResourceMetrics(r.Context(), metricslogic.GetResourceMetricsReq{
			ResourceID: r.PathValue("resourceId"), From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"),
		})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/merchants/{merchantId}/metrics/summary", func(w http.ResponseWriter, r *http.Request) {
		resp, err := metricslogic.NewGetMerchantMetricsLogic(store).GetMerchantMetrics(r.Context(), r.PathValue("merchantId"))
		response.JSON(w, resp, err)
	})
}

func registerAdminUtilityRoutes(mux *http.ServeMux, store AdminUtilityAPIStore) {
	mux.HandleFunc("GET /api/v1/admin/dashboard/overview", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewDashboardLogic(store).GetOverview(r.Context(), adminlogic.DashboardOverviewReq{CityCode: r.URL.Query().Get("cityCode")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/resource-type-configs", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).ListResourceTypeConfigs(r.Context(), adminlogic.ListResourceTypeConfigsReq{CityCode: r.URL.Query().Get("cityCode"), Status: r.URL.Query().Get("status")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("PATCH /api/v1/admin/resource-type-configs/{configId}", func(w http.ResponseWriter, r *http.Request) {
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
		resp, err := adminlogic.NewMatchCaseLogic(store).CreateMatchCase(r.Context(), body)
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("GET /api/v1/admin/match-cases", func(w http.ResponseWriter, r *http.Request) {
		resp, err := adminlogic.NewMatchCaseLogic(store).ListMatchCases(r.Context(), adminlogic.ListMatchCasesReq{Status: r.URL.Query().Get("status"), Page: int64FromQuery(r, "page"), PageSize: int64FromQuery(r, "pageSize")})
		response.JSON(w, resp, err)
	})
	mux.HandleFunc("PATCH /api/v1/admin/match-cases/{matchCaseId}/status", func(w http.ResponseWriter, r *http.Request) {
		var body adminlogic.UpdateMatchCaseStatusReq
		if err := decodeJSONBody(r, &body); err != nil {
			response.JSON(w, nil, err)
			return
		}
		body.MatchCaseID = r.PathValue("matchCaseId")
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
