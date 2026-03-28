package httpapi

import (
	"net/http"
	"slices"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	securitySettingsLegacyGitHubEndpoint = "POST /api/v1/webhooks/github"
	securitySettingsConnectorEndpoint    = "POST /api/v1/webhooks/:connector/:provider"
	//nolint:gosec // This is an environment variable name, not a credential value.
	securitySettingsAgentTokenEnvVar    = "OPENASE_AGENT_TOKEN"
	securitySettingsAgentTokenTransport = "Bearer token"
)

type securityDeferredCapabilityResponse struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type securityAgentTokensResponse struct {
	Transport              string   `json:"transport"`
	EnvironmentVariable    string   `json:"environment_variable"`
	TokenPrefix            string   `json:"token_prefix"`
	DefaultScopes          []string `json:"default_scopes"`
	SupportedProjectScopes []string `json:"supported_project_scopes"`
}

type securityWebhookBoundaryResponse struct {
	LegacyGitHubEndpoint          string `json:"legacy_github_endpoint"`
	ConnectorEndpoint             string `json:"connector_endpoint"`
	LegacyGitHubSignatureRequired bool   `json:"legacy_github_signature_required"`
}

type securitySecretHygieneResponse struct {
	NotificationChannelConfigsRedacted bool `json:"notification_channel_configs_redacted"`
}

type securityGitHubTokenProbeResponse struct {
	State       string   `json:"state"`
	Configured  bool     `json:"configured"`
	Valid       bool     `json:"valid"`
	Permissions []string `json:"permissions"`
	RepoAccess  string   `json:"repo_access"`
	CheckedAt   *string  `json:"checked_at,omitempty"`
	LastError   string   `json:"last_error,omitempty"`
}

type securityGitHubOutboundCredentialResponse struct {
	Scope        string                           `json:"scope,omitempty"`
	Source       string                           `json:"source,omitempty"`
	TokenPreview string                           `json:"token_preview,omitempty"`
	Probe        securityGitHubTokenProbeResponse `json:"probe"`
}

type securitySettingsResponse struct {
	ProjectID     string                                   `json:"project_id"`
	AgentTokens   securityAgentTokensResponse              `json:"agent_tokens"`
	GitHub        securityGitHubOutboundCredentialResponse `json:"github"`
	Webhooks      securityWebhookBoundaryResponse          `json:"webhooks"`
	SecretHygiene securitySecretHygieneResponse            `json:"secret_hygiene"`
	Deferred      []securityDeferredCapabilityResponse     `json:"deferred"`
}

func (s *Server) registerSecuritySettingsRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/security-settings", s.handleGetSecuritySettings)
}

func (s *Server) handleGetSecuritySettings(c echo.Context) error {
	if s.catalog == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if _, err := s.catalog.GetProject(c.Request().Context(), projectID); err != nil {
		return writeCatalogError(c, err)
	}

	githubSecurity := buildMissingGitHubSecurityResponse()
	if s.githubAuthService != nil {
		resolved, err := s.githubAuthService.ReadProjectSecurity(c.Request().Context(), projectID)
		if err != nil {
			return writeAPIError(c, http.StatusBadGateway, "GITHUB_AUTH_UNAVAILABLE", err.Error())
		}
		githubSecurity = mapGitHubSecurityResponse(resolved)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"security": buildSecuritySettingsResponse(projectID, githubSecurity, s.github.WebhookSecret != ""),
	})
}

func buildSecuritySettingsResponse(
	projectID uuid.UUID,
	github securityGitHubOutboundCredentialResponse,
	legacyGitHubSignatureRequired bool,
) securitySettingsResponse {
	return securitySettingsResponse{
		ProjectID: projectID.String(),
		AgentTokens: securityAgentTokensResponse{
			Transport:              securitySettingsAgentTokenTransport,
			EnvironmentVariable:    securitySettingsAgentTokenEnvVar,
			TokenPrefix:            agentplatform.TokenPrefix,
			DefaultScopes:          slices.Clone(agentplatform.DefaultScopes()),
			SupportedProjectScopes: slices.Clone(agentplatform.SupportedScopes()),
		},
		GitHub: github,
		Webhooks: securityWebhookBoundaryResponse{
			LegacyGitHubEndpoint:          securitySettingsLegacyGitHubEndpoint,
			ConnectorEndpoint:             securitySettingsConnectorEndpoint,
			LegacyGitHubSignatureRequired: legacyGitHubSignatureRequired,
		},
		SecretHygiene: securitySecretHygieneResponse{
			NotificationChannelConfigsRedacted: true,
		},
		Deferred: []securityDeferredCapabilityResponse{
			{
				Key:     "human-auth",
				Title:   "Human sign-in and OIDC",
				Summary: "Browser-facing local token distribution and OIDC setup remain outside the shipped Settings boundary.",
			},
			{
				Key:     "rbac",
				Title:   "Roles and permission policy",
				Summary: "Role mapping, approval policy, and broader access governance still need a dedicated control-plane surface.",
			},
			{
				Key:     "provider-secret-rotation",
				Title:   "Provider credential rotation",
				Summary: "Provider auth config is still managed from the Agents surface rather than a security-specific settings API.",
			},
		},
	}
}

func buildMissingGitHubSecurityResponse() securityGitHubOutboundCredentialResponse {
	return securityGitHubOutboundCredentialResponse{
		Probe: mapGitHubTokenProbe(githubauthdomain.MissingProbe()),
	}
}

func mapGitHubSecurityResponse(item githubauthservice.ProjectSecurity) securityGitHubOutboundCredentialResponse {
	response := securityGitHubOutboundCredentialResponse{
		TokenPreview: item.TokenPreview,
		Probe:        mapGitHubTokenProbe(item.Probe),
	}
	if item.Scope.IsValid() {
		response.Scope = string(item.Scope)
	}
	if item.Source.IsValid() {
		response.Source = string(item.Source)
	}
	return response
}

func mapGitHubTokenProbe(probe githubauthdomain.TokenProbe) securityGitHubTokenProbeResponse {
	response := securityGitHubTokenProbeResponse{
		State:       string(probe.State),
		Configured:  probe.Configured,
		Valid:       probe.Valid,
		Permissions: slices.Clone(probe.Permissions),
		RepoAccess:  string(probe.RepoAccess),
		LastError:   probe.LastError,
	}
	if probe.CheckedAt != nil {
		checkedAt := probe.CheckedAt.UTC().Format(time.RFC3339)
		response.CheckedAt = &checkedAt
	}
	return response
}
