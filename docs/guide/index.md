# OpenASE 用户指南

OpenASE 是一个**工单驱动的自动化软件工程平台**。你可以把它想象成一个"AI 开发团队的项目管理中心"——你创建工单描述需求，AI Agent 自动领取并执行，整个过程可追踪、可管控。

## 指南目录

### 入门

- [快速上手（Startup）](./startup.md) — 5 步跑通完整流程，从零到第一个 AI 执行的工单

### 功能模块

| 模块 | 说明 | 文档 |
|------|------|------|
| Tickets | 工单管理，平台的核心单元 | [查看](./tickets.md) |
| Agents | AI 智能体的注册与监控 | [查看](./agents.md) |
| Machines | 执行环境的管理 | [查看](./machines.md) |
| Updates | 人工发布的项目动态 | [查看](./updates.md) |
| Activity | 系统自动生成的活动日志 | [查看](./activity.md) |
| Workflows | Agent 的行为模板定义 | [查看](./workflows.md) |
| Skills | 可复用的自动化技能包 | [查看](./skills.md) |
| Scheduled Jobs | 按计划自动创建工单 | [查看](./scheduled-jobs.md) |
| Settings | 项目基础配置 | [查看](./settings.md) |

### 附录

- [模块关系图](./architecture.md) — 各模块之间的协作关系
- [常见问题](./faq.md) — 常见问题排查指南

## 工作原理概览

```
用户创建工单 → 编排器检测到工单处于领取状态
    → Agent 领取工单 → 在 Machine 上按 Workflow 执行
    → 执行过程记录到 Activity → 工单完成
```

建议从 [快速上手](./startup.md) 开始，然后按需阅读各模块文档。
