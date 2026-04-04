package ticketing

import (
	"fmt"
	"strings"
)

type StatusStage string

const (
	DefaultStatusStage   StatusStage = StatusStageUnstarted
	StatusStageBacklog   StatusStage = "backlog"
	StatusStageUnstarted StatusStage = "unstarted"
	StatusStageStarted   StatusStage = "started"
	StatusStageCompleted StatusStage = "completed"
	StatusStageCanceled  StatusStage = "canceled"
)

func (s StatusStage) String() string {
	return string(s)
}

func (s StatusStage) IsValid() bool {
	switch s {
	case StatusStageBacklog, StatusStageUnstarted, StatusStageStarted, StatusStageCompleted, StatusStageCanceled:
		return true
	default:
		return false
	}
}

func (s StatusStage) IsTerminal() bool {
	switch s {
	case StatusStageCompleted, StatusStageCanceled:
		return true
	default:
		return false
	}
}

func (s StatusStage) AllowsWorkflowPickup() bool {
	switch s {
	case StatusStageBacklog, StatusStageUnstarted, StatusStageStarted:
		return true
	default:
		return false
	}
}

func (s StatusStage) AllowsWorkflowFinish() bool {
	return s.IsTerminal()
}

func ParseStatusStage(raw string) (StatusStage, error) {
	stage := StatusStage(strings.ToLower(strings.TrimSpace(raw)))
	if !stage.IsValid() {
		return "", fmt.Errorf("stage must be one of backlog, unstarted, started, completed, canceled")
	}
	return stage, nil
}

func DefaultTemplateStatusStage(name string) (StatusStage, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "backlog":
		return StatusStageBacklog, true
	case "todo":
		return StatusStageUnstarted, true
	case "in progress", "in-progress":
		return StatusStageStarted, true
	case "in review", "in-review":
		return StatusStageStarted, true
	case "done":
		return StatusStageCompleted, true
	case "cancelled", "canceled":
		return StatusStageCanceled, true
	default:
		return "", false
	}
}

func InferStatusStageFromName(name string) (StatusStage, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "backlog":
		return StatusStageBacklog, true
	case "todo", "ready", "ready for work", "queued", "triaged", "planned":
		return StatusStageUnstarted, true
	case "in progress", "in-progress", "in review", "in-review", "review", "qa", "testing", "blocked":
		return StatusStageStarted, true
	case "done", "completed", "complete", "closed", "resolved", "merged":
		return StatusStageCompleted, true
	case "cancelled", "canceled", "wontfix", "won't fix", "duplicate", "invalid", "archived":
		return StatusStageCanceled, true
	default:
		return "", false
	}
}
