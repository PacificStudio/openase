package orchestrator

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
)

const generatedInstructionHubMarker = "<!-- openase-generated:instruction-hub -->"

var (
	canonicalInstructionDocs = map[string]struct{}{
		"AGENTS.md": {},
		"CLAUDE.md": {},
	}
	prunedInstructionHubDirs = map[string]struct{}{
		".agent":   {},
		".claude":  {},
		".codex":   {},
		".gemini":  {},
		".git":     {},
		".openase": {},
	}
)

type instructionHubSource struct {
	RepoName         string
	RepoOrderKey     string
	WorkspaceDirname string
	RelativePath     string
	Content          []byte
}

func (p *runtimeWorkspaceProvisioner) materializeWorkspaceInstructionHub(
	ctx context.Context,
	machine catalogdomain.Machine,
	resolved machinetransport.ResolvedTransport,
	workspaceItem workspaceinfra.Workspace,
	adapterType string,
	remote bool,
) error {
	targetFile, ok := instructionHubFileName(adapterType)
	if !ok || len(workspaceItem.Repos) <= 1 {
		return nil
	}

	var (
		sources []instructionHubSource
		err     error
	)
	if remote {
		sources, err = discoverRemoteInstructionHubSources(ctx, resolved, machine, workspaceItem.Repos)
	} else {
		sources, err = discoverLocalInstructionHubSources(workspaceItem.Repos)
	}
	if err != nil {
		return err
	}

	content := renderInstructionHub(targetFile, sources)
	if remote {
		if err := syncRemoteInstructionHub(ctx, resolved, machine, workspaceItem.Path, targetFile, content); err != nil {
			return err
		}
		return cleanupRemoteGeneratedInstructionHub(ctx, resolved, machine, workspaceItem.Path, targetFile)
	}

	if err := writeLocalInstructionHub(workspaceItem.Path, targetFile, content); err != nil {
		return err
	}
	return cleanupLocalGeneratedInstructionHub(workspaceItem.Path, targetFile)
}

func instructionHubFileName(adapterType string) (string, bool) {
	switch entagentprovider.AdapterType(strings.TrimSpace(adapterType)) {
	case entagentprovider.AdapterTypeClaudeCodeCli:
		return "CLAUDE.md", true
	case entagentprovider.AdapterTypeCodexAppServer:
		return "AGENTS.md", true
	default:
		return "", false
	}
}

func discoverLocalInstructionHubSources(repos []workspaceinfra.PreparedRepo) ([]instructionHubSource, error) {
	sources := make([]instructionHubSource, 0)
	for _, repo := range repos {
		discovered, err := discoverInstructionHubSources(repo)
		if err != nil {
			return nil, err
		}
		sources = append(sources, discovered...)
	}
	sortInstructionHubSources(sources)
	return sources, nil
}

