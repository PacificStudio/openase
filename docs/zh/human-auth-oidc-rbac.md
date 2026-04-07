# OpenASE 人类认证、OIDC 与 RBAC

本文档描述了 OpenASE 当前已经落地的 OIDC 配置与 RBAC 行为。

关于 `auth.mode=disabled` 与 `auth.mode=oidc` 如何长期共存的正式 IAM
契约，请参见 [`docs/zh/iam-dual-mode-contract.md`](./iam-dual-mode-contract.md)。
关于 instance admin、org admin 与 project settings 之间的稳态边界，请参见
[`docs/zh/iam-admin-boundaries.md`](./iam-admin-boundaries.md)。

## 概述

OpenASE 支持基于浏览器的人类认证，具备以下特性：

- OIDC 作为唯一的上游人类身份来源
- OpenASE 管理的 `openase_session` httpOnly Cookie 用于浏览器请求
- OpenASE 数据库支持的 RBAC 作为授权的权威来源
- 项目聊天、项目会话、提案审批和审计操作者均来自已认证的人类主体

浏览器永远不会收到上游 OIDC 的 access token 或 refresh token。

## 如何选择模式

请按真实部署场景选择认证模式：

- `auth.mode=disabled` 适合个人电脑、本地演示、临时沙箱，以及“实际上只有一个操作者”的控制面环境。
- 当实例开始被团队共享、暴露到 loopback 之外，或需要按用户审计、邀请、成员管理、会话隔离时，应优先使用 `auth.mode=oidc`。
- 如果你现在只有一个管理员，但希望提前启用浏览器登录与未来多用户能力，也可以选择 `oidc + instance_admin`。这是可选升级路径，不是必须动作。
- `instance_admin` 是 OIDC 模式内部的最高授权角色；它不能替代 disabled 模式下的本地引导主体。

实操建议：

- 当 OIDC 只会增加配置负担、却没有带来真实安全或协作收益时，继续使用 `disabled`。
- 当你需要真实用户身份、组织成员生命周期、会话清单或可审计的管理员分离时，切换到 `oidc`。
- 对于非 loopback / 公网暴露实例，`auth.mode=disabled` 只能视为临时应急姿态。

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

## Settings UI 与显式启用 OIDC

Settings -> Security 是当前 IAM setup 的过渡操作面板。`/admin`、org admin 与 project settings 的稳态拆分定义在 [`iam-admin-boundaries.md`](./iam-admin-boundaries.md)。

当 OpenASE 运行在 `auth.mode=disabled` 时，该页面会在不破坏本地管理员体验的前提下，直接提供认证设置面板：

- 页面会明确说明当前处于 disabled / 本地单用户模式
- 本地引导管理员主体会继续可用
- 你可以先保存 OIDC 草稿，而不会中断当前 disabled 使用
- 可以先测试 provider discovery，再决定是否启用
- 切换到 OIDC 必须通过显式的 `Enable OIDC` 操作完成

Disabled 模式下的设置表单支持：

- issuer URL
- client ID
- client secret
- redirect URL
- scopes
- allowed email domains
- bootstrap admin emails

当前产品行为：

1. `Save draft` 会把 OIDC 草稿持久化到配置文件，但不会改变当前运行中的 auth mode。
2. `Test configuration` 会执行 provider discovery，并返回可操作的 endpoint 诊断信息与 warning。
3. `Enable OIDC` 会再次验证 discovery，然后写入 `auth.mode=oidc` 并返回下一步指引。
4. 当前版本仍然需要重启服务，新的 configured mode 才会成为 active mode。

这种显式拆分是有意设计：保存配置绝不能悄悄打断当前 disabled 模式操作者。

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

- 设备元数据，供 session inventory 展示
- 绝对过期时间
- 空闲过期时间
- 撤销状态
- CSRF 密钥
- User-Agent 哈希
- IP 前缀绑定

OpenASE 在活跃使用时刷新空闲截止时间。在 OpenASE 数据库中禁用用户会立即使后续会话使用无效，因为授权在会话认证期间从数据库重新加载。

相关路由：

