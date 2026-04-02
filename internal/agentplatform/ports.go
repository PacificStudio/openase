package agentplatform

import (
	"context"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	"github.com/google/uuid"
)

type Repository interface {
	AgentPrincipal(ctx context.Context, agentID uuid.UUID) (domain.AgentPrincipal, error)
	ProjectConversationPrincipal(ctx context.Context, principalID uuid.UUID) (domain.ProjectConversationPrincipal, error)
	CreateToken(ctx context.Context, record domain.CreateTokenRecord) error
	TokenByHash(ctx context.Context, tokenHash string) (domain.StoredTokenRecord, error)
	TouchTokenLastUsed(ctx context.Context, tokenID uuid.UUID, usedAt time.Time) error
	ProjectTokenInventory(ctx context.Context, projectID uuid.UUID, now time.Time) (domain.ProjectTokenInventory, error)
}
