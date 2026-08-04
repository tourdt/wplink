# 首页最近入驻商家种子数据修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 修复演示商家缺失 `onboarded_at` 导致首页最近入驻列表为空的问题，并为历史数据、重复种子导入和当前测试数据库建立完整修复链路。

**架构：** 使用 `000034` 幂等数据迁移回填已完成资料商家的历史空值；演示种子显式写入并通过 `COALESCE` 保留首次入驻时间。静态 Node.js 测试保证迁移和种子结构，Go/PostgreSQL 集成验证保证连续导入两次后 6 家演示商家仍满足真实首页 SQL 资格。

**技术栈：** PostgreSQL migration、SQL 演示种子、Node.js `node:test`、Go `database/sql`、go-zero 后端验证工具。

## 全局约束

- 所有实施计划、设计文档和业务注释使用中文，代码标识、文件路径和命令保留英文。
- 只修改迁移、演示种子和对应测试，不修改首页前端、接口契约或正式商家入驻流程。
- 历史空值使用原 `created_at` 回填，不使用当前修复时间。
- 只更新 `profile_status = 'completed' AND onboarded_at IS NULL` 的记录，不覆盖任何已有首次入驻时间。
- 不为 `onboarded_at` 增加数据库默认值或触发器。
- 当前测试数据库即时修复不提前写入 `schema_migrations`，后续正式部署仍执行并记录 `000034`。
- 数据库日志、命令输出和提交内容不得包含 DSN 密码、令牌或其他密钥。

---

### Task 1：新增历史入驻时间回填迁移

**文件：**

- 创建：`backend/migrations/000034_repair_missing_merchant_onboarded_at.up.sql`
- 创建：`backend/migrations/000034_repair_missing_merchant_onboarded_at.down.sql`
- 修改：`backend/scripts/validate_migrations.test.mjs`

**接口：**

- 输入：`merchants.profile_status`、`merchants.created_at`、`merchants.onboarded_at`
- 输出：幂等 migration `000034_repair_missing_merchant_onboarded_at`
- 后续依赖：任务 3 使用相同 SQL 修复当前测试数据库

- [ ] **步骤 1：先写缺失迁移的失败测试**

在 `backend/scripts/validate_migrations.test.mjs` 的 `merchant onboarding migration records first completed profile time for homepage exposure` 测试之后加入：

```js
test('merchant onboarding repair backfills completed profiles without destructive rollback', () => {
  const upFileName = '000034_repair_missing_merchant_onboarded_at.up.sql'
  const downFileName = '000034_repair_missing_merchant_onboarded_at.down.sql'
  assert.equal(fs.existsSync(path.resolve(migrationsDir, upFileName)), true, `${upFileName} should exist`)
  assert.equal(fs.existsSync(path.resolve(migrationsDir, downFileName)), true, `${downFileName} should exist`)

  const upSource = fs.readFileSync(path.resolve(migrationsDir, upFileName), 'utf8')
  const downSource = fs.readFileSync(path.resolve(migrationsDir, downFileName), 'utf8')
  assert.match(upSource, /UPDATE merchants[\s\S]*SET onboarded_at = created_at/)
  assert.match(upSource, /profile_status = 'completed'/)
  assert.match(upSource, /onboarded_at IS NULL/)
  assert.doesNotMatch(upSource, /onboarded_at = now\(\)/)
  assert.match(downSource, /不可逆/)
  assert.doesNotMatch(downSource, /SET onboarded_at = NULL/)
})
```

- [ ] **步骤 2：运行测试并确认因迁移文件缺失而失败**

运行：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
```

预期：FAIL，错误包含 `000034_repair_missing_merchant_onboarded_at.up.sql should exist`。

- [ ] **步骤 3：实现最小幂等 up migration**

创建 `backend/migrations/000034_repair_missing_merchant_onboarded_at.up.sql`：

```sql
-- 000032 只能回填迁移执行前已经存在的商家；后续导入的旧种子可能仍缺少正式入驻时间。
-- 使用创建时间恢复历史顺序，避免把旧商家误排成修复当天的新入驻商家。
UPDATE merchants
SET onboarded_at = created_at
WHERE profile_status = 'completed'
  AND onboarded_at IS NULL;
