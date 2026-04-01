import type { SSEFrame } from '$lib/api/sse'
import type {
  TicketRun,
  TicketRunDetail,
  TicketRunTranscriptInterruptOption,
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
  mergeRunTextBlock,
  readPayloadString,
  seedRunBlocks,
  sortTicketRuns,
} from './run-transcript-blocks'
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

function buildInterruptBlock(entry: TicketRunTraceEntry) {
  const options = parseInterruptOptions(entry.payload.options)
  const interruptKind =
    entry.kind === 'user_input_requested'
      ? 'user_input'
      : normalizeApprovalInterruptKind(readPayloadString(entry.payload, 'kind'))
  return {
    kind: 'interrupt' as const,
    id: `interrupt:${readPayloadString(entry.payload, 'request_id') || entry.id}`,
    interruptKind,
    title: interruptTitle(interruptKind),
    summary: entry.output || interruptSummary(interruptKind, entry.payload),
    at: entry.createdAt,
    payload: entry.payload,
    options,
  }
}

function parseInterruptOptions(value: unknown): TicketRunTranscriptInterruptOption[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map((item) => (item && typeof item === 'object' ? (item as Record<string, unknown>) : null))
    .filter((item): item is Record<string, unknown> => item != null)
    .map((item) => ({
      id: typeof item.id === 'string' ? item.id : '',
      label: typeof item.label === 'string' ? item.label : 'Decision',
      rawDecision: typeof item.raw_decision === 'string' ? item.raw_decision : undefined,
    }))
    .filter((item) => item.id !== '')
}

function normalizeApprovalInterruptKind(raw: string) {
  return raw === 'file_change' ? 'file_change_approval' : 'command_execution_approval'
}

function interruptTitle(kind: string) {
  switch (kind) {
    case 'user_input':
      return 'User input required'
    case 'file_change_approval':
      return 'File change approval required'
    default:
      return 'Command approval required'
  }
}

function interruptSummary(kind: string, payload: Record<string, unknown>) {
  const questions = payload.questions
  if (kind === 'user_input' && Array.isArray(questions) && questions.length > 0) {
    const first = questions[0]
    if (
      first &&
      typeof first === 'object' &&
      typeof (first as Record<string, unknown>).question === 'string'
    ) {
      return String((first as Record<string, unknown>).question)
    }
  }

  if (kind === 'file_change_approval') {
    return readInterruptString(payload, 'file', 'path', 'target') || 'Pending file approval.'
  }

  return readInterruptString(payload, 'command') || 'Pending approval.'
}

function readInterruptString(payload: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = payload[key]
    if (typeof value === 'string' && value.trim()) {
      return value
    }
  }
  return ''
}
