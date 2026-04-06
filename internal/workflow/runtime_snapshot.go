package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

type RuntimeSnapshot = domain.RuntimeSnapshot

type RuntimeWorkflowSnapshot = domain.RuntimeWorkflowSnapshot

type RuntimeSkillSnapshot = domain.RuntimeSkillSnapshot

type RuntimeSkillFileSnapshot = domain.RuntimeSkillFileSnapshot

type MaterializeRuntimeSnapshotInput = domain.MaterializeRuntimeSnapshotInput

type ResolveRecordedRuntimeSnapshotInput = domain.ResolveRecordedRuntimeSnapshotInput

type MaterializedRuntimeSnapshot = domain.MaterializedRuntimeSnapshot

func (s *Service) MaterializeRuntimeSnapshot(input MaterializeRuntimeSnapshotInput) (MaterializedRuntimeSnapshot, error) {
	if s == nil {
		return MaterializedRuntimeSnapshot{}, ErrUnavailable
	}
	if input.Snapshot.Workflow.VersionID == uuid.Nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("%w: workflow snapshot version_id must not be empty", ErrHarnessInvalid)
	}

	target, err := resolveSkillTarget(input.WorkspaceRoot, input.AdapterType)
	if err != nil {
		return MaterializedRuntimeSnapshot{}, err
	}
	if err := os.RemoveAll(target.skillsDir.String()); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("reset agent skill directory: %w", err)
	}
	if err := os.MkdirAll(target.skillsDir.String(), 0o750); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("create agent skill directory: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(target.workspace.String(), ".openase", "harnesses")); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("remove legacy runtime harness directory: %w", err)
	}

	skillVersionIDs := make([]uuid.UUID, 0, len(input.Snapshot.Skills))
	for _, skill := range input.Snapshot.Skills {
		if err := writeRuntimeSkillSnapshot(target.skillsDir.String(), skill); err != nil {
			return MaterializedRuntimeSnapshot{}, fmt.Errorf("materialize skill %s: %w", skill.Name, err)
		}
		skillVersionIDs = append(skillVersionIDs, skill.VersionID)
	}
	if err := writeWorkspaceOpenASEWrapper(target.workspace.String()); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("sync openase wrapper: %w", err)
	}

	return MaterializedRuntimeSnapshot{
		SkillsDir:         target.skillsDir.String(),
		WorkflowVersionID: input.Snapshot.Workflow.VersionID,
		SkillVersionIDs:   skillVersionIDs,
	}, nil
}

func writeRuntimeSkillSnapshot(skillsDir string, skill RuntimeSkillSnapshot) error {
	files := make([]SkillBundleFile, 0, len(skill.Files))
	for _, file := range skill.Files {
		files = append(files, SkillBundleFile{
			Path:         file.Path,
			IsExecutable: file.IsExecutable,
			Content:      append([]byte(nil), file.Content...),
		})
	}
	return writeProjectedSkillBundle(skillsDir, skill.Name, files, skill.Content)
}

func runtimeSkillNames(skills []RuntimeSkillSnapshot) []string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name)
	}
	sort.Strings(names)
	return names
}
