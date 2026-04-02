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
  }
}

export function describeProjectAIFocus(focus: ProjectAIFocus): ProjectAIFocusCard {
  switch (focus.kind) {
    case 'workflow': {
      const detail = [
        focus.workflowType,
        focus.selectedArea ?? 'harness',
        focus.hasDirtyDraft ? 'unsaved draft' : '',
      ]
        .filter(Boolean)
        .join(' · ')
      return {
        label: 'Workflow',
        title: `${focus.workflowName} / ${focus.selectedArea ?? 'harness'}`,
        detail,
      }
    }
    case 'skill': {
      const detail = [
        focus.boundWorkflowNames.length > 0
          ? `bound to ${focus.boundWorkflowNames.join(', ')}`
          : 'not bound to a workflow',
        focus.hasDirtyDraft ? 'unsaved draft' : '',
      ]
        .filter(Boolean)
        .join(' · ')
      return {
        label: 'Skill',
        title: `${focus.skillName} / ${focus.selectedFilePath}`,
        detail,
      }
    }
    case 'ticket':
      return {
        label: 'Ticket',
        title: `${focus.ticketIdentifier} / ${focus.ticketTitle}`,
        detail: [focus.ticketStatus, focus.selectedArea ?? 'detail'].filter(Boolean).join(' · '),
      }
    case 'machine':
      return {
        label: 'Machine',
        title: `${focus.machineName} / ${focus.selectedArea ?? 'health'}`,
        detail: [focus.machineHost, focus.machineStatus, focus.healthSummary]
          .filter(Boolean)
          .join(' · '),
      }
  }
}
