package chat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	claudecodeadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func (s *ProjectConversationService) StartTurn(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	message string,
	focus *ProjectConversationFocus,
) (domain.Turn, error) {
	return s.startTurn(ctx, userID, conversationID, message, focus, nil)
}

func (s *ProjectConversationService) StartTurnWithWorkspaceFileDraft(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	message string,
	focus *ProjectConversationFocus,
	workspaceFileDraft *ProjectConversationWorkspaceFileDraftContext,
) (domain.Turn, error) {
	return s.startTurn(ctx, userID, conversationID, message, focus, workspaceFileDraft)
}

func (s *ProjectConversationService) startTurn(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	message string,
	focus *ProjectConversationFocus,
	workspaceFileDraft *ProjectConversationWorkspaceFileDraftContext,
) (domain.Turn, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.Turn{}, err
	}

	unlock := s.core.turnLocks.Lock(projectConversationTurnLockKey(conversation))
	defer unlock()

	project, err := s.core.catalog.GetProject(ctx, conversation.ProjectID)
	if err != nil {
		return domain.Turn{}, fmt.Errorf("get project for chat turn: %w", err)
	}
	providerItem, err := s.core.catalog.GetAgentProvider(ctx, conversation.ProviderID)
	if err != nil {
		return domain.Turn{}, fmt.Errorf("get provider for chat turn: %w", err)
	}

	_, hadLive := s.runtimeManager.Get(conversation.ID)
	if err := s.recoverStaleActiveTurnBeforeStart(ctx, conversation, hadLive); err != nil {
		return domain.Turn{}, err
	}

	live, hadLive, err := s.ensureLiveRuntime(ctx, conversation, project, providerItem)
	if err != nil {
		return domain.Turn{}, err
	}

	promptFocus := focus

	includeRecovery := !hadLive
	resumeThreadID := strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID))
	resumeTurnID := strings.TrimSpace(stringPointerValue(conversation.LastTurnID))
	if !hadLive && resumeThreadID != "" {
		switch providerItem.AdapterType {
		case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
			if live.codex == nil {
				break
			}
			resumePrompt, resumePromptErr := s.buildProjectConversationPromptWithDraft(
				ctx,
				conversation,
				project,
				promptFocus,
				workspaceFileDraft,
				false,
			)
			if resumePromptErr != nil {
				return domain.Turn{}, resumePromptErr
			}
			runtimeInput, inputErr := s.buildConversationRuntimeInput(
				ctx,
				conversation,
				project,
				providerItem,
				live.workspace,
				resumePrompt,
				resumeThreadID,
				resumeTurnID,
				promptFocus,
			)
			if inputErr != nil {
				return domain.Turn{}, inputErr
			}
			resumeErr := live.codex.EnsureSession(ctx, runtimeInput)
			switch {
			case resumeErr == nil:
				includeRecovery = false
			case codexadapter.IsThreadNotFoundError(resumeErr):
				resumeThreadID = ""
				resumeTurnID = ""
				emptyFlags := []string{}
				_, _ = s.core.conversations.UpdateConversationAnchors(ctx, conversationID, conversation.Status, domain.ConversationAnchors{
					ProviderThreadID:          optionalString(""),
					LastTurnID:                optionalString(""),
					ProviderThreadStatus:      optionalString("notLoaded"),
					ProviderThreadActiveFlags: &emptyFlags,
					RollingSummary:            conversation.RollingSummary,
				})
				includeRecovery = true
			default:
				return domain.Turn{}, resumeErr
			}
		case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
			if _, parseErr := provider.ParseClaudeCodeSessionID(resumeThreadID); parseErr == nil {
				includeRecovery = false
			}
		}
	}

	systemPrompt, err := s.buildProjectConversationPromptWithDraft(
		ctx,
		conversation,
		project,
		promptFocus,
		workspaceFileDraft,
		includeRecovery,
	)
	if err != nil {
		return domain.Turn{}, err
	}

	turn, _, err := s.core.entries.CreateTurnWithUserEntry(ctx, conversationID, strings.TrimSpace(message))
	if err != nil {
		return domain.Turn{}, err
	}
	if _, err := s.core.entries.AppendEntry(
		ctx,
		conversationID,
		&turn.ID,
		domain.EntryKindSystem,
		serializeProjectConversationFocus(promptFocus),
	); err != nil {
		return domain.Turn{}, err
	}

	runNow := time.Now().UTC()
	run, err := s.core.runtimeStore.CreateRun(ctx, domain.CreateRunInput{
		RunID:                uuid.New(),
		PrincipalID:          live.principal.ID,
		ConversationID:       conversation.ID,
		ProjectID:            conversation.ProjectID,
		ProviderID:           providerItem.ID,
		TurnID:               &turn.ID,
		Status:               domain.RunStatusLaunching,
		SessionID:            optionalString(conversation.ID.String()),
		WorkspacePath:        optionalString(live.workspace.String()),
		RuntimeStartedAt:     &runNow,
		LastHeartbeatAt:      &runNow,
		CurrentStepStatus:    optionalString("turn_launching"),
		CurrentStepSummary:   optionalString("Starting project conversation turn."),
		CurrentStepChangedAt: &runNow,
	})
	if err != nil {
		return domain.Turn{}, err
	}
	if principal, runtimeErr := s.core.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
		PrincipalID:          live.principal.ID,
		RuntimeState:         domain.RuntimeStateExecuting,
		CurrentSessionID:     optionalString(conversation.ID.String()),
		CurrentWorkspacePath: optionalString(live.workspace.String()),
		CurrentRunID:         &run.ID,
		LastHeartbeatAt:      &runNow,
		CurrentStepStatus:    optionalString("turn_launching"),
		CurrentStepSummary:   optionalString("Starting project conversation turn."),
		CurrentStepChangedAt: &runNow,
	}); runtimeErr == nil {
		live.principal = principal
	}

	environment, err := s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem, promptFocus)
	if err != nil {
		return domain.Turn{}, err
	}
	stream, err := live.runtime.StartTurn(ctx, RuntimeTurnInput{
		SessionID:              SessionID(conversationID.String()),
		Provider:               providerItem,
		Message:                strings.TrimSpace(message),
		SystemPrompt:           systemPrompt,
		WorkingDirectory:       live.workspace,
		Environment:            environment,
		ResumeProviderThreadID: resumeThreadID,
		ResumeProviderTurnID:   resumeTurnID,
		MaxTurns:               0,
		MaxBudgetUSD:           0,
		PersistentConversation: true,
	})
	if err != nil {
		failedStatus := domain.RunStatusFailed
		_, _ = s.core.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
			RunID:                run.ID,
			Status:               &failedStatus,
			TerminalAt:           &runNow,
			LastError:            optionalString(err.Error()),
			LastHeartbeatAt:      &runNow,
			CurrentStepStatus:    optionalString("turn_failed"),
			CurrentStepSummary:   optionalString("Project conversation turn failed to start."),
			CurrentStepChangedAt: &runNow,
		})
		if principal, runtimeErr := s.core.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
			PrincipalID:          live.principal.ID,
			RuntimeState:         domain.RuntimeStateReady,
			CurrentSessionID:     optionalString(conversation.ID.String()),
			CurrentWorkspacePath: optionalString(live.workspace.String()),
			CurrentRunID:         &run.ID,
			LastHeartbeatAt:      &runNow,
			CurrentStepStatus:    optionalString("runtime_ready"),
			CurrentStepSummary:   optionalString("Project conversation runtime ready."),
			CurrentStepChangedAt: &runNow,
		}); runtimeErr == nil {
			live.principal = principal
		}
		return domain.Turn{}, err
	}
	executingStatus := domain.RunStatusExecuting
	_, _ = s.core.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
		RunID:                run.ID,
		Status:               &executingStatus,
		LastHeartbeatAt:      &runNow,
		CurrentStepStatus:    optionalString("turn_executing"),
		CurrentStepSummary:   optionalString("Project conversation turn executing."),
		CurrentStepChangedAt: &runNow,
	})

	go s.consumeTurn(context.WithoutCancel(ctx), conversation, turn, live, run, stream)
	return turn, nil
}

