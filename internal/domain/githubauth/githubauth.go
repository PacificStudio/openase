package githubauth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Scope string

const (
	ScopeOrganization Scope = "organization"
	ScopeProject      Scope = "project"
)

func (s Scope) IsValid() bool {
	switch s {
	case ScopeOrganization, ScopeProject:
		return true
	default:
		return false
	}
}

type Source string

const (
	SourceDeviceFlow  Source = "device_flow"
	SourceGHCLIImport Source = "gh_cli_import"
	SourceManualPaste Source = "manual_paste"
)

func (s Source) IsValid() bool {
	switch s {
	case SourceDeviceFlow, SourceGHCLIImport, SourceManualPaste:
		return true
	default:
		return false
	}
}

type ProbeState string

const (
	ProbeStateMissing                 ProbeState = "missing"
	ProbeStateConfigured              ProbeState = "configured"
	ProbeStateProbing                 ProbeState = "probing"
	ProbeStateValid                   ProbeState = "valid"
	ProbeStateInsufficientPermissions ProbeState = "insufficient_permissions"
	ProbeStateRevoked                 ProbeState = "revoked"
	ProbeStateError                   ProbeState = "error"
)

func (s ProbeState) IsValid() bool {
	switch s {
	case ProbeStateMissing,
		ProbeStateConfigured,
		ProbeStateProbing,
		ProbeStateValid,
		ProbeStateInsufficientPermissions,
		ProbeStateRevoked,
		ProbeStateError:
		return true
	default:
		return false
	}
}

type RepoAccess string

const (
	RepoAccessNotChecked RepoAccess = "not_checked"
	RepoAccessGranted    RepoAccess = "granted"
	RepoAccessDenied     RepoAccess = "denied"
)

func (r RepoAccess) IsValid() bool {
	switch r {
	case RepoAccessNotChecked, RepoAccessGranted, RepoAccessDenied:
		return true
	default:
		return false
	}
}

type StoredCredential struct {
	Algorithm    string    `json:"algorithm"`
	TokenPreview string    `json:"token_preview"`
	Nonce        string    `json:"nonce"`
	Ciphertext   string    `json:"ciphertext"`
	Source       Source    `json:"source"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TokenProbe struct {
	State       ProbeState `json:"state"`
	Configured  bool       `json:"configured"`
	Valid       bool       `json:"valid"`
	Login       string     `json:"login,omitempty"`
	Permissions []string   `json:"permissions"`
	RepoAccess  RepoAccess `json:"repo_access"`
	CheckedAt   *time.Time `json:"checked_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
}

type ProjectContext struct {
	ProjectID              uuid.UUID
	OrganizationID         uuid.UUID
	ProjectRepositoryURL   string
	OrganizationCredential *StoredCredential
	OrganizationProbe      *TokenProbe
	ProjectCredential      *StoredCredential
	ProjectProbe           *TokenProbe
}

type OrgContext struct {
	OrganizationID uuid.UUID
	Credential     *StoredCredential
	Probe          *TokenProbe
}

type ResolvedCredential struct {
	Scope        Scope
	Source       Source
	Token        string
	TokenPreview string
	Probe        TokenProbe
}

type RepositoryRef struct {
	Owner string
	Name  string
}

func (r RepositoryRef) String() string {
	return r.Owner + "/" + r.Name
}

func MissingProbe() TokenProbe {
	return TokenProbe{
		State:      ProbeStateMissing,
		Configured: false,
		Valid:      false,
		RepoAccess: RepoAccessNotChecked,
	}
}

func ConfiguredProbe() TokenProbe {
	return TokenProbe{
		State:      ProbeStateConfigured,
		Configured: true,
		Valid:      false,
		RepoAccess: RepoAccessNotChecked,
	}
}

