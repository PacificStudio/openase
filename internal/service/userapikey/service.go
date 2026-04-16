package userapikey

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
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
	requestedSupported := agentplatform.SupportedScopesForPrincipalKind(agentplatform.PrincipalKindUserAPIKey)
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
	requested, err := agentplatform.ParseExplicitScopesForPrincipalKind(agentplatform.PrincipalKindUserAPIKey, input.Scopes)
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
	plainTextToken, tokenHash, err := agentplatform.GenerateOpaqueToken(agentplatform.UserAPIKeyTokenPrefix(), s.rng)
	if err != nil {
		return domain.CreateResult{}, err
	}
	created, err := s.repo.Create(ctx, userapikeyrepo.CreateRecord{
		UserID:      input.UserID,
		ProjectID:   input.ProjectID,
		Name:        input.Name,
		TokenPrefix: agentplatform.UserAPIKeyTokenPrefix(),
		TokenHint:   agentplatform.TokenPreview(plainTextToken),
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
	plainTextToken, tokenHash, err := agentplatform.GenerateOpaqueToken(agentplatform.UserAPIKeyTokenPrefix(), s.rng)
	if err != nil {
		return domain.CreateResult{}, err
	}
	rotated, err := s.repo.Rotate(ctx, keyID, userapikeyrepo.RotateRecord{
		Name:        item.Name,
		TokenPrefix: agentplatform.UserAPIKeyTokenPrefix(),
		TokenHint:   agentplatform.TokenPreview(plainTextToken),
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
			mapped = append(mapped, string(agentplatform.ScopeActivityRead))
		case humanauthdomain.PermissionProjectUpdateRead:
			mapped = append(mapped, string(agentplatform.ScopeProjectUpdatesRead))
		case humanauthdomain.PermissionProjectUpdateCreate,
			humanauthdomain.PermissionProjectUpdateUpdate,
			humanauthdomain.PermissionProjectUpdateDelete:
			mapped = append(mapped, string(agentplatform.ScopeProjectUpdatesWrite))
		case humanauthdomain.PermissionRepoRead:
			mapped = append(mapped, string(agentplatform.ScopeReposRead))
		case humanauthdomain.PermissionStatusRead:
			mapped = append(mapped, string(agentplatform.ScopeStatusesList))
		case humanauthdomain.PermissionTicketRead:
			mapped = append(mapped, string(agentplatform.ScopeTicketsList))
		case humanauthdomain.PermissionTicketCreate:
			mapped = append(mapped, string(agentplatform.ScopeTicketsCreate))
		case humanauthdomain.PermissionTicketUpdate:
			mapped = append(mapped, string(agentplatform.ScopeTicketsUpdate))
		case humanauthdomain.PermissionWorkflowRead, humanauthdomain.PermissionHarnessRead:
			mapped = append(mapped, string(agentplatform.ScopeWorkflowsRead))
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
