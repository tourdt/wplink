package adminbannertopic

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

func adminListBannerTopicsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminBannerContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListBannerTopicsReq
		if err := parseAdminBannerRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewBannerTopicAdminLogic(store).ListBannerTopics(r.Context(), adminlogic.ListBannerTopicsReq{
			CityCode: req.CityCode, Kind: req.Kind, Status: req.Status,
		})
		if err != nil {
			adminlogic.LogAdminFailure(r.Context(), "加载后台 Banner 配置失败", "list_banner_topics", err,
				logx.Field("operatorId", admin.OperatorID), logx.Field("cityFiltered", req.CityCode != ""),
				logx.Field("kindFiltered", req.Kind != ""), logx.Field("statusFiltered", req.Status != ""))
		}
		response.JSON(w, resp, err)
	}
}

func adminSaveBannerTopicHTTPHandler(svcCtx *svc.ServiceContext, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminBannerContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveBannerTopicReq
		if err := parseAdminBannerRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		input := adminlogic.SaveBannerTopicReq{
			CityCode: req.CityCode, Kind: req.Kind, Title: req.Title, Subtitle: req.Subtitle,
			CoverURL: req.CoverUrl, TypeScope: req.TypeScope, JumpType: req.JumpType, JumpTarget: req.JumpTarget,
			Tags: req.Tags, StartAt: req.StartAt, EndAt: req.EndAt, SortOrder: req.SortOrder, Status: req.Status,
		}
		logic := adminlogic.NewBannerTopicAdminLogic(store)
		var resp adminlogic.SaveBannerTopicResp
		if update {
			resp, err = logic.UpdateBannerTopic(r.Context(), pathvar.Vars(r)["configId"], input)
		} else {
			resp, err = logic.CreateBannerTopic(r.Context(), input)
		}
		if err != nil {
			operation := "create_banner_topic"
			if update {
				operation = "update_banner_topic"
			}
			adminlogic.LogAdminFailure(r.Context(), "保存后台 Banner 配置失败", operation, err,
				logx.Field("operatorId", admin.OperatorID), logx.Field("configId", pathvar.Vars(r)["configId"]),
				logx.Field("kind", input.Kind))
		}
		response.JSON(w, resp, err)
	}
}

func requireAdminBannerContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, adminlogic.BannerTopicAdminStore, error) {
	admin, err := handlerx.AdminFromContext(adminBannerRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.BannerTopicModel == nil {
		logx.WithContext(adminBannerRequestContext(r)).Errorw("后台 Banner Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
		return session.AdminTokenSubject{}, nil, errx.New(errx.CodeInternalError, "Banner 管理服务暂不可用，请稍后重试")
	}
	return admin, svcCtx.APIStore, nil
}

func parseAdminBannerRequest(r *http.Request, target any) error {
	if err := httpx.Parse(r, target); err != nil {
		return errx.New(errx.CodeValidationFailed, "Banner 配置参数格式不正确")
	}
	return nil
}

func adminBannerRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
