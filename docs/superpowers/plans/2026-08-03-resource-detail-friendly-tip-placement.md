# 供需详情友情提示位置调整实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将供需详情“友情提示”移动到商家资料之后、同类推荐之前，使交易提醒紧邻联系行为。

**Architecture:** 保持现有单页模板与数据流不变，只调整 `detail.vue` 中现有 `trust-card` 的模板顺序。使用页面源码结构测试锁定 `merchant-card → trust-card → related-section` 的相对位置，避免后续回归。

**Tech Stack:** Vue 3、uni-app、微信小程序、Node.js test runner

## Global Constraints

- 仅移动现有 `trust-card` 模板位置。
- 不修改友情提示的文案、样式或显示条件。
- 不修改商家资料入口和同类推荐的显示规则。
- 不调整详情数据请求、接口、状态或底部操作栏。
- 商家资料入口不显示时，友情提示仍自然承接在地址或详细参数之后。

---

### Task 1：调整友情提示的详情信息顺序

**Files:**
- Modify: `wxapp/pages/resource/detail.vue`
- Test: `wxapp/pages/resource/detail.test.mjs`

**Interfaces:**
- Consumes: 现有模板区块 `merchant-card`、`trust-card`、`related-section`
- Produces: 固定模板顺序 `merchant-card → trust-card → related-section`

- [ ] **Step 1：编写失败的模板顺序测试**

在 `wxapp/pages/resource/detail.test.mjs` 的友情提示测试之后增加：

```js
test('resource detail places the friendly tip after merchant and before related resources', () => {
  const merchantIndex = source.indexOf('class="merchant-card"')
  const tipIndex = source.indexOf('class="trust-card"')
  const relatedIndex = source.indexOf('class="related-section"')

  assert.ok(merchantIndex >= 0)
  assert.ok(tipIndex > merchantIndex)
  assert.ok(relatedIndex > tipIndex)
})
```

- [ ] **Step 2：运行定向测试并确认 RED**

```bash
cd wxapp
node --experimental-vm-modules --test pages/resource/detail.test.mjs
```

预期：新增测试在 `assert.ok(relatedIndex > tipIndex)` 失败，因为当前友情提示位于同类推荐之后。

- [ ] **Step 3：移动现有友情提示模板区块**

在 `wxapp/pages/resource/detail.vue` 中，将以下完整区块从 `related-section` 之后移动到 `merchant-card` 之后：

```vue
<view class="trust-card">
  <text class="section-title">友情提示</text>
  <text class="section-content contact-tip-content">联系{{ resourceNoun }}方前，建议先确认实物、价格、数量和交付方式。</text>
</view>
```

移动后紧接着保留原有 `related-section`。不得修改区块内容、class、条件或样式。

- [ ] **Step 4：运行定向测试并确认 GREEN**

```bash
cd wxapp
node --experimental-vm-modules --test pages/resource/detail.test.mjs
```

预期：详情页测试全部通过。

- [ ] **Step 5：运行小程序完整检查**

```bash
cd wxapp
npm run check
```

预期：页面校验、流程校验、全部 Node 测试和 `mp-weixin` 构建通过；允许保留项目已有 Sass 弃用提示。

- [ ] **Step 6：检查差异并提交**

```bash
cd ..
git diff --check
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "fix: move resource detail friendly tip"
```

预期：提交只包含模板顺序与对应测试，不包含样式、文案、接口或状态改动。

