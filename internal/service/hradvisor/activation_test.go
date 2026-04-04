package hradvisor

import (
	"context"
	"errors"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	hrdomain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
	"github.com/google/uuid"
)

func TestActivateCreatesWorkflowAgentAndBootstrapTicket(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	todoID := uuid.New()
	doneID := uuid.New()
	agentID := uuid.New()
	workflowID := uuid.New()
	ticketID := uuid.New()

	catalogStub := &stubActivationCatalog{
		project: catalogdomain.Project{ID: projectID, OrganizationID: orgID, DefaultAgentProviderID: &providerID},
		org:     catalogdomain.Organization{ID: orgID},
		providers: []catalogdomain.AgentProvider{
			{ID: providerID, Name: "OpenAI Codex", AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer, Available: true},
		},
		createdAgent: catalogdomain.Agent{ID: agentID, ProjectID: projectID, ProviderID: providerID, Name: "QA Engineer Agent"},
	}
	workflowStub := &stubActivationWorkflows{
		createdWorkflow: ActivationWorkflow{
			ID:              workflowID,
			ProjectID:       projectID,
			AgentID:         &agentID,
			Name:            "QA Engineer",
			Type:            "QA Engineer",
			HarnessPath:     ".openase/harnesses/roles/qa-engineer.md",
			HarnessContent:  "content",
			Version:         1,
			IsActive:        true,
			PickupStatusIDs: []uuid.UUID{todoID},
			FinishStatusIDs: []uuid.UUID{doneID},
		},
	}
	statusStub := &stubActivationStatuses{
		list: []ActivationStatus{
			{ID: todoID, Name: "Todo", Stage: "unstarted"},
			{ID: doneID, Name: "Done", Stage: "completed"},
		},
	}
	ticketStub := &stubActivationTickets{
		createdTicket: ActivationTicket{
			ID:         ticketID,
			ProjectID:  projectID,
			Identifier: "ASE-1",
			Title:      "Bootstrap QA regression coverage",
			StatusID:   todoID,
			StatusName: "Todo",
			Priority:   "medium",
			Type:       "chore",
			WorkflowID: &workflowID,
			CreatedBy:  "system:hr-advisor",
		},
	}

	service := NewActivationService(catalogStub, workflowStub, statusStub, ticketStub)
	result, err := service.Activate(context.Background(), mustParseActivationInput(t, projectID, "qa-engineer", true))
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	if catalogStub.createAgentInput == nil || catalogStub.createAgentInput.ProviderID != providerID {
		t.Fatalf("expected create agent provider %s, got %+v", providerID, catalogStub.createAgentInput)
	}
	if workflowStub.createInput == nil {
		t.Fatal("expected workflow create input")
	}
	if workflowStub.createInput.Name != "QA Engineer" || workflowStub.createInput.Type != "QA Engineer" {
		t.Fatalf("unexpected workflow create input: %+v", workflowStub.createInput)
	}
	if workflowStub.createInput.HarnessPath != ".openase/harnesses/roles/qa-engineer.md" {
		t.Fatalf("expected role harness path, got %+v", workflowStub.createInput.HarnessPath)
	}
	if ticketStub.createInput == nil || ticketStub.createInput.WorkflowID == nil || *ticketStub.createInput.WorkflowID != workflowID {
		t.Fatalf("expected bootstrap ticket bound to workflow %s, got %+v", workflowID, ticketStub.createInput)
	}
	if result.BootstrapTicket.Status != "created" || result.BootstrapTicket.Ticket == nil || result.BootstrapTicket.Ticket.ID != ticketID {
		t.Fatalf("unexpected bootstrap result: %+v", result.BootstrapTicket)
	}
}

