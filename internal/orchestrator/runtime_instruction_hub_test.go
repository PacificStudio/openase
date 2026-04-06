package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
)

func TestDiscoverLocalInstructionHubSourcesPrunesGeneratedDirsAndSortsDeterministically(t *testing.T) {
	t.Parallel()

	repoA := t.TempDir()
	repoB := t.TempDir()

	writeInstructionHubFixture(t, filepath.Join(repoA, "nested", "CLAUDE.md"), "# nested claude\n")
	writeInstructionHubFixture(t, filepath.Join(repoA, "AGENTS.md"), "# repo a agents\n")
	writeInstructionHubFixture(t, filepath.Join(repoA, ".codex", "AGENTS.md"), "# ignored codex\n")
	writeInstructionHubFixture(t, filepath.Join(repoA, ".openase", "CLAUDE.md"), "# ignored openase\n")
	writeInstructionHubFixture(t, filepath.Join(repoB, "docs", "AGENTS.md"), "# repo b docs\n")
	writeInstructionHubFixture(t, filepath.Join(repoB, ".claude", "CLAUDE.md"), "# ignored claude\n")

	sources, err := discoverLocalInstructionHubSources([]workspaceinfra.PreparedRepo{
		{Name: "repo-b", WorkspaceDirname: "repos/b", Path: repoB},
		{Name: "repo-a", WorkspaceDirname: "repos/a", Path: repoA},
	})
	if err != nil {
		t.Fatalf("discoverLocalInstructionHubSources() error = %v", err)
	}

	got := make([]string, 0, len(sources))
	for _, source := range sources {
		got = append(got, workspaceInstructionPath(source))
	}
	want := []string{
		"repos/a/AGENTS.md",
		"repos/a/nested/CLAUDE.md",
		"repos/b/docs/AGENTS.md",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverLocalInstructionHubSources() paths = %#v, want %#v", got, want)
	}
}

func TestRenderClaudeInstructionHubIncludesImportsByRepo(t *testing.T) {
	t.Parallel()

	content := renderClaudeInstructionHub([]instructionHubSource{
		{
			RepoName:         "backend",
			RepoOrderKey:     "repos/backend",
			WorkspaceDirname: "repos/backend",
			RelativePath:     "AGENTS.md",
		},
		{
			RepoName:         "frontend",
			RepoOrderKey:     "repos/frontend",
			WorkspaceDirname: "repos/frontend",
			RelativePath:     "docs/CLAUDE.md",
		},
	})

	if !strings.Contains(content, generatedInstructionHubMarker) {
		t.Fatalf("renderClaudeInstructionHub() missing generated marker: %q", content)
	}
	if !strings.Contains(content, "@repos/backend/AGENTS.md") {
		t.Fatalf("renderClaudeInstructionHub() missing backend import: %q", content)
	}
	if !strings.Contains(content, "@repos/frontend/docs/CLAUDE.md") {
		t.Fatalf("renderClaudeInstructionHub() missing frontend import: %q", content)
	}
}

func TestRenderCodexInstructionHubInlinesSourceContent(t *testing.T) {
	t.Parallel()

	content := renderCodexInstructionHub([]instructionHubSource{
		{
			RepoName:         "backend",
			RepoOrderKey:     "repos/backend",
			WorkspaceDirname: "repos/backend",
			RelativePath:     "AGENTS.md",
			Content:          []byte("# backend agents\nUse backend repo.\n"),
		},
		{
			RepoName:         "frontend",
			RepoOrderKey:     "repos/frontend",
			WorkspaceDirname: "repos/frontend",
			RelativePath:     "docs/CLAUDE.md",
			Content:          []byte("# frontend claude\nUse frontend repo.\n"),
		},
	})

	for _, fragment := range []string{
		generatedInstructionHubMarker,
		"## Repo `repos/backend (backend)`",
		"### Source `repos/backend/AGENTS.md`",
		"# backend agents",
		"### Source `repos/frontend/docs/CLAUDE.md`",
		"# frontend claude",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("renderCodexInstructionHub() missing fragment %q in %q", fragment, content)
		}
	}
}

func TestMaterializeWorkspaceInstructionHubSkipsSingleRepo(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	repoRoot := t.TempDir()
	writeInstructionHubFixture(t, filepath.Join(repoRoot, "AGENTS.md"), "# repo agents\n")

	provisioner := newRuntimeWorkspaceProvisioner(nil, nil, nil, nil)
	err := provisioner.materializeWorkspaceInstructionHub(
		context.Background(),
		catalogdomainLocalMachineForTest(),
		machinetransportResolvedTransportForTest(),
		workspaceinfra.Workspace{
			Path: workspaceRoot,
			Repos: []workspaceinfra.PreparedRepo{
				{Name: "backend", WorkspaceDirname: "backend", Path: repoRoot},
			},
		},
		"codex-app-server",
		false,
	)
	if err != nil {
		t.Fatalf("materializeWorkspaceInstructionHub() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no workspace-root AGENTS.md for single repo, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no workspace-root CLAUDE.md for single repo, stat err=%v", err)
	}
}

func writeInstructionHubFixture(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create fixture dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func catalogdomainLocalMachineForTest() catalogdomain.Machine {
	return catalogdomain.Machine{Name: "local"}
}

func machinetransportResolvedTransportForTest() machinetransport.ResolvedTransport {
	return machinetransport.ResolvedTransport{}
}
