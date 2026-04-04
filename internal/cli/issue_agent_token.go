package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	agentplatformrepo "github.com/BetterAndBetterII/openase/internal/repo/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type issueAgentTokenResponse struct {
	Token       string            `json:"token"`
	ProjectID   string            `json:"project_id"`
	TicketID    string            `json:"ticket_id"`
	Scopes      []string          `json:"scopes"`
	ExpiresAt   string            `json:"expires_at"`
	APIURL      string            `json:"api_url,omitempty"`
	Environment map[string]string `json:"environment"`
}

func newIssueAgentTokenCommand(options *rootOptions) *cobra.Command {
	var agentID string
	var projectID string
	var ticketID string
	var scopes []string
	var ttl time.Duration
	var apiURL string
	var format string

	command := &cobra.Command{
		Use:   "issue-agent-token",
		Short: "Issue a development agent token for the local agent platform API.",
		Long: strings.TrimSpace(`
Issue a development agent token for the local agent platform API.

This command is intended for local agent and harness debugging. The required
--agent-id, --project-id, and --ticket-id values are all UUIDs.

In particular, --ticket-id becomes OPENASE_TICKET_ID in the emitted environment,
and platform routes such as /api/v1/platform/tickets/:ticketId expect that UUID.
Do not pass a human-readable ticket identifier such as ASE-2 here.
`),
		Example: strings.TrimSpace(`
  openase issue-agent-token \
    --agent-id 87bcf5d1-019c-463a-87b5-70a47afb4880 \
    --project-id 7b3b00c6-636c-4a1d-9ce5-9e04b1cd6d7a \
    --ticket-id a6d58b41-4a6d-453d-a5ec-b7c37fe1eaff
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(config.LoadOptions{ConfigFile: options.configFile})
			if err != nil {
				logConfigLoadFailure(options.configFile, nil, err)
				return err
			}

			parsedAgentID, err := parseUUIDFlag("agent-id", agentID)
			if err != nil {
				return err
			}
			parsedProjectID, err := parseUUIDFlag("project-id", projectID)
			if err != nil {
				return err
			}
			parsedTicketID, err := parseUUIDFlag("ticket-id", ticketID)
			if err != nil {
				return err
			}
			if ttl < 0 {
				return fmt.Errorf("ttl must be greater than or equal to zero")
			}

			client, err := database.Open(cmd.Context(), cfg.Database.DSN)
			if err != nil {
				return err
			}
			defer func() {
				_ = client.Close()
			}()

			issued, err := agentplatform.NewService(agentplatformrepo.NewEntRepository(client)).IssueToken(cmd.Context(), agentplatform.IssueInput{
				AgentID:   parsedAgentID,
				ProjectID: parsedProjectID,
				TicketID:  parsedTicketID,
				Scopes:    scopes,
				TTL:       ttl,
			})
			if err != nil {
				return err
			}

			resolvedAPIURL, err := resolveAgentPlatformAPIURL(cfg, apiURL)
			if err != nil {
				return err
			}
			response := issueAgentTokenResponse{
				Token:     issued.Token,
				ProjectID: issued.ProjectID.String(),
				TicketID:  issued.TicketID.String(),
				Scopes:    append([]string(nil), issued.Scopes...),
				ExpiresAt: issued.ExpiresAt.UTC().Format(time.RFC3339),
				APIURL:    resolvedAPIURL,
				Environment: buildIssueAgentTokenEnvironment(
					resolvedAPIURL,
					issued.Token,
					issued.ProjectID,
					issued.TicketID,
					issued.ExpiresAt,
				),
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "json":
				return writeIssueAgentTokenJSON(cmd.Context(), cmd.OutOrStdout(), response)
			case "shell":
				return writeIssueAgentTokenShell(cmd.OutOrStdout(), response.Environment)
			default:
				return fmt.Errorf("unsupported format %q, expected json or shell", format)
			}
		},
	}

	command.Flags().StringVar(&agentID, "agent-id", "", "Agent UUID.")
	command.Flags().StringVar(&projectID, "project-id", "", "Project UUID.")
	command.Flags().StringVar(&ticketID, "ticket-id", "", "Current ticket UUID. This becomes OPENASE_TICKET_ID; do not pass a ticket identifier like ASE-2.")
	command.Flags().StringSliceVar(&scopes, "scope", nil, "Explicit agent scope. Repeat for multiple scopes.")
	command.Flags().DurationVar(&ttl, "ttl", 24*time.Hour, "Token lifetime, for example 30m or 2h.")
	command.Flags().StringVar(&apiURL, "api-url", "", "Platform API base URL override. Defaults to the configured local server URL.")
	command.Flags().StringVar(&format, "format", "json", "Output format: json or shell.")
	_ = command.MarkFlagRequired("agent-id")
	_ = command.MarkFlagRequired("project-id")
	_ = command.MarkFlagRequired("ticket-id")

	return command
}

func parseUUIDFlag(name string, raw string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return uuid.UUID{}, fmt.Errorf("%s must not be empty", name)
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s must be a valid UUID", name)
	}

	return parsed, nil
}

func resolveAgentPlatformAPIURL(cfg config.Config, explicit string) (string, error) {
	trimmed := strings.TrimSpace(explicit)
	if trimmed != "" {
		if _, err := url.ParseRequestURI(trimmed); err != nil {
			return "", fmt.Errorf("parse api-url: %w", err)
		}
		return strings.TrimRight(trimmed, "/"), nil
	}

	host := strings.TrimSpace(cfg.Server.Host)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}

	return "http://" + net.JoinHostPort(host, fmt.Sprintf("%d", cfg.Server.Port)) + "/api/v1/platform", nil
}

func buildIssueAgentTokenEnvironment(apiURL string, token string, projectID uuid.UUID, ticketID uuid.UUID, expiresAt time.Time) map[string]string {
	environment := map[string]string{
		"OPENASE_AGENT_EXPIRES_AT": expiresAt.UTC().Format(time.RFC3339),
	}
	for _, item := range agentplatform.BuildEnvironment(apiURL, token, projectID, ticketID) {
		key, value, found := strings.Cut(item, "=")
		if !found || strings.TrimSpace(key) == "" {
			continue
		}
		environment[key] = value
	}

	return environment
}

func writeIssueAgentTokenJSON(_ context.Context, out interface{ Write([]byte) (int, error) }, response issueAgentTokenResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal issue agent token response: %w", err)
	}

	return writePrettyJSON(out, body)
}

func writeIssueAgentTokenShell(out interface{ Write([]byte) (int, error) }, environment map[string]string) error {
	order := []string{
		"OPENASE_API_URL",
		"OPENASE_AGENT_TOKEN",
		"OPENASE_PROJECT_ID",
		"OPENASE_TICKET_ID",
		"OPENASE_AGENT_EXPIRES_AT",
	}
	for _, key := range order {
		value, ok := environment[key]
		if !ok {
			continue
		}
		if _, err := fmt.Fprintf(out, "export %s=%q\n", key, value); err != nil {
			return err
		}
	}
	return nil
}
