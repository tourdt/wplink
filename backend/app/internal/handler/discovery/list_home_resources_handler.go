package discovery

import (
	"net/http"

	discoverylogic "wplink/backend/app/internal/logic/discovery"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListHomeResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HomeResourcesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "首页资源查询参数格式不正确"))
			return
		}
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(svcCtx.APIStore).ListHomeResources(r.Context(), discoverylogic.ListHomeResourcesReq{
			CityCode: req.CityCode,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}

		items := make([]types.HomeResourceItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.HomeResourceItem{
				Id:           item.ID,
				Direction:    item.Direction,
				TypeCode:     item.TypeCode,
				TypeName:     item.TypeName,
				Title:        item.Title,
				Category:     item.Category,
				CoverUrl:     item.CoverURL,
				District:     item.District,
				PriceText:    item.PriceText,
				QuantityText: item.QuantityText,
				Merchant: types.HomeResourceMerchantBrief{
					Id:        item.Merchant.ID,
					Name:      item.Merchant.Name,
					VipStatus: item.Merchant.VIPStatus,
				},
				CreditTags:  append([]string{}, item.CreditTags...),
				RefreshedAt: item.RefreshedAt,
				DealtAt:     item.DealtAt,
			})
		}
		response.JSON(w, types.HomeResourcesResp{
			Items:    items,
			Page:     resp.Page,
			PageSize: resp.PageSize,
			Total:    resp.Total,
		}, nil)
	}
}
