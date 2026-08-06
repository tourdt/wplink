package maplogic

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"

	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

// SafeMapErrorCategory 只基于稳定 sentinel、context 状态和安全错误码分类，
// 不读取 err.Error()，避免数据库连接串、Token、坐标或完整请求内容进入日志。
func SafeMapErrorCategory(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled)):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)):
		return "timeout"
	case errors.Is(err, sql.ErrNoRows):
		return "not_found"
	case errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn):
		return "unavailable"
	case errx.CodeOf(err) == errx.CodeStateConflict:
		return "conflict"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "unavailable"
	}
	return "unknown"
}

// LogMapDependencyFailure 统一记录地图正式路由链路的依赖失败。
// fields 只允许稳定 ID、枚举或数量摘要，调用方不得传入请求原文或 err.Error()。
func LogMapDependencyFailure(ctx context.Context, message string, operation string, err error, fields ...logx.LogField) {
	baseFields := []logx.LogField{
		logx.Field("operation", operation),
		logx.Field("errorCategory", SafeMapErrorCategory(ctx, err)),
	}
	logx.WithContext(ctx).Errorw(message, append(baseFields, fields...)...)
}
