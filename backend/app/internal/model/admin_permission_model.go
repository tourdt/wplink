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
	OperatorID   string
	LoginName    string
	RealName     string
	PasswordHash string
	Status       string
	Roles        []string
	ActorID      string
}

type AdminOperatorStatusInput struct {
	OperatorID string
	Status     string
	ActorID    string
}

type AdminRoleModulePermissionInput struct {
	RoleCode   string
	Modules    []string
	OperatorID string
}

type AdminOperatorItem struct {
	OperatorID  string
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
	FROM admin_roles
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
	UPDATE admin_roles
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
	    ao.id::text AS operator_id,
	    ao.login_name,
	    ao.real_name,
	    ao.status,
	    COALESCE(array_remove(array_agg(DISTINCT ar.code::text), NULL), ARRAY[]::text[]) AS roles,
	    ao.created_at,
	    COALESCE(ao.last_login_at, alc.last_login_at) AS last_login_at
	  FROM admin_operators ao
	  LEFT JOIN admin_login_credentials alc ON alc.operator_id = ao.id
	  LEFT JOIN admin_operator_role_assignments aora ON aora.operator_id = ao.id
	  LEFT JOIN admin_roles ar ON ar.id = aora.role_id AND ar.code IN ('platform_operator', 'super_admin')
	  WHERE 1 = 1
	    AND (
	      $1 = ''
	      OR ao.login_name ILIKE '%' || $1 || '%'
	      OR ao.real_name ILIKE '%' || $1 || '%'
	    )
	    AND (
	      $2 = ''
	      OR EXISTS (
	        SELECT 1
	        FROM admin_operator_role_assignments aora2
	        JOIN admin_roles ar2 ON ar2.id = aora2.role_id
	        WHERE aora2.operator_id = ao.id AND ar2.code = $2
	      )
	    )
	    AND ($3 = '' OR ao.status = $3)
	  GROUP BY ao.id, ao.login_name, ao.real_name, ao.status, ao.created_at, ao.last_login_at, alc.last_login_at
	)
	SELECT
	  operator_id,
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

		var operatorID string
		if err := tx.QueryRowContext(ctx, `
	INSERT INTO admin_operators (
	  login_name,
	  real_name,
	  status,
	  created_by
	)
	VALUES ($1, $2, $3, NULLIF($4, '')::bigint)
	RETURNING id::text
	`, loginName, strings.TrimSpace(input.RealName), input.Status, strings.TrimSpace(input.ActorID)).Scan(&operatorID); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
	INSERT INTO admin_login_credentials (
	  operator_id,
	  password_hash,
	  password_changed_at
	)
	VALUES ($1, $2, now())
	`, operatorID, input.PasswordHash); err != nil {
			return err
		}
		if err := assignAdminRolesTx(ctx, tx, operatorID, input.Roles); err != nil {
			return err
		}
		if err := recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   strings.TrimSpace(input.ActorID),
			OperatorRole: "super_admin",
			Action:       "admin_operator_create",
			ObjectType:   "admin_operator",
			ObjectID:     operatorID,
			AfterSnapshot: JSONMap{
				"loginName": loginName,
				"realName":  strings.TrimSpace(input.RealName),
				"status":    input.Status,
				"roles":     append([]string(nil), input.Roles...),
			},
		}); err != nil {
			return err
		}
		item, err = getAdminOperatorByOperatorIDTx(ctx, tx, operatorID)
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
		operatorID := strings.TrimSpace(input.OperatorID)
		before, err := getAdminOperatorByOperatorIDTx(ctx, tx, operatorID)
		if err != nil {
			return err
		}
		loginName := strings.TrimSpace(input.LoginName)
		if loginName == "" {
			loginName = before.LoginName
		}
		exists, err := adminLoginNameExists(ctx, tx, loginName, operatorID)
		if err != nil {
			return err
		}
		if exists {
			return ErrAdminOperatorLoginNameExists
		}

		if _, err := tx.ExecContext(ctx, `
	UPDATE admin_operators
	SET
	  login_name = $2,
	  real_name = $3,
	  status = $4,
	  updated_at = now()
	WHERE id = $1
	`, operatorID, loginName, strings.TrimSpace(input.RealName), input.Status); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
	UPDATE admin_login_credentials
	SET
	  password_hash = CASE WHEN $2 = '' THEN password_hash ELSE $2 END,
	  password_changed_at = CASE WHEN $2 = '' THEN password_changed_at ELSE now() END,
	  updated_at = now()
	WHERE operator_id = $1
	`, operatorID, strings.TrimSpace(input.PasswordHash)); err != nil {
			return err
		}
		if err := assignAdminRolesTx(ctx, tx, operatorID, input.Roles); err != nil {
			return err
		}
		if err := recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   strings.TrimSpace(input.ActorID),
			OperatorRole: "super_admin",
			Action:       "admin_operator_update",
			ObjectType:   "admin_operator",
			ObjectID:     operatorID,
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
		item, err = getAdminOperatorByOperatorIDTx(ctx, tx, operatorID)
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
		operatorID := strings.TrimSpace(input.OperatorID)
		before, err := getAdminOperatorByOperatorIDTx(ctx, tx, operatorID)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
	UPDATE admin_operators
	SET status = $2, updated_at = now()
	WHERE id = $1
	`, operatorID, input.Status); err != nil {
			return err
		}
		if err := recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   strings.TrimSpace(input.ActorID),
			OperatorRole: "super_admin",
			Action:       "admin_operator_status_update",
			ObjectType:   "admin_operator",
			ObjectID:     operatorID,
			BeforeSnapshot: JSONMap{
				"status": before.Status,
			},
			AfterSnapshot: JSONMap{
				"status": input.Status,
			},
		}); err != nil {
			return err
		}
		item, err = getAdminOperatorByOperatorIDTx(ctx, tx, operatorID)
		return err
	})
	if err != nil {
		return AdminOperatorItem{}, err
	}
	return item, nil
}

func adminLoginNameExists(ctx context.Context, tx *sql.Tx, loginName string, excludedOperatorID string) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
	SELECT EXISTS (
	  SELECT 1
	  FROM admin_operators
	  WHERE login_name = $1
	    AND ($2 = '' OR id <> $2::bigint)
	)
	`, strings.TrimSpace(loginName), strings.TrimSpace(excludedOperatorID)).Scan(&exists)
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
	FROM admin_roles
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