func (s *ProjectConversationService) RespondInterrupt(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	interruptID uuid.UUID,
	response domain.InterruptResponse,
) (domain.PendingInterrupt, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	interrupt, err := s.core.interrupts.GetPendingInterrupt(ctx, interruptID)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	if interrupt.ConversationID != conversation.ID {
		return domain.PendingInterrupt{}, ErrPendingInterruptNotFound
	}
	live, _ := s.runtimeManager.Get(conversationID)
	if live != nil && live.interrupt == nil && live.codex != nil {
		live.interrupt = live.codex
	}
	var project catalogdomain.Project
	var providerItem catalogdomain.AgentProvider
	if live != nil {
		providerItem = live.provider
	}
	if s.core.catalog != nil {
		var projectErr error
		project, projectErr = s.core.catalog.GetProject(ctx, conversation.ProjectID)
		if projectErr != nil {
			return domain.PendingInterrupt{}, fmt.Errorf("get project for interrupt response: %w", projectErr)
		}
		if providerItem.ID == uuid.Nil {
			var providerErr error
			providerItem, providerErr = s.core.catalog.GetAgentProvider(ctx, conversation.ProviderID)
			if providerErr != nil {
				return domain.PendingInterrupt{}, fmt.Errorf("get provider for interrupt response: %w", providerErr)
			}
		}
	}
	if live == nil || live.interrupt == nil {
		if s.core.catalog == nil || providerItem.ID == uuid.Nil {
			return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
		}
		storedFocus, focusErr := s.loadLatestConversationFocus(ctx, conversationID)
		if focusErr != nil {
			return domain.PendingInterrupt{}, focusErr
		}
		var ensureErr error
		live, _, ensureErr = s.ensureLiveRuntime(ctx, conversation, project, providerItem)
		if ensureErr != nil {
			return domain.PendingInterrupt{}, ensureErr
		}
		if live.interrupt == nil {
			return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
		}
		if live.codex != nil {
			systemPrompt, promptErr := s.buildProjectConversationPrompt(ctx, conversation, project, storedFocus, false)
			if promptErr != nil {
				return domain.PendingInterrupt{}, promptErr
			}
			runtimeInput, inputErr := s.buildConversationRuntimeInput(
				ctx,
				conversation,
				project,
				providerItem,
				live.workspace,
				systemPrompt,
				strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID)),
				strings.TrimSpace(stringPointerValue(conversation.LastTurnID)),
				storedFocus,
			)
			if inputErr != nil {
				return domain.PendingInterrupt{}, inputErr
			}
			ensureErr = live.codex.EnsureSession(ctx, runtimeInput)
			if ensureErr != nil {
				if codexadapter.IsThreadNotFoundError(ensureErr) {
					return domain.PendingInterrupt{}, ErrConversationRuntimeAbsent
				}
				return domain.PendingInterrupt{}, ensureErr
			}
		}
	}

	runtimeKind := runtimeInterruptKind(interrupt.Kind)
	storedFocus, err := s.loadLatestConversationFocus(ctx, conversationID)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	environment, err := s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem, storedFocus)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	stream, err := live.interrupt.RespondInterrupt(ctx, RuntimeInterruptResponseInput{
		SessionID:              SessionID(conversationID.String()),
		ProjectID:              project.ID,
		TicketID:               conversationFocusTicketID(storedFocus),
		Provider:               live.provider,
		RequestID:              interrupt.ProviderRequestID,
		Kind:                   runtimeKind,
		Decision:               stringPointerValue(response.Decision),
		Answer:                 cloneMapAny(response.Answer),
		Payload:                cloneMapAny(interrupt.Payload),
		WorkingDirectory:       live.workspace,
		Environment:            environment,
		ResumeProviderThreadID: strings.TrimSpace(stringPointerValue(conversation.ProviderThreadID)),
		ResumeProviderTurnID:   strings.TrimSpace(stringPointerValue(conversation.LastTurnID)),
		PersistentConversation: true,
	})
	if err != nil {
		return domain.PendingInterrupt{}, err
	}

	resolved, _ /* entry */, err := s.core.interrupts.ResolvePendingInterrupt(ctx, interruptID, response)
	if err != nil {
		return domain.PendingInterrupt{}, err
	}
	anchor := RuntimeSessionAnchor{}
	if live.interrupt != nil {
		anchor = live.interrupt.SessionAnchor(SessionID(conversationID.String()))
	}
	_, _ = s.core.conversations.UpdateConversationAnchors(
		ctx,
		conversationID,
		domain.ConversationStatusActive,
		conversationAnchorsFromRuntimeAnchor(anchor, ""),
	)
	s.broadcastConversationEvent(conversation, StreamEvent{
		Event: "interrupt_resolved",
		Payload: map[string]any{
			"interrupt_id": resolved.ID.String(),
			"decision":     stringPointerValue(resolved.Decision),
		},
	})
	if stream.Events != nil {
		run, runErr := s.core.runtimeStore.GetRunByTurnID(ctx, interrupt.TurnID)
		if runErr == nil {
			now := time.Now().UTC()
			executingStatus := domain.RunStatusExecuting
			_, _ = s.core.runtimeStore.UpdateRun(ctx, domain.UpdateRunInput{
				RunID:                run.ID,
				Status:               &executingStatus,
				LastHeartbeatAt:      &now,
				CurrentStepStatus:    optionalString("interrupt_resolved"),
				CurrentStepSummary:   optionalString("Project conversation interrupt resolved."),
				CurrentStepChangedAt: &now,
			})
			if principal, runtimeErr := s.core.runtimeStore.UpdatePrincipalRuntime(ctx, domain.UpdatePrincipalRuntimeInput{
				PrincipalID:          live.principal.ID,
				RuntimeState:         domain.RuntimeStateExecuting,
				CurrentSessionID:     optionalString(conversation.ID.String()),
				CurrentWorkspacePath: optionalString(live.workspace.String()),
				CurrentRunID:         &run.ID,
				LastHeartbeatAt:      &now,
				CurrentStepStatus:    optionalString("turn_executing"),
				CurrentStepSummary:   optionalString("Project conversation turn executing."),
				CurrentStepChangedAt: &now,
			}); runtimeErr == nil {
				live.principal = principal
			}
			go s.consumeTurn(context.WithoutCancel(ctx), conversation, domain.Turn{
				ID:             interrupt.TurnID,
				ConversationID: conversationID,
			}, live, run, stream)
			return resolved, nil
		}
		go s.consumeTurn(context.WithoutCancel(ctx), conversation, domain.Turn{
			ID:             interrupt.TurnID,
			ConversationID: conversationID,
		}, live, domain.ProjectConversationRun{}, stream)
	}
	return resolved, nil
}

