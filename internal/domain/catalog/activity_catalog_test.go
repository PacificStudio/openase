package catalog

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseListActivityEventsSupportsCursorPaging(t *testing.T) {
	projectID := uuid.New()
	agentID := uuid.New()
	ticketID := uuid.New()
	before := ActivityEventCursor{
		CreatedAt: time.Date(2026, 4, 2, 10, 5, 36, 123456789, time.UTC),
		ID:        uuid.New(),
	}

	parsed, err := ParseListActivityEvents(projectID, ActivityEventListInput{
		AgentID:  " " + agentID.String() + " ",
		TicketID: ticketID.String(),
		Limit:    " 25 ",
		Before:   " " + before.String() + " ",
	})
	if err != nil {
		t.Fatalf("ParseListActivityEvents() error = %v", err)
	}
	if parsed.ProjectID != projectID || parsed.AgentID == nil || *parsed.AgentID != agentID || parsed.TicketID == nil || *parsed.TicketID != ticketID || parsed.Limit != 25 || parsed.Before == nil || *parsed.Before != before {
		t.Fatalf("ParseListActivityEvents() = %+v", parsed)
	}
}

func TestActivityEventCursorHelpers(t *testing.T) {
	item := ActivityEvent{
		ID:        uuid.New(),
		CreatedAt: time.Date(2026, 4, 2, 10, 5, 36, 123456789, time.UTC),
	}

	cursor := ActivityEventCursorFor(item)
	if cursor.ID != item.ID || !cursor.CreatedAt.Equal(item.CreatedAt) {
		t.Fatalf("ActivityEventCursorFor() = %+v", cursor)
	}

	parsed, err := ParseActivityEventCursor(cursor.String())
	if err != nil {
		t.Fatalf("ParseActivityEventCursor() error = %v", err)
	}
	if parsed != cursor {
		t.Fatalf("ParseActivityEventCursor() = %+v, want %+v", parsed, cursor)
	}
}

func TestParseActivityEventCursorErrors(t *testing.T) {
	validID := uuid.New()

	testCases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "invalid format",
			raw:  "2026-04-02T10:05:00Z",
			want: "cursor must be in timestamp|id format",
		},
		{
			name: "invalid timestamp",
			raw:  "bad|" + validID.String(),
			want: "cursor timestamp must be RFC3339",
		},
		{
			name: "invalid id",
			raw:  "2026-04-02T10:05:00Z|bad",
			want: "cursor id must be a valid UUID",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := ParseActivityEventCursor(testCase.raw); err == nil || err.Error() != testCase.want {
				t.Fatalf("ParseActivityEventCursor(%q) error = %v, want %q", testCase.raw, err, testCase.want)
			}
		})
	}

	if _, err := ParseListActivityEvents(uuid.New(), ActivityEventListInput{Limit: "bad"}); err == nil || err.Error() != "limit must be a valid integer" {
		t.Fatalf("expected invalid limit error, got %v", err)
	}
	if _, err := ParseListActivityEvents(uuid.New(), ActivityEventListInput{Before: "bad"}); err == nil || err.Error() != "before cursor must be in timestamp|id format" {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func TestCompareActivityEventCursor(t *testing.T) {
	baseTime := time.Date(2026, 4, 2, 10, 5, 0, 0, time.UTC)
	leftID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	rightID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	if got := CompareActivityEventCursor(
		ActivityEventCursor{CreatedAt: baseTime},
		ActivityEventCursor{CreatedAt: baseTime.Add(time.Second)},
	); got >= 0 {
		t.Fatalf("expected earlier cursor to sort first, got %d", got)
	}
	if got := CompareActivityEventCursor(
		ActivityEventCursor{CreatedAt: baseTime.Add(time.Second)},
		ActivityEventCursor{CreatedAt: baseTime},
	); got <= 0 {
		t.Fatalf("expected later cursor to sort last, got %d", got)
	}
	if got := CompareActivityEventCursor(
		ActivityEventCursor{CreatedAt: baseTime, ID: leftID},
		ActivityEventCursor{CreatedAt: baseTime, ID: rightID},
	); got >= 0 {
		t.Fatalf("expected lower UUID to sort first, got %d", got)
	}
	if got := CompareActivityEventCursor(
		ActivityEventCursor{CreatedAt: baseTime, ID: rightID},
		ActivityEventCursor{CreatedAt: baseTime, ID: leftID},
	); got <= 0 {
		t.Fatalf("expected higher UUID to sort last, got %d", got)
	}
	if got := CompareActivityEventCursor(
		ActivityEventCursor{CreatedAt: baseTime, ID: leftID},
		ActivityEventCursor{CreatedAt: baseTime, ID: leftID},
	); got != 0 {
		t.Fatalf("expected identical cursors to compare equal, got %d", got)
	}
}
