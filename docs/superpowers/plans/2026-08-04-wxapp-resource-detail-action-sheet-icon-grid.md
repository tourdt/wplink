# 供需详情操作弹窗图标宫格实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将供需详情页的访客“更多操作”和发布者“管理”弹窗统一改成图标在上、文字在下的轻量圆形宫格，同时完整保留现有业务行为。

**架构：** 不新增业务组件或第三方依赖，继续在 `pages/resource/detail.vue` 中维护动作状态与事件。动作图标使用小程序本地 SVG 静态资源；管理动作通过 `managementActions` 提供图标路径，访客三个固定动作保留各自的微信分享属性和处理函数。布局使用可居中的 `flex` 宫格，以同一套按钮结构兼容一至三个动作。

**技术栈：** Vue 3、uni-app、微信小程序、SCSS、Node.js `node:test`

## 全局约束

- 所有实施计划和代码注释使用中文，代码标识、文件路径和命令保留英文。
- 只修改供需详情操作弹窗，不改变底部联系栏、弹窗标题区、接口、数据库或权限逻辑。
- 不增加、删除或重新排序业务动作。
- 分享按钮必须保留 `open-type="share"`。
- 不引入第三方图标库，不依赖网络图标。
- 普通动作使用深蓝灰；举报、删除、下架使用克制的危险红色。
- 图标只辅助识别，所有动作必须保留可见文字标签和完整按钮点击热区。

## 文件结构

- 修改 `wxapp/pages/resource/detail.vue`：为两类弹窗渲染图标、补充管理动作图标路径并实现宫格样式。
- 修改 `wxapp/pages/resource/detail.test.mjs`：验证动作结构、图标映射、微信分享属性、危险态和自适应布局。
- 新建 `wxapp/static/action-icons/bookmark.svg`：收藏动作。
- 新建 `wxapp/static/action-icons/share.svg`：分享给朋友动作。
- 新建 `wxapp/static/action-icons/report.svg`：举报动作。
- 新建 `wxapp/static/action-icons/edit.svg`：编辑动作。
- 新建 `wxapp/static/action-icons/refresh.svg`：刷新动作。
- 新建 `wxapp/static/action-icons/top.svg`：置顶动作。
- 新建 `wxapp/static/action-icons/take-down.svg`：下架动作。
- 新建 `wxapp/static/action-icons/repost.svg`：再发类似动作。
- 新建 `wxapp/static/action-icons/delete.svg`：删除动作。

---

### 任务 1：补齐动作图标资源与弹窗语义结构

**文件：**

- 新建：`wxapp/static/action-icons/bookmark.svg`
- 新建：`wxapp/static/action-icons/share.svg`
- 新建：`wxapp/static/action-icons/report.svg`
- 新建：`wxapp/static/action-icons/edit.svg`
- 新建：`wxapp/static/action-icons/refresh.svg`
- 新建：`wxapp/static/action-icons/top.svg`
- 新建：`wxapp/static/action-icons/take-down.svg`
- 新建：`wxapp/static/action-icons/repost.svg`
- 新建：`wxapp/static/action-icons/delete.svg`
- 修改：`wxapp/pages/resource/detail.vue:124-151,361-383`
- 测试：`wxapp/pages/resource/detail.test.mjs:132-199`

**接口：**

- 输入：现有 `managementActions: ComputedRef<Array<{ key: string, label: string, danger?: boolean }>>` 与访客弹窗事件函数。
- 输出：`managementActions` 每项新增 `icon: string`；两类按钮内部统一包含 `.action-icon-wrap > image.action-icon` 和 `text.action-label`。
- 保留：`favoriteResourceFromMore()`、`shareResourceFromMore()`、`reportResourceFromMore()`、`handleManagementAction(key)` 的签名与调用时机。

- [ ] **步骤 1：先写会失败的结构测试**

在 `resource detail groups low-frequency contact actions behind more sheet` 用例中，将旧的纯文字按钮断言替换为以下结构断言，并新增管理动作图标映射断言：

```js
assert.match(source, /<button class="management-action" @click="favoriteResourceFromMore">[\s\S]*bookmark\.svg[\s\S]*\{\{ favorited \? '取消收藏' : '收藏' \}\}[\s\S]*<\/button>/)
assert.match(source, /<button class="management-action" open-type="share" @click="shareResourceFromMore">[\s\S]*share\.svg[\s\S]*分享给朋友[\s\S]*<\/button>/)
assert.match(source, /<button class="management-action danger" @click="reportResourceFromMore">[\s\S]*report\.svg[\s\S]*举报[\s\S]*<\/button>/)
assert.match(source, /\{ key: 'edit', label: '编辑', icon: actionIconPaths\.edit \}/)
assert.match(source, /\{ key: 'repost', label: '再发类似', icon: actionIconPaths\.repost \}/)
assert.match(source, /\{ key: 'delete', label: '删除', icon: actionIconPaths\.delete, danger: true \}/)
assert.match(source, /\{ key: 'refresh', label: '刷新', icon: actionIconPaths\.refresh \}/)
assert.match(source, /\{ key: 'top', label: '置顶', icon: actionIconPaths\.top \}/)
assert.match(source, /\{ key: 'take-down', label: '下架', icon: actionIconPaths\.takeDown, danger: true \}/)
```

