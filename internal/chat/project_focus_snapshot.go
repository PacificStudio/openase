package chat

import (
	"encoding/json"
	"strings"
)

const projectConversationFocusSnapshotType = "focus_snapshot"

func serializeProjectConversationFocus(focus *ProjectConversationFocus) map[string]any {
	payload := map[string]any{
		"type": projectConversationFocusSnapshotType,
	}

	raw := rawProjectConversationFocusFromFocus(focus)
	if raw == nil {
		payload["focus"] = nil
		return payload
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		payload["focus"] = nil
		return payload
	}

	var snapshot map[string]any
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		payload["focus"] = nil
		return payload
	}
	payload["focus"] = snapshot
	return payload
}

func parseProjectConversationFocusSnapshot(payload map[string]any) (*ProjectConversationFocus, bool, error) {
	if strings.TrimSpace(stringValue(payload["type"])) != projectConversationFocusSnapshotType {
		return nil, false, nil
	}

	rawFocus, ok := payload["focus"]
	if !ok || rawFocus == nil {
		return nil, true, nil
	}

	encoded, err := json.Marshal(rawFocus)
	if err != nil {
		return nil, true, err
	}

	var raw RawProjectConversationFocus
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return nil, true, err
	}

	focus, err := ParseProjectConversationFocus(&raw)
	if err != nil {
		return nil, true, err
	}
	return focus, true, nil
}

