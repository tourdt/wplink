# 依赖安全门禁 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 消除当前前端 high/critical 漏洞，并在 CI 中持续阻断依赖漏洞、扫描 Go 模块、每周自动提出依赖更新。

**Architecture:** 两个前端以 lockfile 为唯一安装输入，先有针对性升级再通过 `npm audit --audit-level=high` 验证；根目录新增 `check-dependencies` 作为本地和 CI 的唯一安全检查入口。Go 使用固定版本 `golang.org/x/vuln/cmd/govulncheck@v1.1.4`，Dependabot 仅负责生成更新 PR，仍由同一 CI 判定。

**Tech Stack:** npm lockfile v3、Uni App/Vite、Go 1.25.12、govulncheck v1.1.4、GNU Make、GitHub Actions、Dependabot。

## Global Constraints

- 不使用 `npm audit fix --force`，所有升级必须由 `package.json` 和 lockfile 可读地表达。
- 两个 `@dcloudio/uni-*` 包必须保持相同的已发布版本。
- npm 安全阈值为 `high`：high/critical 阻断，低/中风险不允许通过忽略名单静默屏蔽。
- npm 或 Go 漏洞服务不可用时命令必须失败，不得以 `|| true`、`continue-on-error` 或忽略名单绕过。
- 不修改业务功能代码、数据库结构、公共 API 或 `AGENTS.md`。

## 实施差异（2026-08-07）

- npm 默认 Uni App 标签属于另一条旧发行线，不能与现有 Alpha 工具链混用；经确认后保留当前 Uni 版本，通过 `wxapp/package.json` 的 `overrides` 修复高危传递依赖，并用 `wxapp/.npmrc` 的 `legacy-peer-deps=true` 处理其将 Vite 精确锁定为 `5.2.8` 的上游 peer 冲突。
- `govulncheck` 在首次接入时发现 7 个可达漏洞；修复 `grpc 1.82.1` 要求 Go 1.25，因此实际将后端与 CI 升级为 Go `1.25.12`，并升级 OpenTelemetry 至 `1.44.0`、JWT 至 `4.5.2`。该范围是安全门禁发现后的直接修复，已获确认。

---

### Task 1: 以可复现方式验证安全门禁配置

**Files:**

- Create: `backend/scripts/dependency_security_policy.test.mjs`
- Modify: `Makefile:1-24`
- Modify: `.github/workflows/ci.yml:1-81`
- Create: `.github/dependabot.yml`
- Test: `backend/scripts/dependency_security_policy.test.mjs`

**Interfaces:**

- Consumes: `make check-dependencies`、`.github/workflows/ci.yml`、`.github/dependabot.yml`。
- Produces: 包含两次 `npm audit --audit-level=high`、固定版 `govulncheck@v1.1.4`、CI 调用和三个 Dependabot 生态系统条目的策略测试。

- [ ] **Step 1: 写出失败的安全策略测试**

新建 Node 测试，读取根目录 Makefile、CI 与 Dependabot 文件，并断言：

```js
assert.match(makefile, /check-dependencies:/)
assert.match(makefile, /npm audit --prefix admin-web --audit-level=high/)
assert.match(makefile, /npm audit --prefix wxapp --audit-level=high/)
assert.match(makefile, /govulncheck@v1\.1\.4 \.\/\.\.\//)
assert.match(ci, /make check-dependencies/)
for (const expected of ['package-ecosystem: npm', 'directory: /admin-web', 'directory: /wxapp', 'package-ecosystem: gomod', 'directory: /backend']) {
  assert.match(dependabot, new RegExp(expected))
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && node --test scripts/dependency_security_policy.test.mjs`

Expected: FAIL，因为 Makefile、CI 和 Dependabot 配置尚未提供安全门禁。

- [ ] **Step 3: 实现统一安全门禁与自动更新配置**

将 `.PHONY` 扩展为包含 `check-dependencies`，并添加：

