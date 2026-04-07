# OpenASE IAM 管理控制面边界

本文档定义 ASE-92 引入的三层 IAM 管理控制面的稳态边界：

- 实例级 `/admin`
- 组织级 `/orgs/:orgId/admin`
- 项目级 `/orgs/:orgId/projects/:projectId/settings`

它是 ASE-93 到 ASE-96 的路由归属、权限边界与迁移规划真相来源。

建议配合以下文档一起使用：

- [`iam-dual-mode-contract.md`](./iam-dual-mode-contract.md)
- [`human-auth-oidc-rbac.md`](./human-auth-oidc-rbac.md)
- [`iam-admin-console-rollout.md`](./iam-admin-console-rollout.md)

## 目标

- 将实例级 IAM 操作、组织级管理、项目级设置正式拆开。
- 明确 `/admin` 只属于 `instance_admin` 权限边界。
- 允许 `org_owner` / `org_admin` 处理组织成员与组织级授权，而不获得实例级控制权。
- 让 project settings 只承担项目本地配置与项目本地授权。
- 消除 `disabled` 与 `oidc` 两种模式下的访问歧义。

## 边界规则

1. 实例级拥有全局身份系统与安全姿态。凡是会影响所有 org、所有 project 或所有人类登录的内容，都属于 `/admin`。
2. 组织级拥有“谁属于这个 org”以及“他们在这个 org 上拥有哪些组织级授权”。凡是只影响单个 org 及其下游 project 的内容，都属于 `/orgs/:orgId/admin`。
3. 项目级 settings 只拥有项目本地配置。凡是只影响单个 project 的运行时、集成、workflow 或项目级绑定的内容，都留在 project settings。
4. membership 与 role binding 是两个分层：
   - membership 回答“谁属于这个 org / project，以及当前生命周期状态是什么”
   - role binding 回答“在 instance / org / project 范围授予了哪些权限”
5. invitation 属于 org admin，因为它推进的是组织成员生命周期，而不是实例认证系统。
6. OIDC 配置属于 instance admin，因为它改变的是整套安装的身份系统，而不是某一个 org。

## Scope 归属

| 关注点 | 归属 scope | 主路由 | 说明 |
|---|---|---|---|
| `auth.mode`、OIDC 草稿配置、enable / disable 流程、bootstrap admin 策略 | Instance | `/admin/auth` | 整个 OpenASE 安装共享的全局单例 |
| 缓存用户目录、上游 identity、用户启用 / 停用 | Instance | `/admin/users` | 用户是安装级 identity，不归某个 org 单独拥有 |
| 全局 session 治理、强制撤销某个用户的全部 session | Instance | `/admin/sessions` | 用户自助 session 操作仍可保留在 `/auth/sessions`，但管理员治理是实例级 |
| 实例 auth audit、break-glass 姿态、instance 级 role binding | Instance | `/admin/security` | 这里承载最高权限治理动作 |
| org membership 生命周期、seat 状态、入组与离组 | Organization | `/orgs/:orgId/admin/members` | membership 是到 org 的身份关系 |
| org invitation 生命周期 | Organization | `/orgs/:orgId/admin/invitations` | invitation 创建的是待生效的 org membership，不是全局 identity |
| `org_owner` / `org_admin` 等 org 级 role binding | Organization | `/orgs/:orgId/admin/roles` | 授权与 membership 生命周期分离 |
| 项目名称、描述、仓库、workflow、agent、通知 | Project | `/orgs/:orgId/projects/:projectId/settings` | 纯项目本地配置 |
| 项目凭证与出站集成 | Project | `/orgs/:orgId/projects/:projectId/settings` | 这里的“Security”指项目拥有的 secret / integration，而不是全局人类认证 |
| `project_admin` 等 project 级 role binding | Project | `/orgs/:orgId/projects/:projectId/settings` | 项目访问控制应贴近被治理的项目本身 |

## 路由树

```text
/admin
  /admin/auth
  /admin/users
  /admin/sessions
  /admin/security

/orgs/:orgId/admin
  /orgs/:orgId/admin/members
  /orgs/:orgId/admin/invitations
  /orgs/:orgId/admin/roles

/orgs/:orgId/projects/:projectId/settings
  general
  repositories
  agents
  workflows
  notifications
  security        # 项目自有凭证 / 出站集成
  access          # project 级 role bindings
  archived
```

### 路由意图

- `/admin/auth`
  - 负责 auth mode 摘要、OIDC 草稿字段、provider test、enable / rollback 指引。
  - 绝不放在 org admin 或 project settings 中，因为 auth mode 是全局配置。
