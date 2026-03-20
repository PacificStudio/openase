package builtin

import (
	"strings"
	"testing"
)

func TestDispatcherRoleTemplate(t *testing.T) {
	role, ok := RoleBySlug("dispatcher")
	if !ok {
		t.Fatalf("expected dispatcher role to exist")
	}

	if role.Name != "Dispatcher" {
		t.Fatalf("Name=%q, want Dispatcher", role.Name)
	}
	if role.WorkflowType != "custom" {
		t.Fatalf("WorkflowType=%q, want custom", role.WorkflowType)
	}

	for _, want := range []string{
		`pickup: "Backlog"`,
		`finish: "Backlog"`,
		`- "tickets.update.self"`,
		`- "machines.list"`,
		"project.workflows",
		"project.statuses",
		"project.machines",
	} {
		if !strings.Contains(role.Content, want) {
			t.Fatalf("expected dispatcher harness to contain %q, got:\n%s", want, role.Content)
		}
	}
}
