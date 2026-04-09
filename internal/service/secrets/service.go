package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	repo "github.com/BetterAndBetterII/openase/internal/repo/secrets"
	"github.com/google/uuid"
)

var (
	ErrUnavailable        = errors.New("secret service unavailable")
	ErrInvalidInput       = errors.New("invalid secret input")
	ErrSecretNotFound     = errors.New("secret not found")
	ErrSecretNameConflict = errors.New("secret name already exists at this scope")
	ErrBindingNotFound    = errors.New("secret binding not found")
	ErrBindingConflict    = errors.New("secret binding already exists at this scope")
	ErrBindingTarget      = errors.New("secret binding target not found in project")
)

type Manager interface {
	ListProjectSecretInventory(ctx context.Context, projectID uuid.UUID) ([]domain.InventorySecret, error)
	ListOrganizationSecretInventory(ctx context.Context, organizationID uuid.UUID) ([]domain.InventorySecret, error)
	ListProjectBindings(ctx context.Context, projectID uuid.UUID) ([]domain.BindingRecord, error)
	CreateSecret(ctx context.Context, input CreateSecretInput) (domain.Secret, error)
	CreateBinding(ctx context.Context, input CreateBindingInput) (domain.BindingRecord, error)
	CreateOrganizationSecret(ctx context.Context, input CreateOrganizationSecretInput) (domain.Secret, error)
	UpdateSecretMetadata(ctx context.Context, input UpdateSecretMetadataInput) (domain.Secret, error)
	RotateSecret(ctx context.Context, input RotateSecretInput) (domain.Secret, error)
	RotateOrganizationSecret(ctx context.Context, input RotateOrganizationSecretInput) (domain.Secret, error)
	DisableSecret(ctx context.Context, input DisableSecretInput) (domain.Secret, error)
	DeleteBinding(ctx context.Context, input DeleteBindingInput) error
	DisableOrganizationSecret(ctx context.Context, input DisableOrganizationSecretInput) (domain.Secret, error)
	DeleteSecret(ctx context.Context, input DeleteSecretInput) error
	DeleteOrganizationSecret(ctx context.Context, input DeleteOrganizationSecretInput) error
	ResolveForRuntime(ctx context.Context, input ResolveRuntimeInput) ([]domain.ResolvedSecret, []string, error)
	ResolveBoundForRuntime(ctx context.Context, input ResolveBoundRuntimeInput) ([]domain.ResolvedSecret, error)
}

type CreateSecretInput struct {
	ProjectID   uuid.UUID
	Scope       string
	Name        string
	Kind        string
	Description string
	Value       string
}

type CreateOrganizationSecretInput struct {
	OrganizationID uuid.UUID
	Name           string
	Kind           string
	Description    string
	Value          string
}

type UpdateSecretMetadataInput struct {
	ProjectID   uuid.UUID
	SecretID    uuid.UUID
	Name        *string
	Description *string
}

type RotateSecretInput struct {
	ProjectID uuid.UUID
	SecretID  uuid.UUID
	Value     string
}

type RotateOrganizationSecretInput struct {
	OrganizationID uuid.UUID
	SecretID       uuid.UUID
	Value          string
}

type DisableSecretInput struct {
	ProjectID uuid.UUID
	SecretID  uuid.UUID
}

type CreateBindingInput struct {
	ProjectID       uuid.UUID
	SecretID        uuid.UUID
	Scope           string
	ScopeResourceID uuid.UUID
	BindingKey      string
}

type DeleteBindingInput struct {
	ProjectID uuid.UUID
	BindingID uuid.UUID
}

type DisableOrganizationSecretInput struct {
	OrganizationID uuid.UUID
	SecretID       uuid.UUID
}

type DeleteSecretInput struct {
	ProjectID uuid.UUID
	SecretID  uuid.UUID
}

type DeleteOrganizationSecretInput struct {
	OrganizationID uuid.UUID
	SecretID       uuid.UUID
}

type ResolveRuntimeInput struct {
	ProjectID   uuid.UUID
	BindingKeys []string
	TicketID    *uuid.UUID
	WorkflowID  *uuid.UUID
	AgentID     *uuid.UUID
}

