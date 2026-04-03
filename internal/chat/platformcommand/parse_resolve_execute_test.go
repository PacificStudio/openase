package platformcommand

import (
	"context"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	ticketstatusservice "github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func TestParseProposalParsesSupportedCommands(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"type":    ProposalType,
		"summary": "Update the project and ticket",
		"commands": []any{
			map[string]any{
				"command": string(CommandProjectUpdateCreate),
				"args": map[string]any{
					"project": "CDN",
					"content": "Shift the project to a backend-only control plane.",
				},
			},
			map[string]any{
				"command": string(CommandTicketUpdate),
				"args": map[string]any{
					"ticket": "ASE-1",
					"status": "Todo",
				},
			},
		},
	}

	proposal, err := ParseProposal(payload)
	if err != nil {
		t.Fatalf("ParseProposal() error = %v", err)
	}
	if len(proposal.Commands) != 2 {
		t.Fatalf("command count = %d, want 2", len(proposal.Commands))
	}

	updateArgs, ok := proposal.Commands[0].Args.(ProjectUpdateCreateArgs)
	if !ok || updateArgs.Project != "CDN" || updateArgs.Content == "" {
		t.Fatalf("first command args = %#v", proposal.Commands[0].Args)
	}
	ticketArgs, ok := proposal.Commands[1].Args.(TicketUpdateArgs)
	if !ok || ticketArgs.Ticket != "ASE-1" || ticketArgs.Status == nil || *ticketArgs.Status != "Todo" {
		t.Fatalf("second command args = %#v", proposal.Commands[1].Args)
	}
}

func TestResolverResolvesHumanReadableReferences(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	ticketID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	statusID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	currentProject := catalogdomain.Project{
		ID:             projectID,
		OrganizationID: uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
		Name:           "CDN",
		Slug:           "cdn",
	}
	resolver := Resolver{
		Catalog: fakeCatalogResolver{projects: []catalogdomain.Project{currentProject}},
		Tickets: fakeTicketResolver{items: []ticketservice.Ticket{{ID: ticketID, Identifier: "ASE-1"}}},
		Statuses: fakeStatusResolver{result: ticketstatusservice.ListResult{
			Statuses: []ticketstatusservice.Status{{ID: statusID, Name: "Todo"}},
		}},
	}

	resolved, err := resolver.ResolveCommand(context.Background(), currentProject, Command{
		Name: CommandTicketUpdate,
		Args: TicketUpdateArgs{
			Ticket: "ASE-1",
			Status: stringPointer("Todo"),
		},
	})
	if err != nil {
		t.Fatalf("ResolveCommand() error = %v", err)
	}
	args, ok := resolved.Args.(ResolvedTicketUpdateArgs)
	if !ok {
		t.Fatalf("resolved args = %#v", resolved.Args)
	}
	if args.TicketID != ticketID || args.StatusID == nil || *args.StatusID != statusID {
		t.Fatalf("resolved args = %#v", args)
	}
}

