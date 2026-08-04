# 微信小程序按钮网络请求加载反馈 Implementation Plan（实施计划）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为微信小程序所有需要等待服务端结果的前台按钮补齐原生 loading、禁用和防重复提交，并让后台统计类请求不阻塞用户跳转。

**Architecture:** loading 状态保留在最接近业务动作的页面或组件中，不修改公共 `request.js`。单按钮复用现有布尔状态，列表写操作使用“记录 ID + 动作名”，多段请求链由同一个状态覆盖并统一在 `finally` 恢复。

**Tech Stack:** Vue 3 `<script setup>`、uni-app 微信小程序、Node.js `node:test` 源码合同测试、Vite/uni 构建。

## Global Constraints

- 不修改后端接口、数据库、API 返回结构或公共请求协议。
- 不在 `wxapp/api/request.js` 中默认调用全局 `uni.showLoading`。
- 前台业务按钮必须同时具备原生 `loading`、`disabled`、进行时中文文案和处理函数忙碌短路。
- 确认型操作在用户确认后才进入 loading；校验失败、未登录和取消确认不进入 loading。
- loading 覆盖上传、业务请求、支付和结果刷新完整链路，并在 `finally` 恢复。
- 列表资源写操作同一时刻只允许一个，未操作记录保持可浏览但不可并行发起写操作。
- 曝光、浏览、分享、商家主页点击和消息已读等后台记录不显示 loading，也不延迟主流程。
- 错误提示保持中文、具体、可操作，不暴露原始服务端或支付内部信息。
- 只修改与加载反馈直接相关的代码，不引入 UI 组件库或全局状态库。

---

### Task 1: 为已有忙碌状态的按钮补齐原生动画

**Files:**
- Create: `wxapp/request-loading-contract.test.mjs`
- Modify: `wxapp/pages/login/index.vue`
- Modify: `wxapp/pages/account/settings.vue`
- Modify: `wxapp/pages/resource/report.vue`
- Modify: `wxapp/pages/vip/index.vue`
- Modify: `wxapp/pages/merchant/profile.vue`
- Modify: `wxapp/pages/merchant/map-binding.vue`
- Modify: `wxapp/pages/merchant/location.vue`
- Modify: `wxapp/pages/publish/index.vue`
- Modify: `wxapp/pages/search/index.vue`
- Modify: `wxapp/pages/sourcing-map/index.vue`
- Modify: `wxapp/pages/sourcing-map/legacy-canvas.vue`
- Modify: `wxapp/components/ResourceList.vue`

**Interfaces:**
- Consumes: 各页面已有的 `loggingIn`、`deleting`、`submitting`、`paying`、`payingPackCode`、`candidateLoading`、`loadingCategories`、`loading`、`objectLoading`、`reportSubmitting`。
- Produces: 所有显式等待按钮都使用 `:loading` 和同一个状态的 `:disabled`；旧版地图报告升级为 `reportAction: Ref<'location' | 'risk' | ''>`，只让当前反馈按钮旋转。

- [ ] **Step 1: 新增失败的统一合同测试**

创建 `wxapp/request-loading-contract.test.mjs`，读取目标 Vue 文件并精确验证已有状态被绑定到按钮：

