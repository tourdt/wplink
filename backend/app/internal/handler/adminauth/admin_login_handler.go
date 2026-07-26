package adminauth

import (
	"net"
	"net/http"
	"strings"

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
			ClientIP:  requestClientIP(r),
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
			Roles:      append([]string(nil), resp.Roles...),
			Modules:    append([]string(nil), resp.Modules...),
		}, nil)
	}
}

func requestClientIP(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if remoteAddr := strings.TrimSpace(r.RemoteAddr); remoteAddr != "" {
		return remoteAddr
	}
	return "unknown"
}
