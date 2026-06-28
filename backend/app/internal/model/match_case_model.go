package model

import (
	"context"
	"database/sql"
	"time"
)

const (
	MatchCaseStatusOpen      = "open"
	MatchCaseStatusContacted = "contacted"
	MatchCaseStatusSucceeded = "succeeded"
	MatchCaseStatusFailed    = "failed"
	MatchCaseStatusClosed    = "closed"
)

const matchMerchantProgressMessageSQL = `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
SELECT
  'merchant:' || merchant_id::text,
  'match_progress',
  'match_status_update',
  $1::bigint,
  '撮合进度更新',
  $2,
  '/pages/messages/index',
  'unread'
FROM match_case_participants
WHERE match_case_id = $1::bigint
  AND merchant_id IS NOT NULL
`

const matchDemandOwnerProgressMessageSQL = `
INSERT INTO messages (recipient_user_id, message_type, trigger_type, trigger_id, title, content, target_url, status)
SELECT
  pd.user_id,
  'match_progress',
  'match_status_update',
  $1::bigint,
  '撮合进度更新',
  $2,
  '/pages/my-demands/index?userId=' || pd.user_id::text,
  'unread'
FROM match_cases mc
JOIN purchase_demands pd ON pd.id = mc.purchase_demand_id
WHERE mc.id = $1::bigint
  AND pd.user_id IS NOT NULL
`

const matchCreateDemandOwnerMessageSQL = `
INSERT INTO messages (recipient_user_id, message_type, trigger_type, trigger_id, title, content, target_url, status)
SELECT
  pd.user_id,
  'match_progress',
  'match_create',
  $1::bigint,
  '采购需求已进入撮合',
  '运营已受理您的采购需求，正在为您匹配合适资源。',
  '/pages/my-demands/index?userId=' || pd.user_id::text,
  'unread'
FROM match_cases mc
JOIN purchase_demands pd ON pd.id = mc.purchase_demand_id
WHERE mc.id = $1::bigint
  AND pd.user_id IS NOT NULL
`

const matchCreateMerchantMessageSQL = `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
SELECT
  'merchant:' || merchant_id::text,
  'match_progress',
  'match_create',
  $1::bigint,
  '新的撮合机会',
  '运营已将您加入一个采购需求撮合，请关注后续进展。',
  '/pages/messages/index',
  'unread'
FROM match_case_participants
WHERE match_case_id = $1::bigint
  AND merchant_id IS NOT NULL
`

const matchDemandStatusSyncSQL = `
UPDATE purchase_demands pd
SET status = $2, updated_at = now()
FROM match_cases mc
WHERE mc.purchase_demand_id = pd.id
  AND mc.id = $1::bigint
`

type CreateMatchCaseInput struct {
	PurchaseDemandID       string
	OperatorID             string
	ResourceIDs            []string
	ParticipantMerchantIDs []string
	ResultNote             string
}

type MatchCaseResult struct {
	ID     string
	Status string
}

type ListMatchCasesFilter struct {
	Status   string
	Page     int64
	PageSize int64
}

type MatchCaseListItem struct {
	ID               string
	PurchaseDemandID string
	DemandTitle      string
	Status           string
	Source           string
	ResultNote       string
	ResourceCount    int64
	ParticipantCount int64
	CreatedAt        string
}

type ListMatchCasesResult struct {
	Items    []MatchCaseListItem
	Page     int64
	PageSize int64
	Total    int64
}

type UpdateMatchCaseStatusInput struct {
	MatchCaseID string
	OperatorID  string
	Status      string
	ResultNote  string
}

type AddMatchCaseResourcesInput struct {
	MatchCaseID string
	OperatorID  string
	ResourceIDs []string
}

type AddMatchCaseParticipantsInput struct {
	MatchCaseID string
	OperatorID  string
	MerchantIDs []string
}

type MatchCaseModel struct {
	db *sql.DB
}

func NewMatchCaseModel(db *sql.DB) *MatchCaseModel {
	return &MatchCaseModel{db: db}
}

