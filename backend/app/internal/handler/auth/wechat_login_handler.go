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

func WechatLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WechatLoginReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "微信登录参数格式不正确"))
			return
		}
		resp, err := authlogic.NewWechatLoginLogic(svcCtx.APIStore, svcCtx.UserTokenService, svcCtx.WechatSessionClient).WechatLogin(r.Context(), authlogic.WechatLoginReq{
			Code:                 req.Code,
			DefaultCityCode:      req.DefaultCityCode,
			AgreedToPolicies:     req.AgreedToPolicies,
			PrivacyPolicyVersion: req.PrivacyPolicyVersion,
			UserAgreementVersion: req.UserAgreementVersion,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.WechatLoginResp{
			Token:            resp.Token,
			User:             toAuthUserInfo(resp.User),
			ManagedMerchants: toManagedMerchantInfos(resp.ManagedMerchants),
		}, nil)
	}
}

func toAuthUserInfo(user authlogic.AuthUserInfo) types.AuthUserInfo {
	return types.AuthUserInfo{
		Id:              user.ID,
		Nickname:        user.Nickname,
		AvatarUrl:       user.AvatarURL,
		DefaultCityCode: user.DefaultCityCode,
		Roles:           append([]string{}, user.Roles...),
	}
}

func toManagedMerchantInfos(items []authlogic.ManagedMerchantInfo) []types.ManagedMerchantInfo {
	result := make([]types.ManagedMerchantInfo, 0, len(items))
	for _, item := range items {
		result = append(result, types.ManagedMerchantInfo{
			Id:            item.ID,
			Name:          item.Name,
			Role:          item.Role,
			ProfileStatus: item.ProfileStatus,
		})
	}
	return result
}
