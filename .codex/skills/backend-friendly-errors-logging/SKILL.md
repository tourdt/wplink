---
name: backend-friendly-errors-logging
description: Use when implementing or modifying WPMall Go/go-zero backend services, APIs, business logic, validation, persistence, jobs, integrations, or WeChat Mini Program code where code style, goctl-compatible generated code, detailed Chinese code comments, diagnostic logs, server errors, or frontend-facing error messages must be standardized, complete, Chinese-friendly, and safe for users.
---

# Backend Friendly Errors Logging

## Overview

Use this skill to make backend and Mini Program changes easier to understand, diagnose, maintain, and safely expose to frontend users. Prefer project conventions first, then apply the standards below.

Core principle: **logs are for engineers, returned errors are for users, comments are for future maintainers.**

## Required Workflow

1. Read nearby backend code before editing; follow existing response, logging, transaction, and error patterns.
2. If the change touches go-zero API/RPC/model generated surfaces, identify the source file and goctl command first; do not hand-edit generated code.
3. Identify user-facing failure points: validation, permission, missing data, state conflicts, external services, database writes, concurrency, and timeouts.
4. Keep code simple and scoped; add no new abstraction unless it removes real duplication or matches an existing local pattern.
5. Add Chinese comments where business intent or non-obvious control flow would otherwise require rereading multiple files.
6. Add diagnostic logs at decision boundaries and failure paths, with enough context to reproduce the issue.
7. Return friendly frontend messages; never expose SQL, stack traces, raw dependency errors, tokens, secrets, or internal table/field names.
8. Run `gofmt`/`go test` or the repository's existing verification commands; if not possible, state exactly what could not be verified.

## Code Style

Follow local conventions first. For Go backend code:

- Use idiomatic Go names, small functions, early returns, and explicit error handling.
- Keep business logic in logic/service layers; keep handlers thin and focused on request parsing and response mapping.
- Avoid broad rewrites, speculative helpers, hidden global state, and reflection-heavy code unless the codebase already uses them.
- Prefer typed constants/enums for business states instead of repeated magic strings or numbers.
- Wrap or log internal errors with operation context, but return safe user-facing messages.
- Run `gofmt` on touched Go files and keep imports organized by the formatter.

For Mini Program code:

- Keep page methods small and named by user action or lifecycle purpose.
- Keep UI state updates explicit; avoid scattered `setData` calls that make data flow hard to trace.
- Centralize API calls through existing request helpers when present.
- Keep permission, login, payment, upload, location, and subscription flows readable and guarded.
- Avoid adding hidden global state unless the project already uses that pattern.

## Chinese Comments

Add comments for:

- Business rules, especially money, inventory, order state, permissions, idempotency, retries, and data correction.
- Non-obvious assumptions, such as "只允许默认仓库" or "重复提交时返回已有结果".
- Branches that protect data accuracy or user safety.
- Server error handling that intentionally hides internal details from frontend users.
- Generated-code boundaries, such as "该文件由 goctl 生成，请修改 .api 后重新生成".
- Mini Program lifecycle, cross-page state, permission, payment, login, upload, sharing, and cache logic when the intent is not obvious.

Do not comment obvious assignments. Keep comments close to the code they explain. Detailed comments should explain **why** and **business impact**, not repeat what the next line already says.

```go
// 库存扣减必须在事务内完成，避免并发下同一个 SKU 被重复扣减。
if err := deductStock(ctx, tx, skuId, quantity); err != nil {
    logx.Errorf("扣减库存失败: shopId=%d skuId=%d quantity=%d err=%+v", shopId, skuId, quantity, err)
    return nil, errors.New("库存更新失败，请刷新后重试")
}
```

## Mini Program Comments

Add detailed Chinese comments in Mini Program JavaScript/TypeScript, WXML, WXSS, or component code when the code affects user flow or data correctness:

- Page lifecycle: why `onLoad`, `onShow`, `onPullDownRefresh`, or `onReachBottom` reloads specific data.
- API requests: what business data is fetched, why parameters are required, and how empty/error states are handled.
- User actions: why a button or gesture is disabled, debounced, confirmed, or redirected.
- Permission flows: login, phone number, location, camera, album, subscription message, and payment authorization.
- Cross-page data: route params, event channel, storage/cache keys, global app state, and tab refresh rules.
- UI state: loading, empty, disabled, selected, pagination, optimistic update, and rollback behavior.
- Complex style/layout decisions: sticky areas, safe-area handling, scroll containers, responsive grids, and z-index dependencies.

Do not comment every template binding or style declaration. Prefer comments that explain business intent, edge cases, and why a user-visible behavior exists.

```js
// 进入页面时优先使用路由传入的店铺 ID，避免用户从分享链接进入后读取到上一次缓存的店铺。
const shopId = options.shopId || wx.getStorageSync('currentShopId')

// 提交期间禁用按钮，防止用户连续点击导致重复创建订单。
this.setData({ submitting: true })
```

