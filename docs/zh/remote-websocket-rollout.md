# 远程 WebSocket 传输部署指南

本指南将远程 WebSocket 传输从仅代码实现转变为可运维的功能。涵盖验证矩阵、可观测性契约、部署前提、守护进程安装流程、升级与回滚步骤以及部署检查清单。

本指南中的术语：

- `direct_connect`：控制面可以主动拨到机器。
- `reverse_connect`：机器守护进程可以反向拨回控制面。
- `websocket`：目标远程执行路径。
- `ssh_compat`：需要迁移到 websocket 的遗留执行模式存储值。SSH 仅保留为引导、诊断和紧急修复 helper。

## 自动化验证矩阵

从仓库根目录运行聚焦的传输矩阵：

```bash
scripts/ci/remote_transport_matrix.sh
```

当前矩阵覆盖：

| 场景 | 覆盖 |
| --- | --- |
| 监听与反向拓扑共享的统一 WebSocket runtime 契约 | `TestUnifiedWebsocketRuntimeContractSuite` |
| SSH 引导 + 反向 WebSocket 机器会话 | `TestMachineConnectWebsocketPublishesActivityAndMetrics` |
| SSH 引导 helper 行为 | `TestRunMachineSSHBootstrapUploadsBinaryEnvAndService` |
| SSH 诊断 helper 行为 | `TestRunMachineSSHDiagnosticsReportsBootstrapAndRegistrationIssues` |
| SSH 引导 + 监听 WebSocket 运行时 | `TestRuntimeLauncherLaunchesWebsocketListenerRuntimeWithHooksAndArtifactSync` |
| 直接 SSH runtime 拒绝 | `TestRuntimeLauncherRunTickRejectsSSHRuntimeExecution` |
| 无反向会话时反向 WebSocket 启动不回退到 SSH | `TestRuntimeLauncherDoesNotFallBackToSSHWhenWebsocketReverseTransportUnavailable` |
| 远程二进制/预检失败 | `TestRuntimeLauncherRecordsWebsocketPreflightFailureStageInActivityAndMetrics` |
| 守护进程认证失败 | `TestMachineConnectWebsocketAuthFailurePublishesActivityAndMetric` |

每个正常路径运行时用例验证：

- 机器注册或可达性
- 工作空间准备
- Agent 进程启动
- 输出流或命令握手
- 清理或断开连接记账

runtime 契约本身现在由两种 WebSocket 拓扑共同验证：

- 监听 WebSocket 通过直接 listener 端点验证契约
- 反向 WebSocket 通过 machine-channel `runtime` 消息验证同一份契约

关于消息模型、版本兼容规则与错误分类，请参见 [`docs/zh/websocket-runtime-contract.md`](./websocket-runtime-contract.md)。

## 可观测性契约

### 指标

OpenASE 现在发出以下传输相关指标：

- `openase.machine_channel.active_sessions`
  - 标签：`transport_mode`
- `openase.machine_channel.websocket_reconnect_total`
  - 标签：`transport_mode`
- `openase.machine_channel.events_total`
  - 标签：`event`、`transport_mode`
- `openase.runtime.launch_failures_total`
  - 标签：`failure_stage`、`transport_mode`

推荐告警：

- `openase.machine_channel.websocket_reconnect_total` 的重连速率飙升
- `openase.runtime.launch_failures_total` 中 `failure_stage in {workspace_transport, openase_preflight, agent_cli_preflight, process_start}` 持续非零
- `openase.machine_channel.active_sessions` 意外下降或波动

### 结构化日志

传输相关日志在已知时应包含以下关联字段：

- `machine_id`
- `session_id`
- `transport_mode`
- `workspace_root`
- `failure_stage`
- `ticket_id`
- `run_id`
- `agent_id`

运行时启动器现在在启动失败时记录 failure-stage 元数据，WebSocket 守护进程注册日志携带机器和会话标识符。

### 活动事件

项目活动现在记录：

- `machine.connected`
- `machine.reconnected`
- `machine.disconnected`
- `machine.daemon_auth_failed`
- `agent.failed`
  - 在启动期间失败时包含 `failure_stage`、`transport_mode`、`machine_id` 和 `workspace_root`

