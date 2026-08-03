# 本地真实微信登录与内容审核 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移除本地伪微信登录、伪手机号和伪 OpenID，使本地测试环境只接受真实微信身份并可正常调用真实内容审核。

**Architecture:** 在 `HTTPWechatSessionClient` 的登录和手机号授权入口本地拒绝历史 `local-dev-*` 凭证，其他 code 保持走微信真实接口。移除 `Wechat.AllowDevCode` 配置与所有文档引用，审核器不改动，继续使用资源创建人的真实 OpenID。

**Tech Stack:** Go 1.23、go-zero、Go 标准库 `net/http`、YAML 配置、Go 单元测试。

## Global Constraints

- 只修改真实微信登录/手机号授权、配置模板、配置文档和相关测试；不修改审核决策、重试策略、数据库数据或支付对账文件。
- 后端日志不得包含 code、OpenID、手机号、access token、AppSecret 或请求体。
- 登录和手机号授权错误必须使用中文且对前端可理解；微信接口原始错误仅保留在服务端日志。
- 不手工修改 goctl 生成文件；本任务不涉及 `.api` 变更。
- 每个行为变更先写测试并确认失败，再写最小实现；每个任务只暂存其列出的文件。

---

## 文件结构

| 文件 | 责任 |
|---|---|
| `backend/app/internal/logic/auth/wechat_session.go` | 拒绝历史开发凭证，保留真实微信登录和手机号授权。 |
| `backend/app/internal/logic/auth/wechat_session_test.go` | 覆盖真实微信成功链路与历史开发凭证的本地拒绝。 |
| `backend/app/internal/config/config.go` | 删除已废弃的 `Wechat.AllowDevCode` 配置字段。 |
| `backend/app/internal/config/production_validation.go` | 删除不再可能触发的开发登录配置校验。 |
| `backend/app/internal/config/load_test.go` | 验证废弃字段被严格 YAML 解析拒绝，并验证配置/文档无遗留引用。 |
| `backend/etc/app.yaml`、`backend/etc/app.yaml.example`、`backend/etc/app.production.yaml.example` | 移除废弃键，保持真实微信凭证占位符和审核配置。 |
| `docs/product/deployment-config.md`、`docs/product/production-release-checklist.md` | 将部署说明和发布清单收敛到真实微信链路。 |

### Task 1: 认证客户端拒绝历史开发凭证

**Files:**
- Modify: `backend/app/internal/logic/auth/wechat_session.go:73-200`
- Modify: `backend/app/internal/logic/auth/wechat_session_test.go:1-134`

**Interfaces:**
- Consumes: `HTTPWechatSessionClient.Code2Session(ctx, code)` 与 `GetPhoneNumber(ctx, code)`。
- Produces: `local-dev-*` 在本地返回 `errx.CodeUnauthorized`；真实 code 的接口签名和返回结构不变。

- [ ] **Step 1: 用失败测试定义登录与手机号授权的拒绝行为**

  删除当前两个允许或转发开发 code 的测试，导入 `wplink/backend/common/errx`，加入下列两个测试。测试 Transport 可以返回成功响应，但只要请求发生就将 `called` 设为 `true`；这保证断言检验的是没有任何微信网络调用。

  ```go
  func TestWechatSessionClientRejectsLegacyDevLoginCodeWithoutCallingWechat(t *testing.T) {
      called := false
      httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
          called = true
          return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser{Buffer: bytes.NewBufferString(`{"openid":"openid-1"}`)}, Header: make(http.Header)}, nil
      })}

      client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient)
      _, err := client.Code2Session(context.Background(), "local-dev-123")
      if errx.CodeOf(err) != errx.CodeUnauthorized || errx.PublicMessage(err) != "请在微信内重新登录" {
          t.Fatalf("Code2Session() error = %v, want local unauthorized error", err)
      }
      if called {
          t.Fatal("wechat session endpoint was called for legacy dev code")
      }
  }

  func TestWechatSessionClientRejectsLegacyDevPhoneCodeWithoutCallingWechat(t *testing.T) {
      called := false
      httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
          called = true
          return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser{Buffer: bytes.NewBufferString(`{"access_token":"access-token"}`)}, Header: make(http.Header)}, nil
      })}

      client := NewWechatSessionClient(config.WechatConfig{AppID: "wx-app", AppSecret: "wx-secret"}, "https://wechat.example.test/session", httpClient)
      _, err := client.GetPhoneNumber(context.Background(), "local-dev-phone-18800000000")
      if errx.CodeOf(err) != errx.CodeUnauthorized || errx.PublicMessage(err) != "请在微信内重新授权手机号" {
          t.Fatalf("GetPhoneNumber() error = %v, want local unauthorized error", err)
      }
      if called {
          t.Fatal("wechat phone endpoints were called for legacy dev code")
      }
  }
  ```

