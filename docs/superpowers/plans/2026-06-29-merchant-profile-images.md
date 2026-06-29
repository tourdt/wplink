# 商家主页图片编辑优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将商家资料页的“商家主页图片”改为参考笔记编辑页的 `uni-grid` 网格添加、预览、删除体验，并在保存上传前压缩本地图片。

**Architecture:** 后端仍接收 `images: string[]`，页面只改变前端编辑体验。新增 `wxapp/common/merchantProfileImages.js` 承载图片队列和压缩策略，便于 Node 测试；页面负责调用 `uni.chooseMedia/uni.chooseImage`、预览、上传和保存。

**Tech Stack:** uni-app、Vue 3 `<script setup>`、本地 `uni-grid`/`uni-grid-item` 组件、Node 内置测试脚本。

---

### Task 1: 图片队列与压缩策略工具

**Files:**
- Create: `wxapp/common/merchantProfileImages.js`
- Create: `wxapp/common/merchantProfileImages.test.mjs`

- [ ] **Step 1: Write the failing test**

```javascript
import assert from 'node:assert/strict'
import {
  appendMerchantImageFiles,
  createStoredMerchantImageEntry,
  createPendingMerchantImageEntry,
  getMerchantImagePreviewUrl,
  getMerchantImageUrlsForPreview,
  removeMerchantImageEntry,
  resolveImageCompressionOptions,
} from './merchantProfileImages.js'

const stored = createStoredMerchantImageEntry('https://cdn.test/a.jpg')
const pending = createPendingMerchantImageEntry({ id: 'local-1', path: '/tmp/b.jpg', size: 600_000 })
assert.equal(getMerchantImagePreviewUrl(stored), 'https://cdn.test/a.jpg')
assert.equal(getMerchantImagePreviewUrl(pending), '/tmp/b.jpg')
assert.deepEqual(getMerchantImageUrlsForPreview([stored, pending]), ['https://cdn.test/a.jpg', '/tmp/b.jpg'])
assert.deepEqual(removeMerchantImageEntry([stored, pending], pending.id), [stored])
assert.equal(appendMerchantImageFiles([stored], [{ id: 'local-2', path: '/tmp/c.jpg' }], 2).length, 2)
assert.deepEqual(resolveImageCompressionOptions({ width: 1600, height: 900, size: 600_000 }), {
  shouldCompress: true,
  compressedWidth: 1280,
  quality: 85,
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node wxapp/common/merchantProfileImages.test.mjs`
Expected: FAIL with module not found or missing exports.

- [ ] **Step 3: Write minimal implementation**

实现 entry 生成、追加上限、删除、预览 URL 提取，以及参考页的压缩策略：宽度超过 1280 时压到 1280；大于 3MB 质量 75，大于 2MB 质量 80，大于 500KB 质量 85，否则仅按宽度决定是否压缩。

- [ ] **Step 4: Run test to verify it passes**

Run: `node wxapp/common/merchantProfileImages.test.mjs`
Expected: PASS.

### Task 2: 引入 uni-grid 组件

**Files:**
- Create: `wxapp/components/uni-ui/uni-grid/uni-grid.vue`
- Create: `wxapp/components/uni-ui/uni-grid-item/uni-grid-item.vue`

- [ ] **Step 1: Add components**

从参考项目迁移 `uni-grid` 和 `uni-grid-item` 的最小可用实现，保持 `column`、`showBorder`、`square`、`highlight`、`change` 事件。

- [ ] **Step 2: Verify pages still validate**

Run: `npm run validate:pages`
Expected: PASS.

### Task 3: 接入商家资料页

**Files:**
- Modify: `wxapp/pages/merchant/profile.vue`

- [ ] **Step 1: Replace image UI**

将“选择图片”按钮和普通 CSS grid 替换为 `uni-grid`；已有图片格点击预览，右下角按钮删除，最后一格添加图片。

- [ ] **Step 2: Replace image state**

用 `merchantImageEntries` 管理已保存 URL 和本地待上传文件，加载商家详情时生成已保存 entry，保存时上传本地 entry 后回写 URL 数组。

- [ ] **Step 3: Add compression before upload**

上传本地主页图片前先调用 `compressMerchantImageFile`：通过 `uni.getImageInfo` 取尺寸，再按工具函数计算 `uni.compressImage` 参数；不显示逐张上传进度。

- [ ] **Step 4: Run focused validation**

Run: `node wxapp/common/merchantProfileImages.test.mjs`
Expected: PASS.

Run: `npm run validate:pages`
Expected: PASS.

Run: `npm run validate:flows`
Expected: PASS.
