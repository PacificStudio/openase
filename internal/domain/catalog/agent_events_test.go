package catalog

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAgentRunTranscriptCursorHelpers(t *testing.T) {
	traceID := uuid.New()
	stepID := uuid.New()
	createdAt := time.Date(2026, 4, 1, 10, 5, 36, 123456789, time.UTC)

	traceCursor := AgentRunTranscriptCursorForTrace(AgentTraceEntry{
		ID:        traceID,
		Sequence:  42,
		CreatedAt: createdAt,
	})
	if traceCursor.Kind != AgentRunTranscriptKindTrace || traceCursor.Order != 42 || traceCursor.ID != traceID || !traceCursor.CreatedAt.Equal(createdAt) {
		t.Fatalf("AgentRunTranscriptCursorForTrace() = %+v", traceCursor)
	}

	stepCursor := AgentRunTranscriptCursorForStep(AgentStepEntry{
		ID:        stepID,
		CreatedAt: createdAt,
	})
	if stepCursor.Kind != AgentRunTranscriptKindStep || stepCursor.Order != 0 || stepCursor.ID != stepID || !stepCursor.CreatedAt.Equal(createdAt) {
		t.Fatalf("AgentRunTranscriptCursorForStep() = %+v", stepCursor)
	}

	encoded := traceCursor.String()
	parsed, err := ParseAgentRunTranscriptCursor(encoded)
	if err != nil {
		t.Fatalf("ParseAgentRunTranscriptCursor() error = %v", err)
	}
	if parsed != traceCursor {
		t.Fatalf("ParseAgentRunTranscriptCursor() = %+v, want %+v", parsed, traceCursor)
	}
}

func TestParseListAgentRunTranscriptPage(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	before := AgentRunTranscriptCursor{
		CreatedAt: time.Date(2026, 4, 1, 10, 5, 0, 0, time.UTC),
		Kind:      AgentRunTranscriptKindStep,
		Order:     0,
		ID:        uuid.New(),
	}
	after := AgentRunTranscriptCursor{
		CreatedAt: time.Date(2026, 4, 1, 10, 6, 0, 0, time.UTC),
		Kind:      AgentRunTranscriptKindTrace,
		Order:     7,
		ID:        uuid.New(),
	}

	parsed, err := ParseListAgentRunTranscriptPage(projectID, runID, AgentRunTranscriptPageInput{
		Limit:  " 17 ",
		Before: " " + before.String() + " ",
	})
	if err != nil {
		t.Fatalf("ParseListAgentRunTranscriptPage() error = %v", err)
	}
	if parsed.ProjectID != projectID || parsed.AgentRunID != runID || parsed.Limit != 17 || parsed.Before == nil || *parsed.Before != before || parsed.After != nil {
		t.Fatalf("ParseListAgentRunTranscriptPage() = %+v", parsed)
	}

	parsed, err = ParseListAgentRunTranscriptPage(projectID, runID, AgentRunTranscriptPageInput{
		After: after.String(),
	})
	if err != nil {
		t.Fatalf("ParseListAgentRunTranscriptPage() default limit error = %v", err)
	}
	if parsed.Limit != DefaultAgentRunTranscriptPageLimit || parsed.After == nil || *parsed.After != after || parsed.Before != nil {
		t.Fatalf("ParseListAgentRunTranscriptPage() default parse = %+v", parsed)
	}

	if _, err := ParseListAgentRunTranscriptPage(projectID, runID, AgentRunTranscriptPageInput{
		Before: before.String(),
		After:  after.String(),
	}); err == nil || err.Error() != "before and after cannot be combined" {
		t.Fatalf("expected before/after conflict error, got %v", err)
	}
}

