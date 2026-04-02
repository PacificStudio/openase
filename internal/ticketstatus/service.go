package ticketstatus

import (
	"context"
	"errors"
	"fmt"

	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	domain "github.com/BetterAndBetterII/openase/internal/domain/ticketstatus"
	"github.com/google/uuid"
)

var (
	// Ticket status service errors describe invalid or conflicting status operations.
	ErrUnavailable             = errors.New("ticket status service unavailable")
	ErrProjectNotFound         = domain.ErrProjectNotFound
	ErrStatusNotFound          = domain.ErrStatusNotFound
	ErrDuplicateStatusName     = domain.ErrDuplicateStatusName
	ErrDefaultStatusRequired   = domain.ErrDefaultStatusRequired
	ErrDefaultStatusStage      = domain.ErrDefaultStatusStage
	ErrCannotDeleteLastStatus  = domain.ErrCannotDeleteLastStatus
	ErrReplacementStatusAbsent = domain.ErrReplacementStatusAbsent
)

// Optional captures whether a value was provided for a partial update.
type Optional[T any] = domain.Optional[T]

// Some marks an optional value as explicitly set.
func Some[T any](value T) Optional[T] {
	return Optional[T]{Set: true, Value: value}
}

// Status is the API-facing ticket status model.
type Status = domain.Status

// StatusRuntimeSnapshot describes the live active-run occupancy for a ticket status.
type StatusRuntimeSnapshot = domain.StatusRuntimeSnapshot

// ListResult is the ordered ticket status board payload.
type ListResult = domain.ListResult

// CreateInput carries the fields required to create a ticket status.
type CreateInput = domain.CreateInput

// UpdateInput carries a partial ticket status update request.
type UpdateInput = domain.UpdateInput

// DeleteResult reports which status was deleted and which status replaced it.
type DeleteResult = domain.DeleteResult

type Repository interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.Status, error)
	ResolveStatusIDByName(ctx context.Context, projectID uuid.UUID, name string) (uuid.UUID, error)
	Get(ctx context.Context, statusID uuid.UUID) (domain.Status, error)
	Create(ctx context.Context, input domain.CreateInput) (domain.Status, error)
	Update(ctx context.Context, input domain.UpdateInput) (domain.Status, error)
	Delete(ctx context.Context, statusID uuid.UUID) (domain.DeleteResult, error)
	ResetToDefaultTemplate(ctx context.Context, projectID uuid.UUID, template []domain.TemplateStatus) ([]domain.Status, error)
	ListProjectStatusRuntimeSnapshots(ctx context.Context, projectID uuid.UUID) ([]domain.StatusRuntimeSnapshot, error)
	ListStatusRuntimeSnapshots(ctx context.Context) ([]domain.StatusRuntimeSnapshot, error)
}

// Service provides project ticket status management.
type Service struct {
	repo Repository
}

// NewService constructs a ticket status service backed by the provided repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns the ordered statuses for a project.
func (s *Service) List(ctx context.Context, projectID uuid.UUID) (ListResult, error) {
	if s == nil || s.repo == nil {
		return ListResult{}, ErrUnavailable
	}

	statuses, err := s.repo.List(ctx, projectID)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Statuses: statuses}, nil
}

// ResolveStatusIDByName finds a project status by display name.
func (s *Service) ResolveStatusIDByName(ctx context.Context, projectID uuid.UUID, name string) (uuid.UUID, error) {
	if s == nil || s.repo == nil {
		return uuid.UUID{}, ErrUnavailable
	}
	return s.repo.ResolveStatusIDByName(ctx, projectID, name)
}

func (s *Service) Get(ctx context.Context, statusID uuid.UUID) (Status, error) {
	if s == nil || s.repo == nil {
		return Status{}, ErrUnavailable
	}
	return s.repo.Get(ctx, statusID)
}

// Create persists a new ticket status in a project.
func (s *Service) Create(ctx context.Context, input CreateInput) (Status, error) {
	if s == nil || s.repo == nil {
		return Status{}, ErrUnavailable
	}

	stage, err := normalizeStatusStage(input.Stage)
	if err != nil {
		return Status{}, err
	}
	if input.IsDefault && stage.IsTerminal() {
		return Status{}, ErrDefaultStatusStage
	}
	input.Stage = stage
	return s.repo.Create(ctx, input)
}

