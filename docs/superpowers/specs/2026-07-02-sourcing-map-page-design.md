# wxapp 拿货地图方案 C 分阶段落地设计

## 目标

参考《织里童装拿货导览地图产品开发文档》，在当前系统中建设一套可运营的拿货地图能力。方案 C 覆盖后端数据结构和 API、管理后台标注工具、小程序拿货地图页面，并在 `我的`页增加入口。

本设计不把“全织里完整地图平台”作为第一期目标。第一期只做一个可运营街区闭环：一个场景、一张可商用底图、50 到 100 个档口、10 到 20 个配套 POI、基础搜索筛选、后台矩形和点位标注、小程序展示和详情联动。

## 总体判断

方案 C 技术上可行，且产品方向成立。当前系统具备落地基础：

- `backend` 已有 PostgreSQL 迁移、go-zero API 文件、业务 logic、model 和手写路由接线模式。
- `admin-web` 已是 Vue3 + Element Plus + Vue Router，可承接场景管理和标注工具。
- `wxapp` 已是 uni-app + Vue3，可新增独立页面并从 `我的`页跳转。
- 上传能力已有 `/api/v1/uploads/token`，可扩展 `purpose=map_background` 支撑底图上传。

但方案 C 不能按大而全方式一次完成。主要风险不在建表或接口，而在底图、标注工具、数据采集、坐标版本管理和运营审核。若没有稳定底图和首批真实数据，技术实现只能停留在演示层。

## 落地前置条件

第一期启动前需要准备：

- 一张可商用、尺寸固定的街区底图，例如 `3000 x 1800`。
- 一个明确街区场景，例如 `利济路中段`。
- 50 到 100 个档口基础资料：编号、名称、主营分类、服务标签、地址或路段描述。
- 10 到 20 个 POI：打包站、物流点、快递点、停车场。
- 一名后台运营人员能用标注工具维护坐标和标签。
- 明确示例数据和真实数据的上线边界，避免用户误解为完整导航地图。

如果以上条件暂时不齐，仍可先完成后端表结构、后台标注框架和小程序页面，但验收标准必须定义为“技术闭环完成”，不能定义为“真实拿货地图上线”。

## 分阶段范围

### 第一期：单街区可运营闭环

目标：完成从后台录入到小程序展示的完整链路。

覆盖：

- 后端新增地图表、基础 API 和管理 API。
- 后台新增场景列表、场景编辑、底图上传、矩形档口标注、点位 POI 标注、保存和预览。
- 小程序新增 `拿货地图` 入口和独立页面。
- 小程序支持场景加载、对象绘制、搜索、筛选、点击详情、附近配套推荐。
- 数据只覆盖一个街区场景，控制对象数量。

不覆盖：

- 全织里多街区一次性展示。
- 复杂瓦片系统。
- 真实室内导航和路径规划。
- 商户自助认领和审核流程。
- 物流、快递、支付下单。
- 高级文字避让和热力图。

### 第二期：多场景和运营效率

目标：让运营可以更高效维护多个街区。

覆盖：

- 多场景切换。
- 批量生成档口。
- 批量编号。
- 批量设置分类和标签。
- polygon 几何支持。
- 数据发布前预览。
- 小程序按街区加载，避免一次性加载全量数据。

### 第三期：规模化和体验优化

目标：支撑更多街区和商场楼层。

覆盖：

- 视口加载。
- grid 缓存。
- 大图切片或分辨率分级。
- 搜索服务升级。
- 商场楼层场景。
- 导航经纬度完善。
- 商户提交位置和平台审核。

## 当前系统适配

### 后端适配

当前 `backend/app/api/app.api` 通过 import 聚合业务 API，新增地图能力应增加 `map.api` 并继续挂在 `/api/v1` 下。产品文档中的 `/api/map/...` 建议调整为 `/api/v1/map/...`，保持现有接口风格一致。

项目规则要求能由 goctl 生成的代码优先由 goctl 生成。地图模块应先写 `.api` 和 SQL schema，再通过 goctl 生成可生成部分，手写代码只放在 logic、路由接线和必要的扩展 model 中。

当前 `domain_routes.go` 里已有可选业务路由注册模式。地图模块可以新增 `MapAPIStore` 接口，再在 `registerOptionalDomainRoutes` 中按 store 能力注册地图路由。

### 后台适配

