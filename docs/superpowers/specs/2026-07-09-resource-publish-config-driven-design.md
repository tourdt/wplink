# 发布资源与需求字段配置化重构设计

## 背景

小程序发布页目前复用同一个表单组件发布资源和需求。页面固定展示 `品类`、`数量/产能`、`价格描述`、`描述`，再根据 `resource_type_configs.field_schema` 渲染动态字段。

这个设计在早期 MVP 中可以快速覆盖库存、现货和工厂发布，但随着需求方向和辅助资源类型增加，固定字段已经开始和类型字段冲突：

- 找服务、配套服务中的“品类”应是“服务类型”。
- 找场地、出租转让中的“品类”应是“标的类型/房源类型”。
- 找工厂、工厂接单中的“数量/产能”会和“订单数量/日产能/起订量”重复。
- 招工招聘中的“价格描述”应是“工价/薪资”，而不是资源报价。
- 卡片和详情页固定展示“品类 · 数量”“价格”，对服务、招聘、场地等类型不自然。

项目当前未上线，本次可以直接重构字段配置、迁移种子和前端展示逻辑，不需要兼容旧发布数据。

## 目标

1. 发布页不再用一套供给资源字段硬套所有资源和需求类型。
2. 每个类型的业务字段由 `resource_type_configs.field_schema` 配置驱动。
3. `category`、`quantityText`、`priceText` 保留为系统摘要字段，用于搜索、筛选、列表卡片、详情和后台审核，但不要求所有类型都直接手填。
4. 资源和需求使用同一套配置化表单能力，减少后续新增类型时的前端特殊判断。
5. 小程序列表、详情、我的发布和后台配置说明统一使用类型语义，避免“采购要求”“品类”“数量”误导用户。

## 非目标

- 不新增独立的采购需求表。
- 不拆分资源和需求的生命周期，仍复用统一 `resources` 模型。
- 不实现复杂公式引擎；摘要字段只做简单的字段映射和兜底拼接。
- 不做线上数据迁移兼容；重置开发数据或重新执行迁移后应得到新的字段结构。

## 总体方案

### 1. 发布页字段分层

发布页固定保留真正通用的字段：

- 类型选择：`typeCode`
- 标题：`title`
- 联系人：`contactName`
- 联系电话：`contactPhone`
- 图片：`images`
- 详情描述：`description`

业务差异字段全部来自 `field_schema.fields`。前端根据字段类型渲染文本、数字、下拉、布尔、文本域等控件，并根据 `required_fields` 控制必填。

`category`、`quantityText`、`priceText` 不再默认作为固定输入出现在所有类型中。某个类型确实需要用户直接填这些字段时，可通过基础字段配置显式展示；默认推荐改为从动态字段映射生成。

### 2. 摘要字段映射

在 `resource_type_configs.display_template` 中增加 `summary` 配置，用于声明系统摘要字段来自哪些业务字段：

```json
{
  "summary": {
    "category": "serviceType",
    "quantityText": "serviceArea",
    "priceText": "priceMode"
  },
  "list": ["serviceType", "serviceArea", "priceMode"],
  "detail": ["serviceType", "serviceArea", "leadTime", "caseAvailable"]
}
```

规则：

- `summary.category` 映射到 `resources.category`，供搜索、筛选和卡片主信息使用。
- `summary.quantityText` 映射到 `resources.quantity_text`，供卡片第二摘要位使用。
- `summary.priceText` 映射到 `resources.price_text`，供价格、预算、租金、工价等摘要使用。
- 如果映射字段不存在或为空，前端提交前和后端入库前都应使用类型级兜底文案，例如“待沟通”“面议”。
- 后端必须重新计算摘要字段，不能完全信任前端传入，避免后台代发、接口调用和未来客户端不一致。

### 3. 字段矩阵

#### 供给资源

| 类型 | 用户可见业务字段 | 摘要映射 |
| --- | --- | --- |
| 库存清仓 `inventory` | 品类、季节、尺码段、库存数量、打包价/单件价、支持拿样、支持直播、仓库位置 | `category=categoryName`，`quantityText=stockQuantity`，`priceText=stockPriceText` |
| 现货货源 `goods` | 品类、风格、价格带/供货价、起批量、是否现货、一件代发、供货区域 | `category=categoryName`，`quantityText=minOrderQuantity`，`priceText=supplyPriceText` |
| 工厂接单 `factory` | 擅长品类、日产能、起订量、接小单、空档期、工价范围、加工方式 | `category=categoryName`，`quantityText=dailyCapacity`，`priceText=processingPriceText` |
| 招工招聘 `job` | 岗位、工价/薪资、人数、包吃住、工作地点、结算方式、班次 | `category=position`，`quantityText=headcount`，`priceText=payText` |
| 出租转让 `rental` | 房源类型、面积、租金、楼层、位置、转让费、租期条件 | `category=rentalType`，`quantityText=areaText`，`priceText=rentText` |
| 配套服务 `service` | 服务类型、服务范围、价格方式、响应时效、服务案例、可服务区域 | `category=serviceType`，`quantityText=serviceArea`，`priceText=priceMode` |

#### 需求资源

