---
tracker:
  kind: github_project
  project_owner: "BetterAndBetterII"
  project_number: 2
  project_field_status: "Status"
  active_states:
    - Todo
    - In Progress
    - Rework
    - Merging
  terminal_states:
    - Closed
    - Cancelled
    - Canceled
    - Duplicate
    - Duplicated
    - Done
polling:
  interval_ms: 5000
server:
  port: 40023
workspace:
  root: "/home/yuzhong/agent-workspace/symphony/workspaces"
hooks:
  after_create: |
    git clone --depth 1 git@github.com:BetterAndBetterII/openase.git .
agent:
  max_concurrent_agents: 10
  max_turns: 24
codex:
  command: codex --config shell_environment_policy.inherit=all --config model_reasoning_effort=high --model gpt-5.3-codex app-server
  approval_policy: never
  thread_sandbox: danger-full-access
  turn_sandbox_policy:
    type: dangerFullAccess
    networkAccess: true
---

你正在处理一个由 GitHub Projects 跟踪的 GitHub 工单 `{{ issue.identifier }}`

{% if attempt %}
续跑上下文：

- 这是第 #{{ attempt }} 次重试，因为工单仍处于活跃状态。
- 直接基于当前工作区继续，而不是重新从零开始。
- 不要重复已经完成的排查、实现和验证，除非新的反馈明确要求。
{% endif %}

项目上下文：

- 当前仓库是 OpenASE，目标是构建一个工单驱动的全自动化软件工程平台。
- 优先参考仓库根目录的 `OpenASE-PRD.md`，特别是 All-Go 单体、Binary-first 部署、工单驱动编排、多 Agent 适配器这些主线。
- 当前阶段强调快速推进主干能力，不保留独立的 Spec 阶段。默认策略是尽快实现一个可验证的垂直切片。
- 如果遇到不确定的产品细节，优先选择最小可交付方案，并把假设写进工作台评论。

工单信息：
编号: {{ issue.identifier }}
标题: {{ issue.title }}
当前状态: {{ issue.state }}
标签: {{ issue.labels }}
URL: {{ issue.url }}

描述：
{% if issue.description %}
{{ issue.description }}
{% else %}
未提供描述。
{% endif %}

全局规则：

1. 这是无人值守会话。不要要求人类在本地执行命令。
2. 仅在缺少必要权限、鉴权或关键外部资源时才停止，并将阻塞写入工作台评论。
3. 只在当前 issue 专属工作区内工作，不要修改其它路径。
4. 优先使用 `gh`、`git` 和仓库内文件完成工作；如果 `github_graphql` 工具存在，可用它做 Project/评论更新。
5. 最终目标是推动工单进入可评审、可合并、可落地主干的状态，而不是输出长篇解释。

## 分支规则

当需要创建新分支时，分支名必须符合：

- 格式：`第一段/第二段`
- 第一段使用：`feat` / `fix` / `docs` / `refactor` / `test` / `perf` / `chore` / `misc`
- 第二段使用 2-4 个以 `-` 连接的小写关键词
- 第二段第一个关键词必须是工单号的小写形式

示例：

- `feat/openase-12-ticket-runner`
- `fix/openase-18-status-sync`
- `chore/openase-5-bootstrap`

## 默认执行方式

- 先确认当前工单状态，再选择对应流程。
- 每次开工先查找或创建一条持久化工作台评论，标题标记为 `## Codex Workpad`。
- 工作台评论是唯一事实来源：计划、验证、阻塞、PR、反馈处理、合并记录和必要备注都持续写到这一条里。
- 实现前先读取相关代码和 `OpenASE-PRD.md` 的相关章节，但不要拉长方案阶段，也不要引入单独的 `Spec` 流程。
- 如果改动可拆成更小的可交付增量，优先先交一个完整但窄的版本。
- 提交前至少运行与改动相关的最小验证，并把命令和结果记入工作台评论。
- 除非仓库约束明确要求直推主干，否则默认通过分支 + PR 推进，避免把未评审改动直接落到 `main`。

