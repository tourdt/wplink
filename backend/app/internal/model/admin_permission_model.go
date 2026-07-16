package model

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	AdminCredentialStatusEnabled  = "enabled"
	AdminCredentialStatusDisabled = "disabled"
)

var (
	ErrAdminOperatorNotFound        = errors.New("admin operator not found")
	ErrAdminOperatorLoginNameExists = errors.New("admin operator login name exists")
	ErrAdminRoleNotFound            = errors.New("admin role not found")
)

type AdminOperatorFilter struct {
	Keyword  string
	Role     string
	Status   string
	Page     int64
	PageSize int64
}

type AdminOperatorInput struct {
	UserID       string
	LoginName    string
	RealName     string
	PasswordHash string
	Status       string
	Roles        []string
	OperatorID   string
}

type AdminOperatorStatusInput struct {
	UserID     string
	Status     string
	OperatorID string
}

type AdminRoleModulePermissionInput struct {
	RoleCode   string
	Modules    []string
	OperatorID string
}

type AdminOperatorItem struct {
	UserID      string
	LoginName   string
	RealName    string
	Status      string
	Roles       []string
	CreatedAt   string
	LastLoginAt string
}

type AdminRoleModulePermission struct {
	RoleCode string
	Modules  []string
}

type ListAdminOperatorsResult struct {
	Items    []AdminOperatorItem
	Page     int64
	PageSize int64
	Total    int64
}

func (m *AdminPermissionModel) GetAdminRoleModulePermissions(ctx context.Context, roleCode string) (AdminRoleModulePermission, error) {
	var rolePermission AdminRoleModulePermission
	var rawModules JSONStringSlice
	err := m.db.QueryRowContext(ctx, `
SELECT
  code,
  CASE
    WHEN jsonb_typeof(permissions) = 'object' AND jsonb_typeof(permissions->'adminModules') = 'array'
      THEN permissions->'adminModules'
    ELSE '[]'::jsonb
  END
FROM roles
WHERE code = $1
`, strings.TrimSpace(roleCode)).Scan(&rolePermission.RoleCode, &rawModules)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminRoleModulePermission{}, ErrAdminRoleNotFound
	}
	if err != nil {
		return AdminRoleModulePermission{}, err
	}
	rolePermission.Modules = append([]string(nil), []string(rawModules)...)
	return rolePermission, nil
}

func (m *AdminPermissionModel) UpdateAdminRoleModulePermissions(ctx context.Context, input AdminRoleModulePermissionInput) (AdminRoleModulePermission, error) {
	var rolePermission AdminRoleModulePermission
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		before, roleID, err := getRoleAdminModulesTx(ctx, tx, strings.TrimSpace(input.RoleCode))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE roles
SET
  permissions = jsonb_set(
    CASE WHEN jsonb_typeof(permissions) = 'object' THEN permissions ELSE '{}'::jsonb END,
    '{adminModules}',
    $2::jsonb,
    true
  ),
  updated_at = now()
WHERE code = $1
`, strings.TrimSpace(input.RoleCode), JSONStringSlice(input.Modules)); err != nil {
			return err
		}
		if err := recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   strings.TrimSpace(input.OperatorID),
			OperatorRole: "super_admin",
			Action:       "admin_role_modules_update",
			ObjectType:   "admin_role",
			ObjectID:     roleID,
			BeforeSnapshot: JSONMap{
				"roleCode": strings.TrimSpace(input.RoleCode),
				"modules":  before.Modules,
			},
			AfterSnapshot: JSONMap{
				"roleCode": strings.TrimSpace(input.RoleCode),
				"modules":  append([]string(nil), input.Modules...),
			},
		}); err != nil {
			return err
		}
		rolePermission, _, err = getRoleAdminModulesTx(ctx, tx, strings.TrimSpace(input.RoleCode))
		return err
	})
	if err != nil {
		return AdminRoleModulePermission{}, err
	}
	return rolePermission, nil
}

type AdminPermissionModel struct {
	db *sql.DB
}

func NewAdminPermissionModel(db *sql.DB) *AdminPermissionModel {
	return &AdminPermissionModel{db: db}
}

func (m *AdminPermissionModel) ListAdminOperators(ctx context.Context, filter AdminOperatorFilter) (ListAdminOperatorsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
WITH filtered AS (
  SELECT
    u.id::text AS user_id,
    alc.login_name,
    COALESCE(NULLIF(aop.real_name, ''), NULLIF(u.nickname, ''), alc.login_name) AS real_name,
    alc.status,
    COALESCE(array_remove(array_agg(DISTINCT r.code), NULL), ARRAY[]::text[]) AS roles,
    alc.created_at,
    alc.last_login_at
  FROM admin_login_credentials alc
  JOIN users u ON u.id = alc.user_id
  LEFT JOIN admin_operator_profiles aop ON aop.user_id = u.id
  LEFT JOIN user_role_assignments ura ON ura.user_id = u.id
  LEFT JOIN roles r ON r.id = ura.role_id AND r.code IN ('platform_operator', 'super_admin')
  WHERE u.deleted_at IS NULL
    AND (
      $1 = ''
      OR alc.login_name ILIKE '%' || $1 || '%'
      OR COALESCE(aop.real_name, '') ILIKE '%' || $1 || '%'
      OR COALESCE(u.nickname, '') ILIKE '%' || $1 || '%'
      OR COALESCE(u.phone, '') ILIKE '%' || $1 || '%'
    )
    AND (
      $2 = ''
      OR EXISTS (
        SELECT 1
        FROM user_role_assignments ura2
        JOIN roles r2 ON r2.id = ura2.role_id
        WHERE ura2.user_id = u.id AND r2.code = $2
      )
    )
    AND ($3 = '' OR alc.status = $3)
  GROUP BY u.id, alc.login_name, aop.real_name, u.nickname, alc.status, alc.created_at, alc.last_login_at
)
SELECT
  user_id,
  login_name,
  real_name,
  status,
  roles,
  created_at,
  last_login_at,
  COUNT(*) OVER() AS total
FROM filtered
ORDER BY created_at DESC
LIMIT $4 OFFSET $5
`, strings.TrimSpace(filter.Keyword), strings.TrimSpace(filter.Role), strings.TrimSpace(filter.Status), pageSize, offset)
	if err != nil {
		return ListAdminOperatorsResult{}, err
	}
	defer rows.Close()

	result := ListAdminOperatorsResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		item, total, err := scanAdminOperator(rows)
		if err != nil {
			return ListAdminOperatorsResult{}, err
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return ListAdminOperatorsResult{}, err
	}
	return result, nil
}

