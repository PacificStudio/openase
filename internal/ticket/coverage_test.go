package ticket

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketcomment "github.com/BetterAndBetterII/openase/ent/ticketcomment"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestTicketServiceNilClientGuards(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := context.Background()
	projectID := uuid.New()
	ticketID := uuid.New()
	commentID := uuid.New()
	dependencyID := uuid.New()
	externalLinkID := uuid.New()

	if _, err := service.List(ctx, ListInput{ProjectID: projectID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("List error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Get(ctx, ticketID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Create(ctx, CreateInput{ProjectID: projectID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Update(ctx, UpdateInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Update error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddDependency(ctx, AddDependencyInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddDependency error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveDependency(ctx, ticketID, dependencyID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveDependency error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddExternalLink(ctx, AddExternalLinkInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddExternalLink error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListComments(ctx, ticketID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListComments error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddComment(ctx, AddCommentInput{TicketID: ticketID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddComment error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.UpdateComment(ctx, UpdateCommentInput{TicketID: ticketID, CommentID: commentID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("UpdateComment error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListCommentRevisions(ctx, ticketID, commentID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListCommentRevisions error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveExternalLink(ctx, ticketID, externalLinkID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveExternalLink error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveComment(ctx, ticketID, commentID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveComment error = %v, want %v", err, ErrUnavailable)
	}
}

func TestTicketHelperFunctions(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	statusID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()
	runID := uuid.New()
	workflowID := uuid.New()
	machineID := uuid.New()
	now := time.Date(2026, 3, 27, 14, 0, 0, 0, time.FixedZone("UTC+1", 60*60))

	if got := Some("value"); !got.Set || got.Value != "value" {
		t.Fatalf("Some() = %+v", got)
	}
	if value, ok := parseIdentifierSequence("ASE-42"); !ok || value != 42 {
		t.Fatalf("parseIdentifierSequence() = %d, %v", value, ok)
	}
	if _, ok := parseIdentifierSequence("OTHER-42"); ok {
		t.Fatal("parseIdentifierSequence() expected false for foreign prefix")
	}
	if got := resolveCreatedBy("  "); got != defaultCreatedBy {
		t.Fatalf("resolveCreatedBy(blank) = %q", got)
	}
	if got := resolveCreatedBy(" user:codex "); got != "user:codex" {
		t.Fatalf("resolveCreatedBy(value) = %q", got)
	}
	if !optionalUUIDPointerEqual(nil, nil) || optionalUUIDPointerEqual(&statusID, nil) || !optionalUUIDPointerEqual(&statusID, &statusID) {
		t.Fatal("optionalUUIDPointerEqual() returned unexpected results")
	}

	parentTicket := &ent.Ticket{
		ID:         parentID,
		Identifier: "ASE-1",
		Title:      "Parent",
		StatusID:   statusID,
		Edges: ent.TicketEdges{
			Status: &ent.TicketStatus{Name: "Backlog"},
		},
	}
	childTicket := &ent.Ticket{
		ID:         childID,
		Identifier: "ASE-2",
		Title:      "Child",
		StatusID:   statusID,
	}
	dependency := &ent.TicketDependency{
		ID:   uuid.New(),
		Type: entticketdependency.TypeBlocks,
		Edges: ent.TicketDependencyEdges{
			TargetTicket: childTicket,
		},
	}
	externalLink := &ent.TicketExternalLink{
		ID:         uuid.New(),
		LinkType:   entticketexternallink.LinkTypeGithubPr,
		URL:        "https://github.com/GrandCX/openase/pull/278",
		ExternalID: "278",
		Title:      "coverage rollout",
		Status:     "open",
		Relation:   entticketexternallink.RelationRelated,
		CreatedAt:  now,
	}
	comment := &ent.TicketComment{
		ID:           uuid.New(),
		TicketID:     childID,
		Body:         "ship it",
		CreatedBy:    "user:codex",
		CreatedAt:    now,
		UpdatedAt:    now.Add(5 * time.Minute),
		EditedAt:     &now,
		EditCount:    1,
		LastEditedBy: stringPtr("user:reviewer"),
		IsDeleted:    true,
		DeletedAt:    &now,
		DeletedBy:    stringPtr("user:reviewer"),
	}
	revision := &ent.TicketCommentRevision{
		ID:             uuid.New(),
		CommentID:      comment.ID,
		RevisionNumber: 2,
		BodyMarkdown:   "ship it v2",
		EditedBy:       "user:reviewer",
		EditedAt:       now.Add(10 * time.Minute),
		EditReason:     stringPtr("clarified"),
	}
	ticketItem := &ent.Ticket{
		ID:                childID,
		ProjectID:         projectID,
		Identifier:        "ASE-2",
		Title:             "Coverage rollout",
		Description:       "Raise backend coverage",
		StatusID:          statusID,
		Priority:          entticket.PriorityHigh,
		Type:              entticket.TypeFeature,
		WorkflowID:        &workflowID,
		CurrentRunID:      &runID,
		TargetMachineID:   &machineID,
		CreatedBy:         "user:codex",
		ExternalRef:       "GH-278",
		BudgetUsd:         20,
		CostTokensInput:   100,
		CostTokensOutput:  50,
		CostAmount:        5.5,
		AttemptCount:      2,
		ConsecutiveErrors: 1,
		StartedAt:         &now,
		CompletedAt:       &now,
		NextRetryAt:       &now,
		RetryPaused:       true,
		PauseReason:       "budget",
		CreatedAt:         now,
		Edges: ent.TicketEdges{
			Status:               &ent.TicketStatus{Name: "In Progress"},
			Parent:               parentTicket,
			Children:             []*ent.Ticket{childTicket},
			OutgoingDependencies: []*ent.TicketDependency{dependency},
			ExternalLinks:        []*ent.TicketExternalLink{externalLink},
		},
	}

	mappedTicket := mapTicket(ticketItem)
	if mappedTicket.StatusName != "In Progress" || mappedTicket.Parent == nil || len(mappedTicket.Children) != 1 || len(mappedTicket.Dependencies) != 1 || len(mappedTicket.ExternalLinks) != 1 {
		t.Fatalf("mapTicket() = %+v", mappedTicket)
	}
	if mappedTicket.Parent.StatusName != "Backlog" {
		t.Fatalf("mapTicket() parent = %+v", mappedTicket.Parent)
	}
	if mapped := mapDependency(dependency); mapped.Target.Identifier != "ASE-2" {
		t.Fatalf("mapDependency() = %+v", mapped)
	}
	if mapped := mapExternalLink(externalLink); mapped.URL != externalLink.URL || mapped.CreatedAt != now {
		t.Fatalf("mapExternalLink() = %+v", mapped)
	}
	if mapped := mapComment(comment); mapped.CreatedBy != "user:codex" || mapped.BodyMarkdown != "ship it" || mapped.EditCount != 1 || !mapped.IsDeleted || mapped.UpdatedAt != comment.UpdatedAt {
		t.Fatalf("mapComment() = %+v", mapped)
	}
	if mapped := mapCommentRevision(revision); mapped.RevisionNumber != 2 || mapped.BodyMarkdown != "ship it v2" || mapped.EditReason == nil || *mapped.EditReason != "clarified" {
		t.Fatalf("mapCommentRevision() = %+v", mapped)
	}
	if mapped := mapTicketReference(parentTicket); mapped.StatusName != "Backlog" {
		t.Fatalf("mapTicketReference() = %+v", mapped)
	}

	rollback(nil)
	reconcileBudgetPauseState(nil, nil, 10)
}

func TestTicketRepoScopeAndMetricHelpers(t *testing.T) {
	t.Parallel()

	if got := timeNowUTC(); got.Location() != time.UTC {
		t.Fatalf("timeNowUTC() = %+v", got)
	}

	currentStatusID := uuid.New()
	finishStatusID := uuid.New()
	workflowID := uuid.New()
	if _, err := resolveRepoScopeFinishStatus(currentStatusID, nil); err == nil {
		t.Fatal("resolveRepoScopeFinishStatus(nil) expected error")
	}
	if got, err := resolveRepoScopeFinishStatus(currentStatusID, &ent.Workflow{
		ID: workflowID,
		Edges: ent.WorkflowEdges{
			FinishStatuses: []*ent.TicketStatus{{ID: finishStatusID}},
		},
	}); err != nil || got != finishStatusID {
		t.Fatalf("resolveRepoScopeFinishStatus(single) = %s, %v", got, err)
	}
	if got, err := resolveRepoScopeFinishStatus(currentStatusID, &ent.Workflow{
		ID: workflowID,
		Edges: ent.WorkflowEdges{
			FinishStatuses: []*ent.TicketStatus{{ID: currentStatusID}, {ID: finishStatusID}},
		},
	}); err != nil || got != currentStatusID {
		t.Fatalf("resolveRepoScopeFinishStatus(current) = %s, %v", got, err)
	}
	if _, err := resolveRepoScopeFinishStatus(currentStatusID, &ent.Workflow{
		ID: workflowID,
		Edges: ent.WorkflowEdges{
			FinishStatuses: []*ent.TicketStatus{{ID: finishStatusID}, {ID: uuid.New()}},
		},
	}); err == nil {
		t.Fatal("resolveRepoScopeFinishStatus(ambiguous) expected error")
	}

	tests := []struct {
		name        string
		rawURL      string
		rawFullName string
		want        string
		wantErr     bool
	}{
		{name: "full name", rawFullName: " GrandCX/OpenASE ", want: "grandcx/openase"},
		{name: "ssh", rawURL: "git@github.com:GrandCX/openase.git", want: "grandcx/openase"},
		{name: "ssh url", rawURL: "ssh://git@github.com/GrandCX/openase.git", want: "grandcx/openase"},
		{name: "https", rawURL: "https://github.com/GrandCX/openase", want: "grandcx/openase"},
		{name: "plain path", rawURL: "GrandCX/openase", want: "grandcx/openase"},
		{name: "bad", rawURL: "openase", wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeGitHubRepositoryKey(tt.rawURL, tt.rawFullName)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeGitHubRepositoryKey(%q, %q) expected error", tt.rawURL, tt.rawFullName)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("normalizeGitHubRepositoryKey(%q, %q) = %q, %v", tt.rawURL, tt.rawFullName, got, err)
			}
		})
	}
	if _, err := normalizeOwnerRepoPath("owner"); err == nil {
		t.Fatal("normalizeOwnerRepoPath(owner) expected error")
	}

	metrics := &ticketMetricsProvider{}
	agentItem := &ent.Agent{
		Edges: ent.AgentEdges{
			Provider: &ent.AgentProvider{
				Name:      "Codex",
				ModelName: "gpt-5.4",
			},
		},
	}
	recordTokenUsageMetrics(metrics, agentItem, ticketing.UsageDelta{InputTokens: 10, OutputTokens: 5})
	recordCostUsageMetrics(metrics, agentItem, uuid.New(), 3.5)
	if len(metrics.calls) != 3 {
		t.Fatalf("metric calls = %+v", metrics.calls)
	}
	if got := mergeTags(provider.Tags{"provider": "Codex"}, provider.Tags{"direction": "input"}); got["provider"] != "Codex" || got["direction"] != "input" {
		t.Fatalf("mergeTags() = %+v", got)
	}

	recordTokenUsageMetrics(nil, agentItem, ticketing.UsageDelta{InputTokens: 1})
	recordTokenUsageMetrics(metrics, nil, ticketing.UsageDelta{InputTokens: 1})
	recordCostUsageMetrics(metrics, nil, uuid.New(), 1)
	recordCostUsageMetrics(metrics, agentItem, uuid.New(), 0)
}

type ticketMetricsProvider struct {
	calls []ticketMetricCall
}

type ticketMetricCall struct {
	name  string
	tags  provider.Tags
	value float64
}

func (m *ticketMetricsProvider) Counter(name string, tags provider.Tags) provider.Counter {
	return ticketCounterRecorder{
		provider: m,
		name:     name,
		tags:     tags,
	}
}

func TestTicketHelperErrorBranches(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := NewService(client)

	if err := service.ensureProjectExists(ctx, uuid.New()); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("ensureProjectExists(missing) error = %v, want %v", err, ErrProjectNotFound)
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("start tx: %v", err)
	}
	defer rollback(tx)

	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load fixture project: %v", err)
	}
	projectWithoutStatuses, err := client.Project.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("Empty Status Project").
		SetSlug("empty-status-project").
		Save(ctx)
	if err != nil {
		t.Fatalf("create empty status project: %v", err)
	}
	if _, err := client.TicketStatus.Delete().
		Where(entticketstatus.ProjectIDEQ(projectWithoutStatuses.ID)).
		Exec(ctx); err != nil {
		t.Fatalf("delete empty status project statuses: %v", err)
	}

	if _, err := service.resolveCreateStatusID(ctx, tx, projectWithoutStatuses.ID, nil); !errors.Is(err, ErrStatusNotFound) {
		t.Fatalf("resolveCreateStatusID(no default) error = %v, want %v", err, ErrStatusNotFound)
	}
	if err := ensureStatusAllowedByWorkflowFinishSet(ctx, tx, uuid.New(), fixture.todoID); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("ensureStatusAllowedByWorkflowFinishSet(missing workflow) error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if err := ensureTicketBelongsToProject(ctx, tx, fixture.projectID, uuid.New(), ErrParentTicketNotFound); !errors.Is(err, ErrParentTicketNotFound) {
		t.Fatalf("ensureTicketBelongsToProject(missing) error = %v, want %v", err, ErrParentTicketNotFound)
	}
	if err := ensureParentDoesNotCreateCycle(ctx, tx, fixture.legacyTicketID, uuid.New()); !errors.Is(err, ErrParentTicketNotFound) {
		t.Fatalf("ensureParentDoesNotCreateCycle(missing parent) error = %v, want %v", err, ErrParentTicketNotFound)
	}

	if err := service.mapTicketReadError("load ticket", &ent.NotFoundError{}); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("mapTicketReadError(not found) error = %v, want %v", err, ErrTicketNotFound)
	}
	readWrapped := service.mapTicketReadError("load ticket", errors.New("boom"))
	if readWrapped == nil || !strings.Contains(readWrapped.Error(), "load ticket: boom") {
		t.Fatalf("mapTicketReadError(other) = %v", readWrapped)
	}
	writeNotFound := service.mapTicketWriteError("save ticket", &ent.NotFoundError{})
	if !errors.Is(writeNotFound, ErrTicketNotFound) {
		t.Fatalf("mapTicketWriteError(not found) error = %v, want %v", writeNotFound, ErrTicketNotFound)
	}
	if got := service.mapTicketWriteError("save ticket", &ent.ConstraintError{}); got == nil || !strings.Contains(got.Error(), "save ticket: ent: constraint failed: ") {
		t.Fatalf("mapTicketWriteError(other constraint) = %v", got)
	}
}

func (m *ticketMetricsProvider) Histogram(string, provider.Tags) provider.Histogram {
	return ticketHistogramRecorder{}
}

func (m *ticketMetricsProvider) Gauge(string, provider.Tags) provider.Gauge {
	return ticketGaugeRecorder{}
}

type ticketCounterRecorder struct {
	provider *ticketMetricsProvider
	name     string
	tags     provider.Tags
}

func (r ticketCounterRecorder) Add(value float64) {
	r.provider.calls = append(r.provider.calls, ticketMetricCall{
		name:  r.name,
		tags:  r.tags,
		value: value,
	})
}

type ticketHistogramRecorder struct{}

func (ticketHistogramRecorder) Record(float64) {}

type ticketGaugeRecorder struct{}

func (ticketGaugeRecorder) Set(float64) {}

var _ = entagentprovider.FieldID
var _ = entagentrun.FieldID
var _ = entticketcomment.FieldID

func stringPtr(value string) *string {
	return &value
}
