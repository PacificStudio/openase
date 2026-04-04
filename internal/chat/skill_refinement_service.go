package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

const (
	DefaultSkillRefinementMaxAttempts = 3
)

var (
	ErrSkillRefinementUnavailable     = errors.New("skill refinement service unavailable")
	ErrSkillRefinementSessionNotFound = errors.New("skill refinement session not found")

	skillRefinementPathPattern = regexp.MustCompile(`[^a-z0-9]+`)
)

type skillRefinementRuntime interface {
	Supports(catalogdomain.AgentProvider) bool
	StartTurn(context.Context, RuntimeTurnInput) (TurnStream, error)
	CloseSession(SessionID) bool
	SessionAnchor(SessionID) RuntimeSessionAnchor
}

type SkillRefinementInput struct {
	ProjectID  uuid.UUID
	SkillID    uuid.UUID
	ProviderID *uuid.UUID
	Message    string
	DraftFiles []workflowservice.SkillBundleFileInput
}

type SkillRefinementSessionPayload struct {
	SessionID     string `json:"session_id"`
	WorkspacePath string `json:"workspace_path"`
}

type SkillRefinementStatusPayload struct {
	SessionID string `json:"session_id"`
	Phase     string `json:"phase"`
	Attempt   int    `json:"attempt"`
	Message   string `json:"message"`
}

type SkillRefinementResultPayload struct {
	SessionID            string                            `json:"session_id"`
	Status               string                            `json:"status"`
	WorkspacePath        string                            `json:"workspace_path"`
	ProviderID           string                            `json:"provider_id"`
	ProviderName         string                            `json:"provider_name"`
	ProviderThreadID     string                            `json:"provider_thread_id,omitempty"`
	ProviderTurnID       string                            `json:"provider_turn_id,omitempty"`
	Attempts             int                               `json:"attempts"`
	TranscriptSummary    string                            `json:"transcript_summary,omitempty"`
	CommandOutputSummary string                            `json:"command_output_summary,omitempty"`
	FailureReason        string                            `json:"failure_reason,omitempty"`
	CandidateFiles       []workflowservice.SkillBundleFile `json:"candidate_files,omitempty"`
	CandidateBundleHash  string                            `json:"candidate_bundle_hash,omitempty"`
}

type skillRefinementReport struct {
	Type              string `json:"type"`
	Status            string `json:"status"`
	Summary           string `json:"summary"`
	VerificationNotes string `json:"verification_notes"`
	FailureReason     string `json:"failure_reason"`
}

type skillRefinementAttemptOutcome struct {
	status            string
	summary           string
	failureReason     string
	transcriptSummary string
	commandSummary    string
	bundle            workflowservice.SkillBundle
	providerThreadID  string
	providerTurnID    string
}

type skillRefinementRuntimeEventParseResult struct {
	ForwardEvents []StreamEvent
	AssistantText string
	CommandOutput string
	EmitTesting   bool
	TurnErr       error
}

type skillRefinementSessionRecord struct {
	UserID        UserID
	ProjectID     uuid.UUID
	SkillID       uuid.UUID
	ProviderID    uuid.UUID
	WorkspaceRoot string
}

type skillRefinementSessionRegistry struct {
	mu        sync.Mutex
	bySession map[SessionID]skillRefinementSessionRecord
	byUser    map[UserID]SessionID
}

func (r *skillRefinementSessionRegistry) Register(
	userID UserID,
	sessionID SessionID,
	projectID uuid.UUID,
	skillID uuid.UUID,
	providerID uuid.UUID,
	workspaceRoot string,
) {
	if userID == "" || sessionID == "" || projectID == uuid.Nil || skillID == uuid.Nil || providerID == uuid.Nil || strings.TrimSpace(workspaceRoot) == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bySession == nil {
		r.bySession = map[SessionID]skillRefinementSessionRecord{}
	}
	if r.byUser == nil {
		r.byUser = map[UserID]SessionID{}
	}
	if previousSessionID, ok := r.byUser[userID]; ok && previousSessionID != sessionID {
		delete(r.bySession, previousSessionID)
	}
	r.bySession[sessionID] = skillRefinementSessionRecord{
		UserID:        userID,
		ProjectID:     projectID,
		SkillID:       skillID,
		ProviderID:    providerID,
		WorkspaceRoot: workspaceRoot,
	}
	r.byUser[userID] = sessionID
}

