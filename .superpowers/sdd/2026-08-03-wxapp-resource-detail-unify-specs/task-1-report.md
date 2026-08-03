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