func (s *ProjectConversationService) buildConversationRuntimeInput(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	workspace provider.AbsolutePath,
	systemPrompt string,
	resumeThreadID string,
	resumeTurnID string,
	focus *ProjectConversationFocus,
) (RuntimeTurnInput, error) {
	environment, err := s.buildConversationRuntimeEnvironment(ctx, conversation, project, providerItem, focus)
	if err != nil {
		return RuntimeTurnInput{}, err
	}
	return RuntimeTurnInput{
		SessionID:              SessionID(conversation.ID.String()),
		ProjectID:              project.ID,
		TicketID:               conversationFocusTicketID(focus),
		Provider:               providerItem,
		Message:                "",
		SystemPrompt:           systemPrompt,
		WorkingDirectory:       workspace,
		Environment:            environment,
		ResumeProviderThreadID: strings.TrimSpace(resumeThreadID),
		ResumeProviderTurnID:   strings.TrimSpace(resumeTurnID),
		MaxTurns:               0,
		MaxBudgetUSD:           0,
		PersistentConversation: true,
	}, nil
}

func conversationFocusTicketID(focus *ProjectConversationFocus) *uuid.UUID {
	if focus == nil || focus.Ticket == nil || focus.Ticket.ID == uuid.Nil {
		return nil
	}
	copied := focus.Ticket.ID
	return &copied
}

