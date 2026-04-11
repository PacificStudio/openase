package catalog

import (
	"context"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func (s *service) ListAgentRunRawEvents(
	ctx context.Context,
	input domain.ListAgentRunRawEvents,
) (domain.AgentRunRawEventPage, error) {
	return s.repo.ListAgentRunRawEvents(ctx, input)
}

func (s *service) ListAgentRunActivities(
	ctx context.Context,
	input domain.ListAgentRunActivities,
) ([]domain.AgentActivityInstance, error) {
	return s.repo.ListAgentRunActivities(ctx, input)
}

func (s *service) ListAgentRunTranscriptEntries(
	ctx context.Context,
	input domain.ListAgentRunTranscriptEntries,
) (domain.AgentRunTranscriptEntryPage, error) {
	return s.repo.ListAgentRunTranscriptEntries(ctx, input)
}
