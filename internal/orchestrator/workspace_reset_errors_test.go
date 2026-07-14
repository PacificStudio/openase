package orchestrator

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	filesysteminfra "github.com/BetterAndBetterII/openase/internal/infra/filesystem"
)

func TestSanitizeWorkspaceDeleteReasonErrSymlinkLeak(t *testing.T) {
	t.Parallel()

	leaked := fmt.Errorf(
		"remove local workspace /tmp/ws: %%!v(PANIC=Error method: errSymlink is not user-visible)",
	)
	got := sanitizeWorkspaceDeleteReason(leaked)
	if strings.Contains(strings.ToLower(got), "errsymlink") || strings.Contains(got, "PANIC=") {
		t.Fatalf("sanitizeWorkspaceDeleteReason() = %q, want user-visible message", got)
	}
	if !strings.Contains(strings.ToLower(got), "permission") {
		t.Fatalf("sanitizeWorkspaceDeleteReason() = %q, want permission guidance", got)
	}
}

func TestClassifyLocalWorkspaceDeleteFailure(t *testing.T) {
	t.Parallel()

	root := "/tmp/ticket-ws"
	wrapped := fmt.Errorf("remove local workspace %s: %w", root, filesysteminfra.ErrDeleteFailed)
	got := classifyLocalWorkspaceDeleteFailure(root, wrapped)
	var typed TicketWorkspaceResetDeleteError
	if !errors.As(got, &typed) {
		t.Fatalf("classifyLocalWorkspaceDeleteFailure() = %T, want TicketWorkspaceResetDeleteError", got)
	}
	if typed.WorkspaceRoot != root {
		t.Fatalf("WorkspaceRoot = %q, want %q", typed.WorkspaceRoot, root)
	}
	if strings.TrimSpace(typed.Reason) == "" {
		t.Fatal("expected non-empty Reason")
	}
}