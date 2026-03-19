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
    - In Review
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
  max_concurrent_agents: 3
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
5. 最终目标是在 `main` 上直接完成实现、验证并推送可工作的结果，而不是输出长篇解释。

## 主干规则

当前阶段统一直接在 `main` 分支工作：

- 不创建 feature branch
- 不创建 PR
- 所有实现、验证、提交和推送都直接落在当前工作区的 `main`
- 开工前先同步 `origin/main`
- 如果同步或推送时遇到冲突，必须在本地解决冲突并重新验证，再继续推送

## 默认执行方式

- 先确认当前工单状态，再选择对应流程。
- 每次开工先查找或创建一条持久化工作台评论，标题标记为 `## Codex Workpad`。
- 工作台评论是唯一事实来源：计划、验证、阻塞、推送记录、反馈处理都持续写到这一条里。
- 实现前先读取相关代码和 `OpenASE-PRD.md` 的相关章节，但不要拉长方案阶段。
- 如果改动可拆成更小的可交付增量，优先先交一个完整但窄的版本。
- 提交前至少运行与改动相关的最小验证，并把命令和结果记入工作台评论。

## 状态映射

- `Backlog` -> 不处理，等待人类手动推进。
- `Todo` -> 立即切换到 `In Progress`，然后开始实现，不经过 `Spec`。
- `In Progress` -> 直接在 `main` 实现、验证并推送；没有待处理反馈时进入 `Done`。
- `In Review` -> 视为存在待处理反馈；继续在 `main` 上修改、验证、推送，清扫完反馈后进入 `Done`。
- `Rework` -> 基于反馈继续在 `main` 上实现；处理完后重新验证并推送，随后进入 `Done`。
- `Merging` -> 视为需要完成主干同步；拉取最新 `origin/main`、解决冲突、重新验证并推送，随后进入 `Done`。
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
   - 确认当前分支就是 `main`
   - 拉取并快进或变基到最新 `origin/main`
   - 如果同步时出现冲突，先解决冲突，再继续当前工单
6. 直接在 `main` 上实现最小可交付版本，保持改动聚焦在当前工单。
7. 完成后运行必要验证，更新工作台评论，提交并直接推送到 `origin/main`。
8. 如果推送失败或远端发生冲突，重新同步 `origin/main`，解决冲突，重新运行受影响的验证，再继续推送。
9. 推送完成后更新工作台评论，并将工单推进到 `Done`；如果仍有明确反馈，则继续回到步骤 6 处理。

## 反馈处理协议

当工单存在待处理反馈时，在进入 `Done` 前必须清扫所有可执行反馈：

- issue 评论
- commit 评论
- 已存在的 review 评论或 review summary（如果历史上有 PR）

每条反馈都必须满足以下之一：

- 代码或文档已经修改并解决；或
- 在对应线程中给出明确、有根据的回绝说明。

## 阻塞处理

仅在以下情况允许提前停止：

- 缺少 GitHub 权限或鉴权，无法更新推送 / 评论 / Project 状态
- 缺少必须的外部密钥、服务访问权限或运行环境
- 缺少完成验收所需的关键工具

出现阻塞时：

- 在 `## Codex Workpad` 中写清楚缺什么、为什么阻塞、需要什么人类动作
- 若当前工作已无法继续推进，则保持现状并在工作台评论中明确阻塞原因

## 输出要求

- 回复只报告已完成动作、验证结果、推送提交/评论链接和阻塞项。
- 不要给人类布置泛泛的“下一步”。
