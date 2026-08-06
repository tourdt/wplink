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

func GetHomeOperationConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HomeOperationConfigReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "首页运营配置查询参数格式不正确"))
			return
		}
		resp, err := discoverylogic.NewBannerTopicDiscoveryLogic(svcCtx.APIStore).GetHomeOperationConfig(r.Context(), discoverylogic.GetHomeOperationConfigReq{
			CityCode: req.CityCode,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}

		banners := make([]types.HomeBannerItem, 0, len(resp.Banners))
		for _, item := range resp.Banners {
			banners = append(banners, types.HomeBannerItem{
				Id:         item.ID,
				Title:      item.Title,
				Subtitle:   item.Subtitle,
				CoverUrl:   item.CoverURL,
				JumpType:   item.JumpType,
				JumpTarget: item.JumpTarget,
				Tags:       append([]string{}, item.Tags...),
			})
		}
		cards := make([]types.HomeRecommendCardItem, 0, len(resp.RecommendCards))
		for _, item := range resp.RecommendCards {
			cards = append(cards, types.HomeRecommendCardItem{
				Id:         item.ID,
				Tag:        item.Tag,
				Title:      item.Title,
				Subtitle:   item.Subtitle,
				JumpType:   item.JumpType,
				JumpTarget: item.JumpTarget,
			})
		}
		response.JSON(w, types.HomeOperationConfigResp{
			Banners:        banners,
			RecommendCards: cards,
		}, nil)
	}
}