func (m *AdminPermissionModel) CreateAdminOperator(ctx context.Context, input AdminOperatorInput) (AdminOperatorItem, error) {
	var item AdminOperatorItem
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		// 后台管理员账号涉及登录凭据、资料、角色和审计日志，必须同事务写入，避免创建半成品账号。
		loginName := strings.TrimSpace(input.LoginName)
		exists, err := adminLoginNameExists(ctx, tx, loginName, "")
		if err != nil {
			return err
		}
		if exists {
			return ErrAdminOperatorLoginNameExists
		}

		userID, err := upsertAdminUserTx(ctx, tx, loginName, strings.TrimSpace(input.RealName))
		if err != nil {
			return err
		}
		if exists, err := adminCredentialExistsForUser(ctx, tx, userID); err != nil {
			return err
		} else if exists {
			return ErrAdminOperatorLoginNameExists
		}

		if _, err := tx.ExecContext(ctx, `
INSERT INTO admin_login_credentials (
  user_id,
  login_name,
  password_hash,
  status,
  password_changed_at,
  created_by
)
VALUES ($1, $2, $3, $4, now(), NULLIF($5, '')::bigint)
`, userID, loginName, input.PasswordHash, input.Status, strings.TrimSpace(input.OperatorID)); err != nil {
			return err
		}
		if err := upsertAdminProfileTx(ctx, tx, userID, strings.TrimSpace(input.RealName), "active"); err != nil {
			return err
		}
		if err := assignAdminRolesTx(ctx, tx, userID, input.Roles); err != nil {
			return err
		}
		if err := recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   strings.TrimSpace(input.OperatorID),
			OperatorRole: "super_admin",
			Action:       "admin_operator_create",
			ObjectType:   "admin_operator",
			ObjectID:     userID,
			AfterSnapshot: JSONMap{
				"loginName": loginName,
				"realName":  strings.TrimSpace(input.RealName),
				"status":    input.Status,
				"roles":     append([]string(nil), input.Roles...),
			},
		}); err != nil {
			return err
		}
		item, err = getAdminOperatorByUserIDTx(ctx, tx, userID)
		return err
	})
	if err != nil {
		return AdminOperatorItem{}, err
	}
	return item, nil
}

