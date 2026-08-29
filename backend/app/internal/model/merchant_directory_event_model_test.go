package model

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestRecordMerchantDirectoryEventBypassesSqlxArgumentLogging(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	conn := &merchantDirectoryEventPrivacyConn{SqlConn: sqlx.NewSqlConnFromDB(db)}
	store := NewMerchantDirectoryEventModel(conn)
	mock.ExpectExec(`INSERT INTO "public"."merchant_directory_events"`).
		WithArgs("101", "202", "303", "visitor-private", "session-private", "nearby_list_item_click", "merchant_location").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.RecordMerchantDirectoryEvent(context.Background(), MerchantDirectoryEventInput{
		UserID:           "101",
		MerchantID:       "202",
		TargetMerchantID: "303",
		VisitorKey:       "visitor-private",
		SessionID:        "session-private",
		EventType:        "nearby_list_item_click",
		Source:           "merchant_location",
	})
	if err != nil {
		t.Fatalf("RecordMerchantDirectoryEvent() error = %v", err)
	}
	if conn.execCtxCalled {
		t.Fatal("RecordMerchantDirectoryEvent() used sqlx ExecCtx, which expands visitor/session arguments into SQL logs")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("raw database expectations: %v", err)
	}
}

func TestMerchantDirectoryEventModelDeletesEventsBeforeCutoff(t *testing.T) {
	cutoff := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)
	conn := &capturingMerchantDirectoryEventConn{rowsAffected: 7}
	store := NewMerchantDirectoryEventModel(conn)

	deleted, err := store.DeleteMerchantDirectoryEventsBefore(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("DeleteMerchantDirectoryEventsBefore() error = %v", err)
	}
	if deleted != 7 {
		t.Fatalf("deleted = %d, want 7", deleted)
	}
	if !strings.Contains(conn.query, `DELETE FROM "public"."merchant_directory_events"`) ||
		!strings.Contains(conn.query, "WHERE created_at < $1") {
		t.Fatalf("query = %q, want cutoff delete on merchant_directory_events", conn.query)
	}
	if len(conn.args) != 1 || conn.args[0] != cutoff {
		t.Fatalf("args = %#v, want literal cutoff %v", conn.args, cutoff)
	}
}

type capturingMerchantDirectoryEventConn struct {
	sqlx.SqlConn
	query        string
	args         []any
	rowsAffected int64
}

type merchantDirectoryEventPrivacyConn struct {
	sqlx.SqlConn
	execCtxCalled bool
}

func (c *merchantDirectoryEventPrivacyConn) ExecCtx(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	c.execCtxCalled = true
	return merchantDirectoryEventCleanupResult(1), nil
}

func (c *capturingMerchantDirectoryEventConn) ExecCtx(_ context.Context, query string, args ...any) (sql.Result, error) {
	c.query = query
	c.args = args
	return merchantDirectoryEventCleanupResult(c.rowsAffected), nil
}

type merchantDirectoryEventCleanupResult int64

func (r merchantDirectoryEventCleanupResult) LastInsertId() (int64, error) { return 0, nil }
func (r merchantDirectoryEventCleanupResult) RowsAffected() (int64, error) { return int64(r), nil }
