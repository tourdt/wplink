# 小程序权益与次数购买闭环实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 让用户能从“我的”、发布和刷新额度不足场景直达正确的次数包，同时使权益和促销文案可信、支付后的页面数据可恢复刷新。

**架构：** 保持现有服务端订单、权益与错误码不变。新增前端工具统一次数包路由和确认弹窗；“我的”页按余额展示分级入口；发布表单和我的发布页只拦截 `QUOTA_NOT_ENOUGH` 并调用该工具；VIP 页承接路由参数、突出目标商品、只展示服务端促销标签。

**技术栈：** uni-app、Vue 3 `<script setup>`、Node.js 内置 `node:test`、现有 REST 请求层。

## 全局约束

- 不修改 Go API、数据库迁移、订单创建、支付回调或权益扣减。
- 路由参数只允许 `publish_quota` 和 `refresh_quota`；价格与商品权益始终以服务端订单快照为准。
- 只有服务端返回有效折扣与 `saleLabel` 才可展示促销标签与划线原价。
- 额度不足购买引导可取消；取消后不得创建订单、清空表单或刷新列表。
- 微信支付完成不等同权益已到账，文案必须准确。
- 新增业务分支添加说明用户路径和数据正确性的中文注释。

---

### Task 1: 统一次数购买路由与确认弹窗

**文件：**

- 新建：`wxapp/common/entitlementPurchase.js`
- 新建：`wxapp/common/entitlementPurchase.test.mjs`

**接口：**

- 导出 `QUOTA_TYPE_PUBLISH = 'publish_quota'`、`QUOTA_TYPE_REFRESH = 'refresh_quota'`。
- 导出 `buildQuotaPurchaseUrl(quotaType): string`，返回 `/pages/vip/index?tab=addons&quotaType=<quotaType>`。
- 导出 `confirmQuotaPurchase(quotaType): Promise<boolean>`，使用 `uni.showModal` 返回是否确认购买。

- [ ] **Step 1: 写入失败测试**

```js
test('buildQuotaPurchaseUrl routes publish quota purchase to add-ons', () => {
  assert.equal(
    buildQuotaPurchaseUrl('publish_quota'),
    '/pages/vip/index?tab=addons&quotaType=publish_quota',
  )
})

test('buildQuotaPurchaseUrl routes refresh quota purchase to add-ons', () => {
  assert.equal(
    buildQuotaPurchaseUrl('refresh_quota'),
    '/pages/vip/index?tab=addons&quotaType=refresh_quota',
  )
})
```

- [ ] **Step 2: 运行测试确认失败**

运行：`node --test wxapp/common/entitlementPurchase.test.mjs`

预期：FAIL，因工具模块或目标导出尚不存在。

- [ ] **Step 3: 实现最小工具**

```js
export const QUOTA_TYPE_PUBLISH = 'publish_quota'
export const QUOTA_TYPE_REFRESH = 'refresh_quota'

export function buildQuotaPurchaseUrl(quotaType) {
  const type = quotaType === QUOTA_TYPE_REFRESH ? QUOTA_TYPE_REFRESH : QUOTA_TYPE_PUBLISH
  return `/pages/vip/index?tab=addons&quotaType=${type}`
}

export function confirmQuotaPurchase(quotaType) {
  const isRefresh = quotaType === QUOTA_TYPE_REFRESH
  return new Promise((resolve) => {
    uni.showModal({
      title: isRefresh ? '刷新次数已用完' : '发布次数已用完',
      content: isRefresh ? '购买刷新次数后，可继续刷新当前发布。' : '购买发布次数后，可继续提交审核。',
      confirmText: isRefresh ? '购买刷新次数' : '购买发布次数',
      cancelText: '暂不购买',
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false),
    })
  })
}
```

在弹窗调用处增加中文注释：仅在服务端明确返回额度不足时引导购买，取消后保持用户原操作状态。

- [ ] **Step 4: 运行测试确认通过**

运行：`node --test wxapp/common/entitlementPurchase.test.mjs`

预期：PASS。

- [ ] **Step 5: 提交本任务**

```bash
git add wxapp/common/entitlementPurchase.js wxapp/common/entitlementPurchase.test.mjs
git commit -m "feat: add quota purchase navigation helper"
```

### Task 2: 在“我的”页显示分级购买入口

**文件：**

- 修改：`wxapp/pages/my/index.vue`
- 修改：`wxapp/pages/my/index.test.mjs`

**接口：**

- 消费 Task 1 的常量和 `buildQuotaPurchaseUrl`。
- 新增 `QUOTA_LOW_THRESHOLD = 2`、`publishQuotaLow`、`refreshQuotaLow`、`openQuotaPurchase(quotaType)`。

- [ ] **Step 1: 写入失败测试**

