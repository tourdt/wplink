# wxapp Publish Page Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 wxapp 发布页优化为高效、分区明确、可回归验证的商家发布工具页。

**Architecture:** 只修改 `wxapp/pages/publish/index.vue` 的模板、少量计算属性和局部样式，保留既有接口函数。用 `wxapp/scripts/validate-flows.test.mjs` 增加静态结构测试，配合现有页面校验脚本验证路由和依赖。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、SCSS、Node.js `node:test`。

---

### Task 1: 发布页结构测试

**Files:**
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: 写失败测试**

在 `validate-flows.test.mjs` 增加 `publish page presents grouped fast publishing workflow`，检查发布页包含 `publish-hero`、`completion-percent`、`form-section basic-section`、`image-preview-grid`、`sticky-action-bar` 等结构。

- [ ] **Step 2: 运行测试确认失败**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL，提示当前发布页缺少新结构 token。

### Task 2: 重做发布页模板和状态提示

**Files:**
- Modify: `wxapp/pages/publish/index.vue`

- [ ] **Step 1: 添加计算属性**

新增 `requiredFields`、`completedRequiredCount`、`completionPercent`，只用于显示必填完成度，不改变 `validatePublishForm()`。

- [ ] **Step 1.1: 加载商户联系人默认值**

发布页通过 `getMerchant(form.merchantId)` 获取商户资料，调用 `applyMerchantContactDefaults(detail.contact || {})`。默认填入 `contact.name`；联系电话仅在存在完整 `contact.phone` 时填入，不使用脱敏手机号。

- [ ] **Step 2: 调整模板结构**

将原单卡片表单改为基础信息内紧凑完成度、资源类型选择、多个 `form-section` 和参考商家资料页的全宽固定操作栏。

- [ ] **Step 3: 图片预览操作**

新增资源图片网格计算项、`onResourceImageGridItemClick()`、`previewResourceImage(item)` 和 `removeResourceImage(item)`，配合商家资料页同款网格添加、预览与删除。

### Task 3: 补齐样式并验证

**Files:**
- Modify: `wxapp/pages/publish/index.vue`

- [ ] **Step 1: 重写局部样式**

使用现有 `uni.scss` token，保持深蓝黑主色、橙红强调色、浅蓝灰背景和 12rpx 卡片圆角。

- [ ] **Step 2: 运行验证**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: PASS。

Run: `npm --prefix wxapp run validate:pages`

Expected: PASS，输出 `wxapp pages ok`。
