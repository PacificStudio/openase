import type { SSEFrame } from '$lib/api/sse'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptState,
} from './types'
import {
  buildLifecycleBlock,
  buildRunTimeline,
  finalizeTerminalRunBlocks,
  hasBlock,
  mergeRun,
  seedRunBlocks,
} from './run-transcript-blocks'
import { applyTicketRunTraceEntry as reduceTicketRunTraceEntry } from './run-transcript-trace'
import { syncSelectedBlocks, syncSelectedRun } from './run-transcript-selection'
export {
  createEmptyTicketRunTranscriptState,
  selectTicketRun,
  setTicketRunList,
} from './run-transcript-state'
import {
  insertBlockChronologically,
  mergeDefinedRunFields,
  mergeHydratedRunBlocks,
  mergeHydratedRunSnapshot,
  mergeRunStepSnapshot,
  mergeStreamingRunSnapshot,
} from './run-transcript-run-helpers'
import {
  mapTicketRun,
  mapTicketRunCompletionSummary,
  mapTicketRunStepEntry,
  mapTicketRunStreamLifecycleEvent,
  mapTicketRunTraceEntry,
} from './run-transcript-data'
import type { TicketRunRecord, TicketRunStepRecord, TicketRunTraceRecord } from '$lib/api/contracts'

export function hydrateTicketRunDetail(
  state: TicketRunTranscriptState,
  detail: TicketRunDetail,
  options: { select?: boolean } = {},
): TicketRunTranscriptState {
  const existingRun = state.runs.find((item) => item.id === detail.run.id)
  const hydratedRun = mergeHydratedRunSnapshot(existingRun, detail.run)
  const runs = mergeRun(state.runs, hydratedRun)
  const cachedBlocks =
    state.currentRun?.id === detail.run.id ? state.blocks : (state.blockCache[detail.run.id] ?? [])
  let nextState: TicketRunTranscriptState = {
    ...state,
    runs,
    currentRun: hydratedRun,
    blocks: mergeHydratedRunBlocks(cachedBlocks, hydratedRun),
  }

  const timeline = buildRunTimeline(detail.stepEntries, detail.traceEntries)
  for (const item of timeline) {
    nextState =
      item.kind === 'step'
        ? applyTicketRunStepEntry(nextState, item.entry)
        : applyTicketRunTraceEntry(nextState, item.entry)
  }

  const finalized = finalizeTerminalRunBlocks(nextState)
  const finalizedCurrentRun =
    finalized.currentRun?.id === hydratedRun.id
      ? mergeDefinedRunFields(finalized.currentRun, hydratedRun)
      : finalized.currentRun
  const finalizedRuns = mergeRun(
    finalized.runs,
    mergeDefinedRunFields(
      finalized.runs.find((run) => run.id === hydratedRun.id) ?? hydratedRun,
      hydratedRun,
    ),
  )
  const nextSelection =
    options.select === false
      ? {
          selectedRunId: state.selectedRunId,
          followLatest: state.followLatest,
        }
      : {
          selectedRunId: hydratedRun.id,
          followLatest: runs[0]?.id === hydratedRun.id,
        }

  return syncSelectedRun({
    ...finalized,
    runs: finalizedRuns,
    currentRun: finalizedCurrentRun,
    ...nextSelection,
    blockCache: {
      ...state.blockCache,
      [detail.run.id]: finalized.blocks,
    },
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
  const currentRunForUpdate = state.runs.find((item) => item.id === run.id)
  const nextRun = mergeStreamingRunSnapshot(currentRunForUpdate, run)
  const runs = mergeRun(state.runs, nextRun)
  const baseBlocks =
    state.selectedRunId === run.id
      ? state.blocks
      : (state.blockCache[run.id] ?? seedRunBlocks(nextRun))

  const nextBlock = buildLifecycleBlock(lifecycle)
  const runBlocks =
    !nextBlock || hasBlock(baseBlocks, nextBlock.id)
      ? finalizeTerminalRunBlocks({
          ...state,
          currentRun: nextRun,
          blocks: baseBlocks,
        }).blocks
      : finalizeTerminalRunBlocks({
          ...state,
          currentRun: nextRun,
          blocks: insertBlockChronologically(baseBlocks, nextBlock),
        }).blocks

  return syncSelectedRun({
    ...state,
    runs,
    blockCache: {
      ...syncSelectedBlocks(state),
      [run.id]: runBlocks,
    },
  })
}

export function applyTicketRunStepEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunStepEntry,
): TicketRunTranscriptState {
  const runs = state.runs.map((run) =>
    run.id === entry.agentRunId ? mergeRunStepSnapshot(run, entry) : run,
  )

  const currentRun = state.currentRun
  if (!currentRun || currentRun.id !== entry.agentRunId) {
    return {
      ...state,
      runs,
    }
  }
  if (entry.sourceTraceEventId || hasBlock(state.blocks, `step:${entry.id}`)) {
    return syncSelectedRun({
      ...state,
      runs,
      blockCache: syncSelectedBlocks(state),
    })
  }

  const blocks = [
    ...state.blocks,
    {
      kind: 'step' as const,
      id: `step:${entry.id}`,
      stepStatus: entry.stepStatus,
      summary: entry.summary,
      at: entry.createdAt,
    },
  ]

  return syncSelectedRun({
    ...state,
    runs,
    currentRun: mergeRunStepSnapshot(currentRun, entry),
    blocks,
    blockCache: {
      ...syncSelectedBlocks(state),
      [entry.agentRunId]: blocks,
    },
  })
}

export function applyTicketRunTraceEntry(
  state: TicketRunTranscriptState,
  entry: TicketRunTraceEntry,
): TicketRunTranscriptState {
  return reduceTicketRunTraceEntry(state, entry)
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

  const existingRun = state.runs.find((item) => item.id === runID) ?? state.currentRun
  if (!existingRun || existingRun.id !== runID) {
    return state
  }

  const nextRun = mergeDefinedRunFields(existingRun, run)
  nextRun.completionSummary =
    completionSummary ?? run?.completionSummary ?? existingRun.completionSummary
  const runs = mergeRun(state.runs, nextRun)

  return syncSelectedRun(
    finalizeTerminalRunBlocks({
      ...state,
      runs,
      currentRun: state.currentRun?.id === runID ? nextRun : state.currentRun,
      blockCache: syncSelectedBlocks(state),
    }),
  )
}
