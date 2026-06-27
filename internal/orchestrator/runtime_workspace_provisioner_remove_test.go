package orchestrator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestRemoveWorkspaceRootLocalPnpmLayout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ctx := context.Background()
	provisioner := newRuntimeWorkspaceProvisioner(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, time.Now)

	root := t.TempDir()
	boundary := filepath.Join(root, "workspace")
	workspaceRoot := filepath.Join(boundary, "test", "openase-automation", "ASE-499")
	nodeModules := filepath.Join(workspaceRoot, "openase", "web", "node_modules")
	store := filepath.Join(nodeModules, ".pnpm", "pkg", "node_modules", "eslint")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}
	link := filepath.Join(nodeModules, "eslint")
	if err := os.Symlink(store, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	machine := catalogdomain.Machine{
		Name:          catalogdomain.LocalMachineName,
		Host:          catalogdomain.LocalMachineHost,
		WorkspaceRoot: stringPointer(boundary),
	}
	if err := provisioner.removeWorkspaceRoot(ctx, machine, false, workspaceRoot); err != nil {
		t.Fatalf("removeWorkspaceRoot() error = %v", err)
	}
	if _, err := os.Stat(workspaceRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("workspace still exists: %v", err)
	}
}

func TestRemoveWorkspaceRootLocalPermissionDeniedMessage(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can delete root-owned trees; run as non-root")
	}
	t.Setenv("HOME", t.TempDir())

	ctx := context.Background()
	provisioner := newRuntimeWorkspaceProvisioner(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, time.Now)

	root := t.TempDir()
	boundary := filepath.Join(root, "workspace")
	workspaceRoot := filepath.Join(boundary, "ticket")
	protected := filepath.Join(workspaceRoot, "node_modules")
	if err := os.MkdirAll(protected, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(protected, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(protected, 0o755) })

	machine := catalogdomain.Machine{
		Name:          catalogdomain.LocalMachineName,
		Host:          catalogdomain.LocalMachineHost,
		WorkspaceRoot: stringPointer(boundary),
	}
	err := provisioner.removeWorkspaceRoot(ctx, machine, false, workspaceRoot)
	if err == nil {
		t.Fatal("removeWorkspaceRoot() error = nil, want failure")
	}
	var deleteErr TicketWorkspaceResetDeleteError
	if !errors.As(err, &deleteErr) {
		t.Fatalf("removeWorkspaceRoot() error = %T %v, want TicketWorkspaceResetDeleteError", err, err)
	}
	msg := err.Error()
	if strings.Contains(msg, "errSymlink") || strings.Contains(msg, "PANIC=Error") {
		t.Fatalf("leaked internal error: %q", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "permission") {
		t.Fatalf("error = %q, want permission context", msg)
	}
}