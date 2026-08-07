# 文档来源治理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将四份历史产品/领域文档明确归档，并用自动化校验防止其重新被当作现行规范。

**Architecture:** 历史文档保留原文以支持追溯，但统一在文首给出机器可校验的归档状态及当前事实来源。Node 测试脚本只负责验证文档来源边界，根目录 `Makefile` 与 CI 复用现有后端契约检查入口执行它。

**Tech Stack:** Markdown、Node.js 内置 `node:test` 与 `assert`、GNU Make、GitHub Actions。

## Global Constraints

- 不删除历史文档，不改写其历史规则正文。
- 现行接口、数据库和验收依据分别固定为 `backend/app/api/app.api`、`backend/migrations/`、`docs/product/mvp-acceptance-checklist.md`。
- 不修改业务逻辑、数据库迁移、公共 API 或 `AGENTS.md`。

---

### Task 1: 为历史文档建立可辨识的归档边界

**Files:**

- Modify: `docs/product/apparel-industry-platform-prd.md:1-15`
- Modify: `docs/product/domain-model-ddd.md:1-10`
- Modify: `docs/product/database-er-design.md:1-12`
- Modify: `docs/product/resource-management-rules.md:1-13`
- Modify: `docs/product/technical-architecture.md:1-24`
- Test: `backend/scripts/product_doc_governance.test.mjs`

**Interfaces:**

- Consumes: `ARCHIVED_PRODUCT_DOCS` 测试常量，包含四个文件名和统一状态文本。
- Produces: 每份归档文档均以 `> **文档状态：历史归档，不作为当前实现、验收、数据库迁移或 API 生成依据**` 开头，并包含四个当前事实来源路径。

- [ ] **Step 1: 写出失败的文档来源校验**

新建 `backend/scripts/product_doc_governance.test.mjs`，使用下列结构读取 `docs/product`：

```js
const ARCHIVED_PRODUCT_DOCS = [
  'apparel-industry-platform-prd.md',
  'domain-model-ddd.md',
  'database-er-design.md',
  'resource-management-rules.md',
]
const REQUIRED_STATUS = '文档状态：历史归档，不作为当前实现、验收、数据库迁移或 API 生成依据'
const CURRENT_SOURCES = [
  'backend/app/api/app.api',
  'backend/migrations/',
  'docs/product/technical-architecture.md',
  'docs/product/mvp-acceptance-checklist.md',
]
```

对每个文件断言包含 `REQUIRED_STATUS` 及 `CURRENT_SOURCES`；再断言 `technical-architecture.md` 包含“历史归档文档不得作为当前依据”。断言消息必须带文件名或缺失来源，便于 CI 定位。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && node --test scripts/product_doc_governance.test.mjs`

Expected: FAIL，指出至少一份历史文档缺少统一状态栏。

- [ ] **Step 3: 添加最小归档状态栏和事实来源**

在每份历史文档的标题、版本信息之后插入：

```markdown
> **文档状态：历史归档，不作为当前实现、验收、数据库迁移或 API 生成依据**
>
> 当前事实来源：`backend/app/api/app.api`、`backend/migrations/`、`docs/product/technical-architecture.md`、`docs/product/mvp-acceptance-checklist.md`。
```

在 `technical-architecture.md` 的“文档目标与事实来源”后增加“文档优先级”小节，说明运行时代码和迁移优先于当前实现说明，以上四份文档属于历史归档且不得作为当前依据。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend && node --test scripts/product_doc_governance.test.mjs`

Expected: PASS，全部归档文档均带状态和当前来源。

- [ ] **Step 5: 提交文档治理改动**

```bash
git add docs/product backend/scripts/product_doc_governance.test.mjs
git commit -m "docs: 建立历史文档来源治理"
```

### Task 2: 将来源校验接入本地与持续集成入口

**Files:**

- Modify: `Makefile:1-9`
- Modify: `.github/workflows/ci.yml:39-41`
- Test: `backend/scripts/product_doc_governance.test.mjs`

**Interfaces:**

- Consumes: `node --test scripts/product_doc_governance.test.mjs`。
- Produces: `make check-backend` 与 CI 的“校验迁移与 API 契约”步骤均执行文档来源校验。

- [ ] **Step 1: 先让门禁配置校验失败**

在 `product_doc_governance.test.mjs` 增加测试，读取根目录 `Makefile` 和 `.github/workflows/ci.yml`，断言二者均包含精确命令片段 `scripts/product_doc_governance.test.mjs`。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && node --test scripts/product_doc_governance.test.mjs`

Expected: FAIL，提示 Makefile 或 CI 未执行文档来源校验。

- [ ] **Step 3: 接入两个统一入口**

在 `check-backend` 的 Node 测试列表追加 `scripts/product_doc_governance.test.mjs`；在 CI 的“校验迁移与 API 契约”命令中同样追加该文件。不要增加独立且与本地行为不一致的 CI 命令。

- [ ] **Step 4: 验证来源校验和后端检查**

Run: `cd backend && node --test scripts/product_doc_governance.test.mjs && cd .. && make check-backend`

Expected: PASS，既验证归档边界，也不影响 API 生成和 Go 静态检查。

- [ ] **Step 5: 提交门禁接入改动**

```bash
git add Makefile .github/workflows/ci.yml backend/scripts/product_doc_governance.test.mjs
git commit -m "ci: 校验产品文档来源边界"
```
