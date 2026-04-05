# 统一 WebSocket Runtime 契约

WebSocket runtime 契约是两种 WebSocket 传输拓扑共享的唯一执行契约：

- `ws_listener`：控制面直接拨号到机器监听端点
- `ws_reverse`：机器守护进程保持 machine-channel 会话，并通过 `type=runtime` 隧道转发 runtime envelope

契约实现位置：

- 领域模型：`internal/domain/websocketruntime/contracts.go`
- 协议 client/server：`internal/infra/machinetransport/runtime_protocol.go`
- 共享契约测试套件：`internal/infra/machinetransport/runtime_contract_test.go`

## 消息模型

每个 runtime 帧都是一个 JSON envelope：

```json
{
  "version": 1,
  "type": "request|response|event|hello|hello_ack",
  "request_id": "uuid",
  "operation": "probe|preflight|workspace_prepare|workspace_reset|artifact_sync|command_open|session_input|session_signal|session_close|process_start|process_status|session_output|session_exit",
  "payload": {},
  "error": {
    "code": "invalid_request|protocol_version|workspace|artifact_sync|preflight|session_not_found|process_start|process_signal|transport_unavailable|unauthorized|unsupported|internal",
    "class": "auth|misconfiguration|transient|unsupported|internal",
    "message": "面向人的错误信息",
    "retryable": false,
    "details": {}
  }
}
```

握手流程：

1. Client 发送 `hello`，声明支持的协议版本与能力。
2. Server 返回 `hello_ack`，选择一个协议版本并回显支持的操作集合。
3. 后续所有 `request` 都由 `request_id` 关联到对应的 `response`。
4. 长生命周期会话的输出与退出事件通过 `event` envelope 发送。

## 操作集合

该契约覆盖所有执行关键路径：

- 可达性与探测：`probe`
- 二进制与 CLI 预检：`preflight`
- 工作空间生命周期：`workspace_prepare`、`workspace_reset`
- 产物传输：`artifact_sync`
- 命令会话：`command_open`、`session_input`、`session_signal`、`session_close`
- 进程生命周期：`process_start`、`process_status`、`session_output`、`session_exit`

上层编排应直接面向这些操作，而不是按 websocket 拓扑分支。

## 版本与兼容规则

- 每个 envelope 都必须带 `version`。
- 对未知协议版本，runtime peer 必须返回 `error.code=protocol_version` 且 `error.class=unsupported`。
- 新版本应尽量保持增量兼容：
  - 可以新增操作，但不要改写已有操作的语义
  - 新增 payload 字段应保持对旧 peer 可选
  - 删除或重定义已有操作时，必须升级协议版本
- `hello` 协商是兼容性的唯一入口。只有 `hello_ack` 确认过的操作才能被后续使用。

## 错误分类

契约将“面向编排的错误类别”和“面向操作的错误代码”分开建模。

- `auth`：凭证或权限问题，需要用户/运维修复
- `misconfiguration`：可修复的输入或环境错误，例如坏的 workspace root、非法请求、缺失二进制
- `transient`：可重试的运行时或传输故障
- `unsupported`：协议、版本或能力不匹配
- `internal`：实现侧的未预期错误

建议解释方式：

- UX 可以把 `auth` 和 `misconfiguration` 作为可修复问题直接提示
- 编排器只应在 `retryable=true` 时重试，通常对应 `transient`
- `unsupported` 应阻止继续 rollout，直到双方升级到兼容版本

## 验证

`TestUnifiedWebsocketRuntimeContractSuite` 会把同一套契约测试跑在：

- 直接 listener WebSocket runtime
- 反向 machine-channel WebSocket runtime participant

测试覆盖：

- 握手与版本协商
- probe 与 preflight
- workspace prepare/reset
- artifact sync
- command session open/stream/close
- process start/status/interrupt/exit
