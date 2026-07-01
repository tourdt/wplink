# Resource Contact Unlock Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现资源详情页登录后受控解锁完整电话/微信，成功解锁才计入联系指标，首期不强制绑定手机号。

**Architecture:** 公开资源详情继续返回脱敏联系方式；`POST /resources/{resourceId}/contact-events` 在 `phone`/`wechat` 动作下升级为受控解锁接口。后端逻辑层完成登录、资源状态、商家归属和联系方式缺失校验，模型层提供完整联系方式查询；小程序资源详情页复用现有 `requireLogin()`，拿到后端完整联系方式后再复制或拨号。

**Tech Stack:** Go 1.25、go-zero API 契约、PostgreSQL、uni-app/Vue 3、小程序 `uni` API、Node.js 内置测试。

---

### Task 1: 后端联系解锁逻辑

**Files:**
- Modify: `backend/app/internal/model/resource_contact_event_model.go`
- Modify: `backend/app/internal/logic/metrics/record_contact_logic.go`
- Test: `backend/app/internal/logic/metrics/record_contact_logic_test.go`

- [ ] **Step 1: Write failing metrics logic tests**

Add tests that prove `phone` and `wechat` require登录用户、返回完整联系方式、缺失联系方式不计数、自己资源不计数：

```go
func TestRecordContactReturnsWechatOnlyAfterSuccessfulUnlock(t *testing.T) {
	store := &fakeContactStore{
		contact: model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, Wechat: "stock-demo"},
		eventResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"},
	}
	logic := NewRecordContactLogic(store)

	resp, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: " resource-1 ", UserID: " user-1 ", Action: "wechat"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}
	if resp.Wechat != "stock-demo" || resp.Action != "wechat" || resp.Message != "微信号已解锁" {
		t.Fatalf("resp = %#v, want unlocked wechat", resp)
	}
	if store.metricDelta.WechatCopyCount != 1 || store.metricDelta.ContactClickCount != 1 {
		t.Fatalf("metricDelta = %#v, want wechat contact metric", store.metricDelta)
	}
}
```

- [ ] **Step 2: Verify backend RED**

Run:

```bash
cd backend && go test ./app/internal/logic/metrics
```

Expected: FAIL because `ResourceContactUnlockInfo`, response fields, and unlock validation are not implemented yet.

- [ ] **Step 3: Implement minimal backend logic**

Add `model.ResourceContactUnlockInfo`, `GetResourceContactUnlockInfo(ctx, resourceID)` on `ResourceContactEventModel`, extend `ContactStore`, and update `RecordContact` so:

- `phone`/`wechat` require non-empty `UserID`.
- Load resource contact info before writing event or metric.
- Reject non-`published` resources, expired resources, inactive/deleted merchants, missing contact data, and current user's managed merchant.
- Return `Phone` or `Wechat` only after event and metric writes succeed.
- Keep `merchant_home`、`merchant_profile`、`share` as non-unlock events.

- [ ] **Step 4: Verify backend GREEN**

Run:

```bash
cd backend && go test ./app/internal/logic/metrics
```

Expected: PASS.

### Task 2: HTTP route and API contract

**Files:**
- Modify: `backend/app/api/resource.api`
- Modify: `backend/app/internal/server/auth_routes.go`
- Modify: `backend/app/internal/server/api.go`
- Modify: `backend/app/internal/types/types.go` only through goctl generation if generated output changes
- Test: `backend/app/internal/server/resource_api_test.go`

- [ ] **Step 1: Write failing router tests**

Update contact route tests so anonymous `wechat` returns unauthorized, authorized `phone` ignores body `userId`, and success response includes full contact:

```go
anonymousReq := httptest.NewRequest(http.MethodPost, "/api/v1/resources/resource-1/contact-events", strings.NewReader(`{"action":"wechat"}`))
router.ServeHTTP(anonymousRec, anonymousReq)
decodeEnvelopeError(t, anonymousRec, http.StatusUnauthorized)
```

