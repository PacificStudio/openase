package httpapi

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type organizationPatchRequest struct {
	Name                   *string `json:"name"`
	Slug                   *string `json:"slug"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id"`
}

func parseOrganizationPatchRequest(
	organizationID uuid.UUID,
	current domain.Organization,
	patch organizationPatchRequest,
) (domain.UpdateOrganization, error) {
	request := domain.OrganizationInput{
		Name:                   current.Name,
		Slug:                   current.Slug,
		DefaultAgentProviderID: uuidToStringPointer(current.DefaultAgentProviderID),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.Slug != nil {
		request.Slug = *patch.Slug
	}
	if patch.DefaultAgentProviderID != nil {
		request.DefaultAgentProviderID = patch.DefaultAgentProviderID
	}

	return domain.ParseUpdateOrganization(organizationID, request)
}
