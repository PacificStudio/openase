package workflow

import (
	"testing"

	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
)

func TestRequiredWorkflowRequirements(t *testing.T) {
	if got := RequiredWorkflowPlatformAccessAllowed(); len(got) != 1 || got[0] != string(agentplatformdomain.ScopeTicketsUpdateSelf) {
		t.Fatalf("RequiredWorkflowPlatformAccessAllowed() = %v", got)
	}

	if got := RequiredWorkflowSkillNames(); len(got) != 1 || got[0] != RequiredWorkflowSkillName {
		t.Fatalf("RequiredWorkflowSkillNames() = %v", got)
	}
}

func TestIsRequiredWorkflowSkillName(t *testing.T) {
	if !IsRequiredWorkflowSkillName(" openase-platform ") {
		t.Fatal("expected trimmed required skill name to match")
	}
	if IsRequiredWorkflowSkillName("commit") {
		t.Fatal("expected unrelated skill name to be rejected")
	}
}
