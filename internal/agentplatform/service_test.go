package agentplatform

import (
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	agentplatformrepo "github.com/BetterAndBetterII/openase/internal/repo/agentplatform"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

var testPostgres *pgtest.Server

func TestMain(m *testing.M) {
	os.Exit(pgtest.RunTestMain(m, "agentplatform", func(server *pgtest.Server) {
		testPostgres = server
	}))
}

func TestIssueAndAuthenticateToken(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(agentplatformrepo.NewEntRepository(client))
	service.now = func() time.Time {
		return time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	}

	issued, err := service.IssueToken(ctx, IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  ticketID,
		Scopes:    []string{string(ScopeProjectsUpdate), string(ScopeTicketsCreate)},
		TTL:       time.Hour,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	claims, err := service.Authenticate(ctx, issued.Token)
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}

	if claims.AgentID != agentID || claims.ProjectID != projectID || claims.TicketID != ticketID {
		t.Fatalf("unexpected claims payload: %+v", claims)
	}
	if !claims.HasScope(ScopeProjectsUpdate) || !claims.HasScope(ScopeTicketsCreate) {
		t.Fatalf("expected custom scopes in %+v", claims.Scopes)
	}
	if claims.CreatedBy() != "agent:coding-01" {
		t.Fatalf("CreatedBy=%q, want agent:coding-01", claims.CreatedBy())
	}
}

func TestIssueAndAuthenticateProjectConversationToken(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, _, _ := seedAgentPlatformFixture(ctx, t, client)
	providerID, err := client.AgentProvider.Query().OnlyID(ctx)
	if err != nil {
		t.Fatalf("query provider: %v", err)
	}
	repoStore := chatrepo.NewEntRepository(client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  projectID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	service := NewService(agentplatformrepo.NewEntRepository(client))
	issued, err := service.IssueToken(ctx, IssueInput{
		PrincipalKind:  PrincipalKindProjectConversation,
		PrincipalID:    conversation.ID,
		ProjectID:      projectID,
		ConversationID: conversation.ID,
	})
	if err != nil {
		t.Fatalf("IssueToken(project conversation) returned error: %v", err)
	}

	claims, err := service.Authenticate(ctx, issued.Token)
	if err != nil {
		t.Fatalf("Authenticate(project conversation) returned error: %v", err)
	}
	if claims.PrincipalKind != PrincipalKindProjectConversation || claims.PrincipalID != conversation.ID || claims.ConversationID != conversation.ID {
		t.Fatalf("unexpected project conversation claims: %+v", claims)
	}
	if claims.TicketID != uuid.Nil || claims.AgentID != uuid.Nil {
		t.Fatalf("project conversation claims should not carry ticket/agent ids: %+v", claims)
	}

	wantScopes := DefaultScopesForPrincipalKind(PrincipalKindProjectConversation)
	slices.Sort(wantScopes)
	gotScopes := append([]string(nil), claims.Scopes...)
	slices.Sort(gotScopes)
	if !slices.Equal(gotScopes, wantScopes) {
		t.Fatalf("Scopes=%v, want %v", gotScopes, wantScopes)
	}
	fullProjectScopes := SupportedScopes()
	slices.Sort(fullProjectScopes)
	if !slices.Equal(gotScopes, fullProjectScopes) {
		t.Fatalf("project conversation scopes=%v, want full supported scopes %v", gotScopes, fullProjectScopes)
	}
	if claims.CreatedBy() != "project-conversation:"+conversation.ID.String() {
		t.Fatalf("CreatedBy() = %q", claims.CreatedBy())
	}
}

func TestIssueTokenRejectsMissingProjectConversationPrincipal(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, _, _ := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(agentplatformrepo.NewEntRepository(client))
	_, err := service.IssueToken(ctx, IssueInput{
		PrincipalKind:  PrincipalKindProjectConversation,
		PrincipalID:    uuid.New(),
		ProjectID:      projectID,
		ConversationID: uuid.New(),
	})
	if !errors.Is(err, ErrPrincipalNotFound) {
		t.Fatalf("IssueToken(missing project conversation principal) error = %v, want %v", err, ErrPrincipalNotFound)
	}
}

func TestIssueTokenUsesDefaultScopes(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(agentplatformrepo.NewEntRepository(client))
	issued, err := service.IssueToken(ctx, IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  ticketID,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	claims, err := service.Authenticate(ctx, issued.Token)
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}

	got := append([]string(nil), claims.Scopes...)
	want := append([]string(nil), DefaultScopes()...)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("Scopes=%v, want %v", got, want)
	}
}

func TestIssueTokenConstrainsScopesToWhitelist(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(agentplatformrepo.NewEntRepository(client))
	issued, err := service.IssueToken(ctx, IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  ticketID,
		Scopes: []string{
			string(ScopeProjectsUpdate),
			string(ScopeTicketsCreate),
			string(ScopeTicketsList),
		},
		ScopeWhitelist: ScopeWhitelist{
			Configured: true,
			Scopes: []string{
				string(ScopeProjectsUpdate),
				string(ScopeTicketsList),
			},
		},
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	claims, err := service.Authenticate(ctx, issued.Token)
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}

	got := append([]string(nil), claims.Scopes...)
	want := []string{string(ScopeProjectsUpdate), string(ScopeTicketsList)}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("Scopes=%v, want %v", got, want)
	}
}

