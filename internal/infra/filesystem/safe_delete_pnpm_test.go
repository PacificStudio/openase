package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveTreePnpmStyleSymlinks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	boundary := filepath.Join(root, "workspace")
	target := filepath.Join(boundary, "org", "proj", "ASE-499", "web")
	nodeModules := filepath.Join(target, "node_modules")
	pnpmStore := filepath.Join(nodeModules, ".pnpm", "eslint@9.0.0", "node_modules", "eslint")
	if err := os.MkdirAll(pnpmStore, 0o755); err != nil {
		t.Fatalf("mkdir pnpm store: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pnpmStore, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	symlink := filepath.Join(nodeModules, "eslint")
	if err := os.Symlink(pnpmStore, symlink); err != nil {
		t.Fatalf("symlink eslint: %v", err)
	}

	deleted, err := RemoveTree(boundary, target)
	if err != nil {
		t.Fatalf("RemoveTree() error = %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target still exists: %v", err)
	}
}

func TestRemoveTreePermissionDeniedOnSymlink(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-denied unlink behavior differs for root; run as non-root in CI")
	}

	root := t.TempDir()
	boundary := filepath.Join(root, "workspace")
	target := filepath.Join(boundary, "ticket", "web", "node_modules")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	symlink := filepath.Join(target, "eslint")
	if err := os.Symlink(filepath.Join(root, "outside"), symlink); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.Chmod(symlink, 0o000); err != nil {
		t.Fatalf("chmod symlink: %v", err)
	}

	_, err := RemoveTree(boundary, filepath.Join(boundary, "ticket"))
	if err == nil {
		t.Fatal("RemoveTree() error = nil, want permission failure")
	}
	if !errors.Is(err, ErrDeleteFailed) {
		t.Fatalf("RemoveTree() error = %v, want ErrDeleteFailed", err)
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "errsymlink") || strings.Contains(msg, "PANIC=Error") {
		t.Fatalf("RemoveTree() leaked internal error: %q", msg)
	}
}