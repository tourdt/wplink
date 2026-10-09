package model

// MerchantOperationError 标记商家事务失败的稳定阶段，不把底层错误正文带入外部日志。
// Unwrap 保留根因，用于 sql.ErrNoRows、超时及 PostgreSQL SQLSTATE 分类。
type MerchantOperationError struct {
	Stage      string
	Cause      error
	DurationMs int64
}

func (e *MerchantOperationError) Error() string { return "商家资料事务失败: " + e.Stage }
func (e *MerchantOperationError) Unwrap() error { return e.Cause }
