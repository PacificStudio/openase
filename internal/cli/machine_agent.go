package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	machinechanneldomain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	machinechannelrepo "github.com/BetterAndBetterII/openase/internal/repo/machinechannel"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	"github.com/spf13/cobra"
)

type machineChannelTokenResponse struct {
	Token           string            `json:"token,omitempty"`
	TokenID         string            `json:"token_id"`
	MachineID       string            `json:"machine_id"`
	ExpiresAt       string            `json:"expires_at,omitempty"`
	Revoked         bool              `json:"revoked,omitempty"`
	ControlPlaneURL string            `json:"control_plane_url,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
}

func newMachineAgentCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "machine-agent",
		Short: "Run the reverse websocket machine daemon.",
		Long: strings.TrimSpace(`
Run the reverse websocket machine daemon.

Use this on remote machines that can reach the OpenASE control plane but cannot
accept inbound SSH or direct control-plane connections. The daemon reads machine
identity and credentials from flags or OPENASE_MACHINE_* environment variables,
establishes a long-lived websocket session, and keeps the session alive with
heartbeats until the process stops.
`),
		Example: strings.TrimSpace(`
  openase machine-agent run \
    --machine-id 550e8400-e29b-41d4-a716-446655440000 \
    --token ase_machine_xxxxx \
    --control-plane-url http://127.0.0.1:19836

  OPENASE_MACHINE_ID=550e8400-e29b-41d4-a716-446655440000 \
  OPENASE_MACHINE_CHANNEL_TOKEN=ase_machine_xxxxx \
  OPENASE_MACHINE_CONTROL_PLANE_URL=http://127.0.0.1:19836 \
  openase machine-agent run
`),
	}
	command.AddCommand(newMachineAgentRunCommand())
	return command
}

func newMachineAgentRunCommand() *cobra.Command {
	var (
		machineID         string
		token             string
		controlPlaneURL   string
		heartbeatInterval time.Duration
		reconnectBackoff  time.Duration
		openaseBinaryPath string
		agentCLIPath      string
	)

	command := &cobra.Command{
		Use:   "run",
		Short: "Connect a machine daemon to the control plane over reverse websocket.",
		Long: strings.TrimSpace(`
Connect a machine daemon to the control plane over reverse websocket.

This command reads the machine UUID, dedicated machine channel token, and the
control-plane base URL from flags or OPENASE_MACHINE_* environment variables.
It keeps reconnecting with backoff until the process is interrupted or the
control plane rejects the credential as invalid, revoked, expired, or disabled.

The control plane URL may be an HTTP base such as http://127.0.0.1:19836, an
API base such as http://127.0.0.1:19836/api/v1, or a direct websocket endpoint.
`),
		Example: strings.TrimSpace(`
  openase machine-agent run \
    --machine-id 550e8400-e29b-41d4-a716-446655440000 \
    --token ase_machine_xxxxx \
    --control-plane-url http://127.0.0.1:19836 \
    --heartbeat-interval 15s
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedOpenASEBinaryPath := strings.TrimSpace(openaseBinaryPath)
			if resolvedOpenASEBinaryPath == "" {
				executablePath, err := os.Executable()
				if err == nil {
					resolvedOpenASEBinaryPath = executablePath
				}
			}

			config, err := machinechanneldomain.ParseDaemonConfig(
				firstNonEmpty(machineID, os.Getenv(machinechanneldomain.EnvMachineID)),
				firstNonEmpty(token, os.Getenv(machinechanneldomain.EnvMachineChannelToken)),
				firstNonEmpty(controlPlaneURL, os.Getenv(machinechanneldomain.EnvMachineControlPlaneURL)),
				firstNonZeroDuration(heartbeatInterval, parseEnvDuration(machinechanneldomain.EnvMachineHeartbeatInterval)),
				firstNonZeroDuration(reconnectBackoff, machinechannelservice.DefaultReconnectBackoff),
				resolvedOpenASEBinaryPath,
				agentCLIPath,
			)
			if err != nil {
				return err
			}

			return machinechannelservice.NewDaemon(nil).Run(cmd.Context(), config)
		},
	}

	command.Flags().StringVar(&machineID, "machine-id", "", "Machine UUID. Defaults to OPENASE_MACHINE_ID.")
	command.Flags().StringVar(&token, "token", "", "Dedicated machine channel token. Defaults to OPENASE_MACHINE_CHANNEL_TOKEN.")
	command.Flags().StringVar(&controlPlaneURL, "control-plane-url", "", "OpenASE control-plane base URL or websocket endpoint. Defaults to OPENASE_MACHINE_CONTROL_PLANE_URL.")
	command.Flags().DurationVar(&heartbeatInterval, "heartbeat-interval", machinechannelservice.DefaultHeartbeatInterval, "Heartbeat interval, for example 15s. Defaults to OPENASE_MACHINE_HEARTBEAT_INTERVAL when set.")
	command.Flags().DurationVar(&reconnectBackoff, "reconnect-backoff", machinechannelservice.DefaultReconnectBackoff, "Reconnect backoff after connection loss.")
	command.Flags().StringVar(&openaseBinaryPath, "openase-binary-path", "", "Absolute path to the OpenASE binary. Defaults to the current executable.")
	command.Flags().StringVar(&agentCLIPath, "agent-cli-path", "", "Absolute path to the preferred agent CLI binary to advertise to the control plane.")
	return command
}