- [ ] **Step 2: 运行测试并确认失败**

  Run:

  ```bash
  go test ./app/internal/logic/auth -run 'TestWechatSessionClientRejectsLegacyDev'
  ```

  Expected: FAIL；现有代码会继续向微信请求或生成本地身份，而不是返回本地未授权错误。

- [ ] **Step 3: 编写最小认证实现**

  在 `Code2Session` 和 `GetPhoneNumber` 完成 `strings.TrimSpace` 与空值检查后，使用同一个仅检查前缀的私有函数识别历史凭证：

  ```go
  func isLegacyWechatDevCode(code string) bool {
      return strings.HasPrefix(strings.TrimSpace(code), "local-dev-")
  }
  ```

  两个入口分别记录不含敏感值的日志，并在任何 HTTP 请求前返回：

  ```go
  logx.Errorf("微信登录拒绝历史开发凭证: isLegacyDevCode=true")
  return WechatSession{}, errx.New(errx.CodeUnauthorized, "请在微信内重新登录")
  ```

  ```go
  logx.Errorf("微信手机号授权拒绝历史开发凭证: isLegacyDevCode=true")
  return WechatPhoneNumber{}, errx.New(errx.CodeUnauthorized, "请在微信内重新授权手机号")
  ```

  删除生成 `dev:` OpenID 和伪手机号的分支。同步从真实微信错误日志中移除 `isDevCode`、`allowDevCode` 字段，只保留现有安全的错误码和错误消息。

- [ ] **Step 4: 运行认证测试并确认通过**

  Run:

  ```bash
  gofmt -w app/internal/logic/auth/wechat_session.go app/internal/logic/auth/wechat_session_test.go
  go test ./app/internal/logic/auth
  ```

  Expected: PASS；已有真实 `jscode2session` 和手机号授权成功测试保持通过，新测试证明旧凭证无网络调用。

- [ ] **Step 5: 提交认证客户端变更**

  ```bash
  git add backend/app/internal/logic/auth/wechat_session.go backend/app/internal/logic/auth/wechat_session_test.go
  git commit -m "fix: 禁用微信开发登录凭证"
  ```

### Task 2: 删除配置开关并清理配置与部署文档

**Files:**
- Modify: `backend/app/internal/config/config.go:48-53`
- Modify: `backend/app/internal/config/production_validation.go:80-87`
- Modify: `backend/app/internal/config/load_test.go:8-140,310-328`
- Modify: `backend/etc/app.yaml:32-35`
- Modify: `backend/etc/app.yaml.example:31-34`
- Modify: `backend/etc/app.production.yaml.example:31-34`
- Modify: `docs/product/deployment-config.md:8-13,90-94`
- Modify: `docs/product/production-release-checklist.md:41-47`

**Interfaces:**
- Consumes: `config.Load(path)` 的严格 YAML 解析和 `config.ValidateForProduction(cfg)`。
- Produces: `WechatConfig` 仅保留 `AppID`、`AppSecret`；配置出现 `AllowDevCode` 时启动读取失败。

- [ ] **Step 1: 用失败测试定义废弃配置不可再使用**

  在 `load_test.go` 中加入一个最小配置文件测试；当前 `WechatConfig` 仍含该字段，因此此测试会错误地得到 `nil`：

  ```go
  func TestLoadRejectsDeprecatedWechatAllowDevCode(t *testing.T) {
      path := filepath.Join(t.TempDir(), "app.yaml")
      if err := os.WriteFile(path, []byte(`
  Wechat:
    AppID: "wx-app"
    AppSecret: "secret"
    AllowDevCode: true
  `), 0o600); err != nil {
          t.Fatalf("write config: %v", err)
      }

      _, err := Load(path)
      if err == nil || !strings.Contains(err.Error(), "AllowDevCode") {
          t.Fatalf("Load() error = %v, want deprecated field rejection", err)
      }
  }
  ```

  再加入扫描测试，覆盖三个 YAML 和两份用户文档：

  ```go
  func TestWechatConfigAndDocumentationDoNotReferenceLegacyDevCode(t *testing.T) {
      paths := []string{
          filepath.Join("..", "..", "..", "etc", "app.yaml"),
          filepath.Join("..", "..", "..", "etc", "app.yaml.example"),
          filepath.Join("..", "..", "..", "etc", "app.production.yaml.example"),
          filepath.Join("..", "..", "..", "..", "docs", "product", "deployment-config.md"),
          filepath.Join("..", "..", "..", "..", "docs", "product", "production-release-checklist.md"),
      }
      for _, path := range paths {
          content, err := os.ReadFile(path)
          if err != nil {
              t.Fatalf("read %s: %v", path, err)
          }
          if strings.Contains(string(content), "AllowDevCode") || strings.Contains(string(content), "local-dev-") {
              t.Fatalf("%s still references legacy WeChat development login", path)
          }
      }
  }
  ```

