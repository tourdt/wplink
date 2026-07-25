package adminauth

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"
)

type SQLAdminStore struct {
	db *sql.DB
}

func NewSQLAdminStore(db *sql.DB) *SQLAdminStore {
	return &SQLAdminStore{db: db}
}

func (s *SQLAdminStore) FindCredentialByLoginName(ctx context.Context, loginName string) (AdminCredential, error) {
	var credential AdminCredential
	var roles pq.StringArray
	err := s.db.QueryRowContext(ctx, `
SELECT
  ao.id::text,
  ao.login_name,
  alc.password_hash,
  ao.status,
  ao.auth_version,
  COALESCE(array_agg(DISTINCT ar.code::text) FILTER (WHERE ar.code IS NOT NULL), ARRAY[]::text[])
FROM admin_login_credentials alc
JOIN admin_operators ao ON ao.id = alc.operator_id
LEFT JOIN admin_operator_role_assignments aora ON aora.operator_id = ao.id
LEFT JOIN admin_roles ar ON ar.id = aora.role_id
WHERE ao.login_name = $1
GROUP BY ao.id, ao.login_name, alc.password_hash, ao.status, ao.auth_version
	`, loginName).Scan(&credential.OperatorID, &credential.LoginName, &credential.PasswordHash, &credential.Status, &credential.AuthVersion, &roles)
	if err == sql.ErrNoRows {
		return AdminCredential{}, ErrCredentialNotFound
	}
	if err != nil {
		return AdminCredential{}, err
	}
	credential.Roles = []string(roles)
	credential.RoleModules, err = s.listRoleAdminModules(ctx, credential.Roles)
	if err != nil {
		return AdminCredential{}, err
	}
	return credential, nil
}

func (s *SQLAdminStore) FindAdminIdentityByLoginName(ctx context.Context, loginName string) (AdminCredential, error) {
	var credential AdminCredential
	var roles pq.StringArray
	err := s.db.QueryRowContext(ctx, `
SELECT
  ao.id::text,
  ao.login_name,
  COALESCE(alc.password_hash, ''),
  ao.status,
  ao.auth_version,
  COALESCE(array_agg(DISTINCT ar.code::text) FILTER (WHERE ar.code IS NOT NULL), ARRAY[]::text[])
FROM admin_operators ao
LEFT JOIN admin_login_credentials alc ON alc.operator_id = ao.id
LEFT JOIN admin_operator_role_assignments aora ON aora.operator_id = ao.id
LEFT JOIN admin_roles ar ON ar.id = aora.role_id
WHERE ao.login_name = $1
GROUP BY ao.id, ao.login_name, ao.status, ao.auth_version, alc.password_hash
	`, loginName).Scan(&credential.OperatorID, &credential.LoginName, &credential.PasswordHash, &credential.Status, &credential.AuthVersion, &roles)
	if err == sql.ErrNoRows {
		return AdminCredential{}, ErrCredentialNotFound
	}
	if err != nil {
		return AdminCredential{}, err
	}
	credential.Roles = []string(roles)
	credential.RoleModules, err = s.listRoleAdminModules(ctx, credential.Roles)
	if err != nil {
		return AdminCredential{}, err
	}
	return credential, nil
}

func (s *SQLAdminStore) FindCurrentAdminSession(ctx context.Context, operatorID string) (AdminCredential, error) {
	var credential AdminCredential
	var roles pq.StringArray
	err := s.db.QueryRowContext(ctx, `
SELECT
  ao.id::text,
  ao.login_name,
  ao.status,
  ao.auth_version,
  COALESCE(array_agg(DISTINCT ar.code::text) FILTER (WHERE ar.code IS NOT NULL), ARRAY[]::text[])
FROM admin_operators ao
LEFT JOIN admin_operator_role_assignments aora ON aora.operator_id = ao.id
LEFT JOIN admin_roles ar ON ar.id = aora.role_id
WHERE ao.id = $1
GROUP BY ao.id, ao.login_name, ao.status, ao.auth_version
	`, operatorID).Scan(
		&credential.OperatorID,
		&credential.LoginName,
		&credential.Status,
		&credential.AuthVersion,
		&roles,
	)
	if err == sql.ErrNoRows {
		return AdminCredential{}, ErrCredentialNotFound
	}
	if err != nil {
		return AdminCredential{}, err
	}
	credential.Roles = []string(roles)
	credential.RoleModules, err = s.listRoleAdminModules(ctx, credential.Roles)
	if err != nil {
		return AdminCredential{}, err
	}
	return credential, nil
}

func (s *SQLAdminStore) listRoleAdminModules(ctx context.Context, roles []string) (map[string][]string, error) {
	if len(roles) == 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT
  code,
  CASE
    WHEN jsonb_typeof(permissions) = 'object' AND jsonb_typeof(permissions->'adminModules') = 'array'
      THEN permissions->'adminModules'
    ELSE '[]'::jsonb
  END
FROM admin_roles
WHERE code = ANY($1)
	`, pq.Array(roles))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var role string
		var raw json.RawMessage
		if err := rows.Scan(&role, &raw); err != nil {
			return nil, err
		}
		var modules []string
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &modules); err != nil {
				return nil, err
			}
		}
		result[role] = modules
	}
	return result, rows.Err()
}
