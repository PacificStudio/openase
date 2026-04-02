package agentplatform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagenttoken "github.com/BetterAndBetterII/openase/ent/agenttoken"
	entprojectconversationprincipal "github.com/BetterAndBetterII/openase/ent/projectconversationprincipal"
	"github.com/google/uuid"
)

const (
	TokenPrefix = "ase_agent_"

	ScopeTicketsCreate      Scope = "tickets.create"
	ScopeTicketsList        Scope = "tickets.list"
	ScopeTicketsReportUsage Scope = "tickets.report_usage"
	ScopeTicketsUpdateSelf  Scope = "tickets.update.self"
	ScopeProjectsUpdate     Scope = "projects.update"
	ScopeProjectsAddRepo    Scope = "projects.add_repo"
)

var (
	ErrUnavailable       = errors.New("agent platform service unavailable")
	ErrTokenNotFound     = errors.New("agent token not found")
	ErrTokenExpired      = errors.New("agent token expired")
	ErrInvalidToken      = errors.New("agent token is invalid")
	ErrInvalidScope      = errors.New("agent token scope is invalid")
	ErrInvalidPrincipal  = errors.New("agent token principal is invalid")
	ErrAgentNotFound     = errors.New("agent not found")
	ErrPrincipalNotFound = errors.New("principal not found")
	ErrProjectMismatch   = errors.New("agent token project mismatch")

	defaultAgentScopes = []Scope{
		ScopeTicketsCreate,
		ScopeTicketsList,
		ScopeTicketsReportUsage,
		ScopeTicketsUpdateSelf,
	}
	defaultProjectConversationScopes = []Scope{
		ScopeProjectsUpdate,
		ScopeTicketsCreate,
		ScopeTicketsList,
	}
	supportedAgentScopes = []Scope{
		ScopeProjectsAddRepo,
		ScopeProjectsUpdate,
		ScopeTicketsCreate,
		ScopeTicketsList,
		ScopeTicketsReportUsage,
		ScopeTicketsUpdateSelf,
	}
	supportedProjectConversationScopes = []Scope{
		ScopeProjectsUpdate,
		ScopeTicketsCreate,
		ScopeTicketsList,
	}
)

type Scope string

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

type Service struct {
	client *ent.Client
	now    func() time.Time
	rng    io.Reader
}

func NewService(client *ent.Client) *Service {
	return &Service{
		client: client,
		now:    time.Now,
		rng:    rand.Reader,
	}
}

