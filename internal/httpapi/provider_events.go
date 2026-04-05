package httpapi

import (
	"context"
	"fmt"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var (
	providerCreatedEventType = provider.MustParseEventType("provider.created")
	providerUpdatedEventType = provider.MustParseEventType("provider.updated")
)

func (s *Server) publishProviderLifecycleEvent(
	ctx context.Context,
	eventType provider.EventType,
	item domain.AgentProvider,
) error {
	if s == nil || s.events == nil {
		return nil
	}

	event, err := provider.NewJSONEvent(providerStreamTopic, eventType, map[string]any{
		"organization_id": item.OrganizationID.String(),
		"provider":        mapAgentProviderResponse(item),
	}, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("construct provider lifecycle event: %w", err)
	}
	if err := s.events.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish provider lifecycle event: %w", err)
	}

	return nil
}
