package humanauth

import (
	"context"
	"sort"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/google/uuid"
)

type VisibilityResolver struct {
	repo *repo.Repository
}

func NewVisibilityResolver(repository *repo.Repository) *VisibilityResolver {
	return &VisibilityResolver{repo: repository}
}

func (r *VisibilityResolver) EffectiveVisibility(
	ctx context.Context,
	principal domain.AuthenticatedPrincipal,
) (domain.EffectiveVisibility, error) {
	if r == nil || r.repo == nil {
		return domain.EffectiveVisibility{}, nil
	}

	userKeys, groupKeys := principalSubjectKeys(principal.User, principal.Identity, principal.Groups)
	bindings, err := r.repo.ListSubjectRoleBindings(ctx, userKeys, groupKeys)
	if err != nil {
		return domain.EffectiveVisibility{}, err
	}

	now := time.Now().UTC()
	organizationIDs := map[uuid.UUID]struct{}{}
	organizationScopeIDs := map[uuid.UUID]struct{}{}
	projectIDs := map[uuid.UUID]struct{}{}
	visibility := domain.EffectiveVisibility{}

	for _, binding := range bindings {
		if binding.ExpiresAt != nil && now.After(binding.ExpiresAt.UTC()) {
			continue
		}
		switch binding.ScopeKind {
		case domain.ScopeKindInstance:
			visibility.Instance = true
		case domain.ScopeKindOrganization:
			orgID, err := uuid.Parse(strings.TrimSpace(binding.ScopeID))
			if err != nil {
				continue
			}
			organizationIDs[orgID] = struct{}{}
			organizationScopeIDs[orgID] = struct{}{}
		case domain.ScopeKindProject:
			projectID, err := uuid.Parse(strings.TrimSpace(binding.ScopeID))
			if err != nil {
				continue
			}
			projectIDs[projectID] = struct{}{}
			projectOrgID, err := r.repo.ResolveProjectOrganization(ctx, projectID)
			if err != nil {
				continue
			}
			organizationIDs[projectOrgID] = struct{}{}
		}
	}

	visibility.OrganizationIDs = sortedUUIDKeys(organizationIDs)
	visibility.OrganizationScopeIDs = sortedUUIDKeys(organizationScopeIDs)
	visibility.ProjectIDs = sortedUUIDKeys(projectIDs)
	return visibility, nil
}

func principalSubjectKeys(
	user domain.User,
	identity domain.UserIdentity,
	groups []domain.UserGroupMembership,
) ([]string, []string) {
	userKeys := []string{strings.ToLower(user.ID.String())}
	if email := strings.ToLower(strings.TrimSpace(user.PrimaryEmail)); email != "" {
		userKeys = append(userKeys, email)
	}
	if email := strings.ToLower(strings.TrimSpace(identity.Email)); email != "" {
		userKeys = append(userKeys, email)
	}
	groupKeys := make([]string, 0, len(groups))
	for _, group := range groups {
		groupKey := strings.ToLower(strings.TrimSpace(group.GroupKey))
		if groupKey == "" {
			continue
		}
		groupKeys = append(groupKeys, groupKey)
	}
	return uniqueStrings(userKeys), uniqueStrings(groupKeys)
}

func sortedUUIDKeys(items map[uuid.UUID]struct{}) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(items))
	for item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}