type ResolveBoundRuntimeInput struct {
	ProjectID  uuid.UUID
	TicketID   *uuid.UUID
	WorkflowID *uuid.UUID
	AgentID    *uuid.UUID
}

type Service struct {
	repo  repo.Repository
	block cipher.Block
	now   func() time.Time
}

func New(repository repo.Repository, cipherSeed string) (*Service, error) {
	if repository == nil {
		return nil, errors.New("secret repository is required")
	}
	keyMaterial, err := base64.StdEncoding.DecodeString(domain.DefaultCipherSeed(cipherSeed))
	if err != nil {
		return nil, fmt.Errorf("derive secret cipher key: %w", err)
	}
	block, err := aes.NewCipher(keyMaterial)
	if err != nil {
		return nil, fmt.Errorf("create secret cipher: %w", err)
	}
	return &Service{repo: repository, block: block, now: time.Now}, nil
}

func (s *Service) ListProjectSecretInventory(ctx context.Context, projectID uuid.UUID) ([]domain.InventorySecret, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ListProjectSecretInventory(ctx, projectID)
}

func (s *Service) ListOrganizationSecretInventory(ctx context.Context, organizationID uuid.UUID) ([]domain.InventorySecret, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ListOrganizationSecretInventory(ctx, organizationID)
}

func (s *Service) ListProjectBindings(ctx context.Context, projectID uuid.UUID) ([]domain.BindingRecord, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}
	items, err := s.repo.ListBindings(ctx, projectID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	return items, nil
}

func (s *Service) CreateSecret(ctx context.Context, input CreateSecretInput) (domain.Secret, error) {
	if s.repo == nil {
		return domain.Secret{}, ErrUnavailable
	}
	scope, err := domain.ParseScopeKind(input.Scope)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	kind, err := domain.ParseKind(input.Kind)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	name, err := domain.NormalizeName(input.Name)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	storedValue, err := s.sealValue(input.Value)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	projectContext, err := s.repo.GetProjectContext(ctx, input.ProjectID)
	if err != nil {
		return domain.Secret{}, err
	}
	item := domain.Secret{
		OrganizationID: projectContext.OrganizationID,
		ProjectID:      domain.ProjectIDForSecretScope(scope, input.ProjectID),
		Scope:          scope,
		Name:           name,
		Kind:           kind,
		Description:    strings.TrimSpace(input.Description),
		StoredValue:    storedValue,
	}
	created, err := s.repo.CreateSecret(ctx, item)
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	return created, nil
}

