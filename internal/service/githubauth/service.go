package githubauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	"github.com/BetterAndBetterII/openase/internal/logging"
	repo "github.com/BetterAndBetterII/openase/internal/repo/githubauth"
	"github.com/google/uuid"
)

//nolint:gosec // Algorithm identifier for persisted metadata; not a credential or key.
const credentialAlgorithm = "aes-256-gcm"

var (
	ErrUnavailable             = errors.New("github auth service unavailable")
	ErrInvalidInput            = errors.New("invalid GitHub auth input")
	ErrCredentialNotConfigured = errors.New("GitHub credential is not configured")
	ErrGHCLIImportFailed       = errors.New("failed to import gh auth token")
)

var githubAuthServiceComponent = logging.DeclareComponent("github-auth-service")

type TokenResolver interface {
	ResolveProjectCredential(ctx context.Context, projectID uuid.UUID) (domain.ResolvedCredential, error)
}

type SecurityManager interface {
	ReadProjectSecurity(ctx context.Context, projectID uuid.UUID) (ProjectSecurity, error)
	SaveManualCredential(ctx context.Context, input SaveCredentialInput) (ProjectSecurity, error)
	ImportGHCLICredential(ctx context.Context, input ScopeInput) (ProjectSecurity, error)
	RetestCredential(ctx context.Context, input ScopeInput) (ProjectSecurity, error)
	DeleteCredential(ctx context.Context, input ScopeInput) (ProjectSecurity, error)
}

type OrgSecurityManager interface {
	ReadOrgSecurity(ctx context.Context, orgID uuid.UUID) (OrgSecurity, error)
	SaveOrgManualCredential(ctx context.Context, input OrgSaveCredentialInput) (OrgSecurity, error)
	ImportOrgGHCLICredential(ctx context.Context, input OrgInput) (OrgSecurity, error)
	RetestOrgCredential(ctx context.Context, input OrgInput) (OrgSecurity, error)
	DeleteOrgCredential(ctx context.Context, input OrgInput) (OrgSecurity, error)
}

type ScopedSecurity struct {
	Scope        domain.Scope
	Configured   bool
	Source       domain.Source
	TokenPreview string
	Probe        domain.TokenProbe
}

type ProjectSecurity struct {
	Effective       ScopedSecurity
	Organization    ScopedSecurity
	ProjectOverride ScopedSecurity
}

type OrgSecurity struct {
	Organization ScopedSecurity
}

type SaveCredentialInput struct {
	ProjectID uuid.UUID
	Scope     domain.Scope
	Token     string
}

type ScopeInput struct {
	ProjectID uuid.UUID
	Scope     domain.Scope
}

type OrgSaveCredentialInput struct {
	OrganizationID uuid.UUID
	Token          string
}

type OrgInput struct {
	OrganizationID uuid.UUID
}

type tokenImporter interface {
	ReadToken(ctx context.Context) (string, error)
}

type Service struct {
	repo          repo.Repository
	httpClient    *http.Client
	block         cipher.Block
	baseURL       string
	now           func() time.Time
	tokenImporter tokenImporter
	logger        *slog.Logger
}

