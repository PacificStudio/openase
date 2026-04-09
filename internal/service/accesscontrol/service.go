package accesscontrol

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	repo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	"go.yaml.in/yaml/v3"
)

const storageLocationDB = "db:instance_auth_configs"

//nolint:gosec // Persisted metadata label, not a secret.
const secretAlgorithm = "aes-256-gcm"

type Service struct {
	repo           repo.Repository
	block          cipher.Block
	configFilePath string
	homeDir        string
	bootstrap      config.AuthConfig
	now            func() time.Time
	snapshotMu     sync.RWMutex
	snapshot       ReadResult
	snapshotLoaded bool
}

type ReadResult struct {
	State           iam.AccessControlState
	StorageLocation string
}

type legacyConfigDoc struct {
	Auth struct {
		Mode           string                  `yaml:"mode"`
		OIDC           legacyOIDCSection       `yaml:"oidc"`
		LastValidation legacyValidationSection `yaml:"last_validation"`
	} `yaml:"auth"`
}

type legacyOIDCSection struct {
	IssuerURL            string   `yaml:"issuer_url"`
	ClientID             string   `yaml:"client_id"`
	ClientSecret         string   `yaml:"client_secret"`
	RedirectMode         string   `yaml:"redirect_mode"`
	FixedRedirectURL     string   `yaml:"fixed_redirect_url"`
	RedirectURL          string   `yaml:"redirect_url"`
	Scopes               []string `yaml:"scopes"`
	EmailClaim           string   `yaml:"email_claim"`
	NameClaim            string   `yaml:"name_claim"`
	UsernameClaim        string   `yaml:"username_claim"`
	GroupsClaim          string   `yaml:"groups_claim"`
	AllowedEmailDomains  []string `yaml:"allowed_email_domains"`
	BootstrapAdminEmails []string `yaml:"bootstrap_admin_emails"`
	SessionTTL           string   `yaml:"session_ttl"`
	SessionIdleTTL       string   `yaml:"session_idle_ttl"`
}

type legacyValidationSection struct {
	Status                string   `yaml:"status"`
	Message               string   `yaml:"message"`
	CheckedAt             string   `yaml:"checked_at"`
	IssuerURL             string   `yaml:"issuer_url"`
	AuthorizationEndpoint string   `yaml:"authorization_endpoint"`
	TokenEndpoint         string   `yaml:"token_endpoint"`
	RedirectURL           string   `yaml:"redirect_url"`
	Warnings              []string `yaml:"warnings"`
}

func New(repository repo.Repository, cipherSeed string, configFilePath string, homeDir string, bootstrap config.AuthConfig) (*Service, error) {
	hash := sha256.Sum256([]byte(strings.TrimSpace(cipherSeed)))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, fmt.Errorf("create access control cipher: %w", err)
	}
	return &Service{
		repo:           repository,
		block:          block,
		configFilePath: strings.TrimSpace(configFilePath),
		homeDir:        strings.TrimSpace(homeDir),
		bootstrap:      bootstrap,
		now:            time.Now,
	}, nil
}

func (s *Service) Read(ctx context.Context) (ReadResult, error) {
	s.snapshotMu.RLock()
	if s.snapshotLoaded {
		snapshot := s.snapshot
		s.snapshotMu.RUnlock()
		return snapshot, nil
	}
	s.snapshotMu.RUnlock()

	snapshot, err := s.loadSnapshot(ctx)
	if err != nil {
		return ReadResult{}, err
	}
	s.storeSnapshot(snapshot)
	return snapshot, nil
}

func (s *Service) RuntimeState(ctx context.Context) (iam.RuntimeAccessControlState, error) {
	current, err := s.Read(ctx)
	if err != nil {
		return iam.RuntimeAccessControlState{}, err
	}
	return iam.ResolveRuntimeAccessControlState(current.State), nil
}

func (s *Service) Refresh(ctx context.Context) (ReadResult, error) {
	snapshot, err := s.loadSnapshot(ctx)
	if err != nil {
		return ReadResult{}, err
	}
	s.storeSnapshot(snapshot)
	return snapshot, nil
}

