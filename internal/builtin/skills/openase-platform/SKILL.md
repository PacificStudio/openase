---
name: "openase-platform"
description: "Platform operations for tickets, projects, and runtime coordination inside OpenASE."
---

# OpenASE Platform Operations

优先使用工作区内注入的 wrapper：

```bash
./.openase/bin/openase ticket list --status-name Todo
```

这个 wrapper 就是 `openase` 二进制，已经带好当前工作区的 OpenASE 平台上下文。先用它，不要自己拼 URL、猜接口，也不要绕过平台直接改数据库或伪造平台状态。

## What OpenASE Controls

OpenASE 不是一个“帮你跑命令的小工具”，而是一个 issue-driven automated software engineering 平台。按 PRD，它的职责至少包括：

- 管项目：项目描述、项目状态、项目下的 repo、workflow、skills、scheduled jobs
- 管工单：ticket 生命周期、状态流转、评论原语、usage/cost、外部关联
- 管执行：agent、provider、machine、runtime、编排循环
- 管自治：允许 agent 在受控 scope 内反向操作平台，形成“领工单 -> 执行 -> 回写平台 -> 拆后续任务”的闭环
- 管审计：所有平台写操作都进入 API / ActivityEvent / timeline，可追溯是谁改的

对 agent 来说，`openase` 的核心用途不是“查信息”，而是“在平台允许的边界内读写真实控制面状态”。你创建的 ticket、更新的 project 描述、注册的 repo、追加的评论原语，都会影响后续调度、UI、审计和其他 agent 的上下文。

## Mental Model For Agents

把 `openase` 当成“当前工程项目的控制面 API”。

- 代码仓库只是工作区，不是真正的任务系统
- OpenASE 里的 ticket / project / workflow / skill / machine 才是控制面实体
- 修改平台状态，优先用 `openase` CLI；不要试图靠改本地文件去“间接表达”平台状态
- 先读再写：先列出现状，再做最小必要修改
- 如果越权，平台会直接返回 `403`；这通常说明当前 harness 没授予对应 `platform_access`

## Execution Model

当前 runtime 会额外注入一段 capability contract，明确告诉你本次会话真实可用的 principal kind、scopes 和环境变量。以那段 runtime contract 为准。

常见环境变量包括：

- `OPENASE_API_URL`: OpenASE API 基址
- `OPENASE_AGENT_TOKEN`: 当前 agent token
- `OPENASE_PROJECT_ID`: 当前 project UUID
- `OPENASE_TICKET_ID`: 当前 ticket UUID；只在 ticket runtime 或 ticket-focused Project AI 中出现
- `OPENASE_CONVERSATION_ID`: 当前 project conversation UUID；Project AI 会话可用
- `OPENASE_PRINCIPAL_KIND`: 当前 principal kind，例如 `ticket_agent` 或 `project_conversation`
- `OPENASE_AGENT_SCOPES`: 当前 token scopes，逗号分隔

高频平台子命令会自动按下面顺序补上下文：

- project 作用域：`--project-id` -> `OPENASE_PROJECT_ID`
- ticket 作用域：位置参数 `[ticket-id]` -> `--ticket-id` -> `OPENASE_TICKET_ID`
- API 地址：`--api-url` -> `OPENASE_API_URL`
- Token：`--token` -> `OPENASE_AGENT_TOKEN`

重要限制：

- 大多数 ID 参数都要求 UUID，不接受 `ASE-42` 这种人类可读 ticket identifier
- 默认输出是 JSON；可以配合 `--json`、`--jq`、`--template` 做筛选
- 平台失败时 CLI 会把 HTTP method、path、status 和 API error code 直接打出来，不需要自己猜
- Token 是短期且带 scope 的；不是所有 workflow 都能改 project、repo、scheduled-job
- shared wrapper flags 同时接受 kebab-case 和 snake_case，例如 `--status-name` / `--status_name`、`--body-file` / `--body_file`

## Top-Level Commands

下面是当前源码里 `openase` 的一级子命令完整列表。不是所有命令都适合 agent；前几组是你最常碰到的。

### Agent / API 操作面

- `api`: raw HTTP passthrough，任何已暴露 API 的兜底入口
- `ticket`: shared platform wrapper 负责高频 ticket 读写；非重叠的 detail/run/dependency/external-link 等扩展子命令仍直接走 OpenAPI
- `status`: ticket status board 管理
- `chat`: ephemeral chat 与 project conversation
- `project`: shared platform wrapper 负责 update/add-repo；list/get/create/delete 仍直接走 OpenAPI
- `repo`: project repo、GitHub repo 发现、ticket repo scopes
- `workflow`: workflow 与 harness 读写
- `scheduled-job`: 定时任务管理
- `machine`: 机器注册、探测、资源查看
- `provider`: provider 查看与配置
- `agent`: agent 查看、暂停/恢复、输出与步骤读取
- `activity`: project activity timeline 读取
- `channel`: notification channel 管理与 test
- `notification-rule`: notification rule 管理
- `skill`: skill 查看、更新、绑定、refresh
- `watch`: SSE watch 流
- `stream`: SSE stream 流