func New(repository repo.Repository, httpClient *http.Client, cipherSeed string) (*Service, error) {
	if repository == nil {
		return nil, errors.New("github auth repository is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	keyMaterial, err := base64.StdEncoding.DecodeString(domain.DefaultCipherSeed(cipherSeed))
	if err != nil {
		return nil, fmt.Errorf("derive github auth cipher key: %w", err)
	}
	block, err := aes.NewCipher(keyMaterial)
	if err != nil {
		return nil, fmt.Errorf("create github auth cipher: %w", err)
	}

	return &Service{
		repo:          repository,
		httpClient:    httpClient,
		block:         block,
		baseURL:       "https://api.github.com",
		now:           time.Now,
		tokenImporter: ghCLITokenImporter{},
		logger:        logging.WithComponent(nil, githubAuthServiceComponent),
	}, nil
}

func (s *Service) ReadProjectSecurity(ctx context.Context, projectID uuid.UUID) (ProjectSecurity, error) {
	if s.repo == nil {
		return ProjectSecurity{}, ErrUnavailable
	}
	projectContext, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return ProjectSecurity{}, err
	}
	resolved, err := domain.ResolveProjectCredential(projectContext, s.decryptStoredCredential)
	if err != nil {
		return ProjectSecurity{}, err
	}

	return ProjectSecurity{
		Effective: ScopedSecurity{
			Scope:        resolved.Scope,
			Configured:   strings.TrimSpace(resolved.Token) != "",
			Source:       resolved.Source,
			TokenPreview: resolved.TokenPreview,
			Probe:        resolved.Probe,
		},
		Organization: readScopedSecurity(
			domain.ScopeOrganization,
			projectContext.OrganizationCredential,
			projectContext.OrganizationProbe,
		),
		ProjectOverride: readScopedSecurity(
			domain.ScopeProject,
			projectContext.ProjectCredential,
			projectContext.ProjectProbe,
		),
	}, nil
}

func (s *Service) ResolveProjectCredential(ctx context.Context, projectID uuid.UUID) (domain.ResolvedCredential, error) {
	if s.repo == nil {
		return domain.ResolvedCredential{}, ErrUnavailable
	}
	projectContext, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return domain.ResolvedCredential{}, err
	}
	return domain.ResolveProjectCredential(projectContext, s.decryptStoredCredential)
}

func (s *Service) ProbeResolvedCredential(ctx context.Context, projectID uuid.UUID, repositoryURL string) (domain.TokenProbe, error) {
	projectContext, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return domain.TokenProbe{}, err
	}
	resolved, err := domain.ResolveProjectCredential(projectContext, s.decryptStoredCredential)
	if err != nil {
		return domain.TokenProbe{}, err
	}
	if strings.TrimSpace(resolved.Token) == "" {
		return resolved.Probe, nil
	}
	if strings.TrimSpace(repositoryURL) == "" {
		repositoryURL = projectContext.ProjectRepositoryURL
	}

	probe, err := s.probeToken(ctx, resolved.Token, repositoryURL)
	if err != nil {
		return domain.TokenProbe{}, err
	}
	switch resolved.Scope {
	case domain.ScopeProject:
		if err := s.repo.SaveProjectProbe(ctx, projectID, probe); err != nil {
			return domain.TokenProbe{}, fmt.Errorf("save project GitHub token probe: %w", err)
		}
	case domain.ScopeOrganization:
		if err := s.repo.SaveOrganizationProbe(ctx, projectContext.OrganizationID, probe); err != nil {
			return domain.TokenProbe{}, fmt.Errorf("save organization GitHub token probe: %w", err)
		}
	}
	return probe, nil
}

