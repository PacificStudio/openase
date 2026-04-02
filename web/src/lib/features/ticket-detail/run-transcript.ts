import type { SSEFrame } from '$lib/api/sse'
import {
  asRecord,
  buildProviderStateDetail,
  buildReasoningDetail,
  parseUnifiedDiffPayloads,
  readString,
} from '$lib/features/chat'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunLifecycleEvent,
  TicketRunStepEntry,
  TicketRunTranscriptBlock,
  TicketRunTraceEntry,
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
import { buildInterruptBlock } from './run-transcript-interrupts'
import { cacheSelectedState, syncSelectedBlocks, syncSelectedRun } from './run-transcript-selection'

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
      return appendTraceBlocks(state, [
        {
          kind: 'tool_call',
          id: `tool:${entry.id}`,
          toolName: readPayloadString(entry.payload, 'tool') || entry.output || entry.stream,
          arguments: entry.payload.arguments,
          summary: readPayloadString(entry.payload, 'phase') || undefined,
          at: entry.createdAt,
        },
      ])
    case 'thread_status':
      return appendTraceBlocks(state, [
        {
          kind: 'task_status',
          id: `status:${entry.id}`,
          statusType: 'thread_status',
          title: 'Codex thread status',
          detail: buildProviderStateDetail(asRecord(entry.payload)) || entry.output || undefined,
          raw: Object.keys(entry.payload).length > 0 ? entry.payload : undefined,
          at: entry.createdAt,
        },
      ])
    case 'reasoning_updated':
      return appendTraceBlocks(state, [
        {
          kind: 'task_status',
          id: `reasoning:${entry.id}`,
          statusType: 'reasoning_updated',
          title: 'Reasoning update',
          detail: entry.output || buildReasoningDetail(asRecord(entry.payload)) || undefined,
          raw: Object.keys(entry.payload).length > 0 ? entry.payload : undefined,
          at: entry.createdAt,
        },
      ])
    case 'turn_diff_updated':
      return appendTraceBlocks(
        state,
        createDiffBlocksFromTraceEntry(entry).map((block, index) => ({
          ...block,
          id: index === 0 ? `diff:${entry.id}` : `diff:${entry.id}:${index + 1}`,
        })),
      )
    case 'approval_requested':
    case 'user_input_requested': {
      const interruptBlock = buildInterruptBlock(entry)
      if (!interruptBlock || hasBlock(state.blocks, interruptBlock.id)) {
        return state
      }
      return cacheSelectedState({
        ...state,
        blocks: [...state.blocks, interruptBlock],
      })
    }
    default:
      return state
  }
}

function appendTraceBlocks(
  state: TicketRunTranscriptState,
  blocks: TicketRunTranscriptBlock[],
): TicketRunTranscriptState {
  const nextBlocks = blocks.filter((block) => !hasBlock(state.blocks, block.id))
  if (nextBlocks.length === 0) {
    return state
  }

  return cacheSelectedState({
    ...state,
    blocks: [...state.blocks, ...nextBlocks],
  })
}

function createDiffBlocksFromTraceEntry(
  entry: TicketRunTraceEntry,
): Array<Extract<TicketRunTranscriptBlock, { kind: 'diff' }>> {
  const diffText = readString(asRecord(entry.payload), 'diff') || entry.output
  if (!diffText) {
    return []
  }

  return parseUnifiedDiffPayloads(diffText).map((diff) => ({
    kind: 'diff',
    id: '',
    at: entry.createdAt,
    diff: {
      ...diff,
      entryId: entry.id,
    },
  }))
}
