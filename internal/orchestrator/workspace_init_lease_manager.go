package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	workspaceinitleaserepo "github.com/BetterAndBetterII/openase/internal/repo/workspaceinitlease"
	"github.com/google/uuid"
)

const (
	defaultWorkspacePrepareTimeout             = 5 * time.Minute
	defaultAgentSessionStartTimeout            = 30 * time.Second
	defaultWorkspaceInitLeaseDuration          = 2 * time.Minute
	defaultWorkspaceInitLeaseHeartbeatInterval = 10 * time.Second
	defaultWorkspaceInitLeaseWaitInterval      = time.Second
	defaultWorkspaceInitLeaseReleaseTimeout    = 5 * time.Second
)

type workspaceInitLeaseManager struct {
	repo              *workspaceinitleaserepo.EntRepository
	logger            *slog.Logger
	now               func() time.Time
	leaseDuration     time.Duration
	heartbeatInterval time.Duration
	waitInterval      time.Duration
	releaseTimeout    time.Duration
}

type workspaceInitLeaseHandle struct {
	manager      *workspaceInitLeaseManager
	leaseKey     string
	machineID    uuid.UUID
	ownerRunID   uuid.UUID
	ticketID     uuid.UUID
	ctx          context.Context
	cancel       context.CancelFunc
	done         chan struct{}
	releaseOnce  sync.Once
	waitDuration time.Duration
}

func newWorkspaceInitLeaseManager(client *ent.Client, logger *slog.Logger, now func() time.Time) *workspaceInitLeaseManager {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &workspaceInitLeaseManager{
		repo:              workspaceinitleaserepo.NewEntRepository(client),
		logger:            logger.With("component", "workspace-init-lease"),
		now:               now,
		leaseDuration:     defaultWorkspaceInitLeaseDuration,
		heartbeatInterval: defaultWorkspaceInitLeaseHeartbeatInterval,
		waitInterval:      defaultWorkspaceInitLeaseWaitInterval,
		releaseTimeout:    defaultWorkspaceInitLeaseReleaseTimeout,
	}
}

