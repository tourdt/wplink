# 服装地图首发界面实施计划

用户已确认两入口五核心页面方案，开始实施。沿用当前工作区中已完成但未提交的商家重构，不覆盖已有改动。使用 frontend-design、backend-friendly-errors-logging 和子任务实施技能，按明确文件边界协作。

## 页面与能力约定

- 默认页 pages/sourcing-map/index，Tab 只有找档口、我的；保留旧源码但移除业务页面注册，pages/home/index 保留兼容跳转。
- 找档口：真实图片、主营、醒目的市场楼层档口号；主营使用 common/merchantCategories.js 的统一选项，保留关键词搜索与分页防竞态。
- 店铺主页：照片、主营、简介、地址地图、分享、固定底部导航及商家授权公开的电话/微信，不读取资源、权益、关注。
- 我的：登录、加载失败、未完善、已入驻、暂停或位置异常分别表达；只请求商家资料；预览分享、编辑、暂停恢复、账号隐私。
- 入驻：单页分组，主营选择、实拍、选点及详细地址、联系公开开关、提交入驻/保存修改；草稿为当前设备按 userId+merchantId 隔离存储，明确本机保存，登出清理；保存草稿上传临时照片为稳定URL，不提交公开资料。
- 后端最小配套：保留完整资料保存契约，新增可选 publicPhone/publicWechat 布尔配置及 ownerVisible 店铺展示开关。默认联系方式不公开，店铺默认可展示兼容历史；ownerVisible=false 时目录/位置不可见且非管理者读详情返回安全不可用。单独 POST /merchants/:merchantId/visibility 接收 visible，避免用全表单保存切状态。返回 ownerVisible/publicPhone/publicWechat（管理者读取），public contact 只包含获准字段；不返回私有联系人姓名。
- /merchants/:id 更新不应重置省略的公开配置；用指针解析可选布尔；生成契约按仓库 goctl 流程。不修改资源发布核心。

## 所有权与接口

1. 后端代理：backend 下最小迁移、api、model、logic、handler及测试；wxapp/api/merchant.js 添加公开字段白名单和 setMerchantVisibility(id,visible)。不得修改其他wxapp文件。
2. 页面代理：wxapp/pages/my/index.vue、merchant/detail.vue 及对应测试和必要的 common/merchantStorefront.js；可维护本人新增文件，不修改 profile.vue、pages.json 或 api。依赖上述字段和 setMerchantVisibility。
3. 主任务：目录/卡片、profile.vue、分类/草稿助手、导航注册与登录回跳、流程校验、视觉验证及整合。

## 验证

各子任务先添加有意义的状态/回归测试；小程序完整 check、后端受影响包测试、生成一致性与迁移静态检查。能使用浏览器时渲染真实Vue组件预览，微信行为以开发者工具/真机联调为最终依据。日志不打印联系信息和完整请求。旧资源发布源码保留。

## 进度

- [x] 后端店铺公开配置和暂停恢复
- [x] 我的与店铺主页
- [x] 列表、入驻和导航
- [x] 集成回归、视觉检查、审查与交付说明

## 验证记录与上线边界

- 小程序 `npm run check`：407 项测试通过，页面注册校验、流程校验及微信小程序构建通过。最后补充表单加载态和安全日志后，重跑相关 87 项测试与构建通过。
- 后端 merchant logic、model、merchant handler 及 resource/auth/admin logic 回归通过；使用本机 Go 1.25.10 与临时 GOCACHE，未改项目 Go 版本要求。
- 生成路由的旧商家 SQL mock 已随字段和事务契约更新，补充展示开关匿名拒绝、非管理者拒绝、缺失参数校验及管理者成功路径；完整 server 包测试通过。
- 使用仓库要求的 goctl 1.7.5 检查 API 生成一致性通过；迁移静态校验通过。
- 浏览器以 390×844 预览实际目录、我的、详情和入驻 Vue 组件；模拟商家数据，地图为占位，检查了排版和底部操作区。不能替代微信开发者工具/真机验证。
- 独立审查提出的预览链接参数、旧草稿覆盖风险位置基线、搜索提示与能力不一致问题均已修复并复核。
- API 契约测试 25/26 通过，唯一失败为历史文档 `2026-09-22-product-market-feasibility-review.md` 第 256 行包含“采购需求”词语。已确认 HEAD 原文相同，与此次代码改动无关；未改写历史评估结论。
- 尚未执行真实数据库迁移、部署或小程序发布。上线顺序为迁移 000036 → 后端 → 小程序。微信选点、导航、授权、上传和分享应在真实环境完成验收。
