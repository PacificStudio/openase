package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnsafePath indicates the delete target failed symlink or boundary safety checks.
var ErrUnsafePath = errors.New("unsafe path for deletion")

// ErrDeleteFailed indicates deletion could not be completed (permissions, partial delete, etc.).
var ErrDeleteFailed = errors.New("path deletion failed")

// PathWithinRoot reports whether target is root itself or a descendant path segment under root
// (string prefix check on cleaned paths; use after symlink resolution for enforcement).
func PathWithinRoot(root string, target string) bool {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	if cleanRoot == cleanTarget {
		return true
	}
	return strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator))
}

// RemoveTree deletes target and its contents after local safety checks.
//
// When boundaryRoot is non-empty, target must be a strict descendant of boundaryRoot (neither
// may equal the other after symlink resolution). The boundary root must not be a symlink that
// escapes its lexical path.
//
// Returns whether anything was deleted (false when target did not exist).
func RemoveTree(boundaryRoot string, target string) (bool, error) {
	cleanBoundary := filepath.Clean(strings.TrimSpace(boundaryRoot))
	cleanTarget := filepath.Clean(strings.TrimSpace(target))
	if cleanTarget == "" || cleanTarget == "." {
		return false, fmt.Errorf("%w: target path must not be empty", ErrUnsafePath)
	}

	if cleanBoundary != "" && cleanBoundary != "." {
		if cleanTarget == cleanBoundary || !PathWithinRoot(cleanBoundary, cleanTarget) {
			return false, ErrUnsafePath
		}
	}

	var boundaryReal string
	if cleanBoundary != "" && cleanBoundary != "." {
		resolved, err := resolveExistingPath(cleanBoundary)
		if err != nil {
			return false, fmt.Errorf("resolve deletion boundary %s: %w", cleanBoundary, err)
		}
		boundaryReal = resolved
	}

	info, err := os.Lstat(cleanTarget)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat deletion target %s: %w", cleanTarget, formatPathOpError(err))
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, ErrUnsafePath
	}

	targetReal, err := resolveExistingPath(cleanTarget)
	if err != nil {
		return false, fmt.Errorf("resolve deletion target %s: %w", cleanTarget, err)
	}

	if boundaryReal != "" {
		if targetReal == boundaryReal || !PathWithinRoot(boundaryReal, targetReal) {
			return false, ErrUnsafePath
		}
	}

	if err := removeTreeAtRealPath(targetReal); err != nil {
		return false, fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}
	return true, nil
}

func resolveExistingPath(path string) (string, error) {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", formatPathOpError(err)
	}
	return real, nil
}

// formatPathOpError wraps path resolution/removal errors without calling Error() on
// internal filepath sentinel values (e.g. errSymlink).
func formatPathOpError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return err
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("permission denied: %w", err)
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && pathErr.Err != nil {
		if errors.Is(pathErr.Err, os.ErrPermission) {
			return fmt.Errorf("permission denied: %w", pathErr)
		}
		return fmt.Errorf("%s: %w", pathErr.Op, pathErr.Err)
	}
	return err
}

func removeTreeAtRealPath(root string) error {
	rootInfo, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return formatRemovePathError(root, err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return ErrUnsafePath
	}
	if !rootInfo.IsDir() {
		if err := os.Remove(root); err != nil {
			return formatRemovePathError(root, err)
		}
		return nil
	}

	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return formatRemovePathError(path, walkErr)
		}
		if path == root {
			return nil
		}
		entryInfo, err := d.Info()
		if err != nil {
			return formatRemovePathError(path, err)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", ErrUnsafePath, path)
		}
		return nil
	}); err != nil {
		return err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return formatRemovePathError(root, err)
	}
	for _, entry := range entries {
		child := filepath.Join(root, entry.Name())
		if err := removeTreeAtRealPath(child); err != nil {
			return err
		}
	}
	if err := os.Remove(root); err != nil {
		return formatRemovePathError(root, err)
	}
	return nil
}

func formatRemovePathError(path string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrUnsafePath) || errors.Is(err, ErrDeleteFailed) {
		return err
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("permission denied removing %s: %w", path, err)
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if pathErr.Err != nil && errors.Is(pathErr.Err, os.ErrPermission) {
			return fmt.Errorf("permission denied removing %s: %w", path, pathErr)
		}
		if pathErr.Err != nil {
			return fmt.Errorf("remove %s: %w", path, pathErr.Err)
		}
	}
	return fmt.Errorf("remove %s: %w", path, err)
}