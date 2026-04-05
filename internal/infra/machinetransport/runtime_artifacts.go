package machinetransport

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/BetterAndBetterII/openase/internal/logging"
)

var _ = logging.DeclareComponent("machine-transport-runtime-artifacts")

func buildArtifactSyncEntries(request SyncArtifactsRequest) ([]runtimecontract.ArtifactEntry, error) {
	root := filepath.Clean(strings.TrimSpace(request.LocalRoot))
	if root == "." || strings.TrimSpace(request.LocalRoot) == "" {
		return nil, fmt.Errorf("local artifact root must not be empty")
	}

	entries := make([]runtimecontract.ArtifactEntry, 0, len(request.Paths))
	for _, relative := range request.Paths {
		trimmed := strings.TrimSpace(relative)
		if trimmed == "" {
			continue
		}
		sourcePath := filepath.Join(root, filepath.FromSlash(trimmed))
		info, err := os.Stat(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("stat artifact %s: %w", trimmed, err)
		}
		if info.IsDir() {
			if err := filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				info, err := entry.Info()
				if err != nil {
					return err
				}
				relativePath, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				item, err := buildArtifactEntry(path, filepath.ToSlash(relativePath), info)
				if err != nil {
					return err
				}
				entries = append(entries, item)
				return nil
			}); err != nil {
				return nil, fmt.Errorf("walk artifact %s: %w", trimmed, err)
			}
			continue
		}
		item, err := buildArtifactEntry(sourcePath, filepath.ToSlash(trimmed), info)
		if err != nil {
			return nil, err
		}
		entries = append(entries, item)
	}
	return entries, nil
}

func buildArtifactEntry(sourcePath string, relativePath string, info os.FileInfo) (runtimecontract.ArtifactEntry, error) {
	item := runtimecontract.ArtifactEntry{
		Path: filepath.ToSlash(relativePath),
		Mode: int64(info.Mode().Perm()),
	}
	if info.IsDir() {
		item.Kind = runtimecontract.ArtifactEntryKindDir
		return item, nil
	}
	// #nosec G304 -- artifact paths are derived from orchestrator-managed sync inputs.
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return runtimecontract.ArtifactEntry{}, fmt.Errorf("read artifact %s: %w", relativePath, err)
	}
	item.Kind = runtimecontract.ArtifactEntryKindFile
	item.ContentBase64 = base64.StdEncoding.EncodeToString(content)
	return item, nil
}

func materializeArtifactEntries(targetRoot string, removePaths []string, entries []runtimecontract.ArtifactEntry) error {
	root := filepath.Clean(strings.TrimSpace(targetRoot))
	if root == "." || strings.TrimSpace(targetRoot) == "" {
		return fmt.Errorf("target artifact root must not be empty")
	}
	for _, relative := range removePaths {
		trimmed := strings.TrimSpace(relative)
		if trimmed == "" {
			continue
		}
		if err := removeLocalPath(filepath.Join(root, filepath.FromSlash(trimmed))); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry.Path)
		if trimmed == "" {
			return fmt.Errorf("artifact path must not be empty")
		}
		targetPath := filepath.Join(root, filepath.FromSlash(trimmed))
		switch entry.Kind {
		case runtimecontract.ArtifactEntryKindDir:
			mode, err := parseArtifactFileMode(entry.Mode, 0o750)
			if err != nil {
				return fmt.Errorf("parse artifact directory mode for %s: %w", trimmed, err)
			}
			if err := os.MkdirAll(targetPath, mode); err != nil {
				return fmt.Errorf("create artifact directory %s: %w", targetPath, err)
			}
		case runtimecontract.ArtifactEntryKindFile:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
				return fmt.Errorf("create artifact parent %s: %w", filepath.Dir(targetPath), err)
			}
			content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(entry.ContentBase64))
			if err != nil {
				return fmt.Errorf("decode artifact %s: %w", trimmed, err)
			}
			mode, err := parseArtifactFileMode(entry.Mode, 0o600)
			if err != nil {
				return fmt.Errorf("parse artifact file mode for %s: %w", trimmed, err)
			}
			if err := os.WriteFile(targetPath, content, mode); err != nil {
				return fmt.Errorf("write artifact %s: %w", targetPath, err)
			}
		default:
			return fmt.Errorf("artifact kind %q is not supported", entry.Kind)
		}
	}
	return nil
}

func parseArtifactFileMode(raw int64, fallback fs.FileMode) (fs.FileMode, error) {
	if raw == 0 {
		return fallback, nil
	}
	if raw < 0 || raw > 0o777 {
		return 0, fmt.Errorf("artifact mode %d is out of range", raw)
	}
	return fs.FileMode(raw), nil
}
