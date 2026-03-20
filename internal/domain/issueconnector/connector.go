package issueconnector

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultPollInterval = 5 * time.Minute

var connectorTypePattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

type Type string

const (
	TypeGitHub         Type = "github"
	TypeGitLab         Type = "gitlab"
	TypeJira           Type = "jira"
	TypeInboundWebhook Type = "inbound-webhook"
	TypeCustom         Type = "custom"
)

type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
	StatusError  Status = "error"
)

type SyncDirection string

const (
	SyncDirectionPullOnly      SyncDirection = "pull_only"
	SyncDirectionPushOnly      SyncDirection = "push_only"
	SyncDirectionBidirectional SyncDirection = "bidirectional"
)

type SyncBackAction string

const (
	SyncBackActionUpdateStatus SyncBackAction = "update_status"
	SyncBackActionAddComment   SyncBackAction = "add_comment"
	SyncBackActionAddLabel     SyncBackAction = "add_label"
)

type ExternalComment struct {
	ExternalID string
	Author     string
	Body       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Metadata   map[string]any
}

type ExternalIssue struct {
	ExternalID  string
	ExternalURL string
	Title       string
	Description string
	Status      string
	Priority    string
	Labels      []string
	Author      string
	Assignees   []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Comments    []ExternalComment
	Metadata    map[string]any
}

type WebhookEvent struct {
	Action  string
	Issue   ExternalIssue
	Comment *ExternalComment
}

type SyncBackUpdate struct {
	ExternalID string
	Action     SyncBackAction
	Status     string
	Comment    string
	Label      string
}

type Filters struct {
	Labels        []string
	ExcludeLabels []string
	States        []string
	Authors       []string
}

type FiltersInput struct {
	Labels        []string `json:"labels"`
	ExcludeLabels []string `json:"exclude_labels"`
	States        []string `json:"states"`
	Authors       []string `json:"authors"`
}

type Config struct {
	Type          Type
	BaseURL       string
	AuthToken     string
	ProjectRef    string
	PollInterval  time.Duration
	SyncDirection SyncDirection
	Filters       Filters
	StatusMapping map[string]string
	WebhookSecret string
	AutoWorkflow  string
}

type ConfigInput struct {
	Type          string            `json:"type"`
	BaseURL       string            `json:"base_url"`
	AuthToken     string            `json:"auth_token"`
	ProjectRef    string            `json:"project_ref"`
	PollInterval  string            `json:"poll_interval"`
	SyncDirection string            `json:"sync_direction"`
	Filters       FiltersInput      `json:"filters"`
	StatusMapping map[string]string `json:"status_mapping"`
	WebhookSecret string            `json:"webhook_secret"`
	AutoWorkflow  string            `json:"auto_workflow"`
}

type SyncStats struct {
	TotalSynced int
	Synced24h   int
	FailedCount int
}

type IssueConnector struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	Type       Type
	Name       string
	Config     Config
	Status     Status
	LastSyncAt *time.Time
	LastError  string
	Stats      SyncStats
}

type Input struct {
	Type   string      `json:"type"`
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Config ConfigInput `json:"config"`
}

type CreateIssueConnector struct {
	ProjectID uuid.UUID
	Type      Type
	Name      string
	Status    Status
	Config    Config
}

type UpdateIssueConnector struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Type      Type
	Name      string
	Status    Status
	Config    Config
}

type Connector interface {
	ID() string
	Name() string
	PullIssues(ctx context.Context, cfg Config, since time.Time) ([]ExternalIssue, error)
	ParseWebhook(ctx context.Context, headers http.Header, body []byte) (*WebhookEvent, error)
	SyncBack(ctx context.Context, cfg Config, update SyncBackUpdate) error
	HealthCheck(ctx context.Context, cfg Config) error
}

