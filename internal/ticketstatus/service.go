package ticketstatus

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketstage "github.com/BetterAndBetterII/openase/ent/ticketstage"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/google/uuid"
)

var (
	// Ticket status service errors describe invalid or conflicting status operations.
	ErrUnavailable             = errors.New("ticket status service unavailable")
	ErrProjectNotFound         = errors.New("project not found")
	ErrStageNotFound           = errors.New("ticket stage not found")
	ErrStatusNotFound          = errors.New("ticket status not found")
	ErrDuplicateStageKey       = errors.New("ticket stage key already exists in project")
	ErrDuplicateStatusName     = errors.New("ticket status name already exists in project")
	ErrDefaultStatusRequired   = errors.New("at least one default ticket status is required")
	ErrCannotDeleteLastStatus  = errors.New("cannot delete the last ticket status in a project")
	ErrReplacementStatusAbsent = errors.New("replacement ticket status not found")
)

// Optional captures whether a value was provided for a partial update.
type Optional[T any] struct {
	Set   bool
	Value T
}

// Some marks an optional value as explicitly set.
func Some[T any](value T) Optional[T] {
	return Optional[T]{Set: true, Value: value}
}

// Stage is the API-facing ticket stage model.
type Stage struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	Position      int       `json:"position"`
	ActiveRuns    int       `json:"active_runs"`
	MaxActiveRuns *int      `json:"max_active_runs,omitempty"`
	Description   string    `json:"description"`
}

