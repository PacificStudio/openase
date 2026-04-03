---
name: "openase-platform"
description: "Platform operations for tickets, projects, and runtime coordination inside OpenASE."
---

# OpenASE Platform Operations

优先使用工作区内注入的 wrapper：

```bash
./.openase/bin/openase ticket list --status-name Todo
```

这个 wrapper 就是 `openase` 二进制，已经带好当前工作区的 OpenASE 平台上下文。先用它，不要自己拼 URL 或猜接口。

## Execution Model

默认上下文来自这些环境变量：

- `OPENASE_API_URL`: OpenASE API 基址
- `OPENASE_AGENT_TOKEN`: 当前 agent token
- `OPENASE_PROJECT_ID`: 当前 project UUID
- `OPENASE_TICKET_ID`: 当前 ticket UUID

高频平台子命令会自动按下面顺序补上下文：

- project 作用域：`--project-id` -> `OPENASE_PROJECT_ID`
- ticket 作用域：位置参数 `[ticket-id]` -> `--ticket-id` -> `OPENASE_TICKET_ID`
- API 地址：`--api-url` -> `OPENASE_API_URL`
- Token：`--token` -> `OPENASE_AGENT_TOKEN`

重要限制：

- 大多数 ID 参数都要求 UUID，不接受 `ASE-42` 这种人类可读 ticket identifier。
- 默认输出是 pretty JSON；可以配合 `--json`、`--jq`、`--template` 做筛选。
- 平台失败时 CLI 会把 HTTP method、path、status 和 API error code 直接打出来，不需要自己猜。

## Safe Default Commands

这些是 agent 最应该先用的一层，语义稳定，适合 workflow / harness / workpad 直接调用。

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

能力：

- `ticket comment list` 调 `GET /tickets/{ticketId}/comments`
- `ticket comment create` 调 `POST /tickets/{ticketId}/comments`
- `--body` 和 `--body-file` 二选一

### 6. Upsert `## Codex Workpad`

```bash
cat <<'EOF' >/tmp/workpad.md
## Codex Workpad

Environment
- <host>:<abs-workdir>@<short-sha>

Plan
- step 1
- step 2

Progress
- inspecting current implementation

Validation
- not run yet

Notes
- assumptions or blockers
EOF

./.openase/bin/openase ticket comment workpad --body-file /tmp/workpad.md
```

能力：

- 幂等 upsert，不是盲目新建
- 会先列评论，再找到现有 `## Codex Workpad` 评论；有则更新，无则创建
- 如果正文没带 heading，CLI 会自动补上
- 这是当前 workflow 最推荐的持久化进度记录接口

### 7. 更新项目描述

```bash
./.openase/bin/openase project update --description "更新项目最新上下文"
```

能力：

- 调 `PATCH /projects/{projectId}`
- 当前高频 project 操作主要就是这个

### 8. 给项目注册 repo

```bash
./.openase/bin/openase project add-repo \
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

## Full CLI Surface Beyond The Safe Subset

如果上面这些高频命令不够，`openase` 其实还有更广的 typed CLI，可直接走 OpenAPI 合约，不需要自己查源码再拼 HTTP。

常用 namespace 包括：

- `openase ticket ...`
- `openase project ...`
- `openase workflow ...`
- `openase machine ...`
- `openase provider ...`
- `openase agent ...`
- `openase skill ...`
- `openase watch ...` / `openase stream ...`

高价值例子：

```bash
./.openase/bin/openase ticket get $OPENASE_TICKET_ID
./.openase/bin/openase ticket detail $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase workflow list $OPENASE_PROJECT_ID
./.openase/bin/openase workflow harness get $WORKFLOW_ID
./.openase/bin/openase machine refresh-health $MACHINE_ID
./.openase/bin/openase machine resources $MACHINE_ID
./.openase/bin/openase provider list $OPENASE_ORG_ID --json providers
./.openase/bin/openase agent output $OPENASE_PROJECT_ID $AGENT_ID
./.openase/bin/openase skill list $OPENASE_PROJECT_ID
./.openase/bin/openase watch activity $OPENASE_PROJECT_ID
```

这些 typed commands 的特点：

- 参数和字段来自 OpenAPI 合约，不是手写猜测
- 输出默认是 JSON
- 可以用 `--json` / `--jq` / `--template` 精简结果
- 很适合“先 inspect 再决定是否写操作”

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

## Practical Guidance For Agents

- 先用 `ticket list / get / detail` 读上下文，再决定是否写。
- 写进度优先用 `ticket comment workpad`，不要不断创建普通评论。
- 要改 ticket 状态时优先传 `--status-name`，除非你已经拿到了准确 status UUID。
- 需要 probe 机器最新资源时，先 `machine refresh-health`，再看 `machine resources`。
- 需要更广能力时，先找 typed command；只有 typed command 不覆盖时才退到 `openase api`。
- 不要假设平台会接受 ticket identifier；绝大多数命令都要求 UUID。
