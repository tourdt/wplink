# 商家模块维护说明

更新日期：2026-10-09。范围：商家资料与位置专项重构，以及地图首发界面的公开设置与多业务主体兼容；不代表完整 PRD 已全部实现。

## 1 职责入口

| 文件 | 职责 |
|---|---|
| `backend/app/internal/handler/merchant/update_merchant_handler.go` | 请求解析与商家管理权限检查，继续沿用现有契约 |
| `backend/app/internal/logic/merchant/update_merchant_logic.go` | 编排保存、记录校验失败、将存储错误转为中文提示 |
| `backend/app/internal/logic/merchant/merchant_profile_patch.go` | 资料归一化、主营去重、名称和电话校验、构建 set/clear 位置指令 |
| `backend/app/internal/model/merchant_location.go` | 位置数据类型、状态与操作常量、统一坐标校验 |
| `backend/app/internal/model/merchant_profile_update.go` | 行锁下状态推导、资料更新、报告结案、商家类型变更记录及提交后日志 |
| `backend/app/internal/model/merchant_model.go` | 商家资料读取与商家查询 |
| `backend/app/internal/model/merchant_operation_error.go` | 保留根因的事务阶段错误，错误正文不包含数据库原文 |
| `backend/app/internal/logic/merchant/safe_logging.go` | 统一安全错误日志，覆盖详情、保存、目录和位置报告 |

以上文件为手写业务扩展；生成文件与 `.api` 字段未改动。以后修改接口字段仍必须先改 `.api`，通过仓库 goctl 生成流程更新类型与路由。

## 2 状态和业务边界

商家身份、经营资料与地图展示是不同职责，继续共用同一个 merchantId。

- 资料完成统一要求完整业务档案，按所选主要和兼营业务验证专业资料。不要求 GPS，也不要求有资源或会员。
- 首次完整资料保存时填写 onboarded_at，已有时间保留。不提供历史数据适配分支。
- 主营去除首尾空格、空项和重复项，保留原始顺序；本次未引入新的分类编码迁移。
- 当前保存接口继续采用原全表单语义：主营、简介、图片等字段缺省时沿用既有写入行为，并非通用 PATCH。调用方必须提交完整表单，不能把它当作只改一项资料的接口。
- `set` 仍需要完整有效坐标，允许合法 0；写入位置并按原规则结案 pending 位置报告。
- `clear` 清除结构化位置和坐标、隐藏地图位置，不结案位置报告；名称和主营完整时资料仍为 completed。
- 资料、位置、报告和身份变更记录位于同一事务；任一 SQL 或提交失败，接口不返回保存成功。
- 主页发现无效存量坐标时不公开导航，不能仅凭“两个坐标字段非空”判断有效。
- 资源表、发布流程、管理关系、商家编号和资源归属不变。资源相关回归需纳入共享商家能力变更的验证范围。

多人报告自动隐藏、自助重新选点恢复规则保持不变。地图首发界面已补充联系方式公开开关、入驻表单和暂停展示；资源发布核心继续保留，小程序仅关闭相关页面注册，不等同于后端全局功能开关。

### 地图首发配套

- 全新环境按顺序执行数据库迁移至 `000037_business_profile.up.sql`，再部署后端并发布小程序。`owner_visible` 默认 true，`public_phone` / `public_wechat` / `map_visible` 默认 false，业务资料未完善时不公开入驻。初始化只保留系统字典，不创建演示商家；未在当前任务连接真实数据库执行迁移。
- `POST /api/v1/merchants/:merchantId/visibility` 通过既有管理权限校验，只改变展示开关。暂停后目录、位置上下文和附近店铺查询均排除该店铺；非管理者无法读取详情。成功记录“店铺展示状态已更新”，失败按统一安全日志处理。
- 管理者详情可读取编辑所需联系资料；访客详情只返回已授权电话或微信，不返回联系人姓名。省略公开配置不会重置原设置。
- 草稿使用v2命名空间按用户和商家隔离保存在本机，不读取或迁移v1草稿，存草稿时先将临时照片上传为稳定 URL；草稿不更新公开资料，提交成功、主动退出或注销时清除当前账号草稿。恢复草稿保留最新服务端位置风险基线，当前版本草稿不能代替重新选点。
- `wxapp/pages.json` 仅注册首发页面，默认进入档口目录；`pages.legacy.json` 与 `pages/home/legacy.vue` 保存旧入口配置和旧首页源码。后续恢复资源发布时应逐项恢复路由并验证登录回跳，不能直接覆盖当前注册文件。
- 微信导航、选点、手机号授权、图片上传及分享需要开发者工具和真机联调；浏览器预览仅检查布局，使用模拟商家数据及地图占位组件。

## 3 日志与排错

### 3.1 保存日志

成功事件为“商家资料事务已提交”，`operation=merchant_profile_update`、`stage=committed`。记录 merchantId、位置动作、新旧资料和位置状态、是否变更身份、结案报告数与 durationMs。只在事务提交成功后输出。

失败事件为“商家操作失败”。通过 merchantId 和请求 trace 关联同一操作，检查 stage：

