package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	accesscontrolrepo "github.com/BetterAndBetterII/openase/internal/repo/accesscontrol"
	humanauthrepo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	accesscontrolservice "github.com/BetterAndBetterII/openase/internal/service/accesscontrol"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	"github.com/spf13/cobra"
)

type localBootstrapLinkResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Nonce     string `json:"nonce"`
	Purpose   string `json:"purpose"`
	ExpiresAt string `json:"expires_at"`
	URL       string `json:"url"`
}

func newAuthBootstrapCommand(options *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "bootstrap",
		Short: "Create and redeem local bootstrap browser authorization links.",
	}
	command.AddCommand(newAuthBootstrapCreateLinkCommand(options))
	return command
}

func newAuthBootstrapCreateLinkCommand(options *rootOptions) *cobra.Command {
	var (
		controlPlaneURL string
		returnTo        string
		requestedBy     string
		ttl             time.Duration
		format          string
	)

	command := &cobra.Command{
		Use:   "create-link",
		Short: "Create a short-lived local bootstrap authorization link for browser session setup.",
		Long: strings.TrimSpace(`
Create a short-lived local bootstrap authorization link for browser session setup.

This command talks directly to the local OpenASE database using the configured
server config. It only works when no active OIDC configuration exists. The
generated URL carries one-time authorization material and never embeds a
long-lived bearer token.
`),
		Example: strings.TrimSpace(`
  openase auth bootstrap create-link
  openase auth bootstrap create-link --ttl 5m --format text
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
			authStateSvc, err := accesscontrolservice.New(
				accesscontrolrepo.NewEntRepository(client),
				cfg.Database.DSN,
				options.configFile,
				homeDir,
			)
			if err != nil {
				return err
			}

			requestedByValue := strings.TrimSpace(requestedBy)
			if requestedByValue == "" {
				requestedByValue = defaultLocalBootstrapRequestedBy()
			}

			service := humanauthservice.NewService(humanauthrepo.NewEntRepository(client), nil, authStateSvc)
			issued, err := service.CreateLocalBootstrapRequest(cmd.Context(), humanauthservice.LocalBootstrapIssueInput{
				RequestedBy: requestedByValue,
				Purpose:     "browser_session",
				TTL:         ttl,
			})
			if err != nil {
				return err
			}

			resolvedControlPlaneURL, err := resolveControlPlaneURL(cfg, controlPlaneURL)
			if err != nil {
				return err
			}
			redeemURL, err := buildLocalBootstrapRedeemURL(
				resolvedControlPlaneURL,
				issued.RequestID,
				issued.Code,
				issued.Nonce,
				humanauthservice.NormalizeReturnTo(returnTo),
			)
			if err != nil {
				return err
			}

			response := localBootstrapLinkResponse{
				RequestID: issued.RequestID,
				Code:      issued.Code,
				Nonce:     issued.Nonce,
				Purpose:   issued.Purpose,
				ExpiresAt: issued.ExpiresAt.UTC().Format(time.RFC3339),
				URL:       redeemURL,
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "json":
				return writeLocalBootstrapLinkJSON(cmd.Context(), cmd.OutOrStdout(), response)
			case "text":
				return writeLocalBootstrapLinkText(cmd.OutOrStdout(), response)
			default:
				return fmt.Errorf("unsupported format %q, expected json or text", format)
			}
		},
	}

	command.Flags().StringVar(&controlPlaneURL, "control-plane-url", "", "Control-plane base URL override. Defaults to the configured local server URL.")
	command.Flags().StringVar(&returnTo, "return-to", "/", "Post-login return path, for example / or /admin/auth.")
	command.Flags().StringVar(&requestedBy, "requested-by", "", "Audit actor label for the request creator. Defaults to the current local user.")
	command.Flags().DurationVar(&ttl, "ttl", humanauthservice.DefaultLocalBootstrapRequestTTL, "Authorization link lifetime, for example 5m.")
	command.Flags().StringVar(&format, "format", "json", "Output format: json or text.")

	return command
}

func defaultLocalBootstrapRequestedBy() string {
	currentUser, err := user.Current()
	if err != nil {
		if value := strings.TrimSpace(os.Getenv("USER")); value != "" {
			return "cli:" + value
		}
		return "cli:unknown"
	}
	if value := strings.TrimSpace(currentUser.Username); value != "" {
		return "cli:" + value
	}
	return "cli:unknown"
}

func buildLocalBootstrapRedeemURL(baseURL string, requestID string, code string, nonce string, returnTo string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil {
		return "", fmt.Errorf("parse control-plane-url: %w", err)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/local-bootstrap"
	values := parsed.Query()
	values.Set("request_id", strings.TrimSpace(requestID))
	values.Set("code", strings.TrimSpace(code))
	values.Set("nonce", strings.TrimSpace(nonce))
	values.Set("return_to", humanauthservice.NormalizeReturnTo(returnTo))
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func writeLocalBootstrapLinkJSON(_ context.Context, out interface{ Write([]byte) (int, error) }, response localBootstrapLinkResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal local bootstrap link response: %w", err)
	}
	return writePrettyJSON(out, body)
}

func writeLocalBootstrapLinkText(out interface{ Write([]byte) (int, error) }, response localBootstrapLinkResponse) error {
	_, err := fmt.Fprintf(
		out,
		"Open this URL in a browser before %s:\n%s\n",
		response.ExpiresAt,
		response.URL,
	)
	return err
}
