package catalog

import (
	"context"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/google/uuid"
)

type stubVisibilityResolver struct {
	visibility humanauthdomain.EffectiveVisibility
}

func (s stubVisibilityResolver) EffectiveVisibility(
	context.Context,
	humanauthdomain.AuthenticatedPrincipal,
) (humanauthdomain.EffectiveVisibility, error) {
	return s.visibility, nil
}

func TestListOrganizationsFiltersByEffectiveVisibility(t *testing.T) {
	t.Parallel()

	orgA := catalogdomain.Organization{ID: uuid.New(), Name: "Alpha"}
	orgB := catalogdomain.Organization{ID: uuid.New(), Name: "Beta"}
	repo := &stubRepository{
		organizations: []catalogdomain.Organization{orgA, orgB},
	}
	svc := New(
		repo,
		stubExecutableResolver{},
		nil,
		WithHumanVisibilityResolver(stubVisibilityResolver{
			visibility: humanauthdomain.EffectiveVisibility{
				OrganizationIDs: []uuid.UUID{orgA.ID},
			},
		}),
	)
	ctx := humanauthdomain.WithPrincipal(context.Background(), humanauthdomain.AuthenticatedPrincipal{})

	items, err := svc.ListOrganizations(ctx)
	if err != nil {
		t.Fatalf("ListOrganizations() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != orgA.ID {
		t.Fatalf("ListOrganizations() = %+v, want only %s", items, orgA.ID)
	}
}

func TestListProjectsFiltersToVisibleProjectsWithinOrganization(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectA := catalogdomain.Project{ID: uuid.New(), OrganizationID: orgID, Name: "Alpha"}
	projectB := catalogdomain.Project{ID: uuid.New(), OrganizationID: orgID, Name: "Beta"}
	repo := &stubRepository{
		projects: []catalogdomain.Project{projectA, projectB},
	}
	svc := New(
		repo,
		stubExecutableResolver{},
		nil,
		WithHumanVisibilityResolver(stubVisibilityResolver{
			visibility: humanauthdomain.EffectiveVisibility{
				ProjectIDs: []uuid.UUID{projectA.ID},
			},
		}),
	)
	ctx := humanauthdomain.WithPrincipal(context.Background(), humanauthdomain.AuthenticatedPrincipal{})

	items, err := svc.ListProjects(ctx, orgID)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != projectA.ID {
		t.Fatalf("ListProjects() = %+v, want only %s", items, projectA.ID)
	}
}

func TestListProjectReposReturnsEmptyWhenProjectIsInvisible(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()
	repo := &stubRepository{
		projects: []catalogdomain.Project{{ID: projectID, OrganizationID: orgID}},
		projectRepos: []catalogdomain.ProjectRepo{
			{ID: uuid.New(), ProjectID: projectID, Name: "frontend"},
		},
	}
	svc := New(
		repo,
		stubExecutableResolver{},
		nil,
		WithHumanVisibilityResolver(stubVisibilityResolver{
			visibility: humanauthdomain.EffectiveVisibility{},
		}),
	)
	ctx := humanauthdomain.WithPrincipal(context.Background(), humanauthdomain.AuthenticatedPrincipal{})

	items, err := svc.ListProjectRepos(ctx, projectID)
	if err != nil {
		t.Fatalf("ListProjectRepos() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("ListProjectRepos() = %+v, want empty", items)
	}
}
