# wxapp 独立登录页设计

## 目标

新增一个轻量登录页，供“我的”页和后续所有需要登录的页面统一跳转使用，避免每个页面重复实现微信登录逻辑。

本次只实现微信登录和登录后回跳，不扩展账号体系。

## 范围

本次覆盖：

- 新增 `pages/login/index.vue`。
- 在 `pages.json` 注册登录页。
- 新增通用登录态工具，用于判断是否已登录、构造登录页 URL、跳转登录页。
- “我的”页未登录按钮改为跳转登录页。
- 登录成功后保存 `token` 和 `user.id`，并按 `redirect` 参数回到原页面。

本次不覆盖：

- 密码登录、手机号登录、注册页。
- 退出登录。
- 商家绑定、商家身份选择。
- 登录拦截全局中间件。
- 后端接口协议调整。

## 页面行为

登录页路径为 `pages/login/index`，导航标题为“登录”。

页面展示：

- 平台名“衣货通”。
- 登录说明：“登录后同步收藏、需求、消息和发布记录”。
- 主按钮“微信登录”。

登录页接收可选参数：

- `redirect`：登录成功后跳回的页面路径，需要使用 `encodeURIComponent` 编码。

登录成功后：

- 保存后端返回的 `token`。
- 保存后端返回的 `user.id`。
- 如果存在 `redirect`，优先回跳到该页面。
- 如果没有 `redirect`，默认跳到 `/pages/my/index`。

回跳规则：

- tabBar 页面使用 `uni.switchTab`。
- 非 tabBar 页面使用 `uni.redirectTo`。

## 通用登录工具

新增 `wxapp/common/auth.js`，只做前端登录态和跳转封装：

- `isLoggedIn()`：读取本地 session token。
- `getCurrentPageUrl()`：从页面栈拼出当前页面路径和 query。
- `buildLoginUrl(redirect)`：生成登录页 URL。
- `requireLogin(options)`：已登录返回 `true`；未登录跳转登录页并返回 `false`。

后续页面需要登录时，只调用 `requireLogin({ redirect })`，不直接复制登录页逻辑。

## 我的页接入

“我的”页未登录状态继续展示现有游客卡片和能力说明。

未登录时点击“微信登录”：

- 跳转到 `/pages/login/index?redirect=%2Fpages%2Fmy%2Findex`。
- 不在“我的”页内直接调用微信登录接口。

“我的”页常用入口继续使用 `requireLogin()`，未登录时统一跳转登录页。

## 验证标准

满足以下条件即可验收：

- `pages.json` 注册 `pages/login/index`。
- 登录页文件存在，包含微信登录、保存 token、保存用户 ID、redirect 回跳逻辑。
- 通用登录工具包含 `requireLogin`、`buildLoginUrl`、`getCurrentPageUrl`。
- “我的”页未登录按钮跳转登录页，不再直接调用微信登录接口。
- 现有 `wxapp` 页面和流程验证通过。
