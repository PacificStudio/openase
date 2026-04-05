package catalog

import (
	"context"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestGetAgentRunTranscriptPageReturnsLatestWindowAndStableCursorPaging(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	baseAt := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	traceLaterID := uuid.New()
	traceEarlierID := uuid.New()
	stepMiddleID := uuid.New()

	repo := &stubRepository{
		traceEntries: []domain.AgentTraceEntry{
			{
				ID:         traceEarlierID,
				ProjectID:  projectID,
				AgentRunID: runID,
				Sequence:   1,
				Provider:   "codex",
				Kind:       domain.AgentTraceKindAssistantDelta,
				Stream:     "assistant",
				Output:     "older trace",
				CreatedAt:  baseAt,
			},
			{
				ID:         traceLaterID,
				ProjectID:  projectID,
				AgentRunID: runID,
				Sequence:   2,
				Provider:   "codex",
				Kind:       domain.AgentTraceKindAssistantDelta,
				Stream:     "assistant",
				Output:     "newer trace",
				CreatedAt:  baseAt.Add(2 * time.Second),
			},
		},
		stepEntries: []domain.AgentStepEntry{
			{
				ID:         stepMiddleID,
				ProjectID:  projectID,
				AgentRunID: runID,
				StepStatus: "planning",
				Summary:    "middle step",
				CreatedAt:  baseAt.Add(1 * time.Second),
			},
		},
	}

	svc := New(repo, stubExecutableResolver{}, nil)
	page, err := svc.GetAgentRunTranscriptPage(context.Background(), domain.ListAgentRunTranscriptPage{
		ProjectID:  projectID,
		AgentRunID: runID,
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("GetAgentRunTranscriptPage() error = %v", err)
	}

	if !page.HasOlder || page.HiddenOlderCount != 1 {
		t.Fatalf("expected exactly one older transcript item hidden, got %+v", page)
	}
	if page.HasNewer || page.HiddenNewerCount != 0 {
		t.Fatalf("expected latest window to have no newer items hidden, got %+v", page)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 transcript items, got %+v", page.Items)
	}
	if page.Items[0].Kind != domain.AgentRunTranscriptKindStep || page.Items[0].StepEntry == nil || page.Items[0].StepEntry.ID != stepMiddleID {
		t.Fatalf("expected step item first in latest page, got %+v", page.Items[0])
	}
	if page.Items[1].Kind != domain.AgentRunTranscriptKindTrace || page.Items[1].TraceEntry == nil || page.Items[1].TraceEntry.ID != traceLaterID {
		t.Fatalf("expected newer trace item last in latest page, got %+v", page.Items[1])
	}

	olderPage, err := svc.GetAgentRunTranscriptPage(context.Background(), domain.ListAgentRunTranscriptPage{
		ProjectID:  projectID,
		AgentRunID: runID,
		Limit:      2,
		Before: func() *domain.AgentRunTranscriptCursor {
			cursor, parseErr := domain.ParseAgentRunTranscriptCursor(page.OldestCursor)
			if parseErr != nil {
				t.Fatalf("parse oldest cursor: %v", parseErr)
			}
			return &cursor
		}(),
	})
	if err != nil {
		t.Fatalf("GetAgentRunTranscriptPage(before) error = %v", err)
	}
	if olderPage.HasOlder || olderPage.HiddenOlderCount != 0 {
		t.Fatalf("expected older page to exhaust history, got %+v", olderPage)
	}
	if !olderPage.HasNewer || olderPage.HiddenNewerCount != 2 {
		t.Fatalf("expected newer tail metadata on older page, got %+v", olderPage)
	}
	if len(olderPage.Items) != 1 || olderPage.Items[0].TraceEntry == nil || olderPage.Items[0].TraceEntry.ID != traceEarlierID {
		t.Fatalf("expected only the earliest trace in the older page, got %+v", olderPage.Items)
	}
}

func TestGetAgentRunTranscriptPageKeepsStepBeforeTraceWhenTimestampsMatch(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	sharedAt := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	stepID := uuid.New()
	traceID := uuid.New()

	svc := New(&stubRepository{
		traceEntries: []domain.AgentTraceEntry{
			{
				ID:         traceID,
				ProjectID:  projectID,
				AgentRunID: runID,
				Sequence:   9,
				Provider:   "codex",
				Kind:       domain.AgentTraceKindAssistantDelta,
				Stream:     "assistant",
				Output:     "trace",
				CreatedAt:  sharedAt,
			},
		},
		stepEntries: []domain.AgentStepEntry{
			{
				ID:         stepID,
				ProjectID:  projectID,
				AgentRunID: runID,
				StepStatus: "planning",
				Summary:    "step",
				CreatedAt:  sharedAt,
			},
		},
	}, stubExecutableResolver{}, nil)

	page, err := svc.GetAgentRunTranscriptPage(context.Background(), domain.ListAgentRunTranscriptPage{
		ProjectID:  projectID,
		AgentRunID: runID,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("GetAgentRunTranscriptPage() error = %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected both transcript items, got %+v", page.Items)
	}
	if page.Items[0].Kind != domain.AgentRunTranscriptKindStep || page.Items[1].Kind != domain.AgentRunTranscriptKindTrace {
		t.Fatalf("expected step-before-trace ordering for equal timestamps, got %+v", page.Items)
	}
}
