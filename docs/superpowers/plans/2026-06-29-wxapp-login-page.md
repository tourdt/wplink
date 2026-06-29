# wxapp 独立登录页 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增可复用登录页，让需要登录的页面统一跳转并在登录后回到原页面。

**Architecture:** `pages/login/index.vue` 负责微信登录、保存会话和 redirect 回跳；`common/auth.js` 负责判断登录态、构造登录 URL 和统一跳转登录页；“我的”页复用该工具，不再内嵌微信登录接口调用。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、Node.js `node:test`、现有 `wxapp/scripts/validate-pages.mjs` 和 `wxapp/scripts/validate-flows.mjs`。

---

### Task 1: 登录页和登录工具验证

**Files:**
- Modify: `wxapp/scripts/validate-pages.mjs`
- Modify: `wxapp/scripts/validate-flows.mjs`
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: Write failing validation test**

Append this test to `wxapp/scripts/validate-flows.test.mjs`:

```js
test('login page provides reusable wechat login and redirect guard', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const loginSource = fs.existsSync(path.join(root, 'pages/login/index.vue'))
    ? fs.readFileSync(path.join(root, 'pages/login/index.vue'), 'utf8')
    : ''
  const authSource = fs.existsSync(path.join(root, 'common/auth.js'))
    ? fs.readFileSync(path.join(root, 'common/auth.js'), 'utf8')
    : ''
  const mySource = fs.readFileSync(path.join(root, 'pages/my/index.vue'), 'utf8')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
  const pagePaths = (pagesConfig.pages || []).map((item) => item.path)

  assert.equal(pagePaths.includes('pages/login/index'), true)

  for (const token of ['wechatLogin', 'saveToken', 'saveUserId', 'redirectUrl', 'goAfterLogin', 'DEFAULT_CITY_CODE']) {
    assert.match(loginSource, new RegExp(token))
  }

  for (const token of ['requireLogin', 'buildLoginUrl', 'getCurrentPageUrl', 'encodeURIComponent', 'getSession']) {
    assert.match(authSource, new RegExp(token))
  }

  assert.match(mySource, /buildLoginUrl/)
  assert.equal(mySource.includes("import { bindPhone, sendSmsCode, wechatLogin }"), false)
})
```

- [ ] **Step 2: Run test and confirm red**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL because `pages/login/index.vue` and `common/auth.js` do not exist yet.

- [ ] **Step 3: Update validators for the new login route**

In `wxapp/scripts/validate-pages.mjs`, add `pages/login/index` to `requiredPages`.

In `wxapp/scripts/validate-flows.mjs`, add checks for `common/auth.js` and `pages/login/index.vue`, and update `pages/my/index.vue` checks to include `buildLoginUrl` / `openLogin`.

- [ ] **Step 4: Run validation and keep red**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL until implementation files are added.

### Task 2: 实现通用登录工具和登录页

**Files:**
- Create: `wxapp/common/auth.js`
- Create: `wxapp/pages/login/index.vue`
- Modify: `wxapp/pages.json`

- [ ] **Step 1: Create `wxapp/common/auth.js`**

Implement `isLoggedIn()`、`getCurrentPageUrl()`、`buildLoginUrl()`、`requireLogin()` using `getSession()` and `uni.navigateTo()`.

- [ ] **Step 2: Create `wxapp/pages/login/index.vue`**

Implement a compact login page with title “衣货通”、copy “登录后同步收藏、需求、消息和发布记录”、button “微信登录”、wechat login call, token/user ID persistence, and redirect handling via `uni.switchTab` or `uni.redirectTo`.

- [ ] **Step 3: Register login route**

Add `pages/login/index` to `wxapp/pages.json` with `navigationBarTitleText` set to “登录”.

### Task 3: 我的页接入登录页

**Files:**
- Modify: `wxapp/pages/my/index.vue`

- [ ] **Step 1: Replace inline login with login-page navigation**

Remove direct `wechatLogin` / `DEFAULT_CITY_CODE` imports and local login code from “我的”页. Import `buildLoginUrl` and `requireLogin` from `../../common/auth`.

- [ ] **Step 2: Update button behavior**

Make the unauthenticated “微信登录” button call `openLogin()`, which navigates to `buildLoginUrl('/pages/my/index')`.

- [ ] **Step 3: Keep entry guards**

Use imported `requireLogin()` for “我的需求”、收藏关注、消息、发布资源.

### Task 4: 验证

**Files:**
- Test: `wxapp/scripts/validate-flows.test.mjs`
- Test: `wxapp/scripts/validate-flows.mjs`
- Test: `wxapp/scripts/validate-pages.mjs`

- [ ] **Step 1: Run flow tests**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: PASS.

- [ ] **Step 2: Run flow validator**

Run: `cd wxapp && npm run validate:flows`

Expected: `wxapp flows ok`.

- [ ] **Step 3: Run page validator**

Run: `cd wxapp && npm run validate:pages`

Expected: `wxapp pages ok`.
