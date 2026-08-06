package admin

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

// SafeAdminErrorCategory 只基于稳定 sentinel、context 状态和错误类型分类。
// 不读取 err.Error()，避免数据库连接串、Token 或完整请求配置被带入诊断日志。
func SafeAdminErrorCategory(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled)):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)):
		return "timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "unavailable"
	}
	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn) {
		return "unavailable"
	}
	if errx.CodeOf(err) == errx.CodeStateConflict || errors.Is(err, model.ErrAdminOperatorLoginNameExists) {
		return "conflict"
	}
	return "unknown"
}

// LogAdminFailure 统一输出可检索且不泄密的后台失败日志。调用方只能传稳定 ID、枚举或布尔摘要，
// 不得传请求正文、筛选原文、完整配置和 err.Error()。
func LogAdminFailure(ctx context.Context, message string, operation string, err error, fields ...logx.LogField) {
	baseFields := []logx.LogField{
		logx.Field("operation", operation),
		logx.Field("errorCategory", SafeAdminErrorCategory(ctx, err)),
		logx.Field("errorType", fmt.Sprintf("%T", err)),
	}
	logx.WithContext(ctx).Errorw(message, append(baseFields, fields...)...)
}
