package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	UserStatusActive  = "active"
	UserStatusDeleted = "deleted"
)

var ErrUserDisabled = errors.New("user disabled")

type UpsertWechatUserInput struct {
	WechatOpenID    string
	DefaultCityCode string
}

type UserProfile struct {
	ID               string
	Status           string
	Phone            string
	WechatOpenID     string
	Nickname         string
	AvatarURL        string
	DefaultCityCode  string
	Roles            []string
	ManagedMerchants []ManagedMerchantInfo
}

type ManagedMerchantInfo struct {
	ID            string
	Name          string
	Role          string
	ProfileStatus string
}

type UserModel struct {
	db *sql.DB
}

const listManagedMerchantsSQL = `
SELECT m.id::text, m.name, mab.role, COALESCE(NULLIF(m.profile_status, ''), 'completed')
FROM merchant_admin_bindings mab
JOIN merchants m ON m.id = mab.merchant_id
WHERE mab.user_id = $1
  AND mab.status = 'active'
  AND m.deleted_at IS NULL
  AND m.status = 'active'
ORDER BY m.created_at DESC
`

const getFirstManagedMerchantSQL = `
SELECT m.id::text, m.name, mab.role, COALESCE(NULLIF(m.profile_status, ''), 'completed')
FROM merchant_admin_bindings mab
JOIN merchants m ON m.id = mab.merchant_id
WHERE mab.user_id = $1
  AND mab.status = 'active'
  AND m.deleted_at IS NULL
  AND m.status = 'active'
ORDER BY m.created_at DESC
LIMIT 1
`

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{db: db}
}

func (m *UserModel) UpsertWechatUser(ctx context.Context, input UpsertWechatUserInput) (UserProfile, error) {
	openID := strings.TrimSpace(input.WechatOpenID)
	defaultCityCode := strings.TrimSpace(input.DefaultCityCode)
	var userID string
	var status string
	if err := m.db.QueryRowContext(ctx, `
WITH city AS (
  SELECT id FROM city_stations WHERE code = $2 AND status = 'active' LIMIT 1
),
upserted AS (
  INSERT INTO users (wechat_openid, default_city_station_id, status, last_login_at)
  VALUES ($1, (SELECT id FROM city), 'active', now())
  ON CONFLICT (wechat_openid) DO UPDATE SET
    default_city_station_id = COALESCE(EXCLUDED.default_city_station_id, users.default_city_station_id),
    last_login_at = now(),
    updated_at = now()
  RETURNING id::text, status
)
SELECT id::text, status FROM upserted
`, openID, defaultCityCode).Scan(&userID, &status); err != nil {
		return UserProfile{}, err
	}
	// 登录只能更新最近登录时间，绝不能把后台停用或待注销账号重新激活。
	if status != UserStatusActive {
		return UserProfile{}, ErrUserDisabled
	}
	if err := m.ensureNormalUserRole(ctx, userID); err != nil {
		return UserProfile{}, err
	}
	return m.GetUserProfile(ctx, userID)
}

func (m *UserModel) GetUserProfile(ctx context.Context, userID string) (UserProfile, error) {
	var profile UserProfile
	if err := m.db.QueryRowContext(ctx, `
SELECT
  u.id::text,
  u.status,
  COALESCE(u.phone, ''),
  COALESCE(u.wechat_openid, ''),
  COALESCE(u.nickname, ''),
  COALESCE(u.avatar_url, ''),
  COALESCE(cs.code, '')
FROM users u
LEFT JOIN city_stations cs ON cs.id = u.default_city_station_id
WHERE u.id = $1 AND u.deleted_at IS NULL
`, strings.TrimSpace(userID)).Scan(&profile.ID, &profile.Status, &profile.Phone, &profile.WechatOpenID, &profile.Nickname, &profile.AvatarURL, &profile.DefaultCityCode); err != nil {
		return UserProfile{}, err
	}
	if profile.Status != UserStatusActive {
		return UserProfile{}, ErrUserDisabled
	}

	roles, err := m.listUserRoles(ctx, profile.ID)
	if err != nil {
		return UserProfile{}, err
	}
	managedMerchants, err := m.listManagedMerchants(ctx, profile.ID)
	if err != nil {
		return UserProfile{}, err
	}
	profile.Roles = roles
	profile.ManagedMerchants = managedMerchants
	return profile, nil
}

func (m *UserModel) IsUserActive(ctx context.Context, userID string) (bool, error) {
	var active bool
	err := m.db.QueryRowContext(ctx, `
SELECT status = 'active' AND deleted_at IS NULL
FROM users
WHERE id = $1
`, strings.TrimSpace(userID)).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return active, err
}

