package projectupdate

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultThreadPageLimit = 10
	MaxThreadPageLimit     = 100
)

type ListThreadsPageRequest struct {
	Limit  string
	Before string
}

type ListThreadsPage struct {
	ProjectID uuid.UUID
	Limit     int
	Before    *ThreadCursor
}

type ThreadCursor struct {
	LastActivityAt time.Time
	ID             uuid.UUID
}

func ParseListThreadsPage(projectID uuid.UUID, raw ListThreadsPageRequest) (ListThreadsPage, error) {
	limit, err := parseThreadPageLimit(raw.Limit)
	if err != nil {
		return ListThreadsPage{}, err
	}
	before, err := parseOptionalThreadCursor("before", raw.Before)
	if err != nil {
		return ListThreadsPage{}, err
	}

	return ListThreadsPage{
		ProjectID: projectID,
		Limit:     limit,
		Before:    before,
	}, nil
}

func ParseThreadCursor(raw string) (ThreadCursor, error) {
	parts := strings.Split(strings.TrimSpace(raw), "|")
	if len(parts) != 2 {
		return ThreadCursor{}, fmt.Errorf("cursor must be in timestamp|id format")
	}

	lastActivityAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return ThreadCursor{}, fmt.Errorf("cursor timestamp must be RFC3339")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return ThreadCursor{}, fmt.Errorf("cursor id must be a valid UUID")
	}

	return ThreadCursor{
		LastActivityAt: lastActivityAt.UTC(),
		ID:             id,
	}, nil
}

func (c ThreadCursor) String() string {
	return fmt.Sprintf("%s|%s", c.LastActivityAt.UTC().Format(time.RFC3339Nano), c.ID.String())
}

func parseThreadPageLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultThreadPageLimit, nil
	}

	limit, err := strconv.Atoi(trimmed)
	if err != nil || limit < 1 || limit > MaxThreadPageLimit {
		return 0, fmt.Errorf("limit must be an integer between 1 and %d", MaxThreadPageLimit)
	}
	return limit, nil
}

func parseOptionalThreadCursor(label string, raw string) (*ThreadCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	cursor, err := ParseThreadCursor(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s %w", label, err)
	}
	return &cursor, nil
}
