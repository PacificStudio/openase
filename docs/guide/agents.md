# Agents（智能体）

## 这是什么？

Agent 是平台中的**AI 执行者**。它们是实际"动手干活"的角色——读代码、写代码、跑测试、提交 PR。你不需要手动指挥它们做每一步，只需要创建工单，Agent 会按照 [Workflow](./workflows.md) 自动执行。

## 基本概念

| 概念 | 说明 |
|------|------|
| **Agent 定义** | 项目级的配置，绑定了 AI Provider（如 Claude Code）和项目参数 |
| **Agent Run（执行记录）** | Agent 处理某个工单的一次完整执行过程 |
| **Agent Output（输出）** | 执行过程中产生的日志和结果 |
| **Agent Step（执行步骤）** | 人类可读的操作阶段描述 |

## 支持的 AI Provider

| Provider | 来源 |
|----------|------|
| **Claude Code** | Anthropic |
| **Codex** | OpenAI |
| **Gemini CLI** | Google |

## 常用操作

### 注册 Agent

1. 进入 Agents 页面
2. 从可用 Provider 中选择一个
3. 为 Agent 命名并确认创建

### 监控 Agent

- 侧边栏会显示当前活跃 Agent 数量的徽标
- 点击进入可查看所有 Agent 的状态（活跃 / 暂停 / 已退役）
- 点击具体 Agent 可查看其正在处理的工单和历史执行记录

### 管理 Agent 生命周期

| 操作 | 说明 |
|------|------|
| **暂停（Pause）** | 临时停止 Agent 领取新工单 |
| **恢复（Resume）** | 从暂停状态恢复 |
| **退役（Retire）** | 永久停用一个 Agent |

### 实时查看执行

Agent 执行过程支持实时流式输出（SSE）。你可以在 Agent Run 详情页看到：

- 每一步的操作描述
- 实时日志输出
- 执行状态变更

## 小贴士

- 同一个项目可以注册多个 Agent，分别绑定不同的工作流（如一个负责写代码，一个负责测试）
- 如果 Agent 执行异常，先检查 [Machine](./machines.md) 的连接状态和 [Workflow](./workflows.md) 的 Harness 配置
- Agent 的执行历史可以帮助你理解 AI 的决策过程，优化 Harness 指令