func conversationAnchorsFromRuntimeAnchor(anchor RuntimeSessionAnchor, rollingSummary string) domain.ConversationAnchors {
	activeFlags := append([]string(nil), anchor.ProviderThreadActiveFlags...)
	return domain.ConversationAnchors{
		ProviderThreadID:          optionalString(anchor.ProviderThreadID),
		LastTurnID:                optionalString(anchor.LastTurnID),
		ProviderThreadStatus:      optionalString(anchor.ProviderThreadStatus),
		ProviderThreadActiveFlags: &activeFlags,
		RollingSummary:            rollingSummary,
	}
}

func liveRuntimeSessionAnchor(live *liveProjectConversation, sessionID SessionID) RuntimeSessionAnchor {
	if live == nil {
		return RuntimeSessionAnchor{}
	}
	if live.runtime != nil {
		if anchorer, ok := live.runtime.(projectConversationSessionAnchorer); ok {
			return anchorer.SessionAnchor(sessionID)
		}
	}
	if live.codex != nil {
		return live.codex.SessionAnchor(sessionID)
	}
	return RuntimeSessionAnchor{}
}

func (s *ProjectConversationService) buildProjectConversationPrompt(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
	includeRecovery bool,
) (string, error) {
	return s.buildProjectConversationPromptWithDraft(
		ctx,
		conversation,
		project,
		focus,
		nil,
		includeRecovery,
	)
}

