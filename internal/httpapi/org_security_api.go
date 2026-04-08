package httpapi

import (
	"net/http"
	"strings"

	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerOrgSecurityRoutes(api *echo.Group) {
	api.GET("/orgs/:orgId/security/github-credential", s.handleGetOrgGitHubCredential)
	api.PUT("/orgs/:orgId/security/github-credential", s.handlePutOrgGitHubCredential)
	api.POST("/orgs/:orgId/security/github-credential/import-gh-cli", s.handleImportOrgGitHubCredential)
	api.POST("/orgs/:orgId/security/github-credential/retest", s.handleRetestOrgGitHubCredential)
	api.DELETE("/orgs/:orgId/security/github-credential", s.handleDeleteOrgGitHubCredential)
}

type orgGitHubCredentialResponse struct {
	Credential securityGitHubCredentialSlotResponse `json:"credential"`
}

func (s *Server) handleGetOrgGitHubCredential(c echo.Context) error {
	orgID, err := s.requireOrgSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}
	orgSvc, ok := s.githubAuthService.(githubauthservice.OrgSecurityManager)
	if !ok {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "org security manager not available")
	}
	security, err := orgSvc.ReadOrgSecurity(c.Request().Context(), orgID)
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return c.JSON(http.StatusOK, orgGitHubCredentialResponse{
		Credential: mapGitHubCredentialSlot(security.Organization),
	})
}

func (s *Server) handlePutOrgGitHubCredential(c echo.Context) error {
	orgID, err := s.requireOrgSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}
	orgSvc, ok := s.githubAuthService.(githubauthservice.OrgSecurityManager)
	if !ok {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "org security manager not available")
	}

	var raw struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}
	token := strings.TrimSpace(raw.Token)
	if token == "" {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "token must not be empty")
	}

	security, err := orgSvc.SaveOrgManualCredential(c.Request().Context(), githubauthservice.OrgSaveCredentialInput{
		OrganizationID: orgID,
		Token:          token,
	})
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return c.JSON(http.StatusOK, orgGitHubCredentialResponse{
		Credential: mapGitHubCredentialSlot(security.Organization),
	})
}

func (s *Server) handleImportOrgGitHubCredential(c echo.Context) error {
	orgID, err := s.requireOrgSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}
	orgSvc, ok := s.githubAuthService.(githubauthservice.OrgSecurityManager)
	if !ok {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "org security manager not available")
	}

	security, err := orgSvc.ImportOrgGHCLICredential(c.Request().Context(), githubauthservice.OrgInput{
		OrganizationID: orgID,
	})
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return c.JSON(http.StatusOK, orgGitHubCredentialResponse{
		Credential: mapGitHubCredentialSlot(security.Organization),
	})
}

func (s *Server) handleRetestOrgGitHubCredential(c echo.Context) error {
	orgID, err := s.requireOrgSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}
	orgSvc, ok := s.githubAuthService.(githubauthservice.OrgSecurityManager)
	if !ok {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "org security manager not available")
	}

	security, err := orgSvc.RetestOrgCredential(c.Request().Context(), githubauthservice.OrgInput{
		OrganizationID: orgID,
	})
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return c.JSON(http.StatusOK, orgGitHubCredentialResponse{
		Credential: mapGitHubCredentialSlot(security.Organization),
	})
}

func (s *Server) handleDeleteOrgGitHubCredential(c echo.Context) error {
	orgID, err := s.requireOrgSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubAuthService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", githubauthservice.ErrUnavailable.Error())
	}
	orgSvc, ok := s.githubAuthService.(githubauthservice.OrgSecurityManager)
	if !ok {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "org security manager not available")
	}

	security, err := orgSvc.DeleteOrgCredential(c.Request().Context(), githubauthservice.OrgInput{
		OrganizationID: orgID,
	})
	if err != nil {
		return writeGitHubAuthError(c, err)
	}
	return c.JSON(http.StatusOK, orgGitHubCredentialResponse{
		Credential: mapGitHubCredentialSlot(security.Organization),
	})
}

func (s *Server) requireOrgSecurityContext(c echo.Context) (uuid.UUID, error) {
	if s.catalog.Empty() {
		return uuid.UUID{}, writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return uuid.UUID{}, writeAPIError(c, http.StatusBadRequest, "INVALID_ORG_ID", err.Error())
	}
	if _, err := s.catalog.GetOrganization(c.Request().Context(), orgID); err != nil {
		return uuid.UUID{}, writeCatalogError(c, err)
	}
	return orgID, nil
}