func (m *UserModel) RecordUserConsents(ctx context.Context, userID string, privacyVersion string, agreementVersion string) error {
	userID = strings.TrimSpace(userID)
	privacyVersion = strings.TrimSpace(privacyVersion)
	agreementVersion = strings.TrimSpace(agreementVersion)
	if userID == "" || privacyVersion == "" || agreementVersion == "" {
		return errors.New("consent fields are incomplete")
	}
	return WithTx(ctx, m.db, func(tx *sql.Tx) error {
		for _, consent := range []struct {
			consentType string
			version     string
		}{
			{consentType: "privacy_policy", version: privacyVersion},
			{consentType: "user_agreement", version: agreementVersion},
		} {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO user_consents (user_id, consent_type, version, source, consented_at)
VALUES ($1, $2, $3, 'wechat_mini_program', now())
ON CONFLICT (user_id, consent_type, version) DO UPDATE SET
  withdrawn_at = NULL,
  consented_at = now(),
  updated_at = now()
`, userID, consent.consentType, consent.version); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *UserModel) DeleteUserAccount(ctx context.Context, userID string, reason string) error {
	userID = strings.TrimSpace(userID)
	return WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRowContext(ctx, `
SELECT status
FROM users
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE
`, userID).Scan(&status); err != nil {
			return err
		}
		if status != UserStatusActive {
			return ErrUserDisabled
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO user_account_deletion_requests (user_id, status, reason, completed_at, snapshot)
VALUES ($1, 'completed', NULLIF($2, ''), now(), '{"channel":"wechat_mini_program"}'::jsonb)
`, userID, strings.TrimSpace(reason)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE merchant_admin_bindings
SET status = 'inactive', revoked_at = COALESCE(revoked_at, now())
WHERE user_id = $1 AND status = 'active'
`, userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE user_consents
SET withdrawn_at = COALESCE(withdrawn_at, now()), updated_at = now()
WHERE user_id = $1 AND withdrawn_at IS NULL
`, userID); err != nil {
			return err
		}
		// 收藏、关注和搜索订阅属于可直接删除的个人偏好；交易、支付等依法或履约所需记录只保留匿名用户外键。
		if _, err := tx.ExecContext(ctx, `UPDATE user_favorite_resources SET status = 'inactive', updated_at = now() WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_followed_merchants SET status = 'inactive', updated_at = now() WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_saved_searches SET deleted_at = COALESCE(deleted_at, now()), updated_at = now() WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE search_logs SET user_id = NULL WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE resource_contact_events SET user_id = NULL WHERE user_id = $1`, userID); err != nil {
			return err
		}
		// 地图搜索、导航和周边点击仍需保留聚合价值，但注销后不得继续关联到具体用户。
		// 该更新必须与账号软删除共用事务，失败时整体回滚，避免出现“账号已注销但行为仍可识别”的中间状态。
		if _, err := tx.ExecContext(ctx, `UPDATE merchant_map_events SET user_id = NULL WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("匿名化商家地图行为失败: %w", err)
		}
		_, err := tx.ExecContext(ctx, `
UPDATE users
SET
  phone = NULL,
  wechat_openid = 'deleted:' || id::text,
  nickname = NULL,
  avatar_url = NULL,
  status = 'deleted',
  deleted_at = now(),
  updated_at = now()
WHERE id = $1
`, userID)
		return err
	})
}

func (m *UserModel) GetUserWechatOpenID(ctx context.Context, userID string) (string, error) {
	var openID string
	err := m.db.QueryRowContext(ctx, `
SELECT COALESCE(wechat_openid, '')
FROM users
WHERE id = $1
  AND deleted_at IS NULL
`, strings.TrimSpace(userID)).Scan(&openID)
	return openID, err
}

func (m *UserModel) BindUserPhone(ctx context.Context, userID string, phone string) (UserProfile, error) {
	var updatedID string
	if err := m.db.QueryRowContext(ctx, `
UPDATE users
SET phone = $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id::text
`, strings.TrimSpace(userID), strings.TrimSpace(phone)).Scan(&updatedID); err != nil {
		return UserProfile{}, err
	}
	return m.GetUserProfile(ctx, updatedID)
}

func (m *UserModel) EnsureDefaultMerchantForUser(ctx context.Context, userID string, cityCode string) (ManagedMerchantInfo, error) {
	userID = strings.TrimSpace(userID)
	cityCode = strings.TrimSpace(cityCode)
	if cityCode == "" {
		cityCode = "zhili"
	}

	var merchant ManagedMerchantInfo
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		// 登录兜底商家只允许每个用户自动创建一个；事务级锁避免并发登录重复生成空资料商家。
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "default_merchant:"+userID); err != nil {
			return err
		}

		existing, err := getFirstManagedMerchant(ctx, tx, userID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			merchant = existing
			return nil
		}

		if err := tx.QueryRowContext(ctx, `
WITH city_candidates AS (
  SELECT id, 1 AS priority FROM city_stations WHERE code = $1 AND status = 'active'
  UNION ALL
  SELECT id, 2 AS priority FROM city_stations WHERE code = 'zhili' AND status = 'active'
  UNION ALL
  SELECT id, 3 AS priority FROM city_stations WHERE status = 'active'
),
created AS (
  INSERT INTO merchants (
    city_station_id,
    name,
    merchant_type,
    main_categories,
    contact_name,
    contact_phone,
    profile_status
  )
  SELECT id, '微信用户', 'individual', '[]'::jsonb, '', '', $2
  FROM city_candidates
  ORDER BY priority
  LIMIT 1
  RETURNING id::text, name, profile_status
)
SELECT id, name, profile_status FROM created
`, cityCode, MerchantProfileStatusIncomplete).Scan(&merchant.ID, &merchant.Name, &merchant.ProfileStatus); err != nil {
			return err
		}
		merchant.Role = "owner"
		_, err = tx.ExecContext(ctx, `
INSERT INTO merchant_admin_bindings (merchant_id, user_id, role, status, created_by)
VALUES ($1, $2, 'owner', 'active', $2)
`, merchant.ID, userID)
		return err
	})
	if err != nil {
		return ManagedMerchantInfo{}, err
	}
	merchant.ProfileStatus = normalizeMerchantProfileStatus(merchant.ProfileStatus)
	return merchant, nil
}

func (m *UserModel) UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error) {
	var exists bool
	if err := m.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM merchant_admin_bindings mab
  JOIN merchants m ON m.id = mab.merchant_id
  WHERE mab.user_id = $1
    AND mab.merchant_id = $2
    AND mab.status = 'active'
    AND m.deleted_at IS NULL
    AND m.status = 'active'
)
`, strings.TrimSpace(userID), strings.TrimSpace(merchantID)).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (m *UserModel) ListManagedMerchantIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT m.id::text
FROM merchant_admin_bindings mab
JOIN merchants m ON m.id = mab.merchant_id
WHERE mab.user_id = $1
  AND mab.status = 'active'
  AND m.deleted_at IS NULL
  AND m.status = 'active'
ORDER BY m.created_at DESC
`, strings.TrimSpace(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchantIDs []string
	for rows.Next() {
		var merchantID string
		if err := rows.Scan(&merchantID); err != nil {
			return nil, err
		}
		merchantID = strings.TrimSpace(merchantID)
		if merchantID != "" {
			merchantIDs = append(merchantIDs, merchantID)
		}
	}
	return merchantIDs, rows.Err()
}

func (m *UserModel) ensureNormalUserRole(ctx context.Context, userID string) error {
	_, err := m.db.ExecContext(ctx, `
INSERT INTO user_role_assignments (user_id, role_id)
SELECT $1, r.id
FROM roles r
WHERE r.code = 'normal_user'
  AND NOT EXISTS (
    SELECT 1
    FROM user_role_assignments ura
    WHERE ura.user_id = $1
      AND ura.role_id = r.id
      AND ura.city_station_id IS NULL
      AND ura.merchant_id IS NULL
  )
`, userID)
	return err
}

func (m *UserModel) listUserRoles(ctx context.Context, userID string) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT DISTINCT r.code
FROM user_role_assignments ura
JOIN roles r ON r.id = ura.role_id
WHERE ura.user_id = $1
ORDER BY r.code ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (m *UserModel) listManagedMerchants(ctx context.Context, userID string) ([]ManagedMerchantInfo, error) {
	rows, err := m.db.QueryContext(ctx, listManagedMerchantsSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchants []ManagedMerchantInfo
	for rows.Next() {
		var merchant ManagedMerchantInfo
		if err := rows.Scan(&merchant.ID, &merchant.Name, &merchant.Role, &merchant.ProfileStatus); err != nil {
			return nil, err
		}
		merchant.ProfileStatus = normalizeMerchantProfileStatus(merchant.ProfileStatus)
		merchants = append(merchants, merchant)
	}
	return merchants, rows.Err()
}

func getFirstManagedMerchant(ctx context.Context, tx *sql.Tx, userID string) (ManagedMerchantInfo, error) {
	var merchant ManagedMerchantInfo
	err := tx.QueryRowContext(ctx, getFirstManagedMerchantSQL, userID).Scan(&merchant.ID, &merchant.Name, &merchant.Role, &merchant.ProfileStatus)
	merchant.ProfileStatus = normalizeMerchantProfileStatus(merchant.ProfileStatus)
	return merchant, err
}

func normalizeMerchantProfileStatus(status string) string {
	switch strings.TrimSpace(status) {
	case MerchantProfileStatusIncomplete:
		return MerchantProfileStatusIncomplete
	default:
		return MerchantProfileStatusCompleted
	}
}
