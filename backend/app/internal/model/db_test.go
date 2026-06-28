package model

import (
	"testing"
	"time"
)

func TestOpenPostgresRegistersPostgresDriver(t *testing.T) {
	db, err := OpenPostgres("postgres://user:pass@127.0.0.1:5432/db?sslmode=disable")
	if err != nil {
		t.Fatalf("OpenPostgres() error = %v", err)
	}
	defer db.Close()

	if db.Driver() == nil {
		t.Fatal("postgres driver is not registered")
	}
}

func TestOpenPostgresAppliesPoolOptions(t *testing.T) {
	db, err := OpenPostgres("postgres://user:pass@127.0.0.1:5432/db?sslmode=disable", PostgresOptions{
		MaxOpenConns:    12,
		MaxIdleConns:    4,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("OpenPostgres() error = %v", err)
	}
	defer db.Close()

	if db.Stats().MaxOpenConnections != 12 {
		t.Fatalf("max open connections = %d, want 12", db.Stats().MaxOpenConnections)
	}
}
