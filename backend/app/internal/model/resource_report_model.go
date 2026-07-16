package model

import (
	"context"
	"database/sql"
	"time"
)

const (
	ResourceReportStatusPending = "pending"
	ResourceReportStatusValid   = "valid"
	ResourceReportStatusInvalid = "invalid"

	ResourceReportActionValid   = "valid"
	ResourceReportActionInvalid = "invalid"

	ResourceReportResourceActionNone     = "none"
	ResourceReportResourceActionTakeDown = "take_down"

	EntitlementSourceResourceReportRefund = "resource_report_refund"
)

type CreateResourceReportInput struct {
	ResourceID     string
	ReporterUserID string
	ReasonCode     string
	ReasonText     string
	Evidence       JSONMap
}

type ResourceReportResult struct {
	ID     string
	Status string
}

type AdminResourceReportFilter struct {
	Status   string
	Page     int64
	PageSize int64
}

type AdminResourceReportItem struct {
	ID               string
	Status           string
	ResourceID       string
	ResourceTitle    string
	ResourceStatus   string
	MerchantID       string
	MerchantName     string
	ReportCount      int64
	ReasonCode       string
	ReasonText       string
	LatestReportedAt string
}

type ListAdminResourceReportsResult struct {
	Items    []AdminResourceReportItem
	Page     int64
	PageSize int64
	Total    int64
}

type ReviewResourceReportInput struct {
	ReportID            string
	Action              string
	ResourceAction      string
	Reason              string
	ReviewerID          string
	RefundPublishQuota  bool
}

type ReviewResourceReportResult struct {
	ID                  string
	Status              string
	ResourceID          string
	ResourceStatus      string
	ResolvedReportCount int64
	RefundPublishQuota  bool
}