func (r *skillRefinementSessionRegistry) ResolveForUser(
	userID UserID,
	sessionID SessionID,
) (skillRefinementSessionRecord, bool) {
	if userID == "" || sessionID == "" {
		return skillRefinementSessionRecord{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.bySession[sessionID]
	if !ok || record.UserID != userID {
		return skillRefinementSessionRecord{}, false
	}
	return record, true
}

func (r *skillRefinementSessionRegistry) ResolveUserSession(userID UserID) (SessionID, bool) {
	if userID == "" {
		return "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	sessionID, ok := r.byUser[userID]
	return sessionID, ok
}

func (r *skillRefinementSessionRegistry) Delete(
	sessionID SessionID,
) (skillRefinementSessionRecord, bool) {
	if sessionID == "" {
		return skillRefinementSessionRecord{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.bySession[sessionID]
	if !ok {
		return skillRefinementSessionRecord{}, false
	}
	delete(r.bySession, sessionID)
	if currentSessionID, current := r.byUser[record.UserID]; current && currentSessionID == sessionID {
		delete(r.byUser, record.UserID)
	}
	return record, true
}

type SkillRefinementService struct {
	logger      *slog.Logger
	runtime     skillRefinementRuntime
	catalog     catalogReader
	workflows   workflowReader
	sessions    skillRefinementSessionRegistry
	userLocks   userLockRegistry
	maxAttempts int
}

func NewSkillRefinementService(
	logger *slog.Logger,
	runtime skillRefinementRuntime,
	catalog catalogReader,
	workflows workflowReader,
) *SkillRefinementService {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &SkillRefinementService{
		logger:      logger.With("component", "skill-refinement-service"),
		runtime:     runtime,
		catalog:     catalog,
		workflows:   workflows,
		maxAttempts: DefaultSkillRefinementMaxAttempts,
	}
}

func (s *SkillRefinementService) Start(
	ctx context.Context,
	userID UserID,
	input SkillRefinementInput,
) (TurnStream, error) {
	if s == nil || s.runtime == nil || s.catalog == nil || s.workflows == nil {
		return TurnStream{}, ErrSkillRefinementUnavailable
	}
	if strings.TrimSpace(input.Message) == "" {
		return TurnStream{}, fmt.Errorf("message must not be empty")
	}
	if len(input.DraftFiles) == 0 {
		return TurnStream{}, fmt.Errorf("files must not be empty")
	}

	unlockUser := s.userLocks.Lock(userID)
	defer unlockUser()

	if previousSessionID, ok := s.sessions.ResolveUserSession(userID); ok {
		s.closeSession(previousSessionID)
	}

	project, err := s.catalog.GetProject(ctx, input.ProjectID)
	if err != nil {
		return TurnStream{}, fmt.Errorf("get project for skill refinement: %w", err)
	}
	skillDetail, err := s.workflows.GetSkill(ctx, input.SkillID)
	if err != nil {
		return TurnStream{}, fmt.Errorf("get skill for refinement: %w", err)
	}
	bundle, err := workflowservice.BuildSkillBundle(skillDetail.Name, input.DraftFiles)
	if err != nil {
		return TurnStream{}, err
	}
	providerItem, err := s.resolveProvider(ctx, project, input.ProviderID)
	if err != nil {
		return TurnStream{}, err
	}

	sessionUUID := uuid.NewString()
	sessionID := SessionID("skill-refinement-" + sessionUUID)
	workspaceRoot, err := s.prepareWorkspaceRoot(project.Name, skillDetail.Name, sessionUUID)
	if err != nil {
		return TurnStream{}, err
	}
	projection, err := workflowservice.MaterializeDraftSkillBundle(workflowservice.DraftSkillProjectionInput{
		WorkspaceRoot: workspaceRoot,
		AdapterType:   providerItem.AdapterType.String(),
		Bundle:        bundle,
	})
	if err != nil {
		_ = os.RemoveAll(workspaceRoot)
		return TurnStream{}, err
	}

	s.sessions.Register(userID, sessionID, input.ProjectID, input.SkillID, providerItem.ID, projection.WorkspaceRoot)

	events := make(chan StreamEvent, 64)
	go s.run(ctx, sessionID, providerItem, projection, skillDetail.Name, input.Message, events)
	return TurnStream{Events: events}, nil
}

func (s *SkillRefinementService) CloseSession(userID UserID, sessionID SessionID) bool {
	unlockUser := s.userLocks.Lock(userID)
	defer unlockUser()
	record, ok := s.sessions.ResolveForUser(userID, sessionID)
	if !ok {
		return false
	}
	s.sessions.Delete(sessionID)
	s.shutdown(record.WorkspaceRoot, sessionID)
	return true
}

func (s *SkillRefinementService) ResolveSessionScopeForUser(
	userID UserID,
	sessionID SessionID,
) (uuid.UUID, uuid.UUID, bool) {
	record, ok := s.sessions.ResolveForUser(userID, sessionID)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return record.ProjectID, record.SkillID, true
}

func (s *SkillRefinementService) closeSession(sessionID SessionID) {
	record, ok := s.sessions.Delete(sessionID)
	if !ok {
		return
	}
	s.shutdown(record.WorkspaceRoot, sessionID)
}

func (s *SkillRefinementService) shutdown(workspaceRoot string, sessionID SessionID) {
	s.runtime.CloseSession(sessionID)
	if strings.TrimSpace(workspaceRoot) != "" {
		_ = os.RemoveAll(workspaceRoot)
	}
}

func (s *SkillRefinementService) run(
	ctx context.Context,
	sessionID SessionID,
	providerItem catalogdomain.AgentProvider,
	projection workflowservice.DraftSkillProjection,
	skillName string,
	userMessage string,
	events chan<- StreamEvent,
) {
	defer close(events)

	events <- StreamEvent{
		Event: "session",
		Payload: SkillRefinementSessionPayload{
			SessionID:     sessionID.String(),
			WorkspacePath: projection.WorkspaceRoot,
		},
	}

	workingDirectory, err := provider.ParseAbsolutePath(projection.WorkspaceRoot)
	if err != nil {
		s.emitTerminalError(events, fmt.Sprintf("resolve workspace path: %v", err))
		s.closeSession(sessionID)
		return
	}

	lastFailure := ""
	for attempt := 1; attempt <= s.maxAttempts; attempt++ {
		phase := "editing"
		message := "Codex is editing the draft bundle."
		if attempt > 1 {
			phase = "retrying"
			message = "Retrying after the previous verification attempt failed."
		}
		events <- StreamEvent{
			Event: "status",
			Payload: SkillRefinementStatusPayload{
				SessionID: sessionID.String(),
				Phase:     phase,
				Attempt:   attempt,
				Message:   message,
			},
		}

		outcome, runErr := s.runAttempt(ctx, sessionID, providerItem, projection, skillName, userMessage, lastFailure, attempt, workingDirectory, events)
		if runErr != nil {
			status := "unverified"
			if attempt < s.maxAttempts {
				lastFailure = runErr.Error()
				continue
			}
			s.emitResult(events, SkillRefinementResultPayload{
				SessionID:     sessionID.String(),
				Status:        status,
				WorkspacePath: projection.WorkspaceRoot,
				ProviderID:    providerItem.ID.String(),
				ProviderName:  providerItem.Name,
				Attempts:      attempt,
				FailureReason: runErr.Error(),
			})
			return
		}

		if outcome.status == "verified" {
			events <- StreamEvent{
				Event: "status",
				Payload: SkillRefinementStatusPayload{
					SessionID: sessionID.String(),
					Phase:     "verified",
					Attempt:   attempt,
					Message:   firstNonEmpty(outcome.summary, "Verification passed."),
				},
			}
			s.emitResult(events, SkillRefinementResultPayload{
				SessionID:            sessionID.String(),
				Status:               outcome.status,
				WorkspacePath:        projection.WorkspaceRoot,
				ProviderID:           providerItem.ID.String(),
				ProviderName:         providerItem.Name,
				ProviderThreadID:     outcome.providerThreadID,
				ProviderTurnID:       outcome.providerTurnID,
				Attempts:             attempt,
				TranscriptSummary:    outcome.transcriptSummary,
				CommandOutputSummary: outcome.commandSummary,
				CandidateFiles:       outcome.bundle.Files,
				CandidateBundleHash:  outcome.bundle.BundleHash,
			})
			return
		}

		lastFailure = firstNonEmpty(outcome.failureReason, outcome.summary, "verification did not pass")
		if attempt >= s.maxAttempts {
			finalPhase := outcome.status
			if finalPhase == "" {
				finalPhase = "blocked"
			}
			events <- StreamEvent{
				Event: "status",
				Payload: SkillRefinementStatusPayload{
					SessionID: sessionID.String(),
					Phase:     finalPhase,
					Attempt:   attempt,
					Message:   lastFailure,
				},
			}
			s.emitResult(events, SkillRefinementResultPayload{
				SessionID:            sessionID.String(),
				Status:               finalPhase,
				WorkspacePath:        projection.WorkspaceRoot,
				ProviderID:           providerItem.ID.String(),
				ProviderName:         providerItem.Name,
				ProviderThreadID:     outcome.providerThreadID,
				ProviderTurnID:       outcome.providerTurnID,
				Attempts:             attempt,
				TranscriptSummary:    outcome.transcriptSummary,
				CommandOutputSummary: outcome.commandSummary,
				FailureReason:        lastFailure,
				CandidateFiles:       outcome.bundle.Files,
				CandidateBundleHash:  outcome.bundle.BundleHash,
			})
			return
		}
	}
}

func (s *SkillRefinementService) runAttempt(
	ctx context.Context,
	sessionID SessionID,
	providerItem catalogdomain.AgentProvider,
	projection workflowservice.DraftSkillProjection,
	skillName string,
	userMessage string,
	lastFailure string,
	attempt int,
	workingDirectory provider.AbsolutePath,
	events chan<- StreamEvent,
) (skillRefinementAttemptOutcome, error) {
	stream, err := s.runtime.StartTurn(ctx, RuntimeTurnInput{
		SessionID:        sessionID,
		Provider:         providerItem,
		Message:          buildSkillRefinementPrompt(skillName, userMessage, projection.SkillDir, attempt, lastFailure),
		SystemPrompt:     buildSkillRefinementSystemPrompt(projection.SkillDir),
		WorkingDirectory: workingDirectory,
	})
	if err != nil {
		return skillRefinementAttemptOutcome{}, fmt.Errorf("start skill refinement turn: %w", err)
	}

	assistantTexts := make([]string, 0, 4)
	commandOutputs := make([]string, 0, 8)
	testingEmitted := false
	var turnErr error

	for event := range stream.Events {
		parsed := parseSkillRefinementRuntimeEvent(event, testingEmitted)
		if strings.TrimSpace(parsed.AssistantText) != "" {
			assistantTexts = append(assistantTexts, parsed.AssistantText)
		}
		if strings.TrimSpace(parsed.CommandOutput) != "" {
			commandOutputs = append(commandOutputs, parsed.CommandOutput)
		}
		if parsed.EmitTesting && !testingEmitted {
			testingEmitted = true
			events <- StreamEvent{
				Event: "status",
				Payload: SkillRefinementStatusPayload{
					SessionID: sessionID.String(),
					Phase:     "testing",
					Attempt:   attempt,
					Message:   "Codex is running verification commands.",
				},
			}
		}
		for _, item := range parsed.ForwardEvents {
			events <- item
		}
		if parsed.TurnErr != nil {
			turnErr = parsed.TurnErr
		}
	}
	if turnErr != nil {
		return skillRefinementAttemptOutcome{}, turnErr
	}

	bundle, bundleErr := workflowservice.LoadSkillBundleFromDirectory(skillName, projection.SkillDir)
	anchor := s.runtime.SessionAnchor(sessionID)
	report, hasReport := parseSkillRefinementReport(assistantTexts)
	transcriptSummary := summarizeAttemptOutput(assistantTexts, report)
	commandSummary := summarizeCommandOutput(commandOutputs)

	var status string
	summary := transcriptSummary
	failureReason := ""
	switch {
	case bundleErr != nil:
		status = "blocked"
		failureReason = fmt.Sprintf("final skill bundle is invalid: %v", bundleErr)
	case len(commandOutputs) == 0:
		status = "unverified"
		failureReason = "verification commands were not observed in the runtime transcript"
	case !hasReport:
		status = "unverified"
		failureReason = "Codex did not return a structured verification report"
	default:
		status = normalizeSkillRefinementStatus(report.Status)
		summary = firstNonEmpty(strings.TrimSpace(report.Summary), transcriptSummary)
		failureReason = strings.TrimSpace(report.FailureReason)
	}
	if status == "verified" && bundleErr == nil && len(commandOutputs) > 0 {
		return skillRefinementAttemptOutcome{
			status:            status,
			summary:           summary,
			transcriptSummary: transcriptSummary,
			commandSummary:    commandSummary,
			bundle:            bundle,
			providerThreadID:  strings.TrimSpace(anchor.ProviderThreadID),
			providerTurnID:    strings.TrimSpace(anchor.LastTurnID),
		}, nil
	}
	if failureReason == "" {
		failureReason = firstNonEmpty(summary, "verification did not pass")
	}
	return skillRefinementAttemptOutcome{
		status:            status,
		summary:           summary,
		failureReason:     failureReason,
		transcriptSummary: transcriptSummary,
		commandSummary:    commandSummary,
		bundle:            bundle,
		providerThreadID:  strings.TrimSpace(anchor.ProviderThreadID),
		providerTurnID:    strings.TrimSpace(anchor.LastTurnID),
	}, nil
}

func (s *SkillRefinementService) resolveProvider(
	ctx context.Context,
	project catalogdomain.Project,
	rawProviderID *uuid.UUID,
) (catalogdomain.AgentProvider, error) {
	providers, err := s.catalog.ListAgentProviders(ctx, project.OrganizationID)
	if err != nil {
		return catalogdomain.AgentProvider{}, fmt.Errorf("list project providers for skill refinement: %w", err)
	}
	return resolveProviderForSurface(
		providers,
		project.DefaultAgentProviderID,
		rawProviderID,
		providerSurfaceSkillAI,
		s.runtime.Supports,
	)
}

func (s *SkillRefinementService) prepareWorkspaceRoot(
	projectName string,
	skillName string,
	sessionID string,
) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory for skill refinement: %w", err)
	}
	projectComponent := sanitizeSkillRefinementPathComponent(projectName)
	skillComponent := sanitizeSkillRefinementPathComponent(skillName)
	workspaceRoot := filepath.Join(
		homeDir,
		".openase",
		"skill-tests",
		projectComponent,
		skillComponent,
		sessionID,
		"workspace",
	)
	if err := os.MkdirAll(workspaceRoot, 0o750); err != nil {
		return "", fmt.Errorf("create skill refinement workspace: %w", err)
	}
	return workspaceRoot, nil
}

func sanitizeSkillRefinementPathComponent(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	trimmed = skillRefinementPathPattern.ReplaceAllString(trimmed, "-")
	trimmed = strings.Trim(trimmed, "-")
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func buildSkillRefinementSystemPrompt(skillDir string) string {
	return strings.TrimSpace(fmt.Sprintf(`
你正在执行 OpenASE Skill 的 fix-and-verify refinement。

要求：
- 只能在当前工作区内操作。
- 当前待编辑的 skill bundle 位于 %s。
- 必须直接修改该目录下的真实文件，而不是只给建议。
- 必须亲自运行验证命令；没有运行命令就不能声称 verified。
- 如果发现失败，先修复再重试；只有在确认无法完成时才返回 blocked。
- 最终只输出一个 JSON 对象，不要附加解释文本。

最终 JSON 格式：
{"type":"skill_refinement_result","status":"verified|blocked","summary":"一句话总结","verification_notes":"关键验证步骤摘要","failure_reason":"若 blocked 则填写"}
`, filepath.ToSlash(skillDir)))
}

func buildSkillRefinementPrompt(
	skillName string,
	userMessage string,
	skillDir string,
	attempt int,
	lastFailure string,
) string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "目标 skill: %s\n", skillName)
	_, _ = fmt.Fprintf(&sb, "skill 目录: %s\n", filepath.ToSlash(skillDir))
	_, _ = fmt.Fprintf(&sb, "用户请求: %s\n", strings.TrimSpace(userMessage))
	if attempt > 1 && strings.TrimSpace(lastFailure) != "" {
		_, _ = fmt.Fprintf(&sb, "上一次失败摘要: %s\n", strings.TrimSpace(lastFailure))
	}
	sb.WriteString("\n执行步骤：\n")
	sb.WriteString("1. 阅读并修改 skill bundle 中必要的文件。\n")
	sb.WriteString("2. 运行真实验证命令，至少证明本次改动已被检查。\n")
	sb.WriteString("3. 若验证失败，继续修复直到通过，或确认被阻塞。\n")
	sb.WriteString("4. 结束时仅输出最终 JSON。\n")
	return sb.String()
}

func normalizeSkillRefinementStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "verified":
		return "verified"
	case "blocked":
		return "blocked"
	default:
		return "unverified"
	}
}