```js
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('.', import.meta.url).pathname)
const read = (relativePath) => fs.readFileSync(path.join(root, relativePath), 'utf8')

test('buttons with existing foreground request state render native loading', () => {
  const contracts = [
    ['pages/login/index.vue', /<button[^>]*:loading="loggingIn"[^>]*@click="loginWithWechatAccount"/],
    ['pages/account/settings.vue', /<button[^>]*:loading="deleting"[^>]*@click="confirmDeleteAccount"/],
    ['pages/resource/report.vue', /<button[\s\S]*?:loading="submitting"[\s\S]*?@click="submitReport"/],
    ['pages/vip/index.vue', /<button[^>]*:loading="paying"[^>]*@click="openSelectedPlan"/],
    ['pages/vip/index.vue', /<button[^>]*:loading="payingPackCode === item\.code"[^>]*@click="openQuotaPack\(item\)"/],
    ['pages/merchant/profile.vue', /<button[^>]*:loading="submitting"[^>]*@click="submitMerchantProfile"/],
    ['pages/merchant/map-binding.vue', /<button[^>]*:loading="candidateLoading"[^>]*@click="searchCandidates"/],
    ['pages/merchant/map-binding.vue', /<button[^>]*:loading="submitting"[^>]*@click="submitBindingRequest"/],
    ['pages/merchant/location.vue', /<button[^>]*:loading="loading"[^>]*@click="loadLocationContext\(\)"/],
    ['pages/merchant/location.vue', /<button[^>]*:loading="loading"[^>]*@click="retryNearby"/],
    ['pages/publish/index.vue', /<button[^>]*:loading="loadingCategories"[^>]*@click="loadPublishCategories"/],
    ['pages/search/index.vue', /<button[^>]*:loading="loading"[^>]*@click="search"/],
    ['pages/sourcing-map/index.vue', /<button[^>]*:loading="loading"[^>]*@click="submitSearch"/],
    ['pages/sourcing-map/index.vue', /<button[^>]*:loading="loading"[^>]*@click="loadPlaces\(\{ reset: true \}\)"/],
    ['pages/sourcing-map/legacy-canvas.vue', /<button[^>]*:loading="objectLoading"[^>]*@click="submitSearch"/],
    ['pages/sourcing-map/legacy-canvas.vue', /<button[^>]*:loading="loading \|\| objectLoading"[^>]*@click="refreshCurrentMapData"/],
    ['pages/sourcing-map/legacy-canvas.vue', /<button[^>]*:loading="loading"[^>]*@click="loadScenes"/],
    ['pages/sourcing-map/legacy-canvas.vue', /<button[^>]*:loading="reportAction === 'location'"[^>]*@click="submitSelectedObjectLocationCorrection"/],
    ['pages/sourcing-map/legacy-canvas.vue', /<button[^>]*:loading="reportAction === 'risk'"[^>]*@click="submitSelectedObjectRiskReport"/],
    ['components/ResourceList.vue', /<button[^>]*:loading="loading"[^>]*@click="emit\('load-more'\)"/],
  ]
  for (const [file, pattern] of contracts) assert.match(read(file), pattern, file)
})

test('existing busy buttons show action-specific text', () => {
  assert.match(read('pages/merchant/profile.vue'), /submitting \? '保存中' :/)
  assert.match(read('pages/merchant/map-binding.vue'), /candidateLoading \? '搜索中' : '搜索'/)
  assert.match(read('pages/merchant/map-binding.vue'), /submitting \? '绑定中' : '确认绑定'/)
  assert.match(read('pages/publish/index.vue'), /loadingCategories \? '刷新中' : '刷新'/)
  assert.match(read('pages/sourcing-map/legacy-canvas.vue'), /const reportAction = ref\(''\)/)
  assert.match(read('pages/sourcing-map/legacy-canvas.vue'), /const reportSubmitting = computed\(\(\) => Boolean\(reportAction\.value\)\)/)
})

test('foreground handlers reject programmatic duplicate invocation', () => {
  assert.match(read('pages/login/index.vue'), /async function loginWithWechatAccount\(\) \{[\s\S]*if \(loggingIn\.value\) return/)
  assert.match(read('pages/merchant/profile.vue'), /async function submitMerchantProfile\(\) \{[\s\S]*if \(submitting\.value\) return/)
  assert.match(read('pages/merchant/map-binding.vue'), /async function searchCandidates\(\) \{[\s\S]*if \([^\n]*candidateLoading\.value[^\n]*\) return/)
  assert.match(read('pages/merchant/map-binding.vue'), /async function submitBindingRequest\(\) \{[\s\S]*if \(submitting\.value\) return/)
  assert.match(read('pages/vip/index.vue'), /async function openSelectedPlan\(\) \{[\s\S]*if \(paying\.value\) return/)
  assert.match(read('pages/vip/index.vue'), /async function openQuotaPack\(item\) \{[\s\S]*if \(payingPackCode\.value\) return/)
})
```

- [ ] **Step 2: 运行合同测试并确认按预期失败**

Run: `cd wxapp && node --test request-loading-contract.test.mjs`

Expected: FAIL，首个失败项指出登录等按钮缺少 `:loading`，而不是文件读取或正则语法错误。

- [ ] **Step 3: 为单一状态按钮绑定 loading 和进行时文案**

按以下模式修改每个模板，复用已有状态，不新增请求状态：

