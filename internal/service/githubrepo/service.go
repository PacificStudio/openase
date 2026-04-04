package githubrepo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/githubrepo"
	"github.com/BetterAndBetterII/openase/internal/logging"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
)

const (
	defaultBaseURL    = "https://api.github.com"
	acceptHeaderValue = "application/vnd.github+json"
	apiVersion        = "2022-11-28"
	userAgent         = "openase-github-repo-service"
	pageSize          = 50
	maxSearchPages    = 10
)

var (
	ErrUnavailable       = errors.New("github repo service unavailable")
	ErrInvalidInput      = errors.New("invalid GitHub repository request")
	ErrPermissionDenied  = errors.New("GitHub permissions are insufficient")
	ErrConflict          = errors.New("GitHub repository already exists")
	ErrCredentialMissing = githubauthservice.ErrCredentialNotConfigured
)

var githubRepoServiceComponent = logging.DeclareComponent("github-repo-service")

type Service interface {
	ListNamespaces(ctx context.Context, projectID uuid.UUID) ([]domain.Namespace, error)
	ListRepositories(ctx context.Context, input domain.ListRepositoriesInput) (domain.RepositoryPage, error)
	CreateRepository(ctx context.Context, input domain.CreateRepositoryInput) (domain.Repository, error)
}

type service struct {
	resolver   githubauthservice.TokenResolver
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

type githubUser struct {
	Login string `json:"login"`
}

type githubOrganization struct {
	Login string `json:"login"`
}

type githubRepository struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	FullName      string     `json:"full_name"`
	Private       bool       `json:"private"`
	HTMLURL       string     `json:"html_url"`
	CloneURL      string     `json:"clone_url"`
	DefaultBranch string     `json:"default_branch"`
	Visibility    string     `json:"visibility"`
	Owner         githubUser `json:"owner"`
}

type githubRepositorySearchResponse struct {
	Items []githubRepository `json:"items"`
}

type createUserRepositoryPayload struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private"`
	AutoInit    bool   `json:"auto_init"`
}

type createOrgRepositoryPayload struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private"`
	Visibility  string `json:"visibility"`
	AutoInit    bool   `json:"auto_init"`
}

func NewService(resolver githubauthservice.TokenResolver, httpClient *http.Client) Service {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &service{
		resolver:   resolver,
		httpClient: httpClient,
		baseURL:    defaultBaseURL,
		logger:     logging.WithComponent(nil, githubRepoServiceComponent),
	}
}

func (s *service) ListNamespaces(ctx context.Context, projectID uuid.UUID) ([]domain.Namespace, error) {
	token, err := s.resolveToken(ctx, projectID)
	if err != nil {
		return nil, err
	}

	user, err := s.loadCurrentUser(ctx, token)
	if err != nil {
		return nil, err
	}

	namespaces := []domain.Namespace{{
		Login: user.Login,
		Kind:  domain.NamespaceKindUser,
	}}

	orgsEndpoint, err := s.buildURL("user", "orgs")
	if err != nil {
		return nil, err
	}
	orgsEndpoint += "?per_page=100"

	var organizations []githubOrganization
	headers, err := s.doJSON(ctx, http.MethodGet, orgsEndpoint, token, nil, http.StatusOK, &organizations)
	if err != nil {
		s.logger.Error("list github namespaces failed", "project_id", projectID.String(), "operation", "list_namespaces", "error", err)
		return nil, err
	}

	for _, org := range organizations {
		login := strings.TrimSpace(org.Login)
		if login == "" || strings.EqualFold(login, user.Login) {
			continue
		}
		namespaces = append(namespaces, domain.Namespace{
			Login: login,
			Kind:  domain.NamespaceKindOrganization,
		})
	}

	if hasNextLink(headers) {
		return namespaces, nil
	}
	return namespaces, nil
}

func (s *service) ListRepositories(ctx context.Context, input domain.ListRepositoriesInput) (domain.RepositoryPage, error) {
	token, err := s.resolveToken(ctx, input.ProjectID)
	if err != nil {
		return domain.RepositoryPage{}, err
	}

	query := strings.ToLower(strings.TrimSpace(input.Query))
	if query == "" {
		return s.listBrowsableRepositories(ctx, input.ProjectID, token, input.Page)
	}

	page, err := s.searchRepositories(ctx, input.ProjectID, token, query, input.Page)
	if err != nil {
		return domain.RepositoryPage{}, err
	}
	if len(page.Repositories) > 0 || page.NextCursor != "" {
		return page, nil
	}

	return s.searchBrowsableRepositories(ctx, input.ProjectID, token, query, input.Page)
}