func (s *Service) IssueToken(ctx context.Context, input IssueInput) (IssuedToken, error) {
	if s == nil || s.client == nil {
		return IssuedToken{}, ErrUnavailable
	}

	resolved, err := s.resolvePrincipalIssueInput(ctx, input)
	if err != nil {
		return IssuedToken{}, err
	}

	scopes, err := parseScopesForPrincipalKind(resolved.PrincipalKind, resolved.Scopes)
	if err != nil {
		return IssuedToken{}, err
	}
	scopes, err = constrainScopesForPrincipalKind(resolved.PrincipalKind, scopes, resolved.ScopeWhitelist)
	if err != nil {
		return IssuedToken{}, err
	}

	expiresAt := s.now().UTC().Add(resolveTTL(input.TTL))
	rawToken, tokenHash, err := generateToken(s.rng)
	if err != nil {
		return IssuedToken{}, err
	}

	builder := s.client.AgentToken.Create().
		SetProjectID(resolved.ProjectID).
		SetPrincipalKind(entagenttoken.PrincipalKind(resolved.PrincipalKind)).
		SetPrincipalID(resolved.PrincipalID).
		SetPrincipalName(resolved.PrincipalName).
		SetTokenHash(tokenHash).
		SetScopes(scopeStrings(scopes)).
		SetExpiresAt(expiresAt)
	if resolved.AgentID != uuid.Nil {
		builder.SetAgentID(resolved.AgentID)
	}
	if resolved.TicketID != uuid.Nil {
		builder.SetTicketID(resolved.TicketID)
	}
	if resolved.ConversationID != uuid.Nil {
		builder.SetConversationID(resolved.ConversationID)
	}
	if _, err := builder.Save(ctx); err != nil {
		return IssuedToken{}, fmt.Errorf("create agent token: %w", err)
	}

	return IssuedToken{
		Token:          rawToken,
		PrincipalKind:  resolved.PrincipalKind,
		PrincipalID:    resolved.PrincipalID,
		PrincipalName:  resolved.PrincipalName,
		ProjectID:      resolved.ProjectID,
		TicketID:       resolved.TicketID,
		ConversationID: resolved.ConversationID,
		Scopes:         scopeStrings(scopes),
		ExpiresAt:      expiresAt,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (Claims, error) {
	if s == nil || s.client == nil {
		return Claims{}, ErrUnavailable
	}

	tokenText, err := ParseToken(rawToken)
	if err != nil {
		return Claims{}, err
	}

	record, err := s.client.AgentToken.Query().
		Where(entagenttoken.TokenHashEQ(hashToken(tokenText))).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return Claims{}, ErrTokenNotFound
		}
		return Claims{}, fmt.Errorf("load agent token: %w", err)
	}
	if record.ExpiresAt.Before(s.now().UTC()) {
		return Claims{}, ErrTokenExpired
	}

	scopes, err := parseScopes(record.Scopes)
	if err != nil {
		return Claims{}, err
	}
	if _, err := s.client.AgentToken.UpdateOneID(record.ID).SetLastUsedAt(s.now().UTC()).Save(ctx); err != nil {
		return Claims{}, fmt.Errorf("touch agent token last_used_at: %w", err)
	}

	kind := PrincipalKind(record.PrincipalKind)
	switch kind {
	case PrincipalKindTicketAgent:
		if record.AgentID == nil || *record.AgentID == uuid.Nil {
			return Claims{}, ErrInvalidPrincipal
		}
		agentItem, loadErr := s.client.Agent.Query().
			Where(entagent.IDEQ(*record.AgentID)).
			Only(ctx)
		if loadErr != nil {
			if ent.IsNotFound(loadErr) {
				return Claims{}, ErrAgentNotFound
			}
			return Claims{}, fmt.Errorf("get agent for token auth: %w", loadErr)
		}
		if agentItem.ProjectID != record.ProjectID {
			return Claims{}, ErrProjectMismatch
		}
	case PrincipalKindProjectConversation:
		if record.ConversationID == nil || *record.ConversationID == uuid.Nil {
			return Claims{}, ErrInvalidPrincipal
		}
		principalItem, loadErr := s.client.ProjectConversationPrincipal.Query().
			Where(entprojectconversationprincipal.IDEQ(record.PrincipalID)).
			Only(ctx)
		if loadErr != nil {
			if ent.IsNotFound(loadErr) {
				return Claims{}, ErrPrincipalNotFound
			}
			return Claims{}, fmt.Errorf("get project conversation principal for token auth: %w", loadErr)
		}
		if principalItem.ProjectID != record.ProjectID || principalItem.ConversationID != *record.ConversationID {
			return Claims{}, ErrProjectMismatch
		}
	default:
		return Claims{}, fmt.Errorf("%w: %s", ErrInvalidPrincipal, record.PrincipalKind)
	}

	return Claims{
		TokenID:        record.ID,
		PrincipalKind:  kind,
		PrincipalID:    record.PrincipalID,
		PrincipalName:  record.PrincipalName,
		AgentID:        uuidPointerValue(record.AgentID),
		ProjectID:      record.ProjectID,
		TicketID:       uuidPointerValue(record.TicketID),
		ConversationID: uuidPointerValue(record.ConversationID),
		Scopes:         scopeStrings(scopes),
		ExpiresAt:      record.ExpiresAt.UTC(),
	}, nil
}

func (s *Service) ProjectTokenInventory(ctx context.Context, projectID uuid.UUID) (ProjectTokenInventory, error) {
	if s == nil || s.client == nil {
		return ProjectTokenInventory{}, ErrUnavailable
	}
	if projectID == uuid.Nil {
		return ProjectTokenInventory{}, fmt.Errorf("project_id must be a valid UUID")
	}

	now := s.now().UTC()
	baseQuery := s.client.AgentToken.Query().Where(entagenttoken.ProjectIDEQ(projectID))

	activeTokenCount, err := baseQuery.Clone().Where(entagenttoken.ExpiresAtGTE(now)).Count(ctx)
	if err != nil {
		return ProjectTokenInventory{}, fmt.Errorf("count active project tokens: %w", err)
	}

	expiredTokenCount, err := baseQuery.Clone().Where(entagenttoken.ExpiresAtLT(now)).Count(ctx)
	if err != nil {
		return ProjectTokenInventory{}, fmt.Errorf("count expired project tokens: %w", err)
	}

	var lastIssuedAt *time.Time
	lastIssuedToken, err := baseQuery.Clone().
		Order(entagenttoken.ByCreatedAt(entsql.OrderDesc())).
		First(ctx)
	switch {
	case ent.IsNotFound(err):
	case err != nil:
		return ProjectTokenInventory{}, fmt.Errorf("load latest project token issue: %w", err)
	default:
		lastIssuedAt = timePointer(lastIssuedToken.CreatedAt.UTC())
	}

	var lastUsedAt *time.Time
	lastUsedToken, err := baseQuery.Clone().
		Where(entagenttoken.LastUsedAtNotNil()).
		Order(entagenttoken.ByLastUsedAt(entsql.OrderDesc())).
		First(ctx)
	switch {
	case ent.IsNotFound(err):
	case err != nil:
		return ProjectTokenInventory{}, fmt.Errorf("load latest project token use: %w", err)
	default:
		lastUsedAt = timePointer(lastUsedToken.LastUsedAt.UTC())
	}

	return ProjectTokenInventory{
		ActiveTokenCount:  activeTokenCount,
		ExpiredTokenCount: expiredTokenCount,
		LastIssuedAt:      lastIssuedAt,
		LastUsedAt:        lastUsedAt,
		DefaultScopes:     DefaultScopes(),
		PrivilegedScopes:  PrivilegedScopes(),
	}, nil
}