func rawProjectConversationFocusFromFocus(focus *ProjectConversationFocus) *RawProjectConversationFocus {
	if focus == nil {
		return nil
	}

	switch focus.Kind {
	case ProjectConversationFocusWorkflow:
		if focus.Workflow == nil {
			return nil
		}
		return &RawProjectConversationFocus{
			Kind:          string(ProjectConversationFocusWorkflow),
			WorkflowID:    optionalString(focus.Workflow.ID.String()),
			WorkflowName:  optionalString(focus.Workflow.Name),
			WorkflowType:  optionalString(focus.Workflow.Type),
			HarnessPath:   optionalString(focus.Workflow.HarnessPath),
			IsActive:      projectFocusBoolPointer(focus.Workflow.IsActive),
			SelectedArea:  optionalString(focus.Workflow.SelectedArea),
			HasDirtyDraft: projectFocusBoolPointer(focus.Workflow.HasDirtyDraft),
		}
	case ProjectConversationFocusSkill:
		if focus.Skill == nil {
			return nil
		}
		return &RawProjectConversationFocus{
			Kind:               string(ProjectConversationFocusSkill),
			SkillID:            optionalString(focus.Skill.ID.String()),
			SkillName:          optionalString(focus.Skill.Name),
			SelectedFilePath:   optionalString(focus.Skill.SelectedFilePath),
			BoundWorkflowNames: append([]string(nil), focus.Skill.BoundWorkflowNames...),
			HasDirtyDraft:      projectFocusBoolPointer(focus.Skill.HasDirtyDraft),
		}
	case ProjectConversationFocusTicket:
		if focus.Ticket == nil {
			return nil
		}
		return &RawProjectConversationFocus{
			Kind:                 string(ProjectConversationFocusTicket),
			TicketID:             optionalString(focus.Ticket.ID.String()),
			TicketIdentifier:     optionalString(focus.Ticket.Identifier),
			TicketTitle:          optionalString(focus.Ticket.Title),
			TicketDescription:    optionalString(focus.Ticket.Description),
			TicketStatus:         optionalString(focus.Ticket.Status),
			TicketPriority:       optionalString(focus.Ticket.Priority),
			TicketAttemptCount:   projectFocusIntPointer(focus.Ticket.AttemptCount),
			TicketRetryPaused:    projectFocusBoolPointer(focus.Ticket.RetryPaused),
			TicketPauseReason:    optionalString(focus.Ticket.PauseReason),
			SelectedArea:         optionalString(focus.Ticket.SelectedArea),
			TicketDependencies:   rawTicketDependenciesFromFocus(focus.Ticket.Dependencies),
			TicketRepoScopes:     rawTicketRepoScopesFromFocus(focus.Ticket.RepoScopes),
			TicketRecentActivity: rawTicketActivityFromFocus(focus.Ticket.RecentActivity),
			TicketHookHistory:    rawTicketHooksFromFocus(focus.Ticket.HookHistory),
			TicketAssignedAgent:  rawTicketAssignedAgentFromFocus(focus.Ticket.AssignedAgent),
			TicketCurrentRun:     rawTicketRunFromFocus(focus.Ticket.CurrentRun),
			TicketTargetMachine:  rawTicketTargetMachineFromFocus(focus.Ticket.TargetMachine),
		}
	case ProjectConversationFocusMachine:
		if focus.Machine == nil {
			return nil
		}
		return &RawProjectConversationFocus{
			Kind:          string(ProjectConversationFocusMachine),
			MachineID:     optionalString(focus.Machine.ID.String()),
			MachineName:   optionalString(focus.Machine.Name),
			MachineHost:   optionalString(focus.Machine.Host),
			MachineStatus: optionalString(focus.Machine.Status),
			SelectedArea:  optionalString(focus.Machine.SelectedArea),
			HealthSummary: optionalString(focus.Machine.HealthSummary),
		}
	case ProjectConversationFocusWorkspace:
		if focus.Workspace == nil {
			return nil
		}
		var workingSet []RawProjectConversationWorkspaceWorkingSet
		if len(focus.Workspace.WorkingSet) > 0 {
			workingSet = make([]RawProjectConversationWorkspaceWorkingSet, 0, len(focus.Workspace.WorkingSet))
			for _, item := range focus.Workspace.WorkingSet {
				workingSet = append(workingSet, RawProjectConversationWorkspaceWorkingSet{
					FilePath:       optionalString(item.FilePath),
					ContentExcerpt: optionalString(item.ContentExcerpt),
					Dirty:          projectFocusBoolPointer(item.Dirty),
					Truncated:      projectFocusBoolPointer(item.Truncated),
				})
			}
		}
		return &RawProjectConversationFocus{
			Kind:                            string(ProjectConversationFocusWorkspace),
			ConversationID:                  optionalString(focus.Workspace.ConversationID.String()),
			WorkspaceRepoPath:               optionalString(focus.Workspace.RepoPath),
			WorkspaceFilePath:               optionalString(focus.Workspace.FilePath),
			SelectedArea:                    optionalString(focus.Workspace.SelectedArea),
			HasDirtyDraft:                   projectFocusBoolPointer(focus.Workspace.HasDirtyDraft),
			WorkspaceSelectionFrom:          projectFocusIntPtrFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) int { return item.From }),
			WorkspaceSelectionTo:            projectFocusIntPtrFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) int { return item.To }),
			WorkspaceSelectionStartLine:     projectFocusIntPtrFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) int { return item.StartLine }),
			WorkspaceSelectionStartColumn:   projectFocusIntPtrFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) int { return item.StartColumn }),
			WorkspaceSelectionEndLine:       projectFocusIntPtrFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) int { return item.EndLine }),
			WorkspaceSelectionEndColumn:     projectFocusIntPtrFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) int { return item.EndColumn }),
			WorkspaceSelectionText:          optionalStringFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) string { return item.Text }),
			WorkspaceSelectionContextBefore: optionalStringFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) string { return item.ContextBefore }),
			WorkspaceSelectionContextAfter:  optionalStringFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) string { return item.ContextAfter }),
			WorkspaceSelectionTruncated:     projectFocusBoolPtrFromSelection(focus.Workspace.Selection, func(item *ProjectConversationWorkspaceSelection) bool { return item.Truncated }),
			WorkspaceWorkingSet:             workingSet,
		}
	default:
		return nil
	}
}

func rawTicketDependenciesFromFocus(items []ProjectConversationTicketDependency) []RawProjectConversationTicketDependency {
	result := make([]RawProjectConversationTicketDependency, 0, len(items))
	for _, item := range items {
		result = append(result, RawProjectConversationTicketDependency{
			Identifier: optionalString(item.Identifier),
			Title:      optionalString(item.Title),
			Relation:   optionalString(item.Relation),
			Status:     optionalString(item.Status),
		})
	}
	return result
}

