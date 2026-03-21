package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func (s *service) ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error) {
	return s.repo.ListActivityEvents(ctx, input)
}

func (s *service) GetAgentOutput(ctx context.Context, input domain.GetAgentOutput) (domain.AgentOutput, error) {
	return s.repo.GetAgentOutput(ctx, input)
}
