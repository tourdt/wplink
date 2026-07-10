package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	EntitlementSourceGrowthCampaign = "growth_campaign"

	GrowthEventUserFirstLogin                      = "user_first_login"
	GrowthEventResourceFirstApproved               = "resource_first_approved"
	GrowthEventResourceApprovedCountReached        = "resource_approved_count_reached"
	GrowthEventResourceShareEffectiveView          = "resource_share_effective_view"
	GrowthEventResourceShareEffectiveContact       = "resource_share_effective_contact"
	GrowthEventInviteeFirstResourceApproved        = "invitee_first_resource_approved"
	growthGrantStatusGranted                       = "granted"
	growthGrantStatusSkipped                       = "skipped"
	growthGrantReasonDuplicate                     = "duplicate"
	growthGrantReasonLimitReached                  = "limit_reached"
	growthGrantReasonConditionNotMatched           = "condition_not_matched"
	growthDefaultApprovedCount               int64 = 3
	growthDefaultWindowDays                  int64 = 30
	growthDefaultViewThreshold               int64 = 5
	growthDefaultWindowHours                 int64 = 24
)

type GrowthEventInput struct {
	EventType  string
	MerchantID string
	UserID     string
	ResourceID string
	EventID    string
	OccurredAt time.Time
}

type GrowthRewardGrantResult struct {
	ID            string
	EntitlementID string
	Status        string
	Reason        string
}

type GrowthCampaignModel struct {
	db *sql.DB
}

type AdminGrowthCampaignConfig struct {
	Code           string
	Name           string
	Status         string
	StartsAt       string
	EndsAt         string
	ConfigSnapshot JSONMap
	UpdatedAt      string
}

type SaveAdminGrowthCampaignInput struct {
	Code           string
	Name           string
	Status         string
	StartsAt       string
	EndsAt         string
	ConfigSnapshot JSONMap
	OperatorID     string
	DisableReason  string
}

type AdminGrowthRuleConfig struct {
	CampaignCode          string
	RuleCode              string
	TriggerEvent          string
	Status                string
	Priority              int64
	Conditions            JSONMap
	RewardType            string
	RewardAmount          int64
	ValidDays             int64
	PerUserLimit          int64
	PerUserDailyLimit     int64
	PerResourceDailyLimit int64
	Description           string
	UpdatedAt             string
}

type SaveAdminGrowthRuleInput struct {
	CampaignCode          string
	RuleCode              string
	TriggerEvent          string
	Status                string
	Priority              int64
	Conditions            JSONMap
	RewardType            string
	RewardAmount          int64
	ValidDays             int64
	PerUserLimit          int64
	PerUserDailyLimit     int64
	PerResourceDailyLimit int64
	Description           string
	OperatorID            string
}

type AdminGrowthRewardGrantFilter struct {
	CampaignCode string
	RuleCode     string
	MerchantID   string
	Status       string
	PageSize     int64
}

type AdminGrowthRewardGrant struct {
	ID           string
	CampaignCode string
	RuleCode     string
	MerchantID   string
	ResourceID   string
	RewardType   string
	RewardAmount int64
	Status       string
	Reason       string
	CreatedAt    string
}

type AdminGrowthConfigSaveResult struct {
	Code      string
	UpdatedAt string
}

func NewGrowthCampaignModel(db *sql.DB) *GrowthCampaignModel {
	return &GrowthCampaignModel{db: db}
}

