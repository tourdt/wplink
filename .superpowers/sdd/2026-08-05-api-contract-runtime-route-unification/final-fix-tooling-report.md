# API 工具链最终 Minor 修复报告

## 结论

- 已关闭最终审查中的两个工具链 Minor。
- 路由盘点 CLI 已改为比较当前 `.api` 契约与 goctl 生成路由，不再读取已删除的 Router 文件。
- 关键字 group 与真实 group 发生标识符碰撞时，生成器会分配稳定且唯一的 import alias；现有无碰撞生成结果保持不变。

## 修复内容

### 1. 当前路由盘点

`backend/scripts/api_route_inventory.mjs` 的 CLI 现在读取：

- 契约源：`app/api/app.api` 及其 import 链；
- 生成路由：`app/internal/handler/routes.go`。

CLI 先调用既有 `checkContractGeneratedParity` 校验 method、path、Handler 与 group/package；出现差异时复用逐路由“缺失/多余”诊断并以非零状态退出。文本模式输出双方数量与一致性，JSON 模式输出稳定排序后的双方完整路由清单、相对源码位置和 parity 摘要。

### 2. 关键字 group 唯一别名

`backend/scripts/api_codegen.mjs` 在规范化关键字 group 前，先收集并预占所有真实 group/package 标识符。关键字 group 仍优先使用原有 `${group}handler` 别名；只有发生碰撞时才从后缀 `2` 起递增，保证结果确定且不改变当前无碰撞输出。

`group: map` 与 `group: maphandler` 并存时：

- `map` 目录文件仍声明 `package maphandler`，保持目录内现有与新增 Handler 的 package 一致；
- `routes.go` 对 `handler/map` 使用显式 import alias `maphandler2`；
- `handler/maphandler` 使用 `maphandler`；
- 两个 selector 唯一，重复执行生成结果完全相同。

Go 的显式 import alias 不要求与被导入包的声明名相同，因此上述拆分是合法且可编译的；当前仓库 `go test ./...` 已通过。

## TDD 证据

- RED：CLI text/json smoke tests 均因读取已删除 `app/internal/server/api.go` 返回 ENOENT。
- GREEN：CLI text/json tests 2/2 通过。
- RED：`map`/`maphandler` fixture 生成出两个同名 `maphandler` import，唯一别名断言失败。
- GREEN：冲突 fixture 生成 `maphandler2`/`maphandler`，保留 `package maphandler`，连续两次生成文本一致。

## 验证结果

- `node --test scripts/api_route_inventory.test.mjs`：15/15 PASS。
- `node --test scripts/api_codegen.test.mjs`：15/15 PASS。
- `node scripts/api_route_inventory.mjs`：exit 0，契约路由 139、生成路由 139、一致。
- `node scripts/api_route_inventory.mjs --json`：exit 0，parity 为 139/139 matched。
- `node scripts/api_codegen.mjs --check`：PASS，生成文件为最新。
- `make check-api-generated`：PASS。
- `cd backend && go test ./...`：PASS。
- `git diff --check`：PASS。

## 范围说明

本次仅修改两个工具脚本、对应测试及本报告；未修改 API 契约、生成产物、产品文档或进度文档。
