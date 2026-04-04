# OpenASE 的 Gemini CLI 适配指南

本指南描述了 Gemini CLI 如何以精确的、协议优先的方式集成到 OpenASE 中。

它遵循与 Claude Code 集成相同的设计原则：

1. 定义显式协议类型
2. 将上游字段与 OpenASE 派生的语义分离
3. 记录每个字段来自哪个层
4. 用协议级测试锁定行为

## 范围

本指南关于 Gemini CLI 的无头模式，特别是：

- `--output-format json`
- `--output-format stream-json`

相关上游参考：

- [`references/gemini-cli/docs/cli/headless.md`](../references/gemini-cli/docs/cli/headless.md)
- [`references/gemini-cli/packages/core/src/output/types.ts`](../references/gemini-cli/packages/core/src/output/types.ts)
- [`references/gemini-cli/packages/core/src/agent/types.ts`](../references/gemini-cli/packages/core/src/agent/types.ts)
- [`references/gemini-cli/packages/core/src/agent/event-translator.ts`](../references/gemini-cli/packages/core/src/agent/event-translator.ts)
- [`references/gemini-cli/packages/cli/src/nonInteractiveCliAgentSession.ts`](../references/gemini-cli/packages/cli/src/nonInteractiveCliAgentSession.ts)
- [`references/gemini-cli/docs/reference/tools.md`](../references/gemini-cli/docs/reference/tools.md)

## OpenASE 当前状态

OpenASE 目前以非常薄的一次性模式使用 Gemini：

- [`internal/chat/runtime_gemini.go`](../internal/chat/runtime_gemini.go)

当前行为：

- 调用 `gemini -p ... --output-format json`
- 等待整个进程退出
- 仅解析最终的 `response` 和 `stats`
- 发出：
  - 规范化的助手消息
  - 最终 `done`

今天缺少的内容：

- 无流式增量
- 无 `init`
- 无工具调用事件
- 无工具结果事件
- 无警告/错误事件粒度
- 无 OpenASE 本地会话记账之外的每轮会话元数据
- 无中断/引出建模

如果我们希望 Gemini 接近 Claude/Codex 的对等程度，这是主要差距。

## 上游分层

Gemini CLI 有三个不同的层。

### 1. 内部模型/运行时事件

内部 Gemini Agent 逻辑发出 `ServerGeminiStreamEvent` 值，如：

- `ModelInfo`
- `Content`
- `Thought`
- `Citation`
- `ToolCallRequest`
- `ToolCallResponse`
- `Error`
- `Finished`
- `UserCancelled`
- `MaxSessionTurns`
- `AgentExecutionStopped`
- `AgentExecutionBlocked`
- `InvalidStream`

这些不是 CLI 传输契约。它们是内部的。

### 2. 内部 Agent 协议

Gemini 将内部事件翻译为 `AgentEvent` 值，如：

- `initialize`
- `session_update`
- `message`
- `agent_start`
- `agent_end`
- `tool_request`
- `tool_response`
- `tool_update`
- `usage`
- `error`
- `elicitation_request`
- `elicitation_response`
- `custom`

重要细节：

- `tool_update`
- `usage`
- `session_update`
- `elicitation_request`

即使它们不全部存活到最终无头 `stream-json` 传输中，也在内部存在。

### 3. 无头 CLI 传输契约

最终的 CLI `stream-json` 契约更窄，是 OpenASE 在通过 Gemini CLI 二进制文件使用时应视为上游线协议的内容。

事件类型：

- `init`
- `message`
- `tool_use`
- `tool_result`
- `error`
- `result`

如果通过 CLI 进程集成 Gemini，这是 OpenASE 正确的解析边界。

## 精确的无头传输 DTO

### `init`

```json
{
  "type": "init",
  "timestamp": "ISO-8601",
  "session_id": "string",
  "model": "string"
}
```

语义：启动无头流，标识 Gemini 会话，给出所选模型。

### `message`

```json
{
  "type": "message",
  "timestamp": "ISO-8601",
  "role": "user | assistant",
  "content": "string",
  "delta": true
}
```

说明：`delta` 是可选的。在流模式中，助手文本以 `delta: true` 增量发出。

### `tool_use`

```json
{
  "type": "tool_use",
  "timestamp": "ISO-8601",
  "tool_name": "string",
  "tool_id": "string",
  "parameters": {}
}
```

语义：精确的工具调用请求。`tool_id` 是后续 `tool_result` 的关联键。

### `tool_result`

```json
{
  "type": "tool_result",
  "timestamp": "ISO-8601",
  "tool_id": "string",
  "status": "success | error",
  "output": "string",
  "error": {
    "type": "string",
    "message": "string"
  }
}
```

重要限制：无头传输仅保留字符串 `output`，不保留更丰富的内部结构。如果 OpenASE 需要 Claude/Codex 级别的 diff、结构化文件输出或媒体结果保真度，CLI `stream-json` 已是有损边界。

### `error`

```json
{
  "type": "error",
  "timestamp": "ISO-8601",
  "severity": "warning | error",
  "message": "string"
}
```

语义：非致命警告和浮现的运行时问题。

### `result`

```json
{
  "type": "result",
  "timestamp": "ISO-8601",
  "status": "success | error",
  "error": { "type": "string", "message": "string" },
  "stats": {
    "total_tokens": 0,
    "input_tokens": 0,
    "output_tokens": 0,
    "cached": 0,
    "input": 0,
    "duration_ms": 0,
    "tool_calls": 0,
    "models": {}
  }
}
```

语义：无头运行的终端事件，携带聚合使用量。

## CLI 丢弃的内容

上游非交互式包装器显式忽略了多个内部 `AgentEvent` 类型：

- `initialize`、`session_update`、`agent_start`、`tool_update`
- `elicitation_request`、`elicitation_response`
- `usage`、`custom`

因此如果 OpenASE 仅消费无头 CLI `stream-json`，它无法恢复：内部思考/引用元数据、丰富的工具更新进度、最终结果前的显式使用事件、结构化引出/审批请求和自定义事件。

这是上游行为，不是 OpenASE 解析器的 bug。

## 工具映射指南

Gemini 的内置工具名称：

- 命令工具：`run_shell_command`
- 文件读取工具：`read_file`、`read_many_files`、`list_directory`、`glob`、`grep_search`
- 文件写入工具：`replace`、`write_file`
- 询问用户/中断工具：`ask_user`
- 搜索/获取工具：`google_web_search`、`web_fetch`
- 规划/记账工具：`write_todos`、`enter_plan_mode`、`exit_plan_mode`、`complete_task`、`save_memory`、`activate_skill`、`get_internal_docs`

OpenASE 应使用精确的白名单映射这些工具，而非模糊字符串匹配。

## 推荐的实现顺序

1. 引入 `gemini_protocol.go`，包含精确的传输 DTO
2. 添加读取 JSONL 事件的 Gemini 流解析器
3. 将聊天运行时从 `--output-format json` 切换到 `stream-json`
4. 仅当 stream-json 不可用时保留当前 JSON 解析器作为后备
5. 添加 tool-id 关联和派生语义映射
6. 添加协议和运行时测试
7. 之后再考虑是否需要比 CLI 更丰富的集成边界

## 最终建议

如果目标是通过 Gemini CLI 二进制文件的"精确且完整"集成，OpenASE 应将 `stream-json` 视为权威线协议。

如果目标是"Claude/Codex 级别的完全保真"，请注意 Gemini CLI 无头传输相对于 Gemini 内部 `AgentEvent` 层已是有损的。在这种情况下，下一步将是比 CLI `stream-json` 更丰富的集成边界。
