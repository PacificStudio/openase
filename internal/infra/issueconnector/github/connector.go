package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	"github.com/BetterAndBetterII/openase/internal/logging"
)

const (
	defaultBaseURL     = "https://api.github.com"
	acceptHeaderValue  = "application/vnd.github+json"
	apiVersion         = "2022-11-28"
	userAgent          = "openase-github-issue-connector"
	webhookEventHeader = "X-GitHub-Event"
)

var terminalStatusFallbacks = map[string]struct{}{
	"canceled":  {},
	"cancelled": {},
	"closed":    {},
	"complete":  {},
	"completed": {},
	"done":      {},
	"resolved":  {},
}

var githubIssueConnectorComponent = logging.DeclareComponent("github-issue-connector")

// Connector implements the F47 issueconnector contract for GitHub Issues.
//
// Pull and webhook parsing treat GitHub issue fields as the upstream source of truth.
// SyncBack intentionally limits local-to-GitHub writes to status, comments, and labels
// on already linked issues; it does not create new GitHub issues for local-only tickets.
type Connector struct {
	client *http.Client
	logger *slog.Logger
}

func New(client *http.Client) *Connector {
	if client == nil {
		client = http.DefaultClient
	}

	return &Connector{
		client: client,
		logger: logging.WithComponent(nil, githubIssueConnectorComponent),
	}
}

func (c *Connector) ID() string {
	return string(domain.TypeGitHub)
}

func (c *Connector) Name() string {
	return "GitHub Issues"
}

func (c *Connector) PullIssues(ctx context.Context, cfg domain.Config, since time.Time) ([]domain.ExternalIssue, error) {
	repoRef, err := parseRepositoryRef(cfg.ProjectRef)
	if err != nil {
		return nil, err
	}

	nextURL, err := buildIssuesURL(cfg.BaseURL, repoRef, since)
	if err != nil {
		return nil, err
	}

	issues := make([]domain.ExternalIssue, 0)
	for nextURL != "" {
		var payload []githubIssue
		headers, err := c.doJSON(ctx, http.MethodGet, nextURL, cfg.AuthToken, nil, http.StatusOK, &payload)
		if err != nil {
			return nil, err
		}

		for _, issue := range payload {
			if issue.PullRequest != nil {
				continue
			}
			issues = append(issues, repoRef.toExternalIssue(issue))
		}

		nextURL = parseNextLink(headers.Get("Link"))
	}

	return issues, nil
}

func (c *Connector) ParseWebhook(_ context.Context, headers http.Header, body []byte) (*domain.WebhookEvent, error) {
	eventType := headerValue(headers, webhookEventHeader)
	switch eventType {
	case "issues":
		return parseIssuesWebhook(body)
	case "issue_comment":
		return parseIssueCommentWebhook(body)
	default:
		c.logger.Warn("github issue webhook event unsupported", "event_type", eventType)
		return nil, fmt.Errorf("unsupported GitHub webhook event %q", eventType)
	}
}

func (c *Connector) SyncBack(ctx context.Context, cfg domain.Config, update domain.SyncBackUpdate) error {
	repoRef, err := parseRepositoryRef(cfg.ProjectRef)
	if err != nil {
		return err
	}

	externalRef, err := parseIssueExternalRef(update.ExternalID)
	if err != nil {
		return err
	}
	if !externalRef.Repository.matches(repoRef) {
		return fmt.Errorf("external_id %q does not match project_ref %q", update.ExternalID, repoRef.String())
	}

	switch update.Action {
	case domain.SyncBackActionUpdateStatus:
		state, err := mapSyncBackStatus(cfg.StatusMapping, update.Status)
		if err != nil {
			return err
		}
		endpoint, err := buildRepositoryEndpoint(cfg.BaseURL, repoRef, "issues", strconv.Itoa(externalRef.Number))
		if err != nil {
			return err
		}
		_, err = c.doJSON(
			ctx,
			http.MethodPatch,
			endpoint,
			cfg.AuthToken,
			map[string]string{"state": state},
			http.StatusOK,
			nil,
		)
		return err
	case domain.SyncBackActionAddComment:
		comment := strings.TrimSpace(update.Comment)
		if comment == "" {
			return fmt.Errorf("sync back comment must not be empty")
		}
		endpoint, err := buildRepositoryEndpoint(cfg.BaseURL, repoRef, "issues", strconv.Itoa(externalRef.Number), "comments")
		if err != nil {
			return err
		}
		_, err = c.doJSON(
			ctx,
			http.MethodPost,
			endpoint,
			cfg.AuthToken,
			map[string]string{"body": comment},
			http.StatusCreated,
			nil,
		)
		return err
	case domain.SyncBackActionAddLabel:
		label := strings.TrimSpace(update.Label)
		if label == "" {
			return fmt.Errorf("sync back label must not be empty")
		}
		endpoint, err := buildRepositoryEndpoint(cfg.BaseURL, repoRef, "issues", strconv.Itoa(externalRef.Number), "labels")
		if err != nil {
			return err
		}
		_, err = c.doJSON(
			ctx,
			http.MethodPost,
			endpoint,
			cfg.AuthToken,
			map[string][]string{"labels": []string{label}},
			http.StatusOK,
			nil,
		)
		return err
	default:
		return fmt.Errorf("unsupported sync back action %q", update.Action)
	}
}