func (m *MatchCaseModel) CreateMatchCase(ctx context.Context, input CreateMatchCaseInput) (MatchCaseResult, error) {
	var result MatchCaseResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
INSERT INTO match_cases (purchase_demand_id, city_station_id, status, source, operator_id, result_note)
SELECT id, city_station_id, 'open', 'manual', NULLIF($2, '')::bigint, NULLIF($3, '')
FROM purchase_demands
WHERE id = $1::bigint
RETURNING id::text, status
`, input.PurchaseDemandID, input.OperatorID, input.ResultNote).Scan(&result.ID, &result.Status)
		if err != nil {
			return err
		}
		if err := insertMatchResources(ctx, tx, result.ID, input.ResourceIDs); err != nil {
			return err
		}
		if err := insertMatchParticipants(ctx, tx, result.ID, input.ParticipantMerchantIDs); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE purchase_demands
SET status = 'matching', updated_at = now()
WHERE id = $1::bigint
`, input.PurchaseDemandID); err != nil {
			return err
		}
		// 创建撮合单时即通知需求发布人和初始参与商家，避免只有后台状态变化、双方都没有消息触达。
		if _, err := tx.ExecContext(ctx, matchCreateDemandOwnerMessageSQL, result.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, matchCreateMerchantMessageSQL, result.ID); err != nil {
			return err
		}
		return recordMatchOperation(ctx, tx, input.OperatorID, "match_create", result.ID, JSONMap{
			"purchaseDemandId":       input.PurchaseDemandID,
			"resourceIds":            input.ResourceIDs,
			"participantMerchantIds": input.ParticipantMerchantIDs,
		})
	})
	return result, err
}

func (m *MatchCaseModel) ListMatchCases(ctx context.Context, filter ListMatchCasesFilter) (ListMatchCasesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  mc.id::text,
  COALESCE(mc.purchase_demand_id::text, ''),
  COALESCE(pd.title, ''),
  mc.status,
  mc.source,
  COALESCE(mc.result_note, ''),
  COALESCE(rc.resource_count, 0),
  COALESCE(pc.participant_count, 0),
  mc.created_at,
  COUNT(*) OVER() AS total
FROM match_cases mc
LEFT JOIN purchase_demands pd ON pd.id = mc.purchase_demand_id
LEFT JOIN (
  SELECT match_case_id, COUNT(*) AS resource_count
  FROM match_case_resources
  GROUP BY match_case_id
) rc ON rc.match_case_id = mc.id
LEFT JOIN (
  SELECT match_case_id, COUNT(*) AS participant_count
  FROM match_case_participants
  GROUP BY match_case_id
) pc ON pc.match_case_id = mc.id
WHERE ($1 = '' OR mc.status = $1)
ORDER BY mc.created_at DESC
LIMIT $2 OFFSET $3
`, filter.Status, pageSize, offset)
	if err != nil {
		return ListMatchCasesResult{}, err
	}
	defer rows.Close()
	result := ListMatchCasesResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item MatchCaseListItem
		var createdAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.PurchaseDemandID,
			&item.DemandTitle,
			&item.Status,
			&item.Source,
			&item.ResultNote,
			&item.ResourceCount,
			&item.ParticipantCount,
			&createdAt,
			&result.Total,
		); err != nil {
			return ListMatchCasesResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListMatchCasesResult{}, err
	}
	return result, nil
}

func (m *MatchCaseModel) UpdateMatchCaseStatus(ctx context.Context, input UpdateMatchCaseStatusInput) (MatchCaseResult, error) {
	var result MatchCaseResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
UPDATE match_cases
SET
  status = $2,
  result_note = NULLIF($3, ''),
  updated_at = now(),
  closed_at = CASE WHEN $2 IN ('succeeded', 'failed', 'closed') THEN now() ELSE closed_at END
WHERE id = $1::bigint
RETURNING id::text, status
`, input.MatchCaseID, input.Status, input.ResultNote).Scan(&result.ID, &result.Status)
		if err != nil {
			return err
		}
		if err := recordMatchOperation(ctx, tx, input.OperatorID, "match_status_update", input.MatchCaseID, JSONMap{
			"status":     input.Status,
			"resultNote": input.ResultNote,
		}); err != nil {
			return err
		}
		demandStatus := demandStatusForMatchStatus(input.Status)
		if demandStatus != "" {
			if _, err := tx.ExecContext(ctx, matchDemandStatusSyncSQL, input.MatchCaseID, demandStatus); err != nil {
				return err
			}
		}
		messageContent := matchStatusMessage(input.Status, input.ResultNote)
		// 撮合状态变化需要同时通知参与商家和采购需求发布用户，避免只运营侧可见、用户侧无进展感知。
		if _, err := tx.ExecContext(ctx, matchMerchantProgressMessageSQL, input.MatchCaseID, messageContent); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, matchDemandOwnerProgressMessageSQL, input.MatchCaseID, messageContent)
		return err
	})
	return result, err
}

