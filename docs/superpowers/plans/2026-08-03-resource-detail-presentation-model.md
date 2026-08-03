# 供需详情配置驱动渲染模型 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement task-by-task.

**Goal:** 后端返回可直接渲染的详情字段列表，前端移除摘要拼装和字段去重。

**Architecture:** 使用资源类型快照的 `displayTemplate.fields` 解析字段值、顺序和布局，生成 `presentation.fields`。详情页只渲染该数组；地址仍使用独立模型。

## Task 1: 定义并生成详情展示模型

- 修改 `backend/app/api/resource.api` 与 goctl 生成类型，删除旧详情摘要/属性字段，新增 `presentation.fields`。
- 修改 `get_resource_logic.go`：从快照 `displayTemplate.fields` 构建字段；过滤空值、排除地址、按 order 排序；添加中文日志与错误处理。
- 修改资源类型初始化/配置校验，使 fields 配置可编辑并校验 source、role、layout、order。
- 先添加 Go 失败测试，覆盖同源仅展示一次、空值过滤、排序与 full/half；运行 `go test ./app/internal/logic/resource` 得到 RED，再实现 GREEN。

## Task 2: 迁移小程序详情渲染

- 修改 `wxapp/api/resource.js` 的详情标准化，消费 `presentation.fields`。
- 修改 `wxapp/pages/resource/detail.vue`：删除 `buildResourceDetailSpecItems`、来源 key 去重和单双列推断；循环 `resource.presentation.fields`。
- 删除 `resourceDetailState` 中已废弃的详情字段拼装逻辑；更新 Node 回归测试。
- 先运行详情测试得到 RED；实现后运行 `npm run check`。

## Task 3: 清理与验证

- 删除 `SummarySourceKeys`、旧详情 `quantityText`、`priceText`、`attributeItems` 的接口/类型/测试引用。
- 运行 `go test ./app/internal/logic/resource`、`npm run check` 和 `git diff --check`。
- 提交重构。
