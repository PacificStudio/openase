# OpenASE IAM 双模式认证契约与领域模型

本文档定义 ASE-77 引入的长期 IAM 契约，用来明确 OpenASE 如何在扩展 OIDC 多用户能力的同时，长期保留现有的 `auth.mode=disabled` 单用户产品模式。

## 目标

- 将 `auth.mode=disabled` 保留为正式、长期存在的个人 / 单用户部署模式。
- 将 `auth.mode=oidc` 定义为依赖真实 `User` / `UserIdentity` 的多用户浏览器认证模式。
- 将 IAM 明确分层为 Identity、Membership、Authorization、Session、Audit。
- 在边界处用类型和解析器替换弱字符串拼装。

## 核心结论

1. `disabled` 不是“弱化版 OIDC”，它不依赖 OIDC，也不要求浏览器登录。
2. `disabled` 使用服务端定义的本地引导主体 `local_instance_admin:default`；它的 effective permissions 等同于 `instance_admin`，但不是一个真实 OIDC 用户。
3. `oidc` 使用真实的人类主体和浏览器会话；`instance_admin` 是 OIDC 模式里的最高实例角色，而不是 disabled 模式的替代品。
4. 从 `disabled` 切换到 `oidc` 必须是显式的管理流程：配置草稿、测试、校验、启用；失败时必须保留或恢复 disabled 体验。

## 认证模式契约

| 关注点 | `auth.mode=disabled` | `auth.mode=oidc` |
|---|---|---|
| 产品定位 | 个人 / 单用户部署 | 团队 / 多用户部署 |
| OIDC 依赖 | 无 | 必需 |
| 浏览器登录 | 永不要求 | 常规控制面路由要求登录 |
| 主要交互主体 | `local_instance_admin:default` | `user:<user-id>` |
| 实例级管理权限 | 等效 `instance_admin` | 由实例 / 组织 / 项目绑定决定 |
| 是否需要真实 `User` / `UserIdentity` | 不需要，且不能伪造 | 需要 |
| 多用户隔离 | 不支持 | 支持 |
| 持久会话 owner | `local_instance_admin:default` | `user:<user-id>` |
| 交互式审计 actor | `local_instance_admin:default` | `user:<user-id>` |
| 会话代理审计 actor | `local_instance_admin:default via project-conversation:<conversation-id>` | `user:<user-id> via project-conversation:<conversation-id>` |

### Disabled 模式主体语义

Disabled 模式必须在保持“无需登录”的同时，继续提供个人部署所需的全部管理能力。

- 规范主体：`SubjectRef{Kind: local_instance_admin, Key: default}`。
- 人类语义：表示“当前 OpenASE 实例的本地操作者”，不是上游身份源中的用户。
- 授权语义：其有效权限等同于 `instance_admin`，因此 Settings、RBAC 管理等能力可直接使用。
- 建模规则：不能要求它在 `users` 或 `user_identities` 中拥有一条伪造记录。
- 会话规则：浏览器会话可以继续存在，用于 CSRF、设备识别、撤销等；但会话应指向本地主体，而不是捏造一个用户。

### OIDC 模式主体语义

OIDC 模式是多用户路径。

- 规范交互主体：`SubjectRef{Kind: user, Key: <user-id>}`。
- 浏览器会话 owner、审计 actor、conversation ownership 都来自已认证用户。
- `instance_admin` 是最高实例角色，可通过 role binding 或 bootstrap admin 逻辑授予。
- `instance_admin` 不能替代 disabled，因为它仍然依赖真实 OIDC 用户、OIDC 配置和浏览器登录。

## 本地引导管理员与旧别名

Disabled 模式的规范主体是 `local_instance_admin:default`。

迁移期间，OpenASE 应继续把旧的 ownership 字符串 `local-user:default` 当作兼容别名读取。后续工单再逐步把存量 conversation / audit 数据标准化到 typed subject 形式。

## Settings、Session、Audit 与 Conversation Ownership

### Settings Access

