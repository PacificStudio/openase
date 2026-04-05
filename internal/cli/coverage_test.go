package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCLIExitAndPlatformHelpers(t *testing.T) {
	t.Parallel()

	exitErr := newExitError(7, "boom")
	typed, ok := exitErr.(interface {
		error
		ExitCode() int
	})
	if !ok {
		t.Fatalf("newExitError() type = %#v, want exit code interface", exitErr)
	}
	if typed.Error() != "boom" || typed.ExitCode() != 7 {
		t.Fatalf("newExitError() = %#v", typed)
	}

	platform := platformContext{
		projectID: "project-env",
		ticketID:  "ticket-env",
	}
	listInput, err := platform.parseTicketListInput(ticketListInput{
		statusNames: []string{" Todo ", "", "In Review"},
		priorities:  []string{" high ", "", "medium"},
	})
	if err != nil {
		t.Fatalf("parseTicketListInput() error = %v", err)
	}
	if listInput.projectID != "project-env" || strings.Join(listInput.statusNames, ",") != "Todo,In Review" || strings.Join(listInput.priorities, ",") != "high,medium" {
		t.Fatalf("parseTicketListInput() = %+v", listInput)
	}
	if _, err := (platformContext{}).parseTicketListInput(ticketListInput{}); err == nil || !strings.Contains(err.Error(), "project id is required") {
		t.Fatalf("parseTicketListInput() missing project error = %v", err)
	}

	projectInput, err := platform.parseProjectUpdateInput(projectUpdateInput{
		description:    " raise coverage ",
		descriptionSet: true,
	})
	if err != nil {
		t.Fatalf("parseProjectUpdateInput() error = %v", err)
	}
	if projectInput.projectID != "project-env" || projectInput.description != "raise coverage" || !projectInput.descriptionSet {
		t.Fatalf("parseProjectUpdateInput() = %+v", projectInput)
	}
	if _, err := platform.parseProjectUpdateInput(projectUpdateInput{}); err == nil || !strings.Contains(err.Error(), "at least one of --name, --slug, --description") {
		t.Fatalf("parseProjectUpdateInput() missing fields error = %v", err)
	}

	usageInput, err := platform.parseTicketReportUsageInput(ticketReportUsageInput{
		inputTokens:     int64PointerWhen(true, 12),
		outputTokens:    int64PointerWhen(true, 4),
		costUSD:         float64PointerWhen(true, 0.21),
		inputTokensSet:  true,
		outputTokensSet: true,
		costUSDSet:      true,
	})
	if err != nil {
		t.Fatalf("parseTicketReportUsageInput() error = %v", err)
	}
	if usageInput.ticketID != "ticket-env" || usageInput.inputTokens == nil || *usageInput.inputTokens != 12 || usageInput.costUSD == nil || *usageInput.costUSD != 0.21 {
		t.Fatalf("parseTicketReportUsageInput() = %+v", usageInput)
	}
	if _, err := platform.parseTicketReportUsageInput(ticketReportUsageInput{ticketID: "ticket-env"}); err == nil || !strings.Contains(err.Error(), "at least one of --input-tokens, --output-tokens, or --cost-usd must be set") {
		t.Fatalf("parseTicketReportUsageInput() missing values error = %v", err)
	}
	if _, err := platform.parseTicketReportUsageInput(ticketReportUsageInput{
		ticketID:       "ticket-env",
		inputTokens:    int64PointerWhen(true, -1),
		inputTokensSet: true,
	}); err == nil || !strings.Contains(err.Error(), "input-tokens must be greater than or equal to zero") {
		t.Fatalf("parseTicketReportUsageInput() negative input error = %v", err)
	}

	if got := extractPlatformErrorMessage([]byte(`{"message":"boom"}`)); got != "boom" {
		t.Fatalf("extractPlatformErrorMessage(message) = %q", got)
	}
	if got := extractPlatformErrorMessage([]byte(`{"error":"kaput"}`)); got != "kaput" {
		t.Fatalf("extractPlatformErrorMessage(error) = %q", got)
	}
	if got := extractPlatformErrorMessage([]byte(" plain failure ")); got != "plain failure" {
		t.Fatalf("extractPlatformErrorMessage(plain) = %q", got)
	}

	var invalidJSON bytes.Buffer
	if err := writePrettyJSON(&invalidJSON, []byte("no-json")); err != nil {
		t.Fatalf("writePrettyJSON(invalid) error = %v", err)
	}
	if invalidJSON.String() != "no-json\n" {
		t.Fatalf("writePrettyJSON(invalid) = %q", invalidJSON.String())
	}

	var invalidWithNewline bytes.Buffer
	if err := writePrettyJSON(&invalidWithNewline, []byte("no-json\n")); err != nil {
		t.Fatalf("writePrettyJSON(invalid newline) error = %v", err)
	}
	if invalidWithNewline.String() != "no-json\n" {
		t.Fatalf("writePrettyJSON(invalid newline) = %q", invalidWithNewline.String())
	}
}

