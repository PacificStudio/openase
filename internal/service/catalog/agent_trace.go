package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func (s *service) ListAgentRunTraceEntries(ctx context.Context, input domain.ListAgentRunTraceEntries) ([]domain.AgentTraceEntry, error) {
	return s.repo.ListAgentRunTraceEntries(ctx, input)
}

func (s *service) ListAgentRunStepEntries(ctx context.Context, input domain.ListAgentRunStepEntries) ([]domain.AgentStepEntry, error) {
	return s.repo.ListAgentRunStepEntries(ctx, input)
}
