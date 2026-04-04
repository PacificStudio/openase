package catalog

import "github.com/google/uuid"

type ProjectRepoScopeReference struct {
	ID         uuid.UUID `json:"id"`
	TicketID   uuid.UUID `json:"ticket_id"`
	BranchName string    `json:"branch_name"`
}

type ProjectRepoWorkspaceReference struct {
	ID         uuid.UUID `json:"id"`
	TicketID   uuid.UUID `json:"ticket_id"`
	AgentRunID uuid.UUID `json:"agent_run_id"`
	State      string    `json:"state"`
}

type ProjectRepoDeleteConflict struct {
	RepoID       uuid.UUID                       `json:"repo_id"`
	TicketScopes []ProjectRepoScopeReference     `json:"ticket_scopes"`
	Workspaces   []ProjectRepoWorkspaceReference `json:"workspaces"`
}

func (e *ProjectRepoDeleteConflict) Error() string {
	if e == nil {
		return ""
	}
	return ErrProjectRepoInUseConflict.Error()
}

func (e *ProjectRepoDeleteConflict) Unwrap() error {
	if e == nil {
		return nil
	}
	return ErrProjectRepoInUseConflict
}

type TicketRepoScopeActiveRunReference struct {
	TicketID     uuid.UUID `json:"ticket_id"`
	CurrentRunID uuid.UUID `json:"current_run_id"`
}

type TicketRepoScopeWorkspaceReference struct {
	ID         uuid.UUID `json:"id"`
	AgentRunID uuid.UUID `json:"agent_run_id"`
	State      string    `json:"state"`
}

type TicketRepoScopeDeleteConflict struct {
	ScopeID    uuid.UUID                           `json:"scope_id"`
	TicketID   uuid.UUID                           `json:"ticket_id"`
	ActiveRun  *TicketRepoScopeActiveRunReference  `json:"active_run,omitempty"`
	Workspaces []TicketRepoScopeWorkspaceReference `json:"workspaces"`
}

func (e *TicketRepoScopeDeleteConflict) Error() string {
	if e == nil {
		return ""
	}
	return ErrTicketRepoScopeInUseConflict.Error()
}

func (e *TicketRepoScopeDeleteConflict) Unwrap() error {
	if e == nil {
		return nil
	}
	return ErrTicketRepoScopeInUseConflict
}

type AgentRunReference struct {
	ID       uuid.UUID `json:"id"`
	TicketID uuid.UUID `json:"ticket_id"`
	Status   string    `json:"status"`
}

type AgentDeleteConflict struct {
	AgentID        uuid.UUID           `json:"agent_id"`
	ActiveRuns     []AgentRunReference `json:"active_runs"`
	HistoricalRuns []AgentRunReference `json:"historical_runs"`
}

func (e *AgentDeleteConflict) Error() string {
	if e == nil {
		return ""
	}
	return ErrAgentInUseConflict.Error()
}

func (e *AgentDeleteConflict) Unwrap() error {
	if e == nil {
		return nil
	}
	return ErrAgentInUseConflict
}