### Service / Control Plane 运维面

- `serve`: 仅启动 HTTP API 服务
- `orchestrate`: 仅启动编排循环
- `all-in-one`: 同进程启动 API + orchestrator
- `up`: 本地启动 OpenASE 服务
- `setup`: 初始化本地运行环境
- `down`: 停止本地服务
- `restart`: 重启本地服务
- `logs`: 查看本地服务日志
- `doctor`: 本地环境自检

### Admin / Schema / Utility

- `issue-agent-token`: 签发 agent token
- `openapi`: 导出或查看 OpenAPI 相关产物
- `version`: 查看版本

一般来说，agent 在工作区内最常用的是：

- `ticket`
- `status`
- `chat`
- `project`
- `repo`
- `workflow`
- `activity`
- `scheduled-job`
- `machine`
- `provider`
- `agent`
- `channel`
- `notification-rule`
- `skill`
- `watch` / `stream`
- `api`

而 `serve` / `orchestrate` / `up` / `down` / `restart` / `issue-agent-token` 这些更偏平台运维或控制面启动，不是普通 ticket 执行的第一选择。

## Safe Default Commands

这些是 agent 最应该先用的一层，语义稳定，适合 workflow / harness 直接调用。

### 1. 列当前项目工单

```bash
./.openase/bin/openase ticket list
./.openase/bin/openase ticket list --status-name Todo --priority high
./.openase/bin/openase ticket list --json tickets
```

能力：

- 调 `GET /projects/{projectId}/tickets`
- 支持 `--status-name` 多值过滤
- 支持 `--priority` 多值过滤

### 2. 创建工单

```bash
./.openase/bin/openase ticket create \
  --title "补充集成测试" \
  --description "拆分后续工单" \
  --priority high \
  --type task \
  --external-ref "PacificStudio/openase#39"
```

能力：

- 调 `POST /projects/{projectId}/tickets`
- `--title` 必填
- 可选 `--description`、`--priority`、`--type`、`--external-ref`

适用场景：

- 发现还需要 follow-up ticket
- 需要把超出当前范围的工作拆出去
- 需要把安全、测试、部署等后续工作显式挂回平台

### 3. 更新当前工单

```bash
./.openase/bin/openase ticket update --description "记录执行过程中的新发现"
./.openase/bin/openase ticket update --status-name Done
./.openase/bin/openase ticket update $OPENASE_TICKET_ID --external-ref "gh-123"
```

能力：

- 调 `PATCH /tickets/{ticketId}`
- 可更新 `--title`、`--description`、`--external-ref`
- 可更新状态：`--status` / `--status-name` / `--status-id`
- `--status-name` 和 `--status-id` 互斥
- 至少要给一个更新字段

### 4. 记录 usage / cost

```bash
./.openase/bin/openase ticket report-usage \
  --input-tokens 1200 \
  --output-tokens 340 \
  --cost-usd 0.0215
```

能力：

- 调 `POST /tickets/{ticketId}/report-usage`
- 记录的是增量，不是覆盖总量
- 至少要设置一个字段：`--input-tokens`、`--output-tokens`、`--cost-usd`

### 5. 管理 ticket comments

列评论：

```bash
./.openase/bin/openase ticket comment list
```

新建普通评论：

```bash
./.openase/bin/openase ticket comment create --body "记录当前阻塞"
./.openase/bin/openase ticket comment create --body-file /tmp/comment.md
```

更新已有评论：

```bash
./.openase/bin/openase ticket comment update $OPENASE_TICKET_ID $COMMENT_ID --body-file /tmp/comment.md
```

能力：

- `ticket comment list` 调 `GET /tickets/{ticketId}/comments`
- `ticket comment create` 调 `POST /tickets/{ticketId}/comments`
- `ticket comment update` 调 `PATCH /tickets/{ticketId}/comments/{commentId}`
- `--body` 和 `--body-file` 二选一

`openase-platform` 在这里提供的是 comment 原语，不直接承载 workpad 语义。需要维护持久化 workpad 时，使用单独绑定到 workflow 的 `ticket-workpad` skill；它建立在这里的 comment `list/create/update` 基座之上。

### 6. 更新项目描述

```bash
./.openase/bin/openase project update --description "更新项目最新上下文"
```

能力：

- 调 `PATCH /projects/{projectId}`
- 当前高频 project 操作主要就是这个

适用场景：

