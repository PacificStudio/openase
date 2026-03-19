package catalog

import (
	"context"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("catalog resource not found")
	ErrConflict     = errors.New("catalog resource conflict")
	ErrInvalidInput = errors.New("catalog invalid input")
)

type Repository interface {
	ListOrganizations(ctx context.Context) ([]domain.Organization, error)
	CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error)
	ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error)
	CreateProject(ctx context.Context, input domain.CreateProject) (domain.Project, error)
	GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
	UpdateProject(ctx context.Context, input domain.UpdateProject) (domain.Project, error)
	ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error)
	ListProjectRepos(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRepo, error)
	CreateProjectRepo(ctx context.Context, input domain.CreateProjectRepo) (domain.ProjectRepo, error)
	GetProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error)
	UpdateProjectRepo(ctx context.Context, input domain.UpdateProjectRepo) (domain.ProjectRepo, error)
	DeleteProjectRepo(ctx context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]domain.AgentProvider, error)
	CreateAgentProvider(ctx context.Context, input domain.CreateAgentProvider) (domain.AgentProvider, error)
	GetAgentProvider(ctx context.Context, id uuid.UUID) (domain.AgentProvider, error)
	UpdateAgentProvider(ctx context.Context, input domain.UpdateAgentProvider) (domain.AgentProvider, error)
	ListAgents(ctx context.Context, projectID uuid.UUID) ([]domain.Agent, error)
	CreateAgent(ctx context.Context, input domain.CreateAgent) (domain.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) (domain.Agent, error)
}

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) ListOrganizations(ctx context.Context) ([]domain.Organization, error) {
	items, err := r.client.Organization.Query().Order(entorganization.ByName()).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return mapOrganizations(items), nil
}

func (r *EntRepository) CreateOrganization(ctx context.Context, input domain.CreateOrganization) (domain.Organization, error) {
	builder := r.client.Organization.Create().
		SetName(input.Name).
		SetSlug(input.Slug)
	if input.DefaultAgentProviderID != nil {
		builder.SetDefaultAgentProviderID(*input.DefaultAgentProviderID)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Organization{}, mapWriteError("create organization", err)
	}

	return mapOrganization(item), nil
}

func (r *EntRepository) GetOrganization(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	item, err := r.client.Organization.Get(ctx, id)
	if err != nil {
		return domain.Organization{}, mapReadError("get organization", err)
	}

	return mapOrganization(item), nil
}

