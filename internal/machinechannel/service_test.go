package machinechannel

import (
	"bytes"
	"context"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	"github.com/google/uuid"
)

type fakeRepository struct {
	machine        MachineRecord
	token          TokenRecord
	issued         CreateTokenRecord
	lastUsedAt     *time.Time
	revokedTokenID uuid.UUID
	revokedAt      *time.Time
}

func (f *fakeRepository) GetMachine(_ context.Context, _ uuid.UUID) (MachineRecord, error) {
	return f.machine, nil
}

func (f *fakeRepository) IssueToken(_ context.Context, input CreateTokenRecord) (TokenRecord, error) {
	f.issued = input
	tokenID := uuid.New()
	f.machine.ChannelCredentialKind = "token"
	tokenIDText := tokenID.String()
	f.machine.ChannelTokenID = &tokenIDText
	f.token = TokenRecord{
		TokenID:   tokenID,
		MachineID: input.MachineID,
		TokenHash: input.TokenHash,
		Status:    "active",
		ExpiresAt: input.ExpiresAt.UTC(),
	}
	return f.token, nil
}

func (f *fakeRepository) TokenByHash(_ context.Context, tokenHash string) (TokenRecord, error) {
	if tokenHash != f.token.TokenHash {
		return TokenRecord{}, domain.ErrInvalidToken
	}
	return f.token, nil
}

func (f *fakeRepository) TouchTokenLastUsed(_ context.Context, tokenID uuid.UUID, usedAt time.Time) error {
	f.lastUsedAt = ptrTime(usedAt)
	f.token.TokenID = tokenID
	return nil
}

func (f *fakeRepository) RevokeToken(_ context.Context, _ uuid.UUID, tokenID uuid.UUID, revokedAt time.Time) error {
	f.revokedTokenID = tokenID
	f.revokedAt = ptrTime(revokedAt)
	f.token.Status = "revoked"
	revoked := revokedAt.UTC()
	f.token.RevokedAt = &revoked
	return nil
}

func (f *fakeRepository) RecordConnectedSession(_ context.Context, _ ConnectedSessionRecord) (MachineRecord, error) {
	return f.machine, nil
}

func (f *fakeRepository) RecordHeartbeat(_ context.Context, _ HeartbeatRecord) (MachineRecord, error) {
	return f.machine, nil
}

func (f *fakeRepository) RecordDisconnectedSession(_ context.Context, _ DisconnectedSessionRecord) (MachineRecord, error) {
	return f.machine, nil
}

func TestServiceIssueAuthenticateAndRevokeToken(t *testing.T) {
	ctx := context.Background()
	machineID := uuid.New()
	now := time.Date(2026, time.April, 4, 14, 30, 0, 0, time.UTC)
	repo := &fakeRepository{
		machine: MachineRecord{
			ID:             machineID,
			ConnectionMode: "ws_reverse",
			Status:         "online",
		},
	}
	service := NewService(repo)
	service.now = func() time.Time { return now }
	service.rng = bytes.NewReader(bytes.Repeat([]byte{0x42}, 24))

	issued, err := service.IssueToken(ctx, domain.IssueInput{MachineID: machineID, TTL: 2 * time.Hour})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	if issued.MachineID != machineID || issued.TokenID == uuid.Nil {
		t.Fatalf("unexpected issued token payload: %+v", issued)
	}
	if repo.issued.MachineID != machineID || repo.issued.TokenHash == "" {
		t.Fatalf("expected repository issue input to be recorded, got %+v", repo.issued)
	}

	claims, err := service.Authenticate(ctx, issued.Token)
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if claims.MachineID != machineID || claims.TokenID != issued.TokenID {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if repo.lastUsedAt == nil || !repo.lastUsedAt.Equal(now.UTC()) {
		t.Fatalf("expected token last_used_at to be stamped at %s, got %+v", now.UTC(), repo.lastUsedAt)
	}

	if err := service.RevokeToken(ctx, machineID, issued.TokenID); err != nil {
		t.Fatalf("RevokeToken returned error: %v", err)
	}
	if repo.revokedTokenID != issued.TokenID || repo.revokedAt == nil {
		t.Fatalf("expected revoke call to record token and time, got %+v %+v", repo.revokedTokenID, repo.revokedAt)
	}
}

func ptrTime(value time.Time) *time.Time {
	utc := value.UTC()
	return &utc
}
