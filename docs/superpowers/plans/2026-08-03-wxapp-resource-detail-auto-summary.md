# 供需详情自动摘要 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不增加发布字段、不变更后端接口的前提下，重排供需详情页，使自动生成的业务摘要和关键交易条件在首屏可见。

**Architecture:** 新增纯前端状态工具，统一解析供需方向、首屏摘要、关键事实和参数跨度；详情页只消费该状态工具并重排展示层。继续使用现有详情接口的 `title`、`typeName`、`direction`、`priceText`、`quantityText` 与 `attributeItems`，不增加请求或数据库变更。

**Tech Stack:** Vue 3 `<script setup>`、uni-app、SCSS、Node.js `node:test`。

## Global Constraints

- 不新增手动标题输入框，不修改发布表单及字段校验。
- 不修改资源类型配置、后端 API、数据库结构或迁移。
- 所有供需相关文案必须以 `direction` 为准；历史数据缺少 `direction` 时保留现有 `typeCode` 兼容判断。
- 首屏仅展示有真实值的价格/预算与数量/面积，不生成“待沟通”“面议”等占位值。
- 商家资料入口仍仅在资源有商家 ID 且商家资料状态为 `completed` 时出现。
- 仅修改与详情页自动摘要布局直接相关的文件，不触碰现有收藏后端改动。

---

## File Structure

- Create: `wxapp/common/resourceDetailState.js` — 详情页的纯状态推导：供需方向、首屏摘要、关键事实、参数整行判断与动态文案。
- Create: `wxapp/common/resourceDetailState.test.mjs` — 对纯状态推导的行为回归测试。
- Modify: `wxapp/pages/resource/detail.vue` — 复用状态工具，重排首屏、描述、参数、商家、相关推荐和交易提醒布局。
- Modify: `wxapp/pages/resource/detail.test.mjs` — 验证详情页使用自动摘要状态、动态文案和参数跨度样式。

## Task 1: 建立详情页状态推导工具

**Files:**
- Create: `wxapp/common/resourceDetailState.js`
- Create: `wxapp/common/resourceDetailState.test.mjs`

**Interfaces:**
- Consumes: 详情接口资源对象中的 `direction`、`typeCode`、`typeName`、`title`、`category`、`quantityText`、`priceText` 与 `attributeItems`。
- Produces: `buildResourceDetailPresentation(resource)`，返回 `{ isDemand, noun, typeName, headline, facts }`；`buildDetailSpecItems(attributeItems)`，为每项返回 `{ label, value, fullWidth }`。

- [ ] **Step 1: 写入失败的纯函数测试**

```js
import test from 'node:test'
import assert from 'node:assert/strict'
import {
  buildDetailSpecItems,
  buildResourceDetailPresentation,
} from './resourceDetailState.js'

test('builds demand headline and only includes populated key facts', () => {
  const presentation = buildResourceDetailPresentation({
    direction: 'demand',
    typeName: '招加工厂',
    title: '招加工厂｜童装 5000 件 面议',
    quantityText: '5000 件',
    priceText: '面议',
  })

  assert.deepEqual(presentation, {
    isDemand: true,
    noun: '需求',
    typeName: '招加工厂',
    headline: '招加工厂｜童装 5000 件 面议',
    facts: [
      { key: 'quantity', label: '数量/面积', value: '5000 件' },
      { key: 'price', label: '预算/报价', value: '面议' },
    ],
  })
})

test('falls back to configured summary fields and legacy demand type codes', () => {
  const presentation = buildResourceDetailPresentation({
    typeCode: 'seek_factory_warehouse',
    typeName: '我要求租',
    category: '一楼仓',
    quantityText: '500-800 平',
    priceText: '3 万元/月以内',
  })

  assert.equal(presentation.isDemand, true)
  assert.equal(presentation.noun, '需求')
  assert.equal(presentation.headline, '一楼仓｜500-800 平｜3 万元/月以内')
})

test('marks long and semantic detail values as full width', () => {
  const items = buildDetailSpecItems([
    { label: '数量', value: '3200 件' },
    { label: '交期要求', value: '20 天内完成并支持首批打样确认' },
    { label: '服务范围', value: '织里及周边' },
  ])

  assert.deepEqual(items, [
    { label: '数量', value: '3200 件', fullWidth: false },
    { label: '交期要求', value: '20 天内完成并支持首批打样确认', fullWidth: true },
    { label: '服务范围', value: '织里及周边', fullWidth: true },
  ])
})
```

- [ ] **Step 2: 运行测试，确认因模块不存在而失败**

Run: `node --test wxapp/common/resourceDetailState.test.mjs`

Expected: FAIL，报错 `ERR_MODULE_NOT_FOUND`，目标为 `resourceDetailState.js`。

- [ ] **Step 3: 实现最小状态工具**