func (s *Service) loadSnapshot(ctx context.Context) (ReadResult, error) {
	if s.repo != nil {
		record, err := s.repo.Get(ctx)
		if err != nil {
			return ReadResult{}, err
		}
		if record != nil {
			state, err := s.parseStoredRecord(*record)
			if err != nil {
				return ReadResult{}, err
			}
			return ReadResult{State: state, StorageLocation: storageLocationDB}, nil
		}
	}

	input, location, err := s.readLegacyFallback()
	if err != nil {
		return ReadResult{}, err
	}
	state, err := iam.ParseAccessControlState(input)
	if err != nil {
		return ReadResult{}, err
	}
	return ReadResult{State: state, StorageLocation: location}, nil
}

func (s *Service) storeSnapshot(snapshot ReadResult) {
	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()
	s.snapshot = snapshot
	s.snapshotLoaded = true
}

func (s *Service) SaveDraft(ctx context.Context, draft iam.DraftOIDCConfig) (ReadResult, error) {
	current, err := s.Read(ctx)
	if err != nil {
		return ReadResult{}, err
	}
	record, err := s.recordFromDraft(iam.AccessControlStatusDraft, draft, current.State.Validation, current.State.Activation)
	if err != nil {
		return ReadResult{}, err
	}
	if err := s.persist(ctx, record); err != nil {
		return ReadResult{}, err
	}
	state, err := s.parseStoredRecord(record)
	if err != nil {
		return ReadResult{}, err
	}
	result := ReadResult{State: state, StorageLocation: storageLocationDB}
	s.storeSnapshot(result)
	return result, nil
}

func (s *Service) SaveValidation(ctx context.Context, validation iam.OIDCValidationMetadata) error {
	current, err := s.Read(ctx)
	if err != nil {
		return err
	}
	record, err := s.recordFromState(current.State)
	if err != nil {
		return err
	}
	record.Validation = validationInputFromDomain(validation)
	if err := s.persist(ctx, record); err != nil {
		return err
	}
	state, err := s.parseStoredRecord(record)
	if err != nil {
		return err
	}
	s.storeSnapshot(ReadResult{State: state, StorageLocation: storageLocationDB})
	return nil
}

func (s *Service) Activate(ctx context.Context, active iam.ActiveOIDCConfig, activation iam.OIDCActivationMetadata) (ReadResult, error) {
	current, err := s.Read(ctx)
	if err != nil {
		return ReadResult{}, err
	}
	record, err := s.recordFromActive(active, current.State.Validation, activation)
	if err != nil {
		return ReadResult{}, err
	}
	if err := s.persist(ctx, record); err != nil {
		return ReadResult{}, err
	}
	state, err := s.parseStoredRecord(record)
	if err != nil {
		return ReadResult{}, err
	}
	result := ReadResult{State: state, StorageLocation: storageLocationDB}
	s.storeSnapshot(result)
	return result, nil
}

func (s *Service) Disable(ctx context.Context) (ReadResult, error) {
	current, err := s.Read(ctx)
	if err != nil {
		return ReadResult{}, err
	}

	var record repo.StoredConfigRecord
	switch {
	case current.State.Draft != nil:
		record, err = s.recordFromDraft(iam.AccessControlStatusDraft, *current.State.Draft, current.State.Validation, current.State.Activation)
	case current.State.Active != nil:
		draft := iam.DraftOIDCConfig(*current.State.Active)
		record, err = s.recordFromDraft(iam.AccessControlStatusDraft, draft, current.State.Validation, current.State.Activation)
	default:
		record = repo.StoredConfigRecord{
			Status:     iam.AccessControlStatusAbsent.String(),
			Validation: validationInputFromDomain(current.State.Validation),
			Activation: activationInputFromDomain(current.State.Activation),
		}
	}
	if err != nil {
		return ReadResult{}, err
	}
	if err := s.persist(ctx, record); err != nil {
		return ReadResult{}, err
	}
	state, err := s.parseStoredRecord(record)
	if err != nil {
		return ReadResult{}, err
	}
	result := ReadResult{State: state, StorageLocation: storageLocationDB}
	s.storeSnapshot(result)
	return result, nil
}

