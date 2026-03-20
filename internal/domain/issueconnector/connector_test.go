package issueconnector

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseCreateIssueConnectorDefaultsAndNormalizes(t *testing.T) {
	projectID := uuid.New()

	input, err := ParseCreateIssueConnector(projectID, Input{
		Type: " GitHub ",
		Name: " Acme Backend Issues ",
		Config: ConfigInput{
			Type:          "github",
			ProjectRef:    "acme/backend",
			SyncDirection: "",
			PollInterval:  "",
			Filters: FiltersInput{
				Labels:        []string{" OpenASE ", "openase"},
				ExcludeLabels: []string{"Ignore"},
				States:        []string{" Open "},
				Authors:       []string{" Codex "},
			},
			StatusMapping: map[string]string{
				" OPEN ": "Todo",
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseCreateIssueConnector returned error: %v", err)
	}

	if input.Type != TypeGitHub {
		t.Fatalf("Type = %q, want %q", input.Type, TypeGitHub)
	}
	if input.Name != "Acme Backend Issues" {
		t.Fatalf("Name = %q, want trimmed name", input.Name)
	}
	if input.Status != StatusActive {
		t.Fatalf("Status = %q, want %q", input.Status, StatusActive)
	}
	if input.Config.PollInterval != defaultPollInterval {
		t.Fatalf("PollInterval = %s, want %s", input.Config.PollInterval, defaultPollInterval)
	}
	if input.Config.SyncDirection != SyncDirectionBidirectional {
		t.Fatalf("SyncDirection = %q, want %q", input.Config.SyncDirection, SyncDirectionBidirectional)
	}
	if got := input.Config.Filters.Labels; len(got) != 1 || got[0] != "openase" {
		t.Fatalf("Labels = %+v, want [openase]", got)
	}
	if got := input.Config.Filters.ExcludeLabels; len(got) != 1 || got[0] != "ignore" {
		t.Fatalf("ExcludeLabels = %+v, want [ignore]", got)
	}
	if got := input.Config.StatusMapping["open"]; got != "Todo" {
		t.Fatalf("StatusMapping[open] = %q, want %q", got, "Todo")
	}
}

func TestParseCreateIssueConnectorRejectsTypeMismatch(t *testing.T) {
	_, err := ParseCreateIssueConnector(uuid.New(), Input{
		Type: "github",
		Name: "Mismatch",
		Config: ConfigInput{
			Type: "gitlab",
		},
	})
	if err == nil || err.Error() != "config.type must match type" {
		t.Fatalf("expected type mismatch error, got %v", err)
	}
}

func TestFiltersMatchesUsesIncludeExcludeStateAndAuthor(t *testing.T) {
	filters := ParseFilters(FiltersInput{
		Labels:        []string{"openase"},
		ExcludeLabels: []string{"ignore"},
		States:        []string{"open"},
		Authors:       []string{"codex"},
	})

	if !filters.Matches(ExternalIssue{
		Status: "open",
		Labels: []string{"openase", "backend"},
		Author: "Codex",
	}) {
		t.Fatalf("expected issue to match filters")
	}
	if filters.Matches(ExternalIssue{
		Status: "closed",
		Labels: []string{"openase"},
		Author: "codex",
	}) {
		t.Fatalf("expected closed issue to be rejected")
	}
	if filters.Matches(ExternalIssue{
		Status: "open",
		Labels: []string{"openase", "ignore"},
		Author: "codex",
	}) {
		t.Fatalf("expected excluded label to be rejected")
	}
}

func TestIssueConnectorLifecycleMethods(t *testing.T) {
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	lastSync := now.Add(-10 * time.Minute)
	connector := IssueConnector{
		Status: StatusActive,
		Config: Config{
			SyncDirection: SyncDirectionBidirectional,
			PollInterval:  5 * time.Minute,
		},
		LastSyncAt: &lastSync,
		Stats: SyncStats{
			TotalSynced: 3,
			Synced24h:   3,
		},
	}

	if !connector.CanPullAt(now) {
		t.Fatalf("expected connector to be due for pull")
	}
	if !connector.CanReceiveWebhook() {
		t.Fatalf("expected connector to allow webhook")
	}
	if !connector.CanSyncBack() {
		t.Fatalf("expected connector to allow sync back")
	}
	if got := connector.Config.MapStatus(" OPEN "); got != "OPEN" {
		t.Fatalf("MapStatus without mapping = %q, want original trimmed value", got)
	}

	connector.RecordSync(now, 2)
	if connector.Status != StatusActive {
		t.Fatalf("Status after success = %q, want %q", connector.Status, StatusActive)
	}
	if connector.Stats.TotalSynced != 5 || connector.Stats.Synced24h != 5 {
		t.Fatalf("unexpected sync stats after success: %+v", connector.Stats)
	}

	connector.RecordFailure(assertError("boom"))
	if connector.Status != StatusError {
		t.Fatalf("Status after failure = %q, want %q", connector.Status, StatusError)
	}
	if connector.LastError != "boom" || connector.Stats.FailedCount != 1 {
		t.Fatalf("unexpected failure state: %+v", connector)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