func TestCLIManagedServiceHelpers(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cwd := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir(%q) error = %v", cwd, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	configPath := filepath.Join(cwd, "config.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  mode: all-in-one\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	resolvedConfig, err := resolveManagedServiceConfigPath("")
	if err != nil {
		t.Fatalf("resolveManagedServiceConfigPath() cwd error = %v", err)
	}
	if resolvedConfig.String() != configPath {
		t.Fatalf("resolveManagedServiceConfigPath() = %q, want %q", resolvedConfig, configPath)
	}

	spec, err := buildManagedServiceInstallSpec(configPath)
	if err != nil {
		t.Fatalf("buildManagedServiceInstallSpec() error = %v", err)
	}
	if spec.Name != managedServiceName || spec.Description != managedServiceDescription {
		t.Fatalf("buildManagedServiceInstallSpec() = %+v", spec)
	}
	if strings.Join(spec.Arguments, " ") != "all-in-one --config "+configPath {
		t.Fatalf("install spec args = %v", spec.Arguments)
	}
	if spec.WorkingDirectory.String() != filepath.Join(homeDir, ".openase") {
		t.Fatalf("working directory = %q", spec.WorkingDirectory)
	}
	if spec.EnvironmentFile.String() != filepath.Join(homeDir, ".openase", ".env") {
		t.Fatalf("environment file = %q", spec.EnvironmentFile)
	}
	if !filepath.IsAbs(spec.ProgramPath.String()) {
		t.Fatalf("program path should be absolute, got %q", spec.ProgramPath)
	}

	if _, err := resolveManagedServiceConfigPath(cwd); err == nil || !strings.Contains(err.Error(), "must be a file") {
		t.Fatalf("resolveManagedServiceConfigPath(dir) error = %v", err)
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatalf("Remove(%q) error = %v", configPath, err)
	}
	homeConfigDir := filepath.Join(homeDir, ".openase")
	if err := os.MkdirAll(homeConfigDir, 0o750); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", homeConfigDir, err)
	}
	homeConfigPath := filepath.Join(homeConfigDir, "config.toml")
	if err := os.WriteFile(homeConfigPath, []byte("server = {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", homeConfigPath, err)
	}
	resolvedHomeConfig, err := resolveManagedServiceConfigPath("")
	if err != nil {
		t.Fatalf("resolveManagedServiceConfigPath() home error = %v", err)
	}
	if resolvedHomeConfig.String() != homeConfigPath {
		t.Fatalf("resolveManagedServiceConfigPath() home = %q, want %q", resolvedHomeConfig, homeConfigPath)
	}
}

func TestCLIOpenAPIGenerateCommandWritesJSON(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "nested", "openapi.json")
	command := newOpenAPICommand()
	command.SetArgs([]string{"generate", "--output", outputPath})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	body, err := readCLITestFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
	}
	if !json.Valid(body) {
		t.Fatalf("generated OpenAPI JSON is invalid: %q", string(body))
	}
	if !bytes.Contains(body, []byte(`"openapi"`)) {
		t.Fatalf("generated OpenAPI JSON missing version field: %q", string(body))
	}
}

