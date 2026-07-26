package adminauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

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
	var lockedUntil sql.NullTime
	err := s.db.QueryRowContext(ctx, `
SELECT
  ao.id::text,
  ao.login_name,
  alc.password_hash,
  ao.status,
  ao.auth_version,
  alc.failed_attempts,
  alc.locked_until,
  COALESCE(array_agg(DISTINCT ar.code::text) FILTER (WHERE ar.code IS NOT NULL), ARRAY[]::text[])
FROM admin_login_credentials alc
JOIN admin_operators ao ON ao.id = alc.operator_id
LEFT JOIN admin_operator_role_assignments aora ON aora.operator_id = ao.id
LEFT JOIN admin_roles ar ON ar.id = aora.role_id
WHERE ao.login_name = $1
GROUP BY ao.id, ao.login_name, alc.password_hash, ao.status, ao.auth_version, alc.failed_attempts, alc.locked_until
		`, loginName).Scan(&credential.OperatorID, &credential.LoginName, &credential.PasswordHash, &credential.Status, &credential.AuthVersion, &credential.FailedAttempts, &lockedUntil, &roles)
	if err == sql.ErrNoRows {
		return AdminCredential{}, ErrCredentialNotFound
	}
	if err != nil {
		return AdminCredential{}, err
	}
	credential.Roles = []string(roles)
	if lockedUntil.Valid {
		credential.LockedUntil = lockedUntil.Time
	}
	credential.RoleModules, err = s.listRoleAdminModules(ctx, credential.Roles)
	if err != nil {
		return AdminCredential{}, err
	}
	return credential, nil
}

func (s *SQLAdminStore) CountRecentFailedLoginAttemptsByIP(ctx context.Context, clientIP string, since time.Time) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM admin_login_attempts
WHERE client_ip = $1
  AND result = 'failed'
  AND created_at >= $2
`, clientIP, since).Scan(&count)
	return count, err
}

func (s *SQLAdminStore) RecordFailedLogin(ctx context.Context, loginName string, clientIP string, userAgent string, reason string, lockAfter int64, lockUntil time.Time) (bool, error) {
	var locked bool
	err := withAdminAuthTx(ctx, s.db, func(tx *sql.Tx) error {
		var operatorID sql.NullString
		var attempts sql.NullInt64
		var effectiveLockedUntil sql.NullTime
		err := tx.QueryRowContext(ctx, `
UPDATE admin_login_credentials alc
SET
  failed_attempts = alc.failed_attempts + 1,
  locked_until = CASE
    WHEN alc.failed_attempts + 1 >= $2 THEN $3
    ELSE alc.locked_until
  END,
  updated_at = now()
FROM admin_operators ao
WHERE alc.operator_id = ao.id
  AND ao.login_name = $1
RETURNING alc.operator_id::text, alc.failed_attempts, alc.locked_until
`, loginName, lockAfter, lockUntil).Scan(&operatorID, &attempts, &effectiveLockedUntil)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		locked = attempts.Valid && attempts.Int64 >= lockAfter
		_, err = tx.ExecContext(ctx, `
INSERT INTO admin_login_attempts (operator_id, login_name, client_ip, user_agent, result, failure_reason)
VALUES (NULLIF($1, '')::bigint, $2, $3, NULLIF($4, ''), 'failed', $5)
`, operatorID.String, loginName, clientIP, userAgent, reason)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO admin_security_alerts (alert_type, severity, event_key, details)
VALUES (
  'account_bruteforce',
  'high',
  'account_bruteforce:' || COALESCE(NULLIF($1, ''), $2) || ':' || $3,
  jsonb_build_object('operatorId', NULLIF($1, ''), 'loginName', $2, 'clientIp', $4, 'lockedUntil', $3)
)
ON CONFLICT (event_key) DO NOTHING
`, operatorID.String, loginName, lockUntil.UTC().Format(time.RFC3339), clientIP)
		return err
	})
	return locked, err
}

func (s *SQLAdminStore) RecordBlockedLogin(ctx context.Context, loginName string, clientIP string, userAgent string, reason string) error {
	return withAdminAuthTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO admin_login_attempts (operator_id, login_name, client_ip, user_agent, result, failure_reason)
SELECT ao.id, $1, $2, NULLIF($3, ''), 'blocked', $4
FROM (SELECT 1) seed
LEFT JOIN admin_operators ao ON ao.login_name = $1
`, loginName, clientIP, userAgent, reason); err != nil {
			return err
		}
		if reason != "ip_rate_limited" {
			return nil
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO admin_security_alerts (alert_type, severity, event_key, details)
VALUES (
  'ip_rate_limited',
  'high',
  'ip_rate_limited:' || $1 || ':' || to_char(date_trunc('hour', now()), 'YYYYMMDDHH24'),
  jsonb_build_object('clientIp', $1, 'loginName', $2, 'window', '15m')
)
ON CONFLICT (event_key) DO NOTHING
`, clientIP, loginName)
		return err
	})
}

func (s *SQLAdminStore) RecordSuccessfulLogin(ctx context.Context, operatorID string, loginName string, clientIP string, userAgent string) error {
	return withAdminAuthTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
UPDATE admin_login_credentials
SET failed_attempts = 0, locked_until = NULL, last_login_at = now(), updated_at = now()
WHERE operator_id = $1
`, operatorID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE admin_operators SET last_login_at = now(), updated_at = now() WHERE id = $1
`, operatorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO admin_login_attempts (operator_id, login_name, client_ip, user_agent, result)
VALUES ($1, $2, $3, NULLIF($4, ''), 'success')
`, operatorID, loginName, clientIP, userAgent)
		return err
	})
}

func withAdminAuthTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
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