// StageRuntimeSnapshot describes the live active-run occupancy for a ticket stage.
type StageRuntimeSnapshot struct {
	StageID       uuid.UUID `json:"stage_id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	MaxActiveRuns *int      `json:"max_active_runs,omitempty"`
	ActiveRuns    int       `json:"active_runs"`
}

// Status is the API-facing ticket status model.
type Status struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	StageID     *uuid.UUID `json:"stage_id,omitempty"`
	Stage       *Stage     `json:"stage,omitempty"`
	Name        string     `json:"name"`
	Color       string     `json:"color"`
	Icon        string     `json:"icon"`
	Position    int        `json:"position"`
	IsDefault   bool       `json:"is_default"`
	Description string     `json:"description"`
}

// StatusGroup exposes board-ready status grouping by stage.
type StatusGroup struct {
	Stage    *Stage   `json:"stage,omitempty"`
	Statuses []Status `json:"statuses"`
}

// ListResult is the grouped ticket status board payload.
type ListResult struct {
	Stages      []Stage       `json:"stages"`
	Statuses    []Status      `json:"statuses"`
	StageGroups []StatusGroup `json:"stage_groups"`
}

// CreateStageInput carries the fields required to create a ticket stage.
type CreateStageInput struct {
	ProjectID     uuid.UUID
	Key           string
	Name          string
	Position      Optional[int]
	MaxActiveRuns *int
	Description   string
}

// UpdateStageInput carries a partial ticket stage update request.
type UpdateStageInput struct {
	StageID       uuid.UUID
	Name          Optional[string]
	Position      Optional[int]
	MaxActiveRuns Optional[*int]
	Description   Optional[string]
}

// DeleteStageResult reports which stage was deleted and how many statuses were detached.
type DeleteStageResult struct {
	DeletedStageID   uuid.UUID `json:"deleted_stage_id"`
	DetachedStatuses int       `json:"detached_statuses"`
}

// CreateInput carries the fields required to create a ticket status.
type CreateInput struct {
	ProjectID   uuid.UUID
	StageID     *uuid.UUID
	Name        string
	Color       string
	Icon        string
	Position    Optional[int]
	IsDefault   bool
	Description string
}

// UpdateInput carries a partial ticket status update request.
type UpdateInput struct {
	StatusID    uuid.UUID
	StageID     Optional[*uuid.UUID]
	Name        Optional[string]
	Color       Optional[string]
	Icon        Optional[string]
	Position    Optional[int]
	IsDefault   Optional[bool]
	Description Optional[string]
}

// DeleteResult reports which status was deleted and which status replaced it.
type DeleteResult struct {
	DeletedStatusID     uuid.UUID `json:"deleted_status_id"`
	ReplacementStatusID uuid.UUID `json:"replacement_status_id"`
}

// Service provides project ticket status and stage management.
type Service struct {
	client *ent.Client
}

// NewService constructs a ticket status service backed by the provided ent client.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// List returns the ordered statuses for a project grouped by stage.
func (s *Service) List(ctx context.Context, projectID uuid.UUID) (ListResult, error) {
	if s.client == nil {
		return ListResult{}, ErrUnavailable
	}
	if err := ensureProjectExists(ctx, s.client.Project, projectID); err != nil {
		return ListResult{}, err
	}

	stages, err := s.client.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("list ticket stages: %w", err)
	}
	statuses, err := s.client.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		WithStage().
		All(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("list ticket statuses: %w", err)
	}
	stageSnapshots, err := ListProjectStageRuntimeSnapshots(ctx, s.client, projectID)
	if err != nil {
		return ListResult{}, fmt.Errorf("list ticket stage runtime snapshots: %w", err)
	}

	return buildListResult(stages, statuses, stageSnapshots), nil
}

// ListStages returns the ordered stages for a project.
func (s *Service) ListStages(ctx context.Context, projectID uuid.UUID) ([]Stage, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}
	if err := ensureProjectExists(ctx, s.client.Project, projectID); err != nil {
		return nil, err
	}

	stages, err := s.client.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket stages: %w", err)
	}
	stageSnapshots, err := ListProjectStageRuntimeSnapshots(ctx, s.client, projectID)
	if err != nil {
		return nil, fmt.Errorf("list ticket stage runtime snapshots: %w", err)
	}

	return mapStages(stages, stageActiveRunsByID(stageSnapshots)), nil
}

// CreateStage persists a new ticket stage in a project.
func (s *Service) CreateStage(ctx context.Context, input CreateStageInput) (Stage, error) {
	if s.client == nil {
		return Stage{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Stage{}, fmt.Errorf("start ticket stage create tx: %w", err)
	}
	defer rollback(tx)

	if err := ensureProjectExists(ctx, tx.Project, input.ProjectID); err != nil {
		return Stage{}, err
	}

	projectStages, err := tx.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(input.ProjectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return Stage{}, fmt.Errorf("query project ticket stages: %w", err)
	}

	position := input.Position.Value
	if !input.Position.Set {
		position = nextStagePosition(projectStages)
	}

	builder := tx.TicketStage.Create().
		SetProjectID(input.ProjectID).
		SetKey(input.Key).
		SetName(input.Name).
		SetPosition(position)

	if input.MaxActiveRuns != nil {
		builder.SetMaxActiveRuns(*input.MaxActiveRuns)
	}
	if input.Description != "" {
		builder.SetDescription(input.Description)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return Stage{}, mapPersistenceError("create ticket stage", err)
	}
	if err := tx.Commit(); err != nil {
		return Stage{}, fmt.Errorf("commit ticket stage create tx: %w", err)
	}

	return mapStageWithActiveRuns(created, 0), nil
}

// UpdateStage applies a partial update to an existing ticket stage.
func (s *Service) UpdateStage(ctx context.Context, input UpdateStageInput) (Stage, error) {
	if s.client == nil {
		return Stage{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Stage{}, fmt.Errorf("start ticket stage update tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.TicketStage.Get(ctx, input.StageID)
	if err != nil {
		return Stage{}, mapNotFoundError(err, ErrStageNotFound)
	}

	builder := tx.TicketStage.UpdateOneID(current.ID)
	if input.Name.Set {
		builder.SetName(input.Name.Value)
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
	if input.Description.Set {
		if input.Description.Value == "" {
			builder.ClearDescription()
		} else {
			builder.SetDescription(input.Description.Value)
		}
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return Stage{}, mapPersistenceError("update ticket stage", err)
	}
	if err := tx.Commit(); err != nil {
		return Stage{}, fmt.Errorf("commit ticket stage update tx: %w", err)
	}

	activeRuns, err := countStageActiveRuns(ctx, s.client, updated.ProjectID, updated.ID)
	if err != nil {
		return Stage{}, fmt.Errorf("count ticket stage active runs: %w", err)
	}

	return mapStageWithActiveRuns(updated, activeRuns), nil
}

// DeleteStage removes a ticket stage and ungroups all attached statuses.
func (s *Service) DeleteStage(ctx context.Context, stageID uuid.UUID) (DeleteStageResult, error) {
	if s.client == nil {
		return DeleteStageResult{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return DeleteStageResult{}, fmt.Errorf("start ticket stage delete tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.TicketStage.Get(ctx, stageID); err != nil {
		return DeleteStageResult{}, mapNotFoundError(err, ErrStageNotFound)
	}

	detached, err := tx.TicketStatus.Update().
		Where(entticketstatus.StageIDEQ(stageID)).
		ClearStageID().
		Save(ctx)
	if err != nil {
		return DeleteStageResult{}, fmt.Errorf("clear ticket stage from statuses: %w", err)
	}
	if err := tx.TicketStage.DeleteOneID(stageID).Exec(ctx); err != nil {
		return DeleteStageResult{}, fmt.Errorf("delete ticket stage: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return DeleteStageResult{}, fmt.Errorf("commit ticket stage delete tx: %w", err)
	}

	return DeleteStageResult{
		DeletedStageID:   stageID,
		DetachedStatuses: detached,
	}, nil
}

// Create persists a new ticket status in a project.
func (s *Service) Create(ctx context.Context, input CreateInput) (Status, error) {
	if s.client == nil {
		return Status{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("start ticket status create tx: %w", err)
	}
	defer rollback(tx)

	if err := ensureProjectExists(ctx, tx.Project, input.ProjectID); err != nil {
		return Status{}, err
	}
	if err := ensureStageBelongsToProject(ctx, tx.TicketStage, input.ProjectID, input.StageID); err != nil {
		return Status{}, err
	}

	projectStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(input.ProjectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("query project ticket statuses: %w", err)
	}

	position := input.Position.Value
	if !input.Position.Set {
		position = nextStatusPosition(projectStatuses)
	}

	isDefault := input.IsDefault || !hasDefault(projectStatuses)
	if isDefault {
		if err := clearProjectDefault(ctx, tx, input.ProjectID); err != nil {
			return Status{}, err
		}
	}

	builder := tx.TicketStatus.Create().
		SetProjectID(input.ProjectID).
		SetName(input.Name).
		SetColor(input.Color).
		SetPosition(position).
		SetIsDefault(isDefault)

	if input.StageID != nil {
		builder.SetStageID(*input.StageID)
	}
	if input.Icon != "" {
		builder.SetIcon(input.Icon)
	}
	if input.Description != "" {
		builder.SetDescription(input.Description)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return Status{}, mapPersistenceError("create ticket status", err)
	}
	if err := tx.Commit(); err != nil {
		return Status{}, fmt.Errorf("commit ticket status create tx: %w", err)
	}

	saved, err := s.client.TicketStatus.Query().Where(entticketstatus.ID(created.ID)).WithStage().Only(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("load created ticket status: %w", err)
	}
	activeRunsByStageID := map[uuid.UUID]int{}
	if saved.StageID != nil {
		activeRuns, err := countStageActiveRuns(ctx, s.client, saved.ProjectID, *saved.StageID)
		if err != nil {
			return Status{}, fmt.Errorf("count created ticket status stage active runs: %w", err)
		}
		activeRunsByStageID[*saved.StageID] = activeRuns
	}
	return mapStatus(saved, activeRunsByStageID), nil
}

// Update applies a partial update to an existing ticket status.
func (s *Service) Update(ctx context.Context, input UpdateInput) (Status, error) {
	if s.client == nil {
		return Status{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("start ticket status update tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.TicketStatus.Get(ctx, input.StatusID)
	if err != nil {
		return Status{}, mapNotFoundError(err, ErrStatusNotFound)
	}

	if input.StageID.Set {
		if err := ensureStageBelongsToProject(ctx, tx.TicketStage, current.ProjectID, input.StageID.Value); err != nil {
			return Status{}, err
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
			return Status{}, fmt.Errorf("check remaining default ticket status: %w", err)
		}
		if !otherDefault {
			return Status{}, ErrDefaultStatusRequired
		}
	}

	if input.IsDefault.Set && input.IsDefault.Value {
		if err := clearProjectDefault(ctx, tx, current.ProjectID); err != nil {
			return Status{}, err
		}
	}

	builder := tx.TicketStatus.UpdateOneID(current.ID)
	if input.StageID.Set {
		if input.StageID.Value == nil {
			builder.ClearStageID()
		} else {
			builder.SetStageID(*input.StageID.Value)
		}
	}
	if input.Name.Set {
		builder.SetName(input.Name.Value)
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
		return Status{}, mapPersistenceError("update ticket status", err)
	}
	if err := tx.Commit(); err != nil {
		return Status{}, fmt.Errorf("commit ticket status update tx: %w", err)
	}

	saved, err := s.client.TicketStatus.Query().Where(entticketstatus.ID(updated.ID)).WithStage().Only(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("load updated ticket status: %w", err)
	}
	activeRunsByStageID := map[uuid.UUID]int{}
	if saved.StageID != nil {
		activeRuns, err := countStageActiveRuns(ctx, s.client, saved.ProjectID, *saved.StageID)
		if err != nil {
			return Status{}, fmt.Errorf("count updated ticket status stage active runs: %w", err)
		}
		activeRunsByStageID[*saved.StageID] = activeRuns
	}
	return mapStatus(saved, activeRunsByStageID), nil
}

// Delete removes a ticket status and reassigns affected tickets when required.
func (s *Service) Delete(ctx context.Context, statusID uuid.UUID) (DeleteResult, error) {
	if s.client == nil {
		return DeleteResult{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return DeleteResult{}, fmt.Errorf("start ticket status delete tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.TicketStatus.Get(ctx, statusID)
	if err != nil {
		return DeleteResult{}, mapNotFoundError(err, ErrStatusNotFound)
	}

	projectStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(current.ProjectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return DeleteResult{}, fmt.Errorf("query project ticket statuses: %w", err)
	}

	replacement, err := selectReplacementStatus(projectStatuses, current)
	if err != nil {
		return DeleteResult{}, err
	}

	if err := rebindStatusReferences(ctx, tx, current.ID, replacement.ID, replacement.ID); err != nil {
		return DeleteResult{}, err
	}

	if current.IsDefault && !replacement.IsDefault {
		if err := clearProjectDefault(ctx, tx, current.ProjectID); err != nil {
			return DeleteResult{}, err
		}
		if _, err := tx.TicketStatus.UpdateOneID(replacement.ID).SetIsDefault(true).Save(ctx); err != nil {
			return DeleteResult{}, mapPersistenceError("promote replacement default ticket status", err)
		}
	}

	if err := tx.TicketStatus.DeleteOneID(current.ID).Exec(ctx); err != nil {
		return DeleteResult{}, fmt.Errorf("delete ticket status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return DeleteResult{}, fmt.Errorf("commit ticket status delete tx: %w", err)
	}

	return DeleteResult{
		DeletedStatusID:     current.ID,
		ReplacementStatusID: replacement.ID,
	}, nil
}

// ResetToDefaultTemplate replaces project stages and statuses with the built-in default template.
func (s *Service) ResetToDefaultTemplate(ctx context.Context, projectID uuid.UUID) ([]Status, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("start ticket status reset tx: %w", err)
	}
	defer rollback(tx)

	if err := ensureProjectExists(ctx, tx.Project, projectID); err != nil {
		return nil, err
	}

	existingStages, err := tx.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query project ticket stages: %w", err)
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

	templateStageIDs, err := upsertDefaultStages(ctx, tx, projectID, existingStages)
	if err != nil {
		return nil, err
	}

	existingByName := make(map[string]*ent.TicketStatus, len(existingStatuses))
	for _, status := range existingStatuses {
		existingByName[status.Name] = status
	}

	templateStatusIDs := make(map[string]uuid.UUID, len(defaultStatusTemplate))
	for _, item := range defaultStatusTemplate {
		stageID, ok := templateStageIDs[item.StageKey]
		if !ok {
			return nil, ErrReplacementStatusAbsent
		}

		if current, ok := existingByName[item.Name]; ok {
			builder := tx.TicketStatus.UpdateOneID(current.ID).
				SetStageID(stageID).
				SetColor(item.Color).
				SetPosition(item.Position).
				SetIsDefault(item.IsDefault)

			if item.Icon == "" {
				builder.ClearIcon()
			} else {
				builder.SetIcon(item.Icon)
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
			SetStageID(stageID).
			SetName(item.Name).
			SetColor(item.Color).
			SetPosition(item.Position).
			SetIsDefault(item.IsDefault)
		if item.Icon != "" {
			builder.SetIcon(item.Icon)
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
		return nil, ErrReplacementStatusAbsent
	}
	todoID, ok := templateStatusIDs["Todo"]
	if !ok {
		return nil, ErrReplacementStatusAbsent
	}
	doneID, ok := templateStatusIDs["Done"]
	if !ok {
		return nil, ErrReplacementStatusAbsent
	}

	templateNames := templateNameSet()
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

	templateStageKeys := templateStageKeySet()
	for _, stage := range existingStages {
		if templateStageKeys[stage.Key] {
			continue
		}
		if _, err := tx.TicketStatus.Update().
			Where(entticketstatus.StageIDEQ(stage.ID)).
			ClearStageID().
			Save(ctx); err != nil {
			return nil, fmt.Errorf("clear custom ticket stage status references: %w", err)
		}
		if err := tx.TicketStage.DeleteOneID(stage.ID).Exec(ctx); err != nil {
			return nil, fmt.Errorf("delete non-template ticket stage %q: %w", stage.Key, err)
		}
	}

	finalStages, err := tx.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list default ticket stages: %w", err)
	}
	finalStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		WithStage().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list default ticket statuses: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit ticket status reset tx: %w", err)
	}

	return buildListResult(finalStages, finalStatuses, nil).Statuses, nil
}

// BackfillDefaultStages creates the PRD stage template for legacy projects that still only have default statuses.
func (s *Service) BackfillDefaultStages(ctx context.Context) error {
	if s.client == nil {
		return ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start ticket stage backfill tx: %w", err)
	}
	defer rollback(tx)

	projects, err := tx.Project.Query().
		Where(entproject.HasStatuses()).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list projects for ticket stage backfill: %w", err)
	}
	for _, project := range projects {
		if err := backfillProjectStages(ctx, tx, project.ID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ticket stage backfill tx: %w", err)
	}
	return nil
}

type projectGetter interface {
	Query() *ent.ProjectQuery
}

type stageGetter interface {
	Query() *ent.TicketStageQuery
}

func ensureProjectExists(ctx context.Context, client projectGetter, projectID uuid.UUID) error {
	exists, err := client.Query().Where(entproject.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}
	return nil
}

func ensureStageBelongsToProject(ctx context.Context, client stageGetter, projectID uuid.UUID, stageID *uuid.UUID) error {
	if stageID == nil {
		return nil
	}
	exists, err := client.Query().
		Where(entticketstage.ID(*stageID), entticketstage.ProjectIDEQ(projectID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check ticket stage existence: %w", err)
	}
	if !exists {
		return ErrStageNotFound
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
		return nil, ErrCannotDeleteLastStatus
	}
	for _, status := range others {
		if status.IsDefault && sameStage(status.StageID, deleted.StageID) {
			return status, nil
		}
	}
	for _, status := range others {
		if status.IsDefault {
			return status, nil
		}
	}
	for _, status := range others {
		if sameStage(status.StageID, deleted.StageID) {
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

func upsertDefaultStages(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, existing []*ent.TicketStage) (map[string]uuid.UUID, error) {
	existingByKey := make(map[string]*ent.TicketStage, len(existing))
	for _, stage := range existing {
		existingByKey[stage.Key] = stage
	}

	templateStageIDs := make(map[string]uuid.UUID, len(defaultStageTemplate))
	for _, item := range defaultStageTemplate {
		if current, ok := existingByKey[item.Key]; ok {
			builder := tx.TicketStage.UpdateOneID(current.ID).
				SetName(item.Name).
				SetPosition(item.Position)
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
				return nil, mapPersistenceError("reset existing ticket stage", err)
			}
			templateStageIDs[item.Key] = updated.ID
			continue
		}

		builder := tx.TicketStage.Create().
			SetProjectID(projectID).
			SetKey(item.Key).
			SetName(item.Name).
			SetPosition(item.Position)
		if item.MaxActiveRuns != nil {
			builder.SetMaxActiveRuns(*item.MaxActiveRuns)
		}
		if item.Description != "" {
			builder.SetDescription(item.Description)
		}

		created, err := builder.Save(ctx)
		if err != nil {
			return nil, mapPersistenceError("create default ticket stage", err)
		}
		templateStageIDs[item.Key] = created.ID
	}

	return templateStageIDs, nil
}

func backfillProjectStages(ctx context.Context, tx *ent.Tx, projectID uuid.UUID) error {
	existingStages, err := tx.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query ticket stages for backfill: %w", err)
	}
	existingStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query ticket statuses for backfill: %w", err)
	}
	if len(existingStatuses) == 0 || (!hasTemplateStatus(existingStatuses) && len(existingStages) == 0) {
		return nil
	}

	stageIDs, err := upsertDefaultStages(ctx, tx, projectID, existingStages)
	if err != nil {
		return err
	}
	statusesByName := make(map[string]*ent.TicketStatus, len(existingStatuses))
	for _, status := range existingStatuses {
		statusesByName[status.Name] = status
	}
	for _, item := range defaultStatusTemplate {
		status, ok := statusesByName[item.Name]
		if !ok || status.StageID != nil {
			continue
		}
		stageID, ok := stageIDs[item.StageKey]
		if !ok {
			return ErrStageNotFound
		}
		if _, err := tx.TicketStatus.UpdateOneID(status.ID).SetStageID(stageID).Save(ctx); err != nil {
			return fmt.Errorf("backfill ticket status stage for %q: %w", status.Name, err)
		}
	}

	return nil
}

func mapPersistenceError(action string, err error) error {
	if ent.IsConstraintError(err) {
		lower := strings.ToLower(err.Error())
		switch {
		case strings.Contains(lower, "ticketstage_project_id_key"):
			return ErrDuplicateStageKey
		case strings.Contains(lower, "ticketstatus_project_id_name"):
			return ErrDuplicateStatusName
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

func nextStagePosition(stages []*ent.TicketStage) int {
	if len(stages) == 0 {
		return 0
	}

	maxPosition := stages[0].Position
	for _, stage := range stages[1:] {
		if stage.Position > maxPosition {
			maxPosition = stage.Position
		}
	}
	return maxPosition + 1
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

func hasTemplateStatus(statuses []*ent.TicketStatus) bool {
	templateNames := templateNameSet()
	for _, status := range statuses {
		if templateNames[status.Name] {
			return true
		}
	}
	return false
}

func buildListResult(stages []*ent.TicketStage, statuses []*ent.TicketStatus, stageSnapshots []StageRuntimeSnapshot) ListResult {
	activeRunsByStageID := stageActiveRunsByID(stageSnapshots)
	stageModels := mapStages(stages, activeRunsByStageID)
	statusModels := mapStatuses(statuses, activeRunsByStageID)
	sortStatusesForBoard(stageModels, statusModels)

	return ListResult{
		Stages:      stageModels,
		Statuses:    statusModels,
		StageGroups: buildStatusGroups(stageModels, statusModels),
	}
}

// ListProjectStageRuntimeSnapshots returns ordered runtime occupancy for all stages in a project.
func ListProjectStageRuntimeSnapshots(ctx context.Context, client *ent.Client, projectID uuid.UUID) ([]StageRuntimeSnapshot, error) {
	if client == nil {
		return nil, ErrUnavailable
	}

	stages, err := client.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project ticket stages: %w", err)
	}

	activeRunsByStageID, err := countProjectStageActiveRuns(ctx, client, projectID)
	if err != nil {
		return nil, err
	}

	return buildStageRuntimeSnapshots(stages, activeRunsByStageID), nil
}

// ListStageRuntimeSnapshots returns ordered runtime occupancy for all stages across projects.
func ListStageRuntimeSnapshots(ctx context.Context, client *ent.Client) ([]StageRuntimeSnapshot, error) {
	if client == nil {
		return nil, ErrUnavailable
	}

	stages, err := client.TicketStage.Query().
		Order(ent.Asc(entticketstage.FieldProjectID), ent.Asc(entticketstage.FieldPosition), ent.Asc(entticketstage.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket stages: %w", err)
	}

	activeRunsByStageID, err := countStageActiveRunsAcrossProjects(ctx, client)
	if err != nil {
		return nil, err
	}

	return buildStageRuntimeSnapshots(stages, activeRunsByStageID), nil
}

func sortStatusesForBoard(stages []Stage, statuses []Status) {
	stageOrder := make(map[uuid.UUID]int, len(stages))
	for idx, stage := range stages {
		stageOrder[stage.ID] = idx
	}

	sort.SliceStable(statuses, func(i, j int) bool {
		leftOrder := len(stages)
		if statuses[i].StageID != nil {
			if idx, ok := stageOrder[*statuses[i].StageID]; ok {
				leftOrder = idx
			}
		}
		rightOrder := len(stages)
		if statuses[j].StageID != nil {
			if idx, ok := stageOrder[*statuses[j].StageID]; ok {
				rightOrder = idx
			}
		}
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		if statuses[i].Position != statuses[j].Position {
			return statuses[i].Position < statuses[j].Position
		}
		return statuses[i].Name < statuses[j].Name
	})
}

func buildStatusGroups(stages []Stage, statuses []Status) []StatusGroup {
	stageSet := make(map[uuid.UUID]struct{}, len(stages))
	for _, stage := range stages {
		stageSet[stage.ID] = struct{}{}
	}

	grouped := make(map[uuid.UUID][]Status, len(stages))
	ungrouped := make([]Status, 0)
	for _, status := range statuses {
		if status.StageID != nil {
			if _, ok := stageSet[*status.StageID]; ok {
				grouped[*status.StageID] = append(grouped[*status.StageID], status)
				continue
			}
		}
		ungrouped = append(ungrouped, status)
	}

	groups := make([]StatusGroup, 0, len(stages)+1)
	for _, stage := range stages {
		stageCopy := stage
		stageStatuses := grouped[stage.ID]
		if stageStatuses == nil {
			stageStatuses = []Status{}
		}
		groups = append(groups, StatusGroup{
			Stage:    &stageCopy,
			Statuses: stageStatuses,
		})
	}
	if len(ungrouped) > 0 {
		groups = append(groups, StatusGroup{
			Statuses: ungrouped,
		})
	}
	return groups
}

func mapStages(stages []*ent.TicketStage, activeRunsByStageID map[uuid.UUID]int) []Stage {
	out := make([]Stage, 0, len(stages))
	for _, stage := range stages {
		out = append(out, mapStageWithActiveRuns(stage, activeRunsByStageID[stage.ID]))
	}
	return out
}

func mapStage(stage *ent.TicketStage) Stage {
	return mapStageWithActiveRuns(stage, 0)
}

func mapStageWithActiveRuns(stage *ent.TicketStage, activeRuns int) Stage {
	return Stage{
		ID:            stage.ID,
		ProjectID:     stage.ProjectID,
		Key:           stage.Key,
		Name:          stage.Name,
		Position:      stage.Position,
		ActiveRuns:    activeRuns,
		MaxActiveRuns: cloneIntPointer(stage.MaxActiveRuns),
		Description:   stage.Description,
	}
}

func mapStatuses(statuses []*ent.TicketStatus, activeRunsByStageID map[uuid.UUID]int) []Status {
	out := make([]Status, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, mapStatus(status, activeRunsByStageID))
	}
	return out
}

func mapStatus(status *ent.TicketStatus, activeRunsByStageID map[uuid.UUID]int) Status {
	var stageID *uuid.UUID
	if status.StageID != nil {
		stageID = cloneUUIDPointer(status.StageID)
	}

	var stage *Stage
	if status.Edges.Stage != nil {
		stageValue := mapStageWithActiveRuns(status.Edges.Stage, activeRunsByStageID[status.Edges.Stage.ID])
		stage = &stageValue
	}

	return Status{
		ID:          status.ID,
		ProjectID:   status.ProjectID,
		StageID:     stageID,
		Stage:       stage,
		Name:        status.Name,
		Color:       status.Color,
		Icon:        status.Icon,
		Position:    status.Position,
		IsDefault:   status.IsDefault,
		Description: status.Description,
	}
}

func cloneUUIDPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func buildStageRuntimeSnapshots(stages []*ent.TicketStage, activeRunsByStageID map[uuid.UUID]int) []StageRuntimeSnapshot {
	snapshots := make([]StageRuntimeSnapshot, 0, len(stages))
	for _, stage := range stages {
		snapshots = append(snapshots, StageRuntimeSnapshot{
			StageID:       stage.ID,
			ProjectID:     stage.ProjectID,
			Key:           stage.Key,
			Name:          stage.Name,
			MaxActiveRuns: cloneIntPointer(stage.MaxActiveRuns),
			ActiveRuns:    activeRunsByStageID[stage.ID],
		})
	}
	return snapshots
}

func stageActiveRunsByID(snapshots []StageRuntimeSnapshot) map[uuid.UUID]int {
	counts := make(map[uuid.UUID]int, len(snapshots))
	for _, snapshot := range snapshots {
		counts[snapshot.StageID] = snapshot.ActiveRuns
	}
	return counts
}

func countProjectStageActiveRuns(ctx context.Context, client *ent.Client, projectID uuid.UUID) (map[uuid.UUID]int, error) {
	var statusCounts []stageStatusActiveRunCount
	err := client.Ticket.Query().
		Where(entticket.ProjectIDEQ(projectID), entticket.CurrentRunIDNotNil()).
		GroupBy(entticket.FieldStatusID).
		Aggregate(ent.As(ent.Count(), "active_runs")).
		Scan(ctx, &statusCounts)
	if err != nil {
		return nil, fmt.Errorf("group active project tickets by status for stage occupancy: %w", err)
	}
	return countStageActiveRunsFromStatusCounts(ctx, client, statusCounts)
}

func countStageActiveRunsAcrossProjects(ctx context.Context, client *ent.Client) (map[uuid.UUID]int, error) {
	var statusCounts []stageStatusActiveRunCount
	err := client.Ticket.Query().
		Where(entticket.CurrentRunIDNotNil()).
		GroupBy(entticket.FieldStatusID).
		Aggregate(ent.As(ent.Count(), "active_runs")).
		Scan(ctx, &statusCounts)
	if err != nil {
		return nil, fmt.Errorf("group active tickets by status for stage occupancy: %w", err)
	}
	return countStageActiveRunsFromStatusCounts(ctx, client, statusCounts)
}

func countStageActiveRuns(ctx context.Context, client *ent.Client, projectID uuid.UUID, stageID uuid.UUID) (int, error) {
	count, err := client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(projectID),
			entticket.CurrentRunIDNotNil(),
			entticket.HasStatusWith(entticketstatus.HasStageWith(entticketstage.IDEQ(stageID))),
		).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active runs in stage %s: %w", stageID, err)
	}
	return count, nil
}

func sameStage(left *uuid.UUID, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

type stageStatusActiveRunCount struct {
	StatusID   uuid.UUID `json:"status_id"`
	ActiveRuns int       `json:"active_runs"`
}

func countStageActiveRunsFromStatusCounts(ctx context.Context, client *ent.Client, statusCounts []stageStatusActiveRunCount) (map[uuid.UUID]int, error) {
	if len(statusCounts) == 0 {
		return map[uuid.UUID]int{}, nil
	}

	statusIDs := make([]uuid.UUID, 0, len(statusCounts))
	for _, statusCount := range statusCounts {
		statusIDs = append(statusIDs, statusCount.StatusID)
	}

	statuses, err := client.TicketStatus.Query().
		Where(entticketstatus.IDIn(statusIDs...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list statuses for stage occupancy: %w", err)
	}

	stageIDsByStatusID := make(map[uuid.UUID]uuid.UUID, len(statuses))
	for _, status := range statuses {
		if status.StageID == nil {
			continue
		}
		stageIDsByStatusID[status.ID] = *status.StageID
	}

	counts := make(map[uuid.UUID]int)
	for _, statusCount := range statusCounts {
		stageID, ok := stageIDsByStatusID[statusCount.StatusID]
		if !ok {
			continue
		}
		counts[stageID] += statusCount.ActiveRuns
	}
	return counts, nil
}

type templateStage struct {
	Key           string
	Name          string
	Position      int
	MaxActiveRuns *int
	Description   string
}

type templateStatus struct {
	StageKey    string
	Name        string
	Color       string
	Icon        string
	Position    int
	IsDefault   bool
	Description string
}

var defaultStageTemplate = []templateStage{
	{Key: "backlog", Name: "Backlog", Position: 0, Description: "积压阶段"},
	{Key: "in_progress", Name: "In Progress", Position: 1, Description: "进行中阶段"},
	{Key: "review", Name: "Review", Position: 2, Description: "审查阶段"},
	{Key: "done", Name: "Done", Position: 3, Description: "收尾阶段"},
}

var defaultStatusTemplate = []templateStatus{
	{StageKey: "backlog", Name: "Backlog", Color: "#6B7280", Icon: "archive", Position: 0, IsDefault: true, Description: "积压"},
	{StageKey: "backlog", Name: "Todo", Color: "#3B82F6", Icon: "list-todo", Position: 1, Description: "就绪等待"},
	{StageKey: "in_progress", Name: "In Progress", Color: "#F59E0B", Icon: "play-circle", Position: 2, Description: "进行中"},
	{StageKey: "review", Name: "In Review", Color: "#8B5CF6", Icon: "search-check", Position: 3, Description: "审查中"},
	{StageKey: "done", Name: "Done", Color: "#10B981", Icon: "check-circle-2", Position: 4, Description: "已完成"},
	{StageKey: "done", Name: "Cancelled", Color: "#4B5563", Icon: "circle-slash", Position: 5, Description: "已取消"},
}

func templateStageKeySet() map[string]bool {
	keys := make(map[string]bool, len(defaultStageTemplate))
	for _, item := range defaultStageTemplate {
		keys[item.Key] = true
	}
	return keys
}

func templateNameSet() map[string]bool {
	names := make(map[string]bool, len(defaultStatusTemplate))
	for _, item := range defaultStatusTemplate {
		names[item.Name] = true
	}
	return names
}

// DefaultTemplateNames returns the built-in default status names in display order.
func DefaultTemplateNames() []string {
	names := make([]string, 0, len(defaultStatusTemplate))
	for _, item := range defaultStatusTemplate {
		names = append(names, item.Name)
	}
	return names
}