- `disabled` 下，Settings UI 以及受保护写操作都以 `local_instance_admin:default` 执行。
- `oidc` 下，Settings UI 以当前登录用户执行，授权取决于该用户的有效绑定。

### Session 模型

- 复用 `browser_sessions`，但领域模型需要扩展为“绑定 typed `SubjectRef` + 设备信息”。
- `disabled` 下，会话标识的是浏览器 / 设备，而不是目录中的真实用户。
- `oidc` 下，会话绑定真实用户，并支持按设备撤销。

### Audit 模型

- 审计记录应存储 typed actor 和可选 delegated conversation actor。
- Disabled 模式交互审计统一使用 `local_instance_admin:default`。
- OIDC 模式交互审计统一使用 `user:<user-id>`。

### Conversation Ownership

- 持久 Project Conversation 的 owner 必须始终由服务端推导。
- Disabled 模式使用稳定本地主体，浏览器本地随机 ID 不再决定 owner。
- OIDC 模式使用当前登录用户主体；Project Conversation 运行时仍是 delegated runtime，而不是主要人类 actor。

## 从 Disabled 到 OIDC 的启用流程

模式切换是一个显式配置流程，而不是登录副作用。

### 存储契约

- 运行时生效模式存放在配置提供者中的 `auth.mode`。
- 非敏感 OIDC 草稿字段存放在同一配置提供者中的 `auth.oidc.*`。
- `auth.oidc.client_secret` 等敏感值存放在 secret provider。对于当前文件型本地部署，这意味着 `~/.openase/.env` 或等价环境变量来源，而不是通过普通 Settings 读取接口回显。
- 未来即使 secret store 后端变化，这个契约也不变：敏感值始终与非敏感配置文档分离。

### 管理流程

1. 从 `auth.mode=disabled` 开始，本地引导管理员主体处于激活状态。
2. 在 Settings 中保存 OIDC 草稿配置。保存草稿不会改变当前认证模式。
3. 执行 `Test OIDC`。
   - 校验字段完整性。
   - 拉取 discovery metadata 与 JWKS。
   - 检查 redirect URL 是否与当前 OpenASE base URL 语义匹配。
   - 不创建 user、identity、session 或 role binding。
4. 执行 `Validate OIDC`。
   - 用草稿配置初始化 OIDC client。
   - 验证当前进程确实可以初始化 OIDC provider。
   - 持久化与当前配置版本绑定的校验结果。
   - 校验过程中仍保持 `auth.mode=disabled` 生效。
5. 点击 `Enable OIDC`。
   - 必须要求当前配置版本已有成功校验结果。
   - 原子化持久化 `auth.mode=oidc`，并 reload / restart auth runtime。
   - 如果 reload 失败，继续保持 disabled 运行态并向界面返回错误。
6. 如果后续上线失败，可显式把 `auth.mode` 改回 `disabled`。OIDC 草稿配置可以保留，供后续重试。

### 失败与回退规则

- Test / Validate 失败都不会改变 active mode。
- Enable 失败时，系统保持最后一个已知正常模式。
- Disabled 模式必须永久存在，作为回退目标。
- 任何迁移步骤都不能要求先创建一个“本地 OIDC 用户”才能重新获得管理权限。

## 多用户边界

- `disabled` 只提供单用户语义。
- 在 `disabled` 下，多个人共享同一浏览器或机器，本质上仍然作为同一个本地引导管理员主体操作。
- 多用户 identity、membership、invitation、按用户隔离的 session 都属于 OIDC 模式能力。
- 个人部署当然可以选择启用 OIDC 再给自己 `instance_admin`，但这是可选部署方案，不是 disabled 模式的产品替代物。

## IAM 分层模型

