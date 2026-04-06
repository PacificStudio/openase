package orchestrator

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func (l *RuntimeLauncher) materializeRuntimeSnapshot(
	ctx context.Context,
	runID uuid.UUID,
	workflowID uuid.UUID,
	machine catalogdomain.Machine,
	workspaceRoot string,
	adapterType string,
	remote bool,
) (workflowservice.RuntimeSnapshot, error) {
	if l == nil || l.workflow == nil {
		return workflowservice.RuntimeSnapshot{}, nil
	}

	snapshot, err := l.workflow.ResolveRuntimeSnapshot(ctx, workflowID)
	if err != nil {
		return workflowservice.RuntimeSnapshot{}, err
	}
	if err := l.recordRunVersionUsage(ctx, runID, snapshot); err != nil {
		return workflowservice.RuntimeSnapshot{}, err
	}

	if remote {
		if err := l.materializeRemoteRuntimeSnapshot(ctx, machine, workspaceRoot, adapterType, snapshot); err != nil {
			return workflowservice.RuntimeSnapshot{}, err
		}
	} else {
		if _, err := l.workflow.MaterializeRuntimeSnapshot(workflowservice.MaterializeRuntimeSnapshotInput{
			WorkspaceRoot: workspaceRoot,
			AdapterType:   adapterType,
			Snapshot:      snapshot,
		}); err != nil {
			return workflowservice.RuntimeSnapshot{}, err
		}
	}

	return snapshot, nil
}

func (l *RuntimeLauncher) loadRecordedRuntimeSnapshot(
	ctx context.Context,
	runID uuid.UUID,
) (workflowservice.RuntimeSnapshot, error) {
	if l == nil || l.workflow == nil {
		return workflowservice.RuntimeSnapshot{}, nil
	}

	runItem, err := l.client.AgentRun.Get(ctx, runID)
	if err != nil {
		return workflowservice.RuntimeSnapshot{}, fmt.Errorf("load run for runtime snapshot: %w", err)
	}

	skillVersionIDs := make([]uuid.UUID, 0, len(runItem.SkillVersionIds))
	for _, raw := range runItem.SkillVersionIds {
		id, parseErr := uuid.Parse(strings.TrimSpace(raw))
		if parseErr != nil {
			return workflowservice.RuntimeSnapshot{}, fmt.Errorf("parse recorded skill version id %q: %w", raw, parseErr)
		}
		skillVersionIDs = append(skillVersionIDs, id)
	}

	return l.workflow.ResolveRecordedRuntimeSnapshot(ctx, workflowservice.ResolveRecordedRuntimeSnapshotInput{
		WorkflowID:        runItem.WorkflowID,
		WorkflowVersionID: runItem.WorkflowVersionID,
		SkillVersionIDs:   skillVersionIDs,
	})
}

func (l *RuntimeLauncher) recordRunVersionUsage(
	ctx context.Context,
	runID uuid.UUID,
	snapshot workflowservice.RuntimeSnapshot,
) error {
	if l == nil || l.client == nil {
		return nil
	}

	skillVersionIDs := make(pgarray.StringArray, 0, len(snapshot.Skills))
	for _, skill := range snapshot.Skills {
		skillVersionIDs = append(skillVersionIDs, skill.VersionID.String())
	}

	if _, err := l.client.AgentRun.UpdateOneID(runID).
		SetWorkflowVersionID(snapshot.Workflow.VersionID).
		SetSkillVersionIds(skillVersionIDs).
		Save(ctx); err != nil {
		return fmt.Errorf("record run version usage: %w", err)
	}
	return nil
}

func (l *RuntimeLauncher) materializeRemoteRuntimeSnapshot(
	ctx context.Context,
	machine catalogdomain.Machine,
	workspaceRoot string,
	adapterType string,
	snapshot workflowservice.RuntimeSnapshot,
) error {
	if l == nil || l.transports == nil {
		return fmt.Errorf("machine transport resolver unavailable for remote machine %s", machine.Name)
	}

	resolved, err := l.transports.ResolveRuntime(machine)
	if err != nil {
		return err
	}
	commandSessionExecutor := resolved.CommandSessionExecutor()
	if resolved.Execution.Runtime != nil &&
		resolved.Execution.Runtime.Supports(catalogdomain.MachineTransportCapabilityArtifactSync) &&
		resolved.ArtifactSyncExecutor() != nil {
		return l.materializeRemoteRuntimeSnapshotWithSync(ctx, resolved.ArtifactSyncExecutor(), machine, workspaceRoot, adapterType, snapshot)
	}
	if commandSessionExecutor == nil {
		return fmt.Errorf("%w: remote command session unavailable for machine %s", machinetransport.ErrTransportUnavailable, machine.Name)
	}
	session, err := commandSessionExecutor.OpenCommandSession(ctx, machine)
	if err != nil {
		return fmt.Errorf("open remote command session for machine %s: %w", machine.Name, err)
	}
	defer func() {
		_ = session.Close()
	}()

	command, err := buildRemoteMaterializeRuntimeSnapshotCommand(workspaceRoot, adapterType, snapshot)
	if err != nil {
		return err
	}
	if output, runErr := session.CombinedOutput(command); runErr != nil {
		return fmt.Errorf("materialize remote runtime snapshot: %w: %s", runErr, strings.TrimSpace(string(output)))
	}
	return nil
}