func (c *Connector) HealthCheck(ctx context.Context, cfg domain.Config) error {
	repoRef, err := parseRepositoryRef(cfg.ProjectRef)
	if err != nil {
		return err
	}
	endpoint, err := buildRepositoryEndpoint(cfg.BaseURL, repoRef)
	if err != nil {
		return err
	}

	_, err = c.doJSON(
		ctx,
		http.MethodGet,
		endpoint,
		cfg.AuthToken,
		nil,
		http.StatusOK,
		nil,
	)
	return err
}

type repositoryRef struct {
	Owner string
	Name  string
}

func (r repositoryRef) String() string {
	return r.Owner + "/" + r.Name
}

func (r repositoryRef) matches(other repositoryRef) bool {
	return strings.EqualFold(r.Owner, other.Owner) && strings.EqualFold(r.Name, other.Name)
}

func (r repositoryRef) toExternalIssue(issue githubIssue) domain.ExternalIssue {
	return domain.ExternalIssue{
		ExternalID:  fmt.Sprintf("%s#%d", r.String(), issue.Number),
		ExternalURL: strings.TrimSpace(issue.HTMLURL),
		Title:       strings.TrimSpace(issue.Title),
		Description: strings.TrimSpace(issue.Body),
		Status:      normalizeStatus(issue.State),
		Labels:      collectLabels(issue.Labels),
		Author:      loginOf(issue.User),
		Assignees:   collectAssignees(issue.Assignees),
		CreatedAt:   issue.CreatedAt.UTC(),
		UpdatedAt:   issue.UpdatedAt.UTC(),
		Metadata: map[string]any{
			"github_number": issue.Number,
			"project_ref":   r.String(),
		},
	}
}

type issueExternalRef struct {
	Repository repositoryRef
	Number     int
}

type githubIssue struct {
	Number      int              `json:"number"`
	HTMLURL     string           `json:"html_url"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	State       string           `json:"state"`
	User        *githubUser      `json:"user"`
	Assignees   []githubUser     `json:"assignees"`
	Labels      []githubLabel    `json:"labels"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	PullRequest *json.RawMessage `json:"pull_request"`
}

type githubUser struct {
	Login string `json:"login"`
}

type githubLabel struct {
	Name string `json:"name"`
}

type githubRepository struct {
	FullName string `json:"full_name"`
}

