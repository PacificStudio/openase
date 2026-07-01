package orchestrator

import (
	"errors"
	"fmt"
	"os"
	"strings"

	filesysteminfra "github.com/BetterAndBetterII/openase/internal/infra/filesystem"
)

// TicketWorkspaceResetDeleteError is returned when local workspace cleanup fails for a user-visible reason.
type TicketWorkspaceResetDeleteError struct {
	WorkspaceRoot string
	Reason        string
}

func (e TicketWorkspaceResetDeleteError) Error() string {
	root := strings.TrimSpace(e.WorkspaceRoot)
	reason := strings.TrimSpace(e.Reason)
	if root == "" {
		return reason
	}
	if reason == "" {
		return fmt.Sprintf("failed to remove ticket workspace at %s", root)
	}
	return fmt.Sprintf("failed to remove ticket workspace at %s: %s", root, reason)
}

func (TicketWorkspaceResetDeleteError) WorkspaceResetDeleteFailed() bool {
	return true
}

func classifyLocalWorkspaceDeleteFailure(workspaceRoot string, err error) error {
	if err == nil {
		return nil
	}
	var typed TicketWorkspaceResetDeleteError
	if errors.As(err, &typed) {
		return err
	}

	reason := sanitizeWorkspaceDeleteReason(err)
	return TicketWorkspaceResetDeleteError{
		WorkspaceRoot: strings.TrimSpace(workspaceRoot),
		Reason:        reason,
	}
}

func sanitizeWorkspaceDeleteReason(err error) string {
	if err == nil {
		return ""
	}
	var typed TicketWorkspaceResetDeleteError
	if errors.As(err, &typed) {
		return strings.TrimSpace(typed.Reason)
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "errsymlink") || strings.Contains(lower, "panic=error method") {
		return "permission denied or unreadable path while deleting workspace contents; check ownership of package-manager directories such as node_modules"
	}
	if errors.Is(err, filesysteminfra.ErrUnsafePath) {
		return "workspace path failed safety checks; refuse to delete outside the configured workspace root"
	}
	return extractActionableDeleteDetail(err)
}

func extractActionableDeleteDetail(err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "workspace deletion failed"
	}
	if strings.Contains(strings.ToLower(message), "permission denied") {
		return message
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && pathErr.Path != "" {
		op := strings.TrimSpace(pathErr.Op)
		if op == "" {
			op = "remove"
		}
		if errors.Is(pathErr.Err, os.ErrPermission) {
			return fmt.Sprintf("permission denied on %s (%s)", pathErr.Path, op)
		}
		return fmt.Sprintf("%s %s: %v", op, pathErr.Path, pathErr.Err)
	}
	return message
}

func userVisibleWorkspaceDeleteMessage(err error) string {
	if err == nil {
		return ""
	}
	var typed TicketWorkspaceResetDeleteError
	if errors.As(err, &typed) {
		return typed.Error()
	}
	return sanitizeWorkspaceDeleteReason(err)
}