- `/admin/users`
  - 负责缓存人类目录、identity 检查、用户启用 / 停用、按用户 auth 诊断。
- `/admin/sessions`
  - 负责实例范围的 session 治理，例如按用户 revoke all、可疑 session 排查。
- `/admin/security`
  - 负责实例 auth audit、break-glass 姿态和 instance 级授权控制。
- `/orgs/:orgId/admin/members`
  - 负责组织 seat ledger：active、invited、suspended、removed、ownership transfer 等状态。
- `/orgs/:orgId/admin/invitations`
  - 负责该 org 的 create / resend / revoke invite 流程与历史。
- `/orgs/:orgId/admin/roles`
  - 负责 org 级授权与 org admin 升权规则。
- `/orgs/:orgId/projects/:projectId/settings`
  - 只保留项目本地配置。
  - 不能再承载 instance auth、user directory、org membership 或 org invitation。

## 为什么 OIDC 放在 `/admin`

OIDC 进入 `/admin` 是结构性结论，不是因为它“看起来像安全 UI”：

- `auth.mode` 是整个安装范围的单一开关。
- OIDC issuer、client、redirect、allowed domains、bootstrap admins 会影响所有 org 和 project。
- 在 `disabled` 模式下、甚至在尚未存在真实 org membership 之前，也必须能进入这套配置流程。
- 一旦 OIDC rollout 出问题，需要实例级 rollback 和 break-glass 恢复，而不是 org 局部兜底。

因此 `/admin/auth` 只能由 instance 级权限拥有。

## 为什么 invitation 放在 org admin

Invitation 属于 org admin，因为它处理的是组织成员生命周期：

- invite 决定的是谁可以加入某一个 org
- 同一个 invite 对一个 org 有意义，对另一个 org 可能完全无关
- 日常 onboarding 应该可以委托给 `org_owner` / `org_admin`，而不是要求实例级权限
- 接受 invite 后，首先落地的是 org membership；后续是否附带授权，是独立的 org / project role binding 话题

因此 `/orgs/:orgId/admin/invitations` 是 org 范围，不是 `/admin/invitations`。

## Membership 与 Role Binding

Membership 与授权不能混成一张表或一个 UI 概念。

### Membership

Membership 记录回答：

- 该主体是否属于这个 org？
- 当前状态是 `invited`、`active`、`suspended` 还是 `removed`？
- 谁邀请的、什么时候邀请的、invite 处于什么状态？

Membership 是 org 可见性、seat 生命周期和 onboarding/offboarding 历史的真相来源。

### Role Binding

Role binding 回答：

- 在 instance、org 或 project 范围授予了哪些权限？
- 这是 direct grant 还是 group-backed grant？
- 何时过期？

Role binding 是授权的真相来源。

### 分层规则

一次 invite 或 member-management 流程在时间线上可能会产生两个工件，但概念上仍要分离：

1. invite 创建待生效的 org-membership 关系
2. 接受后激活 membership
3. 然后管理员工作流才可以追加或调整 org / project 范围的 role binding

任何授权检查都不能仅凭 membership 行的存在就推导出持久管理员权限。

## 按认证模式区分的访问语义

### `auth.mode=disabled`

Disabled 模式并不意味着 admin 路由是 anonymous。浏览器实际运行在规范本地主体 `local_instance_admin:default` 下。

规则：

- `/admin/*` 无需 OIDC 登录即可进入，因为当前主体天然拥有等效 instance-admin 权限。
- `/orgs/:orgId/admin/*` 与 project settings 也由同一个本地主体访问。
- OpenASE 不能假装 disabled 模式里存在真实的 `org_owner` / `org_admin` 人类身份；这些路由本质上仍以本地 instance 权限评估。
- 一旦实例切换到 OIDC，这些路由就开始按真实的人类 membership 与 role binding 强制校验。

### `auth.mode=oidc`

OIDC 模式下，常规控制面路由要求真实人类会话。

规则：

- `/admin/*` 要求已登录的人类主体，并且其有效 instance 角色包含 `instance_admin`。
- `org_owner` / `org_admin` 不会自动获得 `/admin` 访问，除非同时持有 `instance_admin`。
- `/orgs/:orgId/admin/*` 要求该 org 的 org-admin 权限，或者继承得到的 instance-admin 权限。
- project settings 要求项目本地权限；在合适场景下 instance-admin 与继承的 org 权限也可以满足。
- 未认证请求会根据浏览器 / API 入口重定向到登录或返回认证挑战。

## 权限矩阵