```vue
<button :disabled="loggingIn" :loading="loggingIn" @click="loginWithWechatAccount">
  {{ loggingIn ? '登录中' : '微信登录' }}
</button>

<button :disabled="submitting" :loading="submitting" @click="submitMerchantProfile">
  {{ submitting ? '保存中' : saveButtonText }}
</button>

<button :disabled="candidateLoading" :loading="candidateLoading" @click="searchCandidates">
  {{ candidateLoading ? '搜索中' : '搜索' }}
</button>
```

VIP 次数包使用 `:loading="payingPackCode === item.code"`；地图旧版的搜索、刷新、纠错和举报分别绑定 `objectLoading` 或 `reportSubmitting`；重试和加载更多按钮绑定其现有加载状态。筛选标签不添加独立 spinner。

在登录、资料保存、VIP 购买、档口搜索和绑定处理函数首行增加对应状态短路。旧版地图将 `reportSubmitting` 布尔 ref 改为派生状态：

```js
const reportAction = ref('')
const reportSubmitting = computed(() => Boolean(reportAction.value))
```

`submitSelectedObjectLocationCorrection()` 与 `submitSelectedObjectRiskReport()` 分别向 `openSelectedObjectReport` 传入 `action: 'location'`、`action: 'risk'`；用户在 action sheet 选中原因后设置 `reportAction.value = action`，并在 `finally` 清空。这样另一个反馈按钮只被禁用，不会同时旋转。

- [ ] **Step 4: 运行新增测试和相关页面测试**

Run: `cd wxapp && node --test request-loading-contract.test.mjs pages/login/index.test.mjs pages/resource/report.test.mjs pages/vip/index.test.mjs pages/merchant/map-binding.test.mjs pages/publish/index.test.mjs pages/search/index.test.mjs pages/sourcing-map/index.test.mjs components/ResourceList.test.mjs`

Expected: PASS，0 failed。

- [ ] **Step 5: 提交本批改动**

```bash
git add wxapp/request-loading-contract.test.mjs wxapp/pages/login/index.vue wxapp/pages/account/settings.vue wxapp/pages/resource/report.vue wxapp/pages/vip/index.vue wxapp/pages/merchant/profile.vue wxapp/pages/merchant/map-binding.vue wxapp/pages/merchant/location.vue wxapp/pages/publish/index.vue wxapp/pages/search/index.vue wxapp/pages/sourcing-map/index.vue wxapp/pages/sourcing-map/legacy-canvas.vue wxapp/components/ResourceList.vue
git commit -m "feat(wxapp): 补齐已有请求状态的按钮加载动画"
```

### Task 2: 防止发布和草稿保存重复提交

**Files:**
- Modify: `wxapp/components/ResourcePublishForm.test.mjs`
- Modify: `wxapp/components/ResourcePublishForm.vue`

**Interfaces:**
- Consumes: `submit()`、`saveDraft()`、`uploadPendingResourceImages()` 和现有发布 API。
- Produces: `publishAction: Ref<'submit' | 'draft' | ''>`、`publishBusy: ComputedRef<boolean>`、`isPublishAction(action)`，发布和草稿保存互斥并覆盖完整上传链路。

- [ ] **Step 1: 为发布互斥状态编写失败测试**

在 `ResourcePublishForm.test.mjs` 追加：

```js
test('resource publish actions show native loading and reject duplicate submission', () => {
  assert.match(source, /const publishAction = ref\(''\)/)
  assert.match(source, /const publishBusy = computed\(\(\) => Boolean\(publishAction\.value\)\)/)
  assert.match(source, /<button[^>]*:disabled="publishBusy"[^>]*:loading="isPublishAction\('draft'\)"[^>]*@click="saveDraft"/)
  assert.match(source, /<button[^>]*:disabled="publishBusy \|\| !canSubmit"[^>]*:loading="isPublishAction\('submit'\)"[^>]*@click="submit"/)
  assert.match(source, /async function submit\(\) \{[\s\S]*if \(publishBusy\.value\) return[\s\S]*publishAction\.value = 'submit'[\s\S]*finally \{[\s\S]*publishAction\.value = ''/)
  assert.match(source, /async function saveDraft\(\) \{[\s\S]*if \(publishBusy\.value\) return[\s\S]*publishAction\.value = 'draft'[\s\S]*finally \{[\s\S]*publishAction\.value = ''/)
})

test('resource publish form locks image mutations while publishing', () => {
  assert.match(source, /<button class="img-del" :disabled="publishBusy"/)
  assert.match(source, /function onResourceImageGridItemClick\(event\) \{[\s\S]*if \(publishBusy\.value\) return/)
  assert.match(source, /function removeResourceImage\(item\) \{[\s\S]*if \(publishBusy\.value\) return/)
})
```