func TestActivateFailsWhenRequiredStatusMissing(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()

	service := NewActivationService(
		&stubActivationCatalog{
			project: catalogdomain.Project{ID: projectID, OrganizationID: orgID, DefaultAgentProviderID: &providerID},
			org:     catalogdomain.Organization{ID: orgID},
			providers: []catalogdomain.AgentProvider{
				{ID: providerID, Name: "OpenAI Codex", AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer, Available: true},
			},
		},
		&stubActivationWorkflows{},
		&stubActivationStatuses{
			list: []ActivationStatus{{ID: uuid.New(), Name: "Todo", Stage: "unstarted"}},
		},
		nil,
	)

	_, err := service.Activate(context.Background(), mustParseActivationInput(t, projectID, "qa-engineer", false))
	if !errors.Is(err, ErrActivationStatusNotFound) {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestActivateRollsBackAgentWhenWorkflowCreateFails(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	todoID := uuid.New()
	doneID := uuid.New()

	catalogStub := &stubActivationCatalog{
		project: catalogdomain.Project{ID: projectID, OrganizationID: orgID, DefaultAgentProviderID: &providerID},
		org:     catalogdomain.Organization{ID: orgID},
		providers: []catalogdomain.AgentProvider{
			{ID: providerID, Name: "OpenAI Codex", AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer, Available: true},
		},
		createdAgent: catalogdomain.Agent{ID: agentID, ProjectID: projectID, ProviderID: providerID, Name: "QA Engineer Agent"},
	}

	service := NewActivationService(
		catalogStub,
		&stubActivationWorkflows{createErr: errors.New("workflow create failed")},
		&stubActivationStatuses{
			list: []ActivationStatus{
				{ID: todoID, Name: "Todo", Stage: "unstarted"},
				{ID: doneID, Name: "Done", Stage: "completed"},
			},
		},
		nil,
	)

	_, err := service.Activate(context.Background(), mustParseActivationInput(t, projectID, "qa-engineer", false))
	if err == nil || !strings.Contains(err.Error(), "workflow create failed") {
		t.Fatalf("expected workflow create error, got %v", err)
	}
	if catalogStub.deletedAgentID != agentID {
		t.Fatalf("expected agent rollback for %s, got %s", agentID, catalogStub.deletedAgentID)
	}
}

func TestActivateDispatcherFallsBackToBacklogStageForRenamedStatuses(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	backlogID := uuid.New()
	todoID := uuid.New()
	agentID := uuid.New()
	workflowID := uuid.New()

	workflowStub := &stubActivationWorkflows{
		createdWorkflow: ActivationWorkflow{
			ID:              workflowID,
			ProjectID:       projectID,
			AgentID:         &agentID,
			Name:            "Dispatcher",
			Type:            "Dispatcher",
			HarnessPath:     ".openase/harnesses/roles/dispatcher.md",
			HarnessContent:  "content",
			Version:         1,
			IsActive:        true,
			PickupStatusIDs: []uuid.UUID{backlogID},
			FinishStatusIDs: []uuid.UUID{todoID},
		},
	}

	service := NewActivationService(
		&stubActivationCatalog{
			project: catalogdomain.Project{ID: projectID, OrganizationID: orgID, DefaultAgentProviderID: &providerID},
			org:     catalogdomain.Organization{ID: orgID},
			providers: []catalogdomain.AgentProvider{
				{ID: providerID, Name: "OpenAI Codex", AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer, Available: true},
			},
			createdAgent: catalogdomain.Agent{ID: agentID, ProjectID: projectID, ProviderID: providerID, Name: "Dispatcher Agent"},
		},
		workflowStub,
		&stubActivationStatuses{
			list: []ActivationStatus{
				{ID: backlogID, Name: "Inbox", Stage: "backlog"},
				{ID: todoID, Name: "Todo", Stage: "unstarted"},
			},
		},
		nil,
	)

	result, err := service.Activate(context.Background(), mustParseActivationInput(t, projectID, "dispatcher", false))
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if workflowStub.createInput == nil {
		t.Fatal("expected workflow create input")
	}
	if strings.Join(uuidStrings(workflowStub.createInput.PickupStatusIDs), ",") != backlogID.String() {
		t.Fatalf("unexpected dispatcher pickup binding: %+v", workflowStub.createInput.PickupStatusIDs)
	}
	if strings.Join(uuidStrings(workflowStub.createInput.FinishStatusIDs), ",") != todoID.String() {
		t.Fatalf("unexpected dispatcher finish binding: %+v", workflowStub.createInput.FinishStatusIDs)
	}
	if strings.Join(uuidStrings(result.Workflow.PickupStatusIDs), ",") != backlogID.String() {
		t.Fatalf("unexpected dispatcher result pickup binding: %+v", result.Workflow.PickupStatusIDs)
	}
	if strings.Join(uuidStrings(result.Workflow.FinishStatusIDs), ",") != todoID.String() {
		t.Fatalf("unexpected dispatcher result finish binding: %+v", result.Workflow.FinishStatusIDs)
	}
}

func TestActivateDispatcherPrefersExistingWorkflowPickupLanesForFinishBindings(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	backlogID := uuid.New()
	codingID := uuid.New()
	researchID := uuid.New()
	agentID := uuid.New()
	workflowID := uuid.New()

	workflowStub := &stubActivationWorkflows{
		listed: []ActivationWorkflow{
			{ID: uuid.New(), RoleSlug: "fullstack-developer", IsActive: true, PickupStatusIDs: []uuid.UUID{codingID}},
			{ID: uuid.New(), RoleSlug: "research-ideation", IsActive: true, PickupStatusIDs: []uuid.UUID{researchID}},
		},
		createdWorkflow: ActivationWorkflow{
			ID:              workflowID,
			ProjectID:       projectID,
			AgentID:         &agentID,
			Name:            "Dispatcher",
			Type:            "Dispatcher",
			HarnessPath:     ".openase/harnesses/roles/dispatcher.md",
			HarnessContent:  "content",
			Version:         1,
			IsActive:        true,
			PickupStatusIDs: []uuid.UUID{backlogID},
			FinishStatusIDs: []uuid.UUID{codingID, researchID},
		},
	}

	service := NewActivationService(
		&stubActivationCatalog{
			project: catalogdomain.Project{ID: projectID, OrganizationID: orgID, DefaultAgentProviderID: &providerID},
			org:     catalogdomain.Organization{ID: orgID},
			providers: []catalogdomain.AgentProvider{
				{ID: providerID, Name: "OpenAI Codex", AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer, Available: true},
			},
			createdAgent: catalogdomain.Agent{ID: agentID, ProjectID: projectID, ProviderID: providerID, Name: "Dispatcher Agent"},
		},
		workflowStub,
		&stubActivationStatuses{
			list: []ActivationStatus{
				{ID: backlogID, Name: "Backlog", Stage: "backlog"},
				{ID: codingID, Name: "Coding", Stage: "unstarted"},
				{ID: researchID, Name: "Researching", Stage: "unstarted"},
			},
		},
		nil,
	)

	if _, err := service.Activate(context.Background(), mustParseActivationInput(t, projectID, "dispatcher", false)); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if workflowStub.createInput == nil {
		t.Fatal("expected workflow create input")
	}
	if got, want := strings.Join(uuidStrings(workflowStub.createInput.FinishStatusIDs), ","), strings.Join([]string{codingID.String(), researchID.String()}, ","); got != want {
		t.Fatalf("dispatcher finish binding = %q, want %q", got, want)
	}
}

func TestActivateDispatcherFailsPreciselyWithoutBacklogStage(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()

	service := NewActivationService(
		&stubActivationCatalog{
			project: catalogdomain.Project{ID: projectID, OrganizationID: orgID, DefaultAgentProviderID: &providerID},
			org:     catalogdomain.Organization{ID: orgID},
			providers: []catalogdomain.AgentProvider{
				{ID: providerID, Name: "OpenAI Codex", AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer, Available: true},
			},
		},
		&stubActivationWorkflows{},
		&stubActivationStatuses{
			list: []ActivationStatus{
				{ID: uuid.New(), Name: "Inbox", Stage: "unstarted"},
			},
		},
		nil,
	)

	_, err := service.Activate(context.Background(), mustParseActivationInput(t, projectID, "dispatcher", false))
	if !errors.Is(err, ErrActivationStatusNotFound) {
		t.Fatalf("expected status error, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), `requires a configured status with stage "backlog"`) {
		t.Fatalf("expected precise dispatcher stage error, got %v", err)
	}
}

func uuidStrings(items []uuid.UUID) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.String())
	}
	return values
}

