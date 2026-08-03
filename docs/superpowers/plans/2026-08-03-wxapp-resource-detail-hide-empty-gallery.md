# 供需详情无图隐藏图片区块 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 无图片的供需详情页不渲染图片区块，自动摘要直接作为详情首个内容区块。

**Architecture:** 在 `detail.vue` 中以 `galleryImages.length` 控制图库整体渲染，不引入新状态或接口。删除仅用于无图占位的模板、计算状态和样式；详情摘要卡保持原有顺序。测试继续采用当前详情页的源码回归断言，分别锁定有图保留图库和无图不含占位分支。

**Tech Stack:** Vue 3 `<script setup>`、uni-app、Node.js 内置测试运行器。

## Global Constraints

- 不改变图片上传、编辑、预览和发布流程。
- 不增加后端接口、字段或数据库变更。
- 保持自动摘要、商家资料入口、联系方式和管理操作规则不变。
- 仅修改与无图隐藏图片区块直接相关的前端代码及测试。

---

### Task 1: 条件渲染图片库并移除无图占位

**Files:**
- Modify: `wxapp/pages/resource/detail.test.mjs:24-40`
- Modify: `wxapp/pages/resource/detail.vue:13-46,319-328,1324-1417`

**Interfaces:**
- Consumes: `galleryImages` 计算属性，返回详情图片 URL 数组。
- Produces: 无图时不输出 `detail-gallery`，有图时保留 `swiper`、单图 `image` 与 `previewGalleryImage` 交互。

- [ ] **Step 1: 写入失败的回归测试**

将现有无图封面断言改为验证图库由 `galleryImages.length` 控制，并断言源码不再含有无图占位标识和占位专用状态：

```js
test('resource detail hides the gallery when no image is available', () => {
  assert.match(source, /<view v-if="galleryImages\.length" class="detail-gallery">/)
  assert.doesNotMatch(source, /gallery-placeholder/)
  assert.doesNotMatch(source, /noImageBadgeText/)
  assert.doesNotMatch(source, /canEditOwnResourceWithoutImage/)
})
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `node --test wxapp/pages/resource/detail.test.mjs`

Expected: `resource detail hides the gallery when no image is available` 失败，因为当前仍有 `v-else` 无图封面和占位状态。

- [ ] **Step 3: 实施最小页面改动**

将图库外层调整为：

```vue
<view v-if="galleryImages.length" class="detail-gallery">
  <swiper
    v-if="galleryImages.length > 1"
    class="gallery-main gallery-swiper"
    indicator-dots
    indicator-color="rgba(255, 255, 255, 0.55)"
    indicator-active-color="#ffffff"
    duration="450"
    easing-function="easeInOutCubic"
    @change="handleGalleryChange"
  >
    <swiper-item v-for="(url, index) in galleryImages" :key="`${url}-${index}`">
      <image class="gallery-slide-image" :src="url" mode="aspectFill" @click="previewGalleryImage(index)" />
    </swiper-item>
  </swiper>
  <image v-else class="gallery-main" :src="mainImage" mode="aspectFill" @click="previewGalleryImage(0)" />
</view>
```

删除原 `v-else` 占位模板，以及以下仅服务于无图占位的计算状态：

```js
const isDemandResource = computed(() => detailPresentation.value.isDemand)
const noImageBadgeText = computed(() => `${resourceNoun.value}信息`)
const noImageStateTitle = computed(() => `${resourceNoun.value}暂无图片`)
const noImageHintText = computed(() => {
  if (isOwnResource.value) return '当前未上传图片，补充后详情展示会更完整。'
  return `重点${resourceNoun.value}信息已整理在下方详情中。`
})
const canEditOwnResourceWithoutImage = computed(() => isOwnResource.value && ['draft', 'rejected'].includes(resource.value.status))
```

同时删除 `.gallery-placeholder`、`.gallery-placeholder.demand`、`.placeholder-*` 的样式；保留 `.detail-gallery`、`.gallery-main`、`.gallery-swiper` 与图片预览样式。

- [ ] **Step 4: 运行详情页测试并确认通过**

Run: `node --test wxapp/pages/resource/detail.test.mjs`

Expected: 所有详情页测试通过，且有图轮播、预览、自动摘要、商家入口和联系行为断言仍为绿色。

- [ ] **Step 5: 运行前端完整验证**

Run: `npm run check`

Expected: 页面校验、流程校验、Node 测试及微信小程序构建均退出码为 0。

- [ ] **Step 6: 提交实现**

```bash
git add wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "feat: hide empty resource detail gallery"
```
