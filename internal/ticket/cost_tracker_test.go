package ticket

import (
	"context"
	"fmt"
	"math"
	"net"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
)

func TestServiceRecordUsageAccumulatesTokensCostAndBudgetPause(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)

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
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		SetCostPerInputToken(0.001).
		SetCostPerOutputToken(0.002).
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-42").
		SetTitle("Track costs").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		SetBudgetUsd(0.20).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("coding-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	service := NewService(client)
	inputTokens := int64(120)
	outputTokens := int64(45)
	result, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID:  agentItem.ID,
		TicketID: ticketItem.ID,
		Usage: ticketing.RawUsageDelta{
			InputTokens:  &inputTokens,
			OutputTokens: &outputTokens,
		},
	}, nil)
	if err != nil {
		t.Fatalf("RecordUsage returned error: %v", err)
	}

	if result.Applied.InputTokens != 120 || result.Applied.OutputTokens != 45 {
		t.Fatalf("unexpected applied usage: %+v", result.Applied)
	}
	if !result.BudgetExceeded || result.Ticket.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("expected budget pause result, got %+v", result)
	}
	if math.Abs(result.Applied.CostUSD-0.21) > 0.0001 {
		t.Fatalf("expected applied cost 0.21, got %.2f", result.Applied.CostUSD)
	}
	if result.Ticket.CostTokensInput != 120 || result.Ticket.CostTokensOutput != 45 {
		t.Fatalf("unexpected ticket token totals: %+v", result.Ticket)
	}
	if math.Abs(result.Ticket.CostAmount-0.21) > 0.0001 {
		t.Fatalf("expected ticket cost 0.21, got %.2f", result.Ticket.CostAmount)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.TotalTokensUsed != 165 {
		t.Fatalf("expected total tokens 165, got %d", agentAfter.TotalTokensUsed)
	}
}

func findStatusIDByName(t *testing.T, items []ticketstatus.Status, want string) uuid.UUID {
	t.Helper()

	for _, item := range items {
		if item.Name == want {
			return item.ID
		}
	}

	t.Fatalf("missing status %q in %+v", want, items)
	return uuid.Nil
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
		t.Fatalf("listen for free port: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	if port < 0 || port > 65535 {
		t.Fatalf("listener returned out-of-range port: %d", port)
	}

	parsed, err := strconv.ParseUint(strconv.Itoa(port), 10, 32)
	if err != nil {
		t.Fatalf("parse free port %d: %v", port, err)
	}

	return uint32(parsed)
}
