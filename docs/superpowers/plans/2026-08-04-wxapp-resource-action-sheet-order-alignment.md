# 供需详情操作弹窗顺序与标题对齐实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 删除访客“更多操作”的说明小字，将举报移动到最左侧，并让访客与管理弹窗的关闭按钮相对左侧标题区域垂直居中。

**架构：** 继续在 `pages/resource/detail.vue` 内使用现有弹窗结构、动作按钮和事件函数，只调整访客按钮模板顺序、删除说明节点，并统一标题栏的交叉轴对齐方式。测试沿用项目现有源码契约方式锁定顺序、文案移除、分享属性和标题栏对齐，再通过完整微信小程序构建验证编译兼容性。

**技术栈：** Vue 3、uni-app、微信小程序、SCSS、Node.js `node:test`

## 全局约束

- 所有实施计划和代码注释使用中文，代码标识、文件路径和命令保留英文。
- 访客“更多操作”顺序固定为：举报、收藏或取消收藏、分享给朋友。
- “更多操作”标题下不显示说明小字，也不保留空白占位。
- 访客与管理弹窗的关闭按钮均相对左侧标题区域垂直居中。
- 管理弹窗无动作时继续显示现有状态说明和“暂无可操作功能”。
- 不增加或删除业务动作，不修改动作处理函数、登录校验、分享回调、跳转或接口。
- 分享按钮必须保留 `open-type="share"`。
- 不修改底部联系栏、图标资源、按钮宫格尺寸或危险动作颜色。

## 文件结构

- 修改 `wxapp/pages/resource/detail.vue`：删除访客说明节点、重排三个访客动作、统一标题栏垂直对齐。
- 修改 `wxapp/pages/resource/detail.test.mjs`：验证访客动作顺序、说明文案移除、分享属性和两个弹窗共用的标题栏对齐契约。

---

### Task 1：调整访客动作顺序与两个弹窗标题对齐

**文件：**

- 修改：`wxapp/pages/resource/detail.vue:141-169,1746-1751`
- 测试：`wxapp/pages/resource/detail.test.mjs:176-231`

**接口：**

- 输入：现有 `reportResourceFromMore()`、`favoriteResourceFromMore()`、`shareResourceFromMore()` 与 `.sheet-head`。
- 输出：访客按钮 DOM 顺序为举报、收藏、分享；访客标题区只有标题；`.sheet-head` 对访客和管理弹窗统一使用垂直居中。
- 保留：三个事件函数的签名和调用，分享按钮的 `open-type="share"`，管理弹窗 `managementNotice` 的条件渲染。

- [ ] **步骤 1：先写会失败的访客弹窗契约测试**

在 `resource detail groups low-frequency contact actions behind more sheet` 用例中取得访客弹窗片段，并增加以下断言：

```js
const contactMoreSheet = source.match(/<view v-if="showContactMoreSheet"[\s\S]*?<view v-if="isOwnResource"/)?.[0] || ''

assert.doesNotMatch(contactMoreSheet, /收藏、分享给同行或反馈问题资源。/)
assert.match(contactMoreSheet, /@click="reportResourceFromMore"/)
assert.match(contactMoreSheet, /@click="favoriteResourceFromMore"/)
assert.match(contactMoreSheet, /open-type="share" @click="shareResourceFromMore"/)
assert.ok(contactMoreSheet.indexOf('reportResourceFromMore') < contactMoreSheet.indexOf('favoriteResourceFromMore'))
assert.ok(contactMoreSheet.indexOf('favoriteResourceFromMore') < contactMoreSheet.indexOf('shareResourceFromMore'))
```

把该用例中旧的收藏、分享、举报顺序断言改成上述新顺序，原处理函数行为断言继续保留。

- [ ] **步骤 2：新增标题栏垂直居中契约测试**

新增独立用例：

```js
test('resource detail action sheet close buttons align with their title areas', () => {
  assert.match(source, /\.sheet-head \{[\s\S]*grid-template-columns: minmax\(0, 1fr\) 120rpx;[\s\S]*align-items: center;/)
  assert.match(source, /<text v-if="!managementActions\.length" class="sheet-desc">\{\{ managementNotice \}\}<\/text>/)
  assert.doesNotMatch(source, /\.sheet-head \{[\s\S]*align-items: start;/)
})
```

- [ ] **步骤 3：运行定向测试并确认按预期失败**

运行：

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

预期：FAIL；失败原因是访客弹窗仍包含说明小字、举报仍在最右侧，或 `.sheet-head` 仍为 `align-items: start`。

- [ ] **步骤 4：删除说明节点并调整访客动作顺序**

将访客弹窗标题区改为只有标题：

```vue
<view class="sheet-copy">
  <text class="sheet-title">更多操作</text>
</view>
```

三个按钮严格按以下顺序放置，复用现有内部图标和文字结构：

```vue
<button class="management-action danger" @click="reportResourceFromMore">
  <view class="action-icon-wrap">
    <image class="action-icon" src="/static/action-icons/report.svg" mode="aspectFit" />
  </view>
  <text class="action-label">举报</text>
</button>
<button class="management-action" @click="favoriteResourceFromMore">
  <view class="action-icon-wrap">
    <image class="action-icon" src="/static/action-icons/bookmark.svg" mode="aspectFit" />
  </view>
  <text class="action-label">{{ favorited ? '取消收藏' : '收藏' }}</text>
</button>
<button class="management-action" open-type="share" @click="shareResourceFromMore">
  <view class="action-icon-wrap">
    <image class="action-icon" src="/static/action-icons/share.svg" mode="aspectFit" />
  </view>
  <text class="action-label">分享给朋友</text>
</button>
```

- [ ] **步骤 5：统一两个弹窗标题栏的垂直对齐**

只修改 `.sheet-head` 的交叉轴对齐：

```scss
.sheet-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 120rpx;
  gap: 16rpx;
  align-items: center;
}
```

保留 `.sheet-copy` 的网格结构和管理弹窗条件说明节点，因此无动作管理状态仍展示 `managementNotice`。

- [ ] **步骤 6：运行定向测试并确认通过**

运行：

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

预期：所有详情页测试 PASS，访客新顺序、说明文案移除、分享属性和标题栏居中断言通过。

- [ ] **步骤 7：运行页面流程和完整小程序检查**

运行：

```bash
cd wxapp && npm run check
```

预期：流程校验通过；页面校验、全部 Node 测试和 `build:mp-weixin` 均通过。

- [ ] **步骤 8：检查差异并提交**

运行：

```bash
git diff --check
git status --short
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "style: refine resource detail action sheets"
```

预期：`git diff --check` 无输出；提交只包含本需求相关的两个文件。
