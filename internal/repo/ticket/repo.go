package ticket

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	"github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

const (
	defaultCreatedBy        = "user:api"
	defaultIdentifierPrefix = "ASE"
)

var errUnavailable = errors.New("ticket repository unavailable")

type ActivityRepository struct {
	client *ent.Client
}

func NewActivityRepository(client *ent.Client) *ActivityRepository {
	return &ActivityRepository{client: client}
}

type QueryRepository struct {
	client *ent.Client
}

func NewQueryRepository(client *ent.Client) *QueryRepository {
	return &QueryRepository{client: client}
}

type CommandRepository struct {
	client *ent.Client
}

func NewCommandRepository(client *ent.Client) *CommandRepository {
	return &CommandRepository{client: client}
}

type LinkRepository struct {
	client *ent.Client
}

func NewLinkRepository(client *ent.Client) *LinkRepository {
	return &LinkRepository{client: client}
}

type CommentRepository struct {
	client *ent.Client
}

func NewCommentRepository(client *ent.Client) *CommentRepository {
	return &CommentRepository{client: client}
}

type RuntimeRepository struct {
	client *ent.Client
}

func NewRuntimeRepository(client *ent.Client) *RuntimeRepository {
	return &RuntimeRepository{client: client}
}

type UsageRepository struct {
	client *ent.Client
}

func NewUsageRepository(client *ent.Client) *UsageRepository {
	return &UsageRepository{client: client}
}

func cloneAnyMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func ensureProjectExists(ctx context.Context, client *ent.Client, projectID uuid.UUID) error {
	exists, err := client.Project.Query().Where(project.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func ensureProjectExistsTx(ctx context.Context, tx *ent.Tx, projectID uuid.UUID) error {
	exists, err := tx.Project.Query().Where(project.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func resolveCreateStatusID(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, inputStatusID *uuid.UUID) (uuid.UUID, error) {
	if inputStatusID != nil {
		if err := ensureStatusBelongsToProject(ctx, tx, projectID, *inputStatusID); err != nil {
			return uuid.UUID{}, err
		}
		return *inputStatusID, nil
	}

	defaultStatus, err := tx.TicketStatus.Query().
		Where(
			entticketstatus.ProjectIDEQ(projectID),
			entticketstatus.IsDefault(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.UUID{}, ErrStatusNotFound
		}
		return uuid.UUID{}, fmt.Errorf("get default project ticket status: %w", err)
	}

	return defaultStatus.ID, nil
}

func mapTicketReadError(action string, err error) error {
	if ent.IsNotFound(err) {
		return ErrTicketNotFound
	}

	return fmt.Errorf("%s: %w", action, err)
}

func mapTicketWriteError(action string, err error) error {
	switch {
	case ent.IsConstraintError(err):
		switch message := strings.ToLower(err.Error()); {
		case strings.Contains(message, "ticketdependency_source_ticket_id_target_ticket_id_type"):
			return ErrDependencyConflict
		case strings.Contains(message, "ticket_external_links_ticket_id_external_id"),
			strings.Contains(message, "ticketexternallink_ticket_id_external_id"),
			(strings.Contains(message, "ticket_external_links") && strings.Contains(message, "external_id")):
			return ErrExternalLinkConflict
		case strings.Contains(message, "ticket_project_id_identifier"),
			strings.Contains(message, "ticket_identifier"):
			return ErrTicketConflict
		default:
			return fmt.Errorf("%s: %w", action, err)
		}
	case ent.IsNotFound(err):
		return ErrTicketNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func ensureStatusBelongsToProject(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, statusID uuid.UUID) error {
	exists, err := tx.TicketStatus.Query().
		Where(
			entticketstatus.ID(statusID),
			entticketstatus.ProjectIDEQ(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check ticket status existence: %w", err)
	}
	if !exists {
		return ErrStatusNotFound
	}

	return nil
}

func ensureWorkflowBelongsToProject(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, workflowID uuid.UUID) error {
	exists, err := tx.Workflow.Query().
		Where(
			entworkflow.ID(workflowID),
			entworkflow.ProjectIDEQ(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow existence: %w", err)
	}
	if !exists {
		return ErrWorkflowNotFound
	}

	return nil
}

func ensureStatusAllowedByWorkflowFinishSet(ctx context.Context, tx *ent.Tx, workflowID uuid.UUID, statusID uuid.UUID) error {
	workflowItem, err := tx.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrWorkflowNotFound
		}
		return fmt.Errorf("load workflow finish statuses: %w", err)
	}

	for _, finishStatus := range workflowItem.Edges.FinishStatuses {
		if finishStatus.ID == statusID {
			return nil
		}
	}
	return ErrStatusNotAllowed
}

func classifyStatusChangeRunDisposition(
	ctx context.Context,
	tx *ent.Tx,
	current *ent.Ticket,
	nextStatusID uuid.UUID,
) (ticketing.StatusChangeRunDisposition, error) {
	if current == nil || current.CurrentRunID == nil {
		return ticketing.StatusChangeRunDispositionRetain, nil
	}
	if current.WorkflowID == nil {
		return ticketing.StatusChangeRunDispositionCancel, nil
	}

	workflowItem, err := tx.Workflow.Query().
		Where(entworkflow.IDEQ(*current.WorkflowID)).
		WithPickupStatuses().
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", ErrWorkflowNotFound
		}
		return "", fmt.Errorf("load workflow status ownership: %w", err)
	}

	pickupStatusIDs := make([]uuid.UUID, 0, len(workflowItem.Edges.PickupStatuses))
	for _, status := range workflowItem.Edges.PickupStatuses {
		pickupStatusIDs = append(pickupStatusIDs, status.ID)
	}
	finishStatusIDs := make([]uuid.UUID, 0, len(workflowItem.Edges.FinishStatuses))
	for _, status := range workflowItem.Edges.FinishStatuses {
		finishStatusIDs = append(finishStatusIDs, status.ID)
	}

	return ticketing.ClassifyStatusChangeRunDisposition(
		true,
		nextStatusID,
		pickupStatusIDs,
		finishStatusIDs,
	), nil
}

func ensureTicketBelongsToProject(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, ticketID uuid.UUID, notFound error) error {
	exists, err := tx.Ticket.Query().
		Where(
			entticket.ID(ticketID),
			entticket.ProjectIDEQ(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check ticket existence: %w", err)
	}
	if !exists {
		return notFound
	}

	return nil
}

func ensureTargetMachineBelongsToProjectOrganization(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, machineID uuid.UUID) error {
	exists, err := tx.Machine.Query().
		Where(
			entmachine.IDEQ(machineID),
			entmachine.HasOrganizationWith(entorganization.HasProjectsWith(project.ID(projectID))),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check target machine existence: %w", err)
	}
	if !exists {
		return ErrTargetMachineNotFound
	}

	return nil
}

func releaseTicketAgentClaim(ctx context.Context, tx *ent.Tx, ticketItem *ent.Ticket, runStatus entagentrun.Status) error {
	if ticketItem == nil {
		return nil
	}

	var runItem *ent.AgentRun
	if ticketItem.CurrentRunID != nil {
		currentRun, err := tx.AgentRun.Get(ctx, *ticketItem.CurrentRunID)
		if err != nil {
			return fmt.Errorf("load current agent run: %w", err)
		}
		runItem = currentRun

		runUpdate := tx.AgentRun.UpdateOneID(currentRun.ID).
			SetStatus(runStatus).
			SetTerminalAt(timeNowUTC()).
			ClearSessionID().
			ClearRuntimeStartedAt().
			ClearLastHeartbeatAt()
		if runStatus != entagentrun.StatusErrored {
			runUpdate.SetLastError("")
		}
		if _, err := runUpdate.Save(ctx); err != nil {
			return fmt.Errorf("finalize current agent run: %w", err)
		}
	}

	if runItem != nil {
		if _, err := tx.Agent.UpdateOneID(runItem.AgentID).
			SetRuntimeControlState(entagent.RuntimeControlStateActive).
			Save(ctx); err != nil {
			return fmt.Errorf("reset current run agent runtime control state: %w", err)
		}
	}

	return nil
}

// InstallRetryTokenHooks keeps retry token semantics consistent for direct ent mutations.
func InstallRetryTokenHooks(client *ent.Client) {
	if client == nil {
		return
	}

	client.Ticket.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			ticketMutation, ok := mutation.(*ent.TicketMutation)
			if !ok {
				return next.Mutate(ctx, mutation)
			}

			ensureTicketCreateRetryToken(ticketMutation)
			normalizeTicketStatusTransition(ticketMutation)

			return next.Mutate(ctx, mutation)
		})
	})
}

// ScheduleRetryOne rotates the retry token and records a delayed retry intent.
func ScheduleRetryOne(update *ent.TicketUpdateOne, nextRetryAt time.Time, pauseReason string) *ent.TicketUpdateOne {
	if update == nil {
		return nil
	}

	update.SetRetryToken(NewRetryToken()).
		SetNextRetryAt(nextRetryAt).
		SetRetryPaused(pauseReason != "")
	if pauseReason == "" {
		return update.ClearPauseReason()
	}

	return update.SetPauseReason(pauseReason)
}

// ScheduleRetry rotates the retry token and records a delayed retry intent.
func ScheduleRetry(update *ent.TicketUpdate, nextRetryAt time.Time, pauseReason string) *ent.TicketUpdate {
	if update == nil {
		return nil
	}

	update.
		SetRetryToken(NewRetryToken()).
		SetNextRetryAt(nextRetryAt).
		SetRetryPaused(pauseReason != "")
	if pauseReason == "" {
		return update.ClearPauseReason()
	}

	return update.SetPauseReason(pauseReason)
}

// ResetRetryBaseline clears active retry-cycle state after a healthy/manual-forward transition
// and rotates the retry token so stale delayed retries are discarded.
// attempt_count stays cumulative per the PRD, while current failure streak state is normalized.
func ResetRetryBaseline(update *ent.TicketUpdateOne, current *ent.Ticket) *ent.TicketUpdateOne {
	if update == nil || current == nil {
		return update
	}

	update.SetRetryToken(NewRetryToken())
	if current.ConsecutiveErrors != 0 {
		update.SetConsecutiveErrors(0)
	}
	if current.StallCount != 0 {
		update.SetStallCount(0)
	}
	if current.NextRetryAt != nil {
		update.ClearNextRetryAt()
	}
	if current.RetryPaused {
		update.SetRetryPaused(false)
	}
	if current.PauseReason != "" {
		update.ClearPauseReason()
	}

	return update
}

func ensureTicketCreateRetryToken(mutation *ent.TicketMutation) {
	if mutation == nil || !mutation.Op().Is(ent.OpCreate) {
		return
	}
	if _, ok := mutation.RetryToken(); ok {
		return
	}

	mutation.SetRetryToken(NewRetryToken())
}

func normalizeTicketStatusTransition(mutation *ent.TicketMutation) {
	if mutation == nil || !mutation.Op().Is(ent.OpUpdate|ent.OpUpdateOne) {
		return
	}
	if _, ok := mutation.StatusID(); !ok {
		return
	}
	if _, ok := mutation.RetryToken(); !ok {
		mutation.SetRetryToken(NewRetryToken())
	}

	mutation.SetConsecutiveErrors(0)
	mutation.ClearNextRetryAt()
	mutation.SetRetryPaused(false)
	mutation.ClearPauseReason()
}

func ensureParentDoesNotCreateCycle(ctx context.Context, tx *ent.Tx, ticketID uuid.UUID, parentTicketID uuid.UUID) error {
	if ticketID == parentTicketID {
		return ErrInvalidDependency
	}

	seen := map[uuid.UUID]struct{}{ticketID: {}}
	currentID := parentTicketID
	for currentID != uuid.Nil {
		if _, ok := seen[currentID]; ok {
			return ErrInvalidDependency
		}
		seen[currentID] = struct{}{}

		current, err := tx.Ticket.Get(ctx, currentID)
		if err != nil {
			if ent.IsNotFound(err) {
				return ErrParentTicketNotFound
			}
			return fmt.Errorf("load ticket parent chain: %w", err)
		}
		if current.ParentTicketID == nil {
			return nil
		}
		currentID = *current.ParentTicketID
	}

	return nil
}

func syncSubIssueDependencies(ctx context.Context, tx *ent.Tx, ticketID uuid.UUID, parentTicketID *uuid.UUID) error {
	existing, err := tx.TicketDependency.Query().
		Where(
			entticketdependency.SourceTicketIDEQ(ticketID),
			entticketdependency.TypeEQ(entticketdependency.TypeSubIssue),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query sub-issue dependencies: %w", err)
	}

	keepID := uuid.Nil
	if parentTicketID != nil {
		for _, dependency := range existing {
			if dependency.TargetTicketID == *parentTicketID {
				keepID = dependency.ID
				break
			}
		}
	}

	for _, dependency := range existing {
		if dependency.ID == keepID {
			continue
		}
		if err := tx.TicketDependency.DeleteOneID(dependency.ID).Exec(ctx); err != nil {
			return fmt.Errorf("delete stale sub-issue dependency: %w", err)
		}
	}

	if parentTicketID == nil || keepID != uuid.Nil {
		return nil
	}

	_, err = tx.TicketDependency.Create().
		SetSourceTicketID(ticketID).
		SetTargetTicketID(*parentTicketID).
		SetType(entticketdependency.TypeSubIssue).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("create sub-issue dependency: %w", err)
	}

	return nil
}