func (m *AdminPermissionModel) UpdateAdminOperator(ctx context.Context, input AdminOperatorInput) (AdminOperatorItem, error) {
	var item AdminOperatorItem
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		userID := strings.TrimSpace(input.UserID)
		before, err := getAdminOperatorByUserIDTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		loginName := strings.TrimSpace(input.LoginName)
		if loginName == "" {
			loginName = before.LoginName
		}
		exists, err := adminLoginNameExists(ctx, tx, loginName, userID)
		if err != nil {
			return err
		}
		if exists {
			return ErrAdminOperatorLoginNameExists
		}
		exists, err = userLoginNameExists(ctx, tx, loginName, userID)
		if err != nil {
			return err
		}
		if exists {
			return ErrAdminOperatorLoginNameExists
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE users
SET phone = $2, nickname = $3, updated_at = now()
WHERE id = $1
`, userID, loginName, strings.TrimSpace(input.RealName)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE admin_login_credentials
SET
  login_name = $2,
  status = $3,
  password_hash = CASE WHEN $4 = '' THEN password_hash ELSE $4 END,
  password_changed_at = CASE WHEN $4 = '' THEN password_changed_at ELSE now() END,
  updated_at = now()
WHERE user_id = $1
`, userID, loginName, input.Status, strings.TrimSpace(input.PasswordHash)); err != nil {
			return err
		}
		if err := upsertAdminProfileTx(ctx, tx, userID, strings.TrimSpace(input.RealName), "active"); err != nil {
			return err
		}
		if err := assignAdminRolesTx(ctx, tx, userID, input.Roles); err != nil {
			return err
		}
		if err := recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   strings.TrimSpace(input.OperatorID),
			OperatorRole: "super_admin",
			Action:       "admin_operator_update",
			ObjectType:   "admin_operator",
			ObjectID:     userID,
			BeforeSnapshot: JSONMap{
				"loginName": before.LoginName,
				"realName":  before.RealName,
				"status":    before.Status,
				"roles":     before.Roles,
			},
			AfterSnapshot: JSONMap{
				"loginName": loginName,
				"realName":  strings.TrimSpace(input.RealName),
				"status":    input.Status,
				"roles":     append([]string(nil), input.Roles...),
			},
		}); err != nil {
			return err
		}
		item, err = getAdminOperatorByUserIDTx(ctx, tx, userID)
		return err
	})
	if err != nil {
		return AdminOperatorItem{}, err
	}
	return item, nil
}

func (m *AdminPermissionModel) UpdateAdminOperatorStatus(ctx context.Context, input AdminOperatorStatusInput) (AdminOperatorItem, error) {
	var item AdminOperatorItem
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		userID := strings.TrimSpace(input.UserID)
		before, err := getAdminOperatorByUserIDTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE admin_login_credentials
SET status = $2, updated_at = now()
WHERE user_id = $1
`, userID, input.Status); err != nil {
			return err
		}
		if err := recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   strings.TrimSpace(input.OperatorID),
			OperatorRole: "super_admin",
			Action:       "admin_operator_status_update",
			ObjectType:   "admin_operator",
			ObjectID:     userID,
			BeforeSnapshot: JSONMap{
				"status": before.Status,
			},
			AfterSnapshot: JSONMap{
				"status": input.Status,
			},
		}); err != nil {
			return err
		}
		item, err = getAdminOperatorByUserIDTx(ctx, tx, userID)
		return err
	})
	if err != nil {
		return AdminOperatorItem{}, err
	}
	return item, nil
}

func adminLoginNameExists(ctx context.Context, tx *sql.Tx, loginName string, excludedUserID string) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM admin_login_credentials
  WHERE login_name = $1
    AND ($2 = '' OR user_id <> $2::bigint)
)
`, strings.TrimSpace(loginName), strings.TrimSpace(excludedUserID)).Scan(&exists)
	return exists, err
}

func adminCredentialExistsForUser(ctx context.Context, tx *sql.Tx, userID string) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM admin_login_credentials
  WHERE user_id = $1
)
`, strings.TrimSpace(userID)).Scan(&exists)
	return exists, err
}

func userLoginNameExists(ctx context.Context, tx *sql.Tx, loginName string, excludedUserID string) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM users
  WHERE phone = $1
    AND id <> $2::bigint
    AND deleted_at IS NULL
)
`, strings.TrimSpace(loginName), strings.TrimSpace(excludedUserID)).Scan(&exists)
	return exists, err
}