```js
const DEMAND_TYPE_PATTERN = /^(buy_|find_|seek_)/
const FULL_WIDTH_LABEL_PATTERN = /(地址|位置|区域|地点|交期|时效|范围|要求|工艺|备注|说明|时间|条件|方式)/

export function buildResourceDetailPresentation(resource = {}) {
  const isDemand = isDemandResource(resource)
  const typeName = normalizeText(resource.typeName) || normalizeText(resource.category) || '供需信息'
  const summaryParts = [resource.category, resource.quantityText, resource.priceText]
    .map(normalizeText)
    .filter(Boolean)
  return {
    isDemand,
    noun: isDemand ? '需求' : '供应',
    typeName,
    headline: normalizeText(resource.title) || summaryParts.join('｜') || typeName,
    facts: [
      { key: 'quantity', label: '数量/面积', value: normalizeText(resource.quantityText) },
      { key: 'price', label: isDemand ? '预算/报价' : '价格/报价', value: normalizeText(resource.priceText) },
    ].filter((item) => item.value),
  }
}

export function buildDetailSpecItems(attributeItems = []) {
  return attributeItems
    .filter((item) => normalizeText(item?.label) && normalizeText(item?.value))
    .map((item) => ({
      label: normalizeText(item.label),
      value: normalizeText(item.value),
      fullWidth: shouldUseFullWidthSpec(item.label, item.value),
    }))
}

function isDemandResource(resource) {
  const direction = normalizeText(resource.direction)
  if (direction === 'demand') return true
  const typeCode = normalizeText(resource.typeCode)
  return DEMAND_TYPE_PATTERN.test(typeCode) || typeCode.endsWith('_buy') || typeCode === 'job_seeking'
}

function shouldUseFullWidthSpec(label, value) {
  return FULL_WIDTH_LABEL_PATTERN.test(normalizeText(label)) || /\r?\n/.test(String(value)) || Array.from(normalizeText(value)).length > 12
}

function normalizeText(value) {
  return String(value || '').trim()
}
```

- [ ] **Step 4: 运行状态工具测试，确认通过**

Run: `node --test wxapp/common/resourceDetailState.test.mjs`

Expected: PASS，3 个测试全部通过。

- [ ] **Step 5: 提交状态工具与用例**

```bash
git add wxapp/common/resourceDetailState.js wxapp/common/resourceDetailState.test.mjs
git commit -m "feat: add resource detail presentation state"
```

## Task 2: 重排详情页并接入自动摘要

**Files:**
- Modify: `wxapp/pages/resource/detail.vue:1-190`
- Modify: `wxapp/pages/resource/detail.vue:296-400`
- Modify: `wxapp/pages/resource/detail.vue:1256-1871`
- Modify: `wxapp/pages/resource/detail.test.mjs`

**Interfaces:**
- Consumes: `buildResourceDetailPresentation(resource)` 和 `buildDetailSpecItems(attributeItems)`。
- Produces: 首屏摘要卡、动态文案与支持整行参数项的详情页视图。

- [ ] **Step 1: 写入失败的详情页结构回归测试**

在 `wxapp/pages/resource/detail.test.mjs` 添加以下测试：

```js
test('resource detail places automatic summary before description and merchant', () => {
  const galleryIndex = source.indexOf('class="detail-gallery"')
  const summaryIndex = source.indexOf('class="detail-summary-card"')
  const descriptionIndex = source.indexOf('class="description-card"')
  const merchantIndex = source.indexOf('class="merchant-card"')

  assert.ok(summaryIndex > galleryIndex)
  assert.ok(descriptionIndex > summaryIndex)
  assert.ok(merchantIndex > descriptionIndex)
  assert.match(source, /<text class="summary-title">\{\{ detailPresentation\.headline \}\}<\/text>/)
  assert.match(source, /v-for="item in detailPresentation\.facts"/)
})

test('resource detail derives supply and demand copy from presentation noun', () => {
  assert.match(source, /const resourceNoun = computed\(\(\) => detailPresentation\.value\.noun\)/)
  assert.match(source, /`\$\{resourceNoun\.value\}管理`/)
  assert.match(source, /`暂无同类\$\{resourceNoun\.value\}`/)
})

test('resource detail renders long specs in full width rows', () => {
  assert.match(source, /:class="\['spec-item', item\.fullWidth \? 'full-width' : ''\]"/)
  assert.match(source, /\.spec-item\.full-width \{[\s\S]*grid-column: 1 \/ -1;/)
})
```

- [ ] **Step 2: 运行详情页测试，确认因新结构不存在而失败**

Run: `node --test wxapp/pages/resource/detail.test.mjs`

Expected: FAIL，断言找不到 `detail-summary-card`、`description-card` 与 `resourceNoun`。

