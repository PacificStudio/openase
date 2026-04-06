package ticket

import (
	"context"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

func diagnosisPriorityPtr(priority Priority) *Priority {
	return &priority
}

func TestGetPickupDiagnosisRetryBackoff(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	ticketItem := createDiagnosisTicket(ctx, t, service, fixture)
	nextRetryAt := time.Now().UTC().Add(20 * time.Minute).Truncate(time.Second)
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetNextRetryAt(nextRetryAt).
		SetRetryPaused(false).
		Save(ctx); err != nil {
		t.Fatalf("schedule retry backoff: %v", err)
	}

	diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("GetPickupDiagnosis() error = %v", err)
	}

	if diagnosis.State != PickupDiagnosisStateWaiting || diagnosis.PrimaryReasonCode != PickupDiagnosisReasonRetryBackoff {
		t.Fatalf("retry backoff diagnosis = %+v", diagnosis)
	}
	if diagnosis.Retry.NextRetryAt == nil || !diagnosis.Retry.NextRetryAt.Equal(nextRetryAt) {
		t.Fatalf("retry backoff next_retry_at = %+v", diagnosis.Retry.NextRetryAt)
	}
}

func TestGetPickupDiagnosisRetryPausedRepeatedStalls(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	ticketItem := createDiagnosisTicket(ctx, t, service, fixture)
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonRepeatedStalls.String()).
		ClearNextRetryAt().
		Save(ctx); err != nil {
		t.Fatalf("pause repeated stalls retry: %v", err)
	}

	diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("GetPickupDiagnosis() error = %v", err)
	}

	if diagnosis.State != PickupDiagnosisStateBlocked || diagnosis.PrimaryReasonCode != PickupDiagnosisReasonRetryPausedRepeatedStalls {
		t.Fatalf("repeated stalls diagnosis = %+v", diagnosis)
	}
	if diagnosis.NextActionHint == "" {
		t.Fatalf("expected manual retry hint, got %+v", diagnosis)
	}
}

func TestGetPickupDiagnosisRetryPausedInterrupted(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	ticketItem := createDiagnosisTicket(ctx, t, service, fixture)
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonUserInterrupted.String()).
		ClearNextRetryAt().
		Save(ctx); err != nil {
		t.Fatalf("pause interrupted retry: %v", err)
	}

	diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("GetPickupDiagnosis() error = %v", err)
	}

	if diagnosis.State != PickupDiagnosisStateBlocked || diagnosis.PrimaryReasonCode != PickupDiagnosisReasonRetryPausedInterrupted {
		t.Fatalf("interrupted diagnosis = %+v", diagnosis)
	}
	if diagnosis.NextActionHint == "" {
		t.Fatalf("expected interrupted retry hint, got %+v", diagnosis)
	}
}

func TestGetPickupDiagnosisBlockedDependency(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	blocker := createDiagnosisTicket(ctx, t, service, fixture)
	target := createDiagnosisTicket(ctx, t, service, fixture)
	if _, err := client.TicketDependency.Create().
		SetSourceTicketID(blocker.ID).
		SetTargetTicketID(target.ID).
		SetType(entticketdependency.TypeBlocks).
		Save(ctx); err != nil {
		t.Fatalf("create blocks dependency: %v", err)
	}

	diagnosis, err := service.GetPickupDiagnosis(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetPickupDiagnosis() error = %v", err)
	}

	if diagnosis.PrimaryReasonCode != PickupDiagnosisReasonBlockedDependency || len(diagnosis.BlockedBy) != 1 {
		t.Fatalf("blocked dependency diagnosis = %+v", diagnosis)
	}
	if diagnosis.BlockedBy[0].ID != blocker.ID || diagnosis.BlockedBy[0].Identifier != blocker.Identifier {
		t.Fatalf("blocked dependency metadata = %+v", diagnosis.BlockedBy)
	}
}

func TestGetPickupDiagnosisNoMatchingActiveWorkflow(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	ticketItem, err := service.Create(ctx, CreateInput{
		ProjectID: fixture.projectID,
		Title:     "Needs dispatcher",
		StatusID:  &fixture.backlogID,
		Priority:  diagnosisPriorityPtr(PriorityMedium),
		Type:      TypeFeature,
	})
	if err != nil {
		t.Fatalf("Create(backlog) error = %v", err)
	}

	diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("GetPickupDiagnosis() error = %v", err)
	}

	if diagnosis.State != PickupDiagnosisStateUnavailable || diagnosis.PrimaryReasonCode != PickupDiagnosisReasonNoMatchingActiveWorkflow {
		t.Fatalf("no matching workflow diagnosis = %+v", diagnosis)
	}
}

