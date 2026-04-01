package hradvisor

import (
	"strings"

	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
)

func parseProjectStatus(raw string) projectStatus {
	switch normalizeCompact(raw) {
	case normalizeCompact(string(projectStatusBacklog)):
		return projectStatusBacklog
	case "planned", "plan", "planning":
		return projectStatusPlanned
	case "inprogress", "active", "started":
		return projectStatusInProgress
	case "completed", "complete", "done", "closed":
		return projectStatusCompleted
	case "canceled", "cancelled":
		return projectStatusCanceled
	case "archived":
		return projectStatusArchived
	default:
		return projectStatusUnknown
	}
}

func normalizeCompact(value string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}

func parseStatusStage(statusName string, rawStage string) ticketing.StatusStage {
	if stage, err := ticketing.ParseStatusStage(rawStage); err == nil {
		return stage
	}
	if stage, ok := ticketing.InferStatusStageFromName(statusName); ok {
		return stage
	}
	return ""
}

func isDoneStatus(statusName string, rawStage string) bool {
	stage := parseStatusStage(statusName, rawStage)
	if stage != "" {
		return stage.IsTerminal()
	}
	switch strings.ToLower(strings.TrimSpace(statusName)) {
	case "done", "completed", "closed", "archived":
		return true
	default:
		return false
	}
}
