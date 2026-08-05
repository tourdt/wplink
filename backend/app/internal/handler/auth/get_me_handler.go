package auth

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/response"
)

func GetMeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := authlogic.NewMeLogic(svcCtx.APIStore).GetMe(r.Context(), subject.UserID)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.MeResp{
			Id:               resp.ID,
			Phone:            resp.Phone,
			Nickname:         resp.Nickname,
			DefaultCityCode:  resp.DefaultCityCode,
			Roles:            append([]string{}, resp.Roles...),
			ManagedMerchants: toManagedMerchantInfos(resp.ManagedMerchants),
		}, nil)
	}
}
