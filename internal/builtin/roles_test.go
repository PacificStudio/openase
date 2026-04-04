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
	if role.WorkflowType != "Dispatcher" {
		t.Fatalf("WorkflowType=%q, want Dispatcher", role.WorkflowType)
	}
	if got := strings.Join(role.PickupStatusNames, ","); got != "Backlog" {
		t.Fatalf("PickupStatusNames=%q, want Backlog", got)
	}
	if got := strings.Join(role.FinishStatusNames, ","); got != "Todo" {
		t.Fatalf("FinishStatusNames=%q, want Todo", got)
	}
	if got := strings.Join(role.PlatformAccessAllowed, ","); got != "activity.read,statuses.list,tickets.create,tickets.list,tickets.update.self,workflows.list" {
		t.Fatalf("PlatformAccessAllowed=%q", got)
	}

	for _, want := range []string{
		"project.workflows",
		"project.updates",
		"project.statuses",
		"project.machines",
		"pickup_statuses | map(attribute=\"name\")",
		"stage={{ item.stage }}",
		"resources={{ item.resources | tojson }}",
		"move it from {{ workflow.pickup_status }} to one of the names already exposed in project.workflows[].pickup_statuses or project.statuses",
		"Finish the run only after moving the ticket out of {{ workflow.pickup_status }}",
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
	if role.WorkflowType != "Harness Optimizer" {
		t.Fatalf("WorkflowType=%q, want Harness Optimizer", role.WorkflowType)
	}
	if got := strings.Join(role.SkillNames, ","); got != "openase-platform,pull,commit,push" {
		t.Fatalf("SkillNames=%q", got)
	}
	if got := strings.Join(role.PlatformAccessAllowed, ","); got != "tickets.create,tickets.list,tickets.update.self" {
		t.Fatalf("PlatformAccessAllowed=%q", got)
	}

	for _, want := range []string{
		"project.workflows",
		"project.updates",
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
	if role.WorkflowType != "Environment Provisioner" {
		t.Fatalf("WorkflowType=%q, want Environment Provisioner", role.WorkflowType)
	}
	if got := strings.Join(role.PickupStatusNames, ","); got != "环境修复" {
		t.Fatalf("PickupStatusNames=%q", got)
	}
	if got := strings.Join(role.FinishStatusNames, ","); got != "环境就绪" {
		t.Fatalf("FinishStatusNames=%q", got)
	}
	if got := strings.Join(role.SkillNames, ","); got != "openase-platform,install-claude-code,install-codex,setup-git,setup-gh-cli" {
		t.Fatalf("SkillNames=%q", got)
	}

	for _, want := range []string{
		"target machine environment over SSH",
		"makes the machine dispatchable again",
		"Current machine: {{ machine.name }}",
		"project.updates",
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
	if roles[0].Slug != "dispatcher" {
		t.Fatalf("Roles()[0].Slug=%q, want dispatcher", roles[0].Slug)
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
	for _, role := range refreshed {
		if role.HarnessPath != ".openase/harnesses/roles/"+role.Slug+".md" {
			t.Fatalf("role %s HarnessPath=%q", role.Slug, role.HarnessPath)
		}
		if strings.TrimSpace(role.Summary) == "" {
			t.Fatalf("role %s expected non-empty summary", role.Slug)
		}
		if strings.HasPrefix(strings.TrimSpace(role.Content), "---") {
			t.Fatalf("role %s content unexpectedly contains frontmatter", role.Slug)
		}
		if !strings.Contains(role.Content, "# "+role.Name) {
			t.Fatalf("role %s content missing heading %q", role.Slug, role.Name)
		}
	}
}
