package githubauth

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	domain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	"github.com/google/uuid"
)

type Repository interface {
	GetProjectContext(ctx context.Context, projectID uuid.UUID) (domain.ProjectContext, error)
	SaveOrganizationCredential(ctx context.Context, organizationID uuid.UUID, credential domain.StoredCredential, probe domain.TokenProbe) error
	SaveProjectCredential(ctx context.Context, projectID uuid.UUID, credential domain.StoredCredential, probe domain.TokenProbe) error
	ClearOrganizationCredential(ctx context.Context, organizationID uuid.UUID) error
	ClearProjectCredential(ctx context.Context, projectID uuid.UUID) error
	SaveOrganizationProbe(ctx context.Context, organizationID uuid.UUID, probe domain.TokenProbe) error
	SaveProjectProbe(ctx context.Context, projectID uuid.UUID, probe domain.TokenProbe) error
}

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) GetProjectContext(ctx context.Context, projectID uuid.UUID) (domain.ProjectContext, error) {
	projectItem, err := r.client.Project.Query().
		Where(entproject.IDEQ(projectID)).
		WithOrganization().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ProjectContext{}, fmt.Errorf("project not found: %w", err)
		}
		return domain.ProjectContext{}, fmt.Errorf("load project GitHub auth context: %w", err)
	}
	if projectItem.Edges.Organization == nil {
		return domain.ProjectContext{}, fmt.Errorf("project organization must be loaded")
	}
	projectRepositoryURL, err := r.loadProjectRepositoryURL(ctx, projectID)
	if err != nil {
		return domain.ProjectContext{}, err
	}

	return domain.ProjectContext{
		ProjectID:              projectItem.ID,
		OrganizationID:         projectItem.OrganizationID,
		ProjectRepositoryURL:   projectRepositoryURL,
		OrganizationCredential: cloneStoredCredential(projectItem.Edges.Organization.GithubOutboundCredential),
		OrganizationProbe:      cloneProbe(projectItem.Edges.Organization.GithubTokenProbe),
		ProjectCredential:      cloneStoredCredential(projectItem.GithubOutboundCredential),
		ProjectProbe:           cloneProbe(projectItem.GithubTokenProbe),
	}, nil
}

func (r *EntRepository) SaveOrganizationCredential(
	ctx context.Context,
	organizationID uuid.UUID,
	credential domain.StoredCredential,
	probe domain.TokenProbe,
) error {
	clonedCredential := credential
	clonedProbe := probe
	return r.client.Organization.UpdateOneID(organizationID).
		SetGithubOutboundCredential(&clonedCredential).
		SetGithubTokenProbe(&clonedProbe).
		Exec(ctx)
}

func (r *EntRepository) SaveProjectCredential(
	ctx context.Context,
	projectID uuid.UUID,
	credential domain.StoredCredential,
	probe domain.TokenProbe,
) error {
	clonedCredential := credential
	clonedProbe := probe
	return r.client.Project.UpdateOneID(projectID).
		SetGithubOutboundCredential(&clonedCredential).
		SetGithubTokenProbe(&clonedProbe).
		Exec(ctx)
}

func (r *EntRepository) ClearOrganizationCredential(ctx context.Context, organizationID uuid.UUID) error {
	return r.client.Organization.UpdateOneID(organizationID).
		ClearGithubOutboundCredential().
		ClearGithubTokenProbe().
		Exec(ctx)
}

func (r *EntRepository) ClearProjectCredential(ctx context.Context, projectID uuid.UUID) error {
	return r.client.Project.UpdateOneID(projectID).
		ClearGithubOutboundCredential().
		ClearGithubTokenProbe().
		Exec(ctx)
}

func (r *EntRepository) SaveOrganizationProbe(ctx context.Context, organizationID uuid.UUID, probe domain.TokenProbe) error {
	cloned := probe
	return r.client.Organization.UpdateOneID(organizationID).
		SetGithubTokenProbe(&cloned).
		Exec(ctx)
}

func (r *EntRepository) SaveProjectProbe(ctx context.Context, projectID uuid.UUID, probe domain.TokenProbe) error {
	cloned := probe
	return r.client.Project.UpdateOneID(projectID).
		SetGithubTokenProbe(&cloned).
		Exec(ctx)
}

func (r *EntRepository) loadProjectRepositoryURL(ctx context.Context, projectID uuid.UUID) (string, error) {
	fallbackRepo, err := r.client.ProjectRepo.Query().
		Where(entprojectrepo.ProjectIDEQ(projectID)).
		Order(ent.Asc(entprojectrepo.FieldName)).
		First(ctx)
	if err == nil {
		return fallbackRepo.RepositoryURL, nil
	}
	if ent.IsNotFound(err) {
		return "", nil
	}
	return "", fmt.Errorf("load fallback project repository for GitHub auth context: %w", err)
}

func cloneStoredCredential(raw *domain.StoredCredential) *domain.StoredCredential {
	if raw == nil {
		return nil
	}
	cloned := *raw
	return &cloned
}

func cloneProbe(raw *domain.TokenProbe) *domain.TokenProbe {
	if raw == nil {
		return nil
	}
	cloned := *raw
	if raw.CheckedAt != nil {
		checkedAt := raw.CheckedAt.UTC()
		cloned.CheckedAt = &checkedAt
	}
	cloned.Permissions = append([]string(nil), raw.Permissions...)
	return &cloned
}
