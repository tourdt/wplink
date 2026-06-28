package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationsUseTSIDBigintIDs(t *testing.T) {
	files := []string{
		"../migrations/000001_admin_auth.up.sql",
		"../migrations/000002_core_domain.up.sql",
	}

	var combined strings.Builder
	for _, file := range files {
		content, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		combined.Write(content)
		combined.WriteByte('\n')
	}

	sql := strings.ToLower(combined.String())
	disallowed := []string{
		" uuid",
		"::uuid",
		"gen_random_uuid",
	}
	for _, token := range disallowed {
		if strings.Contains(sql, token) {
			t.Fatalf("migration should not contain %q after TSID migration", token)
		}
	}
	if !strings.Contains(sql, "create or replace function next_tsid()") {
		t.Fatal("migration should define next_tsid() for database-side TSID defaults")
	}
	if !strings.Contains(sql, "id bigint primary key default next_tsid()") {
		t.Fatal("migration should use bigint primary keys with next_tsid() defaults")
	}
}