func ensureSubIssueDependency(ctx context.Context, tx *ent.Tx, sourceTicketID uuid.UUID, targetTicketID uuid.UUID) (*ent.TicketDependency, error) {
	if err := syncSubIssueDependencies(ctx, tx, sourceTicketID, &targetTicketID); err != nil {
		return nil, err
	}

	dependency, err := tx.TicketDependency.Query().
		Where(
			entticketdependency.SourceTicketIDEQ(sourceTicketID),
			entticketdependency.TargetTicketIDEQ(targetTicketID),
			entticketdependency.TypeEQ(entticketdependency.TypeSubIssue),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("reload sub-issue dependency: %w", err)
	}

	return dependency, nil
}

func nextTicketIdentifier(ctx context.Context, tx *ent.Tx, projectID uuid.UUID) (string, error) {
	items, err := tx.Ticket.Query().
		Where(entticket.ProjectIDEQ(projectID)).
		Select(entticket.FieldIdentifier).
		All(ctx)
	if err != nil {
		return "", fmt.Errorf("list project identifiers: %w", err)
	}

	maxValue := 0
	for _, item := range items {
		value, ok := parseIdentifierSequence(item.Identifier)
		if ok && value > maxValue {
			maxValue = value
		}
	}

	return fmt.Sprintf("%s-%d", defaultIdentifierPrefix, maxValue+1), nil
}

