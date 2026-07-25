package model

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMarkContactUnlockOrderPaidCreatesUnlockForPendingOrder(t *testing.T) {
	state := &contactUnlockPaymentTestState{orderStatus: PaymentOrderStatusPending}
	db := openContactUnlockPaymentTestDB(t, state)
	defer db.Close()

	result, err := NewResourceContactUnlockModel(db).MarkContactUnlockOrderPaid(context.Background(), MarkContactUnlockOrderPaidInput{
		OutTradeNo:    "CU202607140001",
		TransactionID: "wx-transaction-1",
		AmountTotal:   500,
	})
	if err != nil {
		t.Fatalf("MarkContactUnlockOrderPaid() error = %v", err)
	}
	if result.Status != PaymentOrderStatusPaid || result.OrderID != "order-1" || result.ResourceID != "resource-1" {
		t.Fatalf("result = %#v, want paid contact unlock order", result)
	}
	if got := state.unlockInserts.Load(); got != 1 {
		t.Fatalf("unlock inserts = %d, want 1 for first paid callback", got)
	}
}

func TestMarkContactUnlockOrderPaidSkipsUnlockForDuplicatePaidOrder(t *testing.T) {
	state := &contactUnlockPaymentTestState{orderStatus: PaymentOrderStatusPaid}
	db := openContactUnlockPaymentTestDB(t, state)
	defer db.Close()

	result, err := NewResourceContactUnlockModel(db).MarkContactUnlockOrderPaid(context.Background(), MarkContactUnlockOrderPaidInput{
		OutTradeNo:    "CU202607140002",
		TransactionID: "wx-transaction-2",
		AmountTotal:   500,
	})
	if err != nil {
		t.Fatalf("MarkContactUnlockOrderPaid() error = %v", err)
	}
	if result.Status != PaymentOrderStatusPaid {
		t.Fatalf("result = %#v, want paid duplicate callback", result)
	}
	if got := state.unlockInserts.Load(); got != 0 {
		t.Fatalf("unlock inserts = %d, want 0 for duplicate paid callback", got)
	}
}

func TestContactUnlockOrderSQLKeepsCommercialRuleSnapshot(t *testing.T) {
	requiredResourceSnippets := []string{
		"r.resource_type_snapshot -> 'commercialRules'",
		"r.status = 'published'",
	}
	for _, snippet := range requiredResourceSnippets {
		if !strings.Contains(createContactUnlockOrderResourceSQL, snippet) {
			t.Fatalf("createContactUnlockOrderResourceSQL missing %q:\n%s", snippet, createContactUnlockOrderResourceSQL)
		}
	}
	requiredInsertSnippets := []string{
		"INSERT INTO resource_contact_unlock_orders",
		"commercial_rules_snapshot",
		"RETURNING id::text, out_trade_no, price_cent, currency, status",
	}
	for _, snippet := range requiredInsertSnippets {
		if !strings.Contains(insertContactUnlockOrderSQL, snippet) {
			t.Fatalf("insertContactUnlockOrderSQL missing %q:\n%s", snippet, insertContactUnlockOrderSQL)
		}
	}
}

func TestContactUnlockStateSQLChecksViewerAndVIP(t *testing.T) {
	requiredUnlockSnippets := []string{
		"FROM resource_contact_unlocks",
		"resource_id = $1",
		"user_id = $2",
		"expires_at > now()",
	}
	for _, snippet := range requiredUnlockSnippets {
		if !strings.Contains(hasActiveContactUnlockSQL, snippet) {
			t.Fatalf("hasActiveContactUnlockSQL missing %q:\n%s", snippet, hasActiveContactUnlockSQL)
		}
	}
	requiredVIPSnippets := []string{
		"merchant_vip_subscriptions",
		"merchant_admin_bindings",
		"mab.user_id = $1",
		"mvs.expires_at > now()",
	}
	for _, snippet := range requiredVIPSnippets {
		if !strings.Contains(findActiveVIPManagedMerchantSQL, snippet) {
			t.Fatalf("findActiveVIPManagedMerchantSQL missing %q:\n%s", snippet, findActiveVIPManagedMerchantSQL)
		}
	}
}

func openContactUnlockPaymentTestDB(t *testing.T, state *contactUnlockPaymentTestState) *sql.DB {
	t.Helper()
	driverName := fmt.Sprintf("contact-unlock-payment-test-%d", contactUnlockPaymentTestDriverSeq.Add(1))
	sql.Register(driverName, contactUnlockPaymentTestDriver{state: state})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

var contactUnlockPaymentTestDriverSeq atomic.Int64

type contactUnlockPaymentTestState struct {
	orderStatus   string
	unlockInserts atomic.Int64
}

type contactUnlockPaymentTestDriver struct {
	state *contactUnlockPaymentTestState
}

func (d contactUnlockPaymentTestDriver) Open(name string) (driver.Conn, error) {
	return &contactUnlockPaymentTestConn{state: d.state}, nil
}

type contactUnlockPaymentTestConn struct {
	state *contactUnlockPaymentTestState
}

func (c *contactUnlockPaymentTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, fmt.Errorf("Prepare is not supported in contact unlock payment test")
}

func (c *contactUnlockPaymentTestConn) Close() error {
	return nil
}

func (c *contactUnlockPaymentTestConn) Begin() (driver.Tx, error) {
	return contactUnlockPaymentTestTx{}, nil
}

func (c *contactUnlockPaymentTestConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return contactUnlockPaymentTestTx{}, nil
}

func (c *contactUnlockPaymentTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "FROM resource_contact_unlock_orders") && strings.Contains(query, "FOR UPDATE") {
		return &contactUnlockPaymentRows{
			columns: []string{"id", "resource_id", "buyer_user_id", "buyer_merchant_id", "status", "price_cent", "commercial_rules_snapshot"},
			values: []driver.Value{
				"order-1",
				"resource-1",
				"user-1",
				"merchant-2",
				c.state.orderStatus,
				int64(500),
				[]byte(`{"contactUnlock":{"mode":"paid_or_vip","priceCent":500,"currency":"CNY","repeatUnlockDays":30}}`),
			},
		}, nil
	}
	if strings.Contains(query, "INSERT INTO resource_contact_unlocks") {
		c.state.unlockInserts.Add(1)
		return &contactUnlockPaymentRows{
			columns: []string{"id", "resource_id", "user_id", "source_type", "expires_at"},
			values:  []driver.Value{"unlock-1", "resource-1", "user-1", ContactUnlockSourcePaid, time.Now().AddDate(0, 0, 30)},
		}, nil
	}
	return nil, fmt.Errorf("unexpected query: %s", query)
}

func (c *contactUnlockPaymentTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "UPDATE resource_contact_unlock_orders"):
		return driver.RowsAffected(1), nil
	default:
		return nil, fmt.Errorf("unexpected exec: %s", query)
	}
}

type contactUnlockPaymentTestTx struct{}

func (contactUnlockPaymentTestTx) Commit() error {
	return nil
}

func (contactUnlockPaymentTestTx) Rollback() error {
	return nil
}

type contactUnlockPaymentRows struct {
	columns []string
	values  []driver.Value
	read    bool
}

func (r *contactUnlockPaymentRows) Columns() []string {
	return r.columns
}

func (r *contactUnlockPaymentRows) Close() error {
	return nil
}

func (r *contactUnlockPaymentRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	copy(dest, r.values)
	r.read = true
	return nil
}
