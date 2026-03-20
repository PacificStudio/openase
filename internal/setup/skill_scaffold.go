package setup

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/builtin"
)

type scaffoldFileSpec struct {
	path    string
	content string
	mode    os.FileMode
}

func primaryRepoScaffold(repoRoot string) []scaffoldFileSpec {
	files := make([]scaffoldFileSpec, 0, 3+len(builtin.Skills())+len(builtin.Roles()))
	files = append(files,
		scaffoldFileSpec{
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
		scaffoldFileSpec{
			path:    filepath.Join(repoRoot, ".openase", "skills", ".gitkeep"),
			content: "",
			mode:    0o644,
		},
		scaffoldFileSpec{
			path:    filepath.Join(repoRoot, ".openase", "bin", "openase"),
			content: openASECLIWrapperScript(),
			mode:    0o755,
		},
	)

	for _, skill := range builtin.Skills() {
		files = append(files, scaffoldFileSpec{
			path:    filepath.Join(repoRoot, ".openase", "skills", skill.Name, "SKILL.md"),
			content: skill.Content,
			mode:    0o644,
		})
	}

	for _, role := range builtin.Roles() {
		files = append(files, scaffoldFileSpec{
			path:    filepath.Join(repoRoot, filepath.FromSlash(role.HarnessPath)),
			content: role.Content,
			mode:    0o644,
		})
	}

	return files
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
