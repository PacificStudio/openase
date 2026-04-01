package builtin

import (
	"fmt"
	"strings"
)

// SkillTemplate describes a built-in skill scaffold.
type SkillTemplate struct {
	Name        string
	Title       string
	Description string
	Content     string
}

// Skills returns the built-in skill templates.
func Skills() []SkillTemplate {
	return cloneSkills(builtinSkills)
}

// SkillByName returns a built-in skill template by name.
func SkillByName(name string) (SkillTemplate, bool) {
	for _, item := range builtinSkills {
		if item.Name == name {
			return item, true
		}
	}

	return SkillTemplate{}, false
}

// IsBuiltinSkill reports whether a skill name belongs to the built-in set.
func IsBuiltinSkill(name string) bool {
	_, ok := SkillByName(name)
	return ok
}

func cloneSkills(items []SkillTemplate) []SkillTemplate {
	cloned := make([]SkillTemplate, len(items))
	copy(cloned, items)
	return cloned
}

func newSkillContent(name string, description string, body string) string {
	return fmt.Sprintf(
		"---\nname: %q\ndescription: %q\n---\n\n%s\n",
		name,
		description,
		strings.TrimSpace(body),
	)
}

var builtinSkills = []SkillTemplate{
	{
		Name:        "openase-platform",
		Title:       "OpenASE Platform Operations",
		Description: "Platform operations for tickets, projects, and runtime coordination inside OpenASE.",
		Content: newSkillContent("openase-platform", "Platform operations for tickets, projects, and runtime coordination inside OpenASE.", `
# OpenASE Platform Operations

优先使用工作区内注入的 wrapper：

`+"```bash"+`
./.openase/bin/openase ticket list --status-name Todo
`+"```"+`

这个 wrapper 就是 `+"`openase`"+` 二进制，已经带好当前工作区的 OpenASE 平台上下文。先用它，不要自己拼 URL 或猜接口。

## Execution Model

默认上下文来自这些环境变量：

- `+"`OPENASE_API_URL`"+`: OpenASE API 基址
- `+"`OPENASE_AGENT_TOKEN`"+`: 当前 agent token
- `+"`OPENASE_PROJECT_ID`"+`: 当前 project UUID
- `+"`OPENASE_TICKET_ID`"+`: 当前 ticket UUID

高频平台子命令会自动按下面顺序补上下文：

- project 作用域：`+"`--project-id`"+` -> `+"`OPENASE_PROJECT_ID`"+`
- ticket 作用域：位置参数 `+"`[ticket-id]`"+` -> `+"`--ticket-id`"+` -> `+"`OPENASE_TICKET_ID`"+`
- API 地址：`+"`--api-url`"+` -> `+"`OPENASE_API_URL`"+`
- Token：`+"`--token`"+` -> `+"`OPENASE_AGENT_TOKEN`"+`

重要限制：

- 大多数 ID 参数都要求 UUID，不接受 `+"`ASE-42`"+` 这种人类可读 ticket identifier。
- 默认输出是 pretty JSON；可以配合 `+"`--json`"+`、`+"`--jq`"+`、`+"`--template`"+` 做筛选。
- 平台失败时 CLI 会把 HTTP method、path、status 和 API error code 直接打出来，不需要自己猜。

## Safe Default Commands

这些是 agent 最应该先用的一层，语义稳定，适合 workflow / harness / workpad 直接调用。

### 1. 列当前项目工单

`+"```bash"+`
./.openase/bin/openase ticket list
./.openase/bin/openase ticket list --status-name Todo --priority high
./.openase/bin/openase ticket list --json tickets
`+"```"+`

能力：

- 调 `+"`GET /projects/{projectId}/tickets`"+`
- 支持 `+"`--status-name`"+` 多值过滤
- 支持 `+"`--priority`"+` 多值过滤

### 2. 创建工单

`+"```bash"+`
./.openase/bin/openase ticket create \
  --title "补充集成测试" \
  --description "拆分后续工单" \
  --priority high \
  --type task \
  --external-ref "BetterAndBetterII/openase#39"
`+"```"+`

能力：

- 调 `+"`POST /projects/{projectId}/tickets`"+`
- `+"`--title`"+` 必填
- 可选 `+"`--description`"+`、`+"`--priority`"+`、`+"`--type`"+`、`+"`--external-ref`"+`

### 3. 更新当前工单

`+"```bash"+`
./.openase/bin/openase ticket update --description "记录执行过程中的新发现"
./.openase/bin/openase ticket update --status-name Done
./.openase/bin/openase ticket update $OPENASE_TICKET_ID --external-ref "gh-123"
`+"```"+`

能力：

- 调 `+"`PATCH /tickets/{ticketId}`"+`
- 可更新 `+"`--title`"+`、`+"`--description`"+`、`+"`--external-ref`"+`
- 可更新状态：`+"`--status`"+` / `+"`--status-name`"+` / `+"`--status-id`"+`
- `+"`--status-name`"+` 和 `+"`--status-id`"+` 互斥
- 至少要给一个更新字段

### 4. 记录 usage / cost

`+"```bash"+`
./.openase/bin/openase ticket report-usage \
  --input-tokens 1200 \
  --output-tokens 340 \
  --cost-usd 0.0215
`+"```"+`

能力：

- 调 `+"`POST /tickets/{ticketId}/report-usage`"+`
- 记录的是增量，不是覆盖总量
- 至少要设置一个字段：`+"`--input-tokens`"+`、`+"`--output-tokens`"+`、`+"`--cost-usd`"+`

### 5. 管理 ticket comments

列评论：

`+"```bash"+`
./.openase/bin/openase ticket comment list
`+"```"+`

新建普通评论：

`+"```bash"+`
./.openase/bin/openase ticket comment create --body "记录当前阻塞"
./.openase/bin/openase ticket comment create --body-file /tmp/comment.md
`+"```"+`

能力：

- `+"`ticket comment list`"+` 调 `+"`GET /tickets/{ticketId}/comments`"+`
- `+"`ticket comment create`"+` 调 `+"`POST /tickets/{ticketId}/comments`"+`
- `+"`--body`"+` 和 `+"`--body-file`"+` 二选一

### 6. Upsert `+"`## Codex Workpad`"+`

`+"```bash"+`
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
`+"```"+`

能力：

- 幂等 upsert，不是盲目新建
- 会先列评论，再找到现有 `+"`## Codex Workpad`"+` 评论；有则更新，无则创建
- 如果正文没带 heading，CLI 会自动补上
- 这是当前 workflow 最推荐的持久化进度记录接口

### 7. 更新项目描述

`+"```bash"+`
./.openase/bin/openase project update --description "更新项目最新上下文"
`+"```"+`

能力：

- 调 `+"`PATCH /projects/{projectId}`"+`
- 当前高频 project 操作主要就是这个

### 8. 给项目注册 repo

`+"```bash"+`
./.openase/bin/openase project add-repo \
  --name "worker-tools" \
  --url "https://github.com/acme/worker-tools.git" \
  --default-branch main \
  --label go \
  --label backend
`+"```"+`

能力：

- 调 `+"`POST /projects/{projectId}/repos`"+`
- `+"`--name`"+`、`+"`--url`"+` 必填
- `+"`--default-branch`"+` 默认 `+"`main`"+`
- `+"`--label`"+` 可重复

## Full CLI Surface Beyond The Safe Subset

如果上面这些高频命令不够，`+"`openase`"+` 其实还有更广的 typed CLI，可直接走 OpenAPI 合约，不需要自己查源码再拼 HTTP。

常用 namespace 包括：

- `+"`openase ticket ...`"+`
- `+"`openase project ...`"+`
- `+"`openase workflow ...`"+`
- `+"`openase machine ...`"+`
- `+"`openase provider ...`"+`
- `+"`openase agent ...`"+`
- `+"`openase skill ...`"+`
- `+"`openase watch ...`"+` / `+"`openase stream ...`"+`

高价值例子：

`+"```bash"+`
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
`+"```"+`

这些 typed commands 的特点：

- 参数和字段来自 OpenAPI 合约，不是手写猜测
- 输出默认是 JSON
- 可以用 `+"`--json`"+` / `+"`--jq`"+` / `+"`--template`"+` 精简结果
- 很适合“先 inspect 再决定是否写操作”

## Raw API Escape Hatch

如果 typed command 还没有覆盖到，最后再用原始 passthrough：

`+"```bash"+`
./.openase/bin/openase api GET /api/v1/tickets/$OPENASE_TICKET_ID

./.openase/bin/openase api GET /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  --query status_name=Todo \
  --query priority=high

./.openase/bin/openase api POST /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  -f title="Follow-up" \
  -f workflow_id="550e8400-e29b-41d4-a716-446655440000"

./.openase/bin/openase api PATCH /api/v1/tickets/$OPENASE_TICKET_ID/comments/$COMMENT_ID \
  --input payload.json
`+"```"+`

规则：

- `+"`api METHOD PATH`"+` 是原始 HTTP passthrough
- `+"`-f/--field`"+` 用 `+"`key=value`"+` 组 JSON body
- `+"`--query`"+` 追加 query string
- `+"`--input`"+` 发送原始 request body；它不能和 `+"`-f`"+` 混用
- 这是缺少专门子命令时的最后兜底，不是第一选择

## Practical Guidance For Agents

- 先用 `+"`ticket list / get / detail`"+` 读上下文，再决定是否写。
- 写进度优先用 `+"`ticket comment workpad`"+`，不要不断创建普通评论。
- 要改 ticket 状态时优先传 `+"`--status-name`"+`，除非你已经拿到了准确 status UUID。
- 需要 probe 机器最新资源时，先 `+"`machine refresh-health`"+`，再看 `+"`machine resources`"+`。
- 需要更广能力时，先找 typed command；只有 typed command 不覆盖时才退到 `+"`openase api`"+`。
- 不要假设平台会接受 ticket identifier；绝大多数命令都要求 UUID。
`),
	},
	{
		Name:        "ticket-workpad",
		Title:       "Ticket Workpad",
		Description: "Maintain the persistent Workpad comment on the current ticket and use it as the execution log.",
		Content: newSkillContent("ticket-workpad", "Maintain the persistent Workpad comment on the current ticket and use it as the execution log.", `
# Ticket Workpad

Workpad 是当前工单唯一的持久化进度板。开始执行前先创建或更新它，之后在关键节点持续刷新。

调用 `+"`ticket comment workpad`"+` 时，不需要自己手动维护标题；只需要提供正文内容，让平台命令去复用或更新那条持久化 workpad 评论。

推荐写法：

`+"```bash"+`
cat <<'EOF' >/tmp/workpad.md
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
`+"```"+`

执行时遵循：

- 开工前先写第一版 workpad，不要先改代码再补记录。
- 每完成一个关键阶段就更新同一条评论，不要不断创建新评论。
- 至少持续维护 `+"`Plan`"+`、`+"`Progress`"+`、`+"`Validation`"+`、`+"`Notes`"+` 这些段落。
- 如果被阻塞，把阻塞原因和缺失前置条件写进 workpad，而不是静默退出。
`),
	},
	{
		Name:        "commit",
		Title:       "Conventional Commit",
		Description: "Write concise Conventional Commit messages that match the actual scope of the change.",
		Content: newSkillContent("commit", "Write concise Conventional Commit messages that match the actual scope of the change.", `
# Conventional Commit

在提交前整理变更范围，并使用 Conventional Commit 风格：

- `+"`feat(scope): ...`"+` 用于可见能力新增
- `+"`fix(scope): ...`"+` 用于缺陷修复
- `+"`refactor(scope): ...`"+` 用于无行为变化的重构
- `+"`test(scope): ...`"+` 用于测试补充

避免把多个不相关改动塞进同一个提交说明。标题写结果，不写过程。
`),
	},
	{
		Name:        "pull",
		Title:       "Safe Pull",
		Description: "Sync with the latest remote branch before coding or pushing, keeping history linear.",
		Content: newSkillContent("pull", "Sync with the latest remote branch before coding or pushing, keeping history linear.", `
# Safe Pull

开始工作前先同步远端主干：

1. 确认当前分支和工作区状态
2. 获取远端最新提交
3. 使用快进或 rebase 方式同步
4. 如果有冲突，先解决并重新验证受影响范围

不要在不理解影响的情况下覆盖本地文件。
`),
	},
	{
		Name:        "push",
		Title:       "Safe Push",
		Description: "Push verified changes carefully and avoid destructive history rewrites.",
		Content: newSkillContent("push", "Push verified changes carefully and avoid destructive history rewrites.", `
# Safe Push

推送前确认：

- 相关验证已完成
- 本地分支已和远端同步
- 不使用破坏性强推覆盖他人历史

推送失败时先同步远端，再解决冲突后重新验证。
`),
	},
	{
		Name:        "create-pr",
		Title:       "Create PR",
		Description: "Prepare a crisp pull request summary with scope, validation, and rollout notes.",
		Content: newSkillContent("create-pr", "Prepare a crisp pull request summary with scope, validation, and rollout notes.", `
# Create PR

当流程需要 Pull Request 时，PR 描述至少包含：

- 变更目的
- 主要实现点
- 验证命令和结果
- 风险与回滚方式

标题要和提交主题一致，避免模糊表述。
`),
	},
	{
		Name:        "land",
		Title:       "Land Safely",
		Description: "Land a reviewed change only after CI and branch-state checks pass.",
		Content: newSkillContent("land", "Land a reviewed change only after CI and branch-state checks pass.", `
# Land Safely

合并前确认：

1. 当前分支已同步最新主干
2. 关键测试和 CI 通过
3. 已处理 review comment
4. 合并后不留下临时调试代码

优先保持线性、可追踪的历史。
`),
	},
	{
		Name:        "review-code",
		Title:       "Review Code",
		Description: "Review behavior, risk, and test coverage before style nits.",
		Content: newSkillContent("review-code", "Review behavior, risk, and test coverage before style nits.", `
# Review Code

代码审查优先关注：

- 行为回归
- 边界条件
- 权限 / 安全风险
- 数据一致性
- 测试缺口

先给出 findings，再补充总结。
`),
	},
	{
		Name:        "write-test",
		Title:       "Write Tests",
		Description: "Design focused tests that cover the real behavior contract and key edge cases.",
		Content: newSkillContent("write-test", "Design focused tests that cover the real behavior contract and key edge cases.", `
# Write Tests

写测试时优先覆盖：

- 新增逻辑的主路径
- 失败路径和边界输入
- 回归点

避免只为覆盖率而写和实现细节强耦合的脆弱测试。
`),
	},
	{
		Name:        "security-scan",
		Title:       "Security Scan",
		Description: "Use a lightweight security checklist for auth, secrets, injection, and dependency risk.",
		Content: newSkillContent("security-scan", "Use a lightweight security checklist for auth, secrets, injection, and dependency risk.", `
# Security Scan

安全检查至少覆盖：

- 认证与授权边界
- 命令 / SQL / 模板注入风险
- 敏感信息泄露
- 依赖版本与已知漏洞
- 默认配置是否过宽

发现问题时给出可复现路径和修复建议。
`),
	},
	{
		Name:        "install-claude-code",
		Title:       "Install Claude Code",
		Description: "Install Claude Code on a target machine and verify the CLI is available for remote execution.",
		Content: newSkillContent("install-claude-code", "Install Claude Code on a target machine and verify the CLI is available for remote execution.", `
# Install Claude Code

目标：让目标机器具备可用的 `+"`claude`"+` 命令，并记录安装结果。

执行时遵循：

- 先确认当前系统类型、包管理器和是否已安装 `+"`claude`"+`。
- 使用官方支持的安装方式完成安装，避免下载来源不明的二进制。
- 安装后至少验证 `+"`claude --version`"+`，并记录可执行路径。
- 如果还需要登录或额外认证，明确记录当前状态和缺失前置条件。

不要把令牌或凭据写入仓库。
`),
	},
	{
		Name:        "install-codex",
		Title:       "Install Codex CLI",
		Description: "Install the Codex CLI on a target machine and verify it can start successfully.",
		Content: newSkillContent("install-codex", "Install the Codex CLI on a target machine and verify it can start successfully.", `
# Install Codex CLI

目标：让目标机器具备可用的 `+"`codex`"+` 命令，并验证 CLI 能正常启动。

执行时遵循：

- 先检查 `+"`codex`"+` 是否已存在以及当前版本。
- 使用官方支持的安装方式安装或升级 Codex CLI。
- 安装后验证 `+"`codex --version`"+`，必要时补充最小认证检查。
- 如果网络、Python、Node 或系统依赖阻塞安装，记录准确阻塞点，不要留下半安装状态。
`),
	},
	{
		Name:        "setup-git",
		Title:       "Setup Git",
		Description: "Install or repair git plus the minimum identity and credential configuration needed for agent work.",
		Content: newSkillContent("setup-git", "Install or repair git plus the minimum identity and credential configuration needed for agent work.", `
# Setup Git

目标：让目标机器具备可用的 `+"`git`"+`，并补齐最小身份配置。

执行时遵循：

- 检查 `+"`git --version`"+` 是否可用；如果不可用，先安装 git。
- 检查 `+"`git config --global user.name`"+` 和 `+"`git config --global user.email`"+`；缺失时按工单上下文补齐。
- 仅在确有凭据问题时修复 git 认证，避免覆盖已有可用配置。
- 最后用非破坏性命令确认 git 基础能力可用，并记录生效配置。
`),
	},
	{
		Name:        "setup-gh-cli",
		Title:       "Setup GitHub CLI",
		Description: "Install or repair the GitHub CLI and confirm authentication status on the target machine.",
		Content: newSkillContent("setup-gh-cli", "Install or repair the GitHub CLI and confirm authentication status on the target machine.", `
# Setup GitHub CLI

目标：让目标机器具备可用的 `+"`gh`"+`，并确认 GitHub 认证状态。

执行时遵循：

- 检查 `+"`gh --version`"+` 和 `+"`gh auth status`"+` 的当前输出。
- 如果 `+"`gh`"+` 缺失，使用官方支持的安装方式安装。
- 如果 `+"`gh`"+` 已安装但未认证，补齐认证并再次验证状态。
- 认证失败时记录准确原因，例如网络、令牌缺失或主机不可达。

不要把明文令牌写入 shell 历史或仓库文件。
`),
	},
}
