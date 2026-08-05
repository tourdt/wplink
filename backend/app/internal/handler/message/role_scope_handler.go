package message

import (
	"fmt"
	"net/http"
	"strings"

	"wplink/backend/app/internal/handler/handlerx"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

// messageRoleCodesForTokenUser 保留旧路由的收件范围语义：显式商家角色必须先验证管理权限，
// 未指定角色时只推导当前 Token 用户实际管理的商家，禁止客户端借 userId 或 roleCode 横向访问消息。
func messageRoleCodesForTokenUser(r *http.Request, svcCtx *svc.ServiceContext, userID string, explicitRoleCode string) ([]string, error) {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MessageModel == nil || svcCtx.APIStore.UserModel == nil || svcCtx.AdminTokenService == nil {
		logx.WithContext(r.Context()).Error("消息角色范围依赖未配置")
		return nil, errx.New(errx.CodeInternalError, "消息服务暂不可用，请稍后重试")
	}

	explicitRoleCode = strings.TrimSpace(explicitRoleCode)
	if explicitRoleCode != "" {
		if merchantID, ok := merchantIDFromRoleCode(explicitRoleCode); ok {
			if err := handlerx.RequireMerchant(r, handlerx.MerchantPermissionDeps{
				UserTokenService: svcCtx.UserTokenService, AdminTokenService: svcCtx.AdminTokenService, Store: svcCtx.APIStore,
			}, merchantID); err != nil {
				return nil, err
			}
		}
		return []string{explicitRoleCode}, nil
	}

	merchantIDs, err := svcCtx.APIStore.ListManagedMerchantIDs(r.Context(), userID)
	if err != nil {
		// 消息正文和原始存储错误均不进入日志；用户 ID 足以定位收件范围推导链路。
		logx.WithContext(r.Context()).Errorw("推导消息收件商家角色失败", logx.Field("userId", strings.TrimSpace(userID)), logx.Field("errorType", fmt.Sprintf("%T", err)))
		return nil, errx.New(errx.CodeInternalError, "消息加载失败，请稍后重试")
	}
	roleCodes := make([]string, 0, len(merchantIDs))
	for _, merchantID := range merchantIDs {
		merchantID = strings.TrimSpace(merchantID)
		if merchantID != "" {
			roleCodes = append(roleCodes, "merchant:"+merchantID)
		}
	}
	return roleCodes, nil
}

func merchantIDFromRoleCode(roleCode string) (string, bool) {
	const prefix = "merchant:"
	roleCode = strings.TrimSpace(roleCode)
	if !strings.HasPrefix(roleCode, prefix) {
		return "", false
	}
	merchantID := strings.TrimSpace(strings.TrimPrefix(roleCode, prefix))
	return merchantID, merchantID != ""
}