- [ ] **Step 3: 接入状态工具并最小化重排模板**

在 `detail.vue`：

```js
import {
  buildDetailSpecItems,
  buildResourceDetailPresentation,
} from '../../common/resourceDetailState'

const detailPresentation = computed(() => buildResourceDetailPresentation(resource.value))
const resourceNoun = computed(() => detailPresentation.value.noun)
const isDemandResource = computed(() => detailPresentation.value.isDemand)
const specItems = computed(() => buildDetailSpecItems(attributeSpecItems.value))
```

将图库之后的内容重排为：

```vue
<view class="detail-summary-card">
  <view class="summary-kicker">
    <text :class="['direction-badge', detailPresentation.isDemand ? 'demand' : '']">{{ resourceNoun }}</text>
    <text class="summary-type">{{ detailPresentation.typeName }}</text>
  </view>
  <text class="summary-title">{{ detailPresentation.headline }}</text>
  <view v-if="detailPresentation.facts.length" class="summary-facts">
    <view v-for="item in detailPresentation.facts" :key="item.key" class="summary-fact">
      <text class="summary-fact-label">{{ item.label }}</text>
      <text class="summary-fact-value">{{ item.value }}</text>
    </view>
  </view>
  <view v-if="resourceFeatureTags.length" class="tag-row">
    <text v-for="tag in resourceFeatureTags" :key="tag" class="tag feature">{{ tag }}</text>
  </view>
</view>

<view v-if="resource.description" class="description-card">
  <text class="section-title">补充说明</text>
  <text class="desc">{{ resource.description }}</text>
</view>
```

保留已成交通知，但将它作为 `detail-summary-card` 顶部的状态通知。将 `merchant-card` 移至地址区之后，`trust-card` 移至相关推荐之后。将推荐组件的 `empty-text` 绑定改为 ``:empty-text="`暂无同类${resourceNoun}`"``。

将管理页和操作反馈中的固定“供应”替换为 `resourceNoun.value`，至少覆盖：管理标题、审核说明、过期说明、分享受限提示、收藏提示、下架确认标题与内容、置顶确认标题和购买确认内容。

- [ ] **Step 4: 为参数跨度和首屏层级补齐样式**

在现有 SCSS 中新增并使用：

```scss
.detail-summary-card,
.description-card {
  display: grid;
  gap: 16rpx;
  margin-bottom: 20rpx;
  padding: 28rpx 24rpx;
  border-radius: 12rpx;
  background: $wplink-card;
}

.summary-kicker,
.summary-facts {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.summary-title {
  color: $wplink-primary;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 1.35;
  word-break: break-word;
}

.summary-fact {
  display: grid;
  gap: 4rpx;
  min-width: 180rpx;
  padding: 14rpx 16rpx;
  border-radius: 10rpx;
  background: #f8fafc;
}

.spec-item.full-width {
  grid-column: 1 / -1;
}
```

为 `direction-badge.demand` 使用现有需求色，供应继续使用现有主色；不新增全局颜色变量。

- [ ] **Step 5: 运行详情页测试，确认通过**

Run: `node --test wxapp/pages/resource/detail.test.mjs`

Expected: PASS，包含现有行为和新增自动摘要、动态文案、参数跨度断言。

- [ ] **Step 6: 提交详情页布局改动与用例**

```bash
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "feat: improve resource detail summary layout"
```

## Task 3: 全量前端回归验证

**Files:**
- Verify: `wxapp/common/resourceDetailState.test.mjs`
- Verify: `wxapp/pages/resource/detail.test.mjs`
- Verify: `wxapp/scripts/validate-pages.mjs`
- Verify: `wxapp/scripts/validate-flows.mjs`

**Interfaces:**
- Consumes: Task 1 与 Task 2 的已提交实现。
- Produces: 可重复的前端回归验证证据。

- [ ] **Step 1: 运行详情状态与详情页测试**

Run: `node --test wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.test.mjs`

Expected: PASS，全部测试通过。

- [ ] **Step 2: 运行页面与流程静态校验**

Run: `node wxapp/scripts/validate-pages.mjs && node wxapp/scripts/validate-flows.mjs`

Expected: 两个命令退出码均为 0。

- [ ] **Step 3: 检查改动范围与格式**

Run: `git diff --check HEAD~2..HEAD && git status --short`

Expected: 无空白错误；仅显示本次提交之外的用户既有收藏模块改动（如仍存在）。

- [ ] **Step 4: 记录验证结果并提交修正（如验证发现本次改动问题）**

若 Task 1 或 Task 2 的文件需要为通过上述命令而修正，先为该问题添加失败断言，再进行最小修正，并使用：

```bash
git add wxapp/common/resourceDetailState.js wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "fix: verify resource detail summary layout"
```

若所有命令首次即通过，不创建额外提交。
