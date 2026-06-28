package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
)

func main() {
	configPath := flag.String("config", "etc/app.yaml", "后端配置文件路径")
	dsnFlag := flag.String("dsn", "", "PostgreSQL DSN，优先级高于 DATABASE_URL 和配置文件")
	keepDatabase := flag.Bool("keep", false, "失败后保留临时数据库，便于人工排查")
	flag.Parse()

	dsn, err := resolveDSN(*dsnFlag, *configPath)
	if err != nil {
		fatalf("读取 PostgreSQL DSN 失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := verifyMigrations(ctx, dsn, ".", *keepDatabase); err != nil {
		fatalf("migration up/down 验证失败: %v", err)
	}

	fmt.Printf("migration up/down check ok: %s\n", redactDSN(dsn))
}

func resolveDSN(dsnFlag string, configPath string) (string, error) {
	if strings.TrimSpace(dsnFlag) != "" {
		return strings.TrimSpace(dsnFlag), nil
	}
	if envDSN := strings.TrimSpace(os.Getenv("DATABASE_URL")); envDSN != "" {
		return envDSN, nil
	}
	return loadDSNFromConfig(configPath)
}

func loadDSNFromConfig(configPath string) (string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var section string
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if value == "" && len(raw) == len(strings.TrimLeft(raw, " \t")) {
			section = key
			continue
		}
		if section == "Postgres" && key == "DSN" {
			return os.ExpandEnv(strings.Trim(value, `"`)), nil
		}
	}
	return "", errors.New("Postgres.DSN 未配置")
}

func verifyMigrations(ctx context.Context, sourceDSN string, rootDir string, keepDatabase bool) error {
	if err := verifyMigrationUpDown(ctx, sourceDSN, rootDir, keepDatabase); err != nil {
		return err
	}
	if err := verifyDemoSeedImport(ctx, sourceDSN, rootDir, keepDatabase); err != nil {
		return err
	}
	return nil
}

func verifyMigrationUpDown(ctx context.Context, sourceDSN string, rootDir string, keepDatabase bool) error {
	tempName := tempDatabaseName()
	adminDSN, err := databaseDSN(sourceDSN, "postgres")
	if err != nil {
		return err
	}
	tempDSN, err := databaseDSN(sourceDSN, tempName)
	if err != nil {
		return err
	}

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()
	if err := adminDB.PingContext(ctx); err != nil {
		return fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	quotedTempName := pq.QuoteIdentifier(tempName)
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+quotedTempName); err != nil {
		return fmt.Errorf("创建临时数据库失败: %w", err)
	}
	created := true
	defer func() {
		if !created || keepDatabase {
			return
		}
		dropCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, _ = adminDB.ExecContext(dropCtx, "DROP DATABASE IF EXISTS "+quotedTempName+" WITH (FORCE)")
	}()

	tempDB, err := sql.Open("postgres", tempDSN)
	if err != nil {
		return err
	}
	defer tempDB.Close()
	if err := tempDB.PingContext(ctx); err != nil {
		return fmt.Errorf("连接临时数据库失败: %w", err)
	}

	files := []string{
		"migrations/000001_admin_auth.up.sql",
		"migrations/000002_core_domain.up.sql",
		"migrations/000003_seed_zhili.up.sql",
		"migrations/000003_seed_zhili.down.sql",
		"migrations/000002_core_domain.down.sql",
		"migrations/000001_admin_auth.down.sql",
	}
	for _, file := range files {
		if err := executeSQLFile(ctx, tempDB, filepath.Join(rootDir, file)); err != nil {
			return err
		}
	}
	return nil
}

func verifyDemoSeedImport(ctx context.Context, sourceDSN string, rootDir string, keepDatabase bool) error {
	tempName := tempDatabaseName()
	adminDSN, err := databaseDSN(sourceDSN, "postgres")
	if err != nil {
		return err
	}
	tempDSN, err := databaseDSN(sourceDSN, tempName)
	if err != nil {
		return err
	}

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()
	if err := adminDB.PingContext(ctx); err != nil {
		return fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	quotedTempName := pq.QuoteIdentifier(tempName)
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+quotedTempName); err != nil {
		return fmt.Errorf("创建临时数据库失败: %w", err)
	}
	created := true
	defer func() {
		if !created || keepDatabase {
			return
		}
		dropCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, _ = adminDB.ExecContext(dropCtx, "DROP DATABASE IF EXISTS "+quotedTempName+" WITH (FORCE)")
	}()

	tempDB, err := sql.Open("postgres", tempDSN)
	if err != nil {
		return err
	}
	defer tempDB.Close()
	if err := tempDB.PingContext(ctx); err != nil {
		return fmt.Errorf("连接临时数据库失败: %w", err)
	}

	files := []string{
		"migrations/000001_admin_auth.up.sql",
		"migrations/000002_core_domain.up.sql",
		"migrations/000003_seed_zhili.up.sql",
		"scripts/seed_demo_data.sql",
	}
	for _, file := range files {
		if err := executeSQLFile(ctx, tempDB, filepath.Join(rootDir, file)); err != nil {
			return err
		}
	}
	return nil
}

func executeSQLFile(ctx context.Context, db *sql.DB, filePath string) error {
	sqlText, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, string(sqlText)); err != nil {
		return fmt.Errorf("%s 执行失败: %w", filePath, err)
	}
	return nil
}

func databaseDSN(rawDSN string, database string) (string, error) {
	parsed, err := url.Parse(rawDSN)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", fmt.Errorf("仅支持 postgres URL DSN，当前 scheme=%q", parsed.Scheme)
	}
	parsed.Path = "/" + database
	return parsed.String(), nil
}

func redactDSN(rawDSN string) string {
	parsed, err := url.Parse(rawDSN)
	if err != nil || parsed.User == nil {
		return rawDSN
	}
	username := parsed.User.Username()
	if _, hasPassword := parsed.User.Password(); hasPassword {
		parsed.User = url.UserPassword(username, "xxxxx")
	}
	return parsed.String()
}

func tempDatabaseName() string {
	return fmt.Sprintf("wplink_migration_verify_%d", time.Now().UnixNano())
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