- `GET /auth/session`：返回认证模式、当前用户、CSRF Token、有效实例角色和权限。
- `GET /auth/sessions`：返回当前用户的活跃浏览器 session inventory、auth 审计时间线，以及预留的 step-up 元数据。
- `DELETE /auth/sessions/:id`：撤销单个浏览器 session，必要时也可用于当前 session。
- `POST /auth/sessions/revoke-all`：在保留当前 session 的前提下，撤销其它所有浏览器 session。
- `POST /auth/users/:userId/sessions/revoke`：允许实例级管理员强制撤销指定用户的全部 session。
- `GET /api/v1/instance/users`：返回缓存的 OIDC 用户目录，支持搜索与状态过滤。
- `GET /api/v1/instance/users/:userId`：返回单个用户的 identities、缓存 groups、活跃 session 数和最近 auth 审计。
- `POST /api/v1/instance/users/:userId/status`：执行带审计原因的用户启用 / 停用迁移，并可立即撤销现有浏览器 session。
- `POST /auth/logout`：撤销当前会话并清除会话 Cookie。
- `GET /api/v1/auth/me/permissions`：评估实例、组织或项目范围的有效角色和权限。

Auth 审计事件会记录：

- 登录成功
- 登录失败
- 注销
- session 撤销
- session 过期
- 登录后用户被禁用的强制拦截

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

Identity 同步语义：

- 规范的上游 identity 主键是 `issuer + subject`
- email、display name、avatar URL 和 group memberships 都是可变缓存字段，会在同一 `issuer + subject` 的后续登录中刷新
- 当上游 `issuer + subject` 不变时，email 变化不会创建重复本地用户
- 如果一个新的 `issuer + subject` 与已缓存邮箱冲突，登录会直接失败；当前不会自动 link、unlink 或 merge

当前用户目录边界：

- OpenASE 当前只支持一个缓存用户对应一个规范上游 identity
- 手动 link、unlink、merge 暂不支持
- 如果管理员操作或未来迁移造成一个用户拥有多个 identities，这属于当前契约之外状态，系统不会提供自动合并

Group 同步策略：

- 当前只提供 OIDC group cache
- 同步后的 groups 可以直接参与 RBAC 绑定评估
- 独立的本地 group catalog 延后到后续 IAM 工单

去配 / 离职生命周期：

- 当前已支持管理员手动停用用户
- 停用用户会立即撤销其现有浏览器 session，并保留 auth 审计历史
- 上游自动同步、webhook 回调、SCIM 等自动去配入口仍然作为后续集成点保留

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
- 人类权限采用资源/动作粒度。内置角色会展开为明确权限键，例如
  `org.read`、`project.create`、`ticket_comment.update`、`workflow.delete`、
  `harness.update`、`status.read`、`security_setting.update`、
  `notification.read`、`conversation.create`。
- list / index 类 API 在返回组织、项目、仓库等人类可见集合前，会先按当前
  principal 的 effective visibility 过滤。

人类权限与 agent scopes 有关联，但不是同一套键：

- 人类权限用于浏览器认证的人类控制面操作。
- Agent scopes 用于签发给运行时 token 的能力，例如 `projects.update`、
  `tickets.update.self`。
- 名称相似不代表可互换；人类权限不会直接 mint agent scope，agent scope 也
  不会满足人类 RBAC 判定。

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

当 `auth.mode=disabled` 时，持久 Project Conversation 会切换到服务端定义的本地稳定主体：`local-user:default`。

- 这样可以让会话归属跨前端 dev server 端口与浏览器本地存储重置保持稳定
- `localStorage` 仅保留标签布局和草稿等 UI 恢复职责，不再作为持久会话 owner 的真相来源
- 该模式明确是本地单用户 / 共享实例 fallback，不是多用户隔离模型

审计语义：

- 正常的人类写操作记录为 `user:<user-id>`
- 已批准的会话操作归属为 `user:<user-id> via project-conversation:<conversation-id>`

这保留了人类审批者和项目会话运行时主体之间的区别。

## 设置与诊断

控制面板的 Settings 视图暴露了人类认证状态，包括：

- 当前认证模式
- 配置文件中的 configured auth mode
- Issuer URL
- bootstrap admin 摘要
- disabled 模式下的本地引导管理员说明
- 已保存的 OIDC 草稿字段，以及显式的 save / test / enable 动作
- 当前已认证用户
- session inventory，包括当前设备识别与撤销动作
- 浏览器访问相关的 auth 审计时间线
- 稳定的 Project Conversation owner 语义（OIDC 下为 `user:<user-id>`，关闭认证时为 `local-user:default`）
- 有效角色和权限
- 人类权限与可 mint agent scopes 的区别
- 实例 / 组织 / 项目角色绑定管理
- 组织成员与邀请管理
- 指向迁移、回退与 rollout 计划的文档入口

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
