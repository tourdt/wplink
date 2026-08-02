package model

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"
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
	ID      string
	Created bool
}

type MessageItem struct {
	ID          string
	MessageType string
	TriggerID   string
	Title       string
	Content     string
	TargetURL   string
	Status      string
	CreatedAt   string
}

type ListMessagesFilter struct {
	UserID    string
	RoleCode  string
	RoleCodes []string
	Type      string
	Status    string
	Page      int64
	PageSize  int64
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
ON CONFLICT DO NOTHING
RETURNING id::text
`, input.RecipientUserID, input.RecipientRoleCode, input.MessageType, input.TriggerType, input.TriggerID, input.Title, input.Content, input.TargetURL).Scan(&result.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return CreateMessageResult{Created: false}, nil
	}
	if err == nil {
		result.Created = true
	}
	return result, err
}

func (m *MessageModel) ListMessages(ctx context.Context, filter ListMessagesFilter) (ListMessagesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	roleCodes := normalizeMessageRoleCodes(filter.RoleCode, filter.RoleCodes)
	rows, err := m.db.QueryContext(ctx, `
SELECT id::text, message_type, COALESCE(trigger_id::text, ''), title, content, COALESCE(target_url, ''), status, created_at, COUNT(*) OVER() AS total
FROM messages
WHERE (
    ($1 <> '' AND recipient_user_id = NULLIF($1, '')::bigint)
    OR recipient_role_code = ANY($2::text[])
  )
  AND ($3 = '' OR message_type = $3)
  AND ($4 = '' OR status = $4)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6
`, filter.UserID, pq.Array(roleCodes), filter.Type, filter.Status, pageSize, offset)
	if err != nil {
		return ListMessagesResult{}, err
	}
	defer rows.Close()
	result := ListMessagesResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item MessageItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.MessageType, &item.TriggerID, &item.Title, &item.Content, &item.TargetURL, &item.Status, &createdAt, &result.Total); err != nil {
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

func (m *MessageModel) ReadMessage(ctx context.Context, userID string, roleCodes []string, messageID string) (ReadMessageResult, error) {
	var result ReadMessageResult
	roleCodes = normalizeMessageRoleCodes("", roleCodes)
	err := m.db.QueryRowContext(ctx, `
UPDATE messages
SET status = 'read', read_at = now()
WHERE id = $1
  AND (
    ($2 <> '' AND recipient_user_id = NULLIF($2, '')::bigint)
    OR recipient_role_code = ANY($3::text[])
  )
RETURNING id::text, status
`, messageID, userID, pq.Array(roleCodes)).Scan(&result.ID, &result.Status)
	return result, err
}

func normalizeMessageRoleCodes(primary string, values []string) []string {
	seen := map[string]struct{}{}
	roleCodes := make([]string, 0, len(values)+1)
	for _, value := range append([]string{primary}, values...) {
		roleCode := strings.TrimSpace(value)
		if roleCode == "" {
			continue
		}
		if _, ok := seen[roleCode]; ok {
			continue
		}
		seen[roleCode] = struct{}{}
		roleCodes = append(roleCodes, roleCode)
	}
	return roleCodes
}

func (m *MessageModel) MarkExpiredResources(ctx context.Context) ([]LifecycleResource, error) {
	rows, err := m.db.QueryContext(ctx, `
WITH newly_expired AS (
  UPDATE resources
  SET status = 'expired', updated_at = now()
  WHERE status = 'published'
    AND expires_at IS NOT NULL
    AND expires_at <= now()
    AND deleted_at IS NULL
  RETURNING id, merchant_id, title
)
SELECT id::text, merchant_id::text, title
FROM newly_expired
UNION
SELECT r.id::text, r.merchant_id::text, r.title
FROM resources r
WHERE r.status = 'expired'
  AND r.expires_at IS NOT NULL
  AND r.expires_at <= now()
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM messages msg
    WHERE msg.recipient_role_code = 'merchant:' || r.merchant_id::text
      AND msg.trigger_type = 'resource_expired'
      AND msg.trigger_id = r.id
  )
`)
	if err != nil {
		return nil, err
	}
	return scanLifecycleResources(rows)
}

const listResourcesExpiringSoonSQL = `
SELECT r.id::text, r.merchant_id::text, r.title
FROM resources r
WHERE r.status = 'published'
  AND r.expires_at IS NOT NULL
  AND r.expires_at > now()
  AND r.expires_at <= now() + make_interval(days => CASE
    WHEN NULLIF(r.resource_type_snapshot #>> '{messageRules,expiringSoonDays}', '') ~ '^[0-9]+$'
      THEN GREATEST((r.resource_type_snapshot #>> '{messageRules,expiringSoonDays}')::int, 1)
    ELSE 2
  END)
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM messages msg
    WHERE msg.trigger_type = 'resource_expiring'
      AND msg.trigger_id = r.id
  )
`

func (m *MessageModel) ListResourcesExpiringSoon(ctx context.Context) ([]LifecycleResource, error) {
	rows, err := m.db.QueryContext(ctx, listResourcesExpiringSoonSQL)
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