- [ ] **Step 2: 运行配置测试并确认失败**

  Run:

  ```bash
  go test ./app/internal/config -run 'TestLoadRejectsDeprecatedWechatAllowDevCode|TestWechatConfigAndDocumentationDoNotReferenceLegacyDevCode'
  ```

  Expected: FAIL；`AllowDevCode` 仍被接受，配置和文档仍含历史开发登录说明。

- [ ] **Step 3: 编写最小配置和文档实现**

  从 `WechatConfig` 删除 `AllowDevCode` 字段，并删除 `ValidateForProduction` 中该字段的检查。更新已有加载测试：去掉 YAML fixture 的 `AllowDevCode` 和对应断言；把本地配置测试改为断言 `RuntimeMode == "development"`、`ContentAudit.Enabled == true`，说明本地默认走真实内容审核。

  从三个 YAML 文件删除 `AllowDevCode` 行；保持 `Wechat.AppID`、`Wechat.AppSecret` 环境变量占位符不变，不把任何真实凭证写入仓库。将 `deployment-config.md` 文件列表中的“关闭微信开发 code”改为真实微信链路说明，并把“开启开发登录 fallback”改为“微信凭证未配置”；再将微信章节改为“微信登录和手机号授权均使用真实微信接口，测试环境需配置测试小程序 AppID/AppSecret”。从生产发布清单删除 `Wechat.AllowDevCode: false` 项。

- [ ] **Step 4: 运行配置测试并确认通过**

  Run:

  ```bash
  gofmt -w app/internal/config/config.go app/internal/config/production_validation.go app/internal/config/load_test.go
  go test ./app/internal/config
  ```

  Expected: PASS；废弃键触发严格 YAML 错误，所有受控配置和文档均无开发登录遗留引用。

- [ ] **Step 5: 提交配置与文档变更**

  ```bash
  git add backend/app/internal/config/config.go backend/app/internal/config/production_validation.go backend/app/internal/config/load_test.go backend/etc/app.yaml backend/etc/app.yaml.example backend/etc/app.production.yaml.example docs/product/deployment-config.md docs/product/production-release-checklist.md
  git commit -m "fix: 统一真实微信测试配置"
  ```

### Task 3: 全量回归与真实链路验收

**Files:**
- Verify only: `backend/app/internal/logic/contentaudit/wechat_auditor.go`
- Verify only: `backend/app/internal/logic/contentaudit/wechat_auditor_test.go`

**Interfaces:**
- Consumes: 真实登录保存的 `users.wechat_openid` 与现有 `WechatAuditor.AuditResource`。
- Produces: 确认审核器无需改动，并验证改动未破坏认证、配置或内容审核单元测试。

- [ ] **Step 1: 运行相关 Go 回归测试**

  ```bash
  go test ./app/internal/logic/auth ./app/internal/config ./app/internal/logic/contentaudit ./app/internal/logic/resource
  ```

  Expected: PASS；内容审核测试仍验证 `openid` 被提交给审核请求，认证和配置测试验证不再产生本地伪身份。

- [ ] **Step 2: 检查格式、遗留引用和工作区范围**

  ```bash
  rg -n 'AllowDevCode|local-dev-|dev:local-dev-' backend/etc backend/app docs/product
  git diff --check
  git status --short
  ```

  Expected: `rg` 无匹配；`git diff --check` 无输出；变更仅为本计划列出的文件及用户已有的支付对账改动。

- [ ] **Step 3: 执行人工真实微信验收**

  以测试小程序的真实 AppID/AppSecret 启动后端，在微信开发者工具或真机调用 `wx.login()`，将获得的 code 发送至 `POST /api/v1/auth/wechat-login`；再使用微信手机号授权 code 调用对应授权接口。创建一条资源并确认文字审核返回结果；若启用图片审核，确认微信后台已经配置可访问的 HTTPS 回调地址 `/api/v1/wechat/content-audit/media-callback`，再上传图片并确认回调能完成资源状态流转。

- [ ] **Step 4: 提交前复跑并记录结果**

  ```bash
  go test ./app/internal/logic/auth ./app/internal/config ./app/internal/logic/contentaudit ./app/internal/logic/resource
  git diff --check
  ```

  Expected: 全部通过；如果真实微信凭证或公网回调尚未配置，保留单元测试结果并明确记录人工验收阻塞原因，不以模拟凭证替代。
