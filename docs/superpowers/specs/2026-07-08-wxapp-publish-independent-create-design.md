# wxapp 发布独立新增页设计

## 目标

将小程序发布 tab 调整为纯入口页。用户点击“发布资源”或“发布需求”后，进入独立的非 tab 发布表单页完成新增，避免长表单直接占用 tab 页面导致返回、标题、底部安全区和表单状态不清晰。

## 范围

- 修改 `wxapp/pages/publish/index.vue`：保留发布类型选择，不再直接渲染 `ResourcePublishForm`。
- 修改 `wxapp/pages/publish/edit.vue`：继续承载编辑、再发类似，同时支持新增发布并接收 `direction` 参数。
- 修改 `wxapp/scripts/validate-flows.mjs`：更新默认流程校验的必备 token。
- 修改 `wxapp/scripts/validate-flows.test.mjs`：用静态测试约束入口跳转和独立页参数传递。
- 不修改后端接口、发布表单字段、草稿接口、提交接口和上传流程。

## 交互规则

- 发布 tab 首屏只展示“发布资源”和“发布需求”两个入口。
- 点击“发布资源”跳转到 `/pages/publish/edit?direction=supply`。
- 点击“发布需求”跳转到 `/pages/publish/edit?direction=demand`。
- 首页等外部入口如果通过 `wplink_pending_publish_type_code` 预设类型或方向，进入发布 tab 后也应继续跳转到独立发布页。
- `pages/publish/edit` 在无 `resourceId` 时按新增模式运行，在有 `resourceId` 时按编辑模式运行。
- 再发类似继续复用 `repost=1` 和 `publish:repost-initial-form`，不改变现有流程。

## 技术方案

`pages/publish/index.vue` 只负责解析发布方向和可选类型，并通过 `uni.navigateTo` 打开独立发布页。为避免 tab 页面残留上一次选择，入口页不再维护 `selectedDirection` 表单状态，只保留跳转前的参数归一化。

`pages/publish/edit.vue` 增加 `direction` 路由字段，并将 `mode` 从固定 `edit` 改成根据 `resourceId` 计算：存在 `resourceId` 时为 `edit`，否则为 `create`。这样新增、编辑、再发都继续复用同一个 `ResourcePublishForm`，不会新增重复表单页面。

## 验证标准

- 静态测试断言发布 tab 不包含 `ResourcePublishForm`。
- 静态测试断言 `startPublish` 使用 `uni.navigateTo` 打开 `/pages/publish/edit` 并携带 `direction`。
- 静态测试断言 `applyPendingPublishType` 会把外部预设的 `typeCode`、`direction` 带入独立发布页。
- 静态测试断言 `pages/publish/edit.vue` 接收 `direction` 并按是否存在 `resourceId` 切换 `mode`。
- 运行 `node wxapp/scripts/validate-flows.test.mjs` 和 `node wxapp/scripts/validate-pages.mjs` 通过。
