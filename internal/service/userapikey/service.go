package userapikey

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

	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	domain "github.com/BetterAndBetterII/openase/internal/domain/userapikey"
	userapikeyrepo "github.com/BetterAndBetterII/openase/internal/repo/userapikey"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/google/uuid"
)

var (
	ErrUnavailable    = errors.New("user api key service unavailable")
	ErrInvalidInput   = errors.New("user api key input is invalid")
	ErrNotFound       = errors.New("user api key not found")
	ErrForbidden      = errors.New("user api key access is forbidden")
	ErrScopeForbidden = errors.New("user api key scope is forbidden")
)

type Service struct {
	repo       *userapikeyrepo.Repository
	authorizer *humanauthservice.Authorizer
	now        func() time.Time
	rng        io.Reader
}

func NewService(repo *userapikeyrepo.Repository, authorizer *humanauthservice.Authorizer) *Service {
	return &Service{repo: repo, authorizer: authorizer, now: time.Now, rng: rand.Reader}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID, principal humanauthdomain.AuthenticatedPrincipal) ([]domain.APIKey, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	if projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: project_id must be a valid UUID", ErrInvalidInput)
	}
	if principal.User.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: principal user is required", ErrForbidden)
	}
	return s.repo.ListByProjectAndUser(ctx, projectID, principal.User.ID)
}

func (s *Service) AllowedScopes(ctx context.Context, projectID uuid.UUID, principal *humanauthdomain.AuthenticatedPrincipal) ([]string, []agentplatformdomain.ScopeGroup, error) {
	requestedSupported := agentplatformdomain.SupportedScopesForPrincipalKind(agentplatformdomain.PrincipalKindUserAPIKey)
	if principal == nil {
		return requestedSupported, domain.SupportedScopeGroups(requestedSupported), nil
	}
	if s == nil || s.authorizer == nil {
		return nil, nil, ErrUnavailable
	}
	if projectID == uuid.Nil {
		return nil, nil, fmt.Errorf("%w: project_id must be a valid UUID", ErrInvalidInput)
	}
	roles, permissions, err := s.authorizer.Evaluate(
		ctx,
		principal.User,
		principal.Identity,
		principal.Groups,
		humanauthdomain.ScopeRef{Kind: humanauthdomain.ScopeKindProject, ID: projectID.String()},
	)
	_ = roles
	if err != nil {
		return nil, nil, err
	}
	allowed := scopesForPermissions(permissions)
	allowed = intersectStrings(allowed, requestedSupported)
	return allowed, domain.SupportedScopeGroups(allowed), nil
}