- 产品经理或研究类角色把调研结果写回项目
- 当前 ticket 执行中发现需要把长期背景更新到项目级描述

### 7. 管理项目 status board

```bash
./.openase/bin/openase status list $OPENASE_PROJECT_ID
./.openase/bin/openase status create $OPENASE_PROJECT_ID \
  --name "QA" \
  --stage started \
  --color "#FF00AA"
./.openase/bin/openase status update $STATUS_ID --name "Ready for QA"
```

能力：

- `status list` 调 `GET /projects/{projectId}/statuses`
- `status create` 调 `POST /projects/{projectId}/statuses`
- `status update` 调 `PATCH /statuses/{statusId}`
- `status delete` / `status reset` 也已提供 typed CLI

适用场景：

- 需要先读项目当前 status board，再决定是否流转 ticket
- 项目管理员调整状态命名、排序或默认状态

### 8. 给项目注册 repo

```bash
./.openase/bin/openase repo create $OPENASE_PROJECT_ID \
  --name "worker-tools" \
  --url "https://github.com/acme/worker-tools.git" \
  --default-branch main \
  --label go \
  --label backend
```

能力：

- 调 `POST /projects/{projectId}/repos`
- `--name`、`--url` 必填
- `--default-branch` 默认 `main`
- `--label` 可重复

兼容说明：

```bash
./.openase/bin/openase project add-repo \
  --name "worker-tools" \
  --url "https://github.com/acme/worker-tools.git" \
  --default-branch main
```

旧的 `project add-repo` 入口仍然可用，但新的 typed CLI 和后续文档优先使用 `repo create`，因为它更直接表达了 repo 是 project 下的独立控制面实体。

### 9. 管 project conversation / Project AI

```bash
./.openase/bin/openase chat conversation list --project-id $OPENASE_PROJECT_ID
./.openase/bin/openase chat conversation get $OPENASE_CONVERSATION_ID
./.openase/bin/openase chat conversation entries $OPENASE_CONVERSATION_ID
./.openase/bin/openase chat conversation turn $OPENASE_CONVERSATION_ID --message "继续处理上一个问题"
./.openase/bin/openase chat conversation watch $OPENASE_CONVERSATION_ID
```

能力：

- `chat conversation list` 调 `GET /chat/conversations`
- `chat conversation turn` 调 `POST /chat/conversations/{conversationId}/turns`
- `chat conversation watch` 打开 conversation 事件流
- 支持 interrupt 响应、action proposal 执行、runtime close

适用场景：

- 在 Project AI / project conversation principal 下检查和续跑会话
- 需要把 conversation transcript 或 workspace diff 当作控制面状态读取

### 10. 读项目 activity timeline

```bash
./.openase/bin/openase activity list $OPENASE_PROJECT_ID
./.openase/bin/openase activity list $OPENASE_PROJECT_ID --json events
```

能力：

- 调 `GET /projects/{projectId}/activity`
- 读取项目级业务时间线，包括 ticket 变更、workflow 变更、runtime lifecycle 和其他审计事件

适用场景：

- 想快速确认最近是谁修改了 workflow、status、project description
- 需要补全当前 ticket 之外的项目上下文
- 想确认 runtime 或 orchestration 的最近业务事件是否已经落盘

### 11. 查 ticket runs / retry / external links

```bash
./.openase/bin/openase ticket run list $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase ticket run get $OPENASE_PROJECT_ID $OPENASE_TICKET_ID $RUN_ID
./.openase/bin/openase ticket retry-resume $OPENASE_TICKET_ID
./.openase/bin/openase ticket external-link add $OPENASE_TICKET_ID \
  --title "PR 482" \
  --url "https://github.com/acme/repo/pull/482"
```

能力：

- `ticket run list` 调 `GET /projects/{projectId}/tickets/{ticketId}/runs`
- `ticket run get` 调 `GET /projects/{projectId}/tickets/{ticketId}/runs/{runId}`
- `ticket retry-resume` 调 `POST /tickets/{ticketId}/retry/resume`
- `ticket external-link add/delete` 管理 ticket 的外部关联

适用场景：

- 想确认某个 ticket 的最近执行历史、失败原因、重试链路
- 需要在 retryable failure 之后显式恢复执行
- 需要把 PR、issue、文档、事故单等外部系统对象挂回 ticket

### 12. 查 workflow harness history / variables / validate

```bash
./.openase/bin/openase workflow harness history $WORKFLOW_ID
./.openase/bin/openase workflow harness variables
./.openase/bin/openase workflow harness validate --input /tmp/harness.json
```

能力：

- `workflow harness history` 调 `GET /workflows/{workflowId}/harness/history`
- `workflow harness variables` 调 `GET /harness/variables`
- `workflow harness validate` 调 `POST /harness/validate`

适用场景：

