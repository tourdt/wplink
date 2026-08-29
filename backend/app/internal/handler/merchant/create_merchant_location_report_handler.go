package merchant

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	merchantlogic "wplink/backend/app/internal/logic/merchant"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func CreateMerchantLocationReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx == nil {
			logx.WithContext(r.Context()).Error("商家位置反馈 Handler 服务上下文未配置")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "位置反馈服务暂不可用，请稍后重试"))
			return
		}
		subject, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantLocationReportModel == nil {
			logx.WithContext(r.Context()).Errorw("商家位置反馈 Handler 存储依赖未配置",
				logx.Field("merchantId", pathvar.Vars(r)["merchantId"]), logx.Field("reporterUserId", subject.UserID))
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "位置反馈服务暂不可用，请稍后重试"))
			return
		}
		var req types.CreateMerchantLocationReportReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "位置反馈参数格式不正确"))
			return
		}
		resp, err := merchantlogic.NewMerchantLocationReportLogic(svcCtx.APIStore).CreateMerchantLocationReport(
			r.Context(), pathvar.Vars(r)["merchantId"], subject.UserID,
			merchantlogic.CreateMerchantLocationReportReq{ReasonCode: req.ReasonCode, Description: req.Description},
		)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.CreateMerchantLocationReportResp{
			ReportId: resp.ReportID, Status: resp.Status, ActiveReporterCount: resp.ActiveReporterCount,
			AutoHidden: resp.AutoHidden, Idempotent: resp.Idempotent,
		}, nil)
	}
}
