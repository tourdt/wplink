package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	MerchantLocationReportStatusPending  = "pending"
	MerchantLocationReportStatusValid    = "valid"
	MerchantLocationReportStatusInvalid  = "invalid"
	MerchantLocationReportStatusResolved = "resolved"

	merchantLocationReportAutoHideThreshold = int64(3)
)

var ErrMerchantLocationNotReportable = errors.New("merchant location is not reportable")

var (
	ErrMerchantLocationReportAlreadyReviewed = errors.New("merchant location report already reviewed")
	ErrMerchantLocationCoordinatesMissing    = errors.New("merchant location coordinates missing")
	ErrMerchantLocationAutoHideInvariant     = errors.New("merchant location auto-hide invariant violated")
)

const adminMerchantLocationReportStatusFilterSQL = `WHERE ($1 = '' OR mlr.status = $1)`

const adminMerchantLocationReportCountSQL = `
SELECT COUNT(*)
FROM merchant_location_reports mlr
` + adminMerchantLocationReportStatusFilterSQL

const adminMerchantLocationReportListSQL = `
SELECT
  mlr.id::text,
  mlr.status,
  mlr.merchant_id::text,
  m.name,
  mlr.reporter_user_id::text,
  mlr.reason_code,
  mlr.description,
  mlr.location_snapshot,
  COALESCE(mlr.reviewed_by::text, ''),
  COALESCE(mlr.review_note, ''),
  mlr.created_at,
  mlr.reviewed_at
FROM merchant_location_reports mlr
JOIN merchants m ON m.id = mlr.merchant_id
` + adminMerchantLocationReportStatusFilterSQL + `
ORDER BY mlr.created_at DESC, mlr.id DESC
LIMIT $2 OFFSET $3
`

type CreateMerchantLocationReportInput struct {
	MerchantID     string
	ReporterUserID string
	ReasonCode     string
	Description    string
}

type CreateMerchantLocationReportResult struct {
	ID                  string
	Status              string
	ActiveReporterCount int64
	AutoHidden          bool
	Idempotent          bool
}

type ReviewMerchantLocationReportInput struct {
	ReportID   string
	Status     string
	ReviewNote string
	AdminID    string
	AdminRole  string
}

type ReviewMerchantLocationReportResult struct {
	ReportID   string
	MerchantID string
	Status     string
}

type UpdateMerchantLocationStatusInput struct {
	MerchantID string
	Status     string
	Reason     string
	AdminID    string
	AdminRole  string
}

type UpdateMerchantLocationStatusResult struct {
	MerchantID string
	OldStatus  string
	NewStatus  string
}

type AdminMerchantLocationReportFilter struct {
	Status   string
	Page     int64
	PageSize int64
}

type AdminMerchantLocationReportItem struct {
	ID               string
	Status           string
	MerchantID       string
	MerchantName     string
	ReporterUserID   string
	ReasonCode       string
	Description      string
	LocationSnapshot JSONMap
	ReviewedBy       string
	ReviewNote       string
	CreatedAt        string
	ReviewedAt       string
}

type ListAdminMerchantLocationReportsResult struct {
	Items    []AdminMerchantLocationReportItem
	Page     int64
	PageSize int64
	Total    int64
}

type MerchantLocationReportModel struct {
	db *sql.DB
}

func NewMerchantLocationReportModel(db *sql.DB) *MerchantLocationReportModel {
	return &MerchantLocationReportModel{db: db}
}

