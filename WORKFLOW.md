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
- 所有实现都必须与 `OpenASE-PRD.md` 保持一致，避免代码、文档和需求理解之间出现错配。
- 如果 issue 明确要求新增能力、调整行为或修正现有产品定义，且现有 `OpenASE-PRD.md` 已不足以准确描述最终实现，则必须同步更新 PRD。
- 如果 issue 只是实现既有需求、修复缺陷或做非需求变更，代码修改不得脱离 `OpenASE-PRD.md` 已定义的范围与约束。
- 如果改动可拆成更小的可交付增量，优先先交一个完整但窄的版本。
- 提交前至少运行与改动相关的最小验证，并把命令和结果记入工作台评论。
- 除非仓库约束明确要求直推主干，否则默认通过分支 + PR 推进，避免把未评审改动直接落到 `main`。

## 状态映射

- `Backlog` -> 不处理，等待人类手动推进。
- `Todo` -> 立即切换到 `In Progress`，然后开始实现，不经过 `Spec`。
- `In Progress` -> 直接实现、验证、推送分支并创建或更新 PR；但在进入 `In Review` 前，必须先确认 PR 的 base branch，把当前分支更新到该 base branch 的最新提交，并确保与仓库 workflow/CI 对齐的验证已经跑通。
- `In Review` -> 这是可执行状态，必须主动 pick up 并审核当前 PR / 分支代码；最重要的是确认 `OpenASE-PRD.md`、issue 目标与当前实现保持一致，没有需求错配、文档漂移或理解偏差。若无阻塞问题，则推进到 `Merging`；若存在问题，则提交 `change request` 并把工单推进到 `Rework`。
- `Rework` -> 基于 review 反馈继续实现；回到 `In Review` 前，同样必须把当前分支同步到最新的 PR base branch，重跑受影响验证，并确认相关 workflow/CI 已重新跑通。
- `Merging` -> 已批准，可以整理分支、同步最新主干、完成合并或执行仓库既定落地动作；但在移动到 `Done` 前，必须先手动关闭对应 GitHub issue。
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
6. 如果当前状态是 `In Progress` 或 `Rework`，则在 issue 对应分支上实现最小可交付版本，保持改动聚焦在当前工单。
7. 对于 `In Progress` / `Rework` 路径，完成实现后先确定目标 base branch：
   - 若已有 PR，以 PR 的 base branch 为准
   - 若还没有 PR，以仓库默认目标分支为准（通常是 `origin/main`）
8. 对于 `In Progress` / `Rework` 路径，在进入 `In Review` 前，必须获取最新 base branch，并把当前工作分支更新到该 base branch 的最新提交：
   - 按仓库既定策略执行 merge / rebase / update branch
   - 处理冲突
   - 重跑受影响的验证
   - 确认 PR 不再落后于 base branch
9. 对于 `In Progress` / `Rework` 路径，在进入 `In Review` 前，必须完成与仓库 workflow 对齐的自测与检查：
   - 本地或工作区内先执行覆盖本次改动的必要自测，至少覆盖仓库 workflow/CI 会检查到的相关路径
   - 如果仓库已有 PR checks / GitHub Actions / 其他 workflow，确认相关项已经成功，或至少已经拿到明确的通过结果
   - 不允许跳过 workflow 对齐检查就把状态推进到 `In Review`
10. 对于 `In Progress` / `Rework` 路径，完成上述 base branch 同步和 workflow 对齐验证后，更新工作台评论，提交并推送分支。
11. 对于 `In Progress` / `Rework` 路径，如果当前不存在 PR，则创建 PR；如果已存在 PR，则更新其描述、评论和关联信息。
12. 对于 `In Progress` / `Rework` 路径，将工单推进到 `In Review`，并在工作台评论中记录：
   - 分支名
   - 提交 SHA
   - PR 链接
   - base branch 以及已同步到的最新提交
   - 已执行的 workflow 对齐自测、对应命令和结果
   - PR / CI / workflow checks 的状态
   - 与 `OpenASE-PRD.md` 的对齐情况；如果本次实现引入了新的产品定义或行为调整，记录对应 PRD 更新
   - 验证结果
