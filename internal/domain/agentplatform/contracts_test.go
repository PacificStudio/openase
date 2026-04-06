package agentplatform

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestClaimsHelpers(t *testing.T) {
	t.Run("ticket agent claims", func(t *testing.T) {
		claims := Claims{
			PrincipalKind: PrincipalKindTicketAgent,
			PrincipalName: "builder",
			Scopes:        []string{string(ScopeTicketsCreate), string(ScopeProjectsUpdate)},
		}

		if !claims.HasScope(ScopeTicketsCreate) {
			t.Fatal("expected tickets.create scope to be present")
		}
		if claims.HasScope(ScopeTicketsList) {
			t.Fatal("did not expect tickets.list scope to be present")
		}
		if got := claims.CreatedBy(); got != "agent:builder" {
			t.Fatalf("CreatedBy() = %q", got)
		}
		if !claims.IsTicketAgent() {
			t.Fatal("expected IsTicketAgent() to be true")
		}
		if claims.IsProjectConversation() {
			t.Fatal("expected IsProjectConversation() to be false")
		}
	})

	t.Run("project conversation claims", func(t *testing.T) {
		claims := Claims{
			PrincipalKind: PrincipalKindProjectConversation,
			PrincipalName: "project-conversation:daily-sync",
		}
		if got := claims.CreatedBy(); got != "project-conversation:daily-sync" {
			t.Fatalf("CreatedBy() = %q", got)
		}
		if claims.IsTicketAgent() {
			t.Fatal("expected IsTicketAgent() to be false")
		}
		if !claims.IsProjectConversation() {
			t.Fatal("expected IsProjectConversation() to be true")
		}
	})

	t.Run("project conversation name gets prefix", func(t *testing.T) {
		claims := Claims{
			PrincipalKind: PrincipalKindProjectConversation,
			PrincipalName: "retro",
		}
		if got := claims.CreatedBy(); got != "project-conversation:retro" {
			t.Fatalf("CreatedBy() = %q", got)
		}
	})
}

func TestBuildEnvironment(t *testing.T) {
	projectID := uuid.New()
	ticketID := uuid.New()

	got := BuildEnvironment(" https://openase.example/api ", " secret-token ", projectID, ticketID)
	want := []string{
		"OPENASE_PROJECT_ID=" + projectID.String(),
		"OPENASE_TICKET_ID=" + ticketID.String(),
		"OPENASE_API_URL=https://openase.example/api",
		"OPENASE_AGENT_TOKEN=secret-token",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("BuildEnvironment() = %#v, want %#v", got, want)
	}

	withoutOptional := BuildEnvironment("   ", "   ", projectID, ticketID)
	if len(withoutOptional) != 2 {
		t.Fatalf("expected only required environment entries, got %#v", withoutOptional)
	}
}

func TestBuildRuntimeEnvironmentIncludesPrincipalMetadata(t *testing.T) {
	projectID := uuid.New()
	ticketID := uuid.New()
	conversationID := uuid.New()

	got := BuildRuntimeEnvironment(RuntimeContractInput{
		PrincipalKind:  PrincipalKindProjectConversation,
		ProjectID:      projectID,
		TicketID:       ticketID,
		ConversationID: conversationID,
		APIURL:         " https://openase.example/api ",
		Token:          " secret-token ",
		Scopes:         []string{string(ScopeTicketsCreate), string(ScopeProjectsUpdate), string(ScopeTicketsCreate)},
	})
	want := []string{
		"OPENASE_PROJECT_ID=" + projectID.String(),
		"OPENASE_TICKET_ID=" + ticketID.String(),
		"OPENASE_CONVERSATION_ID=" + conversationID.String(),
		"OPENASE_API_URL=https://openase.example/api",
		"OPENASE_AGENT_TOKEN=secret-token",
		"OPENASE_PRINCIPAL_KIND=project_conversation",
		"OPENASE_AGENT_SCOPES=projects.update,tickets.create",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("BuildRuntimeEnvironment() = %#v, want %#v", got, want)
	}
}

func TestBuildCapabilityContractReflectsPrincipalSpecificConstraints(t *testing.T) {
	projectConversationContract := BuildCapabilityContract(RuntimeContractInput{
		PrincipalKind:  PrincipalKindProjectConversation,
		ProjectID:      uuid.New(),
		ConversationID: uuid.New(),
		APIURL:         "https://openase.example/api",
		Token:          "token",
		Scopes:         []string{string(ScopeProjectsUpdate), string(ScopeTicketsList)},
	})
	if !containsAll(projectConversationContract,
		"Current principal: `project_conversation`",
		"`OPENASE_CONVERSATION_ID`",
		"`OPENASE_TICKET_ID` only when this Project AI session is ticket-focused",
		"`projects.update`",
		"Ticket-runtime-only routes can reject this principal kind",
	) {
		t.Fatalf("project conversation contract = %q", projectConversationContract)
	}

	ticketAgentContract := BuildCapabilityContract(RuntimeContractInput{
		PrincipalKind: PrincipalKindTicketAgent,
		ProjectID:     uuid.New(),
		TicketID:      uuid.New(),
		APIURL:        "https://openase.example/api",
		Token:         "token",
		Scopes:        []string{string(ScopeTicketsCreate), string(ScopeTicketsUpdateSelf)},
	})
	if !containsAll(ticketAgentContract,
		"Current principal: `ticket_agent`",
		"`OPENASE_TICKET_ID`",
		"`tickets.update.self`",
		"Current-ticket routes are limited to the ticket identified by `OPENASE_TICKET_ID`.",
	) {
		t.Fatalf("ticket agent contract = %q", ticketAgentContract)
	}
}

