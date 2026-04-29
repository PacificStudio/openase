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

type ProviderProjectReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ProviderWorkflowReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ProviderAgentReference struct {
	ID          uuid.UUID                   `json:"id"`
	ProjectID   uuid.UUID                   `json:"project_id"`
	ProjectName string                      `json:"project_name"`
	Name        string                      `json:"name"`
	Workflows   []ProviderWorkflowReference `json:"workflows"`
}

type ProviderConversationReference struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Status      string    `json:"status"`
}

type ProviderConversationPrincipalReference struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    uuid.UUID `json:"project_id"`
	ProjectName  string    `json:"project_name"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	RuntimeState string    `json:"runtime_state"`
}

type ProviderConversationRunReference struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Status      string    `json:"status"`
}

type AgentProviderDeleteConflict struct {
	ProviderID             uuid.UUID                                `json:"provider_id"`
	OrganizationDefault    bool                                     `json:"organization_default"`
	ProjectDefaults        []ProviderProjectReference               `json:"project_defaults"`
	Agents                 []ProviderAgentReference                 `json:"agents"`
	AgentRuns              []AgentRunReference                      `json:"agent_runs"`
	ChatConversations      []ProviderConversationReference          `json:"chat_conversations"`
	ConversationPrincipals []ProviderConversationPrincipalReference `json:"conversation_principals"`
	ConversationRuns       []ProviderConversationRunReference       `json:"conversation_runs"`
}

func (e *AgentProviderDeleteConflict) Error() string {
	if e == nil {
		return ""
	}
	return ErrAgentProviderInUseConflict.Error()
}

func (e *AgentProviderDeleteConflict) Unwrap() error {
	if e == nil {
		return nil
	}
	return ErrAgentProviderInUseConflict
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
