# API 契约与运行时路由归一实施计划

> **执行要求：** REQUIRED SUB-SKILL: 使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 按任务执行；每个功能/修复任务同时使用 `superpowers:test-driven-development`，完成前使用 `superpowers:verification-before-completion`。所有步骤使用复选框跟踪。

**目标：** 将全部有效业务接口归一到 `app.api` 与 goctl 生成的 go-zero 路由，迁移既有业务 Handler，并彻底删除手写 API Router 和 `/api/` fallback。

**架构：** `.api` 负责方法、路径、DTO 与 Handler 名称；仓库内固定的 goctl 模板生成 `types.go`、`routes.go` 和 Handler 骨架；Handler 只做协议、身份与权限适配，继续调用现有 Logic/Model。迁移期间生成 Handler 不接入生产 Server，按领域完成行为测试后，在最后一个切换任务中一次性注册全部生成路由并删除兼容 Router。

**技术栈：** Go 1.23、go-zero 1.7.6、goctl 1.7.5、PostgreSQL、Node.js `node:test`、Makefile。

## 全局约束

- 所有实施计划、设计文档和产品文档使用中文；代码标识、路径与命令可保留英文。
- `.api`、Handler 和 types 的可生成部分优先使用 goctl，禁止手工维护第二套路由注册表。
- 所有生成命令显式使用仓库内模板，禁止隐式读取用户级 `GOCTL_HOME`。
- 保留当前外部 URL、HTTP 方法、请求字段、响应字段和 `{code,msg,data}` envelope；确认废弃的接口不增加兼容别名。
- 微信支付与内容审核回调保留供应商要求的原始响应、验签、Body 大小限制、重放保护和幂等语义。
- 业务规则、事务和 SQL 保留在现有 Logic/Model；Handler 不承载新的业务状态机。
- 管理员模块权限和商家管理权限默认拒绝；任何依赖缺失不能退化为放行。
- 所有用户提示为安全、明确的中文；诊断日志不得包含 Token、手机号、OpenID、SessionKey、签名、密钥或完整请求/响应体。
- 每个任务只提交本任务相关文件；不修改前端依赖，不部署独立 Worker，不引入 Redis、MQ 或 API Gateway。
- 最终完成标准是删除 `NewAPIRouter`、`NewProductionAPIRouter`、旧路由文件和 `/api/` fallback，并通过 `make check`。

---

## 文件结构与职责

### 生成与门禁

- `backend/app/goctl/api/*.tpl`：项目固定的 goctl 1.7.5 API 模板，仅服务 API 生成。
- `backend/scripts/api_codegen.mjs`：在临时 module 中运行 goctl，写入或检查 types、routes、Handler 骨架和路由 manifest。
- `backend/scripts/api_codegen.test.mjs`：验证模板隔离、路由解析、重复检测和 `--check` 行为。
- `backend/app/internal/handler/routes.go`：goctl 生成的唯一业务路由注册代码。
- `backend/app/internal/handler/routes_manifest_gen.go`：从 `.api` 生成的只读路由指纹，供运行时测试比较，不参与路由分发。
- `backend/app/internal/types/types.go`：从 `app.api` 完整生成的 DTO。

### HTTP 边界

- `backend/app/internal/handler/handlerx/auth.go`：必需/可选用户身份、管理员身份和 Bearer 解析。
- `backend/app/internal/handler/handlerx/permission.go`：商家管理权限前置。
- `backend/app/internal/handler/handlerx/request.go`：有限 Body、原始 JSON、客户端 IP 和安全解析错误。
- `backend/app/internal/handler/handlerx/not_migrated.go`：仅迁移期生成骨架使用；最终必须删除。
- `backend/app/internal/middleware/admin_auth_middleware.go`：后台 Token、账号状态和模块权限统一中间件。
- `backend/app/internal/permission/admin_route.go`：后台路径到模块的默认拒绝映射。
- `backend/app/internal/handler/<group>/*_handler.go`：各领域薄 Handler。

### 服务与测试

- `backend/app/internal/svc/service_context.go`：装配 Handler 依赖与 `AdminAuth` 中间件。
- `backend/app/internal/server/generated_api_test.go`：迁移期间直接挂载生成路由的测试工厂和跨领域路由测试。
- `backend/app/internal/server/gozero_server.go`：最终注册生成路由，只保留健康检查和 `/admin` 静态 fallback。
- `backend/app/app.go`：最终不再构造兼容 API Router。
- `backend/app/internal/server/*_api_test.go`：逐领域迁移到生成路由测试入口。

---

### Task 1：建立真实路由盘点与解析基线

**文件：**
- Create: `backend/scripts/api_route_inventory.mjs`
- Create: `backend/scripts/api_route_inventory.test.mjs`
- Modify: `Makefile`

**接口：**
- Produces: `parseAPIContracts(apiDir) -> Array<{method,path,handler,group,source}>`
- Produces: `parseLegacyRoutes(files) -> Array<{method,path,source,line}>`，解析前必须剔除块注释和行注释。
- Produces: `routeFingerprint(route) -> "METHOD /full/path"`
- Consumes: `backend/app/api/app.api` import 链与四个旧 server 路由文件。

- [ ] **Step 1：写路由解析失败测试**

测试使用临时 fixture，覆盖 `@server prefix`、import、`:param`、旧 `{param}` 标准化、同一路径不同方法、块注释中的伪路由和重复 Handler。核心断言：

```js
assert.deepEqual(parseAPIContracts(fixtureDir).map(routeFingerprint), [
  'GET /api/v1/items/:itemId',
  'POST /api/v1/admin/items',
])
assert.deepEqual(parseLegacyRoutes([legacyFile]).map(routeFingerprint), [
  'GET /api/v1/items/:itemId',
])
```

- [ ] **Step 2：运行测试确认失败**

Run: `cd backend && node --test scripts/api_route_inventory.test.mjs`

Expected: FAIL，提示 `api_route_inventory.mjs` 不存在。

