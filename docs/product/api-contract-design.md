# 服装产业带资源撮合平台 API 契约设计

版本：v0.1  
日期：2026-06-27  
输入文档：

- `docs/product/apparel-industry-platform-prd.md`
- `docs/product/domain-model-ddd.md`
- `docs/product/database-er-design.md`

## 1. 设计目标

API 设计目标：

- 支持小程序端、管理后台和后续城市站扩展。
- 保持 `resources` 统一资源模型，避免为库存、招聘、出租等业务各自设计一套接口。
- 请求和响应使用稳定字段，业务扩展字段放入 `attributes`。
- 所有用户可见错误使用中文友好提示，不暴露 SQL、堆栈、内部表字段、令牌或敏感数据。
- MVP 先覆盖注册登录、城市站、商家、资源、采购需求、认证、权益、消息、发布效果和后台审核。

默认约定：

- Base URL: `/api/v1`
- 数据格式：JSON
- 时间格式：ISO 8601
- ID 类型：TSID 字符串（数据库 BIGINT，JSON 返回字符串）
- 认证方式：`Authorization: Bearer <token>`
- 分页方式：`page` + `pageSize`

## 2. 通用响应

### 2.1 成功响应

列表响应：

```json
{
  "items": [],
  "page": 1,
  "pageSize": 20,
  "total": 0
}
```

对象响应：

```json
{
  "id": "tsid",
  "createdAt": "2026-06-27T10:00:00+08:00"
}
```

### 2.2 错误响应

```json
{
  "code": "RESOURCE_NOT_FOUND",
  "message": "资源不存在或已下架",
  "requestId": "req_20260627100000"
}
```

错误设计原则：

- `message` 面向用户，必须中文、明确、可操作。
- `code` 面向前端分支处理，使用稳定英文枚举。
- 后端日志记录真实错误和安全上下文，接口不返回内部错误细节。

常用错误：

| HTTP | code | message |
|---:|---|---|
| 400 | VALIDATION_FAILED | 请检查提交内容后重试 |
| 401 | UNAUTHORIZED | 请先登录 |
| 403 | FORBIDDEN | 您没有权限进行此操作 |
| 404 | RESOURCE_NOT_FOUND | 资源不存在或已下架 |
| 404 | MERCHANT_NOT_FOUND | 商家不存在或已停用 |
| 409 | STATE_CONFLICT | 状态已变化，请刷新后重试 |
| 409 | QUOTA_NOT_ENOUGH | 可用额度不足，请升级权益或购买套餐 |
| 422 | REVIEW_REQUIRED | 关键内容已变更，需要重新提交审核 |
| 429 | RATE_LIMITED | 操作过于频繁，请稍后再试 |
| 500 | INTERNAL_ERROR | 操作失败，请稍后重试 |

## 3. 认证与账号

### 3.1 微信登录

`POST /auth/wechat-login`

请求：

```json
{
  "code": "wechat_login_code",
  "defaultCityCode": "zhili"
}
```

响应：

```json
{
  "token": "jwt",
  "user": {
    "id": "tsid",
    "nickname": "张三",
    "avatarUrl": "https://example.com/avatar.png",
    "defaultCityCode": "zhili",
    "roles": ["normal_user"]
  }
}
```

失败：

- 微信登录失败：`登录失败，请稍后重试`
- 城市站不存在：`城市站不存在或暂未开通`

### 3.2 获取当前用户

`GET /me`

响应：

```json
{
  "id": "tsid",
  "phone": "138****0000",
  "nickname": "张三",
  "defaultCityCode": "zhili",
  "roles": ["normal_user", "merchant_admin"],
  "managedMerchants": [
    {
      "id": "tsid",
      "name": "织里样板童装厂",
      "role": "owner"
    }
  ]
}
```

### 3.3 绑定手机号

`POST /me/phone`

请求：

```json
{
  "phone": "13800000000",
  "smsCode": "123456"
}
```

响应：

```json
{
  "id": "tsid",
  "phone": "138****0000"
}
```

## 4. 城市站与配置

### 4.1 城市站列表

