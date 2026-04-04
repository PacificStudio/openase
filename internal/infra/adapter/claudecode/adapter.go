package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

const scannerBufferSize = 16 * 1024 * 1024

var claudeCodeAdapterComponent = logging.DeclareComponent("claudecode-adapter")

type Adapter struct {
	processManager provider.AgentCLIProcessManager
	logger         *slog.Logger
}

func NewAdapter(processManager provider.AgentCLIProcessManager) provider.ClaudeCodeAdapter {
	return &Adapter{
		processManager: processManager,
		logger:         logging.WithComponent(nil, claudeCodeAdapterComponent),
	}
}

func (a *Adapter) Start(ctx context.Context, spec provider.ClaudeCodeSessionSpec) (provider.ClaudeCodeSession, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context must not be nil")
	}
	if a.processManager == nil {
		return nil, fmt.Errorf("claude code process manager must not be nil")
	}

	processSpec, err := buildProcessSpec(spec)
	if err != nil {
		return nil, err
	}

	process, err := a.processManager.Start(ctx, processSpec)
	if err != nil {
		return nil, err
	}

	session := &session{
		process: process,
		events:  make(chan provider.ClaudeCodeEvent, 64),
		errors:  make(chan error, 64),
		done:    make(chan struct{}),
		logger:  a.logger,
	}
	session.startReaders()

	return session, nil
}

func buildProcessSpec(spec provider.ClaudeCodeSessionSpec) (provider.AgentCLIProcessSpec, error) {
	args := append([]string(nil), spec.BaseArgs...)
	if !hasArgumentFlag(args, "--verbose") {
		args = append(args, "--verbose")
	}
	args = append(args,
		"-p",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
	)

	if spec.IncludePartialMessages {
		args = append(args, "--include-partial-messages")
	}
	if spec.ResumeSessionID != nil {
		args = append(args, "--resume", spec.ResumeSessionID.String())
	}
	if len(spec.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(spec.AllowedTools, ","))
	}
	if spec.MaxTurns != nil {
		args = append(args, "--max-turns", strconv.Itoa(*spec.MaxTurns))
	}
	if spec.MaxBudgetUSD != nil {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(*spec.MaxBudgetUSD, 'f', -1, 64))
	}
	if spec.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", spec.AppendSystemPrompt)
	}

	return provider.NewAgentCLIProcessSpec(
		spec.Command,
		args,
		spec.WorkingDirectory,
		spec.Environment,
	)
}

func hasArgumentFlag(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
		if strings.HasPrefix(arg, want+"=") {
			return true
		}
	}
	return false
}

type session struct {
	process provider.AgentCLIProcess

	events chan provider.ClaudeCodeEvent
	errors chan error
	done   chan struct{}

	sessionMu sync.RWMutex
	sessionID provider.ClaudeCodeSessionID

	writeMu sync.Mutex
	closeMu sync.Once
	logger  *slog.Logger
}

func (s *session) componentLogger() *slog.Logger {
	return logging.WithComponent(s.logger, claudeCodeAdapterComponent)
}

func (s *session) SessionID() (provider.ClaudeCodeSessionID, bool) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	if s.sessionID == "" {
		return "", false
	}

	return s.sessionID, true
}

func (s *session) Events() <-chan provider.ClaudeCodeEvent {
	return s.events
}

func (s *session) Errors() <-chan error {
	return s.errors
}

func (s *session) Send(ctx context.Context, input provider.ClaudeCodeTurnInput) error {
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return fmt.Errorf("claude code turn prompt must not be empty")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return fmt.Errorf("claude code session already closed")
	default:
	}

	payload, err := encodeTurnInput(input)
	if err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return fmt.Errorf("claude code session already closed")
	default:
	}

	if _, err := s.process.Stdin().Write(payload); err != nil {
		return fmt.Errorf("write claude code turn input: %w", err)
	}

	return nil
}

func (s *session) Close(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}

	var stopErr error
	s.closeMu.Do(func() {
		close(s.done)
		_ = s.process.Stdin().Close()
		stopErr = s.process.Stop(ctx)
	})

	return stopErr
}

func (s *session) startReaders() {
	var readers sync.WaitGroup
	readers.Add(2)

	go func() {
		defer readers.Done()
		s.readStdout(s.process.Stdout())
	}()

	go func() {
		defer readers.Done()
		s.readStderr(s.process.Stderr())
	}()

	go func() {
		waitErr := s.process.Wait()
		readers.Wait()
		if waitErr != nil {
			s.componentLogger().Error("claude code process exited", "error", waitErr)
			s.pushError(fmt.Errorf("claude code process exited: %w", waitErr))
		}
		close(s.events)
		close(s.errors)
	}()
}

func (s *session) readStdout(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		event, err := parseStreamEvent([]byte(line))
		if err != nil {
			s.componentLogger().Warn("parse claude code stream event failed", "line", line, "error", err)
			s.pushError(err)
			continue
		}

		if event.SessionID != "" {
			s.setSessionID(event.SessionID)
		}

		s.events <- event
	}

	if err := scanner.Err(); err != nil {
		s.componentLogger().Error("read claude code stdout failed", "error", err)
		s.pushError(fmt.Errorf("read claude code stdout: %w", err))
	}
}

func (s *session) readStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		s.componentLogger().Warn("claude code stderr output", "line", line)
		s.pushError(fmt.Errorf("claude code stderr: %s", line))
	}

	if err := scanner.Err(); err != nil {
		s.componentLogger().Error("read claude code stderr failed", "error", err)
		s.pushError(fmt.Errorf("read claude code stderr: %w", err))
	}
}

