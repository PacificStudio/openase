package ticketstatus

import (
	"errors"

	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

var (
	ErrProjectNotFound         = errors.New("project not found")
	ErrStatusNotFound          = errors.New("ticket status not found")
	ErrDuplicateStatusName     = errors.New("ticket status name already exists in project")
	ErrDefaultStatusRequired   = errors.New("at least one default ticket status is required")
	ErrDefaultStatusStage      = errors.New("default ticket status must use a non-terminal stage")
	ErrCannotDeleteLastStatus  = errors.New("cannot delete the last ticket status in a project")
	ErrReplacementStatusAbsent = errors.New("replacement ticket status not found")
)

type Optional[T any] struct {
	Set   bool
	Value T
}

type Status struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Name          string    `json:"name"`
	Stage         string    `json:"stage"`
	Color         string    `json:"color"`
	Icon          string    `json:"icon"`
	Position      int       `json:"position"`
	ActiveRuns    int       `json:"active_runs"`
	MaxActiveRuns *int      `json:"max_active_runs,omitempty"`
	IsDefault     bool      `json:"is_default"`
	Description   string    `json:"description"`
}

type StatusRuntimeSnapshot struct {
	StatusID      uuid.UUID `json:"status_id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Name          string    `json:"name"`
	Stage         string    `json:"stage"`
	Position      int       `json:"position"`
	MaxActiveRuns *int      `json:"max_active_runs,omitempty"`
	ActiveRuns    int       `json:"active_runs"`
}

type ListResult struct {
	Statuses []Status `json:"statuses"`
}

type CreateInput struct {
	ProjectID     uuid.UUID
	Name          string
	Stage         ticketing.StatusStage
	Color         string
	Icon          string
	Position      Optional[int]
	MaxActiveRuns *int
	IsDefault     bool
	Description   string
}

type UpdateInput struct {
	StatusID      uuid.UUID
	Name          Optional[string]
	Stage         Optional[ticketing.StatusStage]
	Color         Optional[string]
	Icon          Optional[string]
	Position      Optional[int]
	MaxActiveRuns Optional[*int]
	IsDefault     Optional[bool]
	Description   Optional[string]
}

type DeleteResult struct {
	DeletedStatusID     uuid.UUID `json:"deleted_status_id"`
	ReplacementStatusID uuid.UUID `json:"replacement_status_id"`
}

type TemplateStatus struct {
	Name          string
	Stage         ticketing.StatusStage
	Color         string
	Icon          string
	Position      int
	MaxActiveRuns *int
	IsDefault     bool
	Description   string
}
