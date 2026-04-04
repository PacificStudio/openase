package catalog

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseOrganizationTokenUsageDefaultsToLastThirtyUTCDays(t *testing.T) {
	orgID := uuid.New()
	now := time.Date(2026, 4, 1, 15, 30, 0, 0, time.UTC)

	parsed, err := ParseOrganizationTokenUsage(orgID, OrganizationTokenUsageListInput{}, now)
	if err != nil {
		t.Fatalf("ParseOrganizationTokenUsage() error = %v", err)
	}

	if parsed.OrganizationID != orgID {
		t.Fatalf("expected organization %s, got %s", orgID, parsed.OrganizationID)
	}
	if want := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC); !parsed.FromDate.Equal(want) {
		t.Fatalf("expected from %s, got %s", want.Format(time.RFC3339), parsed.FromDate.Format(time.RFC3339))
	}
	if want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC); !parsed.ToDate.Equal(want) {
		t.Fatalf("expected to %s, got %s", want.Format(time.RFC3339), parsed.ToDate.Format(time.RFC3339))
	}
}

func TestParseOrganizationTokenUsageRejectsInvalidRanges(t *testing.T) {
	orgID := uuid.New()

	if _, err := ParseOrganizationTokenUsage(orgID, OrganizationTokenUsageListInput{
		From: "2026-03-01",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected missing to to fail")
	}
	if _, err := ParseOrganizationTokenUsage(orgID, OrganizationTokenUsageListInput{
		From: "not-a-date",
		To:   "2026-03-01",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected invalid from date to fail")
	}
	if _, err := ParseOrganizationTokenUsage(orgID, OrganizationTokenUsageListInput{
		From: "2026-03-01",
		To:   "bad-date",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected invalid to date to fail")
	}
	if _, err := ParseOrganizationTokenUsage(orgID, OrganizationTokenUsageListInput{
		From: "2026-03-02",
		To:   "2026-03-01",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected descending date range to fail")
	}
}

func TestParseOrganizationTokenUsageAcceptsExplicitRange(t *testing.T) {
	orgID := uuid.New()

	parsed, err := ParseOrganizationTokenUsage(orgID, OrganizationTokenUsageListInput{
		From: " 2026-03-01 ",
		To:   "2026-03-31 ",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("ParseOrganizationTokenUsage() error = %v", err)
	}

	if want := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC); !parsed.FromDate.Equal(want) {
		t.Fatalf("expected from %s, got %s", want.Format(time.RFC3339), parsed.FromDate.Format(time.RFC3339))
	}
	if want := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC); !parsed.ToDate.Equal(want) {
		t.Fatalf("expected to %s, got %s", want.Format(time.RFC3339), parsed.ToDate.Format(time.RFC3339))
	}
}

func TestParseProjectTokenUsageDefaultsToLastThirtyUTCDays(t *testing.T) {
	projectID := uuid.New()
	now := time.Date(2026, 4, 1, 15, 30, 0, 0, time.UTC)

	parsed, err := ParseProjectTokenUsage(projectID, ProjectTokenUsageListInput{}, now)
	if err != nil {
		t.Fatalf("ParseProjectTokenUsage() error = %v", err)
	}

	if parsed.ProjectID != projectID {
		t.Fatalf("expected project %s, got %s", projectID, parsed.ProjectID)
	}
	if want := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC); !parsed.FromDate.Equal(want) {
		t.Fatalf("expected from %s, got %s", want.Format(time.RFC3339), parsed.FromDate.Format(time.RFC3339))
	}
	if want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC); !parsed.ToDate.Equal(want) {
		t.Fatalf("expected to %s, got %s", want.Format(time.RFC3339), parsed.ToDate.Format(time.RFC3339))
	}
}

func TestParseProjectTokenUsageAcceptsExplicitRange(t *testing.T) {
	projectID := uuid.New()

	parsed, err := ParseProjectTokenUsage(projectID, ProjectTokenUsageListInput{
		From: " 2026-03-01 ",
		To:   "2026-03-31 ",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("ParseProjectTokenUsage() error = %v", err)
	}

	if want := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC); !parsed.FromDate.Equal(want) {
		t.Fatalf("expected from %s, got %s", want.Format(time.RFC3339), parsed.FromDate.Format(time.RFC3339))
	}
	if want := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC); !parsed.ToDate.Equal(want) {
		t.Fatalf("expected to %s, got %s", want.Format(time.RFC3339), parsed.ToDate.Format(time.RFC3339))
	}
}

func TestParseProjectTokenUsageRejectsInvalidRanges(t *testing.T) {
	projectID := uuid.New()

	if _, err := ParseProjectTokenUsage(projectID, ProjectTokenUsageListInput{
		From: "2026-03-01",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected missing to to fail")
	}
	if _, err := ParseProjectTokenUsage(projectID, ProjectTokenUsageListInput{
		From: "not-a-date",
		To:   "2026-03-01",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected invalid from date to fail")
	}
	if _, err := ParseProjectTokenUsage(projectID, ProjectTokenUsageListInput{
		From: "2026-03-01",
		To:   "bad-date",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected invalid to date to fail")
	}
	if _, err := ParseProjectTokenUsage(projectID, ProjectTokenUsageListInput{
		From: "2026-03-02",
		To:   "2026-03-01",
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected descending date range to fail")
	}
}
