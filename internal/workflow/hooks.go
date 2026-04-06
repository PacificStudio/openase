package workflow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type workflowHookName string

const (
	workflowHookOnActivate workflowHookName = "on_activate"
	workflowHookOnReload   workflowHookName = "on_reload"
)

type workflowHookFailurePolicy string

const (
	workflowHookFailureBlock  workflowHookFailurePolicy = "block"
	workflowHookFailureWarn   workflowHookFailurePolicy = "warn"
	workflowHookFailureIgnore workflowHookFailurePolicy = "ignore"
)

type workflowHooksConfig struct {
	OnActivate []workflowHookDefinition
	OnReload   []workflowHookDefinition
}

type workflowHookDefinition struct {
	Command   string
	Timeout   time.Duration
	OnFailure workflowHookFailurePolicy
}

type workflowHookRuntime struct {
	ProjectID       uuid.UUID
	WorkflowID      uuid.UUID
	WorkflowName    string
	WorkflowVersion int
}

type workflowHookExecutor struct {
	logger           *slog.Logger
	workingDirectory string
}

var workflowHookTemplatePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

func newWorkflowHookExecutor(workingDirectory string, logger *slog.Logger) *workflowHookExecutor {
	return &workflowHookExecutor{
		logger:           logger.With("component", "workflow-hook-executor"),
		workingDirectory: workingDirectory,
	}
}

func parseWorkflowHooks(raw map[string]any) (workflowHooksConfig, error) {
	if len(raw) == 0 {
		return workflowHooksConfig{}, nil
	}

	workflowHooksRaw, ok := raw["workflow_hooks"]
	if !ok || workflowHooksRaw == nil {
		return workflowHooksConfig{}, nil
	}

	workflowHooksMap, ok := workflowHooksRaw.(map[string]any)
	if !ok {
		return workflowHooksConfig{}, fmt.Errorf("%w: hooks.workflow_hooks must be an object", ErrHookConfigInvalid)
	}

	onActivate, err := parseWorkflowHookList(workflowHooksMap, workflowHookOnActivate)
	if err != nil {
		return workflowHooksConfig{}, err
	}

	onReload, err := parseWorkflowHookList(workflowHooksMap, workflowHookOnReload)
	if err != nil {
		return workflowHooksConfig{}, err
	}

	return workflowHooksConfig{
		OnActivate: onActivate,
		OnReload:   onReload,
	}, nil
}

func parseWorkflowHookList(raw map[string]any, hookName workflowHookName) ([]workflowHookDefinition, error) {
	listRaw, ok := raw[string(hookName)]
	if !ok || listRaw == nil {
		return nil, nil
	}

	switch list := listRaw.(type) {
	case []any:
		return parseWorkflowHookEntries(list, "hooks.workflow_hooks."+string(hookName))
	case []map[string]any:
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, item)
		}
		return parseWorkflowHookEntries(items, "hooks.workflow_hooks."+string(hookName))
	default:
		return nil, fmt.Errorf("%w: hooks.workflow_hooks.%s must be a list", ErrHookConfigInvalid, hookName)
	}
}

func parseWorkflowHookEntries(entries []any, path string) ([]workflowHookDefinition, error) {
	hooks := make([]workflowHookDefinition, 0, len(entries))
	for index, entry := range entries {
		parsed, err := parseWorkflowHookDefinition(entry, path+"["+strconv.Itoa(index)+"]")
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, parsed)
	}

	return hooks, nil
}

func parseWorkflowHookDefinition(raw any, path string) (workflowHookDefinition, error) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return workflowHookDefinition{}, fmt.Errorf("%w: %s must be an object", ErrHookConfigInvalid, path)
	}

	commandRaw, ok := entry["cmd"]
	if !ok {
		return workflowHookDefinition{}, fmt.Errorf("%w: %s.cmd is required", ErrHookConfigInvalid, path)
	}

	command, ok := commandRaw.(string)
	if !ok || strings.TrimSpace(command) == "" {
		return workflowHookDefinition{}, fmt.Errorf("%w: %s.cmd must be a non-empty string", ErrHookConfigInvalid, path)
	}

	timeout, err := parseWorkflowHookTimeout(entry["timeout"], path+".timeout")
	if err != nil {
		return workflowHookDefinition{}, err
	}

	onFailure, err := parseWorkflowHookFailure(entry["on_failure"], path+".on_failure")
	if err != nil {
		return workflowHookDefinition{}, err
	}

	return workflowHookDefinition{
		Command:   strings.TrimSpace(command),
		Timeout:   timeout,
		OnFailure: onFailure,
	}, nil
}

func parseWorkflowHookTimeout(raw any, path string) (time.Duration, error) {
	if raw == nil {
		return 0, nil
	}

	var seconds int64
	switch value := raw.(type) {
	case int:
		seconds = int64(value)
	case int32:
		seconds = int64(value)
	case int64:
		seconds = value
	case float64:
		if math.Trunc(value) != value {
			return 0, fmt.Errorf("%w: %s must be a whole number of seconds", ErrHookConfigInvalid, path)
		}
		seconds = int64(value)
	default:
		return 0, fmt.Errorf("%w: %s must be a whole number of seconds", ErrHookConfigInvalid, path)
	}

	if seconds < 0 {
		return 0, fmt.Errorf("%w: %s must be greater than or equal to zero", ErrHookConfigInvalid, path)
	}

	return time.Duration(seconds) * time.Second, nil
}