- [ ] **Step 2: 确认测试因缺少发布状态而失败**

Run: `cd wxapp && node --test components/ResourcePublishForm.test.mjs`

Expected: FAIL，提示找不到 `publishAction` 或按钮 `:loading`。

- [ ] **Step 3: 增加互斥状态并覆盖完整请求链**

在现有 refs 附近加入：

```js
const publishAction = ref('')
const publishBusy = computed(() => Boolean(publishAction.value))
const isPublishAction = (action) => publishAction.value === action
```

`submit()` 和 `saveDraft()` 都先执行忙碌短路与本地校验，再设置动作状态。实现为：

```js
async function submit() {
  if (publishBusy.value) return
  if (!validatePublishForm()) return
  saveMerchantId(form.merchantId)
  publishAction.value = 'submit'
  try {
    if (editingResourceId.value) {
      const images = await uploadPendingResourceImages()
      if (!editSavedAsDraft.value || editingResourceStatus.value !== 'draft') {
        await saveResourceDraftPayload(images)
      }
      const resp = await submitResource(editingResourceId.value, form.merchantId)
      openPublishSuccess(resp)
      clearPublishLocalDraft()
      resetPublishForm()
      uni.showToast({ title: publishSubmitToast(resp), icon: 'none' })
      return
    }
    const images = await uploadPendingResourceImages()
    const resp = await createResource(buildResourcePublishPayload(images))
    openPublishSuccess(resp)
    clearPublishLocalDraft()
    resetPublishForm()
    uni.showToast({ title: publishSubmitToast(resp), icon: 'none' })
  } catch (err) {
    if (await handlePublishQuotaError(err)) return
    throw err
  } finally {
    publishAction.value = ''
  }
}

async function saveDraft() {
  if (publishBusy.value) return
  if (!validatePublishForm()) return
  saveMerchantId(form.merchantId)
  publishAction.value = 'draft'
  try {
    const merchantId = form.merchantId
    const images = await uploadPendingResourceImages()
    await saveResourceDraftPayload(images)
    clearPublishLocalDraft()
    resetPublishForm()
    uni.showToast({ title: '草稿已保存', icon: 'none' })
    uni.navigateTo({ url: `/pages/my-resources/index?merchantId=${merchantId}` })
  } finally {
    publishAction.value = ''
  }
}
```

模板将两个底部按钮互斥禁用，分别显示 `保存中`、`提交中`；图片删除和图片网格新增入口在 `publishBusy` 时直接返回。不要更改额度不足的购买引导和发布成功跳转。

- [ ] **Step 4: 运行发布表单测试**

Run: `cd wxapp && node --test components/ResourcePublishForm.test.mjs pages/publish/edit.test.mjs`

Expected: PASS，0 failed。

- [ ] **Step 5: 提交本批改动**

```bash
git add wxapp/components/ResourcePublishForm.vue wxapp/components/ResourcePublishForm.test.mjs
git commit -m "feat(wxapp): 防止发布表单重复提交"
```

### Task 3: 为“我的发布”增加资源级动作 loading

**Files:**
- Modify: `wxapp/pages/my-resources/index.test.mjs`
- Modify: `wxapp/pages/my-resources/index.vue`

**Interfaces:**
- Consumes: `refreshResource`、`listTopVouchers`、`redeemTopVoucher`、`takeDownResource`、`getOwnResource`、`deleteTakenDownResource` 及置顶支付链。
- Produces: `resourceAction: Ref<{ resourceId: string, action: string }>`、`resourceActionBusy`、`isResourceAction(item, action)`、`runResourceAction(item, action, operation)`。

- [ ] **Step 1: 编写资源级状态失败测试**

在 `my-resources/index.test.mjs` 追加：