// Update applies a partial update to an existing ticket status.
func (s *Service) Update(ctx context.Context, input UpdateInput) (Status, error) {
	if s == nil || s.repo == nil {
		return Status{}, ErrUnavailable
	}

	current, err := s.repo.Get(ctx, input.StatusID)
	if err != nil {
		return Status{}, err
	}

	nextStage, err := normalizeStatusStage(ticketing.StatusStage(current.Stage))
	if err != nil {
		return Status{}, err
	}
	if input.Stage.Set {
		nextStage, err = normalizeStatusStage(input.Stage.Value)
		if err != nil {
			return Status{}, err
		}
	}
	if current.IsDefault && nextStage.IsTerminal() {
		return Status{}, ErrDefaultStatusStage
	}
	if input.IsDefault.Set && input.IsDefault.Value && nextStage.IsTerminal() {
		return Status{}, ErrDefaultStatusStage
	}
	if input.Stage.Set {
		input.Stage = Some(nextStage)
	}

	return s.repo.Update(ctx, input)
}

// Delete removes a ticket status and reassigns affected tickets when required.
func (s *Service) Delete(ctx context.Context, statusID uuid.UUID) (DeleteResult, error) {
	if s == nil || s.repo == nil {
		return DeleteResult{}, ErrUnavailable
	}
	return s.repo.Delete(ctx, statusID)
}

// ResetToDefaultTemplate replaces project statuses with the built-in default template.
func (s *Service) ResetToDefaultTemplate(ctx context.Context, projectID uuid.UUID) ([]Status, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ResetToDefaultTemplate(ctx, projectID, defaultStatusTemplate)
}

// ListProjectStatusRuntimeSnapshots returns ordered runtime occupancy for all statuses in a project.
func ListProjectStatusRuntimeSnapshots(ctx context.Context, repo Repository, projectID uuid.UUID) ([]StatusRuntimeSnapshot, error) {
	if repo == nil {
		return nil, ErrUnavailable
	}
	return repo.ListProjectStatusRuntimeSnapshots(ctx, projectID)
}

// ListStatusRuntimeSnapshots returns ordered runtime occupancy for all statuses across projects.
func ListStatusRuntimeSnapshots(ctx context.Context, repo Repository) ([]StatusRuntimeSnapshot, error) {
	if repo == nil {
		return nil, ErrUnavailable
	}
	return repo.ListStatusRuntimeSnapshots(ctx)
}

var defaultStatusTemplate = []domain.TemplateStatus{
	{Name: "Backlog", Stage: ticketing.StatusStageBacklog, Color: "#6B7280", Icon: "archive", Position: 0, IsDefault: true, Description: "积压"},
	{Name: "Todo", Stage: ticketing.StatusStageUnstarted, Color: "#3B82F6", Icon: "list-todo", Position: 1, Description: "就绪等待"},
	{Name: "In Progress", Stage: ticketing.StatusStageStarted, Color: "#F59E0B", Icon: "play-circle", Position: 2, Description: "进行中"},
	{Name: "In Review", Stage: ticketing.StatusStageStarted, Color: "#8B5CF6", Icon: "search-check", Position: 3, Description: "审查中"},
	{Name: "Done", Stage: ticketing.StatusStageCompleted, Color: "#10B981", Icon: "check-circle-2", Position: 4, Description: "已完成"},
	{Name: "Cancelled", Stage: ticketing.StatusStageCanceled, Color: "#4B5563", Icon: "circle-slash", Position: 5, Description: "已取消"},
}

// DefaultTemplateNames returns the built-in default status names in display order.
func DefaultTemplateNames() []string {
	names := make([]string, 0, len(defaultStatusTemplate))
	for _, item := range defaultStatusTemplate {
		names = append(names, item.Name)
	}
	return names
}

func normalizeStatusStage(stage ticketing.StatusStage) (ticketing.StatusStage, error) {
	if stage == "" {
		return ticketing.DefaultStatusStage, nil
	}
	if !stage.IsValid() {
		return "", fmt.Errorf("status stage %q is invalid", stage)
	}
	return stage, nil
}
