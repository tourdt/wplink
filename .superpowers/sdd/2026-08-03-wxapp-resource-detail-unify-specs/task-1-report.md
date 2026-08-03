# Task 1 实施报告：供需详情参数合并

## 实施结果

- `buildResourceDetailPresentation(resource)` 现仅返回 `isDemand`、`noun`、`typeName`。
- 新增并导出 `buildResourceDetailSpecItems(resource, attributeItems)`：数量/面积、供需对应的报价标签、自定义属性按顺序交由既有 `buildDetailSpecItems` 处理，因此继续复用空值过滤与长文本整行规则。
- 详情摘要卡移除标题和关键事实，仅保留供需标识、类型和发布标签；详细参数改用新的统一构建函数。
- 补充说明、图片、地址、商家、推荐、联系、收藏、管理、发布以及接口/数据库相关代码均未改动。

## TDD 证据

### RED

先更新 `wxapp/common/resourceDetailState.test.mjs`：需求样例 `5000 件`、`面议`、`交期要求：20 天内完成` 断言统一参数顺序；同时更新 `wxapp/pages/resource/detail.test.mjs` 断言摘要标题/事实及其引用已移除、参数计算改为新函数。

运行：

```bash
cd wxapp && node --test common/resourceDetailState.test.mjs pages/resource/detail.test.mjs
```

结果：失败（3 项）。其中新函数尚未导出，且详情页仍含 `summary-title`/`summary-facts` 并仍调用 `buildDetailSpecItems(attributeSpecItems.value)`，符合预期 RED 原因。

### GREEN

完成最小实现后，重新运行相同聚焦测试。

结果：`27` 项通过，`0` 项失败。

## 全量检查

运行：

```bash
cd wxapp && npm run check
```

结果：退出码 `0`；页面校验、流程校验、全量测试与微信小程序构建均完成。

## 改动文件

- `wxapp/common/resourceDetailState.js`
- `wxapp/common/resourceDetailState.test.mjs`
- `wxapp/pages/resource/detail.vue`
- `wxapp/pages/resource/detail.test.mjs`
- `.superpowers/sdd/2026-08-03-wxapp-resource-detail-unify-specs/task-1-report.md`

## 自审

- 新函数固定先传入数量/面积、再传入按供需方向区分的报价、最后展开属性项，顺序符合需求。
- 空值与 `fullWidth` 判定均由已有 `buildDetailSpecItems` 负责，未复制或改变规则。
- 详情页不再引用 `detailPresentation.headline` 或 `.facts`，相关模板和样式均已删除。
- 工作区差异仅覆盖本任务指定的四个代码/测试文件及本报告；`git diff --check` 无输出。

## 后续校验修复记录

### 根因与 RED

最终全量校验发现 `wxapp/scripts/validate-flows.test.mjs` 仍断言已移除的 `detailPresentation.facts`。在当前实现上运行：

```bash
node --test wxapp/scripts/validate-flows.test.mjs
```

结果：68 项中 67 项通过、1 项失败；失败断言为 `/detailPresentation\.facts/`，说明流程校验的旧契约未同步，不是实现缺失。

### 修复与 GREEN

仅修改该流程校验：断言详情页使用 `buildResourceDetailSpecItems(resource.value, attributeSpecItems.value)`，且不再含自动标题、事实卡片及 `detailPresentation.headline`/`.facts`。

重新运行 `node --test wxapp/scripts/validate-flows.test.mjs`：68/68 通过。

随后从 `wxapp` 目录运行 `npm run check`：退出码 0；页面校验、流程校验、全量测试（339/339）和微信小程序构建均通过。构建仅输出项目既有 Sass 弃用警告，无阻塞错误。

## 摘要同源参数去重修复

### 根因与 RED

类型配置的 `displayTemplate.summary` 已将 `quantityText`、`priceText` 映射到属性键（例如 `minOrderText`、`factoryPriceText`），但资源详情响应此前没有返回这些来源键。客户端只能合并核心摘要与全部 `attributeItems`，因此会重复展示同一属性；按标签或值去重又会误删同名但不同来源的业务字段。

新增真实结构回归测试后：

- `node --test common/resourceDetailState.test.mjs` 失败，仍输出 `demandQuantityText` 和 `budgetRange` 两个同源属性；
- `go test ./app/internal/logic/resource` 失败，`ResourceDetailResp` 不含 `SummarySourceKeys`。

### 修复与 GREEN

- 后端详情响应从 `displayTemplate.summary` 解析并返回 `summarySourceKeys.quantityText` 与 `summarySourceKeys.priceText`；公开和自有详情均复用该响应构造流程。
- 小程序将属性 `key` 传入详情参数构建器，仅排除键命中上述两个来源键的属性；同标签但不同键的属性保留。
- 更新 `resource.api` 与生成类型定义以公开该响应契约。尝试以 `goctl api go` 再生时被既有 `resource.api` 的未解析 `WechatPayParams`（第 299 行）阻止，因此仅同步了这一已知字段到现有生成类型文件，未扩大无关改动。

验证结果：

```bash
cd backend && go test ./app/internal/logic/resource
cd wxapp && node --test common/resourceDetailState.test.mjs pages/resource/detail.test.mjs
cd wxapp && npm run check
```

后端聚焦测试通过；前端聚焦测试 28/28 通过；小程序全量检查退出码 0、全量测试 340/340 通过并完成构建。构建仅有既有 Sass 弃用警告。
