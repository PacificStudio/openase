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
	ScopeTicketsCreate                 = domain.ScopeTicketsCreate
	ScopeTicketsList                   = domain.ScopeTicketsList
	ScopeTicketsReportUsage            = domain.ScopeTicketsReportUsage
	ScopeTicketsUpdateSelf             = domain.ScopeTicketsUpdateSelf
	ScopeProjectsUpdate                = domain.ScopeProjectsUpdate
	ScopeProjectsAddRepo               = domain.ScopeProjectsAddRepo
	ScopeActivityRead                  = domain.ScopeActivityRead
	ScopeReposCreate                   = domain.ScopeReposCreate
	ScopeReposRead                     = domain.ScopeReposRead
	ScopeReposUpdate                   = domain.ScopeReposUpdate
	ScopeReposDelete                   = domain.ScopeReposDelete
	ScopeScheduledJobsList             = domain.ScopeScheduledJobsList
	ScopeScheduledJobsCreate           = domain.ScopeScheduledJobsCreate
	ScopeScheduledJobsUpdate           = domain.ScopeScheduledJobsUpdate
	ScopeScheduledJobsDelete           = domain.ScopeScheduledJobsDelete
	ScopeScheduledJobsTrigger          = domain.ScopeScheduledJobsTrigger
	ScopeSkillsList                    = domain.ScopeSkillsList
	ScopeSkillsRead                    = domain.ScopeSkillsRead
	ScopeSkillsCreate                  = domain.ScopeSkillsCreate
	ScopeSkillsImport                  = domain.ScopeSkillsImport
	ScopeSkillsRefresh                 = domain.ScopeSkillsRefresh
	ScopeSkillsUpdate                  = domain.ScopeSkillsUpdate
	ScopeSkillsDelete                  = domain.ScopeSkillsDelete
	ScopeSkillsEnable                  = domain.ScopeSkillsEnable
	ScopeSkillsDisable                 = domain.ScopeSkillsDisable
	ScopeSkillsBind                    = domain.ScopeSkillsBind
	ScopeSkillsRefine                  = domain.ScopeSkillsRefine
	ScopeStatusesList                  = domain.ScopeStatusesList
	ScopeStatusesCreate                = domain.ScopeStatusesCreate
	ScopeStatusesUpdate                = domain.ScopeStatusesUpdate
	ScopeStatusesDelete                = domain.ScopeStatusesDelete
	ScopeStatusesReset                 = domain.ScopeStatusesReset
	ScopeTicketRepoScopesList          = domain.ScopeTicketRepoScopesList
	ScopeTicketRepoScopesCreate        = domain.ScopeTicketRepoScopesCreate
	ScopeTicketRepoScopesUpdate        = domain.ScopeTicketRepoScopesUpdate
	ScopeTicketRepoScopesDelete        = domain.ScopeTicketRepoScopesDelete
	ScopeWorkflowsList                 = domain.ScopeWorkflowsList
	ScopeWorkflowsRead                 = domain.ScopeWorkflowsRead
	ScopeWorkflowsCreate               = domain.ScopeWorkflowsCreate
	ScopeWorkflowsUpdate               = domain.ScopeWorkflowsUpdate
	ScopeWorkflowsDelete               = domain.ScopeWorkflowsDelete
	ScopeWorkflowsHarnessRead          = domain.ScopeWorkflowsHarnessRead
	ScopeWorkflowsHarnessHistoryRead   = domain.ScopeWorkflowsHarnessHistoryRead
	ScopeWorkflowsHarnessUpdate        = domain.ScopeWorkflowsHarnessUpdate
	ScopeWorkflowsHarnessValidate      = domain.ScopeWorkflowsHarnessValidate
	ScopeWorkflowsHarnessVariablesRead = domain.ScopeWorkflowsHarnessVariablesRead
)

