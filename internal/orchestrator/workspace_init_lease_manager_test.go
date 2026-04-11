package orchestrator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWorkspaceInitLeaseManagerHeartbeatRenewsLease(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	manager := newWorkspaceInitLeaseManager(client, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	manager.leaseDuration = 100 * time.Millisecond
	manager.heartbeatInterval = 20 * time.Millisecond
	manager.waitInterval = 5 * time.Millisecond

	machineID := uuid.New()
	ownerRunID := uuid.New()
	handle, err := manager.Acquire(ctx, machineID, ownerRunID, uuid.New())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer func() {
		if releaseErr := handle.Release(context.Background()); releaseErr != nil {
			t.Fatalf("Release() error = %v", releaseErr)
		}
	}()

	time.Sleep(180 * time.Millisecond)

	record, err := manager.repo.GetByLeaseKey(ctx, workspaceInitLeaseKey(machineID))
	if err != nil {
		t.Fatalf("GetByLeaseKey() error = %v", err)
	}
	if record == nil {
		t.Fatal("expected active workspace init lease record")
	}
	if record.OwnerRunID != ownerRunID {
		t.Fatalf("owner_run_id = %s, want %s", record.OwnerRunID, ownerRunID)
	}
	if remaining := time.Until(record.LeaseExpiresAt); remaining <= 30*time.Millisecond {
		t.Fatalf("lease expiry was not renewed sufficiently, remaining=%s", remaining)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, err := manager.Acquire(waitCtx, machineID, uuid.New(), uuid.New()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire() while heartbeat is active error = %v, want context deadline exceeded", err)
	}
}

func TestWorkspaceInitLeaseManagerAllowsExpiredTakeover(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	manager := newWorkspaceInitLeaseManager(client, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	manager.leaseDuration = 100 * time.Millisecond
	manager.heartbeatInterval = 20 * time.Millisecond
	manager.waitInterval = 5 * time.Millisecond

	machineID := uuid.New()
	oldRunID := uuid.New()
	expiredAt := time.Now().UTC().Add(-time.Minute)
	if _, err := client.WorkspaceInitLease.Create().
		SetLeaseKey(workspaceInitLeaseKey(machineID)).
		SetMachineID(machineID).
		SetOwnerRunID(oldRunID).
		SetLeaseExpiresAt(expiredAt).
		SetHeartbeatAt(expiredAt).
		Save(ctx); err != nil {
		t.Fatalf("seed expired lease: %v", err)
	}

	newRunID := uuid.New()
	handle, err := manager.Acquire(ctx, machineID, newRunID, uuid.New())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer func() {
		if releaseErr := handle.Release(context.Background()); releaseErr != nil {
			t.Fatalf("Release() error = %v", releaseErr)
		}
	}()

	record, err := manager.repo.GetByLeaseKey(ctx, workspaceInitLeaseKey(machineID))
	if err != nil {
		t.Fatalf("GetByLeaseKey() error = %v", err)
	}
	if record == nil {
		t.Fatal("expected active workspace init lease record")
	}
	if record.OwnerRunID != newRunID {
		t.Fatalf("owner_run_id = %s, want %s", record.OwnerRunID, newRunID)
	}
	if !record.LeaseExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("lease_expires_at = %s, want future timestamp", record.LeaseExpiresAt)
	}
}

func TestWorkspaceInitLeaseManagerHeartbeatTimeoutUsesLeaseAwareFloor(t *testing.T) {
	manager := &workspaceInitLeaseManager{
		leaseDuration: 400 * time.Millisecond,
	}

	if got, want := manager.heartbeatTimeout(20*time.Millisecond), 200*time.Millisecond; got != want {
		t.Fatalf("heartbeatTimeout(20ms) = %s, want %s", got, want)
	}

	manager.leaseDuration = 5 * time.Minute
	if got, want := manager.heartbeatTimeout(20*time.Millisecond), minWorkspaceInitLeaseHeartbeatTimeout; got != want {
		t.Fatalf("heartbeatTimeout(20ms) with long lease = %s, want %s", got, want)
	}

	if got, want := manager.heartbeatTimeout(15*time.Second), 15*time.Second; got != want {
		t.Fatalf("heartbeatTimeout(15s) = %s, want %s", got, want)
	}
}