```js
test('my resource writes expose per-action loading and share one write lock', () => {
  assert.match(source, /const resourceAction = ref\(\{ resourceId: '', action: '' \}\)/)
  assert.match(source, /const resourceActionBusy = computed\(\(\) => Boolean\(resourceAction\.value\.resourceId\)\)/)
  for (const [action, handler] of [
    ['refresh', 'refresh'],
    ['top', 'topResource'],
    ['takeDown', 'takeDown'],
    ['repost', 'repost'],
    ['delete', 'deleteTakenDown'],
  ]) {
    assert.match(source, new RegExp(`:loading="isResourceAction\\(item, '${action}'\\)"[\\s\\S]*?@click="${handler}\\(item\\)"`))
  }
  assert.match(source, /async function runResourceAction\(item, action, operation\) \{[\s\S]*if \(resourceActionBusy\.value\) return[\s\S]*await operation\(\)[\s\S]*finally[\s\S]*resourceAction\.value = \{ resourceId: '', action: '' \}/)
})
```

- [ ] **Step 2: 确认测试因旧的局部置顶状态而失败**

Run: `cd wxapp && node --test pages/my-resources/index.test.mjs`

Expected: FAIL，提示缺少 `resourceAction` 或刷新按钮 loading。

- [ ] **Step 3: 实现资源动作执行器和按钮状态**

将 `purchasingTopResourceId` 替换为：

```js
const resourceAction = ref({ resourceId: '', action: '' })
const resourceActionBusy = computed(() => Boolean(resourceAction.value.resourceId))

function isResourceAction(item, action) {
  return resourceAction.value.resourceId === item.id && resourceAction.value.action === action
}

async function runResourceAction(item, action, operation) {
  if (resourceActionBusy.value) return
  resourceAction.value = { resourceId: item.id, action }
  try {
    return await operation()
  } finally {
    resourceAction.value = { resourceId: '', action: '' }
  }
}
```

刷新、置顶、下架、再发类似在本地前置条件通过后调用 `runResourceAction`。删除必须先等待用户确认，再调用执行器。每个动作按钮绑定当前动作 loading，所有写按钮绑定 `:disabled="resourceActionBusy"`，当前按钮文案分别显示 `刷新中`、`置顶中`、`下架中`、`准备中`、`删除中`。loading 覆盖操作后的 `loadRows({ reset: true })`。

- [ ] **Step 4: 运行我的发布测试**

Run: `cd wxapp && node --test pages/my-resources/index.test.mjs common/entitlementPurchase.test.mjs`

Expected: PASS，0 failed。

- [ ] **Step 5: 提交本批改动**

```bash
git add wxapp/pages/my-resources/index.vue wxapp/pages/my-resources/index.test.mjs
git commit -m "feat(wxapp): 增加发布资源动作加载状态"
```

### Task 4: 为供需详情的管理、收藏和联系方式增加可见状态

**Files:**
- Modify: `wxapp/pages/resource/detail.test.mjs`
- Modify: `wxapp/pages/resource/detail.vue`

**Interfaces:**
- Consumes: 现有 `managementActions`、`toggleFavorite()`、`recordContact()`、管理 API 和支付链。
- Produces: `managementAction: Ref<string>`、`favoriteBusy: Ref<boolean>`、`contactAction: Ref<'phone' | 'wechat' | ''>`、`managementActionLabel(action)`。

- [ ] **Step 1: 编写详情页忙碌反馈失败测试**

在 `resource/detail.test.mjs` 追加：

```js
test('resource detail exposes management, favorite, and contact loading', () => {
  assert.match(source, /const managementAction = ref\(''\)/)
  assert.match(source, /const managementBusy = computed\(\(\) => Boolean\(managementAction\.value\)\)/)
  assert.match(source, /:loading="managementAction === action\.key"[\s\S]*:disabled="managementBusy"[\s\S]*@click="handleManagementAction\(action\.key\)"/)
  assert.match(source, /<text class="action-label">\{\{ managementActionLabel\(action\) \}\}<\/text>/)
  assert.match(source, /:loading="favoriteBusy"[\s\S]*@click="favoriteResourceFromMore"/)
  assert.match(source, /:loading="contactAction === 'wechat'"[\s\S]*@click="copyWechat"/)
  assert.match(source, /:loading="contactAction === 'phone'"[\s\S]*@click="callPhone"/)
})

test('resource detail resets every foreground action in finally', () => {
  assert.match(source, /async function toggleFavorite\(\) \{[\s\S]*if \(favoriteBusy\.value\) return false[\s\S]*favoriteBusy\.value = true[\s\S]*finally \{[\s\S]*favoriteBusy\.value = false/)
  assert.match(source, /async function runContactAction\(action, operation\) \{[\s\S]*if \(contactAction\.value\) return[\s\S]*finally \{[\s\S]*contactAction\.value = ''/)
  assert.match(source, /async function handleManagementAction\(action\) \{[\s\S]*managementAction\.value = action[\s\S]*finally \{[\s\S]*managementAction\.value = ''/)
})
```

