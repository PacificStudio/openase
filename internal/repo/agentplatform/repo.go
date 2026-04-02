package agentplatform

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagenttoken "github.com/BetterAndBetterII/openase/ent/agenttoken"
	domain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	"github.com/google/uuid"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) AgentProjectID(ctx context.Context, agentID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.Agent.Query().
		Where(entagent.IDEQ(agentID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.UUID{}, fmt.Errorf("%w: agent", domain.ErrNotFound)
		}
		return uuid.UUID{}, err
	}
	return item.ProjectID, nil
}

func (r *EntRepository) CreateToken(
	ctx context.Context,
	agentID uuid.UUID,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	tokenHash string,
	scopes []string,
	expiresAt time.Time,
) error {
	_, err := r.client.AgentToken.Create().
		SetAgentID(agentID).
		SetProjectID(projectID).
		SetTicketID(ticketID).
		SetTokenHash(tokenHash).
		SetScopes(scopes).
		SetExpiresAt(expiresAt).
		Save(ctx)
	return err
}

func (r *EntRepository) TokenByHash(ctx context.Context, tokenHash string) (domain.StoredTokenRecord, error) {
	record, err := r.client.AgentToken.Query().
		Where(entagenttoken.TokenHashEQ(tokenHash)).
		WithAgent().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.StoredTokenRecord{}, fmt.Errorf("%w: token", domain.ErrNotFound)
		}
		return domain.StoredTokenRecord{}, err
	}
	if record.Edges.Agent == nil {
		return domain.StoredTokenRecord{}, fmt.Errorf("agent token %s is missing agent edge", record.ID)
	}

	return domain.StoredTokenRecord{
		TokenID:        record.ID,
		AgentID:        record.AgentID,
		AgentName:      record.Edges.Agent.Name,
		AgentProjectID: record.Edges.Agent.ProjectID,
		ProjectID:      record.ProjectID,
		TicketID:       record.TicketID,
		Scopes:         append([]string(nil), record.Scopes...),
		ExpiresAt:      record.ExpiresAt.UTC(),
		LastUsedAt:     cloneTimePointer(record.LastUsedAt),
	}, nil
}

func (r *EntRepository) TouchTokenLastUsed(ctx context.Context, tokenID uuid.UUID, usedAt time.Time) error {
	_, err := r.client.AgentToken.UpdateOneID(tokenID).
		SetLastUsedAt(usedAt).
		Save(ctx)
	return err
}

func (r *EntRepository) ProjectTokenInventory(ctx context.Context, projectID uuid.UUID, now time.Time) (domain.ProjectTokenInventory, error) {
	baseQuery := r.client.AgentToken.Query().Where(entagenttoken.ProjectIDEQ(projectID))

	activeTokenCount, err := baseQuery.Clone().Where(entagenttoken.ExpiresAtGTE(now)).Count(ctx)
	if err != nil {
		return domain.ProjectTokenInventory{}, fmt.Errorf("count active project tokens: %w", err)
	}

	expiredTokenCount, err := baseQuery.Clone().Where(entagenttoken.ExpiresAtLT(now)).Count(ctx)
	if err != nil {
		return domain.ProjectTokenInventory{}, fmt.Errorf("count expired project tokens: %w", err)
	}

	var lastIssuedAt *time.Time
	lastIssuedToken, err := baseQuery.Clone().
		Order(entagenttoken.ByCreatedAt(entsql.OrderDesc())).
		First(ctx)
	switch {
	case ent.IsNotFound(err):
	case err != nil:
		return domain.ProjectTokenInventory{}, fmt.Errorf("load latest project token issue: %w", err)
	default:
		lastIssuedAt = timePointer(lastIssuedToken.CreatedAt.UTC())
	}

	var lastUsedAt *time.Time
	lastUsedToken, err := baseQuery.Clone().
		Where(entagenttoken.LastUsedAtNotNil()).
		Order(entagenttoken.ByLastUsedAt(entsql.OrderDesc())).
		First(ctx)
	switch {
	case ent.IsNotFound(err):
	case err != nil:
		return domain.ProjectTokenInventory{}, fmt.Errorf("load latest project token use: %w", err)
	default:
		lastUsedAt = timePointer(lastUsedToken.LastUsedAt.UTC())
	}

	return domain.ProjectTokenInventory{
		ActiveTokenCount:  activeTokenCount,
		ExpiredTokenCount: expiredTokenCount,
		LastIssuedAt:      lastIssuedAt,
		LastUsedAt:        lastUsedAt,
	}, nil
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func timePointer(value time.Time) *time.Time {
	return &value
}
