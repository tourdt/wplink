# wxapp Tab 图标设计规格

## 目标

为 `wplink/wxapp` 的底部 tabBar 设计并替换一套统一图标，参考 `wpmall/wxapp` 的 tab 尺寸节奏，让图标在微信小程序底栏中更清晰、统一，并保持当前 5 个 tab 的业务含义不变。

## 现状依据

- `wpmall/wxapp/pages.json` 使用 `fontSize: "10px"`、`iconWidth: "28px"`。
- `wpmall/wxapp/custom-tab-bar/index.wxss` 中普通 tab 图标显示为 `27px x 27px`，底栏主体高度为 `48px`。
- `wplink/wxapp/pages.json` 当前已有 5 个 tab：`首页`、`资源`、`发布`、`消息`、`我的`。
- `wplink/wxapp/static/tabbar` 当前已有同名 PNG 资源，尺寸为 `81 x 81`，适合作为 3 倍图输出后按约 `27px` 显示。

## 设计方向

采用已确认的 A 方案：精准线性图标。

- 图形风格：圆角线性图标，统一描边粗细，避免过多细节。
- 普通态颜色：沿用 `pages.json` 当前 `color`，即 `#454a54`。
- 选中态颜色：沿用 `pages.json` 当前 `selectedColor`，即 `#c23a00`。
- 图标画布：输出 `81 x 81` 透明 PNG。
- 可视主体：控制在约 `66 x 66` 内，保证按 `28px` 显示时有适当留白。
- 不做实心底、徽章、凸起按钮或自定义 tabBar，避免扩大改动范围。

## 图标语义

- `home` / `home-active`：房屋轮廓，表示首页。
- `search` / `search-active`：资源文档叠加搜索放大镜，表示资源列表与查找。
- `publish` / `publish-active`：圆角方框加号，表示发布。
- `messages` / `messages-active`：会话气泡，表示消息。
- `my` / `my-active`：用户头像轮廓，表示我的。

## 改动范围

- 覆盖 `wxapp/static/tabbar/*.png` 中 10 个 tab 图标文件。
- 在 `wxapp/pages.json` 的 `tabBar` 中补齐 `fontSize: "10px"` 和 `iconWidth: "28px"`，与 `wpmall` 的尺寸节奏一致。
- 不改 tab 数量、页面路径、文案、导航逻辑和业务代码。

## 验证标准

- 10 个 PNG 文件均存在，尺寸均为 `81 x 81`，带透明背景。
- `wxapp/pages.json` 中 tabBar 保持原有 5 个 tab，并新增尺寸配置。
- 图标普通态和选中态颜色分别匹配 `#454a54` 与 `#c23a00`。
- 小尺寸预览下每个图标语义可辨认，图标之间线条粗细和视觉大小一致。
