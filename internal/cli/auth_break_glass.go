package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/config"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
	"github.com/spf13/cobra"
)

type breakGlassDisableOIDCResponse struct {
	Status      string   `json:"status"`
	ActiveMode  string   `json:"active_mode"`
	NextSteps   []string `json:"next_steps"`
	ConfigPath  string   `json:"config_path,omitempty"`
	StoragePath string   `json:"storage_path,omitempty"`
}

func newAuthBreakGlassCommand(options *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "break-glass",
		Short: "Run local recovery actions when browser OIDC access is misconfigured.",
	}
	command.AddCommand(newAuthBreakGlassDisableOIDCCommand(options))
	return command
}

func newAuthBreakGlassDisableOIDCCommand(options *rootOptions) *cobra.Command {
	var format string

	command := &cobra.Command{
		Use:   "disable-oidc",
		Short: "Disable the active OIDC browser auth config locally and keep the saved draft for repair.",
		Long: strings.TrimSpace(`
Disable the active OIDC browser auth config directly through the local
database-backed access-control state. This is the break-glass path when the
current OIDC rollout locks administrators out of the browser. The saved draft
remains available so you can repair it later after re-entering through a local
bootstrap link.
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(config.LoadOptions{ConfigFile: options.configFile})
			if err != nil {
				logConfigLoadFailure(options.configFile, nil, err)
				return err
			}

			client, err := database.Open(cmd.Context(), cfg.Database.DSN)
			if err != nil {
				return err
			}
			defer func() {
				_ = client.Close()
			}()

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve user home directory: %w", err)
			}
			service, err := accesscontrolservice.New(
				accesscontrolrepo.NewEntRepository(client),
				cfg.Database.DSN,
				options.configFile,
				homeDir,
			)
			if err != nil {
				return err
			}

			current, err := service.Read(cmd.Context())
			if err != nil {
				return err
			}
			runtimeState := iam.ResolveRuntimeAccessControlState(current.State)
			response := breakGlassDisableOIDCResponse{
				ConfigPath:  cfg.Metadata.ConfigFile,
				StoragePath: current.StorageLocation,
				NextSteps: []string{
					"Run `openase auth bootstrap create-link --return-to /admin/auth --format text`.",
					"Open the returned local bootstrap URL in a browser on this machine.",
					"After signing in, repair or re-enable OIDC from `/admin/auth`.",
				},
			}

			if runtimeState.AuthMode != iam.AuthModeOIDC {
				response.Status = "already_disabled"
				response.ActiveMode = runtimeState.AuthMode.String()
				return writeBreakGlassDisableOIDCResponse(cmd.Context(), cmd.OutOrStdout(), response, format)
			}

			disabled, err := service.Disable(cmd.Context())
			if err != nil {
				return err
			}
			response.Status = "disabled"
			response.ActiveMode = iam.ResolveRuntimeAccessControlState(disabled.State).AuthMode.String()
			response.StoragePath = disabled.StorageLocation
			return writeBreakGlassDisableOIDCResponse(cmd.Context(), cmd.OutOrStdout(), response, format)
		},
	}

	command.Flags().StringVar(&format, "format", "text", "Output format: text or json.")
	return command
}

func writeBreakGlassDisableOIDCResponse(_ context.Context, out interface{ Write([]byte) (int, error) }, response breakGlassDisableOIDCResponse, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		body, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("marshal break-glass response: %w", err)
		}
		return writePrettyJSON(out, body)
	case "", "text":
		if response.Status == "already_disabled" {
			_, err := fmt.Fprintf(
				out,
				"OIDC is already inactive (active mode: %s).\nNext:\n- %s\n- %s\n- %s\n",
				response.ActiveMode,
				response.NextSteps[0],
				response.NextSteps[1],
				response.NextSteps[2],
			)
			return err
		}
		_, err := fmt.Fprintf(
			out,
			"Disabled active OIDC browser auth and preserved the draft for repair.\nNext:\n- %s\n- %s\n- %s\n",
			response.NextSteps[0],
			response.NextSteps[1],
			response.NextSteps[2],
		)
		return err
	default:
		return fmt.Errorf("unsupported format %q, expected text or json", format)
	}
}
