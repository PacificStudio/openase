package projectpreset

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	presetdomain "github.com/BetterAndBetterII/openase/internal/domain/projectpreset"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

type presetFixture struct {
	projectID uuid.UUID
	agentID   uuid.UUID
}

func TestServiceApplyCreatesPresetStatusesAndWorkflow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openProjectPresetTestEntClient(t)
	fixture := seedPresetFixture(ctx, t, client)
	service, statusSvc, workflowSvc := newProjectPresetTestServices(t, client)

	result, err := service.Apply(ctx, presetdomain.ApplyInput{
		ProjectID: fixture.projectID,
		PresetKey: "fullstack-default",
		AppliedBy: "user:test",
		AgentBindings: []presetdomain.WorkflowAgentBinding{{
			WorkflowKey: "fullstack-developer",
			AgentID:     fixture.agentID,
		}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Preset.Meta.Key != "fullstack-default" {
		t.Fatalf("result preset key = %q", result.Preset.Meta.Key)
	}
	if len(result.Statuses) != 6 {
		t.Fatalf("applied statuses = %d, want 6", len(result.Statuses))
	}
	if len(result.Workflows) != 1 || result.Workflows[0].Action != "created" {
		t.Fatalf("applied workflows = %+v", result.Workflows)
	}
	if got := result.Workflows[0].AgentID; got != fixture.agentID {
		t.Fatalf("workflow agent = %s, want %s", got, fixture.agentID)
	}
	if len(result.Preset.ProjectAI.SkillReferences) != 1 || result.Preset.ProjectAI.SkillReferences[0].Skill != "auto-harness" {
		t.Fatalf("project ai refs = %+v", result.Preset.ProjectAI.SkillReferences)
	}

	statusList, err := statusSvc.List(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("List statuses error = %v", err)
	}
	if got := joinStatusNames(statusList.Statuses); got != "Backlog,Todo,In Progress,In Review,Done,Cancelled" {
		t.Fatalf("status names = %q", got)
	}
	workflowList, err := workflowSvc.List(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("List workflows error = %v", err)
	}
	if len(workflowList) != 1 {
		t.Fatalf("workflows = %+v", workflowList)
	}
	if workflowList[0].AgentID == nil || *workflowList[0].AgentID != fixture.agentID {
		t.Fatalf("workflow agent binding = %+v", workflowList[0].AgentID)
	}
}

func TestServiceApplyRejectsProjectsWithActiveTickets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openProjectPresetTestEntClient(t)
	fixture := seedPresetFixture(ctx, t, client)
	service, statusSvc, _ := newProjectPresetTestServices(t, client)

	todo, err := statusSvc.Create(ctx, ticketstatus.CreateInput{
		ProjectID:   fixture.projectID,
		Name:        "Todo",
		Stage:       "unstarted",
		Color:       "#3B82F6",
		IsDefault:   true,
		Description: "Ready",
	})
	if err != nil {
		t.Fatalf("create todo status: %v", err)
	}
	if _, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-1").
		SetTitle("Active setup ticket").
		SetStatusID(todo.ID).
		SetCreatedBy("user:test").
		Save(ctx); err != nil {
		t.Fatalf("create active ticket: %v", err)
	}

	_, err = service.Apply(ctx, presetdomain.ApplyInput{
		ProjectID: fixture.projectID,
		PresetKey: "fullstack-default",
		AgentBindings: []presetdomain.WorkflowAgentBinding{{
			WorkflowKey: "fullstack-developer",
			AgentID:     fixture.agentID,
		}},
	})
	if !errors.Is(err, presetdomain.ErrActiveTicketsPresent) {
		t.Fatalf("Apply() active tickets error = %v, want %v", err, presetdomain.ErrActiveTicketsPresent)
	}
}

func openProjectPresetTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return testPostgres.NewIsolatedEntClient(t)
}

func newProjectPresetTestServices(t *testing.T, client *ent.Client) (*Service, *ticketstatus.Service, *workflowservice.Service) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), logger, t.TempDir())
	if err != nil {
		t.Fatalf("new workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Fatalf("close workflow service: %v", closeErr)
		}
	})
	statusSvc := ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
	ticketSvc := ticketservice.NewService(ticketservice.Dependencies{
		Activity: ticketrepo.NewActivityRepository(client),
		Query:    ticketrepo.NewQueryRepository(client),
		Command:  ticketrepo.NewCommandRepository(client),
		Link:     ticketrepo.NewLinkRepository(client),
		Comment:  ticketrepo.NewCommentRepository(client),
		Usage:    ticketrepo.NewUsageRepository(client),
		Runtime:  ticketrepo.NewRuntimeRepository(client),
	})
	return NewService(client, ticketSvc, statusSvc, workflowSvc), statusSvc, workflowSvc
}

func seedPresetFixture(ctx context.Context, t *testing.T, client *ent.Client) presetFixture {
	t.Helper()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug(strings.ToLower("better-and-better-" + uuid.NewString()[:8])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug(strings.ToLower("openase-" + uuid.NewString()[:8])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("worker").
		SetHost("127.0.0.1").
		SetSSHUser("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		SetMaxParallelRuns(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(provider.ID).
		SetName("fullstack-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return presetFixture{projectID: project.ID, agentID: agentItem.ID}
}

func joinStatusNames(items []ticketstatus.Status) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return strings.Join(names, ",")
}
