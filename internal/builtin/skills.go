package builtin

import "strings"

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

var builtinSkills = []SkillTemplate{
	{
		Name:        "openase-platform",
		Title:       "OpenASE Platform Operations",
		Description: "Platform operations for tickets, projects, and runtime coordination inside OpenASE.",
		Content: strings.TrimSpace(`
# OpenASE Platform Operations

使用仓库内置的 wrapper 调用平台能力：

`+"```bash"+`
./.openase/bin/openase ticket list --status-name Todo
./.openase/bin/openase ticket create --title "补充集成测试" --description "拆分后续工单"
./.openase/bin/openase ticket update --description "记录执行过程中的新发现"
./.openase/bin/openase ticket comment create --body "记录当前阻塞"
./.openase/bin/openase ticket comment workpad --body-file /tmp/workpad.md
./.openase/bin/openase project update --description "更新项目最新上下文"
./.openase/bin/openase project add-repo --name "worker-tools" --url "https://github.com/acme/worker-tools.git"
`+"```"+`

## Notes

- 该 wrapper 透传到已安装的 `+"`openase`"+` 二进制，并自动使用工作区中注入的 `+"`OPENASE_API_URL`"+`、`+"`OPENASE_AGENT_TOKEN`"+`、`+"`OPENASE_PROJECT_ID`"+`、`+"`OPENASE_TICKET_ID`"+`。
- 当前最小实现覆盖已经落地的 agent platform API：工单列表 / 创建 / 更新 / 评论 workpad，以及项目描述更新和 Repo 注册。
- 如果需要更底层调试，可直接对 `+"`$OPENASE_API_URL`"+` 发 HTTP 请求，并带上 `+"`Authorization: Bearer $OPENASE_AGENT_TOKEN`"+`。
`) + "\n",
	},
	{
		Name:        "ticket-workpad",
		Title:       "Ticket Workpad",
		Description: "Maintain a single ## Codex Workpad comment on the current ticket and use it as the execution log.",
		Content: strings.TrimSpace(`
# Ticket Workpad

`+"`## Codex Workpad`"+` 是当前工单唯一的持久化进度板。开始执行前先创建或更新它，之后在关键节点持续刷新。

推荐写法：

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

执行时遵循：

- 开工前先写第一版 workpad，不要先改代码再补记录。
- 每完成一个关键阶段就更新同一条评论，不要不断创建新评论。
- 至少持续维护 `+"`Plan`"+`、`+"`Progress`"+`、`+"`Validation`"+`、`+"`Notes`"+` 这些段落。
- 如果被阻塞，把阻塞原因和缺失前置条件写进 workpad，而不是静默退出。
`) + "\n",
	},
	{
		Name:        "commit",
		Title:       "Conventional Commit",
		Description: "Write concise Conventional Commit messages that match the actual scope of the change.",
		Content: strings.TrimSpace(`
# Conventional Commit

在提交前整理变更范围，并使用 Conventional Commit 风格：

- `+"`feat(scope): ...`"+` 用于可见能力新增
- `+"`fix(scope): ...`"+` 用于缺陷修复
- `+"`refactor(scope): ...`"+` 用于无行为变化的重构
- `+"`test(scope): ...`"+` 用于测试补充

避免把多个不相关改动塞进同一个提交说明。标题写结果，不写过程。
`) + "\n",
	},
	{
		Name:        "pull",
		Title:       "Safe Pull",
		Description: "Sync with the latest remote branch before coding or pushing, keeping history linear.",
		Content: strings.TrimSpace(`
# Safe Pull

开始工作前先同步远端主干：

1. 确认当前分支和工作区状态
2. 获取远端最新提交
3. 使用快进或 rebase 方式同步
4. 如果有冲突，先解决并重新验证受影响范围

不要在不理解影响的情况下覆盖本地文件。
`) + "\n",
	},
	{
		Name:        "push",
		Title:       "Safe Push",
		Description: "Push verified changes carefully and avoid destructive history rewrites.",
		Content: strings.TrimSpace(`
# Safe Push

推送前确认：

- 相关验证已完成
- 本地分支已和远端同步
- 不使用破坏性强推覆盖他人历史

推送失败时先同步远端，再解决冲突后重新验证。
`) + "\n",
	},
	{
		Name:        "create-pr",
		Title:       "Create PR",
		Description: "Prepare a crisp pull request summary with scope, validation, and rollout notes.",
		Content: strings.TrimSpace(`
# Create PR

当流程需要 Pull Request 时，PR 描述至少包含：

- 变更目的
- 主要实现点
- 验证命令和结果
- 风险与回滚方式

标题要和提交主题一致，避免模糊表述。
`) + "\n",
	},
	{
		Name:        "land",
		Title:       "Land Safely",
		Description: "Land a reviewed change only after CI and branch-state checks pass.",
		Content: strings.TrimSpace(`
# Land Safely

合并前确认：

1. 当前分支已同步最新主干
2. 关键测试和 CI 通过
3. 已处理 review comment
4. 合并后不留下临时调试代码

优先保持线性、可追踪的历史。
`) + "\n",
	},
	{
		Name:        "review-code",
		Title:       "Review Code",
		Description: "Review behavior, risk, and test coverage before style nits.",
		Content: strings.TrimSpace(`
# Review Code

代码审查优先关注：

- 行为回归
- 边界条件
- 权限 / 安全风险
- 数据一致性
- 测试缺口

先给出 findings，再补充总结。
`) + "\n",
	},
	{
		Name:        "write-test",
		Title:       "Write Tests",
		Description: "Design focused tests that cover the real behavior contract and key edge cases.",
		Content: strings.TrimSpace(`
# Write Tests

写测试时优先覆盖：

- 新增逻辑的主路径
- 失败路径和边界输入
- 回归点

避免只为覆盖率而写和实现细节强耦合的脆弱测试。
`) + "\n",
	},
	{
		Name:        "security-scan",
		Title:       "Security Scan",
		Description: "Use a lightweight security checklist for auth, secrets, injection, and dependency risk.",
		Content: strings.TrimSpace(`
# Security Scan

安全检查至少覆盖：

- 认证与授权边界
- 命令 / SQL / 模板注入风险
- 敏感信息泄露
- 依赖版本与已知漏洞
- 默认配置是否过宽

发现问题时给出可复现路径和修复建议。
`) + "\n",
	},
	{
		Name:        "install-claude-code",
		Title:       "Install Claude Code",
		Description: "Install Claude Code on a target machine and verify the CLI is available for remote execution.",
		Content: strings.TrimSpace(`
# Install Claude Code

目标：让目标机器具备可用的 `+"`claude`"+` 命令，并记录安装结果。

执行时遵循：

- 先确认当前系统类型、包管理器和是否已安装 `+"`claude`"+`。
- 使用官方支持的安装方式完成安装，避免下载来源不明的二进制。
- 安装后至少验证 `+"`claude --version`"+`，并记录可执行路径。
- 如果还需要登录或额外认证，明确记录当前状态和缺失前置条件。

不要把令牌或凭据写入仓库。
`) + "\n",
	},
	{
		Name:        "install-codex",
		Title:       "Install Codex CLI",
		Description: "Install the Codex CLI on a target machine and verify it can start successfully.",
		Content: strings.TrimSpace(`
# Install Codex CLI

目标：让目标机器具备可用的 `+"`codex`"+` 命令，并验证 CLI 能正常启动。

执行时遵循：

- 先检查 `+"`codex`"+` 是否已存在以及当前版本。
- 使用官方支持的安装方式安装或升级 Codex CLI。
- 安装后验证 `+"`codex --version`"+`，必要时补充最小认证检查。
- 如果网络、Python、Node 或系统依赖阻塞安装，记录准确阻塞点，不要留下半安装状态。
`) + "\n",
	},
	{
		Name:        "setup-git",
		Title:       "Setup Git",
		Description: "Install or repair git plus the minimum identity and credential configuration needed for agent work.",
		Content: strings.TrimSpace(`
# Setup Git

目标：让目标机器具备可用的 `+"`git`"+`，并补齐最小身份配置。

执行时遵循：

- 检查 `+"`git --version`"+` 是否可用；如果不可用，先安装 git。
- 检查 `+"`git config --global user.name`"+` 和 `+"`git config --global user.email`"+`；缺失时按工单上下文补齐。
- 仅在确有凭据问题时修复 git 认证，避免覆盖已有可用配置。
- 最后用非破坏性命令确认 git 基础能力可用，并记录生效配置。
`) + "\n",
	},
	{
		Name:        "setup-gh-cli",
		Title:       "Setup GitHub CLI",
		Description: "Install or repair the GitHub CLI and confirm authentication status on the target machine.",
		Content: strings.TrimSpace(`
# Setup GitHub CLI

目标：让目标机器具备可用的 `+"`gh`"+`，并确认 GitHub 认证状态。

执行时遵循：

- 检查 `+"`gh --version`"+` 和 `+"`gh auth status`"+` 的当前输出。
- 如果 `+"`gh`"+` 缺失，使用官方支持的安装方式安装。
- 如果 `+"`gh`"+` 已安装但未认证，补齐认证并再次验证状态。
- 认证失败时记录准确原因，例如网络、令牌缺失或主机不可达。

不要把明文令牌写入 shell 历史或仓库文件。
`) + "\n",
	},
}
