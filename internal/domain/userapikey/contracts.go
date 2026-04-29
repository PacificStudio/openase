package userapikey

import (
	"fmt"
	"strings"
	"time"

	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	"github.com/google/uuid"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusRevoked  Status = "revoked"
)

type APIKey struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	TokenPrefix string
	TokenHint   string
	Scopes      []string
	Status      Status
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DisabledAt  *time.Time
	RevokedAt   *time.Time
}

type CreateInput struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
}

type CreateResult struct {
	APIKey         APIKey
	PlainTextToken string
}

type ParseCreateInput struct {
	ProjectID string
	UserID    uuid.UUID
	Name      string
	Scopes    []string
	ExpiresAt *string
}

func ParseCreate(raw ParseCreateInput) (CreateInput, error) {
	projectID, err := uuid.Parse(strings.TrimSpace(raw.ProjectID))
	if err != nil {
		return CreateInput{}, fmt.Errorf("project_id must be a valid UUID")
	}
	if raw.UserID == uuid.Nil {
		return CreateInput{}, fmt.Errorf("user_id must be a valid UUID")
	}
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return CreateInput{}, fmt.Errorf("name must not be empty")
	}
	requestedScopes := make([]string, 0, len(raw.Scopes))
	for _, item := range raw.Scopes {
		scope := strings.TrimSpace(item)
		if scope == "" {
			return CreateInput{}, fmt.Errorf("scopes must not contain empty values")
		}
		if !contains(requestedScopes, scope) {
			requestedScopes = append(requestedScopes, scope)
		}
	}
	if len(requestedScopes) == 0 {
		return CreateInput{}, fmt.Errorf("at least one scope must be selected")
	}

	var expiresAt *time.Time
	if raw.ExpiresAt != nil && strings.TrimSpace(*raw.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw.ExpiresAt))
		if err != nil {
			return CreateInput{}, fmt.Errorf("expires_at must be RFC3339")
		}
		utc := parsed.UTC()
		expiresAt = &utc
	}

	return CreateInput{
		ProjectID: projectID,
		UserID:    raw.UserID,
		Name:      name,
		Scopes:    requestedScopes,
		ExpiresAt: expiresAt,
	}, nil
}

func SupportedScopeGroups(scopes []string) []agentplatformdomain.ScopeGroup {
	if len(scopes) == 0 {
		return nil
	}
	groups := make(map[string][]string)
	for _, scope := range scopes {
		category, _, found := strings.Cut(scope, ".")
		if !found {
			category = scope
		}
		groups[category] = append(groups[category], scope)
	}
	categories := make([]string, 0, len(groups))
	for category := range groups {
		categories = append(categories, category)
	}
	// small local sort avoids a new utility dependency here.
	for i := 0; i < len(categories); i++ {
		for j := i + 1; j < len(categories); j++ {
			if categories[j] < categories[i] {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}
	result := make([]agentplatformdomain.ScopeGroup, 0, len(categories))
	for _, category := range categories {
		items := append([]string(nil), groups[category]...)
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				if items[j] < items[i] {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
		result = append(result, agentplatformdomain.ScopeGroup{Category: category, Scopes: items})
	}
	return result
}

func contains(items []string, candidate string) bool {
	for _, item := range items {
		if item == candidate {
			return true
		}
	}
	return false
}
