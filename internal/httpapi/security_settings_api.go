package httpapi

import (
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	securitySettingsConnectorEndpoint = "Not supported in current version"
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
	Transport              string                       `json:"transport"`
	EnvironmentVariable    string                       `json:"environment_variable"`
	TokenPrefix            string                       `json:"token_prefix"`
	DefaultScopes          []string                     `json:"default_scopes"`
	SupportedProjectScopes []string                     `json:"supported_project_scopes"`
	SupportedScopeGroups   []securityScopeGroupResponse `json:"supported_scope_groups"`
}

type securityScopeGroupResponse struct {
	Category string   `json:"category"`
	Scopes   []string `json:"scopes"`
}

type securityWebhookBoundaryResponse struct {
	ConnectorEndpoint string `json:"connector_endpoint"`
}

type securitySecretHygieneResponse struct {
	NotificationChannelConfigsRedacted bool `json:"notification_channel_configs_redacted"`
}

type securityApprovalPoliciesResponse struct {
	Status     string `json:"status"`
	RulesCount int    `json:"rules_count"`
	Summary    string `json:"summary"`
}

type securityGitHubTokenProbeResponse struct {
	State       string   `json:"state"`
	Configured  bool     `json:"configured"`
	Valid       bool     `json:"valid"`
	Login       string   `json:"login,omitempty"`
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
	ProjectID        string                                   `json:"project_id"`
	Auth             securityAuthSettingsResponse             `json:"auth"`
	AgentTokens      securityAgentTokensResponse              `json:"agent_tokens"`
	GitHub           securityGitHubOutboundCredentialResponse `json:"github"`
	Webhooks         securityWebhookBoundaryResponse          `json:"webhooks"`
	SecretHygiene    securitySecretHygieneResponse            `json:"secret_hygiene"`
	ApprovalPolicies securityApprovalPoliciesResponse         `json:"approval_policies"`
	Deferred         []securityDeferredCapabilityResponse     `json:"deferred"`
}

func (s *Server) registerSecuritySettingsRoutes(api *echo.Group) {
	api.GET("/orgs/:orgId/security-settings/secrets", s.handleListOrganizationScopedSecrets)
	api.POST("/orgs/:orgId/security-settings/secrets", s.handleCreateOrganizationScopedSecret)
	api.POST("/orgs/:orgId/security-settings/secrets/:secretId/rotate", s.handleRotateOrganizationScopedSecret)
	api.POST("/orgs/:orgId/security-settings/secrets/:secretId/disable", s.handleDisableOrganizationScopedSecret)
	api.DELETE("/orgs/:orgId/security-settings/secrets/:secretId", s.handleDeleteOrganizationScopedSecret)
	api.GET("/projects/:projectId/security-settings", s.handleGetSecuritySettings)
	api.GET("/projects/:projectId/security-settings/secrets", s.handleListScopedSecrets)
	api.POST("/projects/:projectId/security-settings/secrets", s.handleCreateScopedSecret)
	api.PATCH("/projects/:projectId/security-settings/secrets/:secretId", s.handlePatchScopedSecret)
	api.POST("/projects/:projectId/security-settings/secrets/:secretId/rotate", s.handleRotateScopedSecret)
	api.POST("/projects/:projectId/security-settings/secrets/:secretId/disable", s.handleDisableScopedSecret)
	api.DELETE("/projects/:projectId/security-settings/secrets/:secretId", s.handleDeleteScopedSecret)
	api.POST("/projects/:projectId/security-settings/secrets/resolve-for-runtime", s.handleResolveScopedSecretsForRuntime)
	api.PUT("/projects/:projectId/security-settings/oidc-draft", s.handlePutOIDCDraft)
	api.POST("/projects/:projectId/security-settings/oidc-draft/test", s.handleTestOIDCDraft)
	api.POST("/projects/:projectId/security-settings/oidc-enable", s.handleEnableOIDC)
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

func (s *Server) handlePutOIDCDraft(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}

	service, err := s.requireInstanceAccessControlService()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}
	current, err := service.Read(c.Request().Context())
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := draftOIDCConfigFromRequest(raw, current.State)
	stored, err := service.SaveDraft(c.Request().Context(), draft)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"security": buildSecuritySettingsResponse(
			projectID,
			stored,
			s.readGitHubSecurity(c, projectID),
			s.resolveApprovalPoliciesSummary(c),
			s.cfg.Host,
		),
	})
}

