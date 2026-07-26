package model

import (
	"context"
	"database/sql"
)

type ResourceMetricDelta struct {
	ResourceID          string
	ExposureCount       int64
	SearchExposureCount int64
	ListExposureCount   int64
	DetailViewCount     int64
	ContactClickCount   int64
	PhoneClickCount     int64
	WechatCopyCount     int64
	ShareCount          int64
	DealFeedbackCount   int64
}

type ResourceExposureItem struct {
	ResourceID        string
	VisibleDurationMS int64
}

type RecordResourceExposuresInput struct {
	UserID     string
	VisitorKey string
	SessionID  string
	Source     string
	Items      []ResourceExposureItem
}

type ResourceMetricsSummary struct {
	ExposureCount     int64
	DetailViewCount   int64
	PhoneClickCount   int64
	WechatCopyCount   int64
	DealFeedbackCount int64
}

type ResourceMetricDailyItem struct {
	Date            string
	ExposureCount   int64
	DetailViewCount int64
	PhoneClickCount int64
	WechatCopyCount int64
}

type ResourceMetricsResult struct {
	ResourceID string
	Summary    ResourceMetricsSummary
	Daily      []ResourceMetricDailyItem
}

type MerchantLast7DaysMetrics struct {
	ExposureCount     int64
	DetailViewCount   int64
	ContactClickCount int64
}

type MerchantMetricsSummary struct {
	MerchantID             string
	PublishedResourceCount int64
	ExpiringResourceCount  int64
	DealtResourceCount     int64
	Last7Days              MerchantLast7DaysMetrics
}

type ResourceMetricDailyModel struct {
	db *sql.DB
}

func NewResourceMetricDailyModel(db *sql.DB) *ResourceMetricDailyModel {
	return &ResourceMetricDailyModel{db: db}
}

func (m *ResourceMetricDailyModel) UpsertResourceMetric(ctx context.Context, delta ResourceMetricDelta) error {
	_, err := m.db.ExecContext(ctx, `
INSERT INTO resource_metrics_daily (
  resource_id,
  merchant_id,
  stat_date,
  exposure_count,
  search_exposure_count,
  list_exposure_count,
  detail_view_count,
  contact_click_count,
  phone_click_count,
  wechat_copy_count,
  share_count,
  deal_feedback_count
)
SELECT
  r.id,
  r.merchant_id,
  CURRENT_DATE,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10
FROM resources r
WHERE r.id = $1
ON CONFLICT (resource_id, stat_date)
DO UPDATE SET
  exposure_count = resource_metrics_daily.exposure_count + EXCLUDED.exposure_count,
  search_exposure_count = resource_metrics_daily.search_exposure_count + EXCLUDED.search_exposure_count,
  list_exposure_count = resource_metrics_daily.list_exposure_count + EXCLUDED.list_exposure_count,
  detail_view_count = resource_metrics_daily.detail_view_count + EXCLUDED.detail_view_count,
  contact_click_count = resource_metrics_daily.contact_click_count + EXCLUDED.contact_click_count,
  phone_click_count = resource_metrics_daily.phone_click_count + EXCLUDED.phone_click_count,
  wechat_copy_count = resource_metrics_daily.wechat_copy_count + EXCLUDED.wechat_copy_count,
  share_count = resource_metrics_daily.share_count + EXCLUDED.share_count,
  deal_feedback_count = resource_metrics_daily.deal_feedback_count + EXCLUDED.deal_feedback_count,
  updated_at = now()
`, delta.ResourceID, delta.ExposureCount, delta.SearchExposureCount, delta.ListExposureCount, delta.DetailViewCount, delta.ContactClickCount, delta.PhoneClickCount, delta.WechatCopyCount, delta.ShareCount, delta.DealFeedbackCount)
	return err
}

