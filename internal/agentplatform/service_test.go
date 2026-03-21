package agentplatform

import (
	"context"
	"fmt"
	"math"
	"net"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
)

func TestIssueAndAuthenticateToken(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(client)
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

func TestIssueTokenUsesDefaultScopes(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID := seedAgentPlatformFixture(ctx, t, client)

	service := NewService(client)
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

	service := NewService(client)
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

	service := NewService(client)
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

	service := NewService(client)
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

func seedAgentPlatformFixture(ctx context.Context, t *testing.T, client *ent.Client) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
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
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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

	port := freePort(t)
	dataDir := t.TempDir()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Username("postgres").
			Password("postgres").
			Database("openase").
			RuntimePath(filepath.Join(dataDir, "runtime")).
			BinariesPath(filepath.Join(dataDir, "binaries")).
			DataPath(filepath.Join(dataDir, "data")),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})

	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/openase?sslmode=disable", port)
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})

	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return client
}

func freePort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP address, got %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > math.MaxUint16 {
		t.Fatalf("expected TCP port in uint16 range, got %d", tcpAddr.Port)
	}
	return uint32(tcpAddr.Port) //nolint:gosec // validated above to fit the TCP port range
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
