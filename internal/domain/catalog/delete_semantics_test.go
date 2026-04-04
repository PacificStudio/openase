package catalog

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestAgentRuntimeControlRetireAndRetiredGuards(t *testing.T) {
	runID := uuid.New()
	ticketID := uuid.New()

	if AgentRuntimeControlStateRetired.String() != "retired" {
		t.Fatalf("AgentRuntimeControlStateRetired.String() = %q", AgentRuntimeControlStateRetired.String())
	}
	if !AgentRuntimeControlStateRetired.IsValid() {
		t.Fatal("AgentRuntimeControlStateRetired should be valid")
	}

	retiredAgent := Agent{RuntimeControlState: AgentRuntimeControlStateRetired}
	if _, err := ResolvePauseRuntimeControlState(retiredAgent); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected retired validation error")
	}
	if _, err := ResolveResumeRuntimeControlState(Agent{
		RuntimeControlState: AgentRuntimeControlStateRetired,
		Runtime: &AgentRuntime{
			CurrentRunID:    &runID,
			CurrentTicketID: &ticketID,
			Status:          AgentStatusRunning,
		},
	}); err == nil {
		t.Fatal("ResolveResumeRuntimeControlState() expected retired validation error")
	}
	if _, err := ResolveRetireRuntimeControlState(retiredAgent); err == nil {
		t.Fatal("ResolveRetireRuntimeControlState() expected already retired validation error")
	}
	state, err := ResolveRetireRuntimeControlState(Agent{})
	if err != nil || state != AgentRuntimeControlStateRetired {
		t.Fatalf("ResolveRetireRuntimeControlState() = %q, %v; want retired, nil", state, err)
	}
	if _, err := ResolveRetireRuntimeControlState(Agent{
		Runtime: &AgentRuntime{CurrentRunID: &runID},
	}); err == nil {
		t.Fatal("ResolveRetireRuntimeControlState() expected active run validation error")
	}
}

func TestDeleteConflictErrorsWrapDomainConflicts(t *testing.T) {
	projectRepoConflict := &ProjectRepoDeleteConflict{RepoID: uuid.New()}
	if !errors.Is(projectRepoConflict, ErrProjectRepoInUseConflict) {
		t.Fatal("ProjectRepoDeleteConflict should wrap ErrProjectRepoInUseConflict")
	}
	if projectRepoConflict.Error() != ErrProjectRepoInUseConflict.Error() {
		t.Fatalf("ProjectRepoDeleteConflict.Error() = %q", projectRepoConflict.Error())
	}

	ticketRepoScopeConflict := &TicketRepoScopeDeleteConflict{
		ScopeID:  uuid.New(),
		TicketID: uuid.New(),
	}
	if !errors.Is(ticketRepoScopeConflict, ErrTicketRepoScopeInUseConflict) {
		t.Fatal("TicketRepoScopeDeleteConflict should wrap ErrTicketRepoScopeInUseConflict")
	}
	if ticketRepoScopeConflict.Error() != ErrTicketRepoScopeInUseConflict.Error() {
		t.Fatalf("TicketRepoScopeDeleteConflict.Error() = %q", ticketRepoScopeConflict.Error())
	}

	agentConflict := &AgentDeleteConflict{AgentID: uuid.New()}
	if !errors.Is(agentConflict, ErrAgentInUseConflict) {
		t.Fatal("AgentDeleteConflict should wrap ErrAgentInUseConflict")
	}
	if agentConflict.Error() != ErrAgentInUseConflict.Error() {
		t.Fatalf("AgentDeleteConflict.Error() = %q", agentConflict.Error())
	}
}

func TestDeleteConflictNilReceivers(t *testing.T) {
	var projectRepoConflict *ProjectRepoDeleteConflict
	if got := projectRepoConflict.Error(); got != "" {
		t.Fatalf("nil ProjectRepoDeleteConflict.Error() = %q", got)
	}
	if projectRepoConflict.Unwrap() != nil {
		t.Fatal("nil ProjectRepoDeleteConflict.Unwrap() should be nil")
	}

	var ticketRepoScopeConflict *TicketRepoScopeDeleteConflict
	if got := ticketRepoScopeConflict.Error(); got != "" {
		t.Fatalf("nil TicketRepoScopeDeleteConflict.Error() = %q", got)
	}
	if ticketRepoScopeConflict.Unwrap() != nil {
		t.Fatal("nil TicketRepoScopeDeleteConflict.Unwrap() should be nil")
	}

	var agentConflict *AgentDeleteConflict
	if got := agentConflict.Error(); got != "" {
		t.Fatalf("nil AgentDeleteConflict.Error() = %q", got)
	}
	if agentConflict.Unwrap() != nil {
		t.Fatal("nil AgentDeleteConflict.Unwrap() should be nil")
	}
}