- 想先看 harness 最近几次修改，再决定是否继续改
- 想确认模板里可以引用哪些变量
- 想在真正提交 harness 更新前先做语义校验

## Full CLI Surface Beyond The Safe Subset

如果上面这些高频命令不够，`openase` 其实还有更广的 typed CLI，可直接走 OpenAPI 合约，不需要自己查源码再拼 HTTP。

常用 namespace 包括：

- `openase ticket ...`
- `openase status ...`
- `openase chat ...`
- `openase project ...`
- `openase repo ...`
- `openase workflow ...`
- `openase scheduled-job ...`
- `openase machine ...`
- `openase provider ...`
- `openase agent ...`
- `openase activity ...`
- `openase channel ...`
- `openase notification-rule ...`
- `openase skill ...`
- `openase watch ...`
- `openase stream ...`

高价值例子：

```bash
./.openase/bin/openase ticket get $OPENASE_TICKET_ID
./.openase/bin/openase ticket detail $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase status list $OPENASE_PROJECT_ID
./.openase/bin/openase chat conversation get $OPENASE_CONVERSATION_ID
./.openase/bin/openase repo list $OPENASE_PROJECT_ID
./.openase/bin/openase workflow list $OPENASE_PROJECT_ID
./.openase/bin/openase workflow harness get $WORKFLOW_ID
./.openase/bin/openase workflow harness history $WORKFLOW_ID
./.openase/bin/openase workflow harness variables
./.openase/bin/openase activity list $OPENASE_PROJECT_ID
./.openase/bin/openase ticket run list $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase ticket retry-resume $OPENASE_TICKET_ID
./.openase/bin/openase scheduled-job list $OPENASE_PROJECT_ID
./.openase/bin/openase machine refresh-health $MACHINE_ID
./.openase/bin/openase machine resources $MACHINE_ID
./.openase/bin/openase provider list $OPENASE_ORG_ID --json providers
./.openase/bin/openase agent output $OPENASE_PROJECT_ID $AGENT_ID
./.openase/bin/openase channel list $OPENASE_ORG_ID
./.openase/bin/openase notification-rule list $OPENASE_PROJECT_ID
./.openase/bin/openase skill list $OPENASE_PROJECT_ID
./.openase/bin/openase watch project $OPENASE_PROJECT_ID
```

这些 additional typed commands 的特点：

- 参数和字段来自 OpenAPI 合约，不是手写猜测
- 输出默认是 JSON
- 可以用 `--json` / `--jq` / `--template` 精简结果
- 很适合“先 inspect 再决定是否写操作”
- 少数 body 字段会和 CLI 输出 flag 重名；例如 `notification-rule create/update` 的 `template` 字段应改用 `-f template=...` 或 `--input payload.json`

## Raw API Escape Hatch

如果 typed command 还没有覆盖到，最后再用原始 passthrough：

```bash
./.openase/bin/openase api GET /api/v1/tickets/$OPENASE_TICKET_ID

./.openase/bin/openase api GET /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  --query status_name=Todo \
  --query priority=high

./.openase/bin/openase api POST /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  -f title="Follow-up" \
  -f workflow_id="550e8400-e29b-41d4-a716-446655440000"

./.openase/bin/openase api PATCH /api/v1/tickets/$OPENASE_TICKET_ID/comments/$COMMENT_ID \
  --input payload.json
```

规则：

- `api METHOD PATH` 是原始 HTTP passthrough
- `-f/--field` 用 `key=value` 组 JSON body
- `--query` 追加 query string
- `--input` 发送原始 request body；它不能和 `-f` 混用
- 这是缺少专门子命令时的最后兜底，不是第一选择

## Platform Boundaries And Safety

使用这个 skill 时，默认要遵守这些边界：

- 只能操作当前项目上下文，不要假设自己能跨项目写数据
- 优先维护当前 ticket 的 comment / description 等平台状态，不要到处留下零散状态
- 拆 follow-up ticket 可以，但不要为了“显得主动”无限拆单
- 修改 project、repo、scheduled-job、workflow 前，先确认当前角色真的需要且当前 token scope 允许
- 遇到 `403` 时，先检查能力边界，不要换别的命令绕过平台

## Practical Guidance For Agents

- 先用 `ticket list / get / detail` 读上下文，再决定是否写
- 需要持久化跨 runtime 的执行日志时，依赖单独的 `ticket-workpad` skill；这个 platform skill 只提供底层 comment 原语
- 要改 ticket 状态时优先传 `--status-name`，除非你已经拿到了准确 status UUID
- 需要 probe 机器最新资源时，先 `machine refresh-health`，再看 `machine resources`
- 需要更广能力时，先找 typed command；只有 typed command 不覆盖时才退到 `openase api`
- 不要假设平台会接受 ticket identifier；绝大多数命令都要求 UUID
