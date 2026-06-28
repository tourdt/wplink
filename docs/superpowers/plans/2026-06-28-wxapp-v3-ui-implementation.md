# wxapp v3 UI 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 按 v3 原型完善 wxapp 上线验收主路径 UI，让首页、搜索、详情、商家、发布、消息、我的发布、专题和 webview 达到可验收状态。

**架构：** 保留现有 uni-app 页面、路由和 API 调用，只重构展示结构、样式和已有数据的组织方式。先统一 `uni.scss` 与 `ResourceCard`，再逐页替换局部结构，避免引入新组件库和后端依赖。

**技术栈：** uni-app、Vue 3 `<script setup>`、SCSS、微信小程序、现有 `wxapp/scripts` 校验脚本。

---

## 文件结构

- 修改：`wxapp/uni.scss`，补充全局色彩、卡片、按钮、标签、空状态、底部操作栏等共享样式。
- 修改：`wxapp/components/ResourceCard.vue`，统一 v3 资源卡信息层级、图片占位、标签、价格、刷新时间和判断提示。
- 修改：`wxapp/pages/home/index.vue`，对齐 v3 首屏 Banner、搜索、信任条、活动入口、场景卡和推荐资源。
- 修改：`wxapp/pages/search/index.vue`，强化筛选、置顶说明、结果判断信息和无结果转需求。
- 修改：`wxapp/pages/resource/detail.vue`，补齐图库、规格摘要、商家信任块、联系前提示、同类推荐和底部操作栏。
- 修改：`wxapp/pages/merchant/detail.vue`，强化认证、主营能力、发布统计、权益说明和资源列表。
- 修改：`wxapp/pages/publish/index.vue`，重组权益卡、类型切换、表单状态、图片上传和提交动作。
- 修改：`wxapp/pages/messages/index.vue`，优化消息筛选、消息卡和效果反馈。
- 修改：`wxapp/pages/my-resources/index.vue`，优化生命周期状态、数据指标和操作区。
- 修改：`wxapp/pages/topic/index.vue`，补齐专题 hero、统计、筛选提示、资源列表和需求兜底。
- 修改：`wxapp/pages/webview/index.vue`，补齐活动包装、URL 展示、白名单阻断和返回平台资源入口。
- 修改：`wxapp/scripts/validate-flows.mjs` 和 `wxapp/scripts/validate-flows.test.mjs`，仅在 UI 文案或结构变化导致旧检查不准确时同步更新检查点。

## Task 1: 共享 UI 与资源卡

**Files:**
- Modify: `wxapp/uni.scss`
- Modify: `wxapp/components/ResourceCard.vue`
- Test: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 先运行现有资源卡流程检查作为 RED/基线**

Run: `npm run validate:flows`

Expected: PASS，确认当前检查点可用。

- [ ] **Step 2: 更新 `ResourceCard` 模板**

保持 `resource` prop 和 `open` 事件不变。模板需要包含 `平台核实`、`查看详情`、`formatRefreshedAt` 等现有流程检查点，并新增缺图占位和判断提示。

关键结构：

```vue
<view class="resource-card" @click="$emit('open', resource)">
  <image v-if="coverUrl" class="resource-thumb" :src="coverUrl" mode="aspectFill" />
  <view v-else class="resource-thumb placeholder-thumb">
    <text>{{ typeLabel }}</text>
  </view>
  <view class="card-main">
    <view class="tag-row">
      <text v-if="isVerifiedMerchant" class="tag verified">已认证</text>
      <text v-if="hasCreditTags" class="tag verified">平台核实</text>
      <text v-if="resource.typeCode" class="tag">{{ resource.typeCode }}</text>
    </view>
    <text class="resource-title">{{ resource.title || '资源标题待完善' }}</text>
    <text class="resource-meta">{{ resource.category || '品类待沟通' }} · {{ resource.quantityText || '数量待沟通' }}</text>
    <view class="card-foot">
      <text class="resource-price">{{ resource.priceText || '价格面议' }}</text>
      <text class="resource-action">查看详情</text>
    </view>
    <view class="merchant-row">
      <text class="merchant-name">{{ merchantName }}</text>
      <text class="refresh-time">{{ formatRefreshedAt(resource.refreshedAt) }}</text>
    </view>
    <text class="decision-tip">{{ decisionTip }}</text>
  </view>
</view>
```