func getRoleAdminModulesTx(ctx context.Context, tx *sql.Tx, roleCode string) (AdminRoleModulePermission, string, error) {
	var roleID string
	var rolePermission AdminRoleModulePermission
	var rawModules JSONStringSlice
	err := tx.QueryRowContext(ctx, `
SELECT
  id::text,
  code,
  CASE
    WHEN jsonb_typeof(permissions) = 'object' AND jsonb_typeof(permissions->'adminModules') = 'array'
      THEN permissions->'adminModules'
    ELSE '[]'::jsonb
  END
FROM roles
WHERE code = $1
`, strings.TrimSpace(roleCode)).Scan(&roleID, &rolePermission.RoleCode, &rawModules)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminRoleModulePermission{}, "", ErrAdminRoleNotFound
	}
	if err != nil {
		return AdminRoleModulePermission{}, "", err
	}
	rolePermission.Modules = append([]string(nil), []string(rawModules)...)
	return rolePermission, roleID, nil
}

func upsertAdminUserTx(ctx context.Context, tx *sql.Tx, loginName string, realName string) (string, error) {
	var userID string
	err := tx.QueryRowContext(ctx, `
INSERT INTO users (phone, nickname, status)
VALUES ($1, $2, 'active')
ON CONFLICT (phone) DO UPDATE SET
  nickname = CASE WHEN EXCLUDED.nickname = '' THEN users.nickname ELSE EXCLUDED.nickname END,
  status = 'active',
  updated_at = now()
RETURNING id::text
`, strings.TrimSpace(loginName), strings.TrimSpace(realName)).Scan(&userID)
	return userID, err
}

func upsertAdminProfileTx(ctx context.Context, tx *sql.Tx, userID string, realName string, status string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO admin_operator_profiles (user_id, real_name, status)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE SET
  real_name = EXCLUDED.real_name,
  status = EXCLUDED.status,
  updated_at = now()
`, strings.TrimSpace(userID), strings.TrimSpace(realName), strings.TrimSpace(status))
	return err
}

func assignAdminRolesTx(ctx context.Context, tx *sql.Tx, userID string, roles []string) error {
	if _, err := tx.ExecContext(ctx, `
DELETE FROM user_role_assignments ura
USING roles r
WHERE ura.role_id = r.id
  AND ura.user_id = $1
  AND r.code IN ('platform_operator', 'super_admin')
`, strings.TrimSpace(userID)); err != nil {
		return err
	}
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO user_role_assignments (user_id, role_id)
SELECT $1, r.id
FROM roles r
WHERE r.code = $2
  AND NOT EXISTS (
    SELECT 1
    FROM user_role_assignments ura
    WHERE ura.user_id = $1
      AND ura.role_id = r.id
      AND ura.city_station_id IS NULL
      AND ura.merchant_id IS NULL
  )
`, strings.TrimSpace(userID), role); err != nil {
			return err
		}
	}
	return nil
}

func getAdminOperatorByUserIDTx(ctx context.Context, tx *sql.Tx, userID string) (AdminOperatorItem, error) {
	item, _, err := scanAdminOperator(tx.QueryRowContext(ctx, `
SELECT
  u.id::text,
  alc.login_name,
  COALESCE(NULLIF(aop.real_name, ''), NULLIF(u.nickname, ''), alc.login_name) AS real_name,
  alc.status,
  COALESCE(array_remove(array_agg(DISTINCT r.code), NULL), ARRAY[]::text[]) AS roles,
  alc.created_at,
  alc.last_login_at,
  1::bigint AS total
FROM admin_login_credentials alc
JOIN users u ON u.id = alc.user_id
LEFT JOIN admin_operator_profiles aop ON aop.user_id = u.id
LEFT JOIN user_role_assignments ura ON ura.user_id = u.id
LEFT JOIN roles r ON r.id = ura.role_id AND r.code IN ('platform_operator', 'super_admin')
WHERE u.id = $1
  AND u.deleted_at IS NULL
GROUP BY u.id, alc.login_name, aop.real_name, u.nickname, alc.status, alc.created_at, alc.last_login_at
`, strings.TrimSpace(userID)))
	if errors.Is(err, sql.ErrNoRows) {
		return AdminOperatorItem{}, ErrAdminOperatorNotFound
	}
	return item, err
}

type adminOperatorScanner interface {
	Scan(dest ...interface{}) error
}

func scanAdminOperator(scanner adminOperatorScanner) (AdminOperatorItem, int64, error) {
	var item AdminOperatorItem
	var roles pq.StringArray
	var createdAt time.Time
	var lastLoginAt sql.NullTime
	var total int64
	if err := scanner.Scan(&item.UserID, &item.LoginName, &item.RealName, &item.Status, &roles, &createdAt, &lastLoginAt, &total); err != nil {
		return AdminOperatorItem{}, 0, err
	}
	item.Roles = append([]string(nil), []string(roles)...)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	if lastLoginAt.Valid {
		item.LastLoginAt = lastLoginAt.Time.Format(time.RFC3339)
	}
	return item, total, nil
}
