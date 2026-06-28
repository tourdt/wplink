package model

import (
	"context"
	"database/sql"
	"strings"
)

type UpsertWechatUserInput struct {
	WechatOpenID    string
	DefaultCityCode string
}

type UserProfile struct {
	ID               string
	Phone            string
	WechatOpenID     string
	Nickname         string
	AvatarURL        string
	DefaultCityCode  string
	Roles            []string
	ManagedMerchants []ManagedMerchantInfo
}

type ManagedMerchantInfo struct {
	ID   string
	Name string
	Role string
}

type UserModel struct {
	db *sql.DB
}

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{db: db}
}

func (m *UserModel) UpsertWechatUser(ctx context.Context, input UpsertWechatUserInput) (UserProfile, error) {
	openID := strings.TrimSpace(input.WechatOpenID)
	defaultCityCode := strings.TrimSpace(input.DefaultCityCode)
	var userID string
	if err := m.db.QueryRowContext(ctx, `
WITH city AS (
  SELECT id FROM city_stations WHERE code = $2 AND status = 'active' LIMIT 1
),
upserted AS (
  INSERT INTO users (wechat_openid, default_city_station_id, status, last_login_at)
  VALUES ($1, (SELECT id FROM city), 'active', now())
  ON CONFLICT (wechat_openid) DO UPDATE SET
    default_city_station_id = COALESCE(EXCLUDED.default_city_station_id, users.default_city_station_id),
    status = 'active',
    last_login_at = now(),
    updated_at = now()
  RETURNING id
)
SELECT id FROM upserted
`, openID, defaultCityCode).Scan(&userID); err != nil {
		return UserProfile{}, err
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
  u.id,
  COALESCE(u.phone, ''),
  COALESCE(u.wechat_openid, ''),
  COALESCE(u.nickname, ''),
  COALESCE(u.avatar_url, ''),
  COALESCE(cs.code, '')
FROM users u
LEFT JOIN city_stations cs ON cs.id = u.default_city_station_id
WHERE u.id = $1 AND u.deleted_at IS NULL
`, strings.TrimSpace(userID)).Scan(&profile.ID, &profile.Phone, &profile.WechatOpenID, &profile.Nickname, &profile.AvatarURL, &profile.DefaultCityCode); err != nil {
		return UserProfile{}, err
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

func (m *UserModel) BindUserPhone(ctx context.Context, userID string, phone string) (UserProfile, error) {
	var updatedID string
	if err := m.db.QueryRowContext(ctx, `
UPDATE users
SET phone = $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id
`, strings.TrimSpace(userID), strings.TrimSpace(phone)).Scan(&updatedID); err != nil {
		return UserProfile{}, err
	}
	return m.GetUserProfile(ctx, updatedID)
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
	rows, err := m.db.QueryContext(ctx, `
SELECT m.id, m.name, mab.role
FROM merchant_admin_bindings mab
JOIN merchants m ON m.id = mab.merchant_id
WHERE mab.user_id = $1
  AND mab.status = 'active'
  AND m.deleted_at IS NULL
ORDER BY m.created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchants []ManagedMerchantInfo
	for rows.Next() {
		var merchant ManagedMerchantInfo
		if err := rows.Scan(&merchant.ID, &merchant.Name, &merchant.Role); err != nil {
			return nil, err
		}
		merchants = append(merchants, merchant)
	}
	return merchants, rows.Err()
}
