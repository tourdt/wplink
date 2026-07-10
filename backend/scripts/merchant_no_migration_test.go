package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMerchantNoMigrationDefinesPublicMerchantNumberRule(t *testing.T) {
	content, err := os.ReadFile(filepath.Clean("../migrations/000021_merchant_no.up.sql"))
	if err != nil {
		t.Fatalf("read merchant no migration: %v", err)
	}
	sql := strings.ToLower(string(content))

	for _, snippet := range []string{
		"create sequence if not exists merchant_no_seq",
		"start with 100000",
		"maxvalue 999999",
		"create or replace function merchant_no_luhn_check_digit",
		"create or replace function next_merchant_no()",
		"right(to_char(clock_timestamp(), 'yyyy'), 1)",
		"lpad(nextval('merchant_no_seq')::text, 6, '0')",
		"return 'm' || payload || merchant_no_luhn_check_digit(payload)::text",
		"merchant_no varchar(9)",
		"merchant_no ~ '^m[0-9]{8}$'",
		"unique index if not exists uniq_merchants_merchant_no",
	} {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("merchant no migration should contain %q", snippet)
		}
	}
}
