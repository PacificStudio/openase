import type { SSEFrame } from '$lib/api/sse'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTraceEntry,
  TicketRunTranscriptBlock,
  TicketRunTranscriptState,
} from './types'
import {
  buildLifecycleBlock,
  buildRunTimeline,
  finalizeTerminalRunBlocks,
  hasBlock,
  mergeRun,
  mergeRunTextBlock,
  readPayloadString,
  seedRunBlocks,
  sortTicketRuns,
} from './run-transcript-blocks'

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
          run: TicketRun
          lifecycle: TicketRunLifecycleEvent
        }
        return applyTicketRunLifecycleEvent(state, payload.run, payload.lifecycle)
      }
      case 'ticket.run.trace': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunTraceEntry }
        return applyTicketRunTraceEntry(state, payload.entry)
      }
      case 'ticket.run.step': {
        const payload = JSON.parse(frame.data) as { entry: TicketRunStepEntry }
        return applyTicketRunStepEntry(state, payload.entry)
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
    state.selectedRunId === run.id
      ? state.blocks
      : state.blockCache[run.id] ?? seedRunBlocks(run)

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

  if (state.currentRun?.id !== entry.agentRunId) {
    return {
      ...state,
      runs,
    }
  }
  if (hasBlock(state.blocks, `step:${entry.id}`)) {
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
      ...state.currentRun,
      currentStepStatus: entry.stepStatus,
      currentStepSummary: entry.summary || state.currentRun.currentStepSummary,
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
  if (state.currentRun?.id !== entry.agentRunId) {
    return state
  }

  switch (entry.kind) {
    case 'assistant_delta':
    case 'assistant_snapshot':
      return cacheSelectedState(mergeRunTextBlock(state, entry, 'assistant_message'))
    case 'command_output_delta':
    case 'command_output_snapshot':
      return cacheSelectedState(mergeRunTextBlock(state, entry, 'terminal_output'))
    case 'tool_call_started':
      if (hasBlock(state.blocks, `tool:${entry.id}`)) {
        return state
      }
      return cacheSelectedState({
        ...state,
        blocks: [
          ...state.blocks,
          {
            kind: 'tool_call',
            id: `tool:${entry.id}`,
            toolName: readPayloadString(entry.payload, 'tool') || entry.output || entry.stream,
            summary: readPayloadString(entry.payload, 'phase') || undefined,
            at: entry.createdAt,
          },
        ],
      })
    default:
      return state
  }
}

function syncSelectedRun(state: TicketRunTranscriptState): TicketRunTranscriptState {
  const latestRun = state.runs[0] ?? null
  if (!latestRun) {
    return {
      ...state,
      selectedRunId: null,
      followLatest: true,
      currentRun: null,
      blocks: [],
    }
  }

  let followLatest = state.followLatest
  let selectedRunId = state.selectedRunId

  if (!selectedRunId) {
    selectedRunId = latestRun.id
    followLatest = true
  }

  if (followLatest) {
    selectedRunId = latestRun.id
  }

  const selectedRun = state.runs.find((run) => run.id === selectedRunId) ?? latestRun
  if (!state.runs.some((run) => run.id === selectedRun.id)) {
    followLatest = true
    selectedRunId = latestRun.id
  }

  return {
    ...state,
    selectedRunId,
    followLatest,
    currentRun: selectedRun,
    blocks: state.blockCache[selectedRun.id] ?? seedRunBlocks(selectedRun),
  }
}

function syncSelectedBlocks(
  state: TicketRunTranscriptState,
): Record<string, TicketRunTranscriptBlock[]> {
  if (!state.currentRun || state.blocks.length === 0) {
    return state.blockCache
  }

  return {
    ...state.blockCache,
    [state.currentRun.id]: state.blocks,
  }
}

function cacheSelectedState(state: TicketRunTranscriptState): TicketRunTranscriptState {
  return {
    ...state,
    blockCache: syncSelectedBlocks(state),
  }
}