const (
	PrincipalKindTicketAgent         = domain.PrincipalKindTicketAgent
	PrincipalKindProjectConversation = domain.PrincipalKindProjectConversation
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
	supportedAgentScopes = []Scope{
		ScopeActivityRead,
		ScopeProjectsAddRepo,
		ScopeProjectsUpdate,
		ScopeReposCreate,
		ScopeReposDelete,
		ScopeReposRead,
		ScopeReposUpdate,
		ScopeScheduledJobsCreate,
		ScopeScheduledJobsDelete,
		ScopeScheduledJobsList,
		ScopeScheduledJobsTrigger,
		ScopeScheduledJobsUpdate,
		ScopeSkillsBind,
		ScopeSkillsCreate,
		ScopeSkillsDelete,
		ScopeSkillsDisable,
		ScopeSkillsEnable,
		ScopeSkillsImport,
		ScopeSkillsList,
		ScopeSkillsRead,
		ScopeSkillsRefine,
		ScopeSkillsRefresh,
		ScopeSkillsUpdate,
		ScopeStatusesCreate,
		ScopeStatusesDelete,
		ScopeStatusesList,
		ScopeStatusesReset,
		ScopeStatusesUpdate,
		ScopeTicketRepoScopesCreate,
		ScopeTicketRepoScopesDelete,
		ScopeTicketRepoScopesList,
		ScopeTicketRepoScopesUpdate,
		ScopeTicketsCreate,
		ScopeTicketsList,
		ScopeTicketsReportUsage,
		ScopeTicketsUpdateSelf,
		ScopeWorkflowsCreate,
		ScopeWorkflowsDelete,
		ScopeWorkflowsHarnessHistoryRead,
		ScopeWorkflowsHarnessRead,
		ScopeWorkflowsHarnessUpdate,
		ScopeWorkflowsHarnessValidate,
		ScopeWorkflowsHarnessVariablesRead,
		ScopeWorkflowsList,
		ScopeWorkflowsRead,
		ScopeWorkflowsUpdate,
	}
)