func (r *EntRepository) UpdateOrganization(ctx context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
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

func (r *EntRepository) ListProjects(ctx context.Context, organizationID uuid.UUID) ([]domain.Project, error) {
	exists, err := r.client.Organization.Query().
		Where(entorganization.ID(organizationID)).
		Exist(ctx)
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
	exists, err := r.client.Organization.Query().
		Where(entorganization.ID(input.OrganizationID)).
		Exist(ctx)
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
		SetStatus(input.Status).
		SetMaxConcurrentAgents(input.MaxConcurrentAgents)
	if input.DefaultWorkflowID != nil {
		builder.SetDefaultWorkflowID(*input.DefaultWorkflowID)
	}
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
		SetStatus(input.Status).
		SetMaxConcurrentAgents(input.MaxConcurrentAgents)
	if input.DefaultWorkflowID != nil {
		builder.SetDefaultWorkflowID(*input.DefaultWorkflowID)
	} else {
		builder.ClearDefaultWorkflowID()
	}
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
	item, err := r.client.Project.UpdateOneID(id).SetStatus(entproject.StatusArchived).Save(ctx)
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
		Order(entprojectrepo.ByIsPrimary(entsql.OrderDesc()), entprojectrepo.ByName()).
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

	repoCount, err := tx.ProjectRepo.Query().
		Where(entprojectrepo.ProjectID(input.ProjectID)).
		Count(ctx)
	if err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("count project repos before create: %w", err)
	}

	makePrimary := repoCount == 0
	if input.RequestedPrimary != nil {
		makePrimary = *input.RequestedPrimary || repoCount == 0
	}
	if makePrimary {
		if err := clearPrimaryRepo(ctx, tx, input.ProjectID); err != nil {
			return domain.ProjectRepo{}, err
		}
	}

	builder := tx.ProjectRepo.Create().
		SetProjectID(input.ProjectID).
		SetName(input.Name).
		SetRepositoryURL(input.RepositoryURL).
		SetDefaultBranch(input.DefaultBranch).
		SetIsPrimary(makePrimary)
	if input.ClonePath != nil {
		builder.SetClonePath(*input.ClonePath)
	}
	if len(input.Labels) > 0 {
		builder.SetLabels(input.Labels)
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

	current, err := tx.ProjectRepo.Query().
		Where(entprojectrepo.ID(input.ID), entprojectrepo.ProjectID(input.ProjectID)).
		Only(ctx)
	if err != nil {
		return domain.ProjectRepo{}, mapReadError("get project repo for update", err)
	}

	makePrimary := input.IsPrimary
	if makePrimary {
		if err := clearPrimaryRepo(ctx, tx, input.ProjectID, input.ID); err != nil {
			return domain.ProjectRepo{}, err
		}
	}

	builder := tx.ProjectRepo.UpdateOneID(input.ID).
		SetName(input.Name).
		SetRepositoryURL(input.RepositoryURL).
		SetDefaultBranch(input.DefaultBranch).
		SetIsPrimary(makePrimary)
	if input.ClonePath != nil {
		builder.SetClonePath(*input.ClonePath)
	} else {
		builder.ClearClonePath()
	}
	if len(input.Labels) > 0 {
		builder.SetLabels(input.Labels)
	} else {
		builder.ClearLabels()
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.ProjectRepo{}, mapWriteError("update project repo", err)
	}

	if current.IsPrimary && !item.IsPrimary {
		if err := ensureProjectPrimaryRepo(ctx, tx, input.ProjectID, item.ID); err != nil {
			return domain.ProjectRepo{}, err
		}
		item, err = tx.ProjectRepo.Get(ctx, item.ID)
		if err != nil {
			return domain.ProjectRepo{}, mapReadError("reload project repo after update", err)
		}
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

	deleted := mapProjectRepo(item)
	if err := tx.ProjectRepo.DeleteOne(item).Exec(ctx); err != nil {
		return domain.ProjectRepo{}, mapWriteError("delete project repo", err)
	}

	if item.IsPrimary {
		if err := ensureProjectPrimaryRepo(ctx, tx, projectID); err != nil {
			return domain.ProjectRepo{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.ProjectRepo{}, fmt.Errorf("commit delete project repo: %w", err)
	}

	return deleted, nil
}

func mapReadError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func mapWriteError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrNotFound
	case ent.IsConstraintError(err):
		return ErrConflict
	default:
		return fmt.Errorf("%s: %w", action, err)
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
		DefaultAgentProviderID: item.DefaultAgentProviderID,
	}
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
		Status:                 item.Status,
		DefaultWorkflowID:      item.DefaultWorkflowID,
		DefaultAgentProviderID: item.DefaultAgentProviderID,
		MaxConcurrentAgents:    item.MaxConcurrentAgents,
	}
}

func rollbackOnError(ctx context.Context, tx *ent.Tx, errp *error) {
	if *errp == nil {
		return
	}
	_ = tx.Rollback()
}

func clearPrimaryRepo(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, excludeIDs ...uuid.UUID) error {
	predicates := []predicate.ProjectRepo{
		entprojectrepo.ProjectID(projectID),
		entprojectrepo.IsPrimary(true),
	}
	for _, id := range excludeIDs {
		predicates = append(predicates, entprojectrepo.IDNEQ(id))
	}

	if _, err := tx.ProjectRepo.Update().
		Where(predicates...).
		SetIsPrimary(false).
		Save(ctx); err != nil {
		return fmt.Errorf("clear primary project repo: %w", err)
	}

	return nil
}

func ensureProjectPrimaryRepo(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, excludeIDs ...uuid.UUID) error {
	exists, err := tx.ProjectRepo.Query().
		Where(entprojectrepo.ProjectID(projectID), entprojectrepo.IsPrimary(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check primary project repo: %w", err)
	}
	if exists {
		return nil
	}

	predicates := []predicate.ProjectRepo{
		entprojectrepo.ProjectID(projectID),
	}
	for _, id := range excludeIDs {
		predicates = append(predicates, entprojectrepo.IDNEQ(id))
	}

	fallback, err := tx.ProjectRepo.Query().
		Where(predicates...).
		Order(entprojectrepo.ByName(), entprojectrepo.ByID()).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			if len(excludeIDs) == 0 {
				return nil
			}

			fallback, err = tx.ProjectRepo.Query().
				Where(entprojectrepo.ProjectID(projectID)).
				Order(entprojectrepo.ByName(), entprojectrepo.ByID()).
				First(ctx)
			if err != nil {
				if ent.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("select fallback primary project repo: %w", err)
			}
		} else {
			return fmt.Errorf("select fallback primary project repo: %w", err)
		}
	}

	if err := tx.ProjectRepo.UpdateOneID(fallback.ID).SetIsPrimary(true).Exec(ctx); err != nil {
		return fmt.Errorf("promote fallback primary project repo: %w", err)
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
		ID:            item.ID,
		ProjectID:     item.ProjectID,
		Name:          item.Name,
		RepositoryURL: item.RepositoryURL,
		DefaultBranch: item.DefaultBranch,
		ClonePath:     optionalString(item.ClonePath),
		IsPrimary:     item.IsPrimary,
		Labels:        append([]string(nil), item.Labels...),
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	copied := value
	return &copied
}
