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
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func ReadMessageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject, err := handlerx.RequiredUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req readMessageRoleReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "消息参数格式不正确"))
			return
		}
		roleCodes, err := messageRoleCodesForTokenUser(r, svcCtx, subject.UserID, req.RoleCode)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := messagelogic.NewReadMessageLogic(svcCtx.APIStore).ReadMessage(r.Context(), messagelogic.ReadMessageReq{
			UserID: subject.UserID, RoleCode: req.RoleCode, RoleCodes: roleCodes, MessageID: pathvar.Vars(r)["messageId"],
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.ReadMessageResp{Id: resp.ID, Status: resp.Status}, nil)
	}
}

type readMessageRoleReq struct {
	RoleCode string `json:"roleCode,optional"`
}
