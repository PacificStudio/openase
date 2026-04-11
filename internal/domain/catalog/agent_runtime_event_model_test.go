package catalog

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAgentRunEventModelParsersAndCursors(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	cursorID := uuid.New()
	cursorTime := time.Date(2026, 4, 11, 10, 30, 45, 123000000, time.UTC)
	cursorText := cursorTime.Format(time.RFC3339Nano) + "|" + cursorID.String()

	rawEvents, err := ParseListAgentRunRawEvents(projectID, runID, AgentRunEventPageInput{
		Limit:  " 25 ",
		Before: " " + cursorText + " ",
	})
	if err != nil {
		t.Fatalf("ParseListAgentRunRawEvents() error = %v", err)
	}
	if rawEvents.ProjectID != projectID || rawEvents.AgentRunID != runID || rawEvents.Limit != 25 {
		t.Fatalf("ParseListAgentRunRawEvents() = %+v", rawEvents)
	}
	if rawEvents.Before == nil || rawEvents.Before.CreatedAt != cursorTime || rawEvents.Before.ID != cursorID {
		t.Fatalf("ParseListAgentRunRawEvents() before = %+v", rawEvents.Before)
	}
	if rawEvents.After != nil {
		t.Fatalf("ParseListAgentRunRawEvents() after = %+v, want nil", rawEvents.After)
	}

	transcriptEntries, err := ParseListAgentRunTranscriptEntries(projectID, runID, AgentRunEventPageInput{
		After: cursorText,
	})
	if err != nil {
		t.Fatalf("ParseListAgentRunTranscriptEntries() error = %v", err)
	}
	if transcriptEntries.Limit != DefaultAgentRunEventPageLimit {
		t.Fatalf("ParseListAgentRunTranscriptEntries() limit = %d, want %d", transcriptEntries.Limit, DefaultAgentRunEventPageLimit)
	}
	if transcriptEntries.After == nil || transcriptEntries.After.ID != cursorID {
		t.Fatalf("ParseListAgentRunTranscriptEntries() after = %+v", transcriptEntries.After)
	}

	activities, err := ParseListAgentRunActivities(projectID, runID, " in_progress ")
	if err != nil {
		t.Fatalf("ParseListAgentRunActivities() error = %v", err)
	}
	if activities.Status != "in_progress" {
		t.Fatalf("ParseListAgentRunActivities() status = %q, want in_progress", activities.Status)
	}

	cursor, err := ParseAgentRunEventCursor(" " + cursorText + " ")
	if err != nil {
		t.Fatalf("ParseAgentRunEventCursor() error = %v", err)
	}
	if cursor.CreatedAt != cursorTime || cursor.ID != cursorID {
		t.Fatalf("ParseAgentRunEventCursor() = %+v", cursor)
	}
	if got := cursor.String(); got != cursorText {
		t.Fatalf("AgentRunEventCursor.String() = %q, want %q", got, cursorText)
	}

	laterCursor := AgentRunEventCursor{CreatedAt: cursorTime.Add(time.Second), ID: uuid.New()}
	if got := CompareAgentRunEventCursor(cursor, laterCursor); got >= 0 {
		t.Fatalf("CompareAgentRunEventCursor(earlier,later) = %d, want < 0", got)
	}
	if got := CompareAgentRunEventCursor(laterCursor, cursor); got <= 0 {
		t.Fatalf("CompareAgentRunEventCursor(later,earlier) = %d, want > 0", got)
	}
	if got := CompareAgentRunEventCursor(cursor, cursor); got != 0 {
		t.Fatalf("CompareAgentRunEventCursor(equal) = %d, want 0", got)
	}

	rawCursor := AgentRunEventCursorForRawEvent(AgentRawEventEntry{
		ID:         cursorID,
		OccurredAt: cursorTime.In(time.FixedZone("offset", 2*3600)),
	})
	if rawCursor.CreatedAt != cursorTime || rawCursor.ID != cursorID {
		t.Fatalf("AgentRunEventCursorForRawEvent() = %+v", rawCursor)
	}

	transcriptCursor := AgentRunEventCursorForTranscriptEntry(AgentTranscriptEntry{
		ID:        cursorID,
		CreatedAt: cursorTime.In(time.FixedZone("offset", -5*3600)),
	})
	if transcriptCursor.CreatedAt != cursorTime || transcriptCursor.ID != cursorID {
		t.Fatalf("AgentRunEventCursorForTranscriptEntry() = %+v", transcriptCursor)
	}

	page, err := parseAgentRunEventPage(AgentRunEventPageInput{})
	if err != nil {
		t.Fatalf("parseAgentRunEventPage() default error = %v", err)
	}
	if page.Limit != DefaultAgentRunEventPageLimit || page.Before != nil || page.After != nil {
		t.Fatalf("parseAgentRunEventPage() = %+v", page)
	}

	if got, err := parseAgentRunEventPageLimit(""); err != nil || got != DefaultAgentRunEventPageLimit {
		t.Fatalf("parseAgentRunEventPageLimit(\"\") = (%d, %v), want (%d, nil)", got, err, DefaultAgentRunEventPageLimit)
	}
	if got, err := parseOptionalAgentRunEventCursor("after", " "); err != nil || got != nil {
		t.Fatalf("parseOptionalAgentRunEventCursor(blank) = (%+v, %v), want (nil, nil)", got, err)
	}
}