- [ ] **Step 2: 确认测试因状态不可见而失败**

Run: `cd wxapp && node --test pages/resource/detail.test.mjs`

Expected: FAIL，提示缺少 `managementAction`、`favoriteBusy` 或联系方式 loading。

- [ ] **Step 3: 将管理布尔值扩展为动作名**

用以下状态替换原 `managementBusy = ref(false)`：

```js
const managementAction = ref('')
const managementBusy = computed(() => Boolean(managementAction.value))
const favoriteBusy = ref(false)
const contactAction = ref('')
```

编辑动作是纯跳转，不进入忙碌状态；其他管理动作设置 `managementAction.value = action` 并在 `finally` 清空。`managementActionLabel` 根据动作返回 `刷新中`、`置顶中`、`下架中`、`准备中`、`删除中`，空闲时返回 `action.label`。

收藏在登录/自有资源检查后设置 `favoriteBusy`，并在 `finally` 恢复。电话和微信通过以下执行器互斥：

```js
async function runContactAction(action, operation) {
  if (contactAction.value) return
  contactAction.value = action
  try {
    return await operation()
  } finally {
    contactAction.value = ''
  }
}
```

按钮显示 `查询中` 或 `解锁中`。置顶购买流程在订单、支付和刷新完成后再关闭管理面板，保证支付前后始终存在可见反馈；保留微信原生支付弹窗。

- [ ] **Step 4: 运行详情页及支付辅助测试**

Run: `cd wxapp && node --test pages/resource/detail.test.mjs common/resourceDetailState.test.mjs common/resourceDetailAuxiliary.test.mjs common/entitlementPurchase.test.mjs`

Expected: PASS，0 failed。

- [ ] **Step 5: 提交本批改动**

```bash
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "feat(wxapp): 增加供需详情操作加载状态"
```

### Task 5: 补齐关注 loading，并让后台记录不阻塞跳转

**Files:**
- Modify: `wxapp/pages/merchant/detail.test.mjs`
- Modify: `wxapp/pages/merchant/detail.vue`
- Modify: `wxapp/pages/messages/index.test.mjs`
- Modify: `wxapp/pages/messages/index.vue`
- Modify: `wxapp/pages/resource/detail.test.mjs`
- Modify: `wxapp/pages/resource/detail.vue`

**Interfaces:**
- Consumes: `setMerchantFollow`、`readMessage`、`recordContact('merchant_home' | 'share')`。
- Produces: `followBusy: Ref<boolean>`；消息已读、商家主页点击和分享统计改为 best-effort 后台任务，主流程立即跳转或关闭面板。

- [ ] **Step 1: 编写关注和非阻塞上报失败测试**

追加以下合同：

```js
test('merchant follow button shows loading and rejects duplicate clicks', () => {
  const source = fs.readFileSync(sourcePath, 'utf8')
  assert.match(source, /const followBusy = ref\(false\)/)
  assert.match(source, /<button[^>]*:disabled="followBusy"[^>]*:loading="followBusy"[^>]*@click="toggleFollow"/)
  assert.match(source, /async function toggleFollow\(\) \{[\s\S]*if \(followBusy\.value\) return[\s\S]*followBusy\.value = true[\s\S]*finally \{[\s\S]*followBusy\.value = false/)
})
```

在 `messages/index.test.mjs` 验证跳转不再等待已读请求：

```js
test('opening a message does not wait for read receipt reporting', () => {
  assert.match(source, /void markRead\(item\)\.catch\(\(err\) => \{[\s\S]*消息已读状态更新失败/)
  assert.doesNotMatch(source, /await markRead\(item\)/)
})
```

