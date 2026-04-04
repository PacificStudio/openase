package workflow

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type DraftSkillProjectionInput struct {
	WorkspaceRoot string
	AdapterType   string
	Bundle        SkillBundle
}

type DraftSkillProjection struct {
	WorkspaceRoot string
	SkillsDir     string
	SkillDir      string
	Bundle        SkillBundle
}

func BuildSkillBundle(name string, files []SkillBundleFileInput) (SkillBundle, error) {
	return parseSkillBundle(name, files)
}

func MaterializeDraftSkillBundle(input DraftSkillProjectionInput) (DraftSkillProjection, error) {
	target, err := resolveSkillTarget(input.WorkspaceRoot, input.AdapterType)
	if err != nil {
		return DraftSkillProjection{}, err
	}
	if err := os.MkdirAll(target.skillsDir.String(), 0o750); err != nil {
		return DraftSkillProjection{}, fmt.Errorf("create agent skill directory: %w", err)
	}
	if err := writeProjectedSkillBundle(
		target.skillsDir.String(),
		input.Bundle.Name,
		input.Bundle.Files,
		input.Bundle.EntrypointBody,
	); err != nil {
		return DraftSkillProjection{}, fmt.Errorf("materialize skill %s: %w", input.Bundle.Name, err)
	}
	if err := writeWorkspaceOpenASEWrapper(target.workspace.String()); err != nil {
		return DraftSkillProjection{}, fmt.Errorf("sync openase wrapper: %w", err)
	}

	return DraftSkillProjection{
		WorkspaceRoot: target.workspace.String(),
		SkillsDir:     target.skillsDir.String(),
		SkillDir:      filepath.Join(target.skillsDir.String(), input.Bundle.Name),
		Bundle:        input.Bundle,
	}, nil
}

func LoadSkillBundleFromDirectory(name string, root string) (SkillBundle, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return SkillBundle{}, fmt.Errorf("resolve skill bundle root: %w", err)
	}
	rootFS, err := os.OpenRoot(absoluteRoot)
	if err != nil {
		return SkillBundle{}, fmt.Errorf("open skill bundle root: %w", err)
	}
	defer func() {
		_ = rootFS.Close()
	}()
	files := make([]SkillBundleFileInput, 0)
	if err := filepath.WalkDir(absoluteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk skill bundle root: %w", walkErr)
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlinks are not supported in skill bundle directories", ErrSkillInvalid)
		}
		relativePath, err := filepath.Rel(absoluteRoot, path)
		if err != nil {
			return fmt.Errorf("resolve skill bundle file path: %w", err)
		}
		content, err := rootFS.ReadFile(relativePath)
		if err != nil {
			return fmt.Errorf("read skill bundle file %s: %w", relativePath, err)
		}
		info, err := rootFS.Stat(relativePath)
		if err != nil {
			return fmt.Errorf("stat skill bundle file %s: %w", relativePath, err)
		}
		files = append(files, SkillBundleFileInput{
			Path:         filepath.ToSlash(relativePath),
			Content:      content,
			IsExecutable: info.Mode().Perm()&0o111 != 0,
		})
		return nil
	}); err != nil {
		return SkillBundle{}, err
	}
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})
	return parseSkillBundle(name, files)
}