func newMachineIssueChannelTokenCommand(options *rootOptions) *cobra.Command {
	var machineID string
	var ttl time.Duration
	var controlPlaneURL string
	var format string

	command := &cobra.Command{
		Use:   "issue-channel-token",
		Short: "Issue a dedicated machine channel token for reverse websocket registration.",
		Long: strings.TrimSpace(`
Issue a dedicated machine channel token for reverse websocket registration.

This command talks directly to the local OpenASE database using the configured
server config. The machine ID must be a UUID and the issued credential is kept
separate from OPENASE_AGENT_TOKEN and other runtime agent credentials.
`),
		Example: strings.TrimSpace(`
  openase machine issue-channel-token \
    --machine-id 550e8400-e29b-41d4-a716-446655440000 \
    --ttl 24h \
    --format shell
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(config.LoadOptions{ConfigFile: options.configFile})
			if err != nil {
				logConfigLoadFailure(options.configFile, nil, err)
				return err
			}

			parsedMachineID, err := parseUUIDFlag("machine-id", machineID)
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

			issued, err := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client)).IssueToken(cmd.Context(), machinechanneldomain.IssueInput{
				MachineID: parsedMachineID,
				TTL:       ttl,
			})
			if err != nil {
				return err
			}

			resolvedControlPlaneURL, err := resolveControlPlaneURL(cfg, controlPlaneURL)
			if err != nil {
				return err
			}
			response := machineChannelTokenResponse{
				Token:           issued.Token,
				TokenID:         issued.TokenID.String(),
				MachineID:       issued.MachineID.String(),
				ExpiresAt:       issued.ExpiresAt.UTC().Format(time.RFC3339),
				ControlPlaneURL: resolvedControlPlaneURL,
				Environment: map[string]string{
					machinechanneldomain.EnvMachineID:              issued.MachineID.String(),
					machinechanneldomain.EnvMachineChannelToken:    issued.Token,
					machinechanneldomain.EnvMachineControlPlaneURL: resolvedControlPlaneURL,
				},
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "json":
				return writeMachineChannelTokenJSON(cmd.Context(), cmd.OutOrStdout(), response)
			case "shell":
				return writeMachineChannelTokenShell(cmd.OutOrStdout(), response.Environment)
			default:
				return fmt.Errorf("unsupported format %q, expected json or shell", format)
			}
		},
	}

	command.Flags().StringVar(&machineID, "machine-id", "", "Machine UUID.")
	command.Flags().DurationVar(&ttl, "ttl", 24*time.Hour, "Token lifetime, for example 30m or 24h.")
	command.Flags().StringVar(&controlPlaneURL, "control-plane-url", "", "Control-plane base URL override. Defaults to the configured local server URL.")
	command.Flags().StringVar(&format, "format", "json", "Output format: json or shell.")
	_ = command.MarkFlagRequired("machine-id")
	return command
}

func newMachineRevokeChannelTokenCommand(options *rootOptions) *cobra.Command {
	var machineID string
	var tokenID string

	command := &cobra.Command{
		Use:   "revoke-channel-token",
		Short: "Revoke a dedicated machine channel token.",
		Long: strings.TrimSpace(`
Revoke a dedicated machine channel token.

This command updates the local OpenASE database directly using the configured
server config. Both --machine-id and --token-id must be UUID values so the
correct machine-bound reverse websocket credential is revoked.
`),
		Example: strings.TrimSpace(`
  openase machine revoke-channel-token \
    --machine-id 550e8400-e29b-41d4-a716-446655440000 \
    --token-id 660e8400-e29b-41d4-a716-446655440000
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(config.LoadOptions{ConfigFile: options.configFile})
			if err != nil {
				logConfigLoadFailure(options.configFile, nil, err)
				return err
			}

			parsedMachineID, err := parseUUIDFlag("machine-id", machineID)
			if err != nil {
				return err
			}
			parsedTokenID, err := parseUUIDFlag("token-id", tokenID)
			if err != nil {
				return err
			}

			client, err := database.Open(cmd.Context(), cfg.Database.DSN)
			if err != nil {
				return err
			}
			defer func() {
				_ = client.Close()
			}()

			if err := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client)).RevokeToken(cmd.Context(), parsedMachineID, parsedTokenID); err != nil {
				return err
			}

			return writeMachineChannelTokenJSON(cmd.Context(), cmd.OutOrStdout(), machineChannelTokenResponse{
				TokenID:   parsedTokenID.String(),
				MachineID: parsedMachineID.String(),
				Revoked:   true,
			})
		},
	}

	command.Flags().StringVar(&machineID, "machine-id", "", "Machine UUID.")
	command.Flags().StringVar(&tokenID, "token-id", "", "Channel token UUID.")
	_ = command.MarkFlagRequired("machine-id")
	_ = command.MarkFlagRequired("token-id")
	return command
}

