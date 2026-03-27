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

func TestNormalizeStringListReturnsEmptySliceForBlankInput(t *testing.T) {
	got := normalizeStringList([]string{"", " ", "\t"})
	if got == nil {
		t.Fatalf("normalizeStringList returned nil")
	}
	if len(got) != 0 {
		t.Fatalf("normalizeStringList length = %d, want 0", len(got))
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

func TestIssueConnectorParsingHelpers(t *testing.T) {
	projectID := uuid.New()
	connectorID := uuid.New()

	updateInput, err := ParseUpdateIssueConnector(connectorID, projectID, Input{
		Type:   " custom ",
		Name:   " Custom Connector ",
		Status: " paused ",
		Config: ConfigInput{
			Type:          "custom",
			BaseURL:       " https://example.com ",
			AuthToken:     " secret ",
			ProjectRef:    " grandcx/openase ",
			PollInterval:  "10m",
			SyncDirection: " push_only ",
			Filters: FiltersInput{
				Labels:        []string{"Backend", " backend "},
				ExcludeLabels: []string{"Ignore"},
				States:        []string{"Open"},
				Authors:       []string{"Codex"},
			},
			StatusMapping: map[string]string{" OPEN ": "Todo"},
			WebhookSecret: " hook ",
			AutoWorkflow:  " coverage-rollout ",
		},
	})
	if err != nil {
		t.Fatalf("ParseUpdateIssueConnector() error = %v", err)
	}
	if updateInput.ID != connectorID || updateInput.Type != TypeCustom || updateInput.Status != StatusPaused || updateInput.Config.SyncDirection != SyncDirectionPushOnly {
		t.Fatalf("ParseUpdateIssueConnector() = %+v", updateInput)
	}
	if updateInput.Config.BaseURL != "https://example.com" || updateInput.Config.AuthToken != "secret" || updateInput.Config.ProjectRef != "grandcx/openase" || updateInput.Config.WebhookSecret != "hook" || updateInput.Config.AutoWorkflow != "coverage-rollout" {
		t.Fatalf("ParseUpdateIssueConnector() normalized config = %+v", updateInput.Config)
	}
	if len(updateInput.Config.Filters.Labels) != 1 || updateInput.Config.Filters.Labels[0] != "backend" {
		t.Fatalf("ParseUpdateIssueConnector() filters = %+v", updateInput.Config.Filters)
	}
	if updateInput.Config.StatusMapping["open"] != "Todo" {
		t.Fatalf("ParseUpdateIssueConnector() status mapping = %+v", updateInput.Config.StatusMapping)
	}

	if got, err := ParseType(" github "); err != nil || got != TypeGitHub {
		t.Fatalf("ParseType() = %q, %v; want github, nil", got, err)
	}
	if _, err := ParseType(""); err == nil {
		t.Fatal("ParseType() expected empty validation error")
	}
	if _, err := ParseType("Bad Type"); err == nil {
		t.Fatal("ParseType() expected pattern validation error")
	}
	if got, err := ParseStatus(""); err != nil || got != StatusActive {
		t.Fatalf("ParseStatus(default) = %q, %v; want active, nil", got, err)
	}
	if _, err := ParseStatus("bad"); err == nil {
		t.Fatal("ParseStatus() expected validation error")
	}
	if got, err := ParseSyncDirection(""); err != nil || got != SyncDirectionBidirectional {
		t.Fatalf("ParseSyncDirection(default) = %q, %v; want bidirectional, nil", got, err)
	}
	if _, err := ParseSyncDirection("bad"); err == nil {
		t.Fatal("ParseSyncDirection() expected validation error")
	}
	if !SyncDirectionPullOnly.AllowsPull() || SyncDirectionPullOnly.AllowsPush() || !SyncDirectionPushOnly.AllowsPush() || SyncDirectionPushOnly.AllowsSyncBack() || !SyncDirectionBidirectional.AllowsSyncBack() {
		t.Fatal("SyncDirection helper methods returned unexpected values")
	}
	if _, err := ParseCreateIssueConnector(projectID, Input{}); err == nil {
		t.Fatal("ParseCreateIssueConnector() expected type validation error")
	}
	if _, err := ParseCreateIssueConnector(projectID, Input{Type: "github", Name: " ", Config: ConfigInput{Type: "github"}}); err == nil {
		t.Fatal("ParseCreateIssueConnector() expected name validation error")
	}
	if _, err := ParseCreateIssueConnector(projectID, Input{Type: "github", Name: "ok", Status: "bad", Config: ConfigInput{Type: "github"}}); err == nil {
		t.Fatal("ParseCreateIssueConnector() expected status validation error")
	}
	if _, err := ParseCreateIssueConnector(projectID, Input{Type: "github", Name: "ok", Config: ConfigInput{Type: "github", PollInterval: "bad"}}); err == nil {
		t.Fatal("ParseCreateIssueConnector() expected config validation error")
	}
	if _, err := ParseUpdateIssueConnector(connectorID, projectID, Input{Type: "github", Name: " ", Config: ConfigInput{Type: "github"}}); err == nil {
		t.Fatal("ParseUpdateIssueConnector() expected validation error")
	}
	if _, err := ParseConfig(ConfigInput{Type: "bad type"}); err == nil {
		t.Fatal("ParseConfig() expected type validation error")
	}
	if _, err := ParseConfig(ConfigInput{Type: "github", SyncDirection: "bad"}); err == nil {
		t.Fatal("ParseConfig() expected sync direction validation error")
	}
	if _, err := ParseConfig(ConfigInput{Type: "github", PollInterval: "bad"}); err == nil {
		t.Fatal("ParseConfig() expected poll interval validation error")
	}
	if _, err := ParseConfig(ConfigInput{Type: "github", StatusMapping: map[string]string{" ": "Todo"}}); err == nil {
		t.Fatal("ParseConfig() expected status mapping validation error")
	}

	if got := (Config{StatusMapping: map[string]string{"open": "Todo"}}).MapStatus(" OPEN "); got != "Todo" {
		t.Fatalf("Config.MapStatus() = %q, want Todo", got)
	}
	if got := (Config{}).MapStatus(" "); got != "" {
		t.Fatalf("Config.MapStatus(blank) = %q, want empty", got)
	}

	now := time.Date(2026, 3, 27, 9, 0, 0, 0, time.UTC)
	connector := IssueConnector{
		Status: StatusActive,
		Config: Config{
			SyncDirection: SyncDirectionBidirectional,
			PollInterval:  5 * time.Minute,
		},
	}
	if !connector.CanPullAt(now) || !connector.CanReceiveWebhook() || !connector.CanSyncBack() {
		t.Fatalf("IssueConnector lifecycle defaults = %+v", connector)
	}
	if (IssueConnector{Status: StatusPaused, Config: Config{SyncDirection: SyncDirectionBidirectional}}).CanPullAt(now) {
		t.Fatal("CanPullAt() expected false for paused connector")
	}
	if (IssueConnector{Status: StatusActive, Config: Config{SyncDirection: SyncDirectionPushOnly}}).CanPullAt(now) {
		t.Fatal("CanPullAt() expected false for push-only connector")
	}
	if !connector.LastSyncCursor().IsZero() {
		t.Fatalf("LastSyncCursor() with nil sync = %v, want zero time", connector.LastSyncCursor())
	}
	connector.RecordSync(now, 2)
	if got := connector.LastSyncCursor(); got != now.UTC() {
		t.Fatalf("LastSyncCursor() = %v, want %v", got, now.UTC())
	}
	oldSync := now.Add(-25 * time.Hour)
	connector.LastSyncAt = &oldSync
	connector.RecordSync(now, 1)
	if connector.Stats.Synced24h != 1 {
		t.Fatalf("RecordSync() expected 24h counter reset, got %+v", connector.Stats)
	}
	connector.RecordFailure(nil)
	if connector.Stats.FailedCount != 0 {
		t.Fatalf("RecordFailure(nil) mutated stats: %+v", connector.Stats)
	}

	if got, err := parseName(" Connector "); err != nil || got != "Connector" {
		t.Fatalf("parseName() = %q, %v; want Connector, nil", got, err)
	}
	if _, err := parseName(" "); err == nil {
		t.Fatal("parseName() expected validation error")
	}
	if got, err := parsePollInterval(""); err != nil || got != defaultPollInterval {
		t.Fatalf("parsePollInterval(default) = %v, %v; want %v, nil", got, err, defaultPollInterval)
	}
	if _, err := parsePollInterval("bad"); err == nil {
		t.Fatal("parsePollInterval() expected parse error")
	}
	if _, err := parsePollInterval("0s"); err == nil {
		t.Fatal("parsePollInterval() expected positive validation error")
	}
	if got, err := parseStatusMapping(nil); err != nil || len(got) != 0 {
		t.Fatalf("parseStatusMapping(nil) = %+v, %v; want empty map, nil", got, err)
	}
	if _, err := parseStatusMapping(map[string]string{" ": "Todo"}); err == nil {
		t.Fatal("parseStatusMapping() expected key validation error")
	}
	if _, err := parseStatusMapping(map[string]string{"open": " "}); err == nil {
		t.Fatal("parseStatusMapping() expected value validation error")
	}
	if got := normalizeStringList([]string{" Foo ", "foo", "bar"}); len(got) != 2 || got[0] != "bar" || got[1] != "foo" {
		t.Fatalf("normalizeStringList() = %+v", got)
	}
	if !hasSortedIntersection([]string{"a", "c"}, []string{"b", "c"}) {
		t.Fatal("hasSortedIntersection() expected true")
	}
	if hasSortedIntersection([]string{"a"}, []string{"b"}) {
		t.Fatal("hasSortedIntersection() expected false")
	}
	if !ParseFilters(FiltersInput{}).Matches(ExternalIssue{}) {
		t.Fatal("Filters.Matches() expected empty filters to allow issue")
	}
	if ParseFilters(FiltersInput{Labels: []string{"backend"}}).Matches(ExternalIssue{Labels: []string{"docs"}}) {
		t.Fatal("Filters.Matches() expected required label miss to fail")
	}
	if ParseFilters(FiltersInput{Authors: []string{"codex"}}).Matches(ExternalIssue{Author: "someone-else"}) {
		t.Fatal("Filters.Matches() expected author mismatch to fail")
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