func (s *Service) recordFromState(state iam.AccessControlState) (repo.StoredConfigRecord, error) {
	switch {
	case state.Active != nil:
		return s.recordFromActive(*state.Active, state.Validation, state.Activation)
	case state.Draft != nil:
		return s.recordFromDraft(state.Status, *state.Draft, state.Validation, state.Activation)
	default:
		return repo.StoredConfigRecord{
			Status:     string(state.Status),
			Validation: validationInputFromDomain(state.Validation),
			Activation: activationInputFromDomain(state.Activation),
		}, nil
	}
}

func (s *Service) recordFromDraft(status iam.AccessControlStatus, draft iam.DraftOIDCConfig, validation iam.OIDCValidationMetadata, activation iam.OIDCActivationMetadata) (repo.StoredConfigRecord, error) {
	sealed, err := s.sealSecret(draft.ClientSecret)
	if err != nil {
		return repo.StoredConfigRecord{}, err
	}
	return repo.StoredConfigRecord{
		Status:                string(status),
		IssuerURL:             draft.IssuerURL,
		ClientID:              draft.ClientID,
		ClientSecretEncrypted: sealed,
		RedirectMode:          draft.RedirectMode.String(),
		FixedRedirectURL:      draft.FixedRedirectURL,
		Scopes:                append([]string(nil), draft.Scopes...),
		EmailClaim:            draft.Claims.EmailClaim,
		NameClaim:             draft.Claims.NameClaim,
		UsernameClaim:         draft.Claims.UsernameClaim,
		GroupsClaim:           draft.Claims.GroupsClaim,
		AllowedEmailDomains:   append([]string(nil), draft.AllowedEmailDomains...),
		BootstrapAdminEmails:  append([]string(nil), draft.BootstrapAdminEmails...),
		SessionTTL:            draft.SessionPolicy.SessionTTL.String(),
		SessionIdleTTL:        draft.SessionPolicy.SessionIdleTTL.String(),
		Validation:            validationInputFromDomain(validation),
		Activation:            activationInputFromDomain(activation),
	}, nil
}

func (s *Service) recordFromActive(active iam.ActiveOIDCConfig, validation iam.OIDCValidationMetadata, activation iam.OIDCActivationMetadata) (repo.StoredConfigRecord, error) {
	sealed, err := s.sealSecret(active.ClientSecret)
	if err != nil {
		return repo.StoredConfigRecord{}, err
	}
	return repo.StoredConfigRecord{
		Status:                iam.AccessControlStatusActive.String(),
		IssuerURL:             active.IssuerURL,
		ClientID:              active.ClientID,
		ClientSecretEncrypted: sealed,
		RedirectMode:          active.RedirectMode.String(),
		FixedRedirectURL:      active.FixedRedirectURL,
		Scopes:                append([]string(nil), active.Scopes...),
		EmailClaim:            active.Claims.EmailClaim,
		NameClaim:             active.Claims.NameClaim,
		UsernameClaim:         active.Claims.UsernameClaim,
		GroupsClaim:           active.Claims.GroupsClaim,
		AllowedEmailDomains:   append([]string(nil), active.AllowedEmailDomains...),
		BootstrapAdminEmails:  append([]string(nil), active.BootstrapAdminEmails...),
		SessionTTL:            active.SessionPolicy.SessionTTL.String(),
		SessionIdleTTL:        active.SessionPolicy.SessionIdleTTL.String(),
		Validation:            validationInputFromDomain(validation),
		Activation:            activationInputFromDomain(activation),
	}, nil
}

func (s *Service) persist(ctx context.Context, record repo.StoredConfigRecord) error {
	if s.repo == nil {
		return errors.New("access control repository is unavailable")
	}
	_, err := s.repo.Upsert(ctx, record)
	return err
}