func (s *ProjectConversationService) buildProjectConversationPromptWithDraft(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
	workspaceFileDraft *ProjectConversationWorkspaceFileDraftContext,
	includeRecovery bool,
) (string, error) {
	basePrompt, err := s.core.promptBuilder.buildSystemPrompt(ctx, StartInput{
		Message: "",
		Source:  SourceProjectSidebar,
		Context: Context{
			ProjectID:    project.ID,
			ProjectFocus: focus,
		},
	}, project)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(basePrompt)
	platformContract := agentplatform.BuildCapabilityContract(
		s.projectConversationPlatformContractInput(
			conversation,
			project,
			focus,
			"<runtime-injected>",
			projectConversationPlatformAccessAllowed(project),
		),
	)
	if strings.TrimSpace(platformContract) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(platformContract)
	}
	if ticketFocus := focusTicket(focus); ticketFocus != nil {
		ticketCapsule, capsuleErr := s.renderProjectConversationTicketCapsule(ctx, project, ticketFocus)
		if capsuleErr != nil {
			return "", capsuleErr
		}
		if strings.TrimSpace(ticketCapsule) != "" {
			builder.WriteString("\n\n")
			builder.WriteString(ticketCapsule)
		}
	}
	if workspaceDraftPrompt := renderProjectConversationWorkspaceFileDraftContext(workspaceFileDraft); strings.TrimSpace(workspaceDraftPrompt) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(workspaceDraftPrompt)
	}
	if !includeRecovery {
		return builder.String(), nil
	}

	entries, err := s.core.entries.ListEntries(ctx, conversation.ID)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 && strings.TrimSpace(conversation.RollingSummary) == "" {
		return builder.String(), nil
	}

	builder.WriteString("\n\n## Previous conversation\n")
	if strings.TrimSpace(conversation.RollingSummary) != "" {
		builder.WriteString("Rolling summary:\n")
		builder.WriteString(strings.TrimSpace(conversation.RollingSummary))
		builder.WriteString("\n\n")
	}
	builder.WriteString("Recent transcript:\n")
	for _, line := range renderRecoveryLines(entries, 12) {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	builder.WriteString("\nContinue from this conversation state without restating the entire history.")
	return builder.String(), nil
}

