# OpenASE IAM 管理控制台 Rollout 清单

本文档用于指导在 `disabled` 与 `oidc` 两种认证模式下，安全落地完整 IAM 管理控制台。

当前的人类 IAM 已经拆分到 `/admin`、org admin 与 Project Settings -> Access；Project Settings -> Security 只保留项目级安全控制与迁移提示。后续实现必须遵循 [`iam-admin-boundaries.md`](./iam-admin-boundaries.md) 定义的目标控制面归属。

建议配合以下文档一起使用：

- [`human-auth-oidc-rbac.md`](./human-auth-oidc-rbac.md)
- [`iam-dual-mode-contract.md`](./iam-dual-mode-contract.md)
- [`iam-admin-boundaries.md`](./iam-admin-boundaries.md)

## Rollout 目标

- 保持 `auth.mode=disabled` 作为个人 / 本地单用户部署的简单路径。
- 为 OIDC 部署提供完整、可运维的 IAM 管理界面。
- 让 OIDC 切换流程显式、可测试、可回退。
- 在更大范围暴露前，先验证 diagnostics、RBAC、sessions、user directory、memberships 与 invitations。

## 管理控制台覆盖面

当前拆分如下：

- `/admin/auth`：auth mode 摘要、OIDC 草稿、验证诊断、显式启用与回滚指引
- `/admin`：实例级 session governance、用户目录诊断与 break-glass 恢复说明
- `/orgs/:orgId/admin/*`：组织成员、邀请与组织级 RBAC
- Project Settings -> `#access`：项目级 role binding 与有效 project access
- Project Settings -> `#security`：项目自有凭证、webhook 边界与运行时 token posture
- 各 surface 上的迁移提示与操作文档，确保管理员在兼容期内仍能找到旧入口

## 部署选择矩阵

### 继续使用 `auth.mode=disabled` 的场景

- 实例只在本地使用，或仅绑定 loopback
- 实际上只有一个操作者
- 不需要独立的人类身份、邀请或会话隔离
- 你希望保持最低配置复杂度，并接受本地管理员语义

### 切换到 `auth.mode=oidc` 的场景

- 实例会被多个人共享
- 控制面会暴露到 loopback 之外
- 需要按用户记录审计归属
- 需要 organization members、invitation 或按用户隔离的 session 管理
- 需要把 `instance_admin` 等角色授予真实用户，而不是 disabled 模式下的本地主体

## Rollout 前检查

在切换模式前先确认：

1. 明确目标 base URL、redirect URL 和 TLS 方案。
2. 在 OIDC provider 中注册 OpenASE callback URL。
3. 确定第一批 bootstrap admin 邮箱。
4. 确定初始 organization / project role binding 方案。
5. 确认回退路径：运维人员必须知道如何把 `auth.mode` 改回 `disabled`。

## Disabled 模式验证

针对个人 / 本地部署，确认 `/admin/auth` 满足以下要求：

- 明确说明当前运行在 disabled 模式
- 明确说明当前操作者已经拥有本地最高权限
- 不强迫用户配置 OIDC
- 当实例不再是 loopback 绑定时给出高风险警告
- 允许在不影响当前使用的前提下保存和测试 OIDC

同时验证 Project Settings -> Security 与 -> Access 会把操作者引导到 `/admin` 与 org admin，而不是继续假装旧的共享 IAM 视图仍然拥有这些控制权。

## OIDC 草稿与启用清单

1. 在仍处于 `auth.mode=disabled` 时打开 `/admin/auth`。
2. 填写：
   - issuer URL
   - client ID
   - client secret
   - redirect URL
   - scopes
   - allowed email domains
   - bootstrap admin emails
3. 点击 `Save draft`。
   - 预期结果：草稿被保存，但 active auth mode 仍是 `disabled`。
4. 点击 `Test configuration`。
   - 预期结果：provider discovery 成功，并返回 issuer / authorization / token endpoint 诊断信息。
5. 点击 `Enable OIDC`。
   - 预期结果：configured mode 切换为 `oidc`，界面返回下一步指引。
6. 如果界面提示需要重启，则重启服务。
7. 使用 bootstrap admin 账号完成第一次 OIDC 登录。
8. 确认该 bootstrap 用户获得 `instance_admin`。

## 启用后的验证

首次 OIDC 登录成功后，使用产品 UI 检查：

- `/admin/auth` 展示了正确的 issuer、auth mode 摘要与 bootstrap 指引
- `/admin` 展示了预期的当前 session、恢复说明与用户目录可见性
- `/orgs/:orgId/admin/*` 暴露了预期的成员、邀请与组织级角色
- Project Settings -> `#access` 展示了预期的项目有效权限与项目级绑定
- Project Settings -> `#security` 仍然只展示项目级安全配置，而没有把实例级 auth 控件拉回来
- audit / diagnostics 摘要能反映新的登录路径

## 从基础 OIDC+RBAC 迁移到完整控制台的检查项

适用于已在使用 OIDC 的实例：

1. 在 `/admin/auth` 确认现有 OIDC 配置仍然与 provider 匹配。
2. 验证保存的 issuer、scopes、redirect URL 与 bootstrap admin emails。
3. 在 `/admin` 检查 session 与用户目录诊断。
4. 在 `/orgs/:orgId/admin/*` 检查组织成员、邀请与组织级角色。
5. 在 Project Settings -> `#access` 检查项目绑定与有效 project access。
6. 验证 Project Settings -> `#security` 仍然只暴露项目自有安全配置。
7. 在一次新的登录后检查 audit / diagnostics 摘要。
8. 在稳态 RBAC 确认完成后，移除或收窄 bootstrap admin emails。

## 回退方案

回退必须明确且快速：

1. 如果 rollout 期间 OIDC 登录或授权失败，将 `auth.mode` 改回 `disabled`。
2. 如果部署模型要求重启，则重启服务。
3. 确认 `/admin/auth` 再次显示 disabled 模式和本地管理员主体，同时 Project Settings -> Security / -> Access 仍然把操作者指回新的控制面。
4. 保留已保存的 OIDC 草稿，便于修复问题后重试。
5. 在再次启用前，先记录并定位失败原因。

回退流程绝不能依赖伪造本地 OIDC 用户。

## 建议验证矩阵

在更广范围 rollout 前，至少验证两条产品路径：

### `disabled`

- `/admin/auth` 渲染 auth setup 面板
- 保存 OIDC 草稿不会改变 active mode
- 测试 OIDC 能返回 discovery diagnostics
- 在合适条件下出现公网风险提示
- Project Settings -> Security 与 -> Access 都展示迁移指引

### `oidc`

- 首次 bootstrap 登录能够授予 `instance_admin`
- `/admin` 与 `/admin/auth` 展示正确的实例态势
- org admin 与 Project Settings -> Access 上的 role-binding CRUD 正常
- session inventory 与 revoke 动作可用
- user directory 与 membership diagnostics 正常加载
- organization invites 可发送和管理

## 运维说明

- `Save draft` 是有意设计为非破坏性动作。
- `Enable OIDC` 只有在 discovery 校验成功后才会改变 configured mode。
- 当前实现可能仍需要重启服务，新的 configured mode 才会变成 active mode。
- Disabled 模式是长期受支持的正式运行模式，而不是只用于迁移失败时的兜底。
