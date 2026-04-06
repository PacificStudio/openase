# Machines（执行机器）

## 这是什么？

Machine 是 Agent 的执行环境。现在 OpenASE 会明确拆开两个以前容易混在一起的概念：

- **可达模式（reachability mode）**：控制面和机器如何互相到达。
- **运行时平面（runtime plane）**：远程执行统一走 websocket，本地保留 `local_process`。
- **辅助通道（helper lane）**：SSH 只用于引导、诊断和紧急修复。

对远程机器来说，产品模型现在是：

- **Direct Connect**：控制面可以主动拨到机器。
- **Reverse Connect**：机器守护进程可以反向拨回控制面。
- **WebSocket 执行**：推荐的远程执行路径。
- **SSH Helper**：用于引导、诊断和紧急修复的可选访问方式；不属于正常 runtime 执行平面。

## 基本概念

| 概念 | 说明 |
|------|------|
| **机器（Machine）** | 一个已注册的执行环境，包含身份、可达方式、工作空间和辅助访问信息。 |
| **可达模式** | `local`、`direct_connect`、`reverse_connect`。 |
| **执行模式** | `local_process` 或 `websocket`。 |
| **连接模式** | `local`、`ws_listener`、`ws_reverse`，以及遗留 helper-only 的 `ssh` 兼容标记。 |
| **SSH Helper** | 可选的 SSH 凭证，用于引导、诊断或紧急修复，而远程 runtime 仍走 websocket。 |
| **健康状态** | 当前可达性和资源状态（在线 / 离线 / 异常 / 维护中）。 |
| **连接探测（Probe）** | 通过当前实际实现的访问路径做检查，并收集诊断信息。 |

## 支持的远程拓扑

| 拓扑 | 存储语义 | 运行时路径 | 适用场景 |
|------|----------|------------|----------|
| **Direct-connect listener** | `reachability_mode=direct_connect`、`execution_mode=websocket` | 控制面直接拨号到公布的 websocket listener | 控制面可以直接通过网络到达机器 |
| **Reverse-connect daemon** | `reachability_mode=reverse_connect`、`execution_mode=websocket` | `openase machine-agent run` 保持反向 websocket 会话 | 机器可以主动向外拨号，但不适合暴露入站 listener |

## 添加机器

1. 进入 Machines 页面。
2. 点击 `Add Machine`。
3. 选择可达模式：
   - `Local`
   - `Direct Connect`
   - `Reverse Connect`
4. 选择执行路径：
   - 保留本地机器使用 `Local Process`
   - 远程机器使用 `WebSocket`
5. 先填写机器身份和工作空间字段。
6. direct-connect 机器需要保存公布的 listener endpoint。
7. reverse-connect 机器先保存记录，再签发 machine channel token，并在远端启动 `openase machine-agent run`。
8. 只有在需要 `openase machine ssh-bootstrap` 或 `openase machine ssh-diagnostics` 时才填写 SSH helper 凭证。
9. 点击 `Connection test` 验证当前配置路径。

## 运维建议

- 当控制面可以直接到达机器端点时，优先使用 `direct_connect + websocket`。
- 当机器只能向外拨号、不适合暴露入站监听时，优先使用 `reverse_connect + websocket`。
- 只有在需要引导访问、诊断或紧急修复时才保留 SSH helper 凭证。
- Direct-connect 机器必须保存有效的 `advertised_endpoint`；reverse-connect 机器必须保留可用的 daemon 注册路径。

## 迁移旧的远程机器记录

1. 盘点缺少 websocket 拓扑必填字段的远程机器记录。
2. 如果控制面可以直接拨到该主机，为它保存 websocket listener endpoint，并重新保存为 `direct_connect + websocket`。
3. 如果机器应主动向外拨号，则重新保存为 `reverse_connect + websocket`，签发 machine channel token，并启动 `openase machine-agent run`。
4. 每次迁移后执行 `openase machine test <machine-id>`。
5. 只有在运维仍需 helper 引导或诊断时才保留 SSH 凭证；执行路径本身不会再回退到 SSH。

## 监控机器

- 查看每台机器的健康状态、平台探测状态和最新资源快照。
- 通过可达模式和执行模式徽标区分“拓扑”与“当前基于 websocket 的 runtime 路径”。
- 排查问题时要区分 `ws_listener` 与 `ws_reverse`，前者偏向网络可达性，后者偏向守护进程注册与会话状态。
- 刷新机器健康状态以更新 heartbeat、WebSocket 会话状态和资源遥测。

## 小贴士

- 建议至少注册一台机器后再创建工单，否则 Agent 无处执行。
- 如果 Agent 执行失败，优先检查机器是否通过预期拓扑可达，以及 websocket 传输或守护进程健康是否异常。
- SSH helper 只应用于引导或诊断；正常执行应继续依赖 websocket runtime 路径。
