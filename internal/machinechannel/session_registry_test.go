package machinechannel

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type stubSessionCloser struct {
	reasons []string
}

func (s *stubSessionCloser) Close(reason string) error {
	s.reasons = append(s.reasons, reason)
	return nil
}

func TestSessionRegistryRegisterHeartbeatAndExpire(t *testing.T) {
	machineID := uuid.New()
	now := time.Date(2026, time.April, 4, 14, 0, 0, 0, time.UTC)
	registry := NewSessionRegistry(30 * time.Second)

	firstCloser := &stubSessionCloser{}
	first, replaced := registry.Register(machineID, "session-1", now, firstCloser)
	if replaced != nil {
		t.Fatalf("expected first registration to avoid replacement, got %+v", replaced)
	}
	if first.SessionID != "session-1" || first.MachineID != machineID {
		t.Fatalf("unexpected first registration: %+v", first)
	}

	secondCloser := &stubSessionCloser{}
	second, replaced := registry.Register(machineID, "session-2", now.Add(5*time.Second), secondCloser)
	if replaced == nil || replaced.SessionID != "session-1" {
		t.Fatalf("expected second registration to replace session-1, got %+v", replaced)
	}
	if len(firstCloser.reasons) != 1 || firstCloser.reasons[0] == "" {
		t.Fatalf("expected replaced session closer to be invoked, got %+v", firstCloser.reasons)
	}
	if !second.Replaced {
		t.Fatalf("expected replacement flag on second session, got %+v", second)
	}
	if _, ok := registry.Snapshot(machineID); !ok {
		t.Fatal("expected registry snapshot to exist for current session")
	}

	heartbeatAt := now.Add(20 * time.Second)
	sessionAfterHeartbeat, ok := registry.Heartbeat("session-2", heartbeatAt)
	if !ok {
		t.Fatal("expected heartbeat to update registered session")
	}
	if !sessionAfterHeartbeat.LastHeartbeatAt.Equal(heartbeatAt.UTC()) {
		t.Fatalf("expected heartbeat timestamp %s, got %+v", heartbeatAt.UTC(), sessionAfterHeartbeat)
	}

	expired := registry.Expire(now.Add(60 * time.Second))
	if len(expired) != 1 || expired[0].SessionID != "session-2" {
		t.Fatalf("expected session-2 to expire, got %+v", expired)
	}
	if _, ok := registry.Snapshot(machineID); ok {
		t.Fatal("expected registry snapshot to be cleared after expiry")
	}
}

func TestSessionRegistryCloseAllClosesRegisteredSessions(t *testing.T) {
	now := time.Date(2026, time.April, 4, 15, 0, 0, 0, time.UTC)
	registry := NewSessionRegistry(30 * time.Second)

	firstMachineID := uuid.New()
	secondMachineID := uuid.New()
	firstCloser := &stubSessionCloser{}
	secondCloser := &stubSessionCloser{}

	registry.Register(firstMachineID, "session-1", now, firstCloser)
	registry.Register(secondMachineID, "session-2", now.Add(time.Second), secondCloser)

	closed := registry.CloseAll("server shutdown")
	if len(closed) != 2 {
		t.Fatalf("expected two closed sessions, got %+v", closed)
	}
	if len(firstCloser.reasons) != 1 || firstCloser.reasons[0] != "server shutdown" {
		t.Fatalf("expected first session close reason to be recorded, got %+v", firstCloser.reasons)
	}
	if len(secondCloser.reasons) != 1 || secondCloser.reasons[0] != "server shutdown" {
		t.Fatalf("expected second session close reason to be recorded, got %+v", secondCloser.reasons)
	}
	if _, ok := registry.Snapshot(firstMachineID); ok {
		t.Fatal("expected first session snapshot to be cleared")
	}
	if _, ok := registry.Snapshot(secondMachineID); ok {
		t.Fatal("expected second session snapshot to be cleared")
	}
	if extra := registry.CloseAll("second pass"); len(extra) != 0 {
		t.Fatalf("expected second CloseAll call to be empty, got %+v", extra)
	}
}

func TestSessionRegistryRemoveStaleSessionKeepsReplacementRegistered(t *testing.T) {
	now := time.Date(2026, time.April, 4, 16, 0, 0, 0, time.UTC)
	registry := NewSessionRegistry(30 * time.Second)
	machineID := uuid.New()

	firstCloser := &stubSessionCloser{}
	registry.Register(machineID, "session-1", now, firstCloser)

	secondCloser := &stubSessionCloser{}
	current, replaced := registry.Register(machineID, "session-2", now.Add(5*time.Second), secondCloser)
	if replaced == nil || replaced.SessionID != "session-1" {
		t.Fatalf("expected session-2 registration to replace session-1, got %+v", replaced)
	}
	if current.SessionID != "session-2" {
		t.Fatalf("expected session-2 to become current registration, got %+v", current)
	}

	if removed, ok := registry.Remove("session-1"); ok {
		t.Fatalf("expected removing stale session-1 to be ignored, got %+v", removed)
	}

	heartbeatAt := now.Add(10 * time.Second)
	snapshot, ok := registry.Heartbeat("session-2", heartbeatAt)
	if !ok {
		t.Fatal("expected session-2 heartbeat to succeed after removing stale session-1")
	}
	if snapshot.SessionID != "session-2" || !snapshot.LastHeartbeatAt.Equal(heartbeatAt.UTC()) {
		t.Fatalf("expected session-2 heartbeat snapshot at %s, got %+v", heartbeatAt.UTC(), snapshot)
	}

	currentSnapshot, ok := registry.Snapshot(machineID)
	if !ok || currentSnapshot.SessionID != "session-2" {
		t.Fatalf("expected machine snapshot to keep session-2, got %+v ok=%t", currentSnapshot, ok)
	}
	if len(secondCloser.reasons) != 0 {
		t.Fatalf("expected current session closer to stay untouched, got %+v", secondCloser.reasons)
	}
}