func resolveControlPlaneURL(cfg config.Config, explicit string) (string, error) {
	trimmed := strings.TrimSpace(explicit)
	if trimmed != "" {
		if _, err := url.ParseRequestURI(trimmed); err != nil {
			return "", fmt.Errorf("parse control-plane-url: %w", err)
		}
		return strings.TrimRight(trimmed, "/"), nil
	}

	host := strings.TrimSpace(cfg.Server.Host)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}

	return "http://" + net.JoinHostPort(host, fmt.Sprintf("%d", cfg.Server.Port)), nil
}

func parseEnvDuration(key string) time.Duration {
	trimmed := strings.TrimSpace(os.Getenv(key))
	if trimmed == "" {
		return 0
	}
	parsed, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0
	}
	return parsed
}

func firstNonZeroDuration(values ...time.Duration) time.Duration {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func writeMachineChannelTokenJSON(_ context.Context, out interface{ Write([]byte) (int, error) }, response machineChannelTokenResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal machine channel token response: %w", err)
	}
	return writePrettyJSON(out, body)
}

func writeMachineChannelTokenShell(out interface{ Write([]byte) (int, error) }, environment map[string]string) error {
	order := []string{
		machinechanneldomain.EnvMachineID,
		machinechanneldomain.EnvMachineChannelToken,
		machinechanneldomain.EnvMachineControlPlaneURL,
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