```js
assert.match(source, /const QUOTA_LOW_THRESHOLD = 2/)
assert.match(source, /v-if="publishQuotaRemaining === 0"[\s\S]*购买发布次数/)
assert.match(source, /v-else-if="publishQuotaLow"[\s\S]*即将用完 · 去补充/)
assert.match(source, /v-if="refreshQuotaRemaining === 0"[\s\S]*购买刷新次数/)
assert.match(source, /function openQuotaPurchase\(quotaType\)[\s\S]*buildQuotaPurchaseUrl\(quotaType\)/)
assert.doesNotMatch(source, /暂无可领取权益/)
```

- [ ] **Step 2: 运行测试确认失败**

运行：`node --test wxapp/pages/my/index.test.mjs`

预期：FAIL，因余额分级入口与稳定权益入口尚不存在。

- [ ] **Step 3: 实现最小页面改动**

1. 导入购买工具并在两块额度下按零余额、低余额、正常余额渲染对应操作。
2. 零余额展示主色“购买发布次数”或“购买刷新次数”；1–2 次展示“即将用完 · 去补充”；余额充足时不展示购买按钮。
3. `openQuotaPurchase(quotaType)` 使用 `uni.navigateTo({ url: buildQuotaPurchaseUrl(quotaType) })`。
4. `openBenefitOverview()` 无条件打开 `/pages/vip/index`，不再 toast“暂无可领取权益”。成长活动存在时，额外显示“免费获得”入口并复用 `openGrowthEntitlement()`。
5. 只增加额度块内部样式，不调整账户卡、服务列表或成长任务页面。

- [ ] **Step 4: 运行测试确认通过**

运行：`node --test wxapp/pages/my/index.test.mjs`

预期：PASS，并保留最近到期提示的既有测试。

- [ ] **Step 5: 提交本任务**

```bash
git add wxapp/pages/my/index.vue wxapp/pages/my/index.test.mjs
git commit -m "feat: highlight quota purchases on my page"
```

### Task 3: 让购买页承接购买意图并修正促销、支付反馈

**文件：**

- 修改：`wxapp/pages/vip/index.vue`
- 修改：`wxapp/pages/vip/index.test.mjs`

**接口：**

- 读取 `tab=addons`、`quotaType=publish_quota|refresh_quota`。
- 新增 `selectedQuotaType`、`quotaPackType(item)`、`sortQuotaPacksBySelectedType(items)`、`shouldShowPlanSaleLabel(plan)`。

- [ ] **Step 1: 写入失败测试**

```js
assert.match(source, /selectedQuotaType\.value = options\.quotaType \|\| ''/)
assert.match(source, /if \(options\.quotaType\) activeTab\.value = 'addons'/)
assert.match(source, /function sortQuotaPacksBySelectedType/)
assert.match(source, /function shouldShowPlanSaleLabel\(plan\)/)
assert.doesNotMatch(source, /monthly: '限时特价'/)
assert.match(source, /await requestWechatPayment\(payment\)[\s\S]*await loadVIPData\(\)[\s\S]*支付已完成，权益到账后会自动更新/)
```

- [ ] **Step 2: 运行测试确认失败**

运行：`node --test wxapp/pages/vip/index.test.mjs`

预期：FAIL，因目标商品排序、真实促销判断和支付后刷新尚不存在。

- [ ] **Step 3: 实现最小页面改动**

1. `onLoad` 读取 `quotaType`，其存在时强制打开 `addons`；目标商品只移到次数包列表首位，其余保持服务端排序。
2. 只有 `saleLabel` 非空且 `salePriceCent` 小于 `standardPriceCent` 时显示 VIP 促销标签；删除按套餐代码固定返回“限时特价”的逻辑。
3. 保持次数包的标签和划线原价也以服务端字段为准。
4. `requestWechatPayment` 返回成功后先 `await loadVIPData()`，再提示“支付已完成，权益到账后会自动更新”；接口直接返回 `paid` 时也执行同样刷新。
5. 不修改下单请求参数、价格计算或微信支付参数校验。

- [ ] **Step 4: 运行测试确认通过**

运行：`node --test wxapp/pages/vip/index.test.mjs`

预期：PASS。

- [ ] **Step 5: 提交本任务**

```bash
git add wxapp/pages/vip/index.vue wxapp/pages/vip/index.test.mjs
git commit -m "feat: route quota purchase intent to add-ons"
```

### Task 4: 在发布和刷新额度不足时承接购买

**文件：**

- 修改：`wxapp/components/ResourcePublishForm.vue`
- 修改：`wxapp/components/ResourcePublishForm.test.mjs`
- 修改：`wxapp/pages/my-resources/index.vue`
- 修改：`wxapp/pages/my-resources/index.test.mjs`

**接口：**

- 消费 Task 1 工具。
- 新增 `handlePublishQuotaError(err): Promise<boolean>` 与 `handleRefreshQuotaError(err): Promise<boolean>`。
- 仅识别请求层设置的 `err.code === 'QUOTA_NOT_ENOUGH'`。

- [ ] **Step 1: 写入失败测试**