func TestParseAgentRunTranscriptCursorErrors(t *testing.T) {
	validID := uuid.New()

	testCases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "invalid format",
			raw:  "2026-04-01T10:05:00Z|step|0",
			want: "cursor must be in timestamp|kind|order|id format",
		},
		{
			name: "invalid timestamp",
			raw:  "bad|step|0|" + validID.String(),
			want: "cursor timestamp must be RFC3339",
		},
		{
			name: "invalid kind",
			raw:  "2026-04-01T10:05:00Z|noise|0|" + validID.String(),
			want: "cursor kind must be step or trace",
		},
		{
			name: "invalid order",
			raw:  "2026-04-01T10:05:00Z|step|nope|" + validID.String(),
			want: "cursor order must be an integer",
		},
		{
			name: "invalid id",
			raw:  "2026-04-01T10:05:00Z|step|0|bad",
			want: "cursor id must be a valid UUID",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := ParseAgentRunTranscriptCursor(testCase.raw); err == nil || err.Error() != testCase.want {
				t.Fatalf("ParseAgentRunTranscriptCursor(%q) error = %v, want %q", testCase.raw, err, testCase.want)
			}
		})
	}

	if _, err := ParseListAgentRunTranscriptPage(uuid.New(), uuid.New(), AgentRunTranscriptPageInput{Limit: "bad"}); err == nil || err.Error() != "limit must be a valid integer" {
		t.Fatalf("expected invalid limit error, got %v", err)
	}
	if _, err := ParseListAgentRunTranscriptPage(uuid.New(), uuid.New(), AgentRunTranscriptPageInput{Limit: "0"}); err == nil || err.Error() != "limit must be greater than zero" {
		t.Fatalf("expected non-positive limit error, got %v", err)
	}
	if _, err := ParseListAgentRunTranscriptPage(uuid.New(), uuid.New(), AgentRunTranscriptPageInput{Limit: "501"}); err == nil || err.Error() != "limit must be less than or equal to 500" {
		t.Fatalf("expected max limit error, got %v", err)
	}
	if _, err := ParseListAgentRunTranscriptPage(uuid.New(), uuid.New(), AgentRunTranscriptPageInput{Before: "bad"}); err == nil || err.Error() != "before cursor must be in timestamp|kind|order|id format" {
		t.Fatalf("expected before cursor error, got %v", err)
	}
	if _, err := ParseListAgentRunTranscriptPage(uuid.New(), uuid.New(), AgentRunTranscriptPageInput{After: "bad"}); err == nil || err.Error() != "after cursor must be in timestamp|kind|order|id format" {
		t.Fatalf("expected after cursor error, got %v", err)
	}
}

func TestCompareAgentRunTranscriptCursor(t *testing.T) {
	baseTime := time.Date(2026, 4, 1, 10, 5, 0, 0, time.UTC)
	leftID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	rightID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime},
		AgentRunTranscriptCursor{CreatedAt: baseTime.Add(time.Second)},
	); got >= 0 {
		t.Fatalf("expected earlier cursor to sort first, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime.Add(time.Second)},
		AgentRunTranscriptCursor{CreatedAt: baseTime},
	); got <= 0 {
		t.Fatalf("expected later cursor to sort last, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindStep},
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace},
	); got >= 0 {
		t.Fatalf("expected step cursor before trace cursor, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: "unknown"},
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace},
	); got <= 0 {
		t.Fatalf("expected unknown cursor kind to sort after trace, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 1},
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2},
	); got >= 0 {
		t.Fatalf("expected lower trace order to sort first, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2},
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 1},
	); got <= 0 {
		t.Fatalf("expected higher trace order to sort last, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2, ID: leftID},
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2, ID: rightID},
	); got >= 0 {
		t.Fatalf("expected lower UUID to sort first, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2, ID: rightID},
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2, ID: leftID},
	); got <= 0 {
		t.Fatalf("expected higher UUID to sort last, got %d", got)
	}
	if got := CompareAgentRunTranscriptCursor(
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2, ID: leftID},
		AgentRunTranscriptCursor{CreatedAt: baseTime, Kind: AgentRunTranscriptKindTrace, Order: 2, ID: leftID},
	); got != 0 {
		t.Fatalf("expected identical cursors to compare equal, got %d", got)
	}
}
