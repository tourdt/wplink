package favorite

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	favoritelogic "wplink/backend/app/internal/logic/favorite"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateSavedSearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := requiredFavoriteUser(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.CreateSavedSearchReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "保存搜索参数格式不正确"))
			return
		}
		resp, err := favoritelogic.NewInteractionLogic(svcCtx.APIStore).CreateSavedSearch(r.Context(), userID, favoritelogic.CreateSavedSearchReq{
			Name: req.Name, CityCode: req.CityCode, TypeCode: req.TypeCode, Keyword: req.Keyword, Category: req.Category,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.SavedSearchResp{Id: resp.ID}, nil)
	}
}

func requiredFavoriteUser(r *http.Request, svcCtx *svc.ServiceContext) (string, error) {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.FavoriteModel == nil {
		logx.WithContext(r.Context()).Error("互动 Handler 存储依赖未配置")
		return "", errx.New(errx.CodeInternalError, "互动服务暂不可用，请稍后重试")
	}
	subject, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
	if err != nil {
		return "", err
	}
	return subject.UserID, nil
}
