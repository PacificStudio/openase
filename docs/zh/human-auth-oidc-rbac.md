# OpenASE 人类认证、OIDC 与 RBAC

本文档描述了 OpenASE 中控制面板的人类认证模型。

## 概述

OpenASE 支持基于浏览器的人类认证，具备以下特性：

- OIDC 作为唯一的上游人类身份来源
- OpenASE 管理的 `openase_session` httpOnly Cookie 用于浏览器请求
- OpenASE 数据库支持的 RBAC 作为授权的权威来源
- 项目聊天、项目会话、提案审批和审计操作者均来自已认证的人类主体

浏览器永远不会收到上游 OIDC 的 access token 或 refresh token。

## 配置

通过设置 `auth.mode=oidc` 并提供以下 OIDC 配置来启用人类认证：

```yaml
auth:
  mode: oidc
  oidc:
    issuer_url: https://idp.example.com/realms/openase
    client_id: openase
    client_secret: ${OIDC_CLIENT_SECRET}
    redirect_url: http://127.0.0.1:19836/api/v1/auth/oidc/callback
    scopes: ["openid", "profile", "email", "groups"]
    email_claim: email
    name_claim: name
    username_claim: preferred_username
    groups_claim: groups
    allowed_email_domains: []
    bootstrap_admin_emails: []
    session_ttl: 8h
    session_idle_ttl: 30m
```

字段说明：

- `issuer_url`：OIDC 发现（Discovery）基础 URL。
- `client_id` / `client_secret`：用于授权码流程的 OAuth 客户端凭证。
- `redirect_url`：必须与 IdP 的重定向注册和 OpenASE 服务 URL 匹配。
- `scopes`：默认为 `openid`、`profile`、`email`、`groups`。
- `allowed_email_domains`：可选的邮件域白名单，在 ID Token 验证后应用。
- `bootstrap_admin_emails`：可选的邮件白名单，首次登录时自动获得 `instance_admin` 角色绑定。
- `session_ttl`：浏览器会话的绝对生命周期。
- `session_idle_ttl`：滑动空闲超时，不得超过 `session_ttl`。

OpenASE 也支持通过正常配置加载器使用等价的 `OPENASE_AUTH_*` 环境变量。

## 浏览器流程

OpenASE 使用授权码 + PKCE：

1. 浏览器访问 `GET /auth/oidc/start`。
2. OpenASE 将签名的登录流程状态存储在一个短期 Cookie 中。
3. IdP 重定向回 `GET /api/v1/auth/oidc/callback`。
4. OpenASE 执行发现、服务端交换授权码、验证 ID Token、同步本地用户缓存并创建浏览器会话。
5. 浏览器仅收到 OpenASE 的 `openase_session` Cookie，登录后使用同源请求。

匿名访问被有意限制在 setup、健康检查和认证入口。在 `auth.mode=oidc` 下，常规的 `/api/v1/...` 浏览器控制面板路由需要有效的人类会话。

## 会话模型

浏览器会话存储在 `browser_sessions` 表中，包含：

- 绝对过期时间
- 空闲过期时间
- 撤销状态
- CSRF 密钥
- User-Agent 哈希
- IP 前缀绑定

OpenASE 在活跃使用时刷新空闲截止时间。在 OpenASE 数据库中禁用用户会立即使后续会话使用无效，因为授权在会话认证期间从数据库重新加载。

相关路由：

- `GET /auth/session`：返回认证模式、当前用户、CSRF Token、有效实例角色和权限。
- `POST /auth/logout`：撤销当前会话并清除会话 Cookie。
- `GET /api/v1/auth/me/permissions`：评估实例、组织或项目范围的有效角色和权限。

## CSRF 保护

浏览器的变更请求使用同源 Cookie 会话，并通过以下方式保护：

- Same-Site Cookie
- Origin 或 Referer 验证
- 受保护写操作上的每会话 CSRF Token 检查

