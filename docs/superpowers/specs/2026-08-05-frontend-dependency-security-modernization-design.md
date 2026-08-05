# 前端依赖安全与工具链现代化设计

**日期：** 2026-08-05  
**状态：** 已确认，待实施  
**范围：** `admin-web`、`wxapp`、前端 CI 门禁

## 1. 背景与目标

项目尚未上线，不需要保留旧前端依赖组合的兼容性。当前依赖审计显示：

- 管理后台存在 1 个 high 漏洞，来源为构建链中的 `postcss`。
- 小程序存在 39 个漏洞，其中 13 个 high、12 个 moderate、14 个 low，主要来自旧版 Uni-app 构建链、Vite、WebSocket、压缩包和样式处理依赖。
- 小程序使用 `3.0.0-alpha-5010420260626001` DCloud Vue 3 版本线和 Vite `5.2.8`，构建会输出 Sass `@import` 与 legacy JS API 弃用警告。
- 当前 CI 只执行测试和构建，没有前端依赖漏洞门禁。

本次目标是在不改变业务接口、页面功能和后端实现的前提下，升级前端工具链，消除 high/critical 漏洞，移除当前 Sass 弃用警告，并建立可持续的 CI 安全门禁。

## 2. 方案选择

采用“当前架构内彻底升级”方案：

1. 保留管理后台的 Vue/Vite 架构和小程序的 Uni-app/Vue 3 架构，不重写页面。
2. 管理后台升级到实施时已确认可安装、可构建的当前主版本。
3. 小程序的 `@dcloudio/uni-app`、`@dcloudio/uni-mp-weixin`、`@dcloudio/vite-plugin-uni` 使用同一个最新 Vue 3 发布号，首选 `3.0.0-alpha-5020320260803001`，禁止混用不同发布号。
4. DCloud 仍锁定但存在漏洞的构建依赖，通过最小、逐项验证的 npm `overrides` 或直接依赖升级修复；不使用无关的大范围强制覆盖。
5. 不为了维持旧版构建结果保留已废弃 API。升级导致的必要构建配置调整应直接完成，但不得借机修改业务行为。

未采用的方案：

- 只升级 DCloud 官方精确依赖组合：会保留已确认的 Vite、WS 等漏洞，无法实现目标。
- 替换 Uni-app 或重写小程序：范围远超依赖治理，会引入不必要的业务回归风险。

## 3. 依赖升级边界

### 3.1 管理后台

升级范围包括：

- Vite 与 `@vitejs/plugin-vue`
- Vue、Vue Router、Pinia
- Axios、Element Plus 与图标包
- 构建链中的 PostCSS 等传递依赖

实施时使用 lockfile 固定最终解析版本。若主版本升级引发 API 变化，只修改直接受影响的路由、状态管理或构建配置，不进行页面重构。

### 3.2 微信小程序

升级范围包括：

- 三个 DCloud 核心包统一到同一 Vue 3 发布号
- Vue、Sass 与 DCloud 允许的 Vite 安全版本
- `ws`、`postcss`、`adm-zip`、`@babel/core` 等被审计命中的传递依赖

`overrides` 必须满足三个条件：

1. 对应当前审计中的真实漏洞路径。
2. 安装后不存在 npm peer dependency 错误。
3. 小程序测试、流程校验和 `build:mp-weixin` 全部通过。

若某个 override 无法满足上述条件，应回退该项并记录剩余漏洞、影响范围与上游限制；但最终 high/critical 不允许保留。

## 4. Sass 现代化

将 `wxapp/App.vue` 中的 `@import './uni.scss'` 改为 Sass 模块加载方式，并使用 `as *` 保留现有变量和 mixin 的消费方式，避免扩大样式改造范围。

在 Vite 样式配置中启用 Sass modern compiler API。验收时构建输出不得再出现：

- `legacy-js-api` 弃用警告
- Sass `@import rules are deprecated` 弃用警告

不通过静默忽略 `silenceDeprecations` 达成验收；必须实际切换 API 和语法。

## 5. 安全门禁

两个前端工程分别提供安全审计命令，统一使用：

```bash
npm audit --audit-level=high
```

CI 在 `npm ci` 后、构建前执行审计。出现 high 或 critical 时直接失败；moderate/low 可以暂时存在，但必须在实施结果中列出数量和来源，并确认不属于直接生产运行依赖中的可利用问题。

审计命令依赖 npm 官方漏洞数据，网络或注册表故障应表现为 CI 失败，不能误判为无漏洞。

## 6. 验证策略

依赖与 lockfile 属于配置/生成结果，本次不增加锁定固定版本文本的脆弱源码测试。使用真实命令完成红绿验证：

1. 升级前保存两个工程 `npm audit --json` 的失败基线。
2. 升级后两个工程执行 `npm ci`，证明 lockfile 可复现。
3. 两个工程执行 `npm audit --audit-level=high`，必须退出 0。
4. 管理后台执行 `npm run check`，全部测试和生产构建通过。
5. 小程序执行 `npm run check`，页面校验、流程校验、Node 测试和微信小程序构建全部通过。
6. 检查小程序构建输出，不得出现本设计第 4 节列出的 Sass 弃用警告。
7. 根目录执行 `make check`，后端、管理后台和小程序完整门禁通过。
8. 最终重新执行 `npm audit --json`，记录 high、critical、moderate、low 的实际数量。

## 7. 错误处理与回退

- `npm install` 或 `npm ci` 出现 peer dependency 错误时，不使用 `--force` 或 `--legacy-peer-deps` 掩盖问题，应调整直接依赖或 override。
- 某个主版本升级导致业务测试失败时，只修复该依赖公开 API 的必要适配；无法在本次范围内可靠适配时，选择仍受支持且满足 high/critical 为 0 的最近版本。
- 若 DCloud 最新 Vue 3 版本自身无法在 Node 22 环境安装或构建，保留完整错误证据，选择同一 Vue 3 通道中最近的可构建安全版本，不降级到 Vue 2。
- 不修改后端接口、数据库、业务状态机或页面业务规则作为依赖升级的回退手段。

## 8. 完成标准

- 管理后台与小程序的 `npm audit --audit-level=high` 均退出 0。
- 两个工程 high、critical 数量均为 0。
- 管理后台和小程序所有现有自动测试通过，生产构建成功。
- 小程序构建不再输出当前 Sass 弃用警告。
- CI 包含两个前端工程的 high/critical 安全审计门禁。
- 根目录 `make check` 退出 0。
- 改动仅涉及依赖声明、lockfile、前端构建/样式入口、CI 和本次设计/计划文档。