| 类型 | 用户可见业务字段 | 摘要映射 |
| --- | --- | --- |
| 找现货 `buy_goods` | 目标品类、目标风格、需求数量、预算范围、交付时效、接受外地发货 | `category=targetCategory`，`quantityText=demandQuantity`，`priceText=budgetRange` |
| 找库存 `find_inventory` | 目标品类、季节、尺码段、需求数量、预算范围、接受混批 | `category=targetCategory`，`quantityText=demandQuantity`，`priceText=budgetRange` |
| 找工厂 `find_factory` | 加工品类、订单数量、交期、需要打样、长期合作、工艺要求 | `category=productCategory`，`quantityText=orderQuantity`，`priceText=budgetRange` |
| 找服务 `find_service` | 服务类型、服务区域、预算范围、交付时效、长期合作 | `category=serviceType`，`quantityText=serviceArea`，`priceText=budgetRange` |
| 找场地 `find_rental` | 标的类型、期望面积、预算租金、期望区域、接受转让费、入驻时间 | `category=rentalNeedType`，`quantityText=expectedAreaText`，`priceText=budgetRentText` |

字段命名原则：

- 供给侧表达“我有什么”：`stockQuantity`、`supplyPriceText`、`dailyCapacity`、`rentText`。
- 需求侧表达“我要什么”：`demandQuantity`、`budgetRange`、`expectedAreaText`、`budgetRentText`。
- 不再让多个动态字段和基础字段表达同一个含义，例如 `quantityText` 与 `orderQuantity` 同时让用户填写。

### 4. 小程序发布页

发布页结构调整为：

1. 基础信息：类型、标题。
2. 类型字段：按 `field_schema.fields` 渲染，标题根据方向显示“资源信息”或“需求要求”，按类型可覆盖为“场地要求”“招聘信息”等。
3. 详情描述：可选文本域，提示语按供给/需求切换。
4. 图片：供给侧显示“资源图片”，需求侧显示“参考图片”。
5. 联系信息：联系人、联系电话。

完成度、必填校验、toast 文案都应读取配置字段标签，不能再写死“请填写品类”。

### 5. 列表和详情展示

资源卡片和需求卡片展示改为摘要驱动：

- 第一行：标题。
- 标签：资源类型/需求类型。
- 摘要行：从 `display_template.list` 取 2 到 3 个字段，优先使用后端返回的摘要字段和值。
- 价格行：如果 `priceText` 有值，显示该值；否则不显示固定“价格面议”文案。

详情页规格列表改为：

1. 优先显示 `display_template.detail` 对应的动态字段。
2. 再显示描述、图片、商家/发布方信息。
3. 只有当摘要字段不是动态字段重复来源时，才展示 `category/quantityText/priceText`。

### 6. 后端与配置

后端创建和更新资源时执行摘要归一化：

1. 读取类型配置中的 `display_template.summary`。
2. 从 `attributes` 中取映射字段值。
3. 写入 `CreateResourceInput.Category`、`QuantityText`、`PriceText`。
4. 如果配置中没有 summary，则保留兼容路径：使用请求中的基础摘要字段。
5. `required_fields` 校验基于配置字段执行，不再在前端和后端默认强制 `category`。

后台资源类型配置页需要支持 `summary` 的 JSON 配置。可视化编辑器可以先不做完整 UI，但高级 JSON 必须保留和校验 summary 引用字段合法。

### 7. 测试与验证

需要覆盖：

- 小程序发布页不再固定展示“品类/数量/价格”给所有类型。
- 找服务、找工厂、找场地只展示对应业务字段。
- 招工招聘、出租转让、配套服务不再出现不自然的固定字段。
- 后端创建资源时按 summary 生成 `category/quantityText/priceText`。
- 搜索、列表、详情仍能展示摘要字段。
- 后台配置校验拒绝引用不存在的 summary 字段。

推荐验证命令：

```bash
node --test wxapp/scripts/validate-flows.test.mjs
npm --prefix wxapp run validate:flows
npm --prefix wxapp run validate:pages
go test ./backend/app/internal/logic/resource ./backend/app/internal/logic/admin ./backend/app/internal/logic/city ./backend/app/internal/server
node --test backend/scripts/validate_migrations.test.mjs
```

## 实施边界

本重构应优先修改以下文件：

- `backend/migrations/000003_seed_zhili.up.sql`
- `backend/migrations/000014_resource_type_display_names.up.sql`
- `backend/migrations/000015_find_rental_resource_type.up.sql`
- `backend/app/internal/logic/resource/create_resource_logic.go`
- `backend/app/internal/logic/admin/resource_type_config_logic.go`
- `wxapp/components/ResourcePublishForm.vue`
- `wxapp/components/ResourceCard.vue`
- `wxapp/components/DemandCard.vue`
- `wxapp/pages/resource/detail.vue`
- `wxapp/pages/my-resources/index.vue`
- 对应测试文件

不应在本次顺手重构无关页面布局、认证流程、地图、权益或消息功能。

## 风险

- 字段配置较多，迁移 SQL 容易出现 JSON 拼写错误，必须用迁移静态测试兜底。
- 前端发布表单依赖配置加载，空配置或错误配置时需要给出明确提示。
- 卡片展示从固定字段切到配置驱动后，需要保证缺失字段不会出现空白分隔符。
- 后台配置高级 JSON 的自由度较高，后端校验必须阻止 summary 引用不存在字段。

## 通过标准

- 发布资源和发布需求页面都不再普遍出现不符合类型语义的“品类/数量/价格”固定输入。
- 每个资源/需求类型的必填项都来自字段矩阵和配置。
- 资源卡片、需求卡片、详情页展示与类型语义一致。
- 后端入库后的 `category/quantityText/priceText` 可继续支撑搜索、列表和后台审核。
- 所有推荐验证命令通过，或明确记录不能运行的环境原因。
