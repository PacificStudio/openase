package workflow

import (
	"errors"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
)

func TestParsePlatformAccess(t *testing.T) {
	content := " tickets.list \nprojects.update\ntickets.list"

	access, err := ParsePlatformAccess(content)
	if err != nil {
		t.Fatalf("ParsePlatformAccess returned error: %v", err)
	}

	if !access.Configured {
		t.Fatalf("expected platform access to be configured")
	}
	want := []string{"tickets.list", "projects.update"}
	if len(access.Allowed) != len(want) {
		t.Fatalf("Allowed=%v, want %v", access.Allowed, want)
	}
	for index := range want {
		if access.Allowed[index] != want[index] {
			t.Fatalf("Allowed=%v, want %v", access.Allowed, want)
		}
	}
}

func TestParsePlatformAccessAbsent(t *testing.T) {
	content := ""

	access, err := ParsePlatformAccess(content)
	if err != nil {
		t.Fatalf("ParsePlatformAccess returned error: %v", err)
	}
	if access.Configured {
		t.Fatalf("expected platform access to be absent, got %+v", access)
	}
	if len(access.Allowed) != 0 {
		t.Fatalf("expected no allowed scopes, got %v", access.Allowed)
	}
}

func TestResolveWorkflowPlatformAccessAllowedDefaultsWhenEmpty(t *testing.T) {
	got, err := resolveWorkflowPlatformAccessAllowed(nil)
	if err != nil {
		t.Fatalf("resolveWorkflowPlatformAccessAllowed(nil) error: %v", err)
	}
	want := []string{"tickets.create", "tickets.list", "tickets.report_usage", "tickets.update.self"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestResolveWorkflowPlatformAccessAllowedRejectsUnsupportedScope(t *testing.T) {
	_, err := resolveWorkflowPlatformAccessAllowed([]string{"skills.list", "bad.scope"})
	if !errors.Is(err, ErrHarnessInvalid) {
		t.Fatalf("resolveWorkflowPlatformAccessAllowed(invalid) error = %v, want %v", err, ErrHarnessInvalid)
	}
}

func TestResolveWorkflowPlatformAccessAllowedDeduplicatesSupportedValues(t *testing.T) {
	got, err := resolveWorkflowPlatformAccessAllowed([]string{" skills.list ", "skills.list", "workflows.read"})
	if err != nil {
		t.Fatalf("resolveWorkflowPlatformAccessAllowed(valid) error: %v", err)
	}
	want := []string{"skills.list", "workflows.read"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestResolveWorkflowPlatformAccessAllowedAcceptsEverySupportedScope(t *testing.T) {
	for _, scope := range agentplatform.SupportedScopes() {
		t.Run(scope, func(t *testing.T) {
			got, err := resolveWorkflowPlatformAccessAllowed([]string{scope})
			if err != nil {
				t.Fatalf("resolveWorkflowPlatformAccessAllowed(%q) error: %v", scope, err)
			}
			if len(got) != 1 || got[0] != scope {
				t.Fatalf("got %v, want [%s]", got, scope)
			}
		})
	}
}
