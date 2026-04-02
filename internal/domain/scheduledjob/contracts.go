package scheduledjob

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrWorkflowNotFound     = errors.New("workflow not found")
	ErrScheduledJobNotFound = errors.New("scheduled job not found")
	ErrScheduledJobConflict = errors.New("scheduled job conflict")
	ErrStatusNotFound       = errors.New("ticket status not found")
)

type Job struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	Name           string
	CronExpression string
	TicketTemplate map[string]any
	IsEnabled      bool
	LastRunAt      *time.Time
	NextRunAt      *time.Time
	WorkflowID     *uuid.UUID
}

type Workflow struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	Name           string
	Type           string
	PickupStatuses []WorkflowStatus
}

type WorkflowStatus struct {
	ID   uuid.UUID
	Name string
}
