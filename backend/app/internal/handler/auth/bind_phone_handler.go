package auth

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func BindPhoneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.BindPhoneReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "手机号绑定参数格式不正确"))
			return
		}
		resp, err := authlogic.NewMeLogic(svcCtx.APIStore, svcCtx.SMSVerifier).BindPhone(r.Context(), subject.UserID, authlogic.BindPhoneReq{
			Phone:   req.Phone,
			SmsCode: req.SmsCode,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.BindPhoneResp{Id: resp.ID, Phone: resp.Phone}, nil)
	}
}