func TestExecutorRunsResolvedCommandsThroughServices(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	ticketID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	statusID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	tickets := &fakeTicketExecutor{
		createResult: ticketservice.Ticket{ID: ticketID, Identifier: "ASE-2", Title: "Build backend switcher"},
		updateResult: ticketservice.Ticket{ID: ticketID, Identifier: "ASE-1", Title: "Rebuild CDN backend"},
	}
	updates := &fakeProjectUpdateExecutor{
		result: projectupdateservice.Thread{ID: uuid.New(), Title: "Backend-only CDN reset"},
	}
	executor := Executor{
		Tickets:        tickets,
		ProjectUpdates: updates,
	}

	projectUpdateResult := executor.Execute(context.Background(), 0, Command{
		Name: CommandProjectUpdateCreate,
		Args: ProjectUpdateCreateArgs{Project: "CDN", Content: "Reset the project."},
	}, ResolvedCommand{
		Name: CommandProjectUpdateCreate,
		Args: ResolvedProjectUpdateCreateArgs{
			ProjectID:   projectID,
			ProjectName: "CDN",
			Content:     "Reset the project.",
			Title:       "Backend-only CDN reset",
			Status:      projectupdateservice.StatusOnTrack,
		},
	}, "user:test")
	if !projectUpdateResult.Ok || updates.lastInput.ProjectID != projectID || updates.lastInput.CreatedBy != "user:test" {
		t.Fatalf("project update result = %+v, input = %+v", projectUpdateResult, updates.lastInput)
	}

	ticketUpdateResult := executor.Execute(context.Background(), 1, Command{
		Name: CommandTicketUpdate,
		Args: TicketUpdateArgs{Ticket: "ASE-1", Status: stringPointer("Todo")},
	}, ResolvedCommand{
		Name: CommandTicketUpdate,
		Args: ResolvedTicketUpdateArgs{
			TicketID:         ticketID,
			TicketIdentifier: "ASE-1",
			StatusID:         &statusID,
			StatusName:       stringPointer("Todo"),
		},
	}, "user:test")
	if !ticketUpdateResult.Ok || tickets.lastUpdate.TicketID != ticketID || !tickets.lastUpdate.CreatedBy.Set {
		t.Fatalf("ticket update result = %+v, input = %+v", ticketUpdateResult, tickets.lastUpdate)
	}

	ticketCreateResult := executor.Execute(context.Background(), 2, Command{
		Name: CommandTicketCreate,
		Args: TicketCreateArgs{Project: "CDN", Title: "Build backend switcher"},
	}, ResolvedCommand{
		Name: CommandTicketCreate,
		Args: ResolvedTicketCreateArgs{
			ProjectID:   projectID,
			ProjectName: "CDN",
			Title:       "Build backend switcher",
			Description: "Route production traffic by version.",
			StatusID:    &statusID,
		},
	}, "user:test")
	if !ticketCreateResult.Ok || tickets.lastCreate.ProjectID != projectID || tickets.lastCreate.StatusID == nil || *tickets.lastCreate.StatusID != statusID {
		t.Fatalf("ticket create result = %+v, input = %+v", ticketCreateResult, tickets.lastCreate)
	}
}

type fakeCatalogResolver struct {
	projects []catalogdomain.Project
}

func (f fakeCatalogResolver) GetProject(context.Context, uuid.UUID) (catalogdomain.Project, error) {
	return catalogdomain.Project{}, nil
}

func (f fakeCatalogResolver) ListProjects(context.Context, uuid.UUID) ([]catalogdomain.Project, error) {
	return append([]catalogdomain.Project(nil), f.projects...), nil
}

type fakeTicketResolver struct {
	items []ticketservice.Ticket
}

func (f fakeTicketResolver) List(context.Context, ticketservice.ListInput) ([]ticketservice.Ticket, error) {
	return append([]ticketservice.Ticket(nil), f.items...), nil
}

type fakeStatusResolver struct {
	result ticketstatusservice.ListResult
}

func (f fakeStatusResolver) List(context.Context, uuid.UUID) (ticketstatusservice.ListResult, error) {
	return f.result, nil
}

type fakeTicketExecutor struct {
	createResult ticketservice.Ticket
	updateResult ticketservice.Ticket
	lastCreate   ticketservice.CreateInput
	lastUpdate   ticketservice.UpdateInput
}

func (f *fakeTicketExecutor) Create(_ context.Context, input ticketservice.CreateInput) (ticketservice.Ticket, error) {
	f.lastCreate = input
	return f.createResult, nil
}

func (f *fakeTicketExecutor) Update(_ context.Context, input ticketservice.UpdateInput) (ticketservice.Ticket, error) {
	f.lastUpdate = input
	return f.updateResult, nil
}

type fakeProjectUpdateExecutor struct {
	result    projectupdateservice.Thread
	lastInput projectupdateservice.AddThreadInput
}

func (f *fakeProjectUpdateExecutor) AddThread(
	_ context.Context,
	input projectupdateservice.AddThreadInput,
) (projectupdateservice.Thread, error) {
	f.lastInput = input
	return f.result, nil
}

func stringPointer(value string) *string {
	return &value
}
