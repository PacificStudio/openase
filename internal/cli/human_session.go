package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultHumanSessionStateRelativePath = ".openase/human-session.json"
	// #nosec G101 -- environment variable names only; no secret material is embedded here.
	envHumanSessionFile = "OPENASE_HUMAN_SESSION_FILE"
	// #nosec G101 -- environment variable names only; no secret material is embedded here.
	envHumanSessionToken = "OPENASE_HUMAN_SESSION_TOKEN"
	// #nosec G101 -- environment variable names only; no secret material is embedded here.
	envHumanCSRFToken            = "OPENASE_HUMAN_CSRF_TOKEN"
	humanSessionCookieHeaderName = "openase_session"
	openASECLIUserAgent          = "openase-cli"
)

var errHumanSessionStateNotFound = errors.New("human session state not found")

type humanSessionState struct {
	APIURL            string `json:"api_url"`
	SessionToken      string `json:"session_token"`
	CSRFToken         string `json:"csrf_token,omitempty"`
	CurrentAuthMethod string `json:"current_auth_method,omitempty"`
	SavedAt           string `json:"saved_at"`
}

func defaultHumanSessionStatePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(homeDir, defaultHumanSessionStateRelativePath), nil
}

func resolveHumanSessionStatePath(explicit string) (string, error) {
	selected := strings.TrimSpace(firstNonEmpty(explicit, os.Getenv(envHumanSessionFile)))
	if selected == "" {
		return defaultHumanSessionStatePath()
	}
	if strings.HasPrefix(selected, "~"+string(os.PathSeparator)) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		selected = filepath.Join(homeDir, strings.TrimPrefix(selected, "~"+string(os.PathSeparator)))
	}
	absolutePath, err := filepath.Abs(selected)
	if err != nil {
		return "", fmt.Errorf("resolve human session path %q: %w", selected, err)
	}
	return absolutePath, nil
}

func loadHumanSessionState(path string) (humanSessionState, error) {
	body, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return humanSessionState{}, errHumanSessionStateNotFound
		}
		return humanSessionState{}, fmt.Errorf("read human session state: %w", err)
	}
	var state humanSessionState
	if err := json.Unmarshal(body, &state); err != nil {
		return humanSessionState{}, fmt.Errorf("decode human session state: %w", err)
	}
	state.APIURL = strings.TrimRight(strings.TrimSpace(state.APIURL), "/")
	state.SessionToken = strings.TrimSpace(state.SessionToken)
	state.CSRFToken = strings.TrimSpace(state.CSRFToken)
	state.CurrentAuthMethod = strings.TrimSpace(state.CurrentAuthMethod)
	state.SavedAt = strings.TrimSpace(state.SavedAt)
	if state.SessionToken == "" {
		return humanSessionState{}, fmt.Errorf("human session state %q is missing session_token", path)
	}
	return state, nil
}

func saveHumanSessionState(path string, state humanSessionState) error {
	state.APIURL = strings.TrimRight(strings.TrimSpace(state.APIURL), "/")
	state.SessionToken = strings.TrimSpace(state.SessionToken)
	state.CSRFToken = strings.TrimSpace(state.CSRFToken)
	state.CurrentAuthMethod = strings.TrimSpace(state.CurrentAuthMethod)
	if state.SessionToken == "" {
		return fmt.Errorf("session token must not be empty")
	}
	if state.SavedAt == "" {
		state.SavedAt = time.Now().UTC().Format(time.RFC3339)
	}
	// #nosec G117 -- persisting the CLI human session file is intentional; it is stored with 0600 permissions.
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal human session state: %w", err)
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create human session directory: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write human session state: %w", err)
	}
	return nil
}

func removeHumanSessionState(path string) error {
	err := os.Remove(filepath.Clean(path))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove human session state: %w", err)
	}
	return nil
}

func apiBaseURLFromControlPlaneURL(base string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", fmt.Errorf("parse control-plane url: %w", err)
	}
	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case path == "":
		parsed.Path = "/api/v1"
	case strings.HasSuffix(path, "/api/v1/platform"):
		parsed.Path = strings.TrimSuffix(path, "/platform")
	case strings.HasSuffix(path, "/api/v1"):
		parsed.Path = path
	default:
		parsed.Path = path + "/api/v1"
	}
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func originFromAPIURL(apiURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(apiURL))
	if err != nil {
		return "", fmt.Errorf("parse api url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("api url must include scheme and host")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}
