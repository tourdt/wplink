# wxapp 商家主页布局优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `wxapp/pages/merchant/detail.vue` 调整为已确认的“信任首屏”商家主页布局。

**Architecture:** 只在商家主页页面内调整模板、计算属性和 scoped 样式。新增的结构测试放入现有 `wxapp/scripts/validate-flows.test.mjs`，通过源码断言保护页面结构和关键行为入口，不新增运行时依赖。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、SCSS、Node.js `node:test`。

---

## 文件结构

- Modify: `wxapp/scripts/validate-flows.test.mjs`
  - 增加商家主页布局结构断言，先失败再实现。
- Modify: `wxapp/pages/merchant/detail.vue`
  - 调整模板为信任头部、商家画像、地址图片、轻量提示、发布记录。
  - 增加少量 computed 用于主营标签和简介兜底。
  - 更新 scoped 样式以匹配工业 B2B 主题。

## Task 1: 写入失败的布局结构测试

**Files:**
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [x] **Step 1: 在 `validate-flows.test.mjs` 末尾添加测试**

```js
test('merchant detail page uses trust-first homepage layout', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/merchant/detail.vue'), 'utf8')

  for (const token of [
    'merchant-hero-card',
    'merchant-identity-row',
    'merchant-summary',
    'hero-stats',
    'profile-panel',
    'profile-chip-row',
    'profile-description',
    'media-section',
    'merchant-gallery',
    'trust-note-section',
    'merchantCategoryTags',
    'profileDescription',
    'statCards',
    '主营品类待补充',
    '从资源详情进入可查看完整联系方式',
  ]) {
    assert.match(source, new RegExp(token))
  }

  for (const removedToken of [
    'class="merchant-stats"',
    '发布概况</text>',
    'benefit-section',
  ]) {
    assert.equal(source.includes(removedToken), false)
  }
})
```

- [x] **Step 2: 运行测试并确认失败**

Run: `cd wxapp && node --test scripts/validate-flows.test.mjs`

Expected: FAIL，失败原因包含 `merchant-hero-card` 或其它新布局 token 未匹配。

## Task 2: 实现信任首屏布局

**Files:**
- Modify: `wxapp/pages/merchant/detail.vue`

- [x] **Step 1: 替换 template 结构**

将页面首屏改为：

- `merchant-hero-card`
- `merchant-identity-row`
- `merchant-summary`
- `hero-stats`
- `profile-panel`
- 条件展示地址块和 `media-section`
- 轻量 `trust-note-section`
- 保留 `resource-list` 和 `contact-bar`

- [x] **Step 2: 增加 computed**

在现有 computed 附近增加：

```js
const merchantCategoryTags = computed(() => merchant.value.mainCategories || [])
const profileDescription = computed(() => merchant.value.description || '暂无简介')
const statCards = computed(() => [
  {
    label: '当前资源',
    value: resourcesSummary.value.publishedCount || merchantResources.value.length || 0,
  },
  {
    label: '历史发布',
    value: resourcesSummary.value.totalCount || resourcesSummary.value.publishedCount || merchantResources.value.length || 0,
  },
  {
    label: '成交反馈',
    value: resourcesSummary.value.dealtCount || 0,
  },
])
```

- [x] **Step 3: 更新 scoped 样式**

删除旧 `.merchant-head`、`.merchant-stats`、`.benefit-section` 的布局依赖，新增深色头部卡、画像卡、媒体区和轻量提示样式。保持卡片圆角不超过现有 12rpx 体系，按钮与主题 token 一致。

- [x] **Step 4: 运行测试确认通过**

Run: `cd wxapp && node --test scripts/validate-flows.test.mjs`

Expected: PASS。

## Task 3: 页面验证

**Files:**
- Modify: `wxapp/pages/merchant/detail.vue`
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [x] **Step 1: 运行页面配置验证**

Run: `cd wxapp && npm run validate:pages`

Expected: `wxapp pages ok`。

- [x] **Step 2: 运行流程验证**

Run: `cd wxapp && npm run validate:flows`

Expected: `wxapp flows ok`。

- [x] **Step 3: 运行小程序构建**

Run: `cd wxapp && npm run build:mp-weixin`

Expected: 构建完成，退出码为 0。

## 自检

- Spec 覆盖：计划覆盖信任头部、商家画像、地址图片、发布记录、底部联系栏和验证标准。
- 占位扫描：无 TBD、TODO、待定项。
- 类型一致性：新增 computed 均只读取现有 `merchant`、`resourcesSummary` 和 `merchantResources`。
