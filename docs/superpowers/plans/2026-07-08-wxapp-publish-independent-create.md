# wxapp 发布独立新增页 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将小程序发布 tab 改为纯入口页，点击“发布资源/发布需求”后进入独立发布表单页完成新增。

**Architecture:** `pages/publish/index.vue` 只负责方向选择和外部预设参数转发；`pages/publish/edit.vue` 继续复用 `ResourcePublishForm`，通过是否存在 `resourceId` 自动选择新增或编辑模式。发布表单、接口、草稿、图片上传和再发类似流程保持不变。

**Tech Stack:** uni-app、Vue 3、Node.js 静态流程测试。

---

### Task 1: 用静态测试锁定独立新增流程

**Files:**
- Modify: `wxapp/scripts/validate-flows.test.mjs`
- Modify: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 写失败测试**

将 `publish tab supports supply and demand entry selection` 中的发布 tab 断言改成：

```js
  for (const token of [
    'publishDirectionOptions',
    '发布资源',
    '发布需求',
    'startPublish',
    'RESOURCE_DIRECTION_SUPPLY',
    'RESOURCE_DIRECTION_DEMAND',
    'navigateToPublishForm',
  ]) {
    assert.match(tabSource, new RegExp(token))
  }

  assert.equal(tabSource.includes('<ResourcePublishForm'), false)
  assert.equal(tabSource.includes('selectedDirection'), false)
  assert.match(tabSource, /function startPublish\(direction\) \{[\s\S]*navigateToPublishForm\(\{ typeCode: '', direction: publishDirection \}\)[\s\S]*\}/)
  assert.match(tabSource, /function navigateToPublishForm\(options = \{\}\) \{[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?\$\{query\.join\('&'\)\}` \}\)/)
  assert.match(tabSource, /initialPublishOptions\.direction && query\.push\(`direction=\$\{encodeURIComponent\(initialPublishOptions\.direction\)\}`\)/)
  assert.match(tabSource, /initialPublishOptions\.typeCode && query\.push\(`typeCode=\$\{encodeURIComponent\(initialPublishOptions\.typeCode\)\}`\)/)