func (m *workspaceInitLeaseManager) syncDependencies(client *ent.Client, logger *slog.Logger, now func() time.Time) {
	if m == nil {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	m.repo = workspaceinitleaserepo.NewEntRepository(client)
	m.logger = logger.With("component", "workspace-init-lease")
	m.now = now
}

func (m *workspaceInitLeaseManager) Acquire(
	ctx context.Context,
	machineID uuid.UUID,
	ownerRunID uuid.UUID,
	ticketID uuid.UUID,
) (*workspaceInitLeaseHandle, error) {
	if m == nil || m.repo == nil {
		return nil, nil
	}
	if machineID == uuid.Nil {
		return nil, fmt.Errorf("workspace init lease machine id must not be empty")
	}
	if ownerRunID == uuid.Nil {
		return nil, fmt.Errorf("workspace init lease owner run id must not be empty")
	}

	waitStart := m.currentTime()
	leaseKey := workspaceInitLeaseKey(machineID)
	waitLogged := false

	for {
		attemptedAt := m.currentTime()
		record, acquired, err := m.repo.TryAcquire(ctx, workspaceinitleaserepo.TryAcquireInput{
			LeaseKey:       leaseKey,
			MachineID:      machineID,
			OwnerRunID:     ownerRunID,
			LeaseExpiresAt: attemptedAt.Add(m.leaseDuration),
			HeartbeatAt:    attemptedAt,
		}, attemptedAt)
		if err != nil {
			return nil, fmt.Errorf("acquire workspace init lease for machine %s: %w", machineID, err)
		}
		if acquired {
			waitDuration := m.currentTime().Sub(waitStart)
			handleCtx, cancel := context.WithCancel(ctx)
			handle := &workspaceInitLeaseHandle{
				manager:      m,
				leaseKey:     leaseKey,
				machineID:    machineID,
				ownerRunID:   ownerRunID,
				ticketID:     ticketID,
				ctx:          handleCtx,
				cancel:       cancel,
				done:         make(chan struct{}),
				waitDuration: waitDuration,
			}
			m.logger.Info(
				"workspace init lease acquired",
				"machine_id", machineID,
				"run_id", ownerRunID,
				"ticket_id", ticketID,
				"lease_key", leaseKey,
				"wait_duration", waitDuration.String(),
				"lease_expires_at", record.LeaseExpiresAt.Format(time.RFC3339Nano),
			)
			go handle.runHeartbeat()
			return handle, nil
		}

		if !waitLogged {
			waitLogged = true
			m.logger.Info(
				"workspace init lease busy",
				"machine_id", machineID,
				"run_id", ownerRunID,
				"ticket_id", ticketID,
				"lease_key", leaseKey,
				"owner_run_id", record.OwnerRunID,
				"lease_expires_at", record.LeaseExpiresAt.Format(time.RFC3339Nano),
			)
		}

		waitInterval := m.waitInterval
		if waitInterval <= 0 {
			waitInterval = defaultWorkspaceInitLeaseWaitInterval
		}
		timer := time.NewTimer(waitInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (h *workspaceInitLeaseHandle) Context() context.Context {
	if h == nil {
		return context.Background()
	}
	return h.ctx
}

func (h *workspaceInitLeaseHandle) Release(ctx context.Context) error {
	if h == nil || h.manager == nil {
		return nil
	}

	var releaseErr error
	h.releaseOnce.Do(func() {
		h.cancel()
		<-h.done

		releaseCtx := context.Background()
		if ctx != nil {
			releaseCtx = context.WithoutCancel(ctx)
		}
		timeout := h.manager.releaseTimeout
		if timeout <= 0 {
			timeout = defaultWorkspaceInitLeaseReleaseTimeout
		}
		if timeout > 0 {
			var cancel context.CancelFunc
			releaseCtx, cancel = context.WithTimeout(releaseCtx, timeout)
			defer cancel()
		}

		releaseErr = h.manager.repo.Release(releaseCtx, workspaceinitleaserepo.ReleaseInput{
			LeaseKey:   h.leaseKey,
			OwnerRunID: h.ownerRunID,
		})
		if releaseErr == nil {
			h.manager.logger.Info(
				"workspace init lease released",
				"machine_id", h.machineID,
				"run_id", h.ownerRunID,
				"ticket_id", h.ticketID,
				"lease_key", h.leaseKey,
				"wait_duration", h.waitDuration.String(),
			)
		}
	})
	return releaseErr
}

func (h *workspaceInitLeaseHandle) runHeartbeat() {
	if h == nil || h.manager == nil || h.manager.repo == nil {
		return
	}
	defer close(h.done)

	interval := h.manager.heartbeatInterval
	if interval <= 0 {
		interval = defaultWorkspaceInitLeaseHeartbeatInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			now := h.manager.currentTime()
			renewCtx, cancel := context.WithTimeout(context.WithoutCancel(h.ctx), interval)
			ok, err := h.manager.repo.Renew(renewCtx, workspaceinitleaserepo.RenewInput{
				LeaseKey:       h.leaseKey,
				OwnerRunID:     h.ownerRunID,
				LeaseExpiresAt: now.Add(h.manager.leaseDuration),
				HeartbeatAt:    now,
			})
			cancel()
			if err != nil {
				h.manager.logger.Warn(
					"workspace init lease heartbeat failed",
					"machine_id", h.machineID,
					"run_id", h.ownerRunID,
					"ticket_id", h.ticketID,
					"lease_key", h.leaseKey,
					"error", err,
				)
				h.cancel()
				return
			}
			if !ok {
				h.manager.logger.Warn(
					"workspace init lease heartbeat lost ownership",
					"machine_id", h.machineID,
					"run_id", h.ownerRunID,
					"ticket_id", h.ticketID,
					"lease_key", h.leaseKey,
				)
				h.cancel()
				return
			}
		}
	}
}

func (m *workspaceInitLeaseManager) currentTime() time.Time {
	if m == nil || m.now == nil {
		return time.Now().UTC()
	}
	return m.now().UTC()
}

func workspaceInitLeaseKey(machineID uuid.UUID) string {
	return "machine:" + strings.TrimSpace(machineID.String())
}
