package merchant

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	merchantlogic "wplink/backend/app/internal/logic/merchant"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func SetMerchantVisibilityHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireMerchantHandlerStore(r, svcCtx, true); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, handlerx.MerchantPermissionDeps{UserTokenService: svcCtx.UserTokenService, AdminTokenService: svcCtx.AdminTokenService, Store: svcCtx.APIStore}, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SetMerchantVisibilityReq
		if err := httpx.Parse(r, &req); err != nil || req.Visible == nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "请选择店铺展示状态"))
			return
		}
		id, err := merchantlogic.SetMerchantVisibility(r.Context(), svcCtx.APIStore, merchantID, req.Visible)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.SetMerchantVisibilityResp{MerchantId: id, OwnerVisible: *req.Visible}, nil)
	}
}