再新增一个静态资源测试，逐项校验九个文件存在且内容包含 `<svg`、`viewBox="0 0 24 24"`：

```js
test('resource detail action icons are bundled local svg assets', () => {
  for (const name of ['bookmark', 'share', 'report', 'edit', 'refresh', 'top', 'take-down', 'repost', 'delete']) {
    const icon = fs.readFileSync(path.join(root, 'static/action-icons', `${name}.svg`), 'utf8')
    assert.match(icon, /<svg/)
    assert.match(icon, /viewBox="0 0 24 24"/)
  }
})
```

- [ ] **步骤 2：运行定向测试，确认先失败**

运行：

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

预期：FAIL；失败信息指出旧按钮没有 `bookmark.svg` 或动作图标文件不存在。

- [ ] **步骤 3：创建九个本地线性 SVG**

每个文件的根元素固定为 `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#23364a" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">`，表中节点之后以 `</svg>` 结束。按下表写入精确节点；危险动作资源使用 `stroke="#be123c"`：

| 文件 | SVG 内部节点 |
|---|---|
| `bookmark.svg` | `<path d="M6.8 3.5h10.4a1 1 0 0 1 1 1v16l-6.2-4-6.2 4v-16a1 1 0 0 1 1-1Z"/>` |
| `share.svg` | `<circle cx="18" cy="5" r="2.5"/><circle cx="6" cy="12" r="2.5"/><circle cx="18" cy="19" r="2.5"/><path d="m8.2 10.8 7.5-4.4M8.2 13.2l7.5 4.4"/>` |
| `report.svg` | `<path d="M12 3 3.8 18a1.4 1.4 0 0 0 1.3 2h13.8a1.4 1.4 0 0 0 1.3-2L12 3Z"/><path d="M12 9v4.5M12 17h.01"/>` |
| `edit.svg` | `<path d="m4 20 4.2-1 10.4-10.4a2.1 2.1 0 0 0-3-3L5.2 16 4 20Z"/><path d="m14.5 6.5 3 3"/>` |
| `refresh.svg` | `<path d="M20 11a8 8 0 1 1-2.3-5.7"/><path d="M20 4v7h-7"/>` |
| `top.svg` | `<path d="M12 21V6M6.5 11.5 12 6l5.5 5.5"/><path d="M5 3h14"/>` |
| `take-down.svg` | `<path d="M12 3v15M6.5 12.5 12 18l5.5-5.5"/><path d="M5 21h14"/>` |
| `repost.svg` | `<path d="M4 12a8 8 0 0 1 13.7-5.6L20 8.7"/><path d="M20 4v4.7h-4.7M20 12a8 8 0 0 1-13.7 5.6L4 15.3"/><path d="M4 20v-4.7h4.7"/>` |
| `delete.svg` | `<path d="M4 7h16M9 7V4h6v3M7 7l1 13h8l1-13"/><path d="M10 11v5M14 11v5"/>` |

- [ ] **步骤 4：为管理动作补充图标路径**

在脚本区定义稳定映射并移除已不再使用的 `primary` 字段：

```js
const actionIconPaths = {
  edit: '/static/action-icons/edit.svg',
  repost: '/static/action-icons/repost.svg',
  delete: '/static/action-icons/delete.svg',
  refresh: '/static/action-icons/refresh.svg',
  top: '/static/action-icons/top.svg',
  takeDown: '/static/action-icons/take-down.svg',
}
```

`managementActions` 的返回值改成：

```js
return [{ key: 'edit', label: '编辑', icon: actionIconPaths.edit }]

return [
  { key: 'repost', label: '再发类似', icon: actionIconPaths.repost },
  { key: 'delete', label: '删除', icon: actionIconPaths.delete, danger: true },
]

return [{ key: 'repost', label: '再发类似', icon: actionIconPaths.repost }]

return [
  { key: 'refresh', label: '刷新', icon: actionIconPaths.refresh },
  { key: 'top', label: '置顶', icon: actionIconPaths.top },
  { key: 'take-down', label: '下架', icon: actionIconPaths.takeDown, danger: true },
]
```

- [ ] **步骤 5：将两个弹窗按钮改成图标与文字纵排结构**

管理弹窗按钮改为：

