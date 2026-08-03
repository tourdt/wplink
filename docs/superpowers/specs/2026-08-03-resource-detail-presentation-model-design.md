# 供需详情配置驱动渲染模型设计

## 目标

将详情展示规则从前端摘要拼装迁移到资源类型配置和后端。详情接口返回已解析、已排序、可直接渲染的字段列表，前端不再处理核心字段去重、供需价格文案或单双列推断。

## 配置模型

`displayTemplate.fields` 为有序字段定义：

```json
{
  "source": "minOrderText",
  "label": "数量/面积",
  "role": "core",
  "layout": "half",
  "order": 10
}
```

- `source`：属性 key 或资源顶层字段名；
- `label`：详情展示名称；
- `role`：`core`、`core_price`、`detail`；
- `layout`：`half`、`full`；
- `order`：字段展示顺序。

## 接口模型

详情接口返回 `presentation.fields`，每项包含 `key`、`label`、`value`、`layout`。后端根据资源类型快照解析配置、读取值、过滤空值和排序。前端直接循环渲染。

## 不保留兼容

系统尚未上线，移除详情接口的 `quantityText`、`priceText`、`summarySourceKeys`、`attributeItems` 及前端对应拼装代码。不新增过渡分支。

## 验收

1. 同源字段只由配置生成一次。
2. 新类型调整字段名称、顺序和布局无需改前端。
3. 地址继续作为独立区块，不进入 `presentation.fields`。
4. 前端详情页不再包含核心字段去重或整行推断规则。
