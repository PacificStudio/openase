package chat

import catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"

func normalizeRuntimePermissionProfile(
	profile catalogdomain.AgentProviderPermissionProfile,
) catalogdomain.AgentProviderPermissionProfile {
	if !profile.IsValid() {
		return catalogdomain.DefaultAgentProviderPermissionProfile
	}
	return profile
}