- [ ] **Step 3: 更新 `ResourceCard` 计算属性**

新增 `typeLabel` 和 `decisionTip`，只使用已有字段。

```js
const typeLabel = computed(() => props.resource.typeCode || props.resource.category || '资源')
const decisionTip = computed(() => {
  if (hasCreditTags.value) return '平台已补充核实信息，联系前仍建议确认实物、价格和交付时间。'
  if (isVerifiedMerchant.value) return '认证商家发布，建议进入详情查看规格和联系方式。'
  return '建议进入详情确认数量、价格、看样方式和刷新时间。'
})
```

- [ ] **Step 4: 补充全局样式 token**

在 `wxapp/uni.scss` 增加共享变量和基础类，保留现有 `page` 和 `button::after` 规则。

```scss
$wplink-primary: #0f766e;
$wplink-primary-soft: #e6f4f1;
$wplink-bg: #f4f6f8;
$wplink-card: #ffffff;
$wplink-text: #1f2933;
$wplink-muted: #697586;
$wplink-line: #d8dde6;
$wplink-warning: #b7791f;
$wplink-price: #c2410c;
```

- [ ] **Step 5: 验证共享改造**

Run: `npm run validate:flows`

Expected: PASS。

## Task 2: 首页、搜索和资源详情

**Files:**
- Modify: `wxapp/pages/home/index.vue`
- Modify: `wxapp/pages/search/index.vue`
- Modify: `wxapp/pages/resource/detail.vue`
- Test: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 首页对齐 v3 首屏**

保留 `listHomeBanners`、`sceneEntries`、`openDemand`、`openPublish`、`ResourceCard` 和 `homeResources`。模板补齐信任条和活动入口：

```vue
<view class="trust-strip">
  <text>平台核实</text>
  <text>认证商家</text>
  <text>过期下架</text>
</view>
```

- [ ] **Step 2: 搜索页补齐判断信息和空状态**

保留 `listCityResourceTypes`、`searchResources`、`listSavedSearches`、`createSavedSearch`、`applySavedSearch` 和 `openDemand`。确保无结果区域继续包含：

```vue
<text class="empty-title">暂未找到合适资源</text>
<button class="primary-button" @click="openDemand">提交采购需求</button>
```

- [ ] **Step 3: 资源详情补齐图库和规格摘要**

保留 `recordResourceDetailView`、`recordContact('phone')`、`recordContact('wechat')`、`recordContact('merchant_home')`、`recordContact('share')`、收藏和分享逻辑。新增 `galleryImages` 和 `specItems`：

```js
const galleryImages = computed(() => {
  const images = resource.value.images || []
  const cover = resource.value.coverUrl ? [resource.value.coverUrl] : []
  return [...cover, ...images].filter(Boolean)
})
const specItems = computed(() => [
  { label: '品类', value: resource.value.category || '待沟通' },
  { label: '数量', value: resource.value.quantityText || '待沟通' },
  { label: '价格', value: resource.value.priceText || '面议' },
  { label: '刷新', value: resource.value.refreshedAt || '近期更新' },
])
```

- [ ] **Step 4: 验证三页流程**

Run: `npm run validate:flows`

Expected: PASS。

## Task 3: 商家、发布、消息和我的发布

**Files:**
- Modify: `wxapp/pages/merchant/detail.vue`
- Modify: `wxapp/pages/publish/index.vue`
- Modify: `wxapp/pages/messages/index.vue`
- Modify: `wxapp/pages/my-resources/index.vue`
- Test: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 商家主页强化认证和能力信息**