- [ ] **Step 3：实现最小解析器和 CLI**

CLI 支持 `--json` 与默认表格输出；标准化规则固定为大写方法、单个 `/`、去除尾斜杠、`{id}` 转为 `:id`。真实仓库输出必须列出三类差异：契约独有、旧 Router 独有、双方共有。不得把注释中的已下线商家认证路由算作运行时接口。

- [ ] **Step 4：把解析测试加入后端门禁**

在 `check-backend` 的 Node 测试命令中加入 `scripts/api_route_inventory.test.mjs`，本任务只验证解析器，不要求两套现有路由相等。

- [ ] **Step 5：运行测试与真实盘点**

Run: `cd backend && node --test scripts/api_route_inventory.test.mjs && node scripts/api_route_inventory.mjs`

Expected: 测试 PASS；盘点明确显示增长活动、微信支付通知、内容审核回调和 `GET /api/v1/admin/resources` 等运行时独有接口，注释内认证接口不出现。

- [ ] **Step 6：提交**

```bash
git add Makefile backend/scripts/api_route_inventory.mjs backend/scripts/api_route_inventory.test.mjs
git commit -m "test: 建立 API 路由盘点基线"
```

### Task 2：固定 goctl 模板并建立可复现生成器

**文件：**
- Create: `backend/app/goctl/api/config.tpl`
- Create: `backend/app/goctl/api/context.tpl`
- Create: `backend/app/goctl/api/etc.tpl`
- Create: `backend/app/goctl/api/handler.tpl`
- Create: `backend/app/goctl/api/logic.tpl`
- Create: `backend/app/goctl/api/main.tpl`
- Create: `backend/app/goctl/api/middleware.tpl`
- Create: `backend/app/goctl/api/route-addition.tpl`
- Create: `backend/app/goctl/api/routes.tpl`
- Create: `backend/app/goctl/api/types.tpl`
- Create: `backend/scripts/api_codegen.mjs`
- Create: `backend/scripts/api_codegen.test.mjs`
- Create: `backend/app/internal/handler/handlerx/not_migrated.go`
- Create: `backend/app/internal/handler/handlerx/not_migrated_test.go`
- Create: `backend/app/internal/handler/routes.go`
- Create: `backend/app/internal/handler/routes_manifest_gen.go`
- Modify: `backend/app/internal/types/types.go`
- Modify: `Makefile`

**接口：**
- Produces: `node scripts/api_codegen.mjs --write`
- Produces: `node scripts/api_codegen.mjs --check`
- Produces: `handler.ContractRoutes() []handler.ContractRoute`
- Produces: `handlerx.NotMigrated(handlerName string) http.HandlerFunc`

- [ ] **Step 1：写生成隔离与过期检测测试**

测试把 `GOCTL_HOME` 指向一个含错误 `zmall/common/response` 模板的临时目录，再执行生成器，断言输出只使用仓库模板；篡改临时复制的 `routes.go` 后，`--check` 必须非零退出并指出过期文件。

- [ ] **Step 2：运行测试确认失败**

Run: `cd backend && node --test scripts/api_codegen.test.mjs`

Expected: FAIL，提示生成器不存在。

- [ ] **Step 3：初始化并裁剪仓库模板**

以 `goctl template init` 生成的 1.7.5 官方 `api/` 模板为基础，只提交上面十个文件。`handler.tpl` 不调用 goctl 空 Logic，而生成可编译的迁移骨架：

```go
func {{.HandlerName}}(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return handlerx.NotMigrated("{{.HandlerName}}")
}
```

模板 import 只允许 `net/http`、项目 `handlerx` 和项目 `svc`，不得出现 `zmall`。

- [ ] **Step 4：实现生成器**

生成器必须：校验 `goctl --version` 为 `1.7.5`；创建 `module wplink/backend` 的临时 module；运行 `goctl api validate` 和带 `--home backend/app/goctl --style go_zero` 的 `goctl api go`；只同步 `types.go`、`routes.go`、缺失 Handler 文件；根据 `parseAPIContracts` 生成排序稳定的 `routes_manifest_gen.go`；不复制 config、svc、logic 或 main。

- [ ] **Step 5：实现迁移骨架的安全失败**

`NotMigrated` 记录 `handler` 字段但不记录请求内容，向客户端返回：

```go
response.JSON(w, nil, errx.New(errx.CodeInternalError, "接口暂不可用，请稍后重试"))
```

该函数只用于尚未接入生产 Server 的生成骨架；最终门禁扫描到任何调用都必须失败。

- [ ] **Step 6：首次生成并处理已有 Handler**

Run: `cd backend && node scripts/api_codegen.mjs --write`

保留已有 `adminauth.AdminLoginHandler`、`city.ListCityStationsHandler` 和 `city.ListCityResourceTypesHandler` 实现；生成器检测函数已存在时不得创建同名函数。其余当前契约 Handler 生成可编译骨架。

- [ ] **Step 7：增加 Makefile 门禁**

增加：

```make
generate-api:
	cd backend && node scripts/api_codegen.mjs --write

check-api-generated:
	cd backend && node scripts/api_codegen.mjs --check
```

让 `check-backend` 在 Go 测试前执行 `check-api-generated`。

- [ ] **Step 8：运行生成与测试**

Run: `make check-api-generated && cd backend && node --test scripts/api_codegen.test.mjs && go test ./app/internal/handler/...`

Expected: 全部 PASS；`rg 'zmall|NotMigrated' app/internal/handler` 只命中迁移骨架与其测试。

- [ ] **Step 9：提交**

```bash
git add Makefile backend/app/goctl backend/scripts/api_codegen.mjs backend/scripts/api_codegen.test.mjs backend/app/internal/handler backend/app/internal/types/types.go
git commit -m "build: 固定 goctl API 生成链路"
```

### Task 3：建立统一身份、权限和请求边界

