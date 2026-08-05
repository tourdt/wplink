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

func DeleteAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.DeleteAccountReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "账号注销参数格式不正确"))
			return
		}
		resp, err := authlogic.NewMeLogic(svcCtx.APIStore).DeleteAccount(r.Context(), subject.UserID, authlogic.DeleteAccountReq{
			Confirmation: req.Confirmation,
			Reason:       req.Reason,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.DeleteAccountResp{Message: resp.Message}, nil)
	}
}
