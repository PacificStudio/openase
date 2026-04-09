package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type localBootstrapRedeemRequest struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Nonce     string `json:"nonce"`
}

type localBootstrapLoginResponse struct {
	Authenticated        bool     `json:"authenticated"`
	CurrentAuthMethod    string   `json:"current_auth_method,omitempty"`
	AvailableAuthMethods []string `json:"available_auth_methods,omitempty"`
	AuthMode             string   `json:"auth_mode,omitempty"`
	Roles                []string `json:"roles,omitempty"`
	Permissions          []string `json:"permissions,omitempty"`
	CSRFToken            string   `json:"csrf_token,omitempty"`
}

type localBootstrapLoginOutput struct {
	Authenticated     bool     `json:"authenticated"`
	AuthMode          string   `json:"auth_mode,omitempty"`
	CurrentAuthMethod string   `json:"current_auth_method,omitempty"`
	AvailableMethods  []string `json:"available_auth_methods,omitempty"`
	Roles             []string `json:"roles,omitempty"`
	Permissions       []string `json:"permissions,omitempty"`
	APIURL            string   `json:"api_url"`
	SessionFile       string   `json:"session_file"`
}

type authBootstrapLoginDeps struct {
	httpClient platformHTTPDoer
}

func newAuthBootstrapCommand(options *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "bootstrap",
		Short: "Create and redeem local bootstrap browser authorization links.",
	}
	command.AddCommand(newAuthBootstrapCreateLinkCommand(options))
	command.AddCommand(newAuthBootstrapLoginCommand(options, authBootstrapLoginDeps{httpClient: http.DefaultClient}))
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
			requestedByValue := strings.TrimSpace(requestedBy)
			if requestedByValue == "" {
				requestedByValue = defaultLocalBootstrapRequestedBy()
			}

			cfg, issued, err := createLocalBootstrapAuthorization(
				cmd.Context(),
				options.configFile,
				requestedByValue,
				ttl,
			)
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

