import type { SSEFrame } from '$lib/api/sse'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptState,
} from './types'
import { buildLifecycleBlock, hasBlock, mergeRun } from './run-transcript-blocks'
import { syncSelectedRun } from './run-transcript-selection'
export {
  createEmptyTicketRunTranscriptState,
  selectTicketRun,
  setTicketRunList,
} from './run-transcript-state'
import {
  insertBlockChronologically,
  mergeDefinedRunFields,
  mergeHydratedRunSnapshot,
  mergeStreamingRunSnapshot,
} from './run-transcript-run-helpers'
import {
  mapTicketRun,
  mapTicketRunCompletionSummary,
  mapTicketRunStepEntry,
  mapTicketRunStreamLifecycleEvent,
  mapTicketRunTraceEntry,
} from './run-transcript-data'
import { buildTicketRunStepCursor, buildTicketRunTraceCursor } from './run-transcript-cursor'
import type { TicketRunRecord, TicketRunStepRecord, TicketRunTraceRecord } from '$lib/api/contracts'
import {
  mergeStepEntryIntoState,
  mergeTraceEntryIntoState,
  mergeTranscriptPageIntoState,
  rebuildRunTranscriptState,
} from './run-transcript-reducer-helpers'

export function hydrateTicketRunDetail(
  state: TicketRunTranscriptState,
  detail: TicketRunDetail,
  options: { select?: boolean } = {},
): TicketRunTranscriptState {
  const existingRun = state.runs.find((item) => item.id === detail.run.id)
  const hydratedRun = mergeHydratedRunSnapshot(existingRun, detail.run)

  let nextState = mergeTranscriptPageIntoState(
    {
      ...state,
      runs: mergeRun(state.runs, hydratedRun),
    },
    detail.run.id,
    detail.transcriptPage,
  )
  nextState = rebuildRunTranscriptState(nextState, detail.run.id, hydratedRun)

  const selection =
    options.select === false
      ? {
          selectedRunId: nextState.selectedRunId,
          followLatest: nextState.followLatest,
        }
      : {
          selectedRunId: hydratedRun.id,
          followLatest: nextState.runs[0]?.id === hydratedRun.id,
        }

  return syncSelectedRun({
    ...nextState,
    ...selection,
  })
}

export function applyTicketRunStreamFrame(
  state: TicketRunTranscriptState,
  frame: Pick<SSEFrame, 'event' | 'data'>,
): TicketRunTranscriptState {
  try {
    switch (frame.event) {
      case 'ticket.run.lifecycle': {
        const payload = JSON.parse(frame.data) as {
          run: TicketRunRecord
          lifecycle: {
            event_type?: string
            eventType?: string
            message?: string
            created_at?: string
            createdAt?: string
          }
        }
        return applyTicketRunLifecycleEvent(
          state,
          mapTicketRun(payload.run),
          mapTicketRunStreamLifecycleEvent(payload.lifecycle),
        )
      }
      case 'ticket.run.trace': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunTraceRecord }
        return applyTicketRunTraceEntry(state, mapTicketRunTraceEntry(payload.entry))
      }
      case 'ticket.run.step': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunStepRecord }
        return applyTicketRunStepEntry(state, mapTicketRunStepEntry(payload.entry))
      }
      case 'ticket.run.summary': {
        const payload = JSON.parse(frame.data) as {
          run_id?: string
          runId?: string
          run?: TicketRunRecord
          completion_summary?: {
            status?: string
            markdown?: string | null
            json?: Record<string, unknown> | null
            generated_at?: string | null
            error?: string | null
          } | null
        }
        return applyTicketRunSummaryEvent(
          state,
          payload.run_id ?? payload.runId ?? '',
          payload.run ? mapTicketRun(payload.run) : undefined,
          mapTicketRunCompletionSummary(payload.completion_summary),
        )
      }
      default:
        return state
    }
  } catch {
    return state
  }
}

export function applyTicketRunLifecycleEvent(
  state: TicketRunTranscriptState,
  run: TicketRun,
  lifecycle: TicketRunLifecycleEvent,
): TicketRunTranscriptState {
  const currentRunForUpdate = getRunForUpdate(state, run.id)
  const nextRun = mergeStreamingRunSnapshot(currentRunForUpdate ?? undefined, run)
  const nextBlock = buildLifecycleBlock(lifecycle)
  const lifecycleBlocks = state.lifecycleBlocksByRun[run.id] ?? []
  const nextLifecycleBlocks =
    !nextBlock || hasBlock(lifecycleBlocks, nextBlock.id)
      ? lifecycleBlocks
      : insertBlockChronologically(lifecycleBlocks, nextBlock)

  return rebuildRunTranscriptState(
    {
      ...state,
      runs: mergeRun(state.runs, nextRun),
      lifecycleBlocksByRun: {
        ...state.lifecycleBlocksByRun,
        [run.id]: nextLifecycleBlocks,
      },
    },
    run.id,
    nextRun,
  )
}

export function applyTicketRunStepEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunStepEntry,
): TicketRunTranscriptState {
  const baseRun = getRunForUpdate(state, entry.agentRunId)
  if (!baseRun) {
    return state
  }

  state = mergeStepEntryIntoState(state, entry)

  return rebuildRunTranscriptState(
    state,
    entry.agentRunId,
    baseRun,
    buildTicketRunStepCursor(entry),
  )
}

export function applyTicketRunTraceEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunTraceEntry,
): TicketRunTranscriptState {
  const baseRun = getRunForUpdate(state, entry.agentRunId)
  if (!baseRun) {
    return state
  }

  state = mergeTraceEntryIntoState(state, entry)

  return rebuildRunTranscriptState(
    state,
    entry.agentRunId,
    baseRun,
    buildTicketRunTraceCursor(entry),
  )
}

export function applyTicketRunSummaryEvent(
  state: TicketRunTranscriptState,
  runID: string,
  run: TicketRun | undefined,
  completionSummary: TicketRun['completionSummary'],
): TicketRunTranscriptState {
  if (!runID || (!completionSummary && !run)) {
    return state
  }

  const existingRun = getRunForUpdate(state, runID)
  if (!existingRun) {
    return state
  }

  const nextRun = mergeDefinedRunFields(existingRun, run)
  nextRun.completionSummary =
    completionSummary ?? run?.completionSummary ?? existingRun.completionSummary

  return rebuildRunTranscriptState(
    {
      ...state,
      runs: mergeRun(state.runs, nextRun),
    },
    runID,
    nextRun,
  )
}

function getRunForUpdate(state: TicketRunTranscriptState, runId: string): TicketRun | null {
  return (
    state.runs.find((item) => item.id === runId) ??
    (state.currentRun?.id === runId ? state.currentRun : null)
  )
}
