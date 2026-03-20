package ticketstatus

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/google/uuid"
)

var (
	// Ticket status service errors describe invalid or conflicting status operations.
	ErrUnavailable             = errors.New("ticket status service unavailable")
	ErrProjectNotFound         = errors.New("project not found")
	ErrStatusNotFound          = errors.New("ticket status not found")
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

// Status is the API-facing ticket status model.
type Status struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	Position    int       `json:"position"`
	IsDefault   bool      `json:"is_default"`
	Description string    `json:"description"`
}

// CreateInput carries the fields required to create a ticket status.
type CreateInput struct {
	ProjectID   uuid.UUID
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

// Service provides project ticket status management.
type Service struct {
	client *ent.Client
}

// NewService constructs a ticket status service backed by the provided ent client.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// List returns the ordered statuses for a project.
func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]Status, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}
	if err := ensureProjectExists(ctx, s.client.Project, projectID); err != nil {
		return nil, err
	}

	statuses, err := s.client.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket statuses: %w", err)
	}

	return mapStatuses(statuses), nil
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

	projectStatuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(input.ProjectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("query project ticket statuses: %w", err)
	}

	position := input.Position.Value
	if !input.Position.Set {
		position = nextPosition(projectStatuses)
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

	return mapStatus(created), nil
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

	return mapStatus(updated), nil
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

	replacement, err := selectReplacementStatus(projectStatuses, current.ID)
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

// ResetToDefaultTemplate replaces project statuses with the built-in default template.
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

	existing, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query project ticket statuses: %w", err)
	}

	if err := clearProjectDefault(ctx, tx, projectID); err != nil {
		return nil, err
	}

	existingByName := make(map[string]*ent.TicketStatus, len(existing))
	for _, status := range existing {
		existingByName[status.Name] = status
	}

	templateIDs := make(map[string]uuid.UUID, len(defaultTemplate))
	for _, item := range defaultTemplate {
		if current, ok := existingByName[item.Name]; ok {
			builder := tx.TicketStatus.UpdateOneID(current.ID).
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
			templateIDs[item.Name] = updated.ID
			continue
		}

		builder := tx.TicketStatus.Create().
			SetProjectID(projectID).
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
		templateIDs[item.Name] = created.ID
	}

	backlogID, ok := templateIDs["Backlog"]
	if !ok {
		return nil, ErrReplacementStatusAbsent
	}
	todoID, ok := templateIDs["Todo"]
	if !ok {
		return nil, ErrReplacementStatusAbsent
	}
	doneID, ok := templateIDs["Done"]
	if !ok {
		return nil, ErrReplacementStatusAbsent
	}

	templateNames := templateNameSet()
	for _, status := range existing {
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

	statuses, err := tx.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list default ticket statuses: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit ticket status reset tx: %w", err)
	}

	return mapStatuses(statuses), nil
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
		return ErrProjectNotFound
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

func selectReplacementStatus(statuses []*ent.TicketStatus, deletedID uuid.UUID) (*ent.TicketStatus, error) {
	others := make([]*ent.TicketStatus, 0, len(statuses)-1)
	for _, status := range statuses {
		if status.ID == deletedID {
			continue
		}
		others = append(others, status)
	}
	if len(others) == 0 {
		return nil, ErrCannotDeleteLastStatus
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
	if _, err := tx.Workflow.Update().
		Where(entworkflow.PickupStatusIDEQ(currentID)).
		SetPickupStatusID(workflowPickupReplacement).
		Save(ctx); err != nil {
		return fmt.Errorf("move workflow pickup status references: %w", err)
	}
	if _, err := tx.Workflow.Update().
		Where(entworkflow.FinishStatusIDEQ(currentID)).
		SetFinishStatusID(workflowFinishReplacement).
		Save(ctx); err != nil {
		return fmt.Errorf("move workflow finish status references: %w", err)
	}
	return nil
}

func mapPersistenceError(action string, err error) error {
	if ent.IsConstraintError(err) && strings.Contains(strings.ToLower(err.Error()), "ticketstatus_project_id_name") {
		return ErrDuplicateStatusName
	}
	return fmt.Errorf("%s: %w", action, err)
}

func mapNotFoundError(err error, replacement error) error {
	if ent.IsNotFound(err) {
		return replacement
	}
	return err
}

func nextPosition(statuses []*ent.TicketStatus) int {
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

func mapStatuses(statuses []*ent.TicketStatus) []Status {
	out := make([]Status, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, mapStatus(status))
	}
	return out
}

func mapStatus(status *ent.TicketStatus) Status {
	return Status{
		ID:          status.ID,
		ProjectID:   status.ProjectID,
		Name:        status.Name,
		Color:       status.Color,
		Icon:        status.Icon,
		Position:    status.Position,
		IsDefault:   status.IsDefault,
		Description: status.Description,
	}
}

type templateStatus struct {
	Name        string
	Color       string
	Icon        string
	Position    int
	IsDefault   bool
	Description string
}

var defaultTemplate = []templateStatus{
	{Name: "Backlog", Color: "#6B7280", Icon: "archive", Position: 0, IsDefault: true, Description: "积压"},
	{Name: "Todo", Color: "#3B82F6", Icon: "list-todo", Position: 1, Description: "就绪等待"},
	{Name: "In Progress", Color: "#F59E0B", Icon: "play-circle", Position: 2, Description: "进行中"},
	{Name: "In Review", Color: "#8B5CF6", Icon: "search-check", Position: 3, Description: "审查中"},
	{Name: "Done", Color: "#10B981", Icon: "check-circle-2", Position: 4, Description: "已完成"},
	{Name: "Cancelled", Color: "#4B5563", Icon: "circle-slash", Position: 5, Description: "已取消"},
}

func templateNameSet() map[string]bool {
	names := make(map[string]bool, len(defaultTemplate))
	for _, item := range defaultTemplate {
		names[item.Name] = true
	}
	return names
}

// DefaultTemplateNames returns the built-in default status names in display order.
func DefaultTemplateNames() []string {
	names := make([]string, 0, len(defaultTemplate))
	for _, item := range defaultTemplate {
		names = append(names, item.Name)
	}
	slices.Sort(names)
	return names
}
