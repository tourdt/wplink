package handlerx

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/permission"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type MerchantPermissionStore interface {
	UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error)
}

type MerchantPermissionDeps struct {
	UserTokenService  authlogic.TokenService
	AdminTokenService AdminTokenService
	Store             MerchantPermissionStore
}

// RequireMerchant 允许平台管理员或有明确管理关系的用户操作商家。
func RequireMerchant(r *http.Request, deps MerchantPermissionDeps, merchantID string) error {
	ctx := requestContext(r)
	if deps.UserTokenService == nil || deps.AdminTokenService == nil || deps.Store == nil {
		// 认证或权限依赖缺失时必须拒绝请求，不能复用旧 Router 的 nil 放行逻辑。
		logx.WithContext(ctx).Errorw(
			"商家权限校验依赖未配置",
			logx.Field("userTokenServiceConfigured", deps.UserTokenService != nil),
			logx.Field("adminTokenServiceConfigured", deps.AdminTokenService != nil),
			logx.Field("permissionStoreConfigured", deps.Store != nil),
		)
		return errx.New(errx.CodeInternalError, "商家权限服务暂不可用，请稍后重试")
	}

	if admin, ok := OptionalAdmin(r, deps.AdminTokenService); ok && permission.CanAccessAdmin(admin.Roles) {
		return nil
	}
	user, err := RequiredUser(r, deps.UserTokenService)
	if err != nil {
		return err
	}
	if permission.CanAccessAdmin(user.Roles) {
		return nil
	}
	allowed, err := deps.Store.UserCanManageMerchant(ctx, strings.TrimSpace(user.UserID), strings.TrimSpace(merchantID))
	if err != nil {
		// 原始依赖错误可能包含数据库细节，只记录错误类型和定位权限关系所需的稳定标识。
		logx.WithContext(ctx).Errorw(
			"查询商家管理权限失败",
			logx.Field("userId", strings.TrimSpace(user.UserID)),
			logx.Field("merchantId", strings.TrimSpace(merchantID)),
			logx.Field("errorType", fmt.Sprintf("%T", err)),
		)
		return errx.New(errx.CodeInternalError, "商家权限校验失败，请稍后重试")
	}
	if !allowed {
		return errx.New(errx.CodeForbidden, "您没有权限操作该商家")
	}
	return nil
}
