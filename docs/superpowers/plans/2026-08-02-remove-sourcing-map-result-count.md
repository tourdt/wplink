# 拿货档口移除结果数量文案实施计划

> **供代理执行：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`，按任务逐项实施。所有步骤使用复选框跟踪。

**目标：** 移除拿货档口页面搜索框下方的“xx 个档口”数量文案，并保持现有搜索、筛选和分页能力不变。

**架构：** 只调整拿货档口页面的模板和专用样式，不改变接口、响应结构或页面状态模型。`total` 继续作为分页判断的数据源，测试同时约束“展示已移除”和“分页依赖仍存在”。

**技术栈：** Vue 3、uni-app、SCSS、Node.js `node:test`

## 全局约束

- 筛选标签自然上移，并继续使用现有 `16rpx` 顶部间距。
- 保留 `total` 状态、接口赋值及 `hasMore` 分页判断。
- 不改变搜索、筛选、加载、空状态或接口逻辑。
- 不新增替代数量文案或空白占位。

---

### 任务 1：移除档口数量展示

**文件：**
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`
- 修改：`wxapp/pages/sourcing-map/index.vue`

**接口：**
- 输入：`listMerchantPlaces` 返回的 `total` 数值。
- 输出：页面不再渲染结果数量；`total` 仍驱动 `hasMore` 和触底分页。

- [ ] **步骤 1：编写失败测试**

在 `wxapp/pages/sourcing-map/index.test.mjs` 增加：

```js
test('merchant booth directory omits redundant result count while keeping pagination total', () => {
  assert.doesNotMatch(source, /\{\{\s*total\s*\}\}\s*个档口/)
  assert.doesNotMatch(source, /directory-summary|result-count/)
  expectTokens(source, [
    'const total = ref(0)',
    'const hasMore = computed(() => places.value.length < total.value)',
    'total.value = Number(resp.total || 0)',
  ])
})
```

- [ ] **步骤 2：运行测试并确认按预期失败**

运行：

```bash
cd wxapp
node --experimental-vm-modules --test pages/sourcing-map/index.test.mjs
```

预期：FAIL，失败原因是页面源码仍包含数量插值或 `directory-summary` / `result-count`。

- [ ] **步骤 3：完成最小实现**

从 `wxapp/pages/sourcing-map/index.vue` 模板删除：

```vue
<view class="directory-summary">
  <text class="result-count">{{ total }} 个档口</text>
</view>
```

从同一文件的样式区域删除：

```scss
.directory-summary {
  display: flex;
  justify-content: flex-end;
  margin-top: 18rpx;
}

.result-count {
  color: #7a8799;
  font-size: 22rpx;
}
```

不要删除 `total`、`hasMore` 或 `loadPlaces` 中的 `total.value` 赋值。`.filter-scroll` 保持 `margin-top: 16rpx`。

- [ ] **步骤 4：运行聚焦测试并确认通过**

运行：

```bash
cd wxapp
node --experimental-vm-modules --test pages/sourcing-map/index.test.mjs
```

预期：该测试文件全部 PASS。

- [ ] **步骤 5：运行小程序完整验证**

运行：

```bash
cd wxapp
npm run check
```

预期：页面与流程校验通过、Node 测试全部通过、`mp-weixin` 构建成功；既有 Sass 弃用警告允许保留。

- [ ] **步骤 6：检查差异并提交**

运行：

```bash
git diff --check
git diff -- wxapp/pages/sourcing-map/index.vue wxapp/pages/sourcing-map/index.test.mjs
git add wxapp/pages/sourcing-map/index.vue wxapp/pages/sourcing-map/index.test.mjs
git commit -m "fix: 移除档口数量文案"
```

预期：差异只包含测试、数量展示节点和对应专用样式；提交成功。
