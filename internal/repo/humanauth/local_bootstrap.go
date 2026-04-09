package humanauth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entlocalbootstrapauthrequest "github.com/BetterAndBetterII/openase/ent/localbootstrapauthrequest"
	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/google/uuid"
)

type CreateLocalBootstrapAuthRequestInput struct {
	CodeHash    string
	NonceHash   string
	Purpose     string
	RequestedBy string
	ExpiresAt   time.Time
}

func (r *Repository) CreateLocalBootstrapAuthRequest(
	ctx context.Context,
	input CreateLocalBootstrapAuthRequestInput,
) (domain.LocalBootstrapAuthRequest, error) {
	item, err := r.client.LocalBootstrapAuthRequest.Create().
		SetCodeHash(strings.TrimSpace(input.CodeHash)).
		SetNonceHash(strings.TrimSpace(input.NonceHash)).
		SetPurpose(strings.TrimSpace(input.Purpose)).
		SetRequestedBy(strings.TrimSpace(input.RequestedBy)).
		SetExpiresAt(input.ExpiresAt.UTC()).
		Save(ctx)
	if err != nil {
		return domain.LocalBootstrapAuthRequest{}, fmt.Errorf("create local bootstrap auth request: %w", err)
	}
	return mapLocalBootstrapAuthRequest(item), nil
}

func (r *Repository) GetLocalBootstrapAuthRequest(
	ctx context.Context,
	id uuid.UUID,
) (domain.LocalBootstrapAuthRequest, error) {
	item, err := r.client.LocalBootstrapAuthRequest.Get(ctx, id)
	if err != nil {
		return domain.LocalBootstrapAuthRequest{}, fmt.Errorf("get local bootstrap auth request: %w", err)
	}
	return mapLocalBootstrapAuthRequest(item), nil
}

func (r *Repository) MarkLocalBootstrapAuthRequestUsed(
	ctx context.Context,
	id uuid.UUID,
	sessionID uuid.UUID,
	usedAt time.Time,
) (domain.LocalBootstrapAuthRequest, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.LocalBootstrapAuthRequest{}, fmt.Errorf("start local bootstrap auth request transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	item, err := tx.LocalBootstrapAuthRequest.Get(ctx, id)
	if err != nil {
		return domain.LocalBootstrapAuthRequest{}, fmt.Errorf("get local bootstrap auth request: %w", err)
	}
	if item.UsedAt != nil {
		if err := tx.Commit(); err != nil {
			return domain.LocalBootstrapAuthRequest{}, fmt.Errorf("commit local bootstrap auth request transaction: %w", err)
		}
		return mapLocalBootstrapAuthRequest(item), nil
	}

	item, err = tx.LocalBootstrapAuthRequest.UpdateOneID(id).
		SetUsedSessionID(sessionID).
		SetUsedAt(usedAt.UTC()).
		Save(ctx)
	if err != nil {
		return domain.LocalBootstrapAuthRequest{}, fmt.Errorf("mark local bootstrap auth request used: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.LocalBootstrapAuthRequest{}, fmt.Errorf("commit local bootstrap auth request transaction: %w", err)
	}
	return mapLocalBootstrapAuthRequest(item), nil
}

func mapLocalBootstrapAuthRequest(item *ent.LocalBootstrapAuthRequest) domain.LocalBootstrapAuthRequest {
	if item == nil {
		return domain.LocalBootstrapAuthRequest{}
	}
	return domain.LocalBootstrapAuthRequest{
		ID:            item.ID,
		CodeHash:      item.CodeHash,
		NonceHash:     item.NonceHash,
		Purpose:       item.Purpose,
		RequestedBy:   item.RequestedBy,
		ExpiresAt:     item.ExpiresAt.UTC(),
		UsedSessionID: cloneUUIDPointer(item.UsedSessionID),
		UsedAt:        cloneTimePointer(item.UsedAt),
		CreatedAt:     item.CreatedAt.UTC(),
		UpdatedAt:     item.UpdatedAt.UTC(),
	}
}

func cloneUUIDPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func (r *Repository) ListExpiredUnusedLocalBootstrapAuthRequests(
	ctx context.Context,
	before time.Time,
) ([]domain.LocalBootstrapAuthRequest, error) {
	items, err := r.client.LocalBootstrapAuthRequest.Query().
		Where(
			entlocalbootstrapauthrequest.ExpiresAtLTE(before.UTC()),
			entlocalbootstrapauthrequest.UsedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expired local bootstrap auth requests: %w", err)
	}
	result := make([]domain.LocalBootstrapAuthRequest, 0, len(items))
	for _, item := range items {
		result = append(result, mapLocalBootstrapAuthRequest(item))
	}
	return result, nil
}