func (s *service) listBrowsableRepositories(
	ctx context.Context,
	projectID uuid.UUID,
	token string,
	page int,
) (domain.RepositoryPage, error) {
	endpoint, err := s.buildURL("user", "repos")
	if err != nil {
		return domain.RepositoryPage{}, err
	}
	endpoint = endpoint + "?" + url.Values{
		"affiliation": []string{"owner,collaborator,organization_member"},
		"page":        []string{strconv.Itoa(page)},
		"per_page":    []string{strconv.Itoa(pageSize)},
		"sort":        []string{"updated"},
	}.Encode()

	var payload []githubRepository
	headers, err := s.doJSON(ctx, http.MethodGet, endpoint, token, nil, http.StatusOK, &payload)
	if err != nil {
		s.logger.Error("list github repositories failed", "project_id", projectID.String(), "operation", "list_repositories", "page", page, "query", "", "error", err)
		return domain.RepositoryPage{}, err
	}

	repositories := make([]domain.Repository, 0, len(payload))
	for _, repo := range payload {
		repositories = append(repositories, mapRepository(repo))
	}
	if !hasNextLink(headers) {
		return domain.RepositoryPage{Repositories: repositories}, nil
	}
	return domain.RepositoryPage{Repositories: repositories, NextCursor: strconv.Itoa(page + 1)}, nil
}

func (s *service) searchRepositories(
	ctx context.Context,
	projectID uuid.UUID,
	token string,
	query string,
	page int,
) (domain.RepositoryPage, error) {
	namespaces, err := s.loadSearchNamespaces(ctx, projectID, token)
	if err != nil {
		return domain.RepositoryPage{}, err
	}

	collected := make([]domain.Repository, 0, pageSize)
	scannedPages := 0
	nextCursor := ""

	for {
		endpoint, err := s.buildURL("search", "repositories")
		if err != nil {
			return domain.RepositoryPage{}, err
		}
		endpoint = endpoint + "?" + url.Values{
			"q":        []string{buildRepositorySearchQuery(query, namespaces)},
			"page":     []string{strconv.Itoa(page)},
			"per_page": []string{strconv.Itoa(pageSize)},
			"sort":     []string{"updated"},
			"order":    []string{"desc"},
		}.Encode()

		var payload githubRepositorySearchResponse
		headers, err := s.doJSON(ctx, http.MethodGet, endpoint, token, nil, http.StatusOK, &payload)
		if err != nil {
			s.logger.Error("search github repositories failed", "project_id", projectID.String(), "operation", "search_repositories", "page", page, "query", query, "error", err)
			return domain.RepositoryPage{}, err
		}

		for _, repo := range payload.Items {
			mapped := mapRepository(repo)
			if !matchesRepositoryQuery(mapped, query) {
				continue
			}
			collected = append(collected, mapped)
			if len(collected) == pageSize {
				if hasNextLink(headers) {
					nextCursor = strconv.Itoa(page + 1)
				}
				return domain.RepositoryPage{Repositories: collected, NextCursor: nextCursor}, nil
			}
		}

		scannedPages++
		if !hasNextLink(headers) {
			return domain.RepositoryPage{Repositories: collected}, nil
		}
		page++
		nextCursor = strconv.Itoa(page)
		if scannedPages >= maxSearchPages {
			return domain.RepositoryPage{Repositories: collected, NextCursor: nextCursor}, nil
		}
	}
}

func (s *service) searchBrowsableRepositories(
	ctx context.Context,
	projectID uuid.UUID,
	token string,
	query string,
	page int,
) (domain.RepositoryPage, error) {
	collected := make([]domain.Repository, 0, pageSize)
	currentPage := page
	scannedPages := 0

	for {
		browsePage, err := s.listBrowsableRepositories(ctx, projectID, token, currentPage)
		if err != nil {
			return domain.RepositoryPage{}, err
		}

		for _, repo := range browsePage.Repositories {
			if !matchesRepositoryQuery(repo, query) {
				continue
			}
			collected = append(collected, repo)
			if len(collected) == pageSize {
				return domain.RepositoryPage{
					Repositories: collected,
					NextCursor:   browsePage.NextCursor,
				}, nil
			}
		}

		scannedPages++
		if browsePage.NextCursor == "" {
			return domain.RepositoryPage{Repositories: collected}, nil
		}
		if scannedPages >= maxSearchPages {
			return domain.RepositoryPage{
				Repositories: collected,
				NextCursor:   browsePage.NextCursor,
			}, nil
		}
		currentPage++
	}
}