func parseSkillRefinementReport(items []string) (skillRefinementReport, bool) {
	for index := len(items) - 1; index >= 0; index-- {
		candidate := extractSkillRefinementJSONObjectCandidate(items[index])
		if candidate == "" {
			continue
		}
		var report skillRefinementReport
		if err := json.Unmarshal([]byte(candidate), &report); err != nil {
			continue
		}
		if normalizeSkillRefinementStatus(report.Status) == "unverified" && strings.TrimSpace(report.Status) == "" {
			continue
		}
		return report, true
	}
	return skillRefinementReport{}, false
}

func extractSkillRefinementJSONObjectCandidate(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") {
		parts := strings.Split(trimmed, "\n")
		if len(parts) >= 3 {
			trimmed = strings.Join(parts[1:len(parts)-1], "\n")
			trimmed = strings.TrimSpace(trimmed)
			if strings.HasPrefix(trimmed, "json") {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "json"))
			}
		}
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return ""
	}
	return strings.TrimSpace(trimmed[start : end+1])
}

func summarizeAttemptOutput(items []string, report skillRefinementReport) string {
	if summary := strings.TrimSpace(report.Summary); summary != "" {
		return summary
	}
	if notes := strings.TrimSpace(report.VerificationNotes); notes != "" {
		return notes
	}
	return summarizeLines(items, 4, 800)
}

