package adminhotsearchkeyword

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func adminListHotSearchKeywordsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminHotSearchContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListHotSearchKeywordsReq
		if err := parseAdminHotSearchRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewHotSearchKeywordAdminLogic(store).ListHotSearchKeywords(r.Context(), adminlogic.ListHotSearchKeywordsReq{
			CityCode: req.CityCode, Status: req.Status,
		})
		response.JSON(w, resp, err)
	}
}

func adminSaveHotSearchKeywordHTTPHandler(svcCtx *svc.ServiceContext, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminHotSearchContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveHotSearchKeywordReq
		if err := parseAdminHotSearchRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		input := adminlogic.SaveHotSearchKeywordReq{
			CityCode: req.CityCode, Keyword: req.Keyword, SortOrder: req.SortOrder,
			Status: req.Status, StartAt: req.StartAt, EndAt: req.EndAt,
		}
		logic := adminlogic.NewHotSearchKeywordAdminLogic(store)
		var resp adminlogic.SaveHotSearchKeywordResp
		if update {
			resp, err = logic.UpdateHotSearchKeyword(r.Context(), pathvar.Vars(r)["configId"], input)
		} else {
			resp, err = logic.CreateHotSearchKeyword(r.Context(), input)
		}
		response.JSON(w, resp, err)
	}
}

func requireAdminHotSearchContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, adminlogic.HotSearchKeywordAdminStore, error) {
	admin, err := handlerx.AdminFromContext(adminHotSearchRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.HotSearchKeywordModel == nil {
		logx.WithContext(adminHotSearchRequestContext(r)).Errorw("后台热词 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
		return session.AdminTokenSubject{}, nil, errx.New(errx.CodeInternalError, "热门搜索词服务暂不可用，请稍后重试")
	}
	return admin, svcCtx.APIStore, nil
}

func parseAdminHotSearchRequest(r *http.Request, target any) error {
	if err := httpx.Parse(r, target); err != nil {
		return errx.New(errx.CodeValidationFailed, "热门搜索词参数格式不正确")
	}
	return nil
}

func adminHotSearchRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