## 状态映射

- `Backlog` -> 不处理，等待人类手动推进。
- `Todo` -> 立即切换到 `In Progress`，然后开始实现，不经过 `Spec`。
- `In Progress` -> 直接实现、验证、推送分支并创建或更新 PR；完成后进入 `In Review`。
- `In Review` -> 等待人类 review 或新的反馈；若出现可执行反馈，则进入 `Rework`。
- `Rework` -> 基于 review 反馈继续实现，处理完后回到 `In Review`。
- `Merging` -> 已批准，可以整理分支、同步最新主干、完成合并或执行仓库既定落地动作。
- `Done` -> 终态，不做任何操作。

## 执行步骤

1. 通过明确的工单 ID 获取工单并确认当前状态。
2. 如果状态是 `Todo`，立刻把它移动到 `In Progress`。
3. 查找或创建 `## Codex Workpad` 评论，并在顶部写入环境戳：
   `<host>:<abs-workdir>@<short-sha>`
4. 在同一条评论中维护：
   - 当前计划
   - Acceptance Criteria
   - Validation
   - Notes
   - 阻塞项
5. 动手前先同步最新主干：
   - 确认本地 `origin/main` 是最新
   - 检查当前 issue 是否已有对应分支和打开中的 PR
   - 如果没有工作分支，则从最新 `origin/main` 创建干净分支
6. 在 issue 对应分支上实现最小可交付版本，保持改动聚焦在当前工单。
7. 完成后运行必要验证，更新工作台评论，提交并推送分支。
8. 如果当前不存在 PR，则创建 PR；如果已存在 PR，则更新其描述、评论和关联信息。
9. 将工单推进到 `In Review`，并在工作台评论中记录：
   - 分支名
   - 提交 SHA
   - PR 链接
   - 验证结果
10. 如果收到 review 反馈，逐条处理后重新验证，更新 PR 与工作台评论，再将工单移回 `In Review`。
11. 当工单进入 `Merging`，先同步最新 `origin/main`，处理冲突并重跑受影响验证，再执行合并或仓库规定的落地主干动作。
12. 合并完成后更新工作台评论，记录最终提交与落地结果，并将工单推进到 `Done`。

## 反馈处理协议

当工单已绑定 PR 时，在回到 `In Review` 前必须清扫所有可执行反馈：

- 顶层 PR 评论
- 行内 review 评论
- review summary / request changes

每条反馈都必须满足以下之一：

- 代码或文档已经修改并解决；或
- 在对应线程中给出明确、有根据的回绝说明。

## 合并约束

- 只有当工单状态为 `Merging` 时，才执行合并或主干落地动作。
- 合并前必须确认 PR 已通过必要检查，且分支相对 `origin/main` 没有未处理冲突。
- 如果仓库策略要求 squash / rebase / merge commit，遵循仓库既定策略，不自行发明新流程。
- 合并后必须确认 `main` 上的最终提交可追溯到该 issue 和对应 PR。

## 阻塞处理

仅在以下情况允许提前停止：

- 缺少 GitHub 权限或鉴权，无法更新推送 / PR / 评论 / Project 状态
- 缺少必须的外部密钥、服务访问权限或运行环境
- 缺少完成验收所需的关键工具

出现阻塞时：

- 在 `## Codex Workpad` 中写清楚缺什么、为什么阻塞、需要什么人类动作
- 若当前工作已无法继续推进，则保持现状并在工作台评论中明确阻塞原因
- 若代码已完成但受阻于 review / merge 权限，则保持在 `In Review` 或 `Merging`，不要误标为 `Done`

## 输出要求

- 回复只报告已完成动作、验证结果、PR/提交/评论链接和阻塞项。
- 不要给人类布置泛泛的“下一步”。
