package favorite

import (
	"net/http"

	favoritelogic "wplink/backend/app/internal/logic/favorite"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func SetMerchantFollowHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := requiredFavoriteUser(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.SetMerchantFollowReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "关注商家参数格式不正确"))
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(svcCtx.APIStore).SetMerchantFollow(r.Context(), userID, favoritelogic.SetMerchantFollowReq{
			MerchantID: pathvar.Vars(r)["merchantId"], Followed: req.Followed,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.MerchantFollowStateResp{MerchantId: resp.MerchantID, Followed: resp.Followed}, nil)
	}
}
