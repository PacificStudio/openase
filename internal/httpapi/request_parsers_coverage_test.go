package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestTicketRequestParserCoverage(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ticketID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	commentID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	statusID := "44444444-4444-4444-4444-444444444444"
	workflowID := "55555555-5555-5555-5555-555555555555"
	parentID := "66666666-6666-6666-6666-666666666666"
	createdBy := " codex "
	externalRef := " GH-42 "
	title := " Updated Title "
	description := " Updated description "
	editReason := " demo workflow "
	priority := " high "
	ticketType := " bugfix "
	negativeBudget := -1.0
	budget := 12.5

	createInput, err := parseCreateTicketRequest(projectID, rawCreateTicketRequest{
		Title:          "  Ticket title  ",
		Description:    "  Ticket description  ",
		StatusID:       &statusID,
		Priority:       &priority,
		Type:           &ticketType,
		WorkflowID:     &workflowID,
		CreatedBy:      &createdBy,
		ParentTicketID: &parentID,
		ExternalRef:    &externalRef,
		BudgetUSD:      &budget,
	})
	if err != nil {
		t.Fatalf("parseCreateTicketRequest() error = %v", err)
	}
	if createInput.Title != "Ticket title" || createInput.Priority == nil || *createInput.Priority != ticketservice.PriorityHigh || createInput.Type != ticketservice.TypeBugfix {
		t.Fatalf("parseCreateTicketRequest() = %+v", createInput)
	}
	if createInput.CreatedBy != "codex" || createInput.ExternalRef != "GH-42" || createInput.BudgetUSD != 12.5 {
		t.Fatalf("parseCreateTicketRequest() = %+v", createInput)
	}
	createInput, err = parseCreateTicketRequest(projectID, rawCreateTicketRequest{
		Title:    "  Ticket title  ",
		Priority: strPtr(" "),
	})
	if err != nil {
		t.Fatalf("parseCreateTicketRequest(blank priority) error = %v", err)
	}
	if createInput.Priority != nil {
		t.Fatalf("parseCreateTicketRequest(blank priority) = %+v", createInput)
	}
	if _, err := parseCreateTicketRequest(projectID, rawCreateTicketRequest{}); err == nil || !strings.Contains(err.Error(), "title must not be empty") {
		t.Fatalf("parseCreateTicketRequest(empty title) error = %v", err)
	}
	if _, err := parseCreateTicketRequest(projectID, rawCreateTicketRequest{Title: "ok", StatusID: strPtr("bad-uuid")}); err == nil || !strings.Contains(err.Error(), "status_id must be a valid UUID") {
		t.Fatalf("parseCreateTicketRequest(bad status) error = %v", err)
	}
	if _, err := parseCreateTicketRequest(projectID, rawCreateTicketRequest{Title: "ok", BudgetUSD: &negativeBudget}); err == nil || !strings.Contains(err.Error(), "budget_usd must be greater than or equal to zero") {
		t.Fatalf("parseCreateTicketRequest(negative budget) error = %v", err)
	}

	updateInput, err := parseUpdateTicketRequest(ticketID, rawUpdateTicketRequest{
		Title:          &title,
		Description:    &description,
		StatusID:       &statusID,
		Priority:       &priority,
		Type:           &ticketType,
		WorkflowID:     strPtr(" "),
		CreatedBy:      &createdBy,
		ParentTicketID: &parentID,
		ExternalRef:    &externalRef,
		BudgetUSD:      &budget,
	})
	if err != nil {
		t.Fatalf("parseUpdateTicketRequest() error = %v", err)
	}
	if !updateInput.Title.Set || updateInput.Title.Value != "Updated Title" {
		t.Fatalf("parseUpdateTicketRequest() = %+v", updateInput)
	}
	if !updateInput.WorkflowID.Set || updateInput.WorkflowID.Value != nil {
		t.Fatalf("parseUpdateTicketRequest().WorkflowID = %+v", updateInput.WorkflowID)
	}
	updateInput, err = parseUpdateTicketRequest(ticketID, rawUpdateTicketRequest{Priority: strPtr(" ")})
	if err != nil {
		t.Fatalf("parseUpdateTicketRequest(blank priority) error = %v", err)
	}
	if !updateInput.Priority.Set || updateInput.Priority.Value != nil {
		t.Fatalf("parseUpdateTicketRequest(blank priority) = %+v", updateInput)
	}
	if _, err := parseUpdateTicketRequest(ticketID, rawUpdateTicketRequest{Title: strPtr("  ")}); err == nil || !strings.Contains(err.Error(), "title must not be empty") {
		t.Fatalf("parseUpdateTicketRequest(blank title) error = %v", err)
	}
	if _, err := parseUpdateTicketRequest(ticketID, rawUpdateTicketRequest{BudgetUSD: &negativeBudget}); err == nil || !strings.Contains(err.Error(), "budget_usd must be greater than or equal to zero") {
		t.Fatalf("parseUpdateTicketRequest(negative budget) error = %v", err)
	}

	agentUpdateInput, err := parseAgentUpdateTicketRequest(
		context.Background(),
		projectID,
		ticketID,
		"agent:codex",
		rawAgentUpdateTicketRequest{
			Description:    &description,
			StatusName:     strPtr(" Done "),
			Archived:       boolPtr(true),
			Priority:       &priority,
			Type:           &ticketType,
			WorkflowID:     strPtr(" "),
			ParentTicketID: &parentID,
			ExternalRef:    &externalRef,
			BudgetUSD:      &budget,
		},
		func(_ context.Context, gotProjectID uuid.UUID, statusName string) (uuid.UUID, error) {
			if gotProjectID != projectID || strings.TrimSpace(statusName) != "Done" {
				t.Fatalf("resolveStatusID() = (%v, %q)", gotProjectID, statusName)
			}
			return uuid.MustParse(statusID), nil
		},
	)
	if err != nil {
		t.Fatalf("parseAgentUpdateTicketRequest() error = %v", err)
	}
	if !agentUpdateInput.Priority.Set || agentUpdateInput.Priority.Value == nil || *agentUpdateInput.Priority.Value != ticketservice.PriorityHigh {
		t.Fatalf("parseAgentUpdateTicketRequest().Priority = %+v", agentUpdateInput.Priority)
	}
	if !agentUpdateInput.Type.Set || agentUpdateInput.Type.Value != ticketservice.TypeBugfix {
		t.Fatalf("parseAgentUpdateTicketRequest().Type = %+v", agentUpdateInput.Type)
	}
	if !agentUpdateInput.WorkflowID.Set || agentUpdateInput.WorkflowID.Value != nil {
		t.Fatalf("parseAgentUpdateTicketRequest().WorkflowID = %+v", agentUpdateInput.WorkflowID)
	}
	if !agentUpdateInput.ParentTicketID.Set || agentUpdateInput.ParentTicketID.Value == nil || agentUpdateInput.ParentTicketID.Value.String() != parentID {
		t.Fatalf("parseAgentUpdateTicketRequest().ParentTicketID = %+v", agentUpdateInput.ParentTicketID)
	}
	if !agentUpdateInput.BudgetUSD.Set || agentUpdateInput.BudgetUSD.Value != budget {
		t.Fatalf("parseAgentUpdateTicketRequest().BudgetUSD = %+v", agentUpdateInput.BudgetUSD)
	}
	if !agentUpdateInput.CreatedBy.Set || agentUpdateInput.CreatedBy.Value != "agent:codex" {
		t.Fatalf("parseAgentUpdateTicketRequest().CreatedBy = %+v", agentUpdateInput.CreatedBy)
	}
	if _, err := parseAgentUpdateTicketRequest(
		context.Background(),
		projectID,
		ticketID,
		"agent:codex",
		rawAgentUpdateTicketRequest{BudgetUSD: &negativeBudget},
		func(context.Context, uuid.UUID, string) (uuid.UUID, error) { return uuid.Nil, nil },
	); err == nil || !strings.Contains(err.Error(), "budget_usd must be greater than or equal to zero") {
		t.Fatalf("parseAgentUpdateTicketRequest(negative budget) error = %v", err)
	}

	dependencyInput, err := parseAddDependencyRequest(ticketID, rawAddDependencyRequest{
		TargetTicketID: parentID,
		Type:           "sub-issue",
	})
	if err != nil {
		t.Fatalf("parseAddDependencyRequest() error = %v", err)
	}
	if dependencyInput.Input.Type != ticketservice.DependencyTypeSubIssue {
		t.Fatalf("parseAddDependencyRequest() = %+v", dependencyInput)
	}
	blockedByInput, err := parseAddDependencyRequest(ticketID, rawAddDependencyRequest{
		TargetTicketID: parentID,
		Type:           "blocked_by",
	})
	if err != nil {
		t.Fatalf("parseAddDependencyRequest(blocked_by) error = %v", err)
	}
	if blockedByInput.Input.Type != ticketservice.DependencyTypeBlocks || blockedByInput.Input.TicketID.String() != parentID || blockedByInput.Input.TargetTicketID != ticketID {
		t.Fatalf("parseAddDependencyRequest(blocked_by) = %+v", blockedByInput)
	}
	if _, err := parseAddDependencyRequest(ticketID, rawAddDependencyRequest{TargetTicketID: parentID, Type: "invalid"}); err == nil || !strings.Contains(err.Error(), "blocks, blocked_by, sub_issue") {
		t.Fatalf("parseAddDependencyRequest(invalid type) error = %v", err)
	}

	externalLinkInput, err := parseAddExternalLinkRequest(ticketID, rawAddExternalLinkRequest{
		Type:       " github_issue ",
		URL:        " https://github.com/acme/backend/issues/42 ",
		ExternalID: " gh-42 ",
		Title:      strPtr(" Ticket "),
		Status:     strPtr(" open "),
		Relation:   strPtr(" caused_by "),
	})
	if err != nil {
		t.Fatalf("parseAddExternalLinkRequest() error = %v", err)
	}
	if externalLinkInput.LinkType != ticketservice.ExternalLinkTypeGithubIssue || externalLinkInput.Relation != ticketservice.ExternalLinkRelationCausedBy {
		t.Fatalf("parseAddExternalLinkRequest() = %+v", externalLinkInput)
	}
	if _, err := parseAddExternalLinkRequest(ticketID, rawAddExternalLinkRequest{Type: "custom", URL: "/relative", ExternalID: "x"}); err == nil || !strings.Contains(err.Error(), "valid absolute URL") {
		t.Fatalf("parseAddExternalLinkRequest(bad URL) error = %v", err)
	}
	if _, err := parseAddExternalLinkRequest(ticketID, rawAddExternalLinkRequest{Type: "custom", URL: "https://example.com", ExternalID: " "}); err == nil || !strings.Contains(err.Error(), "external_id must not be empty") {
		t.Fatalf("parseAddExternalLinkRequest(blank external_id) error = %v", err)
	}

	createCommentInput, err := parseCreateTicketCommentRequest(ticketID, rawCreateTicketCommentRequest{Body: " hello ", CreatedBy: &createdBy})
	if err != nil {
		t.Fatalf("parseCreateTicketCommentRequest() error = %v", err)
	}
	if createCommentInput.Body != "hello" || createCommentInput.CreatedBy != "codex" {
		t.Fatalf("parseCreateTicketCommentRequest() = %+v", createCommentInput)
	}
	if _, err := parseCreateTicketCommentRequest(ticketID, rawCreateTicketCommentRequest{}); err == nil || !strings.Contains(err.Error(), "body must not be empty") {
		t.Fatalf("parseCreateTicketCommentRequest(blank body) error = %v", err)
	}

	updateCommentInput, err := parseUpdateTicketCommentRequest(ticketID, commentID, rawUpdateTicketCommentRequest{
		Body:       " updated ",
		EditedBy:   &createdBy,
		EditReason: &editReason,
	})
	if err != nil {
		t.Fatalf("parseUpdateTicketCommentRequest() error = %v", err)
	}
	if updateCommentInput.Body != "updated" || updateCommentInput.EditedBy != "codex" || updateCommentInput.EditReason != "demo workflow" {
		t.Fatalf("parseUpdateTicketCommentRequest() = %+v", updateCommentInput)
	}
	if _, err := parseUpdateTicketCommentRequest(ticketID, commentID, rawUpdateTicketCommentRequest{}); err == nil || !strings.Contains(err.Error(), "body must not be empty") {
		t.Fatalf("parseUpdateTicketCommentRequest(blank body) error = %v", err)
	}

	if got, err := parseTicketPriority(" urgent "); err != nil || got != ticketservice.PriorityUrgent {
		t.Fatalf("parseTicketPriority() = (%q, %v)", got, err)
	}
	if _, err := parseTicketPriority("invalid"); err == nil || !strings.Contains(err.Error(), "urgent, high, medium, low") {
		t.Fatalf("parseTicketPriority(invalid) error = %v", err)
	}
	if got, err := parseTicketType(" epic "); err != nil || got != ticketservice.TypeEpic {
		t.Fatalf("parseTicketType() = (%q, %v)", got, err)
	}
	if _, err := parseTicketType("invalid"); err == nil || !strings.Contains(err.Error(), "feature, bugfix, refactor, chore, epic") {
		t.Fatalf("parseTicketType(invalid) error = %v", err)
	}
	if got, err := parseExternalLinkType(" custom "); err != nil || got != ticketservice.ExternalLinkTypeCustom {
		t.Fatalf("parseExternalLinkType() = (%q, %v)", got, err)
	}
	if got, err := parseExternalLinkRelation(" related "); err != nil || got != ticketservice.ExternalLinkRelationRelated {
		t.Fatalf("parseExternalLinkRelation() = (%q, %v)", got, err)
	}
	if _, err := parseExternalLinkRelation("invalid"); err == nil || !strings.Contains(err.Error(), "resolves, related, caused_by") {
		t.Fatalf("parseExternalLinkRelation(invalid) error = %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest("GET", "/tickets?status=todo,doing&status=done&empty=%20,%20", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/projects/:projectId/tickets/:ticketId/dependencies/:dependencyId/comments/:commentId/links/:externalLinkId")
	ctx.SetParamNames("ticketId", "dependencyId", "commentId", "externalLinkId")
	ctx.SetParamValues(ticketID.String(), parentID, commentID.String(), workflowID)
	if got := parseCSVQueryValues(ctx, "status"); len(got) != 3 || got[0] != "todo" || got[2] != "done" {
		t.Fatalf("parseCSVQueryValues() = %#v", got)
	}
	if got, err := parseTicketID(ctx); err != nil || got != ticketID {
		t.Fatalf("parseTicketID() = (%v, %v)", got, err)
	}
	if got, err := parseDependencyID(ctx); err != nil || got.String() != parentID {
		t.Fatalf("parseDependencyID() = (%v, %v)", got, err)
	}
	if got, err := parseCommentID(ctx); err != nil || got != commentID {
		t.Fatalf("parseCommentID() = (%v, %v)", got, err)
	}
	if got, err := parseExternalLinkID(ctx); err != nil || got.String() != workflowID {
		t.Fatalf("parseExternalLinkID() = (%v, %v)", got, err)
	}
}

func TestTicketStatusAndWorkflowRequestParserCoverage(t *testing.T) {
	t.Parallel()

	projectID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	stageID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	statusID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	agentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	workflowID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	position := 2
	maxActiveRuns := 3
	maxConcurrent := 5
	maxRetryAttempts := 2
	timeoutMinutes := 15
	stallTimeoutMinutes := 3
	falseBool := false

	createStatusInput, err := parseCreateTicketStatusRequest(projectID, rawCreateTicketStatusRequest{
		Name:          " Ready ",
		Stage:         " started ",
		Color:         " green ",
		Icon:          " play ",
		Position:      &position,
		MaxActiveRuns: &maxActiveRuns,
		IsDefault:     true,
		Description:   " queued ",
	})
	if err != nil {
		t.Fatalf("parseCreateTicketStatusRequest() error = %v", err)
	}
	if createStatusInput.Name != "Ready" || createStatusInput.Stage.String() != "started" || createStatusInput.Color != "green" || !createStatusInput.Position.Set || createStatusInput.MaxActiveRuns == nil || *createStatusInput.MaxActiveRuns != 3 {
		t.Fatalf("parseCreateTicketStatusRequest() = %+v", createStatusInput)
	}
	if _, err := parseCreateTicketStatusRequest(projectID, rawCreateTicketStatusRequest{Name: "ok"}); err == nil || !strings.Contains(err.Error(), "color must not be empty") {
		t.Fatalf("parseCreateTicketStatusRequest(blank color) error = %v", err)
	}

	updateStatusInput, err := parseUpdateTicketStatusRequest(statusID, rawUpdateTicketStatusRequest{
		Name:          strPtr(" Done "),
		Stage:         strPtr(" completed "),
		Color:         strPtr(" blue "),
		Icon:          strPtr(" check "),
		Position:      &position,
		MaxActiveRuns: nullableIntField{Set: true, Value: &maxActiveRuns},
		IsDefault:     &falseBool,
		Description:   strPtr(" complete "),
	})
	if err != nil {
		t.Fatalf("parseUpdateTicketStatusRequest() error = %v", err)
	}
	if !updateStatusInput.Stage.Set || updateStatusInput.Stage.Value.String() != "completed" || !updateStatusInput.MaxActiveRuns.Set || updateStatusInput.MaxActiveRuns.Value == nil || *updateStatusInput.MaxActiveRuns.Value != maxActiveRuns {
		t.Fatalf("parseUpdateTicketStatusRequest() = %+v", updateStatusInput)
	}
	if _, err := parseUpdateTicketStatusRequest(statusID, rawUpdateTicketStatusRequest{Color: strPtr("  ")}); err == nil || !strings.Contains(err.Error(), "color must not be empty") {
		t.Fatalf("parseUpdateTicketStatusRequest(blank color) error = %v", err)
	}

	var nullableInt nullableIntField
	if err := json.Unmarshal([]byte(`3`), &nullableInt); err != nil || !nullableInt.Set || nullableInt.Value == nil || *nullableInt.Value != 3 {
		t.Fatalf("nullableIntField(3) = %+v, %v", nullableInt, err)
	}
	if !isJSONNull([]byte(" null ")) || isJSONNull([]byte(`"null"`)) {
		t.Fatal("isJSONNull() mismatch")
	}

	createWorkflowInput, err := parseCreateWorkflowRequest(projectID, rawCreateWorkflowRequest{
		AgentID:             agentID.String(),
		Name:                " CI ",
		Type:                " Fullstack Developer ",
		CreatedBy:           strPtr(" user:creator "),
		HarnessPath:         strPtr(" ./harness.md "),
		HarnessContent:      "content",
		Hooks:               map[string]any{"pre": true},
		MaxConcurrent:       &maxConcurrent,
		MaxRetryAttempts:    &maxRetryAttempts,
		TimeoutMinutes:      &timeoutMinutes,
		StallTimeoutMinutes: &stallTimeoutMinutes,
		IsActive:            &falseBool,
		PickupStatusIDs:     []string{stageID.String()},
		FinishStatusIDs:     []string{statusID.String()},
	})
	if err != nil {
		t.Fatalf("parseCreateWorkflowRequest() error = %v", err)
	}
	if createWorkflowInput.Name != "CI" || createWorkflowInput.Type != workflowservice.MustParseTypeLabel("Fullstack Developer") || createWorkflowInput.IsActive {
		t.Fatalf("parseCreateWorkflowRequest() = %+v", createWorkflowInput)
	}
	if createWorkflowInput.CreatedBy != "user:creator" {
		t.Fatalf("parseCreateWorkflowRequest().CreatedBy = %q", createWorkflowInput.CreatedBy)
	}
	if _, err := parseCreateWorkflowRequest(projectID, rawCreateWorkflowRequest{Name: "ok", Type: "Fullstack Developer", AgentID: "bad"}); err == nil || !strings.Contains(err.Error(), "agent_id must be a valid UUID") {
		t.Fatalf("parseCreateWorkflowRequest(bad agent) error = %v", err)
	}

	updateWorkflowInput, err := parseUpdateWorkflowRequest(workflowID, rawUpdateWorkflowRequest{
		AgentID:             strPtr(agentID.String()),
		Name:                strPtr(" Updated "),
		Type:                strPtr("QA Engineer"),
		EditedBy:            strPtr(" user:editor "),
		HarnessPath:         strPtr(" ./new.md "),
		Hooks:               &map[string]any{"post": true},
		MaxConcurrent:       &maxConcurrent,
		MaxRetryAttempts:    &maxRetryAttempts,
		TimeoutMinutes:      &timeoutMinutes,
		StallTimeoutMinutes: &stallTimeoutMinutes,
		IsActive:            &falseBool,
		PickupStatusIDs:     &[]string{stageID.String()},
		FinishStatusIDs:     &[]string{statusID.String()},
	})
	if err != nil {
		t.Fatalf("parseUpdateWorkflowRequest() error = %v", err)
	}
	if !updateWorkflowInput.Name.Set || updateWorkflowInput.Name.Value != "Updated" {
		t.Fatalf("parseUpdateWorkflowRequest() = %+v", updateWorkflowInput)
	}
	if updateWorkflowInput.EditedBy != "user:editor" {
		t.Fatalf("parseUpdateWorkflowRequest().EditedBy = %q", updateWorkflowInput.EditedBy)
	}
	if _, err := parseUpdateWorkflowRequest(workflowID, rawUpdateWorkflowRequest{RoleSlug: strPtr("qa-engineer")}); err == nil || !strings.Contains(err.Error(), "role_slug cannot be updated") {
		t.Fatalf("parseUpdateWorkflowRequest(role slug) error = %v", err)
	}
	if _, err := parseUpdateWorkflowRequest(workflowID, rawUpdateWorkflowRequest{MaxRetryAttempts: intPtr(-1)}); err == nil || !strings.Contains(err.Error(), "greater than or equal to zero") {
		t.Fatalf("parseUpdateWorkflowRequest(negative retry) error = %v", err)
	}

	harnessInput, err := parseUpdateHarnessRequest(workflowID, rawUpdateHarnessRequest{
		Content:  "body",
		EditedBy: strPtr(" user:harness "),
	})
	if err != nil || harnessInput.Content != "body" {
		t.Fatalf("parseUpdateHarnessRequest() = (%+v, %v)", harnessInput, err)
	}
	if harnessInput.EditedBy != "user:harness" {
		t.Fatalf("parseUpdateHarnessRequest().EditedBy = %q", harnessInput.EditedBy)
	}
	if _, err := parseUpdateHarnessRequest(workflowID, rawUpdateHarnessRequest{}); err == nil || !strings.Contains(err.Error(), "content must not be empty") {
		t.Fatalf("parseUpdateHarnessRequest(blank) error = %v", err)
	}

	if got, err := parseWorkflowTypeLabel("Release Captain"); err != nil || got != workflowservice.MustParseTypeLabel("Release Captain") {
		t.Fatalf("parseWorkflowTypeLabel() = (%q, %v)", got, err)
	}
	if _, err := parseWorkflowTypeLabel(" \n "); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("parseWorkflowTypeLabel(empty) error = %v", err)
	}
	if got, err := parseUUIDString("agent_id", agentID.String()); err != nil || got != agentID {
		t.Fatalf("parseUUIDString() = (%v, %v)", got, err)
	}
	if got, err := parseOptionalUUIDString("status_id", strPtr(" ")); err != nil || got != nil {
		t.Fatalf("parseOptionalUUIDString(blank) = (%v, %v)", got, err)
	}
	if got, err := parseConcurrencyLimit("max_concurrent", nil); err != nil || got != 0 {
		t.Fatalf("parseConcurrencyLimit(default) = (%d, %v)", got, err)
	}
	if got, err := parseConcurrencyLimit("max_concurrent", intPtr(0)); err != nil || got != 0 {
		t.Fatalf("parseConcurrencyLimit(zero) = (%d, %v)", got, err)
	}
	if _, err := parseConcurrencyLimit("max_concurrent", intPtr(-1)); err == nil || !strings.Contains(err.Error(), "greater than or equal to zero") {
		t.Fatalf("parseConcurrencyLimit(invalid) error = %v", err)
	}
	if got, err := parseMaxRetryAttempts(nil, 2); err != nil || got != 2 {
		t.Fatalf("parseMaxRetryAttempts(default) = (%d, %v)", got, err)
	}
	if _, err := parseMaxRetryAttempts(intPtr(-1), 2); err == nil || !strings.Contains(err.Error(), "greater than or equal to zero") {
		t.Fatalf("parseMaxRetryAttempts(invalid) error = %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest("GET", "/projects/"+projectID.String()+"/ticket-statuses/"+statusID.String(), nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/projects/:projectId/ticket-statuses/:statusId")
	ctx.SetParamNames("projectId", "statusId")
	ctx.SetParamValues(projectID.String(), statusID.String())
	if got, err := parseProjectID(ctx); err != nil || got != projectID {
		t.Fatalf("parseProjectID() = (%v, %v)", got, err)
	}
	if got, err := parseStatusID(ctx); err != nil || got != statusID {
		t.Fatalf("parseStatusID() = (%v, %v)", got, err)
	}
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
