package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/logging"
)

type Executor interface {
	RunAll(ctx context.Context, hookName TicketHookName, hooks []Definition, env Env) ([]Result, error)
}

var hookShellExecutorComponent = logging.DeclareComponent("hook-shell-executor")
var hookShellExecutorLogger = logging.WithComponent(nil, hookShellExecutorComponent)

type ShellExecutor struct{}

func NewShellExecutor() Executor {
	return &ShellExecutor{}
}

func (e *ShellExecutor) RunAll(ctx context.Context, hookName TicketHookName, hooks []Definition, env Env) ([]Result, error) {
	results := make([]Result, 0, len(hooks))
	for _, hook := range hooks {
		if hook.OnFailure == "" {
			hook.OnFailure = FailurePolicyBlock
		}

		result, err := e.run(ctx, hookName, hook, env)
		results = append(results, result)
		if err == nil {
			continue
		}

		switch hook.OnFailure {
		case FailurePolicyWarn, FailurePolicyIgnore:
			continue
		default:
			return results, fmt.Errorf("%w: %s command %q failed: %v", ErrExecutionBlocked, hookName, hook.Command, err)
		}
	}

	return results, nil
}

func (e *ShellExecutor) run(ctx context.Context, hookName TicketHookName, hook Definition, env Env) (Result, error) {
	startedAt := time.Now()
	workingDirectory, err := resolveWorkingDirectory(env.Workspace, hook.Workdir)
	if err != nil {
		return Result{
			Name:             hook.Command,
			HookName:         hookName,
			Command:          hook.Command,
			WorkingDirectory: workingDirectory,
			Policy:           hook.OnFailure,
			Outcome:          OutcomeError,
			Duration:         time.Since(startedAt),
			Error:            err.Error(),
		}, err
	}

	commandContext := ctx
	cancel := func() {}
	if hook.Timeout > 0 {
		commandContext, cancel = context.WithTimeout(ctx, hook.Timeout)
	}
	defer cancel()

	//nolint:gosec // hook commands are an explicit repository-controlled execution surface
	cmd := exec.CommandContext(commandContext, "sh", "-c", hook.Command)
	cmd.Dir = workingDirectory
	cmd.Env = append(os.Environ(), buildEnvironmentVariables(hookName, env)...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	duration := time.Since(startedAt)
	result := Result{
		Name:             hook.Command,
		HookName:         hookName,
		Command:          hook.Command,
		WorkingDirectory: workingDirectory,
		Policy:           hook.OnFailure,
		Duration:         duration,
		Stdout:           strings.TrimSpace(stdout.String()),
		Stderr:           strings.TrimSpace(stderr.String()),
	}

	if errors.Is(commandContext.Err(), context.DeadlineExceeded) {
		result.Outcome = OutcomeTimeout
		result.TimedOut = true
		result.Error = fmt.Sprintf("timed out after %s", hook.Timeout)
		hookShellExecutorLogger.Warn("local hook timed out", "hook_name", hookName, "command", hook.Command, "working_directory", workingDirectory, "timeout", hook.Timeout.String())
		return result, errors.New(result.Error)
	}

	if runErr != nil {
		result.Outcome = OutcomeError
		if exitCode, ok := extractExitCode(runErr); ok {
			result.ExitCode = &exitCode
		}
		result.Error = describeRunError(runErr, result.Stderr)
		hookShellExecutorLogger.Warn("local hook execution failed", "hook_name", hookName, "command", hook.Command, "working_directory", workingDirectory, "duration_ms", duration.Milliseconds(), "stderr", result.Stderr, "error", runErr)
		return result, errors.New(result.Error)
	}

	result.Outcome = OutcomePass
	return result, nil
}

func buildEnvironmentVariables(hookName TicketHookName, env Env) []string {
	reposJSON := "[]"
	repos := env.Repos
	if repos == nil {
		repos = []Repo{}
	}
	if encoded, err := json.Marshal(repos); err == nil {
		reposJSON = string(encoded)
	}

	agentEnvironment := agentplatform.BuildEnvironment(env.APIURL, env.AgentToken, env.ProjectID, env.TicketID)
	environment := make([]string, 0, 8+len(agentEnvironment))
	environment = append(environment,
		"OPENASE_TICKET_IDENTIFIER="+env.TicketIdentifier,
		"OPENASE_WORKSPACE="+env.Workspace,
		"OPENASE_REPOS="+reposJSON,
		"OPENASE_AGENT_NAME="+env.AgentName,
		"OPENASE_WORKFLOW_TYPE="+env.WorkflowType,
		"OPENASE_WORKFLOW_FAMILY="+env.WorkflowFamily,
		"OPENASE_ATTEMPT="+fmt.Sprintf("%d", env.Attempt),
		"OPENASE_HOOK_NAME="+string(hookName),
	)
	environment = append(environment, agentEnvironment...)

	return environment
}

func resolveWorkingDirectory(workspace string, workdir string) (string, error) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return "", fmt.Errorf("workspace must not be empty")
	}

	base, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}

	target := base
	if strings.TrimSpace(workdir) != "" {
		if filepath.IsAbs(workdir) {
			target = filepath.Clean(workdir)
		} else {
			target = filepath.Join(base, filepath.FromSlash(workdir))
		}
	}

	target, err = filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve hook working directory: %w", err)
	}

	relativeToBase, err := filepath.Rel(base, target)
	if err != nil {
		return "", fmt.Errorf("resolve hook working directory: %w", err)
	}
	if relativeToBase == ".." || strings.HasPrefix(relativeToBase, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("workdir %q escapes workspace %q", workdir, base)
	}

	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("stat hook working directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("hook working directory %q is not a directory", target)
	}

	return target, nil
}

func extractExitCode(err error) (int, bool) {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 0, false
	}

	return exitErr.ExitCode(), true
}

func describeRunError(err error, stderr string) string {
	if strings.TrimSpace(stderr) != "" {
		return fmt.Sprintf("%v: %s", err, stderr)
	}

	return err.Error()
}
