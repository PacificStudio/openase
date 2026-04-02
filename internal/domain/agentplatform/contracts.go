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

type ScopeSet []Scope

type ScopeWhitelist struct {
	Configured bool
	Scopes     []string
}

type IssueInput struct {
	AgentID        uuid.UUID
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	Scopes         []string
	ScopeWhitelist ScopeWhitelist
	TTL            time.Duration
}

type IssuedToken struct {
	Token     string
	ProjectID uuid.UUID
	TicketID  uuid.UUID
	Scopes    []string
	ExpiresAt time.Time
}

type Claims struct {
	TokenID   uuid.UUID
	AgentID   uuid.UUID
	AgentName string
	ProjectID uuid.UUID
	TicketID  uuid.UUID
	Scopes    []string
	ExpiresAt time.Time
}

type ProjectTokenInventory struct {
	ActiveTokenCount  int
	ExpiredTokenCount int
	LastIssuedAt      *time.Time
	LastUsedAt        *time.Time
	DefaultScopes     []string
	PrivilegedScopes  []string
}

type StoredTokenRecord struct {
	TokenID        uuid.UUID
	AgentID        uuid.UUID
	AgentName      string
	AgentProjectID uuid.UUID
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
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
	return "agent:" + c.AgentName
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