func (s *Service) SaveManualCredential(ctx context.Context, input SaveCredentialInput) (ProjectSecurity, error) {
	if s.repo == nil {
		return ProjectSecurity{}, ErrUnavailable
	}
	token := strings.TrimSpace(input.Token)
	if token == "" {
		return ProjectSecurity{}, fmt.Errorf("%w: token must not be empty", ErrInvalidInput)
	}
	if !input.Scope.IsValid() {
		return ProjectSecurity{}, fmt.Errorf("%w: invalid GitHub credential scope %q", ErrInvalidInput, input.Scope)
	}

	sealed, err := s.SealToken(token, domain.SourceManualPaste)
	if err != nil {
		return ProjectSecurity{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	return s.saveCredential(ctx, input.ProjectID, input.Scope, sealed, token)
}

func (s *Service) ImportGHCLICredential(ctx context.Context, input ScopeInput) (ProjectSecurity, error) {
	if s.repo == nil {
		return ProjectSecurity{}, ErrUnavailable
	}
	if !input.Scope.IsValid() {
		return ProjectSecurity{}, fmt.Errorf("%w: invalid GitHub credential scope %q", ErrInvalidInput, input.Scope)
	}
	if s.tokenImporter == nil {
		return ProjectSecurity{}, ErrUnavailable
	}

	token, err := s.tokenImporter.ReadToken(ctx)
	if err != nil {
		s.logger.Error("import github credential from gh cli failed", "project_id", input.ProjectID.String(), "scope", input.Scope, "error", err)
		return ProjectSecurity{}, fmt.Errorf("%w: %s", ErrGHCLIImportFailed, err)
	}
	sealed, err := s.SealToken(token, domain.SourceGHCLIImport)
	if err != nil {
		return ProjectSecurity{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	return s.saveCredential(ctx, input.ProjectID, input.Scope, sealed, token)
}

func (s *Service) RetestCredential(ctx context.Context, input ScopeInput) (ProjectSecurity, error) {
	if s.repo == nil {
		return ProjectSecurity{}, ErrUnavailable
	}
	if !input.Scope.IsValid() {
		return ProjectSecurity{}, fmt.Errorf("%w: invalid GitHub credential scope %q", ErrInvalidInput, input.Scope)
	}

	projectContext, err := s.repo.GetProjectContext(ctx, input.ProjectID)
	if err != nil {
		return ProjectSecurity{}, err
	}
	stored, _, err := projectContext.CredentialForScope(input.Scope)
	if err != nil {
		return ProjectSecurity{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	if stored == nil {
		s.logger.Warn("retest github credential requested for missing scope", "project_id", input.ProjectID.String(), "scope", input.Scope)
		return ProjectSecurity{}, ErrCredentialNotConfigured
	}
	token, err := s.decryptStoredCredential(*stored)
	if err != nil {
		return ProjectSecurity{}, err
	}
	if err := s.saveScopedProbe(ctx, projectContext, input.Scope, probingProbe()); err != nil {
		return ProjectSecurity{}, err
	}
	if err := s.probeAndPersistScopedCredential(ctx, projectContext, input.Scope, token); err != nil {
		return ProjectSecurity{}, err
	}
	return s.ReadProjectSecurity(ctx, input.ProjectID)
}

func (s *Service) DeleteCredential(ctx context.Context, input ScopeInput) (ProjectSecurity, error) {
	if s.repo == nil {
		return ProjectSecurity{}, ErrUnavailable
	}
	if !input.Scope.IsValid() {
		return ProjectSecurity{}, fmt.Errorf("%w: invalid GitHub credential scope %q", ErrInvalidInput, input.Scope)
	}
	projectContext, err := s.repo.GetProjectContext(ctx, input.ProjectID)
	if err != nil {
		return ProjectSecurity{}, err
	}

	switch input.Scope {
	case domain.ScopeOrganization:
		if err := s.repo.ClearOrganizationCredential(ctx, projectContext.OrganizationID); err != nil {
			return ProjectSecurity{}, fmt.Errorf("clear organization GitHub credential: %w", err)
		}
	case domain.ScopeProject:
		if err := s.repo.ClearProjectCredential(ctx, input.ProjectID); err != nil {
			return ProjectSecurity{}, fmt.Errorf("clear project GitHub credential: %w", err)
		}
	}
	s.logger.Info("deleted github credential", "project_id", input.ProjectID.String(), "scope", input.Scope)
	return s.ReadProjectSecurity(ctx, input.ProjectID)
}

func (s *Service) saveCredential(
	ctx context.Context,
	projectID uuid.UUID,
	scope domain.Scope,
	sealed domain.StoredCredential,
	token string,
) (ProjectSecurity, error) {
	projectContext, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return ProjectSecurity{}, err
	}
	if err := s.persistScopedCredential(ctx, projectContext, scope, sealed, domain.ConfiguredProbe()); err != nil {
		return ProjectSecurity{}, err
	}
	s.logger.Info("saved github credential", "project_id", projectID.String(), "scope", scope, "source", sealed.Source)
	if err := s.saveScopedProbe(ctx, projectContext, scope, probingProbe()); err != nil {
		return ProjectSecurity{}, err
	}
	if err := s.probeAndPersistScopedCredential(ctx, projectContext, scope, token); err != nil {
		return ProjectSecurity{}, err
	}
	return s.ReadProjectSecurity(ctx, projectID)
}

func (s *Service) persistScopedCredential(
	ctx context.Context,
	projectContext domain.ProjectContext,
	scope domain.Scope,
	credential domain.StoredCredential,
	probe domain.TokenProbe,
) error {
	switch scope {
	case domain.ScopeOrganization:
		if err := s.repo.SaveOrganizationCredential(ctx, projectContext.OrganizationID, credential, probe); err != nil {
			return fmt.Errorf("save organization GitHub credential: %w", err)
		}
	case domain.ScopeProject:
		if err := s.repo.SaveProjectCredential(ctx, projectContext.ProjectID, credential, probe); err != nil {
			return fmt.Errorf("save project GitHub credential: %w", err)
		}
	default:
		return fmt.Errorf("%w: invalid GitHub credential scope %q", ErrInvalidInput, scope)
	}
	return nil
}

func (s *Service) saveScopedProbe(
	ctx context.Context,
	projectContext domain.ProjectContext,
	scope domain.Scope,
	probe domain.TokenProbe,
) error {
	switch scope {
	case domain.ScopeOrganization:
		if err := s.repo.SaveOrganizationProbe(ctx, projectContext.OrganizationID, probe); err != nil {
			return fmt.Errorf("save organization GitHub token probe: %w", err)
		}
	case domain.ScopeProject:
		if err := s.repo.SaveProjectProbe(ctx, projectContext.ProjectID, probe); err != nil {
			return fmt.Errorf("save project GitHub token probe: %w", err)
		}
	default:
		return fmt.Errorf("%w: invalid GitHub credential scope %q", ErrInvalidInput, scope)
	}
	return nil
}

func (s *Service) probeAndPersistScopedCredential(
	ctx context.Context,
	projectContext domain.ProjectContext,
	scope domain.Scope,
	token string,
) error {
	probe, err := s.probeToken(ctx, token, projectContext.ProjectRepositoryURL)
	if err != nil {
		return err
	}
	if err := s.saveScopedProbe(ctx, projectContext, scope, probe); err != nil {
		return err
	}
	return nil
}

func (s *Service) SealToken(token string, source domain.Source) (domain.StoredCredential, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return domain.StoredCredential{}, errors.New("token must not be empty")
	}
	if !source.IsValid() {
		return domain.StoredCredential{}, fmt.Errorf("invalid GitHub token source %q", source)
	}

	gcm, err := cipher.NewGCM(s.block)
	if err != nil {
		return domain.StoredCredential{}, fmt.Errorf("create github auth AEAD: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return domain.StoredCredential{}, fmt.Errorf("generate github auth nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(trimmed), nil)
	return domain.StoredCredential{
		Algorithm:    credentialAlgorithm,
		TokenPreview: domain.RedactToken(trimmed),
		Nonce:        base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:   base64.StdEncoding.EncodeToString(ciphertext),
		Source:       source,
		UpdatedAt:    s.now().UTC(),
	}, nil
}

func (s *Service) decryptStoredCredential(stored domain.StoredCredential) (string, error) {
	if strings.TrimSpace(stored.Algorithm) != credentialAlgorithm {
		return "", fmt.Errorf("unsupported GitHub credential algorithm %q", stored.Algorithm)
	}

	gcm, err := cipher.NewGCM(s.block)
	if err != nil {
		return "", fmt.Errorf("create github auth AEAD: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(strings.TrimSpace(stored.Nonce))
	if err != nil {
		return "", fmt.Errorf("decode GitHub credential nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(stored.Ciphertext))
	if err != nil {
		return "", fmt.Errorf("decode GitHub credential ciphertext: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt GitHub credential: %w", err)
	}
	return strings.TrimSpace(string(plaintext)), nil
}

func (s *Service) probeToken(ctx context.Context, token string, repositoryURL string) (domain.TokenProbe, error) {
	userRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.baseURL, "/")+"/user", http.NoBody)
	if err != nil {
		return domain.TokenProbe{}, fmt.Errorf("build GitHub user probe request: %w", err)
	}
	userRequest.Header.Set("Accept", "application/vnd.github+json")
	userRequest.Header.Set("Authorization", "Bearer "+token)

	userResponse, err := s.httpClient.Do(userRequest)
	if err != nil {
		s.logger.Error("github credential user probe failed", "repository_url", repositoryURL, "operation", "user_probe", "error", err)
		return domain.TokenProbe{
			State:      domain.ProbeStateError,
			Configured: true,
			Valid:      false,
			RepoAccess: domain.RepoAccessNotChecked,
			LastError:  err.Error(),
		}, nil
	}
	defer func() { _ = userResponse.Body.Close() }()

	permissions := parseScopeHeader(userResponse.Header.Get("X-OAuth-Scopes"))
	checkedAt := s.now().UTC()
	probe := domain.TokenProbe{
		Configured:  true,
		Valid:       false,
		Permissions: permissions,
		RepoAccess:  domain.RepoAccessNotChecked,
		CheckedAt:   &checkedAt,
	}

	switch userResponse.StatusCode {
	case http.StatusOK:
		var userPayload struct {
			Login string `json:"login"`
		}
		if err := json.NewDecoder(userResponse.Body).Decode(&userPayload); err != nil {
			s.logger.Warn("github credential user probe decode failed", "repository_url", repositoryURL, "error", err)
			probe.State = domain.ProbeStateError
			probe.LastError = "GitHub user probe returned invalid JSON"
			return probe, nil
		}
		probe.Login = strings.TrimSpace(userPayload.Login)
		probe.State = domain.ProbeStateValid
		probe.Valid = true
	case http.StatusUnauthorized:
		probe.State = domain.ProbeStateRevoked
		probe.LastError = http.StatusText(userResponse.StatusCode)
		return probe, nil
	default:
		probe.State = domain.ProbeStateError
		probe.LastError = fmt.Sprintf("GitHub user probe returned %d", userResponse.StatusCode)
		s.logger.Warn("github credential user probe returned unexpected status", "repository_url", repositoryURL, "status_code", userResponse.StatusCode, "github_request_id", strings.TrimSpace(userResponse.Header.Get("X-GitHub-Request-Id")))
		return probe, nil
	}

	repoRef, ok := domain.ParseGitHubRepositoryURL(repositoryURL)
	if !ok {
		return probe, nil
	}

	repoRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.baseURL, "/")+"/repos/"+repoRef.Owner+"/"+repoRef.Name, http.NoBody)
	if err != nil {
		return domain.TokenProbe{}, fmt.Errorf("build GitHub repo probe request: %w", err)
	}
	repoRequest.Header.Set("Accept", "application/vnd.github+json")
	repoRequest.Header.Set("Authorization", "Bearer "+token)

	repoResponse, err := s.httpClient.Do(repoRequest)
	if err != nil {
		s.logger.Error("github credential repo probe failed", "repository_url", repositoryURL, "operation", "repo_probe", "error", err)
		probe.State = domain.ProbeStateError
		probe.Valid = false
		probe.LastError = err.Error()
		return probe, nil
	}
	defer func() { _ = repoResponse.Body.Close() }()

	switch repoResponse.StatusCode {
	case http.StatusOK:
		probe.RepoAccess = domain.RepoAccessGranted
	case http.StatusUnauthorized:
		probe.State = domain.ProbeStateRevoked
		probe.Valid = false
		probe.RepoAccess = domain.RepoAccessDenied
		probe.LastError = http.StatusText(repoResponse.StatusCode)
	case http.StatusForbidden, http.StatusNotFound:
		probe.State = domain.ProbeStateInsufficientPermissions
		probe.Valid = false
		probe.RepoAccess = domain.RepoAccessDenied
		probe.LastError = fmt.Sprintf("GitHub repo probe returned %d", repoResponse.StatusCode)
	default:
		probe.State = domain.ProbeStateError
		probe.Valid = false
		probe.LastError = fmt.Sprintf("GitHub repo probe returned %d", repoResponse.StatusCode)
	}
	if probe.LastError != "" {
		s.logger.Warn("github credential probe completed with non-success result", "repository_url", repositoryURL, "state", probe.State, "repo_access", probe.RepoAccess, "status_code", repoResponse.StatusCode, "github_request_id", strings.TrimSpace(repoResponse.Header.Get("X-GitHub-Request-Id")), "last_error", probe.LastError)
	}

	return probe, nil
}

func parseScopeHeader(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	permissions := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		permissions = append(permissions, trimmed)
	}
	slices.Sort(permissions)
	return slices.Compact(permissions)
}

// OrgSecurityManager implementation

func (s *Service) ReadOrgSecurity(ctx context.Context, orgID uuid.UUID) (OrgSecurity, error) {
	if s.repo == nil {
		return OrgSecurity{}, ErrUnavailable
	}
	orgContext, err := s.repo.GetOrganizationContext(ctx, orgID)
	if err != nil {
		return OrgSecurity{}, err
	}
	return OrgSecurity{
		Organization: readScopedSecurity(domain.ScopeOrganization, orgContext.Credential, orgContext.Probe),
	}, nil
}

func (s *Service) SaveOrgManualCredential(ctx context.Context, input OrgSaveCredentialInput) (OrgSecurity, error) {
	if s.repo == nil {
		return OrgSecurity{}, ErrUnavailable
	}
	token := strings.TrimSpace(input.Token)
	if token == "" {
		return OrgSecurity{}, fmt.Errorf("%w: token must not be empty", ErrInvalidInput)
	}
	sealed, err := s.SealToken(token, domain.SourceManualPaste)
	if err != nil {
		return OrgSecurity{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	if err := s.repo.SaveOrganizationCredential(ctx, input.OrganizationID, sealed, domain.ConfiguredProbe()); err != nil {
		return OrgSecurity{}, fmt.Errorf("save organization GitHub credential: %w", err)
	}
	s.logger.Info("saved org github credential", "org_id", input.OrganizationID.String())
	if err := s.repo.SaveOrganizationProbe(ctx, input.OrganizationID, probingProbe()); err != nil {
		return OrgSecurity{}, err
	}
	probe, err := s.probeToken(ctx, token, "")
	if err != nil {
		return OrgSecurity{}, err
	}
	if err := s.repo.SaveOrganizationProbe(ctx, input.OrganizationID, probe); err != nil {
		return OrgSecurity{}, fmt.Errorf("save organization GitHub token probe: %w", err)
	}
	return s.ReadOrgSecurity(ctx, input.OrganizationID)
}

func (s *Service) ImportOrgGHCLICredential(ctx context.Context, input OrgInput) (OrgSecurity, error) {
	if s.repo == nil {
		return OrgSecurity{}, ErrUnavailable
	}
	if s.tokenImporter == nil {
		return OrgSecurity{}, ErrUnavailable
	}
	token, err := s.tokenImporter.ReadToken(ctx)
	if err != nil {
		s.logger.Error("import org github credential from gh cli failed", "org_id", input.OrganizationID.String(), "error", err)
		return OrgSecurity{}, fmt.Errorf("%w: %s", ErrGHCLIImportFailed, err)
	}
	sealed, err := s.SealToken(token, domain.SourceGHCLIImport)
	if err != nil {
		return OrgSecurity{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	if err := s.repo.SaveOrganizationCredential(ctx, input.OrganizationID, sealed, domain.ConfiguredProbe()); err != nil {
		return OrgSecurity{}, fmt.Errorf("save organization GitHub credential: %w", err)
	}
	s.logger.Info("imported org github credential from gh cli", "org_id", input.OrganizationID.String())
	if err := s.repo.SaveOrganizationProbe(ctx, input.OrganizationID, probingProbe()); err != nil {
		return OrgSecurity{}, err
	}
	probe, err := s.probeToken(ctx, token, "")
	if err != nil {
		return OrgSecurity{}, err
	}
	if err := s.repo.SaveOrganizationProbe(ctx, input.OrganizationID, probe); err != nil {
		return OrgSecurity{}, fmt.Errorf("save organization GitHub token probe: %w", err)
	}
	return s.ReadOrgSecurity(ctx, input.OrganizationID)
}

func (s *Service) RetestOrgCredential(ctx context.Context, input OrgInput) (OrgSecurity, error) {
	if s.repo == nil {
		return OrgSecurity{}, ErrUnavailable
	}
	orgContext, err := s.repo.GetOrganizationContext(ctx, input.OrganizationID)
	if err != nil {
		return OrgSecurity{}, err
	}
	if orgContext.Credential == nil {
		return OrgSecurity{}, ErrCredentialNotConfigured
	}
	token, err := s.decryptStoredCredential(*orgContext.Credential)
	if err != nil {
		return OrgSecurity{}, err
	}
	if err := s.repo.SaveOrganizationProbe(ctx, input.OrganizationID, probingProbe()); err != nil {
		return OrgSecurity{}, err
	}
	probe, err := s.probeToken(ctx, token, "")
	if err != nil {
		return OrgSecurity{}, err
	}
	if err := s.repo.SaveOrganizationProbe(ctx, input.OrganizationID, probe); err != nil {
		return OrgSecurity{}, fmt.Errorf("save organization GitHub token probe: %w", err)
	}
	return s.ReadOrgSecurity(ctx, input.OrganizationID)
}

func (s *Service) DeleteOrgCredential(ctx context.Context, input OrgInput) (OrgSecurity, error) {
	if s.repo == nil {
		return OrgSecurity{}, ErrUnavailable
	}
	if err := s.repo.ClearOrganizationCredential(ctx, input.OrganizationID); err != nil {
		return OrgSecurity{}, fmt.Errorf("clear organization GitHub credential: %w", err)
	}
	s.logger.Info("deleted org github credential", "org_id", input.OrganizationID.String())
	return s.ReadOrgSecurity(ctx, input.OrganizationID)
}

func readScopedSecurity(
	scope domain.Scope,
	credential *domain.StoredCredential,
	probe *domain.TokenProbe,
) ScopedSecurity {
	state := ScopedSecurity{
		Scope:      scope,
		Configured: credential != nil,
		Probe:      domain.NormalizeProbe(probe, credential != nil),
	}
	if credential == nil {
		return state
	}
	state.Source = credential.Source
	state.TokenPreview = strings.TrimSpace(credential.TokenPreview)
	return state
}

func probingProbe() domain.TokenProbe {
	return domain.TokenProbe{
		State:      domain.ProbeStateProbing,
		Configured: true,
		Valid:      false,
		RepoAccess: domain.RepoAccessNotChecked,
	}
}

type ghCLITokenImporter struct{}

func (ghCLITokenImporter) ReadToken(ctx context.Context) (string, error) {
	command := exec.CommandContext(ctx, "gh", "auth", "token")
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("gh auth token: %s", message)
	}
	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", errors.New("gh auth token returned empty output")
	}
	return token, nil
}
