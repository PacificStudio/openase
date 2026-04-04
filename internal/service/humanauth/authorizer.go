package humanauth

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/google/uuid"
)

type Authorizer struct {
	repo *repo.Repository
}

func NewAuthorizer(repository *repo.Repository) *Authorizer {
	return &Authorizer{repo: repository}
}

func (a *Authorizer) Evaluate(
	ctx context.Context,
	user domain.User,
	identity domain.UserIdentity,
	groups []domain.UserGroupMembership,
	scope domain.ScopeRef,
) ([]domain.RoleKey, []domain.PermissionKey, error) {
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
	bindings, err := a.repo.ListSubjectRoleBindings(ctx, uniqueStrings(userKeys), uniqueStrings(groupKeys))
	if err != nil {
		return nil, nil, err
	}
	roles := make([]domain.RoleKey, 0, len(bindings))
	now := time.Now().UTC()
	projectOrgID := ""
	if scope.Kind == domain.ScopeKindProject && strings.TrimSpace(scope.ID) != "" {
		if parsed, err := uuid.Parse(scope.ID); err == nil {
			if orgID, err := a.repo.ResolveProjectOrganization(ctx, parsed); err == nil {
				projectOrgID = orgID.String()
			}
		}
	}
	roleSet := map[domain.RoleKey]struct{}{}
	for _, binding := range bindings {
		if binding.ExpiresAt != nil && now.After(binding.ExpiresAt.UTC()) {
			continue
		}
		switch binding.ScopeKind {
		case domain.ScopeKindInstance:
			roleSet[binding.RoleKey] = struct{}{}
		case domain.ScopeKindOrganization:
			if scope.Kind == domain.ScopeKindOrganization && binding.ScopeID == scope.ID {
				roleSet[binding.RoleKey] = struct{}{}
			}
			if scope.Kind == domain.ScopeKindProject && binding.ScopeID == projectOrgID {
				roleSet[binding.RoleKey] = struct{}{}
			}
		case domain.ScopeKindProject:
			if scope.Kind == domain.ScopeKindProject && binding.ScopeID == scope.ID {
				roleSet[binding.RoleKey] = struct{}{}
			}
		}
	}
	for role := range roleSet {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })
	return roles, domain.PermissionsForRoles(roles), nil
}

func (a *Authorizer) HasPermission(
	ctx context.Context,
	principal domain.AuthenticatedPrincipal,
	scope domain.ScopeRef,
	permission domain.PermissionKey,
) (bool, []domain.RoleKey, []domain.PermissionKey, error) {
	roles, permissions, err := a.Evaluate(ctx, principal.User, principal.Identity, principal.Groups, scope)
	if err != nil {
		return false, nil, nil, err
	}
	for _, candidate := range permissions {
		if candidate == permission {
			return true, roles, permissions, nil
		}
	}
	return false, roles, permissions, nil
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.ToLower(strings.TrimSpace(item))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (a *Authorizer) ResolveProjectScope(ctx context.Context, resource string, id uuid.UUID) (domain.ScopeRef, error) {
	switch resource {
	case "project":
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: id.String()}, nil
	case "ticket":
		projectID, err := a.repo.ResolveProjectFromTicket(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	case "workflow":
		projectID, err := a.repo.ResolveProjectFromWorkflow(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	case "skill":
		projectID, err := a.repo.ResolveProjectFromSkill(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	case "status":
		projectID, err := a.repo.ResolveProjectFromStatus(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	case "agent":
		projectID, err := a.repo.ResolveProjectFromAgent(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	case "scheduled_job":
		projectID, err := a.repo.ResolveProjectFromScheduledJob(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	case "notification_rule":
		projectID, err := a.repo.ResolveProjectFromNotificationRule(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	case "conversation":
		projectID, err := a.repo.ResolveProjectFromConversation(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindProject, ID: projectID.String()}, nil
	default:
		return domain.ScopeRef{}, fmt.Errorf("unsupported project-scoped resource %q", resource)
	}
}

func (a *Authorizer) ResolveOrganizationScope(ctx context.Context, resource string, id uuid.UUID) (domain.ScopeRef, error) {
	switch resource {
	case "organization":
		return domain.ScopeRef{Kind: domain.ScopeKindOrganization, ID: id.String()}, nil
	case "project":
		orgID, err := a.repo.ResolveProjectOrganization(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindOrganization, ID: orgID.String()}, nil
	case "machine":
		orgID, err := a.repo.ResolveOrganizationFromMachine(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindOrganization, ID: orgID.String()}, nil
	case "provider":
		orgID, err := a.repo.ResolveOrganizationFromProvider(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindOrganization, ID: orgID.String()}, nil
	case "channel":
		orgID, err := a.repo.ResolveOrganizationFromChannel(ctx, id)
		if err != nil {
			return domain.ScopeRef{}, err
		}
		return domain.ScopeRef{Kind: domain.ScopeKindOrganization, ID: orgID.String()}, nil
	default:
		return domain.ScopeRef{}, fmt.Errorf("unsupported organization-scoped resource %q", resource)
	}
}
