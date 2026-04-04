package ticketstatus

import (
	"context"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	domain "github.com/BetterAndBetterII/openase/internal/domain/ticketstatus"
	"github.com/google/uuid"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.Status, error) {
	if err := ensureProjectExists(ctx, r.client.Project, projectID); err != nil {
		return nil, err
	}

	statuses, err := r.client.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket statuses: %w", err)
	}
	activeRunsByStatusID, err := countProjectStatusActiveRuns(ctx, r.client, projectID)
	if err != nil {
		return nil, fmt.Errorf("list ticket status runtime snapshots: %w", err)
	}

	return mapStatuses(statuses, activeRunsByStatusID), nil
}

func (r *EntRepository) ResolveStatusIDByName(ctx context.Context, projectID uuid.UUID, name string) (uuid.UUID, error) {
	if err := ensureProjectExists(ctx, r.client.Project, projectID); err != nil {
		return uuid.UUID{}, err
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return uuid.UUID{}, fmt.Errorf("status name must not be empty")
	}

	statuses, err := r.client.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		All(ctx)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("list ticket statuses for name resolution: %w", err)
	}
	var matchID uuid.UUID
	matchCount := 0
	for _, status := range statuses {
		if strings.EqualFold(strings.TrimSpace(status.Name), trimmed) {
			matchID = status.ID
			matchCount++
		}
	}
	if matchCount > 1 {
		return uuid.UUID{}, domain.ErrDuplicateStatusName
	}
	if matchCount == 1 {
		return matchID, nil
	}

	return uuid.UUID{}, domain.ErrStatusNotFound
}

func (r *EntRepository) Get(ctx context.Context, statusID uuid.UUID) (domain.Status, error) {
	item, err := r.client.TicketStatus.Get(ctx, statusID)
	if err != nil {
		return domain.Status{}, mapNotFoundError(err, domain.ErrStatusNotFound)
	}
	activeRuns, err := countStatusActiveRuns(ctx, r.client, item.ProjectID, item.ID)
	if err != nil {
		return domain.Status{}, fmt.Errorf("count ticket status active runs: %w", err)
	}
	return mapStatus(item, activeRuns), nil
}

func (r *EntRepository) Create(ctx context.Context, input domain.CreateInput) (domain.Status, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Status{}, fmt.Errorf("start ticket status create tx: %w", err)
	}
	defer rollback(tx)

	if err := ensureProjectExists(ctx, tx.Project, input.ProjectID); err != nil {
		return domain.Status{}, err
	}
	if err := ensureUniqueStatusName(ctx, tx.TicketStatus, input.ProjectID, input.Name); err != nil {
		return domain.Status{}, err
	}

	projectStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(input.ProjectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return domain.Status{}, fmt.Errorf("query project ticket statuses: %w", err)
	}

	position := input.Position.Value
	if !input.Position.Set {
		position = nextStatusPosition(projectStatuses)
	}

	isDefault := input.IsDefault || !hasDefault(projectStatuses)
	if isDefault {
		if err := clearProjectDefault(ctx, tx, input.ProjectID); err != nil {
			return domain.Status{}, err
		}
	}

	builder := tx.TicketStatus.Create().
		SetProjectID(input.ProjectID).
		SetName(input.Name).
		SetStage(toEntStatusStage(input.Stage)).
		SetColor(input.Color).
		SetPosition(position).
		SetIsDefault(isDefault)

	if input.Icon != "" {
		builder.SetIcon(input.Icon)
	}
	if input.MaxActiveRuns != nil {
		builder.SetMaxActiveRuns(*input.MaxActiveRuns)
	}
	if input.Description != "" {
		builder.SetDescription(input.Description)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return domain.Status{}, mapPersistenceError("create ticket status", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Status{}, fmt.Errorf("commit ticket status create tx: %w", err)
	}

	saved, err := r.client.TicketStatus.Get(ctx, created.ID)
	if err != nil {
		return domain.Status{}, fmt.Errorf("load created ticket status: %w", err)
	}
	activeRuns, err := countStatusActiveRuns(ctx, r.client, saved.ProjectID, saved.ID)
	if err != nil {
		return domain.Status{}, fmt.Errorf("count created ticket status active runs: %w", err)
	}
	return mapStatus(saved, activeRuns), nil
}