在 `resource/detail.test.mjs` 验证 `openMerchant` 和分享统计不再 `await recordContact`，导航及关闭面板先执行。

- [ ] **Step 2: 确认测试因缺少关注状态和后台化处理而失败**

Run: `cd wxapp && node --test pages/merchant/detail.test.mjs pages/messages/index.test.mjs pages/resource/detail.test.mjs`

Expected: FAIL，提示缺少 `followBusy` 且仍存在 `await markRead(item)`。

- [ ] **Step 3: 实现关注状态与 best-effort 上报**

关注函数采用标准 `try/catch/finally`：

```js
async function toggleFollow() {
  if (!merchant.value.id || followBusy.value) return
  followBusy.value = true
  try {
    const resp = await setMerchantFollow(merchant.value.id, !followed.value)
    followed.value = Boolean(resp.followed)
    uni.showToast({ title: followed.value ? '已关注' : '已取消关注', icon: 'none' })
  } catch (err) {
    uni.showToast({ title: err.message || '操作失败，请稍后重试', icon: 'none' })
  } finally {
    followBusy.value = false
  }
}
```

消息点击使用 `void markRead(item).catch(...)` 后立即执行原跳转。资源详情进入商家主页时先触发不等待的 `recordContact('merchant_home')`，立即 `navigateTo`；分享面板立即关闭，统计失败由 `recordContact` 现有容错吞掉，不新增 loading。

- [ ] **Step 4: 运行相关页面测试**

Run: `cd wxapp && node --test pages/merchant/detail.test.mjs pages/messages/index.test.mjs pages/resource/detail.test.mjs`

Expected: PASS，0 failed。

- [ ] **Step 5: 提交本批改动**

```bash
git add wxapp/pages/merchant/detail.vue wxapp/pages/merchant/detail.test.mjs wxapp/pages/messages/index.vue wxapp/pages/messages/index.test.mjs wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "feat(wxapp): 完善关注反馈和后台上报体验"
```

### Task 6: 全量审计、测试和微信小程序构建

**Files:**
- Test: `wxapp/request-loading-contract.test.mjs`
- Test: Tasks 1-5 中修改的页面与组件测试

**Interfaces:**
- Consumes: Tasks 1-5 的所有按钮状态和测试合同。
- Produces: 可重复执行的全量测试、页面流程校验和微信小程序构建证据。

- [ ] **Step 1: 执行静态按钮与请求入口复核**

Run:

```bash
rg -n '@(click|tap)|@submit|form-type="submit"' wxapp/pages wxapp/components --glob '*.vue'
rg -n ':loading=|showLoading|加载中|提交中|保存中|搜索中|刷新中|关注中|收藏中|置顶中|下架中|删除中|购买中' wxapp/pages wxapp/components --glob '*.vue'
```

Expected: 所有前台业务请求入口均能对应到按钮级或页面级反馈；曝光、浏览、分享、已读等后台记录没有阻塞 loading。

- [ ] **Step 2: 对照设计规格逐项确认审计结果**

逐项确认发布/保存、资源管理、关注/收藏、联系方式、支付、地图绑定、搜索/重试均出现在两组审计结果中；确认曝光、浏览、分享、已读上报没有 `:loading`。任一项不满足时停止最终验证，回到对应的 Task 1-5，先补失败测试，再修复实现。

- [ ] **Step 3: 运行全部 wxapp 测试**

Run: `cd wxapp && npm test`

Expected: 所有 Node 测试 PASS，0 failed。

- [ ] **Step 4: 运行页面和业务流程校验**

Run: `cd wxapp && npm run validate:pages && npm run validate:flows`

Expected: 两个命令均 exit 0，无缺失页面、路由或流程合同错误。

- [ ] **Step 5: 构建微信小程序**

Run: `cd wxapp && npm run build:mp-weixin`

Expected: exit 0，生成 `dist/mp-weixin`，无编译错误。

- [ ] **Step 6: 检查差异和空白错误**

Run: `git diff --check && git status --short && git diff --stat HEAD~5`

Expected: `git diff --check` exit 0；状态中只包含本计划直接相关文件，或工作区干净；差异范围不含后端、API 合同或无关视觉重构。
