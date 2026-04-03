package workflow

import "errors"

var (
	ErrProjectNotFound                   = errors.New("project not found")
	ErrWorkflowNotFound                  = errors.New("workflow not found")
	ErrStatusNotFound                    = errors.New("workflow status not found in project")
	ErrAgentNotFound                     = errors.New("workflow agent not found in project")
	ErrWorkflowNameConflict              = errors.New("workflow name already exists in this project")
	ErrWorkflowHarnessPathConflict       = errors.New("workflow harness path already exists in this project")
	ErrWorkflowConflict                  = errors.New("workflow conflict")
	ErrPickupStatusConflict              = errors.New("workflow pickup status conflict")
	ErrWorkflowReferencedByTickets       = errors.New("workflow cannot be deleted because tickets still reference it")
	ErrWorkflowReferencedByScheduledJobs = errors.New("workflow cannot be deleted because scheduled jobs still reference it")
	ErrWorkflowInUse                     = errors.New("workflow is still referenced by project or tickets")
	ErrSkillInvalid                      = errors.New("skill is invalid")
	ErrSkillNotFound                     = errors.New("skill not found")
)
