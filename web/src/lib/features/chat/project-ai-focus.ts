import type { TranslationKey } from '$lib/i18n'
import { chatT } from './i18n'

export type ProjectAIFocus =
  | {
      kind: 'workflow'
      projectId: string
      workflowId: string
      workflowName: string
      workflowType: string
      harnessPath: string
      isActive: boolean
      selectedArea?: string
      hasDirtyDraft?: boolean
    }
  | {
      kind: 'skill'
      projectId: string
      skillId: string
      skillName: string
      selectedFilePath: string
      boundWorkflowNames: string[]
      hasDirtyDraft?: boolean
    }
  | {
      kind: 'ticket'
      projectId: string
      ticketId: string
      ticketIdentifier: string
      ticketTitle: string
      ticketDescription?: string
      ticketStatus: string
      ticketPriority?: string
      ticketAttemptCount?: number
      ticketRetryPaused?: boolean
      ticketPauseReason?: string
      ticketDependencies?: Array<{
        identifier: string
        title: string
        relation?: string
        status?: string
      }>
      ticketRepoScopes?: Array<{
        repoId?: string
        repoName?: string
        branchName?: string
        pullRequestUrl?: string
      }>
      ticketRecentActivity?: Array<{
        eventType?: string
        message: string
        createdAt?: string
      }>
      ticketHookHistory?: Array<{
        hookName?: string
        status?: string
        output?: string
        timestamp?: string
      }>
      ticketAssignedAgent?: {
        id?: string
        name?: string
        provider?: string
        runtimeControlState?: string
        runtimePhase?: string
        status?: string
        activeRunCount?: number
      }
      ticketCurrentRun?: {
        id?: string
        attemptNumber?: number
        status?: string
        currentStepStatus?: string
        currentStepSummary?: string
        lastError?: string
      }
      ticketTargetMachine?: {
        id?: string
        name?: string
        host?: string
      }
      selectedArea?: string
    }
  | {
      kind: 'machine'
      projectId: string
      machineId: string
      machineName: string
      machineHost: string
      machineStatus?: string
      selectedArea?: string
      healthSummary?: string
    }
  | {
      kind: 'workspace_file'
      projectId: string
      conversationId: string
      repoPath: string
      filePath: string
      selectedArea?: string
      hasDirtyDraft?: boolean
      draftContent?: string
      encoding?: 'utf-8'
      lineEnding?: 'lf' | 'crlf'
      selection?: {
        from: number
        to: number
        startLine: number
        startColumn: number
        endLine: number
        endColumn: number
        text: string
        contextBefore: string
        contextAfter: string
        truncated: boolean
      } | null
      workingSet?: Array<{
        filePath: string
        contentExcerpt: string
        dirty: boolean
        truncated: boolean
      }>
    }

export type ProjectAIFocusCard = {
  label: string
  title: string
  detail?: string
}

export const PROJECT_AI_FOCUS_PRIORITY = {
  workspace: 20,
  overlay: 30,
} as const

export function projectAIFocusKey(focus: ProjectAIFocus | null | undefined): string {
  if (!focus) {
    return ''
  }

  switch (focus.kind) {
    case 'workflow':
      return [
        focus.kind,
        focus.projectId,
        focus.workflowId,
        focus.selectedArea ?? '',
        focus.hasDirtyDraft ? 'dirty' : 'clean',
      ].join(':')
    case 'skill':
      return [
        focus.kind,
        focus.projectId,
        focus.skillId,
        focus.selectedFilePath,
        focus.hasDirtyDraft ? 'dirty' : 'clean',
      ].join(':')
    case 'ticket':
      return [focus.kind, focus.projectId, focus.ticketId, focus.selectedArea ?? ''].join(':')
    case 'machine':
      return [focus.kind, focus.projectId, focus.machineId, focus.selectedArea ?? ''].join(':')
    case 'workspace_file':
      return [
        focus.kind,
        focus.projectId,
        focus.conversationId,
        focus.repoPath,
        focus.filePath,
        focus.selectedArea ?? '',
        focus.hasDirtyDraft ? 'dirty' : 'clean',
      ].join(':')
  }
}

function focusAreaOrFallback(value: string | undefined, fallbackKey: TranslationKey) {
  return value ?? chatT(fallbackKey)
}

export function describeProjectAIFocus(focus: ProjectAIFocus): ProjectAIFocusCard {
  switch (focus.kind) {
    case 'workflow': {
      const areaLabel = focusAreaOrFallback(focus.selectedArea, 'chat.focus.area.harness')
      const detail = [
        focus.workflowType,
        areaLabel,
        focus.hasDirtyDraft ? chatT('chat.focus.unsavedDraft') : '',
      ]
        .filter(Boolean)
        .join(' · ')
      return {
        label: chatT('chat.focus.label.workflow'),
        title: `${focus.workflowName} / ${areaLabel}`,
        detail,
      }
    }
    case 'skill': {
      const detail = [
        focus.boundWorkflowNames.length > 0
          ? chatT('chat.focus.boundToWorkflow', {
              workflows: focus.boundWorkflowNames.join(', '),
            })
          : chatT('chat.focus.notBoundToWorkflow'),
        focus.hasDirtyDraft ? chatT('chat.focus.unsavedDraft') : '',
      ]
        .filter(Boolean)
        .join(' · ')
      return {
        label: chatT('chat.focus.label.skill'),
        title: `${focus.skillName} / ${focus.selectedFilePath}`,
        detail,
      }
    }
    case 'ticket':
      return {
        label: chatT('chat.focus.label.ticket'),
        title: `${focus.ticketIdentifier} / ${focus.ticketTitle}`,
        detail: [
          focus.ticketStatus,
          focusAreaOrFallback(focus.selectedArea, 'chat.focus.area.detail'),
        ]
          .filter(Boolean)
          .join(' · '),
      }
    case 'machine':
      return {
        label: chatT('chat.focus.label.machine'),
        title: `${focus.machineName} / ${focusAreaOrFallback(focus.selectedArea, 'chat.focus.area.health')}`,
        detail: [focus.machineHost, focus.machineStatus, focus.healthSummary]
          .filter(Boolean)
          .join(' · '),
      }
    case 'workspace_file': {
      const detail = [
        focusAreaOrFallback(focus.selectedArea, 'chat.focus.area.edit'),
        focus.hasDirtyDraft ? chatT('chat.focus.unsavedDraft') : chatT('chat.focus.saved'),
        focus.selection
          ? chatT('chat.focus.selectionLines', {
              startLine: focus.selection.startLine,
              endLine: focus.selection.endLine,
            })
          : '',
      ]
        .filter(Boolean)
        .join(' · ')
      return {
        label: chatT('chat.focus.label.workspaceFile'),
        title: `${focus.repoPath} / ${focus.filePath}`,
        detail,
      }
    }
  }
}
