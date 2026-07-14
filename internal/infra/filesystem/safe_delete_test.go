package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveTreeWithinBoundary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	boundary := filepath.Join(root, "workspace")
	target := filepath.Join(boundary, "org", "proj", "ticket-1")
	if err := os.MkdirAll(filepath.Join(target, "repo"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "repo", "file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
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

func TestRemoveTreeRejectsBoundaryTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	boundary := filepath.Join(root, "workspace")
	if err := os.MkdirAll(boundary, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, err := RemoveTree(boundary, boundary)
	if !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("RemoveTree() error = %v, want ErrUnsafePath", err)
	}
}

func TestRemoveTreeRejectsSymlinkTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	boundary := filepath.Join(root, "workspace")
	secret := filepath.Join(root, "secret")
	target := filepath.Join(boundary, "ticket")
	if err := os.MkdirAll(boundary, 0o750); err != nil {
		t.Fatalf("mkdir boundary: %v", err)
	}
	if err := os.MkdirAll(secret, 0o750); err != nil {
		t.Fatalf("mkdir secret: %v", err)
	}
	if err := os.Symlink(secret, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	_, err := RemoveTree(boundary, target)
	if !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("RemoveTree() error = %v, want ErrUnsafePath", err)
	}
}

func TestRemoveTreeNoBoundary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "artifact")
	if err := os.MkdirAll(target, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	deleted, err := RemoveTree("", target)
	if err != nil {
		t.Fatalf("RemoveTree() error = %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}
}