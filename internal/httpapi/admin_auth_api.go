package httpapi

import (
	"net/http"

	"github.com/BetterAndBetterII/openase/internal/config"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/labstack/echo/v4"
)

type adminAuthResponse struct {
	Auth securityAuthSettingsResponse `json:"auth"`
}

type adminAuthModeTransitionResponse struct {
	Transition securityOIDCActivationResponse `json:"transition"`
	Auth       securityAuthSettingsResponse   `json:"auth"`
}

func (s *Server) registerAdminAuthRoutes(api *echo.Group) {
	api.GET("/admin/auth", s.handleGetAdminAuth)
	api.PUT("/admin/auth/oidc-draft", s.handlePutAdminOIDCDraft)
	api.POST("/admin/auth/oidc-draft/test", s.handleTestAdminOIDCDraft)
	api.POST("/admin/auth/oidc-enable", s.handleEnableAdminOIDC)
	api.POST("/admin/auth/disable", s.handleDisableAdminAuth)
}

func (s *Server) handleGetAdminAuth(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	state, err := editor.loadStoredState()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, adminAuthResponse{
		Auth: buildSecurityAuthSettingsResponse(
			s.auth,
			state.Auth,
			state.LastValidation,
			editor.resolvedPath(),
			s.cfg.Host,
		),
	})
}

func (s *Server) handlePutAdminOIDCDraft(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	state, err := editor.loadStoredState()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := parseSecurityOIDCDraftRequest(raw, draftInputFromConfig(state.Auth))
	mode := state.Auth.Mode
	if mode == "" {
		mode = s.auth.Mode
	}
	state, err = editor.saveDraft(draft, mode)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, adminAuthResponse{
		Auth: buildSecurityAuthSettingsResponse(
			s.auth,
			state.Auth,
			state.LastValidation,
			editor.resolvedPath(),
			s.cfg.Host,
		),
	})
}

func (s *Server) handleTestAdminOIDCDraft(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	state, err := editor.loadStoredState()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := parseSecurityOIDCDraftRequest(raw, draftInputFromConfig(state.Auth))
	authCfg, err := completeOIDCAuthConfig(draft)
	if err != nil {
		_, _ = editor.saveValidation(securityOIDCValidationFailureRecord(err.Error(), draft.RedirectURL, s.cfg.Host, s.auth.Mode))
		return writeAPIError(c, http.StatusBadRequest, "OIDC_CONFIG_INVALID", err.Error())
	}

	diagnostics, err := humanauthservice.InspectOIDCProvider(c.Request().Context(), authCfg, nil)
	if err != nil {
		_, _ = editor.saveValidation(securityOIDCValidationFailureRecord(err.Error(), authCfg.OIDC.RedirectURL, s.cfg.Host, s.auth.Mode))
		return writeAPIError(c, http.StatusBadGateway, "OIDC_TEST_FAILED", err.Error())
	}

	response := securityOIDCTestResultResponse{
		Status:                "ok",
		Message:               "OIDC discovery succeeded. Saving this draft still keeps the active auth mode unchanged until you explicitly enable OIDC.",
		IssuerURL:             diagnostics.IssuerURL,
		AuthorizationEndpoint: diagnostics.AuthorizationEndpoint,
		TokenEndpoint:         diagnostics.TokenEndpoint,
		RedirectURL:           authCfg.OIDC.RedirectURL,
		Warnings:              securityPublicExposureWarnings(s.cfg.Host, s.auth.Mode),
	}
	if _, err := editor.saveValidation(securityOIDCValidationSuccessRecord(response)); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleEnableAdminOIDC(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	state, err := editor.loadStoredState()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := parseSecurityOIDCDraftRequest(raw, draftInputFromConfig(state.Auth))
	authCfg, err := completeOIDCAuthConfig(draft)
	if err != nil {
		_, _ = editor.saveValidation(securityOIDCValidationFailureRecord(err.Error(), draft.RedirectURL, s.cfg.Host, s.auth.Mode))
		return writeAPIError(c, http.StatusBadRequest, "OIDC_CONFIG_INVALID", err.Error())
	}

	diagnostics, err := humanauthservice.InspectOIDCProvider(c.Request().Context(), authCfg, nil)
	if err != nil {
		_, _ = editor.saveValidation(securityOIDCValidationFailureRecord(err.Error(), authCfg.OIDC.RedirectURL, s.cfg.Host, s.auth.Mode))
		return writeAPIError(c, http.StatusBadGateway, "OIDC_ENABLE_FAILED", err.Error())
	}

	if _, err := editor.saveValidation(securityOIDCValidationSuccessRecord(securityOIDCTestResultResponse{
		Status:                "ok",
		Message:               "OIDC discovery succeeded. Saving this draft still keeps the active auth mode unchanged until you explicitly enable OIDC.",
		IssuerURL:             diagnostics.IssuerURL,
		AuthorizationEndpoint: diagnostics.AuthorizationEndpoint,
		TokenEndpoint:         diagnostics.TokenEndpoint,
		RedirectURL:           authCfg.OIDC.RedirectURL,
		Warnings:              securityPublicExposureWarnings(s.cfg.Host, s.auth.Mode),
	})); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	state, err = editor.saveDraft(draft, config.AuthModeOIDC)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, adminAuthModeTransitionResponse{
		Transition: securityOIDCActivationResponse{
			Status:          "configured",
			Message:         "OIDC is now the configured auth mode on disk. Restart the service and complete the first OIDC sign-in with a bootstrap admin email to activate it in the running control plane.",
			RestartRequired: true,
			NextSteps: []string{
				"Restart OpenASE so the running control plane picks up auth.mode=oidc.",
				"Sign in through the browser with a bootstrap admin email.",
				"Verify instance, organization, and project role bindings after the first login, then narrow the bootstrap admin list.",
			},
		},
		Auth: buildSecurityAuthSettingsResponse(
			s.auth,
			state.Auth,
			state.LastValidation,
			editor.resolvedPath(),
			s.cfg.Host,
		),
	})
}

func (s *Server) handleDisableAdminAuth(c echo.Context) error {
	editor := newSecuritySettingsConfigEditor(s.configFilePath, s.homeDir, s.auth)
	state, err := editor.loadStoredState()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	state, err = editor.saveDraft(draftInputFromConfig(state.Auth), config.AuthModeDisabled)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	restartRequired := s.auth.Mode != config.AuthModeDisabled
	nextSteps := []string{
		"Keep the saved OIDC draft so you can retest before the next rollout attempt.",
		"Use disabled mode only for trusted personal or local-only deployments.",
	}
	if restartRequired {
		nextSteps = append([]string{
			"Restart OpenASE so the running control plane returns to disabled mode.",
			"Confirm the local instance admin path is reachable again before exposing the service.",
		}, nextSteps...)
	}

	return c.JSON(http.StatusOK, adminAuthModeTransitionResponse{
		Transition: securityOIDCActivationResponse{
			Status:          "disabled",
			Message:         "Disabled mode is now the configured auth mode on disk. The saved OIDC draft stays available for future rollout attempts.",
			RestartRequired: restartRequired,
			NextSteps:       nextSteps,
		},
		Auth: buildSecurityAuthSettingsResponse(
			s.auth,
			state.Auth,
			state.LastValidation,
			editor.resolvedPath(),
			s.cfg.Host,
		),
	})
}