func (s *Service) CreateBinding(ctx context.Context, input CreateBindingInput) (domain.BindingRecord, error) {
	if s.repo == nil {
		return domain.BindingRecord{}, ErrUnavailable
	}
	scope, err := domain.ParseRuntimeBindingScopeKind(input.Scope)
	if err != nil {
		return domain.BindingRecord{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	key, err := domain.NormalizeName(input.BindingKey)
	if err != nil {
		return domain.BindingRecord{}, fmt.Errorf("%w: binding_key: %s", ErrInvalidInput, err)
	}
	secret, err := s.repo.GetSecret(ctx, input.ProjectID, input.SecretID)
	if err != nil {
		return domain.BindingRecord{}, mapRepositoryError(err)
	}
	target, err := s.repo.GetBindingTarget(ctx, input.ProjectID, scope, input.ScopeResourceID)
	if err != nil {
		return domain.BindingRecord{}, mapRepositoryError(err)
	}
	projectContext, err := s.repo.GetProjectContext(ctx, input.ProjectID)
	if err != nil {
		return domain.BindingRecord{}, mapRepositoryError(err)
	}
	binding := domain.Binding{
		OrganizationID:  projectContext.OrganizationID,
		ProjectID:       domain.ProjectIDForBindingScope(scope, input.ProjectID),
		SecretID:        secret.ID,
		Scope:           scope,
		ScopeResourceID: target.ID,
		BindingKey:      key,
	}
	created, err := s.repo.CreateBinding(ctx, binding)
	if err != nil {
		return domain.BindingRecord{}, mapRepositoryError(err)
	}
	return domain.BindingRecord{
		Binding: created,
		Secret:  secret,
		Target:  target,
	}, nil
}

func (s *Service) CreateOrganizationSecret(ctx context.Context, input CreateOrganizationSecretInput) (domain.Secret, error) {
	if s.repo == nil {
		return domain.Secret{}, ErrUnavailable
	}
	name, err := domain.NormalizeName(input.Name)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	kind, err := domain.ParseKind(input.Kind)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	storedValue, err := s.sealValue(input.Value)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	created, err := s.repo.CreateSecret(ctx, domain.Secret{
		OrganizationID: input.OrganizationID,
		ProjectID:      uuid.Nil,
		Scope:          domain.ScopeKindOrganization,
		Name:           name,
		Kind:           kind,
		Description:    strings.TrimSpace(input.Description),
		StoredValue:    storedValue,
	})
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	return created, nil
}

func (s *Service) UpdateSecretMetadata(ctx context.Context, input UpdateSecretMetadataInput) (domain.Secret, error) {
	if s.repo == nil {
		return domain.Secret{}, ErrUnavailable
	}
	current, err := s.repo.GetSecret(ctx, input.ProjectID, input.SecretID)
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	name := current.Name
	if input.Name != nil {
		name, err = domain.NormalizeName(*input.Name)
		if err != nil {
			return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
		}
	}
	description := current.Description
	if input.Description != nil {
		description = strings.TrimSpace(*input.Description)
	}
	updated, err := s.repo.UpdateSecretMetadata(ctx, input.ProjectID, input.SecretID, name, description)
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	return updated, nil
}

func (s *Service) RotateSecret(ctx context.Context, input RotateSecretInput) (domain.Secret, error) {
	if s.repo == nil {
		return domain.Secret{}, ErrUnavailable
	}
	storedValue, err := s.sealValue(input.Value)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	updated, err := s.repo.RotateSecret(ctx, input.ProjectID, input.SecretID, storedValue)
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	return updated, nil
}

func (s *Service) RotateOrganizationSecret(ctx context.Context, input RotateOrganizationSecretInput) (domain.Secret, error) {
	if s.repo == nil {
		return domain.Secret{}, ErrUnavailable
	}
	storedValue, err := s.sealValue(input.Value)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	updated, err := s.repo.RotateOrganizationSecret(ctx, input.OrganizationID, input.SecretID, storedValue)
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	return updated, nil
}

func (s *Service) DisableSecret(ctx context.Context, input DisableSecretInput) (domain.Secret, error) {
	if s.repo == nil {
		return domain.Secret{}, ErrUnavailable
	}
	updated, err := s.repo.DisableSecret(ctx, input.ProjectID, input.SecretID, s.now().UTC())
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	return updated, nil
}

func (s *Service) DeleteBinding(ctx context.Context, input DeleteBindingInput) error {
	if s.repo == nil {
		return ErrUnavailable
	}
	return mapRepositoryError(s.repo.DeleteBinding(ctx, input.ProjectID, input.BindingID))
}

func (s *Service) DisableOrganizationSecret(ctx context.Context, input DisableOrganizationSecretInput) (domain.Secret, error) {
	if s.repo == nil {
		return domain.Secret{}, ErrUnavailable
	}
	updated, err := s.repo.DisableOrganizationSecret(ctx, input.OrganizationID, input.SecretID, s.now().UTC())
	if err != nil {
		return domain.Secret{}, mapRepositoryError(err)
	}
	return updated, nil
}

func (s *Service) DeleteSecret(ctx context.Context, input DeleteSecretInput) error {
	if s.repo == nil {
		return ErrUnavailable
	}
	return mapRepositoryError(s.repo.DeleteSecret(ctx, input.ProjectID, input.SecretID))
}

func (s *Service) DeleteOrganizationSecret(ctx context.Context, input DeleteOrganizationSecretInput) error {
	if s.repo == nil {
		return ErrUnavailable
	}
	return mapRepositoryError(s.repo.DeleteOrganizationSecret(ctx, input.OrganizationID, input.SecretID))
}

func (s *Service) ResolveForRuntime(ctx context.Context, input ResolveRuntimeInput) ([]domain.ResolvedSecret, []string, error) {
	if s.repo == nil {
		return nil, nil, ErrUnavailable
	}
	keys, err := domain.ParseBindingKeys(input.BindingKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	candidates, err := s.repo.ListResolutionCandidates(ctx, input.ProjectID, keys, input.TicketID, input.WorkflowID, input.AgentID)
	if err != nil {
		return nil, nil, err
	}
	return s.resolveCandidates(keys, candidates)
}

func (s *Service) ResolveBoundForRuntime(ctx context.Context, input ResolveBoundRuntimeInput) ([]domain.ResolvedSecret, error) {
	if s.repo == nil {
		return nil, ErrUnavailable
	}
	candidates, err := s.repo.ListResolutionCandidates(ctx, input.ProjectID, nil, input.TicketID, input.WorkflowID, input.AgentID)
	if err != nil {
		return nil, err
	}
	keys, err := domain.BindingKeysFromCandidates(candidates)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}
	resolved, _, err := s.resolveCandidates(keys, candidates)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func (s *Service) sealValue(raw string) (domain.StoredValue, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return domain.StoredValue{}, errors.New("value must not be empty")
	}
	gcm, err := cipher.NewGCM(s.block)
	if err != nil {
		return domain.StoredValue{}, fmt.Errorf("create secret AEAD: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return domain.StoredValue{}, fmt.Errorf("generate secret nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(trimmed), nil)
	return domain.StoredValue{
		Algorithm:  domain.CipherAlgorithmAES256GCM,
		KeySource:  domain.KeySourceDatabaseDSNSHA256,
		KeyID:      domain.DefaultKeyID,
		Preview:    domain.RedactValue(trimmed),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		RotatedAt:  s.now().UTC(),
	}, nil
}

func (s *Service) decryptValue(stored domain.StoredValue) (string, error) {
	if strings.TrimSpace(stored.Algorithm) != domain.CipherAlgorithmAES256GCM {
		return "", fmt.Errorf("unsupported secret algorithm %q", stored.Algorithm)
	}
	gcm, err := cipher.NewGCM(s.block)
	if err != nil {
		return "", fmt.Errorf("create secret AEAD: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(strings.TrimSpace(stored.Nonce))
	if err != nil {
		return "", fmt.Errorf("decode secret nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(stored.Ciphertext))
	if err != nil {
		return "", fmt.Errorf("decode secret ciphertext: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return strings.TrimSpace(string(plaintext)), nil
}

func mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repo.ErrSecretNotFound) {
		return ErrSecretNotFound
	}
	if errors.Is(err, repo.ErrSecretNameConflict) {
		return ErrSecretNameConflict
	}
	if errors.Is(err, repo.ErrBindingNotFound) {
		return ErrBindingNotFound
	}
	if errors.Is(err, repo.ErrBindingConflict) {
		return ErrBindingConflict
	}
	if errors.Is(err, repo.ErrBindingTargetNotFound) {
		return ErrBindingTarget
	}
	return err
}

func (s *Service) resolveCandidates(keys []string, candidates []domain.Candidate) ([]domain.ResolvedSecret, []string, error) {
	selected, missing, err := domain.SelectBindings(keys, candidates)
	if err != nil {
		return nil, nil, err
	}
	resolved := make([]domain.ResolvedSecret, 0, len(selected))
	for _, item := range selected {
		value, err := s.decryptValue(item.Secret.StoredValue)
		if err != nil {
			return nil, nil, err
		}
		resolved = append(resolved, domain.ResolvedSecret{
			BindingKey:   item.BindingKey,
			BindingScope: item.Binding.Scope,
			SecretID:     item.Secret.ID,
			SecretName:   item.Secret.Name,
			SecretScope:  item.Secret.Scope,
			SecretKind:   item.Secret.Kind,
			Value:        value,
		})
	}
	return resolved, missing, nil
}
