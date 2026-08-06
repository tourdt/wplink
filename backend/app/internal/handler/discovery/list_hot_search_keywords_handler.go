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

func ListHotSearchKeywordsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HotSearchKeywordsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "热词查询参数格式不正确"))
			return
		}
		resp, err := discoverylogic.NewHotSearchKeywordDiscoveryLogic(svcCtx.APIStore).ListHotSearchKeywords(r.Context(), discoverylogic.ListHotSearchKeywordsReq{
			CityCode: req.CityCode,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}

		items := make([]types.HotSearchKeywordItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.HotSearchKeywordItem{Keyword: item.Keyword})
		}
		response.JSON(w, types.HotSearchKeywordsResp{Items: items}, nil)
	}
}