func ParseToken(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, TokenPrefix) {
		return "", ErrInvalidToken
	}
	return trimmed, nil
}

func ParseBearerToken(header string) (string, error) {
	scheme, value, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", ErrInvalidToken
	}
	return ParseToken(value)
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

func DefaultScopes() []string {
	return scopeStrings(defaultAgentScopes)
}

func DefaultScopesForPrincipalKind(kind PrincipalKind) []string {
	switch kind {
	case PrincipalKindProjectConversation:
		return scopeStrings(defaultProjectConversationScopes)
	default:
		return scopeStrings(defaultAgentScopes)
	}
}

func SupportedScopes() []string {
	return scopeStrings(supportedAgentScopes)
}

func SupportedScopesForPrincipalKind(kind PrincipalKind) []string {
	switch kind {
	case PrincipalKindProjectConversation:
		return scopeStrings(supportedProjectConversationScopes)
	default:
		return scopeStrings(supportedAgentScopes)
	}
}

func PrivilegedScopes() []string {
	privileged := make([]string, 0, len(supportedAgentScopes))
	for _, scope := range supportedAgentScopes {
		if strings.HasPrefix(string(scope), "projects.") {
			privileged = append(privileged, string(scope))
		}
	}
	return privileged
}

func PrivilegedScopesForPrincipalKind(kind PrincipalKind) []string {
	supported := supportedScopesForPrincipalKind(kind)
	privileged := make([]string, 0, len(supported))
	for _, scope := range supported {
		if strings.HasPrefix(string(scope), "projects.") {
			privileged = append(privileged, string(scope))
		}
	}
	return privileged
}

func parseScopes(raw []string) (ScopeSet, error) {
	return parseScopesForPrincipalKind(PrincipalKindTicketAgent, raw)
}

func parseScopesForPrincipalKind(kind PrincipalKind, raw []string) (ScopeSet, error) {
	if len(raw) == 0 {
		return append(ScopeSet(nil), defaultScopesForPrincipalKind(kind)...), nil
	}

	return parseExplicitScopesForPrincipalKind(kind, raw)
}

func parseExplicitScopes(raw []string) (ScopeSet, error) {
	return parseExplicitScopesForPrincipalKind(PrincipalKindTicketAgent, raw)
}

func parseExplicitScopesForPrincipalKind(kind PrincipalKind, raw []string) (ScopeSet, error) {
	parsed := make(ScopeSet, 0, len(raw))
	supported := supportedScopesForPrincipalKind(kind)
	for _, item := range raw {
		scope := Scope(strings.TrimSpace(item))
		if scope == "" {
			return nil, ErrInvalidScope
		}
		if !slices.Contains(supported, scope) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidScope, scope)
		}
		if !slices.Contains(parsed, scope) {
			parsed = append(parsed, scope)
		}
	}
	slices.Sort(parsed)
	return parsed, nil
}

func constrainScopes(requested ScopeSet, whitelist ScopeWhitelist) (ScopeSet, error) {
	return constrainScopesForPrincipalKind(PrincipalKindTicketAgent, requested, whitelist)
}

