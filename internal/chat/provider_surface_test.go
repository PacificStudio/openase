package chat

import (
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestChatProviderSurfaceForSourceAlwaysUsesEphemeralChat(t *testing.T) {
	for _, source := range []Source{SourceProjectSidebar, SourceTicketDetail} {
		if got := chatProviderSurfaceForSource(source); got != providerSurfaceEphemeralChat {
			t.Fatalf("chatProviderSurfaceForSource(%s) = %s", source, got)
		}
	}
}

func TestResolveProviderCapabilityForSurfaceUsesEphemeralChat(t *testing.T) {
	provider := catalogdomain.AgentProvider{
		AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
		Available:   true,
	}
	capability := resolveProviderCapabilityForSurface(provider, providerSurfaceEphemeralChat)
	if capability.State != catalogdomain.AgentProviderCapabilityStateAvailable {
		t.Fatalf("capability = %+v", capability)
	}
}