func discoverInstructionHubSources(repo workspaceinfra.PreparedRepo) ([]instructionHubSource, error) {
	repoPath := strings.TrimSpace(repo.Path)
	if repoPath == "" {
		return nil, fmt.Errorf("prepared repo %s is missing a path", strings.TrimSpace(repo.Name))
	}

	repoFS := os.DirFS(repoPath)
	sources := make([]instructionHubSource, 0)
	err := filepath.WalkDir(repoPath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if currentPath != repoPath {
				if _, prune := prunedInstructionHubDirs[entry.Name()]; prune {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if _, ok := canonicalInstructionDocs[entry.Name()]; !ok {
			return nil
		}

		relativePath, err := filepath.Rel(repoPath, currentPath)
		if err != nil {
			return fmt.Errorf("derive relative path for %s: %w", currentPath, err)
		}
		content, err := fs.ReadFile(repoFS, filepath.ToSlash(relativePath))
		if err != nil {
			return fmt.Errorf("read instruction doc %s: %w", currentPath, err)
		}

		sources = append(sources, instructionHubSource{
			RepoName:         strings.TrimSpace(repo.Name),
			RepoOrderKey:     repoInstructionOrderKey(repo),
			WorkspaceDirname: filepath.ToSlash(strings.TrimSpace(repo.WorkspaceDirname)),
			RelativePath:     filepath.ToSlash(relativePath),
			Content:          content,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover instruction docs for repo %s: %w", repoDisplayName(repo), err)
	}
	return sources, nil
}

func discoverRemoteInstructionHubSources(
	ctx context.Context,
	resolved machinetransport.ResolvedTransport,
	machine catalogdomain.Machine,
	repos []workspaceinfra.PreparedRepo,
) ([]instructionHubSource, error) {
	commandExecutor := resolved.CommandSessionExecutor()
	if commandExecutor == nil {
		return nil, fmt.Errorf("remote instruction hub discovery is unavailable for machine %s: command session transport is missing", machine.Name)
	}
	session, err := commandExecutor.OpenCommandSession(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("open remote command session for instruction hub discovery on machine %s: %w", machine.Name, err)
	}
	defer func() { _ = session.Close() }()

	command := buildRemoteInstructionHubDiscoveryCommand(repos)
	output, err := session.CombinedOutput(command)
	if err != nil {
		return nil, fmt.Errorf("discover remote instruction docs: %w: %s", err, strings.TrimSpace(string(output)))
	}
	sources, err := parseRemoteInstructionHubSources(output, repos)
	if err != nil {
		return nil, err
	}
	sortInstructionHubSources(sources)
	return sources, nil
}

func buildRemoteInstructionHubDiscoveryCommand(repos []workspaceinfra.PreparedRepo) string {
	lines := make([]string, 0, 1+4*len(repos))
	lines = append(lines, "set -eu")
	for _, repo := range repos {
		lines = append(lines,
			"repo_name="+sshinfra.ShellQuote(strings.TrimSpace(repo.Name)),
			"workspace_dir="+sshinfra.ShellQuote(filepath.ToSlash(strings.TrimSpace(repo.WorkspaceDirname))),
			"repo_path="+sshinfra.ShellQuote(strings.TrimSpace(repo.Path)),
			`find "$repo_path" \
  \( -type d \( -name .git -o -name .openase -o -name .codex -o -name .claude -o -name .gemini -o -name .agent \) -prune \) -o \
  \( -type f \( -name AGENTS.md -o -name CLAUDE.md \) -print \) | LC_ALL=C sort | \
while IFS= read -r path; do
  [ -n "$path" ] || continue
  case "$path" in
    "$repo_path"/*) rel=${path#"$repo_path"/} ;;
    "$repo_path") rel=$(basename "$path") ;;
    *) echo "unexpected instruction doc path: $path" >&2; exit 1 ;;
  esac
  content=$(base64 <"$path" | tr -d '\n')
  printf '%s\t%s\t%s\t%s\n' "$repo_name" "$workspace_dir" "$rel" "$content"
done`,
		)
	}
	return strings.Join(lines, "\n")
}

func parseRemoteInstructionHubSources(output []byte, repos []workspaceinfra.PreparedRepo) ([]instructionHubSource, error) {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	orderKeyByWorkspaceDir := make(map[string]string, len(repos))
	for _, repo := range repos {
		orderKeyByWorkspaceDir[filepath.ToSlash(strings.TrimSpace(repo.WorkspaceDirname))] = repoInstructionOrderKey(repo)
	}

	lines := strings.Split(trimmed, "\n")
	sources := make([]instructionHubSource, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("parse remote instruction doc output: expected 4 tab-separated fields, got %q", line)
		}
		content, err := base64.StdEncoding.DecodeString(parts[3])
		if err != nil {
			return nil, fmt.Errorf("decode remote instruction doc %s/%s: %w", parts[1], parts[2], err)
		}
		workspaceDir := filepath.ToSlash(strings.TrimSpace(parts[1]))
		orderKey := strings.TrimSpace(orderKeyByWorkspaceDir[workspaceDir])
		if orderKey == "" {
			orderKey = workspaceDir
		}
		sources = append(sources, instructionHubSource{
			RepoName:         strings.TrimSpace(parts[0]),
			RepoOrderKey:     orderKey,
			WorkspaceDirname: workspaceDir,
			RelativePath:     filepath.ToSlash(strings.TrimSpace(parts[2])),
			Content:          content,
		})
	}
	return sources, nil
}

func repoInstructionOrderKey(repo workspaceinfra.PreparedRepo) string {
	if workspaceDir := filepath.ToSlash(strings.TrimSpace(repo.WorkspaceDirname)); workspaceDir != "" {
		return workspaceDir
	}
	return filepath.ToSlash(strings.TrimSpace(repo.Path))
}

func repoDisplayName(repo workspaceinfra.PreparedRepo) string {
	if workspaceDir := filepath.ToSlash(strings.TrimSpace(repo.WorkspaceDirname)); workspaceDir != "" {
		return workspaceDir
	}
	if name := strings.TrimSpace(repo.Name); name != "" {
		return name
	}
	return filepath.ToSlash(strings.TrimSpace(repo.Path))
}

func sortInstructionHubSources(sources []instructionHubSource) {
	sort.Slice(sources, func(i, j int) bool {
		left := sources[i]
		right := sources[j]
		if left.RepoOrderKey != right.RepoOrderKey {
			return left.RepoOrderKey < right.RepoOrderKey
		}
		leftDir, leftFile := splitInstructionRelativePath(left.RelativePath)
		rightDir, rightFile := splitInstructionRelativePath(right.RelativePath)
		if leftDir != rightDir {
			return leftDir < rightDir
		}
		if leftFile != rightFile {
			return leftFile < rightFile
		}
		if left.WorkspaceDirname != right.WorkspaceDirname {
			return left.WorkspaceDirname < right.WorkspaceDirname
		}
		return left.RepoName < right.RepoName
	})
}

func splitInstructionRelativePath(relativePath string) (string, string) {
	cleaned := path.Clean(filepath.ToSlash(strings.TrimSpace(relativePath)))
	dir, file := path.Split(cleaned)
	dir = strings.TrimSuffix(dir, "/")
	if dir == "." {
		dir = ""
	}
	return dir, file
}

func renderInstructionHub(targetFile string, sources []instructionHubSource) []byte {
	switch targetFile {
	case "AGENTS.md":
		return []byte(renderCodexInstructionHub(sources))
	case "CLAUDE.md":
		return []byte(renderClaudeInstructionHub(sources))
	default:
		return nil
	}
}

func renderCodexInstructionHub(sources []instructionHubSource) string {
	var builder strings.Builder
	builder.WriteString("# Generated AGENTS.md\n\n")
	builder.WriteString(generatedInstructionHubMarker)
	builder.WriteString("\n\n")
	builder.WriteString("This file is synthesized for a multi-repo Ticket Agent Run so Codex launched from the ticket workspace root can see repo-local instruction docs discovered under the selected repos.\n\n")
	if len(sources) == 0 {
		builder.WriteString("No repo-local `AGENTS.md` or `CLAUDE.md` files were found under the selected repos.\n")
		return builder.String()
	}

	currentRepo := ""
	for _, source := range sources {
		repoHeader := renderedRepoHeader(source)
		if repoHeader != currentRepo {
			if currentRepo != "" {
				builder.WriteString("\n")
			}
			currentRepo = repoHeader
			builder.WriteString("## Repo `")
			builder.WriteString(repoHeader)
			builder.WriteString("`\n\n")
		}

		workspacePath := workspaceInstructionPath(source)
		builder.WriteString("### Source `")
		builder.WriteString(workspacePath)
		builder.WriteString("`\n\n")
		builder.WriteString("- Repo name: `")
		builder.WriteString(source.RepoName)
		builder.WriteString("`\n")
		builder.WriteString("- Workspace dirname: `")
		builder.WriteString(source.WorkspaceDirname)
		builder.WriteString("`\n")
		builder.WriteString("- Repo-relative path: `")
		builder.WriteString(source.RelativePath)
		builder.WriteString("`\n\n")
		builder.WriteString("<!-- BEGIN OPENASE SOURCE ")
		builder.WriteString(workspacePath)
		builder.WriteString(" -->\n")
		builder.Write(source.Content)
		if len(source.Content) == 0 || source.Content[len(source.Content)-1] != '\n' {
			builder.WriteString("\n")
		}
		builder.WriteString("<!-- END OPENASE SOURCE ")
		builder.WriteString(workspacePath)
		builder.WriteString(" -->\n\n")
	}
	return strings.TrimRight(builder.String(), "\n") + "\n"
}

func renderClaudeInstructionHub(sources []instructionHubSource) string {
	var builder strings.Builder
	builder.WriteString("# Generated CLAUDE.md\n\n")
	builder.WriteString(generatedInstructionHubMarker)
	builder.WriteString("\n\n")
	builder.WriteString("This file is synthesized for a multi-repo Ticket Agent Run so Claude Code launched from the ticket workspace root can import repo-local instruction docs discovered under the selected repos.\n\n")
	if len(sources) == 0 {
		builder.WriteString("No repo-local `AGENTS.md` or `CLAUDE.md` files were found under the selected repos.\n")
		return builder.String()
	}

	currentRepo := ""
	for _, source := range sources {
		repoHeader := renderedRepoHeader(source)
		if repoHeader != currentRepo {
			if currentRepo != "" {
				builder.WriteString("\n")
			}
			currentRepo = repoHeader
			builder.WriteString("## Repo `")
			builder.WriteString(repoHeader)
			builder.WriteString("`\n\n")
		}
		workspacePath := workspaceInstructionPath(source)
		builder.WriteString("### Source `")
		builder.WriteString(source.RelativePath)
		builder.WriteString("`\n\n")
		builder.WriteString("- Repo name: `")
		builder.WriteString(source.RepoName)
		builder.WriteString("`\n")
		builder.WriteString("- Workspace dirname: `")
		builder.WriteString(source.WorkspaceDirname)
		builder.WriteString("`\n")
		builder.WriteString("- Workspace-relative import path: `")
		builder.WriteString(workspacePath)
		builder.WriteString("`\n\n")
		builder.WriteString("@")
		builder.WriteString(workspacePath)
		builder.WriteString("\n\n")
	}
	return strings.TrimRight(builder.String(), "\n") + "\n"
}

func renderedRepoHeader(source instructionHubSource) string {
	workspaceDir := strings.TrimSpace(source.WorkspaceDirname)
	repoName := strings.TrimSpace(source.RepoName)
	switch {
	case workspaceDir == "":
		return repoName
	case repoName == "" || repoName == workspaceDir:
		return workspaceDir
	default:
		return workspaceDir + " (" + repoName + ")"
	}
}

func workspaceInstructionPath(source instructionHubSource) string {
	if strings.TrimSpace(source.WorkspaceDirname) == "" {
		return path.Clean(filepath.ToSlash(source.RelativePath))
	}
	return path.Join(filepath.ToSlash(source.WorkspaceDirname), filepath.ToSlash(source.RelativePath))
}

func writeLocalInstructionHub(workspaceRoot string, targetFile string, content []byte) error {
	targetPath := filepath.Join(workspaceRoot, targetFile)
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		return fmt.Errorf("write workspace instruction hub %s: %w", targetPath, err)
	}
	return nil
}

func cleanupLocalGeneratedInstructionHub(workspaceRoot string, targetFile string) error {
	sibling := instructionHubSiblingFile(targetFile)
	if sibling == "" {
		return nil
	}
	siblingPath := filepath.Join(workspaceRoot, sibling)
	// #nosec G304 -- sibling hub path is derived from the workspace root plus a fixed filename.
	content, err := os.ReadFile(siblingPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("read sibling workspace instruction hub %s: %w", siblingPath, err)
	}
	if !bytes.Contains(content, []byte(generatedInstructionHubMarker)) {
		return nil
	}
	if err := os.Remove(siblingPath); err != nil {
		return fmt.Errorf("remove stale generated workspace instruction hub %s: %w", siblingPath, err)
	}
	return nil
}

func syncRemoteInstructionHub(
	ctx context.Context,
	resolved machinetransport.ResolvedTransport,
	machine catalogdomain.Machine,
	workspaceRoot string,
	targetFile string,
	content []byte,
) error {
	syncer := resolved.ArtifactSyncExecutor()
	if syncer == nil {
		return fmt.Errorf("remote instruction hub sync is unavailable for machine %s: artifact sync transport is missing", machine.Name)
	}

	tempRoot, err := os.MkdirTemp("", "openase-instruction-hub-*")
	if err != nil {
		return fmt.Errorf("create temp instruction hub root: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()

	localPath := filepath.Join(tempRoot, targetFile)
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		return fmt.Errorf("write temp instruction hub %s: %w", localPath, err)
	}
	if err := syncer.SyncArtifacts(ctx, machine, machinetransport.SyncArtifactsRequest{
		LocalRoot:   tempRoot,
		TargetRoot:  workspaceRoot,
		Paths:       []string{targetFile},
		RemovePaths: []string{targetFile},
	}); err != nil {
		return fmt.Errorf("sync remote instruction hub %s: %w", targetFile, err)
	}
	return nil
}

func cleanupRemoteGeneratedInstructionHub(
	ctx context.Context,
	resolved machinetransport.ResolvedTransport,
	machine catalogdomain.Machine,
	workspaceRoot string,
	targetFile string,
) error {
	sibling := instructionHubSiblingFile(targetFile)
	if sibling == "" {
		return nil
	}
	commandExecutor := resolved.CommandSessionExecutor()
	if commandExecutor == nil {
		return fmt.Errorf("remote instruction hub cleanup is unavailable for machine %s: command session transport is missing", machine.Name)
	}
	session, err := commandExecutor.OpenCommandSession(ctx, machine)
	if err != nil {
		return fmt.Errorf("open remote command session for instruction hub cleanup on machine %s: %w", machine.Name, err)
	}
	defer func() { _ = session.Close() }()

	command := buildRemoteInstructionHubCleanupCommand(filepath.Join(workspaceRoot, sibling))
	if output, err := session.CombinedOutput(command); err != nil {
		return fmt.Errorf("cleanup remote generated instruction hub %s: %w: %s", sibling, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func buildRemoteInstructionHubCleanupCommand(targetPath string) string {
	return strings.Join([]string{
		"set -eu",
		"target_path=" + sshinfra.ShellQuote(targetPath),
		"if [ -f \"$target_path\" ] && grep -Fq " + sshinfra.ShellQuote(generatedInstructionHubMarker) + " \"$target_path\"; then",
		"  rm -f \"$target_path\"",
		"fi",
	}, "\n")
}

func instructionHubSiblingFile(targetFile string) string {
	switch targetFile {
	case "AGENTS.md":
		return "CLAUDE.md"
	case "CLAUDE.md":
		return "AGENTS.md"
	default:
		return ""
	}
}
