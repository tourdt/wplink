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

func ListSavedSearchesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := requiredFavoriteUser(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListInteractionReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "保存搜索查询参数格式不正确"))
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(svcCtx.APIStore).ListSavedSearches(r.Context(), userID, favoritelogic.ListInteractionReq{Page: req.Page, PageSize: req.PageSize})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		items := make([]types.SavedSearchItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.SavedSearchItem{
				Id: item.ID, Name: item.Name, CityCode: item.CityCode, TypeCode: item.TypeCode,
				Keyword: item.Keyword, Category: item.Category, CreatedAt: item.CreatedAt,
			})
		}
		response.JSON(w, types.ListSavedSearchesResp{Items: items, Page: resp.Page, PageSize: resp.PageSize, Total: resp.Total}, nil)
	}
}