前端代码应从 `GET /auth/session` 获取 CSRF Token，并通过正常的 API 客户端在同源变更请求中发送。

## 用户缓存与身份同步

成功的 OIDC 登录会 upsert 以下表：

- `users`
- `user_identities`
- `user_group_memberships`
- `browser_sessions`

OpenASE 将本地授权缓存和组成员关系存储在自己的数据库中，以便：

- 后续登录时可以刷新个人信息变更
- 组可以支撑 OpenASE 角色绑定
- 禁用的用户可以被 OpenASE 阻止，无论上游浏览器状态是否过期

## RBAC 模型

角色绑定存储在数据库中，按范围评估：

- `instance`（实例）
- `organization`（组织）
- `project`（项目）

主体可以是：

- `user`（用户）
- `group`（组）

当前内置角色：

- `instance_admin`
- `org_owner`
- `org_admin`
- `org_member`
- `project_admin`
- `project_operator`
- `project_reviewer`
- `project_member`
- `project_viewer`

RBAC 评估规则：

- OpenASE 数据库绑定是权威来源。
- OIDC Claims 不直接授予角色。
- 直接用户绑定和组绑定取并集。
- 组织绑定向下继承到子项目范围。
- 权限默认拒绝。

角色绑定可通过以下路由管理：

- `GET/POST/DELETE /api/v1/organizations/:orgId/role-bindings`
- `GET/POST/DELETE /api/v1/projects/:projectId/role-bindings`

## 引导管理员

当 `bootstrap_admin_emails` 包含正在认证的用户邮箱时，OpenASE 确保该用户在登录时拥有 `instance_admin` 角色绑定。

使用此功能避免在首次启用 OIDC 的部署中将自己锁定：

1. 设置 `auth.mode=oidc`。
2. 将至少一个可信管理员邮箱添加到 `bootstrap_admin_emails`。
3. 部署 OpenASE。
4. 使用该账号完成首次浏览器登录。
5. 使用 Settings 或 RBAC API 创建你想要的稳态组织和项目绑定。

引导完成后，你可以缩小或清空引导列表。

## 聊天、会话与审计操作者

AI 会话归属始终派生自服务端定义的主体：

- 在 `auth.mode=oidc` 下，项目聊天、项目会话以及其他浏览器驱动的 AI 流程使用已认证的人类主体
- 在 `auth.mode=disabled` 下，服务端签发并复用 `openase_ai_principal` 浏览器 Cookie，其值是稳定的 `browser-session:<uuid>` 主体

浏览器本地随机 ID 和 `X-OpenASE-Chat-User` 请求头不再是权威 owner 输入。

审计语义：

- 正常的人类写操作记录为 `user:<user-id>`
- 已批准的会话操作归属为 `user:<user-id> via project-conversation:<conversation-id>`

这保留了人类审批者和项目会话运行时主体之间的区别。

## 设置与诊断

控制面板的 Settings 视图暴露了人类认证状态，包括：

- 当前认证模式
- Issuer URL
- 当前已认证用户
- 有效角色和权限
- 组织/项目角色绑定管理

`GET /auth/session` 和 `GET /api/v1/auth/me/permissions` 是用于脚本和诊断的 API 等价物。

## 故障排查

常见检查：

- `auth.mode=oidc` 但 `/login` 立即循环回来：
  验证 `issuer_url`、`client_id`、`client_secret` 和 `redirect_url`，确认 IdP 的重定向注册与绝对回调 URL 匹配。
- 在 IdP 登录成功但 OpenASE 拒绝访问：
  检查 `allowed_email_domains`、用户禁用状态，以及用户是否对目标范围有任何 OpenASE 角色绑定。
- 登录成功但项目页面返回 `403 AUTHORIZATION_DENIED`：
  检查 `GET /api/v1/auth/me/permissions?project_id=<project-id>` 确认有效的组织/项目绑定。
- 注销似乎失败：
  确保浏览器请求包含来自 `GET /auth/session` 的 CSRF Token 且源自同一站点。