func TestBuildCapabilityContractOmitsUndeclaredOptionalEnvironmentAndScopes(t *testing.T) {
	contract := BuildCapabilityContract(RuntimeContractInput{
		PrincipalKind: PrincipalKindTicketAgent,
		ProjectID:     uuid.New(),
		TicketID:      uuid.New(),
		Scopes:        []string{"  ", "", "tickets.create", "tickets.create"},
	})

	if strings.Contains(contract, "`OPENASE_API_URL`") {
		t.Fatalf("contract unexpectedly mentioned OPENASE_API_URL: %q", contract)
	}
	if strings.Contains(contract, "`OPENASE_AGENT_TOKEN`") {
		t.Fatalf("contract unexpectedly mentioned OPENASE_AGENT_TOKEN: %q", contract)
	}
	if strings.Contains(contract, "`OPENASE_CONVERSATION_ID`") {
		t.Fatalf("contract unexpectedly mentioned OPENASE_CONVERSATION_ID: %q", contract)
	}
	if strings.Contains(contract, "`OPENASE_AGENT_SCOPES`\n- none declared") {
		t.Fatalf("contract should list normalized scopes instead of none declared: %q", contract)
	}
	if !containsAll(contract,
		"`OPENASE_TICKET_ID`",
		"`OPENASE_AGENT_SCOPES`",
		"`tickets.create`",
	) {
		t.Fatalf("contract missing normalized scope content: %q", contract)
	}
}

func TestBuildCapabilityContractListsNoneDeclaredWhenScopesEmpty(t *testing.T) {
	contract := BuildCapabilityContract(RuntimeContractInput{
		PrincipalKind:  PrincipalKindProjectConversation,
		ProjectID:      uuid.New(),
		ConversationID: uuid.New(),
		Scopes:         nil,
	})

	if !containsAll(contract,
		"Current principal: `project_conversation`",
		"`OPENASE_CONVERSATION_ID`",
		"- none declared",
		"`OPENASE_TICKET_ID` only when this Project AI session is ticket-focused",
	) {
		t.Fatalf("project conversation contract with empty scopes = %q", contract)
	}
	if strings.Contains(contract, "`OPENASE_AGENT_SCOPES`") {
		t.Fatalf("contract should not mention OPENASE_AGENT_SCOPES when scopes are absent: %q", contract)
	}
}

func TestRuntimeContractInputStringUsesNormalizedScopeOrdering(t *testing.T) {
	input := RuntimeContractInput{
		PrincipalKind:  PrincipalKindProjectConversation,
		ProjectID:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		TicketID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		ConversationID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Scopes:         []string{" tickets.list ", "", "tickets.create", "tickets.list"},
	}

	got := input.String()
	want := "principal=project_conversation project=11111111-1111-1111-1111-111111111111 ticket=22222222-2222-2222-2222-222222222222 conversation=33333333-3333-3333-3333-333333333333 scopes=tickets.create,tickets.list"
	if got != want {
		t.Fatalf("RuntimeContractInput.String() = %q, want %q", got, want)
	}
}

func containsAll(text string, snippets ...string) bool {
	for _, snippet := range snippets {
		if !strings.Contains(text, snippet) {
			return false
		}
	}
	return true
}

func TestDefaultAgentScopes(t *testing.T) {
	got := DefaultAgentScopes()
	want := []string{
		string(ScopeTicketsCreate),
		string(ScopeTicketsList),
		string(ScopeTicketsReportUsage),
		string(ScopeTicketsUpdateSelf),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("DefaultAgentScopes() = %#v, want %#v", got, want)
	}
}

func TestSupportedAgentScopes(t *testing.T) {
	got := SupportedAgentScopes()
	want := []string{
		string(ScopeAgentsInterrupt),
		string(ScopeActivityRead),
		string(ScopeProjectsAddRepo),
		string(ScopeProjectsUpdate),
		string(ScopeReposCreate),
		string(ScopeReposDelete),
		string(ScopeReposRead),
		string(ScopeReposUpdate),
		string(ScopeScheduledJobsCreate),
		string(ScopeScheduledJobsDelete),
		string(ScopeScheduledJobsList),
		string(ScopeScheduledJobsTrigger),
		string(ScopeScheduledJobsUpdate),
		string(ScopeSkillsBind),
		string(ScopeSkillsCreate),
		string(ScopeSkillsDelete),
		string(ScopeSkillsDisable),
		string(ScopeSkillsEnable),
		string(ScopeSkillsImport),
		string(ScopeSkillsList),
		string(ScopeSkillsRead),
		string(ScopeSkillsRefresh),
		string(ScopeSkillsUpdate),
		string(ScopeStatusesCreate),
		string(ScopeStatusesDelete),
		string(ScopeStatusesList),
		string(ScopeStatusesReset),
		string(ScopeStatusesUpdate),
		string(ScopeTicketRepoScopesCreate),
		string(ScopeTicketRepoScopesDelete),
		string(ScopeTicketRepoScopesList),
		string(ScopeTicketRepoScopesUpdate),
		string(ScopeTicketsCreate),
		string(ScopeTicketsList),
		string(ScopeTicketsUpdate),
		string(ScopeTicketsReportUsage),
		string(ScopeTicketsUpdateSelf),
		string(ScopeWorkflowsCreate),
		string(ScopeWorkflowsDelete),
		string(ScopeWorkflowsHarnessHistoryRead),
		string(ScopeWorkflowsHarnessRead),
		string(ScopeWorkflowsHarnessUpdate),
		string(ScopeWorkflowsHarnessValidate),
		string(ScopeWorkflowsHarnessVariablesRead),
		string(ScopeWorkflowsList),
		string(ScopeWorkflowsRead),
		string(ScopeWorkflowsUpdate),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SupportedAgentScopes() = %#v, want %#v", got, want)
	}
}
