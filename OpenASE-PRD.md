# OpenASE — Open Auto Software Engineering

**工单驱动的全自动化软件工程平台 | 产品需求文档 (PRD)**

2026 年 3 月 | 开源协议：Apache License 2.0

---

## 第一章 执行摘要

OpenASE（Open Auto Software Engineering）是一个以工单为核心驱动的全自动化软件工程平台。它将软件开发生命周期中的编码、测试、文档、安全扫描、部署等工作，抽象为标准化的工单（Ticket），由 AI Agent 自主领取并执行，实现从需求到代码合并的全流程自动化。

平台采用 All-Go 单体架构，编排引擎与 API 服务共享同一代码库，通过适配器抽象层兼容多种 Agent CLI（OpenAI Codex、Claude Code、Gemini CLI 等），内置审批治理、生命周期 Hook 和成本管控体系。

**核心价值主张：**

- 让工程团队从"监督 AI 编码"转变为"管理工作本身"
- 将 Agent 的行为定义（Workflow / Harness / Skills）纳入平台控制面版本管理，使其可审计、可回滚、可稳定投影到任意运行时
- 通过生命周期 Hook 体系（任务领取前的仓库准备、执行中的质量检查、完成后的清理验证）让团队自定义"什么算完成"
- 单二进制部署（前端 `go:embed` 内嵌），下载即用，无需 Docker 或 Node.js

---

## 第二章 背景与问题陈述

### 2.1 行业现状

2026 年初，AI 编码 Agent 的能力已经达到了可以独立完成中等复杂度开发任务的水平。OpenAI Codex、Claude Code、Gemini CLI 等工具能够读取代码库、编辑文件、运行命令、提交 PR。然而，从"Agent 能写代码"到"Agent 能可靠地交付软件"之间，存在一条巨大的鸿沟。

### 2.2 核心问题

我们在调研 OpenAI Symphony（Elixir/OTP 编排引擎）和 OpenClaw Mission Control（FastAPI + Next.js 运维平台）两个开源项目的源码后，识别出当前 AI Agent 编码领域的五个核心痛点：

**问题 1：Agent 需要持续人工监督**

- 现状：开发者启动一个编码 Agent 后，需要持续关注其输出，手动判断何时停止、何时干预。这与传统开发相比并没有减少多少人力投入，只是将"写代码"变成了"看 Agent 写代码"。
- 根因：缺乏自动化的质量验证和闭环机制。Agent 不知道什么时候算"完成"，人类不得不充当质量把关者。

**问题 2：多任务并行处理困难**

- 现状：当团队有数十个待办事项时，只能一个一个地交给 Agent 处理。没有编排机制来管理优先级、并发数、依赖关系。
- 根因：现有工具设计为单任务交互模式，缺乏任务队列和调度系统。

**问题 3：Agent 行为不可控、不可追溯**

- 现状：Agent 的行为由每次的 Prompt 决定，没有版本控制，无法回溯"上次它为什么这样做"。不同开发者给出的 Prompt 质量参差不齐，导致输出不一致。
- 根因：Agent 的工作规范（Prompt + 配置）没有被当作"代码"来管理。

**问题 4：多 Agent CLI 绑定严重**

- 现状：Symphony 只支持 Codex，Mission Control 只支持自有协议。团队想同时使用 Claude Code 和 Codex 处理不同类型的任务时，没有统一的编排层。
- 根因：各平台为特定 Agent CLI 设计，没有抽象出通用的适配器接口。

**问题 5：从编码到交付的断裂**

- 现状：Agent 写完代码后，测试、文档更新、安全扫描、部署仍然是割裂的手动流程。没有平台将这些视为统一的软件工程活动来管理。
- 根因：现有工具聚焦于"编码"这一个环节，而非整个软件工程生命周期。

### 2.3 我们的判断

AI Agent 编码领域正在经历从"单 Agent 使用"到"Agent 舰队管理"的范式转移。下一代平台需要解决的不是"让 Agent 写更好的代码"，而是"如何让一支 Agent 团队可靠地交付软件"。这正是 OpenASE 要做的事情。

---

## 第三章 产品愿景与目标

### 3.1 愿景

成为 AI 时代的软件工程操作系统——一个让 AI Agent 团队能够像人类工程团队一样，按工单领取任务、遵循规范执行、通过质量门禁、最终交付可合并代码的自动化平台。

### 3.2 设计原则

**原则 1：工单即一切** — 工单是系统的原子单位，也是 Agent 与人类的唯一协作接口。所有工作——无论是开发需求、Bug 修复、文档更新还是安全扫描——都表达为工单。创建工单即下达指令，关闭工单即确认交付。

**原则 2：Workflow 定义工作方式** — 每个工单挂载一个 Workflow 类型（coding、test、doc、security、deploy 等），Workflow 的 Harness 内容定义了 Agent 的工作交接、工作规范和工作边界。同样的需求，挂不同的 Workflow，Agent 就用不同的方式处理。

补充说明：Workflow 不直接“挑一个运行中的 worker 实例”，而是绑定一个 Agent 定义。Agent 定义再绑定到某个 Provider。也就是说：

- Provider = 某台 Machine 上可用的一种 Coding Agent CLI 适配配置
- Agent = 某个 Provider 在项目里的运行方式定义
- Workflow = 绑定某个 Agent 定义，声明这类工单该由哪种运行方式、哪台机器、哪种 CLI 驱动

**原则 3：行为即受管资产** — Agent 的行为定义（Workflow Harness、Skills、绑定关系）存储在 OpenASE 控制面，而非项目仓库中的 `.openase/` 文件。修改这些资产会生成平台内版本与审计记录；运行时在启动时按版本快照 materialize 到工作区，保证可追溯、可回放、可稳定恢复。

**原则 4：信任但验证** — 平台不假设 Agent 的输出总是正确的。质量验证通过生命周期 Hook 实现——团队自定义在工单的各个阶段执行什么检查（lint、测试、安全扫描等），Hook 失败则阻止状态推进。高风险操作需要人类审批。

**原则 5：渐进式自动化** — 从手动分配 Workflow 和 Agent 开始，逐步引入 AI 自动分配、自动审批、Harness 自优化。让团队按自己的节奏建立对系统的信任。

### 3.3 目标用户

- **主要用户**：3-20 人的工程团队，已经在使用 AI 编码工具（Claude Code、Codex 等），但苦于无法规模化地管理 Agent 工作。
- **次要用户**：个人开发者，希望用 Agent 并行处理多个个人项目的不同任务。
- **未来用户**：企业工程平台团队，需要在组织级别统一管理 Agent 的使用、成本和安全。

### 3.4 成功指标

| 指标 | 定义 | Phase 1 目标 |
|------|------|-------------|
| 工单自动完成率 | Agent 自主完成（无人工干预）的工单占比 | ≥ 60% |
| 平均工单周期 | 从 todo 到 done 的平均时间 | < 30 分钟（中等复杂度） |
| PR 一次通过率 | Agent 提交的 PR 无需 change request 即合并的比例 | ≥ 40% |
| 编排引擎可用性 | 编排服务无故障运行时间 | ≥ 99.5% |
| 部署时间 | 从 git clone 到服务可用 | < 10 分钟 |

---

## 第四章 核心设计哲学

### 4.1 "工单即一切"的运转模型

整个系统围绕工单运转。以下是一个工单的完整生命周期：

1. 人类或系统创建工单（手动创建 / 定时任务触发 / 外部 API 调用）
2. 工单带有 Workflow 类型，定义了处理方式（coding / test / doc / security / deploy）
3. 编排引擎在调度周期中发现可执行的工单，解析其 Workflow 绑定的 Agent 定义，直接从 DB 读取当前已发布的 Workflow / Skills 版本并 materialize 本次运行快照；随后准备所需 repo 工作区（直接 clone/fetch），执行 `on_claim` Hook（派生工单工作副本、密钥解密、依赖准备等），然后创建 AgentRun
4. AgentRun 按照 Workflow Harness 执行工作（编码 + 测试 + PR 创建）
5. AgentRun 在每轮 Turn 后都由编排引擎重新读取工单状态；只要工单仍在 `pickup/active` 状态，就继续同一 session 的后续 Turn，直到状态变化、命中 `max_turns`、暂停或取消
6. 工单离开 `pickup/active` 状态时，编排引擎只停止 continuation 并释放 runtime ownership；业务状态推进必须来自 Agent 的显式平台操作或人类在 UI/API 中的显式状态修改
7. 人类确认完成后将工单移动到 `done`，执行 `on_done` Hook（工作区清理、通知发送等）；业务级生命周期进入 `ActivityEvent`，细粒度 CLI 轨迹进入 `AgentTraceEvent`，人类可读动作阶段进入 `AgentStepEvent`

### 4.2 多 Workflow 类型

| Workflow 类型 | 职责 | 典型产出 |
|--------------|------|---------|
| coding | 需求实现、功能开发、Bug 修复、重构 | Git Branch + PR + CI 绿灯 |
| test | 根据需求/代码编写测试，防回归 | 测试代码 + 覆盖率报告 |
| doc | 扫描代码变更，确保文档对齐最新实现 | 文档更新 PR |
| security | 代码安全分析，漏洞检测，编写 PoC | 安全报告 + 修复 PR |
| deploy | 连接远程机器，更新配置，部署新版本 | 部署日志 + 版本验证结果 |
| refine-harness | 自动优化 Workflow Harness 内容（元工作流） | 更新后的 Harness 模板 |
| custom | 用户自定义的任意工作流 | 用户定义 |

### 4.3 Auto Harness 机制

每个 Workflow 的核心是一个 Harness 文档，由 OpenASE 控制面持久化和版本化管理。Harness 仍使用 YAML Frontmatter + Markdown 格式定义 Agent 的工作规范，但它不再以项目仓库 `.openase/harnesses/` 中的文件作为权威源；运行时会在启动时把当前版本 materialize 到工作区供 Agent 使用。

**Harness 的核心特性：**

- **版本化**：Harness 修改生成新的平台版本，新工单默认使用最新发布版本
- **运行时快照**：每次 AgentRun 固定记录自己使用的 Harness 版本；新 runtime 自动拿最新版本，已运行中的 runtime 不隐式漂移
- **模板变量**：支持 `{{ ticket.identifier }}`、`{{ project.name }}`、`{{ agent.name }}`、`{{ attempt }}` 等
- **版本追溯**：每个工单记录执行时使用的 Harness 版本号
- **自我优化**：refine-harness 元工作流分析执行历史，自动优化 Harness 内容

### 4.4 分阶段策略

当前阶段专注于 Org → Project → Ticket → Workflow → Agent 主线。Team、Role、Permission 等企业级能力延后实现：

- **Phase 1**：走通主干线，证明自动化闭环可行
- **Phase 2**：Git 集成 + 审批治理 + 多 Workflow
- **Phase 3**：定时任务 + Auto Harness 自优化 + 成本管控
- **Phase 4**：多租户 + 插件化 + 开放 API

---

## 第五章 技术架构

### 5.1 架构决策记录

**决策 1：All-Go 单体架构**

| 考虑方案 | 优势 | 劣势 | 结论 |
|---------|------|------|------|
| Python API + Go 编排 | 各取所长 | 双语言：双套依赖、双套 CI、跨语言序列化 | 否决 |
| All Python | 统一语言、生态大 | subprocess 管理脆弱、GIL 限制、部署臃肿 | 否决 |
| Elixir/OTP (Symphony 方案) | BEAM 天然适合 Agent 进程管理 | 人才稀缺、生态小、学习曲线陡 | 否决 |
| All-Go (采用) | goroutine 轻量并发、单二进制部署、人才丰富 | 不如 BEAM 优雅、CRUD 代码略多 | 采用 |

**决策 2：SSE 替代 WebSocket**

| 考虑方案 | 优势 | 劣势 | 结论 |
|---------|------|------|------|
| WebSocket | 双向通信、低延迟 | 连接管理复杂、重连逻辑重、负载均衡困难 | 否决 |
| 纯 SSE (采用) | HTTP 原生、自动重连、浏览器支持好 | 仅单向推送 | 采用 |

理由：Agent 监控场景只需要服务端向客户端推送事件，不需要双向通信。OpenClaw Mission Control 源码验证了 SSE 在此场景下完全够用。

**决策 3：Workflow / Skill 内容存储在平台控制面**

| 考虑方案 | 优势 | 劣势 | 结论 |
|---------|------|------|------|
| Git 仓库 `.openase/` 作为权威源 | 人类可直接看到文件 | 强依赖额外本地缓存层 / branch / 工作树状态，控制面资产和代码仓库耦合过深 | 否决 |
| 纯文件系统共享目录 | 简单 | 多机 / 多实例一致性差，恢复与审计弱 | 否决 |
| 数据库存 Skill / Harness 版本资产 + 文件清单 + 审计表（采用） | 控制面真相清晰、事务一致、可按 bundle 版本 materialize 到任意 runtime | 需要自建版本、文件存储与导出逻辑 | 采用 |

**决策 4：适配器抽象层替代统一协议**

各 Agent CLI 的原生协议差异太大（Codex 是 JSON-RPC over stdio，Claude Code 是 NDJSON stream），强行统一会丢失各自的独特能力。因此采用适配器模式：每个 Agent CLI 有自己的原生适配器，但向编排引擎暴露统一的 Go interface。

### 5.2 技术栈

| 层级 | 技术选型 | 选择理由 |
|------|---------|---------|
| CLI 框架 | cobra | 子命令模式（serve / orchestrate / all-in-one） |
| Web 框架 | Echo v4 | 轻量高性能、中间件生态成熟、OpenAPI 集成好 |
| ORM | ent (Facebook 开源) | 代码生成 + 类型安全 + Graph Traversal 查询 |
| 数据库迁移 | atlas (by Ariga) | ent 原生集成、声明式迁移 + 差异检测 |
| 数据库 | PostgreSQL 16 | JSON 字段 + 全文搜索 + 成熟可靠；用户自行部署，Setup Wizard 引导连接 |
| 进程间通信 | PostgreSQL LISTEN/NOTIFY | 跨服务事件通知（分开部署时）；同进程时直接用 Go channel |
| 定时任务 | robfig/cron v3 | Go 生态最成熟的 cron 库 |
| Git 操作 | go-git v5 | 纯 Go 实现、无 C 依赖 |
| 文件监控 | fsnotify | 工作区文件观测 / 本地调试辅助（不再作为 Harness 权威源监听） |
| 进程管理 | os/exec + context | Agent CLI 子进程管理 + 超时取消 |
| 日志 | slog (标准库) | Go 1.21+ 内置结构化日志 |
| 配置 | viper | 多源配置（文件 / 环境变量 / 命令行） |
| OpenAPI 生成 | OpenASE 内置 exporter + kin-openapi | 从 Go HTTP contract 导出 `api/openapi.json`，作为前后端交接的唯一接口事实来源 |
| 前端框架 | SvelteKit + Tailwind CSS | 编译时框架，无运行时开销；adapter-static 输出纯静态文件，完美匹配 `go:embed`；SSE 流式更新与 Svelte store 天然契合 |
| 前端组件库 | shadcn-svelte (基于 bits-ui) | 复制粘贴式组件源码，无运行时依赖；Tailwind 原生；Linear 风格审美匹配；Kanban 拖拽搭配 svelte-dnd-action |
| 前端图标库 | Lucide (lucide-svelte) | shadcn-svelte 默认图标库；每个图标独立 Svelte 组件，按需 import 完美 tree-shake；1400+ 图标 |
| 前端 API 客户端 | openapi-typescript | 从 `api/openapi.json` 生成 TypeScript 合同类型，配合轻量 fetch wrapper + Svelte store |
| 认证 | 双模（Local Token / OIDC） | 自部署用 Token、多租户用标准 OIDC |

### 5.3 服务架构

| 服务 | 语言 | 职责 |
|------|------|------|
| `openase serve` | Go | 工单 CRUD、状态机、Workflow 管理、SSE 推送、认证 |
| `openase orchestrate` | Go | 工单轮询调度、Agent 进程管理、心跳监控、重试调度、Stall 检测、读取已发布版本并 materialize runtime 快照 |
| `openase all-in-one` | Go | 单进程模式，通过 goroutine 并行运行 serve + orchestrate |
| web-app | Go (embed) | SvelteKit 构建的静态资源编译进 Go 二进制，通过 `go:embed` 内嵌，无需独立部署 |
| PostgreSQL 16 | - | 唯一数据库。用户自行部署（已有实例或 Docker 一行命令启动），OpenASE 只负责连接 |

**部署模型：Binary-first**

OpenASE 编译为单个二进制文件，前端静态资源通过 `go:embed` 内嵌。用户不需要安装 Docker、Node.js 或任何其他运行时。下载二进制 → 运行 → 浏览器打开 → 开始使用。

Docker 作为可选的高级部署方式保留（适合需要容器编排的生产环境），但不是默认推荐。

**进程间通信：PostgreSQL 是唯一的共享状态**

两个进程（serve 和 orchestrate）的主要通信媒介是 PostgreSQL 本身——编排引擎每个 Tick 轮询数据库获取最新工单状态，API 进程写入数据库后编排引擎自然能读到，大多数场景不需要额外的即时通知。

只有少数场景需要即时通信（不能等下个 Tick）：

| 场景 | 方向 | 为什么不能等 Tick？ | 通信方式 |
|------|------|-------------------|---------|
| 用户取消工单 | serve → orchestrate | 正在运行的 Agent 应立即停止，不能白跑 5 秒 | PG `NOTIFY cancel_ticket, 'ASE-42'` |
| Agent 状态变更 | orchestrate → serve | 前端 SSE 需要即时推送，用户应立刻看到进度 | PG `NOTIFY agent_event, '{...}'` |
| Hook 执行结果 | orchestrate → serve | 前端应即时显示 Hook 通过/失败 | PG `NOTIFY hook_result, '{...}'` |
| 工单状态推进 | orchestrate → serve | 前端看板应即时更新 | PG `NOTIFY ticket_status, '{...}'` |

**所有非即时场景（占 90%+）完全靠数据库轮询，不需要通知机制：**

| 场景 | 说明 |
|------|------|
| 新工单出现 | 编排引擎下个 Tick（默认 5s）轮询 `SELECT ... WHERE status = 'todo'` 就看到了 |
| Workflow 变更 | serve 写入新的 Workflow / Skill 版本到 DB；后续 runtime 在创建时直接读取最新已发布版本并 materialize，不涉及额外进程通信或 control-plane sync |
| RepoScope 中的 PR 链接更新 | serve 写入数据库，前端和编排引擎自然读到；不依赖外部 Webhook |
| Agent 注册/配置 | serve 写入数据库，编排引擎下次 dispatch 时自然读到 Workflow -> Agent -> Provider 绑定与并发限制 |

**`all-in-one` 模式下更简单：** serve 和 orchestrate 在同一进程的不同 goroutine 中运行，上述即时通信全部用 Go channel 替代 PG NOTIFY——零序列化、零网络、零延迟。这也是为什么 `all-in-one` 是推荐的默认模式。

**EventProvider 抽象了这个差异：**

```go
// all-in-one 模式 → ChannelBus（Go channel）
// 分开部署模式 → PGNotifyBus（PostgreSQL LISTEN/NOTIFY）
// 业务代码只调用 EventProvider.Publish / Subscribe，不感知底层实现
```

### 5.4 `~/.openase/` — 用户大本营

OpenASE 在用户 Home 目录下维护一个 `~/.openase/` 目录，作为全局配置和敏感信息的存储中心：

```
~/.openase/
├── config.yaml              # 全局配置（数据库连接、监听端口、日志级别）
├── .env                     # 敏感环境变量（DB 密码、API Key 等），权限 0600
├── workspace/               # Ticket 工作区根目录
│   └── {org-slug}/
│       └── {project-slug}/
│           └── {ticket-identifier}/
│               ├── backend/
│               └── frontend/
└── logs/                    # 运行日志（journalctl 的补充，保留最近 7 天）
```

服务配置文件自动安装到系统标准位置：
- Linux: `~/.config/systemd/user/openase.service`
- macOS: `~/Library/LaunchAgents/com.openase.plist`

**敏感信息管理**：`~/.openase/.env` 只用于存放**本机 OpenASE 服务启动时需要读取的敏感环境变量**，例如数据库密码、本地 API 认证 Token、OIDC client secret、第三方 Provider API key、通知 webhook 等，文件权限 `0600`（仅 owner 可读写）。它**不是**“所有 Token 的统一落盘位置”：像 GitHub 出站凭证 `GH_TOKEN` 这类需要被平台统一托管、探测、按作用域解析并投影到本机 / 远端受控 session 的 Secret，必须存放在平台 Secret 存储层，而不是写入 `~/.openase/.env`。同时，Workflow Harness、Skills、绑定关系也不再以项目仓库 `.openase/` 目录作为权威源，而由平台控制面管理。

### 5.5 后端分层架构（DDD + Provider）

OpenASE 后端采用 DDD（Domain-Driven Design）四层架构，配合 Provider 模式处理横切关注点。核心原则：**依赖方向始终向内——外层依赖内层，内层不知道外层的存在。**

```
┌─────────────────────────────────────────────────────────────────┐
│                 Interface / Entry Layer (接口入口层)               │
│  cmd/openase  ·  internal/cli  ·  internal/httpapi              │
│  internal/webui  ·  internal/setup                              │
├─────────────────────────────────────────────────────────────────┤
│              Service / Use-Case Layer (服务 / 用例层)             │
│  internal/service/*  ·  internal/ticket  ·  internal/workflow   │
│  internal/chat  ·  internal/notification  ·  internal/agentplatform │
├────────────────────────────────────────────╥────────────────────┤
│        Domain / Core Types (领域 / 核心类型) ║   Provider (横切)  ║
│  internal/domain/*                         ║                    ║
│  internal/types/*                          ║  TraceProvider     ║
│                                            ║  MetricsProvider   ║
│                                            ║  EventProvider     ║
│  当前仓库以 parse / value object /          ║                    ║
│  pure logic 为主，不强制每个子包都有         ║  ExecutableResolver║
│  entity/repository/service/event 四件套      ║  AgentCLIProcessMgr║
│                                            ║                    ║
│                                            ║  UserServiceMgr    ║
│                                            ║  由 app/cmd 装配    ║
├────────────────────────────────────────────╨────────────────────┤
│                  Infrastructure Layer (基础设施层)                 │
│  internal/repo/     (DB-backed Repository / ent 仓储适配器)      │
│  internal/infra/    (Agent CLI / hook / SSE / workspace 等实现) │
│  internal/provider/ (Provider contracts + noop/default pieces)   │
└─────────────────────────────────────────────────────────────────┘
```

**各层职责：**

**Domain / Core Types（领域 / 核心类型层）**——当前仓库中主要承载领域解析、值对象、纯逻辑和少量稳定枚举映射。

- `internal/domain/catalog`、`internal/domain/ticketing`、`internal/domain/notification` 等：输入解析、值对象、纯业务规则、稳定的数据结构
- `internal/types/*`：底层领域类型和数据库边界类型
- 领域包对外暴露自己的稳定类型和枚举，不直接泄漏 `ent/*` 生成类型；数据库枚举与字段映射留在 `internal/repo/*`
- 不再假设每个领域子包都严格对应 `entity.go / repository.go / service.go / event.go` 四件套；以当前仓库真实职责为准

**Service / Use-Case Layer（服务 / 用例层）**——编排用例、衔接 repository/provider/domain，不再使用旧 PRD 中 `app/command`、`app/query` 的目录命名。

- 当前主要对应 `internal/service/*`、`internal/ticket`、`internal/workflow`、`internal/chat`、`internal/notification`、`internal/scheduledjob`、`internal/agentplatform`
- 这些包承担旧 PRD 中 application layer 的职责：编排完整用例、调用 domain 解析结果、访问 repository、驱动 provider
- repository port/interface 由消费它的上层包拥有；`internal/repo/*` 只提供 adapter 实现，不反向决定 service 的依赖形状
- 某些包会同时包含 command-style 写操作与 query-style 读操作，但以服务对象暴露，而不是按 `app/command`、`app/query` 目录拆分

**Infrastructure Layer（基础设施层）**——所有外部依赖的实现。

- `internal/repo/*`：数据库相关的 repository 适配器，当前仓库里这部分承担了旧 PRD `infra/persistence/` 的职责
- repository adapter 负责把 ent/client/database 细节映射到 domain 稳定类型，不把 persistence 类型直接传出
- `internal/infra/adapter/*`：各 Agent CLI 适配器实现（Claude Code、Codex 等）
- `internal/infra/hook`、`internal/infra/sse`、`internal/infra/workspace`、`internal/infra/event` 等：外部系统与运行时边界实现
- `internal/provider`：横切 Provider 接口与默认实现（见 5.6）

**Interface / Entry Layer（接口 / 入口层）**——外部入口，保持薄层。

- `cmd/openase`：CLI 入口，负责启动命令、参数装配和退出码处理
- `internal/httpapi`：Echo HTTP API handler、路由注册、请求绑定、错误映射、SSE/HTTP 入口；server/runtime 装配与 route/handler 注册保持拆分
- `internal/cli`：CLI 子命令与终端交互
- `internal/setup`、`internal/webui`：首次运行引导与 Web UI 入口

### 5.6 Provider 横切架构

Provider 是 OpenASE 处理横切关注点的统一模式。每个 Provider 定义为 Go interface，在 `cmd/openase` / `internal/app` 装配。任何层的代码都可以使用 Provider，但只依赖接口，不依赖实现。

```go
// 所有 Provider 接口定义在 internal/provider/ 中
package provider

// TraceProvider — 分布式追踪
type TraceProvider interface {
    ExtractHTTPContext(ctx context.Context, header http.Header) context.Context
    InjectHTTPHeaders(ctx context.Context, header http.Header)
    StartSpan(ctx context.Context, name string, opts ...SpanStartOption) (context.Context, Span)
    Shutdown(ctx context.Context) error
}

// MetricsProvider — 指标采集
type MetricsProvider interface {
    Counter(name string, tags Tags) Counter
    Histogram(name string, tags Tags) Histogram
    Gauge(name string, tags Tags) Gauge
}

// EventProvider — 进程间事件通信
type EventProvider interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(ctx context.Context, topics ...Topic) (<-chan Event, error)
    Close() error
}

// ExecutableResolver — 本地可执行文件定位
type ExecutableResolver interface {
    LookPath(name string) (string, error)
}

// AgentCLIProcessManager — Agent CLI 子进程管理
type AgentCLIProcessManager interface {
    Start(ctx context.Context, spec AgentCLIProcessSpec) (AgentCLIProcess, error)
}

// UserServiceManager — 平台相关用户服务管理
type UserServiceManager interface {
    Platform() string
    Apply(context.Context, UserServiceInstallSpec) error
    Down(context.Context, ServiceName) error
    Restart(context.Context, ServiceName) error
    Logs(context.Context, ServiceName, UserServiceLogsOptions) error
}
```

当前仓库里，认证边界主要落在 `internal/httpapi` 的安全设置入口；通知则不再通过一个统一的 `NotifyProvider` 暴露，而是由 `internal/notification` 的 channel adapter / rule engine 管理。

**Provider 实现矩阵：**

| Provider / Contract | 当前默认实现 | 可选实现 / 扩展点 | 注入位置 |
|----------|---------|---------|---------|
| `TraceProvider` | `internal/provider/noop_trace.go` | `internal/infra/otel/trace.go` | interface、service、runtime |
| `MetricsProvider` | `internal/provider/metrics.go`（noop） | `internal/infra/otel/metrics.go` | interface、service、runtime |
| `EventProvider` | `internal/infra/event/channel.go` | `internal/infra/event/pgnotify.go` | service、orchestrator、httpapi |
| `ExecutableResolver` | `internal/infra/executable/path.go` | 自定义 resolver | service |
| `AgentCLIProcessManager` | `internal/infra/agentcli/process.go` | fake / test manager | chat、adapter、orchestrator |
| `UserServiceManager` | `internal/infra/userservice/*.go` | 平台相关实现 | runtime / deploy |

**依赖注入（wiring）：** 当前主要在 `internal/app/app.go`、`cmd/openase/main.go` 和 CLI 子命令中完成组装。根据 `~/.openase/config.yaml` 的配置选择具体实现：

```go
// 伪代码
func buildProviders(cfg Config) Providers {
    var event provider.EventProvider
    if cfg.Event.Driver == "pgnotify" {
        event = pgnotify.New(cfg.Database.DSN)
    } else {
        event = channelbus.New() // all-in-one 默认 channel
    }

    trace := oteltrace.NewOrNoop(cfg.Observability)
    metrics := otelmetrics.NewOrNoop(cfg.Observability)
    resolver := executable.NewPathResolver()

    return Providers{Event: event, Trace: trace, Metrics: metrics, Resolver: resolver}
}
```

### 5.7 项目目录结构

```
openase/
├── cmd/openase/
│   └── main.go                  # CLI 入口
│
├── internal/
│   ├── app/                     # 启动入口与 runtime wiring
│   ├── domain/                  # 领域解析、纯逻辑、值对象
│   ├── types/                   # 底层领域类型 / DB 边界类型
│   ├── provider/                # 跨层 provider contracts
│   ├── repo/                    # ent-backed repository adapters
│   ├── service/                 # 典型 service/use-case 包
│   ├── ticket/                  # 工单 service/use-case
│   ├── workflow/                # workflow service/use-case
│   ├── chat/                    # chat service/use-case
│   ├── notification/            # notification service/use-case
│   ├── scheduledjob/            # scheduled job service/use-case
│   ├── agentplatform/           # agent platform service/use-case
│   ├── infra/                   # adapter / hook / ssh / workspace / event / otel 等实现
│   ├── httpapi/                 # Echo HTTP API、SSE、webhook、OpenAPI handler
│   ├── cli/                     # CLI 子命令
│   ├── orchestrator/            # 调度与运行编排
│   ├── runtime/                 # runtime 支撑（DB、观测）
│   ├── setup/                   # setup 向导
│   └── webui/                   # embed 的 Web UI handler
│
├── web/                         # Svelte 前端（构建后 embed）
├── go.mod
└── Dockerfile                   # 可选
```

项目 Git 仓库中的普通代码与脚本目录示意：

```
your-project/
├── scripts/
│   └── ci/                    # Hook 调用的脚本（仓库代码的一部分）
│       ├── run-tests.sh
│       ├── lint.sh
│       └── cleanup.sh
└── ...（项目代码）
```

> **废弃设计说明**：历史设计中，项目仓库根目录下的 `.openase/harnesses/`、`.openase/skills/` 被当作 Workflow / Skill 的权威源。该设计现已废弃。项目仓库可以完全不包含 `.openase/` 目录，OpenASE 仍可正常运行；Workflow / Skills 由平台控制面管理，并在 runtime 启动时 materialize 到工作区。

---

## 第六章 核心领域模型

### 6.1 实体关系总览

```
Organization → Project → ProjectRepo (1:N)
                ↓
              Ticket → TicketRepoScope (1:N) → ProjectRepo
                ↓
              Workflow (含 Hooks 配置), AgentProvider, Agent,
              ScheduledJob, ActivityEvent
```

一个 Project 关联多个 ProjectRepo（多仓库支持）。一个 Ticket 通过 TicketRepoScope 声明它涉及哪些 Repo，并记录每个 Repo 的工作分支与可选 PR 链接。Workflow 中内嵌 Hook 配置，定义工单各生命周期阶段的自动化检查。

**JSON 字段使用原则**：内容已知且会被查询过滤的字段用结构化列（如 `max_concurrent`、`auto_assign_workflow`）或 PostgreSQL 原生数组 `TEXT[]`（如 `labels`）。只有真正动态、形状不确定、不被查询的数据才用 JSONB（如 `Ticket.metadata`、`Workflow.hooks`、`AgentProvider.auth_config`、`ScheduledJob.ticket_template`、`ActivityEvent.metadata`、`AgentTraceEvent.payload`）。

### 6.2 Organization（组织）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | String | 组织名称 |
| slug | String | URL 友好标识 |
| status | String | `active` / `archived` |
| default_agent_provider_id | FK (nullable) | 默认 Agent Provider |

### 6.3 Project（项目）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| organization_id | FK | 所属组织 |
| name | String | 项目名称 |
| slug | String | URL 友好标识 |
| description | Text | 项目简介（Markdown） |
| status | String | 项目生命周期字符串；数据库不使用 enum。规范写入值仅允许 `Backlog` / `Planned` / `In Progress` / `Completed` / `Canceled` / `Archived` |
| default_agent_provider_id | FK | 默认 Agent Provider |
| max_concurrent_agents | Integer | 项目级最大并发 Agent 数（默认 5） |

**Project.status 规范：**

- 数据库层使用普通字符串列（如 `TEXT` / `VARCHAR`），不使用数据库 enum。
- 后端在写入边界负责解析与规范化；只有以下 6 个 canonical values 可以落库：
  - `Backlog`
  - `Planned`
  - `In Progress`
  - `Completed`
  - `Canceled`
  - `Archived`
- 禁止把任意自由文本直接写入数据库；约束由后端代码保证，而不是依赖数据库 enum。
- API 只接受上述 6 个精确字符串；任何别名、大小写变体、额外空白或历史值都必须返回 `400 Bad Request`。
- UI 必须只提交上述 canonical values，不负责做输入纠正或别名映射。

> **v3.2 变更**：`repository_url` 和 `default_branch` 从 Project 中移除，迁移至新增的 ProjectRepo 实体。一个 Project 可以关联多个 Repo。

> **v3.3 修订**：历史上的 `ProjectRepo.clone_path` 同时混用了“repo 运行时路径”和“Ticket 工作区中的目录名”两种语义，导致 Workflow、Workspace、Git 同步逻辑互相冲突。现将其收敛为两个明确对象：`ProjectRepo`（远端绑定）与 `TicketRepoWorkspace`（工单工作副本）。禁止再用单个字段同时表达远端地址、运行时本地路径、Ticket 工作区挂载路径。

### 6.4 ProjectRepo（项目仓库绑定）

一个 Project 可以关联多个 Git 仓库。`ProjectRepo` 是“项目与远端代码仓库之间的绑定关系”，它回答的是“这个项目涉及哪些 repo”，而不是“该 repo 当前在本机是否已经 clone”。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属项目 |
| name | String | 仓库别名（如 `backend`、`frontend`、`infra`），项目内唯一 |
| repository_url | String | 远端 Git 仓库地址；永远表示 remote truth，不得混写本地绝对路径 |
| default_branch | String | 默认分支（如 `main`） |
| workspace_dirname | String | 该 repo 在 Ticket 工作区中的一级目录名；默认等于 `name`，必须为相对路径片段 |
| labels | TEXT[] | 仓库标签（如 `{"go", "backend", "api"}`），PostgreSQL 原生数组，支持 GIN 索引 |

**职责边界：**

- `ProjectRepo` 只保存静态绑定信息，不保存“当前机器上的 clone 是否存在”这种运行时状态。
- `repository_url` 只表示远端地址；平台若需要本地工作路径，必须读取对应 `TicketRepoWorkspace.repo_path`。
- `workspace_dirname` 只表示 Ticket 工作区里的目录名；不得拿它当项目级镜像根目录。

**设计理由**：现实中一个产品通常是多仓库结构（前端、后端、SDK、基础设施各自独立仓库）。将 Repo 从 Project 中解耦后，一个工单可以声明它涉及哪些 Repo，编排引擎为 Agent 创建包含所有相关 Repo 的联合工作区。

**典型场景**：

- 一个“用户注册功能”工单需要同时修改 `backend`（API 接口）+ `frontend`（注册页面）+ `sdk`（类型定义）
- 一个“数据库迁移”工单只涉及 `backend` 仓库
- 一个“CI 流水线更新”工单只涉及 `infra` 仓库

### 6.4.1 TicketRepoWorkspace（工单工作副本）

`TicketRepoWorkspace` 表示某个 Ticket 在某次执行中的 repo 工作副本。它不是配置实体，而是运行时派生对象。每个工作副本都直接从远端仓库 clone / fetch 并 checkout 到本次运行所需的分支与基线。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| ticket_id | FK | 所属 Ticket |
| agent_run_id | FK | 所属 AgentRun |
| repo_id | FK | 关联的 ProjectRepo |
| workspace_root | String | 本次 Ticket 的工作区根目录 |
| repo_path | String | 该 repo 在工作区中的绝对路径 |
| branch_name | String | 当前工作分支 |
| state | Enum | `planned / materializing / ready / dirty / verifying / completed / failed / cleaning / cleaned` |
| head_commit | String | 当前工作副本 HEAD |
| last_error | Text | 最近一次 materialize / verify / cleanup 错误 |
| prepared_at | DateTime | 工作副本就绪时间 |
| cleaned_at | DateTime | 清理完成时间 |

**状态语义：**

- `planned`
  - 调度器已决定需要该 repo，但尚未开始准备工作副本。
- `materializing`
  - 平台正在 clone / fetch repo、checkout 分支、写入运行时上下文。
- `ready`
  - Agent 可安全进入 repo_path 工作。
- `dirty`
  - Agent 已产生未验证改动。
- `verifying`
  - 平台或 Hook 正在执行测试、lint、CI 聚合等检查。
- `completed`
  - 本次执行已完成，结果待清理。
- `failed`
  - 本次工作副本准备或验证失败。
- `cleaning`
  - 平台正在清理该工作副本。
- `cleaned`
  - 清理完成；保留记录用于审计。

**关键规则：**

- Ticket Workspace 是短生命周期对象。
- 同一个 Ticket 的多次重试 / continuation 默认复用同一个 `workspace_root`，但每个 repo 的工作副本状态仍独立跟踪。

### 6.4.2 Repo 生命周期总览

OpenASE 中一个 repo 只有两层必需形态，必须明确区分：

1. **远端绑定（ProjectRepo）**
   - 真相源：repo 地址、默认分支、标签
   - 生命周期长，基本只受用户配置操作影响
2. **工单工作副本（TicketRepoWorkspace）**
   - 真相源：本次 Ticket / AgentRun 的目录、分支、校验状态
   - 随执行创建，随执行清理

**平台必须显式管理以下事件：**

- `register_repo`
  - 创建 ProjectRepo 绑定；初始不保证本地可用
- `claim_ticket`
  - 直接从远端仓库准备相关 `TicketRepoWorkspace`
- `complete_ticket`
  - 对工作副本执行 verify / cleanup
- `delete_repo`
  - 先阻止新 run，再清理相关工作副本记录，最后删除绑定

**代码基线策略：**

- 默认执行路径是：在创建 `TicketRepoWorkspace` 时直接从远端仓库 fetch / checkout 最新基线。
- 平台不维护项目级代码缓存层，也不提供缓存注册、同步、健康检查、路径推导这类能力。
- “最新代码”语义只由远端 Git 仓库与本次工作副本的 fetch / checkout 决定，不再引入额外中间层状态机。

### 6.5 Ticket（工单）—— 核心实体

工单是整个系统的绝对核心。每一个工作单元都是一个 Ticket。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属项目 |
| identifier | String | 可读标识（如 ASE-42），自动生成 |
| title | String | 工单标题 |
| description | Text | 工单描述（Markdown） |
| status_id | FK | 关联 TicketStatus（用户可见的自定义状态；生命周期语义由该状态的 `stage` 提供） |
| priority | Enum | urgent / high / medium / low |
| type | Enum | feature / bugfix / refactor / chore / epic |
| workflow_id | FK | 分配的 Workflow |
| current_run_id | FK | 当前活跃的 AgentRun；为空表示未被领取 |
| created_by | String | 创建者 |
| parent_ticket_id | FK | 父工单（sub-issue 关系） |
| external_ref | String | 外部关联（GitHub Issue ID 等） |
| attempt_count | Integer | 累计尝试次数 |
| consecutive_errors | Integer | 连续失败次数（成功或人类介入后重置为 0） |
| next_retry_at | DateTime | 下次重试时间（指数退避计算） |
| retry_paused | Boolean | 是否暂停重试（预算耗尽 / 人类暂停） |
| pause_reason | String | 暂停原因（budget_exhausted / user_paused） |
| stall_count | Integer | Stall 次数 |
| retry_token | String | 重试令牌（防过期重试） |
| harness_version | Integer | 执行时使用的 Harness 版本号 |
| budget_usd | Decimal | 单工单预算上限 |
| cost_tokens_input | BigInt | 输入 Token 数 |
| cost_tokens_output | BigInt | 输出 Token 数 |
| cost_amount | Decimal | 执行成本金额 |
| metadata | JSON | 扩展字段 |
| started_at | DateTime | 开始执行时间 |
| completed_at | DateTime | 完成时间 |

**工单依赖关系：**

| 关系类型 | 说明 | 行为 |
|---------|------|------|
| blocks | A blocks B：A 完成前 B 不能开始 | B 不参与调度，直到 A 当前状态对应的 `stage` 进入 `completed` 或 `canceled` |
| sub-issue | A 是 B 的子工单 | A 完成后自动检查 B 的所有子工单是否完成 |

依赖关系在后端只持久化 `blocks` / `sub_issue` 两种结构化边，不单独存储 `blocked_by`。`blocked_by` 只是 `blocks` 的反向阅读语义，用于 UI 表达，而不是额外的数据模型。

**工单外部链接（TicketExternalLink）：**

一个工单可以关联多个外部 Issue（GitHub Issue、GitLab Issue、Jira Ticket 等）和多个 PR。`external_ref` 字段保留为主关联的快捷方式，`TicketExternalLink` 则支持 1:N 的完整关联。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| ticket_id | FK | 所属工单 |
| link_type | Enum | github_issue / gitlab_issue / jira_ticket / github_pr / gitlab_mr / custom |
| url | String | 外部链接 URL（如 `https://github.com/org/repo/issues/42`） |
| external_id | String | 外部系统中的标识（如 `org/repo#42`） |
| title | String | 外部 Issue/PR 的标题（可选缓存，只用于展示） |
| relation | Enum | resolves / related / caused_by |
| created_at | DateTime | 关联创建时间 |

**使用场景：**

- 一个"修复登录 Bug"工单可以同时关联 GitHub Issue #42（Bug 报告）和 #45（相关讨论），以及自动创建的 PR
- Agent 在 Harness Prompt 中能看到所有关联的 Issue 内容（描述、评论），帮助理解上下文
- OpenASE 不自动同步这些外部链接的状态；它们只作为上下文引用与跳转入口

**Harness 模板变量**：

```
{{ range .ExternalLinks }}
- [{{ link.type }}] {{ link.title }}: {{ link.url }}
{{ end }}
```

### 6.5.1 TicketComment（工单评论）

`TicketComment` 是 Ticket Detail 时间线中的人类讨论项。它承载 handoff、review、决策说明、补充上下文，不与系统 Activity 混写。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| ticket_id | FK | 所属工单 |
| author_type | String | `user` / `agent` / `system_proxy` |
| author_id | String | 作者标识（用户 ID、Agent ID 或外部同步标识） |
| author_name | String | 时间线展示名 |
| body_markdown | Text | 当前生效的 Markdown 正文 |
| is_deleted | Boolean | 软删除标记 |
| deleted_at | DateTime | 删除时间 |
| deleted_by | String | 删除操作者 |
| created_at | DateTime | 创建时间 |
| updated_at | DateTime | 最近更新时间 |
| edited_at | DateTime | 最近一次正文编辑时间；未编辑时为空 |
| edit_count | Integer | 正文被保存的次数，不含初始创建 |
| last_edited_by | String | 最近一次编辑者 |

**规则：**

- `TicketComment` 代表“当前版本”；历史版本进入 `TicketCommentRevision`。
- UI 显示 `edited` 时，必须基于 `edited_at != nil` 或 `edit_count > 0`，不能靠字符串猜测。
- 删除采用软删除，默认时间线不展示正文，替换为 `Comment deleted` 占位；历史审计仍可保留。
- 当前阶段不要求 reaction、thread reply、resolved thread，但数据模型不得阻止未来扩展。

### 6.5.2 TicketCommentRevision（评论历史版本）

用户在 Ticket Detail 中可以查看评论编辑历史。为保证这一点，评论正文的每次保存必须写入版本历史。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| comment_id | FK | 关联 TicketComment |
| revision_number | Integer | 从 1 开始递增；`1` 为初始发布版本 |
| body_markdown | Text | 该版本的正文快照 |
| edited_by | String | 本次版本提交者 |
| edited_at | DateTime | 本次版本生成时间 |
| edit_reason | String | 可选编辑说明；当前阶段可为空 |

**规则：**

- 创建评论时必须同时写入 `revision_number=1` 的初始版本。
- 每次正文保存都必须追加一条新 `TicketCommentRevision`，禁止原地覆盖历史。
- 时间线默认只展示 `TicketComment.body_markdown` 当前版本；“History” 面板读取 `TicketCommentRevision`。
- `TicketComment.edit_count = revisions - 1`，该字段是读取优化，不是真相源。

### 6.5.3 TicketTimelineItem（详情页时间线投影）

Ticket Detail 的主视图不应让前端自己把 `Ticket`、`TicketComment`、`ActivityEvent` 临时拼装成时间线。后端必须提供统一的时间线投影对象 `TicketTimelineItem`。

`TicketTimelineItem` 是**读取模型 / API 投影**，不要求独立持久化表，但必须有稳定 schema。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | String | 稳定唯一标识（如 `description:{ticketId}`、`comment:{commentId}`、`activity:{activityId}`） |
| ticket_id | FK | 所属工单 |
| item_type | String | `description` / `comment` / `activity` |
| actor_name | String | 展示用操作者名称 |
| actor_type | String | `user` / `agent` / `system` |
| title | String | activity 标题或 description 标题；comment 可为空 |
| body_markdown | Text | description / comment 正文；activity 可为空 |
| body_text | Text | activity 的纯文本摘要；description / comment 可为空 |
| created_at | DateTime | 条目时间 |
| updated_at | DateTime | 条目最近更新时间 |
| edited_at | DateTime | comment / description 编辑时间 |
| is_collapsible | Boolean | UI 是否允许折叠 |
| is_deleted | Boolean | comment 删除占位 |
| metadata | JSON | activity 图标、状态变化、历史版本计数、链接等附加信息 |

**时间线规则：**

- Ticket description 必须作为时间线的第一个固定条目显示在最前面，语义上等价于“作者 opened this ticket”。
- description 之后的条目按 `created_at` 正序排列，新的 comment / activity 追加在底部。
- comment 与 activity 必须共用一条时间线，但视觉样式可以不同；不能再做成两个彼此割裂的主面板。
- activity 条目不可编辑删除；comment 条目支持 edit / delete / history / collapse。
- 前端不得自己推断 comment 与 activity 的混排顺序；必须消费后端返回的 `TicketTimelineItem[]`。

**工单仓库作用域（TicketRepoScope）：**

一个工单可以涉及 Project 下的一个或多个 Repo。TicketRepoScope 记录了每个工单在每个相关 Repo 中的仓库绑定、工作分支以及可选的 PR 链接。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| ticket_id | FK | 所属工单 |
| repo_id | FK | 关联的 ProjectRepo |
| branch_name | String | 在该 Repo 中创建的分支 |
| pull_request_url | String | 在该 Repo 中创建的 PR 地址 |

**工单状态与 RepoScope PR 链接的关系：**

- `pull_request_url` 只是引用信息，不是状态机输入
- 工单是否进入 `in_review`、`done`、`canceled` 等状态，始终由 Agent 的显式平台操作或人类在 UI/API 中的显式状态修改决定
- OpenASE 不同步 PR 状态，也不同步 CI 状态；RepoScope 不承载 `pr_status` / `ci_status`

**编排引擎的多 Repo 工作区策略：**

当工单涉及多个 Repo 时，编排引擎创建一个联合工作区（workspace），并为每个 repo 直接准备对应的 `TicketRepoWorkspace`。目录结构如下：

```
~/.openase/workspace/{org-slug}/{project-slug}/{ticket-identifier}/
├── backend/        # clone of backend repo, checked out to feature branch
├── frontend/       # clone of frontend repo, checked out to feature branch
└── sdk/            # clone of sdk repo, checked out to feature branch
```

Agent 的 Harness Prompt 中会注入所有 Repo 的路径和说明：

```
你正在处理工单 ASE-42，该工单涉及以下仓库：
- backend (Go API): ~/.openase/workspace/acme/payments/ASE-42/backend/
- frontend (SvelteKit): ~/.openase/workspace/acme/payments/ASE-42/frontend/
- sdk (TypeScript): ~/.openase/workspace/acme/payments/ASE-42/sdk/

请在所有相关仓库中完成必要的修改，并为每个仓库创建独立的 PR。
```

**分支命名规范**：每个 Repo 的默认工作分支统一命名为 `agent/{ticket-identifier}`。分支归 Ticket 所有，而不是归某个 Agent 所有；Agent 只是执行者，可以在不改分支名的前提下接手其他 Agent 的未完成工作。若 TicketRepoScope 已显式记录 `branch_name`，运行时应直接复用该分支，而不是因为 Agent 变更而判定 branch 失效。

**Ticket Workspace 约定：**

- 工作区路径不是用户随意填写的自由字段，而是由平台统一推导
- 默认规则：
  - 本机：`~/.openase/workspace/{org-slug}/{project-slug}/{ticket-identifier}`
  - 远端机器：`{machine.workspace_root}/{org-slug}/{project-slug}/{ticket-identifier}`
- 一个 Ticket 对应一个独立工作目录
- 工作目录下一级目录为该 Ticket 涉及的多个 Repo
- 同一个 Ticket 的多次重试 / continuation 复用同一个 Ticket 工作目录
- Ticket 工作目录是执行态副本，不再存在项目级代码缓存这一中间层
- Ticket Workspace 的代码基线直接来自远端 Git 仓库，工作副本内部的 `git fetch origin` 就是对远端仓库取最新

### 6.6 Workflow（工作流）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属项目 |
| name | String | Workflow 名称 |
| type | Enum | coding / test / doc / security / deploy / refine-harness / custom |
| agent_id | FK | 绑定的 Agent 定义；该 Workflow 每次执行固定采用此 Agent 驱动 |
| harness_key | String | Workflow Harness 的稳定逻辑标识（如 `coding-default`），项目内唯一 |
| current_version_id | FK | 当前发布中的 Harness 版本 |
| hooks | JSONB | 生命周期 Hook 配置（嵌套结构：hook_name → [{cmd, timeout, on_failure}]，详见第八章） |
| pickup_status_ids | NonEmptySet<FK → TicketStatus.id> | Workflow 可从这些状态领取工单；至少 1 个，必须全部属于同一项目 |
| finish_status_ids | NonEmptySet<FK → TicketStatus.id> | Workflow 完成后允许落到这些状态；至少 1 个，必须全部属于同一项目 |
| max_concurrent | Integer | 该 Workflow 的最大并发数（默认 3） |
| max_retry_attempts | Integer | 最大重试次数（默认 3） |
| timeout_minutes | Integer | 单工单超时分钟数（默认 60） |
| stall_timeout_minutes | Integer | Agent 无事件超时分钟数（默认 5） |
| version | Integer | 版本号 |
| is_active | Boolean | 是否启用 |

说明：

- Harness 正文、版本历史、审计记录由平台控制面持久化；不再以 Git 仓库中的 `.openase/harnesses/*` 文件作为权威源。
- Workflow 编辑、发布、技能绑定变更一律通过 Platform API 完成；Agent 不允许直接修改 repo 工作区中的文件来改变平台控制面行为。
- 任何读取 / 编辑 Workflow 与 Skill 的控制面操作，都不依赖 repo 工作区是否已存在。代码仓库 checkout 只影响代码执行路径，不影响 Workflow / Skill 的查看、编辑、版本化与绑定。
- `pickup_status_ids / finish_status_ids` 属于结构化 Workflow 元数据，存储在数据库中，由前端配置并受数据库引用约束维护。
- `pickup_status_ids / finish_status_ids` 可以引用同一项目内的任意 TicketStatus；平台只校验集合非空、引用存在且属于当前项目，不再按 `stage` 限制绑定集合。
- 编排引擎只读取数据库中的状态绑定进行调度；Harness 中不再作为真相源重复声明这些状态。
- AgentRun 启动时，平台把 `current_version_id` 指向的 Harness 版本与当前绑定的 Skills 版本 materialize 到该次运行的 workspace。新 runtime 默认使用最新发布版本；已在运行的 runtime 不自动热更新，除非显式 refresh / restart。

**废弃字段与兼容说明：**

- 历史设计中的 `harness_path` 作为 Git 仓库文件路径的语义已废弃。
- 兼容迁移期可以保留旧字段做只读映射或导出用途，但它不得再作为 Workflow 内容定位、版本真相源或写入目标。

### 6.7 AgentProvider（Agent 提供商）

AgentProvider 表示**某台 Machine 上一个可被 OpenASE 调用的 Coding Agent CLI 入口**。它回答的是：

- 这台机器上安装的是哪种外部 Coding Agent CLI
- OpenASE 应该如何启动它
- 它在该机器上的登录态 / 环境变量 / 并发上限是什么

因此，Provider 不是“抽象工具族”那么宽，而是**机器绑定的可执行入口**。即使两台机器都安装了 Codex CLI，也应建成两个独立 Provider，因为它们的路径、认证状态、环境和可用并发都可能不同。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| organization_id | FK | 所属组织 |
| machine_id | FK | 绑定的 Machine；该 Provider 只在这台机器上可运行 |
| name | String | 名称（如 Codex、Claude Code、Gemini CLI） |
| adapter_type | Enum | claude-code-cli / codex-app-server / gemini-cli / custom |
| cli_command | String | 启动命令（如 `claude`、`codex`） |
| cli_args | TEXT[] | CLI 启动参数数组（如 `{"--max-turns", "20", "--verbose"}`） |
| auth_config | JSONB (encrypted) | 加密认证信息（不同 Provider 结构不同，真正的多态数据） |
| model_name | String | 模型名称（如 `claude-sonnet-4-6`、`gpt-5.3-codex`） |
| model_temperature | Float | 模型温度（默认 0.0） |
| model_max_tokens | Integer | 最大输出 Token 数（默认 16384） |
| max_parallel_runs | Integer | Provider 级最大并发运行数（provider semaphore） |
| cost_per_input_token | Decimal | 输入 Token 单价 |
| cost_per_output_token | Decimal | 输出 Token 单价 |

**GitHub 出站凭证统一约定：**

- OpenASE 对 GitHub 的所有**出站**能力统一复用一份平台托管的 `GH_TOKEN`，包括：
  - GitHub 仓库 `clone / fetch / push`（统一使用 `https://github.com/...git` 传输）
  - GitHub Pull Request / Project API 调用（如果某个 Workflow 或平台操作显式需要）
- `GH_TOKEN` 是平台统一管理的加密 Secret，不允许分散存放在 ProjectRepo、脚本、用户 shell profile 或 repo 文件中。
- `GH_TOKEN` 的来源支持三种：
  - 平台发起 GitHub Device Flow 授权
  - 导入当前用户已有的 `gh auth token`
  - 用户手动粘贴 Personal Access Token 交由平台托管
**归属范围（Scope of Ownership）：**

- 默认归属为**Organization 级**：同一 Organization 下的 ProjectRepo 与 GitHub PR 自动化默认共享一份 `GH_TOKEN`。
- Project 可以显式配置自己的 GitHub 出站凭证覆盖 Organization 默认值；未覆盖时一律回退到 Organization 级 `GH_TOKEN`。
- 不定义“Machine 级真相源”的 GitHub token。Machine 上的 `gh auth status`、credential helper、SSH key 只作为观测或兼容信号，不作为平台鉴权配置源。
- 一个 Project 在同一时刻只允许解析出**一份有效 GitHub 出站凭证**，避免 clone / pr 等路径分别命中不同 token。

**存储模型（Storage Model）：**

- `GH_TOKEN` 必须存放在平台 Secret 存储层中，作为**加密静态配置**持久化；不得明文落在 ProjectRepo、Machine、Workflow、Provider、脚本仓库文件或用户 shell profile 中。
- UI 默认只展示：
  - 来源（`device_flow` / `gh_cli_import` / `manual_paste`）
  - 归属范围（organization / project）
  - 最近探测时间
  - 权限探测结果
  - 最近错误
- UI 不回显完整 token；最多只显示固定格式的脱敏尾部，例如 `ghu_xxx...ABCD`。
- 导出配置、诊断日志、ActivityEvent、AgentTraceEvent、Hook 输出中必须对 `GH_TOKEN` 做统一 redaction。

**来源语义（Source Semantics）：**

- `device_flow`：平台自己完成 GitHub 授权流程，最终保存一份平台托管 token。在 OAuth App / Device Flow wiring 落地前，UI 可显式展示为 deferred，而不是伪装成已可用入口。
- `gh_cli_import`：平台在导入瞬间读取 `gh auth token` 的当前值并复制保存；导入完成后，该 token 与本机 `gh` 登录态**解耦**。
- `manual_paste`：用户显式粘贴 token，由平台保存。
- 一旦 token 已被平台保存，后续所有调度与运行时行为都读取平台 Secret 存储，而不是再次依赖执行机上的 `gh auth token` 命令输出。

**边界与观测规则：**

- `gh auth status` 仅表示某台机器上的 GitHub CLI 登录态，是观测信号，不是平台 GitHub 出站鉴权的真相源。
- 本机 `go-git` 路径与远端 shell `git` 路径都必须显式消费平台托管的 `GH_TOKEN`；不得依赖“机器可能已经配置好 credential helper”的隐式前提。
- 远端运行时只能把 `GH_TOKEN` 作为当前受控 session 的环境变量注入，不能写入用户全局 shell profile、仓库文件或工作区持久配置。
- 平台保存 / 导入 `GH_TOKEN` 后，必须立即执行一次结构化权限探测，产出 `valid / permissions / repo_access / checked_at` 等结果；UI 与调度器读取的是探测结果，而不是“字符串非空”。

**生命周期状态机（Lifecycle）：**

- `missing`
  - 当前 Organization / Project 未配置 GitHub 出站凭证。
- `configured`
  - token 已保存，但尚未完成首次探测。
- `probing`
  - 平台正在执行权限 / repo access 探测。
- `valid`
  - token 有效，且满足当前已启用 GitHub 能力所需最小权限。
- `insufficient_permissions`
  - token 有效，但缺少当前已启用能力所需权限。
- `revoked`
  - GitHub 返回 401 / token 无效 / token 已被撤销。
- `error`
  - 平台无法完成探测（网络异常、GitHub API 暂时失败、配置不一致等）。

**状态迁移规则：**

- 新建 / 导入 token：`missing -> configured -> probing -> valid | insufficient_permissions | revoked | error`
- 手动“重新测试”或定时 probe：`valid | insufficient_permissions | error -> probing -> ...`
- 用户删除 token：任意状态 -> `missing`
- 用户旋转 token：旧 token 立即退出活动配置，新 token 重新走 `configured -> probing -> ...`

**运行时投影规则（Runtime Projection）：**

- 本机工作区管理：
  - `go-git` clone / fetch / push 必须显式使用平台解析后的 GitHub transport auth；不能假设库会自动读取 `GH_TOKEN`。
- 远端工作区管理：
  - shell `git clone / fetch / push`
  - `gh issue / pr / project`
  以上命令只允许通过**当前受控 session 的临时环境变量**读取 `GH_TOKEN`。
- Agent CLI 本身默认**不继承** `GH_TOKEN`，除非当前步骤明确需要调用 GitHub API / git transport 且平台显式投影给该子进程。
- `GH_TOKEN` 不得写入：
  - `.git/config`
  - `.env`
  - workspace 文件
  - shell profile
  - hook 脚本模板

**能力矩阵（Capability Matrix）：**

- 仅使用 GitHub 私有仓库 clone / fetch / push：
  - 要求 repo transport 可用
  - 不要求 Issue / PR / Project API 权限探测通过
- 使用 GitHub Issue / PR 自动化：
  - 要求 Issue / PR API 权限可用
- 使用 GitHub Project：
  - 额外要求 Project 权限可用
- 调度器在判定“GitHub 能力可用”时，必须按**当前启用功能**解析，不得把所有场景都提升为最大权限要求

**最小权限要求：**

- Classic PAT / gh OAuth scope：
  - `repo`
  - `project`（仅当 OpenASE 需要创建 / 更新 GitHub Project 时）
- Fine-grained PAT：
  - Repository `Contents: write`
  - Repository `Pull requests: write`
  - Repository `Issues: write`
  - Repository `Metadata: read`
  - `Projects: write`（仅当 OpenASE 需要创建 / 更新 GitHub Project 时）

**轮换与撤销（Rotation & Revocation）：**

- 用户替换 token 后：
  - 新任务只使用新 token
  - 正在运行的 session 不要求热替换，但不得把旧 token 再次持久化或回写到任何机器
- 如果 probe 判定 token 已撤销：
  - 平台将状态标记为 `revoked`
  - 新的 GitHub clone / pr 相关任务立即阻止启动
  - 已经开始的任务允许失败收敛，但错误中必须说明是 GitHub 出站凭证失效
- 定时 probe 建议周期：
  - 成功态：30 分钟到 6 小时之间，按环境等级配置
  - 错误态：指数退避，避免持续刷 GitHub API

**重要：`AgentProvider` 的“可用性”不是静态配置字段，而是运行时派生状态。**

- `cli_command`、`cli_args`、`auth_config`、`machine_id` 等属于**静态配置**
- Provider 当前能不能被调度执行，属于**运行时派生结果**
- 前端和调度器**不得**把“命令存在于 PATH 中”或“配置字段非空”直接解释为“Provider 可用”

Provider 对外暴露以下派生字段（可通过 API response 返回，不要求持久化为配置列）：

| 字段 | 类型 | 说明 |
|------|------|------|
| availability_state | Enum | `unknown / available / unavailable / stale` |
| available | Boolean | 兼容布尔字段；等价于 `availability_state == available` |
| availability_checked_at | DateTime (nullable) | 最近一次用于判定 Provider 可用性的 L4 检查时间 |
| availability_reason | String (nullable) | 不可用或过期原因（如 `machine_offline`、`cli_missing`、`not_logged_in`、`stale_l4_snapshot`） |

**Provider 可用性的判定规则：**

- `available`
  - `machine.status == online`
  - 最近一次 L4 Agent Environment 检查成功且未过期
  - 对应 adapter 的 CLI 已安装
  - 对应 CLI 认证状态已就绪（如 `logged_in` 或 API key mode）
  - Provider 所需启动配置完整（命令、路径、远端工作区等）
- `unavailable`
  - 最近一次 L4 检查明确证明该 Provider 不可运行
  - 或绑定 Machine 不处于 `online`
- `unknown`
  - 从未完成过可信的 L4 检查，系统尚无足够信息
- `stale`
  - 曾经有 L4 成功快照，但该快照已超过有效期，不能继续作为调度依据

**默认过期窗口：**

- L4 检查周期默认 30 分钟
- `availability_state` 在 `now - availability_checked_at > 2 * L4_interval` 时转为 `stale`
- `stale` 与 `unknown` 一样，均不得参与调度

### 6.8 Agent（执行定义）

这里的 Agent 不是“调度池里的一个可占用 worker”，而是“某个 Provider 在当前项目中的运行方式定义”。它回答的是：

- 用哪个 Provider
- 绑定的是哪台机器上的哪个 Coding Agent CLI
- 这个角色叫什么名字
- 这个角色有哪些静态标签 / 默认配置

Workflow 绑定 Agent，表示“这个 Workflow 执行时采用哪个 Agent 定义来驱动”。真正瞬时的运行中状态不挂在 Agent 本体上，而挂在独立的运行记录上。

**关键澄清：Agent 不是“一次只能绑定一个 Ticket”的单实例。** 同一个 Agent 定义在不同时间可以驱动多个 Ticket，也可以在并发限制允许时同时产生多个 `AgentRun`。真正与 Ticket 一一对应的是 `AgentRun`，不是 Agent 本体。

- 一个 Ticket 在任一时刻最多只有一个 `current_run_id`
- 但同一个 Agent 定义在任一时刻可以同时对应多个 Ticket 的多个 `AgentRun`
- 是否允许这些 `AgentRun` 并行，取决于 global / provider / stage / workflow 并发约束，而不是“Agent 只能单线程”

为便于目录页和 API 快速概览，Agent 定义可以额外暴露只读的 `runtime` 聚合摘要，例如活跃运行数、汇总状态、最近心跳。但这只是 convenience summary，不是运行时真相视图：

- 当同一个 Agent 定义存在多个并发 `AgentRun` 时，不得把该摘要伪装成单一 `current_run_id` / `current_ticket_id`
- 完整的运行时可观测面必须以 `AgentRun` 列表或详情为准
- 任何 Agent 级 `runtime` 字段都必须明确表达“summary / aggregate”语义

**Agent 不再保存用户手填的 `workspace_path`。** 工作目录统一由平台按 Ticket 维度推导：

- 本机 Provider：`~/.openase/workspace/{org-slug}/{project-slug}/{ticket-identifier}`
- 远端 Provider：`{machine.workspace_root}/{org-slug}/{project-slug}/{ticket-identifier}`

目录下的一级子目录是该 Ticket 涉及的多个 Repo。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| provider_id | FK | 所属 Provider |
| project_id | FK | 关联 Project |
| name | String | Agent 定义名称（如 `codex-coding`、`claude-reviewer`） |
| is_enabled | Boolean | 是否允许 Workflow 继续绑定和运行 |
| total_tokens_used | BigInt | 归因到该 Agent 定义的累计 Token 消耗 |
| total_tickets_completed | Integer | 归因到该 Agent 定义的累计完成工单数 |

### 6.8.1 AgentRun（运行会话）

AgentRun 是真正的运行时占位。每次 Workflow 执行都会创建一个新的 AgentRun，并由编排引擎托管其生命周期。它才是 semaphore 和运行时健康检查的直接对象。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| agent_id | FK | 关联的 Agent 定义 |
| workflow_id | FK | 本次运行采用的 Workflow |
| ticket_id | FK | 本次运行服务的工单 |
| provider_id | FK | 冗余记录，便于并发统计和审计 |
| status | Enum | launching / ready / executing / completed / errored / terminated |
| session_id | String | Runtime 建立成功后的只读会话 / thread ID |
| runtime_started_at | DateTime | Runtime 成功启动时间 |
| last_error | String | 最近一次 Runtime 启动 / 会话错误（健康时为空） |
| last_heartbeat_at | DateTime | Runtime 托管的最后心跳时间 |
| current_step_status | String (nullable) | 当前人类可理解的动作阶段（如 `planning` / `editing` / `running_tests` / `opening_pr`） |
| current_step_summary | String (nullable) | 当前动作阶段的人类可读摘要 |
| current_step_changed_at | DateTime (nullable) | 最近一次动作阶段切换时间 |

**Coding Agent Runtime Readiness Contract**

- `launching` 只表示调度器已经创建了本次运行，不等于 Codex 已经启动成功。
- 确定性的启动成功条件是：`status == ready || status == executing`、`session_id != ""`、`last_heartbeat_at` 已填充且足够新。
- Catalog CRUD 不允许用户手工写入 `session_id`、`runtime_started_at`、`last_error`、`last_heartbeat_at`；这些字段只能由 Runtime 启动路径写入。
- `current_step_status` 与 `status` 不是一回事：前者表示“Agent 现在在做什么”，后者表示“Runtime 是否已启动 / 执行 / 失败”。
- 前端必须使用这些 Runtime 字段展示 `waiting -> launching -> ready -> failed`，不能把“没有 activity 文本”解释成启动失败。

### 6.9 Manual Review Hold（人工审核挂起）

需要人工确认的工单不再创建单独的 `ApprovalGate` 实体。进入人工审核态必须来自 Agent 或人类的显式状态更新；Runtime 在普通 turn 结束后只负责停止或续跑，不自动把工单移到 `in_review` / `awaiting_review`。

### 6.10 ScheduledJob（定时任务）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属 Project |
| name | String | 任务名称 |
| cron_expression | String | Cron 表达式 |
| workflow_id | FK | 使用的 Workflow |
| ticket_template | JSON | 工单模板 |
| is_enabled | Boolean | 是否启用 |
| last_run_at | DateTime | 上次执行 |
| next_run_at | DateTime | 下次执行 |

### 6.11 AgentTraceEvent（Agent 细粒度运行轨迹）

Agent CLI 在运行时产生的大量碎片化输出，不应直接污染业务活动流。OpenASE 将这类细粒度运行信号统一归一为 `AgentEvent` 协议，并持久化为 `AgentTraceEvent`。

这一层是**排障与实时观察层**，不是项目活动流。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属 Project |
| ticket_id | FK | 关联工单 |
| agent_id | FK | 关联 Agent |
| agent_run_id | FK | 关联 AgentRun |
| sequence | BigInt | 同一 AgentRun 内的严格单调序号 |
| provider | String | 来源 Provider（codex / claude / gemini / custom） |
| kind | String | 归一化事件类型（如 `assistant_delta`、`assistant_snapshot`、`tool_call_started`、`tool_call_finished`、`command_output_delta`、`runtime_notice`、`error`） |
| stream | String | 逻辑流名称（assistant / tool / command / system） |
| text | Text | 可直接展示的文本片段；允许为空 |
| payload | JSON | Provider 特有的附加信息（tool 参数、item id、phase、原始 stream metadata 等） |
| created_at | DateTime | 事件时间 |

**约束：**

- `AgentTraceEvent` 只用于 Agent 控制台、调试面板、重放运行细节，不进入 Dashboard 的 Activity Feed。
- `delta` 与 `snapshot` 都可以持久化，但必须通过 `sequence` 保证同一 `AgentRun` 内的顺序稳定。
- 任何 Provider 的原始事件都先映射到统一 `AgentEvent`，再写入 `AgentTraceEvent`；前端不得直接解析某个 CLI 的私有协议。

### 6.11.1 AgentStepEvent（人类可读动作流）

并不是每一个 token、每一次输出增量都适合人类看。OpenASE 需要从 `AgentTraceEvent` 中抽取“动作阶段切换”，形成更稳定的人类可读流水。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属 Project |
| ticket_id | FK | 关联工单 |
| agent_id | FK | 关联 Agent |
| agent_run_id | FK | 关联 AgentRun |
| step_status | String | 动作阶段（如 `planning` / `editing` / `running_tests` / `opening_pr` / `waiting_review`） |
| summary | Text | 人类可读摘要（如“分析仓库结构”“运行前端 CI”“准备创建 PR”） |
| source_trace_event_id | FK (nullable) | 触发该阶段变化的原始 TraceEvent |
| created_at | DateTime | 阶段切换时间 |

**规则：**

- 只有 `step_status` 发生变化时，才追加一条 `AgentStepEvent`。
- `AgentRun.current_step_status / current_step_summary / current_step_changed_at` 是当前快照；`AgentStepEvent` 是历史时间线。
- `AgentStepEvent` 是 Agent 详情页默认展示的主时间线，不等于业务级 Activity。

### 6.11.2 ActivityEvent（业务活动事件）

`ActivityEvent` 只记录项目、工单、编排层真正重要的业务事件。它是 Dashboard、项目 Activity 页、Ticket System Activity 的唯一活动信源。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属 Project |
| ticket_id | FK | 关联工单（可为空） |
| agent_id | FK | 关联 Agent（可为空） |
| event_type | String | 事件类型 |
| message | Text | 事件描述 |
| metadata | JSON | 额外数据 |
| created_at | DateTime | 事件时间 |

**`event_type` 命名规则（严格）**

- `event_type` 虽然物理存储为 `String`，但逻辑上必须视为**受控枚举**；写入数据库、通过 API 返回、通过 SSE 推送时都必须使用同一份 canonical catalog
- 命名格式统一为：`lowercase.dot.separated`
- 不允许同义别名混用；例如不得同时出现 `status_changed` / `ticket.status_changed`、`pr_opened` / `pr.opened`、`hook_failed` / `hook.failed`
- UI 可以把 canonical 值 humanize 成更友好的标签，但**不得**改写 wire/storage 层的真实值
- repository / service / projector 边界必须解析为领域枚举；未知类型写入必须报错，读取到未知历史值必须显式归类为 `unknown` 并带诊断日志，不能静默猜测

**Canonical ActivityEvent Catalog**

`ActivityEvent` 必须保持 coarse-grained。允许进入这一层的 `event_type` 只有下表这些值：

| event_type | 含义 | 适用范围 | 必备 metadata |
|-----------|------|----------|---------------|
| `ticket.created` | 工单首次创建 | ticket 必填 | 可选：`identifier`、`title`、`created_by` |
| `ticket.updated` | 工单业务可见字段被修改，但不涉及状态迁移 | ticket 必填 | `changed_fields[]`；可选：`updated_by` |
| `ticket.status_changed` | 工单状态发生迁移 | ticket 必填 | `from_status_id`、`from_status_name`、`to_status_id`、`to_status_name`；可选：`changed_by` |
| `ticket.completed` | 工单到达某个 finish 状态并被视为完成 | ticket 必填 | `status_id`、`status_name`；可选：`completed_by` |
| `ticket.cancelled` | 工单被人类或平台显式取消 | ticket 必填 | 可选：`cancelled_by`、`reason` |
| `ticket.retry_scheduled` | 平台已为该工单安排下一次退避重试 | ticket 必填 | `attempt_count`、`backoff_seconds`、`next_retry_at`、`reason` |
| `ticket.retry_paused` | 工单重试被暂停，等待人工处理或外部条件恢复 | ticket 必填 | `pause_reason`；可选：`stall_count`、`consecutive_errors`、`threshold` |
| `ticket.budget_exhausted` | 工单预算耗尽并进入暂停重试态 | ticket 必填 | `cost_amount`、`budget_usd` |
| `agent.claimed` | 某个 Agent 成功领取该工单 | ticket + agent 必填 | 可选：`run_id`、`agent_name` |
| `agent.launching` | 该工单对应的 runtime/session 启动中 | ticket + agent 必填 | 可选：`run_id`、`session_id` |
| `agent.ready` | runtime/session 已就绪，可继续执行 turn | ticket + agent 必填 | 可选：`run_id`、`session_id` |
| `agent.paused` | runtime/session 被人工或系统暂停 | ticket + agent 必填 | 可选：`run_id`、`reason` |
| `agent.failed` | runtime/session 启动失败或执行失败，需要人工关注或进入重试路径 | ticket + agent 必填 | 可选：`run_id`、`error` |
| `agent.completed` | Agent 成功完成本次工单执行目标 | ticket + agent 必填 | 可选：`run_id`、`result` |
| `agent.terminated` | runtime/session 已结束且 ownership 被释放 | agent 必填，ticket 通常应填写 | 可选：`run_id`、`reason` |
| `hook.started` | 某个 Hook 开始执行 | project 必填；ticket/workflow 视作用域可选 | `hook_name`、`hook_scope`（`ticket` / `workflow`） |
| `hook.passed` | 某个 Hook 执行成功 | project 必填；ticket/workflow 视作用域可选 | `hook_name`、`hook_scope`；可选：`duration_ms` |
| `hook.failed` | 某个 Hook 执行失败 | project 必填；ticket/workflow 视作用域可选 | `hook_name`、`hook_scope`、`error`；可选：`duration_ms`、`blocking` |
| `pr.linked` | 与工单关联的某个 RepoScope 记录了 PR 链接 | ticket 必填 | `repo_id` 或 `repo_name`；可选：`pull_request_url`、`pull_request_number` |

**额外约束：**

- `message` 是人类可读摘要；它不是 machine contract，前端不得依赖 `message` 文本推导类型
- 结构化语义必须进 `metadata`；例如状态变化必须带 `from/to`，不能只在 `message` 里写一句“moved to done”
- 同一件业务事实只允许一个 canonical 类型；例如“领取工单”统一记为 `agent.claimed`，不得在别处再发一个语义等价的 `ticket.assigned`
- `ActivityEvent` 是 append-only 审计流；事件一旦写入不得被“改写成别的类型”

**以下内容不得直接写入 ActivityEvent：**

- token 级别或文本增量级别的 CLI 输出
- `stdout/stderr` 连续滚屏文本
- Provider 原始 reasoning / internal chain-of-thought 文本
- 高频心跳镜像文本，除非被提升为明确的业务告警

### 6.11.3 三层事件模型与页面消费规则

OpenASE 采用严格的三层事件模型：

1. `AgentTraceEvent`
   面向“机器与排障”，记录细粒度 CLI 运行轨迹。
2. `AgentStepEvent`
   面向“操作者理解”，记录人类可读的动作阶段切换。
3. `ActivityEvent`
   面向“产品与业务”，记录真正重要的系统活动。

**页面消费规则：**

- Agent 控制台的 Output/Trace 面板：读取 `AgentTraceEvent`
- Agent 控制台的 Action Timeline：读取 `AgentStepEvent` + `AgentRun.current_step_*`
- Dashboard 的 Activity Feed：只读取 `ActivityEvent`
- 项目级 Activity 页：只读取 `ActivityEvent`
- Ticket 详情页的主时间线：读取 `TicketTimelineItem` 投影；该投影由 `Ticket.description`、`TicketComment`、按 `ticket_id` 过滤后的 `ActivityEvent` 组合而成
- Ticket 详情页的评论历史面板：读取 `TicketCommentRevision`
- Ticket 详情页的 Hook History：优先展示 Hook 执行记录；必要时可由 `ActivityEvent(event_type like 'hook.%')` 投影

**这意味着：**

- Agent 数量再多，Dashboard 也不会被 token 级输出刷屏
- Ticket Activity 只承载“发生了什么”，不承载“终端里滚了什么”
- Ticket Detail 的主体验是 GitHub Issue 风格时间线，而不是“评论一块、系统事件一块”的双面板拼接
- 想看细节时，进入 Agent 控制台看 Trace；想看过程时，看 Step；想看业务结果时，看 Activity

### 6.11.4 运行时事件投影管线

为了确保多 CLI Provider（Codex / Claude / Gemini / Custom）行为一致，运行时事件必须按如下顺序流动：

1. Adapter 从 CLI runtime stream 接收原始 provider 事件。
2. Event Normalizer 将 provider 私有事件映射为统一 `AgentEvent` 协议。
3. Trace Persister 将每个 `AgentEvent` 追加写入 `AgentTraceEvent(sequence++)`。
4. Step Projector 基于最新 `AgentEvent` 推导 `current_step_status` / `current_step_summary`；只有阶段真正切换时，才更新 `AgentRun.current_step_*` 并追加 `AgentStepEvent`。
5. Lifecycle / Business Projector 只在发生 coarse-grained 业务事件时写入 `ActivityEvent`，例如 `agent.claimed`、`ticket.status_changed`、`hook.failed`、`pr.opened`。
6. EventProvider / SSE 层按事件层级分别分发：
   - Trace stream -> `AgentTraceEvent`
   - Step stream -> `AgentStepEvent`
   - Activity stream -> `ActivityEvent`

**禁止行为：**

- 前端直接消费 provider 原始事件并自行猜测状态
- 由 UI 侧把多条 trace 聚合后反推业务 Activity
- 用 `ActivityEvent` 回填或替代 `AgentRun.current_step_*`
- 把 token / delta 输出错误地提升为业务 Activity

---

## 第七章 编排引擎与工单的关系

### 7.1 核心原则：编排引擎不管状态流转

**OpenASE 不是项目管理工具，是编排引擎。** 它不关心工单在看板上怎么流转——用户可以自由定义任意数量的 Custom Status，自由拖拽工单在列之间移动。编排引擎只关心一个问题：**这个工单我该不该接手执行？**

答案由 Workflow 的配置决定，不由硬编码的状态机决定。

### 7.2 Workflow 定义 Pickup / Finish 规则

每个 Workflow 在数据库中配置两组必填状态集合：

```json
{
  "pickup_status_ids": ["Todo", "Ready for AI"],
  "finish_status_ids": ["Done", "Needs Review"]
}
```

上面的状态名只是为了说明语义。真实配置在数据库中保存为 `TicketStatus.id` 的非空集合，并通过 UI 选择维护。

**编排引擎的调度逻辑简化为：**

```go
func (s *Scheduler) runTick(ctx context.Context) {
    // 获取所有活跃 Workflow
    workflows, _ := s.workflowRepo.ListActive(ctx)

    for _, wf := range workflows {
        // 每个 Workflow 有自己的 pickup status 集合
        pickupStatusIDs := wf.PickupStatusIDs

        // 扫描处于任一 pickup status 的工单
        candidates, _ := s.ticketRepo.ListByStatusIDs(ctx, wf.ProjectID, pickupStatusIDs)

        for _, t := range candidates {
            // 检查依赖、并发、Workflow 绑定的 Agent 定义和 Provider semaphore
            // 以及“当前命中的 pickup status”自身是否还有容量
            if s.canDispatch(ctx, t, wf) {
                s.dispatch(ctx, t, wf)
            }
        }
    }
}
```

**就这么简单。** 没有硬编码的状态迁移图。编排引擎只做三件事：

1. **扫描**：找到 `status ∈ workflow.pickup_status_ids` 的工单
2. **执行**：按 Workflow 绑定的 Agent 定义创建 AgentRun，跑 Hook
3. **结束**：
   - 若 `finish_status_ids` 只有 1 个状态，编排引擎自动把工单移到该状态
   - 若 `finish_status_ids` 包含多个状态，Agent 必须通过 Platform API 显式选择其中一个作为目标状态
   - AgentRun 出错 → 指数退避后自动重试（不存在 "failed" 终态）

**没有 "failed" 状态。** 工单只有两种终结方式：到达 `finish`（成功），或被人类 `cancel`（主动放弃）。Agent 执行失败只是触发重试——指数退避、预算扣减、错误率告警，但工单始终留在 pickup 状态等待下一次尝试。只有人类才能决定放弃一个工单。

中间工单在什么状态、用户怎么拖拽、看板有几列——编排引擎不关心具体名字；调度入口完全由 Workflow 的 `pickup_status_ids` 显式定义，`stage` 只用于终态判断以及依赖是否已经解除。

### 7.3 典型配置示例

**标准软件开发流程：**

```yaml
pickup_statuses: ["Todo"]   # 用户把工单拖到 Todo 列 → 编排引擎接手
finish_statuses: ["Done"]   # Agent 完成 → 工单跳到 Done 列
# 无 fail 状态——失败自动重试，人类 cancel 才终止
```

**科研流程（实验验证员）：**

```yaml
status:
  pickup: "待实验"          # 用户审核完 Idea 后拖到"待实验" → Agent 接手
  finish: "实验完成"         # Agent 跑完实验 → 工单跳到"实验完成"
  # 实验失败不终止——Agent 通过 Platform API 自行设状态（如 "Fail 待分析"）
```

**运维流程（DevOps）：**

```yaml
status:
  pickup: "待部署"
  finish: "已部署"
  # 无 fail——部署失败自动重试，告警通知人类
```

**代码审查流程：**

```yaml
status:
  pickup: "待 Review"       # PR 提交后用户拖到"待 Review" → 审查 Agent 接手
  finish: "Review 完成"
  # 无 fail——出错自动重试
```

**关键洞察：同一个项目的不同 Workflow 可以有不同的 pickup/finish 状态。** 这意味着看板上不同列对应不同角色的工作入口——"待开发"列是 coding Workflow 的 pickup，"待测试"列是 test Workflow 的 pickup，"待部署"列是 deploy Workflow 的 pickup。工单在看板上的流转自然形成流水线：

```
Backlog → 待开发 → [coding Agent 执行] → 待测试 → [test Agent 执行] → 待部署 → [deploy Agent 执行] → 已上线
              ↑ pickup                       ↑ pickup                     ↑ pickup
              coding Workflow                test Workflow                deploy Workflow
```

### 7.4 执行过程中的状态

编排引擎在执行过程中需要知道工单是否正在被处理（避免重复分发）。这不通过 Custom Status 实现，而是通过 Ticket 的 `current_run_id` 字段：

- `current_run_id == null`：未被领取，可以分发
- `current_run_id != null`：已被领取，跳过

这里的语义是**“单个 Ticket 只允许一个当前活跃 run”**，不是**“单个 Agent 只允许同时处理一个 Ticket”**。`current_run_id` 是 Ticket 维度的占用标记，用来防止同一工单被重复领取；它不构成 Agent 维度的单并发限制。

```go
func (s *Scheduler) canDispatch(ctx context.Context, t *ticket.Ticket, wf *workflow.Workflow) bool {
    // 已被领取 → 跳过
    if t.CurrentRunID != "" {
        return false
    }
    // 重试暂停（预算耗尽 / 人类暂停）→ 跳过
    if t.RetryPaused {
        return false
    }
    // 退避中（还没到下次重试时间）→ 跳过
    if t.NextRetryAt != nil && time.Now().Before(*t.NextRetryAt) {
        return false
    }
    // 被依赖阻塞 → 跳过
    if s.ticketSvc.IsBlocked(ctx, t.ID) {
        return false
    }
    // 当前命中的 pickup status 并发已满 → 跳过
    if matchedPickupStatus.MaxActiveRuns > 0 &&
       s.pool.ActiveCountForStatus(matchedPickupStatus.ID) >= matchedPickupStatus.MaxActiveRuns {
        return false
    }
    // 并发数已满 → 跳过
    if s.pool.ActiveCountForWorkflow(wf.ID) >= wf.MaxConcurrent {
        return false
    }
    return true
}
```

Agent 完成后：

```go
// Agent 成功完成
t.CurrentRunID = ""                        // 释放当前 AgentRun 占位
t.AttemptCount = 0                         // 重置计数
t.ConsecutiveErrors = 0                    // 重置错误计数
if len(wf.FinishStatusIDs) == 1 {
    t.StatusID = wf.FinishStatusIDs[0]     // 单 finish 时自动落到唯一目标状态
}
// 多 finish 时：Agent 必须自己通过 Platform API 选择 workflow.finish_status_ids 内的某一个目标状态
s.ticketRepo.Save(ctx, t)
```

Agent 出错后——**不存在 "failed" 终态，只有重试**：

```go
// Agent 出错（Hook 失败、CLI 崩溃、Stall 超时等）
t.CurrentRunID = ""                        // 释放当前 AgentRun 占位
t.AttemptCount++
t.ConsecutiveErrors++

// 计算退避时间：10s × 2^(attempt-1)，上限 30 分钟
backoff := min(10 * time.Second * (1 << (t.AttemptCount - 1)), 30 * time.Minute)
t.NextRetryAt = time.Now().Add(backoff)

// 工单留在 pickup 状态，不移走。等退避时间到了，下个 Tick 重新领取
// 没有 max_attempts 上限——Agent 会一直重试，直到：
//   1. 成功（移到 finish）
//   2. 预算耗尽（暂停重试，通知人类）
//   3. 人类主动 cancel

// 预算检查
if t.CostAmount >= t.BudgetUSD && t.BudgetUSD > 0 {
    t.RetryPaused = true   // 暂停重试，不是 failed
    t.PauseReason = "budget_exhausted"
    // 通知人类："ASE-42 预算已耗尽（$5.00/$5.00），是否追加预算？"
    s.notifyBudgetExhausted(ctx, t)
}

// 错误率告警（滑动窗口）
if t.ConsecutiveErrors >= 3 {
    // 连续 3 次失败 → 通知人类，但不停止重试
    s.notifyHighErrorRate(ctx, t)
}

s.ticketRepo.Save(ctx, t)
```

**退避策略：**

| 尝试次数 | 退避时间 | 累计等待 |
|---------|---------|---------|
| 1 | 10s | 10s |
| 2 | 20s | 30s |
| 3 | 40s | 1m10s |
| 4 | 80s | 2m30s |
| 5 | 160s | 5m10s |
| 6 | 320s | 10m30s |
| 7+ | 30m (上限) | 每 30 分钟重试一次 |

**三种暂停（非终止）机制：**

| 暂停原因 | 触发条件 | 行为 | 恢复方式 |
|---------|---------|------|---------|
| 预算耗尽 | `cost_amount >= budget_usd` | 停止重试，通知人类 | 人类追加预算：`openase ticket set-budget ASE-42 10.00` |
| 人类暂停 | 用户在 UI 点"暂停" | 停止重试 | 用户在 UI 点"继续" |
| 依赖阻塞 | 被 blocks 的工单未完成 | 不参与调度 | 阻塞工单完成后自动恢复 |

注意：**这三种都不是 "failed"——工单仍在看板上，人类随时可以恢复。**

**关键规则：任何 `status_id` 变更都清空 `current_run_id`。** 无论是编排引擎自动移状态、Agent 通过 Platform API 改状态、还是人类在看板上手动拖拽——只要 status_id 变了，`current_run_id` 就清空。这保证工单回到任何 pickup 列都能被重新领取。

```go
// internal/httpapi/ticket_api.go — 人类通过 UI 拖拽工单时
func (h *TicketHandler) UpdateStatus(c echo.Context) error {
    ...
    if newStatusID != t.StatusID {
        t.StatusID = newStatusID
        t.CurrentRunID = ""       // 状态变了就清空当前运行占位
        t.ConsecutiveErrors = 0   // 人类介入后重置错误计数
        t.RetryPaused = false     // 人类拖拽 = 主动恢复
    }
    ...
}
```

### 7.5 审批

审批不再通过单独实体建模。`on_complete` Hook 只是显式状态推进前的质量门禁；真正把工单移到非 `pickup` 状态（通常是 `in_review`）必须来自 Agent 或人类的显式状态更新。

```go
// 显式的状态推进请求通过 on_complete Hook 校验后
// 才允许进入人工审核态
t.StatusID = requestedStatusID
s.ticketRepo.Save(ctx, t)
s.notifier.Send(ctx, "ticket.in_review", ...)
```

---

## 第八章 Hook 体系：Workflow Hook 与 Ticket Hook

OpenASE 有两类完全不同的 Hook，必须严格区分：

- **Workflow Hook**：Workflow/Harness 自身生命周期的事件（加载、激活、停用）
- **Ticket Hook**：单个工单执行过程中的事件（领取、启动、完成、出错、取消）

两类 Hook 都在 Harness 的 YAML Frontmatter 中定义，脚本文件跟随项目仓库。

### 8.1 Workflow Hook（Harness 生命周期）

Workflow Hook 在 **Harness 本身被加载/卸载时触发**，与具体工单无关。

| Hook | 触发时机 | 典型用途 |
|------|---------|---------|
| `workflow.on_activate` | Workflow 首次发布或重新启用时 | 验证依赖环境（Agent CLI 是否可用、代码仓库可达性）；预热缓存 |
| `workflow.on_deactivate` | Workflow 被禁用或 Harness 删除时 | 清理全局资源；通知团队该角色已下线 |
| `workflow.on_reload` | Workflow 新版本发布时 | 验证新配置合法性；通知团队 Harness 已更新 |

```yaml
---
workflow_hooks:
  on_activate:
    - cmd: "claude --version"    # 验证 Claude Code 可用
      on_failure: block          # 不可用则 Workflow 不激活
    - cmd: "git ls-remote origin"
      on_failure: warn
  on_reload:
    - cmd: "echo 'Harness v{{ workflow.version }} loaded'"
      on_failure: ignore
---
```

**Workflow Hook 的执行上下文不是工单工作区**——它在服务控制的 project-level runtime context 中执行。若 Hook 需要访问代码仓库，平台可显式 checkout Hook 声明所需的 repo 到轻量 project workspace；但 Workflow/Skill 的控制面编辑本身不依赖任何 repo 工作区是否预先存在。

**前置条件：**

- Workflow Hook 的触发来自控制面中的 Workflow 生命周期事件，而不是仓库 `.openase/harnesses` 文件监听
- 打开、编辑、发布 Workflow 与 Skills 不要求任何 repo 工作区预先存在
- 若某个 Workflow Hook 的命令本身需要访问代码仓库，平台应在执行该 Hook 前单独解析所需代码上下文；这属于 Hook 执行依赖，不属于 Workflow 控制面存储前置条件
- `workflow.on_activate` 可以校验 git transport、CLI、依赖缓存是否可用，但不负责决定 Workflow 是否可被持久化

### 8.2 Ticket Hook（工单执行生命周期）

Ticket Hook 在 **每个工单的每次执行过程中触发**，每个工单每次启动都是独立的执行上下文。

| Hook | 触发时机 | 阻塞行为 | 典型用途 |
|------|---------|---------|---------|
| `ticket.on_claim` | 编排引擎接手工单后、Agent 启动前 | 失败则释放工单，退避后重试 | 准备工作副本、安装依赖、补充运行时上下文、解密密钥 |
| `ticket.on_start` | Agent CLI 子进程启动前 | 失败则触发重试 | 拉取最新代码、检查分支冲突、验证 Agent 可用 |
| `ticket.on_complete` | Agent 声明任务完成时 | 失败则阻止推进，Agent 收到反馈继续 | 运行测试、lint、类型检查、安全扫描 |
| `ticket.on_done` | 工单成功到达 finish 状态后 | 不阻塞 | 工作区清理、通知 |
| `ticket.on_error` | Agent 单次执行出错后（每次重试前） | 不阻塞 | 错误日志、告警通知、部分清理 |
| `ticket.on_cancel` | 用户手动取消工单后 | 不阻塞 | 停止 Agent、清理工作区、关闭未合并 PR |

```yaml
---
ticket_hooks:
  on_claim:
{% for repo in repos %}
    - cmd: "git fetch origin && git checkout -B agent/{{ ticket.identifier }} origin/{{ repo.default_branch }}"
      workdir: "{{ repo.name }}"
      timeout: 60
{% endfor %}
    - cmd: "pnpm install --frozen-lockfile"
      workdir: "web"
      timeout: 300
  on_complete:
    - cmd: "bash scripts/ci/run-tests.sh"
      timeout: 600
      on_failure: block
    - cmd: "bash scripts/ci/lint.sh"
      timeout: 120
      on_failure: block
  on_done:
    - cmd: "bash scripts/ci/cleanup.sh"
      on_failure: ignore
  on_error:
    - cmd: "echo '第 $OPENASE_ATTEMPT 次出错，退避后重试' >> /tmp/errors.log"
      on_failure: ignore
---
```

多 Repo 工单下，`on_claim` 应按 `repos` 列表逐个进入对应工作目录并准备分支；不能再假设只存在一个默认 repo。单 Repo 项目中，这个循环自然只会展开出一条命令。

**Ticket Hook 在工单工作区目录下执行**（即 `on_claim` 准备出的 `~/.openase/workspace/{org}/{project}/{ticket}/` 目录）。

**职责边界：**

- `ticket.on_claim` 可以在已存在的工作副本中执行 `git fetch`、`checkout`、依赖安装等准备动作
- `ticket.on_claim` 不负责创造任何项目级缓存层；调度器直接从远端仓库准备工作副本
- `ticket.on_start` 中的“拉取最新代码”指当前工作副本对远端仓库执行的 `fetch`

### 8.2.1 哪些操作需要最新代码基线

平台统一直接从远端仓库获取最新代码基线。

- **默认路径：直接 fetch / checkout**
  - 创建或恢复 Ticket Workspace
  - 在代码仓库上下文中执行需要最新基线的 Workflow / Ticket Hook
  - 任何要求 Agent 基于最新默认分支创建工作分支的路径
这一定义的核心目标是：默认用最简单、正确的远端仓库语义工作，不再维护额外中间层状态机。

### 8.3 两类 Hook 的关键区别

| 维度 | Workflow Hook | Ticket Hook |
|------|-------------|-------------|
| 触发粒度 | 每个 Workflow 生命周期 | 每个工单每次执行 |
| 执行上下文 | 项目级 runtime context（必要时附带显式 repo checkout） | 工单工作区 |
| 触发频率 | 极低（Harness 变更时） | 高（每个工单都触发） |
| YAML 键名 | `workflow_hooks:` | `ticket_hooks:` |
| 可用环境变量 | `OPENASE_PROJECT_ID`, `OPENASE_WORKFLOW_NAME` | 完整工单上下文（ticket_id, repos, agent 等） |

### 8.4 脚本跟随仓库

**所有 Hook 调用的脚本文件都在项目仓库中**，平台在准备工单工作区时直接 checkout 对应 repo，因此工作副本天然拥有全部 Hook 能力：

```
your-project/
├── scripts/
│   └── ci/
│       ├── run-tests.sh        # Ticket Hook 调用
│       ├── lint.sh
│       ├── typecheck.sh
│       └── cleanup.sh
├── .openase/
│   └── ...                     # 该目录不再是 Workflow / Skill 权威源
└── src/                        # 项目代码
```

### 8.5 Ticket Hook 执行环境

每个 Ticket Hook 命令注入以下环境变量：

| 环境变量 | 说明 | 示例值 |
|---------|------|--------|
| `OPENASE_TICKET_ID` | 工单 UUID | `550e8400-...` |
| `OPENASE_TICKET_IDENTIFIER` | 工单可读标识 | `ASE-42` |
| `OPENASE_WORKSPACE` | 工单工作区根目录 | `/home/openase/.openase/workspace/acme/payments/ASE-42` |
| `OPENASE_REPOS` | 仓库列表 JSON | `[{"name":"backend","path":"/home/openase/.openase/workspace/acme/payments/ASE-42/backend"}]` |
| `OPENASE_AGENT_NAME` | Agent 名称 | `claude-01` |
| `OPENASE_WORKFLOW_TYPE` | Workflow 类型 | `coding` |
| `OPENASE_ATTEMPT` | 当前尝试次数 | `1` |
| `OPENASE_HOOK_NAME` | 当前 Hook 名 | `on_complete` |

`workdir` 字段指定子目录执行（多 Repo 场景）：

```yaml
ticket_hooks:
  on_claim:
    - cmd: "pnpm install --frozen-lockfile"
      workdir: "frontend"    # /home/openase/.openase/workspace/acme/payments/ASE-42/frontend/
    - cmd: "go mod download"
      workdir: "backend"     # /home/openase/.openase/workspace/acme/payments/ASE-42/backend/
```

退出码：`0` = 通过，`非 0` = 失败。on_failure 策略：`block`（默认）/ `warn` / `ignore`。

### 8.6 每个工单 = 独立的 Agent 会话（对齐 Symphony）

**每个工单的每次执行都启动一个全新的、独立的 Agent CLI 子进程。** 这对齐 Symphony 的设计：

```
工单 ASE-42 被领取
  │
  ├── ticket.on_claim Hook（派生工作副本、checkout 分支、安装依赖）
  │
  ├── 启动 Agent CLI 子进程（bash -lc "claude -p ... --output-format stream-json"）
  │   │
  │   ├── Turn 1：发送渲染后的 Harness Prompt
  │   │   Agent 读需求 → 写代码 → 提交
  │   │
  │   ├── Turn 完成 → 检查工单是否仍在 pickup 状态
  │   │   是 → Turn 2：发送续接指导（不重发原始 Prompt，复用线程上下文）
  │   │   否 → 结束
  │   │
  │   ├── Turn 2：Agent 继续工作（同一个 thread，保持上下文）
  │   │   ...最多 max_turns 次
  │   │
  │   └── Agent CLI 子进程退出
  │
  ├── ticket.on_complete Hook（make test, lint, ...）
  │   通过 → 移到 finish 状态
  │   失败 → 退避，1秒后 重新领取，Agent 收到"上次 on_complete 失败原因"作为续接上下文
  │
  └── ticket.on_done Hook（清理、通知）
```

**关键细节（参考 Symphony Section 10.2-10.3）：**

- **Turn 1 发完整 Prompt**：Jinja2 渲染后的 Harness 内容，包含工单描述、仓库路径、工作边界
- **Turn 2+ 发续接指导**：不重发原始 Prompt（已在线程历史中），只发"上次 on_complete 失败原因，请修复后重试"或"继续完成剩余工作"
- **同一个 thread_id 复用**：Agent CLI 子进程在多 Turn 间保持存活，不重启
- **session_id = `<thread_id>-<turn_id>`**：每个 Turn 有独立 ID，用于追踪和日志
- **Stall 检测**：5 分钟无 Agent 事件 → Kill 子进程 → 退避后重新启动新子进程

**必须补充的 Symphony 级运行语义：**

- **Start Session 只做一次**
  - `initialize -> initialized -> thread/start`
  - 在同一个 worker 生命周期内只创建一个 thread
- **Run Turn 可多次**
  - `turn/start` 在同一个 thread 上反复调用
  - 每完成一轮 turn，都要去 tracker 重新读取该工单的最新状态
- **正常退出不等于工单完成**
  - turn 正常完成后，如果工单仍在 active state，调度器应在 1 秒后安排 continuation retry
  - 只有工单离开 active state、进入 terminal state，或 hit `max_turns` 后由上层接管，才算本次 worker 生命周期真正结束
- **max_turns 是单次 worker 生命周期上限**
  - 达到上限后并不是直接 finish ticket
  - 而是把控制权交回 orchestrator，由 orchestrator 决定是否继续下一次 agent run
- **Turn 间上下文来源有且仅有两个**
  - Codex thread 历史
  - 当前 issue workspace / workpad / tracker 最新状态
  - 不应该在 continuation turn 中重复拼接整份原始 Prompt

### 8.7 与 Agent CLI 内部 Hook 的关系

| 层面 | Hook 来源 | 控制范围 |
|------|---------|---------|
| **OpenASE Workflow Hook** | Harness `workflow_hooks:` | Workflow 生命周期（加载/卸载） |
| **OpenASE Ticket Hook** | Harness `ticket_hooks:` | 工单执行生命周期（claim → done） |
| **Agent CLI Hook** | `.claude/settings.json` PreToolUse/PostToolUse | Agent 内部工具调用（每次 Edit/Bash 前后） |

三层互补：Workflow Hook 确保环境可用，Ticket Hook 确保工单交付质量，Agent CLI Hook 确保 Agent 执行过程中的实时行为合规。

---

## 第九章 可观测性设计

OpenASE 管理的是 AI Agent 执行真实工程任务——成本、质量、效率都需要量化。可观测性不是事后加的监控，而是产品核心功能的一部分。

### 9.1 三支柱架构

采用 OpenTelemetry 作为统一的可观测性框架，支持 Traces、Metrics、Logs 三支柱。数据导出到用户自选的后端（Jaeger、Prometheus、Grafana、Datadog 等），OpenASE 自身不内置存储——和数据库一样，可观测性后端是用户的基础设施。

对于不需要完整可观测性的个人用户，默认实现为 NoopTracer + 内存 Metrics（仅在 Web UI 仪表盘中展示），零外部依赖。

### 9.2 核心指标体系

**工单指标（Ticket Metrics）——衡量产出**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openase.ticket.created_total` | Counter | project, workflow_type, source | 工单创建总数（按来源：manual / scheduled / github_issue / api） |
| `openase.ticket.completed_total` | Counter | project, workflow_type, outcome | 工单完成总数（outcome: done / cancelled） |
| `openase.ticket.cycle_time_seconds` | Histogram | project, workflow_type | 工单周期时间（todo → done 的总耗时） |
| `openase.ticket.agent_time_seconds` | Histogram | project, workflow_type | Agent 实际执行时间（in_progress 累计时长，排除等待审批） |
| `openase.ticket.attempts` | Histogram | project, workflow_type | 每个工单的尝试次数分布 |
| `openase.ticket.queue_depth` | Gauge | project, workflow_type, status | 各状态的工单数量（实时快照） |
| `openase.ticket.stall_total` | Counter | project, workflow_type | Stall 次数 |

**Agent 指标（Agent Metrics）——衡量资源**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openase.agent.active` | Gauge | project, provider, adapter_type | 当前活跃 AgentRun 数 |
| `openase.agent.utilization_ratio` | Gauge | project, provider | Provider 并发利用率（active_runs / max_parallel_runs） |
| `openase.agent.session_duration_seconds` | Histogram | provider, adapter_type | 单次 Agent 会话时长 |
| `openase.agent.tokens_used_total` | Counter | provider, model, direction | Token 消耗（direction: input / output） |
| `openase.agent.cost_usd_total` | Counter | provider, model, project | 累计 API 成本（美元） |
| `openase.agent.cost_usd_per_ticket` | Histogram | provider, workflow_type | 单工单成本分布 |
| `openase.agent.heartbeat_age_seconds` | Gauge | agent_id | 距上次心跳的秒数（>300 触发 Stall） |

**Hook 指标（Hook Metrics）——衡量质量门禁**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openase.hook.execution_total` | Counter | hook_name, outcome | Hook 执行总数（outcome: pass / error / timeout） |
| `openase.hook.duration_seconds` | Histogram | hook_name | Hook 执行耗时分布 |
| `openase.hook.block_total` | Counter | hook_name | Hook 阻塞状态推进的次数（on_complete 失败导致工单无法进入 review） |

**编排引擎指标（Orchestrator Metrics）——衡量系统健康**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openase.orchestrator.tick_duration_seconds` | Histogram | — | 每个调度 Tick 的耗时 |
| `openase.orchestrator.tickets_dispatched_total` | Counter | workflow_type | Tick 中分发的工单数 |
| `openase.orchestrator.tickets_skipped_total` | Counter | reason | 跳过的工单数（reason: blocked / no_agent / max_concurrency） |
| `openase.orchestrator.workers_active` | Gauge | — | 当前活跃 Worker 数 |
| `openase.orchestrator.retry_total` | Counter | strategy | 重试总数（strategy: quick / exponential / stall_recovery） |
| `openase.orchestrator.harness_publish_total` | Counter | — | 新发布的 Harness 版本被调度器识别的次数 |

**PR 指标（Git Metrics）——衡量交付质量**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openase.pr.opened_total` | Counter | project, repo | Agent 创建的 PR 总数 |
| `openase.pr.merged_total` | Counter | project, repo | 成功合并的 PR 总数 |
| `openase.pr.first_pass_rate` | Gauge | project | PR 一次通过率（merged without changes_requested / total merged） |
| `openase.pr.time_to_merge_seconds` | Histogram | project, repo | PR 从 open 到 merge 的耗时 |
| `openase.pr.review_rounds` | Histogram | project | PR 经历的 review 轮数 |

**系统指标（System Metrics）——衡量运行时**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openase.system.goroutines` | Gauge | — | 当前 goroutine 数 |
| `openase.system.db_connections_active` | Gauge | — | 活跃数据库连接数 |
| `openase.system.db_query_duration_seconds` | Histogram | operation | 数据库查询耗时 |
| `openase.system.sse_connections_active` | Gauge | — | 当前 SSE 连接数 |
| `openase.system.uptime_seconds` | Gauge | — | 服务运行时间 |

### 9.3 分布式追踪

每个工单的完整生命周期构成一个 Trace，内部的每个关键操作是一个 Span：

```
Trace: ticket/ASE-42
├── Span: orchestrator.dispatch          (调度分发)
├── Span: hook.on_claim                  (领取前 Hook)
│   ├── Span: hook.exec clone-repos.sh
│   └── Span: hook.exec decrypt-secrets.sh
├── Span: agent.session                  (Agent 执行)
│   ├── Span: adapter.claudecode.start
│   ├── Span: adapter.claudecode.turn.1
│   ├── Span: adapter.claudecode.turn.2
│   └── Span: adapter.claudecode.stop
├── Span: hook.on_complete               (完成后 Hook)
│   ├── Span: hook.exec run-tests.sh
│   └── Span: hook.exec lint-check.sh
└── Span: hook.on_done                   (收尾 Hook)
    └── Span: hook.exec cleanup.sh
```

每个 Span 携带标准属性：`ticket.id`、`ticket.identifier`、`workflow.type`、`agent.name`、`agent.provider`。这让用户可以在 Jaeger/Grafana 中按工单、按 Agent、按 Workflow 类型过滤和分析。

### 9.4 结构化日志

所有日志通过 slog 输出结构化 JSON，自动注入 Trace ID 和 Span ID 实现日志与追踪的关联：

```json
{
  "time": "2026-03-18T10:30:00Z",
  "level": "INFO",
  "msg": "ticket dispatched to workflow-bound agent definition",
  "ticket_id": "ASE-42",
  "agent_name": "claude-01",
  "workflow_type": "coding",
  "trace_id": "abc123...",
  "span_id": "def456..."
}
```

### 9.5 Web UI 内置仪表盘

即使不配置外部 Prometheus/Grafana，OpenASE 的 Web UI 内置一个轻量仪表盘（基于内存 Metrics），展示核心指标：

| 仪表盘面板 | 展示内容 |
|-----------|---------|
| 工单吞吐量 | 24h / 7d 的工单创建数、完成数、成功率趋势 |
| 平均周期时间 | 按 Workflow 类型的工单周期时间趋势 |
| Agent 利用率 | 各 Provider / AgentRun 的并发占用、当前任务 |
| 成本追踪 | 按项目 / Agent / 模型的 Token 消耗和成本 |
| Hook 健康度 | 各 Hook 的通过率、平均耗时、最近失败 |
| PR 质量 | 一次通过率、平均 review 轮数、合并耗时 |

---

## 第十章 编排引擎

### 10.1 调度循环

编排引擎的核心是一个周期性调度循环（参考 Symphony 的 Tick 模式），每个 Tick：

1. **对账**：检查所有 running 状态的工单——对应的 AgentRun 是否还在运行？是否 stall？工单是否被取消？
2. **对账 Workflow/Skill 版本**：检查是否有新的已发布 Workflow / Skill 版本需要供后续新 runtime 使用；已运行中的 runtime 不隐式漂移
3. **获取候选工单**：查询所有 todo 状态的工单，排除被 block 的
4. **优先级排序**：按 priority → created_at 排序
5. **并发检查**：检查全局运行 semaphore、Provider 级 semaphore 和每 Workflow 并发位
6. **分发执行**：读取 Workflow 绑定的 Agent 定义，创建 AgentRun，创建/复用工作区，注入 Harness Prompt，通过对应 Provider 适配器启动运行

#### 10.1.1 Symphony 风格运行中工单对账细则

OpenASE 的“对账”不能只做黑盒心跳检查，而应该像 Symphony 一样维护一份**单一 authoritative in-memory runtime state**，并在每个 Tick 对这份状态做三类 reconciliation：

1. **Stall 对账**
   - 对每个 running AgentRun 记录：
     - `started_at`
     - `last_codex_timestamp`
     - `last_codex_event`
     - `session_id`
     - `turn_count`
   - `last_codex_timestamp` 优先取最近一条 Codex 事件时间；若尚无事件，则回退到 `started_at`
   - 若 `now - last_codex_timestamp > codex.stall_timeout_ms`
     - 立即 terminate worker / Codex subprocess
     - 保留工作区
     - 按异常退出进入 retry backoff

2. **Tracker 状态对账**
   - 批量抓取所有 running issue 的最新状态：`fetch_issue_states_by_ids(running_ids)`
   - 对每个 issue：
     - 若已进入 terminal state：停止对应 AgentRun，释放 semaphore，清理该 issue 对应工作区
     - 若已不再路由给当前 Workflow：停止对应 AgentRun，释放 semaphore，但不强制清理工作区
     - 若仍在 active state：仅刷新内存中的 issue snapshot，供下一轮 continuation / retry 使用
     - 若离开 active state 但未进入 terminal state：停止对应 AgentRun，释放 semaphore，不触发完成流程

3. **运行时事实对账**
   - AgentRun `:DOWN` / subprocess exit 并不等于工单完成
   - 正常退出时：
     - 先记录本次 session 的 completion totals（尤其是 runtime 秒数）
     - 再重新检查 issue 是否仍处于 active state
     - 若 issue 仍 active，则走 1 秒 continuation retry，而不是直接 finish
   - 异常退出时：
     - 进入指数退避重试
     - 记录最近错误原因到 runtime state / activity / SSE

#### 10.1.2 分发前再验证（Revalidate Before Dispatch）

Symphony 的一个关键经验是：**候选工单列表不是最终真相，dispatch 前必须二次确认**。

OpenASE 在真正 claim + launch 前应执行：

1. 从候选集中选出 issue/ticket
2. 按 `priority -> created_at -> identifier` 排序
3. 读取该 Ticket 对应 Workflow，以及该 Workflow 绑定的 Agent 定义与 Provider
4. 检查分发门槛：
   - 未被 claim
   - 当前不在 running map
   - Workflow 绑定的 Agent 定义存在且启用
   - 全局运行 semaphore 未满
   - Provider 级 `max_parallel_runs` semaphore 未满
   - 当前命中的 pickup status 若配置了 `max_active_runs`，则该 status semaphore 未满
   - per-workflow 并发未满
   - 若存在 `A blocks B` 且 blocker A 未进入终态 stage，则 B 不可分发
5. 在 dispatch 前再次通过仓储/API 获取该工单最新状态
   - 若状态已变化、依赖已变化、工单已不可见，则直接放弃本次 dispatch

这一步的目标是避免“扫描时可执行、真正领取时已过期”的 stale dispatch。

### 10.2 重试策略

| 场景 | 退避策略 | 说明 |
|------|---------|------|
| AgentRun 正常退出 | 1 秒快速重试 | 可能是 Turn 完成但工单未结束 |
| AgentRun 异常退出 | 10s × 2^(attempt-1)，上限 5 分钟 | 指数退避 |
| Stall 超时 | Kill Worker + 按异常退出处理 | 5 分钟无事件触发 |
| 连续 3 次 Stall | 暂停重试（retry_paused=true），通知人类 | 防止无限消耗资源 |
| 工单被取消 | 立即停止，不重试 | 用户主动取消 |

**Retry Token 机制**：Ticket 维护当前 `Retry Token`。所有延迟重试意图都必须携带该 Token；调度器或恢复器在真正 dispatch 前再次比对当前 Token，不匹配则静默丢弃。

- 以下转换必须轮换 `Retry Token`：
  - 任意 Ticket `status` 变更
  - Orchestrator 创建新的延迟重试意图：异常退出重试、stall 恢复重试、turn-limit continuation 重试
  - 健康前进导致当前重试基线被清空：正常完成、人工前移状态、取消当前重试周期
- 状态变更或健康前进在轮换 Token 的同时，必须清空旧的 `next_retry_at / retry_paused / pause_reason / consecutive_errors` 基线，防止旧重试意图被新状态复活。

### 10.3 内部架构

| 组件 | 实现 | 职责 |
|------|------|------|
| Scheduler | 单一 goroutine + ticker | 序列化所有调度决策 |
| RuntimeRegistry | 编排引擎内存映射 | 管理活动 runtime / session 生命周期 |
| RuntimeRunner | 每个 AgentRun 一个 goroutine | 管理单个 Agent CLI 子进程与事件泵 |
| Harness Loader | DB + 文件系统读取 | 解析 Harness 内容并为运行时提供输入 |
| EventBus | Go channel（同进程）/ PG LISTEN/NOTIFY（分开部署） | 与 API 服务的事件通信 |
| HealthChecker / RetryService | 定时 goroutine | 周期性检查 runtime 心跳、清理异常态、驱动重试 |

#### 10.3.1 并发模型

OpenASE 不维护“可被调度的 Agent 实例池”。可调度对象是 Ticket，Agent 只是 Workflow 绑定的执行定义。真正的资源池由 semaphore 表达：

**明确要求：系统必须支持并行。** 这里的并行既包括“多个不同 Agent 定义并行处理多个 Ticket”，也包括“同一个 Agent 定义在 Provider / Workflow / Status / Global 并发限制允许时，同时产生多个 `AgentRun` 来处理多个 Ticket”。禁止把 Agent 误实现成“同一时刻只能占用一个 Ticket”的单 worker 槽位。

1. **Global Run Semaphore**
   - 限制系统总的同时运行数
   - 防止单机 / 单项目无限拉起外部 CLI 子进程
2. **Provider Semaphore**
   - 每个 AgentProvider 维护自己的 `max_parallel_runs`
   - 例如 Codex 最多并发 8、Claude Code 最多并发 4
3. **Status Semaphore**
   - 每个 TicketStatus 可选配置 `max_active_runs`
   - 表示当前处于该 status 内、且已被编排引擎领取的 Ticket 同时最多允许多少个
   - 多个 Workflow 如果 pickup 的状态都命中同一个 status，则共享这一个 status semaphore
   - 典型例子：`Todo` 设为 1，则该入口列任意时刻只允许 1 个 AgentRun 在干活
4. **Workflow Max Concurrent**
   - 业务级限流，不是资源池定义
   - 表示某类工作同一时间最多允许多少个 Ticket 被该 Workflow 驱动

这三层同时生效：

- `能不能跑` 取决于 global semaphore + provider semaphore + status semaphore
- `该不该跑这么多` 取决于 workflow.max_concurrent
- Agent 本体只是执行定义；真正被 claim、运行、完成和回收的是 `AgentRun`
- 任何实现都不得把“Ticket 只有一个 `current_run_id`”错误收窄为“Agent 只能同时处理一个 Ticket”

---

## 第十一章 Agent 适配器体系

### 11.1 统一接口

所有适配器实现同一个 Go interface：

```go
type AgentAdapter interface {
    Start(ctx context.Context, cfg AgentConfig) (Session, error)
    SendPrompt(ctx context.Context, s Session, prompt string) error
    StreamEvents(ctx context.Context, s Session) (<-chan AgentEvent, error)
    Stop(ctx context.Context, s Session) error
    Resume(ctx context.Context, sessionID string) (Session, error)
}
```

### 11.2 Claude Code 适配器（三阶段演进）

**Phase 1：CLI subprocess**

通过 `os/exec` 启动 Claude Code 子进程：

```bash
claude -p "{{harness_prompt}}" \
  --verbose \
  --output-format stream-json \
  --allowedTools "Bash,Read,Edit,Write,Glob,Grep" \
  --max-turns 20 \
  --max-budget-usd 5.00 \
  --append-system-prompt "{{workflow_constraints_prompt}}"
```

- 解析 NDJSON 事件流，映射到 AgentEvent
- 通过 `--resume session_id` 实现多 Turn 续接
- 对 Claude Code 新版本，`-p/--print + --output-format stream-json` 必须同时带 `--verbose`

**Phase 2：Agent SDK 集成**

- 通过 Go 调用 Claude Agent SDK，获得 Hooks + Subagents + MCP 能力
- PreToolUse Hook 拦截危险操作
- PostToolUse Hook 注入质量反馈
- Stop Hook 阻止 Agent 过早停止（检查是否已提交 PR）
- `--json-schema` 强制结构化输出

**Phase 3：Claude Code Hooks 深度集成**

通过 `.claude/settings.json` 配置 Claude Code 原生 Hooks，与 OpenASE 的编排层 Hook 形成双层质量保障：

- **PreToolUse**：检查是否超出工作边界（不允许修改不相关文件）
- **PostToolUse**：Edit/Write 后自动运行 lint + type-check，失败则反馈给 Agent
- **Stop**：检查是否已完成所有涉及 Repo 的 PR 提交
- **TaskCompleted**：运行全量测试，失败则阻止任务标记完成

### 11.3 Codex 适配器

OpenASE 的 Codex 适配器应直接采用 Symphony 已验证的 **stdio request/response + notification** 模式，而不是只抽象成一个“SendPrompt 后等待结果”的黑盒。

#### 11.3.1 Session / Thread / Turn 三层模型

- **Session**
  - 对应一个长期存活的 Codex app-server 子进程
  - 进程只在一个工单的一次 worker 生命周期内复用
- **Thread**
  - 通过 `thread/start` 创建
  - 一个工单的一次 worker 运行中，多个 turn 复用同一个 `thread_id`
- **Turn**
  - 每次 `turn/start` 都会生成新的 `turn_id`
  - `session_id = <thread_id>-<turn_id>`
  - `turn_count` 在同一个 worker 生命周期内递增

#### 11.3.2 启动握手（参考 Symphony `Codex.AppServer`）

1. `initialize`
   - `clientInfo.name/version/title`
   - `capabilities.experimentalApi = true`
2. 等待 `initialize` response（按 request id 匹配）
3. 发送 `initialized`
4. `thread/start`
   - `cwd`
   - `approvalPolicy`
   - `sandbox`
   - `dynamicTools`
5. 等待 `thread/start` response，提取 `thread.id`

**工程要求：**

- 等待 response 时，必须允许并忽略无关 notification / 日志行，不能因为先收到别的消息就报错
- 非 JSON 行只能记日志，不能中断会话
- response 识别必须基于 request id，而不是基于方法名猜测

#### 11.3.3 Turn 发送 Prompt 的标准方式

每个 turn 通过 `turn/start` 发送：

- `threadId`
- `input: [{type: "text", text: prompt}]`
- `cwd`
- `title`
- `approvalPolicy`
- `sandboxPolicy`

其中：

- **Turn 1**
  - 发送完整 Harness Prompt
  - 包含 ticket 描述、仓库信息、边界、验收标准、当前 attempt
- **Turn 2+**
  - 不重发原始完整 prompt
  - 只发送 continuation guidance
  - 明确告诉 Agent：当前 thread 已保留上下文，应从当前 workspace/workpad 继续

这点是 Symphony 的核心经验：**多 turn 续跑依赖 thread 复用，而不是把同一任务重复 prompt 一遍。**

#### 11.3.4 OpenASE 必须支持的 Codex 通知/请求

| 方法 | 方向 | 说明 |
|------|------|------|
| `initialize` | OpenASE → Codex | 初始化连接 |
| `initialized` | OpenASE → Codex | 握手完成通知 |
| `thread/start` | OpenASE → Codex | 创建 thread |
| `turn/start` | OpenASE → Codex | 发起 turn |
| `item/tool/call` | Codex → OpenASE | 动态工具调用请求 |
| `item/commandExecution/requestApproval` | Codex → OpenASE | 命令执行审批 |
| `item/fileChange/requestApproval` | Codex → OpenASE | 文件修改审批 |
| `item/tool/requestUserInput` | Codex → OpenASE | 请求交互式用户输入 |
| `item/agentMessage/delta` | Codex → OpenASE | Agent 可见文本输出增量 |
| `item/commandExecution/outputDelta` | Codex → OpenASE | 命令执行输出增量 |
| `item/completed` | Codex → OpenASE | item 完成快照（用于无 delta 时兜底） |
| `thread/tokenUsage/updated` | Codex → OpenASE | 线程级 token usage 流 |
| `turn/completed` | Codex → OpenASE | turn 正常完成 |
| `turn/failed` | Codex → OpenASE | turn 失败 |
| `turn/cancelled` | Codex → OpenASE | turn 被取消 |

#### 11.3.5 审批、用户输入与交互模式

Codex 适配器必须区分两种运行模式：

- **无人值守编排模式**
  - 用于 ticket worker / orchestrator
  - `approval_policy == "never"`
  - 对 approval request 自动回 `acceptForSession` / `approved_for_session`
  - 对 `item/tool/requestUserInput` 返回标准化的 operator unavailable / default answer
- **Project Conversation 交互模式**
  - 用于第 31 章的 `project_sidebar`
  - 允许 `approval_policy != "never"`
  - 必须把 approval request / requestUserInput 暴露为可恢复的 interrupt，而不是 auto-approve

Project Conversation 的适配要求：

- `item/commandExecution/requestApproval`
  - 持久化为 `pending_interrupt(kind=command_execution_approval)`
  - 等待用户 decision
- `item/fileChange/requestApproval`
  - 持久化为 `pending_interrupt(kind=file_change_approval)`
  - 等待用户 decision
- `item/tool/requestUserInput`
  - 持久化为 `pending_interrupt(kind=user_input)`
  - 等待用户 answer

工程约束：

- interrupt 恢复必须按 request id 精确路由，不能按“当前 turn 大概只有一个 pending request”之类的推测逻辑处理
- provider-native 决策选项必须保留；OpenASE 只做 envelope 标准化，不得把 Codex 的原生决策空间抹平成一个抽象的“允许/拒绝”
- 若当前会话明确是 non-interactive 且无法获得人工输入
  - 应返回标准化的 unavailable / unsupported 响应，而不是无限等待

#### 11.3.6 Token 对账规则（必须遵守）

OpenASE 应采用 Symphony 已文档化的 token accounting 规则：

- **优先使用绝对累计值**
  - `thread/tokenUsage/updated.params.tokenUsage.total`
  - 或 Codex 核心事件里的 `total_token_usage`
- **不要把 `turn/completed.usage` 当作可无脑累加的总量**
  - 它是 event-specific usage，不一定等于线程累计总量
- **按 thread 维护高水位**
  - `delta = next_total - prev_reported_total`
  - 只在高水位前进时累计
- **session 完成时**
  - 只补录 `seconds_running`
  - 不再重复加 token，避免 double count

#### 11.3.7 弥补当前实现缺口的最小落地方案

基于当前 OpenASE 代码现状，真正缺失的不是“能否拉起 Codex session”，而是以下三段链路尚未闭合：

1. `scheduler -> runtimeLauncher`
   - 目前只做到 claim ticket、启动 app-server、得到 `thread_id`
2. `runtime ready -> start turn`
   - 目前还没有真正的 ticket runner 去调用 `SendPrompt` / `turn/start`
3. `turn finished -> reconcile -> continuation / finish / retry`
   - 目前没有把 turn 结果收敛回 ticket / agent lifecycle

因此建议按下面的**最小可实施架构**补齐：

- **新增 `AgentRunner` 组件**
  - 放在 orchestrator 层
  - 输入：`running + runtime_phase=ready + current_ticket_id != nil` 的 agent
  - 职责：
    - 为该 agent 找到已经启动好的 session
    - 生成本轮 prompt
    - 调用 `session.SendPrompt(...)`
    - 消费 `session.Events()`，直到 `turn/completed` / `turn_failed`
    - turn 结束后重新读取 ticket 最新状态并决定下一步

- **把 orchestrator 主循环改成四段**
  - `healthChecker`
  - `machineMonitor`
  - `scheduler`
  - `runtimeLauncher`
  - `agentRunner`

- **首轮 prompt 与续跑 prompt 分离**
  - Turn 1：
    - 直接复用 `workflow.BuildHarnessTemplateData + RenderHarnessBody`
    - 这是 OpenASE 已有的 full prompt source of truth
  - Turn 2+：
    - 使用固定 continuation builder
    - 输入至少包括：
      - `turn_number`
      - `max_turns`
      - `last_error`
      - `ticket.attempt_count`
      - `ticket.status`
    - 只补“继续做什么 / 上一轮为什么没结束”，不重发整份 harness

- **最小 runtime phase 扩展**
  - 当前仅有 `none / launching / ready / failed`
  - 至少补成：
    - `none`
    - `launching`
    - `ready`
    - `executing`
    - `failed`
  - `executing` 表示“已有 turn 在跑”，避免同一 session 被并发 `turn/start`

- **事件驱动心跳替代定时假心跳**
  - 当前 `refreshHeartbeats()` 会为 `ready` agent 定时写 heartbeat，这只能证明 orchestrator 进程活着，不能证明 Codex turn 仍在推进
  - 修正方式：
    - 任何 Codex notification / tool call / token update / turn event 到达时，都更新 `last_heartbeat_at`
    - stall 检测优先使用最后一条 Codex 事件时间
    - 定时补心跳只能作为兜底，不能作为 stall 真相来源

- **Turn 完成后的状态收敛规则**
  - `turn/completed`
    - 重新加载 ticket + workflow
    - 若 ticket 已离开 pickup/active state：
      - 停止 continuation
      - 释放或终止 session
    - 若 ticket 仍 active 且 `turn_count < max_turns`：
      - 直接在同一 session 上启动下一轮 continuation turn
    - 若 ticket 仍 active 但 hit `max_turns`：
      - 停止当前 worker 生命周期
      - 保留 ticket 为 active
      - 通过 retry/continuation 机制把控制权交回 orchestrator
  - `turn_failed`
    - 记录 `last_error`
    - 停止 session
    - 走 retry backoff

- **最小 approval / tool / token 支持**
  - Adapter 不能只识别 `item/tool/call`
  - 还必须至少支持：
    - `item/commandExecution/requestApproval`
    - `item/fileChange/requestApproval`
    - `item/tool/requestUserInput`
    - `thread/tokenUsage/updated`
  - MVP 阶段默认走无人值守：
    - `approval_policy = never`
    - 自动批准 approval request
    - 对 request user input 返回标准化 unavailable answer

- **最小 token 对账实现**
  - 不要求首版就落完整成本中心，但至少要实现：
    - 以内存 map 按 `thread_id` 记录 `prev_total`
    - 收到 `thread/tokenUsage/updated.total` 时计算 delta
    - delta 通过 `ticket.RecordUsage(...)` 入账
    - turn 完成时不再额外重复加 token

- **最小运行时输出可观测性**
  - `item/agentMessage/delta` 与 `item/commandExecution/outputDelta` 必须即时归一成细粒度 `AgentEvent`，并持久化为 `AgentTraceEvent`
  - `item/completed` 只作为“没有 delta 时”的快照兜底，避免重复持久化同一 item 文本
  - `/api/v1/projects/{projectId}/agents/{agentId}/output` 与 `/output/stream` 在兼容期内可以保留命名，但其单一信源必须是 `AgentTraceEvent`，不能混读 `ActivityEvent`
  - Step 状态变化时再额外提升为 `AgentStepEvent`
  - lifecycle / token usage 事件不能冒充 output；output 也不能直接冒充业务 Activity

OpenASE 第一阶段不要试图一次补完复杂交互式审批、Hook Gate、复杂 pause/resume。只要先把“真实 turn 能跑起来、能连续、能失败重试、不会重复记账”这四件事做扎实，链路就会从 `runtime ready` 变成真正可执行。

### 11.4 Gemini CLI 适配器

与 Claude Code 类似，通过 CLI subprocess + stdio stream 接入。

- 编排运行时通过 `internal/orchestrator/agent_adapter_gemini.go` 提供 Gemini adapter，支持 provider 选择、session start、连续 turn 执行、失败传播和统一 `agentEvent` 归一化。
- 当前实现采用“每个 turn 启动一个 Gemini CLI 进程 + 由 OpenASE 在内存维护对话历史”的最小可交付语义，因此可以满足 orchestrator 的连续执行需求。
- `internal/chat/runtime_gemini.go` 继续服务于即时对话语义；两条路径共享相同的 Gemini CLI 非交互 JSON 输出模式。
- 跨 OpenASE 编排进程的 Gemini session resume 暂不支持；如果未来需要 crash-recovery 级恢复，需要补持久化 transcript 或 Gemini 原生 resume 能力。

---

## 第十二章 Git 仓库与 PR 链接集成

### 12.1 当前范围

- **RepoScope 绑定**：一个工单可以显式绑定一个或多个 Repo；每个 RepoScope 记录仓库、工作分支和可选 PR 链接
- **PR 链接记录**：Agent 或人类可以把某个 Repo 的 PR URL 写回对应的 TicketRepoScope，便于在 Ticket Detail 中查看和跳转
- **多 Repo 展示**：当一个工单涉及多个 Repo 时，前端工单详情页展示所有 Repo 的分支和 PR 链接列表
- **非目标**：当前版本不接收 GitHub/GitLab Webhook，不同步 GitHub Issue，不同步 PR 状态，也不同步 CI 状态

### 12.2 工单与 PR 链接的关系

| 工单状态 | Repo/PR 信息 | 触发条件 |
|---------|------------|---------|
| todo | 无 | 工单创建，等待领取 |
| in_progress | 所有涉及 Repo 的 Branch 已创建 | Agent 领取并在每个 Repo 中创建分支 |
| in_review | 一个或多个 RepoScope 记录了 PR 链接（可选） | 人类或 Agent 显式把工单移动到 review 阶段 |
| done | PR 是否存在、是否 merged 都不是自动判断条件 | 人类或 Agent 显式完成工单 |

**明确约束：**

- `pull_request_url` 只用于引用和跳转，不驱动任何自动状态推进
- PR merged / closed / changes requested 不会自动改变 Ticket 状态
- CI 结果不回写到 TicketRepoScope，也不作为 Hook 或状态机输入

---

## 第十三章 UI 设计

OpenASE 的前端不是“后台管理表单集合”，而是一个让人持续盯着项目运行状态、快速调度工作、即时介入异常的工程控制台。整体体验参考 Linear：**简洁、克制、密度高、交互快、默认键盘友好**，但信息密度更偏向“工程运营”而不只是任务管理。

核心设计目标：

- **Dashboard-first**：用户进入产品后的第一屏不是空白列表，而是 Dashboard，先看到项目是否健康、哪里卡住、Agent 是否在工作。
- **Project-centric**：组织是管理边界，项目是工作上下文。用户在 UI 中始终清楚“我现在看的是哪个组织、哪个项目”。
- **Board-native**：看板是默认工作界面，不是附属视图。状态流转、拖拽、实时推送、异常提示都围绕看板展开。
- **Real-time but calm**：SSE 实时更新，但不能造成视觉噪音。变化要被看见，也要可消化。
- **Deep work friendly**：减少层层弹窗和跳页，更多使用侧栏、抽屉、原位编辑和命令面板。

### 13.1 视觉风格与品牌基调

整体视觉沿用 Linear 的“低噪音专业感”，但面向工程控制台场景做增强：

| 设计维度 | 设计要求 |
|---------|---------|
| 视觉气质 | 冷静、专业、克制，不做营销式大色块 |
| 布局密度 | 信息密度偏高，但通过层级、留白、边框控制阅读压力 |
| 色彩策略 | 中性色为底，状态色只用于表达状态，不用于大面积装饰 |
| 动效策略 | 动画短、轻、少，只服务于状态切换、拖拽反馈、实时更新提示 |
| 图标策略 | 统一使用 Lucide，线性、细描边、避免过度装饰 |
| 组件风格 | 低圆角、小阴影、细边框，卡片和面板要“像工具”而不是“像宣传页” |

**推荐视觉 token：**

| Token | 建议值 | 用途 |
|------|-------|------|
| 背景底色 | `#0F1115` 或浅色版 `#F7F8FA` | App 外层背景 |
| 面板底色 | `#151922` / `#FFFFFF` | 卡片、侧栏、表格、抽屉 |
| 分割线 | `rgba(255,255,255,0.08)` / `#E6E8EC` | 面板边框、列表分割线 |
| 主文本 | `#F3F4F6` / `#111827` | 一级信息 |
| 次文本 | `#9CA3AF` / `#6B7280` | 描述、辅助信息 |
| 强调色 | `#5E6AD2` | 当前选中、焦点、主 CTA |
| 成功色 | `#22C55E` | done、hook passed |
| 警告色 | `#F59E0B` | retry、stalled、预算接近上限 |
| 错误色 | `#EF4444` | hook failed、agent error |
| 信息色 | `#38BDF8` | SSE 实时更新、高亮提示 |

**排版建议：**

- 中文界面优先使用 `Inter + Noto Sans SC` 或等价组合，数字与英文保持紧凑，中文可读性稳定。
- 页面标题 20-24px，区块标题 14-16px，正文 13-14px，辅助信息 12px。
- 行高偏紧凑：标题 1.2-1.3，正文 1.5，表格/列表 1.4。
- 全局采用 8pt spacing system，紧凑型列表允许 4pt 微间距。

### 13.2 信息架构（Information Architecture）

前端采用两层上下文：

1. **Organization Context（组织上下文）**：决定当前看到哪些项目、成员、渠道、预算和治理规则
2. **Project Context（项目上下文）**：决定当前看到哪一个项目的看板、Workflow、Agent、通知、配置

用户主路径：

```text
登录 / 首次进入
  → Dashboard（默认入口）
  → 选择组织
  → 选择项目
  → 在项目内通过侧边栏进入 Board / Workflows / Agents / Settings
  → 在 Board 中查看与拖拽工单
  → 在右侧抽屉 / 详情页中深挖单个 Ticket
```

**导航层级：**

| 层级 | 主要对象 | 典型页面 |
|------|---------|---------|
| 全局层 | Dashboard、组织切换、全局搜索、个人设置 | Dashboard、通知中心、命令面板 |
| 项目层 | 看板、Workflow、Agent、审批、活动、成本、设置 | Project Dashboard、Board、Agents、Settings |
| 实体层 | Ticket、Harness、Approval、Channel | Ticket 详情、Harness 编辑器、审批详情 |

### 13.3 全局应用骨架（App Shell）

整个 Web UI 采用固定 App Shell，避免页面跳来跳去造成上下文丢失：

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Top Bar: 组织切换 | 项目切换 | 搜索/Cmd+K | 新建 | SSE 状态 | 用户菜单        │
├───────────────┬──────────────────────────────────────────────┬───────────────┤
│ Left Sidebar  │ Main Content                                 │ Right Panel   │
│ 全局/项目导航   │ Dashboard / Board / Settings / Editor        │ 详情抽屉/AI     │
│ 最近项目/固定项 │                                              │ 上下文面板       │
└───────────────┴──────────────────────────────────────────────┴───────────────┘
```

**设计原则：**

- Top Bar 负责“上下文切换 + 全局动作”
- Left Sidebar 负责“当前上下文下的导航”
- Main Content 负责“主工作区”
- Right Panel 负责“非阻塞深挖”，比如 Ticket 详情、AI 助手、筛选器、高级信息

### 13.4 顶栏（Top Bar）设计

顶栏是上下文切换和高频操作层，保持单行、紧凑、稳定，不随页面变化而大幅跳动。

**从左到右建议布局：**

1. OpenASE Logo / Home
2. 组织选择器（Organization Switcher）
3. 项目选择器（Project Switcher）
4. 当前页面标题 / 面包屑
5. 全局搜索入口 + `Cmd+K`
6. 主操作按钮：`+ 新建工单`
7. 实时状态区：SSE 在线状态、运行中 Agent 数、待审批数
8. 用户菜单：个人设置、主题、退出

**顶栏交互细节：**

| 元素 | 交互要求 |
|------|---------|
| 组织选择器 | 支持搜索、最近访问、固定组织，切换后刷新项目列表与组织级仪表盘 |
| 项目选择器 | 仅展示当前组织下项目，支持最近项目、收藏项目、按健康状态排序 |
| Cmd+K | 可搜索页面、工单、项目、Workflow、命令动作 |
| SSE 状态 | 正常时弱提示；断开时顶部出现细条警告并显示“正在重连...” |
| 新建按钮 | 支持下拉：新建工单 / 新建项目 / 新建 Workflow |

**项目选择器推荐展示字段：**

- 项目名
- 健康状态（Healthy / Warning / Blocked）
- 运行中工单数
- 最近活动时间

### 13.5 左侧边栏（Sidebar）设计

侧边栏是整个产品的骨架。要求像 Linear 一样稳定、克制、可折叠，但比 Linear 多一层“工程控制台”属性。

**全局侧边栏结构：**

```text
[OpenASE]

Dashboard
我的待处理
审批中心

Projects
  Alpha Platform
  Mobile App
  Infra

Pinned
  本周高优项目
  成本异常项目

Settings
```

**进入项目后的项目侧边栏结构：**

```text
Project: Alpha Platform

Overview
Board
Tickets
Workflows
Agents
Approvals
Activity
Insights
Settings
```

**项目内导航说明：**

| 入口 | 说明 |
|------|------|
| Overview | 项目 Dashboard，作为项目默认入口 |
| Board | 多列看板，默认工作视图 |
| Tickets | 列表视图，适合批量筛选和搜索 |
| Workflows | Workflow 列表、Harness 编辑器、版本历史 |
| Agents | Agent 定义、Provider 配置、运行中的 AgentRun |
| Activity | 审计日志、系统事件、Hook 执行记录 |
| Insights | 成本、吞吐、成功率、重试趋势 |
| Settings | 项目配置、Repo、状态列、通知、机器 |

**侧边栏设计细节：**

- 当前项目名称固定置顶，下面显示项目健康状态和最近更新时间
- 当前页面高亮使用左侧细色条 + 背景填充，不用夸张色块
- 待审批数、运行中 Agent 数、失败告警数以小 Badge 形式显示在导航项右侧
- 侧边栏支持收起为窄栏，保留图标和 Tooltip
- 宽屏下常驻，窄屏下通过抽屉展开

### 13.6 Dashboard 作为默认入口

**Dashboard 是系统默认入口。** 分为组织级 Dashboard 和项目级 Dashboard 两层。

#### 13.6.1 组织级 Dashboard

用户进入系统后，如果尚未聚焦某个项目，默认看到组织级 Dashboard，用于判断“团队整体现在是否健康”。

建议面板：

| 面板 | 展示信息 |
|------|---------|
| Project Health | 所有项目健康度列表：Healthy / Warning / Blocked |
| Running Now | 当前运行中的 AgentRun 数、活跃工单数、待审批数 |
| Delivery Funnel | 本周创建工单、完成工单、平均周期、PR 合并率 |
| Exceptions | Hook 失败、重试暂停、机器离线、预算超限 |
| Recent Activity | 最近 20 条跨项目活动流 |
| Cost Snapshot | 今日 / 本周成本、Top 项目、Top Agent |

**组织级 Dashboard 线框：**

```text
┌─────────────────────────────────────────────────────────────────────┐
│ Dashboard                                            今日 / 本周切换 │
├───────────────────────┬───────────────────────┬─────────────────────┤
│ Projects Health       │ Running Now           │ Cost Snapshot       │
│ Alpha  Healthy        │ 12 Agents Running     │ Today: $24.3        │
│ Infra  Warning        │ 37 Active Tickets     │ Week:  $183.5       │
│ Mobile Blocked        │ 4 Approvals Pending   │ Top: Alpha          │
├───────────────────────┴───────────────────────┬─────────────────────┤
│ Delivery Funnel                                │ Exceptions         │
│ Created / Done / Merge Rate / Avg Cycle Time   │ Hook Failed: 3     │
│                                                │ Budget Alert: 2    │
├────────────────────────────────────────────────┴─────────────────────┤
│ Recent Activity                                                        │
└─────────────────────────────────────────────────────────────────────┘
```

#### 13.6.2 项目级 Dashboard（Project Overview）

当用户选择具体项目后，默认进入项目级 Dashboard，而不是直接掉进某个配置页。

建议面板：

| 面板 | 说明 |
|------|------|
| 项目状态摘要 | 项目健康状态、已绑定 Repo 数、活跃状态列数 |
| Board Snapshot | 各列工单数量 + 风险提示 |
| Active Agents | 正在跑的 Agent、当前处理工单、运行时长 |
| PR / Hook 状态 | 多 Repo PR 概览、最近 Hook 失败 |
| Approval Queue | 待审批工单、等待时长 |
| Cost & Throughput | 近 24h 成本、完成数、平均 attempt |
| Activity Feed | 最近工单创建、状态变更、Agent 完成、Webhook 回写 |

**项目 Dashboard 的价值：**

- 先回答“项目现在是否正常运转”
- 再引导用户进入 Board 进行具体调度
- 异常优先暴露，避免用户必须点很多层才能发现问题

### 13.7 看板（Board）设计

Board 是项目内最核心页面，优先级高于表格列表页。它必须同时满足“直观拖拽”和“高信息密度”。

#### 13.7.1 页面结构

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ Board                                  过滤器  分组  视图切换  + 新建工单   │
├────────────────────────────────────────────────────────────────────────────┤
│ Workflow: All  Agent: All  Priority: All  只看异常 [开关]                    │
├────────────┬────────────┬────────────┬────────────┬────────────┬───────────┤
│ Backlog(8) │ Todo(12)   │ In Prog(4) │ Review(3)  │ Done(27)   │ Cancelled │
│ status tip │ WIP 12      │ 2 Agents    │ 1 blocked  │ today +5   │           │
├────────────┼────────────┼────────────┼────────────┼────────────┼───────────┤
│ ASE-42     │ ASE-51      │ ASE-38      │ ASE-33     │ ASE-11     │           │
│ Fix login  │ Add audit   │ Reconnect   │ Review PR  │ Export CSV │           │
│ high · bug │ coding      │ agent: c-1  │ 2 PRs      │ merged     │           │
│ hook fail  │ backend     │ 3m ago      │ waiting    │            │           │
└────────────┴────────────┴────────────┴────────────┴────────────┴───────────┘
```

#### 13.7.2 列（Column）设计

当前阶段不在 shell 顶栏暴露 Global Search。只有在统一后端搜索契约和可用结果流落地后，才恢复这一入口；在此之前保留页面内局部筛选即可。

每列顶部展示：

- 状态名 + 图标 + 颜色
- 工单数量
- 可选 WIP 信息
- 当前列风险摘要，例如“2 个重试中”“1 个待审批”

列头支持操作：

- 重命名列
- 修改颜色 / 图标
- 设为默认状态
- 添加前后列
- 删除列
- 查看该列筛选视图

#### 13.7.3 卡片（Ticket Card）设计

单张工单卡片必须让用户在 1-2 秒内判断“这是什么、谁在处理、是否有风险”。

**建议卡片字段：**

| 信息 | 是否默认展示 | 说明 |
|------|-------------|------|
| `ASE-42` + 标题 | 必须 | 一级识别信息 |
| 优先级 | 必须 | urgent/high 用颜色点强调 |
| Workflow / 角色 | 必须 | coding / testing / security |
| 当前 Agent | 有则显示当前驱动该工单的 Agent 定义 | `codex-coding` / `claude-reviewer` |
| Repo/PR 链接 | 有则显示 | `2 repos`, `1 PR linked` |
| 异常状态 | 有则高亮 | `重试中`, `Hook failed`, `待审批`, `预算耗尽` |
| 更新时间 | 建议显示 | `2m ago` |

**卡片交互：**

- 单击：打开右侧 Ticket 详情抽屉，不离开看板
- 双击或 `Enter`：进入完整 Ticket 详情页
- 拖拽：改变状态列
- 悬浮：显示更多 meta，如创建者、依赖、最近活动摘要

#### 13.7.4 拖拽交互

拖拽体验必须干净明确，不能出现“不知道是否成功”的情况。

| 场景 | 交互反馈 |
|------|---------|
| 开始拖拽 | 卡片轻微抬起、阴影加深、原位置显示占位轮廓 |
| 拖到目标列 | 目标列头高亮，列内插入占位位置 |
| 拖拽提交成功 | 卡片落位，列计数即时更新，顶部出现短暂 toast |
| SSE 同步更新 | 非当前用户的变动使用柔和闪烁高亮 1 次，不做强动画 |
| 拖拽失败 | 卡片回弹，显示错误原因，例如“无权限”或“状态不存在” |

#### 13.7.5 看板辅助能力

- 顶部快速筛选：Workflow、Agent、优先级、标签、异常状态
- 视图切换：Board / List / My Tickets
- 保存筛选视图：例如“只看 Hook 失败”和“只看待审批”
- 支持横向滚动 + sticky 列头
- 支持空列占位文案，例如“暂无工单，拖进来开始”

### 13.8 Ticket 详情体验

看板不是终点。用户需要在不失去上下文的情况下钻取细节，因此 Ticket 详情采用“抽屉优先，页面次之”的模式。

**打开方式：**

- 从看板卡片单击进入右侧详情抽屉
- 从搜索结果、活动流、通知点击可直达完整详情页

**右侧抽屉建议结构：**

| 模块 | 内容 |
|------|------|
| Header | 工单标识、标题、状态、优先级、主操作按钮 |
| Summary | 描述、Workflow、当前 Agent 定义、涉及 Repo |
| Execution | Agent 实时输出、运行时长、attempt_count、成本 |
| Hooks | `on_claim` / `on_complete` / `on_done` 历史 |
| PRs | 多 Repo PR 列表、状态、链接 |
| Dependencies | 父子工单、阻塞关系 |
| Activity | 时间线、评论、系统事件 |

#### 13.8.1 Ticket 时间线（GitHub Issue 风格）

Ticket Detail 的信息组织应接近 GitHub Issue，而不是“上面一块 description，下面一块 comments，再下面一块 activity”的散装堆叠。主视图必须是**一个统一时间线**：

1. 顶部先显示 Ticket description 作为首条根条目。
2. 根条目之后按时间正序展示 comment 与 system activity。
3. 底部提供新的 comment 输入区。

这条时间线是 Ticket Detail 的第一阅读路径。

**布局契约：**

- Header：Ticket 标题、状态、优先级、主操作按钮
- Root Description Entry：作者、创建时间、正文 Markdown、编辑状态
- Timeline：按时间正序混排 comment 与 activity
- Composer：位于时间线底部，提交新 comment
- 辅助区可存在 Dependencies / Repositories & PRs / Hook History 等模块，但不能替代主时间线

**Description 条目要求：**

- 永远显示在最前面，语义上等价于 “author opened this ticket”
- 卡片必须展示：
  - 作者
  - 创建时间
  - 正文 Markdown
  - 若被编辑则显示 `edited`
- Description 可单独进入 edit flow，但它不是普通 comment，不能被 delete

**Comment 条目要求：**

- comment 与 GitHub issue comment 一样，是时间线中的一级条目
- 每条 comment 必须展示：
  - 作者
  - 发布时间
  - 正文 Markdown
  - `edited` 标记
  - 最近编辑时间
  - 操作按钮：`Edit` / `Delete` / `History` / `Collapse`
- `History` 打开后应看到该 comment 的历史版本，至少显示：
  - revision number
  - edited by
  - edited at
  - body snapshot
- `Collapse` 只影响当前 detail surface 的阅读态，不改变数据

**System Activity 条目要求：**

- system activity 也是时间线中的一级条目，但视觉上比 comment 更轻量
- System Activity 必须直接展示 canonical `ActivityEvent.event_type` 对应的业务事实，不允许 UI 自造别名
- 典型事件包括：
  - `ticket.created` / `ticket.status_changed` / `ticket.completed`
  - `agent.claimed` / `agent.launching` / `agent.ready` / `agent.failed` / `agent.completed`
  - `hook.started` / `hook.passed` / `hook.failed`
  - `pr.opened` / `pr.merged` / `pr.closed`
- activity 条目至少展示：
  - 图标或类型标识
  - 摘要文本
  - 时间
  - 必要时的 metadata（如状态 `from/to`、agent 名称、PR 链接）
- activity 默认不可编辑删除，但可以折叠长 metadata

**编辑与历史要求：**

- comment 编辑必须保留历史版本，不能覆盖旧正文
- 时间线主视图默认只展示最新版本；历史版本通过 `History` 查看
- comment 删除必须有明确 destructive affordance；删除后时间线保留占位，不打乱顺序
- 若 comment 或 description 被编辑，UI 必须显示 `edited` 与对应时间，而不是只显示最新 `updated_at`

**Markdown 约束：**

- 支持常用 Markdown：段落、标题、列表、链接、引用、行内代码、代码块
- 渲染输出必须经过前端安全清洗，禁止脚本和危险属性注入
- 当前阶段不要求 @mention、reaction、附件、resolved thread

**实时刷新要求：**

- 新 comment、comment edit、comment delete、activity append 后，已打开的 Ticket Detail 必须原地刷新
- 刷新语义应以时间线条目为单位，而不是整页闪烁重载
- 评论历史更新后，当前 comment 的 `edited` 标记、编辑时间、history count 必须同步更新

**验收标准：**

- 从 Board 或 Tickets 打开任意 Ticket Detail，顶部先看到 description 根条目
- description 下方看到按时间顺序排列的 comment 和 system activity
- 新增 comment 后，条目立即追加到底部时间线
- 已有 comment 可以 edit / delete / history / collapse
- comment 被编辑后，时间线显示 `edited` 与编辑时间，并可查看历史版本
- system activity 与 comment 共处同一时间线，但视觉层级明显不同
- 整体阅读顺序接近 GitHub Issue，而不是拆成多个互相割裂的面板

### 13.9 Workflows 与 Harness 编辑器设计

Workflow 管理页面是项目的“控制规则中心”，要像 IDE 一样高效，而不是像普通 CMS 文本框。

**页面布局：**

- 左侧：Workflow 列表
- 中间：Harness 编辑区
- 右侧：变量字典 / AI 助手 / 预览面板 Tab

**Workflow 列表字段建议：**

- Workflow 名称
- 角色名称
- 绑定的 Agent 定义
- 绑定的 Provider
- pickup / finish 状态
- 最近修改时间
- 最近运行结果（过去 24h 成功率）

**Workflow 基础配置必须支持：**

- 选择绑定的 Agent 定义
- 在选择 Agent 后只读展示其 Provider / Machine / Model
- 同时展示该 Agent 的 Ticket Workspace 约定（由平台推导，不可手填）
- 修改 pickup / finish 状态
- 修改并发和超时限制

前端不提供“为 Workflow 临时手选一个正在空闲的 Agent 实例”这种交互。Workflow 绑定的是静态 Agent 定义；真正的运行占位由编排引擎在执行时创建 AgentRun。

**Harness 编辑器必须支持：**

- YAML Frontmatter 和 Markdown 双区语法高亮
- 自动补全变量、过滤器、片段
- 预览真实渲染结果
- Diff 比较历史版本
- 未定义变量警告
- AI 辅助 patch 应用

### 13.10 Agents 页面设计

Agent 页面分成“定义视角”和“运行视角”两层，避免把静态 Agent 配置和瞬时运行状态混在一起。

#### 13.10.1 Agent 运行页

展示“现在有哪些 AgentRun 在跑、跑得怎么样”：

| 字段 | 说明 |
|------|------|
| Agent 名称 | 如 `codex-coding` |
| Provider / Model | Claude Code / Codex / Gemini |
| 当前状态 | launching / ready / executing / errored / stalled |
| 当前工单 | 若正在执行，显示 Ticket 标识 |
| 最近心跳 | 用于判断健康 |
| 关联 Workflow | 标识本次运行由哪个 Workflow 触发 |
| 今日成本 | 运营指标 |

支持操作：

- 查看实时输出
- 终止当前运行
- 查看最近失败运行

#### 13.10.2 Agent 定义页

这是你提到的“agent 配置等侧边栏入口”对应页面，位于项目 Settings 下的一级入口，也可从项目侧边栏直接进入。

建议分组：

| 分组 | 内容 |
|------|------|
| Providers | Claude Code / Codex / Gemini CLI 接入配置 |
| Defaults | 默认 Provider、默认模型、并发上限 |
| Agent Definitions | Agent 名称、绑定 Provider、对应 Machine、被哪些 Workflow 使用 |
| Execution Defaults | 默认 Provider、并发约束、后续可扩展的执行策略入口 |
| Budget | 单 Agent / 单项目预算上限 |
| Safety | 审批策略、允许的平台操作范围 |

### 13.11 项目 Settings 信息架构

项目配置页不要做成一个巨大长表单，而是做成二级导航结构：

```text
Settings
  General
  Repositories
  Status Columns
  Workflows
  Agents
  Notifications
  Machines
  Security
```

**每个配置页的共同设计原则：**

- 左侧为小导航，右侧为表单或列表
- 危险操作与普通配置分区
- 所有“测试连接”“验证配置”操作原位完成，不跳页
- 保存后即时反馈，并在必要时提示“服务将重启”或“新配置仅对新工单生效”

当前垂直切片里，`Security` 先交付为项目级安全姿态页：展示 Agent scoped token、Webhook 签名校验与已落地的 secret redaction 边界；人类登录/OIDC、RBAC 与更完整治理面板继续延后到专门控制面实现。

### 13.12 实时状态与反馈设计

OpenASE 是实时系统，必须把“系统正在发生什么”传达清楚，但不能让页面闪烁不停。

**实时反馈规则：**

| 场景 | UI 表现 |
|------|---------|
| Agent 开始执行 | 卡片出现运行中状态点，Agent 页对应行变为 active |
| Agent 输出新日志 | 详情抽屉输出流自动追加，默认跟随到底部，可暂停自动滚动 |
| Hook 失败 | 卡片角标标红；详情页 Hook Tab 自动出现失败提示 |
| 审批生成 | 顶栏待审批数 +1；Approvals 导航项出现 badge |
| SSE 断开 | 顶栏出现重连状态，不阻塞当前浏览 |
| 数据延迟 | 右上角显示“最后同步于 12s 前”弱提示 |

**禁止的做法：**

- 每次 SSE 更新都整列重渲染闪烁
- 用大量 toast 取代页面内状态提示
- 将错误只写在控制台，不在界面标识

### 13.13 键盘优先与效率设计

参考 Linear，OpenASE 应天然支持高频键盘操作：

| 快捷键 | 动作 |
|-------|------|
| `Cmd+K` / `Ctrl+K` | 打开命令面板 |
| `C` | 新建工单 |
| `G` `B` | 跳转到 Board |
| `G` `W` | 跳转到 Workflows |
| `G` `A` | 跳转到 Agents |
| `[` / `]` | 在上一个 / 下一个工单之间切换 |
| `Esc` | 关闭右侧抽屉 / 取消操作 |
| `.` | 打开工单操作菜单 |

### 13.14 响应式与移动端策略

OpenASE 主要面向桌面端，但必须保证移动端可查看关键状态，不要求在手机上完成全部重度操作。

| 设备 | 策略 |
|------|------|
| Desktop（>=1280px） | 三栏完整布局：左导航 + 主工作区 + 右抽屉 |
| Laptop（1024-1279px） | 默认两栏：左导航可收起，右抽屉覆盖式展开 |
| Tablet（768-1023px） | 侧边栏改为抽屉，看板支持横向滑动 |
| Mobile（<768px） | 只保留 Dashboard、Board 浏览、Ticket 详情、审批处理；复杂配置建议提示到桌面端完成 |

### 13.15 可访问性与可读性要求

- 键盘可达：所有关键按钮、卡片、抽屉、切换器都能用键盘访问
- 颜色不能是唯一信号：错误和成功状态同时使用图标/文案
- 拖拽要有非拖拽备选：可以通过详情抽屉中的状态下拉修改状态
- SSE 更新要对屏幕阅读器友好，重要状态变更通过 aria-live 低频播报

### 13.16 核心页面清单（修订版）

| 页面 | 定位 | 关键内容 |
|------|------|---------|
| Dashboard | 默认入口 | 项目健康、运行中 Agent、异常、成本、活动 |
| 项目 Overview | 项目入口 | 项目状态摘要、Board Snapshot、PR/Hook/审批概览 |
| Board | 核心工作台 | 多列状态、拖拽、筛选、实时更新 |
| Ticket 详情 | 深度工作区 | 描述、执行、Hooks、PR、依赖、活动 |
| Workflows | 规则中心 | Workflow 列表、Harness 编辑器、历史、预览 |
| Agents | 运行控制台 | Agent 定义、运行中的 AgentRun、心跳、输出、Provider 配置 |
| Approvals | 风险治理 | 待审批列表、上下文、通过/拒绝 |
| Activity | 审计与回放 | 跨实体事件时间线 |
| Insights | 运营分析 | 成本、吞吐、完成率、重试、SLA |
| Settings | 配置中心 | 项目、Repo、状态列、通知、机器、安全 |

---

## 第十四章 新用户 Onboarding 设计

一个工具再强大，如果新用户在第一个小时内无法体验到价值，就会被放弃。OpenASE 的 onboarding 核心原则：**下载二进制 → 双击运行 → 浏览器自动打开 → 在网页上完成所有配置 → 5 分钟内看到第一个工单被 Agent 处理。**

### 14.1 启动流程：Binary-first + Web Setup Wizard

所有用户（无论个人还是团队）的第一步都一样：

```bash
# 下载并运行（仅此一步）
./openase up
```

**启动序列：**

1. `openase up` 检测 `~/.openase/config.yaml` 是否存在
2. **首次运行**（无 config）：启动 Setup Wizard 模式——仅启动一个轻量 HTTP 服务，监听 `localhost:19836`，自动打开浏览器
3. **已有配置**：正常启动 API + 编排引擎 + 前端，自动打开浏览器进入主界面

首次运行时终端输出：

```
$ ./openase up

  ╔═══════════════════════════════════════════════╗
  ║  OpenASE — 首次启动配置                        ║
  ║                                               ║
  ║  请在浏览器中完成配置:                           ║
  ║  → http://localhost:19836/setup               ║
  ║                                               ║
  ║  浏览器已自动打开。如未打开请手动访问上述地址。     ║
  ╚═══════════════════════════════════════════════╝

  等待配置完成...
```

### 14.2 Web Setup Wizard（浏览器引导）

浏览器打开后，用户看到一个分步引导界面（类似 Notion 或 Linear 的首次使用体验）：

#### 14.2.1 Setup Wizard 的整体页面结构

Setup Wizard 不沿用主应用的三栏 App Shell，而是使用**聚焦单任务的居中向导布局**。目标不是“展示功能全貌”，而是让用户在 5 分钟内无痛完成首次配置。

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ OpenASE                                                      Step 2 of 4   │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  左侧：引导说明 / 当前步骤价值                                              │
│  - 为什么需要这一步                                                         │
│  - 推荐默认值                                                               │
│  - 常见问题 / 帮助链接                                                       │
│                                                                            │
│  右侧：当前步骤表单卡片                                                      │
│  - 输入项                                                                    │
│  - 实时校验                                                                  │
│  - 测试连接 / 自动检测                                                       │
│  - 上一步 / 下一步                                                            │
│                                                                            │
│  底部：返回 / 跳过（如果允许） / 下一步 / 保存并继续                         │
└────────────────────────────────────────────────────────────────────────────┘
```

**页面呈现原则：**

- 宽度控制在 1100-1200px 内，避免表单横向过宽
- 当前步骤的操作区放在视觉重心右侧，帮助信息放在左侧
- 顶部固定 Stepper，用户始终知道自己处于第几步、还剩几步
- 背景比主应用更干净，弱化导航感，强化“安装向导”感
- 每一步只解决一个问题，不在同一步里塞太多设置项

**Stepper 设计：**

| 状态 | UI 表现 |
|------|---------|
| 未开始 | 低对比度圆点 + 标题 |
| 当前步骤 | 强调色圆点 + 标题高亮 + 简短描述 |
| 已完成 | 勾选图标 + 可点击回看 |
| 有错误 | 错误色圆点 + 下方错误摘要 |

#### 14.2.2 视觉风格与文案语气

Setup Wizard 的风格要延续第十三章的 Linear-like 基调，但比主应用更友好、更温和：

- 标题更大，说明文案更少，减少认知负担
- 表单字段之间留白稍大于主应用，营造“从容可配置”的节奏
- 每一步只保留一个主 CTA，避免并列主按钮
- 空状态、检测中、成功、失败都要给出明确文本，不让用户猜

**文案语气要求：**

- 用“下一步会发生什么”取代“技术细节堆砌”
- 尽量告诉用户影响，例如“测试连接成功后会自动初始化数据库 schema”
- 报错时先说问题，再给建议，不只抛底层错误

#### 14.2.3 Step 1 — 选择模式的交互设计

第一步不是下拉框，而是三张可点击的模式卡片，让用户立刻感受到“这套产品理解不同使用者的差异”。

**Step 1/4 — 选择模式**

```
🚀 欢迎使用 OpenASE

你打算怎么用？

  [个人开发者]     — 我一个人，有个项目想让 Agent 帮忙
  [团队]          — 我们几个人，想一起用
  [企业 Pilot]    — 需要 SSO、审批、审计
```

选择不同模式会决定后续步骤的复杂度和默认值。

**页面呈现建议：**

- 顶部标题：`欢迎使用 OpenASE`
- 副标题：`先告诉我们你的使用场景，我们会调整默认配置和后续步骤`
- 下方 3 张模式卡片横向排列，卡片内包含：
  - 模式名称
  - 一句话描述
  - 适合谁
  - 预置的 Provider / Workflow 模板
- 右下角主按钮默认禁用，选中卡片后才可继续

**模式卡片交互：**

| 行为 | 反馈 |
|------|------|
| 悬浮 | 边框增强、背景轻微提亮 |
| 选中 | 强调色边框 + 右上角勾选 |
| 切换模式 | 右侧说明区更新“将为你启用哪些默认值” |

**推荐补充说明文案：**

- 个人开发者：默认最少步骤，优先尽快跑通第一个项目
- 团队：强调多成员协作、Agent 管理、项目治理
- 企业 Pilot：强调 SSO、审批、审计、预算控制

#### 14.2.4 Step 2 — 数据库配置的交互设计

**Step 2 — 数据库配置（所有模式）**

```
📦 数据库

OpenASE 需要连接一个 PostgreSQL 16+ 实例。

  ┌─────────────────────────────────────┐
  │ Host:     localhost                 │
  │ Port:     5432                      │
  │ Database: openase                   │
  │ Username: openase                   │
  │ Password: ••••••••                  │
  │                                     │
  │         [测试连接]  [下一步 →]        │
  └─────────────────────────────────────┘

  还没有 PostgreSQL？一行命令即可启动：

  docker run -d --name openase-pg \
    -e POSTGRES_USER=openase \
    -e POSTGRES_PASSWORD=openase \
    -e POSTGRES_DB=openase \
    -p 5432:5432 \
    -v openase-pgdata:/var/lib/postgresql/data \
    postgres:16-alpine
                                    [复制命令]
```

OpenASE 自身不管理数据库——数据库是用户的基础设施。Setup Wizard 只负责连接和验证。点击"测试连接"实时验证可达性，通过后自动运行 schema 迁移。

**设计理由**：不内嵌数据库，不搞双驱动兼容。OpenASE 只维护自己的单二进制，PostgreSQL 由用户自行管理（已有实例直接连、没有的话 Wizard 页面给一行 Docker 命令复制粘贴）。这样 ent schema、迁移脚本、LISTEN/NOTIFY、JSON 查询永远只有一条代码路径。

**页面呈现建议：**

- 右侧主卡片为数据库表单
- 左侧辅助区展示：
  - 为什么需要 PostgreSQL
  - 推荐版本要求（16+）
  - 本地快速启动命令
  - “你也可以连接已有实例”
- Docker 启动命令默认折叠，点击“我还没有 PostgreSQL”再展开，避免压迫已准备好的用户

**表单交互要求：**

| 字段 | 交互 |
|------|------|
| Host / Port / DB / Username / Password | 实时校验必填与格式，错误在字段下内联提示 |
| Password | 默认隐藏，可切换可见 |
| 测试连接 | 明确异步状态：`测试中...` → `连接成功` 或失败原因 |
| 下一步 | 未测试成功前禁用，避免带着坏配置进入下一步 |

**测试连接成功后的体验：**

1. 先显示 `数据库连接成功`
2. 自动展示二级状态：`正在初始化 schema...`
3. 成功后出现绿色提示：`已完成数据库准备，可以继续`

**错误反馈分层：**

| 错误类型 | 向用户展示 |
|---------|-----------|
| 网络不可达 | `无法连接到数据库地址，请检查 Host/Port 或网络策略` |
| 认证失败 | `用户名或密码错误，请重新检查凭据` |
| 版本不兼容 | `检测到 PostgreSQL 版本过低，需要 16+` |
| schema 初始化失败 | `连接成功，但初始化失败，可查看详细错误日志` |

#### 14.2.5 Step 3 — Agent 检测与配置的交互设计

**Step 3 — Agent 配置（团队/企业模式独立步骤；个人模式在项目步骤中一并确认）**

```
🤖 配置你的 AI Agent

OpenASE 检测到以下已安装的 Agent CLI:

  ✅ Claude Code (claude-sonnet-4-6)    [设为默认]
  ✅ OpenAI Codex                       [添加]
  ⬚  Gemini CLI — 未检测到

(可以稍后在设置中添加更多 Agent)
```

后台自动扫描 PATH 检测可用的 Agent CLI，用户只需确认。

**页面呈现建议：**

- 主体是“检测结果列表”而不是纯表单
- 每个 Agent CLI 一张状态行卡片，展示：
  - CLI 名称
  - 检测结果（已安装 / 未检测到 / 版本异常）
  - 推荐模型
  - 是否设为默认
  - 额外配置入口

**交互细节：**

| 场景 | 交互 |
|------|------|
| 页面初次打开 | 先显示 skeleton + `正在检测已安装的 Agent CLI...` |
| 检测到可用 CLI | 自动高亮推荐项，并解释推荐理由 |
| 未检测到任何 CLI | 页面顶部显示警告，但允许继续进入后续步骤；最终启动前再次提醒 |
| 设为默认 | 单选切换，右侧同步更新“默认执行提供商”摘要 |

**团队/企业模式下增加的设置：**

- 默认并发上限
- 每个 Provider 的启用/禁用
- 预算提示阈值

#### 14.2.6 Step 4 — 首个项目创建的交互设计

**Step 4 — 第一个项目（个人模式仅此步骤；团队/企业模式为 Step 4/5）**

```
📋 创建你的第一个项目

  项目名称:  [my-awesome-app           ]

  Git 仓库:
  ┌──────────────────────────────────────────────────────────────┐
  │  backend   https://github.com/you/backend.git      [Go]      │
  │  frontend  https://github.com/you/frontend.git     [TS]      │
  │                                                    [添加仓库]│
  └──────────────────────────────────────────────────────────────┘

  已检测到仓库语言: Go, TypeScript
  → 已自动选择匹配的 Harness 模板（coding + testing）

         [完成配置，启动 OpenASE →]
```

**Repo 语义说明：** 项目可以关联一个或多个代码仓库，但平台不再定义仓库优先级。任何需要代码上下文的行为都必须显式声明 repo scope，或在单 Repo 项目中自然退化为唯一仓库。Workflow / Harness / Skills 由 OpenASE 控制面持久化，不再要求任何仓库根目录中存在 `.openase/`。多 Repo 工单执行时，编排引擎从平台控制面读取 Workflow / Skills，再把它们 materialize 到本次运行的工作区。

**页面呈现建议：**

- 页面顶部摘要条显示：`你已经完成数据库和 Agent 配置，现在只差一个项目`
- 项目名称输入框置顶
- 仓库列表作为可编辑表格或卡片列表展示
- 检测所有仓库语言与标签后，立即在底部显示“推荐 Harness 模板”
- 对多 Repo 项目，说明“后续工单会按 repo scope 显式选择涉及哪些仓库”

**仓库输入交互：**

| 行为 | 反馈 |
|------|------|
| 输入 Git URL | 实时校验 URL 格式 |
| 添加仓库 | 新行插入，自动 focus 到 URL 输入框 |
| 删除仓库 | 二次确认仅在删除唯一仓库时触发 |
| 语言检测成功 | 仓库行上显示语言 badge，如 `Go`, `TypeScript` |

**推荐模板反馈：**

- 检测到单 Go 仓库项目 → 推荐 `coding + testing`
- 检测到前后端混合仓库项目 → 推荐 `coding + testing + docs`
- 若无法检测语言 → 使用通用模板，并明确提示可稍后调整

#### 14.2.7 企业模式附加步骤的页面要求

企业 Pilot 模式新增一步治理配置，不能只是普通表单堆叠，而应聚焦三个问题：

1. 用什么认证方式
2. 哪些操作需要审批
3. 如何控制预算和审计

建议分成三个卡片分区：

- OIDC 配置
- Approval 策略
- 预算与审计策略

每个分区都要有“推荐默认值”和“稍后可在 Settings 中修改”的文案，降低首次配置阻力。

#### 14.2.8 底部导航与跨步骤行为

所有步骤的底部导航必须一致：

- 左侧：`上一步`
- 右侧：主按钮 `下一步` / `完成配置`
- 允许跳过的步骤显示次按钮 `稍后配置`

**跨步骤规则：**

- 用户返回上一步时保留已输入内容
- 已完成步骤可通过顶部 Stepper 点击返回编辑
- 如果当前步骤存在未保存改动，离开前弹出轻量确认
- 浏览器刷新后可从本地草稿恢复当前步骤

#### 14.2.9 异步状态、加载与失败恢复

Setup Wizard 中存在多处异步动作：测试数据库、检测 Agent、识别仓库语言、初始化平台内置 Workflow / Skills、安装服务、启动系统。

这些场景必须统一设计，避免每步状态风格不一致。

| 状态 | 表现 |
|------|------|
| idle | 默认静态表单 |
| loading | 按钮 loading + 区域级 spinner/skeleton，不全页遮罩 |
| success | 内联成功提示 + 勾选图标 + 可继续按钮 |
| warning | 黄色提示，允许继续但提醒风险 |
| error | 红色内联错误 + “重试”按钮 + 展开详细日志 |

**失败恢复原则：**

- 用户不需要重复填写已成功的步骤
- 部分成功要明确展示，例如“配置文件已写入，但内置 Workflow 模板初始化失败”
- 对不可自动修复的问题，给出下一步操作建议

#### 14.2.10 完成配置页与跳转过渡

点击“完成配置”后，不应直接黑屏等待，而要展示一个**安装中间页**，让用户看到系统正在完成哪些动作：

```text
OpenASE 正在为你准备工作区

✓ 写入配置文件
✓ 初始化数据库
✓ 检测 Agent CLI
⟳ 初始化平台 Workflow / Skills
⟳ 安装系统服务
○ 正在启动 OpenASE
```

完成后显示成功页：

- 标题：`准备完成，OpenASE 已启动`
- 副标题：`接下来你将进入项目 Dashboard，并引导创建第一个工单`
- CTA 1：`进入 OpenASE`
- CTA 2：`查看生成了什么配置`

如果部分步骤失败但不阻塞进入系统，例如内置 Workflow 模板初始化失败：

- 成功页仍可进入系统
- 但展示黄色提示条：`平台内置 Workflow / Skills 尚未初始化完成，请按提示重试初始化`
- 提供一键重试或展开查看补救步骤

**点击"完成配置"后：**

1. Wizard 将所有配置写入 `~/.openase/config.yaml`
2. 敏感信息（数据库密码、API Key）写入 `~/.openase/.env`（权限 0600）
3. 初始化平台内置 Workflow / Skills 模板与内置 Skill 元数据
   - 这些资产写入 OpenASE 控制面，不写入任何项目仓库
   - 若初始化失败 → 提示用户稍后在 Settings 中重试，不阻塞后续步骤
4. 自动生成并安装 systemd user service（Linux）或 launchd plist（macOS）
5. 启动服务：`systemctl --user enable --now openase`
6. 浏览器自动跳转到主界面 `http://localhost:19836`

### 14.3 服务管理：systemd --user

OpenASE 使用操作系统原生的服务管理器来管理进程生命周期，而非自行实现重启逻辑。

**`openase up` 首次运行时自动安装服务：**

1. Setup Wizard 完成 → 写入配置文件
2. 自动生成 `~/.config/systemd/user/openase.service`（Linux）或 `~/Library/LaunchAgents/com.openase.plist`（macOS）
3. 启动服务：`systemctl --user start openase`
4. 设置开机自启：`systemctl --user enable openase`

**生成的 systemd unit 文件：**

```ini
# ~/.config/systemd/user/openase.service
[Unit]
Description=OpenASE — Auto Software Engineering Platform
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/openase all-in-one
EnvironmentFile=%h/.openase/.env
WorkingDirectory=%h/.openase
Restart=on-failure
RestartSec=3

[Install]
WantedBy=default.target
```

**配置变更后的重启流程：**

用户在 Web UI 设置页面修改数据库连接等核心配置 → 后端写入 `~/.openase/.env` → 调用 `systemctl --user restart openase` → 服务在 3 秒内重启并读取新配置 → 前端自动重连。

对用户来说，改完设置点保存，页面短暂加载后恢复，整个过程不需要碰终端。

**常用命令（对高级用户可见）：**

```bash
openase up              # 首次安装服务并启动（包含 Setup Wizard）
openase down            # 停止服务
openase restart         # 重启服务（等价于 systemctl --user restart openase）
openase status          # 查看服务状态
openase logs            # 查看日志（等价于 journalctl --user -u openase -f）
openase uninstall       # 移除服务 + 清理 ~/.openase/（交互确认）
```

这些命令本质上是 `systemctl --user` 的语义化封装，降低用户心智负担。

**macOS 支持：** macOS 上使用 `launchd` 替代 systemd，`openase up` 自动检测操作系统并生成对应的 `~/Library/LaunchAgents/com.openase.plist`。用户不需要关心底层差异，CLI 命令完全一致。

### 14.4 三种用户的差异化体验

Web Setup Wizard 的 Step 1 选择模式后，后续步骤的默认值和复杂度不同：

| 维度 | 个人开发者 | 团队 | 企业 Pilot |
|------|-----------|------|-----------|
| 向导步骤 | 3 步（模式 → 数据库 → 项目） | 4 步（+ Agent 配置） | 5 步（+ OIDC + 治理策略） |
| 数据库 | PostgreSQL（Wizard 提供 Docker 一行命令） | PostgreSQL（连接已有实例） | PostgreSQL（连接已有实例） |
| Agent 配置 | 自动检测 + 一键确认（跳过独立步骤） | 自动检测 + 可配置并发数（独立步骤） | 自动检测 + 预算上限 |
| 项目配置 | 单 Repo 简化流程 | 多 Repo 向导（显式 repo 列表） | 多 Repo + 治理策略 |
| 认证 | 后台自动生成 Local Token，浏览器透明（同机器访问无需输入 Token，cookie session） | Local Token（需分发给团队成员） | OIDC 配置界面 |
| 额外步骤 | 无 | 团队邀请链接生成 | 审批策略 + Pilot Runbook 下载 |

**个人模式认证说明：** 后端仍然启用 `auth.mode=local`（所有 API 请求需要 Bearer Token），但 Wizard 自动生成一个随机 Token 写入 `~/.openase/.env`，浏览器通过首次 Wizard 流程获得 httpOnly cookie session，后续访问无需手动输入 Token。效果是用户完全无感——看起来"没有认证"，实际后台始终有 Token 保护 API（防止局域网其他设备未经授权访问）。

### 14.5 空 Project 的首次导览体验

空 Project onboarding 是 Setup Wizard 之后的**第二段引导**。Wizard 负责把平台跑起来；空 Project 导览负责把一个刚创建好的项目真正推进到“可以让 Agent 开始工作”的状态。

#### 14.5.1 触发条件与入口

当满足以下条件时，前端必须进入 `empty_project_onboarding` 模式，而不是把用户丢进一个普通但空白的项目 Dashboard：

- 当前用户刚创建了一个新项目，或首次进入一个尚未完成初始化的项目
- 当前项目 `tickets.count == 0`
- 当前项目还未完成首轮 bootstrap（详见下文 checklist）

进入行为必须固定：

- Wizard 成功页点击“进入 OpenASE”后，先进入项目级 Dashboard
- 左侧边栏自动聚焦刚创建的项目
- 顶部显示欢迎条：`项目已创建，接下来我们会带你把它配置到可运行状态`
- 主区域渲染一个**不可忽略的 Onboarding 面板**，按顺序展示：
  1. 配置 GitHub Token
  2. 创建或关联 Repo
  3. 配置至少一个 Provider
  4. 自动创建首个 Agent 与 Workflow
  5. 创建首个 Ticket
  6. 在 Ticket 中观察 Agent 更新
  7. 体验 Project AI 与 Harness AI

这个面板不是“建议列表”，而是**项目空态的主工作区**。用户可以离开页面，但再次回到该项目 Dashboard 时必须从第一个未完成步骤继续。

#### 14.5.2 Step A — GitHub Token 必须先完成

空 Project 导览的第一个硬门槛是 GitHub 出站凭证。没有有效 `GH_TOKEN` 时，不允许继续到 Repo 创建 / 关联步骤。

完成条件：

- 当前项目所在 scope（优先 project 覆盖，否则 organization 默认）存在平台托管的 `GH_TOKEN`
- `GH_TOKEN` probe 状态达到 `valid`
- 用户完成一次 GitHub 身份确认

页面要求：

- Onboarding 面板第一张卡片标题固定为：`连接 GitHub`
- 提供两种主路径：
  - `从本机 gh 自动导入`
  - `手动粘贴 Token`
- 手动粘贴路径必须明确教用户如何拿到 token：
  - 若尚未登录 GitHub CLI：先执行 `gh auth login`
  - 获取当前 token：执行 `gh auth token`
- 如果运行机已检测到 `gh` 且可成功返回 token，UI 提供一键导入按钮；点击后平台按 `gh_cli_import` 语义保存 token
- 如果用户走手动粘贴路径，平台按 `manual_paste` 语义保存 token

保存后的强制动作：

1. 平台立即执行 GitHub probe
2. UI 显示 `正在验证 GitHub 身份与权限...`
3. probe 成功后，UI 必须展示探测出的 GitHub 用户名 / login，例如 `将使用 GitHub 账号 octocat`
4. 用户必须点击一次确认：`这是我要用于该项目的 GitHub 身份`

失败与异常规则：

- 如果 probe 未达 `valid`，步骤保持未完成，后续 Repo 步骤禁用
- 如果用户名与用户预期不符，用户可以重新导入 / 替换 token
- UI 不允许因为 token 字符串非空就把该步骤标成完成，必须以 probe 结果为准

#### 14.5.3 Step B — 引导用户创建或关联 Repo

GitHub Token 完成后，导览自动推进到 Repo 步骤。这里的目标不是“让用户知道项目支持多 Repo”，而是**至少让项目拥有一个真实可访问的代码仓库**。

完成条件：

- 当前项目至少存在 1 个 `ProjectRepo`
- 每个新建或关联的 Repo 都已通过 GitHub 权限与可访问性校验

页面要求：

- 卡片标题固定为：`创建或关联代码仓库`
- 必须同时提供两条路径：
  - `创建新仓库`
  - `关联已有仓库`
- `创建新仓库` 至少要求输入：
  - Repo 名称
  - 可见性（private / public）
  - 默认分支（默认 `main`）
- `关联已有仓库` 至少要求输入：
  - Repo 名称
  - Git URL

交互规则：

- 若用户选择创建新仓库，平台调用 GitHub API 创建 repo 后立即自动注册到项目
- 若用户选择关联已有仓库，平台必须先校验当前 `GH_TOKEN` 对该 repo 具有最小可用访问能力，再落库为 `ProjectRepo`
- 单 Repo 项目允许一步完成；多 Repo 项目允许连续添加多个 Repo
- 如果用户的项目是典型前后端结构，UI 可以推荐 `backend` / `frontend` 两个仓库模板，但推荐不等于自动乱建

这个步骤完成后，Dashboard 应更新：

- 顶部欢迎条显示 repo 连接状态
- Board Snapshot 继续显示空列，但不再提示“先连 repo”，而是推进到 Provider 步骤

#### 14.5.4 Step C — 配置 Claude Code / Codex / Gemini CLI

Repo 完成后，必须引导用户至少配置一个可运行的 Provider。Provider onboarding 是空 Project 导览的第三个硬门槛。

完成条件：

- 当前项目至少有 1 个 Provider 达到 `availability_state = available`
- 用户明确选择其中 1 个作为当前项目的默认执行 Provider

页面要求：

- 卡片标题固定为：`选择并配置 AI Provider`
- 默认展示三张 Provider 卡片：
  - `Claude Code`
  - `OpenAI Codex`
  - `Gemini CLI`
- 每张卡片都必须展示：
  - 是否已检测到 CLI
  - 当前可用性状态（available / unavailable / stale / unknown）
  - 推荐模型
  - 是否已登录
  - `设为默认` 或 `继续配置`

交互规则：

- 页面进入时自动执行本机 PATH 检测与 provider availability 检查
- 如果某个 CLI 已可用，对应卡片显示主按钮 `使用这个 Provider`
- 如果某个 CLI 未配置好，点击卡片后进入 provider-specific 教程抽屉，而不是只报一句“不可用”

provider-specific 教程至少要覆盖：

- Claude Code：
  - 安装 Claude Code CLI
  - 登录 / 认证
  - 如何验证 `claude --version` 与登录状态
- OpenAI Codex：
  - 安装 Codex CLI
  - 登录 / API 认证
  - 如何验证 `codex --version` 与可用性
- Gemini CLI：
  - 安装 Gemini CLI
  - 登录 / API 认证
  - 如何验证 `gemini --version` 与可用性

选择规则：

- 如果已有多个 CLI 可用，用户可以直接点选任一可用卡片
- 没有任何 CLI 可用时，不能继续到 Agent/Workflow 步骤
- 完成选择后，当前项目的默认 Provider 必须被显式设置，不允许留空等待系统猜测

#### 14.5.5 Step D — 根据项目状态自动创建首个 Agent 与 Workflow

至少有一个可用 Provider 之后，导览进入 bootstrap 步骤。这里不要求用户先理解 Harness、Workflow、Agent 的抽象概念，而是要求系统根据 `project.status` 自动生成一套**可立即工作的内置预设**。

完成条件：

- 平台已创建 1 个 Agent
- 平台已创建并发布 1 个 Workflow
- 新建 Workflow 已绑定上述 Agent，且 Agent 已绑定用户刚选中的 Provider

分支规则必须写死如下：

- 当 `project.status = Planned`
  - 推荐角色：`产品经理`
  - 内置 `role_name = product-manager`
  - Workflow 类型：`custom`
  - 默认 pickup 状态：`Backlog`
  - 默认 finish 状态：`Done`
  - 默认 Agent 名称建议：`product-manager-01`
- 当 `project.status = In Progress`
  - 推荐角色：`Coder`
  - 内置 `role_name = fullstack-developer`
  - Workflow 类型：`coding`
  - 默认 pickup 状态：`Todo`
  - 默认 finish 状态：`Done`
  - 默认 Agent 名称建议：`coder-01`

其他项目状态的规则：

- `Backlog`：默认按 `Planned` 处理，推荐 `产品经理`
- `Completed` / `Canceled` / `Archived`：不自动创建执行角色，显示说明，要求用户先变更项目状态

页面交互要求：

- UI 必须明确展示即将创建的对象摘要：
  - Provider：哪一个
  - Agent：名称、并发、是否启用
  - Workflow：角色、pickup、finish、使用的内置 Harness
- 用户点击确认后，平台一次性完成：
  1. 创建 Agent
  2. 创建 Workflow
  3. 绑定 Agent ↔ Provider
  4. 绑定 Workflow ↔ Agent
  5. 立即发布 Workflow 当前版本

这一步必须是“自动建好预设”，不是把用户扔到 Workflow 编辑器里从零配。

#### 14.5.6 Step E — 指导用户创建首个 Ticket，并立即跳到 Ticket 详情观察执行

Agent 与 Workflow 创建完成后，空 Project 导览的下一步才是首个 Ticket。

完成条件：

- 当前项目已创建至少 1 个 Ticket
- 至少有 1 条来自 Agent 的运行中事件被用户在 Ticket 详情页看到

Ticket 创建表单要求：

- 标题输入框自动聚焦
- 默认推荐刚创建的 bootstrap Workflow
- 单 Repo 项目自动带出唯一 repo；多 Repo 项目要求显式选择 repo scope
- 表单底部必须解释：
  - 工单会进入哪个 pickup 状态
  - 编排引擎何时会领取
  - Agent 的更新会出现在哪里

按项目状态给默认示例：

- `Planned` / `Backlog`
  - 默认示例 Ticket：`梳理项目需求并输出第一版 PRD`
  - 默认进入 `产品经理` Workflow 的 pickup 状态
- `In Progress`
  - 默认示例 Ticket：`实现项目的第一个核心功能`
  - 默认进入 `Coder` Workflow 的 pickup 状态

创建后的页面流转必须固定：

1. 创建成功后自动跳到 Ticket 详情页，而不是停留在 Dashboard
2. 首次进入 Ticket 详情时，顶部出现引导条：`Agent 开始工作后，更新会实时出现在这里`
3. 用高亮提示用户看三个区域：
  - Ticket 状态与当前 Workflow
  - Activity / Step 时间线
  - Agent 实时输出区域
4. 只有当系统收到第一条 `AgentStepEvent` 或等价的运行时输出后，这一步才标记完成

CLI 路径仍然保留，但不是主路径：

```bash
openase ticket create --title "Add input validation to login form" --workflow coding
openase watch ASE-1
```

#### 14.5.7 Step F — 让用户立刻体验 Project AI 与 Harness AI

首个 Ticket 已经开始执行后，导览不应立刻结束，而要继续把用户带到 OpenASE 的两个高价值 AI 入口：

- `Project AI`
- `Harness AI`

这里的目标不是再讲概念，而是让用户在同一个项目里完成两个最重要的“下一步动作”。

Project AI 的导览要求：

- 在 Ticket 已出现首条 Agent 更新后，Dashboard 或 project sidebar 显示 CTA：`让 Project AI 帮你继续拆需求`
- 点击后打开 `project_sidebar`
- 首次预置问题建议：
  - `基于当前项目和已有 Ticket，再帮我拆 3 个后续工单`
  - `我下一步应该先做什么？`
- 用户确认后，Project AI 可以通过 `action_proposal` 帮助创建后续 Ticket

Harness AI 的导览要求：

- 当项目已存在刚创建的 bootstrap Workflow 后，显示 CTA：`用 Harness AI 调整这个角色的工作方式`
- 点击后直接打开刚创建的 Workflow 编辑器和 Harness AI 侧栏
- 首次预置问题建议：
  - `帮我优化这个 Workflow，让它更适合当前项目`
  - `帮我给这个角色增加更明确的验收标准`

收尾规则：

- 当用户至少完成一次 Project AI 交互，且至少打开过一次 Harness AI 时，空 Project onboarding 才可以标记为完整完成
- 完成后，Dashboard 从“强引导模式”切换为常规项目 Dashboard

#### 14.5.8 首次进入 Dashboard 的固定布局

在 `empty_project_onboarding` 模式下，项目级 Dashboard 不是普通概览页，而是一个带明确顺序的操作面板。布局必须固定为：

| 区域 | 内容 |
|------|------|
| 顶部欢迎条 | 当前项目名、项目状态、GitHub 连接状态、默认 Provider 状态 |
| Onboarding Checklist | GitHub Token → Repo → Provider → Agent/Workflow → First Ticket → Observe Ticket → Project AI / Harness AI |
| Board Snapshot | 初始列为空，但显示每一列的用途说明 |
| 帮助入口 | GitHub token 帮助、CLI 安装文档、示例 Harness、CLI 示例 |

Checklist 规则：

- 未完成步骤显示主 CTA
- 已完成步骤显示摘要与“重新配置”
- 后续步骤可以看见，但在前置步骤未完成时必须禁用
- 页面刷新或切换项目后，Checklist 必须恢复到当前真实完成进度，而不是丢状态

### 14.6 渐进式解锁

初始体验刻意简化，复杂功能通过"里程碑提示"在 Web UI 中渐进式解锁：

| 里程碑 | 触发条件 | 提示内容 |
|--------|---------|---------|
| 首个工单完成 | 首个工单 done | "第一个工单完成了！试试创建多个工单让 Agent 并行处理？" |
| 5 个工单完成 | 累计 5 个 done | "Harness 已经跑了 5 次。去 Workflow 管理页面优化工作规范？" |
| 首次 Hook 失败 | 任意 Hook 返回非 0 | "on_complete Hook 失败了。查看工单详情中的 Hook 日志。" |
| 10 个工单完成 | 累计 10 个 done | "查看成本仪表盘了解 Agent 的表现和 Token 消耗？" |
| 首个多 Repo 工单 | 涉及 2+ Repo | "多 Repo 工单已创建！Agent 会在联合工作区中同时修改所有仓库。" |
| 成本达到 $10 | 累计 cost > 10 | "已消耗 $10 API 成本。在设置中配置预算上限？" |

### 14.7 `openase doctor` — 环境诊断

当用户遇到问题时，CLI 和 Web UI 都提供诊断入口：

```
$ openase doctor

🔍 OpenASE 环境诊断

  ✅ Git 2.44.0
  ✅ Claude Code 已安装且可用
  ⚠️  Codex 未安装（可选）
  ✅ PostgreSQL 16 已连接 (localhost:5432)
  ✅ ~/.openase/ 配置目录完整
  ✅ 2 个 Workflow Harness 版本已初始化
  ✅ 3 个 Hook 脚本已配置
  ⚠️  on_complete Hook "run-tests.sh" 缺少可执行权限
     → 修复: chmod +x scripts/ci/run-tests.sh

总结: 1 个警告，0 个错误
```

---

## 第十五章 工程规范与 Pre-commit

代码规范不是文档——它是自动执行的。OpenASE 使用 **lefthook**（Go 原生的 Git hooks 管理器，比 Python 的 pre-commit 更快、零外部依赖、与 Go 工具链一致）来强制执行所有规范。

### 15.1 Go 代码规范

**格式化：零讨论空间**

| 工具 | 用途 | 规则 |
|------|------|------|
| `gofmt` | 基础格式化 | Go 标准，不可配置 |
| `goimports` | import 排序 + 自动补全 | 按 stdlib → 外部 → 内部分组，空行分隔 |

```go
import (
    "context"          // stdlib
    "fmt"

    "github.com/labstack/echo/v4"    // 外部依赖
    "go.opentelemetry.io/otel"

    "github.com/BetterAndBetterII/openase/internal/domain/ticketing" // 内部包
    "github.com/BetterAndBetterII/openase/internal/httpapi"
)
```

**Lint：golangci-lint（严格配置）**

```yaml
# .golangci.yml
run:
  timeout: 5m

linters:
  enable:
    # 必选
    - errcheck         # 未处理的 error 返回值
    - govet            # go vet 内置检查
    - staticcheck      # 最全面的静态分析
    - unused           # 未使用的代码
    - gosimple         # 可简化的代码
    - ineffassign      # 无效赋值
    - typecheck        # 类型检查

    # 代码质量
    - gocritic         # 代码风格和性能建议
    - revive           # 可配置的 golint 替代
    - misspell         # 拼写错误（注释和字符串）
    - prealloc         # 可预分配的 slice
    - unconvert        # 不必要的类型转换
    - unparam          # 未使用的函数参数

    # 安全
    - gosec            # 安全问题检测
    - bodyclose        # HTTP response body 未关闭

    # DDD 架构守卫
    - depguard         # 依赖方向约束（核心！）

linters-settings:
  depguard:
    rules:
      domain-no-infra:
        files:
          - "domain/**"
        deny:
          - pkg: "github.com/openase/openase/infra"
            desc: "domain 层禁止依赖 infra 层"
          - pkg: "github.com/openase/openase/api"
            desc: "domain 层禁止依赖 interface 层"
          - pkg: "github.com/openase/openase/app"
            desc: "domain 层禁止依赖 application 层"
      app-no-infra-direct:
        files:
          - "app/**"
        deny:
          - pkg: "github.com/openase/openase/infra"
            desc: "application 层禁止直接依赖 infra 层（通过 interface 注入）"
          - pkg: "github.com/openase/openase/api"
            desc: "application 层禁止依赖 interface 层"

  revive:
    rules:
      - name: exported
        arguments: [checkPrivateReceivers]
      - name: unexported-return
      - name: blank-imports
      - name: context-as-argument
      - name: error-return
      - name: error-naming
      - name: increment-decrement
```

**depguard / architecture guard 是架构守卫的关键**——它在 lint 阶段强制执行当前仓库的依赖方向：`internal/domain` / `internal/types` 不能向上依赖 `repo/service/httpapi/app wiring`，`internal/service` 不能 import `internal/httpapi` / `internal/setup` / `cmd/openase`，并且禁止新的 `ent/*` 混入本该保持纯净的 domain/service 边界。违反就报错，PR 无法合并。

**命名规范**

| 场景 | 规则 | 示例 |
|------|------|------|
| 包名 | 小写单词，不用下划线/驼峰 | `ticket`、`claudecode`、`gitops` |
| 接口 | 动词/名词，不加 `I` 前缀 | `Repository`、`Adapter`、`Executor` |
| 接口实现 | 名词性描述 | `EntTicketRepo`、`ClaudeCodeAdapter` |
| Error 变量 | `Err` 前缀 | `ErrTicketNotFound`、`ErrHookFailed` |
| Context | 第一个参数 | `func (s *Service) Claim(ctx context.Context, ...) error` |
| 构造函数 | `New` + 类型名 | `NewScheduler(cfg Config) *Scheduler` |
| DTO | `XxxRequest` / `XxxResponse` | `CreateTicketRequest`、`TicketDetailResponse` |
| Use-case Input | `XxxInput` / `CreateXxx` / `UpdateXxx` | `PullRequestStatusInput`、`CreateInput`、`UpdateAgentProvider` |
| Domain Event | 过去时 | `TicketClaimed`、`HookFailed`、`AgentStalled` |

**Error 处理**

```go
// ✅ 始终 wrap error，附加上下文
return fmt.Errorf("claim ticket %s: %w", ticketID, err)

// ✅ 领域错误用自定义类型
var ErrTicketAlreadyClaimed = errors.New("ticket already claimed")

// ❌ 不要忽略 error
_ = db.Close()  // golangci-lint errcheck 会拦截

// ❌ 不要 panic（除了 init 阶段真正不可恢复的情况）
```

**测试规范**

| 类别 | 位置 | 命名 | 工具 |
|------|------|------|------|
| 单元测试 | 与被测代码同目录 | `*_test.go` | `testing` + `testify/assert` |
| 集成测试 | `tests/integration/` | `*_integration_test.go` | `testcontainers-go`（PostgreSQL） |
| 架构测试 | 随 golangci-lint | depguard 规则 | `golangci-lint` |

```go
// 单元测试命名：Test_<方法名>_<场景>_<期望>
func Test_StateMachine_ClaimTicket_BlockedByDependency(t *testing.T) { ... }
func Test_StateMachine_CompleteTicket_HookFailure_StaysInProgress(t *testing.T) { ... }
```

### 15.2 前端代码规范（SvelteKit）

**格式化：Prettier**

```json
// .prettierrc
{
  "semi": false,
  "singleQuote": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2,
  "useTabs": false,
  "plugins": ["prettier-plugin-svelte", "prettier-plugin-tailwindcss"],
  "overrides": [
    { "files": "*.svelte", "options": { "parser": "svelte" } }
  ]
}
```

`prettier-plugin-tailwindcss` 自动排序 Tailwind class，团队不需要讨论 class 顺序。

**Lint：ESLint（flat config）**

```js
// eslint.config.js
import svelte from 'eslint-plugin-svelte'
import ts from '@typescript-eslint/eslint-plugin'

export default [
  ...svelte.configs['flat/recommended'],
  {
    rules: {
      // TypeScript 严格模式
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/strict-boolean-expressions': 'warn',

      // Svelte 特定
      'svelte/no-at-html-tags': 'error',       // 防 XSS
      'svelte/require-each-key': 'error',       // {#each} 必须有 key
      'svelte/no-unused-svelte-ignore': 'error',

      // 通用
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'prefer-const': 'error',
    }
  }
]
```

**TypeScript：strict 模式**

```json
// tsconfig.json 关键配置
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "exactOptionalPropertyTypes": true
  }
}
```

**Svelte 文件结构约定**

```svelte
<!-- 1. script（逻辑在最上面） -->
<script lang="ts">
  import { Badge } from '$lib/components/ui/badge'
  import type { Ticket } from '$lib/api/types'

  // props → state → derived → lifecycle → functions
  let { ticket }: { ticket: Ticket } = $props()
  let loading = $state(false)
  let statusLabel = $derived(ticket.status.toUpperCase())

  function handleClaim() { ... }
</script>

<!-- 2. markup（结构在中间） -->
<div class="flex items-center gap-2">
  <Badge variant="outline">{statusLabel}</Badge>
  <button onclick={handleClaim}>Claim</button>
</div>

<!-- 3. style 尽量用 Tailwind，少用 <style> -->
```

**前端目录结构约定**

```
web/src/
├── routes/                    # SvelteKit 页面路由
│   ├── (app)/                 # 需要认证的布局组
│   │   ├── tickets/
│   │   ├── workflows/
│   │   ├── agents/
│   │   └── settings/
│   └── (setup)/               # Setup Wizard 布局组
├── lib/
│   ├── components/
│   │   ├── ui/                # shadcn-svelte 基础组件
│   │   ├── ticket/            # 工单相关业务组件
│   │   ├── agent/             # Agent 相关业务组件
│   │   └── layout/            # Shell, Sidebar, Header
│   ├── api/
│   │   ├── types.ts           # openapi-typescript 生成
│   │   ├── client.ts          # fetch wrapper
│   │   └── sse.ts             # SSE 连接 + Svelte store
│   ├── stores/                # 全局 Svelte store
│   └── utils/
└── app.css                    # Tailwind 入口
```

#### 15.2.1 组件分层与复用规范

前端必须遵循“**从通用到具体**”的组件分层，避免页面直接长成一棵巨型模板树。

| 层级 | 目录建议 | 职责 | 禁止做的事 |
|------|---------|------|-----------|
| UI Primitive 层 | `lib/components/ui/` | Button、Card、Dialog、Tabs、Badge 等基础组件 | 不能包含业务语义和接口请求 |
| App Shell 层 | `lib/components/layout/` | Sidebar、TopBar、PageHeader、RightDrawer、EmptyState | 不直接知道 Ticket/Workflow 业务细节 |
| Feature 组件层 | `lib/features/<feature>/components/` | BoardColumn、TicketCard、AgentList、ProjectHealthPanel | 不跨 feature 随意依赖彼此内部实现 |
| Feature 状态层 | `lib/features/<feature>/stores.ts` | feature 局部状态、筛选、视图模式、SSE 合并逻辑 | 不做 DOM 渲染 |
| Feature 数据层 | `lib/features/<feature>/api.ts` | 调用 API、解析返回、封装 feature query | 不做页面布局 |
| Route 组装层 | `routes/**/+page.svelte` | 拼装 page sections、绑定路由参数、组织 feature | 不承载大量业务实现细节 |

**硬规则：**

- `routes/**/+page.svelte` 只能做“组装”，不能同时承载大块数据模型定义、SSE 协议解析、业务规则分支和复杂渲染细节
- 业务组件优先放入 `lib/features/`，而不是无差别堆到 `lib/components/`
- `ui/` 组件不依赖任何 `feature/` 或 `routes/`
- `feature/` 组件不能 import 其他 feature 的私有文件，只能依赖对方暴露的公开组件/类型

#### 15.2.2 Feature-first 目录设计

随着页面复杂度提升，按“页面文件”组织很快会失控。OpenASE 前端应以 feature 为中心组织目录：

```
web/src/lib/features/
├── board/
│   ├── api.ts
│   ├── stores.ts
│   ├── types.ts
│   ├── mappers.ts
│   └── components/
│       ├── board-view.svelte
│       ├── board-toolbar.svelte
│       ├── board-column.svelte
│       ├── ticket-card.svelte
│       └── board-filters.svelte
├── ticket-detail/
│   ├── api.ts
│   ├── stores.ts
│   ├── types.ts
│   └── components/
├── dashboard/
├── agents/
└── workflows/
```

**设计原因：**

- Board、Ticket Detail、Dashboard、Agents 都是持续演化的大 feature
- 让类型、API、store、组件放在一起，修改时上下文更集中
- 降低“页面文件什么都懂”的倾向

#### 15.2.3 页面规范：Route 文件只做组装，不做大脑

每个页面遵循以下职责切分：

| 文件 | 职责 |
|------|------|
| `+page.ts` / `+layout.ts` | 读取路由参数、做首屏 load、处理 URL search params |
| `+page.svelte` | 组装页面 section，连接 feature store，传递少量 props |
| `lib/features/*/api.ts` | 请求和解析远端数据 |
| `lib/features/*/stores.ts` | 本地状态、筛选器、SSE 合并逻辑 |
| `lib/features/*/components/*.svelte` | 单一视觉区块或交互块 |

**Route 文件禁止出现以下气味：**

- 在 `+page.svelte` 中定义几十个领域类型
- 在 `+page.svelte` 中直接写多个 API fetch 封装
- 在 `+page.svelte` 中直接处理多个 SSE topic 的协议解析
- 在 `+page.svelte` 中同时维护 20+ 个 `$state`
- 一个页面既画所有 UI，又包含所有 mapper、formatter、error handling

#### 15.2.4 推荐的页面拆分方式

以项目 Dashboard 为例，`+page.svelte` 不应该包含所有卡片实现，而应该像这样组装：

```svelte
<script lang="ts">
  import ProjectOverviewHeader from '$lib/features/dashboard/components/project-overview-header.svelte'
  import ProjectHealthPanel from '$lib/features/dashboard/components/project-health-panel.svelte'
  import RunningAgentsPanel from '$lib/features/dashboard/components/running-agents-panel.svelte'
  import ExceptionPanel from '$lib/features/dashboard/components/exception-panel.svelte'
  import ActivityFeedPanel from '$lib/features/dashboard/components/activity-feed-panel.svelte'
  import { createProjectDashboardStore } from '$lib/features/dashboard/stores'

  let { data } = $props()
  const dashboard = createProjectDashboardStore(data.projectId)
</script>

<ProjectOverviewHeader project={dashboard.project} />
<div class="grid gap-4 xl:grid-cols-[2fr,1fr]">
  <ProjectHealthPanel summary={dashboard.summary} />
  <RunningAgentsPanel agents={dashboard.agents} />
  <ExceptionPanel items={dashboard.exceptions} />
  <ActivityFeedPanel items={dashboard.activity} />
</div>
```

**目标：** page 文件读起来像页面目录，不像一篇巨型程序小说。

#### 15.2.5 单文件规模预算（File Budget）

为了避免单页面膨胀，前端引入明确的文件规模预算：

| 文件类型 | 软上限 | 硬上限 | 处理要求 |
|---------|-------|-------|---------|
| `routes/**/+page.svelte` | 150 行 | 250 行 | 超过软上限必须拆 section；超过硬上限禁止合入 |
| `routes/**/+layout.svelte` | 180 行 | 300 行 | 复杂布局逻辑应拆到 layout 组件 |
| `lib/features/**/components/*.svelte` | 200 行 | 300 行 | 超过后拆成更小的 panel / row / item |
| `lib/components/ui/*.svelte` | 150 行 | 250 行 | primitive 组件必须保持克制 |
| `*.ts` store / api 文件 | 200 行 | 300 行 | 过长说明 feature 切分不合理 |
| 单个函数 | 40 行 | 60 行 | 超过后抽 helper 或小函数 |

**例外规则：**

- 极少数复合编辑器组件可超过硬上限，但必须在 PR 中说明理由
- 超限文件需要在 code review 中单独说明为什么不可拆

#### 15.2.6 状态管理规范

防止页面膨胀的关键不只是拆 UI，还要拆状态与数据责任。

**规则：**

- 跨页面共享状态才进入 `lib/stores/`
- 单页面但复杂的状态进入 feature store，不要堆在 route 文件
- SSE 合并逻辑进入 `feature/stores.ts` 或 `lib/api/sse.ts` 边界，不写在卡片组件里
- API 响应在边界解析为领域友好的前端类型，再进入组件
- 展示组件尽量做“纯渲染”，即输入 props，输出 UI，不直接 fetch

**推荐做法：**

- `createBoardStore(projectId)` 统一管理列、卡片、筛选器、拖拽更新、SSE 合并
- `TicketCard.svelte` 只关心如何展示一张卡，不关心数据来自 HTTP 还是 SSE

#### 15.2.7 前端依赖方向

前端也必须遵循与后端一致的依赖纪律：

```text
ui primitives
  → layout
  → features
  → routes
```

以及：

```text
types / mappers
  → api / stores
  → components
  → routes
```

**禁止：**

- route 被 feature 反向依赖
- feature A 直接 import feature B 的 route 文件
- UI primitive import feature store
- 页面组件之间互相 copy-paste 相似逻辑而不抽象公共 section

#### 15.2.8 可以用什么 lint / 工具防止页面失控

**有办法，而且应该多层组合：**

1. `ESLint` 做静态规则
2. `dependency-cruiser` 或 `eslint-plugin-boundaries` 做依赖边界检查
3. 自定义 `check-file-budgets` 脚本做行数预算检查
4. CI 强制阻断超限文件合入

**推荐 ESLint 规则：**

```js
// eslint.config.js 追加规则
import sonarjs from 'eslint-plugin-sonarjs'
import importPlugin from 'eslint-plugin-import'

export default [
  ...svelte.configs['flat/recommended'],
  sonarjs.configs.recommended,
  importPlugin.flatConfigs.recommended,
  {
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: true, skipComments: true }],
      'max-lines-per-function': ['warn', { max: 60, skipBlankLines: true, skipComments: true }],
      'complexity': ['warn', 10],
      'max-depth': ['warn', 4],
      'import/no-cycle': 'error',
      'sonarjs/cognitive-complexity': ['warn', 15],
    }
  },
  {
    files: ['src/routes/**/*.svelte'],
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: true, skipComments: true }]
    }
  }
]
```

**说明：**

- `max-lines` 能检查“文件过长”，对 `.svelte` 文件有效，但它只能发现结果，不能指导怎么拆
- `complexity` 和 `sonarjs/cognitive-complexity` 能发现脚本逻辑过重
- `import/no-cycle` 能阻止页面拆分后形成依赖环

#### 15.2.9 仅靠 ESLint 不够，需要文件预算脚本

ESLint 能检查通用规则，但对“不同目录不同预算”这类团队约束，最好补一层自定义脚本，例如：

```bash
node scripts/check-file-budgets.mjs
```

脚本职责：

- 检查 `src/routes/**/*.svelte` 是否超过 250 行
- 检查 `src/lib/features/**/*.svelte` 是否超过 300 行
- 检查 `src/lib/components/ui/**/*.svelte` 是否超过 250 行
- 输出超限文件清单，并在 CI 中返回非 0

这类脚本比 lint 更直白，适合做团队治理红线。

#### 15.2.10 Code Review 的前端结构检查清单

每个前端 PR 除功能正确外，还必须过以下结构审查：

- 页面 route 文件是否只是组装，而不是把所有逻辑塞在一处
- 是否新增了可复用组件而不是复制同类 UI
- 是否把 API、SSE、formatter、UI 状态混写在同一文件
- 是否出现 250+ 行的 route 文件或 300+ 行的 feature 组件
- 是否引入跨层依赖、循环依赖或 feature 间私有 import
- 是否存在“为了图省事”把一组 panels 全写进一个页面组件的情况

**当前仓库已有风险信号：**

- `web/src/routes/+page.svelte` 已达到数千行级别，这种文件规模必须视为架构告警，而不是“以后再拆”
- 一旦 Dashboard、Board、Settings、Onboarding 全部做进去，如果没有上述约束，页面会持续演化成不可维护的巨型文件

#### 15.2.11 推荐新增脚本与命令

```json
// package.json scripts 建议补充
{
  "scripts": {
    "lint": "eslint .",
    "lint:structure": "node scripts/check-file-budgets.mjs",
    "lint:deps": "depcruise src --config .dependency-cruiser.cjs",
    "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
    "ci": "pnpm run lint && pnpm run lint:structure && pnpm run lint:deps && pnpm run check && pnpm run build"
  }
}
```

这样才能把“组件复用规范”和“页面不膨胀”从口头要求变成自动检查。

### 15.3 Pre-commit 配置（lefthook）

```yaml
# lefthook.yml

pre-commit:
  parallel: true
  commands:
    # ── Go ──
    go-fmt:
      glob: "*.go"
      run: goimports -w {staged_files} && git add {staged_files}
      stage_fixed: true

    go-lint:
      glob: "*.go"
      run: golangci-lint run --new-from-rev=HEAD~1 ./...

    go-test:
      glob: "*.go"
      run: go test ./internal/domain/... ./internal/service/... ./internal/ticket ./internal/workflow ./internal/httpapi -short -count=1

    go-vet:
      glob: "*.go"
      run: go vet ./...

    go-mod-tidy:
      glob: "go.{mod,sum}"
      run: go mod tidy && git diff --exit-code go.mod go.sum

    # ── Frontend ──
    fe-format:
      root: "web/"
      glob: "*.{ts,svelte,css,json}"
      run: pnpm exec prettier --write {staged_files} && git add {staged_files}
      stage_fixed: true

    fe-lint:
      root: "web/"
      glob: "*.{ts,svelte}"
      run: pnpm exec eslint {staged_files}

    fe-typecheck:
      root: "web/"
      glob: "*.{ts,svelte}"
      run: pnpm exec svelte-check --tsconfig ./tsconfig.json

    # ── 通用 ──
    no-secrets:
      run: |
        if git diff --cached --diff-filter=ACM | grep -iE '(password|secret|api_key|token)\s*[:=]' | grep -v 'test\|mock\|example'; then
          echo "疑似硬编码密钥，请检查"; exit 1
        fi

    no-large-files:
      run: |
        for f in $(git diff --cached --name-only --diff-filter=ACM); do
          size=$(wc -c < "$f")
          if [ "$size" -gt 1048576 ]; then
            echo "$f 超过 1MB"; exit 1
          fi
        done

commit-msg:
  commands:
    conventional-commit:
      run: |
        MSG=$(cat {1})
        if ! echo "$MSG" | grep -qE '^(feat|fix|refactor|docs|test|chore|ci|perf)(\(.+\))?: .{1,72}$'; then
          echo "Commit message 不符合规范"
          echo "格式: type(scope): description"
          echo "示例: feat(ticket): add multi-repo support"
          exit 1
        fi
```

**执行流程（并行，总耗时 < 10 秒）：**

```
git commit
  ├── [parallel] go-fmt + fe-format      → 自动修复格式，重新 stage
  ├── [parallel] no-secrets              → 扫描硬编码密钥
  ├── [parallel] no-large-files          → 拒绝 >1MB 文件
  ├── [parallel] go-lint (含 depguard)   → 架构守卫 + 代码质量
  ├── [parallel] go-vet + go-test        → 静态检查 + 单元测试
  ├── [parallel] fe-lint + fe-typecheck  → ESLint + 类型检查
  ├── [parallel] go-mod-tidy             → go.mod 一致性
  └── [commit-msg] conventional-commit   → 提交信息格式
```

### 15.4 Conventional Commits

| Type | 含义 | Scope（与 DDD 领域对齐） |
|------|------|------------------------|
| `feat` | 新功能 | `ticket`、`workflow`、`agent`、`project`、`approval`、`hook` |
| `fix` | Bug 修复 | `orchestrator`、`adapter`、`api`、`web`、`setup` |
| `refactor` | 重构 | 同上 |
| `docs` | 文档 | `harness`、`readme` |
| `test` | 测试 | 同 feat |
| `chore` | 杂务 | `deps`、`ci`、`config` |
| `perf` | 性能 | 同 feat |

### 15.5 Pre-commit 与 CI 的分工

| 检查项 | Pre-commit（本地，快） | CI（远程，全） |
|--------|----------------------|--------------|
| gofmt + goimports | 自动修复 | 检查（不修复） |
| golangci-lint | 增量（本次变更） | 全量 |
| go test (unit) | `internal/domain` + service/use-case + `internal/httpapi`，`-short` | 全部包，含覆盖率 |
| go test (integration) | 跳过 | testcontainers-go |
| Prettier | 自动修复 | 检查 |
| ESLint + svelte-check | 增量 | 全量 |
| 密钥扫描 | 简单 grep | gitleaks 全量 |
| depguard 架构守卫 | 随 golangci-lint | 随 golangci-lint |
| SvelteKit build | 跳过 | `pnpm run build` |
| OpenAPI contract regenerate + diff | 按需运行 `make openapi-generate` | 强制执行 `make openapi-check`，要求 `api/openapi.json` 与 `web/src/lib/api/generated/openapi.d.ts` 已提交 |
| 覆盖率报告 | 跳过 | `go test -coverprofile` |

**前后端交接规则**：

1. 后端 HTTP handler、请求结构、响应结构一旦变化，必须先重新生成 `api/openapi.json`。
2. 前端只消费由 `api/openapi.json` 生成的类型化契约，不再手写一套独立接口定义。
3. PR 合入前，CI 必须通过 `make openapi-check`，用生成结果 diff 保证前后端接口已同步。

---

## 第十六章 核心路径数据流（伪代码）

本章用伪代码展示 OpenASE 最关键的 6 条路径的完整数据流、层间调用关系和通知方式。伪代码遵循 DDD 分层结构：`handler →  command/query → domain service → repository / provider`。

### 16.1 路径一：创建工单（用户 → 系统）

这是最基础的写操作路径，展示 Interface → Application → Domain → Infrastructure 的标准调用链。

```
┌──────────┐   POST /api/v1/projects/:id/tickets   ┌──────────────┐
│  Web UI  │ ────────────────────────────────────→ │ internal/httpapi │
│ (Svelte) │    createTicketRequest                │  (Interface)     │
└──────────┘                          └──────┬───────┘
                                             │
                  ┌──────────────────────────┘
                  ▼
         ┌────────────────────────┐
         │ ticket/service.go      │
         │ or service/* package   │  (Service / Use-Case Layer)
         └──────────┬─────────────┘
                  │
                  ▼
```

```go
// internal/httpapi/ticket_api.go — Interface / Entry 层
func (h *TicketHandler) Create(c echo.Context) error {
    ctx := c.Request().Context()
    ctx, span := h.tracer.StartSpan(ctx, "handler.ticket.create")  // Provider: Trace
    defer span.End()

    var req createTicketRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(400, err.Error())
    }

    // 调用 Service / Use-Case 层
    result, err := h.ticketService.Create(ctx, req.toCreateInput())
    if err != nil {
        return mapDomainError(err)  // domain error → HTTP status
    }

    return c.JSON(201, result)
}
```

```go
// internal/ticket/service.go — Service / Use-Case 层
func (s *Service) Create(ctx context.Context, input CreateInput) (Ticket, error) {
    ctx, span := s.trace.StartSpan(ctx, "ticket.create")
    defer span.End()

    params, err := parseCreateInput(input)
    if err != nil {
        return Ticket{}, err
    }

    created, err := s.repo.Create(ctx, params)
    if err != nil {
        return Ticket{}, fmt.Errorf("create ticket: %w", err)
    }

    return created, nil
}
```

```go
// internal/domain/ticketing/retry.go — Domain / Core Types 层（纯业务规则）
func NextRetryAt(attempt int, baseDelay time.Duration, now time.Time) time.Time {
    if attempt < 1 {
        attempt = 1
    }
    delay := baseDelay * time.Duration(1<<(attempt-1))
    return now.Add(delay)
}
```

**通知方式：** `EventProvider.Publish("ticket.events", ...)` → 编排引擎通过 `EventProvider.Subscribe("ticket.events")` 接收 → 下个 Tick 发现新工单。同进程时走 Go channel（零延迟），分开部署时走 PG LISTEN/NOTIFY。

---

### 16.2 路径二：编排引擎调度 Tick（系统内部核心循环）

这是编排引擎每隔 N 秒执行一次的调度循环，展示编排引擎如何与 service/use-case、domain 规则和 provider 边界交互。

```go
// internal/orchestrator/scheduler.go
func (s *Scheduler) runTick(ctx context.Context) {
    ctx, span := s.tracer.StartSpan(ctx, "orchestrator.tick")
    defer span.End()
    tickStart := time.Now()

    // ── Step 1: 对账 ──
    // 检查所有 running 工单的 Agent 是否还活着
    for ticketID, worker := range s.pool.ActiveWorkers() {
        if worker.IsStalled(s.stallTimeout) {
            s.logger.Warn("agent stalled", "ticket", ticketID, "last_event", worker.LastEventAt)
            worker.Kill()
            s.handleRetry(ctx, ticketID, RetryReasonStall)
        }
    }

    // ── Step 2: 同步最新已发布的 Workflow / Skill 版本缓存 ──
    published := s.workflowCatalog.SyncPublishedVersions(ctx)
    s.metrics.Counter("openase.orchestrator.harness_publish_total").Add(len(published.Harnesses))

    // ── Step 3: 获取候选工单 ──
    candidates, err := s.ticketRepo.ListByStatus(ctx, ticket.StatusTodo)
    if err != nil {
        s.logger.Error("list candidates", "err", err)
        return
    }

    // ── Step 4: 过滤 + 排序 ──
    var eligible []*ticket.Ticket
    for _, t := range candidates {
        // 检查依赖：被 block 的跳过
        if blocked, _ := s.ticketSvc.IsBlocked(ctx, t.ID); blocked {
            s.metrics.Counter("openase.orchestrator.tickets_skipped_total",
                provider.Tags{"reason": "blocked"}).Inc()
            continue
        }
        eligible = append(eligible, t)
    }
    // 按优先级 + 创建时间排序
    ticket.SortByPriorityAndAge(eligible)

    // ── Step 5: 并发检查 + 分发 ──
    for _, t := range eligible {
        if s.pool.ActiveCount() >= s.maxConcurrent {
            s.metrics.Counter("openase.orchestrator.tickets_skipped_total",
                provider.Tags{"reason": "max_concurrency"}).Inc()
            break
        }

        // 分发！
        if err := s.dispatch(ctx, t); err != nil {
            s.logger.Error("dispatch failed", "ticket", t.Identifier, "err", err)
        } else {
            s.metrics.Counter("openase.orchestrator.tickets_dispatched_total",
                provider.Tags{"workflow_type": t.WorkflowType()}).Inc()
        }
    }

    // 记录 Tick 耗时
    s.metrics.Histogram("openase.orchestrator.tick_duration_seconds").
        Observe(time.Since(tickStart).Seconds())
}
```

**通知方式：** Scheduler 是纯内部循环，不直接通知外部。状态变更通过修改数据库 + EventProvider 广播，SSE 端点监听 EventProvider 推给前端。

---

### 16.3 路径三：工单分发 → Hook → Agent 执行（核心主线）

这是 OpenASE 最核心的路径：从分发工单到 Agent 真正开始工作的完整流程。

```
Scheduler.dispatch()
    │
    ▼
┌────────────────────┐  on_claim Hook   ┌──────────────┐   Agent CLI   ┌───────────┐
│ ticket.Service     │ ───────────────→ │ HookExecutor │ ────────────→ │  Adapter  │
│ Claim / Assign     │ (工作副本准备等)             │  (Shell)     │   (启动)      │(ClaudeCode)│
└────────────────────┘                  └──────────────┘              └───────────┘
```

```go
// internal/orchestrator/scheduler.go
func (s *Scheduler) dispatch(ctx context.Context, t *ticket.Ticket) error {
    ctx, span := s.tracer.StartSpan(ctx, "orchestrator.dispatch",
        trace.WithAttributes("ticket.id", t.Identifier))
    defer span.End()

    // 1. 选择 Agent（能力匹配 + 负载均衡）
    agent, err := s.agentSvc.SelectForTicket(ctx, t)
    if err != nil {
        return fmt.Errorf("select agent: %w", err)
    }

    // 2. 通过 Service / Use-Case 层执行 Claim（含 on_claim Hook）
    err = s.ticketService.Claim(ctx, t.ID, agent.ID)
    if err != nil {
        return fmt.Errorf("claim ticket: %w", err)
    }

    // 3. 启动 Worker goroutine
    s.pool.Start(t.ID, func(workerCtx context.Context) {
        s.runWorker(workerCtx, t, agent)
    })

    return nil
}
```

```go
// internal/ticket/service.go — Claim / 状态流转包含 Hook 执行
func (s *Service) Claim(ctx context.Context, ticketID uuid.UUID, run *runtime.AgentRun, ag *agent.Agent) error {
    ctx, span := s.trace.StartSpan(ctx, "ticket.claim")
    defer span.End()

    t, err := s.repo.Get(ctx, ticketID)
    if err != nil {
        return err
    }

    // 1. Domain 层状态转换（纯规则校验）
    if err := t.TransitionTo(ticket.StatusInProgress); err != nil {
        return err  // 比如 ErrAlreadyClaimed, ErrBlockedByDependency
    }
    t.CurrentRunID = run.ID

    // 2. 准备 Hook 环境变量
    repos, _ := s.repoScopeRepo.ListByTicket(ctx, t.ID)
    hookEnv := hook.Env{
        TicketID:         t.ID,
        TicketIdentifier: t.Identifier,
        WorkflowType:     t.WorkflowType(),
        AgentName:        ag.Name,
        Repos:            repos,
    }

    // 3. 执行 on_claim Hook（阻塞型：失败则不领取）
    workflow, _ := s.workflowRepo.Get(ctx, t.WorkflowID)
    results, err := s.hookExec.RunAll(ctx, workflow.Hooks.OnClaim, hookEnv)
    if err != nil {
        // Hook 失败 → 记录日志 → 工单不领取，留在 todo
        s.eventBus.Publish(ctx, "ticket.events", ticket.HookFailedEvent{
            TicketID: t.ID,
            Hook:     "on_claim",
            Error:    err.Error(),
            Results:  results,
        })
        s.metrics.Counter("openase.hook.block_total",
            provider.Tags{"hook_name": "on_claim"}).Inc()
        return fmt.Errorf("on_claim hook failed: %w", err)
    }

    // 4. 持久化状态变更
    if err := s.repo.Save(ctx, t); err != nil {
        return err
    }

    // 5. 广播事件
    s.eventBus.Publish(ctx, "ticket.events", ticket.ClaimedEvent{
        TicketID: t.ID,
        AgentID:  agentID,
    })

    return nil
}
```

```go
// internal/orchestrator/runtime_runner.go — Worker 执行 Agent（概念示意）
func (s *Scheduler) runWorker(ctx context.Context, t *ticket.Ticket, agent *agent.Agent) {
    ctx, span := s.tracer.StartSpan(ctx, "worker.run",
        trace.WithAttributes("ticket.id", t.Identifier, "agent.name", agent.Name))
    defer span.End()

    // 1. 创建联合工作区（多 Repo clone + checkout feature branch）
    workspace, err := s.workspaceMgr.Setup(ctx, t)
    if err != nil {
        s.handleFailure(ctx, t, err)
        return
    }
    defer s.workspaceMgr.Cleanup(ctx, workspace)

    // 2. 渲染 Harness Prompt
    harness, _ := s.harnessLoader.Load(ctx, t.WorkflowID)
    prompt, _ := harness.Render(harness.TemplateData{
        Ticket:  t,
        Project: workspace.Project,
        Agent:   agent,
        Repos:   workspace.Repos,
        Attempt: t.AttemptCount,
    })

    // 3. 通过 Adapter 启动 Agent
    adapter := s.adapterRegistry.Get(agent.Provider.AdapterType)
    session, err := adapter.Start(ctx, agent.Config())
    if err != nil {
        s.handleFailure(ctx, t, err)
        return
    }

    // 4. 发送 Prompt + 流式接收事件
    adapter.SendPrompt(ctx, session, prompt)
    events, _ := adapter.StreamEvents(ctx, session)

    for event := range events {
        // 更新心跳
        agent.LastHeartbeatAt = time.Now()
        s.agentRepo.Save(ctx, agent)

        // 记录 Token 消耗
        s.metrics.Counter("openase.agent.tokens_used_total", provider.Tags{
            "provider": agent.Provider.Name,
            "direction": "output",
        }).Add(event.TokensOut)

        // 广播事件给前端 SSE
        s.eventBus.Publish(ctx, "agent.events", AgentProgressEvent{
            TicketID:  t.ID,
            AgentName: agent.Name,
            Event:     event,
        })

        // 检查是否完成（result 类型事件）
        if event.Type == "result" {
            break
        }
    }

    // 5. turn 正常完成后，runtime 重新读取 ticket 状态
    //    若 ticket 仍 active：继续下一轮 continuation turn
    //    若 ticket 已离开 active：停止当前 worker，但不自动改业务状态
}
```

**修订：OpenASE 应按 Symphony 的 AgentRunner 模式落地，而不是停留在上面这个“单 Turn worker”简化版。**

更接近目标实现的伪代码应是：

```go
// runtime/agent_runner.go
func (r *AgentRunner) Run(ctx context.Context, ticket *ticket.Ticket, agent *agent.Agent) error {
    workspace, err := r.workspaceMgr.Setup(ctx, ticket)
    if err != nil {
        return err
    }
    defer r.workspaceMgr.AfterRun(ctx, workspace, ticket)

    session, err := r.adapter.Start(ctx, agent.Config())
    if err != nil {
        return err
    }
    defer r.adapter.Stop(ctx, session)

    for turnNumber := 1; turnNumber <= cfg.Agent.MaxTurns; turnNumber++ {
        prompt := r.buildTurnPrompt(ticket, turnNumber)

        turn, err := r.adapter.StartTurn(ctx, session, StartTurnInput{
            Prompt:          prompt,
            Workspace:       workspace.Path,
            ApprovalPolicy:  cfg.Codex.ApprovalPolicy,
            SandboxPolicy:   cfg.Codex.TurnSandboxPolicy,
            DisplayTitle:    fmt.Sprintf("%s: %s", ticket.Identifier, ticket.Title),
        })
        if err != nil {
            return err
        }

        for event := range turn.Events {
            r.runtimeRepo.UpdateHeartbeat(ctx, agent.ID, event.Timestamp)
            r.runtimeRepo.IntegrateCodexEvent(ctx, agent.ID, session.ThreadID, turn.ID, event)
            r.eventBus.Publish(ctx, "agent.events", event)
        }

        refreshed, err := r.ticketRepo.RefreshState(ctx, ticket.ID)
        if err != nil {
            return err
        }
        ticket = refreshed

        if !ticket.IsActiveState() {
            return nil
        }
    }

    // hit max_turns: 把控制权交回 orchestrator，稍后由 continuation retry 再决定是否续跑
    return nil
}
```

这个修订版有 4 个必须保留的行为：

1. `Start()` 与 `thread/start` 只做一次，整个 worker 生命周期复用同一个 thread
2. `StartTurn()` 在同一 thread 上多次调用，每轮 turn 后重新读取 ticket 最新状态
3. continuation turn 只发续接 guidance，不重发完整原始 prompt
4. `max_turns` 命中后不自动判定 ticket 完成，而是把控制权交回 orchestrator

**通知方式：**
- `on_claim` Hook 失败 → `EventProvider.Publish("ticket.events", HookFailedEvent)` → SSE → 前端显示"领取失败"
- Agent 执行中 → `EventProvider.Publish(agent progress event)` → SSE → 前端实时流
- Agent 或人类显式推进状态 → `internal/ticket` 的状态变更路径按需执行 `on_complete` Hook → 成功则 `EventProvider.Publish(ticket status changed event)` → SSE → 前端状态更新

---

### 16.4 路径四：on_complete Hook 执行 + 显式状态推进

这条路径展示 Hook 如何作为质量门禁阻止或放行显式状态推进。普通 turn 结束不会直接进入这条路径。

```go
// internal/ticket/service.go — 简化伪代码，表示显式状态推进路径
func (s *Service) AdvanceAfterExplicitAction(ctx context.Context, ticketID uuid.UUID, requestedStatusID uuid.UUID) error {
    ctx, span := s.trace.StartSpan(ctx, "ticket.advance_after_explicit_action")
    defer span.End()

    t, _ := s.loadTicket(ctx, ticketID)
    workflow, _ := s.loadWorkflow(ctx, t.WorkflowID)

    // 1. 执行 on_complete Hook（阻塞型）
    hookEnv := hook.Env{
        TicketID:         t.ID,
        TicketIdentifier: t.Identifier,
        Workspace:        t.WorkspacePath(),
        Repos:            t.RepoScopes(),
    }
    results, err := s.runOnCompleteHooks(ctx, workflow, hookEnv)

    // 逐个记录 Hook 结果
    for _, r := range results {
        s.metrics.Histogram("openase.hook.duration_seconds",
            provider.Tags{"hook_name": r.Name}).Observe(r.Duration.Seconds())
        s.metrics.Counter("openase.hook.execution_total",
            provider.Tags{"hook_name": r.Name, "outcome": r.Outcome}).Inc()
    }

    if err != nil {
        // ── Hook 失败：工单留在 in_progress ──
        //    把失败信息作为 "反馈" 返回给 Agent，让它修复后重试
        s.publishHookFailed(ctx, t.ID, "on_complete", err, results)
        return fmt.Errorf("on_complete hook failed: %w", err)
    }

    // ── Hook 全部通过 ──

    // 2. 检查所有 Repo 的 PR 是否已提交
    scopes, _ := s.listRepoScopes(ctx, t.ID)
    allPRsOpen := true
    for _, scope := range scopes {
        if scope.PRStatus == "none" {
            allPRsOpen = false
            break
        }
    }
    if !allPRsOpen {
        return ErrPRNotSubmitted  // 有 Repo 还没提交 PR
    }

    // 3. 显式状态转换（例如进入 in_review）
    if err := t.TransitionTo(requestedStatusID); err != nil {
        return err
    }
    s.saveTicket(ctx, t)

    // 4. 通知 Reviewer / 前端，工单已显式进入审核态
    s.notificationEngine.NotifyTicketReadyForReview(ctx, t, len(results))

    // 5. 广播
    s.publishMovedToReview(ctx, t.ID, results)

    s.metrics.Histogram("openase.ticket.agent_time_seconds",
        provider.Tags{"workflow_type": t.WorkflowType()}).
        Observe(time.Since(t.StartedAt).Seconds())

    return nil
}
```

**通知方式：**
- Hook 失败 → `EventProvider` → SSE → 前端标红显示 + Agent 收到反馈重试
- Hook 通过 → `internal/notification` / `NotificationEngine` → Slack/Email/Webhook → Reviewer 收到通知
- 人类把工单移到 finish → 触发 `on_done` → 工单完成

---

### 16.5 路径五：PR 链接绑定（内部显式写入）

这条路径展示 RepoScope 中的 PR 链接如何被记录。它是一个普通的内部写路径，不依赖任何 GitHub/GitLab Webhook。

```
Agent / Human
    │
    ▼
POST /api/v1/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId
    │
    ▼
httpapi / catalog service
    │
    ▼
更新 TicketRepoScope.pull_request_url
    │
    ▼
写入 ActivityEvent(pr.linked)
    │
    ▼
SSE 推送到前端
```

```go
// internal/httpapi/catalog.go — Interface / Entry 层（概念示意）
func (h *RepoScopeHandler) UpdateRepoScope(c echo.Context) error {
    input := parseRepoScopePatch(c)
    scope, err := h.catalog.UpdateTicketRepoScope(c.Request().Context(), input)
    if err != nil {
        return writeError(c, err)
    }
    return c.JSON(200, scope)
}
```

```go
// internal/catalog/service.go — Service / Use-Case 层（概念示意）
func (s *Service) UpdateTicketRepoScope(ctx context.Context, input UpdateRepoScopeInput) error {
    scope := s.repoScopeRepo.Get(ctx, input.ScopeID)
    scope.PullRequestURL = input.PullRequestURL
    s.repoScopeRepo.Save(ctx, scope)

    s.activityRepo.Append(ctx, ActivityEvent{
        Type: "pr.linked",
        TicketID: scope.TicketID,
        Metadata: map[string]any{
            "repo_id": scope.RepoID,
            "pull_request_url": scope.PullRequestURL,
        },
    })
    return nil
}
```

**明确约束：**
- 绑定 PR 链接不会自动推进 Ticket 状态
- PR merged / closed / review / CI 结果都不会自动回写到 OpenASE
- 如需完成工单，仍由人类或 Agent 显式修改 Ticket 状态

---

### 16.6 路径六：SSE 实时推送（系统 → 前端）

这条路径展示事件如何从后端流到前端，支持多个浏览器同时在线。

```
Domain Event  ──→  EventProvider  ──→  SSE Hub (fan-out)  ──→  Browser A
                   (Go channel)       (每个连接独立 chan)   ──→  Browser B
                                                          ──→  Browser C
```

**关键：fan-out 广播。** 一个事件发布一次，所有 SSE 连接都收到。通过 SSE Hub 管理连接注册/注销和事件广播：

```go
// infra/sse/hub.go — SSE 连接管理器
type Hub struct {
    mu          sync.RWMutex
    connections map[string]map[chan Event]struct{}  // projectID → set of channels
}

func (h *Hub) Register(projectID string) chan Event {
    ch := make(chan Event, 64)  // 带缓冲，防慢消费者阻塞广播
    h.mu.Lock()
    if h.connections[projectID] == nil {
        h.connections[projectID] = make(map[chan Event]struct{})
    }
    h.connections[projectID][ch] = struct{}{}
    h.mu.Unlock()
    return ch
}

func (h *Hub) Unregister(projectID string, ch chan Event) {
    h.mu.Lock()
    delete(h.connections[projectID], ch)
    close(ch)
    h.mu.Unlock()
}

// 广播：所有订阅该 project 的连接都收到
func (h *Hub) Broadcast(event Event) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for ch := range h.connections[event.ProjectID] {
        select {
        case ch <- event:
        default:
            // 缓冲区满 → 跳过该慢消费者（不阻塞其他连接）
        }
    }
}
```

**SSE Hub 监听 EventProvider，广播到所有 SSE 连接：**

```go
// 启动时：Hub 订阅 EventProvider，fan-out 到所有 SSE 连接
func (h *Hub) Run(ctx context.Context, eventBus provider.EventProvider) {
    events, _ := eventBus.Subscribe(ctx, "ticket.events", "agent.events",
        "hook.events", "machine.events")
    for event := range events {
        h.Broadcast(event)
    }
}
```

**SSE 端点：每个浏览器连接注册到 Hub，收到属于自己 project 的事件：**

```go
// internal/httpapi/sse.go
func (h *SSEHandler) TicketStream(c echo.Context) error {
    projectID := c.Param("projectId")
    ctx := c.Request().Context()

    c.Response().Header().Set("Content-Type", "text/event-stream")
    c.Response().Header().Set("Cache-Control", "no-cache")
    c.Response().Header().Set("Connection", "keep-alive")

    // 注册到 Hub，获得独立的事件 channel
    ch := h.hub.Register(projectID)
    defer h.hub.Unregister(projectID, ch)

    pingTicker := time.NewTicker(15 * time.Second)
    defer pingTicker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil

        case event := <-ch:
            data, _ := json.Marshal(event)
            fmt.Fprintf(c.Response(), "event: %s\ndata: %s\n\n", event.Type, data)
            c.Response().Flush()

        case <-pingTicker.C:
            fmt.Fprintf(c.Response(), ": ping\n\n")
            c.Response().Flush()
        }
    }
}
```

**前端 Svelte store：**

```typescript
// web/src/lib/api/sse.ts
import { writable } from 'svelte/store'

export function createTicketStream(projectId: string) {
  const tickets = writable<Map<string, Ticket>>(new Map())
  let retryDelay = 1000

  function connect() {
    const source = new EventSource(`/api/projects/${projectId}/tickets/stream`)

    source.addEventListener('ticket.created', (e) => {
      const event = JSON.parse(e.data)
      tickets.update(map => map.set(event.ticketId, event.ticket))
    })

    source.addEventListener('ticket.status_changed', (e) => {
      const event = JSON.parse(e.data)
      tickets.update(map => {
        const t = map.get(event.ticketId)
        if (t) t.status = event.newStatus
        return map
      })
    })

    source.addEventListener('hook.failed', (e) => {
      const event = JSON.parse(e.data)
      // 更新工单的 Hook 状态，前端显示红色警告
      tickets.update(map => {
        const t = map.get(event.ticketId)
        if (t) t.lastHookError = event.error
        return map
      })
    })

    source.onerror = () => {
      source.close()
      // 指数退避重连：1s → 2s → 4s → ... → 最大 5 分钟
      setTimeout(connect, retryDelay)
      retryDelay = Math.min(retryDelay * 2, 5 * 60 * 1000)
    }

    source.onopen = () => { retryDelay = 1000 }  // 重连成功，重置退避
  }

  connect()
  return { tickets }
}
```

```svelte
<!-- web/src/routes/(app)/tickets/+page.svelte -->
<script lang="ts">
  import { createTicketStream } from '$lib/api/sse'
  import TicketCard from '$lib/components/ticket/TicketCard.svelte'

  const { tickets } = createTicketStream(projectId)
</script>

<!-- 工单看板：tickets store 变化时自动更新 UI -->
{#each [...$tickets.values()] as ticket (ticket.id)}
  <TicketCard {ticket} />
{/each}
```

**通知方式总结表：**

| 事件源 | 传播路径 | 最终接收者 |
|--------|---------|----------|
| 用户创建工单 | Handler → Cmd → EventProvider → SSE Hub → 所有浏览器 | 前端看板实时新增卡片 |
| 用户拖拽工单 | Handler → Cmd → EventProvider → SSE Hub → 所有浏览器 | 其他人的看板实时更新 |
| 编排引擎分发 | Scheduler → ClaimCmd → EventProvider → SSE Hub | 前端看到 Agent 领取 |
| Agent 执行进度 | Worker → EventProvider → SSE Hub | 前端实时输出流 |
| Hook 失败 | HookExec → Cmd → EventProvider → SSE Hub | 前端标红 + Agent 收到反馈 |
| Hook 通过 + 需审批 | Cmd → EventProvider → SSE Hub + NotificationEngine → Telegram/企业微信 | 前端 + 外部通知 |
| GitHub PR merged | Webhook → Cmd → EventProvider → SSE Hub | 前端状态实时更新 |
| Agent 出错重试 | Worker → EventProvider → SSE Hub + NotificationEngine | 前端显示重试状态 + 告警 |
| 配置变更 | Setup UI → 写 .env → systemctl restart | 服务重启，前端 SSE 自动重连 |

---

## 第十七章 系统边界与信息一致性

本章回答一个根本性问题：**OpenASE 负责什么、不负责什么、信息存在哪里、哪些地方可能不一致、出了不一致怎么修复。**

### 17.1 系统边界总览

```
                        OpenASE 的职责边界
┌──────────────────────────────────────────────────────────┐
│                                                          │
│   工单生命周期管理    Workflow / Harness / Hook 编排        │
│   Agent 调度与监控    审批治理    成本追踪    活动审计        │
│                                                          │
└────────┬──────────────┬──────────────┬───────────────────┘
         │              │              │
    ┌────▼────┐   ┌─────▼─────┐  ┌────▼────┐
    │ Git 平台 │   │ Agent CLI │  │  OIDC   │
    │ GitHub   │   │ Claude    │  │ Provider│
    │ GitLab   │   │ Codex     │  │         │
    │          │   │ Gemini    │  │         │
    └─────────┘   └───────────┘  └─────────┘
     外部系统①        外部系统②      外部系统③

    ┌─────────┐   ┌───────────┐  ┌─────────┐
    │PostgreSQL│   │ 通知渠道   │  │  OTel   │
    │ (自管)   │   │ Slack     │  │ Collector│
    │          │   │ Email     │  │ (可选)   │
    └─────────┘   └───────────┘  └─────────┘
     基础设施①       外部系统④      基础设施②
```

**OpenASE 负责的（系统内部）：**

- 工单的完整生命周期（创建 → 分配 → 执行 → 审批 → 完成）
- Workflow、Harness、Hook 的定义和执行
- Agent 的调度、进程管理、心跳监控
- 工单与 Git 分支/PR 的关联关系
- 成本追踪和预算控制
- 活动审计日志
- 前端 UI 和 SSE 推送

**OpenASE 不负责的（外部系统管理）：**

- Git 仓库本身（代码内容、分支权限、CI/CD 流程）
- Agent CLI 的安装、升级、API Key 管理（用户自行管理，OpenASE 只调用）
- PostgreSQL 的运维（备份、升级、高可用）
- 认证身份源（OIDC Provider 的管理）
- 通知渠道的配置（Slack workspace、Email 服务器）
- 可观测性后端（Jaeger、Prometheus 的运维）

### 17.2 外部系统对接清单

| 外部系统 | 对接方式 | OpenASE 侧接口 | 数据流向 | 对接时机 |
|---------|---------|---------------|---------|---------|
| **GitHub / GitLab** | Git + 可选 REST API | ProjectRepo / runtime git&gh integration | 双向（代码与 PR 链接），不做状态同步 | Phase 2 |
| **Claude Code** | CLI subprocess (NDJSON stream) | `internal/infra/adapter/claudecode/` | 双向 | Phase 1 |
| **OpenAI Codex** | JSON-RPC over stdio | `internal/infra/adapter/codex/` | 双向 | Phase 1 |
| **Gemini CLI** | CLI subprocess (stdio stream) | `internal/orchestrator/agent_adapter_gemini.go` + `internal/chat/runtime_gemini.go` | 双向 | Phase 1 |
| **PostgreSQL** | SQL (ent ORM) + LISTEN/NOTIFY | `internal/repo/` + `internal/infra/event/pgnotify.go` | 双向 | Phase 1 |
| **OIDC Provider** | OIDC Discovery + JWT 验证 | 当前仓库尚未落地统一 OIDC adapter，安全边界说明见 `internal/httpapi/security_settings_api.go` | 读 | Phase 4 |
| **Slack / Telegram / Webhook / WeCom** | Webhook / Bot API | `internal/notification/` | 写 | Phase 2 |
| **OTel Collector** | OTLP gRPC/HTTP | `internal/infra/otel/` | 写 | Phase 2 |

### 17.3 信息归属与单一信源（Source of Truth）

这是最重要的一节。每条关键信息只有一个权威来源，其他地方都是缓存或投影。不一致时，以信源为准。

| 信息 | 单一信源 | 存储位置 | 谁写 | 谁读 |
|------|---------|---------|------|------|
| **工单状态** | OpenASE DB | PostgreSQL `tickets.status` | serve (API) / orchestrate (状态机) | serve (API/SSE)、orchestrate (调度) |
| **工单描述/元数据** | OpenASE DB | PostgreSQL `tickets.*` | serve (API) | 所有 |
| **Workflow 定义** | OpenASE DB | PostgreSQL `workflows.*` | serve (API) | orchestrate (调度) |
| **Harness 内容** | OpenASE DB | PostgreSQL `workflow_versions.*`（或等价版本表） | serve (API) / refine-harness Agent | orchestrate（runtime materializer；在 runtime 创建时直接读取已发布版本） |
| **Skill Bundle 内容** | OpenASE DB | PostgreSQL `skills.*` + `skill_versions.*` + `skill_version_files.*`（或等价 bundle 版本表） | serve (API) / Agent Platform API | orchestrate（runtime materializer；在 runtime 创建时直接读取已发布版本） |
| **Hook 脚本** | Git 仓库 | 项目 repo `scripts/ci/*`（Harness 中引用） | 人类 (git push) | orchestrate (HookExecutor) |
| **Agent 注册信息** | OpenASE DB | PostgreSQL `agents.*` + `agent_providers.*` | serve (API) | orchestrate (调度) |
| **Agent 运行生命周期** | OpenASE DB | PostgreSQL `agent_runs.*` + `tickets.current_run_id` | orchestrate (runtime runner / scheduler) | serve (API/SSE)、orchestrate |
| **Agent 当前动作阶段** | OpenASE DB | PostgreSQL `agent_runs.current_step_*` | orchestrate (step projector) | serve (Agent console / API) |
| **Agent CLI 细粒度输出** | OpenASE DB | PostgreSQL `agent_trace_events.*` | orchestrate (adapter event normalizer) | serve (Agent output API / trace stream) |
| **项目/工单业务活动流** | OpenASE DB | PostgreSQL `activity_events.*` | serve / orchestrate / hook executor | serve (dashboard / ticket activity / project activity) |
| **Agent 进程句柄** | 编排引擎内存 | runtime registry（活动 session / worker map） | orchestrate (runtime runner) | orchestrate (health checker / shutdown) |
| **Agent 会话 ID** | Agent CLI | Agent CLI 内部管理 | Agent CLI | orchestrate (Adapter 读取) |
| **Git 分支存在性** | Git 平台 | GitHub / GitLab 仓库 | Agent (git push) | orchestrate / 人类 |
| **PR 链接** | OpenASE DB | PostgreSQL `ticket_repo_scopes.pull_request_url` | serve (API) / orchestrate (平台操作) | serve (API/SSE)、人类 |
| **Token 消耗** | Agent CLI 返回值 | Agent event stream `cost_usd` | Agent CLI → orchestrate | orchestrate → DB |
| **累计成本** | OpenASE DB | PostgreSQL `tickets.cost_amount` | orchestrate (计算) | serve (仪表盘) |
| **Hook 执行结果** | 编排引擎运行时 | 脚本 exit code + stderr | HookExecutor | orchestrate → DB (`ActivityEvent` / Hook 记录) |
| **人工审核结果** | OpenASE DB | PostgreSQL `tickets.status_id` | serve (Reviewer 操作) | serve (API/SSE)、orchestrate |
| **用户身份** | OIDC Provider | 外部 IdP | OIDC Provider | serve (JWT 验证) |
| **用户身份缓存** | OpenASE DB | PostgreSQL `users.*` | serve (首次登录同步) | serve (API) |
| **全局配置** | 文件系统 | `~/.openase/config.yaml` | Setup Wizard / 手动编辑 | serve + orchestrate (启动时读取) |
| **敏感配置** | 文件系统 | `~/.openase/.env` | Setup Wizard / 手动编辑 | serve + orchestrate (启动时读取) |

### 17.4 一致性分析：哪里会不一致、怎么修复

**强一致区域（单写者，不会不一致）：**

| 信息 | 为什么不会不一致 |
|------|----------------|
| 工单状态 | 所有状态变更都经过 `internal/ticket` / `internal/ticketstatus` 的规则与持久化路径，写入同一个 PostgreSQL 事务 |
| Workflow 定义 | 只有 serve 进程的 API 可以写 |
| 累计成本 | 只有 orchestrate 进程的 Worker 在 Agent 事件中累加 |

**最终一致区域（有延迟，但会自动收敛）：**

| 信息 | 不一致窗口 | 自动修复机制 | 最坏情况 |
|------|-----------|------------|---------|
| Harness / Skill 已发布版本 | 新版本发布到下一次 runtime 创建之间（通常 < 5s） | 下次 dispatch / runtime 创建时直接读取 DB 当前已发布版本并 materialize | 单次读取失败：下个 Tick 重试，无需额外 sync |
| Agent 心跳 | 心跳间隔期间（可配置，默认 30s） | Worker 持续上报 | Agent 崩溃未报：HealthChecker 5 分钟 Stall 超时兜底 |
| 用户身份缓存 | OIDC 侧改名/禁用到 OpenASE 同步之间 | 每次 JWT 验证时刷新用户信息 | 已禁用用户的旧 JWT 未过期：JWT 过期时间兜底（建议 ≤ 1h） |
| 前端 UI 状态 | 后端状态变更到 SSE 推送之间（< 1s） | SSE 事件流持续推送 | SSE 断开重连：重连后请求全量最新状态 |

**潜在不一致风险（需要主动防御）：**

**风险 1：Git 分支已存在但 DB 不知道**

- 场景：Agent 创建了 Git 分支但 OpenASE 进程在写入 TicketRepoScope 之前崩溃
- 信源冲突：Git 平台说分支存在，DB 说 branch_name 为空
- 防御：Worker 启动时先检查远端分支是否已存在（`git ls-remote`），如果存在则恢复 TicketRepoScope 记录
- 补偿：`openase reconcile` CLI 命令，扫描所有 running 工单，对比 Git 远端和 DB 状态

**风险 2：Agent 已完成但 OpenASE 未收到完成事件**

- 场景：Agent CLI 子进程被 OS kill（OOM 等），最后的 result 事件丢失
- 信源冲突：Agent 实际已完成工作（PR 已提交），但 OpenASE 认为工单还在 in_progress
- 防御：Stall 检测（5 分钟无事件 → Kill + 重试）。重试时 Agent 会发现 PR 已存在，直接报告完成
- 补偿：on_claim Hook 中检查 PR 是否已存在，如果是则跳过编码直接进入 review 流程

**风险 3：PR 链接缺失或过时**

- 场景：Agent 创建了 PR，但没有把链接写回 RepoScope；或者 PR 链接后来被人工改动
- 设计决策：`pull_request_url` 只是引用信息，不是状态机输入，因此缺失或过时不会阻塞工单流转
- 补偿：人类可在 Ticket Detail 中手动补全或修正 PR 链接；后续如需自动化同步，必须作为独立能力重新立项

**风险 4：双写竞争——serve 和 orchestrate 同时修改工单**

- 场景：用户在 UI 取消工单的同时，编排引擎正在把工单标记为完成
- 信源冲突：两个进程对同一个 ticket.status 做不同的写
- 防御：PostgreSQL 乐观锁。工单表增加 `version` 字段，每次更新 `WHERE version = ?`，冲突时重试
- 实现：ent schema 中 `field.Int("version").Default(0)` + 更新时 `UpdateOne(t).Where(ticket.Version(currentVersion))`

**风险 5：Harness 版本不匹配**

- 场景：工单领取时用的 Harness v3，执行到一半时 Harness 被更新为 v4
- 设计决策：**已在运行的工单不受影响**——Worker 在领取时缓存 Harness 快照，工单记录 `harness_version` 字段。新工单用 v4，旧工单继续用 v3
- 不需要修复：这是预期行为，不是不一致

### 17.5 Reconciliation 策略

针对上述不一致风险，OpenASE 采用两层修复机制：

**自动修复（后台 Reconciler）：**

```go
// internal/orchestrator/health_checker.go / retry_service.go — 周期检查（概念示意）
func (r *Reconciler) Run(ctx context.Context) {
    // 1. 孤儿 runtime 清理
    //    检查活动 runtime registry 中的任务是否仍然存在（未被删除/取消）
    r.cleanOrphanWorkers(ctx)

    // 2. 卡住工单检测
    //    in_progress 超过 1 小时且无心跳 → 暂停重试（retry_paused=true），通知人类
    r.detectStuckTickets(ctx)

    // 3. Git 分支对账
    //    running 工单的 TicketRepoScope → 检查远端分支是否存在
    r.reconcileGitBranches(ctx)
}
```

**手动修复（CLI 命令）：**

```bash
openase reconcile              # 运行全部对账
openase reconcile --branches   # 只对账 Git 分支
openase reconcile --stuck      # 只处理卡住的工单
openase reconcile --dry-run    # 只报告不一致，不修复
```

### 17.6 信息流总览图

```
                 信源：OpenASE DB + Git 仓库
                   ┌─────────────┐
                   │ Harness / Skill 版本 │──publish───→ 下一次 runtime 直接读取并 materialize
                   │ Skill 版本   │
                   └─────────────┘
                           │
                           │
                    ┌─────────────┐
                    │ Hook 脚本    │
                    │ 代码分支/PR   │──Webhook───→ serve → DB (缓存)
                    └─────────────┘               ↑
                                          Reconciler 补偿轮询

                    信源：Agent CLI
                   ┌─────────────┐
                   │ 会话状态      │
                   │ Token 消耗   │──NDJSON/RPC──→ orchestrate Worker
                   │ 执行结果      │                    │
                   └─────────────┘                    写入 DB
                                                       │
                    信源：OpenASE DB                     ▼
                   ┌─────────────┐              ┌──────────────┐
                   │ 工单状态      │◄─────────────│  PostgreSQL   │
                   │ 审批决策      │              │  (唯一持久化)  │
                   │ 成本数据      │─────────────→│              │
                   │ 活动日志      │              └──────┬───────┘
                   └─────────────┘                      │
                                                   LISTEN/NOTIFY
                    信源：OIDC Provider                  │
                   ┌─────────────┐                      ▼
                   │ 用户身份      │──JWT 验证──→ serve ──SSE──→ 前端 (投影)
                   └─────────────┘

图例：
  ──→  数据流方向
  信源   该信息的权威来源，不一致时以此为准
  (缓存)  OpenASE 内部的非权威副本，可从信源重建
  (投影)  只读视图，从信源派生
```

### 17.7 边界设计原则

1. **信源唯一**：每条信息只有一个写入者。工单状态只有状态机写，RepoScope 中的 PR 链接只有显式平台写路径写，Harness 只有 Git 仓库存储
2. **引用不驱动状态机**：外部链接与 PR 链接只提供上下文，不参与 Ticket 自动状态推进
3. **乐观锁防双写**：两个进程可能同时写同一条工单时，用 `version` 字段做乐观并发控制
4. **Reconciler 兜底**：后台 Reconciler 只负责本地运行态与 Git 分支等内部一致性，不依赖 GitHub/GitLab Webhook
5. **幂等操作**：所有状态变更操作设计为幂等——重复执行相同的状态转换不会产生副作用
6. **崩溃恢复**：进程重启后，编排引擎扫描所有 in_progress 工单，恢复 Worker 或标记需要重试

---

## 第十八章 REST API 设计

### 18.1 API 总览

所有 API 以 `/api/v1` 为前缀，返回 JSON，使用标准 HTTP 状态码。认证通过 `Authorization: Bearer <token>` Header。分页使用 cursor-based pagination（`?cursor=xxx&limit=20`）。

### 18.2 资源端点

**Organization**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/orgs` | 列出所有组织 |
| POST | `/api/v1/orgs` | 创建组织 |
| GET | `/api/v1/orgs/:orgId` | 获取组织详情 |
| PATCH | `/api/v1/orgs/:orgId` | 更新组织 |
| DELETE | `/api/v1/orgs/:orgId` | 归档组织并自动归档其下所有项目（软删除） |

**Project**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/projects` | 列出项目 |
| POST | `/api/v1/orgs/:orgId/projects` | 创建项目 |
| GET | `/api/v1/projects/:projectId` | 获取项目详情 |
| PATCH | `/api/v1/projects/:projectId` | 更新项目 |
| DELETE | `/api/v1/projects/:projectId` | 归档项目（软删除） |

**ProjectRepo**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:projectId/repos` | 列出项目仓库 |
| POST | `/api/v1/projects/:projectId/repos` | 添加仓库 |
| PATCH | `/api/v1/projects/:projectId/repos/:repoId` | 更新仓库配置 |
| DELETE | `/api/v1/projects/:projectId/repos/:repoId` | 移除仓库 |

**Ticket（核心）**

| 方法 | 路径 | 说明 | 参数 |
|------|------|------|------|
| GET | `/api/v1/projects/:projectId/tickets` | 列出工单 | `?status_name=Todo,In+Progress&priority=high&workflow_type=coding&cursor=xxx&limit=20`（status_name 为自定义状态名，非硬编码枚举） |
| POST | `/api/v1/projects/:projectId/tickets` | 创建工单 | body: `{title, description, priority, type, workflow_id?, repo_scopes?}` |
| GET | `/api/v1/tickets/:ticketId` | 工单详情（基础详情） | |
| GET | `/api/v1/projects/:projectId/tickets/:ticketId/detail` | 工单详情聚合视图（含 description entry、timeline、RepoScopes、依赖） | |
| PATCH | `/api/v1/tickets/:ticketId` | 更新工单（标题、描述、优先级） | |
| POST | `/api/v1/tickets/:ticketId/transition` | 状态转换 | body: `{to_status_id: "uuid" 或 to_status_name: "待测试", comment?}`（传自定义状态名或 ID） |
| POST | `/api/v1/tickets/:ticketId/cancel` | 取消工单 | body: `{reason?}` |
| POST | `/api/v1/tickets/:ticketId/retry` | 手动触发重试 | |
| GET | `/api/v1/tickets/:ticketId/activity` | 工单活动日志 | `?cursor=xxx&limit=50` |

**Ticket 评论**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/tickets/:ticketId/comments` | 列出当前工单评论 |
| POST | `/api/v1/tickets/:ticketId/comments` | 创建评论 |
| PATCH | `/api/v1/tickets/:ticketId/comments/:commentId` | 编辑评论 |
| DELETE | `/api/v1/tickets/:ticketId/comments/:commentId` | 删除评论 |
| GET | `/api/v1/tickets/:ticketId/comments/:commentId/revisions` | 读取评论历史版本 |

**Ticket 依赖**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/tickets/:ticketId/dependencies` | 添加依赖 `{target_ticket_id, type: "blocks"\|"sub_issue"}` |
| DELETE | `/api/v1/tickets/:ticketId/dependencies/:depId` | 移除依赖 |

**Ticket 外部链接**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/tickets/:ticketId/links` | 列出外部链接（Issues + PRs） |
| POST | `/api/v1/tickets/:ticketId/links` | 添加外部链接 `{link_type, url, relation}` |
| DELETE | `/api/v1/tickets/:ticketId/links/:linkId` | 移除外部链接 |

**Workflow**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:projectId/workflows` | 列出 Workflow |
| POST | `/api/v1/projects/:projectId/workflows` | 创建 Workflow |
| GET | `/api/v1/workflows/:workflowId` | Workflow 详情（含 Harness 内容、Hook 配置） |
| PATCH | `/api/v1/workflows/:workflowId` | 更新 Workflow |
| GET | `/api/v1/workflows/:workflowId/harness` | 获取当前发布中的 Harness 原始内容（Markdown） |
| PUT | `/api/v1/workflows/:workflowId/harness` | 更新 Harness，生成新版本 |
| GET | `/api/v1/workflows/:workflowId/harness/history` | Harness 版本历史（平台版本记录） |

**AgentProvider & Agent**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/providers` | 列出 Agent Provider（含 `availability_state / available / availability_checked_at / availability_reason`） |
| POST | `/api/v1/orgs/:orgId/providers` | 注册 Provider |
| PATCH | `/api/v1/providers/:providerId` | 更新 Provider |
| GET | `/api/v1/providers/:providerId` | Provider 详情（含运行时可用性派生字段） |
| GET | `/api/v1/projects/:projectId/agents` | 列出 Agent |
| POST | `/api/v1/projects/:projectId/agents` | 注册 Agent |
| GET | `/api/v1/agents/:agentId` | Agent 详情（状态、当前工单、心跳、Token 消耗） |
| GET | `/api/v1/projects/:projectId/agents/:agentId/output` | Agent 细粒度运行输出；兼容命名，底层读取 `AgentTraceEvent` |
| GET | `/api/v1/projects/:projectId/agents/:agentId/steps` | Agent 人类可读动作流；读取 `AgentStepEvent` |
| DELETE | `/api/v1/agents/:agentId` | 注销 Agent |

**ScheduledJob**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:projectId/scheduled-jobs` | 列出定时任务 |
| POST | `/api/v1/projects/:projectId/scheduled-jobs` | 创建定时任务 |
| PATCH | `/api/v1/scheduled-jobs/:jobId` | 更新定时任务 |
| DELETE | `/api/v1/scheduled-jobs/:jobId` | 删除定时任务 |
| POST | `/api/v1/scheduled-jobs/:jobId/trigger` | 手动触发一次 |

**SSE 端点**

| 方法 | 路径 | 事件类型 |
|------|------|---------|
| GET | `/api/v1/projects/:projectId/tickets/stream` | `ticket.created`, `ticket.status_changed`, `ticket.updated` |
| GET | `/api/v1/projects/:projectId/agents/stream` | `agent.claimed`, `agent.launching`, `agent.ready`, `agent.heartbeat`, `agent.failed`, `agent.terminated` |
| GET | `/api/v1/projects/:projectId/activity/stream` | 业务活动流；只推 coarse-grained `ActivityEvent`，不推 token 级 output |
| GET | `/api/v1/projects/:projectId/agents/:agentId/output/stream` | Agent 细粒度输出流；只推 `AgentTraceEvent` |
| GET | `/api/v1/projects/:projectId/agents/:agentId/steps/stream` | Agent 动作阶段流；只推 `AgentStepEvent` |
| GET | `/api/v1/projects/:projectId/hooks/stream` | `hook.started`, `hook.passed`, `hook.failed` |

**Coding Agent Launch Verification**

- `claimed + runtime_phase=none`：显示 “Claimed, waiting for launcher”
- `claimed/running + runtime_phase=launching`：显示 “Launching Codex session”
- `running + runtime_phase=ready + session_id`：显示 “Ready”
- `failed` 或 `runtime_phase=failed`：显示失败态和 `last_error`
- activity 面板为空时显示 “No business activity yet”，不能把 0 行 output 当成未启动或失败
- Agent output 面板为空时显示 “No trace events yet”，表示当前还没有细粒度运行输出，不代表 runtime 启动失败
- Black-box 验收至少覆盖：创建 idle Agent + pickup Ticket，观测 `claimed`，再观测 `running + ready + session_id + heartbeat`，并收到 `agent.ready`；随后在 output/step 流中分别看到 `AgentTraceEvent` 与 `AgentStepEvent`

**Webhook 接收**

当前版本不定义 GitHub / GitLab 入站 Webhook API。OpenASE 不通过 Webhook 同步 Issue、PR 状态或 CI 状态。

**系统管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/system/health` | 健康检查（200 = 健康，503 = 不健康） |
| GET | `/api/v1/system/status` | 系统状态（编排引擎运行状态、Worker 数、队列深度） |
| POST | `/api/v1/system/reconcile` | 触发手动对账（等价于 `openase reconcile`） |
| GET | `/api/v1/system/metrics` | Prometheus 格式的指标导出 |

### 18.3 统一响应格式

```json
// 成功（单个资源）
{
  "data": { ... },
  "meta": { "request_id": "req_xxx" }
}

// 成功（列表，cursor-based 分页）
{
  "data": [ ... ],
  "meta": {
    "request_id": "req_xxx",
    "cursor": "eyJpZCI6...",
    "has_more": true,
    "total": 142
  }
}

// 错误
{
  "error": {
    "code": "TICKET_BLOCKED",
    "message": "Ticket ASE-42 is blocked by ASE-30",
    "details": { "blockers": ["ASE-30"] }
  },
  "meta": { "request_id": "req_xxx" }
}
```

### 18.4 错误码体系

| HTTP Status | Error Code | 说明 |
|-------------|-----------|------|
| 400 | `VALIDATION_ERROR` | 请求参数校验失败 |
| 400 | `INVALID_TRANSITION` | 不合法的状态转换（如 backlog → done） |
| 401 | `UNAUTHORIZED` | Token 缺失或无效 |
| 403 | `FORBIDDEN` | 权限不足 |
| 404 | `NOT_FOUND` | 资源不存在 |
| 409 | `CONFLICT` | 乐观锁冲突（version 不匹配） |
| 409 | `TICKET_BLOCKED` | 工单被依赖阻塞 |
| 409 | `AGENT_BUSY` | Agent 正在执行其他工单 |
| 409 | `APPROVAL_PENDING` | 有待处理的审批 |
| 422 | `HOOK_FAILED` | Hook 执行失败，阻止了操作 |
| 429 | `RATE_LIMITED` | 请求过于频繁 |
| 500 | `INTERNAL_ERROR` | 服务内部错误 |
| 503 | `SERVICE_UNAVAILABLE` | 服务暂时不可用（启动中 / 数据库断连） |

---

## 第十九章 错误处理与容错策略

### 19.1 分层错误处理

```
Domain / Core Types Layer → 返回领域错误（ErrTicketBlocked, ErrInvalidTransition）
                            纯业务语义，不含 HTTP 概念
          ↓
Service / Use-Case Layer → 包装上下文（fmt.Errorf("claim ticket %s: %w", id, err)）
                            不吞错误，不转换
          ↓
Interface / Entry Layer → 映射到 HTTP 状态码 + 统一错误响应
                            domain error → 4xx，infra error → 5xx
```

```go
// internal/httpapi/server.go / ticket_api.go
func ErrorHandler(err error, c echo.Context) {
    var domainErr *domain.Error
    if errors.As(err, &domainErr) {
        // 领域错误 → 4xx
        c.JSON(domainErr.HTTPStatus(), ErrorResponse{
            Code:    domainErr.Code(),
            Message: domainErr.Message(),
        })
        return
    }
    // 未知错误 → 500（日志记录完整 stack，响应只返回 request_id）
    logger.Error("unhandled error", "err", err, "request_id", c.Get("request_id"))
    c.JSON(500, ErrorResponse{Code: "INTERNAL_ERROR", Message: "internal error"})
}
```

### 19.2 数据库容错

| 场景 | 行为 |
|------|------|
| 连接丢失 | ent 的连接池自动重试；API 请求返回 503；编排引擎 Tick 跳过，下个 Tick 重试 |
| 连接池耗尽 | 新请求排队等待（默认超时 5s），超时返回 503；报警指标 `db_connections_active` 触发告警 |
| 迁移失败 | 启动中止，日志输出具体迁移错误，不进入服务状态 |
| 慢查询 | `db_query_duration_seconds` Histogram 超过阈值触发告警；编排引擎 Tick 超时则跳过该 Tick |
| 死锁 | ent 事务默认设置 `lock_timeout = 5s`，超时后回滚重试（最多 3 次） |

### 19.3 Agent 进程容错

| 场景 | 行为 |
|------|------|
| Agent CLI 进程 OOM kill | Worker goroutine 通过 `cmd.Wait()` 收到退出信号 → 标记异常退出 → 指数退避重试 |
| Agent CLI 僵尸进程 | Worker 在 Kill 后等待 10 秒，如果进程仍存在则发 SIGKILL；记录 ActivityEvent |
| Agent CLI 启动失败（命令不存在） | Worker 返回错误 → 触发 on_error Hook → 指数退避重试 → ActivityEvent 记录错误 |
| stdout 流中断（管道破裂） | Worker 检测到 EOF → 等同于异常退出 → 重试 |
| NDJSON 解析错误 | 跳过该行，记录 warning 日志，不中断 session |
| 文件句柄泄漏 | Worker 在 defer 中关闭 stdin/stdout/stderr pipe；每次 Tick 检查 `/proc/self/fd` 数量作为告警指标 |

### 19.4 磁盘空间容错

| 场景 | 行为 |
|------|------|
| 工作区过大 | on_done / on_fail / on_cancel Hook 负责清理工作区；Reconciler 定时扫描清理孤儿工作区 |
| 磁盘空间不足 | 编排引擎 Tick 时检查 `~/.openase/workspace/` 所在分区剩余空间；低于阈值（默认 1GB）时暂停分发新工单 |
| 日志文件膨胀 | `~/.openase/logs/` 保留最近 7 天，logrotate 或内置清理 |

### 19.5 优雅关机

收到 `SIGTERM` 或 `SIGINT` 后的关机序列：

```
1. 停止接受新的 HTTP 请求（Echo 优雅关机，等待 in-flight 请求完成，超时 30s）
2. 停止编排引擎 Ticker（不再分发新工单）
3. 向所有活跃 Worker 发送取消信号（context.Cancel）
4. 等待所有 Worker 完成当前操作或超时（默认 60s）
   - Worker 收到 cancel → 向 Agent CLI 发 SIGTERM → 等待退出 → 更新工单状态为 "interrupted"
   - Worker 超时未退出 → 向 Agent CLI 发 SIGKILL → 强制标记工单状态
5. 关闭 SSE 连接
6. 关闭数据库连接池
7. 写入最终日志，退出
```

工单在优雅关机后保持原状态（`current_run_id` 被清空）。下次启动时，编排引擎通过崩溃恢复流程扫描到这些工单并重新分发。

### 19.6 崩溃恢复

进程启动时执行恢复检查：

```go
func (s *Scheduler) recoverOnStartup(ctx context.Context) {
    // 1. 找到所有 in_progress 但没有活跃 Worker 的工单
    orphans, _ := s.ticketRepo.ListByStatus(ctx, ticket.StatusInProgress)

    for _, t := range orphans {
        // 2. 检查 Runtime readiness / heartbeat，而不是宿主机 PID
        if !s.isRuntimeReady(t.AgentRuntimePhase, t.LastHeartbeatAt) {
            // 3. 重置为 todo，下个 Tick 重新分发
            t.TransitionTo(ticket.StatusTodo)
            t.AttemptCount++  // 计为一次失败尝试
            s.ticketRepo.Save(ctx, t)
            s.logger.Warn("recovered orphan ticket", "ticket", t.Identifier)
        }
    }
}
```

---

## 第二十章 数据库索引与性能

### 20.1 核心索引

```sql
-- 编排引擎每 5 秒轮询的核心查询
-- SELECT * FROM tickets WHERE project_id = ? AND status = 'todo' ORDER BY priority, created_at
CREATE INDEX idx_tickets_dispatch ON tickets (project_id, status, priority, created_at);

-- 工单看板页：按项目和状态分组
CREATE INDEX idx_tickets_board ON tickets (project_id, status);

-- 工单依赖检查
CREATE INDEX idx_ticket_deps_blocker ON ticket_dependencies (target_ticket_id, type);
CREATE INDEX idx_ticket_deps_source ON ticket_dependencies (source_ticket_id);

-- TicketRepoScope：Webhook 匹配（通过分支名找工单）
CREATE INDEX idx_repo_scopes_branch ON ticket_repo_scopes (repo_id, branch_name);
CREATE INDEX idx_repo_scopes_ticket ON ticket_repo_scopes (ticket_id);

-- Agent 心跳查询
CREATE INDEX idx_agents_heartbeat ON agents (project_id, status, last_heartbeat_at);

-- 活动日志：按工单和时间查询
CREATE INDEX idx_activity_ticket ON activity_events (ticket_id, created_at DESC);
CREATE INDEX idx_activity_project ON activity_events (project_id, created_at DESC);

-- 定时任务：下次执行时间
CREATE INDEX idx_scheduled_next ON scheduled_jobs (next_run_at) WHERE is_enabled = true;
```

### 20.2 JSON 字段索引

```sql
-- 如果需要按 metadata 中的字段查询（如 external_ref 来源的 GitHub Issue ID）
CREATE INDEX idx_tickets_external_ref ON tickets ((metadata->>'external_ref'));

-- ProjectRepo 标签查询（TEXT[] 原生数组）
CREATE INDEX idx_repos_labels ON project_repos USING GIN (labels);
```

### 20.3 ActivityEvent 归档策略

ActivityEvent 是只追加表，会持续增长。采用按月分区 + 自动归档：

```sql
-- 按月分区
CREATE TABLE activity_events (
    id UUID,
    project_id UUID,
    ticket_id UUID,
    event_type TEXT,
    message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

-- 自动创建每月分区（通过 pg_partman 或应用层 cron）
CREATE TABLE activity_events_2026_03 PARTITION OF activity_events
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

归档策略：保留最近 90 天在线数据，更早的分区 `DETACH` 后归档到冷存储或直接 `DROP`（视合规要求）。

### 20.4 分页方案

所有列表接口使用 cursor-based 分页（不用 offset-based，避免深分页性能问题）：

```go
// cursor 是 base64 编码的 (created_at, id) 组合
// SELECT * FROM tickets
//   WHERE project_id = ? AND status = ? AND (created_at, id) < (cursor_time, cursor_id)
//   ORDER BY created_at DESC, id DESC
//   LIMIT 21  -- 多取一条用于判断 has_more
```

---

## 第二十一章 Harness 模板完整示例

### 21.1 Coding Harness（参考 Symphony WORKFLOW.md）

以下是一个生产级别的 Coding Harness 完整示例。该内容由平台控制面持久化；运行时会在启动时把对应版本 materialize 到工作区中的临时文件供 Agent 使用：

````markdown
---
# ═══ Agent 配置 ═══
agent:
  max_turns: 20                    # 最大 Turn 数
  timeout_minutes: 60              # 单工单超时
  max_budget_usd: 5.00             # 单工单预算上限

# ═══ 分支规范 ═══
git:
  branch_pattern: "agent/{{ ticket.identifier }}"
  commit_convention: conventional   # conventional / freeform
  auto_push: true

# ═══ Hook 配置 ═══
hooks:
  on_claim:
{% for repo in repos %}
    - cmd: "git fetch origin && git checkout -B {{ git.branch_pattern }} origin/{{ repo.default_branch }}"
      workdir: "{{ repo.name }}"
      timeout: 60
{% endfor %}
    - cmd: "pnpm install --frozen-lockfile"
      workdir: "frontend"
      timeout: 300
      on_failure: warn
  on_complete:
    - cmd: "make lint"
      timeout: 120
      on_failure: block
    - cmd: "make test"
      timeout: 600
      on_failure: block
    - cmd: "make typecheck"
      timeout: 120
      on_failure: block
  on_done:
    - cmd: "bash scripts/ci/cleanup.sh"
    - cmd: "bash scripts/ci/notify-slack.sh"

---

# Coding Workflow

你是一个专业的软件工程师 Agent，正在处理工单 `{{ ticket.identifier }}`。

## 工单上下文

- 标识: {{ ticket.identifier }}
- 标题: {{ ticket.title }}
- 优先级: {{ ticket.priority }}
- 类型: {{ ticket.type }}

{{ if .Ticket.Description }}
### 工单描述

{{ ticket.description }}
{{ else }}
（无描述，请根据标题推断需求）
{{ end }}

{{ if gt .Attempt 1 }}
## 续接上下文

这是第 {{ attempt }} 次尝试。工作区中可能已有上次的部分成果。
请从当前工作区状态继续，不要从头开始。检查上次失败的原因并修复。
{{ end }}

## 涉及的仓库

{{ range .Repos }}
- **{{ repo.name }}** ({{ repo.labels | join(", ") }}): `{{ repo.path }}`
{{ end }}

## 工作流程

1. **分析需求**：仔细阅读工单描述，理解需要做什么
2. **评估影响范围**：确定需要修改哪些文件、涉及哪些模块
3. **编写实现**：
   - 遵循项目现有的编码风格和架构模式
   - 保持改动最小化，只修改完成需求必需的内容
   - 每个有意义的变更及时 commit（conventional commit 格式）
4. **编写/更新测试**：
   - 为新功能编写测试
   - 确保现有测试仍然通过
   - 目标：新代码的关键路径都有测试覆盖
5. **创建 PR**：
   - PR 标题格式：`{{ ticket.identifier }}: <简洁描述>`
   - PR 描述包含：变更内容、测试方法、关联工单
   - 为每个涉及的仓库创建独立的 PR
6. **自检**：
   - 运行 lint、测试、类型检查
   - 检查是否有遗漏的文件、未清理的调试代码
   - 确认所有 PR 都已提交并关联工单

## 工作边界

- 不要修改与工单无关的文件
- 不要删除现有测试（除非工单明确要求）
- 不要改变项目架构（除非工单明确要求）
- 不要引入新的第三方依赖（除非工单明确要求）
- 不要修改 CI/CD 配置文件
- 遇到歧义时，选择最保守的实现方式
````

### 21.2 Security Harness 示例

````markdown
---
agent:
  max_turns: 15
  timeout_minutes: 45
hooks:
  on_complete:
    - cmd: "make security-report"
      timeout: 300
      on_failure: warn
  on_done:
    - cmd: "bash scripts/ci/notify-security-team.sh"
---

# Security Scan Workflow

你是一个安全工程师 Agent，负责对工单 `{{ ticket.identifier }}` 相关的代码进行安全审计。

## 扫描范围

{{ range .Repos }}
- **{{ repo.name }}**: `{{ repo.path }}`
{{ end }}

## 工作流程

1. **静态分析**：扫描代码中的常见安全问题（注入、XSS、硬编码密钥、不安全的依赖）
2. **依赖审计**：检查依赖中的已知漏洞（CVE）
3. **PoC 编写**：对发现的高风险问题编写漏洞复现代码
4. **生成报告**：输出结构化的安全报告（发现、严重等级、修复建议）
5. **创建修复工单**：对每个需要修复的问题，创建子工单（sub-issue）

## 输出格式

在工单描述中追加安全报告，格式：

```
## Security Report
- [CRITICAL] SQL Injection in auth/login.go:42 — 修复建议：使用参数化查询
- [HIGH] Hardcoded API key in config/secrets.go:15 — 修复建议：移至环境变量
- [MEDIUM] Outdated dependency lodash@4.17.20 (CVE-2021-xxxx) — 修复建议：升级到 4.17.21
```
````

---

## 第二十二章 配置文件 Schema

### 22.1 `~/.openase/config.yaml` 完整 Schema

```yaml
# ═══ 服务 ═══
server:
  host: "0.0.0.0"               # 监听地址（默认 0.0.0.0）
  port: 19836                    # 监听端口（默认 19836）
  mode: "all-in-one"             # all-in-one（默认）| serve | orchestrate
  # mode 决定启动哪些组件：
  #   all-in-one = API + 编排引擎 + 前端（推荐，绝大多数场景）
  #   serve      = 仅 API + 前端（需另启 orchestrate 进程）
  #   orchestrate = 仅编排引擎（需另启 serve 进程）

# ═══ 数据库 ═══
database:
  host: "localhost"              # 必填
  port: 5432                     # 默认 5432
  name: "openase"                # 默认 openase
  user: "openase"                # 必填
  password: "${DB_PASSWORD}"     # 引用 .env 变量
  ssl_mode: "disable"            # disable | require | verify-full
  max_connections: 20            # 连接池上限（默认 20）
  lock_timeout: "5s"             # 死锁超时

# ═══ 认证 ═══
auth:
  mode: "local"                  # local | oidc
  local:
    token: "${OPENASE_AUTH_TOKEN}"  # ≥ 50 字符
  oidc:                          # mode=oidc 时必填
    issuer_url: ""
    client_id: ""
    client_secret: "${OIDC_CLIENT_SECRET}"

# ═══ 编排引擎 ═══
orchestrator:
  tick_interval: "5s"            # 调度间隔（默认 5s）
  max_concurrent_agents: 5       # 全局最大并发（默认 5）
  stall_timeout: "5m"            # Agent 无事件超时（默认 5m）
  max_retry_backoff: "30m"       # 指数退避上限（默认 30 分钟）
  error_alert_threshold: 3       # 连续错误 N 次后通知人类（默认 3，不停止重试）
  workspace_root: "~/.openase/workspace"  # Ticket 工作区基准根目录
  min_disk_free_gb: 1            # 暂停分发的磁盘空间阈值（默认 1GB）

# ═══ 事件通信 ═══
event:
  driver: "auto"                 # auto（默认）| channel | pgnotify
  # auto = 根据 server.mode 自动选择：
  #   all-in-one → channel（Go channel，零开销）
  #   serve / orchestrate → pgnotify（PostgreSQL LISTEN/NOTIFY）
  # 也可手动覆盖：强制用 pgnotify（如 all-in-one 但想测试分布式通信）

# ═══ 通知 ═══
# 通知渠道和订阅规则通过 Web UI / API 配置（第三十三章），不在 config.yaml 中
# 这里只配置全局默认渠道（Wizard 自动生成）
notify:
  default_channel: "log"         # 未配置 NotificationRule 时的 fallback: log | slack
  slack:
    webhook_url: "${SLACK_WEBHOOK_URL}"

# ═══ 可观测性 ═══
observability:
  tracing:
    enabled: false               # 默认关闭
    endpoint: ""                 # OTel Collector gRPC 地址（如 localhost:4317）
  metrics:
    enabled: true                # 默认开启（内存 Metrics，Web UI 仪表盘用）
    export:
      prometheus: false          # 是否暴露 /api/v1/system/metrics
      otlp_endpoint: ""          # OTel Collector 地址

# ═══ 日志 ═══
log:
  level: "info"                  # debug | info | warn | error
  format: "json"                 # json | text（text 适合本地开发）
  output: "stdout"               # stdout | file
  file_path: "~/.openase/logs/openase.log"
  max_age_days: 7                # 日志保留天数

# ═══ Git ═══
git:
  author_name: "OpenASE"
  author_email: "openase@localhost"
```

### 22.2 `~/.openase/.env` 完整 Schema

```bash
# 权限 0600，仅存放敏感信息，config.yaml 通过 ${VAR} 引用

# 数据库
DB_PASSWORD=your_db_password

# 认证
OPENASE_AUTH_TOKEN=your_local_auth_token_at_least_50_characters_long_xxxxx
OIDC_CLIENT_SECRET=                    # mode=oidc 时需要

# Agent CLI API Keys
ANTHROPIC_API_KEY=sk-ant-xxx           # Claude Code 使用
OPENAI_API_KEY=sk-xxx                  # Codex 使用
GOOGLE_API_KEY=xxx                     # Gemini 使用

# 通知
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx

# 注意：
# - GitHub 出站凭证 GH_TOKEN 不在 .env 中落盘。
# - GH_TOKEN 必须存放在平台 Secret 存储层，并由平台按 org/project 作用域解析，
#   然后在本机 go-git 或远端受控 session 中临时投影使用。

# 可观测性（可选）
OTEL_EXPORTER_OTLP_ENDPOINT=
```

### 22.3 config.yaml 验证规则

```go
// 启动时校验，失败则拒绝启动并输出明确的错误信息
func validateConfig(cfg Config) error {
    // 必填项
    if cfg.Database.Host == "" { return fmt.Errorf("database.host is required") }
    if cfg.Database.User == "" { return fmt.Errorf("database.user is required") }

    // auth token 长度（个人模式下 Wizard 自动生成，不需要用户手动填）
    if cfg.Auth.Mode == "local" && len(cfg.Auth.Local.Token) < 50 {
        return fmt.Errorf("auth.local.token must be at least 50 characters (auto-generated by Setup Wizard)")
    }

    // OIDC 完整性
    if cfg.Auth.Mode == "oidc" && cfg.Auth.OIDC.IssuerURL == "" {
        return fmt.Errorf("auth.oidc.issuer_url is required when auth.mode=oidc")
    }

    // 端口范围
    if cfg.Server.Port < 1024 || cfg.Server.Port > 65535 {
        return fmt.Errorf("server.port must be between 1024 and 65535")
    }

    // server.mode 有效值
    validModes := map[string]bool{"all-in-one": true, "serve": true, "orchestrate": true}
    if !validModes[cfg.Server.Mode] {
        return fmt.Errorf("server.mode must be one of: all-in-one, serve, orchestrate")
    }

    // event.driver 有效值 + 组合校验
    validDrivers := map[string]bool{"auto": true, "channel": true, "pgnotify": true}
    if !validDrivers[cfg.Event.Driver] {
        return fmt.Errorf("event.driver must be one of: auto, channel, pgnotify")
    }
    // channel 只在 all-in-one 模式下有效（单进程才能共享 Go channel）
    if cfg.Event.Driver == "channel" && cfg.Server.Mode != "all-in-one" {
        return fmt.Errorf("event.driver=channel requires server.mode=all-in-one (Go channel cannot cross process boundary)")
    }

    return nil
}
```

---

## 第二十三章 升级与迁移策略

### 23.1 版本号规范

OpenASE 遵循语义化版本（SemVer）：`MAJOR.MINOR.PATCH`

| 类型 | 含义 | 数据库迁移 | 配置变更 |
|------|------|-----------|---------|
| PATCH (0.1.1 → 0.1.2) | Bug 修复 | 无 | 无 |
| MINOR (0.1.x → 0.2.0) | 新功能 | 可能有（自动向前兼容） | 可能有新配置项（有默认值） |
| MAJOR (0.x → 1.0) | Breaking change | 可能有（需 review） | 可能有不兼容变更 |

### 23.2 升级流程

```bash
# 1. 停止服务
openase down

# 2. 备份数据库（强烈建议）
pg_dump -U openase openase > backup_$(date +%Y%m%d).sql

# 3. 替换二进制
# 下载新版本或 go install
mv openase-new /usr/local/bin/openase

# 4. 启动服务（自动检测并执行迁移）
openase up
```

### 23.3 自动迁移机制

OpenASE 启动时自动检测数据库版本并执行迁移：

```go
func (s *Server) startup(ctx context.Context) error {
    // 1. 连接数据库
    client, err := ent.Open("postgres", cfg.Database.DSN())

    // 2. 检查当前 schema 版本 vs 目标版本
    // atlas 的 versioned migration 模式
    dir, _ := iofs.New(migrations, "migrations")
    err = client.Schema.Create(ctx,
        schema.WithDir(dir),
        schema.WithDropColumn(false),  // 绝不自动删列
        schema.WithDropIndex(false),   // 绝不自动删索引
    )
    if err != nil {
        return fmt.Errorf("migration failed: %w\n\nRun 'openase migrate status' for details", err)
    }

    // 3. 记录迁移事件
    s.logger.Info("database migration complete", "version", currentVersion)

    return nil
}
```

关键原则：

- **只加不删**：自动迁移只做 `ADD COLUMN`、`CREATE INDEX`、`CREATE TABLE`。永远不自动 `DROP COLUMN` 或 `DROP TABLE`
- **向后兼容**：新版本的 schema 必须能被旧版本的二进制安全读取（新列有默认值，旧代码忽略不认识的列）
- **迁移失败阻止启动**：不会让服务在 schema 不一致的状态下运行

### 23.4 回滚策略

如果新版本有问题需要回滚：

```bash
# 1. 停止新版本
openase down

# 2. 恢复旧二进制
mv openase-old /usr/local/bin/openase

# 3. 如果需要回滚 schema（仅在 MAJOR 升级时可能需要）
psql -U openase openase < backup_20260318.sql

# 4. 启动旧版本
openase up
```

因为"只加不删"的迁移策略，MINOR 版本回滚通常不需要恢复数据库——旧版本的代码会忽略它不认识的新列。只有 MAJOR 版本升级（可能重命名列或改变数据格式）才需要数据库回滚。

### 23.5 迁移 CLI 命令

```bash
openase migrate status    # 查看当前 schema 版本和待执行的迁移
openase migrate up        # 手动执行所有待执行的迁移（与启动时自动执行的相同）
openase migrate history   # 查看迁移执行历史
```

### 23.6 配置兼容性

- 新版本增加的配置项必须有合理默认值，不需要用户手动修改 config.yaml
- 废弃的配置项在 3 个 MINOR 版本内保留支持（打 warning 日志），之后移除
- Setup Wizard 在版本升级后检测到新配置项时，在 Web UI 设置页面提示用户

---

## 第二十四章 测试策略

### 24.1 测试分层总览

DDD 分层的核心价值之一就是可测试性——每一层有明确的输入输出边界，依赖通过接口注入，外部系统全部可 mock。

```
┌────────────────────────────────────────────────────────────────┐
│ E2E Tests (少量，只覆盖关键路径)                                  │
│ 真实 HTTP → 真实 DB → 真实 Agent CLI (mock server)              │
├────────────────────────────────────────────────────────────────┤
│ Integration Tests (中等，验证组件间协作)                           │
│ 真实 DB (testcontainers) + mock Agent CLI + mock Git            │
├────────────────────────────────────────────────────────────────┤
│ Unit Tests (大量，覆盖所有业务逻辑)                               │
│ 纯内存 mock，零外部依赖，毫秒级执行                                │
└────────────────────────────────────────────────────────────────┘
```

### 24.2 逐层测试策略

**Domain / Core Types Layer — 100% 覆盖率目标，全部纯单元测试**

这一层是纯 Go 代码，零外部依赖，零接口调用。测试直接构造实体对象，调用方法，断言结果。没有任何东西需要 mock——因为 domain 层不依赖任何接口。

| 测试对象 | 测试内容 | mock 需求 | 覆盖率目标 |
|---------|---------|-----------|----------|
| `internal/domain/ticketing/retry.go` | 指数退避、预算暂停判定 | 无 | 100% |
| `internal/domain/ticketing/cost.go` | token/cost 解析、金额舍入 | 无 | 100% |
| `internal/domain/catalog/*.go` | 输入解析、UUID/limit/枚举解析、machine/provider 纯规则 | 无 | 100% |
| `internal/domain/notification/channel.go` | 通知渠道类型、配置规范化、消息结构 | 无 | 100% |
| `internal/domain/notification/rule.go` | 订阅规则解析、匹配逻辑 | 无 | 100% |
| `internal/domain/issueconnector/connector.go` | 历史外部同步解析逻辑（非当前范围） | 无 | 100% |
| `internal/types/pgarray/string_array.go` | PostgreSQL array 边界类型 | 无 | 100% |

```go
// internal/domain/ticketing/retry_test.go / internal/ticket/*_test.go — 示例
func Test_Transition_TodoToInProgress_Success(t *testing.T) {
    ticket := NewTicket("ASE-1", "Fix bug")
    ticket.Status = StatusTodo

    err := ticket.TransitionTo(StatusInProgress)

    assert.NoError(t, err)
    assert.Equal(t, StatusInProgress, ticket.Status)
}

func Test_Transition_BacklogToDone_Rejected(t *testing.T) {
    ticket := NewTicket("ASE-1", "Fix bug")
    ticket.Status = StatusBacklog

    err := ticket.TransitionTo(StatusDone)

    assert.ErrorIs(t, err, ErrInvalidTransition)
}

func Test_Transition_InProgressToInReview_BlockedByDependency(t *testing.T) {
    parent := NewTicket("ASE-1", "Parent")
    parent.Status = StatusInProgress  // 未完成

    child := NewTicket("ASE-2", "Child")
    child.Status = StatusInProgress
    child.AddDependency(parent.ID, DependencyBlocks)

    err := child.CanTransitionTo(StatusInReview)

    assert.ErrorIs(t, err, ErrBlockedByDependency)
}
```

**Service / Use-Case Layer — 95%+ 覆盖率目标，mock Repository + Provider**

这一层编排用例：调用 domain service → 调用 repository → 调用 provider。所有依赖都是接口，全部 mock。

| 测试对象 | mock 的接口 | 验证重点 |
|---------|-----------|---------|
| `internal/service/catalog/*.go` | `internal/repo/catalog.Repository`, `provider.ExecutableResolver`, `MachineTester` | 编排 catalog 用例、资源探测、默认值/联动更新 |
| `internal/ticket/*.go` | Ent client / repository 边界、事件总线、状态模板依赖 | 工单创建/状态流转/依赖关系/预算与外链逻辑 |
| `internal/workflow/*.go` | repo / filesystem / provider 边界 | Harness 校验、模板渲染、技能安装、工作流编排 |
| `internal/chat/*.go`、`internal/notification/*.go` | adapter / provider / service mock | 对话编排、通知发送、副作用传播 |

```go
// internal/service/catalog/agent_catalog_test.go — 示例
func TestCreateAgentProviderRejectsMissingExecutable(t *testing.T) {
    // Arrange
    svc := New(&stubRepository{}, stubExecutableResolver{}, nil)

    // Act
    _, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
        OrganizationID: uuid.New(),
        Name:           "Gemini",
        AdapterType:    entagentprovider.AdapterTypeGeminiCli,
        ModelName:      "gemini-2.5-pro",
        AuthConfig:     map[string]any{},
    })

    // Assert
    assert.ErrorIs(t, err, ErrInvalidInput)
}
```

**Infrastructure Layer — 分组件测试策略**

这一层涉及真实外部系统，不同组件测试策略差异很大：

| 组件 | 测试方式 | mock / 真实 | 覆盖率目标 |
|------|---------|------------|----------|
| `internal/repo/` (ent-backed repository adapters) | 集成测试 | **真实 PostgreSQL**（testcontainers-go） | 90% |
| `internal/infra/adapter/claudecode/` | 单元测试 | **mock CLI subprocess**（fake NDJSON stream） | 85% |
| `internal/infra/adapter/codex/` | 单元测试 | **mock JSON-RPC server**（stdin/stdout pipe） | 85% |
| `internal/infra/hook/shell_executor.go` | 单元测试 + 集成 | **真实 shell**（执行测试脚本） | 90% |
| `internal/infra/workspace/` | 集成测试 | **真实文件系统**（temp dir） | 80% |
| `internal/infra/sse/hub.go` | 单元测试 | **mock HTTP ResponseWriter / fake subscribers** | 90% |
| `internal/infra/otel/*.go` | 单元测试 | fake exporter / noop provider | 80-90% |
| `internal/infra/event/channel.go` | 单元测试 | 无外部依赖（纯 Go channel） | 100% |
| `internal/infra/event/pgnotify.go` | 集成测试 | **真实 PostgreSQL** | 85% |
| `internal/notification/*` | 单元测试 | **mock HTTP**（httptest） | 90% |

```go
// internal/infra/adapter/claudecode/adapter_test.go — mock CLI subprocess
func Test_ClaudeCodeAdapter_StreamEvents_ParsesNDJSON(t *testing.T) {
    // 构造一个 fake claude 进程，输出预定义的 NDJSON
    fakeOutput := strings.Join([]string{
        `{"type":"system","subtype":"init","session_id":"sess-123"}`,
        `{"type":"assistant","content":"I'll fix the bug..."}`,
        `{"type":"tool_use","tool":"Edit","input":{"file":"auth.go"}}`,
        `{"type":"result","result":"Bug fixed","cost_usd":0.03}`,
    }, "\n")

    adapter := claudecode.NewAdapter(claudecode.Config{
        // 用 echo 模拟 claude CLI
        Command: "echo",
        Args:    []string{fakeOutput},
    })

    session, _ := adapter.Start(ctx, agentConfig)
    events, _ := adapter.StreamEvents(ctx, session)

    var collected []agent.AgentEvent
    for e := range events {
        collected = append(collected, e)
    }

    assert.Len(t, collected, 4)
    assert.Equal(t, "system", collected[0].Type)
    assert.Equal(t, "result", collected[3].Type)
    assert.InDelta(t, 0.03, collected[3].CostUSD, 0.001)
}
```

```go
// internal/repo/catalog/repo_test.go — 真实 PostgreSQL
func Test_TicketRepo_ListByStatus_WithPagination(t *testing.T) {
    if testing.Short() { t.Skip("requires PostgreSQL") }

    // testcontainers 启动临时 PostgreSQL
    pg := testutils.StartPostgres(t)
    client := ent.NewClient(ent.Driver(pg.Driver()))
    client.Schema.Create(ctx)

    repo := persistence.NewTicketRepo(client)

    // 插入测试数据
    for i := 0; i < 25; i++ {
        repo.Save(ctx, testutils.NewTicket(fmt.Sprintf("ASE-%d", i), ticket.StatusTodo))
    }

    // Act: 分页查询
    page1, cursor, _ := repo.ListByStatus(ctx, ticket.StatusTodo, persistence.Page{Limit: 10})
    page2, _, _ := repo.ListByStatus(ctx, ticket.StatusTodo, persistence.Page{Limit: 10, Cursor: cursor})

    // Assert
    assert.Len(t, page1, 10)
    assert.Len(t, page2, 10)
    assert.NotEqual(t, page1[0].ID, page2[0].ID)  // 不重叠
}
```

**Orchestrator — 单元测试 + 集成测试混合**

| 组件 | 测试方式 | mock 的接口 | 验证重点 |
|------|---------|-----------|---------|
| `internal/orchestrator/scheduler.go` | 单元测试 + 集成测试 | event provider、ent fixture、service 边界 | Tick 调度逻辑、阻塞判断、并发限制、Machine/Agent 选择 |
| `internal/orchestrator/runtime_launcher.go` / `runtime_runner.go` | 单元测试 | `AgentCLIProcessManager`、`TraceProvider`、filesystem 边界 | runtime 启动、事件泵送、session 生命周期 |
| `internal/orchestrator/health_checker.go` | 单元测试 | ent fixture / fake clock | Stall 检测阈值、僵尸 runtime 清理 |
| `internal/orchestrator/machine_monitor.go` | 单元测试 + 集成测试 | SSH/process 边界 | 远端机器可用性、认证状态、监控事件 |
| `internal/orchestrator/retry_service.go` | 单元测试 | ticket/retry 数据 fixture | backoff、恢复、暂停条件 |
| `internal/orchestrator/connector_syncer.go` | 集成测试 | 真实 DB + connector fake | 历史外部同步流程（非当前范围） |
| 调度循环完整流程 | 集成测试 | 真实 DB + mock Adapter | 工单从 todo → claimed → running → review 全流程 |

```go
// internal/orchestrator/scheduler_test.go — 简化示意
func TestSchedulerRunTickSkipsBlockedTickets(t *testing.T) {
    fixture := newSchedulerFixture(t)
    fixture.createBlockedCandidate("ASE-2")
    fixture.createRunnableCandidate("ASE-3")

    report, err := fixture.scheduler.RunTick(context.Background())

    require.NoError(t, err)
    assert.Equal(t, 1, report.TicketsSkipped["blocked"])
    assert.Equal(t, 1, report.TicketsDispatched)
}
```

**Interface / Entry Layer — 薄层，测试 HTTP 契约**

| 组件 | 测试方式 | mock | 验证重点 |
|------|---------|------|---------|
| `internal/httpapi/*.go` | 单元测试 + 集成测试 | service/use-case 边界、provider | HTTP 状态码、请求参数绑定、响应格式、错误映射 |
| `internal/httpapi/tracing.go` | 单元测试 | `TraceProvider` | Span 创建、request_id 注入 |
| `internal/httpapi/sse.go` | 集成测试 | `EventProvider` (ChannelBus) | SSE 事件格式、ping keepalive、过滤逻辑 |
| `cmd/openase/main.go`、`internal/cli/*.go` | 单元测试 | command/service mock | 入口参数、退出码、错误透传 |

```go
// internal/httpapi/ticket_api_test.go — httptest（简化示意）
func Test_CreateTicket_Returns201_WithIdentifier(t *testing.T) {
    server := newHTTPServerFixture(t)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-1/tickets",
        strings.NewReader(`{"title":"Fix bug","priority":"high"}`))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    server.echo.ServeHTTP(rec, req)

    assert.Equal(t, 201, rec.Code)
    assert.Contains(t, rec.Body.String(), "\"identifier\":\"ASE-1\"")
}

func Test_CreateTicket_Returns400_WhenTitleMissing(t *testing.T) {
    server := newHTTPServerFixture(t)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-1/tickets",
        strings.NewReader(`{"priority":"high"}`))  // 缺少 title
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    server.echo.ServeHTTP(rec, req)

    assert.Equal(t, 400, rec.Code)
}
```

### 24.3 哪些接口可以 mock，哪些不能

**可以且应该 mock 的（单元测试中）：**

| 接口 | 定义位置 | mock 生成方式 |
|------|---------|-------------|
| `catalog.Repository` | `internal/repo/catalog/repo.go` | `mockery` 或手写 stub |
| `MachineTester` | `internal/service/catalog/service.go` | 手写 stub / mock |
| `provider.TraceProvider` | `internal/provider/trace.go` | `mockery` 或用 `NoopTracer` |
| `provider.MetricsProvider` | `internal/provider/metrics.go` | `mockery` 或用 `NoopMetrics` |
| `provider.EventProvider` | `internal/provider/event.go` | `mockery` 或用 `ChannelBus`（真实但轻量） |
| `AgentCLIProcessManager` | `internal/provider/agentcli.go` | `mockery` 或 fake manager |
| `UserServiceManager` | `internal/provider/service.go` | `mockery` |

一个原则：**当前 service / provider 边界上暴露的接口都应该有对应的 mock 或 stub。** 优先在稳定的 provider / repository 边界生成 mock；对局部 service 依赖，手写 stub 往往更可控。

**不应该 mock 的（集成测试中必须用真实实现）：**

| 组件 | 为什么不能 mock |
|------|---------------|
| PostgreSQL | ent 的查询行为、事务隔离、LISTEN/NOTIFY 在 mock 中无法真实模拟。用 `testcontainers-go` 启动临时 PostgreSQL |
| 文件系统（workspace, harness） | 文件权限、符号链接、并发写入等边界情况必须在真实文件系统上测试 |
| Shell Hook 执行 | 子进程管理、exit code、stdin/stdout/stderr 管道在 mock 中行为不同 |
| go-git 操作 | Git 的分支、merge、冲突解决逻辑必须在真实 repo 上测试（可以用 temp dir 初始化） |

### 24.4 必须跑通的集成测试路径

以下 6 条路径是系统的核心骨架，必须有端到端的集成测试覆盖，不能只靠 mock：

**路径 1：工单主线（最关键）**

```
创建工单 → 编排引擎 Tick 发现 → on_claim Hook → Agent Adapter 启动
→ Agent 多轮 Turn 执行 / continuation → Agent 显式请求推进状态 → on_complete Hook → 状态推进到 in_review
```

测试环境：真实 PostgreSQL + mock Agent Adapter（返回预定义的 NDJSON 流）+ 真实 Hook（简单的 `exit 0` 脚本）

**路径 2：Hook 失败阻塞**

```
工单 in_progress → Agent 完成 → on_complete Hook 失败（exit 1）
→ 工单留在 in_progress → Agent 收到反馈 → 重试
```

测试环境：真实 PostgreSQL + mock Adapter + 真实 Hook（`exit 1` 脚本）

**路径 3：多 Repo PR 聚合**

```
工单涉及 2 个 Repo → Agent 提交 2 个 PR → Webhook 回调 PR merged
→ 第一个 merged（工单仍在 in_review）→ 第二个 merged（工单转 done）
```

测试环境：真实 PostgreSQL + mock 平台写路径（HTTP/API 调用）

**路径 4：Stall 检测与恢复**

```
Agent 启动 → 5 分钟无事件 → HealthChecker 标记 Stall → Kill Worker
→ 重试（attempt_count + 1）→ 连续 Stall → 通知人类，继续退避重试
```

测试环境：真实 PostgreSQL + mock Adapter（启动后不输出任何事件）+ 缩短 stall_timeout 为 1 秒

**路径 5：崩溃恢复**

```
工单 in_progress → 进程被 kill → 重新启动
→ recoverOnStartup 扫描孤儿工单 → 重置为 todo → 下个 Tick 重新分发
```

测试环境：真实 PostgreSQL + 手动设置工单状态为 in_progress（模拟崩溃）

**路径 6：SSE 实时推送**

```
建立 SSE 连接 → 创建工单 → SSE 收到 ticket.created 事件
→ 工单状态变更 → SSE 收到 ticket.status_changed 事件
```

测试环境：真实 HTTP 服务（httptest.Server）+ ChannelBus + 真实 PostgreSQL

### 24.5 前端测试策略

| 测试类型 | 工具 | 覆盖范围 | 覆盖率目标 |
|---------|------|---------|----------|
| 组件单元测试 | vitest + @testing-library/svelte | UI 组件渲染、props 行为、事件触发 | 80% |
| Store 逻辑测试 | vitest | Svelte store 的状态管理、SSE 事件处理 | 90% |
| API 客户端测试 | vitest + msw (Mock Service Worker) | fetch wrapper 的请求构造、错误处理、重试 | 90% |
| UX smoke / perf regression | Playwright | 6 条关键交互路径的轻量回归；固定 fixture + mock 数据；记录关键步骤时延并断言预算 | 关键路径覆盖；PR 必跑 |

#### 24.5.1 前端交互流畅度回归（非功能性需求）

OpenASE 前端必须把“交互是否顺手”视为明确的非功能性需求，而不是上线后凭主观感觉判断。为避免 Playwright 套件膨胀为笨重且不稳定的端到端测试，PR 阶段只保留一套**轻量 UX smoke / perf regression** 套件，目标如下：

- 每个前端相关 PR 必跑
- 运行时间短、结果稳定、失败信号清晰
- 仅覆盖固定的高频关键路径，不依赖真实外部服务
- 使用 mock / fixture / 稳定测试数据，不要求完整后端编排链路
- 既验证功能可用，也验证关键交互没有明显变钝

这套 Playwright 回归当前只覆盖以下 6 条路径：

1. 项目主导航切换：项目页内 `Board / Machines / Agents / Settings / Scheduled Jobs / Workflows` 等主导航切换后，主内容应快速进入可交互状态
2. Machines 列表查看与编辑：打开 Machines 页面、点击横条卡片、打开 drawer、编辑并保存
3. Machines 快捷操作：触发 `Test connection`，验证即时反馈、完成反馈和资源信息稳定性
4. Repositories 列表查看与编辑：打开仓库列表、打开 drawer、编辑并保存
5. Agents 页面核心交互：打开 provider / registration 等关键 sheet 或 drawer，验证主要操作流
6. Scheduled Jobs / Workflows 操作：打开定时任务或 Workflow 管理页，进入创建或编辑流程并完成提交

Playwright 在 OpenASE 中承担的是“交互回归预算守卫”，而不是完整的真实环境压测。它必须记录并断言以下指标：

| 指标 | 定义 | 用途 |
|------|------|------|
| `route_to_interactive_ms` | 从导航点击或 `goto` 开始，到新页面主交互元素可见且可操作 | 衡量页面切换是否拖沓 |
| `action_feedback_ms` | 从点击开始，到用户看到首个明确反馈（drawer 打开、loading、toast、状态变化） | 衡量“有没有马上响应” |
| `action_complete_ms` | 从动作开始，到操作真正完成且界面稳定 | 衡量完整操作链路耗时 |
| `stability_assertions` | 操作前后关键 UI 是否保持一致，不出现元素消失、状态错乱、无意义跳动 | 衡量交互稳定性 |
| `continuation_ready` | 当前动作完成后，用户是否能立即继续下一步操作 | 衡量流程连续性 |

首批默认预算如下，作为 PR 回归阈值：

| 场景 | 预算 |
|------|------|
| 主导航切换到页面主内容可交互 | `p75 < 800ms` |
| 点击卡片到 drawer 可见 | `p75 < 150ms` |
| drawer 打开到首个输入可编辑 | `p75 < 250ms` |
| 点击 Save 到出现 loading / disabled / 首反馈 | `p75 < 100ms` |
| 点击 Save 到成功 toast 或成功状态出现 | 本地/CI 固定夹具环境 `p75 < 1500ms` |
| 本地筛选或轻量列表更新 | `p75 < 150ms` |
| Test connection 首反馈 | `p75 < 150ms` |

为了保证结果稳定，PR 阶段的 Playwright 套件必须满足这些约束：

- 不依赖真实 GitHub、真实 SSH、真实 Agent CLI、真实外部网络
- 不依赖大型预置数据库或长启动链路
- 优先读取前端埋点的 `performance.mark/measure`，而不是用脆弱的睡眠或肉眼等待近似估计
- 默认只做 smoke + regression，不做全量真实环境业务验收
- 失败报告必须指出是路由切换、drawer 打开、保存反馈还是列表更新超预算，避免“只知道红了，不知道哪里慢了”

推荐的前端埋点命名约定如下，Playwright 和线上 RUM 应共享同一口径：

- `nav:start` / `nav:interactive`
- `drawer:start` / `drawer:visible` / `drawer:ready`
- `save:start` / `save:feedback` / `save:success` / `save:error`
- `filter:start` / `filter:applied`
- `test_connection:start` / `test_connection:feedback` / `test_connection:done`

这条要求属于非功能性需求：**如果某次变更让关键交互路径虽然“功能仍然正确”，但明显变慢、变卡、变得不可预期，则该变更不应视为完成。**

```typescript
// web/src/lib/api/sse.test.ts — SSE store 测试
import { describe, it, expect, vi } from 'vitest'
import { createTicketStream } from './sse'

describe('createTicketStream', () => {
  it('updates store on ticket.created event', () => {
    // mock EventSource
    const mockES = { addEventListener: vi.fn(), onerror: null, onopen: null, close: vi.fn() }
    vi.stubGlobal('EventSource', vi.fn(() => mockES))

    const { tickets } = createTicketStream('project-1')

    // 模拟 SSE 事件
    const handler = mockES.addEventListener.mock.calls.find(c => c[0] === 'ticket.created')[1]
    handler({ data: JSON.stringify({ ticketId: 'ASE-1', ticket: { id: 'ASE-1', status: 'backlog' } }) })

    let value
    tickets.subscribe(v => value = v)
    expect(value.get('ASE-1')).toEqual({ id: 'ASE-1', status: 'backlog' })
  })
})
```

### 24.6 覆盖率目标与现实

**能否做到 100% 覆盖率？**

逐层回答：

| 层 | 100% 可行？ | 现实目标 | 说明 |
|----|-----------|---------|------|
| Domain / Core Types | **是的，必须** | 100% | 主要对应 `internal/domain/*` 与 `internal/types/*` 的纯逻辑与解析代码 |
| Service / Use-Case | 几乎可以 | 95%+ | 当前仓库主要对应 `internal/service/*`、`internal/ticket`、`internal/workflow`、`internal/chat` 等服务包；多数依赖可 mock |
| Infrastructure | 不现实 | 80-90% | 外部系统交互的边界情况难以完整覆盖（网络超时、并发竞争等） |
| Repository / Persistence | 可以接近 | 90% | 当前仓库主要对应 `internal/repo/*`，适合用真实 PostgreSQL 做集成测试 |
| Interface / Entry | 可以接近 | 90%+ | 当前仓库主要对应 `internal/httpapi`、`internal/cli`、`cmd/openase`；HTTP handler 应保持薄，入口 wiring 单独统计 |
| Orchestrator | 不现实 | 85% | 涉及 goroutine 并发、定时器、子进程管理，某些竞态条件难以确定性触发 |
| Frontend | 不现实 | 80% | UI 交互的边界情况（浏览器兼容、动画时序）难以完整覆盖 |

**整体覆盖率目标：75%+，其中 domain 层 100%。**

不追求整体 100%——那会导致为了覆盖率写无意义的测试（比如测试 getter 方法）。用 `go test -coverprofile` 在 CI 中追踪，设 75% 为门槛，domain 层 100% 为硬性要求。

仓库默认通过 `make check` 执行后端测试与覆盖率门禁；CI 与本地 push gate 统一调用 `scripts/ci/backend_coverage.sh`。默认要求是“全量 backend 测试通过 + domain/core 覆盖率阈值检查通过”；如需额外运行全 backend scope 的总覆盖率检查，必须显式设置 `OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true`。

### 24.7 Mock 生成与测试工具链

```yaml
# 在 Makefile 中定义
test-unit:             ## 运行单元测试（domain/core + service/use-case + httpapi）
	go test ./internal/domain/... ./internal/service/... ./internal/ticket ./internal/workflow ./internal/chat ./internal/notification ./internal/httpapi -short -count=1 -coverprofile=coverage-unit.out

test-integration:      ## 运行集成测试（repository + infra + orchestrator，需要 PostgreSQL）
	go test ./internal/repo/... ./internal/infra/... ./internal/orchestrator ./internal/runtime/... -count=1 -coverprofile=coverage-integration.out

test-all:              ## 运行全部测试
	go test ./... -count=1 -coverprofile=coverage-all.out

test-backend-coverage: ## 运行全量后端测试 + domain+types 100% coverage gate（设置 OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true 时额外要求 overall 75%+）
	./scripts/ci/backend_coverage.sh

test-coverage:         ## 覆盖率报告
	go tool cover -func=coverage-all.out | tail -1
	@echo "Domain/Core coverage:"
	go test ./internal/domain/... ./internal/types/... -coverprofile=coverage-domain.out
	go tool cover -func=coverage-domain.out | tail -1

mock-generate:         ## 生成 mock（mockery）
	mockery --all --dir=./internal --output=./mocks --outpkg=mocks

test-frontend:         ## 前端测试
	cd web && pnpm run test

test-e2e:              ## Playwright UX smoke / perf regression（PR 必跑，轻量固定夹具）
	cd web && pnpm exec playwright test
```

### 24.8 测试目录结构

```
openase/
├── internal/
│   ├── domain/
│   │   ├── ticketing/
│   │   │   ├── retry.go
│   │   │   └── retry_test.go             # 纯逻辑单元测试
│   │   ├── notification/
│   │   └── ...
│   ├── service/
│   │   ├── catalog/
│   │   │   ├── service.go
│   │   │   └── agent_catalog_test.go     # 服务层单元测试
│   │   └── ...
│   ├── repo/
│   │   ├── catalog/
│   │   │   ├── repo.go
│   │   │   └── repo_test.go              # 仓储集成测试（testcontainers / Postgres）
│   │   └── ...
│   ├── infra/
│   │   ├── adapter/
│   │   │   ├── claudecode/
│   │   │   │   ├── adapter.go
│   │   │   │   └── adapter_test.go       # 单元测试（fake NDJSON）
│   │   │   └── ...
│   │   └── ...
│   ├── orchestrator/
│   │   ├── scheduler.go
│   │   ├── scheduler_test.go             # 单元/集成混合测试
│   │   └── ...
│   ├── httpapi/
│   │   ├── ticket_api.go
│   │   └── ticket_api_test.go            # HTTP 契约测试
│   └── ...
├── mocks/                                # mockery 自动生成
│   ├── catalog_repository.go
│   ├── agent_adapter.go
│   ├── event_provider.go
│   └── ...
├── tests/
│   ├── integration/
│   │   ├── ticket_lifecycle_test.go      # 路径 1：工单主线
│   │   ├── hook_blocking_test.go         # 路径 2：Hook 阻塞
│   │   ├── multi_repo_pr_test.go         # 路径 3：多 Repo PR
│   │   ├── stall_recovery_test.go        # 路径 4：Stall 检测
│   │   ├── crash_recovery_test.go        # 路径 5：崩溃恢复
│   │   └── sse_realtime_test.go          # 路径 6：SSE 推送
│   ├── testutils/
│   │   ├── postgres.go                   # testcontainers 封装
│   │   ├── fixtures.go                   # 测试数据工厂
│   │   └── fake_adapter.go               # 假 Agent Adapter
│   └── testdata/
│       ├── harnesses/                    # 测试用 Harness 文件
│       └── hooks/                        # 测试用 Hook 脚本
└── web/
    ├── src/lib/
    │   ├── api/sse.test.ts               # SSE store 测试
    │   ├── api/client.test.ts            # API 客户端测试
    │   └── components/ticket/
    │       └── TicketCard.test.ts         # 组件测试
    └── tests/
        └── e2e/                          # Playwright UX smoke / perf regression
            ├── navigation.spec.ts
            ├── machines.spec.ts
            ├── repositories.spec.ts
            ├── agents.spec.ts
            ├── workflows.spec.ts
            └── perf.ts
```

---

## 第二十五章 多机器支持（SSH 控制平面）

### 25.1 场景与动机

科研和工程团队通常拥有多台异构 Linux 机器——GPU 训练服务器、大内存数据处理节点、通用开发机、本地笔记本。不同工单对计算资源的需求不同：模型训练需要 GPU，数据清洗需要大内存，代码重构跑在任意机器即可。

OpenASE 需要支持：

- 将工单绑定到指定机器上执行（或让编排引擎根据资源需求自动选择）
- 在远端机器上启动 Agent CLI 子进程，实时流回事件
- Agent 执行过程中能通过 SSH 访问其他机器（如从 GPU 机器拷贝训练结果到存储机器）
- 实时监控各机器的资源状态（CPU、内存、GPU、磁盘）

### 25.2 架构：单控制平面 + SSH 执行平面

```
┌──────────────────────────────────────────────────────────┐
│ Control Plane (openase all-in-one)                        │
│ 只跑在一台机器上，管理所有状态                                 │
│                                                          │
│ ┌────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│ │ API Server │  │ Orchestrator │  │ Machine Monitor   │  │
│ │ + Web UI   │  │ (Scheduler)  │  │ (SSH health check)│  │
│ └────────────┘  └──────┬───────┘  └────────┬──────────┘  │
│                        │                   │             │
└────────────────────────┼───────────────────┼─────────────┘
                         │ SSH               │ SSH
              ┌──────────┼───────────────────┼──────────┐
              ▼          ▼                   ▼          ▼
         ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
         │ local   │ │ gpu-01  │ │ gpu-02  │ │ storage │
         │ (本机)   │ │ A100×4  │ │ H100×8  │ │ 64GB    │
         │         │ │ 256GB   │ │ 512GB   │ │ 16TB    │
         └─────────┘ └─────────┘ └─────────┘ └─────────┘
           Worker      Worker      Worker      (无 Agent,
           在本地       SSH 启动    SSH 启动     仅供访问)
           执行        Agent CLI   Agent CLI
```

关键设计决策：**控制平面不分布式。** OpenASE 主进程只跑在一台机器上，PostgreSQL 也在这台（或它能连到的机器）上。远端机器不运行任何 OpenASE 组件——只需要有 SSH 可达 + Agent CLI 已安装。这样远端机器是无状态的执行节点，挂了不影响系统。

### 25.3 Machine 实体

新增 `Machine` 作为一等公民实体：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| organization_id | FK | 所属组织 |
| name | String | 机器别名（如 `gpu-01`、`storage`、`local`），组织内唯一 |
| host | String | SSH 连接地址（如 `10.0.1.10` 或 `gpu-01.lab.internal`）。`local` 表示控制平面本机 |
| port | Integer | SSH 端口（默认 22） |
| ssh_user | String | SSH 用户名 |
| ssh_key_path | String | SSH 私钥路径（相对于 `~/.openase/`，如 `keys/gpu-01.pem`） |
| description | Text | 机器描述（Markdown），供 Agent 执行时参考（如"A100 GPU × 4，用于训练"） |
| labels | TEXT[] | 资源标签（如 `{"gpu", "a100", "cuda-12"}` 或 `{"high-memory", "data-processing"}`） |
| status | Enum | online / offline / degraded / maintenance |
| workspace_root | String | 远端 Ticket 工作区根目录（如 `/home/openase/.openase/workspace`） |
| env_vars | TEXT[] | 远端执行时注入的环境变量（如 `{"CUDA_VISIBLE_DEVICES=0,1", "HF_HOME=/data/huggingface"}`） |
| last_heartbeat_at | DateTime | 最后健康检查时间 |
| resources | JSONB | 最近一次采集的资源快照（动态数据，适合 JSONB） |

**Machine `status` 的语义合同：**

- `online`
  - 只表示**机器基础执行面健康**
  - 机器最近一次 L1 reachability 成功
  - 且没有触发阻止调度的基础设施级故障
  - `online` **不表示**任何具体 Provider 已安装、已登录、或一定能成功启动
- `degraded`
  - 机器可达，但存在需要运营关注的问题
  - 例如：磁盘空间过低、L2/L4/L5 检查失败、部分环境异常
  - 默认不参与自动调度，除非未来产品显式支持“degraded 仍可调度”的策略开关
- `offline`
  - 机器基础不可达，不能作为执行节点使用
- `maintenance`
  - 人工维护状态；无论监控结果如何都不得调度

**关键约束：**

- Machine `status` 只回答“这台机器作为执行节点是否健康”
- Provider `availability_state` 才回答“这台机器上的某个具体 Agent CLI 入口能否执行”
- Scheduler 必须同时检查 `machine.status` 与 `provider.availability_state`，不得只看其中一个
- `workspace_root` 只控制 Ticket Workspace 的根目录；不得被复用成别的 repo 路径语义

**`resources` JSONB 结构**（由 Machine Monitor 定期采集）：

```json
{
  "cpu_cores": 32,
  "cpu_usage_percent": 45.2,
  "memory_total_gb": 256,
  "memory_used_gb": 120,
  "memory_available_gb": 136,
  "disk_total_gb": 2000,
  "disk_available_gb": 1200,
  "gpu": [
    {"index": 0, "name": "A100-80G", "memory_total_gb": 80, "memory_used_gb": 12, "utilization_percent": 0},
    {"index": 1, "name": "A100-80G", "memory_total_gb": 80, "memory_used_gb": 0, "utilization_percent": 0}
  ],
  "agent_cli": {
    "claude_code": {"installed": true, "version": "1.2.3", "path": "/usr/local/bin/claude"},
    "codex": {"installed": false},
    "gemini": {"installed": false}
  },
  "collected_at": "2026-03-19T10:30:00Z"
}
```

### 25.4 `local` 机器——默认零配置

系统初始化时自动创建一个 `name=local` 的 Machine 记录，`host=local`。**当 Machine 表中只有 `local` 一条记录时，所有行为与之前完全一致**——Agent 在本机子进程运行，不涉及任何 SSH。用户只有在需要多机器时才去添加远端机器。

这保证了单机用户完全不需要知道多机器功能的存在。

### 25.5 绑定关系：Workflow × Agent × Provider × Machine

多机器语义改为**显式绑定**，而不是运行时“自动猜测该去哪台机器”：

- Workflow 绑定 Agent 定义
- Agent 定义绑定 Provider
- Provider 绑定 Machine
- 因此 Workflow 的每次运行天然落到固定 Machine 上

换句话说，用户在配置 Agent 时，实际是在选择“哪台机器上的哪个 Coding Agent CLI 入口”。Scheduler 运行时只做可用性检查和 semaphore 控制，不再根据 `required_machine_labels` 在多台机器之间做自动匹配。

**调度准入规则：**

- `machine.status == online`
- `provider.availability_state == available`
- Provider semaphore 未满
- Workflow / Stage / Project 并发限制未满

其中：

- Machine `online` 是基础设施健康判断
- Provider `availability_state == available` 是具体 adapter 入口可执行判断
- 两者都必须满足，缺一不可

分配策略（编排引擎 Tick 中）：

```go
func (s *Scheduler) resolveExecutionTarget(ctx context.Context, wf *workflow.Workflow) (*agent.Agent, *provider.AgentProvider, *machine.Machine, error) {
    ag, err := s.agentRepo.Get(ctx, wf.AgentID)
    if err != nil || !ag.IsEnabled {
        return nil, nil, nil, ErrAgentUnavailable
    }

    p, err := s.providerRepo.Get(ctx, ag.ProviderID)
    if err != nil {
        return nil, nil, nil, ErrProviderUnavailable
    }

    m, err := s.machineRepo.Get(ctx, p.MachineID)
    if err != nil || m.Status != machine.StatusOnline {
        return nil, nil, nil, ErrMachineUnavailable
    }

    if p.AvailabilityState != provider.AvailabilityAvailable {
        return nil, nil, nil, ErrProviderUnavailable
    }

    if s.providerSemaphore.Active(p.ID) >= p.MaxParallelRuns {
        return nil, nil, nil, ErrProviderBusy
    }
    return ag, p, m, nil
}
```

如果同一个 Workflow 未来需要支持 GPU 版和 CPU 版、或多机 failover，应通过显式创建多个 Agent/Workflow 绑定，或未来增加“候选 Agent 集合”能力来建模；当前 PRD 不再使用 `required_machine_labels` 作为默认调度入口。

### 25.6 SSH Agent Runner

当 Workflow 绑定到远端 Machine 上的 Provider 时，Worker 通过 SSH 在远端完成整个工单执行流程：创建 Ticket 工作区 → **远端 git clone** → 写入 Prompt → 启动该 Provider 对应的 Agent CLI。

**Repo 策略：远端 git clone（不是 rsync，不依赖共享存储）。** 每台远端机器独立 clone 仓库到自己的本地工作区。原因：远端机器可能在不同网络，共享存储不可靠；git clone 保证每次都是干净的代码状态；Agent 在远端执行 git push 不需要回传文件到控制平面。

**GitHub 仓库鉴权约定：**

- OpenASE 对 GitHub 仓库统一使用平台托管的 `GH_TOKEN` 作为出站凭证。
- 远端 shell `git clone / fetch / push` 必须通过受控环境注入消费这份 `GH_TOKEN`；不得要求用户先手动在远端执行 `gh auth login` 才能工作。
- 本机 `go-git` 路径也必须显式消费同一份 `GH_TOKEN`，保证本机 / 远端行为一致。
- GitHub 仓库 URL 统一优先使用 `https://github.com/...git`，避免把 SSH key 登录态作为平台功能的隐式前提。

```go
// internal/orchestrator/runtime_runner.go — 远端执行（概念示意）
func (w *Worker) runOnRemote(ctx context.Context, m *machine.Machine, p *provider.AgentProvider, t *ticket.Ticket, harness *Harness) error {
    sshClient, err := w.sshPool.Get(ctx, m)
    if err != nil {
        return fmt.Errorf("ssh connect to %s: %w", m.Name, err)
    }

    workDir := fmt.Sprintf("%s/%s/%s/%s", m.WorkspaceRoot, t.OrganizationSlug, t.ProjectSlug, t.Identifier)

    // 1. 创建 Ticket 工作区 + 远端 git clone 所有涉及的 Repo
    repoScopes, _ := w.repoScopeRepo.ListByTicket(ctx, t.ID)
    var cloneCmds []string
    cloneCmds = append(cloneCmds, fmt.Sprintf("mkdir -p %s", workDir))
    for _, scope := range repoScopes {
        repo, _ := w.repoRepo.Get(ctx, scope.RepoID)
        repoDir := fmt.Sprintf("%s/%s", workDir, repo.Name)
        // clone（如果不存在）或 fetch + checkout（如果已存在，重试场景）
        cloneCmds = append(cloneCmds, fmt.Sprintf(
            `if [ -d "%s/.git" ]; then cd %s && git fetch origin && git checkout -B agent/%s origin/%s; else git clone --depth 50 %s %s && cd %s && git checkout -B agent/%s; fi`,
            repoDir, repoDir, t.Identifier, repo.DefaultBranch,
            repo.RepositoryURL, repoDir, repoDir, t.Identifier,
        ))
    }
    session, _ := sshClient.NewSession()
    session.Run(strings.Join(cloneCmds, " && "))

    // 2. 将渲染后的 Harness Prompt 写入远端
    prompt := harness.Render(w.buildTemplateData(t))
    promptPath := fmt.Sprintf("%s/.harness-prompt.md", workDir)
    w.sshWriteFile(sshClient, promptPath, prompt)

    // 3. 注入 Skills 到远端 Agent CLI skills 目录
    //    从控制平面 scp Skills 文件到远端
    for _, skillName := range harness.Skills {
        localSkill := filepath.Join(w.projectSkillsDir, skillName)
        remoteSkill := fmt.Sprintf("%s/.claude/skills/%s", workDir, skillName)
        w.sshCopyDir(sshClient, localSkill, remoteSkill)
    }

    // 4. 启动 Agent CLI，stdout 通过 SSH session 流回控制平面
    execSession, _ := sshClient.NewSession()
    for _, env := range m.EnvVars {
        parts := strings.SplitN(env, "=", 2)
        execSession.Setenv(parts[0], parts[1])
    }
    // 注入 Platform API 环境变量（Agent 在远端也能调用控制平面 API）
    execSession.Setenv("OPENASE_API_URL", w.platformAPIURL)
    execSession.Setenv("OPENASE_AGENT_TOKEN", w.agentToken)

    cmd := fmt.Sprintf("cd %s && %s -p \"$(cat .harness-prompt.md)\" --output-format stream-json --allowedTools \"Bash,Read,Edit,Write,Glob,Grep\" --max-turns %d",
        workDir, p.CLICommand, harness.MaxTurns)

    stdout, _ := execSession.StdoutPipe()
    execSession.Start(cmd)

    // 5. 解析远端 NDJSON 流（与本地完全相同的事件解析逻辑）
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        event, _ := parseAgentEvent(scanner.Bytes())
        w.handleEvent(ctx, t, event)
    }

    return execSession.Wait()
}
```

**关键设计：远端执行和本地执行共享同一个 `AgentAdapter` 接口。** 区别只在于子进程的启动方式：本地用 `os/exec`，远端用 SSH session。Adapter 层面不感知 Machine 的存在——Machine 是编排引擎层面的概念。

**远端 Agent 也能调用 Platform API：** 控制平面的 `OPENASE_API_URL` 和 `OPENASE_AGENT_TOKEN` 通过 SSH 环境变量注入到远端。远端 Agent 通过 HTTP 调用控制平面的 API（创建子工单、更新项目等）——只需要控制平面的端口对远端机器网络可达。

### 25.7 Machine Monitor（金字塔频率监控）

监控采用**金字塔策略**：最轻量的检测最频繁，最重量的检测间隔最长。减少 SSH 开销同时保证关键状态实时性。

**监控层级（从上到下：频率递减，开销递增）：**

```
           ▲ 频率
           │
 Level 1   │  ████████████████████████  每 15 秒
 网络+SSH  │  ICMP ping + SSH 握手
           │
 Level 2   │  ████████████████          每 60 秒
 系统资源  │  CPU/内存/磁盘（一条 SSH 命令）
           │
 Level 3   │  ████████                  每 5 分钟
 GPU 状态  │  nvidia-smi（如有 GPU 标签）
           │
 Level 4   │  ████                      每 30 分钟
 Agent 环境│  CLI 可用性+版本+登录状态
           │
 Level 5   │  ██                        每 6 小时 / 手动触发
 完整审计  │  Git 凭据+GitHub CLI+网络出口
           │
           └──────────────────────────→ 开销
```

**各层级检测内容：**

| 层级 | 频率 | SSH 次数 | 检测项 | 失败行为 |
|------|------|---------|--------|---------|
| **L1: 网络可达** | 15s | 0（ICMP）+ 1（SSH 握手） | ping 可达性 + SSH 端口连通 + SSH 认证成功 | 连续 3 次失败 → `status=offline`，停止分发，通知告警 |
| **L2: 系统资源** | 60s | 1 | CPU 使用率、内存可用量、磁盘可用量 | 磁盘 < 5GB → `status=degraded`；内存 < 10% → 告警 |
| **L3: GPU 状态** | 5min | 1 | nvidia-smi 显存/利用率（仅 `gpu` 标签机器） | 显存全满 → 暂不分发 GPU 工单到该机器 |
| **L4: Agent 环境** | 30min | 1 | Agent CLI 安装状态、版本号、是否已登录 | 更新对应 Provider 可用性；必要时将 Machine 置为 `degraded`，但不直接置为 `offline` |
| **L5: 完整审计** | 6h / 手动 | 1 | Git 凭据、gh CLI、git config、网络出口（curl 测试） | 只记录报告，不自动改状态 |

#### 25.7.1 Machine 状态机

Machine 的状态机以**基础设施级故障**为中心，而不是以 Provider 运行能力为中心：

```text
maintenance --(operator resume + next successful L1)--> online
maintenance --(operator resume + failed L1)-----------> offline

online --(连续 3 次 L1 失败)---------------------------> offline
online --(L2 触发阻止调度的资源问题)-------------------> degraded
online --(operator set maintenance)--------------------> maintenance

degraded --(L1 连续 3 次失败)-------------------------> offline
degraded --(L2/L3 恢复到安全范围)---------------------> online
degraded --(operator set maintenance)------------------> maintenance

offline --(下一次 L1 成功，且无阻止调度问题)---------> online
offline --(operator set maintenance)-------------------> maintenance
```

**状态来源：**

- `offline` 的唯一自动来源是 L1 连续失败
- `degraded` 的自动来源是“机器仍可达，但资源/环境存在问题”
- `maintenance` 的唯一来源是人工操作

**特别约束：**

- L4/L5 失败默认将 Machine 置为 `degraded`，而不是 `offline`
- `local` 机器可以被标记为 `degraded`，但不应因“本机无需 SSH”而跳过 L4/L5 语义校验

#### 25.7.2 Provider 可用性状态机

Provider 可用性是**Machine Monitor L4 的派生结果**，不是配置录入结果：

```text
unknown --(首次成功 L4 且满足所有条件)---------------> available
unknown --(首次成功 L4 但不满足条件)-----------------> unavailable

available --(绑定 machine != online)-----------------> unavailable
available --(L4 明确检测到 cli_missing/not_logged_in/...)-> unavailable
available --(L4 快照过期)----------------------------> stale

unavailable --(后续成功 L4 且满足所有条件)-----------> available
unavailable --(L4 快照过期)--------------------------> stale

stale --(后续成功 L4 且满足所有条件)-----------------> available
stale --(后续成功 L4 但不满足条件)-------------------> unavailable
```

**Provider `available` 的唯一真值来源：**

- 最近一次可信的 L4 Agent Environment 检查
- 以及绑定 Machine 的实时 `status`

以下信号都**不能单独**推出 `provider.available = true`：

- `cli_command` 非空
- 命令名出现在 PATH 中
- 机器仍然 `online`
- 过去某次启动曾成功

**L4 判定必须至少覆盖：**

- 对应 adapter 的 CLI 已安装
- 版本可读
- 认证状态已就绪
- 远端工作区 / 必需环境变量 / 启动路径满足要求

**调度语义：**

- `availability_state == available`：允许调度
- `unknown / unavailable / stale`：禁止调度

#### 25.7.3 前端展示与 API 合同

前端和 API 必须区分 Machine 状态与 Provider 可用性：

- Machine 显示：`Online / Degraded / Offline / Maintenance`
- Provider 显示：`Available / Unavailable / Unknown / Stale`
- 不得把 Machine `online` 渲染成 Provider “可用”
- 不得把 `cli_command` 命中 PATH 渲染成 Provider “可用”

Provider 列表 / 详情接口应返回：

- `availability_state`
- `available`
- `availability_checked_at`
- `availability_reason`

其中 `available` 只是便捷布尔字段，前端应该优先使用 `availability_state`

**L4 详细检测脚本（一条 SSH 命令完成）：**

```bash
# 通过单次 SSH session 执行，最小化连接开销
echo '{'

# Claude Code
echo '"claude_code":{'
if command -v claude >/dev/null 2>&1; then
  echo '"installed":true,'
  echo '"version":"'$(claude --version 2>/dev/null || echo "unknown")'",'
  echo '"auth_status":"'$(claude auth status --text 2>/dev/null | grep -q "Logged in" && echo "logged_in" || echo "not_logged_in")'"'
else
  echo '"installed":false'
fi
echo '},'

# Codex
echo '"codex":{'
if command -v codex >/dev/null 2>&1; then
  echo '"installed":true,'
  echo '"version":"'$(codex --version 2>/dev/null || echo "unknown")'"'
else
  echo '"installed":false'
fi
echo '},'

# Gemini CLI
echo '"gemini":{'
if command -v gemini >/dev/null 2>&1; then
  echo '"installed":true'
else
  echo '"installed":false'
fi
echo '}'

echo '}'
```

**L5 完整审计（按需触发或定时 6 小时）：**

```bash
echo '{'

# Git 配置
echo '"git":{'
echo '"installed":'$(command -v git >/dev/null && echo true || echo false)','
echo '"user_name":"'$(git config --global user.name 2>/dev/null)'",'
echo '"user_email":"'$(git config --global user.email 2>/dev/null)'"'
echo '},'

# GitHub CLI
echo '"gh_cli":{'
if command -v gh >/dev/null 2>&1; then
  echo '"installed":true,'
  echo '"auth_status":"'$(gh auth status 2>&1 | grep -q "Logged in" && echo "logged_in" || echo "not_logged_in")'"'
else
  echo '"installed":false'
fi
echo '},'

# 网络出口测试
echo '"network":{'
echo '"github_reachable":'$(curl -s --max-time 5 https://api.github.com >/dev/null && echo true || echo false)','
echo '"pypi_reachable":'$(curl -s --max-time 5 https://pypi.org >/dev/null && echo true || echo false)','
echo '"npm_reachable":'$(curl -s --max-time 5 https://registry.npmjs.org >/dev/null && echo true || echo false)
echo '}'

echo '}'
```

**补充说明：**

- `gh_cli.auth_status` 仅表示机器上的 GitHub CLI 登录态，是观测数据，不是平台 GitHub 出站鉴权的真相源。
- L5 审计还必须输出一份独立的 `github_token_probe`：
  - `configured`: 是否已配置平台托管 `GH_TOKEN`
  - `valid`: token 是否有效
  - `permissions`: 解析出的权限快照 / scope
  - `repo_access`: 对目标仓库的访问探测结果
  - `checked_at`: 最近一次探测时间
- UI 与调度器应同时展示 `gh_cli.auth_status` 与 `github_token_probe`，其中真正决定 GitHub clone / issue / PR 是否可用的是后者。

**Environment Provisioner——用 Agent 修机器**

当 L4/L5 检测发现环境问题时（如 Claude Code 未安装、Git 凭据缺失），可以用一个 **Environment Provisioner Agent** 自动修复。这个 Agent 通过 SSH 连到目标机器，执行预设的环境配置 Skill：

```yaml
# 内置 Harness: roles/env-provisioner.md
---
status:
  pickup: "环境修复"
  finish: "环境就绪"
skills:
  - openase-platform
  - install-claude-code     # 安装 Claude Code 的 Skill
  - install-codex           # 安装 Codex 的 Skill
  - setup-git               # 配置 Git 凭据的 Skill
  - setup-gh-cli            # 安装配置 GitHub CLI 的 Skill
---

# Environment Provisioner

你负责在远端机器上配置 Agent 运行环境。

## 目标机器

{{ machine.name }} ({{ machine.host }})

## 检测到的问题

{{ ticket.description }}

## 可用的修复 Skill

- install-claude-code: 安装并登录 Claude Code
- install-codex: 安装 Codex CLI
- setup-git: 配置 git user.name / user.email + 凭据
- setup-gh-cli: 安装 gh CLI 并认证

请根据检测到的问题，使用对应 Skill 修复环境。
```

Machine Monitor 检测到环境问题 → 自动创建"环境修复"工单（绑定目标机器）→ Environment Provisioner Agent 接手 → SSH 到目标机器执行 Skill → 修复完成 → 下次 L4 检测验证。

这个 Agent 也接入全局的 Ephemeral Chat（第三十一章），用户可以在 UI 中对着某台机器说"帮我装一下 Claude Code"，AI 助手直接调 Environment Provisioner 的 Skill 处理。

### 25.8 Agent 跨机器访问

Agent 在执行任务时，可能需要访问其他机器（如从 GPU 机器拷贝训练结果到存储机器）。这通过 Harness Prompt 注入机器信息实现：

Harness 模板新增可用变量：

```
{{ machine.name }}           — 当前执行机器名
{{ machine.host }}           — 当前执行机器地址
{{ machine.description }}    — 当前执行机器描述

{{ range .AccessibleMachines }}
- {{ repo.name }} ({{ machine.host }}): {{ machine.description }}
  标签: {{ repo.labels | join(", ") }}
  SSH: ssh {{ machine.ssh_user }}@{{ machine.host }}
{{ end }}
```

渲染后 Agent 看到的 Prompt 片段：

```
## 执行环境

当前机器: gpu-01 (10.0.1.10)
  描述: NVIDIA A100 × 4, 256GB RAM, CUDA 12.2, 用于模型训练
  工作区: /home/openase/.openase/workspace/acme/research/ASE-42/

## 可访问的其他机器

- storage (10.0.1.20): 数据存储服务器, 16TB NVMe, NFS 共享 /data
  SSH: ssh openase@10.0.1.20
- dev-01 (10.0.1.30): 通用开发机, 64GB RAM
  SSH: ssh openase@10.0.1.30

你可以通过 SSH 访问上述机器来传输文件或执行命令。
```

**安全边界**：Agent 能 SSH 到哪些机器由 `project.accessible_machines` 配置决定（白名单）。不在白名单里的机器，Harness 不会注入其信息，Agent 也没有对应的 SSH Key。

Project 表新增字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| accessible_machine_ids | UUID[] | Agent 可访问的机器 ID 列表（白名单） |

### 25.9 SSH 连接池

频繁建立/断开 SSH 连接开销大。编排引擎维护一个 SSH 连接池：

```go
// infra/ssh/pool.go
type Pool struct {
    mu    sync.Mutex
    conns map[string]*ssh.Client  // key: machine_id
    cfg   map[string]SSHConfig    // 连接配置
}

func (p *Pool) Get(ctx context.Context, m *machine.Machine) (*ssh.Client, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    // 复用已有连接
    if client, ok := p.conns[m.ID]; ok {
        // 检查连接是否存活
        _, _, err := client.SendRequest("keepalive@openase", true, nil)
        if err == nil {
            return client, nil
        }
        // 连接已断，移除
        client.Close()
        delete(p.conns, m.ID)
    }

    // 建立新连接
    key, _ := os.ReadFile(filepath.Join("~/.openase", m.SSHKeyPath))
    signer, _ := ssh.ParsePrivateKey(key)
    client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", m.Host, m.Port), &ssh.ClientConfig{
        User:            m.SSHUser,
        Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),  // TODO: 生产环境用 known_hosts
        Timeout:         10 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    p.conns[m.ID] = client
    return client, nil
}
```

### 25.10 Setup Wizard 中的机器管理

Web Setup Wizard 新增机器管理页面（设置 → 机器）：

```
┌─────────────────────────────────────────────┐
│ 机器管理                            [添加机器]  │
├─────────────────────────────────────────────┤
│ ● local                                    │
│   本机 · 在线 · CPU 32核 45% · 内存 32GB       │
│                                             │
│ ● gpu-01                                   │
│   10.0.1.10 · 在线 · GPU A100×4 · 内存 256GB  │
│   标签: gpu, a100, cuda-12                   │
│                                             │
│ ○ gpu-02                                   │
│   10.0.1.11 · 离线（最后在线: 2小时前）          │
│   标签: gpu, h100                            │
│                                             │
│ ● storage                                  │
│   10.0.1.20 · 在线 · 磁盘 16TB (剩余 12TB)    │
│   标签: storage, nfs                         │
└─────────────────────────────────────────────┘
```

添加新机器的流程：

1. 填写 Host、SSH User、上传或指定 SSH Key 路径
2. 点击"测试连接"→ OpenASE SSH 过去执行 `whoami && uname -a`，验证可达
3. 自动采集资源信息（CPU、内存、GPU、已安装的 Agent CLI）
4. 用户补充 Name、Description、Labels
5. 保存 → SSH Key 存入 `~/.openase/keys/`（权限 0600）

### 25.11 CLI 命令

```bash
openase machine list                     # 列出所有机器及状态
openase machine add gpu-01 \
  --host 10.0.1.10 \
  --user openase \
  --key ~/.ssh/gpu-01.pem \
  --labels gpu,a100,cuda-12 \
  --description "Training server, A100×4"  # 添加机器
openase machine test gpu-01              # 测试 SSH 连接 + 采集资源
openase machine remove gpu-01            # 移除机器
openase machine status                   # 所有机器实时资源状态
openase provider add codex-gpu01 \
  --machine gpu-01 \
  --adapter codex-app-server \
  --cmd /usr/local/bin/codex             # 在 gpu-01 上注册一个 Provider

openase agent create training-codex \
  --project research \
  --provider codex-gpu01                 # 创建 Agent 定义，绑定到该 Provider

openase workflow update training \
  --agent training-codex                 # Workflow 绑定该 Agent，执行机器随之确定
```

### 25.12 对现有架构的影响

**改动最小化**——多机器支持是"加法"，不改变现有单机行为：

| 组件 | 影响 |
|------|------|
| Domain / Core Types | 在 `internal/domain/catalog` 中新增 machine 相关类型、解析与纯逻辑 |
| Service / Use-Case | `internal/service/catalog` 增加 Machine 绑定 Provider 的编排与探测逻辑；`internal/ticket` 不再要求手选 machine |
| Orchestrator | `internal/orchestrator` 新增 `resolveExecutionTarget`、`runOnRemote`、`MachineMonitor` |
| Infrastructure | 新增 `internal/infra/ssh/`（连接池 + 命令执行封装） |
| Adapter 层 | **不变**。远端执行时 Adapter 感知不到 Machine，它只看到一个 stdin/stdout 管道，本地是 os/exec，远端是 SSH session |
| Interface / Entry | `internal/httpapi` 和 Web UI 新增 Machine CRUD，以及 Provider / Agent 选择机器化配置入口 |
| Hook | Hook 在远端机器上执行（SSH session 中运行脚本） |
| 数据库 | 新增 `machines` 表；`agent_providers` 新增 `machine_id`；`agent_runs` 冗余记录 `machine_id` 便于审计；`projects` 新增 `accessible_machine_ids` |

**当只有 `local` 一台机器时（默认），所有行为在用户体验上与没有多机器功能时完全一致。** 但为了给 Provider 可用性提供真值来源，系统仍然运行本机版 Machine Monitor：

- 不需要 SSH 连接池
- L1 使用本地 reachability 语义
- L2-L5 使用本地 shell/exec 采集
- Scheduler 仍然要检查 `machine.status` 与 `provider.availability_state`

### 25.13 新增 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/machines` | 列出机器（含实时资源状态） |
| POST | `/api/v1/orgs/:orgId/machines` | 添加机器 |
| GET | `/api/v1/machines/:machineId` | 机器详情 |
| PATCH | `/api/v1/machines/:machineId` | 更新机器配置 |
| DELETE | `/api/v1/machines/:machineId` | 移除机器 |
| POST | `/api/v1/machines/:machineId/test` | 测试 SSH 连接 + 采集资源 |
| GET | `/api/v1/machines/:machineId/resources` | 获取最新资源快照 |

SSE 新增事件类型：`machine.online`、`machine.offline`、`machine.degraded`、`machine.resources_updated`、`provider.available`、`provider.unavailable`、`provider.stale`

### 25.14 新增可观测性指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openase.machine.status` | Gauge | machine_name | 机器状态（1=online, 0=offline, -1=degraded） |
| `openase.machine.cpu_usage` | Gauge | machine_name | CPU 使用率 |
| `openase.machine.memory_available_gb` | Gauge | machine_name | 可用内存 |
| `openase.machine.gpu_utilization` | Gauge | machine_name, gpu_index | GPU 利用率 |
| `openase.machine.ssh_latency_ms` | Histogram | machine_name | SSH 连接延迟 |
| `openase.machine.ssh_errors_total` | Counter | machine_name | SSH 连接错误次数 |
| `openase.provider.availability` | Gauge | provider_name, machine_name, adapter_type | Provider 可用性（1=available, 0=unknown/stale, -1=unavailable） |
| `openase.provider.l4_age_seconds` | Gauge | provider_name, machine_name | 距离最近一次 L4 可用性检查的年龄 |
| `openase.ticket.machine_dispatch` | Counter | machine_name, workflow_type | 工单分发到各机器的次数 |

---

## 第二十六章 Agent 角色体系与智能招募

### 26.1 核心概念：Harness = 角色 JD

OpenASE 中的 Harness 不只是一份"工作规范"——它是一个**角色的完整定义**，类比人力资源中的 JD（Job Description）。一个 Harness 定义了：

- **这个角色是谁**（身份、职责范围）
- **这个角色怎么工作**（工作流程、方法论）
- **这个角色的交付标准**（验收成果、质量要求）
- **这个角色需要什么资源**（机器标签、工具权限）

不同的 Harness 就是不同的"工种"。同一个 Agent CLI（比如 Claude Code）挂载不同的 Harness，就变成了不同的角色——就像同一个人穿不同的工服、拿不同的工具手册，就能胜任不同的岗位。

### 26.2 内置角色库（Harness Marketplace）

OpenASE 内置一个角色库，提供开箱即用的 Harness 模板。用户可以直接使用，也可以 fork 后自定义。

**软件工程角色：**

| 角色 | Harness 名称 | 典型工单 | 核心 Prompt 特征 |
|------|-------------|---------|----------------|
| 全栈开发者 | `roles/fullstack-developer.md` | 功能开发、Bug 修复 | 读需求 → 写代码 → 写测试 → 提 PR |
| 前端工程师 | `roles/frontend-engineer.md` | UI 组件、页面开发 | 关注可访问性、响应式、组件复用 |
| 后端工程师 | `roles/backend-engineer.md` | API 开发、数据模型 | 关注性能、安全、数据一致性 |
| 测试工程师 | `roles/qa-engineer.md` | 编写测试用例、回归测试 | 分析代码路径 → 写单元/集成/E2E 测试 → 覆盖率报告 |
| DevOps 工程师 | `roles/devops-engineer.md` | CI/CD、部署、基础设施 | 编写 Dockerfile、配置流水线、执行部署 |
| 安全工程师 | `roles/security-engineer.md` | 安全审计、漏洞修复 | 扫描漏洞 → 写 PoC → 生成报告 → 创建修复工单 |
| 技术文档专员 | `roles/technical-writer.md` | API 文档、用户指南 | 读代码变更 → 对比现有文档 → 更新文档 → 提 PR |
| 代码审查员 | `roles/code-reviewer.md` | PR Review、代码质量 | 审查 PR → 检查风格/性能/安全 → 提 change request 或 approve |

**产品与研究角色：**

| 角色 | Harness 名称 | 典型工单 | 核心 Prompt 特征 |
|------|-------------|---------|----------------|
| 产品经理 | `roles/product-manager.md` | 需求分析、PRD 撰写 | 调研市场 → 分析竞品 → 撰写需求文档 → 拆分为子工单 |
| 市场分析师 | `roles/market-analyst.md` | 竞品分析、行业趋势 | 搜索行业报告 → 分析竞品功能 → 生成调研报告 |
| 科研 Idea 挖掘 | `roles/research-ideation.md` | 论文调研、方向探索 | 检索最新论文 → 分析研究趋势 → 提出实验假设 → 输出 Idea 报告 |
| 实验验证员 | `roles/experiment-runner.md` | 实验设计、代码实现 | 读 Idea 报告 → 设计实验 → 写实验代码 → 运行 → 记录结果 |
| 报告撰写员 | `roles/report-writer.md` | 实验报告、论文初稿 | 读实验结果 → 组织结构 → 撰写报告/论文 → 生成图表 |
| 数据分析师 | `roles/data-analyst.md` | 数据清洗、可视化 | 读数据集 → 清洗 → 统计分析 → 生成可视化报告 |

**角色 Harness 示例——科研 Idea 挖掘：**

````markdown
---
agent:
  max_turns: 30
  timeout_minutes: 90
  max_budget_usd: 10.00
hooks:
  on_complete:
    - cmd: "test -f idea-report.md"  # 必须产出报告文件
      on_failure: block
execution_target:
  agent_binding_decides_machine: true
---

# 科研 Idea 挖掘

你是一名专业的科研助理，擅长文献调研和研究方向探索。

## 工单上下文

- 工单: {{ ticket.identifier }} — {{ ticket.title }}
- 研究领域: {{ ticket.description }}

{{ range .ExternalLinks }}
- 参考文献/资源: [{{ link.title }}]({{ link.url }})
{{ end }}

## 工作流程

1. **领域理解**：阅读工单描述，理解研究方向和约束条件
2. **文献检索**：
   - 搜索最近 12 个月的相关论文（arXiv、Google Scholar、Semantic Scholar）
   - 重点关注高引用、顶会/顶刊论文
   - 记录每篇论文的核心贡献和局限性
3. **趋势分析**：
   - 识别该领域的 3-5 个活跃研究子方向
   - 分析哪些方向正在上升、哪些已饱和
4. **Gap 识别**：
   - 找出现有工作中尚未解决的问题
   - 找出不同方向之间可以交叉创新的机会
5. **Idea 生成**：
   - 提出 3-5 个具体可行的研究 Idea
   - 每个 Idea 包含：假设、方法概述、预期贡献、初步实验设计、所需资源
6. **输出报告**：
   - 生成 `idea-report.md`，包含完整的文献综述 + Idea 列表
   - 对每个 Idea 标注可行性评分（1-5）和创新性评分（1-5）

## 验收标准

- 引用至少 15 篇相关论文
- 提出至少 3 个可行的研究 Idea
- 每个 Idea 有明确的假设和实验设计
- 报告结构清晰，可直接作为组会讨论材料
````

### 26.3 角色的组合与协作

角色的真正威力在于组合。一个项目可以同时"雇佣"多个角色，通过工单依赖实现协作：

**场景一：MVP 开发团队**

```
产品经理 ──(输出PRD)──→ 全栈开发者 ──(提交PR)──→ 代码审查员
                           │                       │
                           ├──(输出代码)──→ 测试工程师──→ 审查通过
                           │
                           └──(输出代码)──→ 技术文档专员──→ 文档PR
```

对应的工单：

```
ASE-1: [产品经理] 编写用户注册功能 PRD
ASE-2: [全栈开发者] 实现用户注册功能（blocks: ASE-1）
ASE-3: [测试工程师] 编写用户注册测试用例（blocks: ASE-2）
ASE-4: [技术文档专员] 更新 API 文档（blocks: ASE-2）
ASE-5: [代码审查员] Review 用户注册 PR（blocks: ASE-2）
```

**场景二：科研项目**

```
Idea挖掘 ──(输出Idea报告)──→ 实验验证员 ──(输出实验结果)──→ 报告撰写员
     │                           │
     └──→ 市场分析师               └──→ 数据分析师
          (行业背景)                    (数据可视化)
```

**场景三：从 MVP 到持续迭代的角色演进**

| 阶段 | 需要的角色 | 不需要的角色 |
|------|-----------|------------|
| Idea → POC | Idea 挖掘、全栈开发者 | — |
| POC → MVP | 全栈开发者、测试工程师 | Idea 挖掘（任务完成） |
| MVP → 上线 | DevOps 工程师、安全工程师 | — |
| 持续迭代 | 全栈开发者、测试工程师、代码审查员、技术文档专员 | DevOps（按需） |
| 增长期 | + 市场分析师、产品经理 | — |

### 26.4 Agent 招募推荐引擎（HR Advisor）

OpenASE 内置一个 "HR Advisor" 功能，根据项目当前状态推荐应该"招募"哪些角色。

**输入：**

- 项目描述、状态（`Backlog` / `Planned` / `In Progress` / `Completed` / `Canceled` / `Archived`）
- 现有工单分布（多少 coding / test / doc / security）
- 工单状态分布与各状态的排队压力（例如 `Backlog`、`待测试`、`待文档`）
- 已有角色/Workflow 列表，以及每个 Workflow 的 `pickup` / `finish` 状态绑定
- 最近的活动趋势（PR 合并率下降？测试覆盖率低？文档过时？）

**输出：**

- 推荐的角色列表及理由
- 每个角色对应的 Harness 模板（可一键激活）
- 推荐的人力资源配置（几个开发者 + 几个测试 + 几个文档）

**推荐覆盖契约：**

- HR Advisor 必须维护一份显式的“角色推荐支持矩阵”，区分：
  - `supported_now`
  - `intentionally_unsupported`
  - `planned_not_yet_implemented`
- 这份矩阵必须覆盖全部内置角色，不能通过“没写规则”来隐式遗漏角色。
- 当前至少应自动推荐这些角色：`dispatcher`、`fullstack-developer`、`frontend-engineer`、`backend-engineer`、`qa-engineer`、`devops-engineer`、`security-engineer`、`technical-writer`、`code-reviewer`、`product-manager`、`research-ideation`、`experiment-runner`、`report-writer`、`env-provisioner`、`harness-optimizer`。
- `market-analyst` 可以显式标记为 `intentionally_unsupported`，因为它依赖外部市场信号而不是项目内部执行信号。
- `data-analyst` 可以显式标记为 `planned_not_yet_implemented`，直到 snapshot 提供数据集 / 指标 / 分析产物信号。
- 推荐解释必须绑定到可观测信号，例如：状态 stage、lane 排队压力、workflow pickup/finish 绑定、失败重试、文档漂移、研究流程阶段。
- 当同一角色家族对应多个独立 lane 缺口时，HR Advisor 必须保留多个推荐项，而不是按 `RoleSlug` 折叠成一个。

**推荐逻辑示例：**

```go
func (h *HRAdvisor) Recommend(ctx context.Context, project *project.Project) []RoleRecommendation {
    var recs []RoleRecommendation
    stats := h.getProjectStats(ctx, project.ID)

    // 规则 1: 有代码工单但没有测试工单 → 推荐测试工程师
    if stats.CodingTickets > 5 && stats.TestTickets == 0 {
        recs = append(recs, RoleRecommendation{
            Role:     "qa-engineer",
            Reason:   fmt.Sprintf("项目已有 %d 个编码工单，但尚未有测试工单。建议招募测试工程师确保代码质量。", stats.CodingTickets),
            Priority: "high",
        })
    }

    // 规则 2: PR 合并后文档未更新 → 推荐技术文档专员
    if stats.MergedPRsWithoutDocUpdate > 3 {
        recs = append(recs, RoleRecommendation{
            Role:     "technical-writer",
            Reason:   fmt.Sprintf("最近 %d 个 PR 合并后文档未同步更新。建议招募技术文档专员。", stats.MergedPRsWithoutDocUpdate),
            Priority: "medium",
        })
    }

    // 规则 3: 项目进入 In Progress 但仍无 Agent → 推荐全栈开发者
    if project.Status == "In Progress" && stats.TotalAgents == 0 {
        recs = append(recs, RoleRecommendation{
            Role:   "fullstack-developer",
            Reason: "项目已进入 In Progress 但尚无 Agent。建议招募全栈开发者开始实现。",
        })
    }

    // 规则 4: 有安全相关的 Issue 但没有安全 Workflow → 推荐安全工程师
    if stats.SecurityRelatedIssues > 0 && !stats.HasSecurityWorkflow {
        recs = append(recs, RoleRecommendation{
            Role:   "security-engineer",
            Reason: fmt.Sprintf("发现 %d 个安全相关 Issue，但项目未配置安全 Workflow。", stats.SecurityRelatedIssues),
        })
    }

    // 规则 5: 科研项目标签 → 推荐科研角色
    if project.HasLabel("research") {
        if stats.TotalTickets == 0 {
            recs = append(recs, RoleRecommendation{
                Role:   "research-ideation",
                Reason: "科研项目刚启动，建议先招募 Idea 挖掘角色进行文献调研和方向探索。",
            })
        }
    }

    // 规则 6: 某个状态列积压，但没有 Workflow pickup 该列 → 推荐补齐对应 lane
    if stats.StatusQueue["待测试"] >= 2 && !stats.HasPickupWorkflow("待测试") {
        recs = append(recs, RoleRecommendation{
            Role:   "qa-engineer",
            Reason: fmt.Sprintf("状态列待测试中有 %d 个工单，但没有任何 Workflow pickup 该列。建议新增 QA Workflow 处理待测试 lane。", stats.StatusQueue["待测试"]),
        })
    }

    return recs
}
```

实现约束补充：

- 推荐逻辑优先依赖稳定语义：项目状态类型、ticket status `stage`、workflow 绑定关系；只有在区分具体 lane 能力时才回退到状态名关键词。
- `frontend-engineer` / `backend-engineer` / `devops-engineer` / `code-reviewer` 的自动推荐，应来自显式 lane 压力，而不是把所有实现需求都折叠成 `fullstack-developer`。
- `env-provisioner` 和 `harness-optimizer` 的自动推荐，应能从“重试暂停 / failure burst / workflow stall”这类执行退化信号中得出，而不是要求用户先手动发现问题。

**在 Web UI 中的呈现：**

```
┌─────────────────────────────────────────────────────────┐
│ 🤖 Agent 招募建议                            [查看全部]  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ 🔴 高优先级                                              │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 测试工程师 (qa-engineer)                              │ │
│ │ 项目已有 12 个编码工单完成，但尚无测试工单。               │ │
│ │ 建议招募测试工程师确保代码质量。                          │ │
│ │                                                     │ │
│ │ [查看角色 Harness]  [一键激活]  [稍后再说]             │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ 🟡 中优先级                                              │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 技术文档专员 (technical-writer)                        │ │
│ │ 最近 5 个 PR 合并后文档未同步更新。                      │ │
│ │                                                     │ │
│ │ [查看角色 Harness]  [一键激活]  [稍后再说]             │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**"一键激活"做的事情：**

1. 从角色库复制 Harness 模板到项目的 Workflow 控制面存储
2. 创建对应的 Workflow 记录（链接到新 Harness 版本）
3. 注册一个新的 Agent 定义，并将该 Workflow 绑定到这个 Agent
4. 可选：自动创建第一个工单（如测试工程师自动创建"编写现有代码的测试用例"工单）

### 26.5 自定义角色

用户可以创建自己的角色——本质就是编写一个 Harness 文档版本。Web UI 提供角色编辑器：

```
┌──────────────────────────────────────────────────┐
│ 创建新角色                                         │
├──────────────────────────────────────────────────┤
│ 角色名称: [数据库 DBA                           ]  │
│ 描述:     [负责数据库性能优化、迁移、备份         ]  │
│ 标签:     [database] [postgresql] [performance]   │
│ 机器要求: [high-memory]                           │
│                                                   │
│ Harness 编辑器:                                    │
│ ┌───────────────────────────────────────────────┐ │
│ │ ---                                           │ │
│ │ agent:                                        │ │
│ │   max_turns: 15                               │ │
│ │ hooks:                                        │ │
│ │   on_complete:                                │ │
│ │     - cmd: "pg_dump --version"                │ │
│ │ ---                                           │ │
│ │                                               │ │
│ │ # 数据库 DBA                                   │ │
│ │                                               │ │
│ │ 你是一个数据库管理专家...                       │ │
│ └───────────────────────────────────────────────┘ │
│                                                   │
│                    [保存到项目]  [发布到角色库]     │
└──────────────────────────────────────────────────┘
```

"发布到角色库"允许用户将自己的角色分享给社区（类似 GitHub Actions Marketplace 的模式）。

### 26.6 角色与现有概念的映射

这个功能不需要新增实体——它是现有概念的**语义升级**：

| 现有概念 | 角色体系中的语义 | 实体变化 |
|---------|----------------|---------|
| Workflow | 角色的"岗位定义" | 新增 `role_name` 字段（如 "fullstack-developer"） |
| Harness | 角色的"岗位手册"（JD + SOP） | 不变，内容更丰富 |
| Agent | 角色的"在岗员工" | 不变 |
| Ticket | 角色的"工作任务" | 不变 |
| ScheduledJob | 角色的"周期性职责" | 不变 |
| on_complete Hook | 角色的"验收标准" | 不变 |

Workflow 表新增字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| role_name | String | 角色标识（如 `fullstack-developer`、`qa-engineer`），用于推荐引擎匹配 |
| role_description | Text | 角色简介（一句话描述，展示在 UI 中） |
| role_icon | String | 角色图标标识（如 `code`、`shield`、`flask`，对应 Lucide 图标名） |
| is_builtin | Boolean | 是否为内置角色（内置角色不可删除，但可 fork 后自定义） |

### 26.7 对 PRD 核心叙事的影响

角色体系将 OpenASE 的定位从"工单自动化平台"升级为**"AI 工程团队管理平台"**。用户不再是在"分配任务给 Agent"，而是在"组建一支虚拟团队，给不同角色分配职责"。

这个概念转变的好处：

- **降低认知门槛**：用户不需要理解 Workflow、Harness、Hook 这些技术概念，只需要想"我的项目需要什么角色"
- **自然的扩展路径**：项目初期只需要开发者，随着项目成长逐步招募测试、安全、文档、产品经理等角色
- **社区生态的锚点**：角色库是最天然的社区共享内容——"这是我写的 ML 论文审稿人角色，效果很好"

---

## 第二十七章 Agent 自治闭环（Platform API for Agents）

### 27.1 从"被动执行"到"主动经营"

当前设计中，Agent 是纯粹的执行者——接收工单、写代码、提 PR。但真实的软件工程远不止写代码：一个高级工程师会在开发过程中发现需要拆分新仓库、创建后续工单、更新项目文档、调整 CI 定时任务。如果 Agent 不能做这些事，就永远只是一个"高级打字员"。

**闭环的核心思想：Agent 不仅消费平台的工单，也能反向操作平台本身。** OpenASE 的 REST API 对 Agent 和人类一视同仁——Agent 通过工作区内注入的 Platform API Token 调用 OpenASE API，执行受控的平台操作。

```
                    ┌─────────────────────────────┐
                    │         OpenASE 平台          │
                    │                             │
        创建工单 ──→ │  Ticket ──→ Agent 执行 ──→ │ ──→ PR
        配置项目 ──→ │     ↑                  │   │
        管理 Repo ──→│     │    Agent 反向调用   │   │
                    │     │    Platform API    │   │
                    │     └────────────────────┘   │
                    │                             │
                    │  创建子工单、注册新 Repo、       │
                    │  更新项目描述、配置定时任务       │
                    └─────────────────────────────┘
```

### 27.2 Agent Platform API（受控的自治能力）

Agent 在工作区中可以通过 `openase` CLI 或直接 HTTP 调用 Platform API。编排引擎在启动 Agent 时注入以下环境变量：

| 环境变量 | 值 | 说明 |
|---------|-----|------|
| `OPENASE_API_URL` | `http://localhost:19836/api/v1` | Platform API 地址 |
| `OPENASE_AGENT_TOKEN` | `ase_agent_xxx...` | 该 Agent 专属的短期 Token（工单完成后自动失效） |
| `OPENASE_PROJECT_ID` | UUID | 当前项目 ID |
| `OPENASE_TICKET_ID` | UUID | 当前工单 ID |

Agent 在 Harness Prompt 中看到的说明：

```
## 平台操作能力

你可以通过 openase CLI 操作平台（已预装在工作区中）：

  openase ticket create --title "..." --workflow coding    # 创建子工单
  openase ticket link ASE-42 --url https://github.com/... # 关联外部 Issue
  openase project update --description "..."               # 更新项目描述
  openase project add-repo --name "new-service" --url "..." # 注册新仓库
  openase scheduled-job create --name "..." --cron "..."   # 创建定时任务
  openase project update-status "In Progress"              # 更新项目状态

或通过 HTTP API：
  curl -H "Authorization: Bearer $OPENASE_AGENT_TOKEN" \
       $OPENASE_API_URL/projects/$OPENASE_PROJECT_ID/tickets \
       -d '{"title": "...", "workflow_id": "..."}'

所有操作都会记录在活动日志中，标记为由你（Agent）发起。
```

#### 27.2.1 CLI / HTTP 同构原则（采用 `gh` 风格，而非先引入 GraphQL）

OpenASE 的 CLI 不应成为另一套语义来源。**HTTP + OpenAPI 是唯一接口真相源，CLI 只是同一份接口契约的终端投影。**

设计目标：

1. **可达性优先**：任何已暴露的 HTTP API，都必须存在一条 CLI 路径可调用，不能出现“HTTP 有、CLI 没有”的能力黑洞。
2. **同构优先于糖衣**：CLI 的参数名、请求字段名、返回字段名尽量直接对应 HTTP contract；体验优化可以叠加，但不能替代真实接口形状。
3. **OpenAPI 单一真源**：新增/修改 HTTP handler 后，先更新 `api/openapi.json`；CLI 的 typed commands、帮助文本、参数映射都应尽可能由 OpenAPI 生成或校验。
4. **认证分层明确**：
   - 控制面 CLI：面向人类操作者，使用用户态认证。
   - Agent Platform CLI：面向工作区内 Agent，默认读取 `OPENASE_AGENT_TOKEN`，能力受 Harness scope 限制。
5. **流式接口单独建模**：SSE / chat / watch / stream 类型端点不强行塞进 CRUD 资源命令；它们应以独立的 watch/stream/chat 子命令暴露。

**结论：产品上采用类似 GitHub CLI 的双层结构，而不是先引入 GraphQL。**

- 第一层是 raw passthrough：类似 `gh api`
- 第二层是 typed resource commands：类似 `gh issue` / `gh pr`

GraphQL 不是当前阶段前置条件。OpenASE 当前真实接口面是 REST + OpenAPI，因此第一优先级是让 CLI 与 REST contract 同构，而不是再引入一套新的 GraphQL schema 作为并行真相源。

#### 27.2.2 CLI 分层设计

CLI 必须拆成两层：

**A. Raw API 层（保底 100% 可达）**

```bash
openase api GET /api/v1/tickets/$OPENASE_TICKET_ID
openase api POST /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  -f title="补充集成测试" \
  -f workflow_id="..."
openase api PATCH /api/v1/tickets/$OPENASE_TICKET_ID/comments/$COMMENT_ID \
  -f body="$(cat workpad.md)"
```

要求：

- `openase api` 必须允许指定 HTTP method + path，并支持：
  - `-f key=value` 组装 JSON body
  - `--input <file>` 直接提交原始 JSON body
  - `--query key=value` 追加 query string
  - `--header key:value` 追加额外 header
- 默认原样输出 JSON；允许 `--jq`、`--json`、`--template` 做后处理。
- 这层是所有 HTTP 能力的保底出口。即使 typed commands 还未补齐，只要 HTTP 已存在，CLI 就必须可达。

**B. Typed Resource 层（高频、易用、可发现）**

```bash
openase ticket list
openase ticket get 550e8400-e29b-41d4-a716-446655440000
openase ticket comment list 550e8400-e29b-41d4-a716-446655440000
openase ticket comment workpad 550e8400-e29b-41d4-a716-446655440000 --body-file /tmp/workpad.md
openase workflow create ...
openase scheduled-job update ...
```

要求：

- 按资源分组：`ticket`、`project`、`workflow`、`scheduled-job`、`machine`、`provider`、`agent`、`skill` 等。
- 每个 typed command 的参数名尽量直接复用 HTTP 字段名，例如：
  - `status_name`
  - `workflow_id`
  - `external_ref`
- typed command 只是 raw API 的可读封装，不允许内建一套偏离 HTTP 的私有语义。
- typed commands 的帮助文本、必填参数、body schema、默认值、错误提示，优先从 OpenAPI 生成或校验。

#### 27.2.2.1 CLI Help / Discoverability 约束

CLI 的 `--help` 不是装饰，而是接口契约的可发现投影。要求如下：

1. **沿用 Cobra 默认 help 模板**，不另造一套全局渲染器；真正需要统一的是每类命令的 `Long` / `Example` 内容生成策略。
2. **OpenAPI typed CRUD 命令**：
   - `Long` 由统一 builder 生成；
   - 至少说明：命令用途、位置参数来源、UUID 语义、flag/body/query 的来源约束；
   - `watch/stream` 类命令也必须复用同一层生成逻辑，而不是只有一句 `Short`。
3. **Agent Platform 命令**：
   - 不复用 OpenAPI CRUD 的 help builder；
   - 必须通过单独的 platform help builder 统一说明：`OPENASE_API_URL`、`OPENASE_AGENT_TOKEN`、`OPENASE_PROJECT_ID`、`OPENASE_TICKET_ID` 的默认回退逻辑；
   - 需要明确“当前项目 / 当前工单”语义，以及位置参数、flag、环境变量三者的优先级。
4. **高风险命令必须有显式示例**。至少包括：
   - `openase api ...`
   - `openase watch ...`
   - `ticket update`
   - `ticket report-usage`
   - `ticket comment workpad`
   - `project add-repo`
5. **help 文本必须讲清 ID 语义**：
   - 除非某个命令明确声明支持 human-readable identifier，否则 `*Id` 位置参数和 `OPENASE_*_ID` 环境变量都按 UUID 解释；
   - `ASE-2` 这种项目内 identifier 不允许混入 `ticketId` 一类 UUID 参数位。
6. **help 文本必须反映真实执行语义**：
   - `workpad` 要明确是幂等 upsert；
   - `watch/stream` 要明确会保持连接直到用户中断；
   - `report-usage` 要明确是增量上报，不是覆盖总量；
   - `api --input` 与 field 组合、`workpad --body` 与 `--body-file` 互斥等规则，必须在 help 中可见，而不是只在运行时报错。

#### 27.2.3 `cmd/openase` 顶层命名空间约束

`cmd/openase` 的顶层命令必须同时支持：

- **控制面 typed commands**：人类直接操作平台资源
- **Agent Platform typed commands**：Agent 在工作区内回写当前项目/当前工单
- **通用 raw API**：`openase api ...`

推荐结构：

```bash
openase api ...

openase ticket ...
openase project ...
openase workflow ...
openase scheduled-job ...
openase machine ...
openase provider ...
openase agent ...
openase skill ...

openase watch ...
openase stream ...
openase chat ...
```

其中：

- `openase api ...` 是协议级出口，必须保持通用。
- `openase ticket ...` 等 typed commands 是高频资源入口。
- `watch/stream/chat` 是流式或会话式入口，不归入 CRUD 资源树。

#### 27.2.4 输出与脚本友好性

为了让 CLI 既适合人读，也适合 Agent / shell / CI 调用，统一要求：

- 默认输出 JSON，不做与 HTTP 字段不一致的二次命名。
- 支持：
  - `--jq '<expr>'`
  - `--json field1,field2,...`
  - `--template '<go-template-or-equivalent>'`
- 对于 typed commands，`--json` 只能做字段裁剪，不能改写原始字段语义。
- 错误输出必须保留 HTTP status、error code、message，便于脚本判断。
- CLI help 必须同时面向人类与脚本作者：人类需要通过 `Long/Example` 快速理解命令语义，脚本作者需要通过 help 明确参数来源、环境变量回退和字段约束。

#### 27.2.5 Workpad / Comment 的 CLI 一等公民约束

工单评论不是边缘能力，而是 Workflow 闭环的一部分。CLI 必须原生支持：

```bash
openase ticket comment list
openase ticket comment create
openase ticket comment update
openase ticket comment delete
openase ticket comment revisions
openase ticket comment workpad
```

其中 `openase ticket comment workpad` 是一个**幂等 upsert 命令**：

- 查找当前工单下标题为 `## Codex Workpad` 的评论
- 如果存在，则更新同一条评论
- 如果不存在，则创建该评论

`workpad` 是 typed sugar，不替代底层 `comment create/update/list` 的 HTTP 同构能力。

#### 27.2.6 OpenAPI 驱动的实现约束

实现上不允许长期维持“HTTP handler 手写一套、CLI 手写一套、两边慢慢漂”的模式。约束如下：

1. HTTP contract 一旦变更，必须先更新 OpenAPI。
2. CLI 的 typed commands 至少要做到：
   - 参数与 OpenAPI schema 对齐
   - body 字段与 OpenAPI schema 对齐
   - 路由与 method 与 OpenAPI 对齐
3. CI 必须增加一类校验：
   - OpenAPI 已更新但 CLI generation / snapshot 未更新 → fail
   - CLI 引用不存在的 route / 字段 → fail
4. 在 typed commands 尚未覆盖的新接口上，`openase api` 必须立即可用，保证能力面不被 CLI 拖后腿。

### 27.3 Agent 可执行的平台操作

| 操作 | API 端点 | 典型场景 | 权限级别 |
|------|---------|---------|---------|
| **创建子工单** | `POST /tickets` | 开发过程中发现 Bug → 创建 bugfix 工单；安全扫描发现漏洞 → 创建修复工单 | 默认允许 |
| **关联外部链接** | `POST /tickets/:id/links` | 自动关联创建的 PR、发现的相关 Issue | 默认允许 |
| **更新当前工单描述** | `PATCH /tickets/:id` | 补充执行过程中发现的上下文信息 | 默认允许 |
| **维护当前工单评论 / Workpad** | `GET/POST/PATCH /tickets/:id/comments...` | 维护 `## Codex Workpad`、记录进度、阻塞与验证结果 | 默认允许（仅限当前工单） |
| **更新项目描述** | `PATCH /projects/:id` | 产品经理角色完成调研后更新项目 README / 描述 | 需授权 |
| **更新项目状态** | `POST /projects/:id/transition` | 例如将项目从 `Planned` 推进到 `In Progress`，或从 `In Progress` 标记为 `Completed` | 需授权 |
| **注册新仓库** | `POST /projects/:id/repos` | 按照架构设计创建了新的微服务仓库后注册到平台 | 需授权 |
| **创建定时任务** | `POST /scheduled-jobs` | DevOps 角色配置自动化部署/安全扫描的 cron | 需授权 |
| **更新定时任务** | `PATCH /scheduled-jobs/:id` | 调整 cron 频率或工单模板 | 需授权 |
| **创建 Workflow** | `POST /workflows` | 自定义新角色时创建对应 Workflow | 需授权 |
| **查询工单列表** | `GET /tickets` | 了解项目整体进展，避免重复工作 | 默认允许 |
| **查询机器状态** | `GET /machines` | 选择最合适的机器执行计算任务 | 默认允许 |

### 27.4 权限模型：Harness 级别的 Scope 控制

不是所有 Agent 都应该有相同的平台操作权限。一个 coding Agent 可以创建子工单，但不应该改项目状态；一个产品经理角色可以更新项目描述，但不应该创建定时任务。

权限通过 Harness 的 YAML Frontmatter 声明（白名单模式）：

```yaml
---
# 在 Harness 中声明该角色的平台操作权限
platform_access:
  # 默认：只允许创建子工单和关联链接（最小权限）
  allowed:
    - "tickets.create"        # 创建子工单
    - "tickets.update.self"   # 更新当前工单
    - "tickets.link"          # 关联外部链接
    - "tickets.list"          # 查询工单列表
    - "machines.list"         # 查询机器状态

  # 以下需要在 Harness 中显式声明才开放
  # - "projects.update"       # 更新项目信息
  # - "projects.add_repo"     # 注册新仓库
  # - "projects.transition"   # 更新项目状态
  # - "scheduled_jobs.create" # 创建定时任务
  # - "scheduled_jobs.update" # 更新定时任务
  # - "workflows.create"      # 创建 Workflow
---
```

**产品经理角色的 Harness 可能这样配置：**

```yaml
platform_access:
  allowed:
    - "tickets.create"
    - "tickets.update.self"
    - "tickets.link"
    - "tickets.list"
    - "projects.update"        # 可以更新项目描述
    - "projects.transition"    # 可以推进项目状态
```

**DevOps 角色的 Harness：**

```yaml
platform_access:
  allowed:
    - "tickets.create"
    - "tickets.update.self"
    - "tickets.link"
    - "tickets.list"
    - "projects.add_repo"       # 可以注册新仓库
    - "scheduled_jobs.create"   # 可以创建定时任务
    - "scheduled_jobs.update"   # 可以修改定时任务
    - "machines.list"
```

**编排引擎在启动 Agent 时根据 Harness 的 `platform_access` 生成限定 scope 的 Agent Token。** API 层校验 Token scope，越权操作直接返回 403。

### 27.5 典型闭环场景

**场景一：开发过程中自动拆分工单**

```
[用户] 创建工单 ASE-10: "实现用户系统（注册/登录/权限管理）"

[全栈开发者 Agent] 领取 ASE-10，分析需求后认为太大，通过 Platform API 拆分：
  → openase ticket create --title "实现用户注册 API" --parent ASE-10 --workflow coding
  → openase ticket create --title "实现用户登录 API" --parent ASE-10 --workflow coding
  → openase ticket create --title "实现 RBAC 权限模型" --parent ASE-10 --workflow coding
  → openase ticket create --title "编写用户系统测试" --parent ASE-10 --workflow test

[Agent] 更新 ASE-10 描述，补充拆分理由和架构设计
[Agent] 开始处理第一个子工单 ASE-11
```

**场景二：Agent 创建新仓库并注册到平台**

```
[用户] 创建工单 ASE-20: "按照微服务架构拆分 notification 模块为独立服务"

[后端工程师 Agent] 领取 ASE-20：
  1. 分析现有代码中的 notification 模块
  2. 在 GitHub 上创建新仓库：
     → gh repo create acme/notification-service --public
  3. 将 notification 代码迁移到新仓库，提交初始代码
  4. 通过 Platform API 注册新仓库到项目：
     → openase project add-repo --name "notification-service" \
         --url "https://github.com/acme/notification-service" \
         --labels "go,backend,notification"
  5. 在原仓库中删除 notification 模块代码，替换为 SDK 调用
  6. 为两个仓库分别创建 PR
  7. 创建后续工单：
     → openase ticket create --title "为 notification-service 配置 CI/CD" --workflow devops
```

**场景三：安全工程师发现问题后自动触发修复流程**

```
[安全工程师 Agent] 执行安全扫描工单 ASE-30：
  1. 扫描代码发现 3 个高危漏洞
  2. 生成安全报告
  3. 通过 Platform API 自动创建修复工单：
     → openase ticket create --title "修复 SQL 注入: auth/login.go:42" \
         --priority urgent --workflow coding
     → openase ticket create --title "移除硬编码 API Key: config/secrets.go" \
         --priority urgent --workflow coding
     → openase ticket create --title "升级 lodash 到 4.17.21" \
         --priority high --workflow coding
  4. 创建定期安全扫描定时任务：
     → openase scheduled-job create --name "weekly-security-scan" \
         --cron "0 9 * * 1" --workflow security
```

**场景四：产品经理角色驱动项目演进**

```
[产品经理 Agent] 执行工单 ASE-40: "调研竞品并规划 v2.0 功能"
  1. 搜索竞品信息，分析功能差距
  2. 撰写 v2.0 PRD 文档
  3. 更新项目描述：
     → openase project update --description "$(cat v2-prd.md)"
  4. 根据 PRD 拆分为工单：
     → openase ticket create --title "v2.0: 实时协作编辑功能" --type feature --priority high
     → openase ticket create --title "v2.0: 移动端适配" --type feature --priority medium
     → openase ticket create --title "v2.0: 性能优化（首屏 < 2s）" --type refactor --priority high
  5. 推进项目状态：
     → openase project update-status "In Progress"
```

**场景五：refine-harness 元工作流自我优化**

```
[Harness 优化 Agent] 执行工单 ASE-50: "优化 coding Harness"
  1. 查询最近 20 个 coding 工单的执行历史：
     → openase ticket list --workflow coding --status done --limit 20
  2. 分析失败模式（哪些 on_complete Hook 经常失败、哪些工单超时）
  3. 通过 Platform API 更新 coding Harness 正文：
     - 增加"修改前先运行现有测试"的步骤
     - 调整工作边界规则
  4. 平台生成新的 Harness 版本；后续新 runtime 自动使用新版本
  5. 创建验证工单：
     → openase ticket create --title "验证优化后的 coding Harness" --workflow test
```

### 27.6 安全防线

Agent 自治能力必须有明确的安全边界：

**防线一：Token Scope 限制**

Agent Token 只包含 Harness 中 `platform_access.allowed` 声明的权限。API 层校验每个请求的 Token scope，越权操作直接 403。Token 在工单完成后自动失效。

**防线二：操作速率限制**

| 操作类型 | 速率限制 | 理由 |
|---------|---------|------|
| 创建工单 | 20 次 / 工单生命周期 | 防止 Agent 无限拆分子工单 |
| 注册仓库 | 5 次 / 工单生命周期 | 防止创建大量无用仓库 |
| 创建定时任务 | 3 次 / 工单生命周期 | 防止配置大量 cron |
| 更新项目信息 | 10 次 / 工单生命周期 | 防止频繁改写 |

**防线三：显式授权**

高危操作必须在 Harness 中显式声明白名单：

```yaml
platform_access:
  allowed:
    - "projects.add_repo"
```

未显式授权的平台操作直接拒绝执行，不支持“先挂起等待审批再继续”的中间态。

**防线四：ActivityEvent 全量审计**

所有 Agent 发起的平台操作都记录到 ActivityEvent，`created_by` 标记为 `agent:{agent_name}`，`metadata` 包含完整的请求参数。人类可以在活动时间线中追溯 Agent 的所有平台操作。

**防线五：只能操作当前项目**

Agent Token 的 scope 限定在当前 `project_id`，不能跨项目操作。一个项目的 Agent 不能创建另一个项目的工单或修改另一个项目的配置。

### 27.7 对架构的影响

| 组件 | 变化 |
|------|------|
| `internal/httpapi` | 新增 Agent Token 校验逻辑：解析 scope、校验 project_id 边界、速率限制 |
| `internal/orchestrator` Worker | 启动 Agent 时生成 Agent Token，注入环境变量 |
| `internal/agentplatform` / provider contracts | 认证与 Token 校验需要区分 User Token 和 Agent Token |
| Harness 渲染 | 注入 `OPENASE_API_URL`、`OPENASE_AGENT_TOKEN` 等环境变量 |
| `cmd/openase` / CLI | 形成 `openase api` + typed resource commands 双层结构；Agent CLI 默认读 `OPENASE_AGENT_TOKEN`，控制面 CLI 与 HTTP/OpenAPI 保持同构 |
| ActivityEvent | `created_by` 支持 `user:xxx` 和 `agent:xxx` 两种格式 |
| 数据库 | 新增 `agent_tokens` 表（token_hash, agent_id, ticket_id, scopes, expires_at） |

### 27.8 闭环总览

```
人类                          OpenASE 平台                        Agent
  │                              │                                │
  ├── 创建项目 ───────────────→   │                                │
  ├── 配置角色 (Harness) ────→   │                                │
  ├── 创建工单 ───────────────→   │                                │
  │                              │                                │
  │                              ├── 编排引擎分发 ──────────────→   │
  │                              │                                │
  │                              │   ┌──── Agent 执行工作 ─────┐   │
  │                              │   │ 写代码、跑测试、提 PR    │   │
  │                              │   │                          │   │
  │                              │   │ 同时反向操作平台：         │   │
  │                              │ ←─┤   创建子工单             │   │
  │                              │ ←─┤   注册新仓库             │   │
  │                              │ ←─┤   更新项目描述            │   │
  │                              │ ←─┤   配置定时任务            │   │
  │                              │ ←─┤   关联外部 Issue          │   │
  │                              │   └──────────────────────────┘   │
  │                              │                                │
  │   ← SSE 实时更新 ────────    │                                │
  │   ← 审批请求 ────────────    │  ← Hook 验证 ──────────────    │
  │                              │                                │
  ├── 审批 / 反馈 ────────────→  │                                │
  │                              │                                │
  │                              ├── 状态推进 → done ──────────→  │
  │                              │                                │
  │   ← 新工单通知 ──────────    │  （Agent 创建的子工单进入队列）    │
  │                              ├── 编排引擎分发下一个工单 ────→   │
  │                              │                    ...持续循环  │
```

这才是真正的闭环：Agent 不只是任务的消费者，也是任务的生产者和平台的运营者。人类从"监督 Agent"进化为"设定策略（通过 Harness 权限配置），然后观察系统自运转"。

---

## 第二十八章 外部 Issue 同步（当前不做）

### 28.1 决策

当前版本**不实现**外部 Issue 同步能力。包括但不限于：

- 不从 GitHub Issues / GitLab Issues / Jira / Linear 自动创建 Ticket
- 不向这些外部系统回写 Ticket 状态、评论、标签或 PR 结果
- 不提供针对外部 Issue 系统的 inbound webhook 接收端点
- 不把外部 Issue / PR / CI 状态作为 OpenASE 状态机输入

### 28.2 当前替代方式

如果用户需要把外部系统和 OpenASE 联系起来，当前只使用以下显式、低耦合方式：

- 通过 UI / API / Scheduled Job 手动创建 Ticket
- 在 `TicketExternalLink` 中手动保存外部 Issue 或 PR 的 URL / external_id
- 在 `TicketRepoScope` 中手动或由 Agent 显式写入 `pull_request_url`

这些链接只作为上下文引用与跳转入口，不参与自动同步，也不驱动状态推进。

### 28.3 未来扩展边界

如果未来重新引入外部 Issue 同步，必须满足以下前提：

- 作为独立增量能力重新立项，不默认恢复旧的 Webhook / Connector 设计
- 不得隐式改变当前 Ticket 状态机“显式推进”的原则
- RepoScope 的 `pull_request_url` 仍然首先是引用字段；若要引入任何外部状态缓存，必须在新版本 PRD 中单独定义

---

## 第二十九章 自定义工单状态与看板列

### 29.1 问题：硬编码状态不够用

当前 PRD 中工单状态已改为纯自定义（TicketStatus 实体），无硬编码枚举。但不同团队、不同项目的工作流程差异很大：

| 场景 | 需要的状态 | 固定 7 状态的痛点 |
|------|-----------|----------------|
| 科研项目 | idea → literature_review → experiment → writing → submitted → revision → accepted | in_progress 要承载 4 个完全不同的阶段 |
| 运维团队 | reported → triaging → investigating → mitigating → resolved → postmortem | 没有 triaging 和 postmortem 的概念 |
| 设计协作 | brief → wireframe → mockup → review → handoff → implemented | handoff 在 done 之前但不是 in_review |
| 内容生产 | draft → editing → fact_check → legal_review → scheduled → published | 多轮 review 阶段无法表达 |

### 29.2 设计：自定义状态 + 结构化 Stage + Workflow Pickup 驱动

Custom Status 仍然是唯一可流转、可排序、可限流、可展示的看板状态维度；但平台额外为每个 TicketStatus 引入一个结构化 `stage`，用于表达生命周期语义。换句话说：

- `status` 负责团队命名、看板列顺序、颜色、图标、WIP 限流
- `stage` 负责后端统一判断该状态属于积压、未开始、进行中、已完成还是已取消

`stage` 是稳定的少量枚举，不直接显示为看板列标题，也不替代 Custom Status。用户编辑的是“某个 status 对应哪个 stage”，而不是直接编辑 Ticket 自己的 stage。

```text
Ticket.status_id  ─────→ TicketStatus(name="QA Review", stage="started")
                                     │
                                     ├─ UI 展示: "QA Review"
                                     └─ Runtime 语义: started
```

平台约定以下 Stage：

| Stage | 语义 | 是否终态 |
|------|------|---------|
| `backlog` | 尚未进入执行入口的积压态 | 否 |
| `unstarted` | 已准备好但尚未开始 | 否 |
| `started` | 已开始执行，含开发 / 测试 / 评审等进行中阶段 | 否 |
| `completed` | 正向完成 | 是 |
| `canceled` | 主动取消 / 放弃 | 是 |

Stage 的设计目标不是限制用户拖拽，也不是限制 Workflow 可以绑定哪些状态，而是消除后端对 `"Done"` 这类状态名的魔法依赖。依赖解除、终态判断、Dashboard 统计都必须基于 `stage`，而不能基于状态名称猜测语义。

```
科研项目的看板：

  [💡 Idea] → [📋 待调研] → [📚 文献调研] → [🧪 实验中] → [✍️ 写作中] → [📤 已投稿] → [✅ 已录用]

Workflow: research-ideation
  pickup: "待调研"    ← 编排引擎扫描这一列
  finish: "文献调研"   ← Agent 完成后工单移到这列（等用户审核后拖到下一列）

Workflow: experiment-runner
  pickup: "实验中"    ← 用户审核完调研结果，拖到"实验中" → 编排引擎接手
  finish: "写作中"

Workflow: report-writer
  pickup: "写作中"    ← 用户确认实验结果后拖到"写作中" → 编排引擎接手
  finish: "已投稿"
```

**每个列仍然保留团队自己的业务语义；平台生命周期语义由 stage 统一提供。** 同一个"待测试"列，对 test Workflow 来说可以是 pickup（入口），对 coding Workflow 来说也可能是 finish（出口）；但它在系统层面仍然必须映射到某个稳定 stage，例如 `started` 或 `completed`。

### 29.3 数据模型

**TicketStatus 实体（新增）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属项目 |
| name | String | 状态名称（如"文献调研"、"实验中"），项目内唯一 |
| stage | Enum | 生命周期阶段：`backlog` / `unstarted` / `started` / `completed` / `canceled` |
| color | String | 颜色 Hex（如 `#3B82F6`），用于看板显示 |
| icon | String | 图标标识（Lucide 图标名，如 `flask`、`pen`） |
| position | Integer | 看板列排序位置 |
| max_active_runs | Integer (nullable) | 该状态同时允许的最大活跃 AgentRun 数；为空表示不限制 |
| is_default | Boolean | 是否为新工单的默认状态 |
| description | String | 状态描述（可选，鼠标悬浮提示） |

**Ticket 表的 status 字段变更：**

```
之前：status Enum（硬编码 7 个状态）
之后：status_id FK → TicketStatus.id
```

Workflow 表新增字段（替代之前的 Phase 映射）：

```
pickup_status_ids   NonEmptySet<FK → TicketStatus.id>   -- 编排引擎扫描这些状态的工单
finish_status_ids   NonEmptySet<FK → TicketStatus.id>   -- Agent 完成后允许落到这些状态
```

编排引擎查询语义变为：`WHERE status_id IN workflow.pickup_status_ids AND current_run_id IS NULL AND retry_paused = false AND (next_retry_at IS NULL OR next_retry_at <= now())`。命中候选工单后，还需检查其当前命中的 pickup status 自身若配置了 `max_active_runs` 是否已满。没有 fail 状态，没有硬编码状态机，Agent 出错就退避重试。

结构化约束：

- `pickup_status_ids / finish_status_ids` 可以绑定同项目内任意状态；Workflow 自己定义“哪些状态可以领取”和“完成后允许落到哪些状态”
- dependency `blocks` 的解除条件只看 blocker ticket 当前 status 的 `stage` 是否为终态
- Project 默认状态不应落在 `completed` 或 `canceled`，避免新工单直接创建到终态列

### 29.4 默认状态模板

每个新项目创建时自动生成一组默认的 Custom Status，用户可以直接使用、修改或删除后完全自定义：

| 默认 Custom Status | Stage | 颜色 | 说明 | is_default |
|-------------------|------|------|------|-----------|
| Backlog | `backlog` | 灰色 | 积压 | ✅ |
| Todo | `unstarted` | 蓝色 | 就绪等待 | |
| In Progress | `started` | 黄色 | 进行中 | |
| In Review | `started` | 紫色 | 审查中 | |
| Done | `completed` | 绿色 | 已完成 | |
| Cancelled | `canceled` | 灰色 | 已取消 | |

注意：没有 "Failed" 列。工单出错后留在 pickup 列（如 "Todo"）等待重试，前端通过 `consecutive_errors > 0` 标红显示"重试中（第 N 次）"。用户看到的不是"失败"，而是"还在努力"。

内置的 coding Workflow 初始配置 `pickup_status_ids = ["Todo"]`、`finish_status_ids = ["Done"]`。用户修改看板列后只需要对应修改 Workflow 的状态集合配置即可。

如果团队希望“任意时刻只能有 1 个 Agent 在处理某个入口列”，可以直接把该状态的 `max_active_runs` 设为 `1`。例如把 `Todo` 的 `max_active_runs` 设为 `1`，则所有命中 `Todo` 的 Workflow 都会共享这一个列级并发限制；其他工单即使到了该 pickup 列，也会等待前一个 AgentRun 结束后再被领取。

### 29.5 状态转换规则

**用户自由操作**——任何 Custom Status 之间的手动拖拽都允许，没有状态机阻拦。用户想把工单从"已完成"拖回"进行中"？可以。从"Idea"直接跳到"已投稿"？也可以。OpenASE 不判断业务合理性——那是团队自己的流程规范。

但手动拖拽不会改变 `stage` 定义；`stage` 是状态配置本身的属性，不是 ticket 的临时附加标签。也就是说，把工单拖到某个 status，本质上就是切换到“这个 status 对应的 stage”。

**编排引擎自动触发**——只在两个时刻自动改变 status_id：

1. **Agent 完成**
   - 若 `workflow.finish_status_ids` 只有 1 个值 → 自动设置为该状态
   - 若 `workflow.finish_status_ids` 有多个值 → Agent 必须显式选择其中一个；平台必须拒绝集合外的目标状态
2. **Agent 出错** → 清空 `current_run_id`，指数退避后重试（不存在 fail 状态）

**依赖解除规则**——`A blocks B` 的解除条件是：

- A 当前 `status_id` 对应的 `TicketStatus.stage ∈ {completed, canceled}`，或
- A 已显式写入 `completed_at`

系统不得再通过状态名称是否等于 `"Done"`、`"Cancelled"` 等字符串来猜测依赖是否已经解除。

**Hook 不关心具体 status，只关心编排引擎的生命周期事件（claim / start / progress / complete / done / error / cancel）。**

**工单被拖到某个 Workflow 的 pickup 状态时会发生什么：** 编排引擎下个 Tick 扫描到它 → 执行 on_claim Hook → 分配 Agent → 开始执行。如果用户在 Agent 执行过程中把工单拖走了（离开 pickup 状态），编排引擎不管——Agent 继续执行直到完成。只有用户点"取消"才会触发 on_cancel 停止 Agent。

### 29.6 看板视图

自定义状态直接映射为看板的列。用户可以：

- 拖拽列改变顺序
- 添加/删除/重命名列
- 更改列的颜色和图标
- 在列上方看到该状态的工单数量
- 工单从一列拖到另一列 = 状态变更

```
┌─────────────────────────────────────────────────────────────────────┐
│ 科研项目 AlphaFold-Next                           [+ 添加列] [配置] │
├──────────┬──────────┬──────────┬──────────┬──────────┬─────────────┤
│ 💡 Idea  │ 📋 待调研 │ 📚 文献调研│ 🧪 实验中 │ ✍️ 写作中 │ ✅ 已录用    │
│ (3)      │ (2)      │ (1)      │ (2)      │ (0)      │ (5)         │
├──────────┼──────────┼──────────┼──────────┼──────────┼─────────────┤
│ ASE-15   │ ASE-20   │ ASE-18   │ ASE-12   │          │ ASE-1       │
│ ASE-16   │ ASE-21   │          │ ASE-14   │          │ ASE-3       │
│ ASE-17   │          │          │          │          │ ASE-5       │
│          │          │          │          │          │ ASE-8       │
│          │          │          │          │          │ ASE-10      │
└──────────┴──────────┴──────────┴──────────┴──────────┴─────────────┘
```

**看板渲染顺序契约：**

- 看板列直接按 `TicketStatus.position` 从左到右渲染
- `/api/v1/projects/:projectId/statuses` 返回的有序 `statuses` 列表就是看板主渲染契约；前端不应再自行引入额外分组层
- 若某列配置了 `max_active_runs`，前端可在列头显示 `active_runs / max_active_runs`
- 看板列标题永远显示 status 名称；stage 只在设置页、调试信息、规则校验和自动化语义中出现，不替代用户自定义列名

### 29.7 Harness 和 Hook 的兼容性

Harness 和 Hook 不关心具体的 Custom Status 名称——它们只关心编排引擎的生命周期事件：

```yaml
# Harness 中
hooks:
  on_claim:     # 编排引擎接手工单时触发（不管工单在哪个 status）
  on_complete:  # Agent 声明完成时触发
```

```
# Harness Prompt 模板变量
{{ ticket.status }}     → "文献调研"（Custom Status 名称，给 Agent 看的）
```

Agent 在 Prompt 中看到有语义的 Custom Status 名称（"文献调研"比"in_progress"有用得多），编排引擎通过 `pickup_status_ids` 做调度绑定，通过 `stage` 做终态与依赖判定，而不是通过状态名猜语义。

### 29.8 外部同步不参与状态映射

当前版本不做外部 Issue / PR 状态同步，因此 Custom Status 只由 OpenASE 内部显式状态推进驱动：

- 人类在 UI / API 中修改状态
- Agent 通过 Platform API 显式修改状态
- Scheduled Job / Dispatcher 在创建或分流时显式写入状态

外部系统中的 `open / closed / merged / failed` 等状态，不参与 `TicketStatus` 映射。

### 29.9 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:projectId/statuses` | 列出项目的所有 Custom Status（按 `position` 有序返回，包含 `stage`） |
| POST | `/api/v1/projects/:projectId/statuses` | 创建 Custom Status `{name, stage, color, icon, position, max_active_runs?}` |
| PATCH | `/api/v1/statuses/:statusId` | 更新 Custom Status（名称、stage、颜色、图标、排序、并发上限） |
| DELETE | `/api/v1/statuses/:statusId` | 删除 Custom Status（该状态下的工单迁移到项目默认状态或调用方指定的 replacement） |
| POST | `/api/v1/projects/:projectId/statuses/reorder` | 批量更新排序 `{status_ids: [...]}` |
| POST | `/api/v1/projects/:projectId/statuses/reset` | 重置为默认模板 |

### 29.10 数据库索引更新

```sql
-- 编排引擎按 status_id + current_run_id 查询
-- 每个 Workflow 的 pickup_status_ids 集合不同，所以不在索引中固化具体值
CREATE INDEX idx_tickets_dispatch ON tickets (project_id, status_id, current_run_id, priority, created_at);

-- 状态按位置渲染看板 / 统计列级容量
CREATE INDEX idx_statuses_position ON ticket_statuses (project_id, position);

-- 看板按 status_id 分组
CREATE INDEX idx_tickets_board ON tickets (project_id, status_id);
```

### 29.11 对现有章节的影响

| 章节 | 影响 |
|------|------|
| 第六章 领域模型 | Ticket.status 改为 status_id FK；新增 TicketStatus.stage；Workflow 新增 pickup_status_ids / finish_status_ids |
| 第七章 已重写 | 不再有硬编码状态机；编排引擎看 Workflow 的 pickup_status_ids 做调度，`stage` 只用于终态/依赖语义 |
| 第八章 Hook | 触发点不再引用具体状态名，只关心编排生命周期事件 |
| 第十章 编排引擎 | 调度查询改为 `WHERE status_id IN workflow.pickup_status_ids AND current_run_id IS NULL`，并额外检查匹配状态自身的并发上限；依赖解除改为基于 stage |
| 第十六章 伪代码 | 状态判断改为 `contains(wf.PickupStatusIDs, ticket.StatusID)` |
| 第十八章 API | Ticket 响应体包含 `status: {id, name, color, icon, stage}` 嵌套对象 |
| 第二十章 数据库索引 | 索引字段从 status enum 改为 status_id FK |
| 第二十八章 外部同步 | 当前版本不做外部状态映射，Custom Status 只由 OpenASE 内部显式推进 |

核心简化：**没有硬编码状态迁移图，但有结构化 Stage 语义。** 编排引擎只匹配 `status_id ∈ pickup_status_ids`，状态流转仍由用户（看板拖拽）和 Agent（完成时移到允许的 finish 状态之一）驱动；列顺序由 `TicketStatus.position` 决定，并发上限直接由 `TicketStatus.max_active_runs` 表达。Stage 只负责统一回答“这个状态是否属于终态”，以及依赖和统计该如何解释它。

### 29.12 实施与迁移计划

建议按四个阶段实施，避免一次性把 Workflow、调度器、前端一起打碎：

1. **Schema 与领域模型**
   - 给 `ticket_statuses` 增加非空 `stage` 字段
   - 默认模板直接写入 Stage 映射：
     - `Backlog -> backlog`
     - `Todo -> unstarted`
     - `In Progress / In Review -> started`
     - `Done -> completed`
     - `Cancelled -> canceled`
   - HTTP / OpenAPI / 前端状态模型都补齐 `stage`

2. **后端语义切换**
   - dependency 解除逻辑改为只看 blocker status.stage 是否终态
   - Workspace / dashboard / 活跃工单统计全部改用 stage 判断终态
   - 删除任何基于 `"Done"`、`"Cancelled"` 名称的兜底判断

3. **约束与控制面**
   - TicketStatus create/update API 支持编辑 stage
   - Workflow create/update 校验：
     - `pickup_status_ids / finish_status_ids` 只要求非空、引用存在且属于当前项目
   - 状态设置页增加 Stage 选择器，并明确提示“stage 控制生命周期语义”

4. **数据迁移与验证**
   - 迁移旧项目状态：优先按默认模板名称映射；其余状态结合 workflow finish 绑定推断
   - 为无法安全推断的历史状态输出 migration warning，要求人工确认
   - 用集成测试覆盖：
     - blocker 进入 `completed/canceled` 后解除依赖
     - 自定义名称终态不再依赖 `"Done"` 字符串
     - Workflow finish/pickup stage 约束生效

---

## 第三十章 Harness 模板变量字典与编辑器

### 30.1 模板语法

Harness 的 Markdown 正文部分支持 **Jinja2 语法**的变量替换。选择 Jinja2 而非 Go text/template 的理由：Jinja2 对非程序员更友好（`{{ ticket.title }}` 比 `{{ ticket.title }}` 直觉）、支持过滤器（`{{ ticket.description | truncate(500) }}`）、条件和循环语法更自然。

后端使用 Go 的 Jinja2 兼容库（如 `github.com/nikolalohinski/gonja`）渲染模板。

```jinja
{# 变量替换 #}
你正在处理工单 {{ ticket.identifier }}：{{ ticket.title }}

{# 条件 #}
{% if attempt > 1 %}
这是第 {{ attempt }} 次尝试。请从当前工作区状态继续。
{% endif %}

{# 循环 #}
{% for repo in repos %}
- {{ repo.name }} ({{ repo.labels | join(", ") }}): {{ repo.path }}
{% endfor %}

{# 过滤器 #}
{{ ticket.description | default("无描述") | truncate(1000) }}
```

### 30.2 完整变量字典

以下是 Harness 模板中所有可用变量的完整列表。前端 Harness 编辑器中以侧边栏"变量字典"面板展示，点击变量名自动插入到光标位置。

**工单（ticket）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `ticket.id` | string | 工单 UUID | `550e8400-e29b-41d4-a716-446655440000` |
| `ticket.identifier` | string | 可读标识 | `ASE-42` |
| `ticket.title` | string | 工单标题 | `Fix login form validation` |
| `ticket.description` | string | 工单描述（Markdown） | `The login form doesn't validate...` |
| `ticket.status` | string | 当前 Custom Status 名称 | `In Progress` |
| `ticket.priority` | string | 优先级 | `high` |
| `ticket.type` | string | 工单类型 | `bugfix` |
| `ticket.created_by` | string | 创建者 | `user:gary` 或 `agent:claude-01` |
| `ticket.created_at` | string | 创建时间（ISO 8601） | `2026-03-19T10:30:00Z` |
| `ticket.attempt_count` | int | 当前尝试次数 | `1` |
| `ticket.max_attempts` | int | 最大尝试次数 | `3` |
| `ticket.budget_usd` | float | 预算上限（美元） | `5.00` |
| `ticket.external_ref` | string | 外部关联标识 | `octocat/repo#42` |
| `ticket.parent_identifier` | string | 父工单标识（如果是 sub-issue） | `ASE-30` |
| `ticket.url` | string | 工单在 OpenASE Web UI 的链接 | `http://localhost:19836/tickets/ASE-42` |

**工单外部链接（ticket.links）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `ticket.links` | list | 外部链接列表 | — |
| `ticket.links[].type` | string | 链接类型 | `github_issue` |
| `ticket.links[].url` | string | 外部 URL | `https://github.com/acme/backend/issues/42` |
| `ticket.links[].title` | string | 外部标题 | `Login validation broken on Safari` |
| `ticket.links[].status` | string | 外部状态 | `open` |
| `ticket.links[].relation` | string | 关系 | `resolves` |

**工单依赖（ticket.dependencies）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `ticket.dependencies` | list | 依赖列表 | — |
| `ticket.dependencies[].identifier` | string | 依赖工单标识 | `ASE-30` |
| `ticket.dependencies[].title` | string | 依赖工单标题 | `Design user auth schema` |
| `ticket.dependencies[].type` | string | 依赖类型 | `blocks` 或 `sub_issue` |
| `ticket.dependencies[].status` | string | 依赖工单当前状态 | `Done` |
| `ticket.dependencies[].stage` | string | 依赖工单当前状态对应的 Stage | `completed` |

**工单状态语义（ticket.status）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `ticket.status` | string | 当前 Custom Status 名称 | `In Review` |
| `ticket.status_stage` | string | 当前 Custom Status 对应的 Stage | `started` |

**项目（project）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `project.id` | string | 项目 UUID | — |
| `project.name` | string | 项目名称 | `awesome-saas` |
| `project.slug` | string | URL 标识 | `awesome-saas` |
| `project.description` | string | 项目描述 | `A SaaS platform for...` |
| `project.status` | string | 项目状态；规范值为 `Backlog` / `Planned` / `In Progress` / `Completed` / `Canceled` / `Archived` | `In Progress` |
**仓库（repos）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `repos` | list | 本工单涉及的仓库列表（TicketRepoScope） | — |
| `repos[].name` | string | 仓库别名 | `backend` |
| `repos[].url` | string | Git 仓库地址 | `https://github.com/acme/backend` |
| `repos[].path` | string | 工作区中的本地路径 | `/home/openase/.openase/workspace/acme/payments/ASE-42/backend` |
| `repos[].branch` | string | 当前工作分支 | `agent/ASE-42` |
| `repos[].default_branch` | string | 默认分支 | `main` |
| `repos[].labels` | list | 仓库标签 | `["go", "backend", "api"]` |
| `all_repos` | list | 项目下的所有仓库（不只是本工单涉及的） | 结构同 repos |

**Agent（agent）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `agent.id` | string | Agent UUID | — |
| `agent.name` | string | Agent 名称 | `claude-01` |
| `agent.provider` | string | Provider 名称 | `Claude Code` |
| `agent.adapter_type` | string | 适配器类型 | `claude-code-cli` |
| `agent.model` | string | 模型名称 | `claude-sonnet-4-6` |
| `agent.total_tickets_completed` | int | 历史完成工单数 | `47` |

**机器（machine）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `machine.name` | string | 当前执行机器名 | `gpu-01` |
| `machine.host` | string | 机器地址 | `10.0.1.10` |
| `machine.description` | string | 机器描述 | `NVIDIA A100 × 4, 256GB RAM` |
| `machine.labels` | list | 机器标签 | `["gpu", "a100", "cuda-12"]` |
| `machine.workspace_root` | string | 远端 Ticket 工作区根目录 | `/home/openase/.openase/workspace` |
| `accessible_machines` | list | 可访问的其他机器列表 | — |
| `accessible_machines[].name` | string | 机器名 | `storage` |
| `accessible_machines[].host` | string | 地址 | `10.0.1.20` |
| `accessible_machines[].description` | string | 描述 | `数据存储, 16TB NVMe` |
| `accessible_machines[].labels` | list | 标签 | `["storage", "nfs"]` |
| `accessible_machines[].ssh_user` | string | SSH 用户名 | `openase` |

**执行上下文（context）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `attempt` | int | 当前尝试次数（从 1 开始） | `1` |
| `max_attempts` | int | 最大尝试次数 | `3` |
| `workspace` | string | 工作区根路径 | `/home/openase/.openase/workspace/acme/payments/ASE-42` |
| `timestamp` | string | 当前时间（ISO 8601） | `2026-03-19T10:30:00Z` |
| `openase_version` | string | OpenASE 版本号 | `0.3.1` |

**Workflow 配置（workflow）**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `workflow.name` | string | Workflow 名称 | `coding` |
| `workflow.type` | string | Workflow 类型 | `coding` |
| `workflow.role_name` | string | 角色名称 | `fullstack-developer` |
| `workflow.pickup_statuses` | string[] | Pickup 状态名列表 | `["Todo"]` |
| `workflow.finish_statuses` | string[] | Finish 状态名列表 | `["Done"]` |

**平台 API（platform）—— 供 Agent 自治闭环使用**

| 变量 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `platform.api_url` | string | Platform API 地址 | `http://localhost:19836/api/v1` |
| `platform.agent_token` | string | Agent 专属短期 Token | `ase_agent_xxx...` |
| `platform.project_id` | string | 当前项目 ID | UUID |
| `platform.ticket_id` | string | 当前工单 ID | UUID |

### 30.3 内置过滤器

| 过滤器 | 说明 | 示例 |
|--------|------|------|
| `default(value)` | 变量为空时使用默认值 | `{{ ticket.description \| default("无描述") }}` |
| `truncate(length)` | 截断到指定长度 | `{{ ticket.description \| truncate(500) }}` |
| `join(sep)` | 列表拼接为字符串 | `{{ repos[0].labels \| join(", ") }}` |
| `upper` / `lower` | 大小写转换 | `{{ ticket.priority \| upper }}` |
| `length` | 获取列表长度 | `{{ repos \| length }} 个仓库` |
| `first` / `last` | 取列表首/尾 | `{{ repos \| first }}` |
| `sort` | 排序 | `{{ ticket.links \| sort(attribute="type") }}` |
| `selectattr(attr, value)` | 按属性过滤列表 | `{{ repos \| selectattr("labels", "backend") }}` |
| `map(attribute)` | 提取列表中的属性 | `{{ repos \| map(attribute="name") \| join(", ") }}` |
| `tojson` | 输出为 JSON 字符串 | `{{ repos \| tojson }}` |
| `markdown_escape` | 转义 Markdown 特殊字符 | `{{ ticket.title \| markdown_escape }}` |

### 30.4 前端 Harness 编辑器

Web UI 的 Workflow 管理页面中，Harness 编辑器是核心交互组件：

```
┌──────────────────────────────────────────────────────────────────────┐
│ 编辑 Harness: coding.md                      [预览] [保存] [历史]   │
├──────────────────────────────────────┬───────────────────────────────┤
│                                      │ 📖 变量字典            [搜索] │
│  ---                                 │                               │
│  status:                             │ ▼ 工单 (ticket)               │
│    pickup: "Todo"                    │   ticket.identifier    ASE-42 │
│    finish: "Done"                    │   ticket.title         String │
│  agent:                              │   ticket.description   String │
│    max_turns: 20                     │   ticket.status        String │
│  hooks:                              │   ticket.priority      String │
│    on_complete:                      │   ticket.type          String │
│      - cmd: "make test"             │   ticket.budget_usd    Float  │
│  ---                                 │   ticket.links[]       List   │
│                                      │   ticket.dependencies[] List  │
│  # Coding Workflow                   │                               │
│                                      │ ▼ 仓库 (repos)               │
│  你正在处理工单                       │   repos[].name         String │
│  {{ ticket.identifier }}             │   repos[].path         String │
│  ~~~~~~~~~~~~~~~~~~~~~~~~            │   repos[].url          String │
│  (蓝色高亮，悬浮显示当前值)            │   repos[].branch       String │
│                                      │   repos[].labels[]     List   │
│  {% if attempt > 1 %}                │                               │
│  这是第 {{ attempt }} 次尝试          │ ▼ Agent (agent)               │
│  {% endif %}                         │   agent.name           String │
│                                      │   agent.provider       String │
│  ## 涉及的仓库                        │   agent.model          String │
│                                      │                               │
│  {% for repo in repos %}             │ ▼ 机器 (machine)             │
│  - **{{ repo.name }}**:              │   machine.name         String │
│    {{ repo.path }}                   │   machine.host         String │
│  {% endfor %}                        │   machine.description  String │
│                                      │   accessible_machines[] List  │
│                                      │                               │
│                                      │ ▼ 上下文 (context)           │
│                                      │   attempt              Int    │
│                                      │   workspace            String │
│                                      │   timestamp            String │
│                                      │                               │
│                                      │ ▼ 平台 API (platform)        │
│                                      │   platform.api_url     String │
│                                      │   platform.agent_token String │
│                                      │                               │
│                                      │ ▼ 过滤器                      │
│                                      │   default(val)                │
│                                      │   truncate(len)               │
│                                      │   join(sep)                   │
│                                      │   ...                         │
└──────────────────────────────────────┴───────────────────────────────┘
```

**编辑器特性：**

| 特性 | 实现 |
|------|------|
| **语法高亮** | `{{ }}` 变量标签蓝色高亮，`{% %}` 控制标签绿色高亮，`{# #}` 注释灰色 |
| **悬浮预览** | 鼠标悬浮在 `{{ ticket.identifier }}` 上 → 提示框显示"当前值: ASE-42（最近一个工单的实际值）" |
| **自动补全** | 输入 `{{ t` → 弹出 `ticket.identifier`、`ticket.title`、`timestamp` 等候选 |
| **变量字典面板** | 右侧固定面板，分组展示所有可用变量，点击变量名 → 插入 `{{ variable_name }}` 到光标位置 |
| **YAML Frontmatter 校验** | 实时校验 YAML 语法，标红错误行 |
| **实时预览** | 点击"预览"按钮 → 用最近一个工单的真实数据渲染模板，展示 Agent 实际看到的 Prompt |
| **Diff 视图** | 点击"历史"→ 展示 Git log，选择两个版本对比 Diff |
| **变量未定义检测** | 如果模板中使用了字典中不存在的变量名 → 编辑器标黄警告 |
| **快捷片段** | 常用模板片段快速插入（如"标准工作流程"、"工作边界"、"验收标准"） |

**实时预览的数据来源：**

```go
// 渲染预览时，用最近一个该 Workflow 的已完成工单数据
func (h *HarnessEditor) PreviewData(ctx context.Context, workflowID string) TemplateData {
    // 找最近一个已完成的工单
    recent, err := h.ticketRepo.FindLatestByWorkflow(ctx, workflowID)
    if err != nil {
        // 没有历史工单 → 用 mock 数据
        return mockTemplateData()
    }
    return buildTemplateData(recent)
}
```

### 30.5 模板校验

保存 Harness 时后端执行校验，确保模板可以安全渲染：

```go
func validateHarness(content string) []ValidationError {
    var errors []ValidationError

    // 1. YAML Frontmatter 语法校验
    frontmatter, body, err := parseHarness(content)
    if err != nil {
        errors = append(errors, ValidationError{Line: 1, Message: "YAML 语法错误: " + err.Error()})
    }

    // 2. Jinja2 模板语法校验（不执行渲染，只检查语法）
    _, err = gonja.FromString(body)
    if err != nil {
        errors = append(errors, ValidationError{Message: "模板语法错误: " + err.Error()})
    }

    // 3. 变量引用检查：模板中使用的变量是否都在字典中
    usedVars := extractVariables(body)  // 解析出所有 {{ xxx }} 中的变量名
    for _, v := range usedVars {
        if !isKnownVariable(v) {
            errors = append(errors, ValidationError{
                Message: fmt.Sprintf("未知变量 '%s'，请检查变量字典", v),
                Level:   "warning",  // 警告而非错误（允许保存，但提醒用户）
            })
        }
    }

    // 4. pickup/finish 状态引用检查
    if frontmatter.Status.Pickup != "" {
        if !statusExists(frontmatter.Status.Pickup) {
            errors = append(errors, ValidationError{
                Message: fmt.Sprintf("pickup 状态 '%s' 在项目中不存在", frontmatter.Status.Pickup),
            })
        }
    }

    return errors
}
```

### 30.6 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/harness/variables` | 获取完整变量字典（JSON 格式，含分组、类型、说明、示例值） |
| POST | `/api/v1/harness/validate` | 校验 Harness 内容（语法 + 变量引用） |
| POST | `/api/v1/harness/preview` | 用真实数据渲染模板预览 `{workflow_id, content}` |
| GET | `/api/v1/harness/snippets` | 获取常用模板片段列表 |

变量字典 API 返回格式：

```json
{
  "groups": [
    {
      "name": "工单",
      "prefix": "ticket",
      "variables": [
        {
          "key": "ticket.identifier",
          "type": "string",
          "description": "可读标识",
          "example": "ASE-42",
          "insertText": "{{ ticket.identifier }}"
        },
        {
          "key": "ticket.links",
          "type": "list",
          "description": "外部链接列表",
          "children": [
            {"key": "type", "type": "string", "example": "github_issue"},
            {"key": "url", "type": "string", "example": "https://..."},
            {"key": "title", "type": "string"},
            {"key": "status", "type": "string"}
          ],
          "insertText": "{% for link in ticket.links %}\n- [{{ link.type }}] {{ link.title }}: {{ link.url }}\n{% endfor %}"
        }
      ]
    }
  ],
  "filters": [
    {"name": "default", "args": "value", "description": "变量为空时使用默认值", "example": "{{ ticket.description | default(\"无描述\") }}"},
    {"name": "truncate", "args": "length", "description": "截断到指定长度"},
    {"name": "join", "args": "separator", "description": "列表拼接为字符串"}
  ]
}
```

---

## 第三十一章 内嵌 AI 助手（Ephemeral Chat / Project Conversation）

### 31.1 定位

OpenASE 的核心仍然是工单驱动的编排系统，但日常使用中存在两类**不需要创建工单**的轻量 AI 交互：

- 临时型辅助
  - 编辑 Harness 时让 AI 帮忙写/改 Prompt
  - 在工单详情页问“ASE-42 为什么失败了”
  - 在 cron 输入框旁让 AI 帮忙生成表达式
- 持续型项目对话
  - 在看板旁边持续追问“这个项目现在卡在哪里”
  - 连续几轮讨论如何拆分需求、先做哪些子工单
  - 中途遇到 CLI 工具审批，人工处理后继续同一轮对话

这两类场景都**不经过编排引擎、不触发 Hook、不占用工单状态机**，但它们的生命周期不同：

- `harness_editor`、`ticket_detail`、`command_palette`、`cron_input` 等入口属于 **Ephemeral Chat**
  - 临时对话
  - 关闭即结束
  - transcript 默认不持久化
- `project_sidebar` 属于 **Project Conversation**
  - 项目级持续对话
  - transcript 持久化
  - 允许关闭 UI 后续接之前的对话
  - 允许在 turn 中等待 Codex 原生工具审批 / 用户输入，然后继续同一 turn

因此本章不再把所有内嵌 AI 入口视为同一种 ephemeral 交互，而是定义为一个统一的 Direct Chat 子系统，下分 Ephemeral Chat 与 Project Conversation 两种模式。

### 31.2 架构：复用 Agent Adapter，绕过编排引擎

Direct Chat 仍然复用现有 Agent Adapter，但它不是 orchestrator worker，也不是 ticket runtime。它由 `openase serve` 直接托管，服务浏览器实时交互。

```
┌─────────────────────────────────────────────────────────────┐
│ 前端                                                         │
│                                                             │
│  Harness AI / Ticket AI / Cron AI ───────┐                  │
│  Project Sidebar Ask AI ───────────────┐ │                  │
│                                         ▼ ▼                  │
│                        Chat API + SSE / watch stream         │
└──────────────────────────────┬───────────────────────────────┘
                               │
┌──────────────────────────────┼───────────────────────────────┐
│ openase serve                ▼                               │
│                                                             │
│   ┌──────────────────────────────┐                          │
│   │ Direct Chat Service          │                          │
│   │ - 上下文注入                  │                          │
│   │ - provider 选择               │                          │
│   │ - transcript 持久化           │                          │
│   │ - interrupt / approval 桥接   │                          │
│   └──────────────┬───────────────┘                          │
│                  │                                          │
│      ┌───────────┴───────────┐                              │
│      │ Live Runtime Registry │                              │
│      │ 仅管理浏览器对话会话    │                              │
│      └───────────┬───────────┘                              │
│                  │                                          │
│          local process / remote SSH                         │
│                  │                                          │
│        Claude / Codex / Gemini Adapter                      │
│                                                             │
│   不经过：orchestrator、Hook、ticket 状态机                    │
└─────────────────────────────────────────────────────────────┘
```

核心原则：

- `serve` 持有浏览器会话、SSE/watch stream 和 conversation runtime registry
- provider 对应的 CLI 进程仍然运行在 `provider.machine_id` 指向的机器上
- 本机 provider 用 `os/exec`，远端 provider 用 SSH / sidecar 会话；Direct Chat 复用与 orchestrator 相同的机器边界，而不是重新发明一套“浏览器端 CLI”
- Direct Chat 可以有轻量 runtime registry，但它不是 worker，不参与 ticket claim / retry / health reconciliation
- `project_sidebar` 引入持久 transcript 与 interrupt 恢复；其他入口保持轻量

当前各 CLI 的推荐运行方式：

- Claude Code：`claude -p --verbose --output-format stream-json`
- Codex：复用 `codex app-server` 会话，使用 `thread/start` + `turn/start` + notification
- Gemini CLI：使用 Gemini 的非交互流式输出模式

其中 Claude Code 新版本要求：当使用 `-p/--print` 且 `--output-format stream-json` 时，必须同时加 `--verbose`。

### 31.2.1 双模式聊天模型

| 维度 | Ephemeral Chat | Project Conversation |
|------|----------------|----------------------|
| 典型入口 | Harness AI、Ticket AI、Cron AI | `project_sidebar` |
| transcript | 默认不持久化 | 持久化 |
| 关闭 UI 后 | 会话结束 | UI 关闭但 conversation 保留 |
| 续接方式 | 同一 live session 内 `session_id` | 通过 `conversation_id` 恢复 transcript；若 live runtime 尚在则继续同一 thread，否则按恢复策略重建 |
| 审批 | 仅 `action_proposal` 的用户确认 | `action_proposal` + Codex 原生工具审批 / 用户输入 interrupt |
| API | 单次 turn SSE 足够 | 需要 conversation + turn + stream + interrupt 响应接口 |

### 31.2.2 多 CLI Provider

Direct Chat 不能只绑定单一 CLI。项目应允许用户在对话入口显式选择当前会话使用的 Agent CLI Provider，至少支持：

- Claude Code
- OpenAI Codex
- Gemini CLI

设计原则：

- Provider 选择是 **conversation 级** 配置，不影响项目默认工单 Agent Provider
- 同一个项目可以同时存在多个 Direct Chat provider
- 前端默认优先选择项目默认 provider；如果默认 provider 不支持 Direct Chat，则退回到第一个可用的 chat-capable provider
- 每个 provider 仍走各自原生 adapter，不强行伪装成同一协议
- 切换 provider 必须开启新 conversation，不能复用上一 provider 的 provider-native session / thread

后端契约要求：

- 创建 conversation / 发起 turn 时都允许显式传入 `provider_id`
- 若传入 `provider_id`，后端必须校验该 provider 属于当前项目所在 organization 且支持 Direct Chat
- 若未传入 `provider_id`，后端按“项目默认 provider -> 第一个可用 chat-capable provider”解析
- Provider 控制面响应必须显式暴露 `capabilities.ephemeral_chat`
  - `state`: `available` / `unavailable` / `unsupported`
  - `reason`: 当 provider 不能用于 Direct Chat 时给出结构化原因；对已支持但暂不可用的 provider，优先复用 availability reason
- 前端 selector 和后端默认/fallback 解析都必须基于这个显式 capability，而不是仅靠 runtime.Supports 或隐式 adapter 判断

### 31.2.3 CLI 应该跑在哪里

Direct Chat 的 CLI 运行位置必须满足“**由 `serve` 托管，但跑在 project 下面**”。

具体规则：

- 进程所有权在 `openase serve`
  - 因为浏览器连接、SSE/watch stream、interrupt 响应、conversation 恢复都发生在 API 侧
- 执行目标在 `provider.machine_id`
  - provider 绑定哪台机器，聊天 CLI 就在那台机器启动
  - 不允许把 remote machine 的 `local_path` 误当作 API 进程所在机器的本地目录
- `cwd` 不是 OpenASE 服务进程当前目录，也不是代码仓库自己的 repo root；它应是 **项目级 chat workspace**

定义 `ProjectConversationWorkspace`：

- 路径位于目标 machine 的 project 作用域下，例如：
  - `<machine.workspace_root>/<org>/<project>/.openase/chat/`
- 作用：
  - 作为 Codex / Claude / Gemini 的稳定 `cwd`
  - 放置 `.codex/` / `.agent/` 技能与包装脚本
  - 放置 repo link / manifest，供项目级对话跨 repo 读取上下文
  - 放置非权威的运行时临时文件与诊断信息

为什么不用某个 repo working copy 的 `repo_path` 直接当 `cwd`：

- 多 repo 项目会丢掉 project 级视角
- 单个 repo working copy 只覆盖一个仓库，不能假定有一个天然共同父目录表达整个项目
- `project_sidebar` 的语义是“项目对话”，不是“某个 repo 对话”

因此推荐做法是：

- Project Conversation 一律进入 `ProjectConversationWorkspace`
- workspace 中通过 symlink / manifest 暴露当前项目可见的 repo checkout
- 需要 repo 级精确编辑时，再由用户显式创建 ticket 或进入 Harness / ticket detail 场景

### 31.3 对话上下文注入

Direct Chat 的价值在于它知道项目上下文。后端在调用 Agent CLI 之前，自动将相关信息注入到 system / developer prompt 中。

注入规则：

- `harness_editor`
  - 当前 Harness 内容
  - 模板变量字典
  - Harness 编辑模式下的 diff 输出约束
- `project_sidebar`
  - 项目名称、描述、状态概览
  - 工单统计、最近活动
  - 可见 repo 列表与默认分支
  - 当前 provider、成本/预算说明
  - 当前 conversation 的 rolling summary
- `ticket_detail`
  - 工单完整信息
  - 最近活动
  - Hook 历史
  - repo scope / PR 链接
- `command_palette` / `cron_input`
  - 最小化的项目上下文
  - 当前表单字段值

共同约束：

- AI 不得声称已经执行了平台写操作，除非收到确认后的执行结果事件
- 平台写操作一律通过 `action_proposal`
- Project Conversation 恢复时，Turn 1 不需要把整段历史逐字回放；优先注入 rolling summary + recent entries

### 31.4 平台操作与审批

Direct Chat 存在两种完全不同的“审批/确认”，必须严格分开：

- **平台写操作确认**
  - AI 通过 `action_proposal` 提案
  - 用户确认后，平台 API 才真正执行
- **CLI 原生工具审批 / 用户输入**
  - 由 provider 在当前 turn 中途发起
  - 例如 Codex 的命令执行审批、文件修改审批、requestUserInput
  - 处理后继续同一 turn

这两种机制不能混为一谈。

### 31.4.1 平台写操作：沿用 `action_proposal`

当用户要求创建 / 修改工单、调整 Workflow、更新项目配置等平台写操作时，AI 必须输出结构化 `action_proposal`，等待用户确认。

```json
{
  "type": "action_proposal",
  "actions": [
    {"method": "POST", "path": "/api/v1/projects/xxx/tickets", "body": {"title": "实现用户注册 API", "parent_ticket_id": "ASE-42"}},
    {"method": "POST", "path": "/api/v1/projects/xxx/tickets", "body": {"title": "实现用户登录 API"}},
    {"method": "POST", "path": "/api/v1/projects/xxx/tickets", "body": {"title": "实现 RBAC 权限模型"}}
  ],
  "summary": "创建 3 个子工单"
}
```

执行流程：

1. AI 输出 `action_proposal`
2. 前端渲染确认 UI
3. 用户点击确认
4. 由后端执行 proposal 中的动作，并把执行结果回写为 conversation entries
5. 真正发生的平台写操作继续走标准 API、审计和 ActivityEvent

这里推荐由后端执行 proposal，而不是让前端逐个直接调业务 API。原因：

- 页面刷新不会丢失执行状态
- 执行结果能稳定写回 conversation transcript
- 审计边界更清晰

### 31.4.2 Codex 原生工具审批与用户输入

Project Conversation 需要沿用 **Codex 原生工具审批**，而不是在 OpenASE 再发明一套抽象后把 provider 语义抹平。

OpenASE 对 Codex 的要求：

- 必须支持并转发以下中断：
  - `item/commandExecution/requestApproval`
  - `item/fileChange/requestApproval`
  - `item/tool/requestUserInput`
- OpenASE 对这些中断只做**外层 envelope 标准化**，保留 provider-native 语义
  - `interrupt_id`
  - `provider`
  - `provider_request_id`
  - `kind`
  - `payload`
  - `options`
- UI 必须显示 Codex 原生决策选项
  - 例如 approve once / approve for session / deny
  - 不允许在产品层硬编码成只剩一个“批准”按钮

Project Conversation 中 turn 的中断流程：

1. `turn/start`
2. Codex 运行到需要审批 / 用户输入的位置
3. Codex 发来 request
4. OpenASE 将该 request 持久化为 `pending_interrupt`
5. SSE / watch stream 向前端发出 `interrupt_requested`
6. 前端展示审批 / 输入 UI
7. 用户提交 decision / answer
8. OpenASE 调用 provider-native response 接口
9. 同一 turn 继续，直到 `turn/completed` / `turn/failed`

注意：

- 这类 interrupt 是 conversation runtime 的暂停点，不是第 7.5 节里 ticket 状态审批的替身
- 平台写操作仍然必须走 `action_proposal`，不能借由 Codex 工具审批直接放行平台变更

### 31.5 前端触发入口

| 入口位置 | 模式 | 注入的上下文 | 典型问题 |
|---------|------|------------|---------|
| Harness 编辑器侧栏 | Ephemeral Chat | 当前 Harness 内容 + 变量字典 | “帮我优化工作边界的描述”、“加一个重试时的续接提示” |
| 看板页右侧抽屉 | Project Conversation | 项目信息 + 工单统计 + 最近活动 + conversation summary | “项目进展如何”、“哪些工单卡住了”、“继续我们刚才的拆分方案” |
| 工单详情页底部 | Ephemeral Chat | 工单完整信息 + 活动日志 + Hook 历史 | “为什么失败了”、“帮我拆成子工单”、“Agent 做了什么” |
| 全局命令面板 | Ephemeral Chat | 项目基础信息 | “帮我配一个每天 9 点的安全扫描 cron”、“怎么添加新 Repo” |
| 定时任务配置页 | Ephemeral Chat | 当前 cron 表达式 | “每周一三五上午 10 点” → AI 生成 `0 10 * * 1,3,5` |

所有入口统一带 provider selector，允许用户在 Claude Code / Codex / Gemini CLI 之间切换当前对话使用的 provider。

补充交互约束：

- `project_sidebar` 采用常规聊天体验，不再使用固定 turn cap 作为 transcript UX
- `project_sidebar` 中一个 assistant turn 只对应一个可变 transcript block；流式增量必须合并到当前 assistant 回复中，不能把 chunk 边界暴露成多个气泡
- 当 turn 因 Codex interrupt 暂停时，前端应在 transcript 中插入 interrupt card，而不是让用户误以为 turn 已完成
- provider 切换会创建新的 conversation，不复用上一 provider 的 provider-native thread

### 31.6 Harness 编辑器中的 AI 辅助细节

Harness 编辑器仍然是最高频场景，但它继续保持 Ephemeral Chat，不引入持久 transcript 与 tool interrupt 恢复。

```text
┌──────────────────────────────────────┬───────────────────────────────┐
│ Harness 编辑器                        │ 💬 AI 助手                    │
│                                      │                               │
│  ---                                 │ 帮我改进“工作边界”这一节，     │
│  status:                             │ 增加对测试文件的保护规则       │
│    pickup: "Todo"                    │                               │
│  ---                                 │ AI 返回结构化 diff             │
│                                      │ [应用到编辑器]  [继续修改]     │
└──────────────────────────────────────┴───────────────────────────────┘
```

要求：

- 用户请求修改 Harness 时，优先返回结构化 `diff`
- “应用到编辑器”点击后，patch 直接应用到编辑器内容，用户可以 undo
- 普通 Harness 建议不要输出 `action_proposal`

### 31.7 会话管理与持久化

#### 31.7.1 两类会话策略

| 特性 | Ephemeral Chat | Project Conversation |
|------|----------------|----------------------|
| 生命周期 | 用户关闭侧栏 / 切换页面后结束 | conversation 持久化；live runtime 可因空闲或预算被回收 |
| transcript | 默认不持久化 | 持久化 |
| 恢复入口 | 同一 live session 内续接 | 通过 `conversation_id` 恢复 |
| 并发 | 每个用户同时最多 1 个 ephemeral 会话 | 每个 `(user, project, provider, source=project_sidebar)` 最多 1 个 live runtime，可保留多个历史 conversation |
| 成本控制 | 默认 10 轮 / $2.00 等轻量预算 | transcript 与 runtime 解耦；预算命中时只关闭 live runtime，不删除 conversation |

#### 31.7.2 Project Conversation 持久化模型

Project Conversation 需要最小持久化模型：

| 实体 | 关键字段 | 说明 |
|------|---------|------|
| `chat_conversations` | `id`, `project_id`, `user_id`, `source`, `provider_id`, `status`, `provider_thread_id`, `last_turn_id`, `rolling_summary`, `last_activity_at` | conversation 级稳定主键；前端只认 `conversation_id` |
| `chat_turns` | `id`, `conversation_id`, `turn_index`, `provider_turn_id`, `status`, `started_at`, `completed_at` | 一个用户消息对应一个 turn |
| `chat_entries` | `id`, `conversation_id`, `turn_id`, `seq`, `kind`, `payload_json` | 追加式 transcript，覆盖 text / diff / action_proposal / interrupt / result |
| `chat_pending_interrupts` | `id`, `conversation_id`, `turn_id`, `provider_request_id`, `kind`, `payload_json`, `status`, `resolved_at` | 持久化 Codex 原生审批 / 用户输入中断 |

字段语义：

- `conversation_id` 是 OpenASE 稳定 ID，对前端公开
- `provider_thread_id` 是 provider-native 恢复锚点，只在服务端保存
- `last_turn_id` 只用于 provider 级诊断与恢复，不作为前端主键
- 前端 localStorage 只允许缓存“最近一次打开的 `conversation_id`”，不能作为权威存储

#### 31.7.3 恢复策略

恢复顺序：

1. 若 live runtime 仍在
   - 直接在同一 provider thread 上继续
2. 若 live runtime 不在，但 provider 支持 durable resume
   - 使用 `provider_thread_id` 做 provider-native resume
3. 若 provider-native resume 不可用
   - 创建新 thread
   - 注入 `rolling_summary + recent entries + 当前项目上下文`
   - 从 OpenASE 视角恢复 conversation，而不是丢弃 transcript

Project Conversation 因此是“**conversation 持久化，live runtime 可回收**”的模型，而不是“session_id 永久等于一个活进程”。

### 31.8 API 端点

#### 31.8.1 Ephemeral Chat API

Ephemeral Chat 保持单次 turn 的轻量 API：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/chat` | 发起一轮 Ephemeral Chat，SSE 流式返回 |
| DELETE | `/api/v1/chat/:sessionId` | 关闭当前 ephemeral live session |

请求体：

```json
{
  "message": "帮我优化工作边界的描述",
  "source": "harness_editor",
  "provider_id": "uuid",
  "context": {
    "project_id": "uuid",
    "workflow_id": "uuid",
    "ticket_id": "uuid"
  },
  "session_id": null
}
```

#### 31.8.2 Project Conversation API

`project_sidebar` 必须使用 conversation-based API，而不是继续复用“一次 POST 挂一条 SSE 到 done”的模式。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/chat/conversations` | 创建一个新的 project conversation |
| GET | `/api/v1/chat/conversations` | 查询当前用户在某 project/provider/source 下的 conversation 列表或当前活跃 conversation |
| GET | `/api/v1/chat/conversations/:conversationId` | 获取 conversation 元数据 |
| GET | `/api/v1/chat/conversations/:conversationId/entries` | 拉取历史 transcript |
| POST | `/api/v1/chat/conversations/:conversationId/turns` | 发送一轮用户消息 |
| GET | `/api/v1/chat/conversations/:conversationId/stream` | watch 当前 conversation 的实时事件 |
| POST | `/api/v1/chat/conversations/:conversationId/interrupts/:interruptId/respond` | 对 Codex interrupt 提交 decision / answer |
| POST | `/api/v1/chat/conversations/:conversationId/action-proposals/:entryId/execute` | 确认并执行 action proposal |
| DELETE | `/api/v1/chat/conversations/:conversationId/runtime` | 关闭 live runtime，但保留 conversation 与 transcript |

创建 conversation 请求示例：

```json
{
  "source": "project_sidebar",
  "provider_id": "uuid",
  "context": {
    "project_id": "uuid"
  }
}
```

发送 turn 请求示例：

```json
{
  "message": "继续我们刚才关于 ASE-42 的拆分讨论"
}
```

interrupt 响应示例：

```json
{
  "decision": "approve_once",
  "answer": null
}
```

其中：

- `decision` 取值必须以 provider-native options 为准；OpenASE 只做外层映射
- 对 `requestUserInput`，`answer` 允许是结构化 JSON，而不局限于单行文本

#### 31.8.3 Stream 事件

Project Conversation 的 stream 至少支持以下事件：

```text
event: session
data: {"conversation_id":"conv-xxx","runtime_state":"ready"}

event: message
data: {"type":"text","content":"好的，我们继续。"}

event: message
data: {"type":"diff","file":"harness content","hunks":[...]}

event: message
data: {"type":"action_proposal","summary":"创建 3 个子工单","actions":[...]}

event: interrupt_requested
data: {
  "interrupt_id":"intr-xxx",
  "provider":"codex",
  "kind":"command_execution_approval",
  "options":[{"id":"approve_once","label":"Approve Once"},{"id":"approve_for_session","label":"Approve this Session"},{"id":"deny","label":"Deny"}],
  "payload": {...}
}

event: interrupt_resolved
data: {"interrupt_id":"intr-xxx","decision":"approve_once"}

event: turn_done
data: {"conversation_id":"conv-xxx","turn_id":"turn-xxx","cost_usd":0.03}

event: error
data: {"message":"codex chat turn failed"}
```

约束：

- `project_sidebar` 一个 assistant turn 只对应一个 assistant transcript block
- provider delta 只能 merge 到当前 block，不能把 chunk 边界暴露成多个气泡
- `interrupt_requested` 不是 `turn_done`

### 31.9 与工单 Agent 的区别

| 维度 | 工单 Agent（编排引擎驱动） | Direct Chat（用户直接驱动） |
|------|--------------------------|-----------------------------|
| 触发方式 | 编排引擎 Tick 自动分发 | 用户在 UI 中主动发起 |
| 生命周期 | 工单从 pickup 到 finish | Ephemeral Chat 用完即走；Project Conversation transcript 持久化 |
| 上下文来源 | Harness Prompt + 工单描述 | 后端注入的页面 / 项目 / conversation 上下文 |
| 状态管理 | 工单状态流转、attempt_count | 不驱动 ticket 状态机 |
| Hook | on_claim / on_complete / on_done | 无 |
| 成本追踪 | 记入工单 cost_amount | 记入项目级 direct chat / conversation cost |
| 平台操作 | Agent Token 直接执行 | `action_proposal` 经用户确认后执行 |
| 工具审批 | 编排运行默认无人值守 | Project Conversation 支持 Codex 原生 interrupt |
| 持久化 | `AgentTraceEvent + AgentStepEvent + ActivityEvent` 分层持久化 | Ephemeral Chat 默认不持久化；Project Conversation 持久化 transcript |
| 并发 | 支持多个 Agent 定义并行执行多工单；同一个 Agent 定义在并发限制允许时也可同时驱动多个 AgentRun | 以用户/project/provider 维度限制 live runtime 并发 |

---

## 第三十二章 Dispatcher：用 Workflow 实现自动分配

### 32.1 设计哲学：分配就是一种工作

之前 PRD 中的自动分配是 Project 配置上的两个布尔开关（`auto_assign_workflow`、`auto_assign_agent`），逻辑硬编码在编排引擎内部。这违背了 OpenASE 的核心原则——**一切工作都是工单，一切行为都由 Workflow 定义。**

正确的做法：**分配本身就是一个 Workflow。** 一个"调度员"角色的 Agent 领取 Backlog 里的工单，读需求、判断该给谁干、通过 Platform API 分配给合适的角色。不需要新增任何机制——完全复用现有的角色体系 + 自治闭环 + Platform API。

### 32.2 Dispatcher Workflow

````markdown
---
status:
  pickup: "Backlog"          # Dispatcher 监听 Backlog 列
  finish: "Backlog"          # 分配完成后工单留在 Backlog（已被改了 status，实际上已不在 Backlog）
agent:
  max_turns: 5               # 分配不需要很多轮
  timeout_minutes: 5
  max_budget_usd: 0.50       # 分配很便宜
platform_access:
  allowed:
    - "tickets.update.self"  # 修改工单状态（拖到其他列）
    - "tickets.create"       # 可以拆分子工单
    - "tickets.list"         # 查看其他工单了解上下文
    - "tickets.link"         # 关联相关工单
    - "machines.list"        # 查看机器资源
---

# Dispatcher — 工单调度员

你是项目的工单调度员。你的唯一职责是：**评估 Backlog 中的工单，判断它应该由哪个角色处理，然后分配到正确的状态列。**

## 当前工单

- 标识: {{ ticket.identifier }}
- 标题: {{ ticket.title }}
- 描述: {{ ticket.description | default("无描述") }}
- 优先级: {{ ticket.priority }}
- 类型: {{ ticket.type }}

{% if ticket.links | length > 0 %}
## 关联信息
{% for link in ticket.links %}
- [{{ link.type }}] {{ link.title }}: {{ link.url }}
{% endfor %}
{% endif %}

## 可用的角色和对应的接收列

以下是项目中已配置的角色（Workflow）和它们的 pickup 状态：

{% for wf in project.workflows %}
- **{{ wf.role_name }}** ({{ wf.name }}): pickup 状态 = "{{ wf.pickup_status }}"
  {{ wf.role_description }}
{% endfor %}

## 可用的机器

{% for m in project.machines %}
- **{{ m.name }}** ({{ m.host }}): {{ m.description }}
  标签: {{ m.labels | join(", ") }}
{% endfor %}

## 分配流程

1. **理解需求**：仔细阅读工单标题、描述、关联信息
2. **选择角色**：根据需求类型匹配最合适的角色
   - 功能开发/Bug修复 → fullstack-developer（pickup: "待开发"）
   - 测试相关 → qa-engineer（pickup: "待测试"）
   - 文档更新 → technical-writer（pickup: "待撰写"）
   - 安全问题 → security-engineer（pickup: "待扫描"）
   - 部署相关 → devops-engineer（pickup: "待部署"）
   - 需求不清晰 → 不分配，添加评论说明缺少什么信息
3. **判断是否需要拆分**：如果工单太大（涉及多个模块/角色），拆成子工单
4. **确认目标 Workflow 已绑定正确的 Agent**：例如 GPU 训练类 Workflow 应绑定到某个 GPU 机器上的 Provider，而不是在分配时再手选机器
5. **执行分配**：调用 Platform API 将工单状态改为目标角色的 pickup 状态

## 分配操作

```bash
# 分配给全栈开发者
openase ticket update ASE-{{ ticket.identifier }} --status "待开发"

# 分配给测试工程师
openase ticket update ASE-{{ ticket.identifier }} --status "待测试"

# 拆分子工单
openase ticket create --title "子任务1" --parent {{ ticket.identifier }} --status "待开发"
openase ticket create --title "子任务2" --parent {{ ticket.identifier }} --status "待测试"
```

## 分配原则

- 需求不清晰时**不要强行分配**，在工单上添加评论说明需要补充什么
- 一个工单只分配给一个角色。如果涉及多个角色，拆成子工单
- 优先级为 urgent 的工单优先分配
- 如果所有合适的角色都已满载，在工单上注释说明等待原因
````

### 32.3 工作流程图

```
用户创建工单 → Backlog 列
                  │
                  ▼
         Dispatcher Agent 领取
         （pickup: "Backlog"）
                  │
                  ├── 评估需求
                  │
                  ├── 需求清晰 ──→ 调用 Platform API 改状态
                  │                     │
                  │               ┌─────┼──────┬──────────┐
                  │               ▼     ▼      ▼          ▼
                  │           "待开发" "待测试" "待部署"  "待调研"
                  │               │     │      │          │
                  │               ▼     ▼      ▼          ▼
                  │           coding  test   deploy   research
                  │           Agent   Agent  Agent    Agent
                  │           领取    领取    领取     领取
                  │
                  ├── 需求太大 ──→ 拆分子工单，每个分到对应列
                  │
                  └── 需求不清晰 ──→ 添加评论，工单留在 Backlog
```

### 32.4 新增模板变量

Dispatcher 需要知道项目中有哪些 Workflow 和机器，这要求新增模板变量：

| 变量 | 类型 | 说明 |
|------|------|------|
| `project.workflows` | list | 项目中所有已激活的 Workflow 列表 |
| `project.workflows[].name` | string | Workflow 名称 |
| `project.workflows[].type` | string | Workflow 类型 |
| `project.workflows[].role_name` | string | 角色名称 |
| `project.workflows[].role_description` | string | 角色描述 |
| `project.workflows[].pickup_status` | string | Pickup 状态名 |
| `project.workflows[].pickup_statuses` | list | Pickup 状态的结构化列表；每项至少包含 `id / name / stage / color`，避免多状态绑定时丢信息 |
| `project.workflows[].finish_status` | string | Finish 状态名（便于人类快速阅读） |
| `project.workflows[].finish_statuses` | list | Finish 状态的结构化列表；每项至少包含 `id / name / stage / color` |
| `project.workflows[].max_concurrent` | int | 最大并发数 |
| `project.workflows[].current_active` | int | 当前正在执行的工单数；定义为 `tickets.current_run_id = agent_runs.id AND agent_runs.workflow_id = workflow.id` 的计数，不从 ActivityEvent 推导 |
| `project.machines` | list | 项目可用的机器列表 |
| `project.machines[].name` | string | 机器名 |
| `project.machines[].host` | string | 地址 |
| `project.machines[].description` | string | 描述 |
| `project.machines[].labels` | list | 标签 |
| `project.machines[].status` | string | 机器状态 |
| `project.machines[].resources` | object | 最近一次持久化的资源快照（CPU/内存/GPU/agent 环境等）；若机器尚未完成探测则为空对象 `{}` |
| `project.statuses` | list | 项目所有 Custom Status 列表 |
| `project.statuses[].id` | string | 状态 UUID |
| `project.statuses[].name` | string | 状态名 |
| `project.statuses[].stage` | string | 状态对应的 Stage |
| `project.statuses[].color` | string | 颜色 |

### 32.5 去掉 Project 上的硬编码开关

之前 Project 表上的 `auto_assign_workflow` 和 `auto_assign_agent` 两个布尔字段可以删除了。自动分配的开启/关闭就是：**有没有配一个 pickup 为 "Backlog" 的 Dispatcher Workflow。** 有就自动分配，没有就不分配——完全由用户自己决定，不需要专门的开关。

### 32.6 Dispatcher 的优势

和硬编码的自动分配逻辑相比，用 Workflow 实现 Dispatcher 有几个明显优势：

**可定制**：不同项目可以有完全不同的分配策略。科研项目的 Dispatcher 可能侧重实验资源分配，产品项目的 Dispatcher 可能侧重优先级排序。只需要修改 Harness Prompt，不需要改代码。

**可审计**：Dispatcher Agent 的所有分配操作都记录在 ActivityEvent 里（`created_by: agent:dispatcher-01`），可以回溯"为什么 ASE-42 被分给了 security-engineer 而不是 fullstack-developer"。

**可迭代**：Dispatcher 的 Harness 像其他角色一样由平台版本化管理，可以查看历史、比较版本、回滚。refine-harness 元工作流可以分析 Dispatcher 的分配历史，优化分配策略。

**闭环一致**：不引入新机制，完全复用现有的角色 + Workflow + Platform API + Hook 体系。Dispatcher 就是另一个角色——恰好它的"工作产出"不是代码，而是工单状态变更。

### 32.7 在角色库中的位置

Dispatcher 作为内置角色加入角色库：

| 角色 | Harness | pickup | 特殊之处 |
|------|---------|--------|---------|
| 调度员 | `roles/dispatcher.md` | Backlog | 唯一一个"产出是状态变更而非代码"的角色 |

HR Advisor 推荐引擎的规则也需要更新：

```go
// 规则: Backlog 中积压超过 10 个工单且没有 Dispatcher → 推荐
if stats.BacklogCount > 10 && !stats.HasDispatcherWorkflow {
    recs = append(recs, RoleRecommendation{
        Role:   "dispatcher",
        Reason: fmt.Sprintf("Backlog 中有 %d 个待分配工单。建议招募调度员自动分配。", stats.BacklogCount),
    })
}
```

---

## 第三十三章 统一通知系统

### 33.1 设计：事件订阅 + 渠道适配器

通知系统由两部分组成：**订阅规则**（什么事件触发通知）和**渠道适配器**（通知发到哪里）。用户可以自由组合"事件 × 渠道"。

```
事件源                    订阅规则匹配                渠道适配器
                              │
ticket.status_changed ──→ "待开发"列有新工单？──→ Telegram
agent.completed ──────→ Agent 完成了？────────→ 企业微信
hook.failed ──────────→ Hook 失败了？─────────→ Slack + Email
machine.offline ──────→ 机器掉线了？─────────→ 企业微信
```

### 33.2 NotificationChannel 实体

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| organization_id | FK | 所属组织 |
| name | String | 渠道名称（如"运维 Telegram 群"、"研发企业微信群"） |
| type | String | 渠道类型标识（见适配器列表） |
| config | JSONB (encrypted) | 渠道配置（token、webhook_url 等，加密存储） |
| is_enabled | Boolean | 是否启用 |

### 33.3 渠道适配器

渠道适配器实现同一个 Go interface：

```go
// domain/notification/channel.go
type ChannelAdapter interface {
    Type() string
    Send(ctx context.Context, cfg ChannelConfig, msg Message) error
    Validate(ctx context.Context, cfg ChannelConfig) error  // 测试连接
}

type Message struct {
    Title    string            // 通知标题
    Body     string            // 通知正文（Markdown）
    Level    string            // info / warning / error
    Link     string            // 关联的 OpenASE 页面 URL
    Metadata map[string]string // 额外字段（渠道特定的渲染可能用到）
}
```

**内置适配器：**

| 渠道 | type 标识 | 配置字段 | 实现方式 |
|------|----------|---------|---------|
| Slack | `slack` | `webhook_url` | Incoming Webhook POST |
| Telegram | `telegram` | `bot_token`, `chat_id` | Bot API `sendMessage` |
| 企业微信 | `wecom` | `webhook_key` | 群机器人 Webhook |
| 飞书 | `feishu` | `webhook_url`, `secret` | 自定义机器人 Webhook |
| Discord | `discord` | `webhook_url` | Discord Webhook |
| Email | `email` | `smtp_host`, `smtp_port`, `from`, `to[]`, `username`, `password` | SMTP |
| 通用 Webhook | `webhook` | `url`, `secret`, `headers` | POST JSON + HMAC 签名 |

**通用 Webhook 是万能出口**。任何能接收 HTTP POST 的系统都能作为通知渠道。

**企业微信配置示例：**

```yaml
# 通过 Web UI 或 CLI 配置
channels:
  - name: "研发群"
    type: wecom
    config:
      webhook_key: "${WECOM_WEBHOOK_KEY}"  # 群机器人 key
```

**Telegram 配置示例：**

```yaml
channels:
  - name: "Gary 的通知 Bot"
    type: telegram
    config:
      bot_token: "${TELEGRAM_BOT_TOKEN}"
      chat_id: "123456789"
```

### 33.4 NotificationRule 实体（订阅规则）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属项目（也可为空，表示组织级规则） |
| name | String | 规则名称（如"待开发列有新工单时通知 Telegram"） |
| event_type | String | 监听的事件类型（见事件列表） |
| filter | JSONB | 过滤条件（可选，精确匹配） |
| channel_id | FK | 通知到哪个渠道 |
| template | Text | 消息模板（Jinja2，可选，有默认模板） |
| is_enabled | Boolean | 是否启用 |

**filter 示例：**

filter 的 key 必须来自事件 payload / metadata 的结构化字段名，不能依赖 `message` 文本做模糊匹配。

```json
// 只在工单进入"待开发"列时通知
{"event_type": "ticket.status_changed", "filter": {"to_status_name": "待开发"}}

// 只在 urgent 优先级的工单被创建时通知
{"event_type": "ticket.created", "filter": {"priority": "urgent"}}

// 只在特定 Workflow 的 Hook 失败时通知
{"event_type": "hook.failed", "filter": {"workflow_type": "coding"}}

// 机器离线时通知
{"event_type": "machine.offline", "filter": {"machine_name": "gpu-01"}}
```

### 33.5 可订阅的事件列表

NotificationRule 的 `event_type` 分两类：

1. **Activity-derived 事件**：直接复用 `ActivityEvent.event_type` 的 canonical 值  
2. **Notification-only alert 事件**：只用于告警/通知，不要求必须写入 `ActivityEvent`

**规则：**

- 只要一个通知事件与 `ActivityEvent` 语义相同，就必须复用同一个 canonical 名称，禁止再引入通知专属别名
- 因此这里不再使用 `ticket.assigned`、`ticket.retry`、`agent.error` 这类与活动流语义重叠但名字不同的别名

| 事件类型 | 类别 | 触发时机 | 默认消息模板 |
|---------|------|---------|------------|
| `ticket.created` | Activity-derived | 工单创建 | "📋 新工单 {{ ticket.identifier }}: {{ ticket.title }}" |
| `ticket.status_changed` | Activity-derived | 工单状态变更 | "🔄 {{ ticket.identifier }} 状态变为 {{ to_status_name }}" |
| `ticket.completed` | Activity-derived | 工单完成（到达 finish 状态） | "✅ {{ ticket.identifier }} 已完成" |
| `ticket.cancelled` | Activity-derived | 工单取消 | "🚫 {{ ticket.identifier }} 已取消" |
| `ticket.retry_scheduled` | Activity-derived | 平台已安排下一次退避重试 | "🔄 {{ ticket.identifier }} 将在 {{ next_retry_at }} 重试（第 {{ attempt_count }} 次）" |
| `ticket.retry_paused` | Activity-derived | 工单重试被暂停，等待人工处理 | "⏸️ {{ ticket.identifier }} 的重试已暂停：{{ pause_reason }}" |
| `ticket.budget_exhausted` | Activity-derived | 预算耗尽，重试暂停 | "💰 {{ ticket.identifier }} 预算耗尽（${{ cost_amount }}/${{ budget_usd }}），已暂停重试" |
| `agent.claimed` | Activity-derived | Agent 领取工单 | "🤖 {{ agent.name }} 领取了 {{ ticket.identifier }}" |
| `agent.failed` | Activity-derived | Agent 执行失败 | "🚨 Agent {{ agent.name }} 执行 {{ ticket.identifier }} 失败：{{ error }}" |
| `hook.failed` | Activity-derived | Hook 执行失败 | "🔧 {{ ticket.identifier }} 的 {{ hook_name }} 失败: {{ error }}" |
| `hook.passed` | Activity-derived | Hook 执行通过 | "✅ {{ ticket.identifier }} 的 {{ hook_name }} 通过" |
| `pr.linked` | Activity-derived | RepoScope 记录了 PR 链接 | "📝 {{ ticket.identifier }} 关联了 PR：{{ pull_request_url }}" |
| `ticket.stalled` | Notification-only | Agent Stall 或长时间无心跳 | "⚠️ {{ ticket.identifier }} 的 Agent 无响应" |
| `ticket.error_rate_high` | Notification-only | 连续 3+ 次失败 | "🔴 {{ ticket.identifier }} 连续失败 {{ consecutive_errors }} 次，退避 {{ backoff }} 后重试" |
| `machine.offline` | Notification-only | 机器离线 | "🔴 机器 {{ machine.name }} 离线" |
| `machine.online` | Notification-only | 机器上线 | "🟢 机器 {{ machine.name }} 恢复上线" |
| `machine.degraded` | Notification-only | 机器资源告警 | "⚠️ {{ machine.name }} 磁盘剩余 {{ disk_free_gb }}GB" |
| `budget.threshold` | 成本达到阈值 | "💰 项目 {{ project.name }} 已消耗 ${{ cost_usd }}" |

### 33.6 消息模板

每个订阅规则可以自定义消息模板（Jinja2 语法），也可以使用默认模板。模板变量跟事件类型相关：

```jinja
{# 自定义模板示例：工单完成时发送详细摘要 #}
🎉 **{{ ticket.identifier }}** 已完成

**标题**: {{ ticket.title }}
**角色**: {{ workflow.role_name }}
**Agent**: {{ agent.name }}
**耗时**: {{ duration_minutes }} 分钟
**成本**: ${{ cost_usd }}

{% if pr_urls | length > 0 %}
**PR**:
{% for url in pr_urls %}
- {{ url }}
{% endfor %}
{% endif %}

[查看详情]({{ ticket.url }})
```

### 33.7 通知引擎

通知引擎监听 EventProvider 的事件流，匹配订阅规则，调用渠道适配器发送：

```go
// infra/notification/engine.go
func (e *NotificationEngine) Run(ctx context.Context) {
    events, _ := e.eventBus.Subscribe(ctx, "ticket.events", "agent.events",
        "hook.events", "machine.events")

    for event := range events {
        // 查找匹配的订阅规则
        rules, _ := e.ruleRepo.FindMatching(ctx, event.Type, event.ProjectID)

        for _, rule := range rules {
            // 检查 filter 条件
            if !rule.MatchesFilter(event) {
                continue
            }

            // 渲染消息模板
            msg := e.renderMessage(rule, event)

            // 通过渠道适配器发送
            channel, _ := e.channelRepo.Get(ctx, rule.ChannelID)
            adapter := e.adapterRegistry.Get(channel.Type)

            if err := adapter.Send(ctx, channel.Config, msg); err != nil {
                e.logger.Warn("notification send failed",
                    "channel", channel.Name, "event", event.Type, "err", err)
                // 不重试——通知是尽力而为，不阻塞主流程
            }

            e.metrics.Counter("openase.notification.sent_total",
                provider.Tags{"channel_type": channel.Type, "event_type": event.Type}).Inc()
        }
    }
}
```

**关键设计决策：通知是"尽力而为"（fire-and-forget），不阻塞任何主流程。** 发送失败只记日志和指标，不重试、不影响工单状态。通知是辅助功能，不是关键路径。

### 33.8 Workflow Hook 中的通知

除了通过订阅规则配置通知，用户也可以在 Workflow 的 Hook 中直接调用通知——两种方式互补：

```yaml
# Harness 中通过 Hook 发通知（更灵活，可以执行任意逻辑）
hooks:
  on_done:
    - cmd: 'curl -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" -d "chat_id=${TELEGRAM_CHAT_ID}&text=工单 ${OPENASE_TICKET_IDENTIFIER} 完成了"'
      on_failure: ignore  # 通知失败不阻塞
```

两种方式的区别：

| 维度 | 订阅规则（NotificationRule） | Hook 中直接调用 |
|------|---------------------------|---------------|
| 配置方式 | Web UI / API | Harness YAML 中 |
| 灵活度 | 标准事件 + 过滤条件 | 完全自定义（任意 shell 命令） |
| 版本控制 | 存数据库 | 随 Harness 在 Git 中 |
| 适合场景 | 通用通知（"所有工单完成时通知"） | 特定 Workflow 的通知（"安全扫描完成时通知安全组"） |

### 33.9 Web UI

设置 → 通知页面：

```
┌────────────────────────────────────────────────────────────────────┐
│ 通知设置                                                            │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│ 📡 通知渠道                                              [添加渠道] │
│                                                                    │
│ ┌───────────────────────────────────────────────────────────────┐  │
│ │ 🔵 Telegram: Gary 的通知 Bot                                  │  │
│ │    chat_id: 123456789 · 已启用                  [测试] [编辑]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ 🟢 企业微信: 研发群                                            │  │
│ │    webhook · 已启用                             [测试] [编辑]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ 🔴 Slack: #openase-alerts                                    │  │
│ │    webhook · 已禁用                             [测试] [编辑]  │  │
│ └───────────────────────────────────────────────────────────────┘  │
│                                                                    │
│ 📋 订阅规则                                              [添加规则] │
│                                                                    │
│ ┌───────────────────────────────────────────────────────────────┐  │
│ │ "待开发"列有新工单 → Telegram                                   │  │
│ │   事件: ticket.status_changed                                  │  │
│ │   过滤: new_status = "待开发"                                   │  │
│ │   渠道: Gary 的通知 Bot                          [编辑] [删除]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ 工单失败 → 企业微信                                             │  │
│ │   事件: ticket.error                                            │  │
│ │   过滤: 无（所有失败）                                          │  │
│ │   渠道: 研发群                                   [编辑] [删除]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ GPU 机器离线 → Telegram + 企业微信                              │  │
│ │   事件: machine.offline                                        │  │
│ │   过滤: machine_name contains "gpu"                            │  │
│ │   渠道: Gary 的通知 Bot, 研发群                  [编辑] [删除]  │  │
│ └───────────────────────────────────────────────────────────────┘  │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### 33.10 API 端点

**渠道管理：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/channels` | 列出通知渠道 |
| POST | `/api/v1/orgs/:orgId/channels` | 创建渠道 |
| PATCH | `/api/v1/channels/:channelId` | 更新渠道 |
| DELETE | `/api/v1/channels/:channelId` | 删除渠道 |
| POST | `/api/v1/channels/:channelId/test` | 发送测试消息 |

**订阅规则管理：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:projectId/notification-rules` | 列出订阅规则 |
| POST | `/api/v1/projects/:projectId/notification-rules` | 创建规则 |
| PATCH | `/api/v1/notification-rules/:ruleId` | 更新规则 |
| DELETE | `/api/v1/notification-rules/:ruleId` | 删除规则 |

### 33.11 对 Provider 层的影响

之前 PRD 中的 `NotifyProvider` 接口过于简化（只有一个 `Send` 方法）。现在升级为完整的通知系统后，`NotifyProvider` 变为通知引擎的内部实现细节——对外暴露的是 `NotificationEngine`，它自己内部管理渠道适配器注册表和订阅规则匹配。

Provider 矩阵更新：

| Provider | 之前 | 之后 |
|----------|------|------|
| NotifyProvider | LogNotifier / SlackNotifier / WebhookNotifier | 移除，替换为 NotificationEngine + ChannelAdapter 注册表 |

---

## 第三十四章 Skills 生命周期管理

### 34.1 Skill 是什么

Skill 是 Agent CLI（Claude Code、Codex 等）原生支持的能力扩展单元。在 OpenASE 中，Skill 的正确建模不是“单个 `SKILL.md` 文本”，而是一个**具名目录型 bundle**：

- 必须存在入口文件 `SKILL.md`
- 可以包含 `scripts/`、`references/`、`assets/`、`agents/openai.yaml` 等附属文件
- bundle 内文件之间存在稳定的相对路径关系
- 运行时必须按目录原样 materialize 到 Agent CLI 的 skills 目录

典型结构如下：

```text
skill-name/
├── SKILL.md
├── agents/
│   └── openai.yaml
├── scripts/
│   └── *.sh / *.py / ...
├── references/
│   └── *.md / *.json / ...
└── assets/
    └── 任意被 skill 使用的模板、图片、样例文件
```

例如：

- `commit` skill：教 Agent 如何写 conventional commit message
- `land` skill：教 Agent 如何安全地合并 PR（rebase、squash、CI 检查）
- `openase-platform` skill：教 Agent 如何调用 OpenASE Platform API（创建工单、注册 Repo 等）
- `review-code` skill：教 Agent 如何做代码审查（检查风格、性能、安全）
- `deploy-openase` skill：除了 `SKILL.md` 外，还可能附带 `scripts/redeploy_local.sh`

Skill 对 Agent 来说是“你会什么”；Harness 是“你的角色是什么”。**Workflow 绑定 Skills = 这个角色在本次运行中拥有哪些能力包。**

### 34.2 最新设计与废弃说明

OpenASE 现采用 **DB 权威源 + runtime materialize** 模型：

- Skill bundle 的元数据、文件清单、版本、启用状态、绑定关系都存储在平台控制面
- Agent / 用户若要修改 Skill，必须调用 Platform API
- 新 runtime 创建时，平台把当前版本的 Skill bundle materialize 到工作区中的 Agent CLI skills 目录
- 已运行中的 runtime 不会因为后台 Skill 更新而自动漂移；只有新 runtime 默认拿最新版本，或显式 refresh / restart

> **废弃设计**：历史设计中，Skills 存储在项目仓库 `.openase/skills/`，并通过 `skills refresh` / `skills harvest` 在 repo 与工作区之间双向同步。该设计现已废弃。repo branch、工作树状态不再决定 Skill 的权威内容。

### 34.3 存储模型

Skill 由平台控制面持久化，版本单位是“整个目录 bundle”，不是单个 markdown 文本。

#### 34.3.1 领域对象

- `Skill`
  - 项目内稳定身份与生命周期对象
- `SkillVersion`
  - 某次发布的 bundle 快照
- `SkillFile`
  - 某个版本中的单个文件条目
- `WorkflowSkillBinding`
  - Workflow 与 Skill 的绑定关系
- `RuntimeSkillSnapshot`
  - 某次 AgentRun 实际消费的 SkillVersion 集合

#### 34.3.2 数据库模型

**Skill**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| project_id | FK | 所属项目 |
| name | String | Skill 名称，项目内唯一 |
| description | String | 展示描述 |
| is_builtin | Boolean | 是否为内置 Skill |
| is_enabled | Boolean | 是否启用 |
| created_by | String | 创建者 |
| archived_at | DateTime | 软删除 / 归档时间 |
| created_at | DateTime | 创建时间 |
| updated_at | DateTime | 最近更新时间 |

**SkillVersion**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| skill_id | FK | 所属 Skill |
| version | Integer | 单调递增版本号 |
| bundle_hash | String | 对整个 bundle 的规范化哈希（而非单文件哈希） |
| manifest_json | JSONB | 规范化文件清单、入口文件信息、bundle 元数据 |
| size_bytes | BigInt | 整个 bundle 的总大小 |
| file_count | Integer | bundle 中文件数 |
| created_by | String | 本次版本提交者 |
| created_at | DateTime | 版本创建时间 |

**SkillVersionFile**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| skill_version_id | FK | 所属 SkillVersion |
| path | String | bundle 内相对路径；大小写保留，比较时走规范化 |
| file_kind | Enum | `entrypoint` / `metadata` / `script` / `reference` / `asset` |
| media_type | String | MIME / 文本类型提示 |
| encoding | Enum | `utf8` / `base64` / `binary` |
| is_executable | Boolean | 是否应在 materialize 后设置执行位 |
| size_bytes | BigInt | 文件大小 |
| sha256 | String | 单文件内容哈希 |
| content_blob_id | FK | 指向内容 blob |
| created_at | DateTime | 创建时间 |

**SkillBlob**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| sha256 | String | 内容哈希，唯一 |
| size_bytes | BigInt | 原始大小 |
| compression | Enum | `none` / `gzip` |
| content_bytes | BYTEA | 压缩后或原始字节 |
| created_at | DateTime | 创建时间 |

**WorkflowSkillBinding**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| workflow_id | FK | 绑定的 Workflow |
| skill_id | FK | 绑定的 Skill |
| required_version_id | FK (nullable) | 可选固定版本；为空表示默认跟随当前发布版本 |
| rollout_mode | Enum | `follow_current` / `pin_version` |
| created_at | DateTime | 绑定时间 |

#### 34.3.3 为什么是这套表

- `skills` 存稳定身份与生命周期状态
- `skill_versions` 存 bundle 级版本快照
- `skill_version_files` 存目录内路径结构和文件属性
- `skill_blobs` 负责安全、去重地保存真实字节
- `workflow_skill_bindings` 只表达绑定关系，不复制 bundle 内容

这保证：

- 版本单位正确，是整个 skill 目录
- 数据库存储保留路径关系与执行位
- 不需要把二进制文件硬塞进 `SKILL.md`
- materialize 时可以稳定恢复原始目录结构
- agent run 只需记录 `skill_version_ids` 就能完整回放本次能力包

### 34.4 与 Workflow 的绑定

Skill 与 Workflow 的绑定关系由平台控制面单独持久化，不再把 `skills:` 作为 Harness YAML 中的权威源。

关键规则：

- Workflow 配置页编辑绑定关系时，调用平台绑定 API
- Harness 正文可以渲染出“当前绑定的 Skills 列表”供 Agent 阅读，但那是只读投影，不是配置真相源
- 若需要强制某次 Workflow 绑定固定 Skill 版本，使用 `required_version_id`

### 34.5 运行时投影

Skill 的权威源在 DB，但 Agent CLI 仍然消费文件目录。因此平台必须在 runtime 启动时执行 materialize：

1. 解析本次 AgentRun 使用的 Workflow 版本
2. 读取该 Workflow 当前绑定且启用的 Skill 集合
3. 取出这些 Skill 对应的 bundle 版本及其文件清单
4. 为每个 Skill 创建目录并逐文件写入本次 workspace 的 Agent CLI skills 目录：
   - Claude: `.claude/skills/<skill>/...`
   - Codex: `.codex/skills/<skill>/...`
   - 其他: `.agent/skills/<skill>/...`
5. 在 `agent_runs` 或关联快照表中记录本次使用的 `skill_version_ids`

关键规则：

- runtime 目录只是投影，不是单一信源
- 删除 runtime 目录不会丢失 Skill 内容
- materialize 只能写入目标 root 内部；任何逃逸路径都必须失败
- materialize 时只恢复白名单权限位，例如常规文件 `0644`、可执行脚本 `0755`
- repo 工作区是否已存在不影响 Skill 的查看、编辑、启用、绑定；只影响代码仓库相关执行

### 34.6 `openase-platform` 内置 Skill

这是最重要的 Skill。它教 Agent 如何通过 Platform API 操作 OpenASE，是 Agent 自治闭环的基础能力。

内置 Skill 由平台随安装一并初始化到控制面，不再复制到项目仓库。新 runtime 启动时，平台会像处理其他 Skill 一样把它 materialize 到工作区。

### 34.7 如何存，安全地存

Skill bundle 的存储必须遵循“先解析、后持久化；先规范化、后落库”的原则。

#### 34.7.1 写入流程

1. API 接收一个 skill bundle 请求
   - UI/CLI 可上传 zip/tar.gz
   - 对外 API 也可提交规范化的 `files[]` 结构
2. 在**导入边界**解包并解析为 `ParsedSkillBundle`
3. 执行 bundle 级校验，产出 `ValidatedSkillBundle`
4. 生成规范化 manifest、每个文件的 sha256、整个 bundle 的 `bundle_hash`
5. 在一个数据库事务中写入：
   - `skill_versions`
   - `skill_blobs`
   - `skill_version_files`
   - 必要时更新 `skills.current_version_id`
6. 提交事务成功后，新的 SkillVersion 才对后续 runtime 可见

#### 34.7.2 安全存储规则

- **路径安全**
  - 所有文件路径必须是相对路径
  - 禁止绝对路径、空路径、`.`、`..`、路径穿越、重复规范化路径
  - 禁止 Windows/Unix 混合逃逸技巧（如反斜杠、盘符前缀）
- **文件类型安全**
  - 只接受普通文件
  - 禁止 symlink、hardlink、device file、socket、FIFO
  - 禁止 archive 中的外部链接和权限投毒
- **大小限制**
  - 限制单文件大小、总 bundle 大小、总文件数
  - `SKILL.md` 和引用文档必须控制在可读上限内
  - 二进制 asset 允许存在，但要有更严格的大小阈值
- **编码安全**
  - `SKILL.md` 必须是 UTF-8 文本
  - `agents/openai.yaml`、`references/*`、`scripts/*` 若标记为文本，也必须可解析
  - 二进制内容统一以 blob 存储，不在字符串列中夹带
- **权限安全**
  - 上传时不信任原始 `mode`
  - DB 中只保留一个规范化 `is_executable` 布尔值
  - materialize 时由平台映射成安全权限位
- **内容寻址**
  - `skill_blobs.sha256` 唯一，支持去重
  - `skill_version_files.sha256` 与 `bundle_hash` 用于审计和快照回放
- **事务一致性**
  - 同一个 SkillVersion 的 manifest、file rows、blob 引用必须在一个事务里提交
  - 禁止“版本行已创建但文件未写全”的半成品状态

### 34.8 校验逻辑放在哪个边界

遵循 Parse, Not Validate 原则。校验必须集中在进入系统的边界，不要把 if-check 散落到业务处理路径里。

#### 边界 1：HTTP / CLI / Agent Platform API 输入边界

职责：

- 认证、鉴权、项目归属校验
- 请求 envelope 解析
- 基础字段缺失、格式错误、上传大小超限的拒绝
- archive 上传转成原始文件流或 `RawSkillBundle`

这一层不做业务持久化，不直接决定 Workflow 绑定策略。

#### 边界 2：Skill 导入解析边界（核心边界）

职责：

- 将 `RawSkillBundle` 解析为 `ValidatedSkillBundle`
- 规范化并冻结路径
- 校验目录结构、必需文件、frontmatter、脚本路径、文件数和大小
- 计算 manifest、单文件 hash、bundle hash

这一层产出的领域对象已经是“可安全入库的 skill bundle”。后续 service 只消费这个领域对象。

建议领域类型：

- `SkillName`
- `SkillBundlePath`
- `SkillBundleFile`
- `SkillBundleManifest`
- `ValidatedSkillBundle`

#### 边界 3：Service / Use-Case 层

职责：

- 处理“创建 skill”“发布新版本”“绑定 workflow”“启用/禁用”
- 只接受 `ValidatedSkillBundle`
- 不重复检查路径穿越、frontmatter、archive 结构这些输入约束
- 只处理业务规则，例如“项目内名称唯一”“内置 skill 不允许直接删除”

#### 边界 4：Repository / DB 层

职责：

- 用唯一索引、外键、check constraint 保底
- 例如：
  - `skills(project_id, name)` 唯一
  - `skill_versions(skill_id, version)` 唯一
  - `skill_version_files(skill_version_id, path)` 唯一
  - `skill_blobs(sha256)` 唯一

Repository 不承担 bundle 解析逻辑。

#### 边界 5：Runtime materialize 边界

职责：

- 把已验证、已入库的 SkillVersion 写到工作区
- 再次做“目标路径必须在 root 内”的写出保护
- 不重新做业务校验，只做写文件安全防护

### 34.9 Agent 是否可以创建新 Skill

可以，但必须通过 Platform API，而不是直接在工作区或 repo 目录下写文件并让平台事后 harvest。

正确流程：

1. Agent 在执行中总结出一个可复用模式
2. Agent 调用 `create skill` 或 `update skill` 平台能力提交一个 bundle（最少包含 `SKILL.md`）
3. 平台在导入边界解析 bundle、校验、创建新版本、记录审计
4. 若需要，Agent 或用户再调用 `bind skill to workflow`
5. 新 runtime 默认使用新版本；当前 runtime 若要使用，必须显式 refresh / restart

### 34.10 运行中的 refresh 语义

`refresh` 不再表示“重新扫描 repo 中的 `.openase/skills/` 目录”。

新版语义是：

- `refresh current runtime skills`
  - 从平台控制面重新读取当前绑定 Skill 的**最新已发布 bundle 版本**
  - 覆盖写入当前 runtime 的 Agent CLI skills 目录
  - 需要显式用户/平台动作触发

默认行为仍然是：

- 新 runtime 自动拿最新版本
- 已运行中的 runtime 不自动刷新

### 34.11 管理操作

| 操作 | 说明 |
|------|------|
| 创建 | 通过 Web UI / CLI / Agent Platform API 创建 Skill bundle |
| 更新 | 提交新的 Skill bundle，生成新版本 |
| 绑定 | 将 Skill 绑定到 Workflow |
| 解绑 | 从 Workflow 解绑 |
| 禁用 | Skill 保留，但不再注入新 runtime |
| 启用 | 恢复注入 |
| 删除 / 归档 | Skill 不再可绑定；已有运行中的 runtime 不受影响 |
| 发布到当前 runtime | 显式 refresh 当前 runtime 的 Skill 投影 |

### 34.12 内置 Skill 库

OpenASE 自带一组内置 Skills，由平台在初始化时写入控制面：

| Skill | 说明 | 默认绑定到 |
|-------|------|-----------|
| `openase-platform` | 平台操作能力（创建工单、注册 Repo 等） | 大多数 Workflow |
| `commit` | Conventional Commit 规范 | coding, testing |
| `push` | Git push 规范（force push 保护等） | coding |
| `pull` | Git pull + rebase 流程 | coding |
| `create-pr` | PR 创建规范（标题格式、描述模板、label） | coding |
| `land` | PR 合并流程（CI 检查、squash、clean up） | coding |
| `review-code` | 代码审查规范 | code-reviewer |
| `write-test` | 测试编写规范（命名、覆盖率、mock 策略） | qa-engineer |
| `security-scan` | 安全扫描流程（OWASP Top 10 检查清单） | security-engineer |

### 34.13 Web UI — Skill 管理页面

```
┌──────────────────────────────────────────────────────────────────┐
│ Skills 管理                                        [创建 Skill]  │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│ 🔍 搜索 Skills...                    [全部] [内置] [自创] [禁用]  │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ 📦 openase-platform                          内置 · 已启用  │   │
│ │    平台操作能力（创建工单、注册 Repo 等）                     │   │
│ │    绑定到: coding, testing, dispatcher, security (全部)     │   │
│ │                                        [查看] [解绑]       │   │
│ ├────────────────────────────────────────────────────────────┤   │
│ │ 📦 commit                                    内置 · 已启用  │   │
│ │    Conventional Commit 规范                                │   │
│ │    绑定到: coding, testing                                 │   │
│ │                                  [查看] [绑定更多] [解绑]   │   │
│ ├────────────────────────────────────────────────────────────┤   │
│ │ 🤖 deploy-docker                   自创 (ASE-42) · 已启用   │   │
│ │    Docker 部署标准流程                                      │   │
│ │    绑定到: devops                                          │   │
│ │    创建者: agent:claude-01 via ASE-42                      │   │
│ │                          [查看] [编辑] [绑定更多] [禁用]    │   │
│ ├────────────────────────────────────────────────────────────┤   │
│ │ ⚠️ legacy-deploy                           自创 · 已禁用    │   │
│ │    旧的部署流程（已被 deploy-docker 替代）                    │   │
│ │    绑定到: (无)                                             │   │
│ │                                      [启用] [删除]         │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ 绑定面板 — 拖拽 Skill 到 Harness                                  │
│ ┌──────────────┬──────────────┬──────────────┬──────────────┐   │
│ │ coding       │ testing      │ dispatcher   │ devops       │   │
│ │ ├ platform   │ ├ platform   │ ├ platform   │ ├ platform   │   │
│ │ ├ commit     │ ├ commit     │ └             │ ├ commit     │   │
│ │ ├ push       │ ├ write-test │               │ ├ deploy-    │   │
│ │ ├ pull       │ └             │               │ │  docker   │   │
│ │ ├ create-pr  │              │               │ └             │   │
│ │ └ land       │              │               │              │   │
│ └──────────────┴──────────────┴──────────────┴──────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

Skill 详情页必须能展示：

- 当前版本号与历史版本
- bundle 文件树
- `SKILL.md` 预览
- `scripts/`、`references/`、`assets/` 的存在情况
- 绑定到哪些 Workflow
- 是否为 builtin / enabled

### 34.14 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:projectId/skills` | 列出 Skills（含绑定状态） |
| POST | `/api/v1/projects/:projectId/skills` | 创建 Skill bundle（`multipart archive` 或 `files[]`） |
| GET | `/api/v1/skills/:skillId` | 获取 Skill 详情（含当前版本 manifest） |
| GET | `/api/v1/skills/:skillId/files` | 获取当前版本文件树 |
| GET | `/api/v1/skills/:skillId/files/*path` | 获取单个文件内容 |
| PUT | `/api/v1/skills/:skillId` | 更新 Skill bundle，生成新版本 |
| DELETE | `/api/v1/skills/:skillId` | 删除 Skill |
| POST | `/api/v1/skills/:skillId/enable` | 启用 Skill |
| POST | `/api/v1/skills/:skillId/disable` | 禁用 Skill |
| POST | `/api/v1/skills/:skillId/bind` | 绑定到 Workflow `{workflow_ids: [...]}` |
| POST | `/api/v1/skills/:skillId/unbind` | 从 Workflow 解绑 `{workflow_ids: [...]}` |
| POST | `/api/v1/agent-runs/:runId/skills/refresh` | 显式刷新当前 runtime 的 Skill 投影 |
| GET | `/api/v1/skills/:skillId/history` | 获取 Skill 版本历史 |
| GET | `/api/v1/skills/:skillId/versions/:versionId/files` | 获取历史版本文件树 |

UI 可以用 zip/tar.gz 上传，但平台内部必须在导入边界把 archive 解析成规范化文件集合后再入库。

### 34.15 CLI 导入语义

CLI 必须支持“直接给一个本地目录路径，把它导入到指定 project 作为 Skill bundle”。

推荐命令：

```bash
openase skill import --project <project-id> --path ./path/to/skill-dir
openase skill import <project-id> ./path/to/skill-dir
```

关键规则：

- `--path` / 位置参数必须指向一个本地目录，而不是单文件
- CLI 在**客户端本地**遍历该目录、校验基础文件类型、打包为规范化 bundle 后上传
- 服务端 API **不接受**“请读取服务器上的 `/tmp/foo` 目录”这种请求
- CLI 导入时默认将目录名视为候选 skill name，但仍以 `SKILL.md` frontmatter 中的 `name` 为准
- 若目录名、frontmatter `name`、命令行显式 `--name` 三者冲突，按“显式参数 > frontmatter > 目录名”解析，并要求最终结果一致；不一致则失败
- CLI 不跟随 symlink，不上传 socket、FIFO、device file
- CLI 可以选择两种上传协议：
  - 上传 zip/tar.gz archive
  - 上传规范化后的 `files[]`

推荐最小 CLI 子命令集：

- `openase skill list <project-id>`
- `openase skill import <project-id> <dir>`
- `openase skill export <skill-id> --output ./dir`
- `openase skill bind <skill-id> --workflow <workflow-id>`
- `openase skill unbind <skill-id> --workflow <workflow-id>`
- `openase skill enable <skill-id>`
- `openase skill disable <skill-id>`

补充说明：

- `import` 是“从本地目录导入到平台控制面”
- `export` 是“把某个 SkillVersion 导出回本地目录”
- 两者都不改变“平台 DB 是唯一权威源”这个原则

### 34.16 Skill 校验规则（Fail Fast）

写入 Skill bundle 时严格校验，格式错误的请求直接拒绝，不影响其他 Skill 和 Agent 正常运行：

| 检查项 | 失败行为 |
|--------|---------|
| Skill 名称为空或不合法 | `400 Bad Request` |
| 缺少根目录 `SKILL.md` | `400 Bad Request` |
| `SKILL.md` 非 UTF-8 文本 | `400 Bad Request` |
| `SKILL.md` frontmatter 语法错误 | `400 Bad Request` |
| frontmatter `name` 与 Skill 名称不一致 | `400 Bad Request` |
| 存在绝对路径 / `..` / 重复规范化路径 | `400 Bad Request` |
| 包含 symlink / device / FIFO / socket | `400 Bad Request` |
| 单文件大小超限 | `400 Bad Request` |
| bundle 总大小或文件数超限 | `400 Bad Request` |
| `agents/openai.yaml` 存在但格式非法 | `400 Bad Request` |
| bundle hash 与 manifest 不一致 | `400 Bad Request` |

### 34.17 对架构的影响

| 组件 | 变化 |
|------|------|
| Domain | 新增 `domain/skill/`（`SkillName`、`SkillBundlePath`、`ValidatedSkillBundle`、entity、repository、service） |
| Orchestrator Worker | 启动 Agent 前执行 Skill bundle materialize；不再做 repo harvest |
| Infrastructure | 新增 `infra/skill/`（archive 导入、blob 存储、runtime materialize、格式校验） |
| API | 新增 Skill CRUD + bind/unbind + runtime refresh 端点 |
| CLI | 新增 `openase skill` 子命令（list/create/enable/disable/bind） |
| Web UI | 设置中新增 Skill 管理页面 |
| 数据库 | 新增 `skills`、`skill_versions`、`skill_version_files`、`skill_blobs`、`workflow_skill_bindings` |
| 运行时 | workspace 中的 `.claude/skills` / `.codex/skills` 变成纯投影 |
| 废弃项 | 删除 repo `.openase/skills` 作为权威源；删除 harvest 语义；删除“扫描目录同步索引”模型 |

### 34.18 Skill 生命周期总览

```
        创建 / 更新 bundle
   ┌────────────┬────────────┐
   ▼            ▼            ▼
 Web UI        CLI        Agent Platform API
   │            │            │
   └────────────┼────────────┘
                ▼
      导入边界解析 RawSkillBundle
                ▼
      产出 ValidatedSkillBundle
                ▼
   事务写入 SkillVersion / Files / Blobs
                │
                ▼
       绑定 / 启用 / 发布 / 固定版本
                │
                ▼
     新 runtime materialize 该 bundle 版本
                │
                ├── 默认：仅新 runtime 生效
                │
                └── 显式：refresh 当前 runtime
```

---

## 第三十五章 使用场景验证

本章用三个完整的真实场景，逐步骤验证当前架构是否全部支撑。每个步骤标注依赖的 PRD 章节，不支撑的标注 GAP。

### 35.1 场景一：萌新 MVP——从 Idea 到 Demo

**用户画像**：个人开发者，有一个 App 想法，不想写太多代码，想让 Agent 帮忙快速出 Demo。

**Step 1：用户首次接触**

```
用户下载 openase 二进制 → ./openase up → 浏览器打开 Setup Wizard
→ 选择"个人开发者"模式 → 配置 PostgreSQL（Docker 一行命令）
→ 检测到 Claude Code 已安装 → 创建第一个项目 "my-todo-app"
→ 自动生成默认看板列: Backlog, Todo, In Progress, Done
```

- 依赖：第十四章 Onboarding、第二十九章 Custom Status（默认模板）
- **完全支撑** ✅

**Step 2：用户在侧边栏跟 AI 聊 Idea**

```
用户（在看板页侧边栏）: "我想做一个 Todo App，支持分类和提醒，用 Go 后端 + SvelteKit 前端"

AI 助手（Ephemeral Chat）:
  "不错的想法！根据你的需求，我建议：
   1. 创建两个仓库：backend (Go) 和 frontend (SvelteKit)
   2. 雇佣两个角色：设计助手（负责架构设计）+ 编码助手（负责实现）
   3. 先让设计助手出架构方案，再让编码助手分模块实现

   [一键配置项目]"

用户点击 → AI 通过 action_proposal 调用 Platform API:
  → 注册 backend repo + frontend repo
  → 激活 architect 角色（Harness: roles/architect.md, pickup: "Backlog"）
  → 激活 coding 角色（Harness: roles/fullstack-developer.md, pickup: "Todo"）
  → 创建第一个工单："设计 Todo App 的技术架构"，放入 Backlog
```

- 依赖：第三十一章 Ephemeral Chat（AI 助手）、第二十七章 Agent 自治闭环（action_proposal）、第二十六章 角色体系（角色库）
- **完全支撑** ✅

**Step 3：设计助手接手**

```
编排引擎 Tick:
  → architect Workflow 的 pickup = "Backlog"
  → 发现工单"设计 Todo App 的技术架构"在 Backlog
  → on_claim Hook（克隆两个空 repo）
  → Claude Code 启动，注入 architect Harness

设计助手工作内容:
  1. 分析需求，设计技术架构（API 路由、数据模型、前端页面结构）
  2. 输出架构文档到 backend repo 的 docs/architecture.md
  3. 通过 Platform API 创建编码子工单：
     → "实现 Todo CRUD API" (workflow: coding, status: "Todo")
     → "实现前端 Todo 列表页" (workflow: coding, status: "Todo")
     → "实现前端分类管理页" (workflow: coding, status: "Todo")
     → "实现提醒功能" (workflow: coding, status: "Todo")
  4. 设计助手完成 → 工单移到 Done
```

- 依赖：第七章 编排引擎 Pickup 规则、第二十七章 Agent 创建子工单、第三十四章 Skills（openase-platform skill）
- **完全支撑** ✅

**Step 4：编码助手并行开发**

```
编排引擎 Tick:
  → coding Workflow 的 pickup = "Todo"
  → 发现 4 个编码工单在 Todo
  → max_concurrent = 2，分发前 2 个（按优先级）
  → on_claim Hook（拉取代码、安装依赖）
  → 两个 Claude Code 并行执行

编码助手工作:
  → 直接在 main 分支开发（萌新模式不开 PR 流程）
  → on_complete Hook: "make test"（有测试就跑，没有就跳过）
  → 完成 → 工单移到 Done
  → 编排引擎分发下一批
```

- 依赖：第十章 编排引擎并发控制、第八章 Hook（on_complete 可配为 warn 不阻塞）
- 注意：直接 main 分支开发 = Harness 中 `git.branch_pattern` 不配或配为 main
- **完全支撑** ✅

**Step 5：用户看到效果**

```
用户在看板上看到:
  Backlog (0) | Todo (2) | In Progress (2) | Done (2)

SSE 实时推送 Agent 进度 → 用户看到代码在写
工单全部 Done 后 → 用户 git pull，本地运行看效果
```

- 依赖：第二十九章 Custom Status 看板、SSE 推送
- **完全支撑** ✅

**场景一 GAP 分析：无架构缺口。** 所有环节都有现有章节支撑。唯一需要注意的配置细节是萌新模式下 Harness 的简化配置（无 PR 流程、无审批、Hook 宽松）。

---

### 35.2 场景二：老鸟持续迭代——自主可控流水线

**用户画像**：3-5 人团队，产品已上线，需要持续迭代，代码质量要求高。

**看板配置：**

```
Backlog → Design → Design Review → Todo → In Progress → Code Review → Agent Review → CI/CD → Staging → Production → Done
```

- 依赖：第二十九章 Custom Status（完全自定义列）
- **完全支撑** ✅

**角色配置：**

| 角色 | Harness | pickup | finish | 说明 |
|------|---------|--------|--------|------|
| Dispatcher | dispatcher.md | Backlog | — | 评估需求分配到 Design 或 Todo |
| 设计助手 | architect.md | Design | Design Review | 输出设计方案等人类把关 |
| 编码助手 | fullstack-dev.md | Todo | Code Review | 开 feature branch，提 PR |
| Review 助手 | code-reviewer.md | Agent Review | CI/CD | 自动 code review |
| DevOps | devops.md | CI/CD | Staging | 部署到 staging 环境 |
| 安全工程师 | security-engineer.md | Todo | Done | 定时全量扫描 + PR 合并后增量扫描 |
| 市场调研 | market-analyst.md | — | — | 定时任务触发，不走看板 |

- 依赖：第二十六章角色体系、第三十二章 Dispatcher、第七章 Workflow pickup/finish
- **完全支撑** ✅

**流水线运转：**

```
1. 用户写 Backlog 工单："支持用户头像上传功能"

2. Dispatcher 接手 (pickup: Backlog)
   → 判断需要设计 → 改状态为 "Design"
   → Platform API: openase ticket update --status "Design"

3. 设计助手接手 (pickup: Design)
   → 输出设计方案（API 设计、存储方案、前端交互稿）
   → 完成 → 状态变为 "Design Review"

4. 人类在 "Design Review" 列把关
   → 方案 OK → 手动拖到 "Todo"
   → 方案不行 → 拖回 "Design"，添加评论

5. 编码助手接手 (pickup: Todo)
   → 多 Repo 开发（backend + frontend）
   → 创建 feature branch: agent/ASE-42
   → on_complete Hook: make lint && make test && make typecheck
   → Hook 全部通过 → 提 PR → 状态变为 "Code Review"

6. 人类在 "Code Review" 列 review PR
   → 打回 → 拖回 "Todo"，Agent 下个 Tick 继续处理（attempt + 1）
   → 通过 → 拖到 "Agent Review"

7. Review 助手接手 (pickup: Agent Review)
   → 自动检查代码风格、性能、安全
   → 通过 → 状态变为 "CI/CD"

8. DevOps 助手接手 (pickup: CI/CD)
   → 部署到 staging
   → on_complete Hook: curl staging 健康检查
   → 通过 → 状态变为 "Staging"

9. 人类验证 staging 环境
   → OK → 拖到 "Production"
   → DevOps 助手再次接手 → 部署到生产 → Done

10. 通知系统全程实时推送关键事件到企业微信
```

- 依赖：第七章 pickup/finish、第八章 Hook、第十二章 GitHub 集成（PR）、第二十五章 多机器（staging/production 部署）、第三十三章通知
- **完全支撑** ✅

**市场调研助手（定时触发）：**

```
ScheduledJob:
  cron: "0 9 * * *"  (每天早上 9 点)
  workflow: market-analyst
  ticket_template:
    title: "每日市场调研 - {{ date }}"
    description: "收集竞品动态、行业新闻、用户反馈"
    status: "Backlog"

市场调研助手执行:
  → 搜索竞品更新、行业新闻
  → 生成调研报告
  → 有可行方案 → 通过 Platform API 创建工单到 Backlog
  → 通知人类把关
```

- 依赖：第六章 ScheduledJob、第二十七章 Agent 自治（创建工单）、第三十三章通知
- **完全支撑** ✅

**安全扫描助手（定时 + 事件触发双模式）：**

```
# 模式一：定时全量扫描（每周一早上 9 点）
ScheduledJob:
  cron: "0 9 * * 1"
  workflow: security-engineer
  ticket_template:
    title: "每周安全扫描 - {{ date }}"
    description: "全量代码安全审计 + 依赖漏洞检查"
    status: "Todo"

# 模式二：PR 合并后增量扫描（通过 Notification Rule 触发）
# 当工单到达 Done 且 workflow_type=coding 时，自动创建安全扫描工单
NotificationRule:
  event: ticket.completed
  filter: { workflow_type: "coding" }
  action: 通过 on_done Hook 自动创建安全扫描工单

# 实际在 coding Harness 的 on_done Hook 中配置：
hooks:
  on_done:
    - cmd: |
        openase ticket create \
          --title "安全扫描: ${OPENASE_TICKET_IDENTIFIER} 的代码变更" \
          --workflow security \
          --status "Todo" \
          --parent ${OPENASE_TICKET_IDENTIFIER}
      on_failure: ignore

安全扫描助手执行:
  → 扫描代码变更（增量）或全量仓库
  → 检查依赖漏洞（CVE）
  → 检查硬编码密钥、注入风险、XSS 等
  → 生成安全报告
  → 发现高危漏洞 → 通过 Platform API 创建 urgent 修复工单到 Backlog
  → 通知安全频道（Telegram / 企业微信）

安全扫描角色配置:
  角色: security-engineer
  Harness: roles/security-engineer.md
  pickup: "Todo"
  finish: "Done"
  Skills: openase-platform, security-scan
  Notification: hook.failed → 企业微信 #security-alerts
```

- 依赖：第六章 ScheduledJob（定时触发）、第八章 Hook（on_done 创建扫描工单）、第二十六章角色（security-engineer）、第二十七章 Agent 自治（创建修复工单）、第三十三章通知（安全告警）
- **完全支撑** ✅

**场景二 GAP 分析：**

有一个微小 GAP：**PR 打回后自动重新进入 pickup 的机制**。当前设计中，人类拖回 "Todo" 后，编码助手的 pickup 是 "Todo"，下个 Tick 会重新领取——但此时 `current_run_id` 可能还没清空（上次的 AgentRun 已结束但字段可能没重置）。

修复：Worker 在 Agent 完成（finish 或 fail）后必须清空 `current_run_id`。工单被人类手动拖回 pickup 列时也应该清空。这个在第七章 7.4 节已有设计（`t.CurrentRunID = ""`），但需要确保人类手动拖拽时也触发清空。

→ 需要补充：**任何 status_id 变更（无论是 Agent 自动还是人类手动）都清空 `current_run_id`。** 这保证工单回到任何 pickup 列都能被重新领取。

---

### 35.3 场景三：科研任务——批量 Idea 生产与验证

**用户画像**：科研团队，有 GPU 集群，需要批量探索研究方向。

**看板配置：**

```
Idea Pool → 待调研 → 调研中 → 待实验 → 实验中 → Success → Fail → 待报告 → 报告中 → 人类审核 → Published
```

**机器配置：**

| 机器 | 标签 | 用途 |
|------|------|------|
| local | — | 控制平面 + 文献调研 |
| gpu-01 | gpu, a100 | 模型训练实验 |
| gpu-02 | gpu, h100 | 大模型实验 |
| storage | storage, nfs | 数据存储 |

**角色配置：**

| 角色 | pickup | finish | 执行绑定 | 说明 |
|------|--------|--------|------------|------|
| Idea 生产助手 | — | — | 绑定到 local 上的定时 Agent | 定时任务触发，产出到 "Idea Pool" |
| Dispatcher | Idea Pool | — | 绑定到 local 上的调度 Agent | 评估 Idea 可行性，分到"待调研"或直接"待实验" |
| 文献调研助手 | 待调研 | 待实验 | 绑定到 local 上的调研 Agent | 搜索论文、分析可行性 |
| 实验验证助手 | 待实验 | Success 或 Fail | 绑定到 `gpu-01` 或 `gpu-02` 上的实验 Agent | 编写实验代码、跑实验 |
| 报告编写助手 | 待报告 | 人类审核 | 绑定到 local 上的报告 Agent | 编写精美报告 |

**运转流程：**

```
1. 定时任务：每天 9 点 Idea 生产助手执行
   → ScheduledJob: cron "0 9 * * *", workflow: idea-producer
   → Idea 助手搜索 arXiv 最新论文，分析趋势
   → 通过 Platform API 批量创建工单到 "Idea Pool":
     → "探索: 用 Mamba 替代 Transformer 的 Attention"
     → "探索: 多模态 CoT 在代码生成中的应用"
     → "探索: LoRA 微调在小语种翻译的效果"

2. Dispatcher 接手 (pickup: "Idea Pool")
   → 评估每个 Idea 的可行性和资源需求
   → 高可行性 → 改状态为 "待调研" 或直接 "待实验"
   → 低可行性 → 添加评论说明原因，留在 Idea Pool

3. 文献调研助手接手 (pickup: "待调研")
   → 深度搜索相关论文 15-20 篇
   → 输出文献综述 + 实验设计方案
   → 完成 → 状态变为 "待实验"

4. 实验验证助手接手 (pickup: "待实验")
   → Workflow 绑定到 `experiment-runner-gpu` Agent
   → 该 Agent 绑定 `gpu-01` 上的 Claude Code Provider
   → 编排引擎解析绑定关系后 SSH 到 gpu-01 启动 Claude Code
   → 编写实验代码、跑训练
   → 通过 accessible_machines 访问 storage 机器存放数据

   实验完成后 Agent 判断结果:
   → 效果显著 → Platform API 改状态为 "Success"
   → 效果不佳 → Platform API 改状态为 "Fail"

   → 成功的工单：Agent 自动创建报告工单到 "待报告"
     openase ticket create --title "报告: Mamba Attention 实验结果" \
       --status "待报告" --parent ASE-42

5. 报告编写助手接手 (pickup: "待报告")
   → 读取实验结果（从 storage 机器拉数据）
   → 编写报告：图表、数据分析、结论
   → 输出 report.md 或 report.pdf
   → 完成 → 状态变为 "人类审核"

6. 人类审核
   → 审核报告质量和科研价值
   → 通过 → 拖到 "Published"
   → 需要修改 → 拖回 "待报告" + 添加修改意见

7. 通知系统:
   → "Success" 列有新工单 → Telegram 通知（实验成功了！）
   → "人类审核" 列有新工单 → Telegram 通知（报告待审核）
   → GPU 机器离线 → 企业微信告警
```

- 依赖：第六章 ScheduledJob、第二十五章多机器 SSH、第二十六章角色体系、第二十七章 Agent 自治（创建工单 + 改状态）、第三十二章 Dispatcher、第三十三章通知、第三十四章 Skills
- **完全支撑** ✅

**批量并发：**

```
Idea Pool 中有 10 个 Idea
Dispatcher 快速分配完（每个 Idea 5 秒 × 10 = 50 秒）

gpu-01 和 gpu-02 各跑 1 个实验（max_concurrent=1 per machine）
local 机器跑 3 个文献调研（max_concurrent=3）

→ 10 个 Idea 并行处理，大约 2-3 小时全部出结果
→ 人类第二天看到一批实验报告待审核
```

- 依赖：第十章编排引擎并发控制、第二十五章机器级并发
- **完全支撑** ✅

**场景三 GAP 分析：**

一个小 GAP：**实验验证助手需要根据实验结果动态决定 finish 状态是 "Success" 还是 "Fail"。** 当前如果 finish 只有单值，编排引擎会固定移到那一个状态，表达不了这种分支完成语义。

修复方案：`workflow.finish_status_ids` 必填且允许多值。当集合长度为 `1` 时，编排引擎自动落到该状态；当集合长度大于 `1` 时，Agent 必须通过 `openase ticket update --status ...` 从允许集合中显式选择一个目标状态。平台必须拒绝集合外状态。

→ 需要补充到第七章：**多 finish 时由 Agent 在允许集合中选择；单 finish 时由编排引擎自动完成。**

---

### 35.4 GAP 汇总与修复

| GAP | 影响场景 | 修复方案 | 涉及章节 |
|-----|---------|---------|---------|
| 人类手动拖拽工单时需清空 current_run_id | 场景二（PR 打回重新领取） | 任何 status_id 变更都清空 current_run_id | 第七章 7.4 |
| finish 状态需要支持动态选择（Agent 自决） | 场景三（实验成功/失败不同状态） | `status.finish` 为空时编排引擎不自动移状态，Agent 通过 Platform API 自行设置 | 第七章 7.2 |

两个 GAP 都是配置层面的小补丁，不涉及架构变更。

### 35.5 架构验证结论

三个场景覆盖了 OpenASE 的绝大多数功能模块：

```
                     萌新MVP   老鸟迭代   科研任务
Onboarding (14)        ✅
Ephemeral Chat (31)    ✅        ✅
Custom Status (29)     ✅        ✅         ✅
角色体系 (26)           ✅        ✅         ✅
Dispatcher (32)        ✅        ✅         ✅
Pickup/Finish (7)      ✅        ✅         ✅
Hook 体系 (8)          ✅        ✅         ✅
编排引擎 (10)           ✅        ✅         ✅
Agent 适配器 (11)       ✅        ✅         ✅
多 Repo (6.4)          ✅        ✅         ✅
Agent 自治 (27)        ✅        ✅         ✅
Skills (34)            ✅        ✅         ✅
GitHub 集成 (12)                 ✅
审批 (7.5)                      ✅         ✅
多机器 SSH (25)                             ✅
定时任务 (6.10)                  ✅         ✅
通知系统 (33)                    ✅         ✅
外部同步（28，当前不做）        —
可观测性 (9)                     ✅         ✅
```

**架构完整度评估：当前 35 章设计可以完整支撑三个场景，仅需两个配置级微调（不涉及新增实体或模块）。**

---

## 第三十六章 任务拆解与依赖图

以下是基于完整 PRD（36 章）拆解的所有 Feature，每个标注 ID、依赖关系和预估工作量。依赖关系用 `→` 表示"必须先完成"，`~` 表示"建议先完成但不阻塞"。

### 36.1 依赖关系全局图

```
Layer 0 (基座，无依赖)
  F01 Go 脚手架
  F02 ent Schema + 迁移
  F03 SvelteKit 脚手架 + go:embed
  F71 lefthook pre-commit 配置 → F01
  F72 golangci-lint + depguard 架构守卫 → F71
  F73 SvelteKit ESLint + Prettier + svelte-check → F03, F71

Layer 1 (核心 CRUD，依赖 Layer 0)
  F04 Org/Project CRUD → F01, F02
  F05 TicketStatus 自定义状态 → F02
  F06 Ticket CRUD + 依赖关系 → F04, F05
  F07 ProjectRepo 多仓库 → F04
  F08 TicketRepoScope → F06, F07
  F09 Workflow + Harness 版本存储 → F04
  F10 AgentProvider + Agent 注册 → F04

Layer 2 (编排引擎，依赖 Layer 1)
  F11 Scheduler 调度循环 (Tick) → F06, F09, F10
  F12 Worker + Agent CLI 子进程管理 → F11
  F13 Claude Code CLI 适配器 → F12
  F14 Codex 适配器 → F12
  F15 Ticket Hook (on_claim/on_complete/on_done/on_error) → F11, F12
  F16 Workflow Hook (on_activate/on_reload) → F09
  F17 Pickup/Finish 状态驱动 → F05, F11
  F18 指数退避重试 + 预算暂停 → F11
  F19 Stall 检测 + HealthChecker → F12
  F20 Harness Jinja2 渲染 + 变量注入 → F09, F06

Layer 3 (实时 + 前端，依赖 Layer 1-2)
  F21 EventProvider (ChannelBus + PGNotifyBus) → F01
  F22 SSE Hub (fan-out 广播) → F21
  F23 SSE 端点 (tickets/agents/hooks/activity) → F22
  F24 Web UI 看板页 (自定义状态列 + 拖拽) → F03, F05, F06, F23
  F25 Web UI 工单详情页 → F24
  F26 Web UI Agent 控制台 → F03, F10, F23
  F27 Web UI Workflow 管理 + Harness 编辑器 → F03, F09

Layer 4 (Onboarding，依赖 Layer 2-3)
  F28 Setup Wizard (首次引导 4 步) → F04, F07, F10, F24
  F29 systemd --user / launchd 服务管理 → F01
  F30 openase up 启动流程 → F28, F29
  F31 openase doctor 环境诊断 → F01, F10
  F32 渐进式解锁提示 → F24

Layer 5 (Git 集成，依赖 Layer 2)
  F33 多 Repo 联合工作区 (clone + branch) → F07, F08, F12
  F34 RepoScope PR 链接绑定与展示 → F08, F25

Layer 6 (自治 + 角色，依赖 Layer 2-5)
  F37 Agent Platform API (受控自治) → F06, F07, F12
  F38 Agent Token scope 权限控制 → F37
  F39 openase-platform 内置 Skill → F37
  F40 Skills 生命周期管理 (注入/收割/绑定) → F09, F12, F39
  F41 角色库 (内置 Harness 模板) → F09, F40
  F42 Dispatcher Workflow (自动分配) → F37, F41
  F43 HR Advisor 推荐引擎 → F41, F06

Layer 7 (通知，依赖 Layer 3-6)
  F44 NotificationChannel + ChannelAdapter (Telegram/企业微信/Slack/...) → F21
  F45 NotificationRule 订阅规则 → F44
  F46 通知管理 UI → F03, F44, F45

Layer 8 (高级功能，依赖 Layer 6-7)
  F51 Ephemeral Chat (内嵌 AI 助手) → F13, F06
  F52 Harness 编辑器 AI 辅助 (侧栏对话) → F51, F27
  F53 Harness 变量字典 API + 编辑器自动补全 → F20, F27
  F56 ScheduledJob 定时任务 → F06, F09
  F57 TicketExternalLink (多 Issue 关联) → F06
  F58 Refine-Harness 元工作流 → F42, F40

Layer 9 (多机器，依赖 Layer 2)
  F59 Machine 实体 + SSH 连接池 → F04
  F60 Machine Monitor L1-L3 (网络/资源/GPU) → F59
  F61 Machine Monitor L4-L5 (Agent 环境/完整审计) → F60
  F62 SSH Agent Runner (远端 clone + 启动) → F12, F59, F33
  F63 工单绑定机器 + 标签自动匹配 → F59, F11
  F64 Agent 跨机器访问 (Harness 注入) → F59, F20
  F65 Environment Provisioner Agent → F62, F41, F61
  F66 Machine 管理 UI → F03, F59, F60

Layer 10 (可观测性 + 后期治理)
  F67 OTel TraceProvider 实现 → F01
  F68 OTel MetricsProvider 实现 → F01
  F69 内存 Metrics + Web UI 仪表盘 → F68, F03
  F70 成本追踪 (Token 消耗 + 预算) → F12, F68
  F74 Conventional Commits 校验 → F71

Layer 11 (企业级 + 开放生态)
  F75 Gemini CLI 适配器 → F12
  F76 OIDC 认证 → F04
  F77 Team / Member / Role / Permission → F04, F76
  F79 开放 API + Go SDK + TypeScript SDK → F06
  F80 自定义 Adapter 插件系统 → F12
  F81 升级 + 自动迁移机制 → F02
```

### 36.2 任务清单（按依赖拓扑排序）

**Layer 0 — 基座 + 工程基线（可并行，第 1-2 周，F71-F73 在第 1 周优先完成）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F01 | Go 项目脚手架：cobra CLI + Echo 路由 + viper 配置 + slog 日志 | 3d | 无 | 5 |
| F02 | ent Schema 全量定义 + atlas 迁移 + 数据库索引 | 5d | 无 | 6, 20 |
| F03 | SvelteKit 脚手架 + Tailwind + shadcn-svelte + `go:embed` 集成 | 3d | 无 | 5 |
| F71 | lefthook 配置 + Makefile | 1d | F01 | 15 |
| F72 | golangci-lint 严格配置 + depguard 架构守卫 | 1d | F71 | 15 |
| F73 | SvelteKit ESLint + Prettier + svelte-check | 1d | F03, F71 | 15 |

**Layer 1 — 核心 CRUD（第 2-4 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F04 | Org / Project CRUD（API + 前端页面） | 3d | F01, F02 | 6, 18 |
| F05 | TicketStatus 自定义状态 CRUD + 默认模板生成 | 3d | F02 | 29 |
| F06 | Ticket CRUD + 依赖关系（blocks / sub-issue） | 5d | F04, F05 | 6, 18 |
| F07 | ProjectRepo 多仓库 CRUD | 2d | F04 | 6 |
| F08 | TicketRepoScope（工单绑定多 Repo） | 2d | F06, F07 | 6 |
| F09 | Workflow CRUD + Harness 版本存储 + runtime materialize | 5d | F04 | 6, 7 |
| F10 | AgentProvider + Agent 注册 + PATH 自动检测 | 3d | F04 | 6 |

**Layer 2 — 编排引擎（第 4-7 周，核心路径）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F11 | Scheduler 调度循环（Tick 模式 + pickup 匹配） | 5d | F06, F09, F10 | 7, 10 |
| F12 | Worker + Agent CLI 子进程管理（os/exec + context） | 5d | F11 | 8, 10 |
| F13 | **Claude Code CLI 适配器**（NDJSON stream + multi-turn + --resume） | 5d | F12 | 11 |
| F14 | Codex 适配器（JSON-RPC over stdio） | 3d | F12 | 11 |
| F15 | Ticket Hook 执行引擎（shell 子进程 + 环境变量注入 + on_failure 策略） | 4d | F11, F12 | 8 |
| F16 | Workflow Hook（on_activate / on_reload） | 2d | F09 | 8 |
| F17 | Pickup/Finish 状态驱动 + current_run_id 清空规则 | 2d | F05, F11 | 7 |
| F18 | 指数退避重试 + retry_paused + budget_exhausted 暂停 | 3d | F11 | 7, 19 |
| F19 | Stall 检测 + HealthChecker | 2d | F12 | 10, 19 |
| F20 | Harness Jinja2 渲染（gonja）+ 完整变量字典注入 | 3d | F09, F06 | 30 |

**Layer 3 — 实时推送 + 基础前端（第 5-8 周，与 Layer 2 部分并行）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F21 | EventProvider 接口 + ChannelBus + PGNotifyBus 实现 | 3d | F01 | 5 |
| F22 | SSE Hub（fan-out 广播 + 连接注册/注销） | 3d | F21 | 16 |
| F23 | SSE HTTP 端点（tickets / agents / hooks / activity） | 2d | F22 | 18 |
| F24 | **Web UI 看板页**（自定义状态列 + 拖拽 + SSE 实时更新） | 8d | F03, F05, F06, F23 | 13, 29 |
| F25 | Web UI 工单详情页（多 Repo PR 链接列表 + 活动日志 + Hook 历史） | 5d | F24 | 13 |
| F26 | Web UI Agent 控制台（状态 + 实时输出流 + 心跳） | 3d | F03, F10, F23 | 13 |
| F27 | Web UI Workflow 管理 + Harness 编辑器（语法高亮 + YAML 校验） | 5d | F03, F09 | 13, 30 |

**Layer 4 — Onboarding（第 7-9 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F28 | **Setup Wizard**（4 步引导 + 模式选择 + DB 测试连接 + 项目与内置 Workflow 初始化） | 5d | F04, F07, F10, F24 | 14 |
| F29 | systemd --user / launchd 自动安装 + openase up/down/restart/logs | 3d | F01 | 14 |
| F30 | openase up 完整启动流程（检测配置 → Wizard / 正常启动） | 2d | F28, F29 | 14 |
| F31 | openase doctor 环境诊断 | 2d | F01, F10 | 14 |
| F32 | 渐进式解锁提示（里程碑检测 + UI 横幅） | 2d | F24 | 14 |

**Layer 5 — Git 集成（第 8-10 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F33 | **多 Repo 联合工作区**（clone + feature branch + 分支命名规范） | 4d | F07, F08, F12 | 6, 25 |
| F34 | RepoScope PR 链接绑定与展示（手动/API/平台操作） | 3d | F08, F25 | 12, 16 |

**Layer 6 — Agent 自治 + 角色体系（第 9-12 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F37 | **Agent Platform API**（环境变量注入 + API 路由 + Token 生成） | 4d | F06, F07, F12 | 27 |
| F38 | Agent Token scope 权限控制（Harness platform_access 白名单） | 3d | F37 | 27 |
| F39 | openase-platform 内置 Skill（SKILL.md + openase CLI wrapper） | 2d | F37 | 34 |
| F40 | **Skills 生命周期**（注入到 Agent CLI 目录 + 收割 + 绑定/解绑 + refresh） | 5d | F09, F12, F39 | 34 |
| F41 | 角色库（14 个内置 Harness 模板 + Web UI Skill 管理页） | 5d | F09, F40 | 26, 34 |
| F42 | **Dispatcher Workflow**（Backlog 自动分配角色） | 3d | F37, F41 | 32 |
| F43 | HR Advisor 推荐引擎（规则分析 + UI 推荐面板） | 3d | F41, F06 | 26 |

**Layer 7 — 通知（第 11-14 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F44 | NotificationChannel + ChannelAdapter（Telegram / 企业微信 / Slack / Email / Webhook） | 5d | F21 | 33 |
| F45 | NotificationRule 订阅规则（事件匹配 + filter + Jinja2 消息模板） | 3d | F44 | 33 |
| F46 | 通知管理 UI（渠道配置 + 规则配置 + 测试发送） | 3d | F03, F44, F45 | 33 |

**Layer 8 — 高级功能（第 12-16 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F51 | **Ephemeral Chat**（内嵌 AI 助手 + 上下文注入 + action_proposal） | 5d | F13, F06 | 31 |
| F52 | Harness 编辑器 AI 辅助（侧栏对话 + diff 应用） | 3d | F51, F27 | 31 |
| F53 | Harness 变量字典 API + 编辑器自动补全 + 实时预览 | 3d | F20, F27 | 30 |
| F56 | ScheduledJob 定时任务（robfig/cron + 工单模板 + UI） | 3d | F06, F09 | 6 |
| F57 | TicketExternalLink（多 Issue 关联 + API + UI） | 2d | F06 | 6 |
| F58 | Refine-Harness 元工作流（分析历史 + 自动优化 Harness） | 3d | F42, F40 | 26 |

**Layer 9 — 多机器（第 14-18 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F59 | Machine 实体 + SSH 连接池 | 4d | F04 | 25 |
| F60 | Machine Monitor L1-L3（ping / 资源 / GPU，金字塔频率） | 4d | F59 | 25 |
| F61 | Machine Monitor L4-L5（Agent 环境 / 完整审计） | 3d | F60 | 25 |
| F62 | **SSH Agent Runner**（远端 git clone + Skill 注入 + 启动 Agent CLI） | 5d | F12, F59, F33 | 25 |
| F63 | Provider 绑定 Machine + Workflow 绑定 Agent 调度链路 | 2d | F59, F11 | 25 |
| F64 | Agent 跨机器访问（Harness 注入 accessible_machines） | 2d | F59, F20 | 25 |
| F65 | Environment Provisioner Agent（SSH + 预设 Skill 修复环境） | 3d | F62, F41, F61 | 25 |
| F66 | Machine 管理 UI | 3d | F03, F59, F60 | 25 |

**Layer 10 — 可观测性 + 后期治理（可随时插入，主要在后期完成）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F67 | OTel TraceProvider 实现（Span 创建 + 导出 + 请求追踪中间件） | 3d | F01 | 9 |
| F68 | OTel MetricsProvider 实现（Counter / Histogram / Gauge + 导出） | 3d | F01 | 9 |
| F69 | 内存 Metrics + Web UI 仪表盘（工单吞吐 / 成本 / Agent 利用率） | 4d | F68, F03 | 9, 13 |
| F70 | 成本追踪（Token 消耗记录 + 预算告警 + 自动熔断） | 3d | F12, F68 | 9 |
| F74 | Conventional Commits 校验（commit-msg hook） | 0.5d | F71 | 15 |

**Layer 11 — 企业级 + 开放生态（第 18-26 周）**

| ID | 任务 | 工作量 | 依赖 | PRD 章节 |
|----|------|--------|------|---------|
| F75 | Gemini CLI 适配器 | 3d | F12 | 11 |
| F76 | OIDC 认证（Provider 实现 + Setup Wizard 企业模式） | 4d | F04 | 5, 14 |
| F77 | Team / Member / Role / Permission（RBAC） | 8d | F04, F76 | — |
| F79 | 开放 API 文档 + Go SDK + TypeScript SDK | 5d | F06 | — |
| F80 | 自定义 Adapter 插件系统 | 5d | F12 | — |
| F81 | 升级 + 自动迁移机制（atlas versioned migration） | 3d | F02 | 23 |

### 36.3 关键路径（Critical Path）

最长依赖链决定了最短交付时间：

```
F01 → F02 → F04 → F06 → F11 → F12 → F13 → F33 → F62
 3d    5d    3d    5d    5d    5d    5d    4d    5d  = 40 工作日 ≈ 8 周

并行路径:
F03 → F24 → F28 → F30 (前端 + Onboarding = 18d)
F21 → F22 → F23 (SSE = 8d)
F71 → F72 与 F03 → F73 (工程规范基线 = 2-3d，第 1 周优先完成，随后全程护航)
```

**Phase 1 里程碑（Week 8）：** F01-F32 全部完成，一个用户可以 `openase up` → Setup Wizard → 创建工单 → Claude Code Agent 自动编码 → 看板实时更新 → 工单完成。

**Phase 2 里程碑（Week 14）：** 加上 F33-F46，Git 集成 + Agent 自治 + 角色库 + Dispatcher + 通知 全部可用。

**Phase 3 里程碑（Week 18）：** 加上 F51-F66，Ephemeral Chat + 定时任务 + 多机器 + 审批 全部可用。

**Phase 4 里程碑（Week 26）：** 加上 F67-F70、F74-F81，可观测性 + 企业级 + 开放 API 全部可用；F71-F73 已在前期作为工程基线完成。

### 36.4 并行开发建议

假设 3 人团队：

| 开发者 | Week 1-4 | Week 5-8 | Week 9-12 | Week 13-16 |
|--------|---------|---------|----------|----------|
| **后端 A** | F01, F02, F71, F72, F04, F06 | F11, F12, F13, F15 | F37, F38, F40, F42 | F59, F62, F63 |
| **后端 B** | F09, F10, F05, F07 | F14, F17, F18, F19, F20, F21 | F34, F44, F45 | F56, F54 |
| **前端** | F03, F73, F74, F22 预研 | F22, F23, F24, F25, F26, F27 | F28, F30, F31, F32, F41 | F46, F51, F52, F53, F55 |

---

*— END OF DOCUMENT —*
