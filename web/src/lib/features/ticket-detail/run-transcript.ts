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
  sortTicketRuns,
} from './run-transcript-blocks'
import { applyTicketRunTraceEntry as reduceTicketRunTraceEntry } from './run-transcript-trace'
import { syncSelectedBlocks, syncSelectedRun } from './run-transcript-selection'
import {
  mapTicketRun,
  mapTicketRunCompletionSummary,
  mapTicketRunStepEntry,
  mapTicketRunStreamLifecycleEvent,
  mapTicketRunTraceEntry,
} from './run-transcript-data'
import type { TicketRunRecord, TicketRunStepRecord, TicketRunTraceRecord } from '$lib/api/contracts'

export function createEmptyTicketRunTranscriptState(): TicketRunTranscriptState {
  return {
    runs: [],
    selectedRunId: null,
    followLatest: true,
    currentRun: null,
    blocks: [],
    blockCache: {},
  }
}

export function setTicketRunList(
  state: TicketRunTranscriptState,
  runs: TicketRun[],
): TicketRunTranscriptState {
  const nextRuns = sortTicketRuns(runs)
  const blockCache = syncSelectedBlocks(state)

  return syncSelectedRun({
    ...state,
    runs: nextRuns,
    blockCache,
  })
}

export function hydrateTicketRunDetail(
  state: TicketRunTranscriptState,
  detail: TicketRunDetail,
  options: { select?: boolean } = {},
): TicketRunTranscriptState {
  const runs = mergeRun(state.runs, detail.run)
  let nextState: TicketRunTranscriptState = {
    ...state,
    runs,
    currentRun: detail.run,
    blocks: seedRunBlocks(detail.run),
  }

  const timeline = buildRunTimeline(detail.stepEntries, detail.traceEntries)
  for (const item of timeline) {
    nextState =
      item.kind === 'step'
        ? applyTicketRunStepEntry(nextState, item.entry)
        : applyTicketRunTraceEntry(nextState, item.entry)
  }

  const finalized = finalizeTerminalRunBlocks(nextState)
  const nextSelection =
    options.select === false
      ? {
          selectedRunId: state.selectedRunId,
          followLatest: state.followLatest,
        }
      : {
          selectedRunId: detail.run.id,
          followLatest: runs[0]?.id === detail.run.id,
        }

  return syncSelectedRun({
    ...finalized,
    ...nextSelection,
    blockCache: {
      ...state.blockCache,
      [detail.run.id]: finalized.blocks,
    },
  })
}

export function selectTicketRun(
  state: TicketRunTranscriptState,
  runId: string,
): TicketRunTranscriptState {
  if (!state.runs.some((run) => run.id === runId)) {
    return state
  }

  return syncSelectedRun({
    ...state,
    selectedRunId: runId,
    followLatest: state.runs[0]?.id === runId,
    blockCache: syncSelectedBlocks(state),
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
  const runs = mergeRun(state.runs, run)
  const baseBlocks =
    state.selectedRunId === run.id ? state.blocks : (state.blockCache[run.id] ?? seedRunBlocks(run))

  const nextBlock = buildLifecycleBlock(lifecycle)
  const runBlocks =
    !nextBlock || hasBlock(baseBlocks, nextBlock.id)
      ? finalizeTerminalRunBlocks({
          ...state,
          currentRun: run,
          blocks: baseBlocks,
        }).blocks
      : finalizeTerminalRunBlocks({
          ...state,
          currentRun: run,
          blocks: [...baseBlocks, nextBlock],
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
    run.id === entry.agentRunId
      ? {
          ...run,
          currentStepStatus: entry.stepStatus,
          currentStepSummary: entry.summary || run.currentStepSummary,
        }
      : run,
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
    currentRun: {
      ...currentRun,
      currentStepStatus: entry.stepStatus,
      currentStepSummary: entry.summary || currentRun.currentStepSummary,
    },
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
  completionSummary: TicketRun['completionSummary'],
): TicketRunTranscriptState {
  if (!runID || !completionSummary) {
    return state
  }

  const targetRun = state.runs.find((run) => run.id === runID) ?? state.currentRun
  if (!targetRun || targetRun.id !== runID) {
    return state
  }

  const nextRun: TicketRun = {
    ...targetRun,
    completionSummary,
  }
  const runs = mergeRun(state.runs, nextRun)

  return syncSelectedRun({
    ...state,
    runs,
    currentRun: state.currentRun?.id === runID ? nextRun : state.currentRun,
    blockCache: syncSelectedBlocks(state),
  })
}