func parseWorkflowHookFailure(raw any, path string) (workflowHookFailurePolicy, error) {
	if raw == nil {
		return workflowHookFailureBlock, nil
	}

	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s must be one of block, warn, ignore", ErrHookConfigInvalid, path)
	}

	switch policy := workflowHookFailurePolicy(strings.ToLower(strings.TrimSpace(value))); policy {
	case workflowHookFailureBlock, workflowHookFailureWarn, workflowHookFailureIgnore:
		return policy, nil
	default:
		return "", fmt.Errorf("%w: %s must be one of block, warn, ignore", ErrHookConfigInvalid, path)
	}
}

func (e *workflowHookExecutor) RunAll(ctx context.Context, hookName workflowHookName, hooks []workflowHookDefinition, runtime workflowHookRuntime) error {
	for _, hook := range hooks {
		startedAt := time.Now()
		baseAttrs := []any{
			"hook_name", hookName,
			"hook_scope", "workflow",
			"hook_policy", hook.OnFailure,
			"project_id", runtime.ProjectID,
			"workflow_id", runtime.WorkflowID,
			"workflow_name", runtime.WorkflowName,
			"workflow_version", runtime.WorkflowVersion,
			"command", hook.Command,
			"timeout_seconds", int(hook.Timeout / time.Second),
		}
		e.logger.Info("workflow hook started", baseAttrs...)

		if err := e.run(ctx, hookName, hook, runtime); err != nil {
			attrs := append(append([]any{}, baseAttrs...),
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"error", err,
			)

			switch hook.OnFailure {
			case workflowHookFailureIgnore:
				e.logger.Warn("workflow hook failed but was ignored", attrs...)
				continue
			case workflowHookFailureWarn:
				e.logger.Warn("workflow hook failed with warning policy", attrs...)
				continue
			default:
				e.logger.Error("workflow hook failed and blocked lifecycle", append(attrs, "blocking", true)...)
				return fmt.Errorf("%w: %s command %q failed: %v", ErrWorkflowHookBlocked, hookName, hook.Command, err)
			}
		}

		e.logger.Info("workflow hook completed", append(append([]any{}, baseAttrs...), "duration_ms", time.Since(startedAt).Milliseconds(), "outcome", "passed")...)
	}

	return nil
}

func (e *workflowHookExecutor) run(ctx context.Context, hookName workflowHookName, hook workflowHookDefinition, runtime workflowHookRuntime) error {
	commandText := renderWorkflowHookCommand(hook.Command, hookName, runtime)

	commandContext := ctx
	cancel := func() {}
	if hook.Timeout > 0 {
		commandContext, cancel = context.WithTimeout(ctx, hook.Timeout)
	}
	defer cancel()

	//nolint:gosec // workflow hooks intentionally execute repository-defined shell commands; runtime template values are shell-quoted before insertion.
	cmd := exec.CommandContext(commandContext, "sh", "-c", commandText)
	cmd.Dir = e.workingDirectory
	cmd.Env = append(os.Environ(),
		"OPENASE_PROJECT_ID="+runtime.ProjectID.String(),
		"OPENASE_WORKFLOW_ID="+runtime.WorkflowID.String(),
		"OPENASE_WORKFLOW_NAME="+runtime.WorkflowName,
		"OPENASE_WORKFLOW_VERSION="+strconv.Itoa(runtime.WorkflowVersion),
		"OPENASE_HOOK_NAME="+string(hookName),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(commandContext.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timed out after %s", hook.Timeout)
		}
		if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
			return fmt.Errorf("%w: %s", err, trimmed)
		}
		return err
	}

	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		e.logger.Info(
			"workflow hook output",
			"hook_name", hookName,
			"hook_scope", "workflow",
			"project_id", runtime.ProjectID,
			"workflow_id", runtime.WorkflowID,
			"workflow_name", runtime.WorkflowName,
			"workflow_version", runtime.WorkflowVersion,
			"command", hook.Command,
			"output", trimmed,
		)
	}

	return nil
}

func renderWorkflowHookCommand(command string, hookName workflowHookName, runtime workflowHookRuntime) string {
	return workflowHookTemplatePattern.ReplaceAllStringFunc(command, func(match string) string {
		groups := workflowHookTemplatePattern.FindStringSubmatch(match)
		if len(groups) != 2 {
			return match
		}

		switch groups[1] {
		case "project.id":
			return shellQuote(runtime.ProjectID.String())
		case "workflow.id":
			return shellQuote(runtime.WorkflowID.String())
		case "workflow.name":
			return shellQuote(runtime.WorkflowName)
		case "workflow.version":
			return shellQuote(strconv.Itoa(runtime.WorkflowVersion))
		case "hook.name":
			return shellQuote(string(hookName))
		default:
			return match
		}
	})
}

func shellQuote(raw string) string {
	if raw == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(raw, "'", `'"'"'`) + "'"
}
