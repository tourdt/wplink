package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MerchantMapEventsModel = (*customMerchantMapEventsModel)(nil)

type (
	// MerchantMapEventsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMerchantMapEventsModel.
	MerchantMapEventsModel interface {
		merchantMapEventsModel
		RecordMerchantMapEvent(ctx context.Context, input MerchantMapEventInput) error
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
	_, err := m.conn.ExecCtx(ctx, query,
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
