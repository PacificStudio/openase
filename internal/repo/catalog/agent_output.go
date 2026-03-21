package catalog

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

var agentOutputEventTypes = []string{
	"agent.output",
	"agent.claimed",
	"agent.launching",
	"agent.ready",
	"agent.failed",
	"agent.terminated",
}

func (r *EntRepository) GetAgentOutput(ctx context.Context, input domain.GetAgentOutput) (domain.AgentOutput, error) {
	agentItem, err := r.client.Agent.Query().
		Where(entagent.ID(input.AgentID)).
		Only(ctx)
	if err != nil {
		return domain.AgentOutput{}, mapReadError("get agent output agent", err)
	}

	activityItems, err := r.client.ActivityEvent.Query().
		Where(
			entactivityevent.AgentID(input.AgentID),
			entactivityevent.EventTypeIn(agentOutputEventTypes...),
		).
		Order(entactivityevent.ByCreatedAt(entsql.OrderDesc()), entactivityevent.ByID(entsql.OrderDesc())).
		Limit(input.Limit).
		All(ctx)
	if err != nil {
		return domain.AgentOutput{}, fmt.Errorf("list agent output: %w", err)
	}

	return domain.AgentOutput{
		Agent:   mapAgent(agentItem),
		Entries: mapAgentOutputEntries(activityItems),
	}, nil
}

func mapAgentOutputEntries(items []*ent.ActivityEvent) []domain.AgentOutputEntry {
	entries := make([]domain.AgentOutputEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentOutputEntry(item))
	}

	return entries
}

func mapAgentOutputEntry(item *ent.ActivityEvent) domain.AgentOutputEntry {
	defaultStream := domain.AgentOutputStreamSystem
	if item.EventType == "agent.output" {
		defaultStream = domain.AgentOutputStreamStdout
	}

	stream, err := domain.ParseAgentOutputStream(stringMetadata(item.Metadata, "stream"), defaultStream)
	if err != nil {
		stream = defaultStream
	}

	return domain.AgentOutputEntry{
		ID:        item.ID,
		TicketID:  item.TicketID,
		EventType: item.EventType,
		Stream:    stream,
		Message:   item.Message,
		Metadata:  cloneAnyMap(item.Metadata),
		CreatedAt: cloneAgentOutputCreatedAt(item.CreatedAt),
	}
}

func stringMetadata(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	value, ok := metadata[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}

func cloneAgentOutputCreatedAt(value time.Time) time.Time {
	return value.UTC()
}