```js
assert.match(publishSource, /import \{[\s\S]*confirmQuotaPurchase[\s\S]*QUOTA_TYPE_PUBLISH[\s\S]*\} from '\.\.\/common\/entitlementPurchase'/)
assert.match(publishSource, /async function handlePublishQuotaError\(err\)[\s\S]*err\?\.code !== 'QUOTA_NOT_ENOUGH'/)
assert.match(publishSource, /buildQuotaPurchaseUrl\(QUOTA_TYPE_PUBLISH\)/)
assert.match(listSource, /async function refresh\(item\) \{[\s\S]*catch \(err\) \{[\s\S]*handleRefreshQuotaError\(err\)/)
assert.match(listSource, /async function handleRefreshQuotaError\(err\)[\s\S]*buildQuotaPurchaseUrl\(QUOTA_TYPE_REFRESH\)/)
```

- [ ] **Step 2: 运行测试确认失败**

运行：`node --test wxapp/components/ResourcePublishForm.test.mjs wxapp/pages/my-resources/index.test.mjs`

预期：FAIL，因两处尚未处理 `QUOTA_NOT_ENOUGH`。

- [ ] **Step 3: 实现最小异常承接**

1. 在 `ResourcePublishForm.vue` 的 `submit()` 外层加入 `try/catch`。`handlePublishQuotaError` 对额度不足先确认，确认后跳转发布次数包；用户取消也返回 `true`，以保留已填写表单。其他错误继续抛出，保留请求层的原错误提示。
2. 在 `my-resources/index.vue` 的 `refresh(item)` 外层加入相同范围的 `try/catch`。`handleRefreshQuotaError` 确认后跳转刷新次数包，取消后留在列表，非额度错误继续抛出。
3. 在处理器附近写中文注释：只拦截额度不足，避免将资源过期、已完成或网络异常错误导向购买。

- [ ] **Step 4: 运行测试确认通过**

运行：`node --test wxapp/components/ResourcePublishForm.test.mjs wxapp/pages/my-resources/index.test.mjs`

预期：PASS。

- [ ] **Step 5: 提交本任务**

```bash
git add wxapp/components/ResourcePublishForm.vue wxapp/components/ResourcePublishForm.test.mjs wxapp/pages/my-resources/index.vue wxapp/pages/my-resources/index.test.mjs
git commit -m "feat: guide quota shortages to purchase"
```

### Task 5: 回归验证与范围检查

**文件：**

- 修改：仅修复前四项测试或实现缺陷；不新增功能。

**接口：**

- 验证 `publish_quota`、`refresh_quota`、`buildQuotaPurchaseUrl`、`confirmQuotaPurchase` 与 `QUOTA_NOT_ENOUGH` 在各页面保持一致。

- [ ] **Step 1: 执行前端专项测试**

```bash
node --test wxapp/common/entitlementPurchase.test.mjs wxapp/pages/my/index.test.mjs wxapp/pages/vip/index.test.mjs wxapp/components/ResourcePublishForm.test.mjs wxapp/pages/my-resources/index.test.mjs
```

预期：全部 PASS，且没有 Node 运行时警告。

- [ ] **Step 2: 执行页面与流程校验**

```bash
npm --prefix wxapp run test
npm --prefix wxapp run validate:pages
npm --prefix wxapp run validate:flows
```

预期：全部 PASS。

- [ ] **Step 3: 执行后端相关回归**

```bash
go test ./backend/app/internal/logic/resource ./backend/app/internal/logic/vip
```

预期：PASS，确认发布、刷新额度扣减和 VIP 下单逻辑未被改动。

- [ ] **Step 4: 检查改动范围**

```bash
git diff --check
git status --short
```

预期：没有空白错误，且变更仅为规格内权益购买相关文件。

- [ ] **Step 5: 提交本阶段修复**

```bash
git add wxapp/common/entitlementPurchase.js wxapp/common/entitlementPurchase.test.mjs wxapp/pages/my/index.vue wxapp/pages/my/index.test.mjs wxapp/pages/vip/index.vue wxapp/pages/vip/index.test.mjs wxapp/components/ResourcePublishForm.vue wxapp/components/ResourcePublishForm.test.mjs wxapp/pages/my-resources/index.vue wxapp/pages/my-resources/index.test.mjs
git commit -m "test: verify entitlement purchase closure"
```

只有工作区存在本阶段回归修复时才执行该提交；没有改动时不创建空提交。

## 自检结果

- 规格覆盖：Task 2 覆盖权益入口与分级展示；Task 3 覆盖购买页路由、真实促销标签和支付刷新；Task 4 覆盖发布/刷新额度不足；Task 5 覆盖回归。
- 范围控制：次数包价格、规格、后端订单轮询均不在本计划内。
- 命名一致：全程使用 `publish_quota`、`refresh_quota`、`buildQuotaPurchaseUrl`、`confirmQuotaPurchase` 与服务端 `QUOTA_NOT_ENOUGH`。
