package model

import (
	"context"
	"database/sql"
	"time"
)

type CreateMessageInput struct {
	RecipientUserID   string
	RecipientRoleCode string
	MessageType       string
	TriggerType       string
	TriggerID         string
	Title             string
	Content           string
	TargetURL         string
}

type CreateMessageResult struct {
	ID string
}

type MessageItem struct {
	ID          string
	MessageType string
	Title       string
	Content     string
	TargetURL   string
	Status      string
	CreatedAt   string
}

type ListMessagesFilter struct {
	UserID   string
	RoleCode string
	Type     string
	Status   string
	Page     int64
	PageSize int64
}

type ListMessagesResult struct {
	Items    []MessageItem
	Page     int64
	PageSize int64
	Total    int64
}

type ReadMessageResult struct {
	ID     string
	Status string
}

type LifecycleResource struct {
	ID         string
	MerchantID string
	Title      string
}

type MessageModel struct {
	db *sql.DB
}

func NewMessageModel(db *sql.DB) *MessageModel {
	return &MessageModel{db: db}
}

func (m *MessageModel) CreateMessage(ctx context.Context, input CreateMessageInput) (CreateMessageResult, error) {
	var result CreateMessageResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO messages (
  recipient_user_id,
  recipient_role_code,
  message_type,
  trigger_type,
  trigger_id,
  title,
  content,
  target_url,
  status
)
VALUES (
  NULLIF($1, '')::bigint,
  NULLIF($2, ''),
  $3,
  $4,
  NULLIF($5, '')::bigint,
  $6,
  $7,
  NULLIF($8, ''),
  'unread'
)
RETURNING id::text
`, input.RecipientUserID, input.RecipientRoleCode, input.MessageType, input.TriggerType, input.TriggerID, input.Title, input.Content, input.TargetURL).Scan(&result.ID)
	return result, err
}

func (m *MessageModel) ListMessages(ctx context.Context, filter ListMessagesFilter) (ListMessagesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT id::text, message_type, title, content, COALESCE(target_url, ''), status, created_at, COUNT(*) OVER() AS total
FROM messages
WHERE (
    ($1 <> '' AND recipient_user_id = $1::bigint)
    OR ($2 <> '' AND recipient_role_code = $2)
  )
  AND ($3 = '' OR message_type = $3)
  AND ($4 = '' OR status = $4)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6
`, filter.UserID, filter.RoleCode, filter.Type, filter.Status, pageSize, offset)
	if err != nil {
		return ListMessagesResult{}, err
	}
	defer rows.Close()
	result := ListMessagesResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item MessageItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.MessageType, &item.Title, &item.Content, &item.TargetURL, &item.Status, &createdAt, &result.Total); err != nil {
			return ListMessagesResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListMessagesResult{}, err
	}
	return result, nil
}

func (m *MessageModel) ReadMessage(ctx context.Context, userID string, roleCode string, messageID string) (ReadMessageResult, error) {
	var result ReadMessageResult
	err := m.db.QueryRowContext(ctx, `
UPDATE messages
SET status = 'read', read_at = now()
WHERE id = $1
  AND (
    ($2 <> '' AND recipient_user_id = NULLIF($2, '')::bigint)
    OR ($3 <> '' AND recipient_role_code = $3)
  )
RETURNING id::text, status
`, messageID, userID, roleCode).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *MessageModel) MarkExpiredResources(ctx context.Context) ([]LifecycleResource, error) {
	rows, err := m.db.QueryContext(ctx, `
UPDATE resources
SET status = 'expired', updated_at = now()
WHERE status = 'published'
  AND expires_at IS NOT NULL
  AND expires_at <= now()
  AND deleted_at IS NULL
RETURNING id::text, merchant_id::text, title
`)
	if err != nil {
		return nil, err
	}
	return scanLifecycleResources(rows)
}

func (m *MessageModel) ListResourcesExpiringSoon(ctx context.Context) ([]LifecycleResource, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT r.id::text, r.merchant_id::text, r.title
FROM resources r
WHERE r.status = 'published'
  AND r.expires_at IS NOT NULL
  AND r.expires_at > now()
  AND r.expires_at <= now() + interval '2 days'
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM messages msg
    WHERE msg.trigger_type = 'resource_expiring'
      AND msg.trigger_id = r.id
  )
`)
	if err != nil {
		return nil, err
	}
	return scanLifecycleResources(rows)
}

func scanLifecycleResources(rows *sql.Rows) ([]LifecycleResource, error) {
	defer rows.Close()
	var items []LifecycleResource
	for rows.Next() {
		var item LifecycleResource
		if err := rows.Scan(&item.ID, &item.MerchantID, &item.Title); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
