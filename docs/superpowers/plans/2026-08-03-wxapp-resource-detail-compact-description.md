# 供需详情紧凑说明与参数布局 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将类型/标签与描述合并，并收窄详细参数整行展示条件。

**Architecture:** 仅调整资源详情模板和参数展示状态工具。描述放入现有摘要卡；整行判断使用收窄后的标签集合和 20 字长度阈值，不改变地址独立展示与参数合并顺序。

**Tech Stack:** Vue 3、uni-app、Node.js 内置测试。

## Global Constraints

- 不改变参数合并顺序、图片、地址、商家、推荐、联系、管理、发布、接口或数据库。
- 描述为空时不输出占位内容。

---

### Task 1: 合并描述区并收窄整行参数规则

**Files:**
- Modify: `wxapp/common/resourceDetailState.js`
- Modify: `wxapp/common/resourceDetailState.test.mjs`
- Modify: `wxapp/pages/resource/detail.vue`
- Modify: `wxapp/pages/resource/detail.test.mjs`

- [ ] **Step 1: 写失败测试**

新增断言：`description-card` 与“补充说明”不存在，`resource.description` 位于 `detail-summary-card` 内；“服务范围：织里及周边”是双列，而“交期要求”与 21 字值为整行。

- [ ] **Step 2: 验证 RED**

Run: `node --test wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.test.mjs`

Expected: 当前仍有独立描述卡且服务范围为整行，测试失败。

- [ ] **Step 3: 最小实现**

将描述模板移动到 `detail-summary-card` 的标签后：

```vue
<text v-if="resource.description" class="desc">{{ resource.description }}</text>
```

删除独立 `description-card` 模板和样式。将整行标签规则改为 `/地址|备注|说明|交期|时效|要求/`，长度阈值改为 `> 20`；保留换行规则。

- [ ] **Step 4: 验证 GREEN 与全量检查**

Run: `node --test wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.test.mjs && npm run check`

Expected: 聚焦测试、页面/流程校验、全量测试和小程序构建均通过。

- [ ] **Step 5: 提交实现**

```bash
git add wxapp/common/resourceDetailState.js wxapp/common/resourceDetailState.test.mjs wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "feat: compact resource detail description"
```