func (m *ResourceMetricDailyModel) RecordResourceExposures(ctx context.Context, input RecordResourceExposuresInput) (int64, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var recorded int64
	for _, item := range input.Items {
		result, execErr := tx.ExecContext(ctx, `
WITH candidate AS (
  SELECT r.id AS resource_id, r.merchant_id
  FROM resources r
  WHERE r.id = $1
    AND r.status = 'published'
    AND r.deleted_at IS NULL
    AND (r.expires_at IS NULL OR r.expires_at > now())
    AND (r.dealt_at IS NULL OR r.dealt_at > now() - interval '7 days')
    AND (
      NULLIF($2, '') IS NULL
      OR NOT EXISTS (
        SELECT 1
        FROM merchant_admin_bindings mab
        WHERE mab.merchant_id = r.merchant_id
          AND mab.user_id = NULLIF($2, '')::bigint
          AND mab.status = 'active'
      )
    )
),
inserted AS (
  INSERT INTO resource_exposure_events (
    resource_id, merchant_id, user_id, visitor_key, session_id, source, visible_duration_ms
  )
  SELECT
    candidate.resource_id,
    candidate.merchant_id,
    NULLIF($2, '')::bigint,
    $3,
    $4,
    $5,
    $6
  FROM candidate
  ON CONFLICT (resource_id, visitor_key, source, exposure_date) DO NOTHING
  RETURNING resource_id, merchant_id
)
INSERT INTO resource_metrics_daily (
  resource_id, merchant_id, stat_date, exposure_count, search_exposure_count, list_exposure_count
)
SELECT
  inserted.resource_id,
  inserted.merchant_id,
  CURRENT_DATE,
  1,
  CASE WHEN $5 = 'search' THEN 1 ELSE 0 END,
  CASE WHEN $5 IN ('home', 'list', 'topic', 'merchant') THEN 1 ELSE 0 END
FROM inserted
ON CONFLICT (resource_id, stat_date)
DO UPDATE SET
  exposure_count = resource_metrics_daily.exposure_count + 1,
  search_exposure_count = resource_metrics_daily.search_exposure_count + EXCLUDED.search_exposure_count,
  list_exposure_count = resource_metrics_daily.list_exposure_count + EXCLUDED.list_exposure_count,
  updated_at = now()
`, item.ResourceID, input.UserID, input.VisitorKey, input.SessionID, input.Source, item.VisibleDurationMS)
		if execErr != nil {
			return 0, execErr
		}
		affected, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return 0, affectedErr
		}
		recorded += affected
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return recorded, nil
}

func (m *ResourceMetricDailyModel) GetResourceMetrics(ctx context.Context, resourceID string, from string, to string) (ResourceMetricsResult, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  stat_date::text,
  exposure_count,
  detail_view_count,
  phone_click_count,
  wechat_copy_count
FROM resource_metrics_daily
WHERE resource_id = $1
  AND ($2 = '' OR stat_date >= $2::date)
  AND ($3 = '' OR stat_date <= $3::date)
ORDER BY stat_date ASC
`, resourceID, from, to)
	if err != nil {
		return ResourceMetricsResult{}, err
	}
	defer rows.Close()

	result := ResourceMetricsResult{ResourceID: resourceID}
	for rows.Next() {
		var item ResourceMetricDailyItem
		if err := rows.Scan(&item.Date, &item.ExposureCount, &item.DetailViewCount, &item.PhoneClickCount, &item.WechatCopyCount); err != nil {
			return ResourceMetricsResult{}, err
		}
		result.Summary.ExposureCount += item.ExposureCount
		result.Summary.DetailViewCount += item.DetailViewCount
		result.Summary.PhoneClickCount += item.PhoneClickCount
		result.Summary.WechatCopyCount += item.WechatCopyCount
		result.Daily = append(result.Daily, item)
	}
	if err := rows.Err(); err != nil {
		return ResourceMetricsResult{}, err
	}

	err = m.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(deal_feedback_count), 0)
FROM resource_metrics_daily
WHERE resource_id = $1
  AND ($2 = '' OR stat_date >= $2::date)
  AND ($3 = '' OR stat_date <= $3::date)
`, resourceID, from, to).Scan(&result.Summary.DealFeedbackCount)
	return result, err
}

func (m *ResourceMetricDailyModel) GetMerchantMetricsSummary(ctx context.Context, merchantID string) (MerchantMetricsSummary, error) {
	result := MerchantMetricsSummary{MerchantID: merchantID}
	err := m.db.QueryRowContext(ctx, `
SELECT
  COUNT(*) FILTER (WHERE status = 'published'),
  COUNT(*) FILTER (WHERE status = 'published' AND expires_at IS NOT NULL AND expires_at <= now() + interval '3 days'),
  COUNT(*) FILTER (WHERE dealt_at IS NOT NULL)
FROM resources
WHERE merchant_id = $1
  AND deleted_at IS NULL
`, merchantID).Scan(&result.PublishedResourceCount, &result.ExpiringResourceCount, &result.DealtResourceCount)
	if err != nil {
		return MerchantMetricsSummary{}, err
	}

	err = m.db.QueryRowContext(ctx, `
SELECT
  COALESCE(SUM(exposure_count), 0),
  COALESCE(SUM(detail_view_count), 0),
  COALESCE(SUM(contact_click_count), 0)
FROM resource_metrics_daily
WHERE merchant_id = $1
  AND stat_date >= CURRENT_DATE - interval '6 days'
`, merchantID).Scan(&result.Last7Days.ExposureCount, &result.Last7Days.DetailViewCount, &result.Last7Days.ContactClickCount)
	return result, err
}
