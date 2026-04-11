# Settings（设置）

## 这是什么？

Settings 是项目的**控制中心**，管理项目的各项基础配置。大部分设置只需要在项目初始化时配置一次，之后按需调整。

## 配置模块

### 通用设置（General）

- 项目名称和描述
- 默认 Agent Provider 选择

这是最基础的配置，创建项目后首先设置。

### 工单状态（Statuses）

管理工单的状态列表。这些状态会直接影响：
- [Tickets](./tickets.md) 看板视图的列
- [Workflows](./workflows.md) 的触发条件

每个状态有一个**阶段（Stage）**属性：

| 阶段 | 说明 | 示例 |
|------|------|------|
| `todo` | 待办 | 待处理、已规划 |
| `in-progress` | 进行中 | 开发中、编码中 |
| `reviewing` | 审核中 | Code Review、待审批 |
| `testing` | 测试中 | 测试中、QA 验证 |
| `done` | 已完成 | 已完成、已发布 |

**操作**：
- 创建自定义状态
- 调整状态显示顺序（拖拽排序）
- 编辑和删除状态

> **重要**：状态的阶段属性会影响工作流的自动领取逻辑，务必正确设置。

### 仓库（Repositories）

将代码仓库关联到项目，让 Agent 能够访问代码：

- 关联 GitHub / GitLab 仓库，或使用 `file://...` 注册本机 / 自托管 Git source
- 配置出站凭证（Outbound Credentials）
- Agent 可以克隆代码、创建分支、提交 PR

`file://` 仓库 URL 会在实际执行 clone / fetch 的机器上解释；也就是说，一台机器本地可访问的路径，只有对同样能访问该文件系统路径的 runtime 才可用。

配置好后建议点击"测试连接"，确保 Agent 可以正常访问代码。

### Agent 配置

- 配置默认的 AI Provider
- 管理 Agent 可用性

### 通知（Notifications）

设置项目事件的通知规则：

- 配置通知渠道（Webhook、Telegram、Slack 和 WeCom）
- 选择哪些事件触发通知（工单完成、执行失败等）

### 安全（Security）

管理项目的安全凭证：

- disabled 与 OIDC 模式下的人类认证 / IAM 总览
- disabled 模式中的 OIDC 草稿保存、discovery 测试与显式启用流程
- effective access、role bindings、session inventory、user directory、organization member 诊断
- GitHub 凭证管理
- SSH 密钥管理
- 出站凭证测试

Security 现在同时也是 IAM rollout 的过渡运维控制台。在本地单用户部署中，你可以继续使用 `auth.mode=disabled` 和内置本地管理员主体；当需要多用户浏览器访问时，可以在这里配置 OIDC、测试 provider，并显式启用。`/admin`、org admin 与 project settings 的稳态拆分定义见 [`../../zh/iam-admin-boundaries.md`](../../zh/iam-admin-boundaries.md)。

### 归档工单（Archived Tickets）

- 查看已归档的工单
- 恢复误归档的工单
- 永久删除不再需要的工单

## 初始化配置清单

第一次设置项目时，建议按以下顺序配置：

1. **通用设置** — 填写项目名称和描述
2. **工单状态** — 创建适合团队的状态列表（至少需要"待处理"和"已完成"）
3. **仓库** — 关联代码仓库并测试连接
4. **安全** — 配置必要的凭证
5. **通知** — （可选）设置通知规则

如果实例计划暴露到 loopback 之外，建议把 Security 提前，并先判断 `auth.mode=disabled` 是否仍然可接受，再开始邀请其他用户。

## 小贴士

- 项目创建后优先配置"工单状态"和"仓库"，这是后续所有功能的基础
- 状态不宜过多，5-7 个通常足够覆盖完整的工作流程
- 仓库凭证配置好后一定要测试连接
