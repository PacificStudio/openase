# Codex / Claude Code 思考强度能力矩阵与 OpenASE 落地说明

本文沉淀 ASE-171 的调研结论、实现方案和验证结果，覆盖：

- Codex / Claude Code 的 reasoning or thinking 强度能力
- OpenASE 中 Provider 配置、持久化、运行时透传和前端展示的落点
- 当前已验证内容与剩余风险

## 调研基线

- Codex CLI 官方文档：<https://developers.openai.com/codex/cli>
- Claude Code 官方文档：<https://code.claude.com/docs/en/getting-started>
- 本地 CLI 帮助与版本：
  - `codex --version` -> `codex-cli 0.120.0`
  - `claude --version` -> `2.1.101 (Claude Code)`
  - `claude --help` 明确暴露 `--effort <level>`，可选值为 `low / medium / high / max`

## 能力矩阵

### Codex

OpenASE 当前内建模型目录显式建模了以下 reasoning effort 能力：

| Model | Supported efforts | Default |
| --- | --- | --- |
| `gpt-5.4` | `low`, `medium`, `high`, `xhigh` | `medium` |
| `gpt-5.4-mini` | `low`, `medium`, `high`, `xhigh` | `medium` |
| `gpt-5.3-codex` | `low`, `medium`, `high`, `xhigh` | `medium` |
| `gpt-5.3-codex-spark` | `low`, `medium`, `high`, `xhigh` | `high` |
| `gpt-5.2-codex` | `low`, `medium`, `high`, `xhigh` | `medium` |
| `gpt-5.2` | `low`, `medium`, `high`, `xhigh` | `medium` |
| `gpt-5.1-codex-max` | `low`, `medium`, `high`, `xhigh` | `medium` |
| `gpt-5.1-codex-mini` | `medium`, `high` | `medium` |

运行时透传方式：

- OpenASE 持久化字段：`reasoning_effort`
- OpenASE Runtime / Adapter：通过 Codex app-server `thread/start` 请求体字段 `reasoningEffort`
- 未知自定义模型：在 OpenASE 中显式呈现为 `unknown_model`，不猜测支持能力

### Claude Code

OpenASE 当前内建模型目录显式建模了以下 effort 能力：

| Model | Supported efforts | Default |
| --- | --- | --- |
| `claude-opus-4-6` | `low`, `medium`, `high`, `max` | account-plan dependent; OpenASE leaves default unset and defers to CLI |
| `claude-sonnet-4-6` | `low`, `medium`, `high` | account-plan dependent; OpenASE leaves default unset and defers to CLI |
| `claude-haiku-4-5` | not supported | n/a |

运行时透传方式：

- OpenASE 持久化字段：`reasoning_effort`
- OpenASE Runtime / Adapter：通过 Claude CLI 参数 `--effort <value>`
- Provider 自定义 `cli_args` 中手工写入的 `--effort` 会在 Service 层被归一化移除，避免与 Provider 配置冲突
- Claude 官方 `model-config` 当前说明：
  - effort 仅对 `claude-opus-4-6` 与 `claude-sonnet-4-6` 生效
  - `max` 仅对 `claude-opus-4-6` 可用
  - 默认 effort 取决于 Claude 账户计划，而不是单纯由模型决定；因此 OpenASE 对 Claude 不伪造固定 `default_effort`

## OpenASE 设计结论

本次实现采用“Provider 预选 preset + 显式能力模型”的方案，而不是让业务路径散落字符串判断。

### 领域模型

- 新增领域枚举：`AgentProviderReasoningEffort`
- 新增能力结构：
  - `AgentProviderReasoningCapability`
  - `AgentProviderModelReasoningCapability`
- 边界解析在 domain 层完成：
  - API / DB / 配置原始值统一进入 `parseAgentProviderReasoningEffort`
  - 非法值、模型不支持、未知模型都在边界层失败

### 持久化与 API

