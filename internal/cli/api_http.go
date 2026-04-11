package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/pflag"
)

const defaultAPIURL = "http://127.0.0.1:19836/api/v1"

type apiCommandDeps struct {
	httpClient platformHTTPDoer
}

type apiCommandOptions struct {
	apiURL            string
	token             string
	humanSessionFile  string
	humanSessionToken string
	humanCSRFToken    string
}

type apiCommandContext struct {
	apiURL                string
	token                 string
	humanSession          *humanSessionState
	humanSessionStatePath string
}

type apiRequest struct {
	Method  string
	Path    string
	Body    []byte
	Headers http.Header
}

type apiStreamRequest struct {
	Method  string
	Path    string
	Headers http.Header
}

type apiResponse struct {
	StatusCode int
	Status     string
	Body       []byte
}

type apiHTTPError struct {
	Method     string
	Path       string
	StatusCode int
	Status     string
	Code       string
	Message    string
}

func (err *apiHTTPError) Error() string {
	parts := []string{fmt.Sprintf("%s %s returned %s", err.Method, err.Path, err.Status)}
	if strings.TrimSpace(err.Code) != "" {
		parts = append(parts, fmt.Sprintf("[%s]", strings.TrimSpace(err.Code)))
	}
	if strings.TrimSpace(err.Message) != "" {
		parts = append(parts, strings.TrimSpace(err.Message))
	}
	return strings.Join(parts, " ")
}

func bindAPICommandFlags(flags *pflag.FlagSet, options *apiCommandOptions) {
	flags.StringVar(&options.apiURL, "api-url", "", "API base URL override. Defaults to OPENASE_API_URL or "+defaultAPIURL+".")
	flags.StringVar(&options.token, "token", "", "Bearer token override. Defaults to OPENASE_AGENT_TOKEN.")
	flags.StringVar(&options.humanSessionFile, "session-file", "", "Human session state file override. Defaults to "+defaultHumanSessionStateRelativePath+".")
	flags.StringVar(&options.humanSessionToken, "session-token", "", "Human session token override. Defaults to "+envHumanSessionToken+".")
	flags.StringVar(&options.humanCSRFToken, "csrf-token", "", "Human session CSRF token override. Defaults to "+envHumanCSRFToken+".")
}

func (options apiCommandOptions) resolve() (apiCommandContext, error) {
	return options.resolveWithResourceBase(false)
}

func (options apiCommandOptions) resolveResource() (apiCommandContext, error) {
	return options.resolveWithResourceBase(true)
}

func normalizeResourceAPIBaseURL(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return strings.TrimSpace(baseURL)
	}
	if strings.HasSuffix(parsed.Path, "/api/v1/platform") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/platform")
		parsed.RawPath = ""
	}
	return parsed.String()
}

func (ctx apiCommandContext) do(ctx2 context.Context, deps apiCommandDeps, request apiRequest) (apiResponse, error) {
	requestURL, err := buildRequestURL(ctx.apiURL, request.Path)
	if err != nil {
		return apiResponse{}, err
	}

	var body io.Reader
	if len(request.Body) > 0 {
		body = bytes.NewReader(request.Body)
	}

	httpRequest, err := http.NewRequestWithContext(ctx2, request.Method, requestURL, body)
	if err != nil {
		return apiResponse{}, fmt.Errorf("build %s %s request: %w", request.Method, request.Path, err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	if httpRequest.Header.Get("User-Agent") == "" {
		httpRequest.Header.Set("User-Agent", openASECLIUserAgent)
	}
	if ctx.token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+ctx.token)
	}
	for key, values := range request.Headers {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}
	if len(request.Body) > 0 && httpRequest.Header.Get("Content-Type") == "" {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	if err := ctx.attachHumanSessionHeaders(httpRequest); err != nil {
		return apiResponse{}, err
	}

	response, err := deps.httpClient.Do(httpRequest)
	if err != nil {
		return apiResponse{}, fmt.Errorf("%s %s: %w", request.Method, request.Path, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return apiResponse{}, fmt.Errorf("read %s %s response: %w", request.Method, request.Path, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		code, message := parseAPIErrorBody(payload)
		return apiResponse{}, &apiHTTPError{
			Method:     request.Method,
			Path:       request.Path,
			StatusCode: response.StatusCode,
			Status:     response.Status,
			Code:       code,
			Message:    message,
		}
	}

	return apiResponse{
		StatusCode: response.StatusCode,
		Status:     response.Status,
		Body:       payload,
	}, nil
}

func (ctx apiCommandContext) stream(ctx2 context.Context, deps apiCommandDeps, request apiStreamRequest, out io.Writer) error {
	requestURL, err := buildRequestURL(ctx.apiURL, request.Path)
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(ctx2, request.Method, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build %s %s request: %w", request.Method, request.Path, err)
	}
	httpRequest.Header.Set("Accept", "text/event-stream")
	if httpRequest.Header.Get("User-Agent") == "" {
		httpRequest.Header.Set("User-Agent", openASECLIUserAgent)
	}
	if ctx.token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+ctx.token)
	}
	for key, values := range request.Headers {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}
	if err := ctx.attachHumanSessionHeaders(httpRequest); err != nil {
		return err
	}

	response, err := deps.httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("%s %s: %w", request.Method, request.Path, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return fmt.Errorf("read %s %s error response: %w", request.Method, request.Path, readErr)
		}
		code, message := parseAPIErrorBody(payload)
		return &apiHTTPError{
			Method:     request.Method,
			Path:       request.Path,
			StatusCode: response.StatusCode,
			Status:     response.Status,
			Code:       code,
			Message:    message,
		}
	}

	_, err = io.Copy(out, response.Body)
	return err
}

