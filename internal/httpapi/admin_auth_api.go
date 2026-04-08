package httpapi

import (
	"errors"
	"net/http"
	"time"

	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
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
	stored, err := s.readInstanceAccessControlState(c)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, adminAuthResponse{
		Auth: buildSecurityAuthSettingsResponseFromAccessControl(iam.ResolveRuntimeAccessControlState(stored.State), stored.State, stored.StorageLocation, s.cfg.Host),
	})
}

func (s *Server) handlePutAdminOIDCDraft(c echo.Context) error {
	service, err := s.requireInstanceAccessControlService()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	current, err := service.Read(c.Request().Context())
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := draftOIDCConfigFromRequest(raw, current.State)
	stored, err := service.SaveDraft(c.Request().Context(), draft)
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, adminAuthResponse{
		Auth: buildSecurityAuthSettingsResponseFromAccessControl(iam.ResolveRuntimeAccessControlState(stored.State), stored.State, stored.StorageLocation, s.cfg.Host),
	})
}

func (s *Server) handleTestAdminOIDCDraft(c echo.Context) error {
	service, err := s.requireInstanceAccessControlService()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	current, err := service.Read(c.Request().Context())
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	runtimeState := iam.ResolveRuntimeAccessControlState(current.State)

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := draftOIDCConfigFromRequest(raw, current.State)
	active, err := activeOIDCConfigFromDraft(draft)
	if err != nil {
		_ = service.SaveValidation(c.Request().Context(), securityOIDCValidationFailureMetadata(err.Error(), draft.RedirectURL, s.cfg.Host, runtimeConfigAuthMode(runtimeState)))
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
		Message:               "OIDC discovery succeeded. Saving this draft still keeps the active auth mode unchanged until you explicitly enable OIDC.",
		IssuerURL:             diagnostics.IssuerURL,
		AuthorizationEndpoint: diagnostics.AuthorizationEndpoint,
		TokenEndpoint:         diagnostics.TokenEndpoint,
		RedirectURL:           authCfg.OIDC.RedirectURL,
		Warnings:              securityPublicExposureWarnings(s.cfg.Host, runtimeConfigAuthMode(runtimeState)),
	}
	if err := service.SaveValidation(c.Request().Context(), securityOIDCValidationSuccessMetadata(response)); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleEnableAdminOIDC(c echo.Context) error {
	service, err := s.requireInstanceAccessControlService()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	current, err := service.Read(c.Request().Context())
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}
	runtimeState := iam.ResolveRuntimeAccessControlState(current.State)

	var raw rawSecurityOIDCDraftRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}

	draft := draftOIDCConfigFromRequest(raw, current.State)
	active, err := activeOIDCConfigFromDraft(draft)
	if err != nil {
		_ = service.SaveValidation(c.Request().Context(), securityOIDCValidationFailureMetadata(err.Error(), draft.RedirectURL, s.cfg.Host, runtimeConfigAuthMode(runtimeState)))
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
		Message:               "OIDC discovery succeeded. Saving this draft still keeps the active auth mode unchanged until you explicitly enable OIDC.",
		IssuerURL:             diagnostics.IssuerURL,
		AuthorizationEndpoint: diagnostics.AuthorizationEndpoint,
		TokenEndpoint:         diagnostics.TokenEndpoint,
		RedirectURL:           authCfg.OIDC.RedirectURL,
		Warnings:              securityPublicExposureWarnings(s.cfg.Host, runtimeConfigAuthMode(runtimeState)),
	})
	if err := service.SaveValidation(c.Request().Context(), successValidation); err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	stored, err := service.Activate(c.Request().Context(), active, iam.OIDCActivationMetadata{
		ActivatedAt: func() *time.Time {
			now := time.Now().UTC()
			return &now
		}(),
		Source:  "admin_auth_api",
		Message: "OIDC is now active for this instance. Complete the first OIDC sign-in with a bootstrap admin email to continue with managed access control.",
	})
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	return c.JSON(http.StatusOK, adminAuthModeTransitionResponse{
		Transition: securityOIDCActivationResponse{
			Status:          "configured",
			Message:         "OIDC is now active for this instance. Complete the first OIDC sign-in with a bootstrap admin email to continue with managed access control.",
			RestartRequired: false,
			NextSteps: []string{
				"Sign in through the browser with a bootstrap admin email.",
				"Verify instance, organization, and project role bindings after the first login, then narrow the bootstrap admin list.",
			},
		},
		Auth: buildSecurityAuthSettingsResponseFromAccessControl(iam.ResolveRuntimeAccessControlState(stored.State), stored.State, stored.StorageLocation, s.cfg.Host),
	})
}

func (s *Server) handleDisableAdminAuth(c echo.Context) error {
	service, err := s.requireInstanceAccessControlService()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	stored, err := service.Disable(c.Request().Context())
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "ADMIN_AUTH_CONFIG_FAILED", err.Error())
	}

	nextSteps := []string{
		"Confirm the local instance admin path is reachable again before exposing the service.",
		"Keep the saved OIDC draft so you can retest before the next rollout attempt.",
		"Use disabled mode only for trusted personal or local-only deployments.",
	}

	return c.JSON(http.StatusOK, adminAuthModeTransitionResponse{
		Transition: securityOIDCActivationResponse{
			Status:          "disabled",
			Message:         "Disabled mode is now the configured auth mode for this instance. The saved OIDC draft stays available for future rollout attempts.",
			RestartRequired: false,
			NextSteps:       nextSteps,
		},
		Auth: buildSecurityAuthSettingsResponseFromAccessControl(iam.ResolveRuntimeAccessControlState(stored.State), stored.State, stored.StorageLocation, s.cfg.Host),
	})
}

func (s *Server) requireInstanceAccessControlService() (*accesscontrolservice.Service, error) {
	if s.instanceAuthService == nil {
		return nil, errors.New("instance auth service unavailable")
	}
	return s.instanceAuthService, nil
}

func (s *Server) readInstanceAccessControlState(c echo.Context) (accesscontrolservice.ReadResult, error) {
	if s.instanceAuthService != nil {
		return s.instanceAuthService.Read(c.Request().Context())
	}
	fallback, err := accesscontrolservice.New(nil, s.configFilePath+":"+s.homeDir, s.configFilePath, s.homeDir, s.auth)
	if err != nil {
		return accesscontrolservice.ReadResult{}, err
	}
	return fallback.Read(c.Request().Context())
}