当前后台已有 Vue Router、Element Plus、侧边栏布局和 API 封装。地图管理应新增独立导航项，例如 `拿货地图`，下设一个综合页面：

- 左侧场景列表。
- 中间标注画布。
- 右侧对象详情编辑面板。
- 顶部保存、发布、预览、批量生成按钮。

标注工具建议采用 Konva 生态，例如 `konva` 和 `vue-konva`。当前项目未安装该依赖，实施时需要通过 `npm install` 增加依赖并验证构建体积。

### 小程序适配

当前 `wxapp/pages/my/index.vue` 的 action list 适合新增入口。独立页面建议路径为 `pages/sourcing-map/index`，页面注册到 `pages.json`，并纳入 `validate-pages.mjs`。

小程序第一期应按 scene 加载数据，不做全量加载。Canvas 采用业务缩放层级，不做真实地图 zoom。导航能力第一期优先使用 `uni.openLocation`，而不是自行选择高德、百度、Apple Maps。

## 数据模型

### map_scene

地图场景表，表示总览、街区、路段、商场或楼层。

关键字段：

- `id bigint primary key default next_tsid()`
- `city_station_id bigint references city_stations(id)`
- `code varchar(64) unique not null`
- `name varchar(100) not null`
- `type varchar(30) not null`
- `parent_code varchar(64)`
- `background_url text not null`
- `width int not null`
- `height int not null`
- `min_scale numeric(5,2) not null default 0.5`
- `max_scale numeric(5,2) not null default 5`
- `default_scale numeric(5,2) not null default 1`
- `default_center_x numeric(10,2)`
- `default_center_y numeric(10,2)`
- `floor_no varchar(20)`
- `sort int not null default 0`
- `revision int not null default 1`
- `status varchar(20) not null default 'draft'`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

状态建议：

- `draft`：后台编辑中，小程序不可见。
- `published`：小程序可见。
- `archived`：归档隐藏。

### map_object

地图对象表，表示档口、街区区域、POI、市场设施。

关键字段：

- `id bigint primary key default next_tsid()`
- `scene_code varchar(64) not null`
- `merchant_id bigint references merchants(id)`
- `code varchar(64) not null`
- `name varchar(100) not null`
- `type varchar(30) not null`
- `layer varchar(30) not null`
- `geometry_type varchar(30) not null`
- `geometry jsonb not null`
- `center_x numeric(10,2)`
- `center_y numeric(10,2)`
- `min_x numeric(10,2)`
- `min_y numeric(10,2)`
- `max_x numeric(10,2)`
- `max_y numeric(10,2)`
- `min_zoom int not null default 1`
- `max_zoom int not null default 5`
- `category_codes jsonb not null default '[]'::jsonb`
- `service_tags jsonb not null default '[]'::jsonb`
- `platform_tags jsonb not null default '[]'::jsonb`
- `poi_service_tags jsonb not null default '[]'::jsonb`
- `address text`
- `phone varchar(30)`
- `wechat varchar(50)`
- `lat numeric(10,7)`
- `lng numeric(10,7)`
- `search_text text not null default ''`
- `extra jsonb not null default '{}'::jsonb`
- `sort int not null default 0`
- `status varchar(20) not null default 'normal'`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

约束：

- `unique(scene_code, code)`
- `geometry_type` 第一期开启 `rect` 和 `point`。
- 第二期再开放 `polygon`。

`merchant_id` 用于后续把档口和现有商户体系关联。第一期可以为空，由运营先维护档口对象；后续商户认领后再绑定。

### map_category

地图分类和标签表，承载主营分类、档口服务、平台标签、POI 类型和 POI 服务标签。

关键字段：

- `id bigint primary key default next_tsid()`
- `code varchar(64) unique not null`
- `name varchar(100) not null`
- `type varchar(30) not null`
- `icon_url text`
- `sort int not null default 0`
- `is_visible boolean not null default true`
- `status varchar(20) not null default 'normal'`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

### 索引策略

第一期索引：

- `idx_map_scene_city_status`：`city_station_id, status, sort`
- `idx_map_scene_parent`：`parent_code`
- `idx_map_object_scene`：`scene_code`
- `idx_map_object_scene_type`：`scene_code, type`
- `idx_map_object_scene_bounds`：`scene_code, min_x, max_x, min_y, max_y`
- `idx_map_object_scene_status_sort`：`scene_code, status, sort`

