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

func ListHomeRecentMerchantsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HomeRecentMerchantsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "新入驻商家查询参数格式不正确"))
			return
		}
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(svcCtx.APIStore).ListHomeRecentMerchants(r.Context(), discoverylogic.ListHomeRecentMerchantsReq{
			CityCode: req.CityCode,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}

		items := make([]types.HomeRecentMerchantItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.HomeRecentMerchantItem{
				Id:             item.ID,
				Name:           item.Name,
				MerchantType:   item.MerchantType,
				MainCategories: append([]string{}, item.MainCategories...),
				LogoUrl:        item.LogoURL,
				AddressText:    item.AddressText,
				OnboardedAt:    item.OnboardedAt,
			})
		}
		response.JSON(w, types.HomeRecentMerchantsResp{Items: items}, nil)
	}
}