13. 如果当前状态是 `In Review`，则不要继续堆实现，而是直接审核现有 PR / 分支：
   - 检查 PR diff、历史 review、未解决线程与 CI 状态
   - 优先检查 `OpenASE-PRD.md`、issue 目标与最终实现是否一致，是否存在需求错配、遗漏约束、目录/依赖关系理解错误、接口语义偏差
   - 如果本次 issue 引入了新的需求定义或行为变化，检查 PRD 是否已经同步更新到位
   - 如果本次 issue 不涉及需求变更，检查代码是否偏离 PRD 既有定义
   - 检查本次改动对应的 workflow/CI 是否已经跑通，是否仍有失败、跳过、缺失或与本地自测不一致的地方
   - 只聚焦找阻塞合并的问题、行为回归、缺失验证和明显设计风险
   - 若没有需要阻塞的问题，则批准或给出明确通过结论，并将工单推进到 `Merging`
   - 若存在需要作者处理的问题，则提交 `change request`，在工作台评论中记录问题摘要，并将工单推进到 `Rework`
14. 当工单进入 `Merging`，先同步最新 `origin/main`，处理冲突并重跑受影响验证，再执行合并或仓库规定的落地主干动作。
15. 合并完成后，先确认主干落地结果可追溯到当前 issue / PR，然后手动关闭对应 GitHub issue。
16. 只有在 issue 已关闭后，才更新工作台评论，记录最终提交、PR、issue 关闭结果，并将工单推进到 `Done`。

## 反馈处理协议

当工单已绑定 PR 时，在回到 `In Review` 前必须清扫所有可执行反馈：

- 顶层 PR 评论
- 行内 review 评论
- review summary / request changes

并且必须满足以下分支同步约束：

- 以当前 PR 的 base branch 为准，把工作分支更新到该 base branch 的最新提交
- 解决同步过程中产生的冲突
- 重跑所有受影响验证
- 确认 PR 不处于 “out-of-date” / “behind base branch” 状态

并且必须满足以下 workflow / CI 约束：

- 回到 `In Review` 前，仓库要求的相关 workflow / CI checks 必须已经跑通，或至少已经有明确的通过结果
- 本地自测要与 workflow 覆盖范围对齐，不能只做与 CI 脱节的最小验证
- 如果 workflow 失败、未触发、仍在运行或结果与本地自测矛盾，不能直接推进到 `In Review`

每条反馈都必须满足以下之一：

- 代码或文档已经修改并解决；或
- 在对应线程中给出明确、有根据的回绝说明。

当反馈涉及需求、行为、接口语义、分层边界或文档描述时，还必须额外确认以下事项：

- `OpenASE-PRD.md` 与当前实现一致
- 若实现新增或改变了产品定义，PRD 已同步更新
- 若实现不应改变需求，代码没有脱离 PRD 原有约束
- 不允许带着已知的 PRD/实现错配进入 `In Review` 或 `Merging`

当工单处于 `In Review` 时，审核结论必须二选一，不允许停留在模糊状态：

- 无阻塞问题：给出 approve / 明确通过结论，并推进到 `Merging`
- 有阻塞问题：提交 `change request`，明确列出必须处理的问题，并推进到 `Rework`

## 合并约束

- 只有当工单状态为 `Merging` 时，才执行合并或主干落地动作。
- 合并前必须确认 PR 已通过必要检查，且分支相对 `origin/main` 没有未处理冲突。
- 如果仓库策略要求 squash / rebase / merge commit，遵循仓库既定策略，不自行发明新流程。
- 合并后必须确认 `main` 上的最终提交可追溯到该 issue 和对应 PR。
- 在把 Project 状态移动到 `Done` 之前，必须先手动关闭 GitHub issue。
- 不允许在 issue 仍然打开的情况下把工单推进到 `Done`。

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
