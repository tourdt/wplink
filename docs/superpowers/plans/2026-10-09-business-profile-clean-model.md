# 多业务模型去除历史兼容实施计划

用户明确：本次重构不需要考虑历史数据兼容。此要求覆盖前一轮设计中旧身份推断、旧版本保存分支与旧草稿适配约定。继续当前工作区，不回滚其他改动，不执行真实数据库或清库。

## 新约定

- businessProfile 是唯一业务身份来源。保存必须提交完整版本1档案和显式 mapVisible，缺失返回中文400。
- 移除 LegacyBusinessProfile、旧类型推断、needsSetup、旧完成条件分支、旧类型到新身份及反向旧枚举映射。
- 保留 merchantId、账号管理关系、资源归属，它们是当前业务关系而非历史兼容层。
- 现有数据库 merchant_type 及后台/收藏查询可以作为主要角色的查询投影，直接存 profile.primaryRole 同码；不再接受独立写 merchantType，不再映射为stockist/individual等旧类型。不扩展为全库字段改名工程。
- 迁移000037业务档案默认是结构完整但未填写的版本1对象；map_visible默认false。新账号profile_status=incomplete，空主要角色，无推断；不为旧数据编写回填/恢复完成状态逻辑。
- 读取仅接受规范版本1对象，合法未完善新档案允许空身份；不把{}或旧类型转为新身份。未知版本明确拒绝，不静默覆盖。
- 前端去掉旧身份和旧品类兜底、旧map字段缺省推断；使用版本2本机草稿命名空间，只读新草稿。保留草稿与服务端位置风险保护及同版本编辑功能。
- keep地点动作、独立公开开关、隐私过滤、权限及资源后续扩展边界保持。

## 分工

1. 后端代理：backend相关生产/测试/API/迁移与wxapp/api/merchant.js；必填新契约、干净初始化、同码角色投影、生成校验。
2. 入驻代理：common/merchantBusinessProfile、merchantDraft、merchantOnboarding、profile.vue及测试，request-loading-contract测试；只使用新契约。
3. 主任务：my/detail/card/目录辅助、当前后台和未上线资源相关角色展示枚举、流程测试、文档与总验证。

## 验证

新账号空档案→选身份→提交；缺新档案/开关拒绝；无GPS房东完成；同码投影；地图关闭地址隐私；keep风险保护；新草稿隔离；新版本未知拒绝。运行前端check、后端受影响包/生成路由、API生成与迁移静态验证。

## 进度

- [x] 后端新契约及初始化
- [x] 前端移除旧兜底与旧草稿支持
- [x] 展示、文档与回归整合

## 执行与验证记录

- 新账号创建SQL使用空主要角色；迁移默认完整空v1、未完善状态及地图关闭。冷启动不创建示例商家，手动演示脚本采用新档案格式。
- 前端移除旧身份、旧品类和地图缺省推断；仅读取v2新契约草稿，继续保护服务端地点风险状态。
- 后台、收藏与资源详情的身份展示统一为五类新业务。资源归属与现有权限模型保留，未新增发布入口。
- `npm --prefix wxapp run check`：434项测试通过，页面/流程检查及小程序构建通过；随后增强API字段白名单测试，18项定向测试通过。
- `npm --prefix admin-web run check`：47项测试及构建通过。
- `GOTOOLCHAIN=local GOCACHE=/private/tmp/wplink-merchant-go-cache go test ./app/internal/model ./app/internal/logic/merchant ./app/internal/handler/merchant`：全部通过。
- `PATH="/private/tmp/wplink-goctl-1.7.5:$PATH" node scripts/api_codegen.mjs --check`：API生成文件为最新。
- API路由清单检查通过；文档治理13项通过。独立审查发现的新账号旧身份默认值已修复，复核通过。
- 后端服务全包 `go test ./app/internal/server -count=1` 通过；迁移静态检查通过，迁移测试32/32通过。
- 本次未执行真实数据库迁移、清库或部署。微信选点、分享等仍需真机联调；Go验证使用本机1.25.10，正式环境需按仓库工具链要求验证。