| 分层 | 职责 | 可复用模型 | 新增 / 变化模型 |
|---|---|---|---|
| Identity | 规范的人与上游身份 | `users`, `user_identities` | disabled 下不创建 fake user；为 identity 引用增加 typed parse boundary |
| Membership | 人与组织 / 项目的关系 | 迁移期可部分借助 `role_bindings` | `organization_memberships`, `organization_invitations`, membership status |
| Authorization | 角色、权限、继承、委托校验 | `role_bindings` | 通过 scope-specific role types 替代弱字符串混用 |
| Session | 浏览器 / 设备状态、CSRF、撤销 | `browser_sessions` | 增加 typed subject ref、device kind、disabled 下可为空的真实用户绑定 |
| Audit | 谁通过哪个运行时做了什么 | 现有 activity event 可部分复用 | auth audit event，包含 typed actor / delegated actor |

## 领域类型与 Parse 边界

ASE-77 在 `internal/domain/iam/contracts.go` 中引入第一版 typed IAM 草案。

关键类型：

- `AuthMode` / `AuthModeContract`
- `SubjectKind` / `SubjectRef`
- `InstanceRole` / `OrganizationRole` / `ProjectRole`
- `MembershipStatus` / `InvitationStatus`
- `SessionDevice`

Parse 规则：

- 原始配置值先解析为 `AuthMode`
- 数据库 role 行先解析为 scope-specific role type
- owner / audit actor 字符串先解析为 `SubjectRef`
- session device 标签先解析为 `SessionDevice`
- 业务逻辑只消费已解析的领域类型，不继续在业务路径里拼接字符串判断

## 数据模型复用与新增

### 复用

- `users`：只保留给 OIDC 模式中的真实人类。
- `user_identities`：继续承载上游 OIDC identity 关联。
- `browser_sessions`：继续作为 session 表，但需要演进为引用 typed subject 与 device metadata。
- `role_bindings`：继续作为授权真相来源，后续工单再强化 scope correctness 和 typed role parsing。

### 新增

- `organization_memberships`
- `organization_invitations`
- auth audit events 表或等价事件载荷
- 如果现有 settings store 无法干净表达，还需要 OIDC 配置版本 / 校验结果元数据

## 状态模型

### Membership

- `active` -> `suspended`
- `active` -> `revoked`
- `active` -> `left`
- `suspended` -> `active`
- `suspended` -> `revoked`
- `revoked` 与 `left` 为终态

### Invitation

- `pending` -> `accepted`
- `pending` -> `expired`
- `pending` -> `revoked`
- 终态保持终态

### Session

- 浏览器 / 设备 session 与 principal identity 分离
- disabled 下撤销某个 session，不会改变稳定本地主体
- OIDC 下撤销 session，只影响该用户 / 设备会话，不改变 role binding 本身

## Rollout、Backfill 与 Feature Flag

### Rollout 策略

1. 先落地 typed IAM 契约与设计文档。
2. 为 legacy owner 字符串（例如 `local-user:default`）提供读取兼容映射到 `local_instance_admin:default`。
3. 在 feature flag 下扩展 session / audit schema，再逐步切换 API payload 到 typed subject ref。
4. 引入 organization membership / invitation 时，不改变 disabled 模式语义。

### Backfill

- 为现有 OIDC session 回填 `subject_kind=user` 与 `subject_key=<user-id>`。
- 为 conversation ownership / audit 记录提供 legacy disabled principal 到 `local_instance_admin:default` 的兼容映射。
- 在所有 reader 都理解 typed subject ref 之前，保留旧字符串读取兼容。

### 建议 Feature Flags

- `iam_subject_refs`
- `iam_membership_tables`
- `iam_auth_audit`
- `iam_oidc_settings_enablement`

这些 flag 只用于存储与 API rollout，不允许拿来弱化或删除 disabled 模式本身。

## 后续工单对齐

- ASE-78 使用这里定义的 scoped role model 与 authorization integrity 规则。
- ASE-79 基于这里的 scoped role types 扩展权限覆盖。
- ASE-80 使用这里的 session device 与 audit actor 契约。
- ASE-81 使用这里的 identity 与 deprovision 模型。
- ASE-82 使用这里的 membership 与 invitation 模型。
- ASE-83 使用这里的 settings、diagnostics、validation 与 rollout 流程。
