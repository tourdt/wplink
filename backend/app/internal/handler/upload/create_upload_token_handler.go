package upload

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	uploadlogic "wplink/backend/app/internal/logic/upload"
	"wplink/backend/app/internal/permission"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateUploadTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminSubject, isAdmin := handlerx.OptionalAdmin(r, svcCtx.AdminTokenService)
		if isAdmin {
			// 已识别为后台身份后必须继续校验后台角色，禁止把无权限管理员降级为普通用户身份。
			if !permission.CanAccessAdmin(adminSubject.Roles) {
				response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录"))
				return
			}
		} else if _, err := handlerx.RequiredUser(r, svcCtx.UserTokenService); err != nil {
			// 认证依赖故障必须保留 500 语义，便于监控和运维发现配置问题；
			// 只有无效登录态统一收敛为“请先登录”，且不能降级为匿名上传。
			if errx.CodeOf(err) != errx.CodeUnauthorized {
				response.JSON(w, nil, err)
				return
			}
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录"))
			return
		}

		var req types.CreateUploadTokenReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "上传参数格式不正确"))
			return
		}
		resp, err := svcCtx.UploadTokenService.CreateUploadToken(r.Context(), uploadlogic.CreateUploadTokenReq{
			Purpose:     req.Purpose,
			FileName:    req.FileName,
			ContentType: req.ContentType,
			FileSize:    req.FileSize,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.CreateUploadTokenResp{
			UploadToken:   resp.UploadToken,
			UploadUrl:     resp.UploadURL,
			PublicBaseUrl: resp.PublicBaseURL,
			ObjectKey:     resp.ObjectKey,
			ExpiresAt:     resp.ExpiresAt,
		}, nil)
	}
}
