package setup

import (
	"os"
	"path/filepath"
	"strings"
)

type scaffoldFileSpec struct {
	path    string
	content string
	mode    os.FileMode
}

func primaryRepoScaffold(repoRoot string) []scaffoldFileSpec {
	return []scaffoldFileSpec{
		{
			path: filepath.Join(repoRoot, ".openase", "harnesses", "coding.md"),
			content: `---
workflow:
  name: "Coding"
  type: "coding"
status:
  pickup: "Todo"
  finish: "Done"
---

# Coding

You are implementing the assigned OpenASE ticket in the primary repository.
`,
			mode: 0o644,
		},
		{
			path:    filepath.Join(repoRoot, ".openase", "skills", ".gitkeep"),
			content: "",
			mode:    0o644,
		},
		{
			path:    filepath.Join(repoRoot, ".openase", "skills", "openase-platform", "SKILL.md"),
			content: openASEPlatformSkillMarkdown(),
			mode:    0o644,
		},
		{
			path:    filepath.Join(repoRoot, ".openase", "bin", "openase"),
			content: openASECLIWrapperScript(),
			mode:    0o755,
		},
	}
}

func openASEPlatformSkillMarkdown() string {
	return strings.TrimSpace(`
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
`) + "\n"
}

func openASECLIWrapperScript() string {
	return strings.TrimSpace(`
#!/bin/sh
set -eu

if [ -n "${OPENASE_REAL_BIN:-}" ]; then
  OPENASE_BIN="$OPENASE_REAL_BIN"
elif command -v openase >/dev/null 2>&1; then
  OPENASE_BIN="$(command -v openase)"
else
  echo "openase wrapper: could not find an installed openase binary" >&2
  echo "set OPENASE_REAL_BIN to the desired executable path" >&2
  exit 1
fi

exec "$OPENASE_BIN" "$@"
`) + "\n"
}