func TestAuthenticateRejectsExpiredToken(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(agentplatformrepo.NewEntRepository(client))
	baseTime := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return baseTime }

	issued, err := service.IssueToken(ctx, IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  ticketID,
		TTL:       time.Minute,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	service.now = func() time.Time { return baseTime.Add(2 * time.Minute) }
	if _, err := service.Authenticate(ctx, issued.Token); err != ErrTokenExpired {
		t.Fatalf("Authenticate error=%v, want %v", err, ErrTokenExpired)
	}
}

func TestProjectTokenInventorySummarizesProjectExposure(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(agentplatformrepo.NewEntRepository(client))
	baseTime := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return baseTime }

	activeToken, err := service.IssueToken(ctx, IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  ticketID,
		Scopes:    []string{string(ScopeProjectsUpdate)},
		TTL:       2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("IssueToken(active) returned error: %v", err)
	}

	if _, err := service.Authenticate(ctx, activeToken.Token); err != nil {
		t.Fatalf("Authenticate(active) returned error: %v", err)
	}

	service.now = func() time.Time { return baseTime.Add(-48 * time.Hour) }
	if _, err := service.IssueToken(ctx, IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  ticketID,
		TTL:       time.Hour,
	}); err != nil {
		t.Fatalf("IssueToken(expired) returned error: %v", err)
	}

	service.now = func() time.Time { return baseTime }
	inventory, err := service.ProjectTokenInventory(ctx, projectID)
	if err != nil {
		t.Fatalf("ProjectTokenInventory returned error: %v", err)
	}

	if inventory.ActiveTokenCount != 1 || inventory.ExpiredTokenCount != 1 {
		t.Fatalf("unexpected token counts: %+v", inventory)
	}
	if inventory.LastIssuedAt == nil {
		t.Fatalf("expected LastIssuedAt to be populated, got %+v", inventory)
	}
	if inventory.LastUsedAt == nil || !inventory.LastUsedAt.Equal(baseTime) {
		t.Fatalf("LastUsedAt=%v, want %v", inventory.LastUsedAt, baseTime)
	}
	if !slices.Equal(inventory.DefaultScopes, DefaultScopes()) {
		t.Fatalf("DefaultScopes=%v, want %v", inventory.DefaultScopes, DefaultScopes())
	}
	if !slices.Equal(inventory.PrivilegedScopes, PrivilegedScopes()) {
		t.Fatalf("PrivilegedScopes=%v, want %v", inventory.PrivilegedScopes, PrivilegedScopes())
	}
}

func TestParseBearerTokenRejectsInvalidHeader(t *testing.T) {
	if _, err := ParseBearerToken("Basic nope"); err != ErrInvalidToken {
		t.Fatalf("ParseBearerToken error=%v, want %v", err, ErrInvalidToken)
	}
}

func TestBuildEnvironmentIncludesPlatformVariables(t *testing.T) {
	projectID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ticketID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	environment := BuildEnvironment("http://localhost:19836/api/v1/platform", "ase_agent_token", projectID, ticketID)
	for _, want := range []string{
		"OPENASE_API_URL=http://localhost:19836/api/v1/platform",
		"OPENASE_AGENT_TOKEN=ase_agent_token",
		"OPENASE_PROJECT_ID=11111111-1111-1111-1111-111111111111",
		"OPENASE_TICKET_ID=22222222-2222-2222-2222-222222222222",
	} {
		if !slices.Contains(environment, want) {
			t.Fatalf("expected environment to contain %q, got %v", want, environment)
		}
	}
}