type githubComment struct {
	ID        int64       `json:"id"`
	Body      string      `json:"body"`
	User      *githubUser `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type githubIssueWebhookPayload struct {
	Action     string           `json:"action"`
	Issue      githubIssue      `json:"issue"`
	Repository githubRepository `json:"repository"`
}

type githubIssueCommentWebhookPayload struct {
	Action     string           `json:"action"`
	Issue      githubIssue      `json:"issue"`
	Repository githubRepository `json:"repository"`
	Comment    *githubComment   `json:"comment"`
}

func parseIssuesWebhook(body []byte) (*domain.WebhookEvent, error) {
	var payload githubIssueWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid GitHub issues webhook payload: %w", err)
	}

	repoRef, err := parseRepositoryRef(payload.Repository.FullName)
	if err != nil {
		return nil, err
	}

	action, err := mapIssueWebhookAction(payload.Action)
	if err != nil {
		return nil, err
	}

	return &domain.WebhookEvent{
		Action: action,
		Issue:  repoRef.toExternalIssue(payload.Issue),
	}, nil
}

func parseIssueCommentWebhook(body []byte) (*domain.WebhookEvent, error) {
	var payload githubIssueCommentWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid GitHub issue_comment webhook payload: %w", err)
	}

	repoRef, err := parseRepositoryRef(payload.Repository.FullName)
	if err != nil {
		return nil, err
	}

	switch normalizeStatus(payload.Action) {
	case "created", "edited":
		if payload.Comment == nil {
			return nil, fmt.Errorf("GitHub issue_comment webhook comment is required")
		}
		return &domain.WebhookEvent{
			Action: "commented",
			Issue:  repoRef.toExternalIssue(payload.Issue),
			Comment: &domain.ExternalComment{
				ExternalID: strconv.FormatInt(payload.Comment.ID, 10),
				Author:     loginOf(payload.Comment.User),
				Body:       strings.TrimSpace(payload.Comment.Body),
				CreatedAt:  payload.Comment.CreatedAt.UTC(),
				UpdatedAt:  payload.Comment.UpdatedAt.UTC(),
			},
		}, nil
	case "deleted":
		return &domain.WebhookEvent{
			Action: "updated",
			Issue:  repoRef.toExternalIssue(payload.Issue),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported GitHub issue_comment action %q", strings.TrimSpace(payload.Action))
	}
}

func mapIssueWebhookAction(raw string) (string, error) {
	switch normalizeStatus(raw) {
	case "opened":
		return "created", nil
	case "closed":
		return "closed", nil
	case "reopened":
		return "reopened", nil
	case "edited", "labeled", "unlabeled", "assigned", "unassigned", "milestoned", "demilestoned", "transferred":
		return "updated", nil
	default:
		return "", fmt.Errorf("unsupported GitHub issues action %q", strings.TrimSpace(raw))
	}
}

func parseRepositoryRef(raw string) (repositoryRef, error) {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	if len(parts) != 2 {
		return repositoryRef{}, fmt.Errorf("project_ref must be owner/repo, got %q", strings.TrimSpace(raw))
	}

	owner := strings.ToLower(strings.TrimSpace(parts[0]))
	name := strings.ToLower(strings.TrimSpace(parts[1]))
	if owner == "" || name == "" {
		return repositoryRef{}, fmt.Errorf("project_ref must be owner/repo, got %q", strings.TrimSpace(raw))
	}

	return repositoryRef{Owner: owner, Name: name}, nil
}

func parseIssueExternalRef(raw string) (issueExternalRef, error) {
	trimmed := strings.TrimSpace(raw)
	hashIndex := strings.LastIndex(trimmed, "#")
	if hashIndex <= 0 || hashIndex == len(trimmed)-1 {
		return issueExternalRef{}, fmt.Errorf("external_id must be owner/repo#number, got %q", trimmed)
	}

	repoRef, err := parseRepositoryRef(trimmed[:hashIndex])
	if err != nil {
		return issueExternalRef{}, err
	}

	number, err := strconv.Atoi(strings.TrimSpace(trimmed[hashIndex+1:]))
	if err != nil || number <= 0 {
		return issueExternalRef{}, fmt.Errorf("external_id must end with a positive issue number, got %q", trimmed)
	}

	return issueExternalRef{
		Repository: repoRef,
		Number:     number,
	}, nil
}

func mapSyncBackStatus(statusMapping map[string]string, localStatus string) (string, error) {
	normalized := normalizeStatus(localStatus)
	if normalized == "" {
		return "", fmt.Errorf("sync back status must not be empty")
	}
	if normalized == "open" || normalized == "closed" {
		return normalized, nil
	}

	reverseMatches := make([]string, 0, 1)
	for externalStatus, mappedStatus := range statusMapping {
		if normalizeStatus(mappedStatus) != normalized {
			continue
		}
		reverseMatches = append(reverseMatches, normalizeStatus(externalStatus))
	}

	if len(reverseMatches) > 1 {
		return "", fmt.Errorf("status %q maps to multiple GitHub states", strings.TrimSpace(localStatus))
	}
	if len(reverseMatches) == 1 {
		switch reverseMatches[0] {
		case "open", "closed":
			return reverseMatches[0], nil
		default:
			return "", fmt.Errorf("GitHub status_mapping only supports open/closed external states, got %q", reverseMatches[0])
		}
	}

	if _, closed := terminalStatusFallbacks[normalized]; closed {
		return "closed", nil
	}

	return "open", nil
}

func buildIssuesURL(rawBaseURL string, repoRef repositoryRef, since time.Time) (string, error) {
	base, err := normalizeBaseURL(rawBaseURL)
	if err != nil {
		return "", err
	}

	repoEndpoint, err := buildRepositoryEndpoint(base, repoRef, "issues")
	if err != nil {
		return "", err
	}
	endpoint, err := url.Parse(repoEndpoint)
	if err != nil {
		return "", fmt.Errorf("build issues endpoint: %w", err)
	}

	query := endpoint.Query()
	query.Set("state", "all")
	query.Set("sort", "updated")
	query.Set("direction", "asc")
	query.Set("per_page", "100")
	if !since.IsZero() {
		query.Set("since", since.UTC().Format(time.RFC3339))
	}
	endpoint.RawQuery = query.Encode()

	return endpoint.String(), nil
}

func buildRepositoryEndpoint(rawBaseURL string, repoRef repositoryRef, segments ...string) (string, error) {
	baseURL, err := normalizeBaseURL(rawBaseURL)
	if err != nil {
		return "", err
	}
	pathParts := make([]string, 0, 4+len(segments))
	pathParts = append(pathParts,
		strings.TrimRight(baseURL, "/"),
		"repos",
		url.PathEscape(repoRef.Owner),
		url.PathEscape(repoRef.Name),
	)
	for _, segment := range segments {
		pathParts = append(pathParts, url.PathEscape(strings.TrimSpace(segment)))
	}

	return strings.Join(pathParts, "/"), nil
}

func normalizeBaseURL(raw string) (string, error) {
	baseURL := strings.TrimSpace(raw)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("base_url must be a valid URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("base_url must be absolute, got %q", baseURL)
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeStatus(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func collectLabels(labels []githubLabel) []string {
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		name := strings.TrimSpace(label.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
	}

	return result
}

func collectAssignees(assignees []githubUser) []string {
	result := make([]string, 0, len(assignees))
	for _, assignee := range assignees {
		login := strings.TrimSpace(assignee.Login)
		if login == "" {
			continue
		}
		result = append(result, login)
	}

	return result
}

func loginOf(user *githubUser) string {
	if user == nil {
		return ""
	}

	return strings.TrimSpace(user.Login)
}

func parseNextLink(raw string) string {
	for _, part := range strings.Split(raw, ",") {
		section := strings.TrimSpace(part)
		if !strings.Contains(section, `rel="next"`) {
			continue
		}

		start := strings.Index(section, "<")
		end := strings.Index(section, ">")
		if start >= 0 && end > start {
			return section[start+1 : end]
		}
	}

	return ""
}

func headerValue(headers http.Header, key string) string {
	if value := strings.TrimSpace(headers.Get(key)); value != "" {
		return value
	}

	for headerKey, values := range headers {
		if !strings.EqualFold(headerKey, key) || len(values) == 0 {
			continue
		}
		return strings.TrimSpace(values[0])
	}

	return ""
}

func (c *Connector) doJSON(
	ctx context.Context,
	method string,
	endpoint string,
	authToken string,
	payload any,
	expectedStatus int,
	target any,
) (headers http.Header, err error) {
	request, err := newJSONRequest(ctx, method, endpoint, authToken, payload)
	if err != nil {
		return nil, err
	}

	response, err := c.client.Do(request)
	if err != nil {
		c.logger.Error("github issue upstream request failed", "method", method, "endpoint", endpoint, "error", err)
		return nil, fmt.Errorf("%s %s: %w", method, endpoint, err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("%s %s: close response body: %w", method, endpoint, closeErr)
		}
	}()

	if response.StatusCode != expectedStatus {
		c.logger.Warn(
			"github issue upstream returned unexpected status",
			"method", method,
			"endpoint", endpoint,
			"status_code", response.StatusCode,
			"expected_status", expectedStatus,
			"github_request_id", strings.TrimSpace(response.Header.Get("X-GitHub-Request-Id")),
		)
		body, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return nil, fmt.Errorf("%s %s: unexpected status %d and failed to read response body: %w", method, endpoint, response.StatusCode, readErr)
		}
		return nil, fmt.Errorf("%s %s: unexpected status %d: %s", method, endpoint, response.StatusCode, strings.TrimSpace(string(body)))
	}

	headers = response.Header.Clone()
	if target == nil {
		if _, err := io.Copy(io.Discard, response.Body); err != nil {
			return nil, fmt.Errorf("%s %s: discard response body: %w", method, endpoint, err)
		}
		return headers, nil
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		c.logger.Error("decode github issue upstream response failed", "method", method, "endpoint", endpoint, "status_code", response.StatusCode, "error", err)
		return nil, fmt.Errorf("%s %s: decode response: %w", method, endpoint, err)
	}

	return headers, nil
}

func newJSONRequest(ctx context.Context, method string, endpoint string, authToken string, payload any) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		rawBody, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request payload: %w", err)
		}
		body = bytes.NewReader(rawBody)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	request.Header.Set("Accept", acceptHeaderValue)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-GitHub-Api-Version", apiVersion)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token := strings.TrimSpace(authToken); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	return request, nil
}
