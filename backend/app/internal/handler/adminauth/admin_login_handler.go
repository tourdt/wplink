package adminauth

import (
	"net/http"

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
		})
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, err.Error()))
			return
		}
		response.JSON(w, types.AdminLoginResp{
			Token:  resp.Token,
			UserId: resp.UserID,
			Roles:  append([]string(nil), resp.Roles...),
		}, nil)
	}
}
