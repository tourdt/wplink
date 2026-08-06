package favorite

import (
	"net/http"

	favoritelogic "wplink/backend/app/internal/logic/favorite"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func DeleteSavedSearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := requiredFavoriteUser(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(svcCtx.APIStore).DeleteSavedSearch(r.Context(), userID, pathvar.Vars(r)["savedSearchId"])
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.DeleteSavedSearchResp{Message: resp.Message}, nil)
	}
}