func (r *EntRepository) Update(ctx context.Context, input domain.UpdateInput) (domain.Status, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Status{}, fmt.Errorf("start ticket status update tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.TicketStatus.Get(ctx, input.StatusID)
	if err != nil {
		return domain.Status{}, mapNotFoundError(err, domain.ErrStatusNotFound)
	}
	if input.Name.Set {
		if err := ensureUniqueStatusName(ctx, tx.TicketStatus, current.ProjectID, input.Name.Value, current.ID); err != nil {
			return domain.Status{}, err
		}
	}

	if input.IsDefault.Set && !input.IsDefault.Value && current.IsDefault {
		otherDefault, err := tx.TicketStatus.Query().
			Where(
				entticketstatus.ProjectIDEQ(current.ProjectID),
				entticketstatus.IDNEQ(current.ID),
				entticketstatus.IsDefault(true),
			).
			Exist(ctx)
		if err != nil {
			return domain.Status{}, fmt.Errorf("check remaining default ticket status: %w", err)
		}
		if !otherDefault {
			return domain.Status{}, domain.ErrDefaultStatusRequired
		}
	}

	if input.IsDefault.Set && input.IsDefault.Value {
		if err := clearProjectDefault(ctx, tx, current.ProjectID); err != nil {
			return domain.Status{}, err
		}
	}

	builder := tx.TicketStatus.UpdateOneID(current.ID)
	if input.Name.Set {
		builder.SetName(input.Name.Value)
	}
	if input.Stage.Set {
		builder.SetStage(toEntStatusStage(input.Stage.Value))
	}
	if input.Color.Set {
		builder.SetColor(input.Color.Value)
	}
	if input.Icon.Set {
		if input.Icon.Value == "" {
			builder.ClearIcon()
		} else {
			builder.SetIcon(input.Icon.Value)
		}
	}
	if input.Position.Set {
		builder.SetPosition(input.Position.Value)
	}
	if input.MaxActiveRuns.Set {
		if input.MaxActiveRuns.Value == nil {
			builder.ClearMaxActiveRuns()
		} else {
			builder.SetMaxActiveRuns(*input.MaxActiveRuns.Value)
		}
	}
	if input.IsDefault.Set {
		builder.SetIsDefault(input.IsDefault.Value)
	}
	if input.Description.Set {
		if input.Description.Value == "" {
			builder.ClearDescription()
		} else {
			builder.SetDescription(input.Description.Value)
		}
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return domain.Status{}, mapPersistenceError("update ticket status", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Status{}, fmt.Errorf("commit ticket status update tx: %w", err)
	}

	saved, err := r.client.TicketStatus.Get(ctx, updated.ID)
	if err != nil {
		return domain.Status{}, fmt.Errorf("load updated ticket status: %w", err)
	}
	activeRuns, err := countStatusActiveRuns(ctx, r.client, saved.ProjectID, saved.ID)
	if err != nil {
		return domain.Status{}, fmt.Errorf("count updated ticket status active runs: %w", err)
	}
	return mapStatus(saved, activeRuns), nil
}

func (r *EntRepository) Delete(ctx context.Context, statusID uuid.UUID) (domain.DeleteResult, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.DeleteResult{}, fmt.Errorf("start ticket status delete tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.TicketStatus.Get(ctx, statusID)
	if err != nil {
		return domain.DeleteResult{}, mapNotFoundError(err, domain.ErrStatusNotFound)
	}

	projectStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(current.ProjectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return domain.DeleteResult{}, fmt.Errorf("query project ticket statuses: %w", err)
	}

	replacement, err := selectReplacementStatus(projectStatuses, current)
	if err != nil {
		return domain.DeleteResult{}, err
	}

	if err := rebindStatusReferences(ctx, tx, current.ID, replacement.ID, replacement.ID); err != nil {
		return domain.DeleteResult{}, err
	}

	if current.IsDefault && !replacement.IsDefault {
		if err := clearProjectDefault(ctx, tx, current.ProjectID); err != nil {
			return domain.DeleteResult{}, err
		}
		if _, err := tx.TicketStatus.UpdateOneID(replacement.ID).SetIsDefault(true).Save(ctx); err != nil {
			return domain.DeleteResult{}, mapPersistenceError("promote replacement default ticket status", err)
		}
	}

	if err := tx.TicketStatus.DeleteOneID(current.ID).Exec(ctx); err != nil {
		return domain.DeleteResult{}, fmt.Errorf("delete ticket status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.DeleteResult{}, fmt.Errorf("commit ticket status delete tx: %w", err)
	}

	return domain.DeleteResult{
		DeletedStatusID:     current.ID,
		ReplacementStatusID: replacement.ID,
	}, nil
}

func (r *EntRepository) ResetToDefaultTemplate(ctx context.Context, projectID uuid.UUID, template []domain.TemplateStatus) ([]domain.Status, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("start ticket status reset tx: %w", err)
	}
	defer rollback(tx)

	if err := ensureProjectExists(ctx, tx.Project, projectID); err != nil {
		return nil, err
	}

	existingStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query project ticket statuses: %w", err)
	}

	if err := clearProjectDefault(ctx, tx, projectID); err != nil {
		return nil, err
	}

	existingByName := make(map[string]*ent.TicketStatus, len(existingStatuses))
	for _, status := range existingStatuses {
		existingByName[status.Name] = status
	}

	templateStatusIDs := make(map[string]uuid.UUID, len(template))
	for _, item := range template {
		if current, ok := existingByName[item.Name]; ok {
			builder := tx.TicketStatus.UpdateOneID(current.ID).
				SetStage(toEntStatusStage(item.Stage)).
				SetColor(item.Color).
				SetPosition(item.Position).
				SetIsDefault(item.IsDefault)

			if item.Icon == "" {
				builder.ClearIcon()
			} else {
				builder.SetIcon(item.Icon)
			}
			if item.MaxActiveRuns == nil {
				builder.ClearMaxActiveRuns()
			} else {
				builder.SetMaxActiveRuns(*item.MaxActiveRuns)
			}
			if item.Description == "" {
				builder.ClearDescription()
			} else {
				builder.SetDescription(item.Description)
			}

			updated, err := builder.Save(ctx)
			if err != nil {
				return nil, mapPersistenceError("reset existing ticket status", err)
			}
			templateStatusIDs[item.Name] = updated.ID
			continue
		}

		builder := tx.TicketStatus.Create().
			SetProjectID(projectID).
			SetName(item.Name).
			SetStage(toEntStatusStage(item.Stage)).
			SetColor(item.Color).
			SetPosition(item.Position).
			SetIsDefault(item.IsDefault)
		if item.Icon != "" {
			builder.SetIcon(item.Icon)
		}
		if item.MaxActiveRuns != nil {
			builder.SetMaxActiveRuns(*item.MaxActiveRuns)
		}
		if item.Description != "" {
			builder.SetDescription(item.Description)
		}

		created, err := builder.Save(ctx)
		if err != nil {
			return nil, mapPersistenceError("create default ticket status", err)
		}
		templateStatusIDs[item.Name] = created.ID
	}

	backlogID, ok := templateStatusIDs["Backlog"]
	if !ok {
		return nil, domain.ErrReplacementStatusAbsent
	}
	todoID, ok := templateStatusIDs["Todo"]
	if !ok {
		return nil, domain.ErrReplacementStatusAbsent
	}
	doneID, ok := templateStatusIDs["Done"]
	if !ok {
		return nil, domain.ErrReplacementStatusAbsent
	}

	templateNames := templateNameSet(template)
	for _, status := range existingStatuses {
		if templateNames[status.Name] {
			continue
		}

		if err := rebindStatusReferences(ctx, tx, status.ID, backlogID, todoID, doneID); err != nil {
			return nil, err
		}
		if err := tx.TicketStatus.DeleteOneID(status.ID).Exec(ctx); err != nil {
			return nil, fmt.Errorf("delete non-template ticket status %q: %w", status.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit ticket status reset tx: %w", err)
	}

	return r.List(ctx, projectID)
}

func (r *EntRepository) ListProjectStatusRuntimeSnapshots(ctx context.Context, projectID uuid.UUID) ([]domain.StatusRuntimeSnapshot, error) {
	statuses, err := r.client.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project ticket statuses: %w", err)
	}

	activeRunsByStatusID, err := countProjectStatusActiveRuns(ctx, r.client, projectID)
	if err != nil {
		return nil, err
	}

	return buildStatusRuntimeSnapshots(statuses, activeRunsByStatusID), nil
}

func (r *EntRepository) ListStatusRuntimeSnapshots(ctx context.Context) ([]domain.StatusRuntimeSnapshot, error) {
	statuses, err := r.client.TicketStatus.Query().
		Order(ent.Asc(entticketstatus.FieldProjectID), ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket statuses: %w", err)
	}

	activeRunsByStatusID, err := countStatusActiveRunsAcrossProjects(ctx, r.client)
	if err != nil {
		return nil, err
	}

	return buildStatusRuntimeSnapshots(statuses, activeRunsByStatusID), nil
}

type projectGetter interface {
	Query() *ent.ProjectQuery
}

func ensureProjectExists(ctx context.Context, client projectGetter, projectID uuid.UUID) error {
	exists, err := client.Query().Where(entproject.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return domain.ErrProjectNotFound
	}
	return nil
}

func rollback(tx *ent.Tx) {
	_ = tx.Rollback()
}

func clearProjectDefault(ctx context.Context, tx *ent.Tx, projectID uuid.UUID) error {
	if err := tx.TicketStatus.Update().
		Where(entticketstatus.ProjectIDEQ(projectID), entticketstatus.IsDefault(true)).
		SetIsDefault(false).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear project default ticket status: %w", err)
	}
	return nil
}

func selectReplacementStatus(statuses []*ent.TicketStatus, deleted *ent.TicketStatus) (*ent.TicketStatus, error) {
	others := make([]*ent.TicketStatus, 0, len(statuses)-1)
	for _, status := range statuses {
		if status.ID == deleted.ID {
			continue
		}
		others = append(others, status)
	}
	if len(others) == 0 {
		return nil, domain.ErrCannotDeleteLastStatus
	}
	for _, status := range others {
		if status.IsDefault {
			return status, nil
		}
	}
	return others[0], nil
}

func rebindStatusReferences(ctx context.Context, tx *ent.Tx, currentID uuid.UUID, statusReplacementID uuid.UUID, workflowReplacementIDs ...uuid.UUID) error {
	workflowPickupReplacement := statusReplacementID
	workflowFinishReplacement := statusReplacementID
	if len(workflowReplacementIDs) > 0 {
		workflowPickupReplacement = workflowReplacementIDs[0]
	}
	if len(workflowReplacementIDs) > 1 {
		workflowFinishReplacement = workflowReplacementIDs[1]
	} else if len(workflowReplacementIDs) == 1 {
		workflowFinishReplacement = workflowReplacementIDs[0]
	}

	if _, err := tx.Ticket.Update().
		Where(entticket.StatusIDEQ(currentID)).
		SetStatusID(statusReplacementID).
		Save(ctx); err != nil {
		return fmt.Errorf("move tickets off deleted ticket status: %w", err)
	}

	workflows, err := tx.Workflow.Query().
		Where(
			entworkflow.Or(
				entworkflow.HasPickupStatusesWith(entticketstatus.IDEQ(currentID)),
				entworkflow.HasFinishStatusesWith(entticketstatus.IDEQ(currentID)),
			),
		).
		WithPickupStatuses().
		WithFinishStatuses().
		All(ctx)
	if err != nil {
		return fmt.Errorf("load workflow status references: %w", err)
	}
	for _, workflow := range workflows {
		builder := tx.Workflow.UpdateOneID(workflow.ID)
		pickupIDs, pickupChanged := replaceWorkflowStatusBinding(
			workflow.Edges.PickupStatuses,
			currentID,
			workflowPickupReplacement,
		)
		if pickupChanged {
			builder.ClearPickupStatuses()
			builder.AddPickupStatusIDs(pickupIDs...)
		}
		finishIDs, finishChanged := replaceWorkflowStatusBinding(
			workflow.Edges.FinishStatuses,
			currentID,
			workflowFinishReplacement,
		)
		if finishChanged {
			builder.ClearFinishStatuses()
			builder.AddFinishStatusIDs(finishIDs...)
		}
		if pickupChanged || finishChanged {
			if _, err := builder.Save(ctx); err != nil {
				return fmt.Errorf("move workflow status references for workflow %s: %w", workflow.ID, err)
			}
		}
	}
	return nil
}

func replaceWorkflowStatusBinding(statuses []*ent.TicketStatus, currentID uuid.UUID, replacementID uuid.UUID) ([]uuid.UUID, bool) {
	ids := make([]uuid.UUID, 0, len(statuses))
	changed := false
	seen := make(map[uuid.UUID]struct{}, len(statuses))
	for _, status := range statuses {
		nextID := status.ID
		if status.ID == currentID {
			nextID = replacementID
			changed = true
		}
		if _, ok := seen[nextID]; ok {
			continue
		}
		seen[nextID] = struct{}{}
		ids = append(ids, nextID)
	}
	return ids, changed
}

func mapPersistenceError(action string, err error) error {
	if ent.IsConstraintError(err) {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "ticketstatus_project_id_name") {
			return domain.ErrDuplicateStatusName
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}

func mapNotFoundError(err error, replacement error) error {
	if ent.IsNotFound(err) {
		return replacement
	}
	return err
}

type ticketStatusQueryClient interface {
	Query() *ent.TicketStatusQuery
}

func ensureUniqueStatusName(ctx context.Context, client ticketStatusQueryClient, projectID uuid.UUID, name string, excludeStatusIDs ...uuid.UUID) error {
	normalizedTarget := strings.ToLower(strings.TrimSpace(name))
	if normalizedTarget == "" {
		return fmt.Errorf("status name must not be empty")
	}

	excluded := make(map[uuid.UUID]struct{}, len(excludeStatusIDs))
	for _, statusID := range excludeStatusIDs {
		excluded[statusID] = struct{}{}
	}

	statuses, err := client.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list ticket statuses for duplicate name check: %w", err)
	}
	for _, status := range statuses {
		if _, skip := excluded[status.ID]; skip {
			continue
		}
		if strings.ToLower(strings.TrimSpace(status.Name)) == normalizedTarget {
			return domain.ErrDuplicateStatusName
		}
	}
	return nil
}

func nextStatusPosition(statuses []*ent.TicketStatus) int {
	if len(statuses) == 0 {
		return 0
	}

	maxPosition := statuses[0].Position
	for _, status := range statuses[1:] {
		if status.Position > maxPosition {
			maxPosition = status.Position
		}
	}
	return maxPosition + 1
}

func hasDefault(statuses []*ent.TicketStatus) bool {
	for _, status := range statuses {
		if status.IsDefault {
			return true
		}
	}
	return false
}

func mapStatuses(statuses []*ent.TicketStatus, activeRunsByStatusID map[uuid.UUID]int) []domain.Status {
	out := make([]domain.Status, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, mapStatus(status, activeRunsByStatusID[status.ID]))
	}
	return out
}

func mapStatus(status *ent.TicketStatus, activeRuns int) domain.Status {
	return domain.Status{
		ID:            status.ID,
		ProjectID:     status.ProjectID,
		Name:          status.Name,
		Stage:         string(status.Stage),
		Color:         status.Color,
		Icon:          status.Icon,
		Position:      status.Position,
		ActiveRuns:    activeRuns,
		MaxActiveRuns: cloneIntPointer(status.MaxActiveRuns),
		IsDefault:     status.IsDefault,
		Description:   status.Description,
	}
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func buildStatusRuntimeSnapshots(statuses []*ent.TicketStatus, activeRunsByStatusID map[uuid.UUID]int) []domain.StatusRuntimeSnapshot {
	snapshots := make([]domain.StatusRuntimeSnapshot, 0, len(statuses))
	for _, status := range statuses {
		snapshots = append(snapshots, domain.StatusRuntimeSnapshot{
			StatusID:      status.ID,
			ProjectID:     status.ProjectID,
			Name:          status.Name,
			Stage:         string(status.Stage),
			Position:      status.Position,
			MaxActiveRuns: cloneIntPointer(status.MaxActiveRuns),
			ActiveRuns:    activeRunsByStatusID[status.ID],
		})
	}
	return snapshots
}

func countProjectStatusActiveRuns(ctx context.Context, client *ent.Client, projectID uuid.UUID) (map[uuid.UUID]int, error) {
	var counts []domain.StatusRuntimeSnapshot
	err := client.Ticket.Query().
		Where(entticket.ProjectIDEQ(projectID), entticket.CurrentRunIDNotNil()).
		GroupBy(entticket.FieldStatusID).
		Aggregate(ent.As(ent.Count(), "active_runs")).
		Scan(ctx, &counts)
	if err != nil {
		return nil, fmt.Errorf("group active project tickets by status occupancy: %w", err)
	}
	return runtimeCountMap(counts), nil
}

func countStatusActiveRunsAcrossProjects(ctx context.Context, client *ent.Client) (map[uuid.UUID]int, error) {
	var counts []domain.StatusRuntimeSnapshot
	err := client.Ticket.Query().
		Where(entticket.CurrentRunIDNotNil()).
		GroupBy(entticket.FieldStatusID).
		Aggregate(ent.As(ent.Count(), "active_runs")).
		Scan(ctx, &counts)
	if err != nil {
		return nil, fmt.Errorf("group active tickets by status occupancy: %w", err)
	}
	return runtimeCountMap(counts), nil
}

func countStatusActiveRuns(ctx context.Context, client *ent.Client, projectID uuid.UUID, statusID uuid.UUID) (int, error) {
	count, err := client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(projectID),
			entticket.CurrentRunIDNotNil(),
			entticket.StatusIDEQ(statusID),
		).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active runs in status %s: %w", statusID, err)
	}
	return count, nil
}

func runtimeCountMap(counts []domain.StatusRuntimeSnapshot) map[uuid.UUID]int {
	result := make(map[uuid.UUID]int, len(counts))
	for _, item := range counts {
		result[item.StatusID] = item.ActiveRuns
	}
	return result
}

func templateNameSet(template []domain.TemplateStatus) map[string]bool {
	names := make(map[string]bool, len(template))
	for _, item := range template {
		names[item.Name] = true
	}
	return names
}

func toEntStatusStage(stage ticketing.StatusStage) entticketstatus.Stage {
	return entticketstatus.Stage(stage.String())
}
