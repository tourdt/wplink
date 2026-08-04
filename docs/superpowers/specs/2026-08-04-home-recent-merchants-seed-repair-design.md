# 首页最近入驻商家种子数据修复设计

## 1. 背景与问题

小程序通过 `VITE_API_BASE_URL=http://127.0.0.1:4000` 访问本地 Go 服务时，首页最近入驻商家接口可以正常返回 HTTP 200，但 `items` 为空。普通商家目录能够返回相同测试数据库中的已认领商家，说明前端 API 地址、后端服务和基础商家数据均可用。

首页查询只展示同时满足以下条件的商家：

- 城市站点为 `zhili`；
- `status = 'active'`；
- `profile_status = 'completed'`；
- `deleted_at IS NULL`；
- `onboarded_at IS NOT NULL`；
- 存在 `status = 'active' AND role = 'owner'` 的商家管理员绑定。

当前测试数据库中的 6 家演示商家满足除 `onboarded_at IS NOT NULL` 之外的所有条件。根因是 `backend/scripts/seed_demo_data.sql` 创建和更新商家时没有写入 `onboarded_at`。`000032_merchant_onboarded_at` 只会在迁移执行当时回填已有商家；迁移完成后再次导入演示种子时，新数据仍会留下空值。

## 2. 目标

1. 修复当前测试数据库，使 6 家演示商家重新进入首页最近入驻列表。
2. 修复演示种子，确保新建数据库和重复导入场景都能保留有效入驻时间。
3. 为已经存在于其他环境中的同类空数据提供可部署、幂等的数据迁移。
4. 增加自动化回归验证，防止数据库重置或重复灌种子后再次出现首页空列表。

## 3. 非目标

- 不修改首页前端的空数据降级逻辑。
- 不修改最近入驻商家的筛选条件、排序和最多 6 家的规则。
- 不修改正常商家完成资料时记录首次入驻时间的业务流程。
- 不为 `onboarded_at` 增加数据库默认值或触发器，避免未完成资料的商家被提前标记为已入驻。

## 4. 方案选择

采用“种子修复 + 新数据迁移 + 自动化回归 + 测试库即时回填”的组合方案。

未采用以下方案：

- 只修改种子并直接更新测试库：无法修复其他已经存在的环境，也缺少可追踪的迁移记录。
- 为字段增加默认值或触发器：商家创建时间和正式完成资料时间并不等价，可能破坏首次入驻时间的业务语义。

## 5. 数据修复设计

新增下一顺序迁移 `000034_repair_missing_merchant_onboarded_at.up.sql`，只更新：

```sql
UPDATE merchants
SET onboarded_at = created_at
WHERE profile_status = 'completed'
  AND onboarded_at IS NULL;
```

该语句幂等，多次执行不会覆盖已有的首次入驻时间。历史缺失数据使用原 `created_at` 回填，避免旧商家因修复时间被错误排序为当天新入驻。

对应 down migration 不反向清空 `onboarded_at`。数据回填完成后无法可靠区分“迁移前已有时间”和“本次迁移补齐时间”；反向清空会破坏有效业务数据。down 文件保留明确中文注释，说明这是不可逆的数据修复迁移。

## 6. 演示种子设计

`backend/scripts/seed_demo_data.sql` 中的演示商家显式写入：

- `profile_status = 'completed'`；
- `onboarded_at = now()`。

发生固定 ID 冲突并进入 `ON CONFLICT DO UPDATE` 时：

```sql
onboarded_at = COALESCE(merchants.onboarded_at, EXCLUDED.onboarded_at)
```

该规则只补齐空值，不覆盖已记录的首次入驻时间，保证种子连续导入仍保持幂等和时间稳定。

## 7. 测试设计

### 7.1 静态迁移与种子测试

扩展 `backend/scripts/validate_migrations.test.mjs`，验证：

- `000034` up/down 文件存在；
- up migration 只回填已完成资料且入驻时间为空的商家；
- 回填值来自 `created_at`；
- 种子插入显式包含 `profile_status` 和 `onboarded_at`；
- 冲突更新使用 `COALESCE`，不会覆盖已有入驻时间。

该测试不依赖 PostgreSQL，作为所有开发和 CI 环境都会执行的基础防线。

### 7.2 PostgreSQL 集成测试

扩展现有演示种子临时数据库验证。在完整执行全部 up migrations、连续导入两次种子后，查询 6 个固定演示商家并断言：

- 6 家全部满足首页查询的城市、状态、资料、删除状态、入驻时间和 owner 绑定条件；
- 6 家的 `onboarded_at` 均非空；
- 第二次导入没有改变第一次记录的 `onboarded_at`。

当未配置 `DATABASE_URL` 时，沿用现有策略跳过真实 PostgreSQL 测试；静态测试仍然必须通过。

## 8. 当前测试数据库修复

代码和自动化测试全部通过后，在 `124.223.186.63` 测试数据库执行与 `000034` 相同的幂等回填 SQL，但不提前写入 `schema_migrations`。后续正式部署仍可安全执行并记录 `000034`。

修复后验证：

1. `GET /api/v1/home/recent-merchants?cityCode=zhili` 返回 HTTP 200；
2. `items` 数量为 6；
3. 返回顺序按 `onboarded_at DESC, id DESC`；
4. 普通商家目录和其他首页接口不受影响。

## 9. 风险与保护措施

- 回填范围严格限制为资料已完成且入驻时间为空的商家，不覆盖任何现有时间。
- 不使用当前修复时间回填，避免影响首页“最近”排序的真实性。
- 不直接修改 `schema_migrations`，避免即时修复与后续部署迁移状态不一致。
- 测试库写操作前后分别执行只读资格查询，记录受影响数量并核对接口结果。

## 10. 验收标准

- 静态迁移测试、后端 Go 测试和 PostgreSQL 集成测试通过。
- 演示种子连续执行两次后，首次入驻时间保持不变。
- 当前测试数据库的 6 家演示商家均具备非空 `onboarded_at`。
- 本地小程序所访问的最近入驻接口返回 6 家演示商家。
- 工作区只包含本设计直接相关的迁移、种子和测试改动。
