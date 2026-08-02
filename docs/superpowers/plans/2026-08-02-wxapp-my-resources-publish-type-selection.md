# 我的发布新建发布类型选择实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让“我的发布”的新建入口先进入发布类型选择页，再打开与所选类型匹配的发布表单。

**Architecture:** 复用 `pages/publish/index.vue` 已有的 tab 页和类型选择流程。“我的发布”只在商家资料校验成功后切换至该 tab 页；类型页继续负责加载类型、选择具体类型与携带 `typeCode` 打开编辑页。编辑草稿、驳回内容和再发类似仍保留直接进入编辑页的既有路径。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、Node.js 内置 `node:test`、正则源文件结构测试。

## 全局约束

- 仅修改新建发布入口及其直接关联的校验脚本、测试；不得改变编辑和再发类似的路由。
- 新建入口必须在 `ensurePageMerchantProfile()` 成功后才切换至 `pages/publish/index`。
- 复用现有发布类型页，不增加新的类型接口、状态存储或弹层。
- 保持小程序页面可构建，并运行相关测试、页面校验和流程校验。

---

## 文件结构

- 修改 `wxapp/pages/my-resources/index.vue`：将仅限新建发布的 `openPublish` 跳转目标改为发布 tab 页。
- 修改 `wxapp/pages/my-resources/index.test.mjs`：覆盖商家资料校验后切换发布类型页，防止回归为直接进入编辑页。
- 修改 `wxapp/scripts/validate-flows.mjs`：将“我的发布”流程校验点更新为发布类型页切换。
- 修改 `wxapp/scripts/validate-flows.test.mjs`：验证流程校验脚本包含新的新建发布入口约束。

### Task 1: 将新建发布入口接入现有类型选择页

**Files:**

- Modify: `wxapp/pages/my-resources/index.vue:508-511`
- Modify: `wxapp/pages/my-resources/index.test.mjs:104-114`
- Modify: `wxapp/scripts/validate-flows.mjs:145-147`
- Modify: `wxapp/scripts/validate-flows.test.mjs:700-740`

**Interfaces:**

- Consumes: `ensurePageMerchantProfile(): Promise<boolean>`，成功时已同步当前可管理的商家 ID。
- Consumes: `uni.switchTab({ url: '/pages/publish/index' })`，切换到已配置的发布 tab 页。
- Produces: `openPublish()` 在新建发布时进入类型选择流程；`openDraftEditor`、`openRejectedEditor`、`repost` 的 `uni.navigateTo('/pages/publish/edit?...')` 保持不变。

- [ ] **Step 1: 写入失败测试，定义新建入口必须先选类型**

  在 `wxapp/pages/my-resources/index.test.mjs` 中替换“新建发布直接打开独立编辑页”的断言，使用以下测试：

  ```js
  test('my resources publish action opens the publish type selection tab', () => {
    assert.match(source, /async function openPublish\(\) \{[\s\S]*if \(!\(await ensurePageMerchantProfile\(\)\)\) return[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)[\s\S]*\}/)
    assert.doesNotMatch(source, /async function openPublish\(\) \{[\s\S]*uni\.navigateTo\(\{ url: `\/pages\/publish\/edit\?merchantId=\$\{merchantId\.value\}` \}\)/)
  })
  ```

  同时更新资料校验测试的 `openPublish` 断言：保留 `ensurePageMerchantProfile()` 的断言，并把编辑页 `uni.navigateTo` 改为上面的 `uni.switchTab` 调用。

- [ ] **Step 2: 运行测试，确认因旧跳转实现失败**

  Run: `node --experimental-vm-modules --test pages/my-resources/index.test.mjs`

  Expected: FAIL，失败信息显示 `openPublish` 未调用 `uni.switchTab({ url: '/pages/publish/index' })`。

- [ ] **Step 3: 用最小改动切换新建入口**

  在 `wxapp/pages/my-resources/index.vue` 中仅替换 `openPublish` 的成功分支：

  ```js
  async function openPublish() {
    if (!(await ensurePageMerchantProfile())) return
    uni.switchTab({ url: '/pages/publish/index' })
  }
  ```

  不改动 `openDraftEditor`、`openRejectedEditor`、`repost` 与其他 `uni.navigateTo` 调用。

- [ ] **Step 4: 更新流程静态校验及其测试**

  在 `wxapp/scripts/validate-flows.mjs` 的“我的发布管理动作和指标”检查项中，保留编辑、详情等现有检查，新增以下两个字符串检查项：

  ```js
  'uni.switchTab',
  "url: '/pages/publish/index'",
  ```

  在 `wxapp/scripts/validate-flows.test.mjs` 的 `publish pages split tab creation and independent editing` 测试中读取 `pages/my-resources/index.vue`，并追加：

  ```js
  const myResourcesSource = fs.readFileSync(path.join(root, 'pages/my-resources/index.vue'), 'utf8')
  assert.match(myResourcesSource, /async function openPublish\(\) \{[\s\S]*ensurePageMerchantProfile\(\)[\s\S]*uni\.switchTab\(\{ url: '\/pages\/publish\/index' \}\)/)
  ```

- [ ] **Step 5: 运行相关测试，确认通过**

  Run: `node --experimental-vm-modules --test pages/my-resources/index.test.mjs scripts/validate-flows.test.mjs`

  Expected: PASS，且测试确认新建入口进入类型选择 tab，编辑与再发类似的编辑页路径仍存在。

- [ ] **Step 6: 运行页面与流程校验**

  Run: `npm run validate:pages && npm run validate:flows`

  Expected: 两个命令均以退出码 `0` 完成；流程校验输出中不报告 `pages/my-resources/index.vue` 缺少检查项。

- [ ] **Step 7: 提交功能改动**

  ```bash
  git add wxapp/pages/my-resources/index.vue wxapp/pages/my-resources/index.test.mjs wxapp/scripts/validate-flows.mjs wxapp/scripts/validate-flows.test.mjs
  git commit -m "feat: 我的发布先选择发布类型"
  ```

## 计划自审

- 规格覆盖：Task 1 覆盖新建入口、资料校验顺序、复用类型页、编辑/再发流程不变，以及自动化与脚本校验。
- 占位符检查：计划不含待补充步骤；每个测试、实现和验证命令均已明确。
- 接口一致性：`openPublish` 使用现有的 `ensurePageMerchantProfile` 和 uni-app `switchTab`；发布 tab 页仍使用现有 `typeCode` 路由进入编辑页，无新增接口。