```

- [ ] **步骤 4：增加明确的非破坏性 down migration**

创建 `backend/migrations/000034_repair_missing_merchant_onboarded_at.down.sql`：

```sql
-- 该数据修复不可逆：无法可靠区分迁移前已有的入驻时间和本次回填值。
-- down migration 有意不清空 onboarded_at，避免破坏已经恢复的有效业务数据。
```

- [ ] **步骤 5：运行静态迁移测试并确认通过**

运行：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
```

预期：PASS，`current migrations pass static validation` 和新增 `000034` 测试均通过。

- [ ] **步骤 6：检查迁移差异并提交**

运行：

```bash
git diff --check
git diff -- backend/migrations/000034_repair_missing_merchant_onboarded_at.up.sql backend/migrations/000034_repair_missing_merchant_onboarded_at.down.sql backend/scripts/validate_migrations.test.mjs
git add backend/migrations/000034_repair_missing_merchant_onboarded_at.up.sql backend/migrations/000034_repair_missing_merchant_onboarded_at.down.sql backend/scripts/validate_migrations.test.mjs
git commit -m "fix: 回填缺失的商家入驻时间"
```

---

### Task 2：修复演示种子并验证连续导入资格

**文件：**

- 修改：`backend/scripts/seed_demo_data.sql:46-175`
- 修改：`backend/scripts/validate_migrations.test.mjs`
- 修改：`backend/scripts/verify_migrations.go:310-345`
- 测试：`backend/scripts/verify_migrations_test.go:136-150`

**接口：**

- 输入：6 个固定演示商家 ID、完整 up migrations、连续两次 `seed_demo_data.sql`
- 输出：每家商家显式 `profile_status = 'completed'`、非空且稳定的 `onboarded_at`
- 输出函数：`loadDemoHomeRecentMerchantOnboardedAt(ctx context.Context, db *sql.DB) (map[string]time.Time, error)`
- 后续依赖：任务 3 通过首页公开接口验证 6 家商家可见

- [ ] **步骤 1：先写种子结构的失败测试**

在 `backend/scripts/validate_migrations.test.mjs` 新增：

```js
test('demo merchants seed completed onboarding without overwriting first onboarding time', () => {
  const source = fs.readFileSync(path.resolve(scriptDir, 'seed_demo_data.sql'), 'utf8')
  const merchantSeed = source.match(/INSERT INTO merchants \([\s\S]*?ON CONFLICT \(id\) DO UPDATE SET[\s\S]*?updated_at = now\(\);/)?.[0] || ''

  assert.match(merchantSeed, /profile_status/)
  assert.match(merchantSeed, /onboarded_at/)
  assert.match(merchantSeed, /'completed'/)
  assert.match(
    merchantSeed,
    /onboarded_at = COALESCE\(merchants\.onboarded_at, EXCLUDED\.onboarded_at\)/,
  )
})
```

- [ ] **步骤 2：扩展真实 PostgreSQL 验证，记录第一次种子导入时间并比较第二次结果**

在 `backend/scripts/verify_migrations.go` 中增加固定演示商家资格查询函数：

```go
func loadDemoHomeRecentMerchantOnboardedAt(ctx context.Context, db *sql.DB) (map[string]time.Time, error) {
	rows, err := db.QueryContext(ctx, `
SELECT m.id::text, m.onboarded_at
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id
WHERE m.id IN (
  8020000000000000001,
  8020000000000000002,
  8020000000000000003,
  8020000000000000004,
  8020000000000000005,
  8020000000000000006
)
  AND cs.code = 'zhili'
  AND m.status = 'active'
  AND m.profile_status = 'completed'
  AND m.deleted_at IS NULL
  AND m.onboarded_at IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM merchant_admin_bindings mab
    WHERE mab.merchant_id = m.id
      AND mab.status = 'active'
      AND mab.role = 'owner'
  )
ORDER BY m.id
`)
	if err != nil {
		return nil, fmt.Errorf("查询演示商家首页曝光资格失败: %w", err)
	}
	defer rows.Close()

	items := make(map[string]time.Time, 6)
	for rows.Next() {
		var merchantID string
		var onboardedAt time.Time
		if err := rows.Scan(&merchantID, &onboardedAt); err != nil {
			return nil, fmt.Errorf("读取演示商家入驻时间失败: %w", err)
		}
		items[merchantID] = onboardedAt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历演示商家入驻时间失败: %w", err)
	}
	if len(items) != 6 {
		return nil, fmt.Errorf("满足首页曝光资格的演示商家数量=%d，期望=6", len(items))
	}
	return items, nil
}
```

