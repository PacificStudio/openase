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
./.openase/bin/openase project update --description "更新项目最新上下文"
./.openase/bin/openase project add-repo --name "worker-tools" --url "https://github.com/acme/worker-tools.git"
`+"```"+`

## Notes

- 该 wrapper 透传到已安装的 `+"`openase`"+` 二进制，并自动使用工作区中注入的 `+"`OPENASE_API_URL`"+`、`+"`OPENASE_AGENT_TOKEN`"+`、`+"`OPENASE_PROJECT_ID`"+`、`+"`OPENASE_TICKET_ID`"+`。
- 当前最小实现覆盖已经落地的 agent platform API：工单列表 / 创建 / 更新，以及项目描述更新和 Repo 注册。
- 如果需要更底层调试，可直接对 `+"`$OPENASE_API_URL`"+` 发 HTTP 请求，并带上 `+"`Authorization: Bearer $OPENASE_AGENT_TOKEN`"+`。
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
}
