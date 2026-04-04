import type { SkillRefinementStreamEvent } from '$lib/api/skill-refinement'
import {
  appendProjectConversationTextEntry,
  appendProjectConversationTranscriptEntry,
  createProjectConversationDiffEntriesFromUnifiedDiff,
  createProjectConversationErrorEntry,
  createProjectConversationInterruptEntry,
  createProjectConversationTaskStatusEntry,
  createProjectConversationTurnDoneEntry,
  mapProjectConversationTaskEntry,
  type ProjectConversationTranscriptEntry,
} from '$lib/features/chat'
import { mapSkillRefinementMessageEvent } from './skill-refinement-transcript-message'
import {
  formatSkillRefinementAnchorDetail,
  formatSkillRefinementAnchorTitle,
  formatSkillRefinementPlanDetail,
  mapSkillRefinementStatusEvent,
} from './skill-refinement-transcript-status'

export type SkillRefinementTranscriptState = {
  entries: ProjectConversationTranscriptEntry[]
  nextEntryNumber: number
}

export type SkillRefinementAnchorState = {
  providerName?: string
  anchorKind?: string
  anchorId?: string
  turnId?: string
}

export function createSkillRefinementTranscriptState(): SkillRefinementTranscriptState {
  return {
    entries: [],
    nextEntryNumber: 0,
  }
}

