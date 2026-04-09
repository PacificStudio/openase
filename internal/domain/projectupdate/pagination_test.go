package projectupdate

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseListThreadsPage(t *testing.T) {
	projectID := uuid.New()
	beforeID := uuid.New()
	beforeAt := time.Date(2026, 4, 1, 12, 30, 0, 123456000, time.UTC)

	parsed, err := ParseListThreadsPage(projectID, ListThreadsPageRequest{
		Limit:  " 7 ",
		Before: " " + beforeAt.Format(time.RFC3339Nano) + "|" + beforeID.String() + " ",
	})
	if err != nil {
		t.Fatalf("ParseListThreadsPage() error = %v", err)
	}
	if parsed.ProjectID != projectID || parsed.Limit != 7 || parsed.Before == nil {
		t.Fatalf("ParseListThreadsPage() = %+v", parsed)
	}
	if parsed.Before.LastActivityAt != beforeAt || parsed.Before.ID != beforeID {
		t.Fatalf("ParseListThreadsPage() before = %+v", parsed.Before)
	}

	defaulted, err := ParseListThreadsPage(projectID, ListThreadsPageRequest{})
	if err != nil {
		t.Fatalf("ParseListThreadsPage(default) error = %v", err)
	}
	if defaulted.Limit != DefaultThreadPageLimit || defaulted.Before != nil {
		t.Fatalf("ParseListThreadsPage(default) = %+v", defaulted)
	}
}

func TestParseListThreadsPageRejectsInvalidInput(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name string
		raw  ListThreadsPageRequest
		want string
	}{
		{
			name: "invalid limit",
			raw:  ListThreadsPageRequest{Limit: "0"},
			want: "limit must be an integer between 1 and 100",
		},
		{
			name: "invalid cursor shape",
			raw:  ListThreadsPageRequest{Before: "bad"},
			want: "before cursor must be in timestamp|id format",
		},
		{
			name: "invalid cursor timestamp",
			raw:  ListThreadsPageRequest{Before: "bad|" + uuid.NewString()},
			want: "before cursor timestamp must be RFC3339",
		},
		{
			name: "invalid cursor id",
			raw:  ListThreadsPageRequest{Before: time.Now().UTC().Format(time.RFC3339Nano) + "|bad"},
			want: "before cursor id must be a valid UUID",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseListThreadsPage(projectID, tc.raw); err == nil || err.Error() != tc.want {
				t.Fatalf("ParseListThreadsPage() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestThreadCursorStringRoundTrips(t *testing.T) {
	cursor := ThreadCursor{
		LastActivityAt: time.Date(2026, 4, 1, 15, 45, 30, 987654321, time.UTC),
		ID:             uuid.New(),
	}

	parsed, err := ParseThreadCursor(cursor.String())
	if err != nil {
		t.Fatalf("ParseThreadCursor() error = %v", err)
	}
	if parsed != cursor {
		t.Fatalf("ParseThreadCursor() = %+v, want %+v", parsed, cursor)
	}
}
