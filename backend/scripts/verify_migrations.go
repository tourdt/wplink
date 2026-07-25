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
	"sort"
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

	upFiles, err := collectMigrationFiles(rootDir, "up")
	if err != nil {
		return err
	}
	for _, upFile := range upFiles {
		downFile := strings.TrimSuffix(upFile, ".up.sql") + ".down.sql"
		if _, err := os.Stat(downFile); err != nil {
			return fmt.Errorf("%s 缺少配套 down migration: %w", upFile, err)
		}

		// 每个版本都执行一次 up -> down -> up，并比较回滚前后的结构快照。
		// 这能及时发现 down migration 误删前序版本表、字段或索引的问题，而不是只验证整库最终能否清空。
		before, err := loadPublicSchemaSnapshot(ctx, tempDB)
		if err != nil {
			return fmt.Errorf("%s 执行前读取结构失败: %w", upFile, err)
		}
		if err := executeSQLFile(ctx, tempDB, upFile); err != nil {
			return err
		}
		if err := executeSQLFile(ctx, tempDB, downFile); err != nil {
			return err
		}
		afterRollback, err := loadPublicSchemaSnapshot(ctx, tempDB)
		if err != nil {
			return fmt.Errorf("%s 回滚后读取结构失败: %w", downFile, err)
		}
		if diff := compareSchemaSnapshots(before, afterRollback); diff != "" {
			return fmt.Errorf("%s 单步回滚破坏了前序数据库结构:\n%s", downFile, diff)
		}
		if err := executeSQLFile(ctx, tempDB, upFile); err != nil {
			return fmt.Errorf("%s 回滚后重新执行失败: %w", upFile, err)
		}
	}

	// 单步验证通过后，再按逆序完整回滚，覆盖跨版本依赖和最终清库路径。
	downFiles, err := collectMigrationFiles(rootDir, "down")
	if err != nil {
		return err
	}
	for _, downFile := range downFiles {
		if err := executeSQLFile(ctx, tempDB, downFile); err != nil {
			return err
		}
	}
	return nil
}

func loadPublicSchemaSnapshot(ctx context.Context, db *sql.DB) ([]string, error) {
	const query = `
SELECT object_definition
FROM (
	SELECT
		'column|' || table_name || '|' || column_name || '|' ||
		data_type || '|' || udt_name || '|' || is_nullable || '|' ||
		COALESCE(column_default, '') AS object_definition
	FROM information_schema.columns
	WHERE table_schema = 'public'

	UNION ALL

	SELECT
		'index|' || tablename || '|' || indexname || '|' || indexdef
	FROM pg_indexes
	WHERE schemaname = 'public'

	UNION ALL

	SELECT
		'constraint|' || c.relname || '|' || con.conname || '|' ||
		pg_get_constraintdef(con.oid, true)
	FROM pg_constraint con
	JOIN pg_class c ON c.oid = con.conrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace
	WHERE n.nspname = 'public'

	UNION ALL

	SELECT
		'type|' || t.typname || '|' || e.enumlabel || '|' || e.enumsortorder::text
	FROM pg_type t
	JOIN pg_namespace n ON n.oid = t.typnamespace
	JOIN pg_enum e ON e.enumtypid = t.oid
	WHERE n.nspname = 'public'
) snapshot
ORDER BY object_definition`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshot []string
	for rows.Next() {
		var definition string
		if err := rows.Scan(&definition); err != nil {
			return nil, err
		}
		snapshot = append(snapshot, definition)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func compareSchemaSnapshots(want []string, got []string) string {
	if len(want) == len(got) {
		equal := true
		for index := range want {
			if want[index] != got[index] {
				equal = false
				break
			}
		}
		if equal {
			return ""
		}
	}

	wantSet := make(map[string]struct{}, len(want))
	gotSet := make(map[string]struct{}, len(got))
	for _, item := range want {
		wantSet[item] = struct{}{}
	}
	for _, item := range got {
		gotSet[item] = struct{}{}
	}

	var missing []string
	var unexpected []string
	for _, item := range want {
		if _, ok := gotSet[item]; !ok {
			missing = append(missing, "- 缺失: "+item)
		}
	}
	for _, item := range got {
		if _, ok := wantSet[item]; !ok {
			unexpected = append(unexpected, "+ 多出: "+item)
		}
	}
	return strings.Join(append(missing, unexpected...), "\n")
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

	files, err := collectMigrationFiles(rootDir, "up")
	if err != nil {
		return err
	}
	files = append(files, filepath.Join(rootDir, "scripts/seed_demo_data.sql"))
	for _, file := range files {
		if err := executeSQLFile(ctx, tempDB, file); err != nil {
			return err
		}
	}
	return nil
}

func collectMigrationFiles(rootDir string, direction string) ([]string, error) {
	if direction != "up" && direction != "down" {
		return nil, fmt.Errorf("migration direction must be up or down, got %q", direction)
	}
	pattern := filepath.Join(rootDir, "migrations", "*."+direction+".sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("未找到 migration 文件: %s", pattern)
	}
	sort.Strings(files)
	if direction == "down" {
		for left, right := 0, len(files)-1; left < right; left, right = left+1, right-1 {
			files[left], files[right] = files[right], files[left]
		}
	}
	return files, nil
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
