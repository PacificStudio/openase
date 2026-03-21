package httpapi

import (
	"net/http"
	"slices"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
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

type securitySettingsResponse struct {
	ProjectID     string                               `json:"project_id"`
	AgentTokens   securityAgentTokensResponse          `json:"agent_tokens"`
	Webhooks      securityWebhookBoundaryResponse      `json:"webhooks"`
	SecretHygiene securitySecretHygieneResponse        `json:"secret_hygiene"`
	Deferred      []securityDeferredCapabilityResponse `json:"deferred"`
}

func (s *Server) registerSecuritySettingsRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/security-settings", s.handleGetSecuritySettings)
}

func (s *Server) handleGetSecuritySettings(c echo.Context) error {
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"security": buildSecuritySettingsResponse(projectID, s.github.WebhookSecret != ""),
	})
}

func buildSecuritySettingsResponse(projectID uuid.UUID, legacyGitHubSignatureRequired bool) securitySettingsResponse {
	return securitySettingsResponse{
		ProjectID: projectID.String(),
		AgentTokens: securityAgentTokensResponse{
			Transport:              securitySettingsAgentTokenTransport,
			EnvironmentVariable:    securitySettingsAgentTokenEnvVar,
			TokenPrefix:            agentplatform.TokenPrefix,
			DefaultScopes:          slices.Clone(agentplatform.DefaultScopes()),
			SupportedProjectScopes: slices.Clone(agentplatform.SupportedScopes()),
		},
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
