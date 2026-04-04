package machinechannel

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	"github.com/google/uuid"
)

const (
	DefaultHeartbeatInterval = 15 * time.Second
	DefaultHeartbeatTimeout  = 45 * time.Second
	DefaultReconnectBackoff  = 5 * time.Second
)

type Repository interface {
	GetMachine(ctx context.Context, machineID uuid.UUID) (MachineRecord, error)
	IssueToken(ctx context.Context, input CreateTokenRecord) (TokenRecord, error)
	TokenByHash(ctx context.Context, tokenHash string) (TokenRecord, error)
	TouchTokenLastUsed(ctx context.Context, tokenID uuid.UUID, usedAt time.Time) error
	RevokeToken(ctx context.Context, machineID uuid.UUID, tokenID uuid.UUID, revokedAt time.Time) error
	RecordConnectedSession(ctx context.Context, input ConnectedSessionRecord) (MachineRecord, error)
	RecordHeartbeat(ctx context.Context, input HeartbeatRecord) (MachineRecord, error)
	RecordDisconnectedSession(ctx context.Context, input DisconnectedSessionRecord) (MachineRecord, error)
}

type MachineRecord struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	Name                  string
	ConnectionMode        string
	Status                string
	ChannelCredentialKind string
	ChannelTokenID        *string
}

type CreateTokenRecord struct {
	MachineID uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

type TokenRecord struct {
	TokenID   uuid.UUID
	MachineID uuid.UUID
	TokenHash string
	Status    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type ConnectedSessionRecord struct {
	MachineID        uuid.UUID
	SessionID        string
	ConnectedAt      time.Time
	SystemInfo       domain.SystemInfo
	ToolInventory    []domain.ToolInfo
	ResourceSnapshot *domain.ResourceSnapshot
}

type HeartbeatRecord struct {
	MachineID        uuid.UUID
	SessionID        string
	HeartbeatAt      time.Time
	SystemInfo       *domain.SystemInfo
	ToolInventory    []domain.ToolInfo
	ResourceSnapshot *domain.ResourceSnapshot
}

type DisconnectedSessionRecord struct {
	MachineID      uuid.UUID
	SessionID      string
	DisconnectedAt time.Time
	Reason         string
}

type Service struct {
	repo Repository
	now  func() time.Time
	rng  io.Reader
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
		rng:  rand.Reader,
	}
}

func (s *Service) IssueToken(ctx context.Context, input domain.IssueInput) (domain.IssuedToken, error) {
	if s == nil || s.repo == nil {
		return domain.IssuedToken{}, fmt.Errorf("machine channel service unavailable")
	}
	machine, err := s.repo.GetMachine(ctx, input.MachineID)
	if err != nil {
		return domain.IssuedToken{}, err
	}
	if err := validateMachineForChannel(machine); err != nil {
		return domain.IssuedToken{}, err
	}
	rawToken, tokenHash, err := generateToken(s.rng)
	if err != nil {
		return domain.IssuedToken{}, err
	}
	expiresAt := s.now().UTC().Add(resolveTTL(input.TTL))
	record, err := s.repo.IssueToken(ctx, CreateTokenRecord{
		MachineID: machine.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return domain.IssuedToken{}, fmt.Errorf("create machine channel token: %w", err)
	}
	return domain.IssuedToken{
		Token:     rawToken,
		TokenID:   record.TokenID,
		MachineID: machine.ID,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (domain.Claims, error) {
	if s == nil || s.repo == nil {
		return domain.Claims{}, fmt.Errorf("machine channel service unavailable")
	}
	tokenText, err := domain.ParseToken(rawToken)
	if err != nil {
		return domain.Claims{}, err
	}
	record, err := s.repo.TokenByHash(ctx, hashToken(tokenText))
	if err != nil {
		return domain.Claims{}, err
	}
	if record.Status == "revoked" {
		return domain.Claims{}, domain.ErrTokenRevoked
	}
	if record.ExpiresAt.Before(s.now().UTC()) {
		return domain.Claims{}, domain.ErrTokenExpired
	}
	machine, err := s.repo.GetMachine(ctx, record.MachineID)
	if err != nil {
		return domain.Claims{}, err
	}
	if err := validateMachineForChannel(machine); err != nil {
		return domain.Claims{}, err
	}
	if machine.ChannelCredentialKind != "token" || machine.ChannelTokenID == nil || strings.TrimSpace(*machine.ChannelTokenID) != record.TokenID.String() {
		return domain.Claims{}, domain.ErrInvalidToken
	}
	if err := s.repo.TouchTokenLastUsed(ctx, record.TokenID, s.now().UTC()); err != nil {
		return domain.Claims{}, fmt.Errorf("touch machine channel token: %w", err)
	}
	return domain.Claims{
		TokenID:   record.TokenID,
		MachineID: record.MachineID,
		ExpiresAt: record.ExpiresAt.UTC(),
	}, nil
}

func (s *Service) RevokeToken(ctx context.Context, machineID uuid.UUID, tokenID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("machine channel service unavailable")
	}
	if machineID == uuid.Nil {
		return fmt.Errorf("machine_id must be a valid UUID")
	}
	if tokenID == uuid.Nil {
		return fmt.Errorf("token_id must be a valid UUID")
	}
	return s.repo.RevokeToken(ctx, machineID, tokenID, s.now().UTC())
}

func (s *Service) RecordConnectedSession(ctx context.Context, input ConnectedSessionRecord) (MachineRecord, error) {
	if s == nil || s.repo == nil {
		return MachineRecord{}, fmt.Errorf("machine channel service unavailable")
	}
	return s.repo.RecordConnectedSession(ctx, input)
}

func (s *Service) RecordHeartbeat(ctx context.Context, input HeartbeatRecord) (MachineRecord, error) {
	if s == nil || s.repo == nil {
		return MachineRecord{}, fmt.Errorf("machine channel service unavailable")
	}
	return s.repo.RecordHeartbeat(ctx, input)
}

func (s *Service) RecordDisconnectedSession(ctx context.Context, input DisconnectedSessionRecord) (MachineRecord, error) {
	if s == nil || s.repo == nil {
		return MachineRecord{}, fmt.Errorf("machine channel service unavailable")
	}
	return s.repo.RecordDisconnectedSession(ctx, input)
}

func resolveTTL(raw time.Duration) time.Duration {
	if raw > 0 {
		return raw
	}
	return 24 * time.Hour
}

func validateMachineForChannel(machine MachineRecord) error {
	switch strings.TrimSpace(machine.ConnectionMode) {
	case "ws_reverse":
	default:
		return domain.ErrConnectionMode
	}
	if strings.EqualFold(strings.TrimSpace(machine.Status), "maintenance") {
		return domain.ErrMachineDisabled
	}
	return nil
}

func generateToken(rng io.Reader) (string, string, error) {
	bytes := make([]byte, 24)
	if _, err := io.ReadFull(rng, bytes); err != nil {
		return "", "", fmt.Errorf("generate machine token bytes: %w", err)
	}
	token := domain.TokenPrefix + base64.RawURLEncoding.EncodeToString(bytes)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