func (s *Service) Create(ctx context.Context, principal humanauthdomain.AuthenticatedPrincipal, input domain.CreateInput) (domain.CreateResult, error) {
	if s == nil || s.repo == nil {
		return domain.CreateResult{}, ErrUnavailable
	}
	if principal.User.ID == uuid.Nil || principal.User.ID != input.UserID {
		return domain.CreateResult{}, ErrForbidden
	}
	allowedScopes, _, err := s.AllowedScopes(ctx, input.ProjectID, &principal)
	if err != nil {
		return domain.CreateResult{}, err
	}
	requested, err := parseSupportedUserAPIKeyScopes(input.Scopes)
	if err != nil {
		return domain.CreateResult{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if len(requested) == 0 {
		return domain.CreateResult{}, fmt.Errorf("%w: at least one scope must be selected", ErrInvalidInput)
	}
	if !isSubset(requested, allowedScopes) {
		return domain.CreateResult{}, ErrScopeForbidden
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(s.now().UTC()) {
		return domain.CreateResult{}, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidInput)
	}
	plainTextToken, tokenHash, err := generateOpaqueToken(agentplatformdomain.UserAPIKeyTokenPrefix, s.rng)
	if err != nil {
		return domain.CreateResult{}, err
	}
	created, err := s.repo.Create(ctx, userapikeyrepo.CreateRecord{
		UserID:      input.UserID,
		ProjectID:   input.ProjectID,
		Name:        input.Name,
		TokenPrefix: agentplatformdomain.UserAPIKeyTokenPrefix,
		TokenHint:   tokenPreview(plainTextToken),
		TokenHash:   tokenHash,
		Scopes:      requested,
		ExpiresAt:   input.ExpiresAt,
	})
	if err != nil {
		return domain.CreateResult{}, err
	}
	return domain.CreateResult{APIKey: created, PlainTextToken: plainTextToken}, nil
}

func (s *Service) Disable(ctx context.Context, projectID, keyID uuid.UUID, principal humanauthdomain.AuthenticatedPrincipal) (domain.APIKey, error) {
	item, err := s.requireOwnedKey(ctx, projectID, keyID, principal)
	if err != nil {
		return domain.APIKey{}, err
	}
	if item.Status == domain.StatusRevoked {
		return domain.APIKey{}, ErrNotFound
	}
	if item.Status == domain.StatusDisabled {
		return item, nil
	}
	return s.repo.Disable(ctx, keyID, s.now().UTC())
}

func (s *Service) Rotate(ctx context.Context, projectID, keyID uuid.UUID, principal humanauthdomain.AuthenticatedPrincipal) (domain.CreateResult, error) {
	item, err := s.requireOwnedKey(ctx, projectID, keyID, principal)
	if err != nil {
		return domain.CreateResult{}, err
	}
	if item.Status == domain.StatusRevoked {
		return domain.CreateResult{}, ErrNotFound
	}
	allowedScopes, _, err := s.AllowedScopes(ctx, projectID, &principal)
	if err != nil {
		return domain.CreateResult{}, err
	}
	if !isSubset(item.Scopes, allowedScopes) {
		return domain.CreateResult{}, ErrScopeForbidden
	}
	plainTextToken, tokenHash, err := generateOpaqueToken(agentplatformdomain.UserAPIKeyTokenPrefix, s.rng)
	if err != nil {
		return domain.CreateResult{}, err
	}
	rotated, err := s.repo.Rotate(ctx, keyID, userapikeyrepo.RotateRecord{
		Name:        item.Name,
		TokenPrefix: agentplatformdomain.UserAPIKeyTokenPrefix,
		TokenHint:   tokenPreview(plainTextToken),
		TokenHash:   tokenHash,
		Scopes:      item.Scopes,
		ExpiresAt:   item.ExpiresAt,
		RotatedAt:   s.now().UTC(),
	})
	if err != nil {
		return domain.CreateResult{}, err
	}
	return domain.CreateResult{APIKey: rotated, PlainTextToken: plainTextToken}, nil
}

func (s *Service) Delete(ctx context.Context, projectID, keyID uuid.UUID, principal humanauthdomain.AuthenticatedPrincipal) error {
	item, err := s.requireOwnedKey(ctx, projectID, keyID, principal)
	if err != nil {
		return err
	}
	if item.Status == domain.StatusRevoked {
		return nil
	}
	if err := s.repo.Revoke(ctx, keyID, s.now().UTC()); err != nil {
		if errors.Is(err, userapikeyrepo.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) requireOwnedKey(ctx context.Context, projectID, keyID uuid.UUID, principal humanauthdomain.AuthenticatedPrincipal) (domain.APIKey, error) {
	if s == nil || s.repo == nil {
		return domain.APIKey{}, ErrUnavailable
	}
	if projectID == uuid.Nil || keyID == uuid.Nil || principal.User.ID == uuid.Nil {
		return domain.APIKey{}, ErrInvalidInput
	}
	item, err := s.repo.GetByIDForUser(ctx, keyID, projectID, principal.User.ID)
	if err != nil {
		if errors.Is(err, userapikeyrepo.ErrNotFound) {
			return domain.APIKey{}, ErrNotFound
		}
		return domain.APIKey{}, err
	}
	return item, nil
}

func scopesForPermissions(permissions []humanauthdomain.PermissionKey) []string {
	mapped := make([]string, 0)
	for _, permission := range permissions {
		switch permission {
		case humanauthdomain.PermissionProjectRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeActivityRead))
		case humanauthdomain.PermissionProjectUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeProjectsUpdate))
		case humanauthdomain.PermissionProjectUpdateRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeProjectUpdatesRead))
		case humanauthdomain.PermissionProjectUpdateCreate,
			humanauthdomain.PermissionProjectUpdateUpdate,
			humanauthdomain.PermissionProjectUpdateDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeProjectUpdatesWrite))
		case humanauthdomain.PermissionRepoRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeReposRead))
		case humanauthdomain.PermissionRepoCreate:
			mapped = append(mapped, string(agentplatformdomain.ScopeReposCreate))
		case humanauthdomain.PermissionRepoUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeReposUpdate))
		case humanauthdomain.PermissionRepoDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeReposDelete))
		case humanauthdomain.PermissionStatusRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeStatusesList))
		case humanauthdomain.PermissionStatusCreate:
			mapped = append(mapped, string(agentplatformdomain.ScopeStatusesCreate))
		case humanauthdomain.PermissionStatusUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeStatusesUpdate))
		case humanauthdomain.PermissionStatusDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeStatusesDelete))
		case humanauthdomain.PermissionTicketRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeTicketsList))
		case humanauthdomain.PermissionTicketCreate:
			mapped = append(mapped, string(agentplatformdomain.ScopeTicketsCreate))
		case humanauthdomain.PermissionTicketUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeTicketsUpdate))
		case humanauthdomain.PermissionWorkflowRead:
			mapped = append(mapped,
				string(agentplatformdomain.ScopeWorkflowsList),
				string(agentplatformdomain.ScopeWorkflowsRead),
			)
		case humanauthdomain.PermissionWorkflowCreate:
			mapped = append(mapped, string(agentplatformdomain.ScopeWorkflowsCreate))
		case humanauthdomain.PermissionWorkflowUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeWorkflowsUpdate))
		case humanauthdomain.PermissionWorkflowDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeWorkflowsDelete))
		case humanauthdomain.PermissionHarnessRead:
			mapped = append(mapped,
				string(agentplatformdomain.ScopeWorkflowsHarnessHistoryRead),
				string(agentplatformdomain.ScopeWorkflowsHarnessRead),
				string(agentplatformdomain.ScopeWorkflowsHarnessVariablesRead),
			)
		case humanauthdomain.PermissionHarnessUpdate:
			mapped = append(mapped,
				string(agentplatformdomain.ScopeWorkflowsHarnessUpdate),
				string(agentplatformdomain.ScopeWorkflowsHarnessValidate),
			)
		case humanauthdomain.PermissionSkillRead:
			mapped = append(mapped,
				string(agentplatformdomain.ScopeSkillsList),
				string(agentplatformdomain.ScopeSkillsRead),
			)
		case humanauthdomain.PermissionSkillCreate:
			mapped = append(mapped,
				string(agentplatformdomain.ScopeSkillsCreate),
				string(agentplatformdomain.ScopeSkillsImport),
			)
		case humanauthdomain.PermissionSkillUpdate:
			mapped = append(mapped,
				string(agentplatformdomain.ScopeSkillsRefresh),
				string(agentplatformdomain.ScopeSkillsUpdate),
			)
		case humanauthdomain.PermissionSkillDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeSkillsDelete))
		case humanauthdomain.PermissionAgentRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeAgentsRead))
		case humanauthdomain.PermissionAgentCreate:
			mapped = append(mapped, string(agentplatformdomain.ScopeAgentsCreate))
		case humanauthdomain.PermissionAgentUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeAgentsUpdate))
		case humanauthdomain.PermissionAgentDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeAgentsDelete))
		case humanauthdomain.PermissionAgentControl:
			mapped = append(mapped,
				string(agentplatformdomain.ScopeAgentsInterrupt),
				string(agentplatformdomain.ScopeAgentsPause),
				string(agentplatformdomain.ScopeAgentsResume),
			)
		case humanauthdomain.PermissionJobRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeScheduledJobsList))
		case humanauthdomain.PermissionJobCreate:
			mapped = append(mapped, string(agentplatformdomain.ScopeScheduledJobsCreate))
		case humanauthdomain.PermissionJobUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeScheduledJobsUpdate))
		case humanauthdomain.PermissionJobDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeScheduledJobsDelete))
		case humanauthdomain.PermissionJobTrigger:
			mapped = append(mapped, string(agentplatformdomain.ScopeScheduledJobsTrigger))
		case humanauthdomain.PermissionNotificationRead:
			mapped = append(mapped, string(agentplatformdomain.ScopeNotificationRulesList))
		case humanauthdomain.PermissionNotificationCreate:
			mapped = append(mapped, string(agentplatformdomain.ScopeNotificationRulesCreate))
		case humanauthdomain.PermissionNotificationUpdate:
			mapped = append(mapped, string(agentplatformdomain.ScopeNotificationRulesUpdate))
		case humanauthdomain.PermissionNotificationDelete:
			mapped = append(mapped, string(agentplatformdomain.ScopeNotificationRulesDelete))
		}
	}
	return uniqueStrings(mapped)
}

func uniqueStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || slices.Contains(result, trimmed) {
			continue
		}
		result = append(result, trimmed)
	}
	slices.Sort(result)
	return result
}

func intersectStrings(left, right []string) []string {
	result := make([]string, 0, len(left))
	for _, item := range left {
		if slices.Contains(right, item) {
			result = append(result, item)
		}
	}
	return uniqueStrings(result)
}

func isSubset(left, right []string) bool {
	for _, item := range left {
		if !slices.Contains(right, item) {
			return false
		}
	}
	return true
}

func parseSupportedUserAPIKeyScopes(raw []string) ([]string, error) {
	supported := agentplatformdomain.SupportedScopesForPrincipalKind(agentplatformdomain.PrincipalKindUserAPIKey)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		scope := strings.TrimSpace(item)
		if scope == "" {
			return nil, fmt.Errorf("scope must not be empty")
		}
		if !slices.Contains(supported, scope) {
			return nil, fmt.Errorf("unsupported scope %q", scope)
		}
		if !slices.Contains(result, scope) {
			result = append(result, scope)
		}
	}
	slices.Sort(result)
	return result, nil
}

func generateOpaqueToken(prefix string, rng io.Reader) (string, string, error) {
	bytes := make([]byte, 24)
	if _, err := io.ReadFull(rng, bytes); err != nil {
		return "", "", fmt.Errorf("generate user api key bytes: %w", err)
	}
	token := prefix + base64.RawURLEncoding.EncodeToString(bytes)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

func tokenPreview(token string) string {
	trimmed := strings.TrimSpace(token)
	if len(trimmed) <= 18 {
		return trimmed
	}
	return trimmed[:12] + "..." + trimmed[len(trimmed)-4:]
}
