package httpapi

import (
	"errors"
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

type securityGitHubCredentialSlotResponse struct {
	Scope        string                           `json:"scope,omitempty"`
	Configured   bool                             `json:"configured"`
	Source       string                           `json:"source,omitempty"`
	TokenPreview string                           `json:"token_preview,omitempty"`
	Probe        securityGitHubTokenProbeResponse `json:"probe"`
}

type securityGitHubOutboundCredentialResponse struct {
	Effective       securityGitHubCredentialSlotResponse `json:"effective"`
	Organization    securityGitHubCredentialSlotResponse `json:"organization"`
	ProjectOverride securityGitHubCredentialSlotResponse `json:"project_override"`
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
	api.PUT("/projects/:projectId/security-settings/github-outbound-credential", s.handlePutGitHubOutboundCredential)
	api.POST("/projects/:projectId/security-settings/github-outbound-credential/import-gh-cli", s.handleImportGitHubOutboundCredential)
	api.POST("/projects/:projectId/security-settings/github-outbound-credential/retest", s.handleRetestGitHubOutboundCredential)
	api.DELETE("/projects/:projectId/security-settings/github-outbound-credential", s.handleDeleteGitHubOutboundCredential)
}

func (s *Server) handleGetSecuritySettings(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}

	githubSecurity := buildMissingGitHubSecurityResponse()
	if s.githubAuthService != nil {
		resolved, err := s.githubAuthService.ReadProjectSecurity(c.Request().Context(), projectID)
		if err != nil {
			return writeGitHubAuthError(c, err)
		}
		githubSecurity = mapGitHubSecurityResponse(resolved)
	}

	return s.writeSecuritySettingsResponse(c, projectID, githubSecurity)
}

func (s *Server) handlePutGitHubOutboundCredential(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}

	var raw rawSaveGitHubOutboundCredentialRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}
	input, err := parseSaveGitHubOutboundCredentialRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	security, err := s.githubAuthService.SaveManualCredential(c.Request().Context(), input)
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return s.writeSecuritySettingsResponse(c, projectID, mapGitHubSecurityResponse(security))
}

func (s *Server) handleImportGitHubOutboundCredential(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}

	var raw rawGitHubCredentialScopeRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}
	input, err := parseGitHubCredentialScopeRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	security, err := s.githubAuthService.ImportGHCLICredential(c.Request().Context(), input)
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return s.writeSecuritySettingsResponse(c, projectID, mapGitHubSecurityResponse(security))
}

func (s *Server) handleRetestGitHubOutboundCredential(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}

	var raw rawGitHubCredentialScopeRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}
	input, err := parseGitHubCredentialScopeRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	security, err := s.githubAuthService.RetestCredential(c.Request().Context(), input)
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return s.writeSecuritySettingsResponse(c, projectID, mapGitHubSecurityResponse(security))
}

func (s *Server) handleDeleteGitHubOutboundCredential(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}

	input, err := parseGitHubCredentialScopeQuery(projectID, c.QueryParam("scope"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	security, err := s.githubAuthService.DeleteCredential(c.Request().Context(), input)
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return s.writeSecuritySettingsResponse(c, projectID, mapGitHubSecurityResponse(security))
}

func (s *Server) requireProjectSecurityContext(c echo.Context) (uuid.UUID, error) {
	if s.catalog == nil {
		return uuid.UUID{}, writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return uuid.UUID{}, writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if _, err := s.catalog.GetProject(c.Request().Context(), projectID); err != nil {
		return uuid.UUID{}, writeCatalogError(c, err)
	}
	return projectID, nil
}

func (s *Server) writeSecuritySettingsResponse(
	c echo.Context,
	projectID uuid.UUID,
	github securityGitHubOutboundCredentialResponse,
) error {
	return c.JSON(http.StatusOK, map[string]any{
		"security": buildSecuritySettingsResponse(projectID, github, s.github.WebhookSecret != ""),
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
				Key:     "github-device-flow",
				Title:   "GitHub Device Flow",
				Summary: "GitHub Device Flow remains deferred until the platform has OAuth app wiring for a fully managed browserless authorization hand-off.",
			},
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
		Effective:       buildMissingGitHubCredentialSlot(""),
		Organization:    buildMissingGitHubCredentialSlot(string(githubauthdomain.ScopeOrganization)),
		ProjectOverride: buildMissingGitHubCredentialSlot(string(githubauthdomain.ScopeProject)),
	}
}

func buildMissingGitHubCredentialSlot(scope string) securityGitHubCredentialSlotResponse {
	return securityGitHubCredentialSlotResponse{
		Scope:      scope,
		Configured: false,
		Probe:      mapGitHubTokenProbe(githubauthdomain.MissingProbe()),
	}
}

func mapGitHubSecurityResponse(item githubauthservice.ProjectSecurity) securityGitHubOutboundCredentialResponse {
	return securityGitHubOutboundCredentialResponse{
		Effective:       mapGitHubCredentialSlot(item.Effective),
		Organization:    mapGitHubCredentialSlot(item.Organization),
		ProjectOverride: mapGitHubCredentialSlot(item.ProjectOverride),
	}
}

func mapGitHubCredentialSlot(item githubauthservice.ScopedSecurity) securityGitHubCredentialSlotResponse {
	response := securityGitHubCredentialSlotResponse{
		Configured:   item.Configured,
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

func writeGitHubAuthError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, githubauthservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, githubauthservice.ErrInvalidInput):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, githubauthservice.ErrCredentialNotConfigured):
		return writeAPIError(c, http.StatusNotFound, "GITHUB_CREDENTIAL_NOT_CONFIGURED", err.Error())
	case errors.Is(err, githubauthservice.ErrGHCLIImportFailed):
		return writeAPIError(c, http.StatusBadGateway, "GITHUB_AUTH_IMPORT_FAILED", err.Error())
	default:
		return writeAPIError(c, http.StatusBadGateway, "GITHUB_AUTH_UNAVAILABLE", err.Error())
	}
}
