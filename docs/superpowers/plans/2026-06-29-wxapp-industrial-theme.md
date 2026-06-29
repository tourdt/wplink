# wxapp 工业 B2B 主题配色 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 wxapp 全局配色调整为参考图中的工业 B2B 风格。

**Architecture:** 先用 `wxapp/uni.scss` 定义新的主题 token，再对页面和组件里的旧硬编码色做机械替换，最后用源码级测试确保旧主色不再出现。保留页面结构和业务逻辑，只调整颜色。

**Tech Stack:** uni-app、Vue 3、SCSS、Node.js `node:test`、现有 `validate-flows` / `validate-pages` 脚本。

---

### Task 1: 添加主题验证契约

**Files:**
- Modify: `wxapp/scripts/validate-flows.test.mjs`

- [ ] **Step 1: Write the failing test**

在 `wxapp/scripts/validate-flows.test.mjs` 追加：

```js
test('wxapp uses industrial b2b theme colors without legacy green primary', () => {
  const root = path.resolve(new URL('..', import.meta.url).pathname)
  const sources = collectSourceFiles(root, ['.vue', '.scss', '.js', '.json'])
    .filter((file) => !file.includes(`${path.sep}dist${path.sep}`))
    .filter((file) => !file.includes(`${path.sep}node_modules${path.sep}`))
    .map((file) => fs.readFileSync(file, 'utf8'))
    .join('\n')
  const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))

  for (const token of ['#061625', '#c23a00', '#fff0e8', '#f4f7fd', '#d8e0ec', '#16a36a']) {
    assert.match(sources, new RegExp(token, 'i'))
  }

  for (const legacyColor of ['#0f766e', '#e6f4f1']) {
    assert.equal(sources.toLowerCase().includes(legacyColor), false)
  }

  assert.equal(pagesConfig.tabBar.selectedColor, '#c23a00')
  assert.equal(pagesConfig.globalStyle.backgroundColor, '#f4f7fd')
})

function collectSourceFiles(dir, extensions) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === 'dist') continue
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...collectSourceFiles(fullPath, extensions))
    } else if (entry.isFile() && extensions.includes(path.extname(entry.name))) {
      files.push(fullPath)
    }
  }
  return files
}
```

- [ ] **Step 2: Run test and confirm red**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: FAIL because old `#0f766e` and `#e6f4f1` still exist.

### Task 2: 替换主题色

**Files:**
- Modify: `wxapp/uni.scss`
- Modify: `wxapp/pages.json`
- Modify: all `wxapp/**/*.vue` / `wxapp/**/*.scss` source files containing old theme colors

- [ ] **Step 1: Update global tokens**

Set `wxapp/uni.scss` tokens:

```scss
$wplink-primary: #061625;
$wplink-primary-soft: #eaf2ff;
$wplink-bg: #f4f7fd;
$wplink-card: #ffffff;
$wplink-text: #061625;
$wplink-muted: #6b7280;
$wplink-line: #d8e0ec;
$wplink-warning: #c23a00;
$wplink-warning-soft: #fff0e8;
$wplink-price: #c23a00;
$wplink-blue: #061625;
$wplink-blue-soft: #eaf2ff;
$wplink-coral: #c23a00;
$wplink-coral-soft: #fff0e8;
$wplink-success: #16a36a;
$wplink-success-soft: #e8f7ef;
```

- [ ] **Step 2: Update `pages.json`**

Set:

```json
"selectedColor": "#c23a00",
"backgroundColor": "#f4f7fd"
```

Keep tabBar background `#ffffff`.

- [ ] **Step 3: Replace old hardcoded colors**

Apply these replacements under `wxapp`, excluding `dist` and `node_modules`:

- `#0f766e` -> `#061625`
- `#e6f4f1` -> `#eaf2ff`
- `#f4f6f8` -> `#f4f7fd`
- `#1f2933` -> `#061625`
- `#697586` -> `#6b7280`
- `#d8dde6` -> `#d8e0ec`
- `#c2410c` -> `#c23a00`
- `#b7791f` -> `#c23a00`
- `#fff7e6` -> `#fff0e8`

- [ ] **Step 4: Add independent success colors where needed**

For clearly successful statuses, use:

- `#16a36a`
- `#e8f7ef`

Do not reintroduce `#0f766e`.

### Task 3: 验证

**Files:**
- Test: `wxapp/scripts/validate-flows.test.mjs`
- Test: `wxapp/scripts/validate-flows.mjs`
- Test: `wxapp/scripts/validate-pages.mjs`

- [ ] **Step 1: Run theme test**

Run: `node --test wxapp/scripts/validate-flows.test.mjs`

Expected: PASS.

- [ ] **Step 2: Run validators**

Run:

```bash
cd wxapp && npm run validate:flows
cd wxapp && npm run validate:pages
cd wxapp && npm run build:mp-weixin
```

Expected: validators pass and build completes.
