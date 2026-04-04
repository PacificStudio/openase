package catalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	"github.com/google/uuid"
)

var (
	ErrNotFound     = domain.ErrNotFound
	ErrConflict     = domain.ErrConflict
	ErrInvalidInput = domain.ErrInvalidInput
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) ListOrganizations(ctx context.Context) ([]domain.Organization, error) {
	items, err := r.client.Organization.Query().
		Where(entorganization.StatusEQ(entorganization.StatusActive)).
		Order(entorganization.ByName()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return mapOrganizations(items), nil
}

func (r *EntRepository) CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("start create organization transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	builder := tx.Organization.Create().
		SetName(input.Name).
		SetSlug(input.Slug).
		SetStatus(toEntOrganizationStatus(domain.OrganizationStatusActive))

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Organization{}, mapWriteError("create organization", err)
	}

	localMachine, err := createLocalMachine(ctx, tx, item.ID)
	if err != nil {
		return domain.Organization{}, err
	}
	if err := createBuiltinAgentProviders(ctx, tx, item.ID, localMachine.ID); err != nil {
		return domain.Organization{}, err
	}
	if input.DefaultAgentProviderID != nil {
		if _, err := tx.Organization.UpdateOneID(item.ID).
			SetDefaultAgentProviderID(*input.DefaultAgentProviderID).
			Save(ctx); err != nil {
			return domain.Organization{}, mapWriteError("set organization default provider", err)
		}
		item.DefaultAgentProviderID = input.DefaultAgentProviderID
	}

	if err := tx.Commit(); err != nil {
		return domain.Organization{}, fmt.Errorf("commit create organization: %w", err)
	}

	return mapOrganization(item), nil
}

func (r *EntRepository) GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	item, err := r.getActiveOrganization(ctx, id)
	if err != nil {
		return domain.Organization{}, err
	}

	return mapOrganization(item), nil
}

func (r *EntRepository) UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	if _, err := r.getActiveOrganization(ctx, input.ID); err != nil {
		return domain.Organization{}, err
	}

	builder := r.client.Organization.UpdateOneID(input.ID).
		SetName(input.Name).
		SetSlug(input.Slug)
	if input.DefaultAgentProviderID != nil {
		builder.SetDefaultAgentProviderID(*input.DefaultAgentProviderID)
	} else {
		builder.ClearDefaultAgentProviderID()
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Organization{}, mapWriteError("update organization", err)
	}

	return mapOrganization(item), nil
}

func (r *EntRepository) ArchiveOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("start archive organization transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	if _, err := tx.Organization.Query().Where(activeOrganizationPredicates(id)...).Only(ctx); err != nil {
		return domain.Organization{}, mapReadError("get organization for archive", err)
	}

	if _, err := tx.Project.Update().
		Where(
			entproject.OrganizationID(id),
			entproject.StatusNEQ(string(domain.ProjectStatusArchived)),
		).
		SetStatus(toEntProjectStatus(domain.ProjectStatusArchived)).
		Save(ctx); err != nil {
		return domain.Organization{}, mapWriteError("archive organization projects", err)
	}

	item, err := tx.Organization.UpdateOneID(id).
		SetStatus(toEntOrganizationStatus(domain.OrganizationStatusArchived)).
		Save(ctx)
	if err != nil {
		return domain.Organization{}, mapWriteError("archive organization", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Organization{}, fmt.Errorf("commit archive organization: %w", err)
	}

	return mapOrganization(item), nil
}