func parseIdentifierSequence(identifier string) (int, bool) {
	if !strings.HasPrefix(identifier, defaultIdentifierPrefix+"-") {
		return 0, false
	}

	value, err := strconv.Atoi(strings.TrimPrefix(identifier, defaultIdentifierPrefix+"-"))
	if err != nil || value < 1 {
		return 0, false
	}

	return value, true
}

func resolveCreatedBy(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return defaultCreatedBy
	}

	return strings.TrimSpace(raw)
}

func toEntTicketPriority(priority Priority) entticket.Priority {
	return entticket.Priority(priority.String())
}

func toEntTicketPriorities(priorities []Priority) []entticket.Priority {
	items := make([]entticket.Priority, 0, len(priorities))
	for _, priority := range priorities {
		items = append(items, toEntTicketPriority(priority))
	}
	return items
}

func toEntTicketType(ticketType Type) entticket.Type {
	return entticket.Type(ticketType.String())
}

func toEntDependencyType(dependencyType DependencyType) entticketdependency.Type {
	return entticketdependency.Type(dependencyType.String())
}

func toEntExternalLinkType(linkType ExternalLinkType) string {
	return linkType.String()
}

func optionalUUIDPointerEqual(left *uuid.UUID, right *uuid.UUID) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func mapTicket(item *ent.Ticket) Ticket {
	result := Ticket{
		ID:                   item.ID,
		ProjectID:            item.ProjectID,
		Identifier:           item.Identifier,
		Title:                item.Title,
		Description:          item.Description,
		StatusID:             item.StatusID,
		Priority:             Priority(item.Priority),
		Archived:             item.Archived,
		Type:                 Type(item.Type),
		WorkflowID:           item.WorkflowID,
		CurrentRunID:         item.CurrentRunID,
		TargetMachineID:      item.TargetMachineID,
		CreatedBy:            item.CreatedBy,
		Children:             []TicketReference{},
		Dependencies:         []Dependency{},
		IncomingDependencies: []Dependency{},
		ExternalLinks:        []ExternalLink{},
		ExternalRef:          item.ExternalRef,
		BudgetUSD:            item.BudgetUsd,
		CostTokensInput:      item.CostTokensInput,
		CostTokensOutput:     item.CostTokensOutput,
		CostAmount:           item.CostAmount,
		AttemptCount:         item.AttemptCount,
		ConsecutiveErrors:    item.ConsecutiveErrors,
		StartedAt:            item.StartedAt,
		CompletedAt:          item.CompletedAt,
		NextRetryAt:          item.NextRetryAt,
		RetryPaused:          item.RetryPaused,
		PauseReason:          item.PauseReason,
		CreatedAt:            item.CreatedAt,
	}

	if item.Edges.Status != nil {
		result.StatusName = item.Edges.Status.Name
		result.StatusStage = string(item.Edges.Status.Stage)
	}
	if item.Edges.Parent != nil {
		parent := mapTicketReference(item.Edges.Parent)
		result.Parent = &parent
	}
	for _, child := range item.Edges.Children {
		result.Children = append(result.Children, mapTicketReference(child))
	}
	for _, dependency := range item.Edges.OutgoingDependencies {
		result.Dependencies = append(result.Dependencies, mapDependency(dependency))
	}
	for _, dependency := range item.Edges.IncomingDependencies {
		result.IncomingDependencies = append(result.IncomingDependencies, mapIncomingDependency(dependency))
	}
	for _, externalLink := range item.Edges.ExternalLinks {
		result.ExternalLinks = append(result.ExternalLinks, mapExternalLink(externalLink))
	}
	for _, scope := range item.Edges.RepoScopes {
		if strings.TrimSpace(scope.PullRequestURL) != "" {
			result.PullRequestURLs = append(result.PullRequestURLs, scope.PullRequestURL)
		}
	}

	return result
}