func (l *RuntimeLauncher) materializeRemoteRuntimeSnapshotWithSync(
	ctx context.Context,
	syncer machinetransport.ArtifactSyncExecution,
	machine catalogdomain.Machine,
	workspaceRoot string,
	adapterType string,
	snapshot workflowservice.RuntimeSnapshot,
) error {
	if l == nil || l.workflow == nil {
		return fmt.Errorf("runtime snapshot workflow service unavailable")
	}

	tempRoot, err := os.MkdirTemp("", "openase-runtime-snapshot-*")
	if err != nil {
		return fmt.Errorf("create runtime snapshot temp root: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()

	materialized, err := l.workflow.MaterializeRuntimeSnapshot(workflowservice.MaterializeRuntimeSnapshotInput{
		WorkspaceRoot: tempRoot,
		AdapterType:   adapterType,
		Snapshot:      snapshot,
	})
	if err != nil {
		return fmt.Errorf("materialize runtime snapshot locally for websocket sync: %w", err)
	}

	skillsRelativePath, err := filepath.Rel(tempRoot, materialized.SkillsDir)
	if err != nil {
		return fmt.Errorf("derive runtime skills relative path: %w", err)
	}

	paths := []string{
		filepath.ToSlash(skillsRelativePath),
		".openase/bin/openase",
	}
	if err := syncer.SyncArtifacts(ctx, machine, machinetransport.SyncArtifactsRequest{
		LocalRoot:   tempRoot,
		TargetRoot:  workspaceRoot,
		Paths:       paths,
		RemovePaths: []string{".openase/harnesses", filepath.ToSlash(skillsRelativePath), ".openase/bin"},
	}); err != nil {
		return fmt.Errorf("sync websocket runtime snapshot artifacts: %w", err)
	}
	return nil
}

func buildRemoteMaterializeRuntimeSnapshotCommand(
	workspaceRoot string,
	adapterType string,
	snapshot workflowservice.RuntimeSnapshot,
) (string, error) {
	target, err := workflowservice.ResolveSkillTargetForRuntime(workspaceRoot, adapterType)
	if err != nil {
		return "", err
	}

	lines := []string{
		"set -eu",
		"rm -rf " + sshinfra.ShellQuote(filepath.Join(workspaceRoot, ".openase", "harnesses")),
		"rm -rf " + sshinfra.ShellQuote(target.SkillsDir),
		"mkdir -p " + sshinfra.ShellQuote(target.SkillsDir),
		buildRemoteWriteFileCommand(filepath.Join(workspaceRoot, ".openase", "bin", "openase"), []byte(runtimeOpenASECLIWrapperScript()), true),
	}
	for _, skill := range snapshot.Skills {
		if len(skill.Files) == 0 {
			lines = append(lines, buildRemoteWriteFileCommand(filepath.Join(target.SkillsDir, skill.Name, "SKILL.md"), []byte(skill.Content), false))
			continue
		}
		for _, file := range skill.Files {
			lines = append(lines, buildRemoteWriteFileCommand(filepath.Join(target.SkillsDir, skill.Name, filepath.FromSlash(file.Path)), file.Content, file.IsExecutable))
		}
	}

	return strings.Join(lines, "\n"), nil
}

func buildRemoteWriteFileCommand(path string, content []byte, executable bool) string {
	encoded := base64.StdEncoding.EncodeToString(content)
	lines := []string{
		"mkdir -p " + sshinfra.ShellQuote(filepath.Dir(path)),
		"printf %s " + sshinfra.ShellQuote(encoded) + " | base64 -d > " + sshinfra.ShellQuote(path),
	}
	if executable {
		lines = append(lines, "chmod 700 "+sshinfra.ShellQuote(path))
	}
	return strings.Join(lines, "\n")
}

func runtimeOpenASECLIWrapperScript() string {
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
