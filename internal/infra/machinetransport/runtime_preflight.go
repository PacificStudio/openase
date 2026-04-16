package machinetransport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

const runtimePreflightFailurePrefix = "OPENASE_RUNTIME_PREFLIGHT_FAILURE"
const defaultRemoteOpenASEBinRelativePath = ".openase/bin/openase"

var _ = logging.DeclareComponent("machine-transport-runtime-preflight")

type RuntimePreflightStage string

const (
	RuntimePreflightStageTransport RuntimePreflightStage = "transport"
	RuntimePreflightStageWorkspace RuntimePreflightStage = "workspace"
	RuntimePreflightStageOpenASE   RuntimePreflightStage = "openase"
	RuntimePreflightStageAgentCLI  RuntimePreflightStage = "agent_cli"
)

type RuntimePreflightSpec struct {
	WorkingDirectory string
	AgentCommand     string
	Environment      []string
}

type RuntimePreflightError struct {
	Stage   RuntimePreflightStage
	Message string
	Cause   error
}

func (e *RuntimePreflightError) Error() string {
	if e == nil {
		return "remote runtime preflight failed"
	}

	stage := strings.TrimSpace(string(e.Stage))
	if stage == "" {
		stage = string(RuntimePreflightStageTransport)
	}
	message := strings.TrimSpace(e.Message)
	switch {
	case message != "":
		return fmt.Sprintf("remote runtime preflight (%s): %s", stage, message)
	case e.Cause != nil:
		return fmt.Sprintf("remote runtime preflight (%s): %v", stage, e.Cause)
	default:
		return fmt.Sprintf("remote runtime preflight (%s) failed", stage)
	}
}