## Logging Rules

Log for engineers. Include identifiers and state, not secrets.

Required on failures:

- Operation name and business action.
- Stable identifiers: shopId, userId, orderId, skuId, requestId, jobId when available.
- Key state values that explain the branch.
- Wrapped/root error with enough detail for backend debugging.
- Server-side category, such as validation, permission, dependency, database, timeout, conflict, or unknown.

Required on high-risk success paths:

- State transitions, stock changes, payment/order changes, irreversible writes, external callbacks.

Avoid:

- Full request bodies with phone numbers, addresses, tokens, passwords, auth headers, or payment data.
- Log-only handling where the frontend receives a vague or raw error.
- Silent fallback when data may be inconsistent.
- Logging raw SQL with bound sensitive values.

Prefer a consistent, searchable log format:

```go
logx.Errorf("创建采购单失败: shopId=%d userId=%d supplierId=%d err=%+v", shopId, userId, supplierId, err)
```

## Frontend-Friendly Errors

Return messages in user language, not implementation language.

Good messages:

- "库存不足，请调整数量后重试"
- "订单状态已变化，请刷新页面后再操作"
- "保存失败，请检查网络后重试"
- "您没有权限操作该店铺"
- "商品已下架，请重新选择"

Bad messages:

- "sql: no rows in result set"
- "update pms_sku_inventories failed"
- "invalid status"
- "panic recovered"
- "系统错误"

When mapping errors:

| Backend cause | Frontend message |
| --- | --- |
| Validation failed | Point to the user action: "请填写货号" |
| Permission denied | "您没有权限进行此操作" |
| Missing record | "数据不存在或已被删除" |
| State conflict | "状态已变化，请刷新后重试" |
| Inventory conflict | "库存不足，请调整数量后重试" |
| External dependency | "服务暂时不可用，请稍后重试" |
| Database write failed | "保存失败，请稍后重试" |
| Timeout | "处理超时，请稍后重试" |
| Unknown internal error | "操作失败，请稍后重试" plus detailed backend log |

When a server error is not caused by user input, log the internal detail and return a stable, actionable Chinese message. Avoid returning raw `err.Error()` unless it is already a curated user-facing error from the local error package.

## Goctl and go-zero Rules

Generated server code must follow goctl/go-zero conventions:

- Treat `.api`, `.proto`, or schema/model definition files as the source of truth when code generation is involved.
- Regenerate with the repository's existing `goctl` command or scripts; preserve the package layout generated by goctl.
- Do not manually edit generated handler/types/model files when regeneration would overwrite the change.
- Put custom business logic in the generated logic extension points or existing logic files, not in generated templates.
- Keep request/response types compatible with `.api` definitions; do not add fields only in handwritten code.
- Do not rename generated route, handler, logic, svc, or types packages unless the API source changes accordingly.
- After generation, inspect diffs to ensure only expected files changed and remove accidental churn.

## Implementation Checklist

- Code follows local Go style, is `gofmt` formatted, and keeps the change scope narrow.
- goctl-generated surfaces are changed through the source definition and regeneration path.
- Comments explain why the business rule exists.
- Logs include operation name, IDs, key state, and root error.
- User errors are Chinese, specific, and actionable.
- Sensitive details are only in safe internal logs, never in frontend responses.
- Transaction or concurrency failures are logged with enough context to audit.
- Mini Program lifecycle, permission, API, cache, and cross-page state changes include useful Chinese comments.
- Tests or manual verification cover success and at least one expected failure.

## WPMall Go/go-zero Notes

When working in WPMall:

- Read `docs/backend_rules.md` if it exists, then read nearby files under `backend/app/internal/logic` before editing.
- Read API definitions under `backend/app/api` before changing request/response contracts.
- Read Mini Program code under `prototypes/wxapp-hifi` before editing prototype pages or components.
- Do not edit generated files such as `*_gen.go`.
- Prefer existing go-zero patterns, `logx`, service context models, and response conventions.
- For clothing ERP flows, always preserve SKU accuracy: style number, color, size, SKU matrix, stock quantity, order state, and warehouse assumptions.
- For inventory/order/purchase changes, log shopId, orderId, spuId, skuId, warehouseId, quantity, old/new state, and transaction failures when available.

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Returning raw backend error to frontend | Log raw error; return a friendly Chinese message |
| Adding logs only at the final catch | Log key branch decisions and failed dependencies |
| Commenting every line | Comment business intent and non-obvious safeguards |
| Logging sensitive request payloads | Log safe IDs and summarized state |
| Using "系统错误" for everything | Map common causes to actionable user messages |
| Ignoring concurrency/idempotency context | Log request key, existing state, and conflict reason |
| Hand-editing generated goctl files | Change the source `.api`/schema and regenerate |
| Adding complex helpers for one caller | Keep code local and simple unless reuse is real |
| Leaving Mini Program lifecycle or permission logic uncommented | Add Chinese comments explaining user flow, state source, and failure handling |