Legend：

- `RW`：完整读写管理
- `RO`：只读可见
- `Self`：只能走该 surface 之外的自助接口
- `-`：无权访问

| Surface | Disabled `local_instance_admin` | OIDC `instance_admin` | OIDC `org_owner` | OIDC `org_admin` | OIDC `project_admin` | OIDC 其他成员 / 未登录 |
|---|---|---|---|---|---|---|
| `/admin/auth` | RW | RW | - | - | - | - |
| `/admin/users` | RW | RW | - | - | - | - |
| `/admin/sessions` | RW | RW | - | - | - | 仅可通过 `/auth/sessions` 自助 |
| `/admin/security` | RW | RW | - | - | - | - |
| `/orgs/:orgId/admin/members` | RW | RW | RW | RW | - | - |
| `/orgs/:orgId/admin/invitations` | RW | RW | RW | RW | - | - |
| `/orgs/:orgId/admin/roles` | RW | RW | RW | 受限 `RW`（不能授予或撤销 `org_admin` / `org_owner`） | - | - |
| Project settings: general / repos / workflows / notifications | RW | RW | RW | RW | RW | - |
| Project settings: security（项目凭证 / 集成） | RW | RW | RW | RW | RW | - |
| Project settings: access（project role bindings） | RW | RW | RW | RW | RW | - |

## Org Admin 能力矩阵

`instance_admin` 永远可以覆盖 org 本地限制。对于 org 角色本身，边界如下：

| Org admin 动作 | `org_owner` | `org_admin` |
|---|---|---|
| 查看 members、invites、roles | RW | RW |
| 创建 / resend / revoke invite | RW | RW |
| 激活、suspend、remove 非 owner 成员 | RW | RW |
| 管理后代项目里的 project-level admins | RW | RW |
| 授予或撤销 `org_admin` | RW | - |
| 授予或撤销 `org_owner` | RW | - |
| transfer org ownership | RW | - |
| 移除最后一个 remaining owner | - | - |

这样既能把日常 org 运维委托给 `org_admin`，又能把治理类与防锁死动作保留给 `org_owner`。

## Project Settings：保留与迁出

### 保留在 Project Settings

- project 名称、描述与元数据
- 仓库连接与出站仓库凭证
- agents、workflows、scheduled jobs、notifications
- 项目自有 SSH key 与出站集成 secret
- project 级 role binding 与 access review
- archived tickets 等项目本地维护工具

### 从 Project Settings 迁出

- auth mode 摘要与 OIDC 配置
- bootstrap admin 管理
- instance user directory 与 identity 诊断
- 全局 session 治理与按用户 session 撤销
- org memberships 与 org invitations
- org 级 role binding
- 任何 break-glass 或整机安装级安全姿态控制

### 命名规则

Project settings 可以继续保留名为 “Security” 的 section，但它只允许承载项目自有 secret、outbound credentials 和 project runtime hardening；不能再变成“人类 IAM 全塞这里”的兜底页面。

## 迁移映射

| 当前 / 过渡期 surface | 目标稳态 surface | Scope owner | 后续工单 |
|---|---|---|---|
| Settings -> Security 中的 OIDC setup | `/admin/auth` | Instance | ASE-93 |
| Settings -> Security 中的 user directory | `/admin/users` | Instance | ASE-94 |
| Settings -> Security 中的 session governance | `/admin/sessions` | Instance | ASE-94 |
| Settings -> Security 中的 instance auth audit 与 instance role controls | `/admin/security` | Instance | ASE-93 / ASE-94 |
| 共享 IAM 视图中嵌入的 org members UI | `/orgs/:orgId/admin/members` | Organization | ASE-95 / ASE-96 |
| 共享 IAM 视图中嵌入的 invite 动作 | `/orgs/:orgId/admin/invitations` | Organization | ASE-95 / ASE-96 |
| 共享 IAM 视图中嵌入的 org role binding 管理 | `/orgs/:orgId/admin/roles` | Organization | ASE-96 |
| 共享 IAM 视图中的 project-scoped bindings | Project settings access section | Project | ASE-91 umbrella 下的 follow-up |

## Non-Goals

本文档不重新定义每一个细粒度 permission key。它定义的是后续实现必须遵守的控制面边界。

如果后续工单新增 IAM surface，必须先回答三个问题：

1. 这项变更影响整个安装、单个 org，还是单个 project？
2. 它属于 membership 生命周期，还是 authorization grant 管理？
3. 它应该归属哪一个既有控制面，才能保证 `/admin`、org admin 与 project settings 三者不重叠？
