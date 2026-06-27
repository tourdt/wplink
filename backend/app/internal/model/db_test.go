package model

import "testing"

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