func ParseCreateIssueConnector(projectID uuid.UUID, raw Input) (CreateIssueConnector, error) {
	connectorType, err := ParseType(raw.Type)
	if err != nil {
		return CreateIssueConnector{}, err
	}

	name, err := parseName(raw.Name)
	if err != nil {
		return CreateIssueConnector{}, err
	}

	status, err := ParseStatus(raw.Status)
	if err != nil {
		return CreateIssueConnector{}, err
	}

	config, err := ParseConfig(raw.Config)
	if err != nil {
		return CreateIssueConnector{}, err
	}
	if config.Type != connectorType {
		return CreateIssueConnector{}, fmt.Errorf("config.type must match type")
	}

	return CreateIssueConnector{
		ProjectID: projectID,
		Type:      connectorType,
		Name:      name,
		Status:    status,
		Config:    config,
	}, nil
}

func ParseUpdateIssueConnector(id uuid.UUID, projectID uuid.UUID, raw Input) (UpdateIssueConnector, error) {
	createInput, err := ParseCreateIssueConnector(projectID, raw)
	if err != nil {
		return UpdateIssueConnector{}, err
	}

	return UpdateIssueConnector{
		ID:        id,
		ProjectID: createInput.ProjectID,
		Type:      createInput.Type,
		Name:      createInput.Name,
		Status:    createInput.Status,
		Config:    createInput.Config,
	}, nil
}

func ParseConfig(raw ConfigInput) (Config, error) {
	connectorType, err := ParseType(raw.Type)
	if err != nil {
		return Config{}, err
	}

	syncDirection, err := ParseSyncDirection(raw.SyncDirection)
	if err != nil {
		return Config{}, err
	}

	pollInterval, err := parsePollInterval(raw.PollInterval)
	if err != nil {
		return Config{}, err
	}

	filters := ParseFilters(raw.Filters)
	statusMapping, err := parseStatusMapping(raw.StatusMapping)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Type:          connectorType,
		BaseURL:       strings.TrimSpace(raw.BaseURL),
		AuthToken:     strings.TrimSpace(raw.AuthToken),
		ProjectRef:    strings.TrimSpace(raw.ProjectRef),
		PollInterval:  pollInterval,
		SyncDirection: syncDirection,
		Filters:       filters,
		StatusMapping: statusMapping,
		WebhookSecret: strings.TrimSpace(raw.WebhookSecret),
		AutoWorkflow:  strings.TrimSpace(raw.AutoWorkflow),
	}, nil
}

func ParseFilters(raw FiltersInput) Filters {
	return Filters{
		Labels:        normalizeStringList(raw.Labels),
		ExcludeLabels: normalizeStringList(raw.ExcludeLabels),
		States:        normalizeStringList(raw.States),
		Authors:       normalizeStringList(raw.Authors),
	}
}

func ParseType(raw string) (Type, error) {
	connectorType := Type(strings.ToLower(strings.TrimSpace(raw)))
	if connectorType == "" {
		return "", fmt.Errorf("type must not be empty")
	}
	if !connectorTypePattern.MatchString(string(connectorType)) {
		return "", fmt.Errorf("type must match %s", connectorTypePattern.String())
	}

	return connectorType, nil
}

func ParseStatus(raw string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(raw)))
	if status == "" {
		status = StatusActive
	}

	switch status {
	case StatusActive, StatusPaused, StatusError:
		return status, nil
	default:
		return "", fmt.Errorf("status must be one of active, paused, error")
	}
}

func ParseSyncDirection(raw string) (SyncDirection, error) {
	direction := SyncDirection(strings.ToLower(strings.TrimSpace(raw)))
	if direction == "" {
		direction = SyncDirectionBidirectional
	}

	switch direction {
	case SyncDirectionPullOnly, SyncDirectionPushOnly, SyncDirectionBidirectional:
		return direction, nil
	default:
		return "", fmt.Errorf("sync_direction must be one of pull_only, push_only, bidirectional")
	}
}

func (d SyncDirection) AllowsPull() bool {
	return d == SyncDirectionPullOnly || d == SyncDirectionBidirectional
}

