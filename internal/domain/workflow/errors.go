package workflow

import "errors"

var (
	ErrProjectNotFound                    = errors.New("project not found")
	ErrWorkflowNotFound                   = errors.New("workflow not found")
	ErrStatusNotFound                     = errors.New("workflow status not found in project")
	ErrAgentNotFound                      = errors.New("workflow agent not found in project")
	ErrWorkflowNameConflict               = errors.New("workflow name already exists in this project")
	ErrWorkflowHarnessPathConflict        = errors.New("workflow harness path already exists in this project")
	ErrWorkflowConflict                   = errors.New("workflow conflict")
	ErrPickupStatusConflict               = errors.New("workflow pickup status conflict")
	ErrWorkflowStatusBindingOverlap       = errors.New("workflow pickup and finish statuses must not overlap")
	ErrWorkflowReferencedByTickets        = errors.New("workflow cannot be deleted because tickets still reference it")
	ErrWorkflowReferencedByScheduledJobs  = errors.New("workflow cannot be deleted because scheduled jobs still reference it")
	ErrWorkflowInUse                      = errors.New("workflow is still referenced by project or tickets")
	ErrWorkflowReplacementRequired        = errors.New("workflow references must be replaced before purge")
	ErrWorkflowActiveAgentRuns            = errors.New("workflow has active agent runs")
	ErrWorkflowHistoricalAgentRuns        = errors.New("workflow has historical agent runs")
	ErrWorkflowReplacementInvalid         = errors.New("workflow replacement is invalid")
	ErrWorkflowReplacementNotFound        = errors.New("replacement workflow not found")
	ErrWorkflowReplacementProjectMismatch = errors.New("replacement workflow must belong to the same project")
	ErrWorkflowReplacementInactive        = errors.New("replacement workflow must be active")
	ErrSkillInvalid                       = errors.New("skill is invalid")
	ErrSkillNotFound                      = errors.New("skill not found")
)