`GET /city-stations`

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "code": "zhili",
      "name": "织里",
      "primaryCategory": "童装",
      "status": "active"
    }
  ]
}
```

### 4.2 城市站资源类型

`GET /city-stations/{cityCode}/resource-types`

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "typeCode": "inventory",
      "typeName": "库存",
      "defaultValidDays": 7,
      "requiredFields": ["title", "category", "quantityText", "contactPhone"],
      "filterFields": ["season", "sizeRange", "allowLiveSale"],
      "displayTemplate": {
        "list": ["priceText", "quantityText", "district"],
        "detail": ["season", "sizeRange", "allowSample", "allowLiveSale"]
      }
    }
  ]
}
```

## 5. 商家

### 5.1 创建商家

`POST /merchants`

请求：

```json
{
  "cityCode": "zhili",
  "name": "织里样板童装厂",
  "merchantType": "factory",
  "mainCategories": ["童装", "卫衣", "套装"],
  "contactName": "李厂长",
  "contactPhone": "13800000000",
  "contactWechat": "zhili_factory",
  "addressText": "湖州织里镇",
  "description": "主做中小童卫衣和套装，支持小单快反。"
}
```

响应：

```json
{
  "id": "tsid",
  "name": "织里样板童装厂",
  "verificationStatus": "unverified",
  "status": "active"
}
```

### 5.2 商家详情

`GET /merchants/{merchantId}`

响应：

```json
{
  "id": "tsid",
  "name": "织里样板童装厂",
  "merchantType": "factory",
  "cityCode": "zhili",
  "mainCategories": ["童装", "卫衣"],
  "verificationStatus": "verified",
  "creditTags": [
    { "code": "verified_factory", "label": "已认证工厂" },
    { "code": "recent_active", "label": "近期活跃" }
  ],
  "contact": {
    "name": "李厂长",
    "phoneMasked": "138****0000",
    "wechatMasked": "zhili_****"
  },
  "resourcesSummary": {
    "publishedCount": 12,
    "dealtCount": 3
  }
}
```

### 5.3 更新商家主页

`PATCH /merchants/{merchantId}`

权限：商家管理员、平台运营、超级管理员。

请求：

```json
{
  "mainCategories": ["童装", "卫衣", "套装"],
  "description": "支持小单快反，可打样。",
  "images": ["https://example.com/factory.jpg"]
}
```

失败：

- 非商家管理员：`您没有权限操作该商家`
- 商家被停用：`商家已停用，无法修改资料`

## 6. 资源

### 6.1 发布资源

`POST /resources`

权限：商家管理员、平台运营。

请求：

```json
{
  "merchantId": "tsid",
  "cityCode": "zhili",
  "typeCode": "inventory",
  "title": "女童春款卫衣库存整包清",
  "category": "童装",
  "district": "织里童装城周边",
  "priceText": "打包 18 元/件",
  "quantityText": "3200 件",
  "description": "女童春款卫衣，90-140 码，整包优先，可现场看货。",
  "attributes": {
    "season": "春款",
    "sizeRange": "90-140",
    "allowSample": true,
    "allowLiveSale": true
  },
  "tags": ["急清", "支持看货", "可直播"],
  "images": [],
  "contact": {
    "name": "张老板",
    "phone": "13800000000",
    "wechat": "zhili_stock"
  }
}
```

响应：

```json
{
  "id": "tsid",
  "status": "pending",
  "message": "已提交审核，审核通过后将展示给买家"
}
```

失败：

- 字段缺失：`请补充库存数量或产能信息`
- 无发布权限：`您没有权限以该商家身份发布资源`
- 发布额度不足：`可用发布额度不足，请升级权益或购买套餐`

### 6.2 资源列表

`GET /resources`

查询参数：

| 参数 | 说明 |
|---|---|
| cityCode | 城市站编码 |
| typeCode | 资源类型 |
| keyword | 关键词 |
| category | 品类 |
| verifiedOnly | 是否只看认证资源 |
| page | 页码 |
| pageSize | 每页数量 |

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "typeCode": "inventory",
      "title": "女童春款卫衣库存整包清",
      "category": "童装",
      "district": "织里童装城周边",
      "priceText": "打包 18 元/件",
      "quantityText": "3200 件",
      "merchant": {
        "id": "tsid",
        "name": "织里样板童装厂",
        "verificationStatus": "verified"
      },
      "creditTags": ["已认证库存", "近期活跃"],
      "refreshedAt": "2026-06-27T10:00:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