```vue
<button
  v-for="action in managementActions"
  :key="action.key"
  :class="['management-action', action.danger ? 'danger' : '']"
  @click="handleManagementAction(action.key)"
>
  <view class="action-icon-wrap">
    <image class="action-icon" :src="action.icon" mode="aspectFit" />
  </view>
  <text class="action-label">{{ action.label }}</text>
</button>
```

访客弹窗保留三个独立按钮及原事件属性，分别使用 `bookmark.svg`、`share.svg`、`report.svg`；每个按钮内部都使用相同的 `.action-icon-wrap`、`.action-icon` 和 `.action-label` 结构。分享按钮类名不再包含 `primary`，但仍保留 `open-type="share"`。

- [ ] **步骤 6：运行定向测试，确认结构测试通过**

运行：

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

预期：PASS；原收藏、分享、举报和管理行为断言也继续通过。

- [ ] **步骤 7：提交动作结构与资源**

```bash
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs wxapp/static/action-icons
git commit -m "feat: add icons to resource detail actions"
```

---

### 任务 2：实现轻量圆形自适应宫格并完成全量验证

**文件：**

- 修改：`wxapp/pages/resource/detail.vue:1760-1793`
- 测试：`wxapp/pages/resource/detail.test.mjs:176-199`

**接口：**

- 输入：任务 1 产出的 `.management-actions`、`.management-action`、`.action-icon-wrap`、`.action-icon` 和 `.action-label`。
- 输出：一至三个动作自动居中的轻量圆形宫格；危险动作只改变图标圆形容器语义色。
- 不改变：所有按钮事件、`open-type`、弹窗遮罩和关闭逻辑。

- [ ] **步骤 1：先写会失败的样式契约测试**

在详情页测试中新增：

```js
test('resource detail action sheets use centered icon-over-label layout', () => {
  assert.match(source, /\.management-actions \{[\s\S]*display: flex;[\s\S]*justify-content: space-around;[\s\S]*gap: 14rpx;/)
  assert.match(source, /\.management-action \{[\s\S]*display: flex;[\s\S]*flex-direction: column;[\s\S]*width: 200rpx;[\s\S]*min-height: 132rpx;/)
  assert.match(source, /\.action-icon-wrap \{[\s\S]*width: 76rpx;[\s\S]*height: 76rpx;[\s\S]*border-radius: 50%;/)
  assert.match(source, /\.action-icon \{[\s\S]*width: 40rpx;[\s\S]*height: 40rpx;/)
  assert.match(source, /\.management-action\.danger \.action-icon-wrap \{[\s\S]*background: #fff7f8;[\s\S]*border-color: #f2ced3;/)
  assert.doesNotMatch(source, /\.management-action\.primary/)
})
```

- [ ] **步骤 2：运行定向测试，确认样式契约先失败**

运行：

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

预期：FAIL；旧样式仍使用两列 `grid` 和 76rpx 横向按钮。

- [ ] **步骤 3：实现居中的纵向动作宫格**

将旧 `.management-actions`、`.management-action.primary` 和整块危险按钮样式替换为：

```scss
.management-actions {
  display: flex;
  justify-content: space-around;
  gap: 14rpx;
}

.management-action {
  display: flex;
  flex: 0 1 200rpx;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  width: 200rpx;
  min-width: 0;
  min-height: 132rpx;
  padding: 12rpx 6rpx;
  border: 0;
  border-radius: 12rpx;
  background: transparent;
  color: $wplink-primary;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1.25;
}

.management-action::after {
  border: 0;
}

.action-icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 76rpx;
  height: 76rpx;
  box-sizing: border-box;
  border: 1rpx solid $wplink-line;
  border-radius: 50%;
  background: $wplink-card;
  box-shadow: 0 8rpx 20rpx rgba(15, 23, 42, 0.08);
}

.action-icon {
  width: 40rpx;
  height: 40rpx;
}

.action-label {
  color: inherit;
  white-space: nowrap;
}

.management-action.danger {
  color: #be123c;
}

.management-action.danger .action-icon-wrap {
  border-color: #f2ced3;
  background: #fff7f8;
}
```

- [ ] **步骤 4：运行详情页测试并检查失败项**

运行：

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

预期：PASS。

- [ ] **步骤 5：运行供需详情相关流程校验**

运行：

```bash
cd wxapp && npm run validate:flows
```

预期：PASS，供需详情已有行为令牌均存在。

- [ ] **步骤 6：运行小程序全量检查**

运行：

```bash
cd wxapp && npm run check
```

预期：页面校验、流程校验、全部 Node 测试和 `build:mp-weixin` 均通过；构建产物包含 `static/action-icons/*.svg`。

- [ ] **步骤 7：检查差异与提交最终样式**

运行：

```bash
git diff --check
git status --short
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "style: use icon grid for resource detail sheets"
```

预期：`git diff --check` 无输出；提交只包含本需求相关文件。
