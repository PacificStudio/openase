package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/builtin"
)

type projectAssetFile struct {
	path    string
	content string
	mode    os.FileMode
}

func ensureProjectBuiltinAssets(repoRoot string) error {
	for _, asset := range projectBuiltinAssets(repoRoot) {
		if err := writeAssetIfMissing(asset); err != nil {
			return err
		}
	}

	return nil
}

func syncProjectWrapperToWorkspace(repoRoot string, workspaceRoot string) error {
	src := filepath.Join(repoRoot, ".openase", "bin", "openase")
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat project openase wrapper: %w", err)
	}

	//nolint:gosec // src is derived from the trusted project root
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read project openase wrapper: %w", err)
	}

	dst := filepath.Join(workspaceRoot, ".openase", "bin", "openase")
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create workspace openase wrapper directory: %w", err)
	}
	if err := os.WriteFile(dst, content, info.Mode().Perm()); err != nil { //nolint:gosec // dst stays under the prepared workspace root
		return fmt.Errorf("write workspace openase wrapper: %w", err)
	}

	return nil
}

func projectBuiltinAssets(repoRoot string) []projectAssetFile {
	files := make([]projectAssetFile, 0, 2+len(builtin.Skills()))
	files = append(files,
		projectAssetFile{
			path:    filepath.Join(repoRoot, ".openase", "skills", ".gitkeep"),
			content: "",
			mode:    0o644,
		},
		projectAssetFile{
			path:    filepath.Join(repoRoot, ".openase", "bin", "openase"),
			content: workflowOpenASECLIWrapperScript(),
			mode:    0o755,
		},
	)

	for _, skill := range builtin.Skills() {
		files = append(files, projectAssetFile{
			path:    filepath.Join(repoRoot, ".openase", "skills", skill.Name, "SKILL.md"),
			content: skill.Content,
			mode:    0o644,
		})
	}

	return files
}

func writeAssetIfMissing(asset projectAssetFile) error {
	if err := os.MkdirAll(filepath.Dir(asset.path), 0o750); err != nil {
		return fmt.Errorf("create project asset directory for %s: %w", asset.path, err)
	}
	if _, err := os.Stat(asset.path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat project asset %s: %w", asset.path, err)
	}
	if err := os.WriteFile(asset.path, []byte(asset.content), asset.mode); err != nil {
		return fmt.Errorf("write project asset %s: %w", asset.path, err)
	}

	return nil
}

func workflowOpenASECLIWrapperScript() string {
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
