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
	ErrUnavailable     = errors.New("agent platform service unavailable")
	ErrTokenNotFound   = errors.New("agent token not found")
	ErrTokenExpired    = errors.New("agent token expired")
	ErrInvalidToken    = errors.New("agent token is invalid")
	ErrInvalidScope    = errors.New("agent token scope is invalid")
	ErrAgentNotFound   = errors.New("agent not found")
	ErrProjectMismatch = errors.New("agent token project mismatch")

	defaultAgentScopes = []Scope{
		ScopeTicketsCreate,
		ScopeTicketsList,
		ScopeTicketsReportUsage,
		ScopeTicketsUpdateSelf,
	}
	supportedAgentScopes = []Scope{
		ScopeProjectsAddRepo,
		ScopeProjectsUpdate,
		ScopeTicketsCreate,
		ScopeTicketsList,
		ScopeTicketsReportUsage,
		ScopeTicketsUpdateSelf,
	}
)

type Scope string

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

	scopes, err := parseScopes(input.Scopes)
	if err != nil {
		return IssuedToken{}, err
	}
	scopes, err = constrainScopes(scopes, input.ScopeWhitelist)
	if err != nil {
		return IssuedToken{}, err
	}
	if input.AgentID == uuid.Nil {
		return IssuedToken{}, fmt.Errorf("agent_id must be a valid UUID")
	}
	if input.ProjectID == uuid.Nil {
		return IssuedToken{}, fmt.Errorf("project_id must be a valid UUID")
	}
	if input.TicketID == uuid.Nil {
		return IssuedToken{}, fmt.Errorf("ticket_id must be a valid UUID")
	}

	agentItem, err := s.client.Agent.Query().
		Where(entagent.IDEQ(input.AgentID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return IssuedToken{}, ErrAgentNotFound
		}
		return IssuedToken{}, fmt.Errorf("get agent for token issue: %w", err)
	}
	if agentItem.ProjectID != input.ProjectID {
		return IssuedToken{}, ErrProjectMismatch
	}

	expiresAt := s.now().UTC().Add(resolveTTL(input.TTL))
	rawToken, tokenHash, err := generateToken(s.rng)
	if err != nil {
		return IssuedToken{}, err
	}

	if _, err := s.client.AgentToken.Create().
		SetAgentID(input.AgentID).
		SetProjectID(input.ProjectID).
		SetTicketID(input.TicketID).
		SetTokenHash(tokenHash).
		SetScopes(scopeStrings(scopes)).
		SetExpiresAt(expiresAt).
		Save(ctx); err != nil {
		return IssuedToken{}, fmt.Errorf("create agent token: %w", err)
	}

	return IssuedToken{
		Token:     rawToken,
		ProjectID: input.ProjectID,
		TicketID:  input.TicketID,
		Scopes:    scopeStrings(scopes),
		ExpiresAt: expiresAt,
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
		WithAgent().
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

	agentItem := record.Edges.Agent
	if agentItem == nil {
		return Claims{}, ErrAgentNotFound
	}
	if agentItem.ProjectID != record.ProjectID {
		return Claims{}, ErrProjectMismatch
	}

	return Claims{
		TokenID:   record.ID,
		AgentID:   record.AgentID,
		AgentName: agentItem.Name,
		ProjectID: record.ProjectID,
		TicketID:  record.TicketID,
		Scopes:    scopeStrings(scopes),
		ExpiresAt: record.ExpiresAt.UTC(),
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

func DefaultScopes() []string {
	return scopeStrings(defaultAgentScopes)
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

func parseScopes(raw []string) (ScopeSet, error) {
	if len(raw) == 0 {
		return append(ScopeSet(nil), defaultAgentScopes...), nil
	}

	return parseExplicitScopes(raw)
}

func parseExplicitScopes(raw []string) (ScopeSet, error) {
	parsed := make(ScopeSet, 0, len(raw))
	for _, item := range raw {
		scope := Scope(strings.TrimSpace(item))
		if scope == "" {
			return nil, ErrInvalidScope
		}
		if !slices.Contains(supportedAgentScopes, scope) {
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
	if !whitelist.Configured {
		return requested, nil
	}

	allowed, err := parseExplicitScopes(whitelist.Scopes)
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

func timePointer(value time.Time) *time.Time {
	return &value
}