**文件：**
- Create: `backend/app/internal/handler/handlerx/auth.go`
- Create: `backend/app/internal/handler/handlerx/auth_test.go`
- Create: `backend/app/internal/handler/handlerx/permission.go`
- Create: `backend/app/internal/handler/handlerx/permission_test.go`
- Create: `backend/app/internal/handler/handlerx/request.go`
- Create: `backend/app/internal/handler/handlerx/request_test.go`
- Create: `backend/app/internal/middleware/admin_auth_middleware.go`
- Create: `backend/app/internal/middleware/admin_auth_middleware_test.go`
- Create: `backend/app/internal/permission/admin_route.go`
- Create: `backend/app/internal/permission/admin_route_test.go`
- Modify: `backend/app/internal/svc/service_context.go`

**接口：**
- Produces: `RequiredUser(r *http.Request, service authlogic.TokenService) (session.UserTokenSubject, error)`
- Produces: `OptionalUser(r *http.Request, service authlogic.TokenService) (session.UserTokenSubject, bool, error)`
- Produces: `OptionalAdmin(r *http.Request, service AdminTokenService) (session.AdminTokenSubject, bool)`
- Produces: `AdminFromContext(ctx context.Context) (session.AdminTokenSubject, error)`
- Produces: `RequireMerchant(r *http.Request, deps MerchantPermissionDeps, merchantID string) error`
- Produces: `ReadLimitedBody(r *http.Request, limit int64) ([]byte, error)`
- Produces: `ClientIP(r *http.Request) string`
- Produces: `permission.AdminModuleForPath(path string) string`
- Produces: `middleware.NewAdminAuthMiddleware(tokenService).Handle(next http.HandlerFunc) http.HandlerFunc`

- [ ] **Step 1：写身份与权限失败测试**

覆盖空 Header、非 Bearer、过期用户 Token、可选身份无 Header、可选身份携带坏 Token、管理员 Token、管理员访问商家、普通用户管理自己的商家和普通用户越权。可选身份携带坏 Token 必须返回错误，不能当匿名请求。

- [ ] **Step 2：写后台模块默认拒绝测试**

用表驱动覆盖 dashboard、resources、resource-reports、merchants/entitlements、merchants、banner、hot-search、VIP、growth、map、resource-type、operators/module-permissions、operation/task、search-log；未知 `/api/v1/admin/new-module` 必须返回空模块并被拒绝。

- [ ] **Step 3：运行测试确认失败**

Run: `cd backend && go test ./app/internal/handler/handlerx ./app/internal/middleware ./app/internal/permission`

Expected: FAIL，提示目标函数或包不存在。

- [ ] **Step 4：迁移最小公共实现**

从旧 `auth_routes.go`/`api.go` 迁移现有 Bearer、商家权限、客户端 IP 和有限 Body 语义。`RequireMerchant` 的依赖必须显式非 nil；缺失时返回 `CodeInternalError` 并记录诊断日志，禁止沿用旧代码的 nil 放行。

依赖接口和结构固定为：

```go
type AdminTokenService interface {
    ParseAdminToken(ctx context.Context, token string) (session.AdminTokenSubject, error)
}

type MerchantPermissionStore interface {
    UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error)
}

type MerchantPermissionDeps struct {
    UserTokenService  authlogic.TokenService
    AdminTokenService AdminTokenService
    Store             MerchantPermissionStore
}
```

- [ ] **Step 5：实现 AdminAuth 中间件并装配**

中间件将验证后的 `session.AdminTokenSubject` 写入 request context；登录路由不声明该中间件。`ServiceContext` 增加：

```go
AdminAuth rest.Middleware
```

`NewServiceContext` 使用 `middleware.NewAdminAuthMiddleware(AdminTokenService).Handle` 装配。

- [ ] **Step 6：运行公共边界测试**

Run: `cd backend && go test ./app/internal/handler/handlerx ./app/internal/middleware ./app/internal/permission`

Expected: PASS。

- [ ] **Step 7：提交**

```bash
git add backend/app/internal/handler/handlerx backend/app/internal/middleware backend/app/internal/permission backend/app/internal/svc/service_context.go
git commit -m "refactor: 统一 API 身份与权限边界"
```

### Task 4：建立生成路由测试工厂并迁移公共基础接口

**文件：**
- Create: `backend/app/internal/server/generated_api_test.go`
- Modify: `backend/app/internal/handler/city/city_handler.go`
- Create: `backend/app/internal/handler/location/reverse_geocode_handler.go`
- Create: `backend/app/internal/handler/discovery/get_home_operation_config_handler.go`
- Create: `backend/app/internal/handler/discovery/list_home_resources_handler.go`
- Create: `backend/app/internal/handler/discovery/list_home_recent_merchants_handler.go`
- Create: `backend/app/internal/handler/discovery/list_hot_search_keywords_handler.go`
- Create: `backend/app/internal/handler/discovery/get_topic_resources_handler.go`
- Create: `backend/app/internal/handler/discovery/validate_webview_handler.go`
- Modify: `backend/app/internal/server/city_api_test.go`
- Modify: `backend/app/internal/server/location_api_test.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`

**接口：**
- Produces: 测试内 `newGeneratedAPIServer(t, svcCtx) *rest.Server`，调用 `handler.RegisterHandlers`。
- Consumes: city、location、discovery 现有 Logic 与生成 types。

- [ ] **Step 1：把公共接口测试改到生成路由并确认失败**

测试请求至少覆盖两个城市站接口、逆地理编码、首页运营配置、首页资源、最近商家、热词、专题资源和 webview 校验。每个请求从 `newGeneratedAPIServer` 进入，不直接调用 Handler。

Run: `cd backend && go test ./app/internal/server -run 'Test(City|Location|Discovery).*Generated' -count=1`

Expected: FAIL，响应为“接口暂不可用”。

- [ ] **Step 2：实现公共 Handler**

使用 `httpx.Parse`/`pathvar.Vars` 将生成 DTO 映射为现有 Logic 请求。响应使用 `response.JSON`；切片字段为空时保持当前 API 要求的空数组，不返回未预期的 `null`。

