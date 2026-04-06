package humanauth

import "github.com/google/uuid"

type EffectiveVisibility struct {
	Instance             bool
	OrganizationIDs      []uuid.UUID
	OrganizationScopeIDs []uuid.UUID
	ProjectIDs           []uuid.UUID
}

func (v EffectiveVisibility) AllowsOrganization(id uuid.UUID) bool {
	return v.Instance || uuidInSlice(v.OrganizationIDs, id) || uuidInSlice(v.OrganizationScopeIDs, id)
}

func (v EffectiveVisibility) AllowsOrganizationScope(id uuid.UUID) bool {
	return v.Instance || uuidInSlice(v.OrganizationScopeIDs, id)
}

func (v EffectiveVisibility) AllowsProject(projectID uuid.UUID, organizationID uuid.UUID) bool {
	return v.Instance || uuidInSlice(v.ProjectIDs, projectID) || uuidInSlice(v.OrganizationScopeIDs, organizationID)
}

func uuidInSlice(items []uuid.UUID, want uuid.UUID) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
