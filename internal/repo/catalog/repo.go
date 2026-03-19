package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("catalog resource not found")
	ErrConflict = errors.New("catalog resource conflict")
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
