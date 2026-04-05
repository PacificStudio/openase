package catalog

import (
	"context"
	"slices"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

type transcriptPageItem struct {
	item   domain.AgentRunTranscriptItem
	cursor domain.AgentRunTranscriptCursor
}

func (s *service) GetAgentRunTranscriptPage(
	ctx context.Context,
	input domain.ListAgentRunTranscriptPage,
) (domain.AgentRunTranscriptPage, error) {
	traceEntries, err := s.repo.ListAgentRunTraceEntries(ctx, domain.ListAgentRunTraceEntries{
		ProjectID:  input.ProjectID,
		AgentRunID: input.AgentRunID,
	})
	if err != nil {
		return domain.AgentRunTranscriptPage{}, err
	}
	stepEntries, err := s.repo.ListAgentRunStepEntries(ctx, domain.ListAgentRunStepEntries{
		ProjectID:  input.ProjectID,
		AgentRunID: input.AgentRunID,
	})
	if err != nil {
		return domain.AgentRunTranscriptPage{}, err
	}

	items := make([]transcriptPageItem, 0, len(traceEntries)+len(stepEntries))
	for _, entry := range stepEntries {
		cursor := domain.AgentRunTranscriptCursorForStep(entry)
		entryCopy := entry
		items = append(items, transcriptPageItem{
			item: domain.AgentRunTranscriptItem{
				Kind:       domain.AgentRunTranscriptKindStep,
				Cursor:     cursor.String(),
				StepEntry:  &entryCopy,
				TraceEntry: nil,
			},
			cursor: cursor,
		})
	}
	for _, entry := range traceEntries {
		cursor := domain.AgentRunTranscriptCursorForTrace(entry)
		entryCopy := entry
		items = append(items, transcriptPageItem{
			item: domain.AgentRunTranscriptItem{
				Kind:       domain.AgentRunTranscriptKindTrace,
				Cursor:     cursor.String(),
				TraceEntry: &entryCopy,
				StepEntry:  nil,
			},
			cursor: cursor,
		})
	}

	slices.SortFunc(items, func(left transcriptPageItem, right transcriptPageItem) int {
		return domain.CompareAgentRunTranscriptCursor(left.cursor, right.cursor)
	})

	start, end := resolveTranscriptPageWindow(items, input)
	pageItems := make([]domain.AgentRunTranscriptItem, 0, end-start)
	for _, item := range items[start:end] {
		pageItems = append(pageItems, item.item)
	}

	page := domain.AgentRunTranscriptPage{
		Items:            pageItems,
		HasOlder:         start > 0,
		HiddenOlderCount: start,
		HasNewer:         end < len(items),
		HiddenNewerCount: len(items) - end,
	}
	if len(pageItems) > 0 {
		page.OldestCursor = pageItems[0].Cursor
		page.NewestCursor = pageItems[len(pageItems)-1].Cursor
	}

	return page, nil
}

func resolveTranscriptPageWindow(
	items []transcriptPageItem,
	input domain.ListAgentRunTranscriptPage,
) (start int, end int) {
	if len(items) == 0 {
		return 0, 0
	}

	switch {
	case input.After != nil:
		start = len(items)
		for index, item := range items {
			if domain.CompareAgentRunTranscriptCursor(item.cursor, *input.After) > 0 {
				start = index
				break
			}
		}
		end = min(start+input.Limit, len(items))
	case input.Before != nil:
		end = 0
		for index, item := range items {
			if domain.CompareAgentRunTranscriptCursor(item.cursor, *input.Before) >= 0 {
				end = index
				break
			}
			end = index + 1
		}
		start = max(0, end-input.Limit)
	default:
		end = len(items)
		start = max(0, end-input.Limit)
	}

	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(items) {
		end = len(items)
	}

	return start, end
}