### 6.3 资源详情

`GET /resources/{resourceId}`

响应：

```json
{
  "id": "tsid",
  "status": "published",
  "typeCode": "inventory",
  "title": "女童春款卫衣库存整包清",
  "category": "童装",
  "description": "女童春款卫衣，90-140 码，整包优先。",
  "priceText": "打包 18 元/件",
  "quantityText": "3200 件",
  "attributes": {
    "season": "春款",
    "sizeRange": "90-140",
    "allowSample": true
  },
  "merchant": {
    "id": "tsid",
    "name": "织里样板童装厂",
    "verificationStatus": "verified"
  },
  "contact": {
    "name": "张老板",
    "phoneMasked": "138****0000",
    "wechatMasked": "zhili_****"
  },
  "publishedAt": "2026-06-27T10:00:00+08:00",
  "expiresAt": "2026-07-04T10:00:00+08:00"
}
```

### 6.4 刷新资源

`POST /resources/{resourceId}/refresh`

权限：资源所属商家管理员。

响应：

```json
{
  "id": "tsid",
  "refreshedAt": "2026-06-27T11:00:00+08:00",
  "remainingRefreshQuota": 2
}
```

失败：

- 资源不可刷新：`当前资源状态不支持刷新`
- 刷新额度不足：`今日免费刷新次数已用完`

### 6.5 标记成交

`POST /resources/{resourceId}/deal-feedback`

请求：

```json
{
  "isDealt": true,
  "isReal": true,
  "responseTimely": true,
  "willingToCooperateAgain": true,
  "note": "已对接，准备看样"
}
```

响应：

```json
{
  "id": "tsid",
  "status": "dealt",
  "message": "已记录成交反馈"
}
```

### 6.6 联系资源

`POST /resources/{resourceId}/contact-events`

请求：

```json
{
  "action": "phone"
}
```

`action` 可选：

- `phone`
- `wechat`
- `merchant_profile`
- `share`

响应：

```json
{
  "message": "已记录联系行为"
}
```

## 7. 采购需求

### 7.1 提交采购需求

`POST /purchase-demands`

请求：

```json
{
  "cityCode": "zhili",
  "demandType": "inventory",
  "title": "找 100-140 码女童卫衣库存",
  "category": "童装",
  "priceRange": {
    "min": 10,
    "max": 25
  },
  "quantityRequirement": {
    "quantity": 2000,
    "unit": "件"
  },
  "attributes": {
    "season": "春款",
    "sizeRange": "100-140",
    "allowLiveSale": true
  },
  "contact": {
    "name": "王老板",
    "phone": "13800000000",
    "wechat": "buyer001"
  }
}
```

响应：

```json
{
  "id": "tsid",
  "status": "pending",
  "message": "需求已提交，平台会尽快为您匹配"
}
```

### 7.2 我的采购需求

`GET /me/purchase-demands`

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "title": "找 100-140 码女童卫衣库存",
      "status": "matching",
      "createdAt": "2026-06-27T10:00:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

## 8. 认证

### 8.1 提交商家认证

`POST /merchants/{merchantId}/verifications`

请求：

```json
{
  "verificationType": "factory",
  "businessName": "织里样板童装厂",
  "licenseUrl": "https://example.com/license.jpg",
  "storefrontUrl": "https://example.com/factory.jpg",
  "materials": {
    "locationText": "湖州织里镇",
    "realVideoUrl": "https://example.com/video.mp4"
  }
}
```

响应：

```json
{
  "id": "tsid",
  "status": "pending",
  "message": "认证资料已提交，请等待审核"
}
```

### 8.2 查看认证状态

`GET /merchants/{merchantId}/verifications/latest`

响应：

```json
{
  "id": "tsid",
  "verificationType": "factory",
  "status": "approved",
  "reviewedAt": "2026-06-27T10:00:00+08:00"
}
```

## 9. 商家权益与置顶

### 9.1 查看商家权益