- [ ] **Step 2: Verify router RED**

Run:

```bash
cd backend && go test ./app/internal/server -run 'TestResourceAPIRouter.*Contact'
```

Expected: FAIL because route still allows anonymous `phone`/`wechat`.

- [ ] **Step 3: Implement route and contract changes**

Update `ContactEventResp` in `backend/app/api/resource.api` to:

```go
type ContactEventResp {
	Message string `json:"message"`
	Action  string `json:"action,optional"`
	Phone   string `json:"phone,optional"`
	Wechat  string `json:"wechat,optional"`
}
```

In `backend/app/internal/server/api.go`, require token for `phone` and `wechat` before calling metrics logic; keep optional token parsing for `merchant_home`、`merchant_profile`、`share`.

Run goctl generation only if needed:

```bash
cd backend/app && goctl api go -api ./api/app.api -dir . -home ./goctl
```

- [ ] **Step 4: Verify route GREEN**

Run:

```bash
cd backend && go test ./app/internal/server -run 'TestResourceAPIRouter.*Contact'
```

Expected: PASS.

### Task 3: 小程序资源详情联系交互

**Files:**
- Modify: `wxapp/api/resource.js`
- Modify: `wxapp/pages/resource/detail.vue`
- Test: `wxapp/pages/resource/detail.test.mjs`

- [ ] **Step 1: Write failing resource detail tests**

Add source-level tests that require `requireLogin`, backend contact response use, and removal of protected-contact fallback:

```js
test('resource detail unlocks contact through backend before copy or call', () => {
  assert.match(source, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.match(source, /if \(!requireLogin\(\)\) return false/)
  assert.match(source, /const resp = await recordResourceContact\(resource\.value\.id, action\)/)
  assert.match(source, /uni\.setClipboardData\(\{ data: resp\.wechat \}\)/)
  assert.match(source, /uni\.makePhoneCall\(\{ phoneNumber: resp\.phone \}\)/)
  assert.equal(source.includes('已记录联系，完整微信由平台保护'), false)
})
```

- [ ] **Step 2: Verify wxapp RED**

Run:

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

Expected: FAIL because resource detail still uses masked values.

- [ ] **Step 3: Implement minimal resource detail changes**

Change `recordContact(action)` to require login and return backend response. Change `callPhone()` and `copyWechat()` to use `resp.phone` and `resp.wechat`; show friendly Chinese errors when backend returns no contact.

- [ ] **Step 4: Verify wxapp GREEN**

Run:

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

Expected: PASS.

### Task 4: Full verification and commit

**Files:**
- All touched files above

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
cd backend && go test ./app/internal/logic/metrics ./app/internal/server
```

Expected: PASS.

- [ ] **Step 2: Run focused wxapp tests**

Run:

```bash
cd wxapp && node --test pages/resource/detail.test.mjs
```

Expected: PASS.

- [ ] **Step 3: Validate API contract**

Run:

```bash
cd backend && goctl api validate --api app/api/app.api
```

Expected: PASS.

- [ ] **Step 4: Review diff scope**

Run:

```bash
git diff --stat
git status --short
```

Expected: Only联系方式解锁相关 files plus the plan are changed; existing unrelated `wxapp/pages/merchant/*` changes remain unstaged.

- [ ] **Step 5: Commit only this feature**

Run:

```bash
git add docs/superpowers/plans/2026-07-01-resource-contact-unlock.md backend/app/api/resource.api backend/app/internal/model/resource_contact_event_model.go backend/app/internal/logic/metrics/record_contact_logic.go backend/app/internal/logic/metrics/record_contact_logic_test.go backend/app/internal/server/auth_routes.go backend/app/internal/server/api.go backend/app/internal/server/resource_api_test.go backend/app/internal/types/types.go wxapp/pages/resource/detail.vue wxapp/pages/resource/detail.test.mjs
git commit -m "feat: unlock resource contact after login"
```

Expected: Commit succeeds without staging unrelated merchant-page changes.