func (d SyncDirection) AllowsPush() bool {
	return d == SyncDirectionPushOnly || d == SyncDirectionBidirectional
}

func (d SyncDirection) AllowsSyncBack() bool {
	return d == SyncDirectionBidirectional
}

func (c Config) MapStatus(externalStatus string) string {
	normalized := strings.ToLower(strings.TrimSpace(externalStatus))
	if normalized == "" {
		return ""
	}
	if mapped, ok := c.StatusMapping[normalized]; ok {
		return mapped
	}

	return strings.TrimSpace(externalStatus)
}

func (f Filters) Matches(issue ExternalIssue) bool {
	if len(f.States) > 0 && !slices.Contains(f.States, strings.ToLower(strings.TrimSpace(issue.Status))) {
		return false
	}
	if len(f.Authors) > 0 && !slices.Contains(f.Authors, strings.ToLower(strings.TrimSpace(issue.Author))) {
		return false
	}

	issueLabels := normalizeStringList(issue.Labels)
	if len(f.Labels) > 0 {
		matched := false
		for _, label := range issueLabels {
			if slices.Contains(f.Labels, label) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, label := range issueLabels {
		if slices.Contains(f.ExcludeLabels, label) {
			return false
		}
	}

	return true
}

func (c IssueConnector) CanPullAt(now time.Time) bool {
	if c.Status != StatusActive || !c.Config.SyncDirection.AllowsPull() {
		return false
	}
	if c.LastSyncAt == nil {
		return true
	}

	lastSyncAt := c.LastSyncAt.UTC()
	return !now.UTC().Before(lastSyncAt.Add(c.Config.PollInterval))
}

func (c IssueConnector) CanReceiveWebhook() bool {
	return c.Status == StatusActive && c.Config.SyncDirection.AllowsPush()
}

func (c IssueConnector) CanSyncBack() bool {
	return c.Status == StatusActive && c.Config.SyncDirection.AllowsSyncBack()
}

func (c IssueConnector) LastSyncCursor() time.Time {
	if c.LastSyncAt == nil {
		return time.Time{}
	}

	return c.LastSyncAt.UTC()
}

func (c *IssueConnector) RecordSync(now time.Time, syncedIssues int) {
	syncedAt := now.UTC()
	if c.LastSyncAt == nil || syncedAt.Sub(c.LastSyncAt.UTC()) > 24*time.Hour {
		c.Stats.Synced24h = syncedIssues
	} else {
		c.Stats.Synced24h += syncedIssues
	}
	c.Stats.TotalSynced += syncedIssues
	c.Status = StatusActive
	c.LastSyncAt = &syncedAt
	c.LastError = ""
}

func (c *IssueConnector) RecordFailure(err error) {
	if err == nil {
		return
	}

	c.Status = StatusError
	c.LastError = strings.TrimSpace(err.Error())
	c.Stats.FailedCount++
}

func parseName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", fmt.Errorf("name must not be empty")
	}

	return name, nil
}

func parsePollInterval(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultPollInterval, nil
	}

	pollInterval, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("poll_interval must be a valid duration: %w", err)
	}
	if pollInterval <= 0 {
		return 0, fmt.Errorf("poll_interval must be greater than zero")
	}

	return pollInterval, nil
}

func parseStatusMapping(raw map[string]string) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}

	normalized := make(map[string]string, len(raw))
	for key, value := range raw {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if normalizedKey == "" {
			return nil, fmt.Errorf("status_mapping keys must not be empty")
		}

		normalizedValue := strings.TrimSpace(value)
		if normalizedValue == "" {
			return nil, fmt.Errorf("status_mapping[%s] must not be empty", normalizedKey)
		}
		normalized[normalizedKey] = normalizedValue
	}

	return normalized, nil
}

func normalizeStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(items))
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.ToLower(strings.TrimSpace(item))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	slices.Sort(normalized)

	return normalized
}
