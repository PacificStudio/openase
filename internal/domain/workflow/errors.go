package workflow

import "errors"

var (
	ErrProjectNotFound  = errors.New("project not found")
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrStatusNotFound   = errors.New("workflow status not found in project")
	ErrAgentNotFound    = errors.New("workflow agent not found in project")
	ErrWorkflowConflict = errors.New("workflow conflict")
	ErrWorkflowInUse    = errors.New("workflow is still referenced by project or tickets")
	ErrSkillInvalid     = errors.New("skill is invalid")
	ErrSkillNotFound    = errors.New("skill not found")
)