| stage | 排查方向 |
|---|---|
| `validation` | 校验日志中的 field，检查对应输入规则，不打印用户输入 |
| `begin` | 数据库连接、事务开启失败 |
| `lock_merchant` | 商家存在性、行锁等待或读取失败 |
| `save_profile` | 资料更新、字段约束或写入错误 |
| `affected_rows` | 更新行数异常 |
| `resolve_reports` | 待处理位置报告结案失败；资料更新应一起回滚 |
| `record_type_change` | 商家类型记录失败；资料更新应一起回滚 |
| `commit` | 事务提交失败；不要根据此前 SQL 成功推断保存成功 |
| `store` | 非分阶段存储失败，结合 operation、category、errorType 和 SQLSTATE 排查 |

读取操作使用 `merchant_profile_read`、`merchant_profile_permission`、`list_merchant_directory`、`count_merchant_directory`、`get_visible_merchant_location`、`list_nearby_visible_merchants`；位置反馈使用 `create_location_report`。

### 3.2 敏感信息边界

不记录请求体、电话、微信、地址、精确坐标、SQL 参数或数据库错误 Message/Detail。数据库错误可能携带整行敏感数据，因此只输出错误类型、SQLSTATE、安全分类和事务阶段。不能为了“更详细”恢复直接输出 `err`。

原 cause 通过 `Unwrap` 保留给 `errors.Is/As`，前端继续获得稳定中文提示。正常读取不逐条记成功日志，避免列表访问造成无意义日志量。业务审计记录不由运行日志替代。

## 4 验证方式与限制

在 `backend/` 下运行：

```bash
go test ./app/internal/model ./app/internal/logic/merchant ./app/internal/handler/merchant ./app/internal/logic/resource ./app/internal/logic/auth ./app/internal/logic/admin -count=1
go vet ./app/internal/model ./app/internal/logic/merchant ./app/internal/handler/merchant ./app/internal/logic/resource ./app/internal/logic/auth ./app/internal/logic/admin
```

新增回归覆盖非法存量坐标、主营归一化、无位置资料完成、空主营资料不完整、安全日志、事务失败阶段、回滚与根因保留。事务 SQL 使用 sqlmock 验证，不能替代实际 PostgreSQL 联调。

本次执行环境默认 Go 1.25.13 工具链不可用，定向验证使用本机 Go 1.25.10：在命令前设置 `GOTOOLCHAIN=local GOCACHE=/private/tmp/wplink-merchant-go-cache`。未修改仓库工具链要求；正式交付环境仍应使用项目指定工具链验证。

## 5 多业务主体扩展（迁移 000037）

### 数据与新契约

- 用户账号管理经营主体，主体沿用原 `merchantId`，资源继续归属同一 ID。`businessProfile.entityKind` 是自填的个人/企业机构性质，`primaryRole` / `roles` 是主要与兼营业务，均不表示认证结果或权限。
- `business_profile` 是有明确结构和版本的 JSONB 文档，不接受任意业务字段。版本1支持档口、加工厂、房东/出租方、配套服务、货源供应；专业资料统一限制20项、每项50字，服务区域200字。代码入口为 `model/business_profile.go`，前端对应 `common/merchantBusinessProfile.js`。
- 新账号从完整但未填身份的版本1空档案开始，profile_status=incomplete，map_visible=false。{}等旧格式不适配，旧类型不推断身份，needsSetup已移除。
- businessProfile和mapVisible是资料保存的必填字段；缺失返回中文400。merchant_type仅作为当前查询使用的主要角色同码投影，不接受独立写入，不再映射旧身份。
- 后续增加身份，应同步后端枚举/完整性校验、前端集中身份配置和专业资料、API及对应测试；不要复制一套登录、入驻、主页或权限系统。
- 档案结构升级须明确schemaVersion的读写规则。当前只接受版本1，不为历史数据编写转换或回填层。

### 地点与主页

`ownerVisible` 控制整体主页公开，`mapVisible` 单独控制接待地点公开。无 GPS 且地图关闭的完整业务主体可以公开和分享主页，不应显示为位置异常。地图关闭时管理者仍可编辑私有地点，但访客详情隐藏详细地址、地点结构和坐标；目录、附近和位置上下文也排除该主体。

位置动作 `keep` 保留事务读取的地址、全部地点字段及风险状态，不结案位置反馈；`set` 才写入新点位并按既有规则结案，`clear` 清除结构化位置。普通业务编辑不能通过回传原坐标误触发恢复。旧草稿地点与服务端不同时，前端保留服务端地点并提示重新选点；恢复草稿不新增地图公开授权。

未来出租房源、商品、服务资源各有自己的资料与可选位置，仍使用主体ID归属；不能用房源地址覆盖房东主体接待地点。当前没有实现具体资源发布新入口。

### 上线与回归

部署顺序：迁移000036 → 迁移000037 → 后端 → 小程序。未在本次任务操作真实数据库。应用回退时优先保留新增列和档案数据；000037的down迁移会删除新业务档案，不能当作无损应用回退步骤。

除原商家测试外，应持续覆盖：无GPS出租方完成、多角色专业资料、未知版本拒绝、缺失新必填字段拒绝、访客地点隐私、风险地点keep、草稿恢复后的继续编辑、主体主要角色同码投影和生成路由权限。位置切换及微信选点/分享/上传仍需真实环境联调。