func (m *ResourceModel) CreateResourceReport(ctx context.Context, input CreateResourceReportInput) (ResourceReportResult, error) {
	evidence := input.Evidence
	if evidence == nil {
		evidence = JSONMap{}
	}
	var result ResourceReportResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO resource_reports (
  resource_id,
  reporter_user_id,
  reason_code,
  reason_text,
  evidence
)
SELECT r.id, NULLIF($2, '')::bigint, $3, NULLIF($4, ''), $5
FROM resources r
WHERE r.id = $1
  AND r.status = 'published'
  AND r.deleted_at IS NULL
ON CONFLICT (resource_id, reporter_user_id)
WHERE status = 'pending' AND reporter_user_id IS NOT NULL
DO UPDATE SET
  reason_code = EXCLUDED.reason_code,
  reason_text = EXCLUDED.reason_text,
  evidence = EXCLUDED.evidence,
  updated_at = now()
RETURNING id::text, status
`, input.ResourceID, input.ReporterUserID, input.ReasonCode, input.ReasonText, evidence).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *ResourceModel) ListAdminResourceReports(ctx context.Context, filter AdminResourceReportFilter) (ListAdminResourceReportsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	status := filter.Status
	if status == "" {
		status = ResourceReportStatusPending
	}
	rows, err := m.db.QueryContext(ctx, `
WITH grouped_reports AS (
  SELECT resource_id, COUNT(*) AS report_count, MAX(created_at) AS latest_reported_at
  FROM resource_reports
  WHERE status = $1
  GROUP BY resource_id
),
latest_reports AS (
  SELECT DISTINCT ON (resource_id)
    id,
    status,
    resource_id,
    reason_code,
    COALESCE(reason_text, '') AS reason_text
  FROM resource_reports
  WHERE status = $1
  ORDER BY resource_id, created_at DESC, id DESC
)
SELECT
  lr.id::text,
  lr.status,
  lr.resource_id::text,
  r.title,
  r.status,
  r.merchant_id::text,
  m.name,
  gr.report_count,
  lr.reason_code,
  lr.reason_text,
  gr.latest_reported_at,
  COUNT(*) OVER() AS total
FROM grouped_reports gr
JOIN latest_reports lr ON lr.resource_id = gr.resource_id
JOIN resources r ON r.id = gr.resource_id
JOIN merchants m ON m.id = r.merchant_id
ORDER BY gr.report_count DESC, gr.latest_reported_at DESC
LIMIT $2 OFFSET $3
`, status, pageSize, offset)
	if err != nil {
		return ListAdminResourceReportsResult{}, err
	}
	defer rows.Close()

	result := ListAdminResourceReportsResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item AdminResourceReportItem
		var latestReportedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.Status,
			&item.ResourceID,
			&item.ResourceTitle,
			&item.ResourceStatus,
			&item.MerchantID,
			&item.MerchantName,
			&item.ReportCount,
			&item.ReasonCode,
			&item.ReasonText,
			&latestReportedAt,
			&result.Total,
		); err != nil {
			return ListAdminResourceReportsResult{}, err
		}
		item.LatestReportedAt = latestReportedAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListAdminResourceReportsResult{}, err
	}
	return result, nil
}

func (m *ResourceModel) ReviewResourceReport(ctx context.Context, input ReviewResourceReportInput) (ReviewResourceReportResult, error) {
	var result ReviewResourceReportResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var merchantID string
		var title string
		var resourceDeleted bool
		if err := tx.QueryRowContext(ctx, `
SELECT
  rr.id::text,
  rr.resource_id::text,
  r.merchant_id::text,
  r.title,
  r.status,
  r.deleted_at IS NOT NULL
FROM resource_reports rr
JOIN resources r ON r.id = rr.resource_id
WHERE rr.id = $1
  AND rr.status = 'pending'
FOR UPDATE OF rr, r
`, input.ReportID).Scan(&result.ID, &result.ResourceID, &merchantID, &title, &result.ResourceStatus, &resourceDeleted); err != nil {
			return err
		}

		if input.Action == ResourceReportActionValid && input.ResourceAction == ResourceReportResourceActionTakeDown && !resourceDeleted {
			if err := tx.QueryRowContext(ctx, `
UPDATE resources
SET status = 'taken_down',
    take_down_reason = $2,
    taken_down_at = now(),
    updated_at = now()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING status
`, result.ResourceID, input.Reason).Scan(&result.ResourceStatus); err != nil {
				return err
			}
		}

		nextStatus := ResourceReportStatusInvalid
		if input.Action == ResourceReportActionValid {
			nextStatus = ResourceReportStatusValid
		}
		// 同一资源可能被多名用户举报；人工复核一次后将该资源全部待处理举报结案，避免运营重复判断同一事实。
		if err := tx.QueryRowContext(ctx, `
UPDATE resource_reports
SET status = $2,
    reviewed_by = NULLIF($3, '')::bigint,
    reviewed_at = now(),
    review_action = $4,
    review_reason = NULLIF($5, ''),
    refund_publish_quota = $6,
    batch_review_id = NULLIF($7, '')::bigint,
    updated_at = now()
WHERE resource_id = $1
  AND status = 'pending'
RETURNING status
`, result.ResourceID, nextStatus, input.ReviewerID, input.Action, input.Reason, input.RefundPublishQuota, input.ReportID).Scan(&result.Status); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM resource_reports
WHERE resource_id = $1
  AND batch_review_id = NULLIF($2, '')::bigint
`, result.ResourceID, input.ReportID).Scan(&result.ResolvedReportCount); err != nil {
			return err
		}

		if input.Action == ResourceReportActionValid && input.RefundPublishQuota {
			if err := refundPublishQuotaForResourceReport(ctx, tx, merchantID, input, result); err != nil {
				return err
			}
			result.RefundPublishQuota = true
		}

		if input.Action == ResourceReportActionValid && input.ResourceAction == ResourceReportResourceActionTakeDown {
			content := title + " 因举报核实已由平台下架"
			if input.RefundPublishQuota {
				content += "，已退还 1 次发布次数"
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'resource_report', 'resource_report_take_down', $2, '举报处理结果', $3, $4, 'unread')
`, "merchant:"+merchantID, result.ResourceID, content, MerchantMyResourcesTargetURL(merchantID)); err != nil {
				return err
			}
		}

		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   input.ReviewerID,
			OperatorRole: "platform_operator",
			Action:       "resource_report_review",
			ObjectType:   "resource_report",
			ObjectID:     input.ReportID,
			AfterSnapshot: JSONMap{
				"resourceId":          result.ResourceID,
				"resourceStatus":      result.ResourceStatus,
				"reportStatus":        result.Status,
				"resolvedReportCount": result.ResolvedReportCount,
				"action":              input.Action,
				"resourceAction":      input.ResourceAction,
				"refundPublishQuota":  input.RefundPublishQuota,
				"reason":              input.Reason,
			},
		})
	})
	return result, err
}

func refundPublishQuotaForResourceReport(ctx context.Context, tx *sql.Tx, merchantID string, input ReviewResourceReportInput, result ReviewResourceReportResult) error {
	var entitlementID string
	if err := tx.QueryRowContext(ctx, `
INSERT INTO merchant_entitlements (
  merchant_id,
  entitlement_type,
  source_type,
  total_amount,
  remaining_amount,
  status,
  top_duration_hours
)
VALUES ($1, $2, $3, 1, 1, 'active', 0)
RETURNING id::text
`, merchantID, EntitlementTypePublishQuota, EntitlementSourceResourceReportRefund).Scan(&entitlementID); err != nil {
		return err
	}
	return recordOperationLogTx(ctx, tx, OperationLogInput{
		OperatorID:   input.ReviewerID,
		OperatorRole: "platform_operator",
		Action:       "resource_report_refund_publish_quota",
		ObjectType:   "merchant",
		ObjectID:     merchantID,
		AfterSnapshot: JSONMap{
			"entitlementId": entitlementID,
			"resourceId":    result.ResourceID,
			"reportId":      input.ReportID,
			"sourceType":    EntitlementSourceResourceReportRefund,
			"amount":        1,
			"reason":        input.Reason,
		},
	})
}
