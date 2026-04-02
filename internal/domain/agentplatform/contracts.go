package agentplatform

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const TokenPrefix = "ase_agent_"

var ErrNotFound = errors.New("agent platform record not found")

type Scope string

const (
	ScopeTicketsCreate      Scope = "tickets.create"
	ScopeTicketsList        Scope = "tickets.list"
	ScopeTicketsReportUsage Scope = "tickets.report_usage"
	ScopeTicketsUpdateSelf  Scope = "tickets.update.self"
	ScopeProjectsUpdate     Scope = "projects.update"
	ScopeProjectsAddRepo    Scope = "projects.add_repo"
)

type PrincipalKind string

const (
	PrincipalKindTicketAgent         PrincipalKind = "ticket_agent"
	PrincipalKindProjectConversation PrincipalKind = "project_conversation"
)

type ScopeSet []Scope

type ScopeWhitelist struct {
	Configured bool
	Scopes     []string
}

type IssueInput struct {
	PrincipalKind  PrincipalKind
	PrincipalID    uuid.UUID
	PrincipalName  string
	AgentID        uuid.UUID
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	ConversationID uuid.UUID
	Scopes         []string
	ScopeWhitelist ScopeWhitelist
	TTL            time.Duration
}

type IssuedToken struct {
	Token          string
	PrincipalKind  PrincipalKind
	PrincipalID    uuid.UUID
	PrincipalName  string
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	ConversationID uuid.UUID
	Scopes         []string
	ExpiresAt      time.Time
}

type Claims struct {
	TokenID        uuid.UUID
	PrincipalKind  PrincipalKind
	PrincipalID    uuid.UUID
	PrincipalName  string
	AgentID        uuid.UUID
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	ConversationID uuid.UUID
	Scopes         []string
	ExpiresAt      time.Time
}

type ProjectTokenInventory struct {
	ActiveTokenCount  int
	ExpiredTokenCount int
	LastIssuedAt      *time.Time
	LastUsedAt        *time.Time
	DefaultScopes     []string
	PrivilegedScopes  []string
}

type AgentPrincipal struct {
	ID        uuid.UUID
	Name      string
	ProjectID uuid.UUID
}

type ProjectConversationPrincipal struct {
	ID             uuid.UUID
	Name           string
	ProjectID      uuid.UUID
	ConversationID uuid.UUID
}

type CreateTokenRecord struct {
	AgentID        *uuid.UUID
	ProjectID      uuid.UUID
	TicketID       *uuid.UUID
	ConversationID *uuid.UUID
	PrincipalKind  PrincipalKind
	PrincipalID    uuid.UUID
	PrincipalName  string
	TokenHash      string
	Scopes         []string
	ExpiresAt      time.Time
}

type StoredTokenRecord struct {
	TokenID        uuid.UUID
	AgentID        *uuid.UUID
	ProjectID      uuid.UUID
	TicketID       *uuid.UUID
	ConversationID *uuid.UUID
	PrincipalKind  PrincipalKind
	PrincipalID    uuid.UUID
	PrincipalName  string
	Scopes         []string
	ExpiresAt      time.Time
	LastUsedAt     *time.Time
}

func (c Claims) HasScope(scope Scope) bool {
	for _, item := range c.Scopes {
		if item == string(scope) {
			return true
		}
	}
	return false
}

func (c Claims) CreatedBy() string {
	switch c.PrincipalKind {
	case PrincipalKindProjectConversation:
		if strings.HasPrefix(c.PrincipalName, "project-conversation:") {
			return c.PrincipalName
		}
		return "project-conversation:" + c.PrincipalName
	default:
		return "agent:" + c.PrincipalName
	}
}

func (c Claims) IsTicketAgent() bool {
	return c.PrincipalKind == PrincipalKindTicketAgent
}

func (c Claims) IsProjectConversation() bool {
	return c.PrincipalKind == PrincipalKindProjectConversation
}

func BuildEnvironment(apiURL string, token string, projectID uuid.UUID, ticketID uuid.UUID) []string {
	environment := []string{
		"OPENASE_PROJECT_ID=" + projectID.String(),
		"OPENASE_TICKET_ID=" + ticketID.String(),
	}
	if strings.TrimSpace(apiURL) != "" {
		environment = append(environment, "OPENASE_API_URL="+strings.TrimSpace(apiURL))
	}
	if strings.TrimSpace(token) != "" {
		environment = append(environment, "OPENASE_AGENT_TOKEN="+strings.TrimSpace(token))
	}
	return environment
}