func summarizeCommandOutput(items []string) string {
	return summarizeLines(items, 8, 1600)
}

func summarizeLines(items []string, maxItems int, maxBytes int) string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	if len(filtered) == 0 {
		return ""
	}
	if maxItems > 0 && len(filtered) > maxItems {
		filtered = filtered[len(filtered)-maxItems:]
	}
	joined := strings.Join(filtered, "\n\n")
	if maxBytes > 0 && len(joined) > maxBytes {
		return joined[len(joined)-maxBytes:]
	}
	return joined
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseSkillRefinementRuntimeEvent(
	event StreamEvent,
	testingEmitted bool,
) skillRefinementRuntimeEventParseResult {
	result := skillRefinementRuntimeEventParseResult{}

	switch event.Event {
	case "message":
		result.ForwardEvents = []StreamEvent{event}
		switch payload := event.Payload.(type) {
		case textPayload:
			if payload.Type == chatMessageTypeText && strings.TrimSpace(payload.Content) != "" {
				result.AssistantText = payload.Content
			}
		case map[string]any:
			if strings.TrimSpace(stringValue(payload["type"])) != chatMessageTypeTaskProgress {
				return result
			}
			raw, _ := payload["raw"].(map[string]any)
			text := strings.TrimSpace(stringValue(raw["text"]))
			if text == "" {
				return result
			}
			result.CommandOutput = text
			result.EmitTesting = !testingEmitted
		}
	case "interrupt_requested",
		"thread_status",
		"session_state",
		"thread_compacted",
		"plan_updated",
		"diff_updated",
		"reasoning_updated",
		"session_anchor":
		result.ForwardEvents = []StreamEvent{event}
	case "error", "interrupted":
		switch payload := event.Payload.(type) {
		case errorPayload:
			message := strings.TrimSpace(payload.Message)
			if message == "" {
				message = "skill refinement runtime failed"
			}
			result.TurnErr = errors.New(message)
		default:
			result.TurnErr = errors.New("skill refinement runtime failed")
		}
	default:
		if strings.TrimSpace(event.Event) != "" {
			result.ForwardEvents = []StreamEvent{event}
		}
	}

	return result
}

func (s *SkillRefinementService) emitTerminalError(events chan<- StreamEvent, message string) {
	events <- StreamEvent{
		Event:   "error",
		Payload: errorPayload{Message: strings.TrimSpace(message)},
	}
}

func (s *SkillRefinementService) emitResult(events chan<- StreamEvent, payload SkillRefinementResultPayload) {
	events <- StreamEvent{Event: "result", Payload: payload}
}