func (e *RuntimePreflightError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func PrepareRemoteOpenASEEnvironment(
	ctx context.Context,
	commandExecutor CommandSessionExecution,
	artifactSync ArtifactSyncExecution,
	machine domain.Machine,
	environment []string,
) ([]string, error) {
	if commandExecutor == nil {
		return nil, fmt.Errorf("remote preflight command session unavailable for machine %s", machine.Name)
	}

	resolvedEnvironment := append([]string(nil), environment...)
	openaseBinPath := strings.TrimSpace(resolveEnvironmentValue(resolvedEnvironment, "OPENASE_REAL_BIN"))
	if openaseBinPath == "" {
		remoteHome, err := resolveRemoteHome(ctx, commandExecutor, machine)
		if err != nil {
			return nil, fmt.Errorf("resolve remote home for machine %s: %w", machine.Name, err)
		}
		openaseBinPath = filepath.ToSlash(filepath.Join(remoteHome, defaultRemoteOpenASEBinRelativePath))
		resolvedEnvironment = upsertEnvironmentValue(resolvedEnvironment, "OPENASE_REAL_BIN", openaseBinPath)
	}

	if err := ensureRemoteOpenASEBinary(ctx, commandExecutor, artifactSync, machine, openaseBinPath); err != nil {
		return nil, err
	}
	return resolvedEnvironment, nil
}

func RunRemoteRuntimePreflight(
	ctx context.Context,
	execution CommandSessionExecution,
	machine domain.Machine,
	spec RuntimePreflightSpec,
) error {
	if execution == nil {
		return &RuntimePreflightError{
			Stage:   RuntimePreflightStageTransport,
			Message: fmt.Sprintf("transport unavailable for machine %s", machine.Name),
		}
	}

	session, err := execution.OpenCommandSession(ctx, machine)
	if err != nil {
		return &RuntimePreflightError{
			Stage:   RuntimePreflightStageTransport,
			Message: fmt.Sprintf("open remote preflight session for machine %s", machine.Name),
			Cause:   err,
		}
	}
	defer func() { _ = session.Close() }()

	output, err := session.CombinedOutput(buildRemoteRuntimePreflightCommand(spec))
	if err == nil {
		return nil
	}
	if classified := parseRuntimePreflightFailure(string(output), err); classified != nil {
		return classified
	}
	return &RuntimePreflightError{
		Stage:   RuntimePreflightStageTransport,
		Message: strings.TrimSpace(string(output)),
		Cause:   err,
	}
}

func ensureRemoteOpenASEBinary(
	ctx context.Context,
	commandExecutor CommandSessionExecution,
	artifactSync ArtifactSyncExecution,
	machine domain.Machine,
	openaseBinPath string,
) error {
	trimmedPath := strings.TrimSpace(openaseBinPath)
	if trimmedPath == "" {
		return fmt.Errorf("remote OPENASE_REAL_BIN must not be empty for machine %s", machine.Name)
	}
	if remoteExecutableExists(ctx, commandExecutor, machine, trimmedPath) {
		return nil
	}
	if !filepath.IsAbs(trimmedPath) {
		return fmt.Errorf("remote OPENASE_REAL_BIN for machine %s must be absolute for auto-repair: %s", machine.Name, trimmedPath)
	}
	if artifactSync == nil {
		return fmt.Errorf("%w: artifact sync unavailable for machine %s", ErrTransportUnavailable, machine.Name)
	}

	localExecutable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve local openase executable for remote repair: %w", err)
	}
	if strings.TrimSpace(localExecutable) == "" {
		return fmt.Errorf("resolve local openase executable for remote repair: empty path")
	}

	tempRoot, err := os.MkdirTemp("", "openase-remote-bin-*")
	if err != nil {
		return fmt.Errorf("create remote repair temp root: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()

	targetName := filepath.Base(trimmedPath)
	localCopyPath := filepath.Join(tempRoot, targetName)
	// #nosec G304 -- localExecutable comes from os.Executable for the current process.
	content, err := os.ReadFile(localExecutable)
	if err != nil {
		return fmt.Errorf("read local openase executable for remote repair: %w", err)
	}
	// #nosec G306,G703 -- localCopyPath stays under the temp repair root and is chmodded next.
	if err := os.WriteFile(localCopyPath, content, 0o600); err != nil {
		return fmt.Errorf("stage local openase executable for remote repair: %w", err)
	}
	if err := os.Chmod(localCopyPath, 0o755); err != nil { //nolint:gosec // temp repair binary must remain executable for artifact sync.
		return fmt.Errorf("chmod staged openase executable for remote repair: %w", err)
	}
	if err := artifactSync.SyncArtifacts(ctx, machine, SyncArtifactsRequest{
		LocalRoot:  tempRoot,
		TargetRoot: filepath.Dir(trimmedPath),
		Paths:      []string{targetName},
	}); err != nil {
		return fmt.Errorf("sync remote openase executable for machine %s: %w", machine.Name, err)
	}
	if !remoteExecutableExists(ctx, commandExecutor, machine, trimmedPath) {
		return fmt.Errorf("remote openase executable repair for machine %s did not produce an executable binary at %s", machine.Name, trimmedPath)
	}
	return nil
}

func remoteExecutableExists(
	ctx context.Context,
	commandExecutor CommandSessionExecution,
	machine domain.Machine,
	path string,
) bool {
	output, err := runRemoteCommand(ctx, commandExecutor, machine, "test -x "+sshinfra.ShellQuote(path)+" && printf ok")
	return err == nil && strings.TrimSpace(output) == "ok"
}

func resolveRemoteHome(ctx context.Context, commandExecutor CommandSessionExecution, machine domain.Machine) (string, error) {
	output, err := runRemoteCommand(ctx, commandExecutor, machine, `printf %s "${HOME:-}"`)
	if err != nil {
		return "", err
	}
	remoteHome := strings.TrimSpace(output)
	if remoteHome == "" {
		return "", fmt.Errorf("remote HOME is empty")
	}
	return remoteHome, nil
}

func runRemoteCommand(
	ctx context.Context,
	commandExecutor CommandSessionExecution,
	machine domain.Machine,
	command string,
) (string, error) {
	session, err := commandExecutor.OpenCommandSession(ctx, machine)
	if err != nil {
		return "", err
	}
	defer func() { _ = session.Close() }()

	output, err := session.CombinedOutput("sh -lc " + sshinfra.ShellQuote(command))
	return string(output), err
}

func buildRemoteRuntimePreflightCommand(spec RuntimePreflightSpec) string {
	workingDirectory := strings.TrimSpace(spec.WorkingDirectory)
	agentCommand := strings.TrimSpace(spec.AgentCommand)
	envPrefix := buildRemotePreflightEnvPrefix(spec.Environment)
	wrapperPath := filepath.ToSlash(filepath.Join(".openase", "bin", "openase"))

	lines := []string{
		"set -eu",
		// Emit the classification marker on one stream only. Command-session CombinedOutput
		// merges stdout/stderr without preserving cross-stream line atomicity, so duplicating
		// the marker on both streams can interleave bytes and break parser classification.
		`fail() { stage="$1"; shift; printf '%s\n' "$*" >&2; printf '` + runtimePreflightFailurePrefix + `::%s::%s\n' "$stage" "$*"; exit 97; }`,
		"if [ ! -d " + sshinfra.ShellQuote(workingDirectory) + " ]; then fail workspace " + sshinfra.ShellQuote("working directory does not exist: "+workingDirectory) + "; fi",
		"cd " + sshinfra.ShellQuote(workingDirectory),
		"if [ ! -x " + sshinfra.ShellQuote(wrapperPath) + " ]; then fail openase " + sshinfra.ShellQuote("workspace openase wrapper is missing at "+filepath.ToSlash(filepath.Join(workingDirectory, wrapperPath))) + "; fi",
		"if ! " + envPrefix + " " + sshinfra.ShellQuote("./"+wrapperPath) + " version >/dev/null 2>&1; then fail openase " + sshinfra.ShellQuote("workspace openase wrapper could not resolve a runnable openase binary") + "; fi",
	}

	switch {
	case strings.Contains(agentCommand, "/"):
		lines = append(lines,
			"if [ ! -x "+sshinfra.ShellQuote(agentCommand)+" ]; then fail agent_cli "+sshinfra.ShellQuote("agent CLI path is not executable: "+agentCommand)+"; fi",
		)
	case agentCommand != "":
		lines = append(lines,
			"if ! "+envPrefix+" sh -lc "+sshinfra.ShellQuote("command -v "+sshinfra.ShellQuote(agentCommand)+" >/dev/null 2>&1")+"; then fail agent_cli "+sshinfra.ShellQuote("agent CLI command is not available on PATH: "+agentCommand)+"; fi",
		)
	default:
		lines = append(lines,
			"fail agent_cli "+sshinfra.ShellQuote("agent CLI command must not be empty"),
		)
	}

	return strings.Join(lines, "\n")
}

func resolveEnvironmentValue(environment []string, key string) string {
	value, _ := provider.LookupEnvironmentValue(environment, key)
	return value
}

func upsertEnvironmentValue(environment []string, key string, value string) []string {
	trimmedKey := strings.TrimSpace(key)
	prefix := trimmedKey + "="
	filtered := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(strings.TrimSpace(name), trimmedKey) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, prefix+value)
}

func buildRemotePreflightEnvPrefix(environment []string) string {
	parts := []string{"env"}
	for _, entry := range environment {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		parts = append(parts, sshinfra.ShellQuote(trimmed))
	}
	return strings.Join(parts, " ")
}

func parseRuntimePreflightFailure(output string, cause error) error {
	message := strings.TrimSpace(output)
	for _, line := range strings.Split(message, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, runtimePreflightFailurePrefix+"::") {
			continue
		}
		parts := strings.SplitN(trimmed, "::", 3)
		if len(parts) != 3 {
			continue
		}
		return &RuntimePreflightError{
			Stage:   RuntimePreflightStage(strings.TrimSpace(parts[1])),
			Message: strings.TrimSpace(parts[2]),
			Cause:   cause,
		}
	}
	return nil
}