export function appendSkillRefinementTranscriptEvent(
  state: SkillRefinementTranscriptState,
  event: SkillRefinementStreamEvent,
): SkillRefinementTranscriptState {
  let nextState = { ...state }

  const appendEntry = (entry: ProjectConversationTranscriptEntry) => {
    nextState = {
      ...nextState,
      entries: appendProjectConversationTranscriptEntry(nextState.entries, entry),
    }
  }

  const nextEntryId = () => `skill-refinement-${nextState.nextEntryNumber + 1}`
  const consumeEntryId = () => {
    nextState = { ...nextState, nextEntryNumber: nextState.nextEntryNumber + 1 }
    return `skill-refinement-${nextState.nextEntryNumber}`
  }

  switch (event.kind) {
    case 'session':
      return nextState
    case 'status': {
      const mapped = mapSkillRefinementStatusEvent(event, consumeEntryId())
      if (mapped) {
        appendEntry(mapped)
      }
      return nextState
    }
    case 'message': {
      const mapped = mapSkillRefinementMessageEvent(event.payload, nextEntryId())
      if (!mapped) {
        return nextState
      }
      if (mapped.kind === 'text') {
        nextState = {
          ...nextState,
          nextEntryNumber: nextState.nextEntryNumber + 1,
          entries: appendProjectConversationTextEntry(
            nextState.entries,
            'assistant',
            mapped.content,
            {
              entryId: `skill-refinement-${nextState.nextEntryNumber}`,
              streaming: false,
            },
          ),
        }
        return nextState
      }
      appendEntry({ ...mapped, id: consumeEntryId() })
      return nextState
    }
    case 'interrupt_requested':
      appendEntry(
        createProjectConversationInterruptEntry({
          id: consumeEntryId(),
          interruptId: event.payload.requestId,
          provider: 'skill-refinement',
          interruptKind: event.payload.kind,
          payload: event.payload.payload,
          options: event.payload.options,
        }),
      )
      return nextState
    case 'thread_status': {
      const mapped = mapProjectConversationTaskEntry({
        id: consumeEntryId(),
        type: 'thread_status',
        raw: {
          thread_id: event.payload.threadId,
          status: event.payload.status,
          active_flags: event.payload.activeFlags,
          entry_id: event.payload.entryId,
        },
      })
      if (mapped) {
        appendEntry(mapped)
      }
      return nextState
    }
    case 'session_state': {
      const mapped = mapProjectConversationTaskEntry({
        id: consumeEntryId(),
        type: 'session_state',
        raw: {
          status: event.payload.status,
          active_flags: event.payload.activeFlags,
          detail: event.payload.detail,
          raw: event.payload.raw,
          entry_id: event.payload.entryId,
        },
      })
      if (mapped) {
        appendEntry(mapped)
      }
      return nextState
    }
    case 'plan_updated':
      appendEntry(
        createProjectConversationTaskStatusEntry({
          id: consumeEntryId(),
          statusType: 'task_notification',
          title: 'Plan updated',
          detail: formatSkillRefinementPlanDetail(event.payload.explanation, event.payload.plan),
          raw: {
            thread_id: event.payload.threadId,
            turn_id: event.payload.turnId,
            explanation: event.payload.explanation,
            plan: event.payload.plan,
            entry_id: event.payload.entryId,
          },
        }),
      )
      return nextState
    case 'diff_updated': {
      const diffEntries = createProjectConversationDiffEntriesFromUnifiedDiff({
        idBase: consumeEntryId(),
        diff: event.payload.diff,
      })
      if (diffEntries.length === 0) {
        appendEntry(
          createProjectConversationTaskStatusEntry({
            id: consumeEntryId(),
            statusType: 'task_notification',
            title: 'Diff updated',
            detail: event.payload.diff,
            raw: {
              thread_id: event.payload.threadId,
              turn_id: event.payload.turnId,
              entry_id: event.payload.entryId,
            },
          }),
        )
        return nextState
      }
      for (const entry of diffEntries) {
        appendEntry(entry)
      }
      return nextState
    }
    case 'reasoning_updated':
      appendEntry(
        createProjectConversationTaskStatusEntry({
          id: consumeEntryId(),
          statusType: 'reasoning_updated',
          title: 'Reasoning update',
          detail: event.payload.delta || `Kind: ${event.payload.kind.replaceAll('_', ' ').trim()}`,
          raw: {
            thread_id: event.payload.threadId,
            turn_id: event.payload.turnId,
            item_id: event.payload.itemId,
            kind: event.payload.kind,
            delta: event.payload.delta,
            summary_index: event.payload.summaryIndex,
            content_index: event.payload.contentIndex,
            entry_id: event.payload.entryId,
          },
        }),
      )
      return nextState
    case 'thread_compacted':
      appendEntry(
        createProjectConversationTaskStatusEntry({
          id: consumeEntryId(),
          statusType: 'task_notification',
          title: 'Thread compacted',
          detail: event.payload.threadId
            ? `Thread ${event.payload.threadId} compacted`
            : 'Thread compacted',
          raw: {
            thread_id: event.payload.threadId,
            turn_id: event.payload.turnId,
            entry_id: event.payload.entryId,
          },
        }),
      )
      return nextState
    case 'session_anchor':
      appendEntry(
        createProjectConversationTaskStatusEntry({
          id: consumeEntryId(),
          statusType: 'task_notification',
          title: formatSkillRefinementAnchorTitle(event.payload),
          detail: formatSkillRefinementAnchorDetail(event.payload),
          raw: {
            provider_anchor_id: event.payload.providerAnchorId,
            provider_anchor_kind: event.payload.providerAnchorKind,
            provider_thread_id: event.payload.providerThreadId,
            provider_turn_id: event.payload.providerTurnId,
            provider_turn_supported: event.payload.providerTurnSupported,
          },
        }),
      )
      return nextState
    case 'result':
      if (event.payload.status === 'verified') {
        appendEntry(createProjectConversationTurnDoneEntry({ id: consumeEntryId() }))
      } else {
        appendEntry(
          createProjectConversationErrorEntry({
            id: consumeEntryId(),
            message: event.payload.failureReason || 'Verification did not pass.',
          }),
        )
      }
      return nextState
    case 'error':
      appendEntry(
        createProjectConversationErrorEntry({
          id: consumeEntryId(),
          message: event.payload.message,
        }),
      )
      return nextState
  }
}

export function updateSkillRefinementAnchorState(
  anchor: SkillRefinementAnchorState,
  event: SkillRefinementStreamEvent,
): SkillRefinementAnchorState {
  switch (event.kind) {
    case 'session_anchor':
      return {
        ...anchor,
        anchorKind: event.payload.providerAnchorKind || anchor.anchorKind,
        anchorId:
          event.payload.providerAnchorId || event.payload.providerThreadId || anchor.anchorId,
        turnId: event.payload.providerTurnId || anchor.turnId,
      }
    case 'result':
      return {
        ...anchor,
        providerName: event.payload.providerName || anchor.providerName,
        anchorKind: anchor.anchorKind || 'thread',
        anchorId: event.payload.providerThreadId || anchor.anchorId,
        turnId: event.payload.providerTurnId || anchor.turnId,
      }
    default:
      return anchor
  }
}
