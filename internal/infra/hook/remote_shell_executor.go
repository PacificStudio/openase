package hook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/logging"
	gossh "golang.org/x/crypto/ssh"
)

const remoteInterruptSignal = "INT"

var hookRemoteShellExecutorComponent = logging.DeclareComponent("hook-remote-shell-executor")
var hookRemoteShellExecutorLogger = logging.WithComponent(nil, hookRemoteShellExecutorComponent)

type remoteSessionFactory interface {
	OpenCommandSession(ctx context.Context, machine catalogdomain.Machine) (machinetransport.CommandSession, error)
}

type RemoteShellExecutor struct {
	pool    remoteSessionFactory
	machine catalogdomain.Machine
}

func NewRemoteShellExecutor(pool remoteSessionFactory, machine catalogdomain.Machine) Executor {
	return &RemoteShellExecutor{
		pool:    pool,
		machine: machine,
	}
}

func (e *RemoteShellExecutor) RunAll(ctx context.Context, hookName TicketHookName, hooks []Definition, env Env) ([]Result, error) {
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

func (e *RemoteShellExecutor) run(ctx context.Context, hookName TicketHookName, hook Definition, env Env) (Result, error) {
	startedAt := time.Now()
	workingDirectory, err := resolveRemoteWorkingDirectory(env.Workspace, hook.Workdir)
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

	session, err := e.pool.OpenCommandSession(commandContext, e.machine)
	if err != nil {
		hookRemoteShellExecutorLogger.Warn("acquire remote hook session failed", "hook_name", hookName, "command", hook.Command, "machine_id", e.machine.ID.String(), "machine_name", e.machine.Name, "host", e.machine.Host, "error", err)
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

	defer func() {
		_ = session.Close()
	}()

	stdoutPipe, err := session.StdoutPipe()
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
	stderrPipe, err := session.StderrPipe()
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdout, stdoutPipe)
		close(stdoutDone)
	}()
	go func() {
		_, _ = io.Copy(&stderr, stderrPipe)
		close(stderrDone)
	}()

	commandText := buildRemoteCommand(workingDirectory, hook.Command, buildEnvironmentVariables(hookName, env))
	if err := session.Start(commandText); err != nil {
		hookRemoteShellExecutorLogger.Warn("start remote hook command failed", "hook_name", hookName, "command", hook.Command, "machine_id", e.machine.ID.String(), "machine_name", e.machine.Name, "host", e.machine.Host, "working_directory", workingDirectory, "error", err)
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

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- session.Wait()
	}()

	runErr := error(nil)
	select {
	case runErr = <-waitCh:
	case <-commandContext.Done():
		_ = session.Signal(remoteInterruptSignal)
		_ = session.Close()
		runErr = commandContext.Err()
	}
	<-stdoutDone
	<-stderrDone

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
		hookRemoteShellExecutorLogger.Warn("remote hook timed out", "hook_name", hookName, "command", hook.Command, "machine_id", e.machine.ID.String(), "machine_name", e.machine.Name, "host", e.machine.Host, "working_directory", workingDirectory, "timeout", hook.Timeout.String())
		return result, errors.New(result.Error)
	}
	if errors.Is(commandContext.Err(), context.Canceled) && errors.Is(runErr, context.Canceled) {
		result.Outcome = OutcomeError
		result.Error = commandContext.Err().Error()
		return result, commandContext.Err()
	}
	if runErr != nil {
		result.Outcome = OutcomeError
		if exitCode, ok := extractRemoteExitCode(runErr); ok {
			result.ExitCode = &exitCode
		}
		result.Error = describeRunError(runErr, result.Stderr)
		hookRemoteShellExecutorLogger.Warn("remote hook execution failed", "hook_name", hookName, "command", hook.Command, "machine_id", e.machine.ID.String(), "machine_name", e.machine.Name, "host", e.machine.Host, "working_directory", workingDirectory, "duration_ms", duration.Milliseconds(), "stderr", result.Stderr, "error", runErr)
		return result, errors.New(result.Error)
	}

	result.Outcome = OutcomePass
	return result, nil
}

func resolveRemoteWorkingDirectory(workspace string, workdir string) (string, error) {
	base := strings.TrimSpace(workspace)
	if base == "" {
		return "", fmt.Errorf("workspace must not be empty")
	}
	if !path.IsAbs(base) {
		return "", fmt.Errorf("workspace %q must be an absolute path", workspace)
	}
	base = path.Clean(base)

	target := base
	if strings.TrimSpace(workdir) != "" {
		if path.IsAbs(workdir) {
			target = path.Clean(workdir)
		} else {
			target = path.Clean(path.Join(base, workdir))
		}
	}

	if target != base && !strings.HasPrefix(target, base+"/") {
		return "", fmt.Errorf("workdir %q escapes workspace %q", workdir, base)
	}

	return target, nil
}

func buildRemoteCommand(workingDirectory string, command string, environment []string) string {
	commandParts := make([]string, 0, 2+len(environment))
	commandParts = append(commandParts, "cd "+sshinfra.ShellQuote(workingDirectory))
	if len(environment) > 0 {
		envParts := make([]string, 0, len(environment))
		for _, entry := range environment {
			envParts = append(envParts, sshinfra.ShellQuote(entry))
		}
		commandParts = append(commandParts, "env "+strings.Join(envParts, " "))
	}
	commandParts = append(commandParts, "sh -lc "+sshinfra.ShellQuote(command))
	return strings.Join(commandParts, " && ")
}

func extractRemoteExitCode(err error) (int, bool) {
	var exitErr *gossh.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitStatus(), true
	}
	type exitStatusProvider interface {
		ExitStatus() int
	}
	var provider exitStatusProvider
	if errors.As(err, &provider) {
		return provider.ExitStatus(), true
	}
	return 0, false
}
