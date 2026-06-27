package model

import (
	"context"
	"database/sql"
	"time"
)

type OperationLogInput struct {
	OperatorID     string
	OperatorRole   string
	Action         string
	ObjectType     string
	ObjectID       string
	BeforeSnapshot JSONMap
	AfterSnapshot  JSONMap
}

type OperationLogFilter struct {
	ObjectType string
	ObjectID   string
	OperatorID string
	Page       int64
	PageSize   int64
}

type OperationLogItem struct {
	ID           string
	OperatorID   string
	OperatorRole string
	ObjectType   string
	ObjectID     string
	Action       string
	Reason       string
	CreatedAt    string
}

type ListOperationLogsResult struct {
	Items    []OperationLogItem
	Page     int64
	PageSize int64
	Total    int64
}

type OperationLogModel struct {
	db *sql.DB
}

func NewOperationLogModel(db *sql.DB) *OperationLogModel {
	return &OperationLogModel{db: db}
}

func (m *OperationLogModel) RecordOperationLog(ctx context.Context, input OperationLogInput) error {
	_, err := m.db.ExecContext(ctx, `
INSERT INTO operation_logs (
  operator_id,
  operator_role,
  action,
  object_type,
  object_id,
  before_snapshot,
  after_snapshot
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  NULLIF($5, '')::uuid,
  $6,
  $7
)
`, input.OperatorID, input.OperatorRole, input.Action, input.ObjectType, input.ObjectID, input.BeforeSnapshot, input.AfterSnapshot)
	return err
}

func recordOperationLogTx(ctx context.Context, tx *sql.Tx, input OperationLogInput) error {
	if input.OperatorID == "" {
		return nil
	}
	if input.OperatorRole == "" {
		input.OperatorRole = "platform_operator"
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO operation_logs (
  operator_id,
  operator_role,
  action,
  object_type,
  object_id,
  before_snapshot,
  after_snapshot
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  NULLIF($5, '')::uuid,
  $6,
  $7
)
`, input.OperatorID, input.OperatorRole, input.Action, input.ObjectType, input.ObjectID, input.BeforeSnapshot, input.AfterSnapshot)
	return err
}

func (m *OperationLogModel) ListOperationLogs(ctx context.Context, filter OperationLogFilter) (ListOperationLogsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  id,
  operator_id,
  operator_role,
  object_type,
  COALESCE(object_id::text, ''),
  action,
  created_at,
  COUNT(*) OVER() AS total
FROM operation_logs
WHERE ($1 = '' OR object_type = $1)
  AND ($2 = '' OR object_id = $2::uuid)
  AND ($3 = '' OR operator_id = $3::uuid)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5
`, filter.ObjectType, filter.ObjectID, filter.OperatorID, pageSize, offset)
	if err != nil {
		return ListOperationLogsResult{}, err
	}
	defer rows.Close()
	result := ListOperationLogsResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item OperationLogItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.OperatorID, &item.OperatorRole, &item.ObjectType, &item.ObjectID, &item.Action, &createdAt, &result.Total); err != nil {
			return ListOperationLogsResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListOperationLogsResult{}, err
	}
	return result, nil
}
