package machinetransport

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/logging"
)

const runtimePreflightFailurePrefix = "OPENASE_RUNTIME_PREFLIGHT_FAILURE"

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

func RunRemoteRuntimePreflight(
	ctx context.Context,
	transport Transport,
	machine domain.Machine,
	spec RuntimePreflightSpec,
) error {
	if transport == nil {
		return &RuntimePreflightError{
			Stage:   RuntimePreflightStageTransport,
			Message: fmt.Sprintf("transport unavailable for machine %s", machine.Name),
		}
	}

	session, err := transport.OpenCommandSession(ctx, machine)
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

func buildRemoteRuntimePreflightCommand(spec RuntimePreflightSpec) string {
	workingDirectory := strings.TrimSpace(spec.WorkingDirectory)
	agentCommand := strings.TrimSpace(spec.AgentCommand)
	envPrefix := buildRemotePreflightEnvPrefix(spec.Environment)
	wrapperPath := filepath.ToSlash(filepath.Join(".openase", "bin", "openase"))

	lines := []string{
		"set -eu",
		`fail() { stage="$1"; shift; printf '` + runtimePreflightFailurePrefix + `::%s::%s\n' "$stage" "$*" >&2; exit 97; }`,
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