保留 `getMerchant`、`listResources`、`ResourceCard`、`resourcesSummary`、`merchantImages`、`getMerchantFollowState`、`setMerchantFollow` 和 `toggleFollow`。头部展示认证和主营能力，权益文案继续包含 `权益提示` 与 `联系前建议先从资源详情进入`。

- [ ] **Step 2: 发布页重组表单和权益提示**

保留 `listCityResourceTypes`、`createResource`、`submitResource`、`createResourceDraft`、`chooseAndUploadImage`、`uploadResourceImage`、`validatePublishForm` 和既有必填校验文案。新增只读状态提示：

```vue
<view class="form-status">
  <text>{{ publishReadyText }}</text>
  <strong>{{ form.title && form.category && form.contact.name && form.contact.phone ? '可提交审核' : '继续补充必填项' }}</strong>
</view>
```

- [ ] **Step 3: 消息页强化效果反馈**

保留 `listMessages`、`readMessage`、`selectStatus('unread')`、`selectStatus('read')`、`markRead`、`messageTabs`、`商家本周效果` 和 `查看我的资源`。消息卡需要突出未读、标题、时间、目标跳转提示。

- [ ] **Step 4: 我的发布强化生命周期状态**

保留 `listMyResources`、`MetricStrip`、`refreshResource`、`listTopVouchers`、`redeemTopVoucher`、`markResourceDeal`、`takeDownResource`、`submitDraft`、`submitResource`、`repostSimilarResource`、`canTopResource` 和 `再发类似`。操作按钮保持原有条件渲染。

- [ ] **Step 5: 验证管理相关页面**

Run: `npm run validate:flows`

Expected: PASS。

## Task 4: 专题和 webview

**Files:**
- Modify: `wxapp/pages/topic/index.vue`
- Modify: `wxapp/pages/webview/index.vue`
- Test: `wxapp/scripts/validate-flows.mjs`

- [ ] **Step 1: 专题页补齐 v3 运营专题结构**

保留 `getTopicResources`、`ResourceCard`、`demandEntry`、`openDemand`、`Banner 专题`、`topicStats` 和 `没有找到想要的款`。新增筛选提示行：

```vue
<scroll-view class="filter-row" scroll-x>
  <button class="filter-button active">全部</button>
  <button class="filter-button">整包清</button>
  <button class="filter-button">可直播</button>
  <button class="filter-button">平台核实</button>
</scroll-view>
```

- [ ] **Step 2: webview 页补齐活动包装和阻断态**

保留 `validateWebview`、`allowedUrl` 和 `链接不可访问`。当 `allowedUrl` 存在时，在 web-view 之前展示 URL 状态和活动提示；当不存在时展示白名单说明和返回动作。

- [ ] **Step 3: 验证专题和 webview**

Run: `npm run validate:flows`

Expected: PASS。

## Task 5: 全量验证和收尾

**Files:**
- Test: `wxapp/scripts/validate-pages.mjs`
- Test: `wxapp/scripts/validate-flows.mjs`
- Test: `wxapp/scripts/validate-flows.test.mjs`
- Test: `wxapp/package.json`

- [ ] **Step 1: 跑页面配置校验**

Run: `npm run validate:pages`

Expected: `wxapp pages ok`

- [ ] **Step 2: 跑流程校验**

Run: `npm run validate:flows`

Expected: `wxapp flows ok`

- [ ] **Step 3: 跑 node 测试**

Run: `node --test scripts/validate-flows.test.mjs`

Expected: 3 tests pass, 0 fail。

- [ ] **Step 4: 尝试构建微信小程序**

Run: `npm run build:mp-weixin`

Expected: 构建成功；如果本地 CLI 缺少微信构建依赖，记录失败原因，不伪造通过。

## 自检

- spec 中的 9 个页面均有任务覆盖。
- 未引入新后端接口和数据库字段。
- 计划中的文档说明使用中文；代码标识、路径、命令保留英文。
- 验证命令基于现有 `wxapp/package.json`。
- 每个任务都保留现有流程校验关键字符串，降低 UI 改造破坏业务路径的风险。
