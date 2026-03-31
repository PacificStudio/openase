package httpapi

import (
	"fmt"
	"strings"

	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
)

type rawSaveGitHubOutboundCredentialRequest struct {
	Scope string `json:"scope"`
	Token string `json:"token"`
}

type rawGitHubCredentialScopeRequest struct {
	Scope string `json:"scope"`
}

func parseSaveGitHubOutboundCredentialRequest(
	projectID uuid.UUID,
	raw rawSaveGitHubOutboundCredentialRequest,
) (githubauthservice.SaveCredentialInput, error) {
	scope, err := parseGitHubCredentialScope(raw.Scope)
	if err != nil {
		return githubauthservice.SaveCredentialInput{}, err
	}
	token := strings.TrimSpace(raw.Token)
	if token == "" {
		return githubauthservice.SaveCredentialInput{}, fmt.Errorf("token must not be empty")
	}

	return githubauthservice.SaveCredentialInput{
		ProjectID: projectID,
		Scope:     scope,
		Token:     token,
	}, nil
}

func parseGitHubCredentialScopeRequest(
	projectID uuid.UUID,
	raw rawGitHubCredentialScopeRequest,
) (githubauthservice.ScopeInput, error) {
	scope, err := parseGitHubCredentialScope(raw.Scope)
	if err != nil {
		return githubauthservice.ScopeInput{}, err
	}

	return githubauthservice.ScopeInput{
		ProjectID: projectID,
		Scope:     scope,
	}, nil
}

func parseGitHubCredentialScopeQuery(projectID uuid.UUID, raw string) (githubauthservice.ScopeInput, error) {
	scope, err := parseGitHubCredentialScope(raw)
	if err != nil {
		return githubauthservice.ScopeInput{}, err
	}

	return githubauthservice.ScopeInput{
		ProjectID: projectID,
		Scope:     scope,
	}, nil
}

func parseGitHubCredentialScope(raw string) (githubauthdomain.Scope, error) {
	scope := githubauthdomain.Scope(strings.TrimSpace(raw))
	if !scope.IsValid() {
		return "", fmt.Errorf("scope must be one of organization or project")
	}
	return scope, nil
}