func TestCLIOpenAPICLIContractCommandWritesJSON(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "nested", "openapi-cli-contract.json")
	command := newOpenAPICommand()
	command.SetArgs([]string{"cli-contract", "--output", outputPath})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	body, err := readCLITestFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
	}
	if !json.Valid(body) {
		t.Fatalf("generated CLI contract JSON is invalid: %q", string(body))
	}
	if !bytes.Contains(body, []byte(`"openapi_sha256"`)) || !bytes.Contains(body, []byte(`"commands"`)) {
		t.Fatalf("generated CLI contract JSON missing expected fields: %q", string(body))
	}
}

func TestCLIWriteIssueAgentTokenShellAndJSONHelpers(t *testing.T) {
	t.Parallel()

	projectID := mustParseUUIDForCLI(t, "550e8400-e29b-41d4-a716-446655440000")
	ticketID := mustParseUUIDForCLI(t, "660e8400-e29b-41d4-a716-446655440000")
	expiresAt := time.Date(2026, 3, 27, 15, 0, 0, 0, time.UTC)

	environment := buildIssueAgentTokenEnvironment("http://127.0.0.1:19836/api/v1/platform", "ase_agent_test", projectID, ticketID, expiresAt)
	if environment["OPENASE_AGENT_TOKEN"] != "ase_agent_test" || environment["OPENASE_PROJECT_ID"] != projectID.String() {
		t.Fatalf("buildIssueAgentTokenEnvironment() = %+v", environment)
	}

	var shell bytes.Buffer
	if err := writeIssueAgentTokenShell(&shell, environment); err != nil {
		t.Fatalf("writeIssueAgentTokenShell() error = %v", err)
	}
	for _, want := range []string{
		`export OPENASE_API_URL="http://127.0.0.1:19836/api/v1/platform"`,
		`export OPENASE_AGENT_TOKEN="ase_agent_test"`,
		`export OPENASE_PROJECT_ID="550e8400-e29b-41d4-a716-446655440000"`,
		`export OPENASE_TICKET_ID="660e8400-e29b-41d4-a716-446655440000"`,
		`export OPENASE_AGENT_EXPIRES_AT="2026-03-27T15:00:00Z"`,
	} {
		if !strings.Contains(shell.String(), want) {
			t.Fatalf("shell output %q missing %q", shell.String(), want)
		}
	}

	var pretty bytes.Buffer
	if err := writeIssueAgentTokenJSON(context.Background(), &pretty, issueAgentTokenResponse{
		Token:       "ase_agent_test",
		ProjectID:   projectID.String(),
		TicketID:    ticketID.String(),
		Environment: environment,
	}); err != nil {
		t.Fatalf("writeIssueAgentTokenJSON() error = %v", err)
	}
	if !strings.Contains(pretty.String(), `"token": "ase_agent_test"`) {
		t.Fatalf("writeIssueAgentTokenJSON() = %q", pretty.String())
	}
}