- [ ] **Step 3：运行公共接口与 Logic 测试**

Run: `cd backend && go test ./app/internal/handler/city ./app/internal/handler/location ./app/internal/handler/discovery ./app/internal/server -run 'Test(City|Location|Discovery).*' -count=1`

Expected: PASS。

- [ ] **Step 4：确认本批无迁移骨架**

Run: `! rg 'NotMigrated' backend/app/internal/handler/{city,location,discovery}`

Expected: 退出 0。

- [ ] **Step 5：提交**

```bash
git add backend/app/internal/handler/city backend/app/internal/handler/location backend/app/internal/handler/discovery backend/app/internal/server/generated_api_test.go backend/app/internal/server/city_api_test.go backend/app/internal/server/location_api_test.go backend/app/internal/server/remaining_api_test.go
git commit -m "refactor: 迁移公共基础 API Handler"
```

### Task 5：迁移用户认证、管理员登录与上传接口

**文件：**
- Create/Modify: `backend/app/internal/handler/auth/*_handler.go`
- Modify: `backend/app/internal/handler/adminauth/admin_login_handler.go`
- Create: `backend/app/internal/handler/upload/create_upload_token_handler.go`
- Modify: `backend/app/internal/server/auth_api_test.go`
- Modify: `backend/app/internal/server/upload_api_test.go`
- Modify: `backend/app/internal/server/gozero_server_test.go`

**接口：**
- Implements: `WechatLoginHandler`、`SendSMSCodeHandler`、`GetMeHandler`、`BindPhoneHandler`、`BindWechatPhoneHandler`、`DeleteAccountHandler`、`AdminLoginHandler`、`CreateUploadTokenHandler`。
- Consumes: Task 3 的身份辅助组件和现有 auth/adminauth/upload Logic。

- [ ] **Step 1：把认证与上传测试改到生成路由**

覆盖登录成功、原始认证错误不泄漏、短信发送、`/me` 未登录/过期、手机号绑定、微信手机号绑定、账号注销、后台登录限流、用户上传、管理员上传和匿名上传拒绝。

- [ ] **Step 2：运行测试确认生成骨架失败**

Run: `cd backend && go test ./app/internal/server -run 'Test(Auth|Upload|GoZeroAdminLogin).*' -count=1`

Expected: FAIL，未实现 Handler 返回内部错误。

- [ ] **Step 3：实现认证与上传薄 Handler**

`GetMe`、绑定和注销只使用 `handlerx.RequiredUser` 返回的 UserID；上传调用 `handlerx.OptionalAdmin` 或 `handlerx.RequiredUser`，两者都失败时返回“请先登录”。后台登录保留客户端 IP、User-Agent、频率限制错误码和安全错误消息。

- [ ] **Step 4：运行测试并扫描迁移骨架**

Run: `cd backend && go test ./app/internal/handler/auth ./app/internal/handler/adminauth ./app/internal/handler/upload ./app/internal/server -run 'Test(Auth|Upload|GoZeroAdminLogin).*' -count=1 && ! rg 'NotMigrated' app/internal/handler/{auth,adminauth,upload}`

Expected: PASS。

- [ ] **Step 5：提交**

```bash
git add backend/app/internal/handler/auth backend/app/internal/handler/adminauth backend/app/internal/handler/upload backend/app/internal/server/auth_api_test.go backend/app/internal/server/upload_api_test.go backend/app/internal/server/gozero_server_test.go
git commit -m "refactor: 迁移认证与上传 API Handler"
```

### Task 6：迁移商家、互动、消息与指标接口

**文件：**
- Create/Modify: `backend/app/internal/handler/merchant/*_handler.go`
- Create/Modify: `backend/app/internal/handler/favorite/*_handler.go`
- Create/Modify: `backend/app/internal/handler/message/*_handler.go`
- Create/Modify: `backend/app/internal/handler/metrics/*_handler.go`
- Modify: `backend/app/internal/server/favorite_api_test.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`
- Modify: `backend/app/internal/server/resource_api_test.go`

**接口：**
- Implements: merchant 2、favorite 9、message 2、metrics 4 个契约 Handler。
- Consumes: Task 3 的 RequiredUser、OptionalUser、RequireMerchant。

- [ ] **Step 1：增加生成路由行为测试**

覆盖公开商家详情的可选身份、商家编辑越权、收藏/关注/保存搜索的用户身份、消息角色范围、公开曝光/地图事件的可选身份、资源指标与商家指标的管理权限。

- [ ] **Step 2：运行目标测试确认失败**

Run: `cd backend && go test ./app/internal/server -run 'Test(Merchant|Favorite|Message|Metrics).*Generated' -count=1`

Expected: FAIL，命中迁移骨架。

- [ ] **Step 3：实现商家与互动 Handler**

`GetMerchant` 使用 `OptionalUser`；`UpdateMerchant` 使用路径 `merchantId` 和 `RequireMerchant`。九个互动 Handler 的 `userID` 只来自 Token，不接受 query/body 覆盖。

- [ ] **Step 4：实现消息与指标 Handler**

消息角色根据 Token 用户、商家管理关系和请求 roleCode 复用当前 `messageRoleCodesForTokenUser` 语义并迁入 message Handler 的私有辅助文件。资源指标先读取资源所属商家再校验权限；公开埋点携带非法 Token 时返回未授权。

- [ ] **Step 5：运行领域测试和骨架扫描**

Run: `cd backend && go test ./app/internal/handler/merchant ./app/internal/handler/favorite ./app/internal/handler/message ./app/internal/handler/metrics ./app/internal/server -run 'Test(Merchant|Favorite|Message|Metrics).*' -count=1 && ! rg 'NotMigrated' app/internal/handler/{merchant,favorite,message,metrics}`

Expected: PASS。

- [ ] **Step 6：提交**