func TestAgentPlatformUtilityAndFailurePaths(t *testing.T) {
	t.Run("scope helpers and parser failures", func(t *testing.T) {
		gotSupported := SupportedScopes()
		wantSupported := []string{
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
			string(ScopeSkillsRefine),
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
		if !slices.Equal(gotSupported, wantSupported) {
			t.Fatalf("SupportedScopes() = %v, want %v", gotSupported, wantSupported)
		}
		gotProjectConversationSupported := SupportedScopesForPrincipalKind(PrincipalKindProjectConversation)
		if !slices.Equal(gotProjectConversationSupported, wantSupported) {
			t.Fatalf("SupportedScopesForPrincipalKind(project conversation) = %v, want %v", gotProjectConversationSupported, wantSupported)
		}
		gotProjectConversationDefaults := DefaultScopesForPrincipalKind(PrincipalKindProjectConversation)
		if !slices.Equal(gotProjectConversationDefaults, wantSupported) {
			t.Fatalf("DefaultScopesForPrincipalKind(project conversation) = %v, want %v", gotProjectConversationDefaults, wantSupported)
		}
		gotGroups := SupportedScopeGroups()
		if len(gotGroups) == 0 {
			t.Fatal("SupportedScopeGroups() expected non-empty groups")
		}
		if gotGroups[0].Category != "activity" || !slices.Equal(gotGroups[0].Scopes, []string{string(ScopeActivityRead)}) {
			t.Fatalf("SupportedScopeGroups()[0] = %+v", gotGroups[0])
		}
		lastGroup := gotGroups[len(gotGroups)-1]
		if lastGroup.Category != "workflows" || !slices.Contains(lastGroup.Scopes, string(ScopeWorkflowsHarnessVariablesRead)) {
			t.Fatalf("SupportedScopeGroups()[last] = %+v", lastGroup)
		}

		if _, err := parseExplicitScopes([]string{" "}); !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("parseExplicitScopes(blank) error = %v, want %v", err, ErrInvalidScope)
		}
		if _, err := parseExplicitScopes([]string{"tickets.nope"}); !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("parseExplicitScopes(unsupported) error = %v, want %v", err, ErrInvalidScope)
		}
		defaultScopes, err := parseScopes(nil)
		if err != nil {
			t.Fatalf("parseScopes(nil) error = %v", err)
		}
		if _, err := constrainScopes(defaultScopes, ScopeWhitelist{Configured: true, Scopes: []string{"bad"}}); !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("constrainScopes(invalid whitelist) error = %v, want %v", err, ErrInvalidScope)
		}
		if token, err := ParseToken("  " + TokenPrefix + "trimmed  "); err != nil || token != TokenPrefix+"trimmed" {
			t.Fatalf("ParseToken(trimmed) = %q, %v", token, err)
		}
		if _, err := ParseToken("not_a_token"); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("ParseToken(invalid) error = %v, want %v", err, ErrInvalidToken)
		}
		if _, err := ParseBearerToken("Bearer"); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("ParseBearerToken(missing token) error = %v, want %v", err, ErrInvalidToken)
		}

		claims := Claims{Scopes: []string{string(ScopeTicketsList)}}
		if !claims.HasScope(ScopeTicketsList) {
			t.Fatal("Claims.HasScope() expected true for present scope")
		}
		if claims.HasScope(ScopeProjectsUpdate) {
			t.Fatal("Claims.HasScope() expected false for missing scope")
		}
	})

	t.Run("service error branches", func(t *testing.T) {
		ctx := context.Background()
		projectID := uuid.New()

		var nilService *Service
		if _, err := nilService.IssueToken(ctx, IssueInput{}); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("nil IssueToken() error = %v, want %v", err, ErrUnavailable)
		}
		if _, err := nilService.Authenticate(ctx, TokenPrefix+"missing"); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("nil Authenticate() error = %v, want %v", err, ErrUnavailable)
		}
		if _, err := nilService.ProjectTokenInventory(ctx, projectID); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("nil ProjectTokenInventory() error = %v, want %v", err, ErrUnavailable)
		}

		client := openTestEntClient(t)
		projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)
		service := NewService(agentplatformrepo.NewEntRepository(client))
		service.now = func() time.Time { return time.Date(2026, 3, 27, 16, 0, 0, 0, time.UTC) }

		if _, err := service.IssueToken(ctx, IssueInput{ProjectID: projectID, TicketID: ticketID}); err == nil || err.Error() != "agent_id must be a valid UUID" {
			t.Fatalf("IssueToken(nil agent) error = %v", err)
		}
		if _, err := service.IssueToken(ctx, IssueInput{AgentID: agentID, TicketID: ticketID}); err == nil || err.Error() != "project_id must be a valid UUID" {
			t.Fatalf("IssueToken(nil project) error = %v", err)
		}
		if _, err := service.IssueToken(ctx, IssueInput{AgentID: agentID, ProjectID: projectID}); err == nil || err.Error() != "ticket_id must be a valid UUID" {
			t.Fatalf("IssueToken(nil ticket) error = %v", err)
		}
		if _, err := service.IssueToken(ctx, IssueInput{
			AgentID:   uuid.New(),
			ProjectID: projectID,
			TicketID:  ticketID,
		}); !errors.Is(err, ErrAgentNotFound) {
			t.Fatalf("IssueToken(missing agent) error = %v, want %v", err, ErrAgentNotFound)
		}
		if _, err := service.IssueToken(ctx, IssueInput{
			AgentID:   agentID,
			ProjectID: uuid.New(),
			TicketID:  ticketID,
		}); !errors.Is(err, ErrProjectMismatch) {
			t.Fatalf("IssueToken(project mismatch) error = %v, want %v", err, ErrProjectMismatch)
		}

		service.rng = failingReader{}
		if _, err := service.IssueToken(ctx, IssueInput{
			AgentID:   agentID,
			ProjectID: projectID,
			TicketID:  ticketID,
		}); err == nil || !strings.Contains(err.Error(), "generate agent token bytes") {
			t.Fatalf("IssueToken(rng failure) error = %v", err)
		}
		service.rng = strings.NewReader(strings.Repeat("a", 24))

		issued, err := service.IssueToken(ctx, IssueInput{
			AgentID:   agentID,
			ProjectID: projectID,
			TicketID:  ticketID,
		})
		if err != nil {
			t.Fatalf("IssueToken(valid) error = %v", err)
		}
		if _, err := service.Authenticate(ctx, TokenPrefix+"missing"); !errors.Is(err, ErrTokenNotFound) {
			t.Fatalf("Authenticate(missing token) error = %v, want %v", err, ErrTokenNotFound)
		}
		if _, err := service.Authenticate(ctx, "bad"); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("Authenticate(invalid token) error = %v, want %v", err, ErrInvalidToken)
		}

		currentProject, err := client.Project.Get(ctx, projectID)
		if err != nil {
			t.Fatalf("load current project: %v", err)
		}
		otherProject, err := client.Project.Create().
			SetOrganizationID(currentProject.OrganizationID).
			SetName("Other Project").
			SetSlug("other-project").
			Save(ctx)
		if err != nil {
			t.Fatalf("create other project: %v", err)
		}
		if _, err := client.Agent.UpdateOneID(agentID).SetProjectID(otherProject.ID).Save(ctx); err != nil {
			t.Fatalf("rebind agent project: %v", err)
		}
		if _, err := service.Authenticate(ctx, issued.Token); !errors.Is(err, ErrProjectMismatch) {
			t.Fatalf("Authenticate(project mismatch) error = %v, want %v", err, ErrProjectMismatch)
		}
		if _, err := service.ProjectTokenInventory(ctx, uuid.Nil); err == nil || err.Error() != "project_id must be a valid UUID" {
			t.Fatalf("ProjectTokenInventory(nil project) error = %v", err)
		}
	})
}

type failingReader struct{}

func (failingReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func seedAgentPlatformFixture(ctx context.Context, t *testing.T, client *ent.Client) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-42").
		SetTitle("Build platform API").
		SetStatusID(findStatusIDByName(t, statuses, "Todo")).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(provider.ID).
		SetName("coding-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return project.ID, agentItem.ID, ticketItem.ID
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func findStatusIDByName(t *testing.T, statuses []ticketstatus.Status, name string) uuid.UUID {
	t.Helper()

	for _, status := range statuses {
		if status.Name == name {
			return status.ID
		}
	}
	t.Fatalf("status %q not found in %+v", name, statuses)
	return uuid.UUID{}
}