// CreateMerchantLocationReport 串行化同一商家的报告写入，保证“第 3 名不同用户”计数与隐藏状态原子生效。
func (m *MerchantLocationReportModel) CreateMerchantLocationReport(ctx context.Context, input CreateMerchantLocationReportInput) (CreateMerchantLocationReportResult, error) {
	var result CreateMerchantLocationReportResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var marketName, buildingName, floorNo, boothNo, addressText, locationName string
		var latitude, longitude sql.NullFloat64
		var locationStatus string
		if err := tx.QueryRowContext(ctx, `
SELECT
  COALESCE(market_name, ''),
  COALESCE(building_name, ''),
  COALESCE(floor_no, ''),
  COALESCE(booth_no, ''),
  COALESCE(address_text, ''),
  latitude,
  longitude,
  COALESCE(location_name, ''),
  location_status
FROM merchants
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE
`, strings.TrimSpace(input.MerchantID)).Scan(
			&marketName, &buildingName, &floorNo, &boothNo, &addressText,
			&latitude, &longitude, &locationName, &locationStatus,
		); err != nil {
			return err
		}

		// 重复提交优先返回原 pending 报告；即使第 3 人已触发 risk_hidden，也不能把同一请求变成冲突。
		err := tx.QueryRowContext(ctx, `
SELECT
  mlr.id::text,
  mlr.status,
  (
    SELECT COUNT(DISTINCT active.reporter_user_id)
    FROM merchant_location_reports active
    WHERE active.merchant_id = mlr.merchant_id AND active.status = 'pending'
  ) AS active_reporter_count
FROM merchant_location_reports mlr
WHERE mlr.merchant_id = $1
  AND mlr.reporter_user_id = $2
  AND mlr.status = 'pending'
ORDER BY mlr.created_at DESC, mlr.id DESC
LIMIT 1
`, strings.TrimSpace(input.MerchantID), strings.TrimSpace(input.ReporterUserID)).Scan(
			&result.ID, &result.Status, &result.ActiveReporterCount,
		)
		if err == nil {
			result.Idempotent = true
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if locationStatus != MerchantLocationStatusVisible || !validStoredMerchantCoordinates(latitude, longitude) {
			return ErrMerchantLocationNotReportable
		}

		snapshot := JSONMap{
			"marketName": marketName, "buildingName": buildingName, "floorNo": floorNo,
			"boothNo": boothNo, "addressText": addressText, "latitude": latitude.Float64,
			"longitude": longitude.Float64, "locationName": locationName,
		}
		if err := tx.QueryRowContext(ctx, `
INSERT INTO merchant_location_reports (
  merchant_id,
  reporter_user_id,
  reason_code,
  description,
  location_snapshot
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id::text, status
`, strings.TrimSpace(input.MerchantID), strings.TrimSpace(input.ReporterUserID), strings.TrimSpace(input.ReasonCode), strings.TrimSpace(input.Description), snapshot).
			Scan(&result.ID, &result.Status); err != nil {
			return err
		}

		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(DISTINCT reporter_user_id)
FROM merchant_location_reports
WHERE merchant_id = $1 AND status = 'pending'
`, strings.TrimSpace(input.MerchantID)).Scan(&result.ActiveReporterCount); err != nil {
			return err
		}
		if result.ActiveReporterCount < merchantLocationReportAutoHideThreshold {
			return nil
		}

		updated, err := tx.ExecContext(ctx, `
UPDATE merchants
SET location_status = 'risk_hidden', updated_at = now()
WHERE id = $1 AND location_status = 'visible' AND deleted_at IS NULL
`, strings.TrimSpace(input.MerchantID))
		if err != nil {
			return err
		}
		affected, err := updated.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			// 商家行已在本事务开头锁定且状态为 visible；此处 0/多行都表示事务不变量被破坏，禁止提交孤立报告。
			return fmt.Errorf("%w: merchantId=%s affectedRows=%d", ErrMerchantLocationAutoHideInvariant, strings.TrimSpace(input.MerchantID), affected)
		}
		result.AutoHidden = true
		return nil
	})
	if err != nil {
		// 即使事务回滚，也保留事务内已取得的业务标识，供上层结构化错误日志定位；返回值不代表写入成功。
		return result, fmt.Errorf("创建商家位置报告事务失败: %w", err)
	}
	return result, nil
}

func (m *MerchantLocationReportModel) ResolvePendingLocationReports(ctx context.Context, merchantID string) (int64, error) {
	var resolved int64
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var err error
		resolved, err = resolvePendingLocationReportsTx(ctx, tx, strings.TrimSpace(merchantID))
		return err
	})
	return resolved, err
}

func resolvePendingLocationReportsTx(ctx context.Context, tx *sql.Tx, merchantID string) (int64, error) {
	result, err := tx.ExecContext(ctx, `
UPDATE merchant_location_reports
SET status = 'resolved', updated_at = now()
WHERE merchant_id = $1 AND status = 'pending'
`, merchantID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *MerchantLocationReportModel) ListAdminMerchantLocationReports(ctx context.Context, filter AdminMerchantLocationReportFilter) (ListAdminMerchantLocationReportsResult, error) {
	page, pageSize, offset, err := normalizeMerchantLocationReportPagination(filter.Page, filter.PageSize)
	if err != nil {
		return ListAdminMerchantLocationReportsResult{}, err
	}
	status := strings.TrimSpace(filter.Status)
	result := ListAdminMerchantLocationReportsResult{
		Items: make([]AdminMerchantLocationReportItem, 0), Page: page, PageSize: pageSize,
	}
	// total 必须独立于当前页是否有行；COUNT 和列表共用同一筛选片段，避免状态条件漂移。
	if err := m.db.QueryRowContext(ctx, adminMerchantLocationReportCountSQL, status).Scan(&result.Total); err != nil {
		return ListAdminMerchantLocationReportsResult{}, err
	}
	rows, err := m.db.QueryContext(ctx, adminMerchantLocationReportListSQL, status, pageSize, offset)
	if err != nil {
		return ListAdminMerchantLocationReportsResult{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var item AdminMerchantLocationReportItem
		var createdAt time.Time
		var reviewedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.Status, &item.MerchantID, &item.MerchantName, &item.ReporterUserID,
			&item.ReasonCode, &item.Description, &item.LocationSnapshot, &item.ReviewedBy,
			&item.ReviewNote, &createdAt, &reviewedAt,
		); err != nil {
			return ListAdminMerchantLocationReportsResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		if reviewedAt.Valid {
			item.ReviewedAt = reviewedAt.Time.Format(time.RFC3339)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListAdminMerchantLocationReportsResult{}, err
	}
	return result, nil
}

func normalizeMerchantLocationReportPagination(page int64, pageSize int64) (int64, int64, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	if page > 1 && page-1 > math.MaxInt64/pageSize {
		return 0, 0, 0, errors.New("位置反馈分页参数超出范围")
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

// ReviewMerchantLocationReport 把人工结论与操作日志放在同一事务，避免报告状态已变但审计记录缺失。
func (m *MerchantLocationReportModel) ReviewMerchantLocationReport(ctx context.Context, input ReviewMerchantLocationReportInput) (ReviewMerchantLocationReportResult, error) {
	var result ReviewMerchantLocationReportResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var oldStatus, reasonCode, description string
		if err := tx.QueryRowContext(ctx, `
SELECT merchant_id::text, status, reason_code, description
FROM merchant_location_reports
WHERE id = $1
FOR UPDATE
`, strings.TrimSpace(input.ReportID)).Scan(&result.MerchantID, &oldStatus, &reasonCode, &description); err != nil {
			return err
		}
		if oldStatus != MerchantLocationReportStatusPending {
			return ErrMerchantLocationReportAlreadyReviewed
		}

		updateResult, err := tx.ExecContext(ctx, `
UPDATE merchant_location_reports
SET status = $2,
    reviewed_by = $3,
    review_note = NULLIF($4, ''),
    reviewed_at = now(),
    updated_at = now()
WHERE id = $1 AND status = 'pending'
`, strings.TrimSpace(input.ReportID), strings.TrimSpace(input.Status), strings.TrimSpace(input.AdminID), strings.TrimSpace(input.ReviewNote))
		if err != nil {
			return err
		}
		affected, err := updateResult.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrMerchantLocationReportAlreadyReviewed
		}

		result.ReportID = strings.TrimSpace(input.ReportID)
		result.Status = strings.TrimSpace(input.Status)
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID: input.AdminID, OperatorRole: input.AdminRole,
			Action: "merchant_location_report_review", ObjectType: "merchant_location_report", ObjectID: input.ReportID,
			BeforeSnapshot: JSONMap{"merchantId": result.MerchantID, "status": oldStatus, "reasonCode": reasonCode, "description": description},
			AfterSnapshot:  JSONMap{"merchantId": result.MerchantID, "status": result.Status, "reviewNote": strings.TrimSpace(input.ReviewNote)},
		})
	})
	if err != nil {
		return result, fmt.Errorf("审核商家位置报告事务失败: %w", err)
	}
	return result, nil
}

// UpdateMerchantLocationStatus 只允许在商家行锁内恢复有效坐标，并把人工状态变更与审计日志原子提交。
func (m *MerchantLocationReportModel) UpdateMerchantLocationStatus(ctx context.Context, input UpdateMerchantLocationStatusInput) (UpdateMerchantLocationStatusResult, error) {
	result := UpdateMerchantLocationStatusResult{MerchantID: strings.TrimSpace(input.MerchantID), NewStatus: strings.TrimSpace(input.Status)}
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var latitude, longitude sql.NullFloat64
		if err := tx.QueryRowContext(ctx, `
SELECT location_status, latitude, longitude
FROM merchants
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE
`, result.MerchantID).Scan(&result.OldStatus, &latitude, &longitude); err != nil {
			return err
		}
		if result.NewStatus == MerchantLocationStatusVisible && !validStoredMerchantCoordinates(latitude, longitude) {
			return ErrMerchantLocationCoordinatesMissing
		}

		updated, err := tx.ExecContext(ctx, `
UPDATE merchants
SET location_status = $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
`, result.MerchantID, result.NewStatus)
		if err != nil {
			return err
		}
		affected, err := updated.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return sql.ErrNoRows
		}

		action := "merchant_location_hide"
		if result.NewStatus == MerchantLocationStatusVisible {
			action = "merchant_location_restore"
		}
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID: input.AdminID, OperatorRole: input.AdminRole,
			Action: action, ObjectType: "merchant", ObjectID: result.MerchantID,
			BeforeSnapshot: JSONMap{"locationStatus": result.OldStatus},
			AfterSnapshot:  JSONMap{"locationStatus": result.NewStatus, "reason": strings.TrimSpace(input.Reason)},
		})
	})
	if err != nil {
		return result, fmt.Errorf("更新商家位置状态事务失败: %w", err)
	}
	return result, nil
}

func validStoredMerchantCoordinates(latitude sql.NullFloat64, longitude sql.NullFloat64) bool {
	return latitude.Valid && longitude.Valid && ValidMerchantCoordinates(latitude.Float64, longitude.Float64)
}
