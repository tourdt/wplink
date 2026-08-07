# P1：文档来源治理与依赖安全门禁设计

## 目标与范围

本次仅解决两项 P1 维护风险：

1. 防止历史产品/领域文档被误认为当前系统规范；
2. 消除当前前端高危依赖漏洞，并建立持续阻断与自动更新机制。

不改动业务接口、数据库结构、任务调度或部署拓扑。

## 当前事实与约束

- `docs/product/apparel-industry-platform-prd.md`、`domain-model-ddd.md`、`database-er-design.md`、`resource-management-rules.md` 含已退役认证域描述，且缺少醒目的历史状态标识。
- 当前可执行事实来源分别是：`backend/app/api/app.api`（接口）、`backend/migrations/`（数据库）、`docs/product/technical-architecture.md`（系统架构）与 `docs/product/mvp-acceptance-checklist.md`（验收范围）。
- 2026-08-07 的 npm 审计显示：管理后台有 1 个高危漏洞；小程序有 39 个漏洞，其中 13 个高危。小程序根因是过期的 Uni App Alpha 工具链及其传递依赖。
- 项目尚未上线，但 npm Registry 的默认 Uni App 标签属于另一条旧发行线，不能与现有 Alpha 系列混用；因此保留当前 Uni 系列，以根级 `overrides` 升级高危传递依赖，并以 `npm ci`、构建和审计结果作为兼容性验收。

## 方案选择

采用完整治理方案，而非只新增告警：

- 历史文档保留可追溯性，但在标题后声明“历史归档、不得作为现行实现或验收依据”，并给出当前事实来源。
- 新增一条可执行的文档来源校验，检查全部归档文档均带有状态和事实来源，避免未来回退。
- 升级锁定依赖直至 npm 审计不再存在 high/critical 漏洞；低/中风险也尽量随工具链升级一并消除。若上游仍无修复版本，安全检查仍阻断 high/critical，并要求在后续变更中显式处理，不在本次引入静默豁免。
- CI 在每次推送和拉取请求中执行 npm 审计与 Go `govulncheck`；Dependabot 每周为两个 npm 项目和 Go 模块创建依赖更新请求。

## 组件与数据流

```text
产品历史文档 ──> 文档来源校验 ──> CI 失败/通过
当前事实来源 ──^                     |
                                    `---> 维护人员按唯一事实来源更新

package.json/package-lock.json ──> npm audit (high) ──> CI 失败/通过
backend/go.mod/go.sum ───────────> govulncheck ─────> CI 失败/通过
Dependabot ──────────────────────> 每周更新 PR ────> 同一套 CI 验证
```

## 实现边界

### 文档来源治理

- 修改四份历史文档的开头，只添加标准化状态栏和当前事实来源，不改写其历史内容。
- 在 `technical-architecture.md` 的维护章节列出“现行事实来源”和“历史归档文档”，明确优先级。
- 新建 Node 校验脚本并接入 `make check-backend`、CI 的契约检查步骤。校验失败应指出缺少状态栏或来源链接的具体文件。

### 依赖安全

- 管理后台使用锁文件的最小安全升级修复 `postcss`。
- 小程序将 `@dcloudio/uni-*` 组件保持同一已发布版本，使用 `overrides` 固定高危传递依赖并将 Vite 升至 `6.4.3`；`wxapp/.npmrc` 仅为上游精确 peer 冲突设置 `legacy-peer-deps=true`，不以 `npm audit fix --force` 盲目改写依赖树。
- 新增 `make check-dependencies`：分别对管理后台、小程序执行 `npm audit --audit-level=high`，对 `backend` 执行固定版本的 `govulncheck`。
- CI 调用该目标，确保本地和 CI 的安全判断一致；Go 漏洞工具版本固定，避免扫描器自身升级造成不可复现结果。
- Dependabot 配置为每周检查 `/admin-web`、`/wxapp` 与 `/backend`，限制未处理更新请求数量，避免噪声淹没维护者。
- Go 扫描发现可达漏洞后，后端升级至 Go `1.25.12`、`grpc 1.82.1`、OpenTelemetry `1.44.0` 与 JWT `4.5.2`；版本下限由策略测试锁定。

## 异常处理与维护规则

- npm 或 Go 漏洞数据库短暂不可用时，安全检查必须失败，不能把“无法审计”误判为“安全”。CI 日志保留原始命令输出以供重试和排障。
- 不引入 audit 忽略名单；任何 high/critical 漏洞都必须通过升级、替换依赖或另行审批的风险处置变更解决。
- 历史文档不会被删除；其状态栏是唯一允许其继续保存在主文档目录的前提。

## 验收标准

1. 四份历史文档均含统一的“历史归档”状态与当前事实来源；架构文档声明优先级。
2. 文档来源校验已接入 `make check-backend` 与 CI，删除任一状态栏时校验失败。
3. 两个前端目录的 `npm audit --audit-level=high` 均通过，且相关构建检查通过。
4. `govulncheck` 已固定版本并接入 CI；其策略配置有自动化校验。
5. `.github/dependabot.yml` 覆盖两个 npm 项目和 Go 模块。
6. 不修改 `AGENTS.md` 和任何业务功能代码。
