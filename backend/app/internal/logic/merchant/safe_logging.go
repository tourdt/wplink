package merchant

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"wplink/backend/app/internal/model"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
)

// logMerchantFailure 不调用 err.Error：数据库 Detail/Message 可能携带整行电话、地址和请求参数。
// 通过上下文 trace、稳定阶段、错误类型及 SQLSTATE 定位故障，原始 cause 仍保留用于 errors.Is/As。
func logMerchantFailure(ctx context.Context, operation, merchantID string, err error, contextFields ...logx.LogField) {
	stage := "store"
	var operationErr *model.MerchantOperationError
	if errors.As(err, &operationErr) {
		stage = operationErr.Stage
	}
	category := "dependency"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		category = "timeout"
	case errors.Is(err, context.Canceled):
		category = "canceled"
	case errors.Is(err, sql.ErrNoRows):
		category = "not_found"
	case errors.Is(err, driver.ErrBadConn):
		category = "unavailable"
	}
	fields := []logx.LogField{
		logx.Field("operation", operation), logx.Field("merchantId", merchantID), logx.Field("stage", stage),
		logx.Field("category", category), logx.Field("errorType", fmt.Sprintf("%T", err)),
	}
	if operationErr != nil {
		fields = append(fields, logx.Field("durationMs", operationErr.DurationMs))
	}
	fields = append(fields, contextFields...)
	var databaseErr *pq.Error
	if errors.As(err, &databaseErr) {
		fields = append(fields, logx.Field("sqlState", string(databaseErr.Code)))
	}
	logx.WithContext(ctx).Errorw("商家操作失败", fields...)
}