func (s *session) setSessionID(raw string) {
	parsed, err := provider.ParseClaudeCodeSessionID(raw)
	if err != nil {
		return
	}

	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	if s.sessionID == "" {
		s.sessionID = parsed
	}
}

func (s *session) pushError(err error) {
	select {
	case <-s.done:
		return
	default:
	}

	s.errors <- err
}

type rawStreamEvent struct {
	Type            string          `json:"type"`
	Subtype         string          `json:"subtype"`
	SessionID       string          `json:"session_id"`
	ParentToolUseID string          `json:"parent_tool_use_id"`
	Message         json.RawMessage `json:"message"`
	Data            json.RawMessage `json:"data"`
	Result          string          `json:"result"`
	Model           string          `json:"model"`
	Usage           json.RawMessage `json:"usage"`
	RateLimitInfo   json.RawMessage `json:"rate_limit_info"`
	Event           json.RawMessage `json:"event"`
	UUID            string          `json:"uuid"`
	IsError         bool            `json:"is_error"`
	NumTurns        int             `json:"num_turns"`
	DurationMS      int             `json:"duration_ms"`
	DurationAPIMS   int             `json:"duration_api_ms"`
	TotalCostUSD    *float64        `json:"total_cost_usd"`
}

type rawSystemData struct {
	SessionID string `json:"session_id"`
}

func parseStreamEvent(line []byte) (provider.ClaudeCodeEvent, error) {
	var raw rawStreamEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return provider.ClaudeCodeEvent{}, fmt.Errorf("parse claude code ndjson event: %w", err)
	}

	eventType := strings.TrimSpace(raw.Type)
	if eventType == "" {
		return provider.ClaudeCodeEvent{}, fmt.Errorf("parse claude code ndjson event: missing type")
	}

	sessionID := strings.TrimSpace(raw.SessionID)
	if sessionID == "" && len(raw.Data) > 0 {
		var data rawSystemData
		if err := json.Unmarshal(raw.Data, &data); err == nil {
			sessionID = strings.TrimSpace(data.SessionID)
		}
	}

	event := provider.ClaudeCodeEvent{
		Kind:            mapEventKind(eventType),
		Raw:             append(json.RawMessage(nil), line...),
		UnknownType:     eventType,
		Subtype:         strings.TrimSpace(raw.Subtype),
		SessionID:       sessionID,
		ParentToolUseID: strings.TrimSpace(raw.ParentToolUseID),
		Message:         cloneRawJSON(raw.Message),
		Data:            cloneRawJSON(raw.Data),
		Result:          raw.Result,
		Model:           strings.TrimSpace(raw.Model),
		Usage:           cloneRawJSON(raw.Usage),
		RateLimit:       cloneRawJSON(raw.RateLimitInfo),
		Event:           cloneRawJSON(raw.Event),
		UUID:            strings.TrimSpace(raw.UUID),
		IsError:         raw.IsError,
		NumTurns:        raw.NumTurns,
		DurationMS:      raw.DurationMS,
		DurationAPIMS:   raw.DurationAPIMS,
		TotalCostUSD:    cloneOptionalFloat(raw.TotalCostUSD),
	}
	usageInfo, err := provider.ParseClaudeCodeUsage(raw.Usage, event.Model, raw.TotalCostUSD)
	if err == nil {
		event.UsageInfo = usageInfo
	}
	rateLimitInfo, err := provider.ParseClaudeCodeRateLimit(raw.RateLimitInfo)
	if err == nil {
		event.RateLimitInfo = rateLimitInfo
	}
	if event.Kind != provider.ClaudeCodeEventKindUnknown {
		event.UnknownType = ""
	}

	return event, nil
}

func mapEventKind(eventType string) provider.ClaudeCodeEventKind {
	switch eventType {
	case "system":
		return provider.ClaudeCodeEventKindSystem
	case "assistant":
		return provider.ClaudeCodeEventKindAssistant
	case "user":
		return provider.ClaudeCodeEventKindUser
	case "result":
		return provider.ClaudeCodeEventKindResult
	case "rate_limit_event":
		return provider.ClaudeCodeEventKindRateLimit
	case "stream_event":
		return provider.ClaudeCodeEventKindStream
	case "task_started":
		return provider.ClaudeCodeEventKindTaskStart
	case "task_progress":
		return provider.ClaudeCodeEventKindTaskProgress
	case "task_notification":
		return provider.ClaudeCodeEventKindTaskNotice
	default:
		return provider.ClaudeCodeEventKindUnknown
	}
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	return append(json.RawMessage(nil), raw...)
}

func cloneOptionalFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

type rawTurnInput struct {
	Type    string         `json:"type"`
	Message rawTurnMessage `json:"message"`
}

type rawTurnMessage struct {
	Role    string             `json:"role"`
	Content []rawTurnTextBlock `json:"content"`
}

type rawTurnTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func encodeTurnInput(input provider.ClaudeCodeTurnInput) ([]byte, error) {
	payload, err := json.Marshal(rawTurnInput{
		Type: "user",
		Message: rawTurnMessage{
			Role: "user",
			Content: []rawTurnTextBlock{
				{
					Type: "text",
					Text: input.Prompt,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encode claude code turn input: %w", err)
	}

	return append(payload, '\n'), nil
}
