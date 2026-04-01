package orchestrator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	"github.com/google/uuid"
)

func providerActivityProjectIDs(
	ctx context.Context,
	client *ent.Client,
	organizationID uuid.UUID,
	providerID uuid.UUID,
) ([]uuid.UUID, error) {
	if client == nil || organizationID == uuid.Nil || providerID == uuid.Nil {
		return nil, nil
	}

	projectItems, err := client.Project.Query().
		Where(entproject.OrganizationIDEQ(organizationID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list projects for provider activity: %w", err)
	}

	projectIDs := make([]uuid.UUID, 0, len(projectItems))
	for _, item := range projectItems {
		if item.DefaultAgentProviderID != nil && *item.DefaultAgentProviderID == providerID {
			projectIDs = append(projectIDs, item.ID)
			continue
		}

		used, err := client.Agent.Query().
			Where(
				entagent.ProjectIDEQ(item.ID),
				entagent.ProviderIDEQ(providerID),
			).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("list provider-bound agents for activity: %w", err)
		}
		if used {
			projectIDs = append(projectIDs, item.ID)
		}
	}

	slices.SortFunc(projectIDs, func(left uuid.UUID, right uuid.UUID) int {
		return strings.Compare(left.String(), right.String())
	})
	return slices.Compact(projectIDs), nil
}
