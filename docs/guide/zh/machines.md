# Machines（执行机器）

## 这是什么？

Machine 是 Agent 的执行环境。现在 OpenASE 会明确拆开两个以前容易混在一起的概念：

- **可达模式（reachability mode）**：控制面和机器如何互相到达。
- **执行模式（execution mode）**：机器可达之后，运行时命令实际通过什么路径执行。

对远程机器来说，产品模型现在是：

- **Direct Connect**：控制面可以主动拨到机器。
- **Reverse Connect**：机器守护进程可以反向拨回控制面。
- **WebSocket 执行**：推荐的远程执行路径。
- **SSH Helper**：用于引导、诊断，以及 rollout 期间的临时兼容；不再被描述为长期的一等 runtime mode。

## 基本概念

| 概念 | 说明 |
|------|------|
| **机器（Machine）** | 一个已注册的执行环境，包含身份、可达方式、工作空间和辅助访问信息。 |
| **可达模式** | `local`、`direct_connect`、`reverse_connect`。 |
| **执行模式** | `local_process`、`websocket`，或迁移期的 `ssh_compat`。 |
| **SSH Helper** | 可选的 SSH 凭证，用于引导、诊断，或旧机器记录迁移期间的临时兼容。 |
| **健康状态** | 当前可达性和资源状态（在线 / 离线 / 异常 / 维护中）。 |
| **连接探测（Probe）** | 通过当前实际实现的访问路径做检查，并收集诊断信息。 |

## 添加机器

1. 进入 Machines 页面。
2. 点击 `Add Machine`。
3. 选择可达模式：
   - `Local`
   - `Direct Connect`
   - `Reverse Connect`
4. 选择执行路径：
   - 远程机器优先使用 `WebSocket`
   - 只有在迁移旧的直连机器时才使用 `SSH Compat`
5. 填写机器身份、工作空间、端点或 helper 字段。
6. 点击 `Connection test` 验证当前配置路径。

## 运维建议

- 当控制面可以直接到达机器端点时，优先使用 `direct_connect + websocket`。
- 当机器只能向外拨号、不适合暴露入站监听时，优先使用 `reverse_connect + websocket`。
- 只有在需要引导访问、诊断，或临时 `ssh_compat` 过渡时才保留 SSH helper 凭证。
- 旧记录可能仍会表现为 `execution_mode=ssh_compat`；在把 SSH 视为可删除之前，先迁移到 websocket。

## 监控机器

- 查看每台机器的健康状态、平台探测状态和最新资源快照。
- 通过可达模式和执行模式徽标区分“拓扑”与“当前 runtime 路径”。
- 刷新机器健康状态以更新 heartbeat、WebSocket 会话状态和资源遥测。

## 小贴士

- 建议至少注册一台机器后再创建工单，否则 Agent 无处执行。
- 如果 Agent 执行失败，优先检查机器是否通过预期拓扑可达，以及当前执行路径是 `websocket` 还是遗留 `ssh_compat`。
- SSH helper 只应作为辅助能力；机器完成 websocket 迁移后应尽量移除。