func mustParseActivationInput(
	t *testing.T,
	projectID uuid.UUID,
	roleSlug string,
	createBootstrapTicket bool,
) hrdomain.ActivateRecommendationInput {
	t.Helper()

	input, err := hrdomain.ParseActivateRecommendation(projectID, hrdomain.ActivateRecommendationRequest{
		RoleSlug:              roleSlug,
		CreateBootstrapTicket: &createBootstrapTicket,
	})
	if err != nil {
		t.Fatalf("ParseActivateRecommendation() error = %v", err)
	}

	return input
}

type stubActivationCatalog struct {
	project          catalogdomain.Project
	org              catalogdomain.Organization
	providers        []catalogdomain.AgentProvider
	createdAgent     catalogdomain.Agent
	createAgentInput *catalogdomain.CreateAgent
	deletedAgentID   uuid.UUID
}

func (s *stubActivationCatalog) GetProject(context.Context, uuid.UUID) (catalogdomain.Project, error) {
	return s.project, nil
}

func (s *stubActivationCatalog) GetOrganization(context.Context, uuid.UUID) (catalogdomain.Organization, error) {
	return s.org, nil
}

func (s *stubActivationCatalog) ListAgentProviders(context.Context, uuid.UUID) ([]catalogdomain.AgentProvider, error) {
	return append([]catalogdomain.AgentProvider(nil), s.providers...), nil
}

func (s *stubActivationCatalog) CreateAgent(_ context.Context, input catalogdomain.CreateAgent) (catalogdomain.Agent, error) {
	s.createAgentInput = &input
	return s.createdAgent, nil
}

func (s *stubActivationCatalog) DeleteAgent(_ context.Context, id uuid.UUID) (catalogdomain.Agent, error) {
	s.deletedAgentID = id
	return s.createdAgent, nil
}

type stubActivationWorkflows struct {
	listed          []ActivationWorkflow
	createInput     *ActivateWorkflowInput
	createdWorkflow ActivationWorkflow
	createErr       error
}

func (s *stubActivationWorkflows) List(context.Context, uuid.UUID) ([]ActivationWorkflow, error) {
	return append([]ActivationWorkflow(nil), s.listed...), nil
}

func (s *stubActivationWorkflows) Create(_ context.Context, input ActivateWorkflowInput) (ActivationWorkflow, error) {
	s.createInput = &input
	if s.createErr != nil {
		return ActivationWorkflow{}, s.createErr
	}
	return s.createdWorkflow, nil
}

type stubActivationStatuses struct {
	list []ActivationStatus
}

func (s *stubActivationStatuses) List(context.Context, uuid.UUID) ([]ActivationStatus, error) {
	return append([]ActivationStatus(nil), s.list...), nil
}

type stubActivationTickets struct {
	createInput   *CreateActivationTicketInput
	createdTicket ActivationTicket
}

func (s *stubActivationTickets) Create(_ context.Context, input CreateActivationTicketInput) (ActivationTicket, error) {
	s.createInput = &input
	return s.createdTicket, nil
}