```bash
git add backend/app/internal/handler/merchant backend/app/internal/handler/favorite backend/app/internal/handler/message backend/app/internal/handler/metrics backend/app/internal/server/favorite_api_test.go backend/app/internal/server/remaining_api_test.go backend/app/internal/server/resource_api_test.go
git commit -m "refactor: 迁移商家互动与指标 API Handler"
```

### Task 7：迁移资源全链路与内容审核回调

**文件：**
- Create: `backend/app/api/callback.api`
- Modify: `backend/app/api/app.api`
- Create/Modify: `backend/app/internal/handler/resource/*_handler.go`
- Create: `backend/app/internal/handler/callback/verify_content_audit_callback_handler.go`
- Create: `backend/app/internal/handler/callback/handle_content_audit_callback_handler.go`
- Modify: `backend/app/internal/server/resource_api_test.go`
- Modify: `backend/scripts/api_contract.test.mjs`

**接口：**
- Implements: resource 21 个契约 Handler。
- Adds: `GET /api/v1/wechat/content-audit/media-callback` -> `VerifyContentAuditCallback`。
- Adds: `POST /api/v1/wechat/content-audit/media-callback` -> `HandleContentAuditCallback`。

- [ ] **Step 1：先写回调契约和生成路由失败测试**

`callback.api` 为 GET 定义 signature/timestamp/nonce/echostr query DTO；POST 的签名参数进入 query DTO，Body 由 Handler 按 256 KiB 限制读取。两个回调不声明用户或管理员中间件。

- [ ] **Step 2：运行契约校验与目标测试确认失败**

Run: `cd backend && goctl api validate --api app/api/app.api && node scripts/api_codegen.mjs --write && go test ./app/internal/server -run 'Test(Resource|ContentAuditCallback).*Generated' -count=1`

Expected: goctl 校验 PASS；行为测试因迁移骨架 FAIL。

- [ ] **Step 3：实现资源创建、草稿、提交与查询 Handler**

复用现有创建请求字段映射；创建/草稿从 Token 绑定创建者并校验商家权限；更新/提交先从 Store 获取资源所属商家；列表、搜索、相关推荐与详情保持当前公开/私有字段语义。

- [ ] **Step 4：实现资源管理、联系与支付发起 Handler**

刷新、成交、下架、删除、相似重发全部验证资源归属；联系事件与联系解锁从 RequiredUser 获取 userID；支付发起继续使用当前 dev mock 开关和 WechatPayGateway，不在 Handler 中增加重试。

- [ ] **Step 5：实现内容审核回调 Handler**

GET 验签成功原样返回 `echostr`。POST 必须按“验签但不记重放 → 读取有限 Body → AppID 校验 → Logic 幂等处理 → 成功后记录重放指纹”顺序执行；状态冲突按当前语义返回纯文本 `success`。

- [ ] **Step 6：运行资源与回调测试**

Run: `cd backend && go test ./app/internal/handler/resource ./app/internal/handler/callback ./app/internal/server -run 'Test(Resource|ContentAuditCallback).*' -count=1 && ! rg 'NotMigrated' app/internal/handler/{resource,callback}`

Expected: PASS。

- [ ] **Step 7：提交**

```bash
git add backend/app/api/app.api backend/app/api/callback.api backend/app/internal/types/types.go backend/app/internal/handler/routes.go backend/app/internal/handler/routes_manifest_gen.go backend/app/internal/handler/resource backend/app/internal/handler/callback backend/app/internal/server/resource_api_test.go backend/scripts/api_contract.test.mjs
git commit -m "refactor: 迁移资源与内容审核回调 Handler"
```

### Task 8：迁移权益、VIP 与微信支付通知

**文件：**
- Modify: `backend/app/api/callback.api`
- Create/Modify: `backend/app/internal/handler/entitlement/*_handler.go`
- Create/Modify: `backend/app/internal/handler/vip/*_handler.go`
- Create: `backend/app/internal/handler/callback/wechat_pay_notify_handler.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`
- Modify: `backend/app/internal/server/resource_api_test.go`

**接口：**
- Implements: entitlement 4、vip 5 个契约 Handler。
- Adds: `POST /api/v1/wechat-pay/notify` -> `WechatPayNotify`。

- [ ] **Step 1：增加支付通知契约与失败测试**

支付通知不定义普通 JSON Body DTO；Handler 读取最多 1 MiB 原始 Body 并传入全部首个 Header 值。成功响应必须是 `{"code":"SUCCESS","message":"成功"}`，失败响应必须是 HTTP 500 的 `{"code":"FAIL","message":"处理失败"}`。

- [ ] **Step 2：运行目标测试确认失败**

Run: `cd backend && node scripts/api_codegen.mjs --write && go test ./app/internal/server -run 'Test(Entitlement|VIP|WechatPayNotify).*Generated' -count=1`

Expected: FAIL，命中迁移骨架。

- [ ] **Step 3：实现权益与 VIP Handler**

商家权益、使用记录、置顶券和商家 VIP/订单/支付全部使用 `RequireMerchant`；公开计划与配额包保持匿名。订单 userID 只能来自 Token，不接受客户端伪造。

- [ ] **Step 4：实现支付通知 Handler**

调用 `payment.NewUnifiedWechatPayNotifyLogic(APIStore, WechatPayGateway).HandleNotify`，使用专用 raw JSON 输出；不得调用普通 `response.JSON`，不得记录原始 Body 或签名 Header。

- [ ] **Step 5：运行测试与骨架扫描**

Run: `cd backend && go test ./app/internal/handler/entitlement ./app/internal/handler/vip ./app/internal/handler/callback ./app/internal/server -run 'Test(Entitlement|VIP|WechatPayNotify).*' -count=1 && ! rg 'NotMigrated' app/internal/handler/{entitlement,vip,callback}`

Expected: PASS。

- [ ] **Step 6：提交**

