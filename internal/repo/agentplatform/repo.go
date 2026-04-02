package agentplatform

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagenttoken "github.com/BetterAndBetterII/openase/ent/agenttoken"
	entprojectconversationprincipal "github.com/BetterAndBetterII/openase/ent/projectconversationprincipal"
	domain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	"github.com/google/uuid"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) AgentPrincipal(ctx context.Context, agentID uuid.UUID) (domain.AgentPrincipal, error) {
	item, err := r.client.Agent.Query().
		Where(entagent.IDEQ(agentID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.AgentPrincipal{}, fmt.Errorf("%w: agent", domain.ErrNotFound)
		}
		return domain.AgentPrincipal{}, err
	}
	return domain.AgentPrincipal{
		ID:        item.ID,
		Name:      item.Name,
		ProjectID: item.ProjectID,
	}, nil
}

func (r *EntRepository) ProjectConversationPrincipal(ctx context.Context, principalID uuid.UUID) (domain.ProjectConversationPrincipal, error) {
	item, err := r.client.ProjectConversationPrincipal.Query().
		Where(entprojectconversationprincipal.IDEQ(principalID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ProjectConversationPrincipal{}, fmt.Errorf("%w: principal", domain.ErrNotFound)
		}
		return domain.ProjectConversationPrincipal{}, err
	}
	return domain.ProjectConversationPrincipal{
		ID:             item.ID,
		Name:           item.Name,
		ProjectID:      item.ProjectID,
		ConversationID: item.ConversationID,
	}, nil
}

func (r *EntRepository) CreateToken(ctx context.Context, record domain.CreateTokenRecord) error {
	builder := r.client.AgentToken.Create().
		SetProjectID(record.ProjectID).
		SetPrincipalKind(entagenttoken.PrincipalKind(record.PrincipalKind)).
		SetPrincipalID(record.PrincipalID).
		SetPrincipalName(record.PrincipalName).
		SetTokenHash(record.TokenHash).
		SetScopes(copyStrings(record.Scopes)).
		SetExpiresAt(record.ExpiresAt)
	if record.AgentID != nil {
		builder.SetAgentID(*record.AgentID)
	}
	if record.TicketID != nil {
		builder.SetTicketID(*record.TicketID)
	}
	if record.ConversationID != nil {
		builder.SetConversationID(*record.ConversationID)
	}
	_, err := builder.Save(ctx)
	return err
}

func (r *EntRepository) TokenByHash(ctx context.Context, tokenHash string) (domain.StoredTokenRecord, error) {
	record, err := r.client.AgentToken.Query().
		Where(entagenttoken.TokenHashEQ(tokenHash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.StoredTokenRecord{}, fmt.Errorf("%w: token", domain.ErrNotFound)
		}
		return domain.StoredTokenRecord{}, err
	}

	return domain.StoredTokenRecord{
		TokenID:        record.ID,
		AgentID:        cloneUUIDPointer(record.AgentID),
		ProjectID:      record.ProjectID,
		TicketID:       cloneUUIDPointer(record.TicketID),
		ConversationID: cloneUUIDPointer(record.ConversationID),
		PrincipalKind:  domain.PrincipalKind(record.PrincipalKind),
		PrincipalID:    record.PrincipalID,
		PrincipalName:  record.PrincipalName,
		Scopes:         copyStrings(record.Scopes),
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

func cloneUUIDPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func copyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	return append([]string(nil), items...)
}
