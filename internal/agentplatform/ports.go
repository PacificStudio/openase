package agentplatform

import (
	"context"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	"github.com/google/uuid"
)

type Repository interface {
	AgentProjectID(ctx context.Context, agentID uuid.UUID) (uuid.UUID, error)
	CreateToken(
		ctx context.Context,
		agentID uuid.UUID,
		projectID uuid.UUID,
		ticketID uuid.UUID,
		tokenHash string,
		scopes []string,
		expiresAt time.Time,
	) error
	TokenByHash(ctx context.Context, tokenHash string) (domain.StoredTokenRecord, error)
	TouchTokenLastUsed(ctx context.Context, tokenID uuid.UUID, usedAt time.Time) error
	ProjectTokenInventory(ctx context.Context, projectID uuid.UUID, now time.Time) (domain.ProjectTokenInventory, error)
}