type Scope = domain.Scope
type PrincipalKind = domain.PrincipalKind
type ScopeSet = domain.ScopeSet
type ScopeWhitelist = domain.ScopeWhitelist
type ScopeGroup = domain.ScopeGroup
type IssueInput = domain.IssueInput
type IssuedToken = domain.IssuedToken
type Claims = domain.Claims
type ProjectTokenInventory = domain.ProjectTokenInventory
type RuntimeContractInput = domain.RuntimeContractInput

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

	record := domain.CreateTokenRecord{
		ProjectID:     resolved.ProjectID,
		PrincipalKind: resolved.PrincipalKind,
		PrincipalID:   resolved.PrincipalID,
		PrincipalName: resolved.PrincipalName,
		TokenHash:     tokenHash,
		Scopes:        scopeStrings(scopes),
		ExpiresAt:     expiresAt,
	}
	if resolved.AgentID != uuid.Nil {
		record.AgentID = uuidPointer(resolved.AgentID)
	}
	if resolved.TicketID != uuid.Nil {
		record.TicketID = uuidPointer(resolved.TicketID)
	}
	if resolved.ConversationID != uuid.Nil {
		record.ConversationID = uuidPointer(resolved.ConversationID)
	}
	if err := s.repo.CreateToken(ctx, record); err != nil {
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

	scopes, err := parseScopesForPrincipalKind(record.PrincipalKind, record.Scopes)
	if err != nil {
		return Claims{}, err
	}
	if err := s.repo.TouchTokenLastUsed(ctx, record.TokenID, s.now().UTC()); err != nil {
		return Claims{}, fmt.Errorf("touch agent token last_used_at: %w", err)
	}

	switch record.PrincipalKind {
	case PrincipalKindTicketAgent:
		if record.AgentID == nil || *record.AgentID == uuid.Nil {
			return Claims{}, ErrInvalidPrincipal
		}
		principal, err := s.repo.AgentPrincipal(ctx, *record.AgentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return Claims{}, ErrAgentNotFound
			}
			return Claims{}, fmt.Errorf("get agent for token auth: %w", err)
		}
		if principal.ProjectID != record.ProjectID {
			return Claims{}, ErrProjectMismatch
		}
	case PrincipalKindProjectConversation:
		if record.ConversationID == nil || *record.ConversationID == uuid.Nil {
			return Claims{}, ErrInvalidPrincipal
		}
		principal, err := s.repo.ProjectConversationPrincipal(ctx, record.PrincipalID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return Claims{}, ErrPrincipalNotFound
			}
			return Claims{}, fmt.Errorf("get project conversation principal for token auth: %w", err)
		}
		if principal.ProjectID != record.ProjectID || principal.ConversationID != *record.ConversationID {
			return Claims{}, ErrProjectMismatch
		}
	default:
		return Claims{}, fmt.Errorf("%w: %s", ErrInvalidPrincipal, record.PrincipalKind)
	}

	return Claims{
		TokenID:        record.TokenID,
		PrincipalKind:  record.PrincipalKind,
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

func BuildRuntimeEnvironment(input RuntimeContractInput) []string {
	return domain.BuildRuntimeEnvironment(input)
}

func BuildCapabilityContract(input RuntimeContractInput) string {
	return domain.BuildCapabilityContract(input)
}

func DefaultScopes() []string {
	return domain.DefaultAgentScopes()
}

func DefaultScopesForPrincipalKind(kind PrincipalKind) []string {
	return scopeStrings(defaultScopesForPrincipalKind(kind))
}

func SupportedScopes() []string {
	return scopeStrings(supportedAgentScopes)
}

func SupportedScopesForPrincipalKind(kind PrincipalKind) []string {
	return scopeStrings(supportedScopesForPrincipalKind(kind))
}

func PrivilegedScopes() []string {
	return privilegedScopesForPrincipalKind(PrincipalKindTicketAgent)
}

func PrivilegedScopesForPrincipalKind(kind PrincipalKind) []string {
	return privilegedScopesForPrincipalKind(kind)
}

func SupportedScopeGroups() []ScopeGroup {
	return scopeGroupsFor(supportedAgentScopes)
}

func SupportedScopeGroupsForPrincipalKind(kind PrincipalKind) []ScopeGroup {
	return scopeGroupsFor(supportedScopesForPrincipalKind(kind))
}

func parseScopes(raw []string) (ScopeSet, error) {
	return parseScopesForPrincipalKind(PrincipalKindTicketAgent, raw)
}

func parseScopesForPrincipalKind(kind PrincipalKind, raw []string) (ScopeSet, error) {
	if len(raw) == 0 {
		return defaultScopesForPrincipalKind(kind), nil
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

func scopeGroupsFor(scopes []Scope) []ScopeGroup {
	grouped := make(map[string][]string)
	for _, scope := range scopes {
		category, _, found := strings.Cut(string(scope), ".")
		if !found {
			category = string(scope)
		}
		grouped[category] = append(grouped[category], string(scope))
	}

	categories := make([]string, 0, len(grouped))
	for category := range grouped {
		categories = append(categories, category)
	}
	slices.Sort(categories)

	result := make([]ScopeGroup, 0, len(categories))
	for _, category := range categories {
		items := grouped[category]
		slices.Sort(items)
		result = append(result, ScopeGroup{
			Category: category,
			Scopes:   items,
		})
	}
	return result
}

func privilegedScopesForPrincipalKind(kind PrincipalKind) []string {
	supported := supportedScopesForPrincipalKind(kind)
	defaults := defaultScopesForPrincipalKind(kind)
	privileged := make([]string, 0, len(supported))
	for _, scope := range supported {
		if !slices.Contains(defaults, scope) {
			privileged = append(privileged, string(scope))
		}
	}
	slices.Sort(privileged)
	return privileged
}

func defaultScopesForPrincipalKind(kind PrincipalKind) ScopeSet {
	switch kind {
	case PrincipalKindProjectConversation:
		return append(ScopeSet(nil), supportedAgentScopes...)
	default:
		return append(ScopeSet(nil), defaultAgentScopes...)
	}
}

func supportedScopesForPrincipalKind(kind PrincipalKind) ScopeSet {
	switch kind {
	case PrincipalKindProjectConversation:
		return append(ScopeSet(nil), supportedAgentScopes...)
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
		principal, err := s.repo.AgentPrincipal(ctx, resolved.AgentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return IssueInput{}, ErrAgentNotFound
			}
			return IssueInput{}, fmt.Errorf("get agent for token issue: %w", err)
		}
		if principal.ProjectID != resolved.ProjectID {
			return IssueInput{}, ErrProjectMismatch
		}
		resolved.PrincipalID = principal.ID
		resolved.PrincipalName = strings.TrimSpace(principal.Name)
	case PrincipalKindProjectConversation:
		if resolved.PrincipalID == uuid.Nil {
			return IssueInput{}, fmt.Errorf("principal_id must be a valid UUID")
		}
		if resolved.ConversationID == uuid.Nil {
			resolved.ConversationID = resolved.PrincipalID
		}
		principal, err := s.repo.ProjectConversationPrincipal(ctx, resolved.PrincipalID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return IssueInput{}, ErrPrincipalNotFound
			}
			return IssueInput{}, fmt.Errorf("get project conversation principal for token issue: %w", err)
		}
		if principal.ProjectID != resolved.ProjectID || principal.ConversationID != resolved.ConversationID {
			return IssueInput{}, ErrProjectMismatch
		}
		resolved.PrincipalName = strings.TrimSpace(principal.Name)
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

func uuidPointer(value uuid.UUID) *uuid.UUID {
	return &value
}

func uuidPointerValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}