func TestGetPickupDiagnosisAgentPaused(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	ticketItem := createDiagnosisTicket(ctx, t, service, fixture)
	if _, err := client.Agent.UpdateOneID(fixture.agentID).
		SetRuntimeControlState(entagent.RuntimeControlStatePaused).
		Save(ctx); err != nil {
		t.Fatalf("pause workflow agent: %v", err)
	}

	diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("GetPickupDiagnosis() error = %v", err)
	}

	if diagnosis.PrimaryReasonCode != PickupDiagnosisReasonAgentPaused {
		t.Fatalf("agent paused diagnosis = %+v", diagnosis)
	}
}

func TestGetPickupDiagnosisProviderAvailability(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(context.Context, *testing.T, *ent.Client, ticketServiceFixture)
		want   PickupDiagnosisReasonCode
	}{
		{
			name: "machine_offline",
			mutate: func(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture) {
				t.Helper()
				makeProviderReady(ctx, t, client, fixture, time.Now().UTC(), true)
				if _, err := client.Machine.UpdateOneID(fixture.workerOneID).
					SetStatus(entmachine.StatusOffline).
					Save(ctx); err != nil {
					t.Fatalf("set machine offline: %v", err)
				}
			},
			want: PickupDiagnosisReasonMachineOffline,
		},
		{
			name: "provider_stale",
			mutate: func(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture) {
				t.Helper()
				makeProviderReady(ctx, t, client, fixture, time.Now().UTC().Add(-3*time.Hour), true)
			},
			want: PickupDiagnosisReasonProviderStale,
		},
		{
			name: "provider_unavailable",
			mutate: func(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture) {
				t.Helper()
				makeProviderReady(ctx, t, client, fixture, time.Now().UTC(), false)
			},
			want: PickupDiagnosisReasonProviderUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := openTestEntClient(t)
			fixture := seedTicketServiceFixture(ctx, t, client)
			service := newTicketService(client)
			ticketItem := createDiagnosisTicket(ctx, t, service, fixture)

			tc.mutate(ctx, t, client, fixture)

			diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
			if err != nil {
				t.Fatalf("GetPickupDiagnosis() error = %v", err)
			}
			if diagnosis.PrimaryReasonCode != tc.want {
				t.Fatalf("%s diagnosis = %+v", tc.name, diagnosis)
			}
		})
	}
}

func TestGetPickupDiagnosisConcurrencyBlocks(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(context.Context, *testing.T, *ent.Client, ticketServiceFixture)
		want   PickupDiagnosisReasonCode
	}{
		{
			name: "workflow",
			mutate: func(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture) {
				t.Helper()
				if _, err := client.Workflow.UpdateOneID(fixture.workflowID).SetMaxConcurrent(1).Save(ctx); err != nil {
					t.Fatalf("set workflow max concurrent: %v", err)
				}
			},
			want: PickupDiagnosisReasonWorkflowConcurrencyFull,
		},
		{
			name: "project",
			mutate: func(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture) {
				t.Helper()
				if _, err := client.Workflow.UpdateOneID(fixture.workflowID).SetMaxConcurrent(2).Save(ctx); err != nil {
					t.Fatalf("relax workflow capacity: %v", err)
				}
				if _, err := client.Project.UpdateOneID(fixture.projectID).SetMaxConcurrentAgents(1).Save(ctx); err != nil {
					t.Fatalf("set project max concurrent: %v", err)
				}
			},
			want: PickupDiagnosisReasonProjectConcurrencyFull,
		},
		{
			name: "provider",
			mutate: func(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture) {
				t.Helper()
				if _, err := client.Workflow.UpdateOneID(fixture.workflowID).SetMaxConcurrent(2).Save(ctx); err != nil {
					t.Fatalf("relax workflow capacity: %v", err)
				}
				if _, err := client.Project.UpdateOneID(fixture.projectID).SetMaxConcurrentAgents(2).Save(ctx); err != nil {
					t.Fatalf("relax project capacity: %v", err)
				}
				if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).SetMaxParallelRuns(1).Save(ctx); err != nil {
					t.Fatalf("set provider max parallel: %v", err)
				}
			},
			want: PickupDiagnosisReasonProviderConcurrencyFull,
		},
		{
			name: "status",
			mutate: func(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture) {
				t.Helper()
				if _, err := client.Workflow.UpdateOneID(fixture.workflowID).SetMaxConcurrent(2).Save(ctx); err != nil {
					t.Fatalf("relax workflow capacity: %v", err)
				}
				if _, err := client.Project.UpdateOneID(fixture.projectID).SetMaxConcurrentAgents(2).Save(ctx); err != nil {
					t.Fatalf("relax project capacity: %v", err)
				}
				limit := 1
				if _, err := client.TicketStatus.UpdateOneID(fixture.todoID).SetMaxActiveRuns(limit).Save(ctx); err != nil {
					t.Fatalf("set status max active runs: %v", err)
				}
				if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).SetMaxParallelRuns(2).Save(ctx); err != nil {
					t.Fatalf("relax provider capacity: %v", err)
				}
			},
			want: PickupDiagnosisReasonStatusCapacityFull,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := openTestEntClient(t)
			fixture := seedTicketServiceFixture(ctx, t, client)
			service := newTicketService(client)
			makeProviderReady(ctx, t, client, fixture, time.Now().UTC(), true)

			candidate := createDiagnosisTicket(ctx, t, service, fixture)
			occupied := createDiagnosisTicket(ctx, t, service, fixture)
			seedDiagnosisCurrentRun(ctx, t, client, fixture, occupied.ID)
			tc.mutate(ctx, t, client, fixture)

			diagnosis, err := service.GetPickupDiagnosis(ctx, candidate.ID)
			if err != nil {
				t.Fatalf("GetPickupDiagnosis() error = %v", err)
			}
			if diagnosis.PrimaryReasonCode != tc.want {
				t.Fatalf("%s diagnosis = %+v", tc.name, diagnosis)
			}
		})
	}
}