func (s *service) CreateRepository(ctx context.Context, input domain.CreateRepositoryInput) (domain.Repository, error) {
	token, err := s.resolveToken(ctx, input.ProjectID)
	if err != nil {
		return domain.Repository{}, err
	}

	user, err := s.loadCurrentUser(ctx, token)
	if err != nil {
		return domain.Repository{}, err
	}

	owner := strings.TrimSpace(input.Owner)
	private := input.Visibility == domain.VisibilityPrivate
	endpointParts := []string{"user", "repos"}
	var payload any = createUserRepositoryPayload{
		Name:        input.Name,
		Description: input.Description,
		Private:     private,
		AutoInit:    input.AutoInit,
	}
	if !strings.EqualFold(owner, user.Login) {
		endpointParts = []string{"orgs", owner, "repos"}
		payload = createOrgRepositoryPayload{
			Name:        input.Name,
			Description: input.Description,
			Private:     private,
			Visibility:  string(input.Visibility),
			AutoInit:    input.AutoInit,
		}
	}

	endpoint, err := s.buildURL(endpointParts...)
	if err != nil {
		return domain.Repository{}, err
	}

	var repo githubRepository
	if _, err := s.doJSON(ctx, http.MethodPost, endpoint, token, payload, http.StatusCreated, &repo); err != nil {
		s.logger.Error("create github repository failed", "project_id", input.ProjectID.String(), "operation", "create_repository", "owner", owner, "name", input.Name, "visibility", input.Visibility, "error", err)
		return domain.Repository{}, err
	}
	return mapRepository(repo), nil
}

func (s *service) resolveToken(ctx context.Context, projectID uuid.UUID) (string, error) {
	if s == nil || s.resolver == nil || s.httpClient == nil {
		return "", ErrUnavailable
	}
	resolved, err := s.resolver.ResolveProjectCredential(ctx, projectID)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(resolved.Token)
	if token == "" {
		return "", ErrCredentialMissing
	}
	return token, nil
}

func (s *service) loadCurrentUser(ctx context.Context, token string) (githubUser, error) {
	endpoint, err := s.buildURL("user")
	if err != nil {
		return githubUser{}, err
	}

	var user githubUser
	if _, err := s.doJSON(ctx, http.MethodGet, endpoint, token, nil, http.StatusOK, &user); err != nil {
		return githubUser{}, err
	}
	if strings.TrimSpace(user.Login) == "" {
		return githubUser{}, fmt.Errorf("GitHub user probe returned an empty login")
	}
	return user, nil
}

func (s *service) loadSearchNamespaces(
	ctx context.Context,
	projectID uuid.UUID,
	token string,
) ([]domain.Namespace, error) {
	user, err := s.loadCurrentUser(ctx, token)
	if err != nil {
		s.logger.Error("load github search namespaces failed", "project_id", projectID.String(), "operation", "search_namespaces", "error", err)
		return nil, err
	}

	namespaces := []domain.Namespace{{
		Login: user.Login,
		Kind:  domain.NamespaceKindUser,
	}}

	orgsEndpoint, err := s.buildURL("user", "orgs")
	if err != nil {
		return nil, err
	}
	orgsEndpoint += "?per_page=100"

	var organizations []githubOrganization
	if _, err := s.doJSON(ctx, http.MethodGet, orgsEndpoint, token, nil, http.StatusOK, &organizations); err != nil {
		s.logger.Error("load github organizations for search failed", "project_id", projectID.String(), "operation", "search_namespaces", "error", err)
		return nil, err
	}

	for _, org := range organizations {
		login := strings.TrimSpace(org.Login)
		if login == "" || strings.EqualFold(login, user.Login) {
			continue
		}
		namespaces = append(namespaces, domain.Namespace{
			Login: login,
			Kind:  domain.NamespaceKindOrganization,
		})
	}

	return namespaces, nil
}

func (s *service) buildURL(parts ...string) (string, error) {
	base, err := url.Parse(strings.TrimRight(s.baseURL, "/"))
	if err != nil {
		return "", fmt.Errorf("%w: invalid GitHub base URL: %s", ErrUnavailable, err)
	}
	pathParts := make([]string, 0, len(parts)+1)
	pathParts = append(pathParts, strings.TrimRight(base.Path, "/"))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		pathParts = append(pathParts, url.PathEscape(trimmed))
	}
	base.Path = strings.Join(pathParts, "/")
	return strings.TrimRight(base.String(), "/"), nil
}

