package workflow

import (
	"strings"

	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
)

const RequiredWorkflowSkillName = "openase-platform"

func RequiredWorkflowPlatformAccessAllowed() []string {
	return []string{string(agentplatformdomain.ScopeTicketsUpdateSelf)}
}

func RequiredWorkflowSkillNames() []string {
	return []string{RequiredWorkflowSkillName}
}

func IsRequiredWorkflowSkillName(raw string) bool {
	return strings.TrimSpace(raw) == RequiredWorkflowSkillName
}