结合项目活动和日志来回答：

- 哪台机器丢失或替换了 WebSocket 会话？
- 守护进程是认证失败还是仅重连？
- 运行时在传输设置、工作空间准备、二进制预检还是进程启动时失败？

## 部署前提

### 控制面板 URL、TLS 和 DNS

- 机器守护进程通过 `--control-plane-url` 接受控制面板基础 URL、API 基础 URL 或直接 WebSocket 端点。
- 生产环境中，优先使用具有有效 DNS 和 TLS 的稳定 HTTPS 源，然后将该源传递给守护进程。
- 反向 WebSocket 需要从远程机器到控制面板的出站可达性。
- 监听 WebSocket 需要从控制面板到机器公布的 WebSocket 端点的入站可达性。
- 如果 TLS 终止在 OpenASE 前面，确保公布的主机名与监听证书匹配，且 WebSocket 升级头能通过代理。

### 可达性与兼容前提

反向 WebSocket：

- 最适合机器可以向外拨号但无法暴露入站监听器的场景
- 需要机器通道令牌
- 仅当运维需要 helper 引导或诊断时才需要在机器记录上保留 SSH 凭证

监听 WebSocket：

- 最适合控制面板可以直接到达机器的场景
- 需要机器公布的 WebSocket 端点
- 如需直接修复入口，可保留 SSH 凭证用于 helper 引导或诊断

SSH 兼容路径：

- 不再是受支持的 runtime 回退路径
- 应被视为遗留记录状态加 helper 基础设施，而不是远程执行模型

## 引导与守护进程安装

### 1. 构建或安装 OpenASE 二进制文件

从源码构建当前二进制文件：

```bash
make build-web
```

### 2. 创建或更新机器记录

在启动守护进程之前，确保机器记录有：

- 预期的可达性与执行组合：
  - `reverse_connect + websocket`
  - `direct_connect + websocket`
- 有效的 `workspace_root`
- 只有在仍需引导、诊断或紧急修复时才保留 SSH helper 凭证
- 远程 CLI 无法从 `PATH` 发现时的 `agent_cli_path`

### 3. 签发专用机器通道令牌

在控制面板主机上：

```bash
./bin/openase machine issue-channel-token \
  --machine-id <machine-uuid> \
  --ttl 24h \
  --format shell
```

这会打印以下 shell 导出：

- `OPENASE_MACHINE_ID`
- `OPENASE_MACHINE_CHANNEL_TOKEN`
- `OPENASE_MACHINE_CONTROL_PLANE_URL`

### 4. 启动反向 WebSocket 守护进程

如果控制面已经可以通过 SSH 到达机器，可以直接使用 helper 流而不必手工拷贝文件：

```bash
./bin/openase machine ssh-bootstrap <machine-uuid>
```

它会上传当前 `openase` 二进制、写入 machine-agent 环境文件、安装用户级服务并重启。

不使用 helper 的运维手工方式如下：

在远程机器上：

```bash
export OPENASE_MACHINE_ID=<machine-uuid>
export OPENASE_MACHINE_CHANNEL_TOKEN=<issued-token>
export OPENASE_MACHINE_CONTROL_PLANE_URL=https://openase.example.com

/usr/local/bin/openase machine-agent run \
  --agent-cli-path /usr/local/bin/codex \
  --openase-binary-path /usr/local/bin/openase
```

推荐的 `systemd --user` 单元：

```ini
[Unit]
Description=OpenASE machine-agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/openase machine-agent run --agent-cli-path /usr/local/bin/codex --openase-binary-path /usr/local/bin/openase
Restart=always
RestartSec=5
Environment=OPENASE_MACHINE_ID=<machine-uuid>
Environment=OPENASE_MACHINE_CHANNEL_TOKEN=<issued-token>
Environment=OPENASE_MACHINE_CONTROL_PLANE_URL=https://openase.example.com

[Install]
WantedBy=default.target
```

对于监听 WebSocket 模式，先部署监听端点，然后在机器记录上保存公布的 WebSocket URL，并验证控制面板可以拨号到该端点。

## 升级与回滚

### 升级

