package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDSNFromConfigExpandsEnvironment(t *testing.T) {
	t.Setenv("POSTGRES_PASSWORD", "secret-pass")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "app.yaml")
	if err := os.WriteFile(configPath, []byte(`Name: wplink-api
Postgres:
  DSN: "postgres://wplink_app:${POSTGRES_PASSWORD}@127.0.0.1:5432/wplink?sslmode=disable"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	dsn, err := loadDSNFromConfig(configPath)
	if err != nil {
		t.Fatalf("loadDSNFromConfig() error = %v", err)
	}

	if dsn != "postgres://wplink_app:secret-pass@127.0.0.1:5432/wplink?sslmode=disable" {
		t.Fatalf("dsn = %q, want expanded password", dsn)
	}
}

func TestRedactDSNHidesPassword(t *testing.T) {
	redacted := redactDSN("postgres://wplink_app:secret-pass@127.0.0.1:5432/wplink?sslmode=disable")

	if strings.Contains(redacted, "secret-pass") {
		t.Fatalf("redacted dsn leaked password: %s", redacted)
	}
	if redacted != "postgres://wplink_app:xxxxx@127.0.0.1:5432/wplink?sslmode=disable" {
		t.Fatalf("redacted = %q, want masked password", redacted)
	}
}

func TestDatabaseDSNReplacesDatabaseName(t *testing.T) {
	dsn, err := databaseDSN("postgres://wplink_app:secret@127.0.0.1:5432/wplink?sslmode=disable", "postgres")
	if err != nil {
		t.Fatalf("databaseDSN() error = %v", err)
	}

	if dsn != "postgres://wplink_app:secret@127.0.0.1:5432/postgres?sslmode=disable" {
		t.Fatalf("dsn = %q, want postgres maintenance database", dsn)
	}
}

func TestTempDatabaseNameUsesSafeIdentifier(t *testing.T) {
	name := tempDatabaseName()

	if !strings.HasPrefix(name, "wplink_migration_verify_") {
		t.Fatalf("name = %q, want migration verification prefix", name)
	}
	for _, char := range name {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			t.Fatalf("name = %q contains unsafe character %q", name, char)
		}
	}
}

func TestCollectMigrationFilesUsesAllFilesInOrder(t *testing.T) {
	rootDir := t.TempDir()
	migrationsDir := filepath.Join(rootDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	for _, name := range []string{
		"000001_init.up.sql",
		"000001_init.down.sql",
		"000002_extra.up.sql",
		"000002_extra.down.sql",
		"000003_future.up.sql",
		"000003_future.down.sql",
	} {
		if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte("-- "+name), 0o600); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}

	upFiles, err := collectMigrationFiles(rootDir, "up")
	if err != nil {
		t.Fatalf("collectMigrationFiles(up) error = %v", err)
	}
	downFiles, err := collectMigrationFiles(rootDir, "down")
	if err != nil {
		t.Fatalf("collectMigrationFiles(down) error = %v", err)
	}

	if got, want := baseNames(upFiles), []string{"000001_init.up.sql", "000002_extra.up.sql", "000003_future.up.sql"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("up files = %#v, want %#v", got, want)
	}
	if got, want := baseNames(downFiles), []string{"000003_future.down.sql", "000002_extra.down.sql", "000001_init.down.sql"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("down files = %#v, want %#v", got, want)
	}
}

func baseNames(paths []string) []string {
	names := make([]string, 0, len(paths))
	for _, item := range paths {
		names = append(names, filepath.Base(item))
	}
	return names
}