```make
check-dependencies:
	npm audit --prefix admin-web --audit-level=high
	npm audit --prefix wxapp --audit-level=high
	cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

在 CI 新增独立“依赖安全”步骤（前端 `npm ci` 完成后执行 `make check-dependencies`）；不要把漏洞扫描藏进构建步骤。新增 Dependabot v2 配置，为 `/admin-web`、`/wxapp` 的 npm 和 `/backend` 的 gomod 各配置每周一次更新、`open-pull-requests-limit: 5`。

- [ ] **Step 4: 运行策略测试确认通过**

Run: `cd backend && node --test scripts/dependency_security_policy.test.mjs`

Expected: PASS，策略测试确认本地、CI 和 Dependabot 均采用同一门禁。

- [ ] **Step 5: 提交安全门禁配置**

```bash
git add Makefile .github/workflows/ci.yml .github/dependabot.yml backend/scripts/dependency_security_policy.test.mjs
git commit -m "ci: 增加依赖安全门禁"
```

### Task 2: 修复管理后台的高危依赖漏洞

**Files:**

- Modify: `admin-web/package-lock.json`
- Test: `admin-web/package-lock.json` 的 npm audit 结果、`admin-web` 的 check 脚本

**Interfaces:**

- Consumes: `admin-web/package.json` 的现有版本范围和 npm 官方审计数据。
- Produces: 管理后台 `npm audit --audit-level=high` 成功，保留原有直接依赖 API。

- [ ] **Step 1: 记录修复前的失败结果**

Run: `npm audit --prefix admin-web --package-lock-only --audit-level=high`

Expected: FAIL，报告 `postcss` high 漏洞。

- [ ] **Step 2: 仅更新受影响的锁定传递依赖**

Run: `npm install --prefix admin-web --package-lock-only postcss@^8.5.24`

Expected: 仅变更 `admin-web/package-lock.json` 中 `postcss` 及 npm 为满足解析所需的最小关联条目；`admin-web/package.json` 不因传递依赖修复而新增直接依赖。

- [ ] **Step 3: 验证审计和构建检查**

Run: `npm audit --prefix admin-web --package-lock-only --audit-level=high && npm ci --prefix admin-web && npm --prefix admin-web run check`

Expected: PASS，审计无 high/critical，原有类型检查与构建通过。

- [ ] **Step 4: 提交管理后台锁文件修复**

```bash
git add admin-web/package-lock.json
git commit -m "fix: 修复管理后台依赖漏洞"
```

### Task 3: 升级小程序 Uni App 工具链并清除高危漏洞

**Files:**

- Modify: `wxapp/package.json`
- Modify: `wxapp/package-lock.json`
- Test: `wxapp` 的 npm audit 与 check 脚本

**Interfaces:**

- Consumes: npm Registry 已发布的 `@dcloudio/uni-app`、`@dcloudio/uni-mp-weixin`、`@dcloudio/vite-plugin-uni` 同版本发行物。
- Produces: 三个 Uni 包锁定到同一最新版稳定 `3.x` 发行物，Vite 升级到该插件声明的兼容版本，`npm audit --audit-level=high` 成功。

- [ ] **Step 1: 查询三件套可用版本与 peer 依赖**

Run: `npm view @dcloudio/uni-app version peerDependencies --json && npm view @dcloudio/uni-mp-weixin version peerDependencies --json && npm view @dcloudio/vite-plugin-uni version peerDependencies --json`

Expected: 三个包可选同一稳定 `3.x` 版本；记录 Vite 的兼容范围后再修改依赖。

- [ ] **Step 2: 记录升级前的失败基线**

Run: `npm audit --prefix wxapp --package-lock-only --audit-level=high`

Expected: FAIL，报告 Uni App 传递的 high 漏洞。

- [ ] **Step 3: 用同一版本升级 Uni 三件套及兼容 Vite**

先读取上一步返回的版本 `U` 与兼容 Vite 范围 `V`，再执行：

```bash
npm install --prefix wxapp --save-exact @dcloudio/uni-app@U @dcloudio/uni-mp-weixin@U
npm install --prefix wxapp --save-dev --save-exact @dcloudio/vite-plugin-uni@U vite@V
```

随后检查 `wxapp/package.json`：三条 `@dcloudio/*` 版本必须完全相同，禁止混用 Alpha 与稳定版。若当前构建配置报出公开 API 迁移错误，只修正被报错文件中的工具链调用，禁止顺带重构页面业务。

- [ ] **Step 4: 验证锁定安装、审计与小程序构建**

Run: `npm ci --prefix wxapp && npm audit --prefix wxapp --audit-level=high && npm --prefix wxapp run check`

Expected: PASS，无 high/critical 漏洞；类型检查和构建均通过。

- [ ] **Step 5: 提交小程序工具链升级**

```bash
git add wxapp/package.json wxapp/package-lock.json
git commit -m "fix: 升级小程序安全工具链"
```

### Task 4: 进行端到端门禁验证

**Files:**

- Verify: `Makefile`
- Verify: `.github/workflows/ci.yml`
- Verify: `.github/dependabot.yml`
- Verify: `backend/scripts/dependency_security_policy.test.mjs`

**Interfaces:**

- Consumes: 已修复的两个 lockfile 与安全门禁配置。
- Produces: 本地复现 CI 的依赖安全和受影响工程检查结果。

- [ ] **Step 1: 运行策略测试与统一门禁**

Run: `cd backend && node --test scripts/dependency_security_policy.test.mjs && cd .. && make check-dependencies`

Expected: PASS，npm high/critical 与 Go 漏洞扫描均通过。

- [ ] **Step 2: 运行受影响项目的全部验证**

Run: `make check-admin check-wxapp check-backend`

Expected: PASS，依赖升级未破坏两端构建、API 契约或 Go 测试。

- [ ] **Step 3: 检查改动范围并提交最终验证状态**

Run: `git diff --check && git status --short`

Expected: 除用户已有的 `AGENTS.md` 修改外，无意外文件；每项生产改动均已由对应提交覆盖。
