package httpapi

import (
	"errors"
	"net/http"

	domain "github.com/BetterAndBetterII/openase/internal/domain/githubrepo"
	githubreposervice "github.com/BetterAndBetterII/openase/internal/service/githubrepo"
	"github.com/labstack/echo/v4"
)

type githubRepositoryNamespaceResponse struct {
	Login string `json:"login"`
	Kind  string `json:"kind"`
}

type githubRepositoryResponse struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         string `json:"owner"`
	DefaultBranch string `json:"default_branch"`
	Visibility    string `json:"visibility"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
}

type githubRepositoryNamespacesResponse struct {
	Namespaces []githubRepositoryNamespaceResponse `json:"namespaces"`
}

type githubRepositoryListResponse struct {
	Repositories []githubRepositoryResponse `json:"repositories"`
	NextCursor   string                     `json:"next_cursor,omitempty"`
}

type githubRepositoryCreateResponse struct {
	Repository githubRepositoryResponse `json:"repository"`
}

func (s *Server) registerGitHubRepoRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/github/namespaces", s.handleListGitHubNamespaces)
	api.GET("/projects/:projectId/github/repos", s.handleListGitHubRepositories)
	api.POST("/projects/:projectId/github/repos", s.handleCreateGitHubRepository)
}

func (s *Server) handleListGitHubNamespaces(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubRepoService == nil {
		return writeGitHubRepoError(c, githubreposervice.ErrUnavailable)
	}

	namespaces, err := s.githubRepoService.ListNamespaces(c.Request().Context(), projectID)
	if err != nil {
		s.logger.Error("list github namespaces api failed", "project_id", projectID.String(), "error", err)
		return writeGitHubRepoError(c, err)
	}
	return c.JSON(http.StatusOK, githubRepositoryNamespacesResponse{
		Namespaces: mapGitHubNamespaceResponses(namespaces),
	})
}

func (s *Server) handleListGitHubRepositories(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubRepoService == nil {
		return writeGitHubRepoError(c, githubreposervice.ErrUnavailable)
	}

	input, err := domain.ParseListRepositories(projectID, c.QueryParam("query"), c.QueryParam("cursor"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	page, err := s.githubRepoService.ListRepositories(c.Request().Context(), input)
	if err != nil {
		s.logger.Error("list github repositories api failed", "project_id", projectID.String(), "query", input.Query, "cursor", c.QueryParam("cursor"), "error", err)
		return writeGitHubRepoError(c, err)
	}
	return c.JSON(http.StatusOK, githubRepositoryListResponse{
		Repositories: mapGitHubRepositoryResponses(page.Repositories),
		NextCursor:   page.NextCursor,
	})
}

func (s *Server) handleCreateGitHubRepository(c echo.Context) error {
	projectID, err := s.requireProjectSecurityContext(c)
	if err != nil {
		return err
	}
	if s.githubRepoService == nil {
		return writeGitHubRepoError(c, githubreposervice.ErrUnavailable)
	}

	var raw domain.CreateRepositoryRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := domain.ParseCreateRepository(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	repository, err := s.githubRepoService.CreateRepository(c.Request().Context(), input)
	if err != nil {
		s.logger.Error("create github repository api failed", "project_id", projectID.String(), "owner", input.Owner, "name", input.Name, "visibility", input.Visibility, "error", err)
		return writeGitHubRepoError(c, err)
	}
	return c.JSON(http.StatusCreated, githubRepositoryCreateResponse{
		Repository: mapGitHubRepositoryResponse(repository),
	})
}

func writeGitHubRepoError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, githubreposervice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, githubreposervice.ErrInvalidInput):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, githubreposervice.ErrCredentialMissing):
		return writeAPIError(c, http.StatusPreconditionFailed, "GITHUB_CREDENTIAL_REQUIRED", err.Error())
	case errors.Is(err, githubreposervice.ErrPermissionDenied):
		return writeAPIError(c, http.StatusForbidden, "GITHUB_PERMISSION_DENIED", err.Error())
	case errors.Is(err, githubreposervice.ErrConflict):
		return writeAPIError(c, http.StatusConflict, "RESOURCE_CONFLICT", err.Error())
	default:
		return writeAPIError(c, http.StatusBadGateway, "GITHUB_UPSTREAM_ERROR", err.Error())
	}
}

func mapGitHubNamespaceResponses(items []domain.Namespace) []githubRepositoryNamespaceResponse {
	result := make([]githubRepositoryNamespaceResponse, 0, len(items))
	for _, item := range items {
		result = append(result, githubRepositoryNamespaceResponse{
			Login: item.Login,
			Kind:  string(item.Kind),
		})
	}
	return result
}

func mapGitHubRepositoryResponses(items []domain.Repository) []githubRepositoryResponse {
	result := make([]githubRepositoryResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapGitHubRepositoryResponse(item))
	}
	return result
}

func mapGitHubRepositoryResponse(item domain.Repository) githubRepositoryResponse {
	defaultBranch := item.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	return githubRepositoryResponse{
		ID:            item.ID,
		Name:          item.Name,
		FullName:      item.FullName,
		Owner:         item.Owner,
		DefaultBranch: defaultBranch,
		Visibility:    string(item.Visibility),
		Private:       item.Private,
		HTMLURL:       item.HTMLURL,
		CloneURL:      item.CloneURL,
	}
}
