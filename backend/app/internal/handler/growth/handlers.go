package growth

import (
	"context"
	"fmt"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	growthlogic "wplink/backend/app/internal/logic/growth"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func listActiveGrowthCampaignsHTTPHandler(store growthlogic.CampaignPublicStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			logx.WithContext(growthRequestContext(r)).Error("增长活动公开 Handler 存储依赖未配置")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "增长活动服务暂不可用，请稍后重试"))
			return
		}
		resp, err := growthlogic.NewCampaignPublicLogic(store).ListActiveCampaigns(r.Context())
		if err != nil {
			// 下游错误可能携带数据库细节，只记录稳定操作名和错误类型。
			logx.WithContext(r.Context()).Errorw("查询公开增长活动失败", logx.Field("errorType", fmt.Sprintf("%T", err)))
		}
		response.JSON(w, resp, err)
	}
}

func getMerchantGrowthTasksHTTPHandler(store growthlogic.GrowthTaskStore, permissionDeps handlerx.MerchantPermissionDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			logx.WithContext(growthRequestContext(r)).Error("商家增长任务 Handler 存储依赖未配置")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "增长任务服务暂不可用，请稍后重试"))
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, permissionDeps, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := growthlogic.NewGrowthTaskLogic(store).GetGrowthTasks(r.Context(), merchantID)
		if err != nil {
			// 任务规则和奖励配置不写日志，避免完整规则或奖励信息进入日志系统。
			logx.WithContext(r.Context()).Errorw(
				"查询商家增长任务失败",
				logx.Field("merchantId", merchantID),
				logx.Field("errorType", fmt.Sprintf("%T", err)),
			)
		}
		response.JSON(w, resp, err)
	}
}

func publicGrowthStoreFromServiceContext(svcCtx *svc.ServiceContext) growthlogic.CampaignPublicStore {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.GrowthCampaignModel == nil {
		return nil
	}
	return svcCtx.APIStore
}

func growthTaskStoreFromServiceContext(svcCtx *svc.ServiceContext) growthlogic.GrowthTaskStore {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.GrowthCampaignModel == nil || svcCtx.APIStore.MerchantEntitlementModel == nil {
		return nil
	}
	return svcCtx.APIStore
}

func growthPermissionDepsFromServiceContext(svcCtx *svc.ServiceContext) handlerx.MerchantPermissionDeps {
	if svcCtx == nil {
		return handlerx.MerchantPermissionDeps{}
	}
	var permissionStore handlerx.MerchantPermissionStore
	if svcCtx.APIStore != nil && svcCtx.APIStore.UserModel != nil {
		permissionStore = svcCtx.APIStore
	}
	return handlerx.MerchantPermissionDeps{
		UserTokenService:  svcCtx.UserTokenService,
		AdminTokenService: svcCtx.AdminTokenService,
		Store:             permissionStore,
	}
}

func growthRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
