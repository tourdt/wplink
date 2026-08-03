# 供需详情展示标签去重实施计划

> **执行方式：** 按任务逐项执行时，必须使用 `superpowers:subagent-driven-development` 或 `superpowers:executing-plans`。

**目标：** 后端返回已过滤的 `presentation.tags`，详情页仅渲染该展示模型，避免类型/分类与标签重复。

**设计依据：** `docs/superpowers/specs/2026-08-03-resource-detail-presentation-tags-design.md`

## Task 1：后端生成展示标签

**涉及文件：**
- 修改：`backend/app/api/resource.api`
- 修改：`backend/app/internal/types/types.go`（通过 goctl 生成结果同步目标类型）
- 修改：`backend/app/internal/logic/resource/get_resource_logic.go`
- 修改或新增：`backend/app/internal/logic/resource/get_resource_logic_test.go`

1. 先为展示标签过滤函数编写 Go 测试，覆盖空值、重复标签、与 `typeName`/`category` 相同、被其包含，以及保留标签的原始顺序。
2. 先运行 `GOCACHE=/private/tmp/wplink-resource-tags-go-cache GOTMPDIR=/private/tmp/wplink-resource-tags-go-tmp go test ./app/internal/logic/resource`，确认测试处于 RED 状态。
3. 在 `ResourcePresentation` API 契约中增加 `tags`，用 `goctl` 生成临时结果，并仅同步目标生成类型，避免覆盖仓库内的无关手写类型。
4. 在资源详情展示模型构建处实现过滤：去除空标签与重复项；当标签等于 `typeName`/`category`，或被其完整包含时隐藏；其他标签按原始顺序保留。该纯映射逻辑不引入新的错误分支或外部调用。
5. 重新运行同一 Go 测试，确认 GREEN。

## Task 2：前端消费展示标签

**涉及文件：**
- 修改：`wxapp/api/resource.js`
- 修改：`wxapp/pages/resource/detail.vue`
- 修改：相关前端单元测试或静态契约测试

1. 先补充前端测试：详情 API 标准化 `presentation.tags`，详情页不得再从资源原始 `tags` 取值。
2. 运行对应 Node 测试，确认 RED 状态。
3. 在 API 标准化边界为 `presentation.tags` 提供数组默认值，不重复实现后端的业务过滤规则。
4. 详情页标签区只读取 `resource.presentation.tags`；保留现有“没有可展示标签时不渲染标签行”的结构。
5. 重新运行对应 Node 测试，确认 GREEN。

## Task 3：回归验证与交付

**涉及范围：** 本次改动文件及资源详情相关契约测试。

1. 运行后端全量验证：`GOCACHE=/private/tmp/wplink-resource-tags-all-go-cache GOTMPDIR=/private/tmp/wplink-resource-tags-all-go-tmp go test ./app/...`。
2. 在 `wxapp` 目录运行 `npm run check`，覆盖单元测试、页面/流程验证与小程序构建。
3. 检查差异，确认没有修改无关文件，且示例“工厂直供 / 童装 / 工厂直供、童装、现货、可混批”仅展示“现货、可混批”。
4. 提交本次实现；提交信息应明确为详情展示标签去重。

## 验收标准

1. 接口 `presentation.tags` 只含应展示的标签，保持业务标签的先后顺序。
2. 详情首行类型/分类后，不再重复出现相同或已被其包含的标签。
3. “现货”“可混批”等不重复的特征标签仍正常展示。
4. 无可展示标签时，详情摘要卡不保留空标签区域。
5. 后端测试与 `wxapp` 的 `npm run check` 全部通过。
