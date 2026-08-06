package admingrowth

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func adminListGrowthCampaignsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminGrowthContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListGrowthCampaignsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "活动筛选参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).ListGrowthCampaigns(r.Context(), req.Status)
		response.JSON(w, resp, err)
	}
}

func adminSaveGrowthCampaignHTTPHandler(svcCtx *svc.ServiceContext, usePathCode bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminGrowthContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SaveGrowthCampaignReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "活动配置参数格式不正确"))
			return
		}
		pathCode := ""
		if usePathCode {
			pathCode = pathvar.Vars(r)["campaignCode"]
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).SaveGrowthCampaign(r.Context(), pathCode, adminlogic.SaveGrowthCampaignReq{
			Code:           req.Code,
			Name:           req.Name,
			Status:         req.Status,
			StartsAt:       req.StartsAt,
			EndsAt:         req.EndsAt,
			ConfigSnapshot: model.JSONMap(req.ConfigSnapshot),
			DisableReason:  req.DisableReason,
		}, admin.OperatorID)
		response.JSON(w, resp, err)
	}
}

func adminListGrowthRulesHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminGrowthContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).ListGrowthRules(r.Context(), pathvar.Vars(r)["campaignCode"])
		response.JSON(w, resp, err)
	}
}

func adminSaveGrowthRuleHTTPHandler(svcCtx *svc.ServiceContext, usePathCode bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminGrowthContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SaveGrowthRuleReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "活动规则参数格式不正确"))
			return
		}
		vars := pathvar.Vars(r)
		pathRuleCode := ""
		if usePathCode {
			pathRuleCode = vars["ruleCode"]
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).SaveGrowthRule(r.Context(), vars["campaignCode"], pathRuleCode, adminlogic.SaveGrowthRuleReq{
			RuleCode:              req.RuleCode,
			RuleName:              req.RuleName,
			TriggerEvent:          req.TriggerEvent,
			Status:                req.Status,
			Priority:              req.Priority,
			Conditions:            model.JSONMap(req.Conditions),
			RewardType:            req.RewardType,
			RewardAmount:          req.RewardAmount,
			ValidDays:             req.ValidDays,
			PerUserLimit:          req.PerUserLimit,
			PerUserDailyLimit:     req.PerUserDailyLimit,
			PerResourceDailyLimit: req.PerResourceDailyLimit,
			Description:           req.Description,
		}, admin.OperatorID)
		response.JSON(w, resp, err)
	}
}

func adminListGrowthRewardGrantsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminGrowthContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListGrowthRewardGrantsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "发放记录筛选参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewGrowthCampaignAdminLogic(store).ListGrowthRewardGrants(r.Context(), model.AdminGrowthRewardGrantFilter{
			CampaignCode: pathvar.Vars(r)["campaignCode"],
			RuleCode:     req.RuleCode,
			MerchantID:   req.MerchantId,
			Status:       req.Status,
			PageSize:     req.PageSize,
		})
		response.JSON(w, resp, err)
	}
}

func requireAdminGrowthContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, adminlogic.GrowthCampaignAdminStore, error) {
	admin, err := handlerx.AdminFromContext(adminGrowthRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.GrowthCampaignModel == nil {
		logx.WithContext(adminGrowthRequestContext(r)).Errorw("后台增长活动 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
		return session.AdminTokenSubject{}, nil, errx.New(errx.CodeInternalError, "增长活动管理服务暂不可用，请稍后重试")
	}
	return admin, svcCtx.APIStore, nil
}

func adminGrowthRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