func newAuthBootstrapLoginCommand(options *rootOptions, deps authBootstrapLoginDeps) *cobra.Command {
	if deps.httpClient == nil {
		deps.httpClient = http.DefaultClient
	}

	var (
		controlPlaneURL string
		requestedBy     string
		ttl             time.Duration
		format          string
		sessionFile     string
	)

	command := &cobra.Command{
		Use:   "login",
		Short: "Create and redeem a local bootstrap authorization, then persist a CLI human session.",
		Long: strings.TrimSpace(`
Create and redeem a local bootstrap authorization, then persist a CLI human
session for later OpenASE commands.

This command still requires local bootstrap mode. It stores the resulting
browser-equivalent session token and CSRF token in a local file with 0600
permissions so typed CLI commands can call the protected human API without
copying cookies by hand.
`),
		Example: strings.TrimSpace(`
  openase auth bootstrap login
  openase auth bootstrap login --control-plane-url http://127.0.0.1:19836 --format json
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			requestedByValue := strings.TrimSpace(requestedBy)
			if requestedByValue == "" {
				requestedByValue = defaultLocalBootstrapRequestedBy()
			}

			cfg, issued, err := createLocalBootstrapAuthorization(
				cmd.Context(),
				options.configFile,
				requestedByValue,
				ttl,
			)
			if err != nil {
				return err
			}

			resolvedControlPlaneURL, err := resolveControlPlaneURL(cfg, controlPlaneURL)
			if err != nil {
				return err
			}
			apiURL, err := apiBaseURLFromControlPlaneURL(resolvedControlPlaneURL)
			if err != nil {
				return err
			}

			loginResult, sessionToken, err := redeemLocalBootstrapAuthorization(
				cmd.Context(),
				deps.httpClient,
				apiURL,
				issued,
			)
			if err != nil {
				return err
			}

			sessionPath, err := resolveHumanSessionStatePath(sessionFile)
			if err != nil {
				return err
			}
			if err := saveHumanSessionState(sessionPath, humanSessionState{
				APIURL:            apiURL,
				SessionToken:      sessionToken,
				CSRFToken:         loginResult.CSRFToken,
				CurrentAuthMethod: loginResult.CurrentAuthMethod,
			}); err != nil {
				return err
			}

			return writeLocalBootstrapLoginOutput(cmd.OutOrStdout(), localBootstrapLoginOutput{
				Authenticated:     loginResult.Authenticated,
				AuthMode:          loginResult.AuthMode,
				CurrentAuthMethod: loginResult.CurrentAuthMethod,
				AvailableMethods:  append([]string(nil), loginResult.AvailableAuthMethods...),
				Roles:             append([]string(nil), loginResult.Roles...),
				Permissions:       append([]string(nil), loginResult.Permissions...),
				APIURL:            apiURL,
				SessionFile:       sessionPath,
			}, format)
		},
	}

	command.Flags().StringVar(&controlPlaneURL, "control-plane-url", "", "Control-plane base URL override. Defaults to the configured local server URL.")
	command.Flags().StringVar(&requestedBy, "requested-by", "", "Audit actor label for the request creator. Defaults to the current local user.")
	command.Flags().DurationVar(&ttl, "ttl", humanauthservice.DefaultLocalBootstrapRequestTTL, "Authorization link lifetime, for example 5m.")
	command.Flags().StringVar(&format, "format", "text", "Output format: text or json.")
	command.Flags().StringVar(&sessionFile, "session-file", "", "Session state file override. Defaults to "+defaultHumanSessionStateRelativePath+".")

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

func createLocalBootstrapAuthorization(
	ctx context.Context,
	configFile string,
	requestedBy string,
	ttl time.Duration,
) (config.Config, humanauthservice.LocalBootstrapIssueResult, error) {
	cfg, err := config.Load(config.LoadOptions{ConfigFile: configFile})
	if err != nil {
		logConfigLoadFailure(configFile, nil, err)
		return config.Config{}, humanauthservice.LocalBootstrapIssueResult{}, err
	}

	client, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		return config.Config{}, humanauthservice.LocalBootstrapIssueResult{}, err
	}
	defer func() {
		_ = client.Close()
	}()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config.Config{}, humanauthservice.LocalBootstrapIssueResult{}, fmt.Errorf("resolve user home directory: %w", err)
	}
	authStateSvc, err := accesscontrolservice.New(
		accesscontrolrepo.NewEntRepository(client),
		cfg.Database.DSN,
		configFile,
		homeDir,
	)
	if err != nil {
		return config.Config{}, humanauthservice.LocalBootstrapIssueResult{}, err
	}

	service := humanauthservice.NewService(humanauthrepo.NewEntRepository(client), nil, authStateSvc)
	issued, err := service.CreateLocalBootstrapRequest(ctx, humanauthservice.LocalBootstrapIssueInput{
		RequestedBy: strings.TrimSpace(requestedBy),
		Purpose:     "browser_session",
		TTL:         ttl,
	})
	if err != nil {
		return config.Config{}, humanauthservice.LocalBootstrapIssueResult{}, err
	}

	return cfg, issued, nil
}

func redeemLocalBootstrapAuthorization(
	ctx context.Context,
	httpClient platformHTTPDoer,
	apiURL string,
	issued humanauthservice.LocalBootstrapIssueResult,
) (localBootstrapLoginResponse, string, error) {
	requestURL, err := buildRequestURL(apiURL, "auth/local-bootstrap/redeem")
	if err != nil {
		return localBootstrapLoginResponse{}, "", err
	}
	payload, err := json.Marshal(localBootstrapRedeemRequest{
		RequestID: issued.RequestID,
		Code:      issued.Code,
		Nonce:     issued.Nonce,
	})
	if err != nil {
		return localBootstrapLoginResponse{}, "", fmt.Errorf("marshal local bootstrap redeem request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return localBootstrapLoginResponse{}, "", fmt.Errorf("build local bootstrap redeem request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("User-Agent", openASECLIUserAgent)
	origin, err := originFromAPIURL(apiURL)
	if err != nil {
		return localBootstrapLoginResponse{}, "", err
	}
	httpRequest.Header.Set("Origin", origin)

	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return localBootstrapLoginResponse{}, "", fmt.Errorf("POST auth/local-bootstrap/redeem: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return localBootstrapLoginResponse{}, "", fmt.Errorf("read local bootstrap redeem response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		code, message := parseAPIErrorBody(body)
		return localBootstrapLoginResponse{}, "", &apiHTTPError{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/local-bootstrap/redeem",
			StatusCode: response.StatusCode,
			Status:     response.Status,
			Code:       code,
			Message:    message,
		}
	}

	var loginResponse localBootstrapLoginResponse
	if err := json.Unmarshal(body, &loginResponse); err != nil {
		return localBootstrapLoginResponse{}, "", fmt.Errorf("decode local bootstrap redeem response: %w", err)
	}

	sessionToken := ""
	for _, cookie := range response.Cookies() {
		if cookie != nil && cookie.Name == humanSessionCookieHeaderName && strings.TrimSpace(cookie.Value) != "" {
			sessionToken = strings.TrimSpace(cookie.Value)
			break
		}
	}
	if sessionToken == "" {
		return localBootstrapLoginResponse{}, "", fmt.Errorf("local bootstrap redeem response did not set %s cookie", humanSessionCookieHeaderName)
	}
	if strings.TrimSpace(loginResponse.CSRFToken) == "" {
		return localBootstrapLoginResponse{}, "", fmt.Errorf("local bootstrap redeem response did not include csrf_token")
	}

	return loginResponse, sessionToken, nil
}

func writeLocalBootstrapLoginOutput(
	out interface{ Write([]byte) (int, error) },
	response localBootstrapLoginOutput,
	format string,
) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		body, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("marshal local bootstrap login response: %w", err)
		}
		return writePrettyJSON(out, body)
	case "", "text":
		_, err := fmt.Fprintf(
			out,
			"Stored CLI human session.\nAPI: %s\nSession file: %s\nAuth mode: %s\nCurrent auth method: %s\n",
			response.APIURL,
			response.SessionFile,
			firstNonEmpty(response.AuthMode, "unknown"),
			firstNonEmpty(response.CurrentAuthMethod, "unknown"),
		)
		return err
	default:
		return fmt.Errorf("unsupported format %q, expected text or json", format)
	}
}
