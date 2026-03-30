package ticket

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
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
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
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

func TestServiceRecordUsageEdgeCases(t *testing.T) {
	ctx := context.Background()

	nilService := NewService(nil)
	if _, err := nilService.RecordUsage(ctx, RecordUsageInput{}, nil); err != ErrUnavailable {
		t.Fatalf("RecordUsage(nil service) error = %v, want %v", err, ErrUnavailable)
	}

	client := openTestEntClient(t)
	org, err := client.Organization.Create().
		SetName("GrandCX").
		SetSlug("grandcx").
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
	otherProject, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Other").
		SetSlug("openase-other").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other project: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4-mini").
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
		SetIdentifier("ASE-99").
		SetTitle("Edge cases").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		SetBudgetUsd(10).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	projectAgent, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("coding-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project agent: %v", err)
	}
	otherProjectAgent, err := client.Agent.Create().
		SetProjectID(otherProject.ID).
		SetProviderID(providerItem.ID).
		SetName("coding-02").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other project agent: %v", err)
	}

	service := NewService(client)
	if _, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID: uuid.Nil,
	}, nil); err == nil || err.Error() != "agent_id must be a valid UUID" {
		t.Fatalf("RecordUsage(nil agent) error = %v", err)
	}
	if _, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID:  projectAgent.ID,
		TicketID: uuid.Nil,
	}, nil); err == nil || err.Error() != "ticket_id must be a valid UUID" {
		t.Fatalf("RecordUsage(nil ticket) error = %v", err)
	}
	if _, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID:  projectAgent.ID,
		TicketID: ticketItem.ID,
		Usage:    ticketing.RawUsageDelta{},
	}, nil); err == nil || err.Error() != "usage delta must include input_tokens, output_tokens, or cost_usd" {
		t.Fatalf("RecordUsage(empty usage) error = %v", err)
	}

	negativeTokens := int64(-1)
	if _, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID:  projectAgent.ID,
		TicketID: ticketItem.ID,
		Usage: ticketing.RawUsageDelta{
			InputTokens: &negativeTokens,
		},
	}, nil); err == nil || err.Error() != "input_tokens must be greater than or equal to zero" {
		t.Fatalf("RecordUsage(negative input) error = %v", err)
	}

	if _, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID:  uuid.New(),
		TicketID: ticketItem.ID,
		Usage: ticketing.RawUsageDelta{
			CostUSD: floatPtr(0.25),
		},
	}, nil); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("RecordUsage(missing agent) error = %v", err)
	}

	if _, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID:  otherProjectAgent.ID,
		TicketID: ticketItem.ID,
		Usage: ticketing.RawUsageDelta{
			CostUSD: floatPtr(0.25),
		},
	}, nil); err == nil || !strings.Contains(err.Error(), "does not belong to ticket project") {
		t.Fatalf("RecordUsage(cross-project agent) error = %v", err)
	}

	result, err := service.RecordUsage(ctx, RecordUsageInput{
		AgentID:  projectAgent.ID,
		TicketID: ticketItem.ID,
		Usage: ticketing.RawUsageDelta{
			CostUSD: floatPtr(0.25),
		},
	}, nil)
	if err != nil {
		t.Fatalf("RecordUsage(explicit cost only) error = %v", err)
	}
	if result.Applied.InputTokens != 0 || result.Applied.OutputTokens != 0 || math.Abs(result.Applied.CostUSD-0.25) > 0.0001 {
		t.Fatalf("RecordUsage(explicit cost only) result = %+v", result)
	}
	if result.BudgetExceeded || result.Ticket.RetryPaused || result.Ticket.PauseReason != "" {
		t.Fatalf("RecordUsage(explicit cost only) ticket = %+v", result.Ticket)
	}
	projectAgentAfter, err := client.Agent.Get(ctx, projectAgent.ID)
	if err != nil {
		t.Fatalf("reload project agent: %v", err)
	}
	if projectAgentAfter.TotalTokensUsed != 0 {
		t.Fatalf("expected zero total tokens after explicit-cost-only update, got %d", projectAgentAfter.TotalTokensUsed)
	}
}

func floatPtr(value float64) *float64 {
	return &value
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

	return testPostgres.NewIsolatedEntClient(t)
}
