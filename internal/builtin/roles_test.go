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
		"pickup_statuses | map(attribute=\"name\")",
		"stage={{ item.stage }}",
		"resources={{ item.resources | tojson }}",
		"move it from {{ workflow.pickup_status }} to one of the names already exposed in project.workflows[].pickup_statuses or project.statuses",
		"keep it in {{ workflow.finish_status }}",
	} {
		if !strings.Contains(role.Content, want) {
			t.Fatalf("expected dispatcher harness to contain %q, got:\n%s", want, role.Content)
		}
	}
}

func TestHarnessOptimizerRoleTemplate(t *testing.T) {
	role, ok := RoleBySlug("harness-optimizer")
	if !ok {
		t.Fatalf("expected harness optimizer role to exist")
	}

	if role.Name != "Harness Optimizer" {
		t.Fatalf("Name=%q, want Harness Optimizer", role.Name)
	}
	if role.WorkflowType != "refine-harness" {
		t.Fatalf("WorkflowType=%q, want refine-harness", role.WorkflowType)
	}

	for _, want := range []string{
		`type: "refine-harness"`,
		`- openase-platform`,
		`- pull`,
		`- commit`,
		`- push`,
		`- "tickets.create"`,
		`- "tickets.list"`,
		`- "tickets.update.self"`,
		"project.workflows",
		"recent_tickets",
		"Only move the ticket to {{ workflow.finish_status }}",
	} {
		if !strings.Contains(role.Content, want) {
			t.Fatalf("expected harness optimizer content to contain %q, got:\n%s", want, role.Content)
		}
	}
}

func TestEnvProvisionerRoleTemplate(t *testing.T) {
	role, ok := RoleBySlug("env-provisioner")
	if !ok {
		t.Fatalf("expected env provisioner role to exist")
	}

	if role.Name != "Environment Provisioner" {
		t.Fatalf("Name=%q, want Environment Provisioner", role.Name)
	}
	if role.WorkflowType != "custom" {
		t.Fatalf("WorkflowType=%q, want custom", role.WorkflowType)
	}

	for _, want := range []string{
		`pickup: "环境修复"`,
		`finish: "环境就绪"`,
		`- install-claude-code`,
		`- install-codex`,
		`- setup-git`,
		`- setup-gh-cli`,
		"target machine environment over SSH",
		"makes the machine dispatchable again",
		"Current machine: {{ machine.name }}",
		"Only move the ticket to {{ workflow.finish_status }}",
	} {
		if !strings.Contains(role.Content, want) {
			t.Fatalf("expected env provisioner content to contain %q, got:\n%s", want, role.Content)
		}
	}
}

func TestRolesHelpers(t *testing.T) {
	roles := Roles()
	if len(roles) == 0 {
		t.Fatal("Roles() expected built-in roles")
	}

	originalName := roles[0].Name
	roles[0].Name = "mutated"
	refreshed := Roles()
	if len(refreshed) == 0 || refreshed[0].Name != originalName {
		t.Fatalf("Roles() should clone templates, got %+v", refreshed)
	}

	if _, ok := RoleBySlug("missing"); ok {
		t.Fatal("RoleBySlug(missing) expected false")
	}

	custom := buildRoleTemplate("coding", "Coding", "coding", "Ship code", []string{"write-test", "review-code"}, "Do the work.")
	if custom.HarnessPath != ".openase/harnesses/roles/coding.md" {
		t.Fatalf("HarnessPath=%q, want %q", custom.HarnessPath, ".openase/harnesses/roles/coding.md")
	}
	for _, want := range []string{
		`name: "Coding"`,
		`type: "coding"`,
		`role: "coding"`,
		`- write-test`,
		`- review-code`,
		"## Runtime Context",
		"{{ workflow.pickup_status }}",
		"{{ workflow.finish_status }}",
		"## Workpad",
		"## Status Control",
		"## Execution Rules",
		"Do the work.",
	} {
		if !strings.Contains(custom.Content, want) {
			t.Fatalf("expected generated content to contain %q, got:\n%s", want, custom.Content)
		}
	}
}
