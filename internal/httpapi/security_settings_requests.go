package httpapi

import (
	"fmt"
	"strings"

	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
)

type rawSaveGitHubOutboundCredentialRequest struct {
	Token string `json:"token"`
}

type rawSecurityOIDCDraftRequest struct {
	IssuerURL            string   `json:"issuer_url"`
	ClientID             string   `json:"client_id"`
	ClientSecret         string   `json:"client_secret,omitempty"`
	RedirectMode         string   `json:"redirect_mode,omitempty"`
	FixedRedirectURL     string   `json:"fixed_redirect_url,omitempty"`
	RedirectURL          string   `json:"redirect_url,omitempty"`
	Scopes               []string `json:"scopes"`
	AllowedEmailDomains  []string `json:"allowed_email_domains,omitempty"`
	BootstrapAdminEmails []string `json:"bootstrap_admin_emails,omitempty"`
}

func parseSaveGitHubOutboundCredentialRequest(
	projectID uuid.UUID,
	raw rawSaveGitHubOutboundCredentialRequest,
) (githubauthservice.SaveCredentialInput, error) {
	token := strings.TrimSpace(raw.Token)
	if token == "" {
		return githubauthservice.SaveCredentialInput{}, fmt.Errorf("token must not be empty")
	}
	return githubauthservice.SaveCredentialInput{
		ProjectID: projectID,
		Scope:     githubauthdomain.ScopeProject,
		Token:     token,
	}, nil
}

func parseGitHubCredentialScopeRequest(projectID uuid.UUID) githubauthservice.ScopeInput {
	return githubauthservice.ScopeInput{
		ProjectID: projectID,
		Scope:     githubauthdomain.ScopeProject,
	}
}
