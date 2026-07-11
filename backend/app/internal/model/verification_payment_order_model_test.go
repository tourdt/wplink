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
)

func TestMarkVerificationPaymentPaidSkipsMessageForDuplicatePaidOrder(t *testing.T) {
	state := &verificationPaymentTestState{previousStatus: "paid"}
	db := openVerificationPaymentTestDB(t, state)
	defer db.Close()

	_, err := NewVerificationModel(db).MarkVerificationPaymentPaid(context.Background(), MarkVerificationPaymentPaidInput{
		OutTradeNo:    "VP202607110001",
		TransactionID: "wx-transaction-1",
		AmountTotal:   9900,
	})
	if err != nil {
		t.Fatalf("MarkVerificationPaymentPaid() error = %v", err)
	}
	if got := state.messageInserts.Load(); got != 0 {
		t.Fatalf("message inserts = %d, want 0 for duplicate paid callback", got)
	}
}

func TestMarkVerificationPaymentPaidCreatesMessageForPendingOrder(t *testing.T) {
	state := &verificationPaymentTestState{previousStatus: "pending"}
	db := openVerificationPaymentTestDB(t, state)
	defer db.Close()

	_, err := NewVerificationModel(db).MarkVerificationPaymentPaid(context.Background(), MarkVerificationPaymentPaidInput{
		OutTradeNo:    "VP202607110002",
		TransactionID: "wx-transaction-2",
		AmountTotal:   9900,
	})
	if err != nil {
		t.Fatalf("MarkVerificationPaymentPaid() error = %v", err)
	}
	if got := state.messageInserts.Load(); got != 1 {
		t.Fatalf("message inserts = %d, want 1 for first paid callback", got)
	}
}

func openVerificationPaymentTestDB(t *testing.T, state *verificationPaymentTestState) *sql.DB {
	t.Helper()
	driverName := fmt.Sprintf("verification-payment-test-%d", verificationPaymentTestDriverSeq.Add(1))
	sql.Register(driverName, verificationPaymentTestDriver{state: state})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

var verificationPaymentTestDriverSeq atomic.Int64

type verificationPaymentTestState struct {
	previousStatus string
	messageInserts atomic.Int64
}

type verificationPaymentTestDriver struct {
	state *verificationPaymentTestState
}

func (d verificationPaymentTestDriver) Open(name string) (driver.Conn, error) {
	return &verificationPaymentTestConn{state: d.state}, nil
}

type verificationPaymentTestConn struct {
	state *verificationPaymentTestState
}

func (c *verificationPaymentTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, fmt.Errorf("Prepare is not supported in verification payment test")
}

func (c *verificationPaymentTestConn) Close() error {
	return nil
}

func (c *verificationPaymentTestConn) Begin() (driver.Tx, error) {
	return verificationPaymentTestTx{}, nil
}

func (c *verificationPaymentTestConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return verificationPaymentTestTx{}, nil
}

func (c *verificationPaymentTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "UPDATE verification_payment_orders"):
		if strings.Contains(query, "previous_status") {
			return &verificationPaymentRows{
				columns: []string{"id", "verification_id", "merchant_id", "status", "previous_status"},
				values:  []driver.Value{"order-1", "verification-1", "merchant-1", "paid", c.state.previousStatus},
			}, nil
		}
		return &verificationPaymentRows{
			columns: []string{"id", "verification_id", "merchant_id", "status"},
			values:  []driver.Value{"order-1", "verification-1", "merchant-1", "paid"},
		}, nil
	case strings.Contains(query, "UPDATE verifications"):
		return &verificationPaymentRows{columns: []string{"verification_type"}, values: []driver.Value{"factory"}}, nil
	case strings.Contains(query, "SELECT EXISTS"):
		return &verificationPaymentRows{columns: []string{"exists"}, values: []driver.Value{true}}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (c *verificationPaymentTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if strings.Contains(query, "INSERT INTO messages") {
		c.state.messageInserts.Add(1)
		return driver.RowsAffected(1), nil
	}
	if strings.Contains(query, "UPDATE merchants") {
		return driver.RowsAffected(1), nil
	}
	return nil, fmt.Errorf("unexpected exec: %s", query)
}

type verificationPaymentTestTx struct{}

func (verificationPaymentTestTx) Commit() error {
	return nil
}

func (verificationPaymentTestTx) Rollback() error {
	return nil
}

type verificationPaymentRows struct {
	columns []string
	values  []driver.Value
	read    bool
}

func (r *verificationPaymentRows) Columns() []string {
	return r.columns
}

func (r *verificationPaymentRows) Close() error {
	return nil
}

func (r *verificationPaymentRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	copy(dest, r.values)
	r.read = true
	return nil
}
