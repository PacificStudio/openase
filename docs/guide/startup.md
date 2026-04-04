# 快速上手（Startup）

如果你是第一次使用 OpenASE，按照下面 5 个步骤即可快速跑通整个流程。

## 第 1 步：创建项目并配置基础设置

进入 **[Settings](./settings.md)**，完成以下操作：

- 填写项目名称和描述
- 配置工单状态（如 `待处理`、`进行中`、`已完成`），这些状态会成为看板的列
- 关联代码仓库（GitHub / GitLab），让 Agent 能够访问代码

## 第 2 步：注册机器

进入 **[Machines](./machines.md)**，添加 Agent 将要运行的执行环境：

- 可以是本地机器、远程 SSH 主机或云端 VM
- 添加后点击"测试连接"确认机器可达

## 第 3 步：注册 Agent

进入 **[Agents](./agents.md)**，从可用的 AI Provider（如 Claude Code、Codex、Gemini CLI）中注册一个 Agent。

## 第 4 步：创建工作流

进入 **[Workflows](./workflows.md)**，创建一个工作流模板：

- 选择要绑定的 Agent
- 编写 Harness（指令文档），告诉 Agent 该如何执行任务
- 配置哪些工单状态会触发 Agent 自动领取

## 第 5 步：创建工单

进入 **[Tickets](./tickets.md)**，创建你的第一个工单：

- 填写标题和描述
- 选择刚才创建的工作流
- 将工单状态设为"待处理"

此时 Agent 会自动检测到这个工单，领取并开始执行。你可以在 **[Activity](./activity.md)** 中实时查看执行过程。

## 工作原理

```
用户创建工单 → 编排器检测到工单处于领取状态
    → Agent 领取工单 → 在 Machine 上按 Workflow 执行
    → 执行过程记录到 Activity → 工单完成
```

## 下一步

完成上面的流程后，你已经体验了 OpenASE 的核心能力。接下来可以：

- 深入了解 [Workflows](./workflows.md) 的 Harness 编写，优化 Agent 执行效果
- 探索 [Skills](./skills.md)，给工作流添加可复用的技能
- 设置 [Scheduled Jobs](./scheduled-jobs.md)，让重复性工作完全自动化
- 使用 [Updates](./updates.md) 记录项目进展，方便团队协作
