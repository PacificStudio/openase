package httpapi

import (
	"net/http"

	"github.com/BetterAndBetterII/openase/internal/config"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/labstack/echo/v4"
)

type adminSecuritySettingsResponse struct {
	Auth             securityAuthSettingsResponse     `json:"auth"`
	ApprovalPolicies securityApprovalPoliciesResponse `json:"approval_policies"`
}

type adminSecuritySettingsEnvelope struct {
	Settings adminSecuritySettingsResponse `json:"settings"`
}

type adminSecurityOIDCEnableResponse struct {
	Activation securityOIDCActivationResponse `json:"activation"`
	Settings   adminSecuritySettingsResponse  `json:"settings"`
}

func (s *Server) registerAdminSecurityRoutes(api *echo.Group) {
	api.GET("/admin/security-settings", s.handleGetAdminSecuritySettings)
	api.PUT("/admin/security-settings/oidc-draft", s.handlePutAdminOIDCDraft)
	api.POST("/admin/security-settings/oidc-draft/test", s.handleTestAdminOIDCDraft)
	api.POST("/admin/security-settings/oidc-enable", s.handleEnableAdminOIDC)
}

func (s *Server) handleGetAdminSecuritySettings(c echo.Context) error {
	return s.writeAdminSecuritySettingsResponse(c)
}

func (s *Server) handlePutAdminOIDCDraft(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	storedAuth, err := editor.loadStoredAuth()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := parseSecurityOIDCDraftRequest(raw, draftInputFromConfig(storedAuth))
	mode := storedAuth.Mode
	if mode == "" {
		mode = s.auth.Mode
	}
	storedAuth, err = editor.saveDraft(draft, mode)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, adminSecuritySettingsEnvelope{
		Settings: buildAdminSecuritySettingsResponse(
			storedAuth,
			s.resolveApprovalPoliciesSummary(c),
			s.auth,
			s.cfg.Host,
			editor.resolvedPath(),
		),
	})
}

func (s *Server) handleTestAdminOIDCDraft(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	storedAuth, err := editor.loadStoredAuth()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := parseSecurityOIDCDraftRequest(raw, draftInputFromConfig(storedAuth))
	authCfg, err := completeOIDCAuthConfig(draft)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "OIDC_CONFIG_INVALID", err.Error())
	}
	diagnostics, err := humanauthservice.InspectOIDCProvider(c.Request().Context(), authCfg, nil)
	if err != nil {
		return writeAPIError(c, http.StatusBadGateway, "OIDC_TEST_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, securityOIDCTestResultResponse{
		Status:                "ok",
		Message:               "OIDC discovery succeeded. Saving this draft still keeps the active mode unchanged until you explicitly enable OIDC.",
		IssuerURL:             diagnostics.IssuerURL,
		AuthorizationEndpoint: diagnostics.AuthorizationEndpoint,
		TokenEndpoint:         diagnostics.TokenEndpoint,
		RedirectURL:           authCfg.OIDC.RedirectURL,
		Warnings:              securityPublicExposureWarnings(s.cfg.Host, s.auth.Mode),
	})
}

func (s *Server) handleEnableAdminOIDC(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	storedAuth, err := editor.loadStoredAuth()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := parseSecurityOIDCDraftRequest(raw, draftInputFromConfig(storedAuth))
	authCfg, err := completeOIDCAuthConfig(draft)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "OIDC_CONFIG_INVALID", err.Error())
	}
	if _, err := humanauthservice.InspectOIDCProvider(c.Request().Context(), authCfg, nil); err != nil {
		return writeAPIError(c, http.StatusBadGateway, "OIDC_ENABLE_FAILED", err.Error())
	}

	storedAuth, err = editor.saveDraft(draft, config.AuthModeOIDC)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, adminSecurityOIDCEnableResponse{
		Activation: securityOIDCActivationResponse{
			Status:          "configured",
			Message:         "OIDC is now the configured auth mode on disk. Restart the service and complete the first OIDC sign-in with a bootstrap admin email to activate it in the running control plane.",
			RestartRequired: true,
			NextSteps: []string{
				"Restart OpenASE so the running control plane picks up auth.mode=oidc.",
				"Sign in through the browser with a bootstrap admin email.",
				"Verify instance admins, organization admins, and project bindings after the first login, then narrow the bootstrap admin list.",
			},
		},
		Settings: buildAdminSecuritySettingsResponse(
			storedAuth,
			s.resolveApprovalPoliciesSummary(c),
			s.auth,
			s.cfg.Host,
			editor.resolvedPath(),
		),
	})
}

func (s *Server) writeAdminSecuritySettingsResponse(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	storedAuth, err := editor.loadStoredAuth()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "SECURITY_SETTINGS_CONFIG_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, adminSecuritySettingsEnvelope{
		Settings: buildAdminSecuritySettingsResponse(
			storedAuth,
			s.resolveApprovalPoliciesSummary(c),
			s.auth,
			s.cfg.Host,
			editor.resolvedPath(),
		),
	})
}

func buildAdminSecuritySettingsResponse(
	storedAuth config.AuthConfig,
	approvalPolicies securityApprovalPoliciesResponse,
	activeAuth config.AuthConfig,
	host string,
	configPath string,
) adminSecuritySettingsResponse {
	if storedAuth.Mode == "" {
		storedAuth = activeAuth
	}
	return adminSecuritySettingsResponse{
		Auth:             buildSecurityAuthSettingsResponse(activeAuth, storedAuth, configPath, host),
		ApprovalPolicies: approvalPolicies,
	}
}