在 `verifyDemoSeedImport` 执行文件的循环中，首次执行种子后保存结果，第二次执行后逐个比较：

```go
var firstSeedOnboardedAt map[string]time.Time
for _, file := range files {
	if err := executeSQLFile(ctx, tempDB, file); err != nil {
		return err
	}
	if filepath.Base(file) != "seed_demo_data.sql" {
		continue
	}

	currentOnboardedAt, err := loadDemoHomeRecentMerchantOnboardedAt(ctx, tempDB)
	if err != nil {
		return err
	}
	if firstSeedOnboardedAt == nil {
		firstSeedOnboardedAt = currentOnboardedAt
		continue
	}
	for merchantID, firstValue := range firstSeedOnboardedAt {
		currentValue, ok := currentOnboardedAt[merchantID]
		if !ok {
			return fmt.Errorf("第二次导入后演示商家不再满足首页曝光资格: merchantId=%s", merchantID)
		}
		if !currentValue.Equal(firstValue) {
			return fmt.Errorf(
				"重复导入覆盖了首次入驻时间: merchantId=%s first=%s current=%s",
				merchantID,
				firstValue.Format(time.RFC3339Nano),
				currentValue.Format(time.RFC3339Nano),
			)
		}
	}
}
```

- [ ] **步骤 3：运行测试并确认当前种子缺少入驻时间**

先运行始终可用的静态测试：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs
```

预期：FAIL，错误指向 `profile_status`、`onboarded_at` 或 `COALESCE` 断言。

在 shell 已导出与本地后端一致的 `DATABASE_URL` 时，再运行：

```bash
cd backend
go test ./scripts -run TestDemoSeedImportsTwiceIntoTemporaryDatabase -count=1 -v
```

预期：FAIL，错误包含 `满足首页曝光资格的演示商家数量=0，期望=6`。如果数据库账号没有创建临时数据库权限，保留静态测试的 RED 证据，并在任务 3 通过受控测试数据库执行完整集成验证。

- [ ] **步骤 4：修改演示商家插入字段和值**

在 `backend/scripts/seed_demo_data.sql` 的 `INSERT INTO merchants` 字段中，将尾部调整为：

```sql
  images,
  status,
  profile_status,
  onboarded_at,
  last_active_at
```

对应 `SELECT` 尾部调整为：

```sql
  m.images::jsonb,
  'active',
  'completed',
  now(),
  now()
```

- [ ] **步骤 5：修改冲突更新并保留首次入驻时间**

在同一段 `ON CONFLICT (id) DO UPDATE SET` 中，将状态相关字段调整为：

```sql
  images = EXCLUDED.images,
  status = EXCLUDED.status,
  profile_status = EXCLUDED.profile_status,
  onboarded_at = COALESCE(merchants.onboarded_at, EXCLUDED.onboarded_at),
  last_active_at = EXCLUDED.last_active_at,
  updated_at = now();
```

- [ ] **步骤 6：格式化 Go 验证代码并运行静态、集成测试**

运行：

```bash
cd backend
gofmt -w scripts/verify_migrations.go
node --test scripts/validate_migrations.test.mjs
go test ./scripts -run TestDemoSeedImportsTwiceIntoTemporaryDatabase -count=1 -v
```

预期：静态测试 PASS；配置了具备临时数据库权限的 `DATABASE_URL` 时，PostgreSQL 集成测试 PASS，且连续两次导入没有改变任一演示商家的 `onboarded_at`。

- [ ] **步骤 7：运行脚本包测试并提交**

运行：

```bash
cd backend
go test ./scripts -count=1
cd ..
git diff --check
git add backend/scripts/seed_demo_data.sql backend/scripts/validate_migrations.test.mjs backend/scripts/verify_migrations.go
git commit -m "fix: 保持演示商家入驻时间完整"
```

预期：`wplink/backend/scripts` 测试通过并产生一个仅包含种子和验证逻辑的提交。

---

### Task 3：全量验证并修复当前测试数据库

**文件：**

- 验证：`backend/migrations/000034_repair_missing_merchant_onboarded_at.up.sql`
- 验证：`backend/scripts/seed_demo_data.sql`
- 外部状态：`124.223.186.63` 测试数据库

**接口：**

- 输入：任务 1 的幂等回填 SQL、任务 2 的种子和验证逻辑
- 输出：测试数据库 6 家演示商家全部满足首页曝光资格
- 输出接口：`GET http://127.0.0.1:4000/api/v1/home/recent-merchants?cityCode=zhili`

