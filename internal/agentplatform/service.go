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

	domain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	"github.com/google/uuid"
)

const TokenPrefix = domain.TokenPrefix

const (
	ScopeTicketsCreate      = domain.ScopeTicketsCreate
	ScopeTicketsList        = domain.ScopeTicketsList
	ScopeTicketsReportUsage = domain.ScopeTicketsReportUsage
	ScopeTicketsUpdateSelf  = domain.ScopeTicketsUpdateSelf
	ScopeProjectsUpdate     = domain.ScopeProjectsUpdate
	ScopeProjectsAddRepo    = domain.ScopeProjectsAddRepo
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

type Scope = domain.Scope
type ScopeSet = domain.ScopeSet
type ScopeWhitelist = domain.ScopeWhitelist
type IssueInput = domain.IssueInput
type IssuedToken = domain.IssuedToken
type Claims = domain.Claims
type ProjectTokenInventory = domain.ProjectTokenInventory

type Service struct {
	repo Repository
	now  func() time.Time
	rng  io.Reader
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
		rng:  rand.Reader,
	}
}

func (s *Service) IssueToken(ctx context.Context, input IssueInput) (IssuedToken, error) {
	if s == nil || s.repo == nil {
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

	projectID, err := s.repo.AgentProjectID(ctx, input.AgentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return IssuedToken{}, ErrAgentNotFound
		}
		return IssuedToken{}, fmt.Errorf("get agent for token issue: %w", err)
	}
	if projectID != input.ProjectID {
		return IssuedToken{}, ErrProjectMismatch
	}

	expiresAt := s.now().UTC().Add(resolveTTL(input.TTL))
	rawToken, tokenHash, err := generateToken(s.rng)
	if err != nil {
		return IssuedToken{}, err
	}

	if err := s.repo.CreateToken(
		ctx,
		input.AgentID,
		input.ProjectID,
		input.TicketID,
		tokenHash,
		scopeStrings(scopes),
		expiresAt,
	); err != nil {
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
	if s == nil || s.repo == nil {
		return Claims{}, ErrUnavailable
	}

	tokenText, err := ParseToken(rawToken)
	if err != nil {
		return Claims{}, err
	}

	record, err := s.repo.TokenByHash(ctx, hashToken(tokenText))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
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
	if err := s.repo.TouchTokenLastUsed(ctx, record.TokenID, s.now().UTC()); err != nil {
		return Claims{}, fmt.Errorf("touch agent token last_used_at: %w", err)
	}
	if record.AgentID == uuid.Nil || strings.TrimSpace(record.AgentName) == "" {
		return Claims{}, ErrAgentNotFound
	}
	if record.ProjectID == uuid.Nil || record.AgentProjectID == uuid.Nil {
		return Claims{}, ErrProjectMismatch
	}
	if record.AgentProjectID != record.ProjectID {
		return Claims{}, ErrProjectMismatch
	}

	return Claims{
		TokenID:   record.TokenID,
		AgentID:   record.AgentID,
		AgentName: record.AgentName,
		ProjectID: record.ProjectID,
		TicketID:  record.TicketID,
		Scopes:    scopeStrings(scopes),
		ExpiresAt: record.ExpiresAt.UTC(),
	}, nil
}

func (s *Service) ProjectTokenInventory(ctx context.Context, projectID uuid.UUID) (ProjectTokenInventory, error) {
	if s == nil || s.repo == nil {
		return ProjectTokenInventory{}, ErrUnavailable
	}
	if projectID == uuid.Nil {
		return ProjectTokenInventory{}, fmt.Errorf("project_id must be a valid UUID")
	}

	inventory, err := s.repo.ProjectTokenInventory(ctx, projectID, s.now().UTC())
	if err != nil {
		return ProjectTokenInventory{}, err
	}
	inventory.DefaultScopes = DefaultScopes()
	inventory.PrivilegedScopes = PrivilegedScopes()
	return inventory, nil
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

func BuildEnvironment(apiURL string, token string, projectID uuid.UUID, ticketID uuid.UUID) []string {
	return domain.BuildEnvironment(apiURL, token, projectID, ticketID)
}

func DefaultScopes() []string {
	return scopeStrings(defaultAgentScopes)
}

func SupportedScopes() []string {
	return scopeStrings(supportedAgentScopes)
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