func assignAdminRolesTx(ctx context.Context, tx *sql.Tx, operatorID string, roles []string) error {
	if _, err := tx.ExecContext(ctx, `
	DELETE FROM admin_operator_role_assignments
	WHERE operator_id = $1
	`, strings.TrimSpace(operatorID)); err != nil {
		return err
	}
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
	INSERT INTO admin_operator_role_assignments (operator_id, role_id)
	SELECT $1, r.id
	FROM admin_roles r
	WHERE r.code = $2
	  AND NOT EXISTS (
	    SELECT 1
	    FROM admin_operator_role_assignments aora
	    WHERE aora.operator_id = $1
		      AND aora.role_id = r.id
	  )
	`, strings.TrimSpace(operatorID), role); err != nil {
			return err
		}
	}
	return nil
}

func getAdminOperatorByOperatorIDTx(ctx context.Context, tx *sql.Tx, operatorID string) (AdminOperatorItem, error) {
	item, _, err := scanAdminOperator(tx.QueryRowContext(ctx, `
	SELECT
	  ao.id::text,
	  ao.login_name,
	  ao.real_name,
	  ao.status,
	  COALESCE(array_remove(array_agg(DISTINCT ar.code::text), NULL), ARRAY[]::text[]) AS roles,
	  ao.created_at,
	  COALESCE(ao.last_login_at, alc.last_login_at) AS last_login_at,
	  1::bigint AS total
	FROM admin_operators ao
	LEFT JOIN admin_login_credentials alc ON alc.operator_id = ao.id
	LEFT JOIN admin_operator_role_assignments aora ON aora.operator_id = ao.id
	LEFT JOIN admin_roles ar ON ar.id = aora.role_id AND ar.code IN ('platform_operator', 'super_admin')
	WHERE ao.id = $1
	GROUP BY ao.id, ao.login_name, ao.real_name, ao.status, ao.created_at, ao.last_login_at, alc.last_login_at
	`, strings.TrimSpace(operatorID)))
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
	if err := scanner.Scan(&item.OperatorID, &item.LoginName, &item.RealName, &item.Status, &roles, &createdAt, &lastLoginAt, &total); err != nil {
		return AdminOperatorItem{}, 0, err
	}
	item.Roles = append([]string(nil), []string(roles)...)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	if lastLoginAt.Valid {
		item.LastLoginAt = lastLoginAt.Time.Format(time.RFC3339)
	}
	return item, total, nil
}