- [ ] **步骤 1：运行仓库后端静态检查和全量测试**

运行：

```bash
cd backend
node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs
go test ./...
go vet ./...
```

预期：所有命令退出码为 0，无新增 warning 或 error。

- [ ] **步骤 2：在测试数据库写入前只读确认影响范围**

运行以下命令；它只输出状态统计，不输出 DSN：

```bash
ssh -i /Users/ldh/.ssh/ebyby.pem -o BatchMode=yes root@124.223.186.63 \
  'set -a; source /etc/wplink/wplink.env; set +a; psql "$DATABASE_URL" -P pager=off -c "SELECT profile_status, (onboarded_at IS NULL) AS missing_onboarded_at, count(*) FROM merchants WHERE deleted_at IS NULL GROUP BY profile_status, (onboarded_at IS NULL) ORDER BY profile_status, missing_onboarded_at;"'
```

预期：能看到 `profile_status=completed` 且 `missing_onboarded_at=true` 的受影响记录；不得继续到比设计范围更宽的更新条件。

- [ ] **步骤 3：在事务中执行幂等回填，但不写迁移版本表**

运行：

```bash
ssh -i /Users/ldh/.ssh/ebyby.pem -o BatchMode=yes root@124.223.186.63 \
  'set -a; source /etc/wplink/wplink.env; set +a; psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -P pager=off -c "BEGIN; UPDATE merchants SET onboarded_at = created_at WHERE profile_status = '\''completed'\'' AND onboarded_at IS NULL; COMMIT;"'
```

预期：`UPDATE` 只影响资料已完成且入驻时间为空的记录；命令不包含 `INSERT INTO schema_migrations`。

- [ ] **步骤 4：只读验证 6 家演示商家的完整资格**

运行：

```bash
ssh -i /Users/ldh/.ssh/ebyby.pem -o BatchMode=yes root@124.223.186.63 \
  'set -a; source /etc/wplink/wplink.env; set +a; psql "$DATABASE_URL" -P pager=off -c "SELECT count(*) AS eligible_demo_merchants FROM merchants m JOIN city_stations cs ON cs.id=m.city_station_id WHERE m.id IN (8020000000000000001,8020000000000000002,8020000000000000003,8020000000000000004,8020000000000000005,8020000000000000006) AND cs.code='\''zhili'\'' AND m.status='\''active'\'' AND m.profile_status='\''completed'\'' AND m.deleted_at IS NULL AND m.onboarded_at IS NOT NULL AND EXISTS (SELECT 1 FROM merchant_admin_bindings mab WHERE mab.merchant_id=m.id AND mab.status='\''active'\'' AND mab.role='\''owner'\'');"'
```

预期：`eligible_demo_merchants = 6`。

- [ ] **步骤 5：验证本地首页和普通商家目录接口**

在本地 Go 后端仍监听 `127.0.0.1:4000` 时运行：

```bash
node -e "Promise.all([fetch('http://127.0.0.1:4000/api/v1/home/recent-merchants?cityCode=zhili').then(r=>r.json()),fetch('http://127.0.0.1:4000/api/v1/map/merchant-places?cityCode=zhili&page=1&pageSize=20').then(r=>r.json())]).then(([home,map])=>{if(home.code!==200||home.data.items.length!==6)throw new Error('首页最近入驻商家验收失败');if(map.code!==200||map.data.total!==7)throw new Error('普通商家目录回归失败');console.log({recentMerchants:home.data.items.length,merchantPlaces:map.data.total})})"
```

预期输出：

```text
{ recentMerchants: 6, merchantPlaces: 7 }
```

- [ ] **步骤 6：检查最终工作区和提交历史**

运行：

```bash
git status --short
git log --oneline -4
```

预期：工作区干净；最近历史包含设计文档提交、任务 1 迁移提交和任务 2 种子修复提交。

## 最终验收

- [ ] `000034` up migration 幂等回填 `created_at`，down migration 不破坏数据。
- [ ] 演示种子显式完成资料并记录入驻时间，重复导入不覆盖首次值。
- [ ] 静态测试、Go 测试、Go vet 和真实 PostgreSQL 验证全部通过。
- [ ] 测试数据库 6 家演示商家全部具备首页资格。
- [ ] 首页最近入驻接口返回 6 家，普通商家目录仍返回 7 个点位。
