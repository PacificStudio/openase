package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	issueconnectorservice "github.com/BetterAndBetterII/openase/internal/service/issueconnector"
	"github.com/google/uuid"
)

type rawCreateIssueConnectorRequest domain.Input

type rawUpdateIssueConnectorRequest struct {
	Name   *string                        `json:"name"`
	Status *string                        `json:"status"`
	Config *rawUpdateIssueConnectorConfig `json:"config"`
}

type rawUpdateIssueConnectorConfig struct {
	BaseURL       *string                  `json:"base_url"`
	AuthToken     *string                  `json:"auth_token"`
	ProjectRef    *string                  `json:"project_ref"`
	PollInterval  *string                  `json:"poll_interval"`
	SyncDirection *string                  `json:"sync_direction"`
	Filters       optionalConnectorFilters `json:"filters"`
	StatusMapping optionalStringMap        `json:"status_mapping"`
	WebhookSecret *string                  `json:"webhook_secret"`
	AutoWorkflow  *string                  `json:"auto_workflow"`
}

type optionalConnectorFilters struct {
	Set   bool
	Value domain.FiltersInput
}

func (o *optionalConnectorFilters) UnmarshalJSON(data []byte) error {
	o.Set = true
	if isJSONNull(data) {
		o.Value = domain.FiltersInput{}
		return nil
	}

	return json.Unmarshal(data, &o.Value)
}

type optionalStringMap struct {
	Set   bool
	Value map[string]string
}

func (o *optionalStringMap) UnmarshalJSON(data []byte) error {
	o.Set = true
	if isJSONNull(data) {
		o.Value = map[string]string{}
		return nil
	}

	return json.Unmarshal(data, &o.Value)
}

func parseCreateIssueConnectorRequest(
	projectID uuid.UUID,
	raw rawCreateIssueConnectorRequest,
) (domain.CreateIssueConnector, error) {
	return domain.ParseCreateIssueConnector(projectID, domain.Input(raw))
}

func parseUpdateIssueConnectorRequest(
	connectorID uuid.UUID,
	raw rawUpdateIssueConnectorRequest,
) (issueconnectorservice.UpdateInput, error) {
	input := issueconnectorservice.UpdateInput{ID: connectorID}
	changed := false

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return issueconnectorservice.UpdateInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = issueconnectorservice.Optional[string]{Set: true, Value: name}
		changed = true
	}

	if raw.Status != nil {
		status, err := domain.ParseStatus(*raw.Status)
		if err != nil {
			return issueconnectorservice.UpdateInput{}, err
		}
		input.Status = issueconnectorservice.Optional[domain.Status]{Set: true, Value: status}
		changed = true
	}

	if raw.Config != nil {
		changed = true
		if raw.Config.BaseURL != nil {
			input.BaseURL = issueconnectorservice.Optional[string]{
				Set:   true,
				Value: strings.TrimSpace(*raw.Config.BaseURL),
			}
		}
		if raw.Config.AuthToken != nil {
			input.AuthToken = issueconnectorservice.Optional[string]{
				Set:   true,
				Value: strings.TrimSpace(*raw.Config.AuthToken),
			}
		}
		if raw.Config.ProjectRef != nil {
			input.ProjectRef = issueconnectorservice.Optional[string]{
				Set:   true,
				Value: strings.TrimSpace(*raw.Config.ProjectRef),
			}
		}
		if raw.Config.PollInterval != nil {
			pollInterval, err := time.ParseDuration(strings.TrimSpace(*raw.Config.PollInterval))
			if err != nil {
				return issueconnectorservice.UpdateInput{}, fmt.Errorf("poll_interval must be a valid duration: %w", err)
			}
			if pollInterval <= 0 {
				return issueconnectorservice.UpdateInput{}, fmt.Errorf("poll_interval must be greater than zero")
			}
			input.PollInterval = issueconnectorservice.Optional[time.Duration]{Set: true, Value: pollInterval}
		}
		if raw.Config.SyncDirection != nil {
			direction, err := domain.ParseSyncDirection(*raw.Config.SyncDirection)
			if err != nil {
				return issueconnectorservice.UpdateInput{}, err
			}
			input.SyncDirection = issueconnectorservice.Optional[domain.SyncDirection]{
				Set:   true,
				Value: direction,
			}
		}
		if raw.Config.Filters.Set {
			input.Filters = issueconnectorservice.Optional[domain.Filters]{
				Set:   true,
				Value: domain.ParseFilters(raw.Config.Filters.Value),
			}
		}
		if raw.Config.StatusMapping.Set {
			for key, value := range raw.Config.StatusMapping.Value {
				if strings.TrimSpace(key) == "" {
					return issueconnectorservice.UpdateInput{}, fmt.Errorf("status_mapping keys must not be empty")
				}
				if strings.TrimSpace(value) == "" {
					return issueconnectorservice.UpdateInput{}, fmt.Errorf("status_mapping[%s] must not be empty", strings.ToLower(strings.TrimSpace(key)))
				}
			}
			input.StatusMapping = issueconnectorservice.Optional[map[string]string]{
				Set:   true,
				Value: cloneHTTPStringMap(raw.Config.StatusMapping.Value),
			}
		}
		if raw.Config.WebhookSecret != nil {
			input.WebhookSecret = issueconnectorservice.Optional[string]{
				Set:   true,
				Value: strings.TrimSpace(*raw.Config.WebhookSecret),
			}
		}
		if raw.Config.AutoWorkflow != nil {
			input.AutoWorkflow = issueconnectorservice.Optional[string]{
				Set:   true,
				Value: strings.TrimSpace(*raw.Config.AutoWorkflow),
			}
		}
	}

	if !changed {
		return issueconnectorservice.UpdateInput{}, fmt.Errorf("at least one connector field must be provided")
	}

	return input, nil
}

func cloneHTTPStringMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}

	return cloned
}
