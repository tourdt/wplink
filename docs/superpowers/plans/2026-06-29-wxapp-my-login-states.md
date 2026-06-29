# wxapp 我的页登录态展示 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `wxapp/pages/my/index.vue` 按未登录和已登录两种状态展示账号中心，并暂时隐藏商家绑定相关能力。

**Architecture:** 页面只通过本地 `token` 判断登录态。测试脚本用源码级断言锁定关键文案、入口和不应出现的商家能力；页面实现保留微信登录、短信验证码、手机号绑定和常用入口跳转。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、Node.js `node:test`、现有 `wxapp/scripts/validate-flows.mjs` 验证脚本。

---

### Task 1: 更新登录态源码验证

**Files:**
- Modify: `wxapp/scripts/validate-flows.mjs`
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: Write the failing test**

在 `wxapp/scripts/validate-flows.test.mjs` 追加测试，读取 `pages/my/index.vue` 并断言新登录态结构存在、商家绑定相关可见文案不存在：

```js
test('my page separates guest and logged-in account states without merchant binding', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const source = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')

  for (const token of [
    'isLoggedIn',
    '未登录',
    '登录后管理需求、收藏和发布记录',
    '微信登录',
    '我的账号',
    '已登录，可管理需求、收藏和消息',
    '手机号绑定',
    'openMessages',
    'requireLogin',
  ]) {
    assert.match(source, new RegExp(token))
  }

  for (const hiddenToken of ['保存身份', '商家 ID', '我的权益', '商家认证', '权益提醒']) {
    assert.equal(source.includes(hiddenToken), false)
  }
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL because the current page still contains `保存身份`、`商家 ID`、`我的权益` and does not contain `isLoggedIn` / `requireLogin` / the new account-state copy.

- [ ] **Step 3: Update flow checks for the new my page contract**

In `wxapp/scripts/validate-flows.mjs`, replace the `pages/my/index.vue` flow checks with:

```js
{
  file: 'pages/my/index.vue',
  description: '我的页登录态和核心入口',
  checks: [
    'isLoggedIn',
    'wechatLogin',
    'sendSmsCode',
    'bindPhone',
    'saveUserId',
    'requireLogin',
    'openMessages',
    'openMyDemands',
    'openFavorites',
    'openPublish',
    '未登录',
    '登录后管理需求、收藏和发布记录',
    '我的账号',
    '已登录，可管理需求、收藏和消息',
    '手机号绑定',
  ],
}
```

- [ ] **Step 4: Run test to keep it red until page implementation**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL because the new checks require page code that has not been implemented yet.

### Task 2: 实现我的页两态展示

**Files:**
- Modify: `wxapp/pages/my/index.vue`

- [ ] **Step 1: Replace template with two-state account UI**

Use:

```vue
<template>
  <view class="my-page">
    <view class="account-card">
      <view class="avatar" :class="{ 'avatar-guest': !isLoggedIn }">{{ avatarText }}</view>
      <view class="account-main">
        <text class="account-name">{{ accountName }}</text>
        <text class="account-desc">{{ accountDesc }}</text>
        <text v-if="isLoggedIn" class="account-meta">{{ accountMeta }}</text>
      </view>
      <button v-if="!isLoggedIn" class="login-button" :disabled="loggingIn" @click="loginWithWechat">
        {{ loggingIn ? '登录中' : '微信登录' }}
      </button>
    </view>

    <view v-if="!isLoggedIn" class="guest-card">
      <text class="section-title">登录后可用</text>
      <view class="benefit-list">
        <text>同步收藏关注</text>
        <text>查看我的需求</text>
        <text>接收审核和联系消息</text>
      </view>
    </view>

    <view v-else class="profile-card">
      <text class="section-title">手机号绑定</text>
      <view class="sms-row">
        <input v-model="phone" class="field" type="number" placeholder="手机号" />
        <button class="sms-button" :disabled="smsSending || smsCountdown > 0" @click="sendSmsCodeForPhone">
          {{ smsCountdown > 0 ? `${smsCountdown}s` : '验证码' }}
        </button>
      </view>
      <input v-model="smsCode" class="field" type="number" placeholder="短信验证码" />
      <button class="secondary-button" :disabled="bindingPhone" @click="bindCurrentPhone">绑定手机号</button>
    </view>

    <view class="action-list">
      <view class="action-item" @click="openMyDemands">
        <text>我的需求</text>
        <text class="action-meta">采购需求和处理进展</text>
      </view>
      <view class="action-item" @click="openFavorites">
        <text>收藏关注</text>
        <text class="action-meta">收藏资源、关注商家和保存搜索</text>
      </view>
      <view class="action-item" @click="openMessages">
        <text>消息</text>
        <text class="action-meta">审核、联系和系统提醒</text>
      </view>
      <view class="action-item" @click="openPublish">
        <text>发布资源</text>
        <text class="action-meta">新增库存、货源、工厂或服务</text>
      </view>
    </view>
  </view>
</template>
```

- [ ] **Step 2: Simplify script to token-based login state**

Keep imports for `computed`, `ref`, `onLoad`, `onUnload`, `DEFAULT_CITY_CODE`, `bindPhone`, `sendSmsCode`, `wechatLogin`, `getSession`, `saveToken`, `saveUserId`. Remove entitlement and merchant imports. Implement:

```js
const token = ref('')
const userId = ref('')
const loggingIn = ref(false)
const isLoggedIn = computed(() => Boolean(token.value))
const avatarText = computed(() => (isLoggedIn.value ? '我' : '游'))
const accountName = computed(() => (isLoggedIn.value ? '我的账号' : '未登录'))
const accountDesc = computed(() => (isLoggedIn.value ? '已登录，可管理需求、收藏和消息' : '登录后管理需求、收藏和发布记录'))
const accountMeta = computed(() => (userId.value ? `用户 ID：${userId.value}` : '账号信息已同步'))
```

In `onLoad`, read `getSession()` and fill `token` / `userId`. In `loginWithWechat`, set `loggingIn` during request and update `token` after `saveToken`. Add:

```js
function requireLogin() {
  if (isLoggedIn.value) return true
  uni.showToast({ title: '请先登录', icon: 'none' })
  return false
}
```

Guard `openMyDemands`、`openFavorites`、`openMessages`、`openPublish` with `requireLogin()`.

- [ ] **Step 3: Replace styles for the simplified cards**

Remove unused entitlement, merchant and benefit styles. Keep page background, account card, guest card, profile card, action list, buttons, input and SMS row styles using existing `#0f766e` main color.

- [ ] **Step 4: Run tests to verify green**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: PASS.

- [ ] **Step 5: Run existing flow validator**

Run: `cd wxapp && npm run validate:flows`

Expected: `wxapp flows ok`.
