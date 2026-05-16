package userapikey

import (
	"slices"
	"testing"

	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
)

func TestScopesForPermissionsCoversExpandedProjectAutomationScopes(t *testing.T) {
	permissions := []humanauthdomain.PermissionKey{
		humanauthdomain.PermissionProjectRead,
		humanauthdomain.PermissionProjectUpdate,
		humanauthdomain.PermissionProjectUpdateRead,
		humanauthdomain.PermissionProjectUpdateCreate,
		humanauthdomain.PermissionRepoRead,
		humanauthdomain.PermissionRepoCreate,
		humanauthdomain.PermissionRepoUpdate,
		humanauthdomain.PermissionRepoDelete,
		humanauthdomain.PermissionStatusRead,
		humanauthdomain.PermissionStatusCreate,
		humanauthdomain.PermissionStatusUpdate,
		humanauthdomain.PermissionStatusDelete,
		humanauthdomain.PermissionTicketRead,
		humanauthdomain.PermissionTicketCreate,
		humanauthdomain.PermissionTicketUpdate,
		humanauthdomain.PermissionWorkflowRead,
		humanauthdomain.PermissionWorkflowCreate,
		humanauthdomain.PermissionWorkflowUpdate,
		humanauthdomain.PermissionWorkflowDelete,
		humanauthdomain.PermissionHarnessRead,
		humanauthdomain.PermissionHarnessUpdate,
		humanauthdomain.PermissionSkillRead,
		humanauthdomain.PermissionSkillCreate,
		humanauthdomain.PermissionSkillUpdate,
		humanauthdomain.PermissionSkillDelete,
		humanauthdomain.PermissionAgentRead,
		humanauthdomain.PermissionAgentCreate,
		humanauthdomain.PermissionAgentUpdate,
		humanauthdomain.PermissionAgentDelete,
		humanauthdomain.PermissionAgentControl,
		humanauthdomain.PermissionJobRead,
		humanauthdomain.PermissionJobCreate,
		humanauthdomain.PermissionJobUpdate,
		humanauthdomain.PermissionJobDelete,
		humanauthdomain.PermissionJobTrigger,
		humanauthdomain.PermissionNotificationRead,
		humanauthdomain.PermissionNotificationCreate,
		humanauthdomain.PermissionNotificationUpdate,
		humanauthdomain.PermissionNotificationDelete,
	}

	got := scopesForPermissions(permissions)
	want := []string{
		string(agentplatformdomain.ScopeActivityRead),
		string(agentplatformdomain.ScopeAgentsCreate),
		string(agentplatformdomain.ScopeAgentsDelete),
		string(agentplatformdomain.ScopeAgentsInterrupt),
		string(agentplatformdomain.ScopeAgentsPause),
		string(agentplatformdomain.ScopeAgentsRead),
		string(agentplatformdomain.ScopeAgentsResume),
		string(agentplatformdomain.ScopeAgentsUpdate),
		string(agentplatformdomain.ScopeNotificationRulesCreate),
		string(agentplatformdomain.ScopeNotificationRulesDelete),
		string(agentplatformdomain.ScopeNotificationRulesList),
		string(agentplatformdomain.ScopeNotificationRulesUpdate),
		string(agentplatformdomain.ScopeProjectUpdatesRead),
		string(agentplatformdomain.ScopeProjectUpdatesWrite),
		string(agentplatformdomain.ScopeProjectsUpdate),
		string(agentplatformdomain.ScopeReposCreate),
		string(agentplatformdomain.ScopeReposDelete),
		string(agentplatformdomain.ScopeReposRead),
		string(agentplatformdomain.ScopeReposUpdate),
		string(agentplatformdomain.ScopeScheduledJobsCreate),
		string(agentplatformdomain.ScopeScheduledJobsDelete),
		string(agentplatformdomain.ScopeScheduledJobsList),
		string(agentplatformdomain.ScopeScheduledJobsTrigger),
		string(agentplatformdomain.ScopeScheduledJobsUpdate),
		string(agentplatformdomain.ScopeSkillsCreate),
		string(agentplatformdomain.ScopeSkillsDelete),
		string(agentplatformdomain.ScopeSkillsImport),
		string(agentplatformdomain.ScopeSkillsList),
		string(agentplatformdomain.ScopeSkillsRead),
		string(agentplatformdomain.ScopeSkillsRefresh),
		string(agentplatformdomain.ScopeSkillsUpdate),
		string(agentplatformdomain.ScopeStatusesCreate),
		string(agentplatformdomain.ScopeStatusesDelete),
		string(agentplatformdomain.ScopeStatusesList),
		string(agentplatformdomain.ScopeStatusesUpdate),
		string(agentplatformdomain.ScopeTicketsCreate),
		string(agentplatformdomain.ScopeTicketsList),
		string(agentplatformdomain.ScopeTicketsUpdate),
		string(agentplatformdomain.ScopeWorkflowsCreate),
		string(agentplatformdomain.ScopeWorkflowsDelete),
		string(agentplatformdomain.ScopeWorkflowsHarnessHistoryRead),
		string(agentplatformdomain.ScopeWorkflowsHarnessRead),
		string(agentplatformdomain.ScopeWorkflowsHarnessUpdate),
		string(agentplatformdomain.ScopeWorkflowsHarnessValidate),
		string(agentplatformdomain.ScopeWorkflowsHarnessVariablesRead),
		string(agentplatformdomain.ScopeWorkflowsList),
		string(agentplatformdomain.ScopeWorkflowsRead),
		string(agentplatformdomain.ScopeWorkflowsUpdate),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("scopesForPermissions() = %#v, want %#v", got, want)
	}
}
