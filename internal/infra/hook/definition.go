package hook

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/google/uuid"
)

var (
	_                   = logging.DeclareComponent("hook-definition")
	ErrConfigInvalid    = errors.New("ticket hook config is invalid")
	ErrExecutionBlocked = errors.New("ticket hook blocked ticket progress")
)

type TicketHookName string

const (
	TicketHookOnClaim    TicketHookName = "on_claim"
	TicketHookOnStart    TicketHookName = "on_start"
	TicketHookOnComplete TicketHookName = "on_complete"
	TicketHookOnDone     TicketHookName = "on_done"
	TicketHookOnError    TicketHookName = "on_error"
	TicketHookOnCancel   TicketHookName = "on_cancel"
)

type FailurePolicy string

const (
	FailurePolicyBlock  FailurePolicy = "block"
	FailurePolicyWarn   FailurePolicy = "warn"
	FailurePolicyIgnore FailurePolicy = "ignore"
)

type Outcome string

const (
	OutcomePass    Outcome = "pass"
	OutcomeError   Outcome = "error"
	OutcomeTimeout Outcome = "timeout"
)

type Definition struct {
	Command   string
	Timeout   time.Duration
	Workdir   string
	OnFailure FailurePolicy
}

type TicketHooks struct {
	OnClaim    []Definition
	OnStart    []Definition
	OnComplete []Definition
	OnDone     []Definition
	OnError    []Definition
	OnCancel   []Definition
}

type Repo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Env struct {
	TicketID         uuid.UUID
	ProjectID        uuid.UUID
	TicketIdentifier string
	Workspace        string
	Repos            []Repo
	AgentName        string
	WorkflowType     string
	WorkflowFamily   string
	Attempt          int
	APIURL           string
	AgentToken       string
}

type Result struct {
	Name             string         `json:"name"`
	HookName         TicketHookName `json:"hook_name"`
	Command          string         `json:"command"`
	WorkingDirectory string         `json:"working_directory"`
	Policy           FailurePolicy  `json:"policy"`
	Outcome          Outcome        `json:"outcome"`
	Duration         time.Duration  `json:"duration"`
	ExitCode         *int           `json:"exit_code,omitempty"`
	TimedOut         bool           `json:"timed_out"`
	Stdout           string         `json:"stdout,omitempty"`
	Stderr           string         `json:"stderr,omitempty"`
	Error            string         `json:"error,omitempty"`
}

func ParseTicketHooks(raw map[string]any) (TicketHooks, error) {
	if len(raw) == 0 {
		return TicketHooks{}, nil
	}

	ticketHooksRaw, ok := raw["ticket_hooks"]
	if !ok || ticketHooksRaw == nil {
		return TicketHooks{}, nil
	}

	ticketHooksMap, ok := ticketHooksRaw.(map[string]any)
	if !ok {
		return TicketHooks{}, fmt.Errorf("%w: hooks.ticket_hooks must be an object", ErrConfigInvalid)
	}

	onClaim, err := parseHookList(ticketHooksMap, TicketHookOnClaim)
	if err != nil {
		return TicketHooks{}, err
	}
	onStart, err := parseHookList(ticketHooksMap, TicketHookOnStart)
	if err != nil {
		return TicketHooks{}, err
	}
	onComplete, err := parseHookList(ticketHooksMap, TicketHookOnComplete)
	if err != nil {
		return TicketHooks{}, err
	}
	onDone, err := parseHookList(ticketHooksMap, TicketHookOnDone)
	if err != nil {
		return TicketHooks{}, err
	}
	onError, err := parseHookList(ticketHooksMap, TicketHookOnError)
	if err != nil {
		return TicketHooks{}, err
	}
	onCancel, err := parseHookList(ticketHooksMap, TicketHookOnCancel)
	if err != nil {
		return TicketHooks{}, err
	}

	return TicketHooks{
		OnClaim:    onClaim,
		OnStart:    onStart,
		OnComplete: onComplete,
		OnDone:     onDone,
		OnError:    onError,
		OnCancel:   onCancel,
	}, nil
}

func parseHookList(raw map[string]any, hookName TicketHookName) ([]Definition, error) {
	listRaw, ok := raw[string(hookName)]
	if !ok || listRaw == nil {
		return nil, nil
	}

	switch list := listRaw.(type) {
	case []any:
		return parseHookEntries(list, "hooks.ticket_hooks."+string(hookName))
	case []map[string]any:
		items := make([]any, 0, len(list))
		for _, item := range list {
			items = append(items, item)
		}
		return parseHookEntries(items, "hooks.ticket_hooks."+string(hookName))
	default:
		return nil, fmt.Errorf("%w: hooks.ticket_hooks.%s must be a list", ErrConfigInvalid, hookName)
	}
}

func parseHookEntries(entries []any, path string) ([]Definition, error) {
	hooks := make([]Definition, 0, len(entries))
	for index, entry := range entries {
		parsed, err := parseHookDefinition(entry, path+"["+strconv.Itoa(index)+"]")
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, parsed)
	}

	return hooks, nil
}

func parseHookDefinition(raw any, path string) (Definition, error) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return Definition{}, fmt.Errorf("%w: %s must be an object", ErrConfigInvalid, path)
	}

	commandRaw, ok := entry["cmd"]
	if !ok {
		return Definition{}, fmt.Errorf("%w: %s.cmd is required", ErrConfigInvalid, path)
	}

	command, ok := commandRaw.(string)
	if !ok || strings.TrimSpace(command) == "" {
		return Definition{}, fmt.Errorf("%w: %s.cmd must be a non-empty string", ErrConfigInvalid, path)
	}

	timeout, err := parseTimeout(entry["timeout"], path+".timeout")
	if err != nil {
		return Definition{}, err
	}

	workdir, err := parseWorkdir(entry["workdir"], path+".workdir")
	if err != nil {
		return Definition{}, err
	}

	onFailure, err := parseFailurePolicy(entry["on_failure"], path+".on_failure")
	if err != nil {
		return Definition{}, err
	}

	return Definition{
		Command:   strings.TrimSpace(command),
		Timeout:   timeout,
		Workdir:   workdir,
		OnFailure: onFailure,
	}, nil
}

func parseTimeout(raw any, path string) (time.Duration, error) {
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
			return 0, fmt.Errorf("%w: %s must be a whole number of seconds", ErrConfigInvalid, path)
		}
		seconds = int64(value)
	default:
		return 0, fmt.Errorf("%w: %s must be a whole number of seconds", ErrConfigInvalid, path)
	}

	if seconds < 0 {
		return 0, fmt.Errorf("%w: %s must be greater than or equal to zero", ErrConfigInvalid, path)
	}

	return time.Duration(seconds) * time.Second, nil
}

func parseWorkdir(raw any, path string) (string, error) {
	if raw == nil {
		return "", nil
	}

	workdir, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s must be a string", ErrConfigInvalid, path)
	}

	return strings.TrimSpace(workdir), nil
}

func parseFailurePolicy(raw any, path string) (FailurePolicy, error) {
	if raw == nil {
		return FailurePolicyBlock, nil
	}

	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s must be one of block, warn, ignore", ErrConfigInvalid, path)
	}

	switch policy := FailurePolicy(strings.ToLower(strings.TrimSpace(value))); policy {
	case FailurePolicyBlock, FailurePolicyWarn, FailurePolicyIgnore:
		return policy, nil
	default:
		return "", fmt.Errorf("%w: %s must be one of block, warn, ignore", ErrConfigInvalid, path)
	}
}