func TestCLIPlatformParserAndHTTPErrorCoverage(t *testing.T) {
	platformToken := " " + "ase_agent_" + "test "
	platform, err := rawPlatformContext{
		apiURL:    " http://127.0.0.1:19836/api/v1/platform/ ",
		token:     platformToken,
		projectID: " project-123 ",
		ticketID:  " ticket-123 ",
	}.resolve()
	if err != nil {
		t.Fatalf("rawPlatformContext.resolve() error = %v", err)
	}
	if platform.apiURL != "http://127.0.0.1:19836/api/v1/platform" || platform.token != "ase_agent_test" {
		t.Fatalf("rawPlatformContext.resolve() = %+v", platform)
	}

	t.Setenv("OPENASE_API_URL", "")
	t.Setenv("OPENASE_AGENT_TOKEN", "")
	if _, err := (rawPlatformContext{}).resolve(); err == nil || !strings.Contains(err.Error(), "platform api url is required") {
		t.Fatalf("rawPlatformContext.resolve() missing api url error = %v", err)
	}

	createInput, err := platform.parseTicketCreateInput(ticketCreateInput{
		title:       " Raise backend coverage ",
		description: " finish CI gate ",
		priority:    " high ",
		typeName:    " bug ",
		externalRef: " PacificStudio/openase#278 ",
	})
	if err != nil {
		t.Fatalf("parseTicketCreateInput() error = %v", err)
	}
	if createInput.title != "Raise backend coverage" || createInput.typeName != "bug" || createInput.externalRef != "PacificStudio/openase#278" {
		t.Fatalf("parseTicketCreateInput() = %+v", createInput)
	}
	if _, err := platform.parseTicketCreateInput(ticketCreateInput{projectID: "project-123", title: " "}); err == nil || !strings.Contains(err.Error(), "title must not be empty") {
		t.Fatalf("parseTicketCreateInput() blank title error = %v", err)
	}

	updateInput, err := platform.parseTicketUpdateInput(ticketUpdateInput{
		title:          " tighten gate ",
		description:    " ship coverage ",
		externalRef:    " PacificStudio/openase#278 ",
		titleSet:       true,
		descriptionSet: true,
		externalRefSet: true,
	})
	if err != nil {
		t.Fatalf("parseTicketUpdateInput() error = %v", err)
	}
	if updateInput.ticketID != "ticket-123" || updateInput.title != "tighten gate" || updateInput.externalRef != "PacificStudio/openase#278" {
		t.Fatalf("parseTicketUpdateInput() = %+v", updateInput)
	}
	if _, err := platform.parseTicketUpdateInput(ticketUpdateInput{ticketID: "ticket-123"}); err == nil || !strings.Contains(err.Error(), "at least one of --title, --description, --external-ref, --status, --status-name, --status-id, --priority, --type, --workflow-id, --parent-ticket-id, --budget-usd, or --archived must be set") {
		t.Fatalf("parseTicketUpdateInput() missing fields error = %v", err)
	}

	addRepoInput, err := platform.parseProjectAddRepoInput(projectAddRepoInput{
		name:          " backend ",
		repositoryURL: " https://github.com/acme/backend.git ",
		defaultBranch: " ",
		labels:        []string{" go ", "", " backend "},
	})
	if err != nil {
		t.Fatalf("parseProjectAddRepoInput() error = %v", err)
	}
	if addRepoInput.defaultBranch != "main" || strings.Join(addRepoInput.labels, ",") != "go,backend" {
		t.Fatalf("parseProjectAddRepoInput() = %+v", addRepoInput)
	}
	if _, err := platform.parseProjectAddRepoInput(projectAddRepoInput{projectID: "project-123", repositoryURL: "https://github.com/acme/backend.git"}); err == nil || !strings.Contains(err.Error(), "name must not be empty") {
		t.Fatalf("parseProjectAddRepoInput() missing name error = %v", err)
	}

	if got := firstArg(nil); got != "" {
		t.Fatalf("firstArg(nil) = %q", got)
	}
	if got := int64PointerWhen(false, 3); got != nil {
		t.Fatalf("int64PointerWhen(false) = %v", got)
	}
	if got := float64PointerWhen(false, 0.5); got != nil {
		t.Fatalf("float64PointerWhen(false) = %v", got)
	}

	client := platformClient{httpClient: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed")
	})}
	if _, err := client.doJSON(context.Background(), platform, http.MethodGet, "/tickets", nil); err == nil || !strings.Contains(err.Error(), "GET /tickets: dial failed") {
		t.Fatalf("doJSON(network) error = %v", err)
	}

	client = platformClient{httpClient: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(strings.NewReader(`{"message":"bad request"}`)),
			Header:     make(http.Header),
		}, nil
	})}
	if _, err := client.doJSON(context.Background(), platform, http.MethodGet, "/tickets", nil); err == nil || !strings.Contains(err.Error(), "returned 400 Bad Request: bad request") {
		t.Fatalf("doJSON(status) error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func mustParseUUIDForCLI(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	return uuid.MustParse(strings.TrimSpace(raw))
}

func readCLITestFile(path string) ([]byte, error) {
	//nolint:gosec // Tests read files they created under temp directories.
	return os.ReadFile(path)
}