func (m *GrowthCampaignModel) ListAdminGrowthCampaigns(ctx context.Context, status string) ([]AdminGrowthCampaignConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT code, name, status, starts_at, ends_at, config_snapshot, updated_at
FROM growth_campaigns
WHERE ($1 = '' OR status = $1)
ORDER BY updated_at DESC, id DESC
`, strings.TrimSpace(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminGrowthCampaignConfig, 0)
	for rows.Next() {
		var item AdminGrowthCampaignConfig
		var startsAt time.Time
		var endsAt sql.NullTime
		var updatedAt time.Time
		if err := rows.Scan(&item.Code, &item.Name, &item.Status, &startsAt, &endsAt, &item.ConfigSnapshot, &updatedAt); err != nil {
			return nil, err
		}
		item.StartsAt = startsAt.Format(time.RFC3339)
		if endsAt.Valid {
			item.EndsAt = endsAt.Time.Format(time.RFC3339)
		}
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *GrowthCampaignModel) SaveAdminGrowthCampaign(ctx context.Context, input SaveAdminGrowthCampaignInput) (AdminGrowthConfigSaveResult, error) {
	var result AdminGrowthConfigSaveResult
	if input.ConfigSnapshot == nil {
		input.ConfigSnapshot = JSONMap{}
	}
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var updatedAt time.Time
		if err := tx.QueryRowContext(ctx, `
INSERT INTO growth_campaigns (
  code, name, status, starts_at, ends_at, config_snapshot, created_by, updated_by
) VALUES (
  $1, $2, $3, COALESCE(NULLIF($4, '')::timestamptz, now()), NULLIF($5, '')::timestamptz, $6, NULLIF($7, '')::bigint, NULLIF($7, '')::bigint
)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  status = EXCLUDED.status,
  starts_at = COALESCE(NULLIF($4, '')::timestamptz, growth_campaigns.starts_at),
  ends_at = NULLIF($5, '')::timestamptz,
  config_snapshot = EXCLUDED.config_snapshot,
  updated_by = NULLIF($7, '')::bigint,
  updated_at = now()
RETURNING code, updated_at
`, input.Code, input.Name, input.Status, input.StartsAt, input.EndsAt, input.ConfigSnapshot, input.OperatorID).Scan(&result.Code, &updatedAt); err != nil {
			return err
		}
		result.UpdatedAt = updatedAt.Format(time.RFC3339)
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:     input.OperatorID,
			OperatorRole:   "platform_operator",
			Action:         "growth_campaign_save",
			ObjectType:     "growth_campaign",
			BeforeSnapshot: JSONMap{"code": input.Code, "disableReason": input.DisableReason},
			AfterSnapshot:  JSONMap{"code": input.Code, "name": input.Name, "status": input.Status},
		})
	})
	return result, err
}

func (m *GrowthCampaignModel) ListAdminGrowthRules(ctx context.Context, campaignCode string) ([]AdminGrowthRuleConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  campaign_code, rule_code, trigger_event, status, priority, conditions, reward_type, reward_amount,
  valid_days, per_user_limit, per_user_daily_limit, per_resource_daily_limit, description, updated_at
FROM growth_campaign_rules
WHERE campaign_code = $1
ORDER BY priority ASC, id ASC
`, strings.TrimSpace(campaignCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminGrowthRuleConfig, 0)
	for rows.Next() {
		var item AdminGrowthRuleConfig
		var perUserLimit sql.NullInt64
		var perUserDailyLimit sql.NullInt64
		var perResourceDailyLimit sql.NullInt64
		var updatedAt time.Time
		if err := rows.Scan(
			&item.CampaignCode,
			&item.RuleCode,
			&item.TriggerEvent,
			&item.Status,
			&item.Priority,
			&item.Conditions,
			&item.RewardType,
			&item.RewardAmount,
			&item.ValidDays,
			&perUserLimit,
			&perUserDailyLimit,
			&perResourceDailyLimit,
			&item.Description,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		item.PerUserLimit = nullableInt64Value(perUserLimit)
		item.PerUserDailyLimit = nullableInt64Value(perUserDailyLimit)
		item.PerResourceDailyLimit = nullableInt64Value(perResourceDailyLimit)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *GrowthCampaignModel) SaveAdminGrowthRule(ctx context.Context, input SaveAdminGrowthRuleInput) (AdminGrowthConfigSaveResult, error) {
	var result AdminGrowthConfigSaveResult
	if input.Conditions == nil {
		input.Conditions = JSONMap{}
	}
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var updatedAt time.Time
		if err := tx.QueryRowContext(ctx, `
INSERT INTO growth_campaign_rules (
  campaign_code, rule_code, trigger_event, status, priority, conditions, reward_type, reward_amount,
  valid_days, per_user_limit, per_user_daily_limit, per_resource_daily_limit, description
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9,
  NULLIF($10, 0), NULLIF($11, 0), NULLIF($12, 0), $13
)
ON CONFLICT (campaign_code, rule_code) DO UPDATE SET
  trigger_event = EXCLUDED.trigger_event,
  status = EXCLUDED.status,
  priority = EXCLUDED.priority,
  conditions = EXCLUDED.conditions,
  reward_type = EXCLUDED.reward_type,
  reward_amount = EXCLUDED.reward_amount,
  valid_days = EXCLUDED.valid_days,
  per_user_limit = EXCLUDED.per_user_limit,
  per_user_daily_limit = EXCLUDED.per_user_daily_limit,
  per_resource_daily_limit = EXCLUDED.per_resource_daily_limit,
  description = EXCLUDED.description,
  updated_at = now()
RETURNING rule_code, updated_at
`, input.CampaignCode, input.RuleCode, input.TriggerEvent, input.Status, input.Priority, input.Conditions, input.RewardType, input.RewardAmount, input.ValidDays, input.PerUserLimit, input.PerUserDailyLimit, input.PerResourceDailyLimit, input.Description).Scan(&result.Code, &updatedAt); err != nil {
			return err
		}
		result.UpdatedAt = updatedAt.Format(time.RFC3339)
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:     input.OperatorID,
			OperatorRole:   "platform_operator",
			Action:         "growth_campaign_rule_save",
			ObjectType:     "growth_campaign",
			BeforeSnapshot: JSONMap{"campaignCode": input.CampaignCode, "ruleCode": input.RuleCode},
			AfterSnapshot:  JSONMap{"campaignCode": input.CampaignCode, "ruleCode": input.RuleCode, "status": input.Status, "rewardType": input.RewardType, "rewardAmount": input.RewardAmount, "validDays": input.ValidDays},
		})
	})
	return result, err
}

