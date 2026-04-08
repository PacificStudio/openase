package secrets

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ScopeKind string

const (
	ScopeKindOrganization ScopeKind = "organization"
	ScopeKindProject      ScopeKind = "project"
)

func ParseScopeKind(raw string) (ScopeKind, error) {
	switch ScopeKind(strings.ToLower(strings.TrimSpace(raw))) {
	case ScopeKindOrganization:
		return ScopeKindOrganization, nil
	case ScopeKindProject:
		return ScopeKindProject, nil
	default:
		return "", fmt.Errorf("scope must be one of organization or project")
	}
}

type Kind string

const (
	KindOpaque Kind = "opaque"
)

func ParseKind(raw string) (Kind, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return KindOpaque, nil
	}
	switch Kind(trimmed) {
	case KindOpaque:
		return KindOpaque, nil
	default:
		return "", fmt.Errorf("kind must be opaque")
	}
}

type BindingScopeKind string

const (
	BindingScopeKindOrganization BindingScopeKind = "organization"
	BindingScopeKindProject      BindingScopeKind = "project"
	BindingScopeKindWorkflow     BindingScopeKind = "workflow"
	BindingScopeKindAgent        BindingScopeKind = "agent"
	BindingScopeKindTicket       BindingScopeKind = "ticket"
)

func ParseBindingScopeKind(raw string) (BindingScopeKind, error) {
	switch BindingScopeKind(strings.ToLower(strings.TrimSpace(raw))) {
	case BindingScopeKindOrganization:
		return BindingScopeKindOrganization, nil
	case BindingScopeKindProject:
		return BindingScopeKindProject, nil
	case BindingScopeKindWorkflow:
		return BindingScopeKindWorkflow, nil
	case BindingScopeKindAgent:
		return BindingScopeKindAgent, nil
	case BindingScopeKindTicket:
		return BindingScopeKindTicket, nil
	default:
		return "", fmt.Errorf("binding scope must be one of organization, project, workflow, agent, or ticket")
	}
}

type KeySource string

const (
	CipherAlgorithmAES256GCM             = "aes-256-gcm"
	KeySourceDatabaseDSNSHA256 KeySource = "database_dsn_sha256"
	DefaultKeyID                         = "database-dsn-sha256:v1"
)

type StoredValue struct {
	Algorithm  string    `json:"algorithm"`
	KeySource  KeySource `json:"key_source"`
	KeyID      string    `json:"key_id"`
	Preview    string    `json:"preview"`
	Nonce      string    `json:"nonce"`
	Ciphertext string    `json:"ciphertext"`
	RotatedAt  time.Time `json:"rotated_at"`
}

type Secret struct {
	ID             uuid.UUID   `json:"id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	ProjectID      uuid.UUID   `json:"project_id"`
	Scope          ScopeKind   `json:"scope"`
	Name           string      `json:"name"`
	Kind           Kind        `json:"kind"`
	Description    string      `json:"description"`
	DisabledAt     *time.Time  `json:"disabled_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	StoredValue    StoredValue `json:"stored_value"`
}

type InventorySecret struct {
	Secret      Secret             `json:"secret"`
	UsageCount  int                `json:"usage_count"`
	UsageScopes []BindingScopeKind `json:"usage_scopes"`
}

type Binding struct {
	ID              uuid.UUID        `json:"id"`
	OrganizationID  uuid.UUID        `json:"organization_id"`
	ProjectID       uuid.UUID        `json:"project_id"`
	SecretID        uuid.UUID        `json:"secret_id"`
	Scope           BindingScopeKind `json:"scope"`
	ScopeResourceID uuid.UUID        `json:"scope_resource_id"`
	BindingKey      string           `json:"binding_key"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type Candidate struct {
	Binding Binding
	Secret  Secret
}

type SelectedBinding struct {
	BindingKey string
	Binding    Binding
	Secret     Secret
}

type ResolvedSecret struct {
	BindingKey   string           `json:"binding_key"`
	BindingScope BindingScopeKind `json:"binding_scope"`
	SecretID     uuid.UUID        `json:"secret_id"`
	SecretName   string           `json:"secret_name"`
	SecretScope  ScopeKind        `json:"secret_scope"`
	SecretKind   Kind             `json:"secret_kind"`
	Value        string           `json:"value"`
}

var (
	namePattern                = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,127}$`)
	ErrResolutionScopeConflict = errors.New("secret binding precedence conflict")
)

