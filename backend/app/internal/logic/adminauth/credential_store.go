package adminauth

import (
	"context"
	"database/sql"

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
  alc.user_id::text,
  alc.login_name,
  alc.password_hash,
  alc.status,
  COALESCE(array_agg(DISTINCT r.code) FILTER (WHERE r.code IS NOT NULL), ARRAY[]::text[])
FROM admin_login_credentials alc
LEFT JOIN user_role_assignments ura ON ura.user_id = alc.user_id
LEFT JOIN roles r ON r.id = ura.role_id
WHERE alc.login_name = $1
GROUP BY alc.user_id, alc.login_name, alc.password_hash, alc.status
`, loginName).Scan(&credential.UserID, &credential.LoginName, &credential.PasswordHash, &credential.Status, &roles)
	if err == sql.ErrNoRows {
		return AdminCredential{}, ErrCredentialNotFound
	}
	if err != nil {
		return AdminCredential{}, err
	}
	credential.Roles = []string(roles)
	return credential, nil
}

func (s *SQLAdminStore) FindAdminIdentityByLoginName(ctx context.Context, loginName string) (AdminCredential, error) {
	var credential AdminCredential
	var roles pq.StringArray
	err := s.db.QueryRowContext(ctx, `
SELECT
  u.id::text,
  COALESCE(alc.login_name, u.phone, ''),
  COALESCE(alc.password_hash, ''),
  CASE
    WHEN u.status <> 'active' THEN 'disabled'
    WHEN alc.id IS NOT NULL AND alc.status <> 'enabled' THEN 'disabled'
    ELSE 'enabled'
  END,
  COALESCE(array_agg(DISTINCT r.code) FILTER (WHERE r.code IS NOT NULL), ARRAY[]::text[])
FROM users u
LEFT JOIN admin_login_credentials alc ON alc.user_id = u.id
LEFT JOIN user_role_assignments ura ON ura.user_id = u.id
LEFT JOIN roles r ON r.id = ura.role_id
WHERE (u.phone = $1 OR alc.login_name = $1)
  AND u.deleted_at IS NULL
GROUP BY u.id, u.phone, u.status, alc.id, alc.login_name, alc.password_hash, alc.status
`, loginName).Scan(&credential.UserID, &credential.LoginName, &credential.PasswordHash, &credential.Status, &roles)
	if err == sql.ErrNoRows {
		return AdminCredential{}, ErrCredentialNotFound
	}
	if err != nil {
		return AdminCredential{}, err
	}
	credential.Roles = []string(roles)
	return credential, nil
}