```bash
git add backend/app/api/callback.api backend/app/internal/types/types.go backend/app/internal/handler/routes.go backend/app/internal/handler/routes_manifest_gen.go backend/app/internal/handler/entitlement backend/app/internal/handler/vip backend/app/internal/handler/callback backend/app/internal/server/remaining_api_test.go backend/app/internal/server/resource_api_test.go
git commit -m "refactor: 迁移权益与支付 API Handler"
```

### Task 9：把增长活动补入正式契约并迁移 Handler

**文件：**
- Create: `backend/app/api/growth.api`
- Modify: `backend/app/api/app.api`
- Create: `backend/app/internal/handler/growth/list_active_growth_campaigns_handler.go`
- Create: `backend/app/internal/handler/growth/get_merchant_growth_tasks_handler.go`
- Create: `backend/app/internal/handler/admingrowth/admin_list_growth_campaigns_handler.go`
- Create: `backend/app/internal/handler/admingrowth/admin_create_growth_campaign_handler.go`
- Create: `backend/app/internal/handler/admingrowth/admin_update_growth_campaign_handler.go`
- Create: `backend/app/internal/handler/admingrowth/admin_list_growth_rules_handler.go`
- Create: `backend/app/internal/handler/admingrowth/admin_create_growth_rule_handler.go`
- Create: `backend/app/internal/handler/admingrowth/admin_update_growth_rule_handler.go`
- Create: `backend/app/internal/handler/admingrowth/admin_list_growth_reward_grants_handler.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`
- Modify: `backend/scripts/api_contract.test.mjs`

**接口：**
- Adds: 公开活动 1、商家任务 1、后台活动/规则/发放记录 7 个正式契约 Handler。
- Consumes: growth/admin Logic 现有 JSON 字段与 Task 3 管理员上下文。

- [ ] **Step 1：写增长契约静态测试**

断言九条当前有效运行时路由全部进入 `growth.api`；DTO 字段与 `PublicGrowthCampaignItem`、`GrowthTaskItem`、`GrowthCampaignItem`、`GrowthRuleItem`、`GrowthGrantItem` 的现有 JSON 字段一一对应。

- [ ] **Step 2：运行静态测试确认失败**

Run: `cd backend && node --test scripts/api_contract.test.mjs`

Expected: FAIL，提示 `growth.api` 或目标路由不存在。

- [ ] **Step 3：实现契约并重新生成**

公开/商家接口 prefix 为 `/api/v1`；后台 group 使用 `/api/v1/admin` 并声明 `middleware: AdminAuth`。执行 `node scripts/api_codegen.mjs --write`。

- [ ] **Step 4：把增长路由测试切到生成 Server**

覆盖公开列表、商家越权、后台状态过滤、创建/更新活动、规则创建/更新、发放记录过滤和 operatorID 只能来自管理员 Token。

- [ ] **Step 5：实现增长 Handler 并运行测试**

Run: `cd backend && go test ./app/internal/handler/growth ./app/internal/handler/admingrowth ./app/internal/server -run 'TestGrowth.*' -count=1 && ! rg 'NotMigrated' app/internal/handler/{growth,admingrowth}`

Expected: PASS。

- [ ] **Step 6：提交**

```bash
git add backend/app/api/app.api backend/app/api/growth.api backend/app/internal/types/types.go backend/app/internal/handler/routes.go backend/app/internal/handler/routes_manifest_gen.go backend/app/internal/handler/growth backend/app/internal/handler/admingrowth backend/app/internal/server/remaining_api_test.go backend/scripts/api_contract.test.mjs
git commit -m "feat: 将增长活动纳入 API 契约"
```

### Task 10：迁移公开地图、商家绑定与后台地图接口

**文件：**
- Create/Modify: `backend/app/internal/handler/map/*_handler.go`
- Modify: `backend/app/api/map.api`
- Modify: `backend/app/internal/server/map_api_test.go`
- Modify: `backend/scripts/api_contract.test.mjs`

**接口：**
- Implements: map 28 个契约 Handler。
- Consumes: Task 3 身份/商家权限与现有 map Logic。

- [ ] **Step 1：修正后台地图契约中间件并生成**

公开 `/api/v1/map/**` 不声明 AdminAuth；`/api/v1/admin/map/**` group 声明 `middleware: AdminAuth`。运行 goctl validate 与 `api_codegen --write`。

- [ ] **Step 2：把地图测试切到生成 Server 并确认失败**

覆盖公开场景/对象/分类/搜索/附近点位、纠错/风险上报、商家位置上下文、绑定候选/状态/申请，以及后台场景、对象、分类、批量生成和绑定审核。

Run: `cd backend && go test ./app/internal/server -run 'TestMap.*Generated' -count=1`

Expected: FAIL，命中迁移骨架。

- [ ] **Step 3：实现公开地图和商家绑定 Handler**

路径参数统一来自 `pathvar`；视野边界与 zoom 查询使用生成 DTO；绑定候选和绑定申请按当前规则执行 RequiredUser/RequireMerchant；纠错和风险上报只从 Token 写入 userID。

- [ ] **Step 4：实现后台地图 Handler**

管理员身份从中间件 context 读取；保存/发布/状态/审核操作把 operatorID 传给现有 Logic，不接受 Body 中的管理员 ID。

- [ ] **Step 5：运行地图测试与骨架扫描**

Run: `cd backend && go test ./app/internal/handler/map ./app/internal/server -run 'TestMap.*' -count=1 && ! rg 'NotMigrated' app/internal/handler/map`

Expected: PASS。

- [ ] **Step 6：提交**

```bash
git add backend/app/api/map.api backend/app/internal/types/types.go backend/app/internal/handler/routes.go backend/app/internal/handler/routes_manifest_gen.go backend/app/internal/handler/map backend/app/internal/server/map_api_test.go backend/scripts/api_contract.test.mjs
git commit -m "refactor: 迁移地图 API Handler"
```

### Task 11：迁移后台管理全部 Handler 并补齐后台资源契约

