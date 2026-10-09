# 多业务主体实施计划

> 本文保留前轮实施记录；用户随后明确不考虑历史数据兼容，当前规则以 [去历史兼容计划](2026-10-09-business-profile-clean-model.md) 和更新后的设计为准。

依据：../specs/2026-10-09-business-profile-design.md。用户已要求实施，继续当前工作区，保留所有前序改动，不提交/部署。

## 任务与文件边界

1. 后端：backend目录所有相关API/schema/model/logic/handler/test与wxapp/api/merchant.js。新增000037、业务档案类型和校验、可选保存兼容、详情裁剪、地图过滤、goctl生成与回归。不能改其它wxapp文件。
2. 入驻：wxapp/common/merchantBusinessProfile.js及测试、merchantOnboarding.js及测试、pages/merchant/profile.vue及测试。依照设计固定字段契约，前端集中角色展示配置；旧草稿兼容并保留服务器风险基线，不把恢复草稿当作地图授权。不能改api和其它页面。
3. 整合：主任务负责my/detail、merchantStorefront助手、目录卡片与页面文案、相关测试/流程断言、文档与总验证。依赖新增业务助手导出 getBusinessProfile(detail), getBusinessRoleLabel(role), getBusinessRoleLabels(profile), getBusinessProfileSummary(detail)；上述函数按设计提供，summary返回字符串（按角色呈现主营/加工/出租/服务）。
4. 独立审查：重点旧客户端兼容、无GPS入驻、地图关闭隐私、风险位置和权限，修复后限定范围复核。

## 必要兼容约定

businessProfile字段详见设计。mapVisible不替代ownerVisible；mapVisible=false不清空私有地址但对访客隐去精确地址。旧merchantType继续返回兼容值，既有资源merchantId不变。无新档案的历史主体保持旧完成状态；前端主动保存新档案时严格应用新规则。业务类型不表示认证/权限。

## 进度

- [x] 后端与数据契约
- [x] 入驻与本机草稿
- [x] 主页、我的、目录整合
- [x] 回归、审查、维护说明

## 实施与验证结果

- 后端已新增000037、版本化业务档案、地图独立开关、keep地点动作、旧类型兼容与目录角色投影。新业务档案与地图开关省略保留，原merchantId/资源归属不变。
- 入驻已支持五类主业务及兼营、独立主体性质、动态专业资料和服务区域；无照片/GPS出租方可完成，公开地点需明确开启。主页/我的/位置页采用适配业务的展示，地图关闭不再误判为入驻失败。
- 原草稿位置与服务端不同则保留服务端地点并说明；审查发现的恢复草稿后再次编辑档口号被keep忽略问题已修复，针对性测试先失败后通过。独立限定复核规格和代码质量均通过。
- 小程序 npm run check：429项测试全部通过，页面/流程校验及mp-weixin构建通过，仅既有Sass弃用提示。
- 后端model、merchant logic/handler、resource/auth/admin logic六包回归通过；完整server包通过，商家生成路由另行定向验证通过。
- goctl1.7.5生成一致性通过，迁移静态校验及30项迁移测试通过；git diff --check通过。Go测试使用GOTOOLCHAIN=local GOCACHE=/private/tmp/wplink-merchant-go-cache，未改项目工具链。
- 实际Vue组件在390×844浏览器预览验证了工厂主/兼营表单、无GPS房东主页和管理状态。使用模拟数据，地图和微信原生能力未通过浏览器代替验收。

## 尚未执行与上线顺序

未提交Git、未部署、未执行真实数据库迁移。按000036→000037→后端→小程序上线，真机验证登录、选点、上传、联系和分享。旧资源发布代码及归属保留，具体商品/房源发布不在此次范围。

上轮已确认API契约套件存在历史产品评估文档包含“采购需求”的既有检查失败，本轮未修改该历史原文；不将它报告为全仓库测试通过。