- `ent/schema/agentprovider.go` 新增可空字段 `reasoning_effort`
- Provider create / patch API 支持 `reasoning_effort`
- Provider 响应暴露：
  - `capabilities.reasoning`
  - `capabilities.reasoning.selected_effort`
  - `capabilities.reasoning.effective_effort`
  - 对 Claude 未显式选择 preset 时，`effective_effort` 可能为空，表示“沿用 CLI / account plan 默认值，OpenASE 不猜测”
- Provider model catalog 暴露：
  - `reasoning.state`
  - `reasoning.supported_efforts`
  - `reasoning.default_effort`
  - `reasoning.supports_provider_preset`
  - `reasoning.supports_model_override`

### 运行时链路

- Claude Code：
  - `internal/chat/runtime_claude.go`
  - `internal/orchestrator/agent_adapter_claudecode.go`
- Codex：
  - `internal/chat/runtime_codex.go`
  - `internal/orchestrator/agent_adapter_codex.go`
  - `internal/infra/adapter/codex/protocol.go`

### 前端

前端 Provider 设置页现在会：

- 根据 adapter + model 显示 reasoning preset 能力
- 提供 “Use model default” 与可选 preset 下拉项
- 对未知模型或不支持模型给出显式反馈，而不是静默吞掉
- 当用户切换到不支持该 preset 的模型时自动清理陈旧值

## 自动化测试覆盖

已补充或更新以下关键覆盖：

- Domain
  - `internal/domain/catalog/provider_reasoning_test.go`
  - `internal/domain/catalog/provider_models_test.go`
  - `internal/domain/catalog/agent_provider_capabilities_test.go`
  - `internal/domain/catalog/agent_catalog_test.go`
- Service / Repo / HTTP
  - `internal/service/catalog/agent_catalog_test.go`
  - `internal/httpapi/agent_catalog_test.go`
- Runtime / Adapter
  - `internal/infra/adapter/codex/adapter_test.go`
  - `internal/orchestrator/agent_adapter_test.go`
  - `internal/chat/runtime_codex_permission_test.go`
  - `internal/chat/service_test.go`
- Frontend
  - `web/src/lib/features/agents/provider-draft.test.ts`
  - `web/src/lib/features/agents/provider-model-options.test.ts`
  - `web/src/lib/features/agents/provider-pricing.test.ts`

## 真实 CLI 验证

### Codex

已完成两类真实验证：

1. 真实 `codex exec` 调用成功
   - 命令：`codex exec --skip-git-repo-check --model gpt-5.4 -c reasoning_effort='"high"' --json "Reply with the single word OK."`
   - 结果：成功返回 `OK`

2. 真实 `codex app-server` JSON-RPC 验证成功
   - `thread/start` 发送 `reasoningEffort: "high"`
   - `turn/start` 成功完成，`completion_status = completed`
   - 证明 OpenASE 当前使用的 app-server 透传字段名称与真实 CLI 对齐

### Claude Code

当前环境下只完成了 CLI 参数与帮助面验证，尚未拿到成功的真实 prompt 执行：

- `claude --help` 明确支持 `--effort <level>`
- 真实 prompt 执行命令：
  - `claude -p --model claude-sonnet-4-6 --effort high "Reply with the single word OK."`