func (s *Service) parseStoredRecord(record repo.StoredConfigRecord) (iam.AccessControlState, error) {
	secret, err := s.openSecret(record.ClientSecretEncrypted)
	if err != nil {
		return iam.AccessControlState{}, err
	}
	return iam.ParseAccessControlState(iam.AccessControlStateInput{
		Status:               record.Status,
		IssuerURL:            record.IssuerURL,
		ClientID:             record.ClientID,
		ClientSecret:         secret,
		RedirectMode:         record.RedirectMode,
		FixedRedirectURL:     record.FixedRedirectURL,
		Scopes:               append([]string(nil), record.Scopes...),
		EmailClaim:           record.EmailClaim,
		NameClaim:            record.NameClaim,
		UsernameClaim:        record.UsernameClaim,
		GroupsClaim:          record.GroupsClaim,
		AllowedEmailDomains:  append([]string(nil), record.AllowedEmailDomains...),
		BootstrapAdminEmails: append([]string(nil), record.BootstrapAdminEmails...),
		SessionTTL:           record.SessionTTL,
		SessionIdleTTL:       record.SessionIdleTTL,
		Validation:           record.Validation,
		Activation:           record.Activation,
	})
}

func (s *Service) sealSecret(secret string) (*iam.EncryptedSecret, error) {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return nil, nil
	}
	gcm, err := cipher.NewGCM(s.block)
	if err != nil {
		return nil, fmt.Errorf("create access control cipher mode: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate access control nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(trimmed), nil)
	return &iam.EncryptedSecret{
		Algorithm:  secretAlgorithm,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		UpdatedAt:  s.now().UTC(),
	}, nil
}

func (s *Service) openSecret(secret *iam.EncryptedSecret) (string, error) {
	if secret == nil {
		return "", nil
	}
	if strings.TrimSpace(secret.Algorithm) != secretAlgorithm {
		return "", fmt.Errorf("unsupported encrypted secret algorithm %q", secret.Algorithm)
	}
	nonce, err := base64.StdEncoding.DecodeString(strings.TrimSpace(secret.Nonce))
	if err != nil {
		return "", fmt.Errorf("decode access control nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(secret.Ciphertext))
	if err != nil {
		return "", fmt.Errorf("decode access control ciphertext: %w", err)
	}
	gcm, err := cipher.NewGCM(s.block)
	if err != nil {
		return "", fmt.Errorf("create access control cipher mode: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt access control secret: %w", err)
	}
	return string(plaintext), nil
}

func (s *Service) readLegacyFallback() (iam.AccessControlStateInput, string, error) {
	resolvedPath := s.resolvedConfigPath()
	input := legacyInputFromBootstrap(s.bootstrap)
	if resolvedPath == "" {
		return input, "runtime:bootstrap", nil
	}
	// #nosec G304 -- The resolved config path comes from trusted bootstrap runtime configuration.
	payload, err := os.ReadFile(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return input, resolvedPath, nil
		}
		return iam.AccessControlStateInput{}, "", fmt.Errorf("read config file: %w", err)
	}

	var doc legacyConfigDoc
	if err := yaml.Unmarshal(payload, &doc); err != nil {
		return iam.AccessControlStateInput{}, "", fmt.Errorf("parse config file: %w", err)
	}
	return mergeLegacyInput(input, doc), resolvedPath, nil
}

func (s *Service) resolvedConfigPath() string {
	if s.configFilePath != "" {
		return s.configFilePath
	}
	if s.homeDir != "" {
		return filepath.Join(s.homeDir, ".openase", "config.yaml")
	}
	if guessed, err := os.UserHomeDir(); err == nil && strings.TrimSpace(guessed) != "" {
		return filepath.Join(guessed, ".openase", "config.yaml")
	}
	return ""
}

func legacyInputFromBootstrap(cfg config.AuthConfig) iam.AccessControlStateInput {
	status := iam.AccessControlStatusAbsent.String()
	if cfg.Mode == config.AuthModeOIDC {
		status = iam.AccessControlStatusActive.String()
	} else if hasOIDCDraft(cfg) {
		status = iam.AccessControlStatusDraft.String()
	}
	return iam.AccessControlStateInput{
		Status:               status,
		IssuerURL:            strings.TrimSpace(cfg.OIDC.IssuerURL),
		ClientID:             strings.TrimSpace(cfg.OIDC.ClientID),
		ClientSecret:         strings.TrimSpace(cfg.OIDC.ClientSecret),
		FixedRedirectURL:     strings.TrimSpace(cfg.OIDC.RedirectURL),
		RedirectURL:          strings.TrimSpace(cfg.OIDC.RedirectURL),
		Scopes:               append([]string(nil), cfg.OIDC.Scopes...),
		EmailClaim:           strings.TrimSpace(cfg.OIDC.EmailClaim),
		NameClaim:            strings.TrimSpace(cfg.OIDC.NameClaim),
		UsernameClaim:        strings.TrimSpace(cfg.OIDC.UsernameClaim),
		GroupsClaim:          strings.TrimSpace(cfg.OIDC.GroupsClaim),
		AllowedEmailDomains:  append([]string(nil), cfg.OIDC.AllowedEmailDomains...),
		BootstrapAdminEmails: append([]string(nil), cfg.OIDC.BootstrapAdminEmails...),
		SessionTTL:           durationStringOrEmpty(cfg.OIDC.SessionTTL),
		SessionIdleTTL:       durationStringOrEmpty(cfg.OIDC.SessionIdleTTL),
		Validation: iam.OIDCValidationMetadataInput{
			Status:   "not_tested",
			Message:  "No OIDC validation has been recorded yet.",
			Warnings: []string{},
		},
	}
}

func mergeLegacyInput(base iam.AccessControlStateInput, doc legacyConfigDoc) iam.AccessControlStateInput {
	switch strings.ToLower(strings.TrimSpace(doc.Auth.Mode)) {
	case string(config.AuthModeOIDC):
		base.Status = iam.AccessControlStatusActive.String()
	case string(config.AuthModeDisabled), "":
		if hasLegacyOIDCValues(doc.Auth.OIDC) {
			base.Status = iam.AccessControlStatusDraft.String()
		} else if base.Status == "" {
			base.Status = iam.AccessControlStatusAbsent.String()
		}
	default:
		if hasLegacyOIDCValues(doc.Auth.OIDC) {
			base.Status = iam.AccessControlStatusDraft.String()
		}
	}

	mergeString := func(current string, next string) string {
		if strings.TrimSpace(next) == "" {
			return current
		}
		return strings.TrimSpace(next)
	}
	mergeList := func(current []string, next []string) []string {
		if next == nil {
			return current
		}
		return append([]string(nil), next...)
	}

	base.IssuerURL = mergeString(base.IssuerURL, doc.Auth.OIDC.IssuerURL)
	base.ClientID = mergeString(base.ClientID, doc.Auth.OIDC.ClientID)
	base.ClientSecret = mergeString(base.ClientSecret, doc.Auth.OIDC.ClientSecret)
	base.RedirectMode = mergeString(base.RedirectMode, doc.Auth.OIDC.RedirectMode)
	base.FixedRedirectURL = mergeString(base.FixedRedirectURL, doc.Auth.OIDC.FixedRedirectURL)
	if base.FixedRedirectURL == "" {
		base.FixedRedirectURL = mergeString(base.FixedRedirectURL, doc.Auth.OIDC.RedirectURL)
	}
	base.RedirectURL = mergeString(base.RedirectURL, doc.Auth.OIDC.RedirectURL)
	base.Scopes = mergeList(base.Scopes, doc.Auth.OIDC.Scopes)
	base.EmailClaim = mergeString(base.EmailClaim, doc.Auth.OIDC.EmailClaim)
	base.NameClaim = mergeString(base.NameClaim, doc.Auth.OIDC.NameClaim)
	base.UsernameClaim = mergeString(base.UsernameClaim, doc.Auth.OIDC.UsernameClaim)
	base.GroupsClaim = mergeString(base.GroupsClaim, doc.Auth.OIDC.GroupsClaim)
	base.AllowedEmailDomains = mergeList(base.AllowedEmailDomains, doc.Auth.OIDC.AllowedEmailDomains)
	base.BootstrapAdminEmails = mergeList(base.BootstrapAdminEmails, doc.Auth.OIDC.BootstrapAdminEmails)
	base.SessionTTL = mergeString(base.SessionTTL, doc.Auth.OIDC.SessionTTL)
	base.SessionIdleTTL = mergeString(base.SessionIdleTTL, doc.Auth.OIDC.SessionIdleTTL)
	base.Validation = iam.OIDCValidationMetadataInput{
		Status:                mergeString(base.Validation.Status, doc.Auth.LastValidation.Status),
		Message:               mergeString(base.Validation.Message, doc.Auth.LastValidation.Message),
		CheckedAt:             parseOptionalRFC3339(doc.Auth.LastValidation.CheckedAt),
		IssuerURL:             mergeString(base.Validation.IssuerURL, doc.Auth.LastValidation.IssuerURL),
		AuthorizationEndpoint: mergeString(base.Validation.AuthorizationEndpoint, doc.Auth.LastValidation.AuthorizationEndpoint),
		TokenEndpoint:         mergeString(base.Validation.TokenEndpoint, doc.Auth.LastValidation.TokenEndpoint),
		RedirectURL:           mergeString(base.Validation.RedirectURL, doc.Auth.LastValidation.RedirectURL),
		Warnings:              mergeList(base.Validation.Warnings, doc.Auth.LastValidation.Warnings),
	}
	if base.Status == iam.AccessControlStatusAbsent.String() && hasLegacyOIDCValues(doc.Auth.OIDC) {
		base.Status = iam.AccessControlStatusDraft.String()
	}
	return base
}

func hasOIDCDraft(cfg config.AuthConfig) bool {
	return strings.TrimSpace(cfg.OIDC.IssuerURL) != "" ||
		strings.TrimSpace(cfg.OIDC.ClientID) != "" ||
		strings.TrimSpace(cfg.OIDC.ClientSecret) != "" ||
		strings.TrimSpace(cfg.OIDC.RedirectURL) != "" ||
		len(cfg.OIDC.AllowedEmailDomains) > 0 ||
		len(cfg.OIDC.BootstrapAdminEmails) > 0
}

func hasLegacyOIDCValues(raw legacyOIDCSection) bool {
	return strings.TrimSpace(raw.IssuerURL) != "" ||
		strings.TrimSpace(raw.ClientID) != "" ||
		strings.TrimSpace(raw.ClientSecret) != "" ||
		strings.TrimSpace(raw.RedirectMode) != "" ||
		strings.TrimSpace(raw.FixedRedirectURL) != "" ||
		strings.TrimSpace(raw.RedirectURL) != "" ||
		len(raw.Scopes) > 0 ||
		strings.TrimSpace(raw.EmailClaim) != "" ||
		strings.TrimSpace(raw.NameClaim) != "" ||
		strings.TrimSpace(raw.UsernameClaim) != "" ||
		strings.TrimSpace(raw.GroupsClaim) != "" ||
		len(raw.AllowedEmailDomains) > 0 ||
		len(raw.BootstrapAdminEmails) > 0 ||
		strings.TrimSpace(raw.SessionTTL) != "" ||
		strings.TrimSpace(raw.SessionIdleTTL) != ""
}

func durationStringOrEmpty(raw time.Duration) string {
	if raw <= 0 {
		return ""
	}
	return raw.String()
}

func parseOptionalRFC3339(raw string) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

func validationInputFromDomain(validation iam.OIDCValidationMetadata) iam.OIDCValidationMetadataInput {
	return iam.OIDCValidationMetadataInput{
		Status:                validation.Status,
		Message:               validation.Message,
		CheckedAt:             cloneTime(validation.CheckedAt),
		IssuerURL:             validation.IssuerURL,
		AuthorizationEndpoint: validation.AuthorizationEndpoint,
		TokenEndpoint:         validation.TokenEndpoint,
		RedirectURL:           validation.RedirectURL,
		Warnings:              append([]string(nil), validation.Warnings...),
	}
}

func activationInputFromDomain(activation iam.OIDCActivationMetadata) iam.OIDCActivationMetadataInput {
	return iam.OIDCActivationMetadataInput{
		ActivatedAt: cloneTime(activation.ActivatedAt),
		ActivatedBy: activation.ActivatedBy,
		Source:      activation.Source,
		Message:     activation.Message,
	}
}

func cloneTime(raw *time.Time) *time.Time {
	if raw == nil {
		return nil
	}
	cloned := raw.UTC()
	return &cloned
}
