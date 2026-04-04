package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entmigrate "github.com/BetterAndBetterII/openase/ent/migrate"
	_ "github.com/BetterAndBetterII/openase/ent/runtime"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	agentplatformrepo "github.com/BetterAndBetterII/openase/internal/repo/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func TestNewRootCommandIncludesIssueAgentTokenCommand(t *testing.T) {
	root := NewRootCommand("dev")

	command, _, err := root.Find([]string{"issue-agent-token"})
	if err != nil {
		t.Fatalf("Find(issue-agent-token) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected issue-agent-token command")
	}
}

func TestIssueAgentTokenCommandOutputsJSON(t *testing.T) {
	client, dsn := openCLIEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedCLIPlatformFixture(ctx, t, client)

	t.Setenv("OPENASE_DATABASE_DSN", dsn)
	t.Setenv("OPENASE_SERVER_HOST", "127.0.0.1")
	t.Setenv("OPENASE_SERVER_PORT", "19836")

	command := newIssueAgentTokenCommand(&rootOptions{})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{
		"--agent-id", agentID.String(),
		"--project-id", projectID.String(),
		"--ticket-id", ticketID.String(),
		"--scope", string(agentplatform.ScopeTicketsCreate),
		"--scope", string(agentplatform.ScopeProjectsUpdate),
		"--ttl", "90m",
	})

	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	var response issueAgentTokenResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if response.ProjectID != projectID.String() || response.TicketID != ticketID.String() {
		t.Fatalf("unexpected response identifiers: %+v", response)
	}
	if response.APIURL != "http://127.0.0.1:19836/api/v1/platform" {
		t.Fatalf("unexpected api url %q", response.APIURL)
	}
	if response.Environment["OPENASE_AGENT_TOKEN"] == "" {
		t.Fatalf("expected issued token in environment: %+v", response.Environment)
	}

	claims, err := agentplatform.NewService(agentplatformrepo.NewEntRepository(client)).Authenticate(ctx, response.Token)
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if !claims.HasScope(agentplatform.ScopeTicketsCreate) || !claims.HasScope(agentplatform.ScopeProjectsUpdate) {
		t.Fatalf("unexpected scopes in claims: %+v", claims)
	}
}

func TestIssueAgentTokenCommandOutputsShellExports(t *testing.T) {
	client, dsn := openCLIEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedCLIPlatformFixture(ctx, t, client)

	t.Setenv("OPENASE_DATABASE_DSN", dsn)
	t.Setenv("OPENASE_SERVER_HOST", "0.0.0.0")
	t.Setenv("OPENASE_SERVER_PORT", "40023")

	command := newIssueAgentTokenCommand(&rootOptions{})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs([]string{
		"--agent-id", agentID.String(),
		"--project-id", projectID.String(),
		"--ticket-id", ticketID.String(),
		"--format", "shell",
	})

	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		`export OPENASE_API_URL="http://127.0.0.1:40023/api/v1/platform"`,
		`export OPENASE_PROJECT_ID="` + projectID.String() + `"`,
		`export OPENASE_TICKET_ID="` + ticketID.String() + `"`,
		`export OPENASE_AGENT_TOKEN="ase_agent_`,
		`export OPENASE_AGENT_EXPIRES_AT="`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected shell output to contain %q, got %q", want, output)
		}
	}
}

func TestIssueAgentTokenHelpClarifiesTicketUUIDSemantics(t *testing.T) {
	command := newIssueAgentTokenCommand(&rootOptions{})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)

	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"OPENASE_TICKET_ID",
		"platform routes such as /api/v1/platform/tickets/:ticketId expect that UUID",
		"Do not pass a human-readable ticket identifier such as ASE-2 here.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func openCLIEntClient(t *testing.T) (*ent.Client, string) {
	t.Helper()

	databaseInfo := testPostgres.NewIsolatedDatabase(t)
	dsn := databaseInfo.DSN
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	if err := client.Schema.Create(context.Background(), entmigrate.WithDropColumn(false), entmigrate.WithDropIndex(false)); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	return client, dsn
}

func seedCLIPlatformFixture(ctx context.Context, t *testing.T, client *ent.Client) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("CLI Better And Better").
		SetSlug("cli-better-and-better").
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
		SetName("CLI OpenASE").
		SetSlug("cli-openase").
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
		SetIdentifier("ASE-99").
		SetTitle("CLI smoke ticket").
		SetStatusID(findCLIStatusIDByName(t, statuses, "Todo")).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(provider.ID).
		SetName("coding-cli").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return project.ID, agentItem.ID, ticketItem.ID
}

func findCLIStatusIDByName(t *testing.T, statuses []ticketstatus.Status, want string) uuid.UUID {
	t.Helper()

	for _, status := range statuses {
		if status.Name == want {
			return status.ID
		}
	}

	t.Fatalf("status %q not found in %+v", want, statuses)
	return uuid.Nil
}
