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

func TestRecordMerchantMapEventBypassesSqlxArgumentLogging(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	conn := &merchantMapEventPrivacyConn{SqlConn: sqlx.NewSqlConnFromDB(db)}
	store := NewMerchantMapEventsModel(conn)
	mock.ExpectExec(`INSERT INTO "public"."merchant_map_events"`).
		WithArgs("101", "202", "303", "visitor-private", "session-private", "nearby_list_item_click", "merchant_location").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.RecordMerchantMapEvent(context.Background(), MerchantMapEventInput{
		UserID:           "101",
		MerchantID:       "202",
		TargetMerchantID: "303",
		VisitorKey:       "visitor-private",
		SessionID:        "session-private",
		EventType:        "nearby_list_item_click",
		Source:           "merchant_location",
	})
	if err != nil {
		t.Fatalf("RecordMerchantMapEvent() error = %v", err)
	}
	if conn.execCtxCalled {
		t.Fatal("RecordMerchantMapEvent() used sqlx ExecCtx, which expands visitor/session arguments into SQL logs")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("raw database expectations: %v", err)
	}
}

func TestMerchantMapEventsModelDeletesEventsBeforeCutoff(t *testing.T) {
	cutoff := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)
	conn := &capturingMerchantMapEventConn{rowsAffected: 7}
	store := NewMerchantMapEventsModel(conn)

	deleted, err := store.DeleteMerchantMapEventsBefore(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("DeleteMerchantMapEventsBefore() error = %v", err)
	}
	if deleted != 7 {
		t.Fatalf("deleted = %d, want 7", deleted)
	}
	if !strings.Contains(conn.query, `DELETE FROM "public"."merchant_map_events"`) ||
		!strings.Contains(conn.query, "WHERE created_at < $1") {
		t.Fatalf("query = %q, want cutoff delete on merchant_map_events", conn.query)
	}
	if len(conn.args) != 1 || conn.args[0] != cutoff {
		t.Fatalf("args = %#v, want literal cutoff %v", conn.args, cutoff)
	}
}

type capturingMerchantMapEventConn struct {
	sqlx.SqlConn
	query        string
	args         []any
	rowsAffected int64
}

type merchantMapEventPrivacyConn struct {
	sqlx.SqlConn
	execCtxCalled bool
}

func (c *merchantMapEventPrivacyConn) ExecCtx(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	c.execCtxCalled = true
	return merchantMapEventCleanupResult(1), nil
}

func (c *capturingMerchantMapEventConn) ExecCtx(_ context.Context, query string, args ...any) (sql.Result, error) {
	c.query = query
	c.args = args
	return merchantMapEventCleanupResult(c.rowsAffected), nil
}

type merchantMapEventCleanupResult int64

func (r merchantMapEventCleanupResult) LastInsertId() (int64, error) { return 0, nil }
func (r merchantMapEventCleanupResult) RowsAffected() (int64, error) { return int64(r), nil }
