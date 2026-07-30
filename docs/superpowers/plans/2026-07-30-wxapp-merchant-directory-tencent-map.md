# 微信小程序商家目录与腾讯地图重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将“拿货地图”重构为默认商家列表、可切换微信原生腾讯地图的商家目录；首页 Banner 位置不变且后台可配置跳转；首发隐藏 VIP 套餐入口。

**架构：** 后端以现有 `map_object` 和商家绑定关系为唯一数据源，新增面向商家目录的公开查询语义；小程序使用共享查询状态驱动列表与原生 `<map>` 两种视图，保留旧 Canvas 文件但退出用户访问链路。入口调整、数据接口、列表视图、地图视图和认领/纠错分别交付并验证。

**技术栈：** Go 1.x、go-zero 风格 HTTP 服务、PostgreSQL、Vue 3、uni-app、微信小程序原生 `map`、Node.js 原生测试。

## 全局约束

- 所有用户可见错误使用清晰中文；后端保留可诊断日志，不向前端暴露内部错误。
- 商家目录不得展示电话、微信等联系方式；联系方式仅存在于有效供需详情。
- “已入驻”表示地图点位已经绑定平台商家；“待认领”表示平台预录且尚未绑定。
- 经纬度缺失或非法的结果仍进入列表，但不生成地图标记、不提供导航。
- 不删除 `canvasRenderer.js`、`mapGeometry.js`、`mapGesture.js`、`mapHitTest.js` 等旧自定义地图资产。
- 不修改当前工作区中与本需求无关的 `backend/app/internal/adminweb/dist` 变更。

---

## 任务 1：固定产品入口并隐藏 VIP

**文件：**

- 修改：`wxapp/pages.json`
- 修改：`wxapp/pages/my/index.vue`
- 修改：`wxapp/pages/my/index.test.mjs`
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`
- 修改：`wxapp/scripts/validate-flows.mjs`
- 修改：`admin-web/src/views/BannerTopicView.vue`
- 新增：`admin-web/src/views/BannerTopicView.test.mjs`
- 新增：`wxapp/static/tabbar/map.png`
- 新增：`wxapp/static/tabbar/map-active.png`

- [ ] 先更新测试，断言 TabBar 为“首页｜拿货地图｜发布｜供需｜我的”，消息页仍可从“我的”进入。
- [ ] 更新“我的”测试，断言不再渲染 VIP 套餐入口或跳转逻辑，但保留成长权益和额度展示。
- [ ] 为 Banner 内部页选项增加“拿货地图”，删除已不存在的认证页选项。
- [ ] 生成风格与现有 TabBar 一致的地图普通态、选中态 PNG 图标。
- [ ] 修改入口配置与页面实现，使首页 Banner 继续通过已有 `jumpType=internal`、`jumpTarget=/pages/sourcing-map/index` 跳转。
- [ ] 运行：`cd wxapp && node --test pages/my/index.test.mjs pages/sourcing-map/index.test.mjs scripts/validate-flows.test.mjs`
- [ ] 运行：`cd admin-web && npm test -- --runInBand`（若项目无统一测试脚本，则运行新增的 Node 测试）。

## 任务 2：新增商家目录后端查询

**文件：**

- 修改：`backend/app/api/map.api`
- 修改：`backend/app/internal/model/map_model.go`
- 修改：`backend/app/internal/model/map_model_test.go`
- 修改：`backend/app/internal/logic/map/public_logic.go`
- 修改：`backend/app/internal/logic/map/public_logic_test.go`
- 修改：`backend/app/internal/server/domain_routes.go`
- 修改：`backend/app/internal/server/map_routes.go`
- 修改：`backend/app/internal/server/map_api_test.go`

- [ ] 在模型测试中定义目录过滤、分页、已入驻/待认领状态、经纬度边界和稳定排序的预期。
- [ ] 新增 `MerchantPlaceFilter`、`MerchantPlace` 和查询/计数方法；只查询已发布场景、正常档口。
- [ ] 在 `map.api` 增加 `GET /api/v1/map/merchant-places` 的请求和响应契约，字段覆盖商家身份、档口地址、分类标签、经纬度、距离和风险提示。
- [ ] 在公开逻辑测试中覆盖关键词、类型、认领状态、分页参数校验，以及公共响应不含联系方式。
- [ ] 实现目录逻辑：已绑定点位使用商家名称和资料；未绑定点位使用平台预录名称；`sourceType` 固定为 `merchant_claimed` 或 `platform_prelisted`。
- [ ] 在路由测试中验证查询参数透传和中文错误响应，再注册公开路由。
- [ ] 运行：`cd backend && go test ./app/internal/model ./app/internal/logic/map ./app/internal/server`

## 任务 3：建立小程序共享查询状态与商家卡片

**文件：**

- 修改：`wxapp/api/sourcingMap.js`
- 新增：`wxapp/pages/sourcing-map/merchantPlaceState.js`
- 新增：`wxapp/pages/sourcing-map/merchantPlaceState.test.mjs`
- 新增：`wxapp/components/MerchantPlaceCard.vue`
- 新增：`wxapp/components/MerchantPlaceCard.test.mjs`

- [ ] 先写状态测试：查询参数清洗、分页重置、已入驻/待认领文案、合法坐标判断和列表/地图共享筛选。
- [ ] 增加 `listMerchantPlaces` API 方法。
- [ ] 实现无框架依赖的状态辅助函数，确保 Node 测试可直接执行。
- [ ] 创建“档口门牌”商家卡片，展示名称、编号、市场楼层、品类、来源状态和距离；禁止出现电话/微信。
- [ ] 为缺坐标卡片显示“位置待完善”，隐藏导航按钮。
- [ ] 运行：`cd wxapp && node --test pages/sourcing-map/merchantPlaceState.test.mjs components/MerchantPlaceCard.test.mjs`

## 任务 4：将拿货地图首页替换为列表优先

**文件：**

- 移动：`wxapp/pages/sourcing-map/index.vue` → `wxapp/pages/sourcing-map/legacy-canvas.vue`
- 新增：`wxapp/pages/sourcing-map/index.vue`
- 重写：`wxapp/pages/sourcing-map/index.test.mjs`

- [ ] 先把页面测试改为断言默认列表、顶部搜索、筛选、列表/地图切换、加载/空/失败状态，以及不存在 Canvas 入口。
- [ ] 将旧页面完整移动到未注册的 `legacy-canvas.vue`，保留所有旧模块和数据能力。
- [ ] 新页面以列表为默认视图，提供关键词、已入驻/待认领、品类筛选和下拉刷新/分页加载。
- [ ] 搜索结果只渲染商家档口，不混入供需资源卡片。
- [ ] 点击已入驻商家进入公开商家页；点击待认领点位展示平台预录说明和认领入口。
- [ ] 运行：`cd wxapp && node --test pages/sourcing-map/index.test.mjs pages/sourcing-map/merchantPlaceState.test.mjs`

## 任务 5：接入微信原生腾讯地图

**文件：**

- 新增：`wxapp/pages/sourcing-map/tencentMapState.js`
- 新增：`wxapp/pages/sourcing-map/tencentMapState.test.mjs`
- 修改：`wxapp/pages/sourcing-map/index.vue`
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`