func (m *MatchCaseModel) AddMatchCaseResources(ctx context.Context, input AddMatchCaseResourcesInput) error {
	return WithTx(ctx, m.db, func(tx *sql.Tx) error {
		if err := insertMatchResources(ctx, tx, input.MatchCaseID, input.ResourceIDs); err != nil {
			return err
		}
		return recordMatchOperation(ctx, tx, input.OperatorID, "match_resource_add", input.MatchCaseID, JSONMap{"resourceIds": input.ResourceIDs})
	})
}

func (m *MatchCaseModel) AddMatchCaseParticipants(ctx context.Context, input AddMatchCaseParticipantsInput) error {
	return WithTx(ctx, m.db, func(tx *sql.Tx) error {
		if err := insertMatchParticipants(ctx, tx, input.MatchCaseID, input.MerchantIDs); err != nil {
			return err
		}
		return recordMatchOperation(ctx, tx, input.OperatorID, "match_participant_add", input.MatchCaseID, JSONMap{"merchantIds": input.MerchantIDs})
	})
}

func insertMatchResources(ctx context.Context, tx *sql.Tx, matchCaseID string, resourceIDs []string) error {
	for _, resourceID := range resourceIDs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO match_case_resources (match_case_id, resource_id, role)
VALUES ($1::bigint, $2::bigint, 'candidate')
ON CONFLICT (match_case_id, resource_id) DO NOTHING
`, matchCaseID, resourceID); err != nil {
			return err
		}
	}
	return nil
}

func insertMatchParticipants(ctx context.Context, tx *sql.Tx, matchCaseID string, merchantIDs []string) error {
	for _, merchantID := range merchantIDs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO match_case_participants (match_case_id, merchant_id, participant_role)
SELECT $1::bigint, $2::bigint, 'merchant'
WHERE NOT EXISTS (
  SELECT 1 FROM match_case_participants
  WHERE match_case_id = $1::bigint AND merchant_id = $2::bigint
)
`, matchCaseID, merchantID); err != nil {
			return err
		}
	}
	return nil
}

func recordMatchOperation(ctx context.Context, tx *sql.Tx, operatorID string, action string, matchCaseID string, snapshot JSONMap) error {
	return recordOperationLogTx(ctx, tx, OperationLogInput{
		OperatorID:    operatorID,
		OperatorRole:  "platform_operator",
		Action:        action,
		ObjectType:    "match_case",
		ObjectID:      matchCaseID,
		AfterSnapshot: snapshot,
	})
}

func matchStatusMessage(status string, resultNote string) string {
	switch status {
	case MatchCaseStatusContacted:
		return "运营已联系候选资源，撮合正在推进中"
	case MatchCaseStatusSucceeded:
		if resultNote != "" {
			return "撮合已成功：" + resultNote
		}
		return "撮合已成功"
	case MatchCaseStatusFailed:
		if resultNote != "" {
			return "撮合未成功：" + resultNote
		}
		return "撮合未成功"
	case MatchCaseStatusClosed:
		return "撮合已关闭"
	default:
		return "撮合已创建，运营会继续跟进"
	}
}

func demandStatusForMatchStatus(status string) string {
	switch status {
	case MatchCaseStatusOpen:
		return "matching"
	case MatchCaseStatusContacted:
		return "contacted"
	case MatchCaseStatusSucceeded, MatchCaseStatusFailed, MatchCaseStatusClosed:
		return "closed"
	default:
		return ""
	}
}
