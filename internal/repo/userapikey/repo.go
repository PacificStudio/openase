package userapikey

import (
	"context"
	"errors"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entuserapikey "github.com/BetterAndBetterII/openase/ent/userapikey"
	domain "github.com/BetterAndBetterII/openase/internal/domain/userapikey"
	"github.com/google/uuid"
)

type Repository struct {
	client *ent.Client
}

var ErrNotFound = errors.New("user api key not found")

func NewEntRepository(client *ent.Client) *Repository {
	return &Repository{client: client}
}

type CreateRecord struct {
	UserID      uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	TokenPrefix string
	TokenHint   string
	TokenHash   string
	Scopes      []string
	ExpiresAt   *time.Time
}

type RotateRecord struct {
	Name        string
	TokenPrefix string
	TokenHint   string
	TokenHash   string
	Scopes      []string
	ExpiresAt   *time.Time
	RotatedAt   time.Time
}

func (r *Repository) ListByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) ([]domain.APIKey, error) {
	items, err := r.client.UserAPIKey.Query().
		Where(
			entuserapikey.ProjectIDEQ(projectID),
			entuserapikey.UserIDEQ(userID),
			entuserapikey.StatusNEQ(entuserapikey.StatusRevoked),
		).
		Order(ent.Desc(entuserapikey.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.APIKey, 0, len(items))
	for _, item := range items {
		result = append(result, mapAPIKey(item))
	}
	return result, nil
}

func (r *Repository) Create(ctx context.Context, record CreateRecord) (domain.APIKey, error) {
	builder := r.client.UserAPIKey.Create().
		SetUserID(record.UserID).
		SetProjectID(record.ProjectID).
		SetName(record.Name).
		SetTokenPrefix(record.TokenPrefix).
		SetTokenHint(record.TokenHint).
		SetTokenHash(record.TokenHash).
		SetScopes(copyStrings(record.Scopes))
	if record.ExpiresAt != nil {
		builder.SetExpiresAt(record.ExpiresAt.UTC())
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.APIKey{}, err
	}
	return mapAPIKey(item), nil
}

func (r *Repository) GetByIDForUser(ctx context.Context, keyID, projectID, userID uuid.UUID) (domain.APIKey, error) {
	item, err := r.client.UserAPIKey.Query().
		Where(
			entuserapikey.IDEQ(keyID),
			entuserapikey.ProjectIDEQ(projectID),
			entuserapikey.UserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.APIKey{}, ErrNotFound
		}
		return domain.APIKey{}, err
	}
	return mapAPIKey(item), nil
}

func (r *Repository) Disable(ctx context.Context, keyID uuid.UUID, disabledAt time.Time) (domain.APIKey, error) {
	item, err := r.client.UserAPIKey.UpdateOneID(keyID).
		SetStatus(entuserapikey.StatusDisabled).
		SetDisabledAt(disabledAt.UTC()).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.APIKey{}, ErrNotFound
		}
		return domain.APIKey{}, err
	}
	return mapAPIKey(item), nil
}

func (r *Repository) Revoke(ctx context.Context, keyID uuid.UUID, revokedAt time.Time) error {
	_, err := r.client.UserAPIKey.UpdateOneID(keyID).
		SetStatus(entuserapikey.StatusRevoked).
		SetRevokedAt(revokedAt.UTC()).
		Save(ctx)
	if ent.IsNotFound(err) {
		return ErrNotFound
	}
	return err
}

func (r *Repository) Rotate(ctx context.Context, keyID uuid.UUID, record RotateRecord) (domain.APIKey, error) {
	builder := r.client.UserAPIKey.UpdateOneID(keyID).
		SetName(record.Name).
		SetTokenPrefix(record.TokenPrefix).
		SetTokenHint(record.TokenHint).
		SetTokenHash(record.TokenHash).
		SetScopes(copyStrings(record.Scopes)).
		SetStatus(entuserapikey.StatusActive).
		ClearDisabledAt().
		ClearRevokedAt()
	if record.ExpiresAt != nil {
		builder.SetExpiresAt(record.ExpiresAt.UTC())
	} else {
		builder.ClearExpiresAt()
	}
	item, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.APIKey{}, ErrNotFound
		}
		return domain.APIKey{}, err
	}
	return mapAPIKey(item), nil
}

func (r *Repository) TokenByHash(ctx context.Context, tokenHash string) (domain.APIKey, error) {
	item, err := r.client.UserAPIKey.Query().
		Where(entuserapikey.TokenHashEQ(tokenHash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.APIKey{}, ErrNotFound
		}
		return domain.APIKey{}, err
	}
	return mapAPIKey(item), nil
}

func (r *Repository) TouchLastUsed(ctx context.Context, keyID uuid.UUID, usedAt time.Time) error {
	_, err := r.client.UserAPIKey.UpdateOneID(keyID).
		SetLastUsedAt(usedAt.UTC()).
		Save(ctx)
	if ent.IsNotFound(err) {
		return ErrNotFound
	}
	return err
}

func mapAPIKey(item *ent.UserAPIKey) domain.APIKey {
	return domain.APIKey{
		ID:          item.ID,
		UserID:      item.UserID,
		ProjectID:   item.ProjectID,
		Name:        item.Name,
		TokenPrefix: item.TokenPrefix,
		TokenHint:   item.TokenHint,
		Scopes:      copyStrings(item.Scopes),
		Status:      domain.Status(item.Status),
		ExpiresAt:   cloneTime(item.ExpiresAt),
		LastUsedAt:  cloneTime(item.LastUsedAt),
		CreatedAt:   item.CreatedAt.UTC(),
		UpdatedAt:   item.UpdatedAt.UTC(),
		DisabledAt:  cloneTime(item.DisabledAt),
		RevokedAt:   cloneTime(item.RevokedAt),
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func copyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	return append([]string(nil), items...)
}
