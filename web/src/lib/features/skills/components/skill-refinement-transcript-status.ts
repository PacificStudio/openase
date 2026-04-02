import type {
  SkillRefinementSessionAnchorPayload,
  SkillRefinementStreamEvent,
} from '$lib/api/skill-refinement'
import {
  createProjectConversationErrorEntry,
  createProjectConversationTaskStatusEntry,
  createProjectConversationTurnDoneEntry,
} from '$lib/features/chat'

export function mapSkillRefinementStatusEvent(
  event: Extract<SkillRefinementStreamEvent, { kind: 'status' }>,
  id: string,
) {
  switch (event.payload.phase) {
    case 'editing':
      return createProjectConversationTaskStatusEntry({
        id,
        statusType: 'task_started',
        title: 'Draft refinement started',
        detail: event.payload.message,
      })
    case 'testing':
      return createProjectConversationTaskStatusEntry({
        id,
        statusType: 'task_progress',
        title: 'Verification running',
        detail: event.payload.message,
      })
    case 'retrying':
      return createProjectConversationTaskStatusEntry({
        id,
        statusType: 'task_notification',
        title: 'Retrying refinement',
        detail: event.payload.message,
      })
    case 'verified':
      return createProjectConversationTurnDoneEntry({ id })
    case 'blocked':
    case 'unverified':
      return createProjectConversationErrorEntry({
        id,
        message: event.payload.message,
      })
    default:
      return null
  }
}

export function formatSkillRefinementPlanDetail(
  explanation: string | undefined,
  plan: Array<{ step: string; status: string }>,
) {
  const steps = plan
    .map((item) => `${item.status.replaceAll('_', ' ')}: ${item.step}`)
    .filter((item) => item.trim() !== '')
  return [explanation, steps.join('\n')].filter(Boolean).join('\n\n') || undefined
}

export function formatSkillRefinementAnchorTitle(anchor: SkillRefinementSessionAnchorPayload) {
  switch ((anchor.providerAnchorKind || '').trim()) {
    case 'session':
      return 'Provider session anchored'
    case 'thread':
      return 'Provider thread anchored'
    default:
      return 'Provider anchor updated'
  }
}

export function formatSkillRefinementAnchorDetail(anchor: SkillRefinementSessionAnchorPayload) {
  const lines: string[] = []
  if (anchor.providerAnchorId) lines.push(`anchor: ${anchor.providerAnchorId}`)
  if (anchor.providerTurnId) lines.push(`turn: ${anchor.providerTurnId}`)
  if (typeof anchor.providerTurnSupported === 'boolean') {
    lines.push(`turn support: ${anchor.providerTurnSupported ? 'yes' : 'no'}`)
  }
  return lines.join('\n') || undefined
}