中文搜索不建议第一期依赖 `to_tsvector('simple')`。第一期用 `search_text ILIKE`、编号精确匹配和标签过滤即可。后期数据量变大后再接 MeiliSearch 或 Elasticsearch。

## API 设计

### 小程序公开 API

使用 `/api/v1/map` 前缀。

- `GET /api/v1/map/scenes`
  - 查询已发布场景。
  - 参数：`cityCode`、`parentCode`、`type`。

- `GET /api/v1/map/scenes/{sceneCode}`
  - 获取场景详情。
  - 只返回 `published` 场景。

- `GET /api/v1/map/scenes/{sceneCode}/objects`
  - 获取场景对象。
  - 参数：`types`、`categories`、`serviceTags`、`poiServiceTags`、`keyword`。
  - 第一版返回当前场景全部匹配对象，不做分页。

- `GET /api/v1/map/objects/search`
  - 跨场景或场景内搜索。
  - 参数：`sceneCode`、`keyword`、`types`、`limit`。

- `GET /api/v1/map/objects/{objectId}`
  - 获取对象详情。

- `GET /api/v1/map/objects/{objectId}/nearby-pois`
  - 获取附近配套。
  - 参数：`types`、`limit`。
  - 第一版基于同场景坐标距离计算。

### 后台管理 API

使用 `/api/v1/admin/map` 前缀，沿用后台登录态和管理员鉴权。

- `GET /api/v1/admin/map/scenes`
- `POST /api/v1/admin/map/scenes`
- `GET /api/v1/admin/map/scenes/{sceneCode}`
- `POST /api/v1/admin/map/scenes/{sceneCode}`
- `POST /api/v1/admin/map/scenes/{sceneCode}/publish`
- `GET /api/v1/admin/map/scenes/{sceneCode}/objects`
- `POST /api/v1/admin/map/scenes/{sceneCode}/objects`
- `POST /api/v1/admin/map/objects/{objectId}`
- `POST /api/v1/admin/map/objects/{objectId}/status`
- `POST /api/v1/admin/map/scenes/{sceneCode}/objects/batch-generate`
- `GET /api/v1/admin/map/categories`
- `POST /api/v1/admin/map/categories`

后台保存对象时，后端统一计算 `center_x`、`center_y`、`min_x`、`min_y`、`max_x`、`max_y` 和 `search_text`，避免前端保存不一致。

## 后台标注工具方案

第一期标注工具必须具备：

- 加载场景底图。
- 画矩形档口。
- 画点位 POI。
- 拖动对象。
- 调整矩形大小。
- 编辑对象名称、编号、类型、分类、标签、地址、电话、微信。
- 批量生成横向或纵向档口。
- 保存对象。
- 发布场景。
- 预览小程序效果所需数据。

第一期可以不做：

- polygon 编辑。
- 自动吸附。
- 复杂图层树。
- 文字避让。
- 多人协作编辑。

技术建议：

- 画布使用 Konva。
- 底图和对象共用同一个坐标系，坐标以底图原始尺寸为准。
- 画布缩放只影响视图，不改变保存坐标。
- 保存前校验对象是否在底图范围内。
- 发布前校验场景存在底图、至少一个对象、对象编号唯一。

## 小程序地图页面方案

页面入口：

- 在 `我的`页新增 `拿货地图`。
- 未登录用户也可进入。
- 后续收藏、认领、纠错再要求登录。

页面结构：

- 顶部搜索框。
- 当前区域提示。
- 快捷筛选。
- Canvas 地图导览区。
- 搜索结果列表。
- 对象详情底部抽屉。
- 附近配套列表。
- 回到总览按钮。

渲染规则：

- 远看显示区域和重点 POI。
- 中等缩放显示档口块。
- 近看显示档口编号。
- 更近显示档口名称或主营标签。

性能策略：

- 单场景第一期控制在 200 个对象以内。
- 小程序端只请求当前场景对象。
- 搜索结果点击后设置高亮和中心点。
- 暂不做全织里对象一次性展示。

## 数据流

后台数据流：

1. 管理员创建场景。
2. 管理员上传底图，保存宽高和默认视图。
3. 管理员在画布上标注档口和 POI。
4. 后端保存对象并计算边界字段和搜索字段。
5. 管理员发布场景。

小程序数据流：

1. 用户从 `我的`页进入拿货地图。
2. 小程序请求已发布场景列表。
3. 小程序加载默认场景详情。
4. 小程序请求当前场景对象。
5. 用户搜索、筛选或点击对象。
6. 小程序展示详情和附近配套。
7. 有经纬度时调用 `uni.openLocation`。

