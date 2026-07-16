# 商户资料绑定拿货地图点位设计

## 目标

让已入驻商户能从小程序商户资料入口快速申请绑定拿货地图上的档口点位。平台运营审核通过后，系统把地图点位和现有商户主体关联起来，使认证商户在拿货地图上高亮展示，并能从点位进入商户资料体系。

本次不做商户直接改地图坐标，不做自动经纬度落点，也不做复杂认领仲裁。地图坐标仍由后台运营维护，商户只选择已有点位并提交证明材料。

## 当前基础

现有地图能力已经具备：

- `map_object.merchant_id` 可关联 `merchants.id`。
- 后台地图点位表单已有“绑定商户”字段。
- 小程序公开地图会在商户已认证时把点位展示为高亮商户点位。
- 商户资料页已有地址、外部导航定位和商户图片能力。

缺口在于：普通商户没有从小程序发起点位绑定申请的链路，后台也没有集中审核这些申请的入口。

## 推荐方案

采用“商户提交绑定申请 + 后台审核通过后写入 `map_object.merchant_id`”。

### 为什么不直接自动绑定

拿货地图使用的是底图坐标，商户资料里的 `location` 是外部经纬度，两套坐标不能直接互相替代。直接按经纬度自动落点会带来偏移、楼层错误和抢占档口风险。

### 为什么不完全靠后台

纯后台绑定准确但运营成本高，商户无法主动补充自己所在档口。申请审核模式能让商户提供线索，仍由运营兜底确认准确性。

## 功能范围

### 小程序商户侧

- 在商户资料页增加“拿货地图档口”模块。
- 展示当前绑定状态：未绑定、审核中、已绑定、已驳回。
- 提供绑定入口，进入点位选择页面。
- 点位选择页支持按场景、关键词搜索档口候选。
- 用户选择档口后填写备注，并提交门头或档口照片 URL 列表。
- 提交成功后显示待审核状态。

### 后台运营侧

- 在拿货地图页面新增“绑定审核”标签页。
- 运营可按状态查看申请。
- 审核通过时写入 `map_object.merchant_id`。
- 审核驳回时保存驳回原因。
- 若目标点位已绑定其他商户，审核通过必须失败，提示运营先处理原绑定。

### 后端

- 新增 `map_object_bind_request` 表保存申请。
- 新增商户侧接口：查询绑定状态、查询候选点位、提交绑定申请。
- 新增后台接口：列表查询申请、审核申请。
- 继续复用现有 `map_object.merchant_id` 作为最终绑定关系。

## 数据模型

### map_object_bind_request

字段：

- `id bigint primary key default next_tsid()`
- `merchant_id bigint references merchants(id) not null`
- `object_id bigint references map_object(id) not null`
- `scene_code varchar(64) not null`
- `applicant_user_id bigint references users(id)`
- `evidence_images jsonb not null default '[]'::jsonb`
- `note text not null default ''`
- `status varchar(20) not null default 'pending'`
- `review_note text not null default ''`
- `reviewed_by bigint references admin_operators(id)`
- `reviewed_at timestamptz`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

状态：

- `pending`：待审核。
- `approved`：已通过，并已写入 `map_object.merchant_id`。
- `rejected`：已驳回。
- `cancelled`：后续预留，首期不开放。

约束：

- 同一商户同一点位同一时间只允许一条 `pending` 申请。
- 通过审核时如果点位已绑定其他商户，返回冲突错误，不自动覆盖。

## API 设计

### 商户侧

- `GET /api/v1/merchants/{merchantId}/map-binding`
  - 返回当前已绑定点位和最新申请。
  - 需要当前用户能管理该商户。

- `GET /api/v1/map/bind-candidates`
  - 参数：`merchantId`、`sceneCode`、`keyword`、`limit`。
  - 返回已发布场景下的正常档口候选。
  - 优先展示未绑定点位，但保留已绑定标识，避免用户误选。

- `POST /api/v1/merchants/{merchantId}/map-binding-requests`
  - 请求：`objectId`、`note`、`evidenceImages`。
  - 需要当前用户能管理该商户。

### 后台侧

- `GET /api/v1/admin/map/bind-requests`
  - 参数：`status`、`keyword`、`page`、`pageSize`。
  - 返回申请列表。

- `POST /api/v1/admin/map/bind-requests/{requestId}/review`
  - 请求：`action=approve|reject`、`reviewNote`。
  - 通过时绑定点位；驳回时必须填写原因。

## 权限和错误处理

- 商户侧接口复用 `requireMerchantPermission`，禁止用户替非本人管理的商户申请。
- 后台接口沿用后台全局 token 包装能力。
- 绑定申请失败时返回中文用户可理解错误，例如“该档口已绑定其他商家，请联系平台处理”。
- 后端日志记录 `merchantId`、`objectId`、`requestId`、`reviewerId`、状态和错误原因，不记录完整图片 URL 列表。

## 验收标准

- 商户资料页能看到地图档口绑定状态和绑定入口。
- 商户能搜索已发布地图档口并提交绑定申请。
- 重复提交同一待审核申请会被拦截。
- 后台能看到待审核申请并通过或驳回。
- 通过后 `map_object.merchant_id` 被写入，商户地图绑定状态变为已绑定。
- 点位已绑定其他商户时审核通过失败，不覆盖原绑定。
- 小程序公开地图继续按现有规则高亮已认证商户点位。
