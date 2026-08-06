package favorite

import (
	"net/http"

	favoritelogic "wplink/backend/app/internal/logic/favorite"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListFollowedMerchantsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := requiredFavoriteUser(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListInteractionReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "关注商家查询参数格式不正确"))
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(svcCtx.APIStore).ListFollowedMerchants(r.Context(), userID, favoritelogic.ListInteractionReq{Page: req.Page, PageSize: req.PageSize})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		items := make([]types.FollowedMerchantItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.FollowedMerchantItem{
				Id: item.ID, Name: item.Name, MerchantType: item.MerchantType,
				MainCategories: append([]string{}, item.MainCategories...), LogoUrl: item.LogoUrl, FollowedAt: item.FollowedAt,
			})
		}
		response.JSON(w, types.ListFollowedMerchantsResp{Items: items, Page: resp.Page, PageSize: resp.PageSize, Total: resp.Total}, nil)
	}
}
