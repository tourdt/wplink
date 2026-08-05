package message

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	messagelogic "wplink/backend/app/internal/logic/message"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListMessagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireMessageHandlerDependencies(r, svcCtx); err != nil {
			response.JSON(w, nil, err)
			return
		}
		subject, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ListMessagesReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "消息查询参数格式不正确"))
			return
		}
		roleCode := r.URL.Query().Get("roleCode")
		roleCodes, err := messageRoleCodesForTokenUser(r, svcCtx, subject.UserID, roleCode)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := messagelogic.NewListMessagesLogic(svcCtx.APIStore).ListMessages(r.Context(), messagelogic.ListMessagesReq{
			UserID: subject.UserID, RoleCode: roleCode, RoleCodes: roleCodes,
			Type: req.Type, Status: req.Status, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		items := make([]types.MessageListItem, 0, len(resp.Items))
		for _, item := range resp.Items {
			items = append(items, types.MessageListItem{
				Id: item.ID, MessageType: item.MessageType, TriggerId: item.TriggerID, Title: item.Title,
				Content: item.Content, TargetUrl: item.TargetURL, Status: item.Status, CreatedAt: item.CreatedAt,
			})
		}
		response.JSON(w, types.ListMessagesResp{Items: items, Page: resp.Page, PageSize: resp.PageSize, Total: resp.Total}, nil)
	}
}