func constrainScopesForPrincipalKind(kind PrincipalKind, requested ScopeSet, whitelist ScopeWhitelist) (ScopeSet, error) {
	if !whitelist.Configured {
		return requested, nil
	}

	allowed, err := parseExplicitScopesForPrincipalKind(kind, whitelist.Scopes)
	if err != nil {
		return nil, err
	}

	constrained := make(ScopeSet, 0, len(requested))
	for _, scope := range requested {
		if slices.Contains(allowed, scope) {
			constrained = append(constrained, scope)
		}
	}
	return constrained, nil
}

func resolveTTL(raw time.Duration) time.Duration {
	if raw > 0 {
		return raw
	}
	return 24 * time.Hour
}

func generateToken(rng io.Reader) (string, string, error) {
	bytes := make([]byte, 24)
	if _, err := io.ReadFull(rng, bytes); err != nil {
		return "", "", fmt.Errorf("generate agent token bytes: %w", err)
	}

	token := TokenPrefix + base64.RawURLEncoding.EncodeToString(bytes)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func scopeStrings(scopes []Scope) []string {
	items := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		items = append(items, string(scope))
	}
	return items
}

func defaultScopesForPrincipalKind(kind PrincipalKind) ScopeSet {
	switch kind {
	case PrincipalKindProjectConversation:
		return append(ScopeSet(nil), defaultProjectConversationScopes...)
	default:
		return append(ScopeSet(nil), defaultAgentScopes...)
	}
}

func supportedScopesForPrincipalKind(kind PrincipalKind) ScopeSet {
	switch kind {
	case PrincipalKindProjectConversation:
		return append(ScopeSet(nil), supportedProjectConversationScopes...)
	default:
		return append(ScopeSet(nil), supportedAgentScopes...)
	}
}

func (s *Service) resolvePrincipalIssueInput(ctx context.Context, input IssueInput) (IssueInput, error) {
	resolved := input
	if resolved.PrincipalKind == "" {
		if resolved.AgentID != uuid.Nil {
			resolved.PrincipalKind = PrincipalKindTicketAgent
		} else {
			return IssueInput{}, fmt.Errorf("agent_id must be a valid UUID")
		}
	}
	if resolved.ProjectID == uuid.Nil {
		return IssueInput{}, fmt.Errorf("project_id must be a valid UUID")
	}

	switch resolved.PrincipalKind {
	case PrincipalKindTicketAgent:
		if resolved.AgentID == uuid.Nil && resolved.PrincipalID != uuid.Nil {
			resolved.AgentID = resolved.PrincipalID
		}
		if resolved.AgentID == uuid.Nil {
			return IssueInput{}, fmt.Errorf("agent_id must be a valid UUID")
		}
		if resolved.TicketID == uuid.Nil {
			return IssueInput{}, fmt.Errorf("ticket_id must be a valid UUID")
		}
		agentItem, err := s.client.Agent.Query().
			Where(entagent.IDEQ(resolved.AgentID)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return IssueInput{}, ErrAgentNotFound
			}
			return IssueInput{}, fmt.Errorf("get agent for token issue: %w", err)
		}
		if agentItem.ProjectID != resolved.ProjectID {
			return IssueInput{}, ErrProjectMismatch
		}
		resolved.PrincipalID = agentItem.ID
		resolved.PrincipalName = strings.TrimSpace(agentItem.Name)
	case PrincipalKindProjectConversation:
		if resolved.PrincipalID == uuid.Nil {
			return IssueInput{}, fmt.Errorf("principal_id must be a valid UUID")
		}
		if resolved.ConversationID == uuid.Nil {
			resolved.ConversationID = resolved.PrincipalID
		}
		principalItem, err := s.client.ProjectConversationPrincipal.Query().
			Where(entprojectconversationprincipal.IDEQ(resolved.PrincipalID)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return IssueInput{}, ErrPrincipalNotFound
			}
			return IssueInput{}, fmt.Errorf("get project conversation principal for token issue: %w", err)
		}
		if principalItem.ProjectID != resolved.ProjectID || principalItem.ConversationID != resolved.ConversationID {
			return IssueInput{}, ErrProjectMismatch
		}
		resolved.PrincipalName = strings.TrimSpace(principalItem.Name)
		resolved.AgentID = uuid.Nil
		resolved.TicketID = uuid.Nil
	default:
		return IssueInput{}, fmt.Errorf("%w: %s", ErrInvalidPrincipal, resolved.PrincipalKind)
	}

	if strings.TrimSpace(resolved.PrincipalName) == "" {
		return IssueInput{}, fmt.Errorf("principal_name must not be empty")
	}
	return resolved, nil
}

func uuidPointerValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

func timePointer(value time.Time) *time.Time {
	return &value
}
