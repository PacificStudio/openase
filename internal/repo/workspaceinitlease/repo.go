package workspaceinitlease

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entworkspaceinitlease "github.com/BetterAndBetterII/openase/ent/workspaceinitlease"
	"github.com/google/uuid"
)

type LeaseRecord struct {
	ID             uuid.UUID
	LeaseKey       string
	MachineID      uuid.UUID
	OwnerRunID     uuid.UUID
	LeaseExpiresAt time.Time
	HeartbeatAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TryAcquireInput struct {
	LeaseKey       string
	MachineID      uuid.UUID
	OwnerRunID     uuid.UUID
	LeaseExpiresAt time.Time
	HeartbeatAt    time.Time
}

type RenewInput struct {
	LeaseKey       string
	OwnerRunID     uuid.UUID
	LeaseExpiresAt time.Time
	HeartbeatAt    time.Time
}

type ReleaseInput struct {
	LeaseKey   string
	OwnerRunID uuid.UUID
}

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) TryAcquire(ctx context.Context, input TryAcquireInput, now time.Time) (LeaseRecord, bool, error) {
	if r == nil || r.client == nil {
		return LeaseRecord{}, false, fmt.Errorf("workspace init lease repository unavailable")
	}

	item, err := r.client.WorkspaceInitLease.Create().
		SetLeaseKey(strings.TrimSpace(input.LeaseKey)).
		SetMachineID(input.MachineID).
		SetOwnerRunID(input.OwnerRunID).
		SetLeaseExpiresAt(input.LeaseExpiresAt.UTC()).
		SetHeartbeatAt(input.HeartbeatAt.UTC()).
		Save(ctx)
	switch {
	case err == nil:
		return mapLeaseRecord(item), true, nil
	case !ent.IsConstraintError(err):
		return LeaseRecord{}, false, fmt.Errorf("create workspace init lease %s: %w", input.LeaseKey, err)
	}

	updated, err := r.client.WorkspaceInitLease.Update().
		Where(
			entworkspaceinitlease.LeaseKeyEQ(strings.TrimSpace(input.LeaseKey)),
			entworkspaceinitlease.Or(
				entworkspaceinitlease.OwnerRunIDEQ(input.OwnerRunID),
				entworkspaceinitlease.LeaseExpiresAtLTE(now.UTC()),
			),
		).
		SetMachineID(input.MachineID).
		SetOwnerRunID(input.OwnerRunID).
		SetLeaseExpiresAt(input.LeaseExpiresAt.UTC()).
		SetHeartbeatAt(input.HeartbeatAt.UTC()).
		Save(ctx)
	if err != nil {
		return LeaseRecord{}, false, fmt.Errorf("take over workspace init lease %s: %w", input.LeaseKey, err)
	}
	if updated == 0 {
		item, err := r.GetByLeaseKey(ctx, input.LeaseKey)
		if err != nil {
			return LeaseRecord{}, false, err
		}
		if item == nil {
			return LeaseRecord{}, false, nil
		}
		return *item, false, nil
	}

	item, err = r.client.WorkspaceInitLease.Query().
		Where(entworkspaceinitlease.LeaseKeyEQ(strings.TrimSpace(input.LeaseKey))).
		Only(ctx)
	if err != nil {
		return LeaseRecord{}, false, fmt.Errorf("reload acquired workspace init lease %s: %w", input.LeaseKey, err)
	}
	return mapLeaseRecord(item), true, nil
}

func (r *EntRepository) Renew(ctx context.Context, input RenewInput) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("workspace init lease repository unavailable")
	}

	updated, err := r.client.WorkspaceInitLease.Update().
		Where(
			entworkspaceinitlease.LeaseKeyEQ(strings.TrimSpace(input.LeaseKey)),
			entworkspaceinitlease.OwnerRunIDEQ(input.OwnerRunID),
		).
		SetLeaseExpiresAt(input.LeaseExpiresAt.UTC()).
		SetHeartbeatAt(input.HeartbeatAt.UTC()).
		Save(ctx)
	if err != nil {
		return false, fmt.Errorf("renew workspace init lease %s: %w", input.LeaseKey, err)
	}
	return updated == 1, nil
}

func (r *EntRepository) Release(ctx context.Context, input ReleaseInput) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("workspace init lease repository unavailable")
	}

	if _, err := r.client.WorkspaceInitLease.Delete().
		Where(
			entworkspaceinitlease.LeaseKeyEQ(strings.TrimSpace(input.LeaseKey)),
			entworkspaceinitlease.OwnerRunIDEQ(input.OwnerRunID),
		).
		Exec(ctx); err != nil {
		return fmt.Errorf("release workspace init lease %s: %w", input.LeaseKey, err)
	}
	return nil
}

func (r *EntRepository) GetByLeaseKey(ctx context.Context, leaseKey string) (*LeaseRecord, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("workspace init lease repository unavailable")
	}

	item, err := r.client.WorkspaceInitLease.Query().
		Where(entworkspaceinitlease.LeaseKeyEQ(strings.TrimSpace(leaseKey))).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load workspace init lease %s: %w", leaseKey, err)
	}
	record := mapLeaseRecord(item)
	return &record, nil
}

func mapLeaseRecord(item *ent.WorkspaceInitLease) LeaseRecord {
	return LeaseRecord{
		ID:             item.ID,
		LeaseKey:       item.LeaseKey,
		MachineID:      item.MachineID,
		OwnerRunID:     item.OwnerRunID,
		LeaseExpiresAt: item.LeaseExpiresAt.UTC(),
		HeartbeatAt:    item.HeartbeatAt.UTC(),
		CreatedAt:      item.CreatedAt.UTC(),
		UpdatedAt:      item.UpdatedAt.UTC(),
	}
}