func NormalizeProbe(raw *TokenProbe, configured bool) TokenProbe {
	if !configured {
		return MissingProbe()
	}
	if raw == nil {
		return ConfiguredProbe()
	}

	probe := TokenProbe{
		State:       raw.State,
		Configured:  true,
		Valid:       raw.Valid,
		Login:       strings.TrimSpace(raw.Login),
		Permissions: append([]string(nil), raw.Permissions...),
		RepoAccess:  raw.RepoAccess,
		CheckedAt:   cloneTime(raw.CheckedAt),
		LastError:   strings.TrimSpace(raw.LastError),
	}
	if !probe.State.IsValid() {
		probe.State = ProbeStateConfigured
	}
	if !probe.RepoAccess.IsValid() {
		probe.RepoAccess = RepoAccessNotChecked
	}
	slices.Sort(probe.Permissions)
	probe.Permissions = slices.Compact(probe.Permissions)
	return probe
}

func ParseGitHubRepositoryURL(raw string) (RepositoryRef, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return RepositoryRef{}, false
	}

	if repoRef, ok := parseGitHubSCPURL(trimmed); ok {
		return repoRef, true
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return RepositoryRef{}, false
	}
	switch {
	case strings.EqualFold(parsed.Scheme, "https") && strings.EqualFold(parsed.Host, "github.com"):
		return parseGitHubRepositoryPath(parsed.Path)
	case strings.EqualFold(parsed.Scheme, "ssh") && strings.EqualFold(parsed.Host, "github.com"):
		return parseGitHubRepositoryPath(parsed.Path)
	default:
		return RepositoryRef{}, false
	}
}

func NormalizeGitHubRepositoryURL(raw string) (string, bool) {
	repoRef, ok := ParseGitHubRepositoryURL(raw)
	if !ok {
		return "", false
	}
	return "https://github.com/" + repoRef.Owner + "/" + repoRef.Name + ".git", true
}

func parseGitHubSCPURL(raw string) (RepositoryRef, bool) {
	if !strings.HasPrefix(raw, "git@github.com:") {
		return RepositoryRef{}, false
	}
	return parseGitHubRepositoryPath(strings.TrimPrefix(raw, "git@github.com:"))
}

func parseGitHubRepositoryPath(rawPath string) (RepositoryRef, bool) {
	repoPath := strings.Trim(path.Clean(rawPath), "/")
	segments := strings.Split(repoPath, "/")
	if len(segments) != 2 {
		return RepositoryRef{}, false
	}

	name := strings.TrimSuffix(segments[1], ".git")
	if strings.TrimSpace(segments[0]) == "" || strings.TrimSpace(name) == "" {
		return RepositoryRef{}, false
	}

	return RepositoryRef{
		Owner: strings.ToLower(strings.TrimSpace(segments[0])),
		Name:  strings.ToLower(strings.TrimSpace(name)),
	}, true
}

func RedactToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= 4 {
		return strings.Repeat("*", len(runes))
	}
	prefix := string(runes[:min(7, len(runes)-4)])
	suffix := string(runes[len(runes)-4:])
	return prefix + "..." + suffix
}

func DefaultCipherSeed(dsn string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(dsn)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func ResolveProjectCredential(context ProjectContext, decrypt func(StoredCredential) (string, error)) (ResolvedCredential, error) {
	selectedCredential := context.OrganizationCredential
	selectedProbe := context.OrganizationProbe
	scope := ScopeOrganization
	if context.ProjectCredential != nil {
		selectedCredential = context.ProjectCredential
		selectedProbe = context.ProjectProbe
		scope = ScopeProject
	}
	if selectedCredential == nil {
		return ResolvedCredential{
			Probe: MissingProbe(),
		}, nil
	}
	token, err := decrypt(*selectedCredential)
	if err != nil {
		return ResolvedCredential{}, fmt.Errorf("decrypt %s GitHub credential: %w", scope, err)
	}

	return ResolvedCredential{
		Scope:        scope,
		Source:       selectedCredential.Source,
		Token:        token,
		TokenPreview: strings.TrimSpace(selectedCredential.TokenPreview),
		Probe:        NormalizeProbe(selectedProbe, true),
	}, nil
}

func (c ProjectContext) CredentialForScope(scope Scope) (*StoredCredential, *TokenProbe, error) {
	switch scope {
	case ScopeOrganization:
		return c.OrganizationCredential, c.OrganizationProbe, nil
	case ScopeProject:
		return c.ProjectCredential, c.ProjectProbe, nil
	default:
		return nil, nil, fmt.Errorf("invalid GitHub credential scope %q", scope)
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}