`GET /merchants/{merchantId}/entitlements`

响应：

```json
{
  "items": [
    {
      "type": "posting_quota",
      "sourceType": "verification_gift",
      "totalAmount": 20,
      "usedAmount": 3,
      "remainingAmount": 17,
      "expiresAt": "2026-07-31T23:59:59+08:00"
    },
    {
      "type": "refresh_quota",
      "totalAmount": 90,
      "usedAmount": 12,
      "remainingAmount": 78
    }
  ]
}
```

### 9.2 查看置顶券

`GET /merchants/{merchantId}/top-vouchers`

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "status": "unused",
      "topDurationHours": 24,
      "allowedTypeCodes": ["inventory", "goods", "factory"],
      "expiresAt": "2026-07-31T23:59:59+08:00"
    }
  ]
}
```

### 9.3 使用置顶券

`POST /top-vouchers/{voucherId}/redeem`

请求：

```json
{
  "resourceId": "tsid"
}
```

响应：

```json
{
  "voucherId": "tsid",
  "resourceId": "tsid",
  "status": "used",
  "message": "置顶券已使用"
}
```

失败：

- 资源不是当前商家的：`该置顶券不能用于此资源`
- 资源未发布：`资源审核通过后才能置顶`
- 券已过期：`置顶券已过期`

## 10. 发布效果

### 10.1 单条资源效果

`GET /resources/{resourceId}/metrics`

权限：资源所属商家管理员、平台运营。

查询参数：

| 参数 | 说明 |
|---|---|
| from | 开始日期 |
| to | 结束日期 |

响应：

```json
{
  "resourceId": "tsid",
  "summary": {
    "exposureCount": 1200,
    "detailViewCount": 130,
    "phoneClickCount": 12,
    "wechatCopyCount": 9,
    "dealFeedbackCount": 2
  },
  "daily": [
    {
      "date": "2026-06-27",
      "exposureCount": 200,
      "detailViewCount": 30,
      "phoneClickCount": 3,
      "wechatCopyCount": 2
    }
  ]
}
```

### 10.2 商家效果总览

`GET /merchants/{merchantId}/metrics/summary`

响应：

```json
{
  "merchantId": "tsid",
  "publishedResourceCount": 12,
  "expiringResourceCount": 3,
  "dealtResourceCount": 2,
  "last7Days": {
    "exposureCount": 3600,
    "detailViewCount": 420,
    "contactClickCount": 35
  }
}
```

## 11. 消息

### 11.1 我的消息

`GET /messages`

查询参数：

| 参数 | 说明 |
|---|---|
| type | review, lifecycle, interaction, matching, system |
| status | pending, sent, read |

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "messageType": "review",
      "title": "资源审核通过",
      "content": "您发布的女童春款卫衣库存已审核通过。",
      "targetUrl": "/resources/tsid",
      "status": "sent",
      "createdAt": "2026-06-27T10:00:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

### 11.2 标记已读

`POST /messages/{messageId}/read`

响应：

```json
{
  "id": "tsid",
  "status": "read"
}
```

## 12. 管理后台 API

后台接口统一以 `/admin` 开头。

### 12.0 管理后台登录

`POST /admin/auth/login`

请求：

```json
{
  "loginName": "13800000000",
  "password": "secret123"
}
```

响应：

```json
{
  "token": "admin_jwt",
  "userId": "tsid",
  "roles": ["platform_operator"]
}
```

规则：

- 后台不开放注册，只能由超级管理员创建账号。
- 登录账号关联统一 `users` 用户主体。
- 只有 `platform_operator` 或 `super_admin` 角色允许登录后台。
- 密码只保存 bcrypt 哈希，不保存明文。
- 账号停用、密码错误、无后台权限都必须返回中文友好错误，不暴露内部表结构。

### 12.1 待审核资源列表

`GET /admin/resources/pending`

查询参数：

| 参数 | 说明 |
|---|---|
| cityCode | 城市站 |
| typeCode | 资源类型 |
| page | 页码 |
| pageSize | 每页数量 |

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "title": "女童春款卫衣库存整包清",
      "typeCode": "inventory",
      "merchantName": "织里样板童装厂",
      "createdAt": "2026-06-27T10:00:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

### 12.2 审核资源

`POST /admin/resources/{resourceId}/review`

请求：

```json
{
  "action": "approve",
  "reason": ""
}
```

`action` 可选：

- `approve`
- `reject`
- `take_down`

响应：

```json
{
  "id": "tsid",
  "status": "published",
  "message": "资源已审核通过"
}
```

失败：

- 状态变化：`状态已变化，请刷新后重试`
- 权限不足：`您没有权限审核该城市站资源`

### 12.3 认证审核

`GET /admin/verifications/pending`

`POST /admin/verifications/{verificationId}/review`

请求：

```json
{
  "action": "approve",
  "reviewNote": "资料真实，认证通过"
}
```

响应：

```json
{
  "id": "tsid",
  "status": "approved",
  "message": "认证已通过"
}
```

### 12.4 发放商家权益

`POST /admin/merchants/{merchantId}/entitlements`

请求：

```json
{
  "entitlementType": "posting_quota",
  "sourceType": "operator_grant",
  "totalAmount": 10,
  "expiresAt": "2026-07-31T23:59:59+08:00",
  "reason": "首批认证商家扶持"
}
```

响应：

```json
{
  "id": "tsid",
  "message": "权益已发放"
}
```

### 12.5 人工撮合

`POST /admin/match-cases`

请求：

```json
{
  "purchaseDemandId": "tsid",
  "resourceIds": ["tsid"],
  "participantMerchantIds": ["tsid"],
  "resultNote": "已联系双方，等待看样"
}
```

响应：

```json
{
  "id": "tsid",
  "status": "open",
  "message": "撮合记录已创建"
}
```

### 12.6 操作日志

`GET /admin/operation-logs`

查询参数：

| 参数 | 说明 |
|---|---|
| objectType | resource, merchant, credit, entitlement |
| objectId | 对象 ID |
| operatorId | 操作人 |

响应：

```json
{
  "items": [
    {
      "id": "tsid",
      "operatorId": "tsid",
      "operatorRole": "operator",
      "objectType": "resource",
      "objectId": "tsid",
      "action": "approve",
      "reason": "信息完整",
      "createdAt": "2026-06-27T10:00:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

## 13. 日志要求

后端日志服务工程排查，不直接返回给前端。

必须记录日志的场景：

- 资源审核通过、驳回、下架。
- 发布额度扣减失败。
- 置顶券核销失败。
- 商家认证通过或驳回。
- 权限拒绝。
- 采购需求进入人工撮合。
- 资源状态并发冲突。

日志字段建议：

- `requestId`
- `operation`
- `userId`
- `merchantId`
- `resourceId`
- `cityCode`
- `currentStatus`
- `targetStatus`
- `error`

禁止记录：

- 完整手机号
- 完整微信号
- 身份证、营业执照原图私密 URL
- token
- 原始授权 header

## 14. MVP 接口范围

MVP 必做接口：

- 微信登录
- 当前用户
- 城市站列表
- 城市站资源类型
- 创建商家
- 商家详情
- 更新商家主页
- 发布资源
- 资源列表
- 资源详情
- 刷新资源
- 联系资源
- 提交采购需求
- 提交认证
- 查看认证状态
- 查看商家权益
- 查看置顶券
- 使用置顶券
- 单条资源效果
- 商家效果总览
- 消息列表
- 后台资源审核
- 后台认证审核
- 后台权益发放
- 后台人工撮合
- 后台操作日志

V1.1 再做：

- 收藏资源
- 关注商家
- 关注品类/搜索条件
- 新货提醒
- 商家多管理员邀请
- 复杂会员套餐购买
- 自动推荐撮合

## 15. 待确认问题

1. 微信登录是否 MVP 唯一登录方式，还是同时支持短信验证码登录。
2. 联系电话和微信是否在点击前只展示脱敏信息。
3. 普通用户能否发布资源，还是必须先创建商家。
4. 采购需求 MVP 是否前台公开，还是只进入后台撮合。
5. 置顶券使用后是否需要单独的置顶结束时间字段。
6. 后台接口是否需要按城市站做数据权限隔离。
