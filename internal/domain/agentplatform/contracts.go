package agentplatform

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const TokenPrefix = "ase_agent_"

const (
	EnvAPIURL         = "OPENASE_API_URL"
	EnvAgentToken     = "OPENASE_AGENT_TOKEN" // #nosec G101 -- environment variable key name, not a credential
	EnvProjectID      = "OPENASE_PROJECT_ID"
	EnvTicketID       = "OPENASE_TICKET_ID"
	EnvConversationID = "OPENASE_CONVERSATION_ID"
	EnvPrincipalKind  = "OPENASE_PRINCIPAL_KIND"
	EnvAgentScopes    = "OPENASE_AGENT_SCOPES"
)

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

func DefaultAgentScopes() []string {
	return []string{
		string(ScopeTicketsCreate),
		string(ScopeTicketsList),
		string(ScopeTicketsReportUsage),
		string(ScopeTicketsUpdateSelf),
	}
}

func SupportedAgentScopes() []string {
	return []string{
		string(ScopeProjectsAddRepo),
		string(ScopeProjectsUpdate),
		string(ScopeTicketsCreate),
		string(ScopeTicketsList),
		string(ScopeTicketsReportUsage),
		string(ScopeTicketsUpdateSelf),
	}
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

type RuntimeContractInput struct {
	PrincipalKind  PrincipalKind
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	ConversationID uuid.UUID
	APIURL         string
	Token          string
	Scopes         []string
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
		EnvProjectID + "=" + projectID.String(),
		EnvTicketID + "=" + ticketID.String(),
	}
	if strings.TrimSpace(apiURL) != "" {
		environment = append(environment, EnvAPIURL+"="+strings.TrimSpace(apiURL))
	}
	if strings.TrimSpace(token) != "" {
		environment = append(environment, EnvAgentToken+"="+strings.TrimSpace(token))
	}
	return environment
}

func BuildRuntimeEnvironment(input RuntimeContractInput) []string {
	environment := []string{
		EnvProjectID + "=" + input.ProjectID.String(),
	}
	if input.TicketID != uuid.Nil {
		environment = append(environment, EnvTicketID+"="+input.TicketID.String())
	}
	if input.ConversationID != uuid.Nil {
		environment = append(environment, EnvConversationID+"="+input.ConversationID.String())
	}
	if strings.TrimSpace(input.APIURL) != "" {
		environment = append(environment, EnvAPIURL+"="+strings.TrimSpace(input.APIURL))
	}
	if strings.TrimSpace(input.Token) != "" {
		environment = append(environment, EnvAgentToken+"="+strings.TrimSpace(input.Token))
	}
	if strings.TrimSpace(string(input.PrincipalKind)) != "" {
		environment = append(environment, EnvPrincipalKind+"="+strings.TrimSpace(string(input.PrincipalKind)))
	}
	scopes := normalizedScopeStrings(input.Scopes)
	if len(scopes) > 0 {
		environment = append(environment, EnvAgentScopes+"="+strings.Join(scopes, ","))
	}
	return environment
}

func BuildCapabilityContract(input RuntimeContractInput) string {
	var builder strings.Builder
	builder.WriteString("## OpenASE Platform Capability Contract\n")
	builder.WriteString("\n")
	builder.WriteString("Current principal: `")
	builder.WriteString(strings.TrimSpace(string(input.PrincipalKind)))
	builder.WriteString("`\n")
	builder.WriteString("\n")
	builder.WriteString("Guaranteed environment:\n")
	builder.WriteString("- `OPENASE_PROJECT_ID`\n")
	if input.TicketID != uuid.Nil {
		builder.WriteString("- `OPENASE_TICKET_ID`\n")
	}
	if input.ConversationID != uuid.Nil {
		builder.WriteString("- `OPENASE_CONVERSATION_ID`\n")
	}
	if strings.TrimSpace(input.APIURL) != "" {
		builder.WriteString("- `OPENASE_API_URL`\n")
	}
	if strings.TrimSpace(input.Token) != "" {
		builder.WriteString("- `OPENASE_AGENT_TOKEN`\n")
	}
	builder.WriteString("- `OPENASE_PRINCIPAL_KIND`\n")
	if len(normalizedScopeStrings(input.Scopes)) > 0 {
		builder.WriteString("- `OPENASE_AGENT_SCOPES`\n")
	}

	if input.PrincipalKind == PrincipalKindProjectConversation && input.TicketID == uuid.Nil {
		builder.WriteString("\n")
		builder.WriteString("Optional environment:\n")
		builder.WriteString("- `OPENASE_TICKET_ID` only when this Project AI session is ticket-focused\n")
	}

	scopes := normalizedScopeStrings(input.Scopes)
	builder.WriteString("\n")
	builder.WriteString("Available scopes:\n")
	if len(scopes) == 0 {
		builder.WriteString("- none declared\n")
	} else {
		for _, scope := range scopes {
			builder.WriteString("- `")
			builder.WriteString(scope)
			builder.WriteString("`\n")
		}
	}

	builder.WriteString("\n")
	builder.WriteString("Constraints:\n")
	switch input.PrincipalKind {
	case PrincipalKindProjectConversation:
		builder.WriteString("- Treat this as a project-scoped conversation runtime, not a ticket runtime.\n")
		builder.WriteString("- Do not assume current-ticket comment/update/report-usage endpoints are available.\n")
		builder.WriteString("- Ticket-runtime-only routes can reject this principal kind even when `OPENASE_TICKET_ID` is present.\n")
	default:
		builder.WriteString("- Treat this as the current ticket runtime.\n")
		builder.WriteString("- Current-ticket routes are limited to the ticket identified by `OPENASE_TICKET_ID`.\n")
		builder.WriteString("- Project-level writes still depend on the scopes listed above.\n")
	}

	return strings.TrimSpace(builder.String())
}

func normalizedScopeStrings(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	scopes := make([]string, 0, len(raw))
	for _, item := range raw {
		scope := strings.TrimSpace(item)
		if scope == "" || slices.Contains(scopes, scope) {
			continue
		}
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	return scopes
}

func (i RuntimeContractInput) String() string {
	return fmt.Sprintf("principal=%s project=%s ticket=%s conversation=%s scopes=%s", i.PrincipalKind, i.ProjectID, i.TicketID, i.ConversationID, strings.Join(normalizedScopeStrings(i.Scopes), ","))
}