func (s *Server) handleTestOIDCDraft(c echo.Context) error {
	if _, err := s.requireProjectSecurityContext(c); err != nil {
		return err
	}

	service, err := s.requireInstanceAccessControlService()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}
	current, err := service.Read(c.Request().Context())
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}
	runtimeState := iam.ResolveRuntimeAccessControlState(current.State)

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := draftOIDCConfigFromRequest(raw, current.State)
	active, err := activeOIDCConfigFromDraft(draft)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "OIDC_CONFIG_INVALID", err.Error())
	}
	authCfg := completeOIDCAuthConfigFromAccessControl(active)
	diagnostics, err := humanauthservice.InspectOIDCProvider(c.Request().Context(), authCfg, nil)
	if err != nil {
		_ = service.SaveValidation(c.Request().Context(), securityOIDCValidationFailureMetadata(err.Error(), authCfg.OIDC.RedirectURL, s.cfg.Host, runtimeConfigAuthMode(runtimeState)))
		return writeAPIError(c, http.StatusBadGateway, "OIDC_TEST_FAILED", err.Error())
	}
	response := securityOIDCTestResultResponse{
		Status:                "ok",
		Message:               "OIDC discovery succeeded. Saving this draft still keeps the active mode unchanged until you explicitly enable OIDC.",
		IssuerURL:             diagnostics.IssuerURL,
		AuthorizationEndpoint: diagnostics.AuthorizationEndpoint,
		TokenEndpoint:         diagnostics.TokenEndpoint,
		RedirectURL:           authCfg.OIDC.RedirectURL,
		Warnings:              securityPublicExposureWarnings(s.cfg.Host, runtimeConfigAuthMode(runtimeState)),
	}
	if err := service.SaveValidation(c.Request().Context(), securityOIDCValidationSuccessMetadata(response)); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleEnableOIDC(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}

	service, err := s.requireInstanceAccessControlService()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}
	current, err := service.Read(c.Request().Context())
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}
	runtimeState := iam.ResolveRuntimeAccessControlState(current.State)

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := draftOIDCConfigFromRequest(raw, current.State)
	active, err := activeOIDCConfigFromDraft(draft)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "OIDC_CONFIG_INVALID", err.Error())
	}
	authCfg := completeOIDCAuthConfigFromAccessControl(active)
	diagnostics, err := humanauthservice.InspectOIDCProvider(c.Request().Context(), authCfg, nil)
	if err != nil {
		_ = service.SaveValidation(c.Request().Context(), securityOIDCValidationFailureMetadata(err.Error(), authCfg.OIDC.RedirectURL, s.cfg.Host, runtimeConfigAuthMode(runtimeState)))
		return writeAPIError(c, http.StatusBadGateway, "OIDC_ENABLE_FAILED", err.Error())
	}
	successValidation := securityOIDCValidationSuccessMetadata(securityOIDCTestResultResponse{
		Status:                "ok",
		Message:               "OIDC discovery succeeded. Saving this draft still keeps the active mode unchanged until you explicitly enable OIDC.",
		IssuerURL:             diagnostics.IssuerURL,
		AuthorizationEndpoint: diagnostics.AuthorizationEndpoint,
		TokenEndpoint:         diagnostics.TokenEndpoint,
		RedirectURL:           authCfg.OIDC.RedirectURL,
		Warnings:              securityPublicExposureWarnings(s.cfg.Host, runtimeConfigAuthMode(runtimeState)),
	})
	if err := service.SaveValidation(c.Request().Context(), successValidation); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	now := time.Now().UTC()
	stored, err := service.Activate(c.Request().Context(), active, iam.OIDCActivationMetadata{
		ActivatedAt: &now,
		Source:      "project_security_settings_api",
		Message:     "OIDC is now active for this instance. Complete the first OIDC sign-in with a bootstrap admin email to continue with managed access control.",
	})
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	githubSecurity := s.readGitHubSecurity(c, projectID)
	return c.JSON(http.StatusOK, securityOIDCEnableResponse{
		Activation: securityOIDCActivationResponse{
			Status:          "configured",
			Message:         "OIDC is now active for this instance. Complete the first OIDC sign-in with a bootstrap admin email to continue with managed access control.",
			RestartRequired: false,
			NextSteps: []string{
				"Sign in through the browser with a bootstrap admin email.",
				"Verify instance, organization, and project role bindings after the first login, then narrow the bootstrap admin list.",
			},
		},
		Security: buildSecuritySettingsResponse(projectID, stored, githubSecurity, s.resolveApprovalPoliciesSummary(c), s.cfg.Host),
	})
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
	if s.catalog.Empty() {
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

func (s *Server) requireOrganizationSecurityContext(c echo.Context) (uuid.UUID, error) {
	if s.catalog.Empty() {
		return uuid.UUID{}, writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}

	organizationID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return uuid.UUID{}, err
	}
	if _, err := s.catalog.GetOrganization(c.Request().Context(), organizationID); err != nil {
		return uuid.UUID{}, writeCatalogError(c, err)
	}
	return organizationID, nil
}