- [ ] 先写标记构造测试：已入驻深蓝实心、待认领描边、选中橙色，非法坐标不生成标记。
- [ ] 页面地图态使用原生 `<map>`，并由同一批商家结果生成 `markers`。
- [ ] `markertap` 选中商家并弹出自定义底部商家卡片，不依赖系统气泡承载业务信息。
- [ ] 地图拖动只显示“搜索此区域”，用户点击后才提交可视边界查询，避免请求风暴。
- [ ] 首次进入地图优先定位用户；定位失败回退到当前城市/市场的有效点位中心。
- [ ] 导航统一使用 `uni.openLocation`，缺失坐标时给出中文提示。
- [ ] 运行：`cd wxapp && node --test pages/sourcing-map/tencentMapState.test.mjs pages/sourcing-map/index.test.mjs`

## 任务 6：打通待认领与导航后纠错

**文件：**

- 修改：`wxapp/pages/merchant/map-binding.vue`
- 修改：`wxapp/pages/merchant/map-binding.test.mjs`
- 修改：`wxapp/pages/sourcing-map/index.vue`
- 修改：`wxapp/pages/sourcing-map/index.test.mjs`

- [ ] 测试从目录传入 `objectId` 时，绑定页自动定位候选档口且仍执行登录/商家资料守卫。
- [ ] “这是我的档口”跳转现有绑定流程；绑定页按 `objectId` 预选，不新增重复认领系统。
- [ ] 导航前记录目标；用户返回地图页后提供轻量“位置是否准确”反馈，错误位置复用现有位置纠错接口。
- [ ] 风险反馈复用现有风险上报接口，并保持登录校验和中文失败提示。
- [ ] 运行：`cd wxapp && node --test pages/merchant/map-binding.test.mjs pages/sourcing-map/index.test.mjs`

## 任务 7：全量验证与交付检查

**文件：**

- 修改：`docs/superpowers/plans/2026-07-30-wxapp-merchant-directory-tencent-map.md`

- [ ] 运行后端：`cd backend && go test ./...`
- [ ] 运行小程序：`cd wxapp && npm run check`
- [ ] 运行管理后台相关测试及构建：`cd admin-web && npm test`、`npm run build`（按现有脚本执行）。
- [ ] 检查 `git diff --check`，确认未误改无关构建产物。
- [ ] 对照设计逐项确认：Banner 位置未动、稳定地图入口存在、列表默认、原生腾讯地图、两类结果区分、VIP 隐藏、联系方式隔离、旧 Canvas 未删除。
- [ ] 将本计划完成项勾选，并按任务形成小而清晰的提交。
