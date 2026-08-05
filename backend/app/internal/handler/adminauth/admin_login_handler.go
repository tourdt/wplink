package adminauth

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminauthlogic "wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func AdminLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminLoginReq
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "登录参数格式不正确"))
			return
		}
		resp, err := svcCtx.AdminLoginService.Login(r.Context(), adminauthlogic.LoginRequest{
			LoginName: req.LoginName,
			Password:  req.Password,
			ClientIP:  handlerx.ClientIP(r),
			UserAgent: r.UserAgent(),
		})
		if err != nil {
			code := errx.CodeUnauthorized
			if adminauthlogic.IsLoginRateLimited(err) {
				code = errx.CodeRateLimited
			}
			response.JSON(w, nil, errx.New(code, adminauthlogic.PublicLoginErrorMessage(err)))
			return
		}
		response.JSON(w, types.AdminLoginResp{
			Token:      resp.Token,
			OperatorId: resp.OperatorID,
			Roles:      append([]string{}, resp.Roles...),
			Modules:    append([]string{}, resp.Modules...),
		}, nil)
	}
}