**文件：**
- Modify: `backend/app/api/admin.api`
- Create/Modify: `backend/app/internal/handler/adminpermission/*_handler.go`
- Create/Modify: `backend/app/internal/handler/admindashboard/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminresource/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminentitlement/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminlog/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminmerchant/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminbannertopic/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminhotsearchkeyword/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminvipconfig/*_handler.go`
- Create/Modify: `backend/app/internal/handler/adminconfig/*_handler.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`
- Modify: `backend/scripts/api_contract.test.mjs`

**接口：**
- Adds: `GET /api/v1/admin/resources` -> `AdminListResources`，使用包含 `cityCode`、`typeCode`、`status`、`page`、`pageSize` 的独立 `AdminResourcesReq`，响应复用 `AdminPendingResourcesResp`。
- Implements: admin.api 当前 35 个 Handler 加新增后台资源列表。

- [ ] **Step 1：写后台契约与中间件静态测试**

除 `adminauth` group 外，admin.api 的每个 `@server` 块必须包含 `middleware: AdminAuth`；增加 `AdminListResources` 契约并断言不会与 `/resources/pending` 冲突。

- [ ] **Step 2：运行契约测试确认失败并重新生成**

Run: `cd backend && node --test scripts/api_contract.test.mjs`

Expected: FAIL，提示中间件或后台资源列表缺失。修正契约后运行 `node scripts/api_codegen.mjs --write`。

- [ ] **Step 3：把后台 API 测试切到生成 Server**

覆盖管理员/角色权限、dashboard、资源列表/待审/审核、举报、权益发放、运营/搜索日志、生命周期任务、商家列表、Banner、热词、VIP 配置和资源类型配置。

- [ ] **Step 4：实现后台权限、资源和运营 Handler**

所有 operatorID/actor 从 AdminAuth context 获取；权限管理仍要求超级管理员；生命周期任务调用现有 `task.NewResourceLifecycleTask`；未知后台路径模块默认拒绝。

- [ ] **Step 5：实现后台配置 Handler**

Banner、热词、VIP 和资源类型的创建/更新使用 path 参数覆盖 Body 中可伪造的主键；保留资源类型 optimistic version 校验和中文冲突提示。

- [ ] **Step 6：运行后台测试与全局骨架扫描**

Run: `cd backend && go test ./app/internal/handler/... ./app/internal/server -run 'TestAdmin.*' -count=1 && ! rg 'NotMigrated' app/internal/handler --glob '!handlerx/not_migrated.go' --glob '!handlerx/not_migrated_test.go'`

Expected: PASS，除定义与测试外无 `NotMigrated` 调用。

- [ ] **Step 7：提交**

```bash
git add backend/app/api/admin.api backend/app/internal/types/types.go backend/app/internal/handler backend/app/internal/server/remaining_api_test.go backend/scripts/api_contract.test.mjs
git commit -m "refactor: 迁移后台管理 API Handler"
```

### Task 12：将契约与生成路由变成强一致门禁

**文件：**
- Modify: `backend/scripts/api_route_inventory.mjs`
- Modify: `backend/scripts/api_route_inventory.test.mjs`
- Modify: `backend/scripts/api_codegen.mjs`
- Modify: `backend/scripts/api_codegen.test.mjs`
- Modify: `backend/app/internal/server/generated_api_test.go`
- Modify: `Makefile`

**接口：**
- Produces: `checkContractGeneratedParity()`，精确比较 method/path/handler。
- Produces: `checkNoMigrationStubs(handlerDir)`。
- Consumes: `handler.ContractRoutes()` 与迁移期 `newGeneratedAPIServer` 的 `rest.Server.Routes()`。

- [ ] **Step 1：写最终一致性失败测试**

测试篡改 prefix、method、Handler 名称、制造重复路由、加入 `NotMigrated` 调用，分别断言 `--check` 非零退出并输出具体指纹。

- [ ] **Step 2：实现生成层强门禁**

`api_codegen --check` 同时验证：goctl 版本、types 一致、routes 一致、manifest 一致、契约无重复、生成路由无重复、无迁移骨架。不得检查或依赖旧 Router。

- [ ] **Step 3：实现生成 Server 全集比较测试**

测试把 `handler.ContractRoutes()` 转成 `METHOD path` 集合，再与 `newGeneratedAPIServer` 的 `srv.Routes()` 精确比较，不允许额外路由。缺失、多余和重复均输出排序后的差异。生产 `NewGoZeroServer` 的 health/ready 例外在 Task 13 切换后单独验证。

- [ ] **Step 4：把强门禁加入 make check**

`check-backend` 顺序固定为：Node 静态测试、`check-api-generated`、Go 全量测试、go vet。

- [ ] **Step 5：运行门禁**

Run: `make check-api-generated && cd backend && node --test scripts/api_route_inventory.test.mjs scripts/api_codegen.test.mjs && go test ./app/internal/server -run 'TestGeneratedRouteParity' -count=1`

Expected: PASS。

- [ ] **Step 6：提交**

```bash
git add Makefile backend/scripts/api_route_inventory.mjs backend/scripts/api_route_inventory.test.mjs backend/scripts/api_codegen.mjs backend/scripts/api_codegen.test.mjs backend/app/internal/server/generated_api_test.go
git commit -m "test: 强制 API 契约与生成路由一致"
```

### Task 13：切换生产 Server 并删除兼容 API Router

**文件：**
- Modify: `backend/app/internal/server/gozero_server.go`
- Modify: `backend/app/internal/server/gozero_server_test.go`
- Modify: `backend/app/app.go`
- Modify: `backend/app/internal/svc/service_context.go`
- Delete: `backend/app/internal/server/goctl_routes.go`
- Delete: `backend/app/internal/server/auth_routes.go`
- Delete: `backend/app/internal/server/domain_routes.go`
- Delete: `backend/app/internal/server/map_routes.go`
- Delete or reduce to non-route helpers: `backend/app/internal/server/api.go`
- Delete: `backend/app/internal/server/api_production_test.go`
- Delete: `backend/app/internal/handler/handlerx/not_migrated.go`
- Delete: `backend/app/internal/handler/handlerx/not_migrated_test.go`
- Modify: all remaining `backend/app/internal/server/*_api_test.go` references to `NewAPIRouter`

