package githubrepo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/githubrepo"
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

type Service interface {
	ListNamespaces(ctx context.Context, projectID uuid.UUID) ([]domain.Namespace, error)
	ListRepositories(ctx context.Context, input domain.ListRepositoriesInput) (domain.RepositoryPage, error)
	CreateRepository(ctx context.Context, input domain.CreateRepositoryInput) (domain.Repository, error)
}

type service struct {
	resolver   githubauthservice.TokenResolver
	httpClient *http.Client
	baseURL    string
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
	page := input.Page
	collected := make([]domain.Repository, 0, pageSize)
	scannedPages := 0
	nextCursor := ""

	for {
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
			return domain.RepositoryPage{}, err
		}

		for _, repo := range payload {
			mapped := mapRepository(repo)
			if query != "" && !matchesRepositoryQuery(mapped, query) {
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
		if query == "" || scannedPages >= maxSearchPages {
			return domain.RepositoryPage{Repositories: collected, NextCursor: nextCursor}, nil
		}
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
		return nil, fmt.Errorf("%s %s: %w", method, endpoint, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != expectedStatus {
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

func hasNextLink(headers http.Header) bool {
	for _, part := range strings.Split(headers.Get("Link"), ",") {
		if strings.Contains(part, `rel="next"`) {
			return true
		}
	}
	return false
}
