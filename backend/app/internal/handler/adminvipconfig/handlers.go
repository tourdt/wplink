package adminvipconfig

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

const (
	vipConfigPlans = iota
	vipConfigQuotaPacks
	vipConfigPromotions
)

func adminListVIPConfigHTTPHandler(svcCtx *svc.ServiceContext, kind int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminVIPContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		logic := adminlogic.NewVIPConfigAdminLogic(store)
		switch kind {
		case vipConfigPlans:
			resp, err := logic.ListVIPPlans(r.Context())
			response.JSON(w, resp, err)
		case vipConfigQuotaPacks:
			resp, err := logic.ListQuotaPacks(r.Context())
			response.JSON(w, resp, err)
		default:
			resp, err := logic.ListVIPPromotions(r.Context())
			response.JSON(w, resp, err)
		}
	}
}

func adminSaveVIPPlanHTTPHandler(svcCtx *svc.ServiceContext, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminVIPContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveVIPPlanConfigReq
		if err := parseAdminVIPRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		pathCode := ""
		if update {
			pathCode = pathvar.Vars(r)["planCode"]
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveVIPPlan(r.Context(), pathCode, adminlogic.SaveVIPPlanConfigReq{
			Code: req.Code, Name: req.Name, DurationMonths: req.DurationMonths,
			StandardPriceCent: req.StandardPriceCent, Status: req.Status, DisplayOrder: req.DisplayOrder,
			Benefits: mapAdminVIPBenefits(req.Benefits),
		}, admin.OperatorID)
		response.JSON(w, resp, err)
	}
}

func adminSaveQuotaPackHTTPHandler(svcCtx *svc.ServiceContext, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminVIPContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveQuotaPackConfigReq
		if err := parseAdminVIPRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		pathCode := ""
		if update {
			pathCode = pathvar.Vars(r)["packCode"]
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveQuotaPack(r.Context(), pathCode, adminlogic.SaveQuotaPackConfigReq{
			Code: req.Code, Name: req.Name, Description: req.Description,
			StandardPriceCent: req.StandardPriceCent, SalePriceCent: req.SalePriceCent, SaleLabel: req.SaleLabel,
			Status: req.Status, DisplayOrder: req.DisplayOrder, Benefits: mapAdminVIPBenefits(req.Benefits),
		}, admin.OperatorID)
		response.JSON(w, resp, err)
	}
}

func adminSaveVIPPromotionHTTPHandler(svcCtx *svc.ServiceContext, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminVIPContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveVIPPromotionConfigReq
		if err := parseAdminVIPRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		pathCode := ""
		if update {
			pathCode = pathvar.Vars(r)["promotionCode"]
		}
		resp, err := adminlogic.NewVIPConfigAdminLogic(store).SaveVIPPromotion(r.Context(), pathCode, adminlogic.SaveVIPPromotionConfigReq{
			Code: req.Code, PlanCode: req.PlanCode, PromotionType: req.PromotionType,
			SalePriceCent: req.SalePriceCent, StartsAt: req.StartsAt, EndsAt: req.EndsAt,
			QuotaLimit: req.QuotaLimit, Status: req.Status,
		}, admin.OperatorID)
		response.JSON(w, resp, err)
	}
}

func mapAdminVIPBenefits(req types.AdminVIPBenefitConfig) adminlogic.VIPBenefitConfig {
	return adminlogic.VIPBenefitConfig{
		PublishPolicy: req.PublishPolicy, PublishQuota: req.PublishQuota, RefreshQuota: req.RefreshQuota,
		TopVoucherCount: req.TopVoucherCount, TopDurationHours: req.TopDurationHours, HomepageImageLimit: req.HomepageImageLimit,
	}
}

func requireAdminVIPContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, adminlogic.VIPConfigAdminStore, error) {
	admin, err := handlerx.AdminFromContext(adminVIPRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.VIPModel == nil {
		logx.WithContext(adminVIPRequestContext(r)).Errorw("后台 VIP 配置 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
		return session.AdminTokenSubject{}, nil, errx.New(errx.CodeInternalError, "VIP 配置服务暂不可用，请稍后重试")
	}
	return admin, svcCtx.APIStore, nil
}

func parseAdminVIPRequest(r *http.Request, target any) error {
	if err := httpx.Parse(r, target); err != nil {
		return errx.New(errx.CodeValidationFailed, "VIP 配置参数格式不正确")
	}
	return nil
}

func adminVIPRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