func mapDependency(item *ent.TicketDependency) Dependency {
	dependency := Dependency{
		ID:   item.ID,
		Type: DependencyType(item.Type),
	}
	if item.Edges.TargetTicket != nil {
		dependency.Target = mapTicketReference(item.Edges.TargetTicket)
	}

	return dependency
}

func mapIncomingDependency(item *ent.TicketDependency) Dependency {
	dependency := Dependency{
		ID:   item.ID,
		Type: DependencyType(item.Type),
	}
	if item.Edges.SourceTicket != nil {
		dependency.Target = mapTicketReference(item.Edges.SourceTicket)
	}

	return dependency
}

func mapExternalLink(item *ent.TicketExternalLink) ExternalLink {
	return ExternalLink{
		ID:         item.ID,
		LinkType:   ExternalLinkType(item.LinkType),
		URL:        item.URL,
		ExternalID: item.ExternalID,
		Title:      item.Title,
		Status:     item.Status,
		CreatedAt:  item.CreatedAt,
	}
}

func mapComment(item *ent.TicketComment) Comment {
	return Comment{
		ID:           item.ID,
		TicketID:     item.TicketID,
		BodyMarkdown: item.Body,
		CreatedBy:    item.CreatedBy,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
		EditedAt:     item.EditedAt,
		EditCount:    item.EditCount,
		LastEditedBy: item.LastEditedBy,
		IsDeleted:    item.IsDeleted,
		DeletedAt:    item.DeletedAt,
		DeletedBy:    item.DeletedBy,
	}
}

func mapCommentRevision(item *ent.TicketCommentRevision) CommentRevision {
	return CommentRevision{
		ID:             item.ID,
		CommentID:      item.CommentID,
		RevisionNumber: item.RevisionNumber,
		BodyMarkdown:   item.BodyMarkdown,
		EditedBy:       item.EditedBy,
		EditedAt:       item.EditedAt,
		EditReason:     item.EditReason,
	}
}

func mapTicketReference(item *ent.Ticket) TicketReference {
	reference := TicketReference{
		ID:         item.ID,
		Identifier: item.Identifier,
		Title:      item.Title,
		StatusID:   item.StatusID,
	}
	if item.Edges.Status != nil {
		reference.StatusName = item.Edges.Status.Name
	}

	return reference
}

func rollback(tx *ent.Tx) {
	if tx == nil {
		return
	}
	_ = tx.Rollback()
}

func reconcileBudgetPauseState(builder *ent.TicketUpdateOne, current *ent.Ticket, budgetUSD float64) {
	if builder == nil || current == nil {
		return
	}

	if ticketing.ShouldPauseForBudget(current.CostAmount, budgetUSD) {
		if !current.RetryPaused || current.PauseReason == "" || current.PauseReason == ticketing.PauseReasonBudgetExhausted.String() {
			builder.SetRetryPaused(true).
				SetPauseReason(ticketing.PauseReasonBudgetExhausted.String())
		}
		return
	}

	if current.PauseReason == ticketing.PauseReasonBudgetExhausted.String() {
		if current.RetryPaused {
			builder.SetRetryPaused(false)
		}
		builder.ClearPauseReason()
	}
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := strings.TrimSpace(*value)
	return &copied
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneOptionalText(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