func NormalizeName(raw string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if !namePattern.MatchString(normalized) {
		return "", fmt.Errorf("name must match ^[A-Z][A-Z0-9_]{0,127}$")
	}
	return normalized, nil
}

func ParseBindingKeys(raw []string) ([]string, error) {
	keys := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		normalized, err := NormalizeName(item)
		if err != nil {
			return nil, fmt.Errorf("binding key: %w", err)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		keys = append(keys, normalized)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("binding_keys must not be empty")
	}
	slices.Sort(keys)
	return keys, nil
}

func BindingKeysFromCandidates(candidates []Candidate) ([]string, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	raw := make([]string, 0, len(candidates))
	for _, item := range candidates {
		raw = append(raw, item.Binding.BindingKey)
	}
	return ParseBindingKeys(raw)
}

func DefaultCipherSeed(seed string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(seed)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func RedactValue(raw string) string {
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

func ProjectIDForSecretScope(scope ScopeKind, projectID uuid.UUID) uuid.UUID {
	if scope == ScopeKindOrganization {
		return uuid.Nil
	}
	return projectID
}

func ProjectIDForBindingScope(scope BindingScopeKind, projectID uuid.UUID) uuid.UUID {
	if scope == BindingScopeKindOrganization {
		return uuid.Nil
	}
	return projectID
}

func ResolutionRank(scope BindingScopeKind) int {
	switch scope {
	case BindingScopeKindTicket:
		return 0
	case BindingScopeKindWorkflow, BindingScopeKindAgent:
		return 1
	case BindingScopeKindProject:
		return 2
	case BindingScopeKindOrganization:
		return 3
	default:
		return 99
	}
}

func SelectBindings(keys []string, candidates []Candidate) ([]SelectedBinding, []string, error) {
	selected := make([]SelectedBinding, 0, len(keys))
	missing := make([]string, 0)
	byKey := make(map[string][]Candidate, len(keys))
	for _, candidate := range candidates {
		if candidate.Secret.DisabledAt != nil {
			continue
		}
		byKey[candidate.Binding.BindingKey] = append(byKey[candidate.Binding.BindingKey], candidate)
	}

	for _, key := range keys {
		items := byKey[key]
		if len(items) == 0 {
			missing = append(missing, key)
			continue
		}
		slices.SortFunc(items, func(a, b Candidate) int {
			ra := ResolutionRank(a.Binding.Scope)
			rb := ResolutionRank(b.Binding.Scope)
			if ra != rb {
				return ra - rb
			}
			if a.Binding.Scope != b.Binding.Scope {
				return strings.Compare(string(a.Binding.Scope), string(b.Binding.Scope))
			}
			return strings.Compare(a.Secret.Name, b.Secret.Name)
		})
		bestRank := ResolutionRank(items[0].Binding.Scope)
		best := make([]Candidate, 0, len(items))
		for _, item := range items {
			if ResolutionRank(item.Binding.Scope) != bestRank {
				break
			}
			best = append(best, item)
		}
		if err := ensureUnambiguousSelection(key, best); err != nil {
			return nil, nil, err
		}
		selected = append(selected, SelectedBinding{
			BindingKey: key,
			Binding:    best[0].Binding,
			Secret:     best[0].Secret,
		})
	}
	return selected, missing, nil
}

func ensureUnambiguousSelection(key string, items []Candidate) error {
	if len(items) <= 1 {
		return nil
	}
	firstSecret := items[0].Secret.ID
	for _, item := range items[1:] {
		if item.Secret.ID != firstSecret {
			return fmt.Errorf("%w: binding key %s is configured at multiple equal-precedence scopes", ErrResolutionScopeConflict, key)
		}
	}
	return nil
}