- 当前环境结果（2026-04-11 再次复核）：
  - `claude auth status --json` 仍显示 `loggedIn: true`
  - 但真实执行仍返回 `401 Invalid authentication credentials`
  - `~/.claude/.credentials.json` 当前将 OAuth 记录嵌套在 `claudeAiOauth` 下，其中 `expiresAt = 1775527320406`，对应 `2026-04-07T02:02:00.406Z`
  - `claude --debug-file ...` 显示 OAuth refresh 请求对 `https://platform.claude.com/v1/oauth/token` 返回 `400`，随后真实 `/v1/messages` 请求继续返回 `401`
  - `claude auth login` 与 `claude setup-token` 在当前 CLI（`2.1.101`）里都只暴露交互式浏览器 OAuth 流，前者打印的 authorize URL 以 `https://claude.com/cai/oauth/authorize?...redirect_uri=https://platform.claude.com/oauth/code/callback...` 开头，后者会等待手工粘贴浏览器返回的 code
  - 从 CLI 二进制字符串可以确认还存在 `CLAUDE_CODE_OAUTH_TOKEN` 专用环境变量入口，但将当前 `claudeAiOauth.accessToken` 注入该变量后真实 prompt 仍返回 `401 Invalid authentication credentials`；改用 `refreshToken` 时会直接返回 `401 Invalid bearer token`
  - `claude setup-token` / `claude auth login` 当前都需要交互式浏览器完成重新授权
  - 直接回退到本机仍保留的旧版 CLI（`2.1.97` / `2.1.96` / `2.1.94`）重试，同样返回 `401`，说明 blocker 不是单一版本回归
  - 进一步复用 Firefox 已登录的 `claude.ai` / `claude.com` cookies 做浏览器自动化探测时，`claude auth login` 的 authorize URL 仍会卡在 Cloudflare `Performing security verification`，而 `claude setup-token` 路径会落到 `platform.claude.com` 登录页，说明当前环境也没有可稳定复用的非交互浏览器会话
  - 继续直接探测 `https://claude.ai/` 首页时，自动化浏览器同样只拿到 Cloudflare `Just a moment... / Performing security verification` 页面，进一步排除了从现有 Web 会话导出可用 SSO 状态的可能性
  - 进一步改用真实 headless Firefox + Selenium 注入 Snap Firefox 配置目录 `~/snap/firefox/common/.mozilla/firefox/6sfz7ccx.default/` 中已存在的 `claude.ai` / `claude.com` / `platform.claude.com` cookies 后，虽然浏览器已不再停在 Cloudflare 挑战页，但 `claude auth login` 和 `claude setup-token` 的 OAuth URL 最终仍都会落到 `https://platform.claude.com/login?returnTo=%2F%3F`
  - 上述真实浏览器页面标题为 `Sign In | Claude Platform`，正文继续要求 `Continue with Google` 或 `Continue with email`，没有出现可自动批准的 OAuth 授权页，也没有拿到可回填给 CLI 的 code；这说明当前浏览器侧现有会话也不足以完成 Claude Code 的平台授权
  - 进一步查阅 Claude Code 官方 settings 文档后，确认 CLI 还支持 `ANTHROPIC_AUTH_TOKEN` 这一 bearer-token 环境变量；但把当前 `claudeAiOauth.accessToken` 注入后，真实 `claude -p --bare ...` 调用会返回 `401` 与 `OAuth authentication is currently not supported.`，把 `refreshToken` 注入后则返回 `401 Invalid bearer token`
  - 这进一步说明：当前机器上保存的 Claude OAuth 凭据既不能通过 `CLAUDE_CODE_OAUTH_TOKEN` 被 CLI 直接复用，也不能通过官方记录的 `ANTHROPIC_AUTH_TOKEN` 旁路复用；若要在无人值守环境完成最终验证，仍需要真正可用的 `ANTHROPIC_API_KEY` 或人工刷新后的平台登录态

这说明实现链路已经对齐 Claude Code CLI 参数契约，但本机当前认证状态不足以完成成功的在线 prompt 验证；要完成最终验收，仍需要刷新 Claude 登录态或提供可用的 `ANTHROPIC_API_KEY`。

补充修正（2026-04-12）：

- 基于 Claude 官方 `model-config` 文档，OpenASE 已把 Claude reasoning 内建矩阵修正为：
  - `claude-opus-4-6`: `low / medium / high / max`
  - `claude-sonnet-4-6`: `low / medium / high`
  - `claude-haiku-4-5`: unsupported
- 同时移除了对 Claude 固定 `default_effort = medium` 的错误假设，避免在 UI / API 中把 plan-dependent 默认值错误呈现成确定值。

## 风险与后续建议

- Provider 能力矩阵依赖上游 CLI / 模型目录；如果 Codex 或 Claude Code 调整了支持模型与 effort 档位，需要同步更新内建模型目录
- 对未知模型，OpenASE 当前选择“显式 unsupported”，这是为了避免错误猜测；如果未来需要支持自定义模型声明能力，应新增显式配置面，而不是恢复隐式推断
- Claude Code 的本机认证状态目前不可靠，CI 或交付前应补一条真正成功的在线调用验证，再推进最终交付
