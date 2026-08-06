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

func UpdateMerchantHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireMerchantHandlerStore(r, svcCtx, true); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, handlerx.MerchantPermissionDeps{
			UserTokenService: svcCtx.UserTokenService, AdminTokenService: svcCtx.AdminTokenService, Store: svcCtx.APIStore,
		}, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.UpdateMerchantReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "商家资料参数格式不正确"))
			return
		}
		resp, err := merchantlogic.NewUpdateMerchantLogic(svcCtx.APIStore, svcCtx.SMSVerifier).UpdateMerchant(r.Context(), merchantID, merchantlogic.UpdateMerchantReq{
			Name: req.Name, MainCategories: append([]string(nil), req.MainCategories...), MerchantType: req.MerchantType,
			Description: req.Description, LogoURL: req.LogoUrl, Images: append([]string(nil), req.Images...),
			ContactName: req.ContactName, ContactPhone: req.ContactPhone, ContactWechat: req.ContactWechat,
			AddressText: req.AddressText, Location: req.Location, SmsCode: req.SmsCode,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.UpdateMerchantResp{Id: resp.ID, UpdatedAt: resp.UpdatedAt}, nil)
	}
}
