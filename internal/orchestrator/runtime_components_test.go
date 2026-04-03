package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type runtimeTestSession struct{}

func (runtimeTestSession) SessionID() (string, bool) { return "session-1", true }
func (runtimeTestSession) Events() <-chan agentEvent { return nil }
func (runtimeTestSession) SendPrompt(context.Context, string) (agentTurnStartResult, error) {
	return agentTurnStartResult{}, nil
}
func (runtimeTestSession) Stop(context.Context) error         { return nil }
func (runtimeTestSession) Err() error                         { return nil }
func (runtimeTestSession) Diagnostic() agentSessionDiagnostic { return agentSessionDiagnostic{} }

func TestRuntimeSessionRegistryStoresAndDrainsSessions(t *testing.T) {
	registry := newRuntimeSessionRegistry()
	runOne := uuid.New()
	runTwo := uuid.New()
	session := runtimeTestSession{}

	registry.store(runOne, session)
	registry.store(runTwo, nil)

	if got := registry.load(runOne); got == nil {
		t.Fatal("expected stored session for first run")
	}
	runIDs := registry.runIDs()
	if len(runIDs) != 2 {
		t.Fatalf("expected 2 tracked sessions, got %d", len(runIDs))
	}

	drained := registry.drain()
	if len(drained) != 2 {
		t.Fatalf("expected 2 drained sessions, got %d", len(drained))
	}
	if registry.load(runOne) != nil {
		t.Fatal("expected drain to clear first run session")
	}
	if len(registry.runIDs()) != 0 {
		t.Fatalf("expected drain to clear registry, got %+v", registry.runIDs())
	}
}

func TestRuntimeRunTrackerDeduplicatesActiveRuns(t *testing.T) {
	tracker := newRuntimeRunTracker()
	runID := uuid.New()

	if !tracker.begin(runID) {
		t.Fatal("expected first begin to claim run")
	}
	if tracker.begin(runID) {
		t.Fatal("expected duplicate begin to be rejected")
	}
	if !tracker.active(runID) {
		t.Fatal("expected run to remain active after first begin")
	}
	if len(tracker.list()) != 1 {
		t.Fatalf("expected exactly one active run, got %+v", tracker.list())
	}

	tracker.finish(runID)
	if tracker.active(runID) {
		t.Fatal("expected finish to clear active run")
	}
}

func TestRuntimeCompletionSummaryCoordinatorDeduplicatesRunScheduling(t *testing.T) {
	coordinator := newRuntimeCompletionSummaryCoordinator(nil, nil, nil, nil, nil, nil, nil, nil, 0)
	runID := uuid.New()

	if !coordinator.beginRunCompletionSummary(runID) {
		t.Fatal("expected first summary schedule to claim run")
	}
	if coordinator.beginRunCompletionSummary(runID) {
		t.Fatal("expected duplicate summary schedule to be rejected")
	}

	coordinator.endRunCompletionSummary(runID)
	if !coordinator.beginRunCompletionSummary(runID) {
		t.Fatal("expected summary run to be claimable again after end")
	}
}

func TestRuntimeLauncherComponentsTrackUpdatedClock(t *testing.T) {
	launcher := NewRuntimeLauncher(nil, nil, nil, nil, nil, nil)
	initial := time.Date(2026, 4, 3, 4, 45, 0, 0, time.UTC)
	updated := initial.Add(5 * time.Minute)

	launcher.now = func() time.Time { return initial }
	workspaceProvisioner := launcher.ensureWorkspaceProvisioner()
	summaryCoordinator := launcher.ensureCompletionSummaryCoordinator()
	if got := workspaceProvisioner.now(); !got.Equal(initial) {
		t.Fatalf("expected initial workspace provisioner clock %s, got %s", initial, got)
	}
	if got := summaryCoordinator.now(); !got.Equal(initial) {
		t.Fatalf("expected initial summary coordinator clock %s, got %s", initial, got)
	}

	launcher.now = func() time.Time { return updated }
	workspaceProvisioner = launcher.ensureWorkspaceProvisioner()
	summaryCoordinator = launcher.ensureCompletionSummaryCoordinator()
	if got := workspaceProvisioner.now(); !got.Equal(updated) {
		t.Fatalf("expected updated workspace provisioner clock %s, got %s", updated, got)
	}
	if got := summaryCoordinator.now(); !got.Equal(updated) {
		t.Fatalf("expected updated summary coordinator clock %s, got %s", updated, got)
	}
}