func TestAgentRunEventModelParseErrors(t *testing.T) {
	cursorTime := time.Date(2026, 4, 11, 10, 30, 45, 0, time.UTC)
	cursorID := uuid.New()
	validCursor := cursorTime.Format(time.RFC3339Nano) + "|" + cursorID.String()

	testCases := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "cursor format",
			run: func() error {
				_, err := ParseAgentRunEventCursor("missing-delimiter")
				return err
			},
			want: "timestamp|id",
		},
		{
			name: "cursor timestamp",
			run: func() error {
				_, err := ParseAgentRunEventCursor("not-a-time|" + cursorID.String())
				return err
			},
			want: "RFC3339",
		},
		{
			name: "cursor uuid",
			run: func() error {
				_, err := ParseAgentRunEventCursor(cursorTime.Format(time.RFC3339Nano) + "|bad-uuid")
				return err
			},
			want: "valid UUID",
		},
		{
			name: "limit integer",
			run: func() error {
				_, err := parseAgentRunEventPageLimit("nope")
				return err
			},
			want: "positive integer",
		},
		{
			name: "limit positive",
			run: func() error {
				_, err := parseAgentRunEventPageLimit("0")
				return err
			},
			want: "positive integer",
		},
		{
			name: "limit max",
			run: func() error {
				_, err := parseAgentRunEventPageLimit("501")
				return err
			},
			want: "<= 500",
		},
		{
			name: "cursor field wrapper",
			run: func() error {
				_, err := parseOptionalAgentRunEventCursor("before", "bad")
				return err
			},
			want: "before cursor",
		},
		{
			name: "page before cursor",
			run: func() error {
				_, err := parseAgentRunEventPage(AgentRunEventPageInput{Before: "bad"})
				return err
			},
			want: "before cursor",
		},
		{
			name: "before and after",
			run: func() error {
				_, err := parseAgentRunEventPage(AgentRunEventPageInput{
					Before: validCursor,
					After:  validCursor,
				})
				return err
			},
			want: "cannot be combined",
		},
		{
			name: "raw events parse limit",
			run: func() error {
				_, err := ParseListAgentRunRawEvents(uuid.New(), uuid.New(), AgentRunEventPageInput{Limit: "bad"})
				return err
			},
			want: "positive integer",
		},
		{
			name: "transcript parse cursor",
			run: func() error {
				_, err := ParseListAgentRunTranscriptEntries(uuid.New(), uuid.New(), AgentRunEventPageInput{After: "bad"})
				return err
			},
			want: "after cursor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatalf("%s: expected error", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%s: error = %q, want substring %q", tc.name, err.Error(), tc.want)
			}
		})
	}
}
