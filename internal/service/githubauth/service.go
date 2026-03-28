package githubauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	repo "github.com/BetterAndBetterII/openase/internal/repo/githubauth"
	"github.com/google/uuid"
)

const credentialAlgorithm = "aes-256-gcm"

type TokenResolver interface {
	ResolveProjectCredential(ctx context.Context, projectID uuid.UUID) (domain.ResolvedCredential, error)
}

type SecurityReader interface {
	ReadProjectSecurity(ctx context.Context, projectID uuid.UUID) (ProjectSecurity, error)
}

type ProjectSecurity struct {
	Scope        domain.Scope
	Source       domain.Source
	TokenPreview string
	Probe        domain.TokenProbe
}

type Service struct {
	repo       repo.Repository
	httpClient *http.Client
	block      cipher.Block
	baseURL    string
	now        func() time.Time
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
		repo:       repository,
		httpClient: httpClient,
		block:      block,
		baseURL:    "https://api.github.com",
		now:        time.Now,
	}, nil
}

func (s *Service) ReadProjectSecurity(ctx context.Context, projectID uuid.UUID) (ProjectSecurity, error) {
	resolved, err := s.ResolveProjectCredential(ctx, projectID)
	if err != nil {
		return ProjectSecurity{}, err
	}

	return ProjectSecurity{
		Scope:        resolved.Scope,
		Source:       resolved.Source,
		TokenPreview: resolved.TokenPreview,
		Probe:        resolved.Probe,
	}, nil
}

func (s *Service) ResolveProjectCredential(ctx context.Context, projectID uuid.UUID) (domain.ResolvedCredential, error) {
	context, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return domain.ResolvedCredential{}, err
	}
	return domain.ResolveProjectCredential(context, s.decryptStoredCredential)
}

func (s *Service) ProbeResolvedCredential(ctx context.Context, projectID uuid.UUID, repositoryURL string) (domain.TokenProbe, error) {
	resolved, err := s.ResolveProjectCredential(ctx, projectID)
	if err != nil {
		return domain.TokenProbe{}, err
	}
	if strings.TrimSpace(resolved.Token) == "" {
		return resolved.Probe, nil
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
		context, ctxErr := s.repo.GetProjectContext(ctx, projectID)
		if ctxErr != nil {
			return domain.TokenProbe{}, ctxErr
		}
		if err := s.repo.SaveOrganizationProbe(ctx, context.OrganizationID, probe); err != nil {
			return domain.TokenProbe{}, fmt.Errorf("save organization GitHub token probe: %w", err)
		}
	}
	return probe, nil
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
		probe.State = domain.ProbeStateValid
		probe.Valid = true
	case http.StatusUnauthorized:
		probe.State = domain.ProbeStateRevoked
		probe.LastError = http.StatusText(userResponse.StatusCode)
		return probe, nil
	default:
		probe.State = domain.ProbeStateError
		probe.LastError = fmt.Sprintf("GitHub user probe returned %d", userResponse.StatusCode)
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