func renderProjectConversationWorkspaceFileDraftContext(
	draft *ProjectConversationWorkspaceFileDraftContext,
) string {
	if draft == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Active workspace file draft\n")
	builder.WriteString("This draft is request-scoped context only. It has not been saved to the workspace yet.\n")
	_, _ = fmt.Fprintf(&builder, "- repo_path: %s\n", draft.RepoPath)
	_, _ = fmt.Fprintf(&builder, "- path: %s\n", draft.Path)
	_, _ = fmt.Fprintf(&builder, "- encoding: %s\n", draft.Encoding)
	_, _ = fmt.Fprintf(&builder, "- line_ending: %s\n", draft.LineEnding)
	builder.WriteString("\nDraft content:\n```text\n")
	builder.WriteString(draft.Content.String())
	if !strings.HasSuffix(draft.Content.String(), "\n") {
		builder.WriteByte('\n')
	}
	builder.WriteString("```\n")
	return builder.String()
}

func (s *ProjectConversationService) projectConversationPlatformContractInput(
	conversation domain.Conversation,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
	token string,
	scopes []string,
) agentplatform.RuntimeContractInput {
	input := agentplatform.RuntimeContractInput{
		PrincipalKind:  agentplatform.PrincipalKindProjectConversation,
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		APIURL:         s.core.platformAPIURL,
		Token:          token,
		Scopes:         scopes,
	}
	if ticketFocus := focusTicket(focus); ticketFocus != nil {
		input.TicketID = ticketFocus.ID
	}
	return input
}

func projectConversationPlatformAccessAllowed(project catalogdomain.Project) []string {
	if len(project.ProjectAIPlatformAccessAllowed) > 0 {
		return append([]string(nil), project.ProjectAIPlatformAccessAllowed...)
	}
	return agentplatform.SupportedScopesForPrincipalKind(agentplatform.PrincipalKindProjectConversation)
}

func runtimeInterruptKind(kind domain.InterruptKind) string {
	switch kind {
	case domain.InterruptKindCommandExecutionApproval:
		return "command_execution"
	case domain.InterruptKindFileChangeApproval:
		return "file_change"
	default:
		return "user_input"
	}
}

func mapDomainInterruptKind(kind string) domain.InterruptKind {
	switch strings.TrimSpace(kind) {
	case codexadapterApprovalKindCommandExecution():
		return domain.InterruptKindCommandExecutionApproval
	case codexadapterApprovalKindFileChange():
		return domain.InterruptKindFileChangeApproval
	default:
		return domain.InterruptKindUserInput
	}
}

func codexadapterApprovalKindCommandExecution() string { return "command_execution" }
func codexadapterApprovalKindFileChange() string       { return "file_change" }

func providerInterruptProviderName(providerItem catalogdomain.AgentProvider) string {
	switch providerItem.AdapterType {
	case catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI:
		return "claude"
	case catalogdomain.AgentProviderAdapterTypeCodexAppServer:
		return "codex"
	case catalogdomain.AgentProviderAdapterTypeGeminiCLI:
		return "gemini"
	default:
		if name := strings.TrimSpace(providerItem.Name); name != "" {
			return name
		}
		if command := strings.TrimSpace(providerItem.CliCommand); command != "" {
			return command
		}
		if adapterType := strings.TrimSpace(string(providerItem.AdapterType)); adapterType != "" {
			return adapterType
		}
		return "provider"
	}
}

