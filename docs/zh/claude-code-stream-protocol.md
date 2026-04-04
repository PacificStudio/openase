# OpenASE 中的 Claude Code 流协议

本文档描述了 OpenASE 当前理解的 Claude Code `stream-json` 协议。

它有意分离三个关注点：

1. OpenASE 实际读取的 Claude CLI 传输事件
2. 嵌入在 `assistant` 和 `user` 消息中的 Claude 消息块模式
3. OpenASE 派生的语义，例如"此工具结果是命令输出"

实现位于：

- [`internal/infra/adapter/claudecode/adapter.go`](../internal/infra/adapter/claudecode/adapter.go)
- [`internal/provider/claudecode.go`](../internal/provider/claudecode.go)
- [`internal/orchestrator/claude_protocol.go`](../internal/orchestrator/claude_protocol.go)
- [`internal/orchestrator/agent_adapter_claudecode.go`](../internal/orchestrator/agent_adapter_claudecode.go)

## 来源

OpenASE 使用两个证据来源：

- Claude Code 参考包：
  - [`references/claude-code-source/claude-code-2.1.88/cli.js.map`](../references/claude-code-source/claude-code-2.1.88/cli.js.map)
  - [`references/claude-code-source/claude-code-2.1.88/sdk-tools.d.ts`](../references/claude-code-source/claude-code-2.1.88/sdk-tools.d.ts)
- 观察到的本地 Claude Code 运行和适配器固定数据：
  - [`internal/orchestrator/agent_adapter_test.go`](../internal/orchestrator/agent_adapter_test.go)

当某字段未在参考模式中出现但存在于本地桥接输出中时，本文档称其为"观察到的桥接扩展"。

## 传输事件

Claude 适配器从 CLI `--output-format stream-json` 读取行分隔的 JSON。

OpenASE 当前解析的顶层事件类型：

- `assistant`
- `user`
- `result`
- `rate_limit_event`
- `stream_event`
- `system`
- `task_started`
- `task_progress`
- `task_notification`

OpenASE 保留的事件信封字段：

- `uuid`
  含义：此发出帧的事件标识。
- `session_id`
  含义：Claude 会话/线程标识。
- `parent_tool_use_id`
  含义：与产生后续消息的工具使用上下文的链接。
  重要：这是一个关系 ID，不是事件 ID。
- `raw`
  含义：为跟踪/调试目的存储的原始 JSON 帧。

## 消息块

`assistant.message.content` 和 `user.message.content` 被解析为类型化块。

当前显式建模的块类型：

- `text`
- `tool_use`
- `server_tool_use`
- `mcp_tool_use`
- `tool_result`

重要字段：

- `tool_use.id`
  含义：Claude 创建的稳定工具调用标识。
- `tool_result.tool_use_id`
  含义：此结果回答的工具调用。
- `tool_use.name`
  含义：Claude 调用的具体工具名称。
- `tool_use.input`
  含义：结构化工具参数。

OpenASE 将这些与自身派生的语义分开保存。它不在解析层将它们重命名为 OpenASE 概念。

## 任务与会话事件

参考包暴露了以下逻辑事件的 SDK 端模式：

- `system / task_started`
- `system / task_progress`
- `system / task_notification`
- `system / session_state_changed`

OpenASE 消费的 Claude CLI 桥接将其中一些展平为顶层传输类型：

- `task_started`
- `task_progress`
- `task_notification`

参考支持的任务/会话字段：

- `task_id`
- `tool_use_id`
- `description`
- `task_type`
- `workflow_name`
- `prompt`
- `usage.total_tokens`
- `usage.tool_uses`
- `usage.duration_ms`
- `last_tool_name`
- `summary`
- `status`（任务通知上）
- `output_file`
- `state`（会话状态变更上）

在本地运行/测试中观察到并由 OpenASE 保留的桥接扩展：

- `turn_id`
- `stream`
- `command`
- `text`
- `snapshot`
- 旧版顶层 `message`
- 旧版顶层 `status`

这些观察到的桥接扩展在 [`claude_protocol.go`](../internal/orchestrator/claude_protocol.go) 中被有意保持显式，以便读者能判断它们不是参考支持的模式保证。

## OpenASE 派生的语义

OpenASE 从 Claude 协议数据中派生了一些更高层级的运行时事件。

这些不是原生 Claude 协议字段：

- `ToolCallRequested`
  从 `tool_use` / `server_tool_use` / `mcp_tool_use` 块派生。
- `command` 输出流
  仅当工具名称在命令能力工具的显式白名单中且工具输入包含 `cmd` 或 `command` 时派生。
- `turn_diff_updated`
  当工具结果文本看起来像统一 diff 时派生。

命令工具白名单是有意显式的，非模糊匹配：

- `functions.exec_command`
- `exec_command`
- `bash`

这防止 OpenASE 仅因为未来工具的名称包含 `shell` 或 `terminal` 等词就静默地将其重新分类为命令工具。

## 项目标识规则

OpenASE 区分：

- 事件标识：`uuid`
- 工具链接：`parent_tool_use_id`
- 工具调用标识：`tool_use.id` / `tool_result.tool_use_id`

当 Claude 不提供可用的事件 ID 进行快照分组时，OpenASE 会合成一个。该合成是 UI 稳定性的后备手段，不是 Claude 协议契约的一部分。

## 非目标

OpenASE 当前不声称每个 Claude 流字段都被完全建模。

已知差距：

- 超出上述类型集的 Claude 特定块变体可能仅保留在原始负载中。
- Diff 提取仍然是基于统一 diff 文本的派生启发式方法，不是协议原生事件。
- 某些仅桥接的字段可能会在上游演进；当它们变化时，同步更新本文档和类型化解析器。