func (r *EntRepository) ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error) {
	exists, err := r.organizationIsActive(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("check organization before listing projects: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	items, err := r.client.Project.Query().
		Where(entproject.OrganizationID(organizationID)).
		Order(entproject.ByName()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return mapProjects(items), nil
}

func (r *EntRepository) CreateProject(ctx context.Context, input domain.CreateProject) (domain.Project, error) {
	exists, err := r.organizationIsActive(ctx, input.OrganizationID)
	if err != nil {
		return domain.Project{}, fmt.Errorf("check organization before creating project: %w", err)
	}
	if !exists {
		return domain.Project{}, ErrNotFound
	}

	builder := r.client.Project.Create().
		SetOrganizationID(input.OrganizationID).
		SetName(input.Name).
		SetSlug(input.Slug).
		SetDescription(input.Description).
		SetStatus(toEntProjectStatus(input.Status)).
		SetAccessibleMachineIds(input.AccessibleMachineIDs).
		SetMaxConcurrentAgents(input.MaxConcurrentAgents).
		SetAgentRunSummaryPrompt(input.AgentRunSummaryPrompt)
	if input.DefaultAgentProviderID != nil {
		builder.SetDefaultAgentProviderID(*input.DefaultAgentProviderID)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Project{}, mapWriteError("create project", err)
	}

	return mapProject(item), nil
}

func (r *EntRepository) GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	item, err := r.client.Project.Get(ctx, id)
	if err != nil {
		return domain.Project{}, mapReadError("get project", err)
	}

	return mapProject(item), nil
}

func (r *EntRepository) UpdateProject(ctx context.Context, input domain.UpdateProject) (domain.Project, error) {
	builder := r.client.Project.UpdateOneID(input.ID).
		SetOrganizationID(input.OrganizationID).
		SetName(input.Name).
		SetSlug(input.Slug).
		SetDescription(input.Description).
		SetStatus(toEntProjectStatus(input.Status)).
		SetAccessibleMachineIds(input.AccessibleMachineIDs).
		SetMaxConcurrentAgents(input.MaxConcurrentAgents).
		SetAgentRunSummaryPrompt(input.AgentRunSummaryPrompt)
	if input.DefaultAgentProviderID != nil {
		builder.SetDefaultAgentProviderID(*input.DefaultAgentProviderID)
	} else {
		builder.ClearDefaultAgentProviderID()
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Project{}, mapWriteError("update project", err)
	}

	return mapProject(item), nil
}

func (r *EntRepository) ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	item, err := r.client.Project.UpdateOneID(id).SetStatus(toEntProjectStatus(domain.ProjectStatusArchived)).Save(ctx)
	if err != nil {
		return domain.Project{}, mapWriteError("archive project", err)
	}

	return mapProject(item), nil
}

func (r *EntRepository) ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRepo, error) {
	exists, err := r.client.Project.Query().
		Where(entproject.ID(projectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check project before listing repos: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	items, err := r.client.ProjectRepo.Query().
		Where(entprojectrepo.ProjectID(projectID)).
		Order(entprojectrepo.ByName(), entprojectrepo.ByID()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project repos: %w", err)
	}

	return mapProjectRepos(items), nil
}

func (r *EntRepository) CreateProjectRepo(ctx context.Context, input domain.CreateProjectRepo) (repo domain.ProjectRepo, err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("start create project repo transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	exists, err := tx.Project.Query().
		Where(entproject.ID(input.ProjectID)).
		Exist(ctx)
	if err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("check project before creating repo: %w", err)
	}
	if !exists {
		return domain.ProjectRepo{}, ErrNotFound
	}

	builder := tx.ProjectRepo.Create().
		SetProjectID(input.ProjectID).
		SetName(input.Name).
		SetRepositoryURL(input.RepositoryURL).
		SetDefaultBranch(input.DefaultBranch).
		SetWorkspaceDirname(input.WorkspaceDirname)
	if len(input.Labels) > 0 {
		builder.SetLabels(pgarray.StringArray(input.Labels))
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectRepo{}, mapWriteError("create project repo", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("commit create project repo: %w", err)
	}

	return mapProjectRepo(item), nil
}

func (r *EntRepository) GetProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	item, err := r.client.ProjectRepo.Query().
		Where(entprojectrepo.ID(id), entprojectrepo.ProjectID(projectID)).
		Only(ctx)
	if err != nil {
		return domain.ProjectRepo{}, mapReadError("get project repo", err)
	}

	return mapProjectRepo(item), nil
}

func (r *EntRepository) UpdateProjectRepo(ctx context.Context, input domain.UpdateProjectRepo) (repo domain.ProjectRepo, err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("start update project repo transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	if _, err := tx.ProjectRepo.Query().
		Where(entprojectrepo.ID(input.ID), entprojectrepo.ProjectID(input.ProjectID)).
		Only(ctx); err != nil {
		return domain.ProjectRepo{}, mapReadError("get project repo for update", err)
	}

	builder := tx.ProjectRepo.UpdateOneID(input.ID).
		SetName(input.Name).
		SetRepositoryURL(input.RepositoryURL).
		SetDefaultBranch(input.DefaultBranch).
		SetWorkspaceDirname(input.WorkspaceDirname)
	if len(input.Labels) > 0 {
		builder.SetLabels(pgarray.StringArray(input.Labels))
	} else {
		builder.ClearLabels()
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectRepo{}, mapWriteError("update project repo", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("commit update project repo: %w", err)
	}

	return mapProjectRepo(item), nil
}

func (r *EntRepository) DeleteProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (repo domain.ProjectRepo, err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("start delete project repo transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	item, err := tx.ProjectRepo.Query().
		Where(entprojectrepo.ID(id), entprojectrepo.ProjectID(projectID)).
		Only(ctx)
	if err != nil {
		return domain.ProjectRepo{}, mapReadError("get project repo for delete", err)
	}

	conflict, err := r.projectRepoDeleteConflict(ctx, item.ID)
	if err != nil {
		return domain.ProjectRepo{}, err
	}
	if conflict != nil {
		return domain.ProjectRepo{}, conflict
	}

	deleted := mapProjectRepo(item)
	if err := tx.ProjectRepo.DeleteOne(item).Exec(ctx); err != nil {
		return domain.ProjectRepo{}, mapWriteError("delete project repo", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("commit delete project repo: %w", err)
	}

	return deleted, nil
}

func (r *EntRepository) ListTicketRepoScopes(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]domain.TicketRepoScope, error) {
	exists, err := r.client.Ticket.Query().
		Where(entticket.ID(ticketID), entticket.ProjectID(projectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check ticket before listing repo scopes: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	items, err := r.client.TicketRepoScope.Query().
		Where(entticketreposcope.TicketID(ticketID)).
		Order(
			entticketreposcope.ByRepoField(entprojectrepo.FieldName),
			entticketreposcope.ByID(),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket repo scopes: %w", err)
	}

	return mapTicketRepoScopes(items), nil
}

func (r *EntRepository) CreateTicketRepoScope(ctx context.Context, input domain.CreateTicketRepoScope) (scope domain.TicketRepoScope, err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, fmt.Errorf("start create ticket repo scope transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	_, err = tx.Ticket.Query().
		Where(entticket.ID(input.TicketID), entticket.ProjectID(input.ProjectID)).
		Only(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, mapReadError("get ticket for repo scope create", err)
	}

	_, err = tx.ProjectRepo.Query().
		Where(entprojectrepo.ID(input.RepoID), entprojectrepo.ProjectID(input.ProjectID)).
		Only(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, mapReadError("get project repo for repo scope create", err)
	}

	branchName := ""
	if input.BranchName != nil {
		branchName = *input.BranchName
	}

	builder := tx.TicketRepoScope.Create().
		SetTicketID(input.TicketID).
		SetRepoID(input.RepoID).
		SetBranchName(branchName)
	if input.PullRequestURL != nil {
		builder.SetPullRequestURL(*input.PullRequestURL)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, mapWriteError("create ticket repo scope", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.TicketRepoScope{}, fmt.Errorf("commit create ticket repo scope: %w", err)
	}

	return mapTicketRepoScope(item), nil
}

func (r *EntRepository) GetTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error) {
	item, err := r.client.TicketRepoScope.Query().
		Where(
			entticketreposcope.ID(id),
			entticketreposcope.TicketID(ticketID),
			entticketreposcope.HasTicketWith(entticket.ProjectID(projectID)),
		).
		Only(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, mapReadError("get ticket repo scope", err)
	}

	return mapTicketRepoScope(item), nil
}

func (r *EntRepository) UpdateTicketRepoScope(ctx context.Context, input domain.UpdateTicketRepoScope) (scope domain.TicketRepoScope, err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, fmt.Errorf("start update ticket repo scope transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	_, err = tx.TicketRepoScope.Query().
		Where(
			entticketreposcope.ID(input.ID),
			entticketreposcope.TicketID(input.TicketID),
			entticketreposcope.HasTicketWith(entticket.ProjectID(input.ProjectID)),
		).
		Only(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, mapReadError("get ticket repo scope for update", err)
	}

	builder := tx.TicketRepoScope.UpdateOneID(input.ID)
	if input.BranchNameSet {
		builder.SetBranchName(strings.TrimSpace(derefString(input.BranchName)))
	}
	if input.PullRequestSet {
		if input.PullRequestURL != nil {
			builder.SetPullRequestURL(*input.PullRequestURL)
		} else {
			builder.ClearPullRequestURL()
		}
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, mapWriteError("update ticket repo scope", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.TicketRepoScope{}, fmt.Errorf("commit update ticket repo scope: %w", err)
	}

	return mapTicketRepoScope(item), nil
}

func (r *EntRepository) DeleteTicketRepoScope(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (scope domain.TicketRepoScope, err error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, fmt.Errorf("start delete ticket repo scope transaction: %w", err)
	}
	defer rollbackOnError(ctx, tx, &err)

	item, err := tx.TicketRepoScope.Query().
		Where(
			entticketreposcope.ID(id),
			entticketreposcope.TicketID(ticketID),
			entticketreposcope.HasTicketWith(entticket.ProjectID(projectID)),
		).
		Only(ctx)
	if err != nil {
		return domain.TicketRepoScope{}, mapReadError("get ticket repo scope for delete", err)
	}

	conflict, err := r.ticketRepoScopeDeleteConflict(ctx, item)
	if err != nil {
		return domain.TicketRepoScope{}, err
	}
	if conflict != nil {
		return domain.TicketRepoScope{}, conflict
	}

	deleted := mapTicketRepoScope(item)
	if err := tx.TicketRepoScope.DeleteOne(item).Exec(ctx); err != nil {
		return domain.TicketRepoScope{}, mapWriteError("delete ticket repo scope", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.TicketRepoScope{}, fmt.Errorf("commit delete ticket repo scope: %w", err)
	}

	return deleted, nil
}

func (r *EntRepository) projectRepoDeleteConflict(ctx context.Context, repoID uuid.UUID) (*domain.ProjectRepoDeleteConflict, error) {
	scopeItems, err := r.client.TicketRepoScope.Query().
		Where(entticketreposcope.RepoID(repoID)).
		Order(ent.Asc(entticketreposcope.FieldTicketID), ent.Asc(entticketreposcope.FieldBranchName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project repo ticket scopes: %w", err)
	}

	workspaceItems, err := r.client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.RepoID(repoID)).
		Order(ent.Asc(entticketrepoworkspace.FieldTicketID), ent.Asc(entticketrepoworkspace.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project repo workspaces: %w", err)
	}

	if len(scopeItems) == 0 && len(workspaceItems) == 0 {
		return nil, nil
	}

	conflict := &domain.ProjectRepoDeleteConflict{
		RepoID:       repoID,
		TicketScopes: make([]domain.ProjectRepoScopeReference, 0, len(scopeItems)),
		Workspaces:   make([]domain.ProjectRepoWorkspaceReference, 0, len(workspaceItems)),
	}
	for _, item := range scopeItems {
		conflict.TicketScopes = append(conflict.TicketScopes, domain.ProjectRepoScopeReference{
			ID:         item.ID,
			TicketID:   item.TicketID,
			BranchName: item.BranchName,
		})
	}
	for _, item := range workspaceItems {
		conflict.Workspaces = append(conflict.Workspaces, domain.ProjectRepoWorkspaceReference{
			ID:         item.ID,
			TicketID:   item.TicketID,
			AgentRunID: item.AgentRunID,
			State:      item.State.String(),
		})
	}
	return conflict, nil
}

func (r *EntRepository) ticketRepoScopeDeleteConflict(
	ctx context.Context,
	scope *ent.TicketRepoScope,
) (*domain.TicketRepoScopeDeleteConflict, error) {
	ticketItem, err := r.client.Ticket.Query().
		Where(entticket.ID(scope.TicketID)).
		Only(ctx)
	if err != nil {
		return nil, mapReadError("get ticket for repo scope delete", err)
	}

	workspaceItems, err := r.client.TicketRepoWorkspace.Query().
		Where(
			entticketrepoworkspace.TicketIDEQ(scope.TicketID),
			entticketrepoworkspace.RepoIDEQ(scope.RepoID),
		).
		Order(ent.Asc(entticketrepoworkspace.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repo scope workspaces: %w", err)
	}

	var activeRun *domain.TicketRepoScopeActiveRunReference
	if ticketItem.CurrentRunID != nil {
		activeRun = &domain.TicketRepoScopeActiveRunReference{
			TicketID:     ticketItem.ID,
			CurrentRunID: *ticketItem.CurrentRunID,
		}
	}

	blockingWorkspaces := make([]domain.TicketRepoScopeWorkspaceReference, 0, len(workspaceItems))
	for _, item := range workspaceItems {
		if !workspaceStateBlocksScopeDelete(item.State.String()) {
			continue
		}
		blockingWorkspaces = append(blockingWorkspaces, domain.TicketRepoScopeWorkspaceReference{
			ID:         item.ID,
			AgentRunID: item.AgentRunID,
			State:      item.State.String(),
		})
	}

	if activeRun == nil && len(blockingWorkspaces) == 0 {
		return nil, nil
	}

	return &domain.TicketRepoScopeDeleteConflict{
		ScopeID:    scope.ID,
		TicketID:   scope.TicketID,
		ActiveRun:  activeRun,
		Workspaces: blockingWorkspaces,
	}, nil
}

func workspaceStateBlocksScopeDelete(state string) bool {
	switch state {
	case entticketrepoworkspace.StateCompleted.String(),
		entticketrepoworkspace.StateFailed.String(),
		entticketrepoworkspace.StateCleaned.String():
		return false
	default:
		return true
	}
}

func mapReadError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mapWriteError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrNotFound
	case ent.IsConstraintError(err):
		return mapConstraintError(action, err)
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func mapConstraintError(action string, err error) error {
	detail := strings.ToLower(err.Error())

	switch {
	case strings.Contains(detail, "organization_slug"):
		return domain.ErrOrganizationSlugConflict
	case strings.Contains(detail, "project_organization_id_slug"):
		return domain.ErrProjectSlugConflict
	case strings.Contains(detail, "machine_organization_id_name"):
		return domain.ErrMachineNameConflict
	case strings.Contains(detail, "agentprovider_organization_id_name"):
		return domain.ErrAgentProviderNameConflict
	case strings.Contains(detail, "projectrepo_project_id_name"):
		return domain.ErrProjectRepoNameConflict
	case strings.Contains(detail, "ticketreposcope_ticket_id_repo_id"):
		return domain.ErrTicketRepoScopeConflict
	case strings.Contains(detail, "agent_project_id_name"):
		return domain.ErrAgentNameConflict
	case action == "delete machine":
		return domain.ErrMachineInUseConflict
	case action == "delete project repo":
		return domain.ErrProjectRepoInUseConflict
	case action == "delete agent":
		return domain.ErrAgentInUseConflict
	default:
		return ErrConflict
	}
}

func mapOrganizations(items []*ent.Organization) []domain.Organization {
	organizations := make([]domain.Organization, 0, len(items))
	for _, item := range items {
		organizations = append(organizations, mapOrganization(item))
	}

	return organizations
}

func mapOrganization(item *ent.Organization) domain.Organization {
	return domain.Organization{
		ID:                     item.ID,
		Name:                   item.Name,
		Slug:                   item.Slug,
		Status:                 toDomainOrganizationStatus(item.Status),
		DefaultAgentProviderID: item.DefaultAgentProviderID,
	}
}

func activeOrganizationPredicates(id uuid.UUID) []predicate.Organization {
	return []predicate.Organization{
		entorganization.ID(id),
		entorganization.StatusEQ(entorganization.StatusActive),
	}
}

func (r *EntRepository) organizationIsActive(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.client.Organization.Query().Where(activeOrganizationPredicates(id)...).Exist(ctx)
}

func (r *EntRepository) getActiveOrganization(ctx context.Context, id uuid.UUID) (*ent.Organization, error) {
	item, err := r.client.Organization.Query().Where(activeOrganizationPredicates(id)...).Only(ctx)
	if err != nil {
		return nil, mapReadError("get organization", err)
	}

	return item, nil
}

func mapProjects(items []*ent.Project) []domain.Project {
	projects := make([]domain.Project, 0, len(items))
	for _, item := range items {
		projects = append(projects, mapProject(item))
	}

	return projects
}

func mapProject(item *ent.Project) domain.Project {
	return domain.Project{
		ID:                     item.ID,
		OrganizationID:         item.OrganizationID,
		Name:                   item.Name,
		Slug:                   item.Slug,
		Description:            item.Description,
		Status:                 toDomainProjectStatus(item.Status),
		DefaultAgentProviderID: item.DefaultAgentProviderID,
		AccessibleMachineIDs:   append([]uuid.UUID(nil), item.AccessibleMachineIds...),
		MaxConcurrentAgents:    item.MaxConcurrentAgents,
		AgentRunSummaryPrompt:  item.AgentRunSummaryPrompt,
	}
}

func rollbackOnError(_ context.Context, tx *ent.Tx, errp *error) {
	if *errp == nil {
		return
	}
	_ = tx.Rollback()
}

func createBuiltinAgentProviders(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, machineID uuid.UUID) error {
	for _, template := range domain.BuiltinAgentProviderTemplates() {
		pricingConfig := domain.ResolveAgentProviderPricingConfig(
			template.AdapterType,
			template.ModelName,
			0,
			0,
			nil,
		)
		builder := tx.AgentProvider.Create().
			SetOrganizationID(organizationID).
			SetMachineID(machineID).
			SetName(template.Name).
			SetAdapterType(toEntAgentProviderAdapterType(template.AdapterType)).
			SetCliCommand(template.Command).
			SetCliArgs(append([]string(nil), template.CliArgs...)).
			SetAuthConfig(map[string]any{}).
			SetModelName(template.ModelName).
			SetCostPerInputToken(pricingConfig.SummaryInputPerToken()).
			SetCostPerOutputToken(pricingConfig.SummaryOutputPerToken()).
			SetPricingConfig(pricingConfig.ToMap())
		if _, err := builder.Save(ctx); err != nil {
			return mapWriteError("create builtin agent provider", err)
		}
	}

	return nil
}

func mapProjectRepos(items []*ent.ProjectRepo) []domain.ProjectRepo {
	repos := make([]domain.ProjectRepo, 0, len(items))
	for _, item := range items {
		repos = append(repos, mapProjectRepo(item))
	}

	return repos
}

func mapProjectRepo(item *ent.ProjectRepo) domain.ProjectRepo {
	return domain.ProjectRepo{
		ID:               item.ID,
		ProjectID:        item.ProjectID,
		Name:             item.Name,
		RepositoryURL:    item.RepositoryURL,
		DefaultBranch:    item.DefaultBranch,
		WorkspaceDirname: item.WorkspaceDirname,
		Labels:           append([]string(nil), item.Labels...),
	}
}

func mapTicketRepoScopes(items []*ent.TicketRepoScope) []domain.TicketRepoScope {
	scopes := make([]domain.TicketRepoScope, 0, len(items))
	for _, item := range items {
		scopes = append(scopes, mapTicketRepoScope(item))
	}

	return scopes
}

func mapTicketRepoScope(item *ent.TicketRepoScope) domain.TicketRepoScope {
	return domain.TicketRepoScope{
		ID:             item.ID,
		TicketID:       item.TicketID,
		RepoID:         item.RepoID,
		BranchName:     item.BranchName,
		PullRequestURL: optionalString(item.PullRequestURL),
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	copied := value
	return &copied
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	copied := *value
	return &copied
}
