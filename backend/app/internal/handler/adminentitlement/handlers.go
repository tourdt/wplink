package adminentitlement

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func adminGrantMerchantEntitlementHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, err := handlerx.AdminFromContext(adminEntitlementRequestContext(r))
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantEntitlementModel == nil {
			logx.WithContext(adminEntitlementRequestContext(r)).Errorw("后台权益发放 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "权益管理服务暂不可用，请稍后重试"))
			return
		}
		var req types.AdminGrantEntitlementReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "权益发放参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewEntitlementAdminLogic(svcCtx.APIStore).GrantMerchantEntitlement(r.Context(), adminlogic.GrantEntitlementReq{
			MerchantID: pathvar.Vars(r)["merchantId"], OperatorID: admin.OperatorID,
			EntitlementType: req.EntitlementType, SourceType: req.SourceType, TotalAmount: req.TotalAmount,
			ExpiresAt: req.ExpiresAt, Reason: req.Reason,
		})
		response.JSON(w, resp, err)
	}
}

func adminEntitlementRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
