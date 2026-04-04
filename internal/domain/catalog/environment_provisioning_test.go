package catalog

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPlanMachineEnvironmentProvisioningBuildsRunnablePlan(t *testing.T) {
	machine := Machine{
		ID:     uuid.New(),
		Name:   "builder-01",
		Host:   "10.0.1.13",
		Status: MachineStatusDegraded,
		Resources: map[string]any{
			"last_success": true,
			"monitor": map[string]any{
				"l1": map[string]any{
					"reachable": true,
				},
			},
			"agent_environment": map[string]any{
				"claude_code": map[string]any{
					"installed":   false,
					"auth_status": "unknown",
				},
				"codex": map[string]any{
					"installed":   true,
					"auth_status": "not_logged_in",
				},
			},
			"full_audit": map[string]any{
				"git": map[string]any{
					"installed":  true,
					"user_name":  "OpenASE",
					"user_email": "",
				},
				"gh_cli": map[string]any{
					"installed":   true,
					"auth_status": "not_logged_in",
				},
				"network": map[string]any{
					"github_reachable": false,
					"pypi_reachable":   false,
					"npm_reachable":    false,
				},
			},
		},
	}

	plan := PlanMachineEnvironmentProvisioning(machine)

	if !plan.Available || !plan.Needed || !plan.Runnable {
		t.Fatalf("expected available runnable plan, got %+v", plan)
	}
	if plan.RoleSlug != EnvironmentProvisionerRoleSlug || plan.RoleName != EnvironmentProvisionerRoleName {
		t.Fatalf("unexpected role identity: %+v", plan)
	}
	if len(plan.Issues) != 4 {
		t.Fatalf("expected 4 issues, got %+v", plan.Issues)
	}

	expectedSkills := []string{
		EnvironmentProvisionerSkillInstallClaude,
		EnvironmentProvisionerSkillInstallCodex,
		EnvironmentProvisionerSkillSetupGit,
		EnvironmentProvisionerSkillSetupGitHubCLI,
	}
	if len(plan.RequiredSkills) != len(expectedSkills) {
		t.Fatalf("expected skills %v, got %v", expectedSkills, plan.RequiredSkills)
	}
	for index, want := range expectedSkills {
		if plan.RequiredSkills[index] != want {
			t.Fatalf("required skills[%d]=%q, want %q", index, plan.RequiredSkills[index], want)
		}
	}

	if !strings.Contains(plan.TicketDescription, "Claude Code is not installed") {
		t.Fatalf("expected ticket description to include Claude Code issue, got %q", plan.TicketDescription)
	}
	if !strings.Contains(plan.TicketDescription, "PyPI is unreachable") {
		t.Fatalf("expected ticket description to include network note, got %q", plan.TicketDescription)
	}
}

func TestPlanMachineEnvironmentProvisioningMarksOfflineMachineUnrunnable(t *testing.T) {
	machine := Machine{
		ID:     uuid.New(),
		Name:   "builder-02",
		Host:   "10.0.1.14",
		Status: MachineStatusOffline,
		Resources: map[string]any{
			"last_success": false,
			"agent_environment": map[string]any{
				"claude_code": map[string]any{
					"installed":   false,
					"auth_status": "unknown",
				},
				"codex": map[string]any{
					"installed":   true,
					"auth_status": "logged_in",
				},
			},
		},
	}

	plan := PlanMachineEnvironmentProvisioning(machine)

	if !plan.Available || !plan.Needed {
		t.Fatalf("expected offline machine to still yield actionable plan, got %+v", plan)
	}
	if plan.Runnable {
		t.Fatalf("expected offline machine plan to be unrunnable, got %+v", plan)
	}
	if len(plan.Notes) == 0 || !strings.Contains(plan.Notes[len(plan.Notes)-1], "SSH agent runner") {
		t.Fatalf("expected note about SSH runner availability, got %+v", plan.Notes)
	}
}

func TestPlanMachineEnvironmentProvisioningHandlesMissingSnapshots(t *testing.T) {
	plan := PlanMachineEnvironmentProvisioning(Machine{
		ID:        uuid.New(),
		Name:      "builder-03",
		Host:      "10.0.1.15",
		Status:    MachineStatusOnline,
		Resources: map[string]any{},
	})

	if plan.Available || plan.Needed || plan.Runnable {
		t.Fatalf("expected unavailable empty plan, got %+v", plan)
	}
	if !strings.Contains(plan.Summary, "not available") {
		t.Fatalf("expected missing snapshot summary, got %q", plan.Summary)
	}
}
