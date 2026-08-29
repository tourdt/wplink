package model

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type MerchantDirectoryEventInput struct {
	UserID           string
	MerchantID       string
	TargetMerchantID string
	VisitorKey       string
	SessionID        string
	EventType        string
	Source           string
}

type MerchantDirectoryEventModel struct {
	conn  sqlx.SqlConn
	table string
}

func NewMerchantDirectoryEventModel(conn sqlx.SqlConn) *MerchantDirectoryEventModel {
	return &MerchantDirectoryEventModel{conn: conn, table: `"public"."merchant_directory_events"`}
}

func (m *MerchantDirectoryEventModel) RecordMerchantDirectoryEvent(ctx context.Context, input MerchantDirectoryEventInput) error {
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
	// visitorKey 和 sessionId 属于匿名分析标识，不能交给 sqlx 通用语句日志展开。
	rawDB, err := m.conn.RawDB()
	if err != nil {
		return fmt.Errorf("获取目录行为数据库连接失败: %w", err)
	}
	_, err = rawDB.ExecContext(ctx, query,
		input.UserID, input.MerchantID, input.TargetMerchantID, input.VisitorKey,
		input.SessionID, input.EventType, input.Source,
	)
	return err
}

func (m *MerchantDirectoryEventModel) DeleteMerchantDirectoryEventsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	// 目录行为只用于短期体验分析，按发生时间物理清理，避免匿名标识无限期保留。
	result, err := m.conn.ExecCtx(ctx, fmt.Sprintf(`DELETE FROM %s WHERE created_at < $1`, m.table), cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
