# 任务 8 实施报告

## 设计校准

### 第一遍：基于业务主体的短方案

- 主体：在织里线下拿货、需要快速确认真实入驻档口的买家。
- 页面唯一任务：按商家名、市场、楼栋、档口号或主营品类，尽快找到已入驻商家并进入主页或查看位置。
- 色彩：沿用现有 token，不修改全局主题。正文墨色 `#061625`、页面底色 `#f4f7fd`、卡片白 `#ffffff`、分隔线 `#d8e0ec`、辅助灰 `#6b7280`、动作橙 `#c23a00`。
- 字体：沿用小程序既有 `-apple-system / PingFang SC / Microsoft YaHei` 正文字体；商家名使用较高字重，档口坐标使用同字体的紧凑字距和中等字重，避免引入字体包和增加主包体积。
- 布局：顶部只保留关键词与可清空主营品类文字输入，下方直接进入稳定分页商家列表；不设置大地图 hero、全局点位画布或列表/地图切换。
- 签名元素：卡片内使用一条紧凑的“市场 · 楼栋 · 楼层 · 档口号”坐标条，让买家以织里线下档口语言扫读位置。

```text
┌ 搜商家、市场、楼栋、档口号 ── 搜索 ┐
│ 主营品类（逗号分隔，可清空）        │
└───────────────────────────────────┘
┌ 商家名                         ───┐
│ 织里童装城 · 一期 · 1F · A108     │
│ 童装  现货  一件代发               │
│                 进入主页  查看位置 │
└───────────────────────────────────┘
```

### 第二遍：对照 brief 的去模板化复核

- 删除首屏宣传 hero、统计数字和大地图氛围，因为它们会延迟买家找到档口，也违背“列表优先”。
- 删除“全部/已入驻/待认领”等来源胶囊；目录只表达真实商家，来源不是买家要理解的维度。
- 将通用品类 chip 阵列改为可清空文字输入并按逗号解析，既贴合后端目录契约，也避免维护第二套分类配置。
- 将视觉强调只花在档口坐标条上；搜索、状态卡和按钮继续使用现有 token 与克制的工业风边框，避免全局重做。
- 位置页地图只呈现当前、附近、选中三类 marker；目录页不展示任何地图或点位装饰。

## TDD 记录

1. 目录状态、统计和请求契约：先运行
   `node --test pages/sourcing-map/merchantDirectoryState.test.mjs common/merchantDirectoryAnalytics.test.mjs api/request.test.mjs`，RED 原因为新状态/统计模块不存在；实现后使用项目要求的 VM 参数运行同组测试，17 项全部 GREEN。
2. 目录页、共享卡片和位置页：先运行
   `node --test pages/sourcing-map/index.test.mjs components/MerchantPlaceCard.test.mjs pages/merchant/locationState.test.mjs pages/merchant/location.test.mjs`，旧实现的来源筛选、认领身份和旧位置 DTO 使 21 项失败；替换为商家目录流程后 21 项全部 GREEN。
3. 详情页与页面/流程注册：先运行定向用例，旧统计事件、仍注册的绑定页和未删除文件使 3 项失败；切换目录统计并精确删除后 3 项全部 GREEN。
4. 分页错误恢复：新增“加载更多失败保留既有列表并允许上拉重试”的行为测试，RED 时页面仍显示普通加载提示；补充错误提示与成功清理后，目录页 7 项全部 GREEN。
5. 实现只修改使上述失败通过所需的生产代码；页面测试通过 VM 执行真实 `setup`/生命周期和并发 Promise，不以源码关键词扫描代替状态、竞态、存储一次消费和交互测试。

## 删除与保留清单

- 删除 API/统计：`api/sourcingMap.js`、`common/merchantMapAnalytics.js` 及其测试。
- 删除旧页面：`pages/merchant/map-binding.vue` 及其测试、`pages/sourcing-map/legacy-canvas.vue`。
- 删除画布/地图状态：`canvasRenderer`、`mapGeometry`、`mapGesture`、`mapHitTest`、`mapObjectState`、`merchantPlaceState`、`tencentMapState` 的实现和测试。
- 删除旧 marker：`marker-claimed.png`、`marker-prelisted.png`。
- 保留位置页需要的 marker：`marker-selected.png`、`marker-nearby.png`、`marker-nearby-selected.png`。
- 保留任务 7 的商家资料/详情结构化位置逻辑；`visible && available`、`risk_hidden` 和详情请求竞态未被放宽或重写。

## 残留分类

- 商家目录、位置页、API、统计和页面校验的生产代码：不存在旧认领/预置点身份、旧 API、旧统计或旧画布引用。
- 测试与删除守卫：保留少量旧名称作为反向断言、恶意输入净化样本和精确文件不存在校验，防止旧身份或旧页面回流；不进入运行时产物。
- 无关业务：`profile.vue` 与 `common/upload.js` 的 `sourceType: ['album', 'camera']` 是微信图片选择来源参数，与商家目录来源筛选无关，未改动。

## 验证

- `npm run validate:pages`：通过。
- `npm run validate:flows`：通过。
- brief 指定的 5 个目录/位置测试文件：通过。
- `node --experimental-vm-modules --test api/request.test.mjs request-loading-contract.test.mjs`：请求 URL、payload 和页面 loading 契约通过。
- `npm test`：全量 wxapp Node 测试通过。
- `npm run build:mp-weixin`：构建通过；仅输出项目既有 Sass legacy API / `@import` 弃用警告。
- 生产代码残留扫描：零命中；测试/守卫与无关图片选择参数按上一节分类。
- 提交：`refactor(wxapp): 收缩拿货档口为真实商家目录`（本报告随该提交交付，基线 `bf619e4e4fbd05903fd6e726696d8330c08fbb95`）。