func renderRecoveryLines(entries []domain.Entry, limit int) []string {
	if len(entries) == 0 {
		return nil
	}
	if limit <= 0 || len(entries) < limit {
		limit = len(entries)
	}
	selected := entries[len(entries)-limit:]
	lines := make([]string, 0, len(selected))
	for _, entry := range selected {
		switch entry.Kind {
		case domain.EntryKindUserMessage:
			lines = append(lines, "user: "+strings.TrimSpace(stringValue(entry.Payload["content"])))
		case domain.EntryKindAssistantTextDelta:
			lines = append(lines, "assistant: "+strings.TrimSpace(stringValue(entry.Payload["content"])))
		case domain.EntryKindInterrupt:
			lines = append(lines, "system: turn paused for interrupt")
		case domain.EntryKindSystem:
			if stringValue(entry.Payload["type"]) == "turn_interrupted" {
				message := strings.TrimSpace(stringValue(entry.Payload["message"]))
				if message == "" {
					message = "Turn stopped by user."
				}
				lines = append(lines, "system: "+message)
			}
		}
	}
	return lines
}

func buildRollingSummary(entries []domain.Entry) string {
	lines := renderRecoveryLines(entries, 10)
	summary := strings.TrimSpace(strings.Join(lines, "\n"))
	if len(summary) > 4000 {
		return summary[len(summary)-4000:]
	}
	return summary
}

func optionalNonEmptyString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func uuidPointer(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	return &value
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneMapAny(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

func (s *ProjectConversationService) ensureLiveRuntime(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
) (*liveProjectConversation, bool, error) {
	s.runtimeManager.newCodexRuntime = s.newCodexRuntime
	s.runtimeManager.ConfigureGitHubCredentials(s.core.githubAuth)
	s.runtimeManager.ConfigureSkillSync(s.core.skillSync)
	return s.runtimeManager.ensureLiveRuntime(ctx, conversation, project, providerItem)
}

func (s *ProjectConversationService) ensureConversationWorkspace(
	ctx context.Context,
	machine catalogdomain.Machine,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	conversationID uuid.UUID,
) (provider.AbsolutePath, error) {
	s.runtimeManager.ConfigureGitHubCredentials(s.core.githubAuth)
	s.runtimeManager.ConfigureSkillSync(s.core.skillSync)
	return s.runtimeManager.ensureConversationWorkspace(ctx, machine, project, providerItem, conversationID)
}

func (s *ProjectConversationService) buildConversationRuntimeEnvironment(
	ctx context.Context,
	conversation domain.Conversation,
	project catalogdomain.Project,
	providerItem catalogdomain.AgentProvider,
	focus *ProjectConversationFocus,
) ([]string, error) {
	environment := make([]string, 0, 6)
	if providerItem.MachineHost == "" || providerItem.MachineHost == catalogdomain.LocalMachineHost {
		if executable, err := os.Executable(); err == nil && strings.TrimSpace(executable) != "" {
			environment = append(environment, "OPENASE_REAL_BIN="+executable)
		}
	}

	if s != nil && s.core.agentPlatform != nil && s.core.catalog != nil && strings.TrimSpace(s.core.platformAPIURL) != "" {
		principal, err := s.core.runtimeStore.EnsurePrincipal(ctx, domain.EnsurePrincipalInput{
			ConversationID: conversation.ID,
			ProjectID:      conversation.ProjectID,
			ProviderID:     conversation.ProviderID,
			Name:           projectConversationPrincipalName(conversation.ID),
		})
		if err != nil {
			s.logger.Warn("ensure project conversation principal failed", "conversation_id", conversation.ID, "error", err)
		} else {
			scopes := projectConversationPlatformAccessAllowed(project)
			issued, issueErr := s.core.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
				PrincipalKind:  agentplatform.PrincipalKindProjectConversation,
				PrincipalID:    principal.ID,
				PrincipalName:  principal.Name,
				ProjectID:      project.ID,
				ConversationID: conversation.ID,
				Scopes:         scopes,
			})
			if issueErr != nil {
				s.logger.Warn("issue project conversation platform token failed", "conversation_id", conversation.ID, "error", issueErr)
			} else {
				contractScopes := issued.Scopes
				if len(contractScopes) == 0 {
					contractScopes = scopes
				}
				environment = append(environment, agentplatform.BuildRuntimeEnvironment(
					s.projectConversationPlatformContractInput(conversation, project, focus, issued.Token, contractScopes),
				)...)
			}
		}
	}

	runtimeSecrets, err := s.buildConversationSecretEnvironment(ctx, project, focus)
	if err != nil {
		return nil, err
	}
	environment = append(environment, runtimeSecrets...)
	return environment, nil
}

