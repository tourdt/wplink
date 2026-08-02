package model

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MerchantMapEventsModel = (*customMerchantMapEventsModel)(nil)

type (
	// MerchantMapEventsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMerchantMapEventsModel.
	MerchantMapEventsModel interface {
		merchantMapEventsModel
		RecordMerchantMapEvent(ctx context.Context, input MerchantMapEventInput) error
		DeleteMerchantMapEventsBefore(ctx context.Context, cutoff time.Time) (int64, error)
		withSession(session sqlx.Session) MerchantMapEventsModel
	}

	customMerchantMapEventsModel struct {
		*defaultMerchantMapEventsModel
	}

	MerchantMapEventInput struct {
		UserID           string
		MerchantID       string
		TargetMerchantID string
		VisitorKey       string
		SessionID        string
		EventType        string
		Source           string
	}
)

// NewMerchantMapEventsModel returns a model for the database table.
func NewMerchantMapEventsModel(conn sqlx.SqlConn) MerchantMapEventsModel {
	return &customMerchantMapEventsModel{
		defaultMerchantMapEventsModel: newMerchantMapEventsModel(conn),
	}
}

func (m *customMerchantMapEventsModel) withSession(session sqlx.Session) MerchantMapEventsModel {
	return NewMerchantMapEventsModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customMerchantMapEventsModel) RecordMerchantMapEvent(ctx context.Context, input MerchantMapEventInput) error {
	query := fmt.Sprintf(`
INSERT INTO %s (
  user_id,
  merchant_id,
  target_merchant_id,
  visitor_key,
  session_id,
  event_type,
  source
)
VALUES (
  NULLIF($1, '')::bigint,
  $2::bigint,
  NULLIF($3, '')::bigint,
  $4,
  $5,
  $6,
  $7
)
`, m.table)
	// visitorKey 和 sessionId 属于匿名分析标识，不能交给 go-zero sqlx 的通用语句日志展开。
	// 这里只借用同一连接池直接执行参数化 SQL；不关闭 RawDB，也不在错误中拼接任何请求参数。
	rawDB, err := m.conn.RawDB()
	if err != nil {
		return fmt.Errorf("获取地图行为数据库连接失败: %w", err)
	}
	_, err = rawDB.ExecContext(ctx, query,
		input.UserID,
		input.MerchantID,
		input.TargetMerchantID,
		input.VisitorKey,
		input.SessionID,
		input.EventType,
		input.Source,
	)
	return err
}

func (m *customMerchantMapEventsModel) DeleteMerchantMapEventsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	// 事件只用于短期地图体验分析，按发生时间直接物理清理，避免 visitor/session 标识无限期留存。
	query := fmt.Sprintf(`DELETE FROM %s WHERE created_at < $1`, m.table)
	result, err := m.conn.ExecCtx(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