```

将 `publish pages split tab creation and independent editing` 中的断言改成：

```js
  assert.equal(tabSource.includes('<ResourcePublishForm'), false)
  assert.match(tabSource, /PUBLISH_TYPE_KEY/)
  assert.match(tabSource, /onShow\(applyPendingPublishType\)/)
  assert.match(tabSource, /function applyPendingPublishType\(\) \{[\s\S]*uni\.getStorageSync\(PUBLISH_TYPE_KEY\)[\s\S]*navigateToPublishForm\(\{ typeCode: pendingTypeCode, direction: pendingDirection \}\)/)
  assert.match(editSource, /direction: ''/)
  assert.match(editSource, /routeOptions\.direction = options\.direction \|\| ''/)
  assert.match(editSource, /const publishFormMode = computed\(\(\) => routeOptions\.resourceId \? 'edit' : 'create'\)/)
  assert.match(editSource, /<ResourcePublishForm[\s\S]*:mode="publishFormMode"[\s\S]*:initial-options="routeOptions"/)
  assert.equal(tabSource.includes('publish:pending-edit-context'), false)
```

将 `publish tab page does not reserve bottom safe area for fixed save bar` 改成只断言独立页不需要该 prop：

```js
  assert.equal(publishSource.includes('reserve-bottom-safe-area'), false)
  assert.equal(editSource.includes('reserve-bottom-safe-area'), false)
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
node wxapp/scripts/validate-flows.test.mjs
```

Expected: 测试失败，失败原因是发布 tab 仍包含 `ResourcePublishForm`、`selectedDirection`，独立发布页还没有 `direction` 和动态 `mode`。

- [ ] **Step 3: 更新默认流程校验 token**

将 `wxapp/scripts/validate-flows.mjs` 中发布页相关检查改成：

```js
  {
    file: 'pages/publish/index.vue',
    description: 'tab 发布页入口',
    checks: ['publishDirectionOptions', 'navigateToPublishForm', '/pages/publish/edit?', 'PUBLISH_TYPE_KEY', 'applyPendingPublishType'],
  },
  {
    file: 'pages/publish/edit.vue',
    description: '独立资源编辑页入口',
    checks: ['ResourcePublishForm', 'onLoad', 'routeOptions', 'publishFormMode', ':mode="publishFormMode"', 'direction'],
  },
```

### Task 2: 调整发布 tab 和独立发布页

**Files:**
- Modify: `wxapp/pages/publish/index.vue`
- Modify: `wxapp/pages/publish/edit.vue`

- [ ] **Step 1: 修改发布 tab 为纯入口页**

把 `wxapp/pages/publish/index.vue` 调整为：

```vue
<template>
  <view class="publish-entry-page">
    <view class="entry-head">
      <text class="entry-title">发布</text>
      <text class="entry-desc">选择本次要发布的内容类型，后续字段会按资源或需求自动切换。</text>
    </view>
    <view class="publish-direction-grid">
      <button
        v-for="item in publishDirectionOptions"
        :key="item.value"
        :class="['publish-direction-card', item.value]"
        @click="startPublish(item.value)"
      >
        <text class="direction-title">{{ item.title }}</text>
        <text class="direction-desc">{{ item.desc }}</text>
      </button>
    </view>
  </view>
</template>

<script setup>
import { onLoad, onShow } from '@dcloudio/uni-app'

const PUBLISH_TYPE_KEY = 'wplink_pending_publish_type_code'
const RESOURCE_DIRECTION_SUPPLY = 'supply'
const RESOURCE_DIRECTION_DEMAND = 'demand'
const publishDirectionOptions = [
  {
    title: '发布资源',
    desc: '发布现货、库存、工厂、招聘、服务等供给信息',
    value: RESOURCE_DIRECTION_SUPPLY,
  },
  {
    title: '发布需求',
    desc: '发布找现货、找库存、找工厂、找服务等采购需求',
    value: RESOURCE_DIRECTION_DEMAND,
  },
]

onLoad(applyPendingPublishType)
onShow(applyPendingPublishType)

function applyPendingPublishType() {
  const pendingPublish = uni.getStorageSync(PUBLISH_TYPE_KEY)
  if (!pendingPublish) return
  uni.removeStorageSync(PUBLISH_TYPE_KEY)
  const pendingTypeCode = typeof pendingPublish === 'object'
    ? pendingPublish.typeCode || ''
    : pendingPublish
  const pendingDirection = normalizePublishDirection(
    typeof pendingPublish === 'object' ? pendingPublish.direction : '',
  ) || RESOURCE_DIRECTION_SUPPLY
  navigateToPublishForm({ typeCode: pendingTypeCode, direction: pendingDirection })
}

function startPublish(direction) {
  const publishDirection = normalizePublishDirection(direction) || RESOURCE_DIRECTION_SUPPLY
  navigateToPublishForm({ typeCode: '', direction: publishDirection })
}

function navigateToPublishForm(options = {}) {
  const initialPublishOptions = {
    typeCode: options.typeCode || '',
    direction: normalizePublishDirection(options.direction || '') || RESOURCE_DIRECTION_SUPPLY,
  }
  const query = []
  initialPublishOptions.direction && query.push(`direction=${encodeURIComponent(initialPublishOptions.direction)}`)
  initialPublishOptions.typeCode && query.push(`typeCode=${encodeURIComponent(initialPublishOptions.typeCode)}`)
  uni.navigateTo({ url: `/pages/publish/edit?${query.join('&')}` })
}

function normalizePublishDirection(value) {
  return [RESOURCE_DIRECTION_SUPPLY, RESOURCE_DIRECTION_DEMAND].includes(value) ? value : ''
}
</script>
```

保留原有 `<style scoped>`，删除 `ResourcePublishForm` import、`ref` import、`selectedDirection` 和 `initialOptions`。

- [ ] **Step 2: 修改独立发布页支持新增模式**

把 `wxapp/pages/publish/edit.vue` 调整为：

```vue
<template>
  <ResourcePublishForm v-if="routeReady" :mode="publishFormMode" :initial-options="routeOptions" />
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ResourcePublishForm from '../../components/ResourcePublishForm.vue'

const routeReady = ref(false)
const routeOptions = reactive({
  merchantId: '',
  resourceId: '',
  typeCode: '',
  direction: '',
  repost: '',
})
const publishFormMode = computed(() => routeOptions.resourceId ? 'edit' : 'create')

onLoad((options) => {
  routeOptions.merchantId = options.merchantId || ''
  routeOptions.resourceId = options.resourceId || ''
  routeOptions.typeCode = options.typeCode || ''
  routeOptions.direction = options.direction || ''
  routeOptions.repost = options.repost || ''
  if (!routeOptions.resourceId) {
    uni.setNavigationBarTitle({
      title: routeOptions.direction === 'demand' ? '发布需求' : '发布资源',
    })
  }
  routeReady.value = true
})
</script>
```

- [ ] **Step 3: 运行测试确认通过**

Run:

```bash
node wxapp/scripts/validate-flows.test.mjs
```

Expected: PASS，发布流程静态测试和默认流程校验全部通过。

### Task 3: 页面注册校验和收口

**Files:**
- Verify: `wxapp/pages.json`
- Verify: `wxapp/scripts/validate-pages.mjs`

- [ ] **Step 1: 运行页面校验**

Run:

```bash
node wxapp/scripts/validate-pages.mjs
```

Expected: PASS，`pages/publish/index` 仍是 tab 页，`pages/publish/edit` 仍是普通页面且已注册。

- [ ] **Step 2: 检查差异**

Run:

```bash
git diff -- wxapp/pages/publish/index.vue wxapp/pages/publish/edit.vue wxapp/scripts/validate-flows.mjs wxapp/scripts/validate-flows.test.mjs docs/superpowers/specs/2026-07-08-wxapp-publish-independent-create-design.md docs/superpowers/plans/2026-07-08-wxapp-publish-independent-create.md
```

Expected: 差异只包含发布入口、独立发布页、静态测试和本次设计/计划文档。