func (m *GrowthCampaignModel) ListAdminGrowthRewardGrants(ctx context.Context, filter AdminGrowthRewardGrantFilter) ([]AdminGrowthRewardGrant, error) {
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}
	rows, err := m.db.QueryContext(ctx, `
SELECT
  id::text, campaign_code, rule_code, merchant_id::text, COALESCE(resource_id::text, ''),
  reward_type, reward_amount, status, reason, created_at
FROM growth_reward_grants
WHERE campaign_code = $1
  AND ($2 = '' OR rule_code = $2)
  AND ($3 = '' OR merchant_id = NULLIF($3, '')::bigint)
  AND ($4 = '' OR status = $4)
ORDER BY created_at DESC, id DESC
LIMIT $5
`, strings.TrimSpace(filter.CampaignCode), strings.TrimSpace(filter.RuleCode), strings.TrimSpace(filter.MerchantID), strings.TrimSpace(filter.Status), pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminGrowthRewardGrant, 0)
	for rows.Next() {
		var item AdminGrowthRewardGrant
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.CampaignCode, &item.RuleCode, &item.MerchantID, &item.ResourceID, &item.RewardType, &item.RewardAmount, &item.Status, &item.Reason, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *GrowthCampaignModel) TriggerGrowthEvent(ctx context.Context, input GrowthEventInput) ([]GrowthRewardGrantResult, error) {
	input = normalizeGrowthEventInput(input)
	if input.EventType == "" {
		return nil, nil
	}
	if input.MerchantID == "" && input.ResourceID != "" {
		merchantID, err := m.resourceMerchantID(ctx, input.ResourceID)
		if err != nil {
			return nil, err
		}
		input.MerchantID = merchantID
	}
	if input.MerchantID == "" {
		return nil, nil
	}

	rules, err := m.listActiveGrowthRules(ctx, input.EventType)
	if err != nil {
		return nil, err
	}
	results := make([]GrowthRewardGrantResult, 0, len(rules))
	for _, rule := range rules {
		matched, reason, err := m.growthRuleMatches(ctx, input, rule)
		if err != nil {
			return results, err
		}
		if !matched {
			results = append(results, GrowthRewardGrantResult{Status: growthGrantStatusSkipped, Reason: reason})
			continue
		}
		result, err := m.grantGrowthRule(ctx, input, rule)
		if err != nil {
			logx.Errorf("成长权益发放失败: campaignCode=%s ruleCode=%s eventType=%s merchantId=%s resourceId=%s eventId=%s err=%+v", rule.CampaignCode, rule.RuleCode, input.EventType, input.MerchantID, input.ResourceID, input.EventID, err)
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

type activeGrowthRule struct {
	CampaignCode          string
	RuleCode              string
	TriggerEvent          string
	Conditions            JSONMap
	RewardType            string
	RewardAmount          int64
	ValidDays             int64
	PerUserLimit          sql.NullInt64
	PerUserDailyLimit     sql.NullInt64
	PerResourceDailyLimit sql.NullInt64
}

func (m *GrowthCampaignModel) listActiveGrowthRules(ctx context.Context, eventType string) ([]activeGrowthRule, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  gcr.campaign_code,
  gcr.rule_code,
  gcr.trigger_event,
  gcr.conditions,
  gcr.reward_type,
  gcr.reward_amount,
  gcr.valid_days,
  gcr.per_user_limit,
  gcr.per_user_daily_limit,
  gcr.per_resource_daily_limit
FROM growth_campaign_rules gcr
JOIN growth_campaigns gc ON gc.code = gcr.campaign_code
WHERE gc.status = 'active'
  AND gcr.status = 'active'
  AND gcr.trigger_event = $1
  AND gc.starts_at <= now()
  AND (gc.ends_at IS NULL OR gc.ends_at > now())
ORDER BY gcr.priority ASC, gcr.id ASC
`, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []activeGrowthRule
	for rows.Next() {
		var rule activeGrowthRule
		if err := rows.Scan(
			&rule.CampaignCode,
			&rule.RuleCode,
			&rule.TriggerEvent,
			&rule.Conditions,
			&rule.RewardType,
			&rule.RewardAmount,
			&rule.ValidDays,
			&rule.PerUserLimit,
			&rule.PerUserDailyLimit,
			&rule.PerResourceDailyLimit,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (m *GrowthCampaignModel) growthRuleMatches(ctx context.Context, input GrowthEventInput, rule activeGrowthRule) (bool, string, error) {
	if rule.PerUserLimit.Valid {
		reached, err := m.growthGrantLimitReached(ctx, `
SELECT COUNT(*)
FROM growth_reward_grants
WHERE campaign_code = $1
  AND rule_code = $2
  AND merchant_id = $3
  AND status = 'granted'
`, rule.PerUserLimit.Int64, rule.CampaignCode, rule.RuleCode, input.MerchantID)
		if err != nil || reached {
			return !reached, growthGrantReasonLimitReached, err
		}
	}
	if rule.PerUserDailyLimit.Valid {
		reached, err := m.growthGrantLimitReached(ctx, `
SELECT COUNT(*)
FROM growth_reward_grants
WHERE campaign_code = $1
  AND rule_code = $2
  AND merchant_id = $3
  AND status = 'granted'
  AND created_at >= date_trunc('day', now())
`, rule.PerUserDailyLimit.Int64, rule.CampaignCode, rule.RuleCode, input.MerchantID)
		if err != nil || reached {
			return !reached, growthGrantReasonLimitReached, err
		}
	}
	if rule.PerResourceDailyLimit.Valid && input.ResourceID != "" {
		reached, err := m.growthGrantLimitReached(ctx, `
SELECT COUNT(*)
FROM growth_reward_grants
WHERE campaign_code = $1
  AND rule_code = $2
  AND resource_id = NULLIF($3, '')::bigint
  AND status = 'granted'
  AND created_at >= date_trunc('day', now())
`, rule.PerResourceDailyLimit.Int64, rule.CampaignCode, rule.RuleCode, input.ResourceID)
		if err != nil || reached {
			return !reached, growthGrantReasonLimitReached, err
		}
	}

	switch input.EventType {
	case GrowthEventResourceApprovedCountReached:
		matched, err := m.resourceApprovedCountReached(ctx, input, rule)
		if err != nil || !matched {
			return matched, growthGrantReasonConditionNotMatched, err
		}
	case GrowthEventResourceShareEffectiveView:
		matched, err := m.resourceShareViewThresholdReached(ctx, input, rule)
		if err != nil || !matched {
			return matched, growthGrantReasonConditionNotMatched, err
		}
	}
	return true, "", nil
}

func (m *GrowthCampaignModel) growthGrantLimitReached(ctx context.Context, query string, limit int64, args ...interface{}) (bool, error) {
	var count int64
	if err := m.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count >= limit, nil
}

func (m *GrowthCampaignModel) resourceApprovedCountReached(ctx context.Context, input GrowthEventInput, rule activeGrowthRule) (bool, error) {
	approvedCount := growthConditionInt(rule.Conditions, "approvedCount", growthDefaultApprovedCount)
	windowDays := growthConditionInt(rule.Conditions, "windowDays", growthDefaultWindowDays)
	var count int64
	err := m.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM resources
WHERE merchant_id = $1
  AND status = 'published'
  AND deleted_at IS NULL
  AND published_at >= now() - make_interval(days => GREATEST($2, 1)::int)
`, input.MerchantID, windowDays).Scan(&count)
	return count >= approvedCount, err
}

func (m *GrowthCampaignModel) resourceShareViewThresholdReached(ctx context.Context, input GrowthEventInput, rule activeGrowthRule) (bool, error) {
	if input.ResourceID == "" {
		return false, nil
	}
	viewThreshold := growthConditionInt(rule.Conditions, "viewThreshold", growthDefaultViewThreshold)
	windowHours := growthConditionInt(rule.Conditions, "windowHours", growthDefaultWindowHours)
	var count int64
	err := m.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM resource_contact_events
WHERE resource_id = $1
  AND action = 'share_view'
  AND created_at >= now() - make_interval(hours => GREATEST($2, 1)::int)
`, input.ResourceID, windowHours).Scan(&count)
	return count >= viewThreshold, err
}

func (m *GrowthCampaignModel) grantGrowthRule(ctx context.Context, input GrowthEventInput, rule activeGrowthRule) (GrowthRewardGrantResult, error) {
	var result GrowthRewardGrantResult
	idempotencyKey := growthRewardIdempotencyKey(input, rule)
	expiresAt := input.OccurredAt.AddDate(0, 0, int(rule.ValidDays))
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
INSERT INTO growth_reward_grants (
  campaign_code, rule_code, merchant_id, user_id, resource_id, event_id, idempotency_key,
  reward_type, reward_amount, valid_days, status, reason, snapshot
) VALUES (
  $1, $2, $3, NULLIF($4, '')::bigint, NULLIF($5, '')::bigint, NULLIF($6, '')::bigint, $7,
  $8, $9, $10, 'granted', 'matched', '{}'::jsonb
)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING id::text, status, reason
`, rule.CampaignCode, rule.RuleCode, input.MerchantID, input.UserID, input.ResourceID, input.EventID, idempotencyKey, rule.RewardType, rule.RewardAmount, rule.ValidDays).Scan(&result.ID, &result.Status, &result.Reason)
		if err == sql.ErrNoRows {
			result.Status = growthGrantStatusSkipped
			result.Reason = growthGrantReasonDuplicate
			return nil
		}
		if err != nil {
			return err
		}

		var entitlementID string
		if err := tx.QueryRowContext(ctx, `
INSERT INTO merchant_entitlements (
  merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status
) VALUES (
  $1, $2, $3, $4, $4, now(), $5, 'active'
) RETURNING id::text
`, input.MerchantID, rule.RewardType, EntitlementSourceGrowthCampaign, rule.RewardAmount, expiresAt).Scan(&entitlementID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
UPDATE growth_reward_grants
SET entitlement_id = $2
WHERE id = $1
`, result.ID, entitlementID)
		if err == nil {
			result.EntitlementID = entitlementID
		}
		return err
	})
	return result, err
}

func (m *GrowthCampaignModel) resourceMerchantID(ctx context.Context, resourceID string) (string, error) {
	var merchantID string
	err := m.db.QueryRowContext(ctx, `
SELECT merchant_id::text
FROM resources
WHERE id = $1
  AND deleted_at IS NULL
`, resourceID).Scan(&merchantID)
	return merchantID, err
}

func normalizeGrowthEventInput(input GrowthEventInput) GrowthEventInput {
	input.EventType = strings.TrimSpace(input.EventType)
	input.MerchantID = strings.TrimSpace(input.MerchantID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.EventID = strings.TrimSpace(input.EventID)
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	return input
}

func growthRewardIdempotencyKey(input GrowthEventInput, rule activeGrowthRule) string {
	if rule.PerUserLimit.Valid && rule.PerUserLimit.Int64 == 1 {
		return fmt.Sprintf("%s:%s:%s", rule.CampaignCode, rule.RuleCode, input.MerchantID)
	}
	if rule.PerResourceDailyLimit.Valid && input.ResourceID != "" {
		return fmt.Sprintf("%s:%s:%s:%s", rule.CampaignCode, rule.RuleCode, input.ResourceID, input.OccurredAt.Format("2006-01-02"))
	}
	objectKey := input.EventID
	if objectKey == "" {
		objectKey = input.ResourceID
	}
	if objectKey == "" {
		objectKey = input.MerchantID
	}
	return fmt.Sprintf("%s:%s:%s:%s", rule.CampaignCode, rule.RuleCode, input.MerchantID, objectKey)
}

func growthConditionInt(conditions JSONMap, key string, fallback int64) int64 {
	value, ok := conditions[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return fallback
	}
}

func nullableInt64Value(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}
