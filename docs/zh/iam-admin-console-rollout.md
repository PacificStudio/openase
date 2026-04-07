# OpenASE IAM 管理控制台 Rollout 清单

本文档用于指导在 `disabled` 与 `oidc` 两种认证模式下，安全落地完整 IAM 管理控制台。

当前过渡期里，部分控件仍可能暂时集中在 Settings -> Security 中；但后续实现必须遵循 [`iam-admin-boundaries.md`](./iam-admin-boundaries.md) 定义的目标控制面归属。

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

Settings -> Security 现在提供：

- active auth mode 与 configured auth mode
- issuer 与 bootstrap admin 摘要
- disabled 模式说明与公网暴露风险提示
- 包含 save / test / enable 动作的 OIDC 草稿表单
- instance / organization / project 范围的 effective access
- role bindings
- session inventory 与 audit 摘要
- user directory
- organization members 与 invitations
- rollout / rollback 文档入口

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

针对个人 / 本地部署，确认 Security 页面满足以下要求：

- 明确说明当前运行在 disabled 模式
- 明确说明当前操作者已经拥有本地最高权限
- 不强迫用户配置 OIDC
- 当实例不再是 loopback 绑定时给出高风险警告
- 允许在不影响当前使用的前提下保存和测试 OIDC

这能保证 disabled 模式仍然是正式产品路径，而不是“迁移前临时页面”。

## OIDC 草稿与启用清单

1. 在仍处于 `auth.mode=disabled` 时打开 Settings -> Security。
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

- issuer 与模式摘要是否正确
- 当前用户是否为预期的真实 OIDC 用户
- instance / organization / project effective access 面板是否有内容
- role bindings 是否可以查看和更新
- session inventory 是否能看到当前浏览器会话
- user directory 是否列出同步后的用户
- organization members 与 invitations 是否可用
- audit / diagnostics 摘要是否反映新的登录路径

## 从基础 OIDC+RBAC 迁移到完整控制台的检查项

适用于已在使用 OIDC 的实例：

1. 确认现有 OIDC 配置仍与 provider 匹配。
2. 打开 Security 页面，检查保存下来的 issuer、scopes 与 redirect URL。
3. 确认 bootstrap admin 邮箱仍适用于 break-glass 恢复。
4. 检查实例、组织、项目范围的管理员与操作员绑定。
5. 验证 session inventory 与 user directory 是否符合预期用户集合。
6. 验证 organization memberships 与 invitations。
7. 在一次全新登录后复查 audit / diagnostics 摘要。
8. 当稳态 RBAC 已确认后，收窄或清空 bootstrap admin 邮箱列表。

## 回退方案

回退必须明确且快速：

1. 如果 rollout 期间 OIDC 登录或授权失败，将 `auth.mode` 改回 `disabled`。
2. 如果部署模型要求重启，则重启服务。
3. 确认 Security 页面重新显示 disabled 模式和本地管理员主体。
4. 保留已保存的 OIDC 草稿，便于修复问题后重试。
5. 在再次启用前，先记录并定位失败原因。

回退流程绝不能依赖伪造本地 OIDC 用户。

## 建议验证矩阵

在更广范围 rollout 前，至少验证两条产品路径：

### `disabled`

- Security 页面渲染 auth setup 面板
- 保存 OIDC 草稿不会改变 active mode
- 测试 OIDC 能返回 discovery diagnostics
- 在合适条件下出现公网风险提示

### `oidc`

- 首次 bootstrap 登录能够授予 `instance_admin`
- effective access 面板与预期绑定一致
- instance / org / project 三个范围的 role-binding CRUD 正常
- session inventory 与 revoke 动作可用
- user directory 与 membership diagnostics 正常加载
- organization invites 可发送和管理

## 运维说明

- `Save draft` 是有意设计为非破坏性动作。
- `Enable OIDC` 只有在 discovery 校验成功后才会改变 configured mode。
- 当前实现可能仍需要重启服务，新的 configured mode 才会变成 active mode。
- Disabled 模式是长期受支持的正式运行模式，而不是只用于迁移失败时的兜底。