func (s *service) doJSON(
	ctx context.Context,
	method string,
	endpoint string,
	token string,
	body any,
	expectedStatus int,
	target any,
) (http.Header, error) {
	request, err := newJSONRequest(ctx, method, endpoint, token, body)
	if err != nil {
		return nil, err
	}

	response, err := s.httpClient.Do(request)
	if err != nil {
		s.logger.Error("github upstream request failed", "method", method, "endpoint", endpoint, "error", err)
		return nil, fmt.Errorf("%s %s: %w", method, endpoint, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != expectedStatus {
		s.logger.Warn(
			"github upstream returned unexpected status",
			"method", method,
			"endpoint", endpoint,
			"status_code", response.StatusCode,
			"expected_status", expectedStatus,
			"github_request_id", strings.TrimSpace(response.Header.Get("X-GitHub-Request-Id")),
			"ratelimit_remaining", strings.TrimSpace(response.Header.Get("X-RateLimit-Remaining")),
		)
		return nil, mapGitHubError(method, endpoint, response)
	}

	headers := response.Header.Clone()
	if target == nil {
		if _, err := io.Copy(io.Discard, response.Body); err != nil {
			return nil, fmt.Errorf("%s %s: discard response body: %w", method, endpoint, err)
		}
		return headers, nil
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		s.logger.Error("decode github upstream response failed", "method", method, "endpoint", endpoint, "status_code", response.StatusCode, "error", err)
		return nil, fmt.Errorf("%s %s: decode response: %w", method, endpoint, err)
	}
	return headers, nil
}

func newJSONRequest(ctx context.Context, method string, endpoint string, token string, body any) (*http.Request, error) {
	var rawBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal GitHub request body: %w", err)
		}
		rawBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint, rawBody)
	if err != nil {
		return nil, fmt.Errorf("build GitHub request: %w", err)
	}
	request.Header.Set("Accept", acceptHeaderValue)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-GitHub-Api-Version", apiVersion)
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return request, nil
}

func mapGitHubError(method string, endpoint string, response *http.Response) error {
	body, _ := io.ReadAll(response.Body)
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	switch response.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: GitHub credential is invalid", ErrCredentialMissing)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrPermissionDenied, message)
	case http.StatusConflict, http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %s", ErrConflict, message)
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrInvalidInput, message)
	default:
		return fmt.Errorf("%s %s: unexpected GitHub status %d: %s", method, endpoint, response.StatusCode, message)
	}
}

func mapRepository(repo githubRepository) domain.Repository {
	visibility := domain.Visibility(strings.ToLower(strings.TrimSpace(repo.Visibility)))
	if !visibility.IsValid() {
		if repo.Private {
			visibility = domain.VisibilityPrivate
		} else {
			visibility = domain.VisibilityPublic
		}
	}
	return domain.Repository{
		ID:            repo.ID,
		Name:          strings.TrimSpace(repo.Name),
		FullName:      strings.TrimSpace(repo.FullName),
		Owner:         strings.TrimSpace(repo.Owner.Login),
		DefaultBranch: strings.TrimSpace(repo.DefaultBranch),
		Visibility:    visibility,
		Private:       repo.Private,
		HTMLURL:       strings.TrimSpace(repo.HTMLURL),
		CloneURL:      strings.TrimSpace(repo.CloneURL),
	}
}

func matchesRepositoryQuery(repo domain.Repository, query string) bool {
	return strings.Contains(strings.ToLower(repo.Name), query) ||
		strings.Contains(strings.ToLower(repo.FullName), query) ||
		strings.Contains(strings.ToLower(repo.Owner), query)
}

func buildRepositorySearchQuery(query string, namespaces []domain.Namespace) string {
	parts := []string{query, "fork:true"}
	qualifiers := make([]string, 0, len(namespaces))
	for _, namespace := range namespaces {
		login := strings.TrimSpace(namespace.Login)
		if login == "" {
			continue
		}
		switch namespace.Kind {
		case domain.NamespaceKindUser:
			qualifiers = append(qualifiers, "user:"+login)
		case domain.NamespaceKindOrganization:
			qualifiers = append(qualifiers, "org:"+login)
		}
	}

	if len(qualifiers) == 1 {
		parts = append(parts, qualifiers[0])
	} else if len(qualifiers) > 1 {
		parts = append(parts, "("+strings.Join(qualifiers, " OR ")+")")
	}

	return strings.Join(parts, " ")
}

func hasNextLink(headers http.Header) bool {
	for _, part := range strings.Split(headers.Get("Link"), ",") {
		if strings.Contains(part, `rel="next"`) {
			return true
		}
	}
	return false
}