func rawTicketRepoScopesFromFocus(items []ProjectConversationTicketRepoScope) []RawProjectConversationTicketRepoScope {
	result := make([]RawProjectConversationTicketRepoScope, 0, len(items))
	for _, item := range items {
		result = append(result, RawProjectConversationTicketRepoScope{
			RepoID:         optionalString(item.RepoID),
			RepoName:       optionalString(item.RepoName),
			BranchName:     optionalString(item.BranchName),
			PullRequestURL: optionalString(item.PullRequestURL),
		})
	}
	return result
}

func rawTicketActivityFromFocus(items []ProjectConversationTicketActivity) []RawProjectConversationTicketActivity {
	result := make([]RawProjectConversationTicketActivity, 0, len(items))
	for _, item := range items {
		result = append(result, RawProjectConversationTicketActivity{
			EventType: optionalString(item.EventType),
			Message:   optionalString(item.Message),
			CreatedAt: optionalString(item.CreatedAt),
		})
	}
	return result
}

func rawTicketHooksFromFocus(items []ProjectConversationTicketHook) []RawProjectConversationTicketHook {
	result := make([]RawProjectConversationTicketHook, 0, len(items))
	for _, item := range items {
		result = append(result, RawProjectConversationTicketHook{
			HookName:  optionalString(item.HookName),
			Status:    optionalString(item.Status),
			Output:    optionalString(item.Output),
			Timestamp: optionalString(item.Timestamp),
		})
	}
	return result
}

func rawTicketAssignedAgentFromFocus(item *ProjectConversationTicketAssignedAgent) *RawProjectConversationTicketAssignedAgent {
	if item == nil {
		return nil
	}
	return &RawProjectConversationTicketAssignedAgent{
		ID:                  optionalString(item.ID),
		Name:                optionalString(item.Name),
		Provider:            optionalString(item.Provider),
		RuntimeControlState: optionalString(item.RuntimeControlState),
		RuntimePhase:        optionalString(item.RuntimePhase),
	}
}

func rawTicketRunFromFocus(item *ProjectConversationTicketRun) *RawProjectConversationTicketRun {
	if item == nil {
		return nil
	}
	return &RawProjectConversationTicketRun{
		ID:                 optionalString(item.ID),
		AttemptNumber:      projectFocusIntPointer(item.AttemptNumber),
		Status:             optionalString(item.Status),
		CurrentStepStatus:  optionalString(item.CurrentStepStatus),
		CurrentStepSummary: optionalString(item.CurrentStepSummary),
		LastError:          optionalString(item.LastError),
	}
}

func rawTicketTargetMachineFromFocus(item *ProjectConversationTicketTargetMachine) *RawProjectConversationTicketTargetMachine {
	if item == nil {
		return nil
	}
	return &RawProjectConversationTicketTargetMachine{
		ID:   optionalString(item.ID),
		Name: optionalString(item.Name),
		Host: optionalString(item.Host),
	}
}

func projectFocusBoolPointer(value bool) *bool {
	return &value
}

func projectFocusIntPointer(value int) *int {
	return &value
}

func projectFocusIntPtrFromSelection(
	selection *ProjectConversationWorkspaceSelection,
	getter func(*ProjectConversationWorkspaceSelection) int,
) *int {
	if selection == nil {
		return nil
	}
	return projectFocusIntPointer(getter(selection))
}

func projectFocusBoolPtrFromSelection(
	selection *ProjectConversationWorkspaceSelection,
	getter func(*ProjectConversationWorkspaceSelection) bool,
) *bool {
	if selection == nil {
		return nil
	}
	return projectFocusBoolPointer(getter(selection))
}

func optionalStringFromSelection(
	selection *ProjectConversationWorkspaceSelection,
	getter func(*ProjectConversationWorkspaceSelection) string,
) *string {
	if selection == nil {
		return nil
	}
	return optionalString(getter(selection))
}
