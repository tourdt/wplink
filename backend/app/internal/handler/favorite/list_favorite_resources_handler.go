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

func ListFavoriteResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := requiredFavoriteUser(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListInteractionReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "收藏资源查询参数格式不正确"))
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(svcCtx.APIStore).ListFavoriteResources(r.Context(), userID, favoritelogic.ListInteractionReq{Page: req.Page, PageSize: req.PageSize})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		items := make([]types.ResourceListItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.ResourceListItem{
				Id: item.ID, Direction: item.Direction, TypeCode: item.TypeCode, TypeName: item.TypeName,
				Title: item.Title, Category: item.Category, CoverUrl: item.CoverURL, District: item.District,
				PriceText: item.PriceText, QuantityText: item.QuantityText, Tags: append([]string{}, item.Tags...),
				Merchant:   types.ResourceMerchantBrief{Id: item.Merchant.ID, Name: item.Merchant.Name, VipStatus: item.Merchant.VIPStatus},
				CreditTags: append([]string{}, item.CreditTags...), RefreshedAt: item.RefreshedAt, DealtAt: item.DealtAt,
			})
		}
		response.JSON(w, types.ListResourcesResp{Items: items, Page: resp.Page, PageSize: resp.PageSize, Total: resp.Total}, nil)
	}
}