## 技术问题评估

### 表结构是否有问题

基础方向正确，但原文档表结构需要补充 `city_station_id`、`merchant_id`、`revision` 和明确发布状态。这样才能与现有织里站、多城市站和商户体系衔接。

`geometry JSONB + center/bounds 冗余字段` 是合理设计。JSONB 保持灵活，冗余字段用于检索、附近计算和视口优化。

### API 是否有问题

公开 API 和后台 API 必须分开。公开 API 只返回 `published` 数据；后台 API 可操作 `draft` 数据。

路径应统一到 `/api/v1/map` 和 `/api/v1/admin/map`，避免与当前系统接口前缀不一致。

### 搜索方案是否有问题

第一期不建议上全文搜索。中文短词、档口编号、标签筛选更适合用 `ILIKE`、编号精确匹配和 JSONB 标签过滤。数据量扩大后再升级搜索服务。

### 后台标注方案是否有问题

Konva 方案可行，适合后台标注和对象拖拽。风险在于依赖新增和复杂交互测试。第一期要克制功能，只做矩形和点位，先不做 polygon。

### 小程序 Canvas 是否有问题

Canvas 方案可行，但要真机验证触摸、缩放、文字清晰度和点击命中。第一期必须限制对象数量，不做多场景全量渲染。

### 底图方案是否有问题

底图是最大业务风险。不能直接使用第三方地图截图作为商用底图。应使用自绘底图、授权底图或运营绘制底图，并固定尺寸。底图更新必须保持尺寸和坐标系不变，否则坐标需要整体迁移。

## 风险与应对

### 范围风险

风险：方案 C 跨后端、后台、小程序和运营数据，容易失控。

应对：第一期只做一个街区闭环。验收以单场景可运营为准。

### 数据风险

风险：档口和 POI 数据不准确会直接伤害用户信任。

应对：后台发布前预览；对象状态支持隐藏；小程序展示“信息以现场为准”；后续增加纠错入口。

### 坐标风险

风险：底图尺寸变化导致坐标失效。

应对：保存底图宽高；禁止发布后更换不同尺寸底图；需要换图时创建新 revision 或新场景。

### 性能风险

风险：小程序 Canvas 渲染大量对象卡顿。

应对：按 scene 加载；低 zoom 不显示文字；对象数量超过阈值时引入视口加载和 grid 缓存。

### 后台复杂度风险

风险：标注工具容易演变成完整设计软件。

应对：第一期只做矩形、点位、拖拽、缩放、批量生成和属性编辑。

### go-zero 规范风险

风险：直接手写大量 handler/type/model，偏离项目规则。

应对：`.api`、handler、types、model 能生成的部分优先通过 goctl 生成；业务校验、友好错误和日志放在 logic。

## 第一期验收标准

后端：

- 迁移创建 `map_scene`、`map_object`、`map_category`。
- `.api` 定义公开和后台地图接口。
- goctl 生成可生成代码。
- model 通过 schema 生成。
- 公开 API 只返回发布状态数据。
- 后台 API 可创建场景、保存对象、发布场景。
- 附近配套可基于同场景坐标返回。

后台：

- 侧边栏出现 `拿货地图`。
- 可创建一个场景。
- 可上传并显示底图。
- 可画矩形档口和点位 POI。
- 可编辑对象属性。
- 可批量生成一排档口。
- 可保存并发布场景。

小程序：

- `我的`页出现 `拿货地图` 入口。
- 点击可进入独立页面。
- 可加载发布场景和对象。
- 可搜索档口和 POI。
- 可按分类筛选。
- 可点击对象展示详情。
- 可展示附近打包站、物流点、快递点、停车场。
- 有经纬度时可打开系统位置。

质量：

- 后端单元测试通过。
- 后台构建通过。
- 小程序页面校验和构建通过。
- 单场景真机验证无明显卡顿。

## 后续实施建议

建议先把本设计拆成三个实施计划，而不是一个超大计划：

1. 后端地图数据和 API 计划。
2. 后台地图管理和标注工具计划。
3. 小程序拿货地图页面计划。

三个计划按顺序推进。后端先提供稳定契约，后台负责生产数据，小程序消费已发布数据。这样每一步都有可验收产物，也能及时发现底图和数据问题。