func buildRequestURL(baseURL string, targetPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse api url: %w", err)
	}

	targetPath = strings.TrimSpace(targetPath)
	switch {
	case targetPath == "":
		return "", fmt.Errorf("request path must not be empty")
	case strings.HasPrefix(targetPath, "http://") || strings.HasPrefix(targetPath, "https://"):
		parsed, parseErr := url.Parse(targetPath)
		if parseErr != nil {
			return "", fmt.Errorf("parse request path: %w", parseErr)
		}
		return parsed.String(), nil
	case strings.HasPrefix(targetPath, "/"):
		ref, parseErr := url.Parse(targetPath)
		if parseErr != nil {
			return "", fmt.Errorf("parse request path: %w", parseErr)
		}
		base.Path = ref.Path
		base.RawPath = ref.RawPath
		base.RawQuery = ref.RawQuery
	default:
		ref, parseErr := url.Parse(targetPath)
		if parseErr != nil {
			return "", fmt.Errorf("parse request path: %w", parseErr)
		}
		prefix := strings.TrimSuffix(base.Path, "/")
		base.Path = prefix + "/" + strings.TrimPrefix(ref.Path, "/")
		base.RawPath = ""
		base.RawQuery = ref.RawQuery
	}

	return base.String(), nil
}

func (options apiCommandOptions) resolveHumanSessionState() (*humanSessionState, string, error) {
	explicitSessionToken := strings.TrimSpace(firstNonEmpty(options.humanSessionToken, os.Getenv(envHumanSessionToken)))
	explicitCSRFToken := strings.TrimSpace(firstNonEmpty(options.humanCSRFToken, os.Getenv(envHumanCSRFToken)))
	if explicitSessionToken != "" {
		return &humanSessionState{
			SessionToken: explicitSessionToken,
			CSRFToken:    explicitCSRFToken,
		}, "", nil
	}

	sessionPath, err := resolveHumanSessionStatePath(options.humanSessionFile)
	if err != nil {
		return nil, "", err
	}
	state, err := loadHumanSessionState(sessionPath)
	if err != nil {
		if errors.Is(err, errHumanSessionStateNotFound) {
			return nil, sessionPath, nil
		}
		return nil, "", err
	}
	return &state, sessionPath, nil
}

func (options apiCommandOptions) resolveWithResourceBase(normalizeResourceBase bool) (apiCommandContext, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(firstNonEmpty(options.apiURL, os.Getenv("OPENASE_API_URL"), defaultAPIURL)), "/")
	if baseURL == "" {
		return apiCommandContext{}, fmt.Errorf("api url must not be empty")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return apiCommandContext{}, fmt.Errorf("parse api url: %w", err)
	}
	if normalizeResourceBase {
		baseURL = normalizeResourceAPIBaseURL(baseURL)
	}

	token := strings.TrimSpace(firstNonEmpty(options.token, os.Getenv("OPENASE_AGENT_TOKEN")))
	context := apiCommandContext{
		apiURL: strings.TrimRight(baseURL, "/"),
		token:  token,
	}
	if token != "" {
		return context, nil
	}

	state, sessionPath, err := options.resolveHumanSessionState()
	if err != nil {
		return apiCommandContext{}, err
	}
	context.humanSession = state
	context.humanSessionStatePath = sessionPath
	return context, nil
}

func (ctx apiCommandContext) attachHumanSessionHeaders(request *http.Request) error {
	if ctx.token != "" || ctx.humanSession == nil || request == nil {
		return nil
	}
	if strings.TrimSpace(request.Header.Get("Authorization")) != "" || strings.TrimSpace(request.Header.Get("Cookie")) != "" {
		return nil
	}
	request.Header.Set("Cookie", humanSessionCookieHeaderName+"="+ctx.humanSession.SessionToken)
	if !requestMethodNeedsCSRFAuth(request.Method) {
		return nil
	}
	if strings.TrimSpace(ctx.humanSession.CSRFToken) == "" {
		return fmt.Errorf("human session csrf token is missing; rerun `openase auth bootstrap login`")
	}
	if request.Header.Get("X-OpenASE-CSRF") == "" {
		request.Header.Set("X-OpenASE-CSRF", ctx.humanSession.CSRFToken)
	}
	if request.Header.Get("Origin") == "" && request.Header.Get("Referer") == "" {
		origin, err := originFromAPIURL(ctx.apiURL)
		if err != nil {
			return err
		}
		request.Header.Set("Origin", origin)
	}
	return nil
}

func requestMethodNeedsCSRFAuth(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func parseAPIErrorBody(body []byte) (string, string) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", strings.TrimSpace(string(body))
	}

	code := strings.TrimSpace(fmt.Sprint(payload["code"]))
	if code == "<nil>" {
		code = ""
	}
	for _, key := range []string{"message", "error"} {
		if message := strings.TrimSpace(fmt.Sprint(payload[key])); message != "" && message != "<nil>" {
			return code, message
		}
	}

	return code, strings.TrimSpace(string(body))
}

func envVarForParameter(name string) string {
	var builder strings.Builder
	builder.WriteString("OPENASE_")
	for index, r := range name {
		switch {
		case r == '-':
			builder.WriteByte('_')
		case r == '_':
			builder.WriteByte('_')
		case unicode.IsUpper(r):
			if index > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(unicode.ToUpper(r))
		default:
			builder.WriteRune(unicode.ToUpper(r))
		}
	}
	return builder.String()
}