1. 构建目标 `openase` 二进制文件。
2. 在远程机器上，将新二进制文件安装在当前二进制文件旁边。
3. 重启 `machine-agent` 服务。
4. 确认：
   - 出现 `machine.connected` 或 `machine.reconnected` 活动
   - `openase.machine_channel.active_sessions` 恢复到预期水平
   - 金丝雀运行时启动期间 `openase.runtime.launch_failures_total` 保持平稳

### 回滚

1. 停止守护进程服务。
2. 恢复之前的二进制文件。
3. 重启守护进程。
4. 如果 WebSocket 启动仍不稳定，使用 `openase machine ssh-diagnostics <machine-uuid>` 和 `openase machine ssh-bootstrap <machine-uuid>` 修复守护进程配置，再决定是否继续扩大部署。

## 故障排查

| 症状 | 可能信号 | 检查内容 |
| --- | --- | --- |
| 守护进程从未注册 | `machine.daemon_auth_failed`，`auth_failed` 指标 | 令牌已撤销、令牌已过期、错误的机器 ID、错误的控制面板 URL |
| 频繁重连 | `machine.reconnected`，重连计数器 | 控制面板重启、网络抖动、代理空闲超时、心跳间隔不匹配 |
| 运行时启动前失败 | `agent.failed` 带 `failure_stage` | 查看阶段是 `workspace_transport`、`workspace_root`、`repo_auth`、`openase_preflight`、`agent_cli_preflight` 还是 `process_start` |
| 监听 WebSocket 无法拨号 | 启动器日志带 `failure_stage=workspace_transport` 或 `preflight_transport` | DNS 解析、TLS 链、公布端点正确性、防火墙可达性 |
| 远程二进制缺失 | `failure_stage=openase_preflight` | `.openase/bin/openase` 存在于已准备的工作空间中且可以解析 `OPENASE_REAL_BIN` 或 `PATH` |
| Git 克隆失败 | `failure_stage=repo_auth` | 仓库凭证投射、`GH_TOKEN`、部署密钥或远程 git 传输策略 |
| 工作空间根目录无效 | `failure_stage=workspace_root` | 保存的机器 `workspace_root`、权限、挂载的文件系统可用性 |

当你需要快速确认工作空间权限、远程二进制是否存在、服务状态或最近日志时，使用 `openase machine ssh-diagnostics <machine-uuid>` 获取 helper-only 诊断输出。

## 部署检查清单

### 阶段 1：传输兼容性

- 运行 `scripts/ci/remote_transport_matrix.sh`
- 确认直接 SSH runtime 会被拒绝
- 确认反向 WebSocket 启动失败不会回退到 SSH
- 确认 WebSocket 预检失败被分类为 `failure_stage`

### 阶段 2：反向 WebSocket 金丝雀

- 在小规模机器子集上启用 `ws_reverse`
- 只在运维需要 helper 引导或诊断的机器上保留 SSH 凭证
- 验证守护进程重启下的 `machine.connected` 和重连行为
- 确认强制 WebSocket 传输失败仍被归类为 WebSocket 侧启动错误

### 阶段 3：监听扩展

- 仅在反向 WebSocket 金丝雀稳定后启用 `ws_listener`
- 在每次监听部署前验证 DNS、TLS 和控制面板直接可达性
- 每批部署运行至少一个监听运行时正常路径

### 阶段 4：保留可选 SSH Helper 的广泛部署

- 在部署窗口期间不要把 SSH 当作 runtime 执行计划的一部分
- 保持以下仪表盘视图：
  - 按 `transport_mode` 的启动成功率
  - 重连恢复
  - 孤立或卡住的运行时计数
  - 活跃 WebSocket 机器会话

### 成功标准

- 运行时启动成功率在连续两个部署窗口中保持在 SSH 基线或以上
- 重连恢复在不需要运维操作的情况下为目标机器群恢复会话
- `openase.runtime.launch_failures_total` 保持在商定的错误预算以下，且主要由已知的、可操作的阶段主导
- WebSocket 启用后孤立进程或卡住会话计数不增加
- 项目活动和日志足以在不登录机器的情况下识别机器、会话、传输和失败阶段
