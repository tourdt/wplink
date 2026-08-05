package auth

import (
	"net/http"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func SendSMSCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendSMSCodeReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "短信验证码参数格式不正确"))
			return
		}
		sender, _ := any(svcCtx.SMSVerifier).(authlogic.SMSCodeSender)
		resp, err := authlogic.NewSendSMSCodeLogic(sender).SendSMSCode(r.Context(), authlogic.SendSMSCodeReq{
			Phone: req.Phone,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.SendSMSCodeResp{Message: resp.Message}, nil)
	}
}