func (s *ProjectConversationService) buildConversationSecretEnvironment(
	ctx context.Context,
	project catalogdomain.Project,
	focus *ProjectConversationFocus,
) ([]string, error) {
	if s == nil || s.core.secretManager == nil {
		return nil, nil
	}

	var ticketID *uuid.UUID
	if focus != nil && focus.Kind == ProjectConversationFocusTicket && focus.Ticket != nil {
		ticketID = uuidPointer(focus.Ticket.ID)
	}

	resolved, err := s.core.secretManager.ResolveBoundForRuntime(ctx, secretsservice.ResolveBoundRuntimeInput{
		ProjectID: project.ID,
		TicketID:  ticketID,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve conversation secret bindings: %w", err)
	}
	environment, err := secretsservice.BuildRuntimeEnvironment(resolved)
	if err != nil {
		return nil, fmt.Errorf("build conversation secret environment: %w", err)
	}
	return environment, nil
}

func conversationWorkspaceArtifactPaths(workspaceRoot string, adapterType string) []string {
	paths := []string{}
	for _, relative := range []string{".openase", skillTargetRelativePath(workspaceRoot, adapterType)} {
		if strings.TrimSpace(relative) == "" {
			continue
		}
		absolute := filepath.Join(workspaceRoot, relative)
		if _, err := os.Stat(absolute); err == nil {
			paths = append(paths, relative)
		}
	}
	return paths
}

func skillTargetRelativePath(workspaceRoot string, adapterType string) string {
	target, err := workflowservice.ResolveSkillTargetForRuntime(workspaceRoot, adapterType)
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(workspaceRoot, target.SkillsDir)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(relative)
}

func mapConversationWorkspaceRepos(items []catalogdomain.ProjectRepo) []workspaceinfra.RepoInput {
	repos := make([]workspaceinfra.RepoInput, 0, len(items))
	for _, item := range items {
		repo := workspaceinfra.RepoInput{
			Name:          item.Name,
			RepositoryURL: item.RepositoryURL,
			DefaultBranch: item.DefaultBranch,
		}
		if branchName := strings.TrimSpace(item.DefaultBranch); branchName != "" {
			value := branchName
			repo.BranchName = &value
		}
		if workspaceDirname := strings.TrimSpace(item.WorkspaceDirname); workspaceDirname != "" && workspaceDirname != strings.TrimSpace(item.Name) {
			value := workspaceDirname
			repo.WorkspaceDirname = &value
		}
		repos = append(repos, repo)
	}
	return repos
}

func (s *ProjectConversationService) applyGitHubWorkspaceAuth(
	ctx context.Context,
	projectID uuid.UUID,
	request workspaceinfra.SetupRequest,
) (workspaceinfra.SetupRequest, error) {
	if s == nil {
		return request, nil
	}
	return githubauthservice.ApplyWorkspaceAuth(ctx, s.core.githubAuth, projectID, request)
}

func projectConversationPrincipalName(conversationID uuid.UUID) string {
	return "project-conversation:" + strings.TrimSpace(conversationID.String())
}

func projectConversationWorkspaceName(conversationID uuid.UUID) string {
	return "conv-" + strings.TrimSpace(conversationID.String())
}

func newCodexAdapterForManager(manager provider.AgentCLIProcessManager) (*codexadapter.Adapter, error) {
	return codexadapter.NewAdapter(codexadapter.AdapterOptions{ProcessManager: manager})
}

func newClaudeAdapterForManager(manager provider.AgentCLIProcessManager) provider.ClaudeCodeAdapter {
	return claudecodeadapter.NewAdapter(manager)
}
