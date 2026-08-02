package model

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestManagedMerchantQueriesOnlyReturnActiveMerchants(t *testing.T) {
	queries := map[string]string{
		"list managed merchants": listManagedMerchantsSQL,
		"first managed merchant": getFirstManagedMerchantSQL,
	}
	for name, query := range queries {
		if !strings.Contains(query, "AND m.status = 'active'") {
			t.Fatalf("%s query must filter inactive merchants:\n%s", name, query)
		}
	}
}

func TestDeleteUserAccountAnonymizesMerchantMapEventsInsideTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	expectDeleteUserAccountPrelude(mock, "101", "不再使用")
	mock.ExpectExec(`UPDATE merchant_map_events SET user_id = NULL WHERE user_id = \$1`).
		WithArgs("101").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE users SET phone = NULL, wechat_openid = 'deleted:' \|\| id::text, nickname = NULL, avatar_url = NULL, status = 'deleted', deleted_at = now\(\), updated_at = now\(\) WHERE id = \$1`).
		WithArgs("101").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := NewUserModel(db).DeleteUserAccount(context.Background(), " 101 ", " 不再使用 "); err != nil {
		t.Fatalf("DeleteUserAccount() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("transaction expectations: %v", err)
	}
}

func TestDeleteUserAccountRollsBackWhenMerchantMapEventAnonymizationFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	expectDeleteUserAccountPrelude(mock, "101", "")
	mock.ExpectExec(`UPDATE merchant_map_events SET user_id = NULL WHERE user_id = \$1`).
		WithArgs("101").
		WillReturnError(errors.New("database unavailable"))
	mock.ExpectRollback()

	err = NewUserModel(db).DeleteUserAccount(context.Background(), "101", "")
	if err == nil || !strings.Contains(err.Error(), "匿名化商家地图行为失败") {
		t.Fatalf("DeleteUserAccount() error = %v, want safe Chinese diagnostic", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("rollback expectations: %v", err)
	}
}

func expectDeleteUserAccountPrelude(mock sqlmock.Sqlmock, userID string, reason string) {
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM users WHERE id = \$1 AND deleted_at IS NULL FOR UPDATE`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(UserStatusActive))
	mock.ExpectExec(`INSERT INTO user_account_deletion_requests \(user_id, status, reason, completed_at, snapshot\) VALUES \(\$1, 'completed', NULLIF\(\$2, ''\), now\(\), '\{"channel":"wechat_mini_program"\}'::jsonb\)`).
		WithArgs(userID, reason).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE merchant_admin_bindings SET status = 'inactive', revoked_at = COALESCE\(revoked_at, now\(\)\) WHERE user_id = \$1 AND status = 'active'`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE user_consents SET withdrawn_at = COALESCE\(withdrawn_at, now\(\)\), updated_at = now\(\) WHERE user_id = \$1 AND withdrawn_at IS NULL`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	for _, query := range []string{
		`UPDATE user_favorite_resources SET status = 'inactive', updated_at = now\(\) WHERE user_id = \$1`,
		`UPDATE user_followed_merchants SET status = 'inactive', updated_at = now\(\) WHERE user_id = \$1`,
		`UPDATE user_saved_searches SET deleted_at = COALESCE\(deleted_at, now\(\)\), updated_at = now\(\) WHERE user_id = \$1`,
		`UPDATE search_logs SET user_id = NULL WHERE user_id = \$1`,
		`UPDATE resource_contact_events SET user_id = NULL WHERE user_id = \$1`,
	} {
		mock.ExpectExec(query).WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))
	}
}
