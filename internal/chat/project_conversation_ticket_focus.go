package chat

import (
	"context"
	"fmt"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

func (s *ProjectConversationService) loadLatestConversationFocus(
	ctx context.Context,
	conversationID uuid.UUID,
) (*ProjectConversationFocus, error) {
	if s == nil || s.entries == nil {
		return nil, nil
	}

	entries, err := s.entries.ListEntries(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	for index := len(entries) - 1; index >= 0; index-- {
		entry := entries[index]
		if entry.Kind != domain.EntryKindSystem {
			continue
		}
		focus, matched, err := parseProjectConversationFocusSnapshot(entry.Payload)
		if !matched {
			continue
		}
		if err != nil {
			return nil, err
		}
		return focus, nil
	}
	return nil, nil
}

func focusTicket(focus *ProjectConversationFocus) *ProjectConversationTicketFocus {
	if focus == nil || focus.Kind != ProjectConversationFocusTicket {
		return nil
	}
	return focus.Ticket
}

func (s *ProjectConversationService) renderProjectConversationTicketCapsule(
	ctx context.Context,
	project catalogdomain.Project,
	focus *ProjectConversationTicketFocus,
) (string, error) {
	if s == nil || s.promptBuilder == nil || focus == nil {
		return "", nil
	}

	contextItem, err := s.promptBuilder.loadTicketPromptContext(ctx, project.ID, focus.ID)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("## Ticket Capsule\n")
	_, _ = fmt.Fprintf(&builder, "项目: %s\n", project.Name)
	s.promptBuilder.writeTicketPromptContext(&builder, contextItem)
	if focus.SelectedArea != "" {
		_, _ = fmt.Fprintf(&builder, "\n### 当前 ticket 子区域\n- selected_area: %s\n", focus.SelectedArea)
	}
	builder.WriteString("\n### 运行时摘要\n")
	builder.WriteString(renderProjectConversationTicketRuntimeSummary(ctx, contextItem.Ticket, focus, s.catalog))
	return builder.String(), nil
}

func renderProjectConversationTicketRuntimeSummary(
	ctx context.Context,
	ticketItem ticketservice.Ticket,
	focus *ProjectConversationTicketFocus,
	catalog projectConversationCatalog,
) string {
	var builder strings.Builder

	if focus != nil && focus.AssignedAgent != nil {
		_, _ = fmt.Fprintf(&builder, "- assigned_agent: %s\n", focus.AssignedAgent.Name)
		if focus.AssignedAgent.Provider != "" {
			_, _ = fmt.Fprintf(&builder, "- assigned_provider: %s\n", focus.AssignedAgent.Provider)
		}
		if focus.AssignedAgent.RuntimeControlState != "" {
			_, _ = fmt.Fprintf(&builder, "- agent_runtime_control_state: %s\n", focus.AssignedAgent.RuntimeControlState)
		}
		if focus.AssignedAgent.RuntimePhase != "" {
			_, _ = fmt.Fprintf(&builder, "- agent_runtime_phase: %s\n", focus.AssignedAgent.RuntimePhase)
		}
	} else {
		builder.WriteString("- assigned_agent: unassigned\n")
	}

	_, _ = fmt.Fprintf(&builder, "- retry_paused: %t\n", ticketItem.RetryPaused)
	_, _ = fmt.Fprintf(&builder, "- consecutive_errors: %d\n", ticketItem.ConsecutiveErrors)
	if ticketItem.PauseReason != "" {
		_, _ = fmt.Fprintf(&builder, "- pause_reason: %s\n", ticketItem.PauseReason)
	}

	if focus != nil && focus.CurrentRun != nil {
		if focus.CurrentRun.ID != "" {
			_, _ = fmt.Fprintf(&builder, "- current_run_id: %s\n", focus.CurrentRun.ID)
		}
		if focus.CurrentRun.AttemptNumber > 0 {
			_, _ = fmt.Fprintf(&builder, "- current_run_attempt: %d\n", focus.CurrentRun.AttemptNumber)
		}
		if focus.CurrentRun.Status != "" {
			_, _ = fmt.Fprintf(&builder, "- current_run_status: %s\n", focus.CurrentRun.Status)
		}
		if focus.CurrentRun.CurrentStepStatus != "" {
			_, _ = fmt.Fprintf(&builder, "- current_run_step_status: %s\n", focus.CurrentRun.CurrentStepStatus)
		}
		if focus.CurrentRun.CurrentStepSummary != "" {
			_, _ = fmt.Fprintf(&builder, "- current_run_step_summary: %s\n", focus.CurrentRun.CurrentStepSummary)
		}
		if focus.CurrentRun.LastError != "" {
			_, _ = fmt.Fprintf(&builder, "- current_run_last_error: %s\n", focus.CurrentRun.LastError)
		}
	} else if ticketItem.CurrentRunID != nil {
		_, _ = fmt.Fprintf(&builder, "- current_run_id: %s\n", ticketItem.CurrentRunID.String())
	}

	if focus != nil && focus.TargetMachine != nil {
		if focus.TargetMachine.ID != "" {
			_, _ = fmt.Fprintf(&builder, "- target_machine_id: %s\n", focus.TargetMachine.ID)
		}
		if focus.TargetMachine.Name != "" {
			_, _ = fmt.Fprintf(&builder, "- target_machine_name: %s\n", focus.TargetMachine.Name)
		}
		if focus.TargetMachine.Host != "" {
			_, _ = fmt.Fprintf(&builder, "- target_machine_host: %s\n", focus.TargetMachine.Host)
		}
	} else if ticketItem.TargetMachineID != nil {
		_, _ = fmt.Fprintf(&builder, "- target_machine_id: %s\n", ticketItem.TargetMachineID.String())
		if catalog != nil {
			if machine, err := catalog.GetMachine(ctx, *ticketItem.TargetMachineID); err == nil {
				if machine.Name != "" {
					_, _ = fmt.Fprintf(&builder, "- target_machine_name: %s\n", machine.Name)
				}
				if machine.Host != "" {
					_, _ = fmt.Fprintf(&builder, "- target_machine_host: %s\n", machine.Host)
				}
			}
		}
	}

	if builder.Len() == 0 {
		return "- 无\n"
	}
	return builder.String()
}