**接口：**
- Changes: `NewGoZeroServer(cfg, svcCtx, adminHandler) (*rest.Server, error)`，删除 `apiHandler` 参数。
- Consumes: `handler.RegisterHandlers(srv, svcCtx)`。

- [ ] **Step 1：先改生产 Server 测试并确认失败**

断言：全部 ContractRoutes 已注册；未知 `/api/v1/not-declared` 返回 404；对仅允许 POST 的已声明路径发送 GET 返回 405；`/admin/assets/index.js` 仍由静态 Handler 返回；health/ready 正常；构造函数不再接受 apiHandler。

- [ ] **Step 2：运行切换测试确认失败**

Run: `cd backend && go test ./app/internal/server -run 'TestNewGoZeroServer|TestGeneratedRouteParity' -count=1`

Expected: 编译或断言 FAIL，因为旧构造函数仍含 fallback。

- [ ] **Step 3：切换 Server 与 app 启动**

`NewGoZeroServer` 注册 health、ready、`handler.RegisterHandlers`；NotFound 只在 `isAdminPath` 时调用 admin Handler，否则 `http.NotFound`。`app.go` 删除 `NewProductionAPIRouter` 及全部 `With...` options，直接传 `svcCtx` 和 `adminHandler`。

同时把 `TestGeneratedRouteParity` 切到正式 `NewGoZeroServer`：运行时集合必须等于 `handler.ContractRoutes()` 加 `GET /healthz`、`GET /readyz`，不得存在其他 API 路由。

- [ ] **Step 4：迁移生产依赖校验**

把旧 `validateProductionAPIRouterDependencies` 改成 `svc.ValidateAPIServiceContext(svcCtx) error` 或等价明确入口，在生产启动前检查 APIStore、Token、上传、微信、短信、支付、回调和地图依赖；错误文本列出缺失依赖。不得按依赖有无减少路由注册。

- [ ] **Step 5：删除旧 Router 与遗留测试入口**

将仍被 Handler 使用的纯辅助函数迁到 `handlerx` 或对应领域后，删除四个路由文件和 `api.go` 的路由内容；删除注释中的已下线认证 Router；所有 API 行为测试只使用生成 Server。

- [ ] **Step 6：执行引用与 fallback 扫描**

Run: `! rg 'NewAPIRouter|NewProductionAPIRouter|registerOptionalDomainRoutes|registerMapRoutes|registerAuthRoutes|goctl_routes|HasPrefix\(.*"/api/"' backend/app --glob '*.go'`

Expected: 退出 0。

- [ ] **Step 7：运行后端全量测试**

Run: `cd backend && node --test scripts/*.test.mjs && go test ./... && go vet ./...`

Expected: 全部 PASS。

- [ ] **Step 8：提交**

```bash
git add backend/app/app.go backend/app/internal/server backend/app/internal/svc/service_context.go backend/app/internal/handler/handlerx
git commit -m "refactor: 删除兼容 API Router"
```

### Task 14：更新架构与维护文档并完成全量验收

**文件：**
- Modify: `docs/product/technical-architecture.md`
- Modify: `docs/product/api-implementation-checklist.md`
- Modify: `docs/product/local-dev-runbook.md`
- Modify: `README.md`（仅当其中仍描述旧路由或旧生成命令）

**接口：**
- Documents: 单轨请求链路、固定 goctl 1.7.5、`make generate-api`、`make check-api-generated`、新增 API 流程和允许的平台路由。

- [ ] **Step 1：写文档过期内容检查**

在现有 `backend/scripts/api_contract.test.mjs` 增加断言，产品文档不得再出现“兼容 API Router”“未迁移端点由 fallback”“迁移期双轨制”；技术架构必须出现 `make check-api-generated`。

- [ ] **Step 2：运行文档测试确认失败**

Run: `cd backend && node --test scripts/api_contract.test.mjs`

Expected: FAIL，命中过期描述。

- [ ] **Step 3：更新文档**

启动流程改为创建 ServiceContext 后直接创建 go-zero Server；路由章节说明 `.api -> goctl routes -> Handler -> Logic`；新增 API 指引要求先改 `.api`、运行生成、实现 Handler、增加权限/行为测试、运行生成门禁；本地手册明确全局 `GOCTL_HOME` 不参与项目生成。

- [ ] **Step 4：执行格式与全量验证**

Run: `goctl api validate --api backend/app/api/app.api`

Run: `make check-api-generated`

Run: `make check`

Run: `git diff --check`

Expected: 四条命令全部退出 0。

- [ ] **Step 5：执行最终技术债关闭扫描**

Run: `! rg 'NewAPIRouter|NewProductionAPIRouter|兼容 API Router|迁移期双轨制|NotMigrated' backend docs/product --glob '!docs/superpowers/**'`

Expected: 退出 0。

- [ ] **Step 6：提交文档与最终门禁**

```bash
git add docs/product README.md backend/scripts/api_contract.test.mjs
git commit -m "docs: 更新 API 单轨架构维护说明"
```

---

## 最终验收清单

- [ ] `app.api` import 链包含所有有效业务接口，包含增长、支付通知和内容审核回调。
- [ ] goctl 生成的 routes 与契约 method/path/handler 精确一致。
- [ ] 运行时 routes 与契约一致，额外只有 healthz/readyz。
- [ ] 未知 `/api/` 路径返回 404，`/admin` 静态资源正常。
- [ ] 用户、管理员、后台模块、商家权限和回调验签测试通过。
- [ ] 无 `NotMigrated`、`NewAPIRouter`、`NewProductionAPIRouter` 或旧 register 函数引用。
- [ ] `goctl api validate`、`make check-api-generated`、`make check`、`git diff --check` 全部通过。
- [ ] 技术架构、API 清单和本地手册均描述单轨架构。
