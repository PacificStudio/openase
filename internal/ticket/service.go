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
	entticketcomment "github.com/BetterAndBetterII/openase/ent/ticketcomment"
	entticketcommentrevision "github.com/BetterAndBetterII/openase/ent/ticketcommentrevision"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

const (
	defaultCreatedBy        = "user:api"
	defaultIdentifierPrefix = "ASE"
)

var (
	// Ticket service errors describe invalid or missing ticket resources.
	ErrUnavailable           = errors.New("ticket service unavailable")
	ErrProjectNotFound       = errors.New("project not found")
	ErrTicketNotFound        = errors.New("ticket not found")
	ErrTicketConflict        = errors.New("ticket identifier already exists in project")
	ErrStatusNotFound        = errors.New("ticket status not found")
	ErrWorkflowNotFound      = errors.New("workflow not found")
	ErrStatusNotAllowed      = errors.New("ticket status is not allowed by the workflow finish set")
	ErrParentTicketNotFound  = errors.New("parent ticket not found")
	ErrTargetMachineNotFound = errors.New("target machine not found in project organization")
	ErrDependencyNotFound    = errors.New("ticket dependency not found")
	ErrDependencyConflict    = errors.New("ticket dependency already exists")
	ErrCommentNotFound       = errors.New("ticket comment not found")
	ErrExternalLinkNotFound  = errors.New("ticket external link not found")
	ErrExternalLinkConflict  = errors.New("ticket external link already exists")
	ErrInvalidDependency     = errors.New("invalid ticket dependency")
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

// TicketReference is a compact ticket summary used inside relationships.
type TicketReference struct {
	ID         uuid.UUID `json:"id"`
	Identifier string    `json:"identifier"`
	Title      string    `json:"title"`
	StatusID   uuid.UUID `json:"status_id"`
	StatusName string    `json:"status_name"`
}

// Dependency describes a typed dependency edge to another ticket.
type Dependency struct {
	ID     uuid.UUID                `json:"id"`
	Type   entticketdependency.Type `json:"type"`
	Target TicketReference          `json:"target"`
}

// ExternalLink describes a ticket association to an external issue or PR.
type ExternalLink struct {
	ID         uuid.UUID                      `json:"id"`
	LinkType   entticketexternallink.LinkType `json:"link_type"`
	URL        string                         `json:"url"`
	ExternalID string                         `json:"external_id"`
	Title      string                         `json:"title,omitempty"`
	Status     string                         `json:"status,omitempty"`
	Relation   entticketexternallink.Relation `json:"relation"`
	CreatedAt  time.Time                      `json:"created_at"`
}

// Comment describes a first-class user discussion item on a ticket.
type Comment struct {
	ID           uuid.UUID  `json:"id"`
	TicketID     uuid.UUID  `json:"ticket_id"`
	BodyMarkdown string     `json:"body_markdown"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	EditCount    int        `json:"edit_count"`
	LastEditedBy *string    `json:"last_edited_by,omitempty"`
	IsDeleted    bool       `json:"is_deleted"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	DeletedBy    *string    `json:"deleted_by,omitempty"`
}

// CommentRevision describes an immutable snapshot in a comment's edit history.
type CommentRevision struct {
	ID             uuid.UUID `json:"id"`
	CommentID      uuid.UUID `json:"comment_id"`
	RevisionNumber int       `json:"revision_number"`
	BodyMarkdown   string    `json:"body_markdown"`
	EditedBy       string    `json:"edited_by"`
	EditedAt       time.Time `json:"edited_at"`
	EditReason     *string   `json:"edit_reason,omitempty"`
}

// Ticket is the API-facing ticket aggregate returned by the service layer.
type Ticket struct {
	ID                uuid.UUID          `json:"id"`
	ProjectID         uuid.UUID          `json:"project_id"`
	Identifier        string             `json:"identifier"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	StatusID          uuid.UUID          `json:"status_id"`
	StatusName        string             `json:"status_name"`
	Priority          entticket.Priority `json:"priority"`
	Type              entticket.Type     `json:"type"`
	WorkflowID        *uuid.UUID         `json:"workflow_id,omitempty"`
	CurrentRunID      *uuid.UUID         `json:"current_run_id,omitempty"`
	TargetMachineID   *uuid.UUID         `json:"target_machine_id,omitempty"`
	CreatedBy         string             `json:"created_by"`
	Parent            *TicketReference   `json:"parent,omitempty"`
	Children          []TicketReference  `json:"children"`
	Dependencies      []Dependency       `json:"dependencies"`
	ExternalLinks     []ExternalLink     `json:"external_links"`
	ExternalRef       string             `json:"external_ref"`
	BudgetUSD         float64            `json:"budget_usd"`
	CostTokensInput   int64              `json:"cost_tokens_input"`
	CostTokensOutput  int64              `json:"cost_tokens_output"`
	CostAmount        float64            `json:"cost_amount"`
	AttemptCount      int                `json:"attempt_count"`
	ConsecutiveErrors int                `json:"consecutive_errors"`
	StartedAt         *time.Time         `json:"started_at,omitempty"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
	NextRetryAt       *time.Time         `json:"next_retry_at,omitempty"`
	RetryPaused       bool               `json:"retry_paused"`
	PauseReason       string             `json:"pause_reason,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
}

// ListInput filters ticket queries within a project.
type ListInput struct {
	ProjectID   uuid.UUID
	StatusNames []string
	Priorities  []entticket.Priority
	Limit       int
}

// CreateInput carries the fields required to create a ticket.
type CreateInput struct {
	ProjectID       uuid.UUID
	Title           string
	Description     string
	StatusID        *uuid.UUID
	Priority        entticket.Priority
	Type            entticket.Type
	WorkflowID      *uuid.UUID
	TargetMachineID *uuid.UUID
	CreatedBy       string
	ParentTicketID  *uuid.UUID
	ExternalRef     string
	BudgetUSD       float64
}

// UpdateInput carries a partial ticket update request.
type UpdateInput struct {
	TicketID                          uuid.UUID
	Title                             Optional[string]
	Description                       Optional[string]
	StatusID                          Optional[uuid.UUID]
	Priority                          Optional[entticket.Priority]
	Type                              Optional[entticket.Type]
	WorkflowID                        Optional[*uuid.UUID]
	TargetMachineID                   Optional[*uuid.UUID]
	CreatedBy                         Optional[string]
	ParentTicketID                    Optional[*uuid.UUID]
	ExternalRef                       Optional[string]
	BudgetUSD                         Optional[float64]
	RestrictStatusToWorkflowFinishSet bool
}

// AddDependencyInput adds a dependency edge to a ticket.
type AddDependencyInput struct {
	TicketID       uuid.UUID
	TargetTicketID uuid.UUID
	Type           entticketdependency.Type
}

// AddExternalLinkInput adds an external issue or PR association to a ticket.
type AddExternalLinkInput struct {
	TicketID   uuid.UUID
	LinkType   entticketexternallink.LinkType
	URL        string
	ExternalID string
	Title      string
	Status     string
	Relation   entticketexternallink.Relation
}

// DeleteDependencyResult reports which dependency edge was removed.
type DeleteDependencyResult struct {
	DeletedDependencyID uuid.UUID `json:"deleted_dependency_id"`
}

// DeleteExternalLinkResult reports which external link was removed.
type DeleteExternalLinkResult struct {
	DeletedExternalLinkID uuid.UUID `json:"deleted_external_link_id"`
}

// AddCommentInput creates a new ticket comment.
type AddCommentInput struct {
	TicketID  uuid.UUID
	Body      string
	CreatedBy string
}

// UpdateCommentInput updates an existing ticket comment body.
type UpdateCommentInput struct {
	TicketID   uuid.UUID
	CommentID  uuid.UUID
	Body       string
	EditedBy   string
	EditReason string
}

// DeleteCommentResult reports which comment was removed.
type DeleteCommentResult struct {
	DeletedCommentID uuid.UUID `json:"deleted_comment_id"`
}

// Service provides ticket CRUD and dependency orchestration.
type Service struct {
	client *ent.Client
}

// NewService constructs a ticket service backed by the provided ent client.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// List returns tickets in a project ordered for UI consumption.
func (s *Service) List(ctx context.Context, input ListInput) ([]Ticket, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return nil, err
	}

	query := s.client.Ticket.Query().
		Where(entticket.ProjectIDEQ(input.ProjectID)).
		Order(ent.Asc(entticket.FieldCreatedAt), ent.Asc(entticket.FieldIdentifier)).
		WithStatus().
		WithParent(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		WithExternalLinks(func(query *ent.TicketExternalLinkQuery) {
			query.Order(ent.Asc(entticketexternallink.FieldCreatedAt), ent.Asc(entticketexternallink.FieldID))
		})

	if len(input.StatusNames) > 0 {
		query = query.Where(entticket.HasStatusWith(entticketstatus.NameIn(input.StatusNames...)))
	}
	if len(input.Priorities) > 0 {
		query = query.Where(entticket.PriorityIn(input.Priorities...))
	}
	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	tickets := make([]Ticket, 0, len(items))
	for _, item := range items {
		tickets = append(tickets, mapTicket(item))
	}

	return tickets, nil
}

// Get loads a single ticket with its related status, parent, children, and dependencies.
func (s *Service) Get(ctx context.Context, ticketID uuid.UUID) (Ticket, error) {
	if s.client == nil {
		return Ticket{}, ErrUnavailable
	}

	item, err := s.client.Ticket.Query().
		Where(entticket.ID(ticketID)).
		WithStatus().
		WithParent(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		WithChildren(func(query *ent.TicketQuery) {
			query.Order(ent.Asc(entticket.FieldCreatedAt), ent.Asc(entticket.FieldIdentifier)).WithStatus()
		}).
		WithOutgoingDependencies(func(query *ent.TicketDependencyQuery) {
			query.Order(ent.Asc(entticketdependency.FieldType), ent.Asc(entticketdependency.FieldTargetTicketID)).
				WithTargetTicket(func(ticketQuery *ent.TicketQuery) {
					ticketQuery.WithStatus()
				})
		}).
		WithExternalLinks(func(query *ent.TicketExternalLinkQuery) {
			query.Order(ent.Asc(entticketexternallink.FieldCreatedAt), ent.Asc(entticketexternallink.FieldID))
		}).
		Only(ctx)
	if err != nil {
		return Ticket{}, s.mapTicketReadError("get ticket", err)
	}

	return mapTicket(item), nil
}

// Create persists a new ticket and applies project defaults.
func (s *Service) Create(ctx context.Context, input CreateInput) (Ticket, error) {
	if s.client == nil {
		return Ticket{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Ticket{}, fmt.Errorf("start ticket create tx: %w", err)
	}
	defer rollback(tx)

	if err := s.ensureProjectExistsTx(ctx, tx, input.ProjectID); err != nil {
		return Ticket{}, err
	}

	statusID, err := s.resolveCreateStatusID(ctx, tx, input.ProjectID, input.StatusID)
	if err != nil {
		return Ticket{}, err
	}
	if input.WorkflowID != nil {
		if err := ensureWorkflowBelongsToProject(ctx, tx, input.ProjectID, *input.WorkflowID); err != nil {
			return Ticket{}, err
		}
	}
	if input.TargetMachineID != nil {
		if err := ensureTargetMachineBelongsToProjectOrganization(ctx, tx, input.ProjectID, *input.TargetMachineID); err != nil {
			return Ticket{}, err
		}
	}
	if input.ParentTicketID != nil {
		if err := ensureTicketBelongsToProject(ctx, tx, input.ProjectID, *input.ParentTicketID, ErrParentTicketNotFound); err != nil {
			return Ticket{}, err
		}
	}

	identifier, err := nextTicketIdentifier(ctx, tx, input.ProjectID)
	if err != nil {
		return Ticket{}, err
	}

	builder := tx.Ticket.Create().
		SetProjectID(input.ProjectID).
		SetIdentifier(identifier).
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetStatusID(statusID).
		SetPriority(input.Priority).
		SetType(input.Type).
		SetCreatedBy(resolveCreatedBy(input.CreatedBy)).
		SetBudgetUsd(input.BudgetUSD)

	if input.WorkflowID != nil {
		builder.SetWorkflowID(*input.WorkflowID)
	}
	if input.TargetMachineID != nil {
		builder.SetTargetMachineID(*input.TargetMachineID)
	}
	if input.ParentTicketID != nil {
		builder.SetParentTicketID(*input.ParentTicketID)
	}
	if input.ExternalRef != "" {
		builder.SetExternalRef(input.ExternalRef)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return Ticket{}, s.mapTicketWriteError("create ticket", err)
	}

	if input.ParentTicketID != nil {
		if _, err := ensureSubIssueDependency(ctx, tx, created.ID, *input.ParentTicketID); err != nil {
			return Ticket{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Ticket{}, fmt.Errorf("commit ticket create tx: %w", err)
	}

	return s.Get(ctx, created.ID)
}

// Update applies a partial update to an existing ticket.
func (s *Service) Update(ctx context.Context, input UpdateInput) (Ticket, error) {
	if s.client == nil {
		return Ticket{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Ticket{}, fmt.Errorf("start ticket update tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return Ticket{}, s.mapTicketReadError("get ticket for update", err)
	}

	builder := tx.Ticket.UpdateOneID(current.ID)
	statusChanged := false
	targetMachineChanged := false

	if input.Title.Set {
		builder.SetTitle(input.Title.Value)
	}
	if input.Description.Set {
		builder.SetDescription(input.Description.Value)
	}
	if input.StatusID.Set {
		if err := ensureStatusBelongsToProject(ctx, tx, current.ProjectID, input.StatusID.Value); err != nil {
			return Ticket{}, err
		}
		if input.RestrictStatusToWorkflowFinishSet && current.WorkflowID != nil {
			if err := ensureStatusAllowedByWorkflowFinishSet(ctx, tx, *current.WorkflowID, input.StatusID.Value); err != nil {
				return Ticket{}, err
			}
		}
		statusChanged = input.StatusID.Value != current.StatusID
		builder.SetStatusID(input.StatusID.Value)
	}
	if input.Priority.Set {
		builder.SetPriority(input.Priority.Value)
	}
	if input.Type.Set {
		builder.SetType(input.Type.Value)
	}
	if input.WorkflowID.Set {
		if input.WorkflowID.Value == nil {
			builder.ClearWorkflowID()
		} else {
			if err := ensureWorkflowBelongsToProject(ctx, tx, current.ProjectID, *input.WorkflowID.Value); err != nil {
				return Ticket{}, err
			}
			builder.SetWorkflowID(*input.WorkflowID.Value)
		}
	}
	if input.TargetMachineID.Set {
		if input.TargetMachineID.Value == nil {
			builder.ClearTargetMachineID()
		} else {
			if err := ensureTargetMachineBelongsToProjectOrganization(ctx, tx, current.ProjectID, *input.TargetMachineID.Value); err != nil {
				return Ticket{}, err
			}
			builder.SetTargetMachineID(*input.TargetMachineID.Value)
		}
		targetMachineChanged = !optionalUUIDPointerEqual(current.TargetMachineID, input.TargetMachineID.Value)
		if targetMachineChanged {
			builder.ClearCurrentRunID()
		}
	}
	if input.CreatedBy.Set {
		builder.SetCreatedBy(resolveCreatedBy(input.CreatedBy.Value))
	}
	if input.ExternalRef.Set {
		if strings.TrimSpace(input.ExternalRef.Value) == "" {
			builder.ClearExternalRef()
		} else {
			builder.SetExternalRef(strings.TrimSpace(input.ExternalRef.Value))
		}
	}
	if input.BudgetUSD.Set {
		builder.SetBudgetUsd(input.BudgetUSD.Value)
		reconcileBudgetPauseState(builder, current, input.BudgetUSD.Value)
	}
	if input.ParentTicketID.Set {
		if input.ParentTicketID.Value == nil {
			builder.ClearParentTicketID()
		} else {
			if *input.ParentTicketID.Value == current.ID {
				return Ticket{}, ErrInvalidDependency
			}
			if err := ensureTicketBelongsToProject(ctx, tx, current.ProjectID, *input.ParentTicketID.Value, ErrParentTicketNotFound); err != nil {
				return Ticket{}, err
			}
			if err := ensureParentDoesNotCreateCycle(ctx, tx, current.ID, *input.ParentTicketID.Value); err != nil {
				return Ticket{}, err
			}
			builder.SetParentTicketID(*input.ParentTicketID.Value)
		}
	}
	if statusChanged {
		builder.ClearCurrentRunID()
	}

	if _, err := builder.Save(ctx); err != nil {
		return Ticket{}, s.mapTicketWriteError("update ticket", err)
	}
	if statusChanged || targetMachineChanged {
		if err := releaseTicketAgentClaim(ctx, tx, current, entagentrun.StatusTerminated); err != nil {
			return Ticket{}, err
		}
	}

	if input.ParentTicketID.Set {
		if err := syncSubIssueDependencies(ctx, tx, current.ID, input.ParentTicketID.Value); err != nil {
			return Ticket{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Ticket{}, fmt.Errorf("commit ticket update tx: %w", err)
	}

	return s.Get(ctx, current.ID)
}

// AddDependency creates a dependency edge between two tickets.
func (s *Service) AddDependency(ctx context.Context, input AddDependencyInput) (Dependency, error) {
	if s.client == nil {
		return Dependency{}, ErrUnavailable
	}
	if input.TicketID == input.TargetTicketID {
		return Dependency{}, ErrInvalidDependency
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Dependency{}, fmt.Errorf("start add ticket dependency tx: %w", err)
	}
	defer rollback(tx)

	source, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return Dependency{}, s.mapTicketReadError("get source ticket", err)
	}
	if err := ensureTicketBelongsToProject(ctx, tx, source.ProjectID, input.TargetTicketID, ErrTicketNotFound); err != nil {
		return Dependency{}, err
	}

	var dependency *ent.TicketDependency
	if input.Type == entticketdependency.TypeSubIssue {
		if err := ensureParentDoesNotCreateCycle(ctx, tx, source.ID, input.TargetTicketID); err != nil {
			return Dependency{}, err
		}
		if _, err := tx.Ticket.UpdateOneID(source.ID).SetParentTicketID(input.TargetTicketID).Save(ctx); err != nil {
			return Dependency{}, s.mapTicketWriteError("set ticket parent", err)
		}
		dependency, err = ensureSubIssueDependency(ctx, tx, source.ID, input.TargetTicketID)
		if err != nil {
			return Dependency{}, err
		}
	} else {
		dependency, err = tx.TicketDependency.Create().
			SetSourceTicketID(source.ID).
			SetTargetTicketID(input.TargetTicketID).
			SetType(input.Type).
			Save(ctx)
		if err != nil {
			return Dependency{}, s.mapTicketWriteError("create ticket dependency", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Dependency{}, fmt.Errorf("commit add ticket dependency tx: %w", err)
	}

	dependency, err = s.client.TicketDependency.Query().
		Where(entticketdependency.ID(dependency.ID)).
		WithTargetTicket(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		Only(ctx)
	if err != nil {
		return Dependency{}, fmt.Errorf("reload ticket dependency: %w", err)
	}

	return mapDependency(dependency), nil
}

// RemoveDependency deletes a dependency edge from a ticket.
func (s *Service) RemoveDependency(ctx context.Context, ticketID uuid.UUID, dependencyID uuid.UUID) (DeleteDependencyResult, error) {
	if s.client == nil {
		return DeleteDependencyResult{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return DeleteDependencyResult{}, fmt.Errorf("start delete ticket dependency tx: %w", err)
	}
	defer rollback(tx)

	dependency, err := tx.TicketDependency.Query().
		Where(
			entticketdependency.ID(dependencyID),
			entticketdependency.SourceTicketIDEQ(ticketID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return DeleteDependencyResult{}, ErrDependencyNotFound
		}
		return DeleteDependencyResult{}, fmt.Errorf("get ticket dependency for delete: %w", err)
	}

	if dependency.Type == entticketdependency.TypeSubIssue {
		source, sourceErr := tx.Ticket.Get(ctx, ticketID)
		if sourceErr != nil {
			return DeleteDependencyResult{}, s.mapTicketReadError("get ticket for dependency delete", sourceErr)
		}
		if source.ParentTicketID != nil && *source.ParentTicketID == dependency.TargetTicketID {
			if _, err := tx.Ticket.UpdateOneID(ticketID).ClearParentTicketID().Save(ctx); err != nil {
				return DeleteDependencyResult{}, s.mapTicketWriteError("clear ticket parent", err)
			}
		}
	}

	if err := tx.TicketDependency.DeleteOneID(dependencyID).Exec(ctx); err != nil {
		return DeleteDependencyResult{}, s.mapTicketWriteError("delete ticket dependency", err)
	}
	if err := tx.Commit(); err != nil {
		return DeleteDependencyResult{}, fmt.Errorf("commit delete ticket dependency tx: %w", err)
	}

	return DeleteDependencyResult{DeletedDependencyID: dependencyID}, nil
}

// AddExternalLink creates a new external issue or PR association for a ticket.
func (s *Service) AddExternalLink(ctx context.Context, input AddExternalLinkInput) (ExternalLink, error) {
	if s.client == nil {
		return ExternalLink{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ExternalLink{}, fmt.Errorf("start add ticket external link tx: %w", err)
	}
	defer rollback(tx)

	source, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return ExternalLink{}, s.mapTicketReadError("get ticket for external link create", err)
	}

	builder := tx.TicketExternalLink.Create().
		SetTicketID(source.ID).
		SetLinkType(input.LinkType).
		SetURL(input.URL).
		SetExternalID(input.ExternalID).
		SetRelation(input.Relation)
	if input.Title != "" {
		builder.SetTitle(input.Title)
	}
	if input.Status != "" {
		builder.SetStatus(input.Status)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return ExternalLink{}, s.mapTicketWriteError("create ticket external link", err)
	}

	if strings.TrimSpace(source.ExternalRef) == "" {
		if _, err := tx.Ticket.UpdateOneID(source.ID).SetExternalRef(input.ExternalID).Save(ctx); err != nil {
			return ExternalLink{}, s.mapTicketWriteError("set ticket external_ref", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return ExternalLink{}, fmt.Errorf("commit add ticket external link tx: %w", err)
	}

	return mapExternalLink(created), nil
}

// ListComments returns user discussion comments ordered oldest-first for stable thread rendering.
func (s *Service) ListComments(ctx context.Context, ticketID uuid.UUID) ([]Comment, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}
	if _, err := s.client.Ticket.Get(ctx, ticketID); err != nil {
		return nil, s.mapTicketReadError("get ticket for comment list", err)
	}

	items, err := s.client.TicketComment.Query().
		Where(entticketcomment.TicketIDEQ(ticketID)).
		Order(ent.Asc(entticketcomment.FieldCreatedAt), ent.Asc(entticketcomment.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket comments: %w", err)
	}

	comments := make([]Comment, 0, len(items))
	for _, item := range items {
		comments = append(comments, mapComment(item))
	}

	return comments, nil
}

// ListCommentRevisions returns immutable comment history oldest-first.
func (s *Service) ListCommentRevisions(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) ([]CommentRevision, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}

	comment, err := s.client.TicketComment.Query().
		Where(
			entticketcomment.IDEQ(commentID),
			entticketcomment.TicketIDEQ(ticketID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("get ticket comment for revisions: %w", err)
	}

	revisions, err := s.client.TicketCommentRevision.Query().
		Where(entticketcommentrevision.CommentIDEQ(comment.ID)).
		Order(ent.Asc(entticketcommentrevision.FieldRevisionNumber), ent.Asc(entticketcommentrevision.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket comment revisions: %w", err)
	}
	if len(revisions) == 0 {
		return []CommentRevision{s.syntheticInitialRevision(comment)}, nil
	}

	items := make([]CommentRevision, 0, len(revisions))
	for _, item := range revisions {
		items = append(items, mapCommentRevision(item))
	}

	return items, nil
}

// AddComment creates a new user discussion comment on a ticket.
func (s *Service) AddComment(ctx context.Context, input AddCommentInput) (Comment, error) {
	if s.client == nil {
		return Comment{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("start add ticket comment tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.Ticket.Get(ctx, input.TicketID); err != nil {
		return Comment{}, s.mapTicketReadError("get ticket for comment create", err)
	}

	now := timeNowUTC()
	createdBy := resolveCreatedBy(input.CreatedBy)
	item, err := tx.TicketComment.Create().
		SetTicketID(input.TicketID).
		SetBody(input.Body).
		SetCreatedBy(createdBy).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return Comment{}, s.mapTicketWriteError("create ticket comment", err)
	}
	if err := s.appendCommentRevisionTx(ctx, tx, item.ID, 1, item.Body, createdBy, now, ""); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit add ticket comment tx: %w", err)
	}

	return mapComment(item), nil
}

// UpdateComment updates the markdown body of an existing ticket discussion comment.
func (s *Service) UpdateComment(ctx context.Context, input UpdateCommentInput) (Comment, error) {
	if s.client == nil {
		return Comment{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("start update ticket comment tx: %w", err)
	}
	defer rollback(tx)

	existing, err := tx.TicketComment.Query().
		Where(
			entticketcomment.IDEQ(input.CommentID),
			entticketcomment.TicketIDEQ(input.TicketID),
			entticketcomment.IsDeleted(false),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return Comment{}, ErrCommentNotFound
		}
		return Comment{}, fmt.Errorf("get ticket comment for update: %w", err)
	}

	now := timeNowUTC()
	revisionNumber, err := s.ensureInitialRevisionTx(ctx, tx, existing)
	if err != nil {
		return Comment{}, err
	}
	editor := resolveCreatedBy(input.EditedBy)
	revisionNumber++

	item, err := tx.TicketComment.UpdateOneID(existing.ID).
		SetBody(input.Body).
		SetUpdatedAt(now).
		SetEditedAt(now).
		SetEditCount(revisionNumber - 1).
		SetLastEditedBy(editor).
		Save(ctx)
	if err != nil {
		return Comment{}, s.mapTicketWriteError("update ticket comment", err)
	}
	if err := s.appendCommentRevisionTx(ctx, tx, existing.ID, revisionNumber, input.Body, editor, now, input.EditReason); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit update ticket comment tx: %w", err)
	}

	return mapComment(item), nil
}

// RemoveExternalLink deletes an external issue or PR association from a ticket.
func (s *Service) RemoveExternalLink(ctx context.Context, ticketID uuid.UUID, externalLinkID uuid.UUID) (DeleteExternalLinkResult, error) {
	if s.client == nil {
		return DeleteExternalLinkResult{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return DeleteExternalLinkResult{}, fmt.Errorf("start delete ticket external link tx: %w", err)
	}
	defer rollback(tx)

	link, err := tx.TicketExternalLink.Query().
		Where(
			entticketexternallink.ID(externalLinkID),
			entticketexternallink.TicketIDEQ(ticketID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return DeleteExternalLinkResult{}, ErrExternalLinkNotFound
		}
		return DeleteExternalLinkResult{}, fmt.Errorf("get ticket external link for delete: %w", err)
	}

	source, err := tx.Ticket.Get(ctx, ticketID)
	if err != nil {
		return DeleteExternalLinkResult{}, s.mapTicketReadError("get ticket for external link delete", err)
	}

	if err := tx.TicketExternalLink.DeleteOneID(externalLinkID).Exec(ctx); err != nil {
		return DeleteExternalLinkResult{}, s.mapTicketWriteError("delete ticket external link", err)
	}

	if strings.TrimSpace(source.ExternalRef) == link.ExternalID {
		replacement, replacementErr := tx.TicketExternalLink.Query().
			Where(entticketexternallink.TicketIDEQ(ticketID)).
			Order(ent.Asc(entticketexternallink.FieldCreatedAt), ent.Asc(entticketexternallink.FieldID)).
			First(ctx)
		switch {
		case ent.IsNotFound(replacementErr):
			if _, err := tx.Ticket.UpdateOneID(ticketID).ClearExternalRef().Save(ctx); err != nil {
				return DeleteExternalLinkResult{}, s.mapTicketWriteError("clear ticket external_ref", err)
			}
		case replacementErr != nil:
			return DeleteExternalLinkResult{}, fmt.Errorf("select replacement external link: %w", replacementErr)
		default:
			if _, err := tx.Ticket.UpdateOneID(ticketID).SetExternalRef(replacement.ExternalID).Save(ctx); err != nil {
				return DeleteExternalLinkResult{}, s.mapTicketWriteError("replace ticket external_ref", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return DeleteExternalLinkResult{}, fmt.Errorf("commit delete ticket external link tx: %w", err)
	}

	return DeleteExternalLinkResult{DeletedExternalLinkID: externalLinkID}, nil
}

// RemoveComment deletes a user discussion comment from a ticket.
func (s *Service) RemoveComment(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) (DeleteCommentResult, error) {
	if s.client == nil {
		return DeleteCommentResult{}, ErrUnavailable
	}

	now := timeNowUTC()
	deleted, err := s.client.TicketComment.Update().
		Where(
			entticketcomment.IDEQ(commentID),
			entticketcomment.TicketIDEQ(ticketID),
			entticketcomment.IsDeleted(false),
		).
		SetIsDeleted(true).
		SetDeletedAt(now).
		SetDeletedBy(defaultCreatedBy).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return DeleteCommentResult{}, fmt.Errorf("soft delete ticket comment: %w", err)
	}
	if deleted == 0 {
		return DeleteCommentResult{}, ErrCommentNotFound
	}

	return DeleteCommentResult{DeletedCommentID: commentID}, nil
}

func (s *Service) ensureInitialRevisionTx(ctx context.Context, tx *ent.Tx, comment *ent.TicketComment) (int, error) {
	latest, err := tx.TicketCommentRevision.Query().
		Where(entticketcommentrevision.CommentIDEQ(comment.ID)).
		Order(ent.Desc(entticketcommentrevision.FieldRevisionNumber), ent.Desc(entticketcommentrevision.FieldID)).
		First(ctx)
	switch {
	case err == nil:
		return latest.RevisionNumber, nil
	case !ent.IsNotFound(err):
		return 0, fmt.Errorf("load latest ticket comment revision: %w", err)
	}

	if err := s.appendCommentRevisionTx(ctx, tx, comment.ID, 1, comment.Body, comment.CreatedBy, comment.CreatedAt, ""); err != nil {
		return 0, err
	}

	return 1, nil
}

func (s *Service) appendCommentRevisionTx(
	ctx context.Context,
	tx *ent.Tx,
	commentID uuid.UUID,
	revisionNumber int,
	bodyMarkdown string,
	editedBy string,
	editedAt time.Time,
	editReason string,
) error {
	create := tx.TicketCommentRevision.Create().
		SetCommentID(commentID).
		SetRevisionNumber(revisionNumber).
		SetBodyMarkdown(bodyMarkdown).
		SetEditedBy(resolveCreatedBy(editedBy)).
		SetEditedAt(editedAt)
	if trimmed := strings.TrimSpace(editReason); trimmed != "" {
		create.SetEditReason(trimmed)
	}
	if _, err := create.Save(ctx); err != nil {
		return s.mapTicketWriteError("create ticket comment revision", err)
	}

	return nil
}

func (s *Service) syntheticInitialRevision(comment *ent.TicketComment) CommentRevision {
	return CommentRevision{
		ID:             uuid.Nil,
		CommentID:      comment.ID,
		RevisionNumber: 1,
		BodyMarkdown:   comment.Body,
		EditedBy:       comment.CreatedBy,
		EditedAt:       comment.CreatedAt,
	}
}

func (s *Service) ensureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := s.client.Project.Query().Where(project.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func (s *Service) ensureProjectExistsTx(ctx context.Context, tx *ent.Tx, projectID uuid.UUID) error {
	exists, err := tx.Project.Query().Where(project.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func (s *Service) resolveCreateStatusID(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, inputStatusID *uuid.UUID) (uuid.UUID, error) {
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

func (s *Service) mapTicketReadError(action string, err error) error {
	if ent.IsNotFound(err) {
		return ErrTicketNotFound
	}

	return fmt.Errorf("%s: %w", action, err)
}

func (s *Service) mapTicketWriteError(action string, err error) error {
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
		ID:                item.ID,
		ProjectID:         item.ProjectID,
		Identifier:        item.Identifier,
		Title:             item.Title,
		Description:       item.Description,
		StatusID:          item.StatusID,
		Priority:          item.Priority,
		Type:              item.Type,
		WorkflowID:        item.WorkflowID,
		CurrentRunID:      item.CurrentRunID,
		TargetMachineID:   item.TargetMachineID,
		CreatedBy:         item.CreatedBy,
		Children:          []TicketReference{},
		Dependencies:      []Dependency{},
		ExternalLinks:     []ExternalLink{},
		ExternalRef:       item.ExternalRef,
		BudgetUSD:         item.BudgetUsd,
		CostTokensInput:   item.CostTokensInput,
		CostTokensOutput:  item.CostTokensOutput,
		CostAmount:        item.CostAmount,
		AttemptCount:      item.AttemptCount,
		ConsecutiveErrors: item.ConsecutiveErrors,
		StartedAt:         item.StartedAt,
		CompletedAt:       item.CompletedAt,
		NextRetryAt:       item.NextRetryAt,
		RetryPaused:       item.RetryPaused,
		PauseReason:       item.PauseReason,
		CreatedAt:         item.CreatedAt,
	}

	if item.Edges.Status != nil {
		result.StatusName = item.Edges.Status.Name
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
	for _, externalLink := range item.Edges.ExternalLinks {
		result.ExternalLinks = append(result.ExternalLinks, mapExternalLink(externalLink))
	}

	return result
}

func mapDependency(item *ent.TicketDependency) Dependency {
	dependency := Dependency{
		ID:   item.ID,
		Type: item.Type,
	}
	if item.Edges.TargetTicket != nil {
		dependency.Target = mapTicketReference(item.Edges.TargetTicket)
	}

	return dependency
}

func mapExternalLink(item *ent.TicketExternalLink) ExternalLink {
	return ExternalLink{
		ID:         item.ID,
		LinkType:   item.LinkType,
		URL:        item.URL,
		ExternalID: item.ExternalID,
		Title:      item.Title,
		Status:     item.Status,
		Relation:   item.Relation,
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