func TestGetPickupDiagnosisRunningAndCompleted(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		ctx := context.Background()
		client := openTestEntClient(t)
		fixture := seedTicketServiceFixture(ctx, t, client)
		service := newTicketService(client)
		ticketItem := createDiagnosisTicket(ctx, t, service, fixture)
		seedDiagnosisCurrentRun(ctx, t, client, fixture, ticketItem.ID)

		diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
		if err != nil {
			t.Fatalf("GetPickupDiagnosis() error = %v", err)
		}
		if diagnosis.State != PickupDiagnosisStateRunning || diagnosis.PrimaryReasonCode != PickupDiagnosisReasonRunningCurrentRun {
			t.Fatalf("running diagnosis = %+v", diagnosis)
		}
	})

	t.Run("completed", func(t *testing.T) {
		ctx := context.Background()
		client := openTestEntClient(t)
		fixture := seedTicketServiceFixture(ctx, t, client)
		service := newTicketService(client)
		ticketItem, err := service.Create(ctx, CreateInput{
			ProjectID: fixture.projectID,
			Title:     "Completed ticket",
			StatusID:  &fixture.doneID,
			Priority:  diagnosisPriorityPtr(PriorityMedium),
			Type:      TypeFeature,
		})
		if err != nil {
			t.Fatalf("Create(done) error = %v", err)
		}
		completedAt := time.Now().UTC()
		if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetCompletedAt(completedAt).Save(ctx); err != nil {
			t.Fatalf("set completed_at: %v", err)
		}

		diagnosis, err := service.GetPickupDiagnosis(ctx, ticketItem.ID)
		if err != nil {
			t.Fatalf("GetPickupDiagnosis() error = %v", err)
		}
		if diagnosis.State != PickupDiagnosisStateCompleted || diagnosis.PrimaryReasonCode != PickupDiagnosisReasonCompleted {
			t.Fatalf("completed diagnosis = %+v", diagnosis)
		}
	})
}

func createDiagnosisTicket(
	ctx context.Context,
	t *testing.T,
	service *Service,
	fixture ticketServiceFixture,
) Ticket {
	t.Helper()

	ticketItem, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Diagnosis target",
		StatusID:   &fixture.todoID,
		WorkflowID: &fixture.workflowID,
		Priority:   diagnosisPriorityPtr(PriorityMedium),
		Type:       TypeFeature,
	})
	if err != nil {
		t.Fatalf("Create(diagnosis target) error = %v", err)
	}

	return ticketItem
}

func makeProviderReady(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	fixture ticketServiceFixture,
	checkedAt time.Time,
	ready bool,
) {
	t.Helper()

	resources := map[string]any{
		"monitor": map[string]any{
			"l4": map[string]any{
				"checked_at": checkedAt.UTC().Format(time.RFC3339),
				"codex": map[string]any{
					"installed":   true,
					"auth_mode":   "api_key",
					"auth_status": "unknown",
					"ready":       ready,
				},
			},
		},
	}
	if _, err := client.Machine.UpdateOneID(fixture.workerOneID).
		SetStatus(entmachine.StatusOnline).
		SetWorkspaceRoot("/workspace/openase").
		SetResources(resources).
		Save(ctx); err != nil {
		t.Fatalf("prepare ready provider machine: %v", err)
	}
}

func seedDiagnosisCurrentRun(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	fixture ticketServiceFixture,
	ticketID uuid.UUID,
) {
	t.Helper()

	now := time.Now().UTC()
	runItem, err := client.AgentRun.Create().
		SetAgentID(fixture.agentID).
		SetWorkflowID(fixture.workflowID).
		SetTicketID(ticketID).
		SetProviderID(fixture.providerID).
		SetStatus(entagentrun.StatusExecuting).
		SetRuntimeStartedAt(now).
		SetLastHeartbeatAt(now).
		Save(ctx)
	if err != nil {
		t.Fatalf("create current run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketID).
		SetCurrentRunID(runItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("attach current run: %v", err)
	}
}