func (s *Server) writeSecuritySettingsResponse(
	c echo.Context,
	projectID uuid.UUID,
	github securityGitHubOutboundCredentialResponse,
) error {
	stored, err := s.readInstanceAccessControlState(c)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{
		"security": buildSecuritySettingsResponse(projectID, stored, github, s.resolveApprovalPoliciesSummary(c), s.cfg.Host),
	})
}

func (s *Server) readGitHubSecurity(c echo.Context, projectID uuid.UUID) securityGitHubOutboundCredentialResponse {
	githubSecurity := buildMissingGitHubSecurityResponse()
	if s.githubAuthService == nil {
		return githubSecurity
	}
	resolved, err := s.githubAuthService.ReadProjectSecurity(c.Request().Context(), projectID)
	if err != nil {
		return githubSecurity
	}
	return mapGitHubSecurityResponse(resolved)
}

func buildSecuritySettingsResponse(
	projectID uuid.UUID,
	stored accesscontrolservice.ReadResult,
	github securityGitHubOutboundCredentialResponse,
	approvalPolicies securityApprovalPoliciesResponse,
	host string,
) securitySettingsResponse {
	runtimeState := iam.ResolveRuntimeAccessControlState(stored.State)
	return securitySettingsResponse{
		ProjectID: projectID.String(),
		Auth:      buildSecurityAuthSettingsResponseFromAccessControl(runtimeState, stored.State, stored.StorageLocation, host),
		AgentTokens: securityAgentTokensResponse{
			Transport:              securitySettingsAgentTokenTransport,
			EnvironmentVariable:    securitySettingsAgentTokenEnvVar,
			TokenPrefix:            agentplatform.TokenPrefix,
			DefaultScopes:          slices.Clone(agentplatform.DefaultScopes()),
			SupportedProjectScopes: slices.Clone(agentplatform.SupportedScopes()),
			SupportedScopeGroups:   mapSecurityScopeGroups(agentplatform.SupportedScopeGroups()),
		},
		GitHub: github,
		Webhooks: securityWebhookBoundaryResponse{
			ConnectorEndpoint: securitySettingsConnectorEndpoint,
		},
		SecretHygiene: securitySecretHygieneResponse{
			NotificationChannelConfigsRedacted: true,
		},
		ApprovalPolicies: approvalPolicies,
		Deferred: []securityDeferredCapabilityResponse{
			{
				Key:     "github-device-flow",
				Title:   "GitHub Device Flow",
				Summary: "GitHub Device Flow remains deferred until the platform has OAuth app wiring for a fully managed browserless authorization hand-off.",
			},
			{
				Key:     "provider-secret-rotation",
				Title:   "Provider credential rotation",
				Summary: "Provider auth config is still managed from the Agents surface rather than a security-specific settings API.",
			},
		},
	}
}

func (s *Server) resolveApprovalPoliciesSummary(c echo.Context) securityApprovalPoliciesResponse {
	response := securityApprovalPoliciesResponse{
		Status:  "reserved",
		Summary: "Approval policy storage is reserved for future second-factor or approver requirements and stays separate from RBAC grants.",
	}
	if s == nil || s.humanAuthService == nil {
		return response
	}
	count, err := s.humanAuthService.CountApprovalPolicies(c.Request().Context())
	if err != nil {
		if errors.Is(err, humanauthservice.ErrAuthDisabled) {
			return response
		}
		response.Status = "unavailable"
		response.Summary = "Approval policy diagnostics are unavailable because stored policy rules could not be queried."
		return response
	}
	response.RulesCount = count
	return response
}

func mapSecurityScopeGroups(items []agentplatform.ScopeGroup) []securityScopeGroupResponse {
	response := make([]securityScopeGroupResponse, 0, len(items))
	for _, item := range items {
		response = append(response, securityScopeGroupResponse{
			Category: item.Category,
			Scopes:   append([]string(nil), item.Scopes...),
		})
	}
	return response
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
		Login:       probe.Login,
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
